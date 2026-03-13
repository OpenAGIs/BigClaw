package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/worker"
)

type fakeWorkerStatus struct{}

func (fakeWorkerStatus) Snapshot() worker.Status {
	return worker.Status{WorkerID: "worker-a", State: "idle", SuccessfulRuns: 2, LeaseRenewals: 3, LastResult: "ok"}
}

func TestCreateTaskAndQueryStatus(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		Executors: []domain.ExecutorKind{domain.ExecutorLocal},
		Bus:       bus,
		Now:       func() time.Time { return time.Unix(1700000000, 0) },
	}
	handler := server.Handler()

	payload := map[string]any{"id": "task-api-1", "title": "hello", "required_executor": "local", "entrypoint": "echo hello"}
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/tasks/task-api-1", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusResponse.Code)
	}
	var decoded map[string]any
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if got := decoded["state"]; got != string(domain.TaskQueued) {
		t.Fatalf("expected queued state, got %v", got)
	}
	if got := decoded["trace_id"]; got != "task-api-1" {
		t.Fatalf("expected generated trace_id to match task id, got %v", got)
	}

	eventsRequest := httptest.NewRequest(http.MethodGet, "/events?trace_id=task-api-1&limit=10", nil)
	eventsResponse := httptest.NewRecorder()
	handler.ServeHTTP(eventsResponse, eventsRequest)
	if eventsResponse.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d", eventsResponse.Code)
	}
	if !strings.Contains(eventsResponse.Body.String(), "task-api-1-queued") {
		t.Fatalf("expected queued event via trace lookup, got %s", eventsResponse.Body.String())
	}
}

func TestAuditAndReplayEndpoints(t *testing.T) {
	recorder := observability.NewRecorder()
	recorder.Record(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", Timestamp: time.Now()})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Now: time.Now}
	handler := server.Handler()

	auditRequest := httptest.NewRequest(http.MethodGet, "/audit?limit=10", nil)
	auditResponse := httptest.NewRecorder()
	handler.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d", auditResponse.Code)
	}

	replayRequest := httptest.NewRequest(http.MethodGet, "/replay/task-1", nil)
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusOK {
		t.Fatalf("expected replay 200, got %d", replayResponse.Code)
	}
}

func TestDeadLetterEndpoints(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	q := queue.NewMemoryQueue()
	ctx := context.Background()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-dead", Title: "dead", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	if err := q.DeadLetter(ctx, lease, "boom"); err != nil {
		t.Fatalf("dead letter: %v", err)
	}
	server := &Server{Recorder: recorder, Queue: q, Bus: bus, Now: time.Now}
	handler := server.Handler()

	listRequest := httptest.NewRequest(http.MethodGet, "/deadletters?limit=10", nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected dead letter list 200, got %d", listResponse.Code)
	}
	if !strings.Contains(listResponse.Body.String(), "task-dead") {
		t.Fatalf("expected dead letter task in response, got %s", listResponse.Body.String())
	}

	replayRequest := httptest.NewRequest(http.MethodPost, "/deadletters/task-dead/replay", nil)
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusAccepted {
		t.Fatalf("expected replay 202, got %d", replayResponse.Code)
	}

	listRequest = httptest.NewRequest(http.MethodGet, "/deadletters?limit=10", nil)
	listResponse = httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if strings.Contains(listResponse.Body.String(), "task-dead") {
		t.Fatalf("expected dead letter list to be empty after replay, got %s", listResponse.Body.String())
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/tasks/task-dead", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200 after replay event, got %d", statusResponse.Code)
	}
	if !strings.Contains(statusResponse.Body.String(), string(domain.TaskQueued)) {
		t.Fatalf("expected queued state after replay, got %s", statusResponse.Body.String())
	}
}

func TestStreamEventsEndpoint(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resultCh := make(chan string, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- "ERROR: " + err.Error()
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- "ERROR: " + err.Error()
			return
		}
		resultCh <- strings.TrimSpace(line)
	}()

	time.Sleep(100 * time.Millisecond)
	bus.Publish(domain.Event{ID: "evt-stream-1", Type: domain.EventTaskQueued, TaskID: "task-stream-1", Timestamp: time.Now()})

	select {
	case line := <-resultCh:
		if !strings.HasPrefix(line, "data: ") {
			t.Fatalf("expected sse data line, got %q", line)
		}
		if !strings.Contains(line, "evt-stream-1") {
			t.Fatalf("expected event id in stream, got %q", line)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for streamed event")
	}
}

