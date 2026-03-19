package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type workflowAPITestRunner struct {
	kind   domain.ExecutorKind
	result executor.Result
}

func (runner workflowAPITestRunner) Kind() domain.ExecutorKind { return runner.kind }

func (runner workflowAPITestRunner) Capability() executor.Capability {
	return executor.Capability{Kind: runner.kind, MaxConcurrency: 1, SupportsShell: true}
}

func (runner workflowAPITestRunner) Execute(_ context.Context, _ domain.Task) executor.Result {
	result := runner.result
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Unix(1700000002, 0).UTC()
	}
	return result
}

func TestV2WeeklyReportBuildsSummaryActionsAndMarkdownExport(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{ID: "task-weekly-1", TraceID: "trace-weekly-1", Title: "Ship alpha", State: domain.TaskSucceeded, RiskLevel: domain.RiskHigh, BudgetCents: 900, Metadata: map[string]string{"team": "platform", "project": "alpha", "plan": "premium", "regression_count": "1"}, CreatedAt: base, UpdatedAt: base.Add(24 * time.Hour)},
		{ID: "task-weekly-2", TraceID: "trace-weekly-2", Title: "Fix blocker", State: domain.TaskBlocked, BudgetCents: 300, Metadata: map[string]string{"team": "platform", "project": "alpha", "blocked_reason": "waiting on release"}, CreatedAt: base, UpdatedAt: base.Add(48 * time.Hour)},
		{ID: "task-weekly-3", TraceID: "trace-weekly-3", Title: "Other team", State: domain.TaskSucceeded, BudgetCents: 200, Metadata: map[string]string{"team": "growth", "project": "beta"}, CreatedAt: base, UpdatedAt: base.Add(72 * time.Hour)},
	}
	for _, task := range tasks {
		recorder.StoreTask(task)
	}
	recorder.Record(domain.Event{ID: "evt-weekly-1", Type: domain.EventRunTakeover, TaskID: "task-weekly-2", TraceID: "trace-weekly-2", Timestamp: base.Add(48 * time.Hour), Payload: map[string]any{"actor": "alice", "note": "manual review"}})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(6 * 24 * time.Hour) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/weekly?team=platform&project=alpha&week_start=2026-03-09T00:00:00Z&week_end=2026-03-15T00:00:00Z", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected weekly report 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Summary struct {
			TotalRuns          int   `json:"total_runs"`
			CompletedRuns      int   `json:"completed_runs"`
			BlockedRuns        int   `json:"blocked_runs"`
			HighRiskRuns       int   `json:"high_risk_runs"`
			RegressionFindings int   `json:"regression_findings"`
			HumanInterventions int   `json:"human_interventions"`
			BudgetCentsTotal   int64 `json:"budget_cents_total"`
			PremiumRuns        int   `json:"premium_runs"`
		} `json:"summary"`
		TeamBreakdown []struct {
			Key       string `json:"key"`
			TotalRuns int    `json:"total_runs"`
		} `json:"team_breakdown"`
		Highlights []string `json:"highlights"`
		Actions    []string `json:"actions"`
		Report     struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode weekly report: %v", err)
	}
	if decoded.Summary.TotalRuns != 2 || decoded.Summary.CompletedRuns != 1 || decoded.Summary.BlockedRuns != 1 || decoded.Summary.HighRiskRuns != 1 || decoded.Summary.RegressionFindings != 1 || decoded.Summary.HumanInterventions != 1 || decoded.Summary.BudgetCentsTotal != 1200 || decoded.Summary.PremiumRuns != 1 {
		t.Fatalf("unexpected weekly summary: %+v", decoded.Summary)
	}
	if len(decoded.TeamBreakdown) != 1 || decoded.TeamBreakdown[0].Key != "platform" || decoded.TeamBreakdown[0].TotalRuns != 2 {
		t.Fatalf("unexpected team breakdown: %+v", decoded.TeamBreakdown)
	}
	if len(decoded.Highlights) == 0 || len(decoded.Actions) == 0 || !strings.Contains(decoded.Report.Markdown, "# BigClaw Weekly Ops Report") || !strings.Contains(decoded.Report.ExportURL, "/v2/reports/weekly/export") {
		t.Fatalf("expected highlights/actions/export in weekly report, got %+v", decoded)
	}

	exportResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("expected weekly export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
	}
	if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown export, got %q", contentType)
	}
	if !strings.Contains(exportResponse.Body.String(), "Completed runs: 1") || !strings.Contains(exportResponse.Body.String(), "Human interventions: 1") {
		t.Fatalf("unexpected weekly export body: %s", exportResponse.Body.String())
	}
}

