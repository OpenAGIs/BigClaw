package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
)

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
		MetricSpec struct {
			Name   string `json:"name"`
			Values []struct {
				MetricID     string `json:"metric_id"`
				DisplayValue string `json:"display_value"`
			} `json:"values"`
		} `json:"metric_spec"`
		Report struct {
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
	if len(decoded.Highlights) == 0 || len(decoded.Actions) == 0 || decoded.MetricSpec.Name != "Operations Metric Spec" || len(decoded.MetricSpec.Values) != 7 || !strings.Contains(decoded.Report.Markdown, "# BigClaw Weekly Ops Report") || !strings.Contains(decoded.Report.ExportURL, "/v2/reports/weekly/export") {
		t.Fatalf("expected highlights/actions/metric spec/export in weekly report, got %+v", decoded)
	}
	if decoded.MetricSpec.Values[0].MetricID == "" || decoded.MetricSpec.Values[0].DisplayValue == "" {
		t.Fatalf("expected populated metric spec values, got %+v", decoded.MetricSpec.Values)
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

	savedViewsResponse := httptest.NewRecorder()
	savedViewsRequest := httptest.NewRequest(http.MethodGet, "/v2/saved-views?team=platform&project=apollo&actor=alice", nil)
	handler.ServeHTTP(savedViewsResponse, savedViewsRequest)
	if savedViewsResponse.Code != http.StatusOK {
		t.Fatalf("expected saved views 200, got %d %s", savedViewsResponse.Code, savedViewsResponse.Body.String())
	}
	var savedViewsDecoded struct {
		Catalog struct {
			Views []struct {
				ViewID string `json:"view_id"`
				Route  string `json:"route"`
			} `json:"views"`
			Subscriptions []struct {
				SavedViewID string   `json:"saved_view_id"`
				Recipients  []string `json:"recipients"`
			} `json:"subscriptions"`
		} `json:"catalog"`
		Audit struct {
			ReadinessScore float64 `json:"readiness_score"`
		} `json:"audit"`
		Report struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(savedViewsResponse.Body.Bytes(), &savedViewsDecoded); err != nil {
		t.Fatalf("decode saved views: %v", err)
	}
	if savedViewsDecoded.Audit.ReadinessScore != 100 || len(savedViewsDecoded.Catalog.Views) < 6 {
		t.Fatalf("unexpected saved views payload: %+v", savedViewsDecoded)
	}
	if !strings.Contains(savedViewsDecoded.Catalog.Views[0].Route, "team=platform") || !strings.Contains(savedViewsDecoded.Report.Markdown, "Saved Views & Alert Digests Report") {
		t.Fatalf("expected scoped routes and report markdown, got %+v", savedViewsDecoded)
	}
	if len(savedViewsDecoded.Catalog.Subscriptions) != 2 || savedViewsDecoded.Catalog.Subscriptions[1].SavedViewID == "" || len(savedViewsDecoded.Catalog.Subscriptions[1].Recipients) == 0 {
		t.Fatalf("expected digest subscriptions, got %+v", savedViewsDecoded.Catalog.Subscriptions)
	}

	savedViewsExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(savedViewsExportResponse, httptest.NewRequest(http.MethodGet, savedViewsDecoded.Report.ExportURL, nil))
	if savedViewsExportResponse.Code != http.StatusOK {
		t.Fatalf("expected saved views export 200, got %d %s", savedViewsExportResponse.Code, savedViewsExportResponse.Body.String())
	}
	if contentType := savedViewsExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown export, got %q", contentType)
	}
	if !strings.Contains(savedViewsExportResponse.Body.String(), "Weekly Ops Review") || !strings.Contains(savedViewsExportResponse.Body.String(), "Readiness Score: 100.0") {
		t.Fatalf("unexpected saved views export body: %s", savedViewsExportResponse.Body.String())
	}

	dashboardRunContractResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardRunContractResponse, httptest.NewRequest(http.MethodGet, "/v2/dashboard-run-contract", nil))
	if dashboardRunContractResponse.Code != http.StatusOK {
		t.Fatalf("expected dashboard run contract 200, got %d %s", dashboardRunContractResponse.Code, dashboardRunContractResponse.Body.String())
	}
	var dashboardRunContractDecoded struct {
		Contract struct {
			ContractID string `json:"contract_id"`
			Version    string `json:"version"`
		} `json:"contract"`
		Audit struct {
			ReleaseReady bool `json:"release_ready"`
		} `json:"audit"`
		Report struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(dashboardRunContractResponse.Body.Bytes(), &dashboardRunContractDecoded); err != nil {
		t.Fatalf("decode dashboard run contract: %v", err)
	}
	if dashboardRunContractDecoded.Contract.ContractID != "BIG-GOM-305" || dashboardRunContractDecoded.Contract.Version != "go-v1" || !dashboardRunContractDecoded.Audit.ReleaseReady {
		t.Fatalf("unexpected dashboard run contract payload: %+v", dashboardRunContractDecoded)
	}
	if !strings.Contains(dashboardRunContractDecoded.Report.Markdown, "\"closeout\"") || dashboardRunContractDecoded.Report.ExportURL != "/v2/dashboard-run-contract/export" {
		t.Fatalf("expected closeout contract report and export url, got %+v", dashboardRunContractDecoded.Report)
	}

	dashboardRunContractExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardRunContractExportResponse, httptest.NewRequest(http.MethodGet, dashboardRunContractDecoded.Report.ExportURL, nil))
	if dashboardRunContractExportResponse.Code != http.StatusOK {
		t.Fatalf("expected dashboard run contract export 200, got %d %s", dashboardRunContractExportResponse.Code, dashboardRunContractExportResponse.Body.String())
	}
	if contentType := dashboardRunContractExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown export, got %q", contentType)
	}
	if !strings.Contains(dashboardRunContractExportResponse.Body.String(), "Dashboard and Run Contract") || !strings.Contains(dashboardRunContractExportResponse.Body.String(), "Release Ready: true") {
		t.Fatalf("unexpected dashboard run contract export body: %s", dashboardRunContractExportResponse.Body.String())
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
		TraceBundle struct {
			TotalTraces             int `json:"total_traces"`
			TracesWithTerminalState int `json:"traces_with_terminal_state"`
			RecentTraces            []struct {
				TraceID  string `json:"trace_id"`
				Executor string `json:"executor"`
				State    string `json:"state"`
				TraceURL string `json:"trace_url"`
				EventURL string `json:"event_url"`
			} `json:"recent_traces"`
			ValidationArtifacts   []string `json:"validation_artifacts"`
			ReviewerNavigation    []string `json:"reviewer_navigation"`
			BackendLimitations    []string `json:"backend_limitations"`
			AmbiguousPublishProof struct {
				Path       string   `json:"path"`
				ScenarioID string   `json:"scenario_id"`
				Outcomes   []string `json:"outcomes"`
			} `json:"ambiguous_publish_proof"`
		} `json:"trace_export_bundle"`
		BrokerReviewPack struct {
			AmbiguousPublishProof struct {
				Path       string   `json:"path"`
				ScenarioID string   `json:"scenario_id"`
				Outcomes   []string `json:"outcomes"`
			} `json:"ambiguous_publish_proof"`
		} `json:"broker_review_pack"`
		PublishAckOutcomes struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				ScenarioID         string `json:"scenario_id"`
				CommittedCount     int    `json:"committed_count"`
				RejectedCount      int    `json:"rejected_count"`
				UnknownCommitCount int    `json:"unknown_commit_count"`
			} `json:"summary"`
		} `json:"publish_ack_outcomes"`
		SequenceBridge struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				BackendCount                 int `json:"backend_count"`
				LiveProvenBackends           int `json:"live_proven_backends"`
				HarnessProvenBackends        int `json:"harness_proven_backends"`
				ContractOnlyBackends         int `json:"contract_only_backends"`
				OneToOneMappings             int `json:"one_to_one_mappings"`
				ProviderEpochBridgedBackends int `json:"provider_epoch_bridged_backends"`
			} `json:"summary"`
		} `json:"sequence_bridge_surface"`
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
		Recovery struct {
			ActiveTakeovers          int `json:"active_takeovers"`
			TakeoverEvents           int `json:"takeover_events"`
			ReleaseEvents            int `json:"release_events"`
			UnreleasedTakeovers      int `json:"unreleased_takeovers"`
			RetriedRuns              int `json:"retried_runs"`
			DeadLetterRuns           int `json:"dead_letter_runs"`
			LeaseExpiredEvents       int `json:"lease_expired_events"`
			CheckpointRejectedEvents int `json:"checkpoint_rejected_events"`
		} `json:"recovery"`
		Fairness struct {
			TotalRoutedDecisions int `json:"total_routed_decisions"`
			CapacityWeightTotal  int `json:"capacity_weight_total"`
			ExecutorShares       []struct {
				Executor        string  `json:"executor"`
				RoutedDecisions int     `json:"routed_decisions"`
				ExpectedShare   float64 `json:"expected_share"`
				ActualShare     float64 `json:"actual_share"`
				ShareDelta      float64 `json:"share_delta"`
				Status          string  `json:"status"`
			} `json:"executor_shares"`
		} `json:"fairness"`
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
	if decoded.Recovery.ActiveTakeovers != 1 || decoded.Recovery.RetriedRuns != 0 || decoded.Recovery.DeadLetterRuns != 0 || decoded.Recovery.UnreleasedTakeovers != 0 {
		t.Fatalf("unexpected recovery diagnostics payload: %+v", decoded.Recovery)
	}
	if decoded.Fairness.TotalRoutedDecisions != 3 || decoded.Fairness.CapacityWeightTotal <= 0 || len(decoded.Fairness.ExecutorShares) != 3 {
		t.Fatalf("unexpected fairness diagnostics payload: %+v", decoded.Fairness)
	}
	if decoded.Fairness.ExecutorShares[0].Executor != "kubernetes" || decoded.Fairness.ExecutorShares[0].RoutedDecisions != 1 {
		t.Fatalf("unexpected fairness executor share payload: %+v", decoded.Fairness.ExecutorShares[0])
	}
	if decoded.TraceBundle.TotalTraces != 3 || decoded.TraceBundle.TracesWithTerminalState != 2 || len(decoded.TraceBundle.RecentTraces) != 3 {
		t.Fatalf("unexpected trace export bundle summary: %+v", decoded.TraceBundle)
	}
	if decoded.TraceBundle.RecentTraces[0].TraceURL == "" || decoded.TraceBundle.RecentTraces[0].EventURL == "" {
		t.Fatalf("expected trace bundle navigation urls, got %+v", decoded.TraceBundle.RecentTraces[0])
	}
	if len(decoded.TraceBundle.ValidationArtifacts) == 0 || len(decoded.TraceBundle.ReviewerNavigation) == 0 || len(decoded.TraceBundle.BackendLimitations) == 0 {
		t.Fatalf("expected trace bundle reviewer metadata, got %+v", decoded.TraceBundle)
	}
	if decoded.TraceBundle.AmbiguousPublishProof.Path != "docs/reports/ambiguous-publish-outcome-proof-summary.json" || decoded.TraceBundle.AmbiguousPublishProof.ScenarioID != "BF-05" || len(decoded.TraceBundle.AmbiguousPublishProof.Outcomes) != 3 {
		t.Fatalf("expected trace bundle ambiguous publish proof reference, got %+v", decoded.TraceBundle.AmbiguousPublishProof)
	}
	if decoded.BrokerReviewPack.AmbiguousPublishProof.Path != "docs/reports/ambiguous-publish-outcome-proof-summary.json" || decoded.BrokerReviewPack.AmbiguousPublishProof.ScenarioID != "BF-05" || len(decoded.BrokerReviewPack.AmbiguousPublishProof.Outcomes) != 3 {
		t.Fatalf("expected broker review pack ambiguous publish proof reference, got %+v", decoded.BrokerReviewPack.AmbiguousPublishProof)
	}
	if decoded.PublishAckOutcomes.ReportPath != publishAckOutcomeSurfacePath || decoded.PublishAckOutcomes.Summary.ScenarioID != "BF-05" || decoded.PublishAckOutcomes.Summary.CommittedCount != 1 || decoded.PublishAckOutcomes.Summary.RejectedCount != 1 || decoded.PublishAckOutcomes.Summary.UnknownCommitCount != 1 {
		t.Fatalf("expected publish ack outcome surface, got %+v", decoded.PublishAckOutcomes)
	}
	if decoded.SequenceBridge.ReportPath != sequenceBridgeSurfacePath || decoded.SequenceBridge.Summary.BackendCount != 5 || decoded.SequenceBridge.Summary.LiveProvenBackends != 3 || decoded.SequenceBridge.Summary.HarnessProvenBackends != 1 || decoded.SequenceBridge.Summary.ContractOnlyBackends != 1 || decoded.SequenceBridge.Summary.OneToOneMappings != 2 || decoded.SequenceBridge.Summary.ProviderEpochBridgedBackends != 3 {
		t.Fatalf("expected sequence bridge surface, got %+v", decoded.SequenceBridge)
	}
	if !strings.Contains(decoded.Report.Markdown, "# BigClaw Distributed Diagnostics Report") || !strings.Contains(decoded.Report.Markdown, "gpu workloads default to ray executor") || !strings.Contains(decoded.Report.Markdown, "Team breakdown") || !strings.Contains(decoded.Report.Markdown, "## Recovery Signals") || !strings.Contains(decoded.Report.Markdown, "## Fairness") || !strings.Contains(decoded.Report.Markdown, "## Trace Export Bundle") || !strings.Contains(decoded.Report.Markdown, "## Durable Sequence Bridge") {
		t.Fatalf("unexpected distributed markdown: %s", decoded.Report.Markdown)
	}
	if !strings.Contains(decoded.Report.Markdown, "## Shared Queue Coordination") || !strings.Contains(decoded.Report.Markdown, "Dead-letter backlog:") {
		t.Fatalf("expected shared queue coordination section in distributed markdown: %s", decoded.Report.Markdown)
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
	if !strings.Contains(exportResponse.Body.String(), "Executor Capacity") || !strings.Contains(exportResponse.Body.String(), "ray: gpu workloads default to ray executor") || !strings.Contains(exportResponse.Body.String(), "Takeover owners") || !strings.Contains(exportResponse.Body.String(), "## Recovery Signals") || !strings.Contains(exportResponse.Body.String(), "## Fairness") || !strings.Contains(exportResponse.Body.String(), "Validation artifacts: docs/reports/live-validation-index.md") || !strings.Contains(exportResponse.Body.String(), "Ambiguous publish proof: docs/reports/ambiguous-publish-outcome-proof-summary.json (BF-05: committed, rejected, unknown_commit)") || !strings.Contains(exportResponse.Body.String(), "Backend limitations: no external tracing backend") {
		t.Fatalf("unexpected distributed export markdown: %s", exportResponse.Body.String())
	}
	if !strings.Contains(exportResponse.Body.String(), "## Shared Queue Coordination") {
		t.Fatalf("expected shared queue coordination in distributed export markdown: %s", exportResponse.Body.String())
	}
	if !strings.Contains(exportResponse.Body.String(), "Lane local: latest_status=succeeded") {
		t.Fatalf("expected continuation lane coverage in distributed export markdown: %s", exportResponse.Body.String())
	}
}