func TestStreamEventsSupportsReplayAndFiltersByTrace(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	bus.Publish(domain.Event{ID: "evt-old-1", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-old-2", Type: domain.EventTaskQueued, TaskID: "task-b", TraceID: "trace-b", Timestamp: time.Now()})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?replay=1&limit=10&trace_id=trace-a", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resultCh := make(chan string, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- "ERROR: " + err.Error()
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- "ERROR: " + err.Error()
			return
		}
		resultCh <- strings.TrimSpace(line)
	}()

	select {
	case line := <-resultCh:
		if !strings.Contains(line, "evt-old-1") {
			t.Fatalf("expected replayed filtered event, got %q", line)
		}
		if strings.Contains(line, "evt-old-2") {
			t.Fatalf("expected trace filter to exclude evt-old-2, got %q", line)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for replayed event")
	}
}

func TestDebugStatusIncludesWorkerSnapshot(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Worker: fakeWorkerStatus{}, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "worker-a") || !strings.Contains(response.Body.String(), "successful_runs") {
		t.Fatalf("expected worker snapshot in debug payload, got %s", response.Body.String())
	}
}

func TestDebugTraceEndpointsExposeTraceSummary(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Now()
	recorder.Record(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: base})
	recorder.Record(domain.Event{ID: "evt-2", Type: domain.EventTaskCompleted, TaskID: "task-1", TraceID: "trace-1", Timestamp: base.Add(2 * time.Second)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Now: time.Now}

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/debug/traces?limit=10", nil)
	server.Handler().ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected trace list 200, got %d", listResponse.Code)
	}
	if !strings.Contains(listResponse.Body.String(), "trace-1") {
		t.Fatalf("expected trace id in trace list, got %s", listResponse.Body.String())
	}

	detailResponse := httptest.NewRecorder()
	detailRequest := httptest.NewRequest(http.MethodGet, "/debug/traces/trace-1?limit=10", nil)
	server.Handler().ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK {
		t.Fatalf("expected trace detail 200, got %d", detailResponse.Code)
	}
	if !strings.Contains(detailResponse.Body.String(), "duration_seconds") || !strings.Contains(detailResponse.Body.String(), "evt-2") {
		t.Fatalf("expected trace summary and events in detail payload, got %s", detailResponse.Body.String())
	}

	metricsResponse := httptest.NewRecorder()
	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	server.Handler().ServeHTTP(metricsResponse, metricsRequest)
	if !strings.Contains(metricsResponse.Body.String(), "trace_count") {
		t.Fatalf("expected trace_count in metrics payload, got %s", metricsResponse.Body.String())
	}
}

type dashboardResponse struct {
	Summary struct {
		TotalTasks        int            `json:"total_tasks"`
		ActiveRuns        int            `json:"active_runs"`
		Blockers          int            `json:"blockers"`
		PremiumRuns       int            `json:"premium_runs"`
		SLARiskRuns       int            `json:"sla_risk_runs"`
		BudgetCentsTotal  int64          `json:"budget_cents_total"`
		StateDistribution map[string]int `json:"state_distribution"`
	} `json:"summary"`
	TicketToMergeFunnel struct {
		Tickets   int `json:"tickets"`
		PROpened  int `json:"prs_opened"`
		MergedPRs int `json:"merged_prs"`
	} `json:"ticket_to_merge_funnel"`
	ProjectBreakdown []struct {
		Key              string `json:"key"`
		TotalTasks       int    `json:"total_tasks"`
		ActiveRuns       int    `json:"active_runs"`
		Blockers         int    `json:"blockers"`
		BudgetCentsTotal int64  `json:"budget_cents_total"`
		MergedPRs        int    `json:"merged_prs"`
	} `json:"project_breakdown"`
	TeamBreakdown []struct {
		Key              string `json:"key"`
		TotalTasks       int    `json:"total_tasks"`
		ActiveRuns       int    `json:"active_runs"`
		Blockers         int    `json:"blockers"`
		BudgetCentsTotal int64  `json:"budget_cents_total"`
		MergedPRs        int    `json:"merged_prs"`
	} `json:"team_breakdown"`
	Trend []struct {
		Start            time.Time `json:"start"`
		End              time.Time `json:"end"`
		Label            string    `json:"label"`
		TotalTasks       int       `json:"total_tasks"`
		ActiveRuns       int       `json:"active_runs"`
		Blockers         int       `json:"blockers"`
		PremiumRuns      int       `json:"premium_runs"`
		SLARiskRuns      int       `json:"sla_risk_runs"`
		BudgetCentsTotal int64     `json:"budget_cents_total"`
	} `json:"trend"`
	BlockedTasks []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	} `json:"blocked_tasks"`
	HighRiskTasks []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	} `json:"high_risk_tasks"`
	Tasks []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
		Policy struct {
			Plan         string `json:"plan"`
			ApprovalFlow string `json:"approval_flow"`
			ResourcePool string `json:"resource_pool"`
			Quota        struct {
				ConcurrentLimit int   `json:"concurrent_limit"`
				QueueDepthLimit int   `json:"queue_depth_limit"`
				BudgetCapCents  int64 `json:"budget_cap_cents"`
				MaxAgents       int   `json:"max_agents"`
			} `json:"quota"`
		} `json:"policy"`
		Drilldown struct {
			Run               string `json:"run"`
			Events            string `json:"events"`
			Replay            string `json:"replay"`
			IssueKey          string `json:"issue_key"`
			IssueURL          string `json:"issue_url"`
			PullRequestURL    string `json:"pull_request_url"`
			PullRequestStatus string `json:"pull_request_status"`
			Workpad           string `json:"workpad"`
		} `json:"drilldown"`
	} `json:"tasks"`
}

type operationsDashboardResponse struct {
	Summary struct {
		TotalRuns         int            `json:"total_runs"`
		ActiveRuns        int            `json:"active_runs"`
		BlockedRuns       int            `json:"blocked_runs"`
		SLARiskRuns       int            `json:"sla_risk_runs"`
		OverdueRuns       int            `json:"overdue_runs"`
		BudgetCentsTotal  int64          `json:"budget_cents_total"`
		StateDistribution map[string]int `json:"state_distribution"`
		RiskDistribution  map[string]int `json:"risk_distribution"`
	} `json:"summary"`
	ProjectBreakdown []struct {
		Key        string `json:"key"`
		TotalTasks int    `json:"total_tasks"`
	} `json:"project_breakdown"`
	TeamBreakdown []struct {
		Key        string `json:"key"`
		TotalTasks int    `json:"total_tasks"`
	} `json:"team_breakdown"`
	Trend []struct {
		Label       string `json:"label"`
		TotalTasks  int    `json:"total_tasks"`
		Blockers    int    `json:"blockers"`
		SLARiskRuns int    `json:"sla_risk_runs"`
	} `json:"trend"`
	SLARiskTasks []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
		Drilldown struct {
			Run      string `json:"run"`
			IssueKey string `json:"issue_key"`
		} `json:"drilldown"`
	} `json:"sla_risk_tasks"`
	OverdueTasks []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	} `json:"overdue_tasks"`
}