func TestV2FlowTemplateLifecyclePRDIntakeChecklistAndSupportHandoff(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Control: control.New(), Now: func() time.Time { return time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC) }}
	handler := server.Handler()

	prdBody, _ := json.Marshal(map[string]any{"title": "Launch AI Ops", "body": "Need launch checklist, documentation, release review, support readiness, and approval workflow.", "acceptance_criteria": []string{"ship launch flow", "support readiness"}})
	prdResponse := httptest.NewRecorder()
	handler.ServeHTTP(prdResponse, httptest.NewRequest(http.MethodPost, "/v2/prd/intake", bytes.NewReader(prdBody)))
	if prdResponse.Code != http.StatusOK {
		t.Fatalf("expected prd intake 200, got %d %s", prdResponse.Code, prdResponse.Body.String())
	}
	var prdDecoded struct {
		Title          string `json:"title"`
		SuggestedTasks []struct {
			Title string `json:"title"`
		} `json:"suggested_tasks"`
		SuggestedTemplate struct {
			ID    string `json:"id"`
			Nodes []struct {
				Department string `json:"department"`
			} `json:"nodes"`
		} `json:"suggested_template"`
		Signals []string `json:"signals"`
	}
	if err := json.Unmarshal(prdResponse.Body.Bytes(), &prdDecoded); err != nil {
		t.Fatalf("decode prd intake: %v", err)
	}
	if prdDecoded.Title != "Launch AI Ops" || len(prdDecoded.SuggestedTasks) != 4 || len(prdDecoded.SuggestedTemplate.Nodes) != 4 || len(prdDecoded.Signals) == 0 {
		t.Fatalf("unexpected prd intake payload: %+v", prdDecoded)
	}

	templateBody, _ := json.Marshal(map[string]any{
		"id":      "launch-kit",
		"name":    "Launch Kit",
		"summary": "Cross-functional launch flow",
		"nodes": []map[string]any{
			{"id": "engineering", "name": "Engineering", "department": "engineering", "owner": "eng-a", "validation": "ship implementation", "approval": "eng_lead"},
			{"id": "docs", "name": "Docs", "department": "docs", "owner": "docs-a", "validation": "publish docs", "approval": "docs_owner", "depends_on": []string{"engineering"}},
			{"id": "release", "name": "Release", "department": "release", "owner": "release-a", "validation": "complete checklist", "approval": "release_manager", "depends_on": []string{"docs"}},
			{"id": "support", "name": "Support", "department": "support", "owner": "support-a", "validation": "prepare support packet", "approval": "support_lead", "depends_on": []string{"release"}},
		},
	})
	templateResponse := httptest.NewRecorder()
	handler.ServeHTTP(templateResponse, httptest.NewRequest(http.MethodPost, "/v2/flows/templates", bytes.NewReader(templateBody)))
	if templateResponse.Code != http.StatusOK {
		t.Fatalf("expected template save 200, got %d %s", templateResponse.Code, templateResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/v2/flows/templates", nil))
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "launch-kit") {
		t.Fatalf("expected saved template in list response, got %d %s", listResponse.Code, listResponse.Body.String())
	}

	launchBody, _ := json.Marshal(map[string]any{"actor": "pm-1", "team": "platform", "project": "apollo", "title": "Apollo GA"})
	launchResponse := httptest.NewRecorder()
	handler.ServeHTTP(launchResponse, httptest.NewRequest(http.MethodPost, "/v2/flows/templates/launch-kit/launch", bytes.NewReader(launchBody)))
	if launchResponse.Code != http.StatusAccepted {
		t.Fatalf("expected flow launch 202, got %d %s", launchResponse.Code, launchResponse.Body.String())
	}
	var launchDecoded struct {
		FlowID        string        `json:"flow_id"`
		LaunchedTasks []domain.Task `json:"launched_tasks"`
	}
	if err := json.Unmarshal(launchResponse.Body.Bytes(), &launchDecoded); err != nil {
		t.Fatalf("decode flow launch: %v", err)
	}
	if launchDecoded.FlowID == "" || len(launchDecoded.LaunchedTasks) != 4 {
		t.Fatalf("unexpected flow launch payload: %+v", launchDecoded)
	}

	for _, task := range launchDecoded.LaunchedTasks {
		stored, ok := recorder.Task(task.ID)
		if !ok {
			t.Fatalf("expected launched task %s to be stored", task.ID)
		}
		switch stored.Metadata["department"] {
		case "engineering", "docs":
			stored.State = domain.TaskSucceeded
		case "release":
			stored.State = domain.TaskBlocked
			stored.Metadata["blocked_reason"] = "waiting for approval"
		case "support":
			stored.State = domain.TaskQueued
		}
		stored.UpdatedAt = server.Now().Add(time.Minute)
		recorder.StoreTask(stored)
	}

	overviewResponse := httptest.NewRecorder()
	handler.ServeHTTP(overviewResponse, httptest.NewRequest(http.MethodGet, "/v2/flows/overview?flow_id="+launchDecoded.FlowID, nil))
	if overviewResponse.Code != http.StatusOK {
		t.Fatalf("expected flow overview 200, got %d %s", overviewResponse.Code, overviewResponse.Body.String())
	}
	if !strings.Contains(overviewResponse.Body.String(), "blocked") || !strings.Contains(overviewResponse.Body.String(), "release") || !strings.Contains(overviewResponse.Body.String(), "support") {
		t.Fatalf("expected blocked release/support data in flow overview, got %s", overviewResponse.Body.String())
	}

	checklistResponse := httptest.NewRecorder()
	handler.ServeHTTP(checklistResponse, httptest.NewRequest(http.MethodGet, "/v2/launch/checklist?flow_id="+launchDecoded.FlowID, nil))
	if checklistResponse.Code != http.StatusOK {
		t.Fatalf("expected launch checklist 200, got %d %s", checklistResponse.Code, checklistResponse.Body.String())
	}
	if !strings.Contains(checklistResponse.Body.String(), "docs_ready") || !strings.Contains(checklistResponse.Body.String(), "blocked") || !strings.Contains(checklistResponse.Body.String(), "engineering_validation") {
		t.Fatalf("expected launch checklist states, got %s", checklistResponse.Body.String())
	}

	handoffResponse := httptest.NewRecorder()
	handler.ServeHTTP(handoffResponse, httptest.NewRequest(http.MethodGet, "/v2/support/handoff?flow_id="+launchDecoded.FlowID, nil))
	if handoffResponse.Code != http.StatusOK {
		t.Fatalf("expected support handoff 200, got %d %s", handoffResponse.Code, handoffResponse.Body.String())
	}
	if !strings.Contains(handoffResponse.Body.String(), "KnownIssues") && !strings.Contains(strings.ToLower(handoffResponse.Body.String()), "known_issues") {
		t.Fatalf("expected support handoff fields, got %s", handoffResponse.Body.String())
	}
	if !strings.Contains(handoffResponse.Body.String(), "waiting for approval") || !strings.Contains(handoffResponse.Body.String(), "Support ticket") {
		t.Fatalf("expected blocked issue and support template in handoff response, got %s", handoffResponse.Body.String())
	}
}