func TestV2DistributedReportAppliesTimeWindowFiltersToResponseAndExport(t *testing.T) {
	base := time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)
	recorder := observability.NewRecorder()
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		Executors: []domain.ExecutorKind{domain.ExecutorLocal, domain.ExecutorKubernetes},
		Control:   control.New(),
		Worker:    fakeNodeAwareWorkerPoolStatus{now: base},
		Now:       func() time.Time { return base },
	}
	for _, task := range []domain.Task{
		{ID: "report-old", TraceID: "trace-report-old", Title: "Old routed task", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "apollo"}, UpdatedAt: base.Add(-2 * time.Hour)},
		{ID: "report-current", TraceID: "trace-report-current", Title: "Current routed task", State: domain.TaskRunning, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "apollo"}, UpdatedAt: base.Add(-10 * time.Minute)},
	} {
		recorder.StoreTask(task)
	}
	for _, event := range []domain.Event{
		{ID: "report-old-routed", Type: domain.EventSchedulerRouted, TaskID: "report-old", TraceID: "trace-report-old", Timestamp: base.Add(-2*time.Hour + time.Minute), Payload: map[string]any{"executor": domain.ExecutorLocal, "reason": "legacy routing path"}},
		{ID: "report-old-completed", Type: domain.EventTaskCompleted, TaskID: "report-old", TraceID: "trace-report-old", Timestamp: base.Add(-2*time.Hour + 2*time.Minute), Payload: map[string]any{"executor": domain.ExecutorLocal}},
		{ID: "report-current-routed", Type: domain.EventSchedulerRouted, TaskID: "report-current", TraceID: "trace-report-current", Timestamp: base.Add(-9 * time.Minute), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "reason": "browser workloads default to kubernetes executor"}},
		{ID: "report-current-started", Type: domain.EventTaskStarted, TaskID: "report-current", TraceID: "trace-report-current", Timestamp: base.Add(-8 * time.Minute), Payload: map[string]any{"executor": domain.ExecutorKubernetes}},
	} {
		recorder.Record(event)
	}

	since := base.Add(-30 * time.Minute)
	requestURL := fmt.Sprintf(
		"/v2/reports/distributed?team=platform&project=apollo&since=%s&until=%s&limit=10",
		url.QueryEscape(since.Format(time.RFC3339)),
		url.QueryEscape(base.Format(time.RFC3339)),
	)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestURL, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected distributed report 200, got %d %s", response.Code, response.Body.String())
	}

	var decoded struct {
		Filters struct {
			Since time.Time `json:"since"`
			Until time.Time `json:"until"`
		} `json:"filters"`
		Summary struct {
			TotalTasks           int `json:"total_tasks"`
			TotalRoutedDecisions int `json:"total_routed_decisions"`
		} `json:"summary"`
		RoutingReasons []struct {
			Reason string `json:"reason"`
		} `json:"routing_reasons"`
		Report struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode time-window distributed report: %v", err)
	}
	if !decoded.Filters.Since.Equal(since) || !decoded.Filters.Until.Equal(base) {
		t.Fatalf("expected distributed report filters to preserve the time window, got %+v", decoded.Filters)
	}
	if decoded.Summary.TotalTasks != 1 || decoded.Summary.TotalRoutedDecisions != 1 {
		t.Fatalf("expected distributed report to only include the current task slice, got %+v", decoded.Summary)
	}
	if len(decoded.RoutingReasons) != 1 || decoded.RoutingReasons[0].Reason != "browser workloads default to kubernetes executor" {
		t.Fatalf("expected old routing evidence to be excluded by the time window, got %+v", decoded.RoutingReasons)
	}
	if !strings.Contains(decoded.Report.Markdown, "since=2026-03-23T10:30:00Z") || !strings.Contains(decoded.Report.Markdown, "until=2026-03-23T11:00:00Z") {
		t.Fatalf("expected markdown filter header to include the time window, got %s", decoded.Report.Markdown)
	}
	if !strings.Contains(decoded.Report.ExportURL, "since=2026-03-23T10%3A30%3A00Z") || !strings.Contains(decoded.Report.ExportURL, "until=2026-03-23T11%3A00%3A00Z") {
		t.Fatalf("expected export url to preserve the time window, got %s", decoded.Report.ExportURL)
	}

	exportResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("expected distributed export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
	}
	if !strings.Contains(exportResponse.Body.String(), "since=2026-03-23T10:30:00Z") || !strings.Contains(exportResponse.Body.String(), "until=2026-03-23T11:00:00Z") {
		t.Fatalf("expected exported markdown to include the retained time window, got %s", exportResponse.Body.String())
	}
	if strings.Contains(exportResponse.Body.String(), "legacy routing path") {
		t.Fatalf("expected exported markdown to exclude out-of-window routing evidence, got %s", exportResponse.Body.String())
	}
}