func TestV2DashboardAggregatesEngineeringMetrics(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2023, 11, 14, 10, 0, 0, 0, time.UTC)
	recorder.StoreTask(domain.Task{ID: "task-a", TraceID: "trace-a", Title: "A", State: domain.TaskRunning, BudgetCents: 1200, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"team": "platform", "project": "alpha", "plan": "premium", "pr_status": "merged", "sla_risk": "true", "issue_key": "BIG-801", "issue_url": "https://linear.app/openagis/issue/BIG-801", "pr_url": "https://github.com/OpenAGIs/BigClaw/pull/36", "workpad": "https://docs.example.com/workpads/task-a"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour)})
	recorder.StoreTask(domain.Task{ID: "task-b", TraceID: "trace-b", Title: "B", State: domain.TaskBlocked, BudgetCents: 500, Metadata: map[string]string{"team": "platform", "project": "alpha", "pr_status": "open", "blocked": "true", "issue_key": "BIG-802", "issue_url": "https://linear.app/openagis/issue/BIG-802"}, CreatedAt: base, UpdatedAt: base.Add(time.Hour)})
	recorder.StoreTask(domain.Task{ID: "task-c", TraceID: "trace-c", Title: "C", State: domain.TaskSucceeded, BudgetCents: 300, Metadata: map[string]string{"team": "growth", "project": "beta", "issue_key": "BIG-999"}, CreatedAt: base, UpdatedAt: base.Add(3 * time.Hour)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(4 * time.Hour) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/dashboard/engineering?team=platform&project=alpha&limit=10&bucket=day", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected dashboard 200, got %d", response.Code)
	}
	var decoded dashboardResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	if decoded.Summary.TotalTasks != 2 || decoded.Summary.ActiveRuns != 2 || decoded.Summary.Blockers != 1 {
		t.Fatalf("unexpected dashboard summary: %+v", decoded.Summary)
	}
	if decoded.Summary.PremiumRuns != 1 || decoded.Summary.SLARiskRuns != 1 || decoded.Summary.BudgetCentsTotal != 1700 {
		t.Fatalf("unexpected premium/sla/budget summary: %+v", decoded.Summary)
	}
	if decoded.TicketToMergeFunnel.Tickets != 2 || decoded.TicketToMergeFunnel.PROpened != 2 || decoded.TicketToMergeFunnel.MergedPRs != 1 {
		t.Fatalf("unexpected funnel summary: %+v", decoded.TicketToMergeFunnel)
	}
	if len(decoded.Tasks) != 2 || decoded.Tasks[0].Task.ID != "task-a" || decoded.Tasks[0].Policy.Plan != "premium" {
		t.Fatalf("unexpected dashboard task ordering: %+v", decoded.Tasks)
	}
	if decoded.Tasks[0].Policy.ApprovalFlow != "risk-reviewed" || decoded.Tasks[0].Policy.ResourcePool != "premium/platform" || decoded.Tasks[0].Policy.Quota.ConcurrentLimit != 32 || decoded.Tasks[0].Policy.Quota.MaxAgents != 8 {
		t.Fatalf("expected premium policy boundary details, got %+v", decoded.Tasks[0].Policy)
	}
	if decoded.Tasks[0].Drilldown.Run != "/v2/runs/task-a" || decoded.Tasks[0].Drilldown.IssueKey != "BIG-801" || decoded.Tasks[0].Drilldown.IssueURL == "" || decoded.Tasks[0].Drilldown.PullRequestURL == "" || decoded.Tasks[0].Drilldown.Workpad == "" {
		t.Fatalf("expected drilldown links in dashboard payload, got %+v", decoded.Tasks[0].Drilldown)
	}
	if len(decoded.ProjectBreakdown) != 1 || decoded.ProjectBreakdown[0].Key != "alpha" || decoded.ProjectBreakdown[0].TotalTasks != 2 || decoded.ProjectBreakdown[0].MergedPRs != 1 {
		t.Fatalf("unexpected project breakdown: %+v", decoded.ProjectBreakdown)
	}
	if len(decoded.TeamBreakdown) != 1 || decoded.TeamBreakdown[0].Key != "platform" || decoded.TeamBreakdown[0].TotalTasks != 2 {
		t.Fatalf("unexpected team breakdown: %+v", decoded.TeamBreakdown)
	}
	if len(decoded.BlockedTasks) != 1 || decoded.BlockedTasks[0].Task.ID != "task-b" {
		t.Fatalf("unexpected blocked tasks payload: %+v", decoded.BlockedTasks)
	}
	if len(decoded.HighRiskTasks) != 1 || decoded.HighRiskTasks[0].Task.ID != "task-a" {
		t.Fatalf("unexpected high risk tasks payload: %+v", decoded.HighRiskTasks)
	}
	if len(decoded.Trend) != 1 || decoded.Trend[0].Label != "2023-11-14" || decoded.Trend[0].TotalTasks != 2 || decoded.Trend[0].Blockers != 1 || decoded.Trend[0].PremiumRuns != 1 {
		t.Fatalf("unexpected dashboard trend payload: %+v", decoded.Trend)
	}
}