func TestV2ProductizationAndBillingEndpoints(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2026, 3, 14, 11, 0, 0, 0, time.UTC)
	for _, task := range []domain.Task{
		{ID: "task-product-1", TraceID: "trace-product-1", Title: "Premium platform task", State: domain.TaskRunning, RiskLevel: domain.RiskHigh, BudgetCents: 1400, Metadata: map[string]string{"team": "platform", "project": "apollo", "plan": "premium", "owner": "alice", "reviewer": "bob", "created_by": "pm-1", "flow_id": "flow-1", "department": "engineering", "regression_count": "1"}, CreatedAt: base, UpdatedAt: base},
		{ID: "task-product-2", TraceID: "trace-product-2", Title: "Support task", State: domain.TaskBlocked, BudgetCents: 200, Metadata: map[string]string{"team": "support", "project": "apollo", "owner": "carol", "reviewer": "dave", "created_by": "ops-1", "flow_id": "flow-1", "department": "support"}, CreatedAt: base, UpdatedAt: base.Add(time.Minute)},
		{ID: "task-product-3", TraceID: "trace-product-3", Title: "Completed run", State: domain.TaskSucceeded, BudgetCents: 100, Metadata: map[string]string{"team": "platform", "project": "apollo", "owner": "alice", "created_by": "pm-1"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Minute)},
	} {
		recorder.StoreTask(task)
	}
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(time.Hour) }}
	handler := server.Handler()

	navigationResponse := httptest.NewRecorder()
	handler.ServeHTTP(navigationResponse, httptest.NewRequest(http.MethodGet, "/v2/navigation?role=vp_eng", nil))
	if navigationResponse.Code != http.StatusOK || !strings.Contains(navigationResponse.Body.String(), "Billing") || !strings.Contains(navigationResponse.Body.String(), "Flows") {
		t.Fatalf("expected product navigation response, got %d %s", navigationResponse.Code, navigationResponse.Body.String())
	}

	homeResponse := httptest.NewRecorder()
	handler.ServeHTTP(homeResponse, httptest.NewRequest(http.MethodGet, "/v2/home?role=vp_eng", nil))
	if homeResponse.Code != http.StatusOK {
		t.Fatalf("expected role home 200, got %d %s", homeResponse.Code, homeResponse.Body.String())
	}
	if !strings.Contains(homeResponse.Body.String(), "throughput") || !strings.Contains(homeResponse.Body.String(), "risk") || !strings.Contains(homeResponse.Body.String(), "spend") {
		t.Fatalf("expected VP Eng home cards, got %s", homeResponse.Body.String())
	}

	designResponse := httptest.NewRecorder()
	handler.ServeHTTP(designResponse, httptest.NewRequest(http.MethodGet, "/v2/design-system", nil))
	if designResponse.Code != http.StatusOK || !strings.Contains(designResponse.Body.String(), "status-badge") || !strings.Contains(designResponse.Body.String(), "dark_mode") {
		t.Fatalf("expected design system payload, got %d %s", designResponse.Code, designResponse.Body.String())
	}

	usageResponse := httptest.NewRecorder()
	handler.ServeHTTP(usageResponse, httptest.NewRequest(http.MethodGet, "/v2/billing/usage?organization=openagi&tier=enterprise", nil))
	if usageResponse.Code != http.StatusOK {
		t.Fatalf("expected billing usage 200, got %d %s", usageResponse.Code, usageResponse.Body.String())
	}
	var usageDecoded struct {
		Organization     string `json:"organization"`
		Tier             string `json:"tier"`
		SeatCount        int    `json:"seat_count"`
		ActiveSeats      int    `json:"active_seats"`
		BudgetCentsTotal int64  `json:"budget_cents_total"`
		PremiumRuns      int    `json:"premium_runs"`
		StandardRuns     int    `json:"standard_runs"`
		ByTeam           []struct {
			Key       string `json:"key"`
			SeatCount int    `json:"seat_count"`
		} `json:"by_team"`
		Alerts []string `json:"alerts"`
	}
	if err := json.Unmarshal(usageResponse.Body.Bytes(), &usageDecoded); err != nil {
		t.Fatalf("decode billing usage: %v", err)
	}
	if usageDecoded.Organization != "openagi" || usageDecoded.Tier != "enterprise" || usageDecoded.SeatCount != 6 || usageDecoded.ActiveSeats != 6 || usageDecoded.BudgetCentsTotal != 1700 || usageDecoded.PremiumRuns != 1 || usageDecoded.StandardRuns != 2 || len(usageDecoded.ByTeam) != 2 {
		t.Fatalf("unexpected billing usage payload: %+v", usageDecoded)
	}

	entitlementsResponse := httptest.NewRecorder()
	handler.ServeHTTP(entitlementsResponse, httptest.NewRequest(http.MethodGet, "/v2/billing/entitlements?tier=enterprise", nil))
	if entitlementsResponse.Code != http.StatusOK {
		t.Fatalf("expected entitlements 200, got %d %s", entitlementsResponse.Code, entitlementsResponse.Body.String())
	}
	if !strings.Contains(entitlementsResponse.Body.String(), "premium_orchestration") || !strings.Contains(entitlementsResponse.Body.String(), "flow_canvas") || !strings.Contains(entitlementsResponse.Body.String(), "regression") {
		t.Fatalf("expected enterprise entitlements payload, got %s", entitlementsResponse.Body.String())
	}
}