func TestV2ClawHostExpansionEndpoints(t *testing.T) {
	base := time.Date(2026, 3, 24, 9, 0, 0, 0, time.UTC)
	recorder := observability.NewRecorder()
	for _, task := range []domain.Task{
		{
			ID:        "clawhost-task-1",
			TraceID:   "trace-clawhost-1",
			Title:     "Upgrade platform tenant wave",
			State:     domain.TaskBlocked,
			RiskLevel: domain.RiskHigh,
			TenantID:  "tenant-a",
			Metadata: map[string]string{
				"team":     "platform",
				"project":  "apollo",
				"app":      "alpha-app",
				"provider": "openai",
				"channel":  "slack",
				"device":   "ios",
			},
			CreatedAt: base.Add(-2 * time.Hour),
			UpdatedAt: base.Add(-30 * time.Minute),
		},
		{
			ID:       "clawhost-task-2",
			TraceID:  "trace-clawhost-2",
			Title:    "Restart growth bot ring",
			State:    domain.TaskRunning,
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"team":     "platform",
				"project":  "apollo",
				"app":      "beta-app",
				"provider": "anthropic",
				"channel":  "telegram",
				"device":   "android",
			},
			CreatedAt: base.Add(-90 * time.Minute),
			UpdatedAt: base.Add(-10 * time.Minute),
		},
		{
			ID:       "clawhost-task-3",
			TraceID:  "trace-clawhost-3",
			Title:    "Out-of-scope tenant",
			State:    domain.TaskSucceeded,
			TenantID: "tenant-c",
			Metadata: map[string]string{
				"team":     "growth",
				"project":  "beta",
				"app":      "gamma-app",
				"provider": "minimax",
				"channel":  "discord",
				"device":   "desktop",
			},
			CreatedAt: base.Add(-3 * time.Hour),
			UpdatedAt: base.Add(-5 * time.Minute),
		},
	} {
		recorder.StoreTask(task)
	}

	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      func() time.Time { return base },
	}
	handler := server.Handler()

	t.Run("fleet", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v2/clawhost/fleet", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected fleet endpoint 200, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Inventory struct {
				SurfaceID string `json:"surface_id"`
				Summary   struct {
					AppCount    int `json:"app_count"`
					BotCount    int `json:"bot_count"`
					RunningBots int `json:"running_bots"`
				} `json:"summary"`
			} `json:"inventory"`
			Audit struct {
				ControlPlaneReady bool `json:"control_plane_ready"`
			} `json:"audit"`
			Report struct {
				Markdown  string `json:"markdown"`
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode fleet response: %v", err)
		}
		if decoded.Inventory.SurfaceID != "BIG-PAR-287" || decoded.Inventory.Summary.AppCount != 2 || decoded.Inventory.Summary.BotCount != 2 || decoded.Inventory.Summary.RunningBots != 1 {
			t.Fatalf("unexpected fleet inventory payload: %+v", decoded.Inventory)
		}
		if !decoded.Audit.ControlPlaneReady {
			t.Fatalf("expected fleet audit to be ready, got %+v", decoded.Audit)
		}
		if decoded.Report.ExportURL != "/v2/clawhost/fleet/export" || !strings.Contains(decoded.Report.Markdown, "# ClawHost Fleet Inventory & Control Plane Report") {
			t.Fatalf("unexpected fleet report payload: %+v", decoded.Report)
		}

		exportResponse := httptest.NewRecorder()
		handler.ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
		if exportResponse.Code != http.StatusOK {
			t.Fatalf("expected fleet export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
		}
		if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
			t.Fatalf("expected fleet export markdown content type, got %q", contentType)
		}
		if !strings.Contains(exportResponse.Body.String(), "Control Plane Ready: true") || !strings.Contains(exportResponse.Body.String(), "platform-release-bot") {
			t.Fatalf("unexpected fleet export body: %s", exportResponse.Body.String())
		}
	})

	t.Run("rollout planner", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v2/clawhost/rollout-planner?team=platform&project=apollo", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected rollout planner 200, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Plan struct {
				PlanID  string            `json:"plan_id"`
				Filters map[string]string `json:"filters"`
				Summary struct {
					WaveCount   int `json:"wave_count"`
					TenantCount int `json:"tenant_count"`
					AppCount    int `json:"app_count"`
				} `json:"summary"`
			} `json:"plan"`
			Audit struct {
				ReleaseReady bool `json:"release_ready"`
			} `json:"audit"`
			Report struct {
				Markdown  string `json:"markdown"`
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode rollout planner response: %v", err)
		}
		if decoded.Plan.PlanID != "BIG-PAR-288" || decoded.Plan.Summary.WaveCount != 3 || decoded.Plan.Summary.TenantCount != 2 || decoded.Plan.Summary.AppCount != 2 {
			t.Fatalf("unexpected rollout planner payload: %+v", decoded.Plan)
		}
		if decoded.Plan.Filters["team"] != "platform" || decoded.Plan.Filters["project"] != "apollo" {
			t.Fatalf("expected rollout filters to persist, got %+v", decoded.Plan.Filters)
		}
		if !decoded.Audit.ReleaseReady {
			t.Fatalf("expected rollout planner audit to be ready, got %+v", decoded.Audit)
		}
		exportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse rollout export url: %v", err)
		}
		if exportURL.Path != "/v2/clawhost/rollout-planner/export" || exportURL.Query().Get("team") != "platform" || exportURL.Query().Get("project") != "apollo" {
			t.Fatalf("unexpected rollout export url: %s", decoded.Report.ExportURL)
		}
		if !strings.Contains(decoded.Report.Markdown, "# ClawHost Rollout Planner") || !strings.Contains(decoded.Report.Markdown, "Tenant Ring 1") {
			t.Fatalf("unexpected rollout markdown: %s", decoded.Report.Markdown)
		}

		exportResponse := httptest.NewRecorder()
		handler.ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
		if exportResponse.Code != http.StatusOK {
			t.Fatalf("expected rollout export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
		}
		if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
			t.Fatalf("expected rollout export markdown content type, got %q", contentType)
		}
		if !strings.Contains(exportResponse.Body.String(), "alpha-app") || !strings.Contains(exportResponse.Body.String(), "tenant-a") {
			t.Fatalf("unexpected rollout export body: %s", exportResponse.Body.String())
		}
	})

	t.Run("workflows", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo&actor=alice", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected workflows endpoint 200, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Surface struct {
				Name               string            `json:"name"`
				Filters            map[string]string `json:"filters"`
				OperationalSignals map[string]int    `json:"operational_signals"`
				Summary            struct {
					LaneCount              int `json:"lane_count"`
					TokenSessionGatedLanes int `json:"token_session_gated_lanes"`
				} `json:"summary"`
			} `json:"surface"`
			Audit struct {
				LaneCount int `json:"lane_count"`
			} `json:"audit"`
			Report struct {
				Markdown  string `json:"markdown"`
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode workflows response: %v", err)
		}
		if decoded.Surface.Name != "clawhost-workflow-surface" || decoded.Surface.Filters["team"] != "platform" || decoded.Surface.Filters["project"] != "apollo" || decoded.Surface.Filters["actor"] != "alice" {
			t.Fatalf("unexpected workflow surface identity or filters: %+v", decoded.Surface)
		}
		if decoded.Surface.Summary.LaneCount != 5 || decoded.Surface.Summary.TokenSessionGatedLanes == 0 || decoded.Audit.LaneCount != 5 {
			t.Fatalf("unexpected workflow summary or audit: surface=%+v audit=%+v", decoded.Surface.Summary, decoded.Audit)
		}
		if decoded.Surface.OperationalSignals["total_tasks"] != 2 || decoded.Surface.OperationalSignals["blocked_tasks"] != 1 || decoded.Surface.OperationalSignals["provider_tagged_tasks"] != 2 {
			t.Fatalf("unexpected workflow operational signals: %+v", decoded.Surface.OperationalSignals)
		}
		workflowExportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse workflow export url: %v", err)
		}
		if workflowExportURL.Path != "/v2/clawhost/workflows/export" || workflowExportURL.Query().Get("team") != "platform" || workflowExportURL.Query().Get("project") != "apollo" || workflowExportURL.Query().Get("actor") != "alice" {
			t.Fatalf("unexpected workflow export url: %s", decoded.Report.ExportURL)
		}
		if !strings.Contains(decoded.Report.Markdown, "# ClawHost Workflow Surface") || !strings.Contains(decoded.Report.Markdown, "clawhost-parallel-rollout-control") {
			t.Fatalf("unexpected workflow markdown: %s", decoded.Report.Markdown)
		}

		exportResponse := httptest.NewRecorder()
		handler.ServeHTTP(exportResponse, httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil))
		if exportResponse.Code != http.StatusOK {
			t.Fatalf("expected workflow export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
		}
		if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
			t.Fatalf("expected workflow export markdown content type, got %q", contentType)
		}
		if !strings.Contains(exportResponse.Body.String(), "token_session=true") || !strings.Contains(exportResponse.Body.String(), "IM Channels and Device Approval Workflows") {
			t.Fatalf("unexpected workflow export body: %s", exportResponse.Body.String())
		}
	})

	t.Run("workflows actor header fallback", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo", nil)
		request.Header.Set("X-BigClaw-Actor", "header-actor")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected workflows endpoint 200 with actor header fallback, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Surface struct {
				Filters map[string]string `json:"filters"`
			} `json:"surface"`
			Report struct {
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode workflow header fallback response: %v", err)
		}
		if decoded.Surface.Filters["actor"] != "header-actor" {
			t.Fatalf("expected header actor fallback in workflow filters, got %+v", decoded.Surface.Filters)
		}

		workflowExportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse workflow header fallback export url: %v", err)
		}
		if workflowExportURL.Query().Get("actor") != "header-actor" {
			t.Fatalf("expected workflow export url to preserve header actor fallback, got %s", decoded.Report.ExportURL)
		}

		exportRequest := httptest.NewRequest(http.MethodGet, decoded.Report.ExportURL, nil)
		exportResponse := httptest.NewRecorder()
		handler.ServeHTTP(exportResponse, exportRequest)
		if exportResponse.Code != http.StatusOK {
			t.Fatalf("expected workflow export 200 with header actor fallback, got %d %s", exportResponse.Code, exportResponse.Body.String())
		}
		if !strings.Contains(exportResponse.Body.String(), "owner=header-actor") {
			t.Fatalf("expected workflow export body to include header actor fallback owner, got %s", exportResponse.Body.String())
		}
	})

	t.Run("workflows blank actor query falls back to header", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo&actor=%20%20", nil)
		request.Header.Set("X-BigClaw-Actor", "header-actor")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected workflows endpoint 200 with blank actor query fallback, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Surface struct {
				Filters map[string]string `json:"filters"`
			} `json:"surface"`
			Report struct {
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode blank actor query fallback response: %v", err)
		}
		if decoded.Surface.Filters["actor"] != "header-actor" {
			t.Fatalf("expected blank actor query to fall back to header actor, got %+v", decoded.Surface.Filters)
		}

		workflowExportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse blank actor query fallback export url: %v", err)
		}
		if workflowExportURL.Query().Get("actor") != "header-actor" {
			t.Fatalf("expected export url to preserve header actor after blank query fallback, got %s", decoded.Report.ExportURL)
		}
	})

	t.Run("workflows omit actor from export url when actor is absent", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected workflows endpoint 200 without actor, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Surface struct {
				Filters map[string]string `json:"filters"`
			} `json:"surface"`
			Report struct {
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode no-actor workflow response: %v", err)
		}
		if decoded.Surface.Filters["actor"] != "workflow-operator" {
			t.Fatalf("expected default workflow owner when actor is absent, got %+v", decoded.Surface.Filters)
		}

		workflowExportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse no-actor export url: %v", err)
		}
		if workflowExportURL.Query().Get("actor") != "" {
			t.Fatalf("expected export url to omit actor query when actor is absent, got %s", decoded.Report.ExportURL)
		}
		if workflowExportURL.Query().Get("team") != "platform" || workflowExportURL.Query().Get("project") != "apollo" {
			t.Fatalf("expected export url to preserve non-empty filters, got %s", decoded.Report.ExportURL)
		}
	})

	t.Run("workflows scope normalization", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=%20platform%20&project=%20apollo%20&actor=%20query-actor%20", nil)
		request.Header.Set("X-BigClaw-Actor", "header-actor")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("expected workflows endpoint 200 with normalized scope filters, got %d %s", response.Code, response.Body.String())
		}

		var decoded struct {
			Surface struct {
				Filters map[string]string `json:"filters"`
			} `json:"surface"`
			Report struct {
				ExportURL string `json:"export_url"`
			} `json:"report"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode normalized workflow scope response: %v", err)
		}
		if decoded.Surface.Filters["team"] != "platform" || decoded.Surface.Filters["project"] != "apollo" || decoded.Surface.Filters["actor"] != "query-actor" {
			t.Fatalf("expected normalized workflow filters with query actor precedence, got %+v", decoded.Surface.Filters)
		}

		workflowExportURL, err := url.Parse(decoded.Report.ExportURL)
		if err != nil {
			t.Fatalf("parse normalized workflow export url: %v", err)
		}
		if workflowExportURL.Query().Get("team") != "platform" || workflowExportURL.Query().Get("project") != "apollo" || workflowExportURL.Query().Get("actor") != "query-actor" {
			t.Fatalf("expected normalized export url with query actor precedence, got %s", decoded.Report.ExportURL)
		}
		if strings.Contains(decoded.Report.ExportURL, "%20") {
			t.Fatalf("expected normalized export url without encoded whitespace, got %s", decoded.Report.ExportURL)
		}
	})
}

func TestClawHostReportAndExpansionSurfacesCoexist(t *testing.T) {
	base := time.Date(2026, 3, 24, 9, 0, 0, 0, time.UTC)
	recorder := observability.NewRecorder()
	for _, task := range []domain.Task{
		{
			ID:        "clawhost-coexist-1",
			TraceID:   "trace-clawhost-coexist-1",
			Title:     "Upgrade platform tenant wave",
			State:     domain.TaskBlocked,
			RiskLevel: domain.RiskHigh,
			TenantID:  "tenant-a",
			Metadata: map[string]string{
				"team":     "platform",
				"project":  "apollo",
				"app":      "alpha-app",
				"provider": "openai",
				"channel":  "slack",
				"device":   "ios",
			},
			CreatedAt: base.Add(-2 * time.Hour),
			UpdatedAt: base.Add(-30 * time.Minute),
		},
		{
			ID:       "clawhost-coexist-2",
			TraceID:  "trace-clawhost-coexist-2",
			Title:    "Restart growth bot ring",
			State:    domain.TaskRunning,
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"team":     "platform",
				"project":  "apollo",
				"app":      "beta-app",
				"provider": "anthropic",
				"channel":  "telegram",
				"device":   "android",
			},
			CreatedAt: base.Add(-90 * time.Minute),
			UpdatedAt: base.Add(-10 * time.Minute),
		},
	} {
		recorder.StoreTask(task)
	}

	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      func() time.Time { return base },
	}
	handler := server.Handler()

	debugResponse := httptest.NewRecorder()
	handler.ServeHTTP(debugResponse, httptest.NewRequest(http.MethodGet, "/debug/status", nil))
	if debugResponse.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", debugResponse.Code, debugResponse.Body.String())
	}
	var debugDecoded struct {
		Fleet struct {
			ReportPath string `json:"report_path"`
		} `json:"clawhost_fleet_inventory"`
		Proxy struct {
			ValidationLane string `json:"validation_lane"`
		} `json:"clawhost_proxy_admin_validation"`
	}
	if err := json.Unmarshal(debugResponse.Body.Bytes(), &debugDecoded); err != nil {
		t.Fatalf("decode debug clawhost payloads: %v", err)
	}
	if debugDecoded.Fleet.ReportPath != clawHostFleetInventorySurfacePath || debugDecoded.Proxy.ValidationLane != "clawhost_proxy_admin_parallel_probe" {
		t.Fatalf("unexpected debug clawhost payloads: %+v %+v", debugDecoded.Fleet, debugDecoded.Proxy)
	}

	centerResponse := httptest.NewRecorder()
	handler.ServeHTTP(centerResponse, httptest.NewRequest(http.MethodGet, "/v2/control-center?team=platform&project=apollo&limit=5&audit_limit=5", nil))
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", centerResponse.Code, centerResponse.Body.String())
	}
	var centerDecoded struct {
		Fleet struct {
			ReportPath string `json:"report_path"`
		} `json:"clawhost_fleet_inventory"`
		Proxy struct {
			ReportPath string `json:"report_path"`
		} `json:"clawhost_proxy_admin_validation"`
	}
	if err := json.Unmarshal(centerResponse.Body.Bytes(), &centerDecoded); err != nil {
		t.Fatalf("decode control center clawhost payloads: %v", err)
	}
	if centerDecoded.Fleet.ReportPath != clawHostFleetInventorySurfacePath || centerDecoded.Proxy.ReportPath != clawHostProxyAdminValidationLanePath {
		t.Fatalf("unexpected control center clawhost payloads: %+v %+v", centerDecoded.Fleet, centerDecoded.Proxy)
	}

	distributedResponse := httptest.NewRecorder()
	handler.ServeHTTP(distributedResponse, httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?team=platform&project=apollo&limit=5", nil))
	if distributedResponse.Code != http.StatusOK {
		t.Fatalf("expected distributed report 200, got %d %s", distributedResponse.Code, distributedResponse.Body.String())
	}
	var distributedDecoded struct {
		Rollout struct {
			ReportPath string `json:"report_path"`
		} `json:"clawhost_rollout_planner"`
		Policy struct {
			ReportPath string `json:"report_path"`
		} `json:"clawhost_tenant_policy"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(distributedResponse.Body.Bytes(), &distributedDecoded); err != nil {
		t.Fatalf("decode distributed clawhost payloads: %v", err)
	}
	if distributedDecoded.Rollout.ReportPath != clawHostRolloutPlannerSurfacePath || distributedDecoded.Policy.ReportPath != clawHostTenantPolicySurfacePath {
		t.Fatalf("unexpected distributed clawhost payloads: %+v %+v", distributedDecoded.Rollout, distributedDecoded.Policy)
	}
	if !strings.Contains(distributedDecoded.Report.Markdown, "## ClawHost Proxy and Admin Validation") || !strings.Contains(distributedDecoded.Report.Markdown, "## ClawHost Tenant Policy") {
		t.Fatalf("expected distributed markdown to retain report-backed clawhost sections, got %s", distributedDecoded.Report.Markdown)
	}
	distributedExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(distributedExportResponse, httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=platform&project=apollo&limit=5", nil))
	if distributedExportResponse.Code != http.StatusOK {
		t.Fatalf("expected distributed export 200, got %d %s", distributedExportResponse.Code, distributedExportResponse.Body.String())
	}
	if contentType := distributedExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected distributed export markdown content type, got %q", contentType)
	}
	if !strings.Contains(distributedExportResponse.Body.String(), "## ClawHost Proxy and Admin Validation") || !strings.Contains(distributedExportResponse.Body.String(), "## ClawHost Rollout Planner") {
		t.Fatalf("expected distributed export to retain report-backed clawhost sections, got %s", distributedExportResponse.Body.String())
	}

	fleetResponse := httptest.NewRecorder()
	handler.ServeHTTP(fleetResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/fleet", nil))
	if fleetResponse.Code != http.StatusOK {
		t.Fatalf("expected fleet expansion 200, got %d %s", fleetResponse.Code, fleetResponse.Body.String())
	}
	var fleetDecoded struct {
		Inventory struct {
			SurfaceID string `json:"surface_id"`
		} `json:"inventory"`
	}
	if err := json.Unmarshal(fleetResponse.Body.Bytes(), &fleetDecoded); err != nil {
		t.Fatalf("decode fleet expansion response: %v", err)
	}
	if fleetDecoded.Inventory.SurfaceID != "BIG-PAR-287" {
		t.Fatalf("unexpected fleet expansion surface id: %+v", fleetDecoded.Inventory)
	}
	fleetExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(fleetExportResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/fleet/export", nil))
	if fleetExportResponse.Code != http.StatusOK {
		t.Fatalf("expected fleet expansion export 200, got %d %s", fleetExportResponse.Code, fleetExportResponse.Body.String())
	}
	if contentType := fleetExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected fleet expansion export markdown content type, got %q", contentType)
	}
	if !strings.Contains(fleetExportResponse.Body.String(), "# ClawHost Fleet Inventory & Control Plane Report") || !strings.Contains(fleetExportResponse.Body.String(), "platform-release-bot") {
		t.Fatalf("unexpected fleet expansion export body: %s", fleetExportResponse.Body.String())
	}

	rolloutResponse := httptest.NewRecorder()
	handler.ServeHTTP(rolloutResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/rollout-planner?team=platform&project=apollo", nil))
	if rolloutResponse.Code != http.StatusOK {
		t.Fatalf("expected rollout planner expansion 200, got %d %s", rolloutResponse.Code, rolloutResponse.Body.String())
	}
	var rolloutDecoded struct {
		Plan struct {
			PlanID string `json:"plan_id"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(rolloutResponse.Body.Bytes(), &rolloutDecoded); err != nil {
		t.Fatalf("decode rollout planner expansion response: %v", err)
	}
	if rolloutDecoded.Plan.PlanID != "BIG-PAR-288" {
		t.Fatalf("unexpected rollout planner expansion payload: %+v", rolloutDecoded.Plan)
	}
	rolloutExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(rolloutExportResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/rollout-planner/export?team=platform&project=apollo", nil))
	if rolloutExportResponse.Code != http.StatusOK {
		t.Fatalf("expected rollout planner expansion export 200, got %d %s", rolloutExportResponse.Code, rolloutExportResponse.Body.String())
	}
	if contentType := rolloutExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected rollout planner expansion export markdown content type, got %q", contentType)
	}
	if !strings.Contains(rolloutExportResponse.Body.String(), "# ClawHost Rollout Planner") || !strings.Contains(rolloutExportResponse.Body.String(), "Tenant Ring 1") {
		t.Fatalf("unexpected rollout planner expansion export body: %s", rolloutExportResponse.Body.String())
	}

	workflowsResponse := httptest.NewRecorder()
	handler.ServeHTTP(workflowsResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo&actor=alice", nil))
	if workflowsResponse.Code != http.StatusOK {
		t.Fatalf("expected workflows expansion 200, got %d %s", workflowsResponse.Code, workflowsResponse.Body.String())
	}
	var workflowsDecoded struct {
		Surface struct {
			Name string `json:"name"`
		} `json:"surface"`
	}
	if err := json.Unmarshal(workflowsResponse.Body.Bytes(), &workflowsDecoded); err != nil {
		t.Fatalf("decode workflows expansion response: %v", err)
	}
	if workflowsDecoded.Surface.Name != "clawhost-workflow-surface" {
		t.Fatalf("unexpected workflows expansion payload: %+v", workflowsDecoded.Surface)
	}
	workflowsExportResponse := httptest.NewRecorder()
	handler.ServeHTTP(workflowsExportResponse, httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows/export?team=platform&project=apollo&actor=alice", nil))
	if workflowsExportResponse.Code != http.StatusOK {
		t.Fatalf("expected workflows expansion export 200, got %d %s", workflowsExportResponse.Code, workflowsExportResponse.Body.String())
	}
	if contentType := workflowsExportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected workflows expansion export markdown content type, got %q", contentType)
	}
	if !strings.Contains(workflowsExportResponse.Body.String(), "# ClawHost Workflow Surface") || !strings.Contains(workflowsExportResponse.Body.String(), "IM Channels and Device Approval Workflows") {
		t.Fatalf("unexpected workflows expansion export body: %s", workflowsExportResponse.Body.String())
	}
}

func TestV2ClawHostEndpointsRejectNonGETMethods(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	handler := server.Handler()

	for _, path := range []string{
		"/v2/clawhost/fleet",
		"/v2/clawhost/fleet/export",
		"/v2/clawhost/rollout-planner?team=platform&project=apollo",
		"/v2/clawhost/rollout-planner/export?team=platform&project=apollo",
		"/v2/clawhost/workflows?team=platform&project=apollo&actor=alice",
		"/v2/clawhost/workflows/export?team=platform&project=apollo&actor=alice",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected %s to reject POST with 405, got %d %s", path, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "method not allowed") {
			t.Fatalf("expected %s to return method-not-allowed body, got %q", path, response.Body.String())
		}
	}
}

func TestV2ClawHostExportEndpointsSetAttachmentFilenames(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	handler := server.Handler()

	for _, tc := range []struct {
		path     string
		filename string
	}{
		{path: "/v2/clawhost/fleet/export", filename: `attachment; filename="clawhost-fleet.md"`},
		{path: "/v2/clawhost/rollout-planner/export?team=platform&project=apollo", filename: `attachment; filename="clawhost-rollout-planner.md"`},
		{path: "/v2/clawhost/workflows/export?team=platform&project=apollo&actor=alice", filename: `attachment; filename="clawhost-workflows.md"`},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d %s", tc.path, response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
			t.Fatalf("expected %s to return markdown content type, got %q", tc.path, contentType)
		}
		if contentDisposition := response.Header().Get("Content-Disposition"); contentDisposition != tc.filename {
			t.Fatalf("expected %s to return %q, got %q", tc.path, tc.filename, contentDisposition)
		}
	}
}

func TestClawHostScopeFilters(t *testing.T) {
	t.Run("trims query values and prefers actor query over header", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=%20platform%20&project=%20apollo%20&actor=%20query-actor%20", nil)
		request.Header.Set("X-BigClaw-Actor", "header-actor")

		team, project, actor := clawHostScopeFilters(request)
		if team != "platform" || project != "apollo" || actor != "query-actor" {
			t.Fatalf("unexpected normalized scope filters: team=%q project=%q actor=%q", team, project, actor)
		}
	})

	t.Run("falls back to trimmed actor header", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v2/clawhost/workflows?team=platform&project=apollo", nil)
		request.Header.Set("X-BigClaw-Actor", " header-actor ")

		team, project, actor := clawHostScopeFilters(request)
		if team != "platform" || project != "apollo" || actor != "header-actor" {
			t.Fatalf("unexpected header fallback scope filters: team=%q project=%q actor=%q", team, project, actor)
		}
	})
}

func TestClawHostExportURL(t *testing.T) {
	t.Run("omits empty filters", func(t *testing.T) {
		if url := clawHostExportURL("/v2/clawhost/workflows/export", "", "", ""); url != "/v2/clawhost/workflows/export" {
			t.Fatalf("expected export url without query string when filters are empty, got %s", url)
		}
	})

	t.Run("normalizes filters and preserves query actor precedence output", func(t *testing.T) {
		exportURL := clawHostExportURL("/v2/clawhost/workflows/export", " platform ", " apollo ", " query-actor ")
		parsed, err := url.Parse(exportURL)
		if err != nil {
			t.Fatalf("parse normalized export url: %v", err)
		}
		if parsed.Query().Get("team") != "platform" || parsed.Query().Get("project") != "apollo" || parsed.Query().Get("actor") != "query-actor" {
			t.Fatalf("unexpected normalized export query: %s", exportURL)
		}
		if strings.Contains(exportURL, "%20") {
			t.Fatalf("expected export url without encoded whitespace, got %s", exportURL)
		}
	})

	t.Run("omits blank team and project while preserving actor", func(t *testing.T) {
		exportURL := clawHostExportURL("/v2/clawhost/workflows/export", "   ", "", " actor-only ")
		parsed, err := url.Parse(exportURL)
		if err != nil {
			t.Fatalf("parse actor-only export url: %v", err)
		}
		if parsed.Query().Get("actor") != "actor-only" {
			t.Fatalf("expected actor-only export url to preserve actor, got %s", exportURL)
		}
		if parsed.Query().Get("team") != "" || parsed.Query().Get("project") != "" {
			t.Fatalf("expected actor-only export url to omit blank team/project, got %s", exportURL)
		}
	})
}