func TestV2DashboardBuildsHourlyTrendSeries(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2023, 11, 14, 10, 0, 0, 0, time.UTC)
	recorder.StoreTask(domain.Task{ID: "task-hour-1", TraceID: "trace-hour-1", Title: "Hour 1", State: domain.TaskRunning, BudgetCents: 100, Metadata: map[string]string{"team": "platform", "project": "alpha", "plan": "premium"}, CreatedAt: base, UpdatedAt: base.Add(15 * time.Minute)})
	recorder.StoreTask(domain.Task{ID: "task-hour-2", TraceID: "trace-hour-2", Title: "Hour 2", State: domain.TaskBlocked, BudgetCents: 200, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"team": "platform", "project": "alpha", "blocked": "true"}, CreatedAt: base, UpdatedAt: base.Add(75 * time.Minute)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(2 * time.Hour) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/dashboard/engineering?team=platform&project=alpha&since=2023-11-14T10:00:00Z&until=2023-11-14T11:59:00Z&bucket=hour", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected hourly dashboard 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded dashboardResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode hourly dashboard: %v", err)
	}
	if len(decoded.Trend) != 2 {
		t.Fatalf("expected 2 hourly trend points, got %+v", decoded.Trend)
	}
	if decoded.Trend[0].Label != "2023-11-14T10:00:00Z" || decoded.Trend[0].TotalTasks != 1 || decoded.Trend[0].PremiumRuns != 1 {
		t.Fatalf("unexpected first hourly trend point: %+v", decoded.Trend[0])
	}
	if decoded.Trend[1].Label != "2023-11-14T11:00:00Z" || decoded.Trend[1].TotalTasks != 1 || decoded.Trend[1].Blockers != 1 || decoded.Trend[1].SLARiskRuns != 1 {
		t.Fatalf("unexpected second hourly trend point: %+v", decoded.Trend[1])
	}
}

func TestV2OperationsDashboardAggregatesSLAMetrics(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2023, 11, 16, 10, 0, 0, 0, time.UTC)
	recorder.StoreTask(domain.Task{ID: "task-ops-1", TraceID: "trace-ops-1", Title: "Ops A", State: domain.TaskRunning, BudgetCents: 800, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"team": "platform", "project": "alpha", "plan": "premium", "sla_risk": "true", "issue_key": "BIG-901", "issue_url": "https://linear.app/openagis/issue/BIG-901", "sla_due_at": base.Add(-time.Hour).Format(time.RFC3339)}, CreatedAt: base, UpdatedAt: base.Add(30 * time.Minute)})
	recorder.StoreTask(domain.Task{ID: "task-ops-2", TraceID: "trace-ops-2", Title: "Ops B", State: domain.TaskBlocked, BudgetCents: 200, Metadata: map[string]string{"team": "platform", "project": "alpha", "blocked": "true", "issue_key": "BIG-902"}, CreatedAt: base, UpdatedAt: base.Add(40 * time.Minute)})
	recorder.StoreTask(domain.Task{ID: "task-ops-3", TraceID: "trace-ops-3", Title: "Ops C", State: domain.TaskSucceeded, BudgetCents: 100, Metadata: map[string]string{"team": "growth", "project": "beta", "issue_key": "BIG-903", "sla_due_at": base.Add(2 * time.Hour).Format(time.RFC3339)}, CreatedAt: base, UpdatedAt: base.Add(50 * time.Minute)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(2 * time.Hour) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/dashboard/operations?limit=10&bucket=day", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected operations dashboard 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded operationsDashboardResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode operations dashboard: %v", err)
	}
	if decoded.Summary.TotalRuns != 3 || decoded.Summary.ActiveRuns != 2 || decoded.Summary.BlockedRuns != 1 || decoded.Summary.SLARiskRuns != 1 || decoded.Summary.OverdueRuns != 1 || decoded.Summary.BudgetCentsTotal != 1100 {
		t.Fatalf("unexpected operations summary: %+v", decoded.Summary)
	}
	if len(decoded.ProjectBreakdown) != 2 || decoded.ProjectBreakdown[0].Key != "alpha" {
		t.Fatalf("unexpected project breakdown: %+v", decoded.ProjectBreakdown)
	}
	if len(decoded.TeamBreakdown) != 2 || decoded.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("unexpected team breakdown: %+v", decoded.TeamBreakdown)
	}
	if len(decoded.Trend) != 1 || decoded.Trend[0].TotalTasks != 3 || decoded.Trend[0].Blockers != 1 || decoded.Trend[0].SLARiskRuns != 1 {
		t.Fatalf("unexpected operations trend: %+v", decoded.Trend)
	}
	if len(decoded.SLARiskTasks) != 1 || decoded.SLARiskTasks[0].Task.ID != "task-ops-1" || decoded.SLARiskTasks[0].Drilldown.Run != "/v2/runs/task-ops-1" || decoded.SLARiskTasks[0].Drilldown.IssueKey != "BIG-901" {
		t.Fatalf("unexpected sla risk task drilldown payload: %+v", decoded.SLARiskTasks)
	}
	if len(decoded.OverdueTasks) != 1 || decoded.OverdueTasks[0].Task.ID != "task-ops-1" {
		t.Fatalf("unexpected overdue tasks payload: %+v", decoded.OverdueTasks)
	}
}