func TestV2IntakeConnectorsMappingAndWorkflowDefinitionRender(t *testing.T) {
	recorder := observability.NewRecorder()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC) }}
	handler := server.Handler()

	connectorResponse := httptest.NewRecorder()
	handler.ServeHTTP(connectorResponse, httptest.NewRequest(http.MethodGet, "/v2/intake/connectors/github/issues?project=OpenAGIs/BigClaw&states=Todo,In%20Progress", nil))
	if connectorResponse.Code != http.StatusOK {
		t.Fatalf("expected intake connector 200, got %d %s", connectorResponse.Code, connectorResponse.Body.String())
	}
	var connectorDecoded struct {
		Connector string `json:"connector"`
		Issues    []struct {
			Source string `json:"source"`
			Title  string `json:"title"`
		} `json:"issues"`
		MappedTasks []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"mapped_tasks"`
	}
	if err := json.Unmarshal(connectorResponse.Body.Bytes(), &connectorDecoded); err != nil {
		t.Fatalf("decode intake connector response: %v", err)
	}
	if connectorDecoded.Connector != "github" || len(connectorDecoded.Issues) != 1 || len(connectorDecoded.MappedTasks) != 1 {
		t.Fatalf("unexpected intake connector payload: %+v", connectorDecoded)
	}
	if connectorDecoded.Issues[0].Source != "github" || connectorDecoded.MappedTasks[0].ID == "" || connectorDecoded.MappedTasks[0].State != "queued" {
		t.Fatalf("unexpected mapped issue payload: %+v", connectorDecoded)
	}

	mapBody, _ := json.Marshal(map[string]any{
		"source":      "linear",
		"source_id":   "BIG-102",
		"title":       "Implement prod model",
		"description": "desc",
		"labels":      []string{"p0"},
		"priority":    "P0",
		"state":       "Todo",
		"links":       map[string]string{"issue": "https://linear.app/openagi/issue/BIG-102"},
	})
	mapResponse := httptest.NewRecorder()
	handler.ServeHTTP(mapResponse, httptest.NewRequest(http.MethodPost, "/v2/intake/issues/map", bytes.NewReader(mapBody)))
	if mapResponse.Code != http.StatusOK {
		t.Fatalf("expected intake map 200, got %d %s", mapResponse.Code, mapResponse.Body.String())
	}
	var mappedDecoded struct {
		Task struct {
			ID        string `json:"id"`
			Priority  int    `json:"priority"`
			RiskLevel string `json:"risk_level"`
		} `json:"task"`
	}
	if err := json.Unmarshal(mapResponse.Body.Bytes(), &mappedDecoded); err != nil {
		t.Fatalf("decode intake map response: %v", err)
	}
	if mappedDecoded.Task.ID != "BIG-102" || mappedDecoded.Task.Priority != 0 || mappedDecoded.Task.RiskLevel != "high" {
		t.Fatalf("unexpected mapped task payload: %+v", mappedDecoded.Task)
	}

	renderBody, _ := json.Marshal(map[string]any{
		"definition": map[string]any{
			"name":                  "release-closeout",
			"steps":                 []map[string]any{{"name": "execute", "kind": "scheduler"}},
			"report_path_template":  "reports/{task_id}/{run_id}.md",
			"journal_path_template": "journals/{workflow}/{run_id}.json",
		},
		"task": map[string]any{
			"id":     "BIG-401",
			"source": "linear",
			"title":  "DSL",
		},
		"run_id": "run-1",
	})
	renderResponse := httptest.NewRecorder()
	handler.ServeHTTP(renderResponse, httptest.NewRequest(http.MethodPost, "/v2/workflows/definitions/render", bytes.NewReader(renderBody)))
	if renderResponse.Code != http.StatusOK {
		t.Fatalf("expected workflow definition render 200, got %d %s", renderResponse.Code, renderResponse.Body.String())
	}
	var renderDecoded struct {
		Rendered struct {
			ReportPath  string `json:"report_path"`
			JournalPath string `json:"journal_path"`
		} `json:"rendered"`
	}
	if err := json.Unmarshal(renderResponse.Body.Bytes(), &renderDecoded); err != nil {
		t.Fatalf("decode workflow definition render response: %v", err)
	}
	if renderDecoded.Rendered.ReportPath != "reports/BIG-401/run-1.md" || renderDecoded.Rendered.JournalPath != "journals/release-closeout/run-1.json" {
		t.Fatalf("unexpected rendered workflow paths: %+v", renderDecoded.Rendered)
	}

	recorder = observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	q := queue.NewMemoryQueue()
	runtime := &worker.Runtime{
		WorkerID:    "worker-api-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(workflowAPITestRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	runServer := &Server{
		Recorder: recorder,
		Queue:    q,
		Bus:      bus,
		Worker:   runtime,
		Runtime:  runtime,
		Control:  control.New(),
		Now:      func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}
	runHandler := runServer.Handler()
	tempDir := t.TempDir()
	runBody, _ := json.Marshal(map[string]any{
		"definition": map[string]any{
			"name":                  "release-closeout",
			"steps":                 []map[string]any{{"name": "execute", "kind": "scheduler"}},
			"report_path_template":  filepath.Join(tempDir, "reports", "{task_id}", "{run_id}.md"),
			"journal_path_template": filepath.Join(tempDir, "journals", "{workflow}", "{run_id}.json"),
			"validation_evidence":   []string{"go test ./..."},
		},
		"task": map[string]any{
			"id":                  "BIG-402",
			"title":               "Execute workflow run",
			"acceptance_criteria": []string{"go test ./..."},
			"validation_plan":     []string{"go test ./..."},
		},
		"run_id":              "run-2",
		"validation_evidence": []string{"go test ./..."},
		"git_push_succeeded":  true,
		"git_log_stat_output": " cmd/file.go | 10 +++++-----",
		"repo_sync_audit": map[string]any{
			"sync": map[string]any{
				"status": "synced",
			},
			"pull_request": map[string]any{
				"branch_state": "in-sync",
				"body_state":   "fresh",
			},
		},
	})
	runResponse := httptest.NewRecorder()
	runHandler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodPost, "/v2/workflows/run", bytes.NewReader(runBody)))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected workflow run 200, got %d %s", runResponse.Code, runResponse.Body.String())
	}
	var runDecoded struct {
		Result struct {
			JournalPath string `json:"journal_path"`
			ReportPath  string `json:"report_path"`
			Acceptance  struct {
				Status string `json:"status"`
			} `json:"acceptance"`
			Closeout struct {
				Complete      bool `json:"complete"`
				RepoSyncAudit struct {
					Sync struct {
						Status string `json:"status"`
					} `json:"sync"`
				} `json:"repo_sync_audit"`
			} `json:"closeout"`
			Task struct {
				State string `json:"state"`
			} `json:"task"`
		} `json:"result"`
	}
	if err := json.Unmarshal(runResponse.Body.Bytes(), &runDecoded); err != nil {
		t.Fatalf("decode workflow run response: %v", err)
	}
	if runDecoded.Result.Acceptance.Status != "accepted" || !runDecoded.Result.Closeout.Complete || runDecoded.Result.Task.State != "succeeded" || runDecoded.Result.Closeout.RepoSyncAudit.Sync.Status != "synced" {
		t.Fatalf("unexpected workflow run result: %+v", runDecoded.Result)
	}
	reportContents, err := os.ReadFile(runDecoded.Result.ReportPath)
	if err != nil {
		t.Fatalf("read workflow report: %v", err)
	}
	journalContents, err := os.ReadFile(runDecoded.Result.JournalPath)
	if err != nil {
		t.Fatalf("read workflow journal: %v", err)
	}
	if !strings.Contains(string(reportContents), "Acceptance: accepted") || !strings.Contains(string(journalContents), `"status": "complete"`) {
		t.Fatalf("unexpected workflow artifacts report=%s journal=%s", string(reportContents), string(journalContents))
	}
}

func TestV2DistributedReportBuildsCapacityViewAndMarkdownExport(t *testing.T) {
	base := time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC)
	recorder := observability.NewRecorder()
	controller := control.New()
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		Executors: []domain.ExecutorKind{domain.ExecutorLocal, domain.ExecutorKubernetes, domain.ExecutorRay},
		Control:   controller,
		Worker:    fakeWorkerPoolStatus{},
		Now:       func() time.Time { return base.Add(6 * time.Hour) },
	}
	for _, task := range []domain.Task{
		{ID: "report-local", TraceID: "trace-report-local", Title: "Local", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "apollo"}, UpdatedAt: base.Add(time.Minute)},
		{ID: "report-k8s", TraceID: "trace-report-k8s", Title: "K8s", State: domain.TaskSucceeded, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "apollo"}, UpdatedAt: base.Add(2 * time.Minute)},
		{ID: "report-ray", TraceID: "trace-report-ray", Title: "Ray", State: domain.TaskSucceeded, RequiredTools: []string{"gpu"}, Metadata: map[string]string{"team": "platform", "project": "apollo"}, UpdatedAt: base.Add(3 * time.Minute)},
	} {
		recorder.StoreTask(task)
	}
	for _, event := range []domain.Event{
		{ID: "report-local-routed", Type: domain.EventSchedulerRouted, TaskID: "report-local", TraceID: "trace-report-local", Timestamp: base.Add(time.Second), Payload: map[string]any{"executor": domain.ExecutorLocal, "reason": "default local executor for low/medium risk"}},
		{ID: "report-local-completed", Type: domain.EventTaskCompleted, TaskID: "report-local", TraceID: "trace-report-local", Timestamp: base.Add(2 * time.Second), Payload: map[string]any{"executor": domain.ExecutorLocal}},
		{ID: "report-k8s-routed", Type: domain.EventSchedulerRouted, TaskID: "report-k8s", TraceID: "trace-report-k8s", Timestamp: base.Add(3 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "reason": "browser workloads default to kubernetes executor"}},
		{ID: "report-k8s-started", Type: domain.EventTaskStarted, TaskID: "report-k8s", TraceID: "trace-report-k8s", Timestamp: base.Add(4 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes}},
		{ID: "report-ray-routed", Type: domain.EventSchedulerRouted, TaskID: "report-ray", TraceID: "trace-report-ray", Timestamp: base.Add(5 * time.Second), Payload: map[string]any{"executor": domain.ExecutorRay, "reason": "gpu workloads default to ray executor"}},
		{ID: "report-ray-completed", Type: domain.EventTaskCompleted, TaskID: "report-ray", TraceID: "trace-report-ray", Timestamp: base.Add(6 * time.Second), Payload: map[string]any{"executor": domain.ExecutorRay}},
	} {
		recorder.Record(event)
	}
	controller.Takeover("report-k8s", "alice", "bob", "watch rollout", base.Add(7*time.Second))

	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?team=platform&project=apollo&limit=10", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected distributed report 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Summary struct {
			RegisteredExecutors  int `json:"registered_executors"`
			TotalRoutedDecisions int `json:"total_routed_decisions"`
		} `json:"summary"`
		RoutingReasons []struct {
			Executor string `json:"executor"`
			Reason   string `json:"reason"`
		} `json:"routing_reasons"`
		ExecutorCapacity []struct {
			Executor      string `json:"executor"`
			Health        string `json:"health"`
			QueuedTasks   int    `json:"queued_tasks"`
			ActiveTasks   int    `json:"active_tasks"`
			TeamBreakdown []struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			} `json:"team_breakdown"`
			TopRoutingReasons []struct {
				Reason string `json:"reason"`
				Count  int    `json:"count"`
			} `json:"top_routing_reasons"`
		} `json:"executor_capacity"`
		ClusterHealth struct {
			TeamBreakdown []struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			} `json:"team_breakdown"`
			TakeoverOwners []struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			} `json:"takeover_owners"`
		} `json:"cluster_health"`
		Report struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed report: %v", err)
	}
	if decoded.Summary.RegisteredExecutors != 3 || decoded.Summary.TotalRoutedDecisions != 3 {
		t.Fatalf("unexpected distributed report summary: %+v", decoded.Summary)
	}
	if len(decoded.RoutingReasons) != 3 || len(decoded.ExecutorCapacity) != 3 {
		t.Fatalf("unexpected distributed report payload: %+v", decoded)
	}
	if decoded.ExecutorCapacity[0].Executor != "kubernetes" || decoded.ExecutorCapacity[0].ActiveTasks != 1 || len(decoded.ExecutorCapacity[0].TopRoutingReasons) == 0 {
		t.Fatalf("unexpected executor detail payload: %+v", decoded.ExecutorCapacity[0])
	}
	if len(decoded.ClusterHealth.TeamBreakdown) == 0 || decoded.ClusterHealth.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("unexpected cluster team breakdown: %+v", decoded.ClusterHealth)
	}
	if len(decoded.ClusterHealth.TakeoverOwners) == 0 || decoded.ClusterHealth.TakeoverOwners[0].Key != "alice" {
		t.Fatalf("unexpected takeover owner breakdown: %+v", decoded.ClusterHealth)
	}
	if !strings.Contains(decoded.Report.Markdown, "# BigClaw Distributed Diagnostics Report") || !strings.Contains(decoded.Report.Markdown, "gpu workloads default to ray executor") || !strings.Contains(decoded.Report.Markdown, "Team breakdown") {
		t.Fatalf("unexpected distributed markdown: %s", decoded.Report.Markdown)
	}
	if !strings.Contains(decoded.Report.ExportURL, "/v2/reports/distributed/export") {
		t.Fatalf("unexpected distributed export url: %s", decoded.Report.ExportURL)
	}

	exportResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("expected distributed export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
	}
	if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown export content type, got %q", contentType)
	}
	if !strings.Contains(exportResponse.Body.String(), "Executor Capacity") || !strings.Contains(exportResponse.Body.String(), "ray: gpu workloads default to ray executor") || !strings.Contains(exportResponse.Body.String(), "Takeover owners") {
		t.Fatalf("unexpected distributed export markdown: %s", exportResponse.Body.String())
	}
}