func TestV2ControlCenterActionsAndRunDetail(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	controller := control.New()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Control: controller, Now: func() time.Time { return time.Unix(1700001000, 0) }}
	handler := server.Handler()

	payload := map[string]any{
		"id":                  "task-v2-1",
		"title":               "Premium run",
		"budget_cents":        900,
		"required_tools":      []string{"browser"},
		"metadata":            map[string]any{"team": "platform", "project": "alpha", "plan": "premium", "workpad": "handoff ready"},
		"acceptance_criteria": []string{"merge PR"},
		"validation_plan":     []string{"run benchmark"},
	}
	body, _ := json.Marshal(payload)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("expected task create 202, got %d", createResponse.Code)
	}

	pauseBody, _ := json.Marshal(map[string]any{"action": "pause", "actor": "ops", "role": "platform_admin", "reason": "maintenance window"})
	pauseResponse := httptest.NewRecorder()
	handler.ServeHTTP(pauseResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(pauseBody)))
	if pauseResponse.Code != http.StatusOK || !strings.Contains(pauseResponse.Body.String(), "maintenance window") {
		t.Fatalf("expected pause action payload, got %d %s", pauseResponse.Code, pauseResponse.Body.String())
	}

	takeoverBody, _ := json.Marshal(map[string]any{"action": "takeover", "task_id": "task-v2-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "reviewer": "bob", "note": "Investigating flaky validation"})
	takeoverResponse := httptest.NewRecorder()
	handler.ServeHTTP(takeoverResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(takeoverBody)))
	if takeoverResponse.Code != http.StatusOK || !strings.Contains(takeoverResponse.Body.String(), "alice") {
		t.Fatalf("expected takeover action payload, got %d %s", takeoverResponse.Code, takeoverResponse.Body.String())
	}

	centerResponse := httptest.NewRecorder()
	handler.ServeHTTP(centerResponse, httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil))
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d", centerResponse.Code)
	}
	if !strings.Contains(centerResponse.Body.String(), "active_takeovers") || !strings.Contains(centerResponse.Body.String(), "task-v2-1") {
		t.Fatalf("expected takeover in control center payload, got %s", centerResponse.Body.String())
	}

	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-v2-1?limit=50", nil))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected run detail 200, got %d", runResponse.Code)
	}
	bodyString := runResponse.Body.String()
	if !strings.Contains(bodyString, "premium") || !strings.Contains(bodyString, "Investigating flaky validation") || !strings.Contains(bodyString, "merge PR") || !strings.Contains(bodyString, "handoff ready") {
		t.Fatalf("expected premium policy, collaboration note, validation, and workpad in run detail, got %s", bodyString)
	}
}

func TestV2RunDetailExposesToolTraceArtifactsAuditAndReport(t *testing.T) {
	recorder := observability.NewRecorder()
	controller := control.New()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Control: controller, Now: func() time.Time { return time.Unix(1700005000, 0) }}
	handler := server.Handler()
	base := time.Unix(1700005000, 0)
	task := domain.Task{
		ID:                 "task-run-report",
		TraceID:            "trace-run-report",
		Title:              "Replay me",
		State:              domain.TaskDeadLetter,
		BudgetCents:        900,
		Priority:           1,
		Labels:             []string{"prod"},
		RequiredTools:      []string{"browser", "git"},
		AcceptanceCriteria: []string{"ship report", "capture artifacts"},
		ValidationPlan:     []string{"replay trace", "download report"},
		Metadata: map[string]string{
			"team":      "platform",
			"project":   "alpha",
			"plan":      "premium",
			"workpad":   "https://docs.example.com/workpads/task-run-report",
			"issue_url": "https://linear.app/openagi/issue/OPE-72/big-804-run-detail-与执行回放页",
			"pr_url":    "https://github.com/OpenAGIs/BigClaw/pull/36",
		},
		CreatedAt: base,
		UpdatedAt: base.Add(3 * time.Second),
	}
	recorder.StoreTask(task)
	recorder.Record(domain.Event{ID: "evt-routed", Type: domain.EventSchedulerRouted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: base.Add(time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "reason": "browser workloads default to kubernetes executor"}})
	recorder.Record(domain.Event{ID: "evt-started", Type: domain.EventTaskStarted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: base.Add(2 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "required_tools": []string{"browser", "git"}}})
	recorder.Record(domain.Event{ID: "evt-dead", Type: domain.EventTaskDeadLetter, TaskID: task.ID, TraceID: task.TraceID, Timestamp: base.Add(3 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "message": "pod crashed during validation", "artifacts": []string{"k8s://jobs/bigclaw/run-report", "k8s://pods/bigclaw/run-report-0"}}})
	controller.Takeover(task.ID, "alice", "bob", "Manual inspection required", base.Add(4*time.Second))
	recorder.Record(domain.Event{ID: "evt-takeover", Type: domain.EventRunTakeover, TaskID: task.ID, TraceID: task.TraceID, Timestamp: base.Add(4 * time.Second), Payload: map[string]any{"actor": "alice", "role": "eng_lead", "reviewer": "bob", "note": "Manual inspection required", "team": "platform", "project": "alpha"}})

	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-report?limit=20", nil))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected run detail 200, got %d %s", runResponse.Code, runResponse.Body.String())
	}
	var decoded struct {
		FailureReason string `json:"failure_reason"`
		Validation    struct {
			Status string `json:"status"`
			Checks int    `json:"checks"`
		} `json:"validation"`
		Risk struct {
			Total            int    `json:"total"`
			Summary          string `json:"summary"`
			RequiresApproval bool   `json:"requires_approval"`
		} `json:"risk_score"`
		Artifacts    map[string]string `json:"artifacts"`
		ArtifactRefs []struct {
			Kind string `json:"kind"`
			URI  string `json:"uri"`
		} `json:"artifact_refs"`
		ToolTraces []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			Executor string `json:"executor"`
		} `json:"tool_traces"`
		AuditSummary struct {
			Total      int `json:"total"`
			NotesCount int `json:"notes_count"`
		} `json:"audit_summary"`
		Reports []struct {
			URL      string `json:"url"`
			Format   string `json:"format"`
			Download bool   `json:"download"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(runResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode run detail: %v", err)
	}
	if decoded.FailureReason != "pod crashed during validation" {
		t.Fatalf("expected failure reason, got %+v", decoded)
	}
	if decoded.Validation.Status != "failed" || decoded.Validation.Checks != 4 {
		t.Fatalf("expected failed validation summary, got %+v", decoded.Validation)
	}
	if decoded.Risk.Total < 40 || decoded.Risk.Summary == "" || decoded.Risk.RequiresApproval {
		t.Fatalf("expected explainable medium risk score, got %+v", decoded.Risk)
	}
	if decoded.Artifacts["report"] != "/v2/runs/task-run-report/report?limit=20" || decoded.Artifacts["audit"] != "/v2/runs/task-run-report/audit?limit=20" || decoded.Artifacts["trace"] == "" {
		t.Fatalf("expected report/audit/trace links, got %+v", decoded.Artifacts)
	}
	if len(decoded.ArtifactRefs) < 4 {
		t.Fatalf("expected artifact refs for executor, workpad, and linked records, got %+v", decoded.ArtifactRefs)
	}
	if len(decoded.ToolTraces) < 4 {
		t.Fatalf("expected tool traces for declared tools and executor events, got %+v", decoded.ToolTraces)
	}
	if decoded.AuditSummary.Total != 1 || decoded.AuditSummary.NotesCount != 1 {
		t.Fatalf("expected audit summary for takeover note, got %+v", decoded.AuditSummary)
	}
	if len(decoded.Reports) != 1 || decoded.Reports[0].Format != "markdown" || !decoded.Reports[0].Download {
		t.Fatalf("expected downloadable markdown report, got %+v", decoded.Reports)
	}

	auditResponse := httptest.NewRecorder()
	handler.ServeHTTP(auditResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-report/audit?limit=20", nil))
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected run audit 200, got %d %s", auditResponse.Code, auditResponse.Body.String())
	}
	if !strings.Contains(auditResponse.Body.String(), "Manual inspection required") || !strings.Contains(auditResponse.Body.String(), "audit_summary") {
		t.Fatalf("expected audit view payload, got %s", auditResponse.Body.String())
	}

	reportResponse := httptest.NewRecorder()
	handler.ServeHTTP(reportResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-report/report?limit=20", nil))
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("expected run report 200, got %d %s", reportResponse.Code, reportResponse.Body.String())
	}
	if contentType := reportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown report content type, got %q", contentType)
	}
	if disposition := reportResponse.Header().Get("Content-Disposition"); !strings.Contains(disposition, "task-run-report-run-report.md") {
		t.Fatalf("expected attachment filename, got %q", disposition)
	}
	for _, want := range []string{"# BigClaw Run Report", "Task ID: task-run-report", "Failure Reason: pod crashed during validation", "k8s://jobs/bigclaw/run-report", "Manual inspection required"} {
		if !strings.Contains(reportResponse.Body.String(), want) {
			t.Fatalf("expected %q in run report, got %s", want, reportResponse.Body.String())
		}
	}
}

func TestV2ControlCenterShowsQueueTasksAndSupportsCancel(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	controller := control.New()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Control: controller, Now: func() time.Time { return time.Unix(1700002000, 0) }}
	handler := server.Handler()

	payload := map[string]any{
		"id":           "task-v2-cancel",
		"title":        "Queued for cancel",
		"priority":     1,
		"budget_cents": 400,
		"metadata":     map[string]any{"team": "platform", "project": "alpha"},
	}
	body, _ := json.Marshal(payload)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("expected task create 202, got %d", createResponse.Code)
	}

	centerResponse := httptest.NewRecorder()
	handler.ServeHTTP(centerResponse, httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil))
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d", centerResponse.Code)
	}
	centerBody := centerResponse.Body.String()
	if !strings.Contains(centerBody, "queue_task") || !strings.Contains(centerBody, "cancellable") || !strings.Contains(centerBody, "task-v2-cancel") || !strings.Contains(centerBody, "drilldown") || !strings.Contains(centerBody, "/v2/runs/task-v2-cancel") {
		t.Fatalf("expected queue task visibility and drilldown in control center, got %s", centerBody)
	}

	cancelBody, _ := json.Marshal(map[string]any{"action": "cancel", "task_id": "task-v2-cancel", "actor": "ops", "role": "platform_admin", "reason": "duplicate request"})
	cancelResponse := httptest.NewRecorder()
	handler.ServeHTTP(cancelResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(cancelBody)))
	if cancelResponse.Code != http.StatusOK || !strings.Contains(cancelResponse.Body.String(), string(domain.TaskCancelled)) {
		t.Fatalf("expected cancel action success, got %d %s", cancelResponse.Code, cancelResponse.Body.String())
	}

	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/tasks/task-v2-cancel", nil))
	if statusResponse.Code != http.StatusOK || !strings.Contains(statusResponse.Body.String(), string(domain.TaskCancelled)) {
		t.Fatalf("expected cancelled task status, got %d %s", statusResponse.Code, statusResponse.Body.String())
	}
}

func TestV2ControlCenterSummariesFiltersAndAudit(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	controller := control.New()
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      bus,
		Control:  controller,
		Worker:   fakeWorkerStatus{},
		Now:      func() time.Time { return time.Unix(1700003000, 0) },
	}
	handler := server.Handler()

	for _, payload := range []map[string]any{
		{
			"id":           "task-control-1",
			"title":        "High risk premium",
			"priority":     1,
			"risk_level":   "high",
			"budget_cents": 600,
			"metadata": map[string]any{
				"team":    "platform",
				"project": "alpha",
				"plan":    "premium",
			},
		},
		{
			"id":           "task-control-2",
			"title":        "Low risk background",
			"priority":     4,
			"risk_level":   "low",
			"budget_cents": 100,
			"metadata": map[string]any{
				"team":    "growth",
				"project": "beta",
			},
		},
	} {
		body, _ := json.Marshal(payload)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
		if response.Code != http.StatusAccepted {
			t.Fatalf("expected task create 202, got %d body=%s", response.Code, response.Body.String())
		}
	}

	takeoverBody, _ := json.Marshal(map[string]any{"action": "transfer_to_human", "task_id": "task-control-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "reviewer": "bob", "note": "Manual validation required"})
	takeoverResponse := httptest.NewRecorder()
	handler.ServeHTTP(takeoverResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(takeoverBody)))
	if takeoverResponse.Code != http.StatusOK {
		t.Fatalf("expected takeover action 200, got %d %s", takeoverResponse.Code, takeoverResponse.Body.String())
	}

	centerResponse := httptest.NewRecorder()
	centerRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center?team=platform&risk_level=high&priority=1&state=blocked&limit=10&audit_limit=10", nil)
	handler.ServeHTTP(centerResponse, centerRequest)
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d", centerResponse.Code)
	}
	centerBody := centerResponse.Body.String()
	for _, want := range []string{"queue_budget_cents_total", "600", "high_risk_runs", "premium_runs", "active_takeovers", "worker_pool", "idle_workers", "task-control-1", "effective_state", "blocked", "Manual validation required", "queue_by_project", "queue_by_team", "alpha", "platform", "recent_actions", "audit_summary", "notes_timeline"} {
		if !strings.Contains(centerBody, want) {
			t.Fatalf("expected %q in control center payload, got %s", want, centerBody)
		}
	}
	if strings.Contains(centerBody, "task-control-2") {
		t.Fatalf("expected filters to exclude task-control-2, got %s", centerBody)
	}

	auditResponse := httptest.NewRecorder()
	auditRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/audit?action=takeover&task_id=task-control-1&actor=alice&audit_limit=10", nil)
	handler.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected control center audit 200, got %d", auditResponse.Code)
	}
	auditBody := auditResponse.Body.String()
	if !strings.Contains(auditBody, "takeover") || !strings.Contains(auditBody, "alice") || !strings.Contains(auditBody, "task-control-1") || !strings.Contains(auditBody, "audit_summary") || !strings.Contains(auditBody, "notes_timeline") || !strings.Contains(auditBody, "platform") || !strings.Contains(auditBody, "alpha") {
		t.Fatalf("expected filtered audit payload with summary facets, got %s", auditBody)
	}
	if strings.Contains(auditBody, "task-control-2") {
		t.Fatalf("expected audit filter to exclude task-control-2, got %s", auditBody)
	}
}

func TestV2ControlCenterAuthorizationEnforcedByRole(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	controller := control.New()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Control: controller, Now: func() time.Time { return time.Unix(1700004000, 0) }}
	handler := server.Handler()

	for _, payload := range []map[string]any{
		{
			"id":       "task-authz-1",
			"title":    "Authz target",
			"priority": 1,
			"metadata": map[string]any{"team": "platform", "project": "alpha"},
		},
		{
			"id":       "task-authz-2",
			"title":    "Other team target",
			"priority": 2,
			"metadata": map[string]any{"team": "growth", "project": "beta"},
		},
	} {
		body, _ := json.Marshal(payload)
		createResponse := httptest.NewRecorder()
		handler.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
		if createResponse.Code != http.StatusAccepted {
			t.Fatalf("expected task create 202, got %d", createResponse.Code)
		}
	}

	centerRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil)
	centerRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	centerRequest.Header.Set("X-BigClaw-Actor", "lead-1")
	centerRequest.Header.Set("X-BigClaw-Team", "platform")
	centerResponse := httptest.NewRecorder()
	handler.ServeHTTP(centerResponse, centerRequest)
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d", centerResponse.Code)
	}
	centerBody := centerResponse.Body.String()
	if !strings.Contains(centerBody, "eng_lead") || !strings.Contains(centerBody, "allowed_actions") || !strings.Contains(centerBody, "takeover") {
		t.Fatalf("expected authorization payload in control center response, got %s", centerBody)
	}
	if strings.Contains(centerBody, "\"cancel\"") {
		t.Fatalf("expected eng_lead authorization to exclude cancel, got %s", centerBody)
	}
	if !strings.Contains(centerBody, "task-authz-1") || strings.Contains(centerBody, "task-authz-2") {
		t.Fatalf("expected control center to be scoped to platform team, got %s", centerBody)
	}

	forbiddenBody, _ := json.Marshal(map[string]any{"action": "cancel", "task_id": "task-authz-1", "actor": "lead-1", "role": "eng_lead", "viewer_team": "platform", "reason": "not allowed"})
	forbiddenResponse := httptest.NewRecorder()
	handler.ServeHTTP(forbiddenResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(forbiddenBody)))
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden cancel for eng_lead, got %d %s", forbiddenResponse.Code, forbiddenResponse.Body.String())
	}

	allowedBody, _ := json.Marshal(map[string]any{"action": "takeover", "task_id": "task-authz-1", "actor": "lead-1", "role": "eng_lead", "viewer_team": "platform", "note": "Escalating review"})
	allowedResponse := httptest.NewRecorder()
	handler.ServeHTTP(allowedResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(allowedBody)))
	if allowedResponse.Code != http.StatusOK {
		t.Fatalf("expected allowed takeover for eng_lead, got %d %s", allowedResponse.Code, allowedResponse.Body.String())
	}

	outsideScopeBody, _ := json.Marshal(map[string]any{"action": "takeover", "task_id": "task-authz-2", "actor": "lead-1", "role": "eng_lead", "viewer_team": "platform", "note": "Should fail outside scope"})
	outsideScopeResponse := httptest.NewRecorder()
	handler.ServeHTTP(outsideScopeResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(outsideScopeBody)))
	if outsideScopeResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden takeover for out-of-scope team, got %d %s", outsideScopeResponse.Code, outsideScopeResponse.Body.String())
	}

	auditResponse := httptest.NewRecorder()
	auditRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/audit?action=takeover&task_id=task-authz-1&audit_limit=10", nil)
	auditRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	auditRequest.Header.Set("X-BigClaw-Actor", "ops-1")
	handler.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected control center audit 200, got %d", auditResponse.Code)
	}
	auditBody := auditResponse.Body.String()
	if !strings.Contains(auditBody, "eng_lead") || !strings.Contains(auditBody, "takeover") || !strings.Contains(auditBody, "task-authz-1") {
		t.Fatalf("expected role-tagged audit payload, got %s", auditBody)
	}
}

func TestV2DashboardAndRunDetailEnforceViewerTeamScope(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Unix(1700005000, 0)
	recorder.StoreTask(domain.Task{ID: "task-scope-1", TraceID: "trace-scope-1", Title: "Scoped", State: domain.TaskRunning, Metadata: map[string]string{"team": "platform", "project": "alpha"}, CreatedAt: base, UpdatedAt: base.Add(time.Hour)})
	recorder.StoreTask(domain.Task{ID: "task-scope-2", TraceID: "trace-scope-2", Title: "Other", State: domain.TaskBlocked, Metadata: map[string]string{"team": "growth", "project": "beta"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(3 * time.Hour) }}
	handler := server.Handler()

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/v2/dashboard/engineering?limit=10", nil)
	dashboardRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	dashboardRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	dashboardRequest.Header.Set("X-BigClaw-Team", "platform")
	dashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardResponse, dashboardRequest)
	if dashboardResponse.Code != http.StatusOK {
		t.Fatalf("expected scoped dashboard 200, got %d %s", dashboardResponse.Code, dashboardResponse.Body.String())
	}
	dashboardBody := dashboardResponse.Body.String()
	if !strings.Contains(dashboardBody, "task-scope-1") || strings.Contains(dashboardBody, "task-scope-2") {
		t.Fatalf("expected dashboard to be scoped to platform team, got %s", dashboardBody)
	}

	operationsRequest := httptest.NewRequest(http.MethodGet, "/v2/dashboard/operations?limit=10", nil)
	operationsRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	operationsRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	operationsRequest.Header.Set("X-BigClaw-Team", "platform")
	operationsResponse := httptest.NewRecorder()
	handler.ServeHTTP(operationsResponse, operationsRequest)
	if operationsResponse.Code != http.StatusOK {
		t.Fatalf("expected scoped operations dashboard 200, got %d %s", operationsResponse.Code, operationsResponse.Body.String())
	}
	operationsBody := operationsResponse.Body.String()
	if !strings.Contains(operationsBody, "task-scope-1") || strings.Contains(operationsBody, "task-scope-2") {
		t.Fatalf("expected operations dashboard to be scoped to platform team, got %s", operationsBody)
	}

	forbiddenDashboardRequest := httptest.NewRequest(http.MethodGet, "/v2/dashboard/engineering?team=growth&limit=10", nil)
	forbiddenDashboardRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	forbiddenDashboardRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	forbiddenDashboardRequest.Header.Set("X-BigClaw-Team", "platform")
	forbiddenDashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(forbiddenDashboardResponse, forbiddenDashboardRequest)
	if forbiddenDashboardResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden dashboard for mismatched team, got %d %s", forbiddenDashboardResponse.Code, forbiddenDashboardResponse.Body.String())
	}

	runRequest := httptest.NewRequest(http.MethodGet, "/v2/runs/task-scope-2", nil)
	runRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	runRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	runRequest.Header.Set("X-BigClaw-Team", "platform")
	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, runRequest)
	if runResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden run detail for out-of-scope team, got %d %s", runResponse.Code, runResponse.Body.String())
	}
}
