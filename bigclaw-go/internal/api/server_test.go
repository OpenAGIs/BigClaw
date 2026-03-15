package api

import (
	"bufio"
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
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type fakeWorkerStatus struct{}

func (fakeWorkerStatus) Snapshot() worker.Status {
	return worker.Status{WorkerID: "worker-a", State: "idle", SuccessfulRuns: 2, LeaseRenewals: 3, LastResult: "ok"}
}

type fakeWorkerPoolStatus struct{}

func (fakeWorkerPoolStatus) Snapshot() worker.Status {
	return worker.Status{WorkerID: "worker-a", State: "running", CurrentExecutor: domain.ExecutorLocal, SuccessfulRuns: 5, LeaseRenewals: 7, LastResult: "ok"}
}

func (fakeWorkerPoolStatus) Snapshots() []worker.Status {
	return []worker.Status{
		{WorkerID: "worker-a", State: "running", CurrentExecutor: domain.ExecutorLocal, SuccessfulRuns: 5, LeaseRenewals: 7, LastResult: "ok"},
		{WorkerID: "worker-b", State: "leased", CurrentExecutor: domain.ExecutorKubernetes, SuccessfulRuns: 3, LeaseRenewals: 2, LastResult: "warming", PreemptionActive: true, CurrentPreemptionTaskID: "task-low", CurrentPreemptionWorkerID: "worker-low", LastPreemptedTaskID: "task-low", LastPreemptionAt: time.Unix(1700000100, 0), LastPreemptionReason: "preempted by urgent task task-urgent (priority=1)", PreemptionsIssued: 1},
		{WorkerID: "worker-c", State: "idle", SuccessfulRuns: 8, LeaseRenewals: 0, LastResult: "idle"},
	}
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

func TestDebugStatusIncludesWorkerPoolSummary(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Worker: fakeWorkerPoolStatus{}, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"worker_pool", "total_workers", "3", "active_workers", "2", "idle_workers", "1", "worker-b", "leased", "preemption_active", "last_preempted_task_id", "task-low"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in debug payload, got %s", want, body)
		}
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

func TestMetricsSupportsPrometheusFormat(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Now()
	recorder.Record(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: base})
	controller := control.New()
	controller.Pause("ops", "maintenance", base)
	controller.Takeover("task-1", "alice", "bob", "investigating", base.Add(time.Second))
	workQueue := queue.NewMemoryQueue()
	if err := workQueue.Enqueue(context.Background(), domain.Task{ID: "queued-1", TraceID: "trace-queue", Title: "queued-1"}); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	server := &Server{
		Recorder:  recorder,
		Queue:     workQueue,
		Bus:       events.NewBus(),
		Worker:    fakeWorkerPoolStatus{},
		Control:   controller,
		Executors: []domain.ExecutorKind{domain.ExecutorLocal, domain.ExecutorKubernetes},
		Now:       time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics?format=prometheus", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected prometheus metrics 200, got %d", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", contentType)
	}
	body := response.Body.String()
	checks := []string{
		"# HELP bigclaw_queue_size Current queue size.",
		"bigclaw_queue_size 1",
		"bigclaw_trace_count 1",
		"bigclaw_events_total{event_type=\"task.queued\"} 1",
		"bigclaw_executor_registered{executor=\"kubernetes\"} 1",
		"bigclaw_worker_pool_total 3",
		"bigclaw_worker_pool_active 2",
		"bigclaw_worker_pool_idle 1",
		"bigclaw_control_paused 1",
		"bigclaw_control_active_takeovers 1",
		"bigclaw_worker_status{current_executor=\"kubernetes\",state=\"leased\",worker_id=\"worker-b\"} 1",
		"bigclaw_worker_successful_runs_total{worker_id=\"worker-a\"} 5",
		"bigclaw_worker_lease_renewals_total{worker_id=\"worker-b\"} 2",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected %q in prometheus body, got %s", check, body)
		}
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

type triageCenterResponse struct {
	Summary struct {
		FlaggedRuns    int            `json:"flagged_runs"`
		InboxSize      int            `json:"inbox_size"`
		Recommendation string         `json:"recommendation"`
		SeverityCounts map[string]int `json:"severity_counts"`
		OwnerCounts    map[string]int `json:"owner_counts"`
	} `json:"summary"`
	Findings []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
		Policy struct {
			ApprovalFlow string `json:"approval_flow"`
		} `json:"policy"`
		Risk struct {
			Total            int    `json:"total"`
			RequiresApproval bool   `json:"requires_approval"`
			Summary          string `json:"summary"`
		} `json:"risk_score"`
		Severity          string  `json:"severity"`
		Owner             string  `json:"owner"`
		Reason            string  `json:"reason"`
		NextAction        string  `json:"next_action"`
		SuggestedWorkflow string  `json:"suggested_workflow"`
		SuggestedPriority string  `json:"suggested_priority"`
		SuggestedOwner    string  `json:"suggested_owner"`
		Confidence        float64 `json:"confidence"`
		Drilldown         struct {
			Run string `json:"run"`
		} `json:"drilldown"`
		SimilarCases []struct {
			TaskID string  `json:"task_id"`
			Score  float64 `json:"score"`
		} `json:"similar_cases"`
	} `json:"findings"`
	Clusters []struct {
		Reason   string `json:"reason"`
		Count    int    `json:"count"`
		Workflow string `json:"workflow"`
	} `json:"clusters"`
}

type regressionCenterResponse struct {
	Authorization struct {
		ViewerTeam string `json:"viewer_team"`
	} `json:"authorization"`
	Filters struct {
		Team       string `json:"team"`
		Project    string `json:"project"`
		ViewerTeam string `json:"viewer_team"`
		Limit      int    `json:"limit"`
		Bucket     string `json:"bucket"`
	} `json:"filters"`
	Summary struct {
		TotalRegressions    int    `json:"total_regressions"`
		AffectedTasks       int    `json:"affected_tasks"`
		CriticalRegressions int    `json:"critical_regressions"`
		ReworkEvents        int    `json:"rework_events"`
		TopSource           string `json:"top_source"`
		TopWorkflow         string `json:"top_workflow"`
	} `json:"summary"`
	CompareSummary struct {
		Current struct {
			TotalRegressions    int `json:"total_regressions"`
			AffectedTasks       int `json:"affected_tasks"`
			CriticalRegressions int `json:"critical_regressions"`
			ReworkEvents        int `json:"rework_events"`
		} `json:"current"`
		Baseline struct {
			TotalRegressions    int `json:"total_regressions"`
			AffectedTasks       int `json:"affected_tasks"`
			CriticalRegressions int `json:"critical_regressions"`
			ReworkEvents        int `json:"rework_events"`
		} `json:"baseline"`
		DeltaRegressions         int `json:"delta_regressions"`
		DeltaAffectedTasks       int `json:"delta_affected_tasks"`
		DeltaCriticalRegressions int `json:"delta_critical_regressions"`
		DeltaReworkEvents        int `json:"delta_rework_events"`
	} `json:"compare_summary"`
	WorkflowBreakdown []struct {
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"workflow_breakdown"`
	TeamBreakdown []struct {
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"team_breakdown"`
	TemplateBreakdown []struct {
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"template_breakdown"`
	ServiceBreakdown []struct {
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"service_breakdown"`
	AttributionBreakdown []struct {
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"attribution_breakdown"`
	Hotspots []struct {
		Dimension        string `json:"dimension"`
		Key              string `json:"key"`
		TotalRegressions int    `json:"total_regressions"`
	} `json:"hotspots"`
	Trend []struct {
		Label               string `json:"label"`
		TotalRegressions    int    `json:"total_regressions"`
		AffectedTasks       int    `json:"affected_tasks"`
		CriticalRegressions int    `json:"critical_regressions"`
		ReworkEvents        int    `json:"rework_events"`
	} `json:"trend"`
	Findings []struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
		Policy struct {
			Plan         string `json:"plan"`
			ApprovalFlow string `json:"approval_flow"`
		} `json:"policy"`
		Risk struct {
			RequiresApproval bool `json:"requires_approval"`
		} `json:"risk_score"`
		Workflow        string `json:"workflow"`
		Team            string `json:"team"`
		Template        string `json:"template"`
		Service         string `json:"service"`
		Severity        string `json:"severity"`
		RegressionCount int    `json:"regression_count"`
		ReworkEvents    int    `json:"rework_events"`
		Attribution     string `json:"attribution"`
		Summary         string `json:"summary"`
		Drilldown       struct {
			Run      string `json:"run"`
			Events   string `json:"events"`
			Replay   string `json:"replay"`
			IssueKey string `json:"issue_key"`
		} `json:"drilldown"`
	} `json:"findings"`
}

type controlCenterAuditResponse struct {
	Filters struct {
		TaskID   string `json:"task_id"`
		Team     string `json:"team"`
		Action   string `json:"action"`
		Actor    string `json:"actor"`
		Owner    string `json:"owner"`
		Reviewer string `json:"reviewer"`
		Scope    string `json:"scope"`
		Limit    int    `json:"limit"`
	} `json:"filters"`
	AuditSummary struct {
		Total      int `json:"total"`
		NotesCount int `json:"notes_count"`
		ByScope    []struct {
			Key   string `json:"key"`
			Count int    `json:"count"`
		} `json:"by_scope"`
		ByOwner []struct {
			Key   string `json:"key"`
			Count int    `json:"count"`
		} `json:"by_owner"`
		ByReviewer []struct {
			Key   string `json:"key"`
			Count int    `json:"count"`
		} `json:"by_reviewer"`
	} `json:"audit_summary"`
	Audit []struct {
		OperationID      string `json:"operation_id"`
		Action           string `json:"action"`
		Scope            string `json:"scope"`
		TaskID           string `json:"task_id"`
		TaskStateBefore  string `json:"task_state_before"`
		TaskStateAfter   string `json:"task_state_after"`
		PreviousOwner    string `json:"previous_owner"`
		Owner            string `json:"owner"`
		PreviousReviewer string `json:"previous_reviewer"`
		Reviewer         string `json:"reviewer"`
		Note             string `json:"note"`
	} `json:"audit"`
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

func TestV2TriageCenterBuildsRecommendationsAndSimilarity(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Unix(1700002800, 0)
	failed := domain.Task{ID: "task-triage-browser", TraceID: "run-browser", Title: "Browser replay failure", State: domain.TaskDeadLetter, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "alpha", "source": "linear"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Minute)}
	riskReview := domain.Task{ID: "task-triage-security", TraceID: "run-security", Title: "Security approval", State: domain.TaskBlocked, Priority: 1, Labels: []string{"security", "prod"}, RequiredTools: []string{"deploy"}, Metadata: map[string]string{"team": "platform", "project": "alpha", "source": "linear"}, CreatedAt: base, UpdatedAt: base.Add(3 * time.Minute)}
	similar := domain.Task{ID: "task-triage-similar", TraceID: "run-browser-2", Title: "Browser replay failure", State: domain.TaskSucceeded, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "alpha", "source": "linear"}, CreatedAt: base, UpdatedAt: base.Add(4 * time.Minute)}
	healthy := domain.Task{ID: "task-triage-healthy", TraceID: "run-healthy", Title: "Healthy run", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha", "source": "linear"}, CreatedAt: base, UpdatedAt: base.Add(5 * time.Minute)}
	for _, task := range []domain.Task{failed, riskReview, similar, healthy} {
		recorder.StoreTask(task)
	}
	recorder.Record(domain.Event{ID: "evt-browser-dead", Type: domain.EventTaskDeadLetter, TaskID: failed.ID, TraceID: failed.TraceID, Timestamp: base.Add(2 * time.Minute), Payload: map[string]any{"message": "browser session crashed"}})
	recorder.Record(domain.Event{ID: "evt-security-blocked", Type: domain.EventRunTakeover, TaskID: riskReview.ID, TraceID: riskReview.TraceID, Timestamp: base.Add(3 * time.Minute), Payload: map[string]any{"reason": "requires approval for high-risk task"}})
	recorder.Record(domain.Event{ID: "evt-browser-similar", Type: domain.EventTaskCompleted, TaskID: similar.ID, TraceID: similar.TraceID, Timestamp: base.Add(4 * time.Minute), Payload: map[string]any{"message": "browser session crashed"}})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(6 * time.Minute) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/triage/center?team=platform&limit=10", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected triage center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded triageCenterResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode triage center: %v", err)
	}
	if decoded.Summary.FlaggedRuns != 2 || decoded.Summary.InboxSize != 2 || decoded.Summary.Recommendation != "immediate-attention" {
		t.Fatalf("unexpected triage summary: %+v", decoded.Summary)
	}
	if decoded.Summary.SeverityCounts["critical"] != 1 || decoded.Summary.SeverityCounts["high"] != 1 {
		t.Fatalf("unexpected triage severity counts: %+v", decoded.Summary.SeverityCounts)
	}
	if len(decoded.Findings) != 2 || decoded.Findings[0].Task.ID != "task-triage-browser" || decoded.Findings[1].Task.ID != "task-triage-security" {
		t.Fatalf("unexpected triage ordering: %+v", decoded.Findings)
	}
	if decoded.Findings[0].Owner != "engineering" || decoded.Findings[0].SuggestedWorkflow != "run-replay" || decoded.Findings[0].SuggestedPriority != "P0" || decoded.Findings[0].Drilldown.Run != "/v2/runs/task-triage-browser" {
		t.Fatalf("expected browser triage recommendation, got %+v", decoded.Findings[0])
	}
	if len(decoded.Findings[0].SimilarCases) == 0 || decoded.Findings[0].SimilarCases[0].TaskID != "task-triage-similar" {
		t.Fatalf("expected similarity evidence in triage payload, got %+v", decoded.Findings[0])
	}
	if decoded.Findings[1].Owner != "security" || decoded.Findings[1].SuggestedWorkflow != "security-review" || decoded.Findings[1].SuggestedPriority != "P1" || !decoded.Findings[1].Risk.RequiresApproval || decoded.Findings[1].Policy.ApprovalFlow != "risk-reviewed" {
		t.Fatalf("expected risk review triage recommendation, got %+v", decoded.Findings[1])
	}
	if len(decoded.Clusters) < 2 {
		t.Fatalf("expected clustered triage reasons, got %+v", decoded.Clusters)
	}
}

func TestV2RegressionCenterBuildsBreakdownsTrendAndCompareSummary(t *testing.T) {
	recorder := observability.NewRecorder()
	base := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	baseline := domain.Task{ID: "task-reg-baseline", TraceID: "trace-reg-baseline", Title: "Baseline regression", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha", "workflow": "deploy", "template": "release", "service": "api", "regression_count": "1", "regression_source": "legacy baseline", "issue_key": "BIG-899"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour)}
	currentCritical := domain.Task{ID: "task-reg-current-1", TraceID: "trace-reg-current-1", Title: "Deploy regression", State: domain.TaskDeadLetter, Priority: 1, Labels: []string{"regression", "prod"}, RequiredTools: []string{"deploy"}, Metadata: map[string]string{"team": "platform", "project": "alpha", "workflow": "deploy", "template": "release", "service": "api", "plan": "premium", "issue_key": "BIG-904", "regression_count": "2", "regression_source": "security scan failed", "code_impact": "high"}, CreatedAt: base.Add(24 * time.Hour), UpdatedAt: base.Add(25 * time.Hour)}
	currentHigh := domain.Task{ID: "task-reg-current-2", TraceID: "trace-reg-current-2", Title: "Prompt regression", State: domain.TaskBlocked, Labels: []string{"regression"}, Metadata: map[string]string{"team": "platform", "project": "alpha", "workflow": "prompt-tune", "template": "triage-system", "service": "assistant", "regression_count": "1", "regression_source": "prompt drift", "issue_key": "BIG-905"}, CreatedAt: base.Add(48 * time.Hour), UpdatedAt: base.Add(49 * time.Hour)}
	currentMedium := domain.Task{ID: "task-reg-current-3", TraceID: "trace-reg-current-3", Title: "Migration regression", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha", "workflow": "migrate", "template": "schema", "service": "database", "regression": "true", "regression_cause": "migration rollback", "issue_key": "BIG-906"}, CreatedAt: base.Add(48 * time.Hour), UpdatedAt: base.Add(50 * time.Hour)}
	ignored := domain.Task{ID: "task-reg-ignored", TraceID: "trace-reg-ignored", Title: "Out of scope regression", State: domain.TaskDeadLetter, Metadata: map[string]string{"team": "growth", "project": "beta", "workflow": "deploy", "template": "release", "service": "api", "regression_count": "4", "regression_source": "other team"}, CreatedAt: base.Add(48 * time.Hour), UpdatedAt: base.Add(49 * time.Hour)}
	healthy := domain.Task{ID: "task-reg-healthy", TraceID: "trace-reg-healthy", Title: "Healthy run", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha", "workflow": "deploy", "template": "release", "service": "api"}, CreatedAt: base.Add(48 * time.Hour), UpdatedAt: base.Add(51 * time.Hour)}
	for _, task := range []domain.Task{baseline, currentCritical, currentHigh, currentMedium, ignored, healthy} {
		recorder.StoreTask(task)
	}
	recorder.Record(domain.Event{ID: "evt-reg-retry-1", Type: domain.EventTaskRetried, TaskID: currentCritical.ID, TraceID: currentCritical.TraceID, Timestamp: currentCritical.UpdatedAt.Add(-time.Minute), Payload: map[string]any{"reason": "retry deploy"}})
	recorder.Record(domain.Event{ID: "evt-reg-dead-1", Type: domain.EventTaskDeadLetter, TaskID: currentCritical.ID, TraceID: currentCritical.TraceID, Timestamp: currentCritical.UpdatedAt, Payload: map[string]any{"message": "security scan failed"}})
	recorder.Record(domain.Event{ID: "evt-reg-blocked-1", Type: domain.EventRunTakeover, TaskID: currentHigh.ID, TraceID: currentHigh.TraceID, Timestamp: currentHigh.UpdatedAt, Payload: map[string]any{"reason": "prompt drift"}})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return base.Add(72 * time.Hour) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/regression/center?team=platform&project=alpha&since=2026-03-11T00:00:00Z&until=2026-03-12T23:59:59Z&compare_since=2026-03-10T00:00:00Z&compare_until=2026-03-10T23:59:59Z&bucket=day&limit=2", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected regression center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded regressionCenterResponse
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode regression center: %v", err)
	}
	if decoded.Filters.Team != "platform" || decoded.Filters.Project != "alpha" || decoded.Filters.Limit != 2 || decoded.Filters.Bucket != "day" {
		t.Fatalf("unexpected regression filters: %+v", decoded.Filters)
	}
	if decoded.Summary.TotalRegressions != 4 || decoded.Summary.AffectedTasks != 3 || decoded.Summary.CriticalRegressions != 1 || decoded.Summary.ReworkEvents != 1 {
		t.Fatalf("unexpected regression summary: %+v", decoded.Summary)
	}
	if decoded.Summary.TopSource != "security scan failed" || decoded.Summary.TopWorkflow != "deploy" {
		t.Fatalf("unexpected regression summary leaders: %+v", decoded.Summary)
	}
	if decoded.CompareSummary.Current.TotalRegressions != 4 || decoded.CompareSummary.Baseline.TotalRegressions != 1 || decoded.CompareSummary.DeltaRegressions != 3 || decoded.CompareSummary.DeltaAffectedTasks != 2 || decoded.CompareSummary.DeltaCriticalRegressions != 1 || decoded.CompareSummary.DeltaReworkEvents != 1 {
		t.Fatalf("unexpected regression compare summary: %+v", decoded.CompareSummary)
	}
	if len(decoded.WorkflowBreakdown) != 3 || decoded.WorkflowBreakdown[0].Key != "deploy" || decoded.WorkflowBreakdown[0].TotalRegressions != 2 {
		t.Fatalf("unexpected workflow breakdown: %+v", decoded.WorkflowBreakdown)
	}
	if len(decoded.TeamBreakdown) != 1 || decoded.TeamBreakdown[0].Key != "platform" || decoded.TeamBreakdown[0].TotalRegressions != 4 {
		t.Fatalf("unexpected team breakdown: %+v", decoded.TeamBreakdown)
	}
	if len(decoded.TemplateBreakdown) != 3 || decoded.TemplateBreakdown[0].Key != "release" || decoded.TemplateBreakdown[0].TotalRegressions != 2 {
		t.Fatalf("unexpected template breakdown: %+v", decoded.TemplateBreakdown)
	}
	if len(decoded.ServiceBreakdown) != 3 || decoded.ServiceBreakdown[0].Key != "api" || decoded.ServiceBreakdown[0].TotalRegressions != 2 {
		t.Fatalf("unexpected service breakdown: %+v", decoded.ServiceBreakdown)
	}
	if len(decoded.AttributionBreakdown) != 3 || decoded.AttributionBreakdown[0].Key != "security scan failed" || decoded.AttributionBreakdown[0].TotalRegressions != 2 {
		t.Fatalf("unexpected attribution breakdown: %+v", decoded.AttributionBreakdown)
	}
	if len(decoded.Hotspots) == 0 || decoded.Hotspots[0].Dimension != "team" || decoded.Hotspots[0].Key != "platform" || decoded.Hotspots[0].TotalRegressions != 4 {
		t.Fatalf("unexpected regression hotspots: %+v", decoded.Hotspots)
	}
	if len(decoded.Trend) != 2 || decoded.Trend[0].Label != "2026-03-11" || decoded.Trend[0].TotalRegressions != 2 || decoded.Trend[0].AffectedTasks != 1 || decoded.Trend[0].CriticalRegressions != 1 || decoded.Trend[0].ReworkEvents != 1 {
		t.Fatalf("unexpected first trend point: %+v", decoded.Trend)
	}
	if decoded.Trend[1].Label != "2026-03-12" || decoded.Trend[1].TotalRegressions != 2 || decoded.Trend[1].AffectedTasks != 2 || decoded.Trend[1].CriticalRegressions != 0 || decoded.Trend[1].ReworkEvents != 0 {
		t.Fatalf("unexpected second trend point: %+v", decoded.Trend[1])
	}
	if len(decoded.Findings) != 2 || decoded.Findings[0].Task.ID != "task-reg-current-1" || decoded.Findings[1].Task.ID != "task-reg-current-2" {
		t.Fatalf("unexpected regression findings ordering/limit: %+v", decoded.Findings)
	}
	if decoded.Findings[0].Policy.Plan != "premium" || decoded.Findings[0].Policy.ApprovalFlow != "risk-reviewed" || !decoded.Findings[0].Risk.RequiresApproval {
		t.Fatalf("expected premium risk-reviewed first finding, got %+v", decoded.Findings[0])
	}
	if decoded.Findings[0].RegressionCount != 2 || decoded.Findings[0].ReworkEvents != 1 || decoded.Findings[0].Drilldown.Run != "/v2/runs/task-reg-current-1" || decoded.Findings[0].Drilldown.Events != "/events?task_id=task-reg-current-1&limit=200" || decoded.Findings[0].Drilldown.Replay != "/replay/task-reg-current-1" || decoded.Findings[0].Drilldown.IssueKey != "BIG-904" {
		t.Fatalf("unexpected first regression drilldown payload: %+v", decoded.Findings[0])
	}
	if decoded.Findings[1].Workflow != "prompt-tune" || decoded.Findings[1].Team != "platform" || decoded.Findings[1].Template != "triage-system" || decoded.Findings[1].Service != "assistant" || decoded.Findings[1].Severity != "high" || decoded.Findings[1].Attribution != "prompt drift" {
		t.Fatalf("unexpected second regression finding: %+v", decoded.Findings[1])
	}
	body := response.Body.String()
	if strings.Contains(body, "task-reg-ignored") || strings.Contains(body, "task-reg-baseline") || strings.Contains(body, "task-reg-healthy") {
		t.Fatalf("expected response to exclude ignored/baseline/healthy tasks, got %s", body)
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

	assignOwnerBody, _ := json.Marshal(map[string]any{"action": "assign_owner", "task_id": "task-v2-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "owner": "carol", "note": "handoff owner"})
	assignOwnerResponse := httptest.NewRecorder()
	handler.ServeHTTP(assignOwnerResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(assignOwnerBody)))
	if assignOwnerResponse.Code != http.StatusOK {
		t.Fatalf("expected assign owner action payload, got %d %s", assignOwnerResponse.Code, assignOwnerResponse.Body.String())
	}
	var assignOwnerDecoded struct {
		Action   string `json:"action"`
		Takeover struct {
			Owner    string `json:"owner"`
			Reviewer string `json:"reviewer"`
		} `json:"takeover"`
		Operation struct {
			Scope         string `json:"scope"`
			PreviousOwner string `json:"previous_owner"`
			Owner         string `json:"owner"`
		} `json:"operation"`
	}
	if err := json.Unmarshal(assignOwnerResponse.Body.Bytes(), &assignOwnerDecoded); err != nil {
		t.Fatalf("decode assign owner action: %v", err)
	}
	if assignOwnerDecoded.Action != "assign_owner" || assignOwnerDecoded.Takeover.Owner != "carol" || assignOwnerDecoded.Takeover.Reviewer != "bob" || assignOwnerDecoded.Operation.Scope != "collaboration" || assignOwnerDecoded.Operation.PreviousOwner != "alice" || assignOwnerDecoded.Operation.Owner != "carol" {
		t.Fatalf("unexpected assign owner payload: %+v", assignOwnerDecoded)
	}

	assignReviewerBody, _ := json.Marshal(map[string]any{"action": "assign_reviewer", "task_id": "task-v2-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "reviewer": "dave", "note": "peer review owner updated"})
	assignReviewerResponse := httptest.NewRecorder()
	handler.ServeHTTP(assignReviewerResponse, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(assignReviewerBody)))
	if assignReviewerResponse.Code != http.StatusOK {
		t.Fatalf("expected assign reviewer action payload, got %d %s", assignReviewerResponse.Code, assignReviewerResponse.Body.String())
	}
	var assignReviewerDecoded struct {
		Action   string `json:"action"`
		Takeover struct {
			Owner    string `json:"owner"`
			Reviewer string `json:"reviewer"`
		} `json:"takeover"`
		Operation struct {
			PreviousReviewer string `json:"previous_reviewer"`
			Reviewer         string `json:"reviewer"`
			TaskStateBefore  string `json:"task_state_before"`
			TaskStateAfter   string `json:"task_state_after"`
		} `json:"operation"`
	}
	if err := json.Unmarshal(assignReviewerResponse.Body.Bytes(), &assignReviewerDecoded); err != nil {
		t.Fatalf("decode assign reviewer action: %v", err)
	}
	if assignReviewerDecoded.Action != "assign_reviewer" || assignReviewerDecoded.Takeover.Owner != "carol" || assignReviewerDecoded.Takeover.Reviewer != "dave" || assignReviewerDecoded.Operation.PreviousReviewer != "bob" || assignReviewerDecoded.Operation.Reviewer != "dave" || assignReviewerDecoded.Operation.TaskStateBefore != string(domain.TaskBlocked) || assignReviewerDecoded.Operation.TaskStateAfter != string(domain.TaskBlocked) {
		t.Fatalf("unexpected assign reviewer payload: %+v", assignReviewerDecoded)
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
	if !strings.Contains(bodyString, "premium") || !strings.Contains(bodyString, "Investigating flaky validation") || !strings.Contains(bodyString, "merge PR") || !strings.Contains(bodyString, "handoff ready") || !strings.Contains(bodyString, "carol") || !strings.Contains(bodyString, "dave") || !strings.Contains(bodyString, "assign_reviewer") {
		t.Fatalf("expected premium policy, collaboration details, and action timeline in run detail, got %s", bodyString)
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

func TestV2ControlCenterIncludesMultiWorkerPoolSummary(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Worker: fakeWorkerPoolStatus{}, Control: control.New(), Now: func() time.Time { return time.Unix(1700003600, 0) }}
	handler := server.Handler()

	body, _ := json.Marshal(map[string]any{"id": "task-pool-1", "title": "Pool target", "priority": 1, "metadata": map[string]any{"team": "platform", "project": "alpha"}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected task create 202, got %d body=%s", response.Code, response.Body.String())
	}

	centerResponse := httptest.NewRecorder()
	handler.ServeHTTP(centerResponse, httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=10&audit_limit=10", nil))
	if centerResponse.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d", centerResponse.Code)
	}
	bodyText := centerResponse.Body.String()
	for _, want := range []string{"worker_pool", "total_workers", "3", "active_workers", "2", "idle_workers", "1", "worker-c", "idle", "preemption_active", "task-low"} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("expected %q in control center payload, got %s", want, bodyText)
		}
	}
}

func TestV2ControlCenterIncludesDistributedDiagnostics(t *testing.T) {
	recorder := observability.NewRecorder()
	controller := control.New()
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		Executors: []domain.ExecutorKind{domain.ExecutorLocal, domain.ExecutorKubernetes, domain.ExecutorRay},
		Control:   controller,
		Worker:    fakeWorkerPoolStatus{},
		Now:       func() time.Time { return time.Unix(1700007200, 0) },
	}
	handler := server.Handler()
	base := time.Unix(1700000000, 0)
	for _, task := range []domain.Task{
		{ID: "diag-local", TraceID: "trace-local", Title: "Local diag", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha"}, UpdatedAt: base.Add(time.Minute)},
		{ID: "diag-k8s", TraceID: "trace-k8s", Title: "K8s diag", State: domain.TaskSucceeded, RequiredTools: []string{"browser"}, Metadata: map[string]string{"team": "platform", "project": "alpha"}, UpdatedAt: base.Add(2 * time.Minute)},
		{ID: "diag-ray", TraceID: "trace-ray", Title: "Ray diag", State: domain.TaskSucceeded, RequiredTools: []string{"gpu"}, Metadata: map[string]string{"team": "platform", "project": "alpha"}, UpdatedAt: base.Add(3 * time.Minute)},
	} {
		recorder.StoreTask(task)
	}
	controller.Takeover("diag-k8s", "alice", "bob", "monitor rollout", base.Add(4*time.Minute))
	for _, event := range []domain.Event{
		{ID: "evt-local-routed", Type: domain.EventSchedulerRouted, TaskID: "diag-local", TraceID: "trace-local", Timestamp: base.Add(time.Second), Payload: map[string]any{"executor": domain.ExecutorLocal, "reason": "default local executor for low/medium risk"}},
		{ID: "evt-local-completed", Type: domain.EventTaskCompleted, TaskID: "diag-local", TraceID: "trace-local", Timestamp: base.Add(2 * time.Second), Payload: map[string]any{"executor": domain.ExecutorLocal}},
		{ID: "evt-k8s-routed", Type: domain.EventSchedulerRouted, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(3 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes, "reason": "browser workloads default to kubernetes executor"}},
		{ID: "evt-k8s-started", Type: domain.EventTaskStarted, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(4 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes}},
		{ID: "evt-k8s-completed", Type: domain.EventTaskCompleted, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(5 * time.Second), Payload: map[string]any{"executor": domain.ExecutorKubernetes}},
		{ID: "evt-ray-routed", Type: domain.EventSchedulerRouted, TaskID: "diag-ray", TraceID: "trace-ray", Timestamp: base.Add(6 * time.Second), Payload: map[string]any{"executor": domain.ExecutorRay, "reason": "gpu workloads default to ray executor"}},
		{ID: "evt-ray-completed", Type: domain.EventTaskCompleted, TaskID: "diag-ray", TraceID: "trace-ray", Timestamp: base.Add(7 * time.Second), Payload: map[string]any{"executor": domain.ExecutorRay}},
	} {
		recorder.Record(event)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v2/control-center?team=platform&project=alpha&limit=10&audit_limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Diagnostics struct {
			Summary struct {
				RegisteredExecutors  int `json:"registered_executors"`
				TotalRoutedDecisions int `json:"total_routed_decisions"`
				ActiveWorkers        int `json:"active_workers"`
				ActiveTakeovers      int `json:"active_takeovers"`
			} `json:"summary"`
			RoutingReasons []struct {
				Executor string `json:"executor"`
				Reason   string `json:"reason"`
				Count    int    `json:"count"`
			} `json:"routing_reasons"`
			ExecutorCapacity []struct {
				Executor       string   `json:"executor"`
				Health         string   `json:"health"`
				MaxConcurrency int      `json:"max_concurrency"`
				ActiveTasks    int      `json:"active_tasks"`
				QueuedTasks    int      `json:"queued_tasks"`
				SampleTasks    []string `json:"sample_tasks"`
				TeamBreakdown  []struct {
					Key   string `json:"key"`
					Count int    `json:"count"`
				} `json:"team_breakdown"`
				TopRoutingReasons []struct {
					Reason string `json:"reason"`
					Count  int    `json:"count"`
				} `json:"top_routing_reasons"`
			} `json:"executor_capacity"`
			ClusterHealth struct {
				HealthyExecutors int            `json:"healthy_executors"`
				WorkerStates     map[string]int `json:"worker_states"`
				TeamBreakdown    []struct {
					Key   string `json:"key"`
					Count int    `json:"count"`
				} `json:"team_breakdown"`
				SaturatedExecutors []string `json:"saturated_executors"`
				TakeoverOwners     []struct {
					Key   string `json:"key"`
					Count int    `json:"count"`
				} `json:"takeover_owners"`
			} `json:"cluster_health"`
			RolloutReport struct {
				Markdown  string `json:"markdown"`
				ExportURL string `json:"export_url"`
			} `json:"rollout_report"`
		} `json:"distributed_diagnostics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed diagnostics: %v", err)
	}
	if decoded.Diagnostics.Summary.RegisteredExecutors != 3 || decoded.Diagnostics.Summary.TotalRoutedDecisions != 3 {
		t.Fatalf("unexpected diagnostics summary: %+v", decoded.Diagnostics.Summary)
	}
	if decoded.Diagnostics.Summary.ActiveWorkers != 2 || decoded.Diagnostics.Summary.ActiveTakeovers != 1 {
		t.Fatalf("unexpected worker/takeover summary: %+v", decoded.Diagnostics.Summary)
	}
	if len(decoded.Diagnostics.RoutingReasons) != 3 {
		t.Fatalf("expected 3 routing reasons, got %+v", decoded.Diagnostics.RoutingReasons)
	}
	if len(decoded.Diagnostics.ExecutorCapacity) != 3 {
		t.Fatalf("expected executor capacity for 3 executors, got %+v", decoded.Diagnostics.ExecutorCapacity)
	}
	if decoded.Diagnostics.ExecutorCapacity[0].Executor != "kubernetes" || decoded.Diagnostics.ExecutorCapacity[0].ActiveTasks != 1 || len(decoded.Diagnostics.ExecutorCapacity[0].TopRoutingReasons) == 0 {
		t.Fatalf("unexpected kubernetes executor diagnostics: %+v", decoded.Diagnostics.ExecutorCapacity[0])
	}
	if len(decoded.Diagnostics.ExecutorCapacity[0].TeamBreakdown) == 0 || decoded.Diagnostics.ExecutorCapacity[0].TeamBreakdown[0].Key != "platform" {
		t.Fatalf("expected team drilldown in executor diagnostics, got %+v", decoded.Diagnostics.ExecutorCapacity[0])
	}
	if decoded.Diagnostics.ClusterHealth.HealthyExecutors != 3 || decoded.Diagnostics.ClusterHealth.WorkerStates["running"] != 1 {
		t.Fatalf("unexpected cluster health payload: %+v", decoded.Diagnostics.ClusterHealth)
	}
	if len(decoded.Diagnostics.ClusterHealth.TeamBreakdown) == 0 || decoded.Diagnostics.ClusterHealth.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("expected cluster team breakdown, got %+v", decoded.Diagnostics.ClusterHealth)
	}
	if len(decoded.Diagnostics.ClusterHealth.TakeoverOwners) == 0 || decoded.Diagnostics.ClusterHealth.TakeoverOwners[0].Key != "alice" {
		t.Fatalf("expected takeover owner rollup, got %+v", decoded.Diagnostics.ClusterHealth)
	}
	if !strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "# BigClaw Distributed Diagnostics Report") || !strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Takeover owners") || !strings.Contains(decoded.Diagnostics.RolloutReport.ExportURL, "/v2/reports/distributed/export") {
		t.Fatalf("unexpected rollout report payload: %+v", decoded.Diagnostics.RolloutReport)
	}
}

func TestV2ControlCenterAuditFiltersOwnerReviewerAndScope(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	controller := control.New()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Control: controller, Now: func() time.Time { return time.Unix(1700003500, 0) }}
	handler := server.Handler()

	body, _ := json.Marshal(map[string]any{"id": "task-audit-1", "title": "Audit target", "priority": 1, "metadata": map[string]any{"team": "platform", "project": "alpha"}})
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))
	if createResponse.Code != http.StatusAccepted {
		t.Fatalf("expected task create 202, got %d %s", createResponse.Code, createResponse.Body.String())
	}

	for _, actionBody := range [][]byte{
		mustJSON(map[string]any{"action": "takeover", "task_id": "task-audit-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "reviewer": "bob", "note": "starting review"}),
		mustJSON(map[string]any{"action": "assign_owner", "task_id": "task-audit-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "owner": "carol", "note": "handoff owner"}),
		mustJSON(map[string]any{"action": "assign_reviewer", "task_id": "task-audit-1", "actor": "alice", "role": "eng_lead", "viewer_team": "platform", "reviewer": "dave", "note": "handoff reviewer"}),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v2/control-center/actions", bytes.NewReader(actionBody)))
		if response.Code != http.StatusOK {
			t.Fatalf("expected collaboration action success, got %d %s", response.Code, response.Body.String())
		}
	}

	auditResponse := httptest.NewRecorder()
	auditRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/audit?task_id=task-audit-1&action=assign_reviewer&owner=carol&reviewer=dave&scope=collaboration&audit_limit=10", nil)
	auditRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	auditRequest.Header.Set("X-BigClaw-Actor", "ops-1")
	handler.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected filtered control center audit 200, got %d %s", auditResponse.Code, auditResponse.Body.String())
	}
	var decoded controlCenterAuditResponse
	if err := json.Unmarshal(auditResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode filtered control center audit: %v", err)
	}
	if decoded.Filters.TaskID != "task-audit-1" || decoded.Filters.Action != "assign_reviewer" || decoded.Filters.Owner != "carol" || decoded.Filters.Reviewer != "dave" || decoded.Filters.Scope != "collaboration" {
		t.Fatalf("unexpected filtered audit filters: %+v", decoded.Filters)
	}
	if decoded.AuditSummary.Total != 1 || decoded.AuditSummary.NotesCount != 1 {
		t.Fatalf("unexpected filtered audit summary: %+v", decoded.AuditSummary)
	}
	if len(decoded.AuditSummary.ByScope) != 1 || decoded.AuditSummary.ByScope[0].Key != "collaboration" || decoded.AuditSummary.ByScope[0].Count != 1 {
		t.Fatalf("unexpected by_scope summary: %+v", decoded.AuditSummary.ByScope)
	}
	if len(decoded.AuditSummary.ByOwner) != 1 || decoded.AuditSummary.ByOwner[0].Key != "carol" || decoded.AuditSummary.ByOwner[0].Count != 1 {
		t.Fatalf("unexpected by_owner summary: %+v", decoded.AuditSummary.ByOwner)
	}
	if len(decoded.AuditSummary.ByReviewer) != 1 || decoded.AuditSummary.ByReviewer[0].Key != "dave" || decoded.AuditSummary.ByReviewer[0].Count != 1 {
		t.Fatalf("unexpected by_reviewer summary: %+v", decoded.AuditSummary.ByReviewer)
	}
	if len(decoded.Audit) != 1 {
		t.Fatalf("expected one filtered audit entry, got %+v", decoded.Audit)
	}
	entry := decoded.Audit[0]
	if entry.Action != "assign_reviewer" || entry.Scope != "collaboration" || entry.TaskID != "task-audit-1" || entry.TaskStateBefore != string(domain.TaskBlocked) || entry.TaskStateAfter != string(domain.TaskBlocked) || entry.PreviousOwner != "carol" || entry.Owner != "carol" || entry.PreviousReviewer != "bob" || entry.Reviewer != "dave" || entry.Note != "handoff reviewer" || entry.OperationID == "" {
		t.Fatalf("unexpected filtered audit entry: %+v", entry)
	}
}

func TestV2ControlCenterPolicyEndpoints(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "scheduler-policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"default_executor":"ray","tool_executors":{"browser":"ray"},"urgent_priority_threshold":2,"fairness":{"window_seconds":30,"max_recent_decisions_per_tenant":1}}`), 0o644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
	policySQLitePath := filepath.Join(dir, "scheduler-policy.db")
	store, err := scheduler.NewPolicyStoreWithSQLite(policyPath, policySQLitePath)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	defer func() { _ = store.Close() }()
	fairnessPath := filepath.Join(dir, "fairness.db")
	fairnessStore, err := scheduler.NewFairnessStore(fairnessPath)
	if err != nil {
		t.Fatalf("new fairness store: %v", err)
	}
	if closable, ok := fairnessStore.(interface{ Close() error }); ok {
		defer func() { _ = closable.Close() }()
	}
	schedulerRuntime := scheduler.NewWithStores(store, fairnessStore)
	schedulerRuntime.Decide(domain.Task{ID: "fair-1", TenantID: "tenant-a", Priority: 3}, scheduler.QuotaSnapshot{})
	schedulerRuntime.Decide(domain.Task{ID: "fair-2", TenantID: "tenant-b", Priority: 3}, scheduler.QuotaSnapshot{})
	recorder := observability.NewRecorder()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), SchedulerPolicy: store, SchedulerRuntime: schedulerRuntime, Now: time.Now}
	handler := server.Handler()

	policyResponse := httptest.NewRecorder()
	policyRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/policy", nil)
	policyRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	handler.ServeHTTP(policyResponse, policyRequest)
	if policyResponse.Code != http.StatusOK {
		t.Fatalf("expected scheduler policy 200, got %d %s", policyResponse.Code, policyResponse.Body.String())
	}
	var policyDecoded struct {
		Backend          string `json:"backend"`
		Shared           bool   `json:"shared"`
		SourcePath       string `json:"source_path"`
		SharedPath       string `json:"shared_path"`
		ReloadSupported  bool   `json:"reload_supported"`
		ReloadAuthorized bool   `json:"reload_authorized"`
		Policy           struct {
			DefaultExecutor         string            `json:"default_executor"`
			UrgentPriorityThreshold int               `json:"urgent_priority_threshold"`
			ToolExecutors           map[string]string `json:"tool_executors"`
			Fairness                struct {
				WindowSeconds               int `json:"window_seconds"`
				MaxRecentDecisionsPerTenant int `json:"max_recent_decisions_per_tenant"`
			} `json:"fairness"`
		} `json:"policy"`
		Fairness struct {
			Enabled                     bool   `json:"enabled"`
			Shared                      bool   `json:"shared"`
			Backend                     string `json:"backend"`
			WindowSeconds               int    `json:"window_seconds"`
			MaxRecentDecisionsPerTenant int    `json:"max_recent_decisions_per_tenant"`
			ActiveTenants               int    `json:"active_tenants"`
			Tenants                     []struct {
				TenantID            string `json:"tenant_id"`
				RecentAcceptedCount int    `json:"recent_accepted_count"`
			} `json:"tenants"`
		} `json:"fairness"`
	}
	if err := json.Unmarshal(policyResponse.Body.Bytes(), &policyDecoded); err != nil {
		t.Fatalf("decode scheduler policy response: %v", err)
	}
	if policyDecoded.Backend != "sqlite" || !policyDecoded.Shared || policyDecoded.SourcePath != policyPath || policyDecoded.SharedPath != policySQLitePath || !policyDecoded.ReloadSupported || !policyDecoded.ReloadAuthorized || policyDecoded.Policy.DefaultExecutor != string(domain.ExecutorRay) || policyDecoded.Policy.ToolExecutors["browser"] != string(domain.ExecutorRay) || policyDecoded.Policy.UrgentPriorityThreshold != 2 || policyDecoded.Policy.Fairness.WindowSeconds != 30 || policyDecoded.Policy.Fairness.MaxRecentDecisionsPerTenant != 1 {
		t.Fatalf("unexpected scheduler policy payload: %+v", policyDecoded)
	}
	if !policyDecoded.Fairness.Enabled || !policyDecoded.Fairness.Shared || policyDecoded.Fairness.Backend != "sqlite" || policyDecoded.Fairness.ActiveTenants != 2 || len(policyDecoded.Fairness.Tenants) != 2 {
		t.Fatalf("unexpected fairness runtime payload: %+v", policyDecoded.Fairness)
	}

	if err := os.WriteFile(policyPath, []byte(`{"default_executor":"kubernetes","high_risk_executor":"ray","fairness":{"window_seconds":10,"max_recent_decisions_per_tenant":2}}`), 0o644); err != nil {
		t.Fatalf("rewrite policy file: %v", err)
	}
	reloadResponse := httptest.NewRecorder()
	reloadRequest := httptest.NewRequest(http.MethodPost, "/v2/control-center/policy/reload", nil)
	reloadRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	handler.ServeHTTP(reloadResponse, reloadRequest)
	if reloadResponse.Code != http.StatusOK || !strings.Contains(reloadResponse.Body.String(), `"reloaded":true`) {
		t.Fatalf("expected reload response, got %d %s", reloadResponse.Code, reloadResponse.Body.String())
	}

	policyResponse = httptest.NewRecorder()
	handler.ServeHTTP(policyResponse, policyRequest)
	if !strings.Contains(policyResponse.Body.String(), `"default_executor":"kubernetes"`) || !strings.Contains(policyResponse.Body.String(), `"high_risk_executor":"ray"`) || !strings.Contains(policyResponse.Body.String(), `"window_seconds":10`) {
		t.Fatalf("expected reloaded policy in get response, got %s", policyResponse.Body.String())
	}

	forbiddenResponse := httptest.NewRecorder()
	forbiddenRequest := httptest.NewRequest(http.MethodPost, "/v2/control-center/policy/reload", nil)
	forbiddenRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	forbiddenRequest.Header.Set("X-BigClaw-Team", "platform")
	handler.ServeHTTP(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden reload for eng lead, got %d %s", forbiddenResponse.Code, forbiddenResponse.Body.String())
	}
}

func mustJSON(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return body
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
	if !strings.Contains(centerBody, "eng_lead") || !strings.Contains(centerBody, "allowed_actions") || !strings.Contains(centerBody, "takeover") || !strings.Contains(centerBody, "assign_owner") || !strings.Contains(centerBody, "assign_reviewer") {
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
	recorder.StoreTask(domain.Task{ID: "task-scope-1", TraceID: "trace-scope-1", Title: "Scoped", State: domain.TaskBlocked, Metadata: map[string]string{"team": "platform", "project": "alpha", "blocked_reason": "waiting for platform review", "regression_count": "1", "workflow": "deploy", "template": "release", "service": "api"}, CreatedAt: base, UpdatedAt: base.Add(time.Hour)})
	recorder.StoreTask(domain.Task{ID: "task-scope-2", TraceID: "trace-scope-2", Title: "Other", State: domain.TaskBlocked, Metadata: map[string]string{"team": "growth", "project": "beta", "regression_count": "1", "workflow": "prompt-tune", "template": "triage-system", "service": "assistant"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Hour)})
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

	triageRequest := httptest.NewRequest(http.MethodGet, "/v2/triage/center?limit=10", nil)
	triageRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	triageRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	triageRequest.Header.Set("X-BigClaw-Team", "platform")
	triageResponse := httptest.NewRecorder()
	handler.ServeHTTP(triageResponse, triageRequest)
	if triageResponse.Code != http.StatusOK {
		t.Fatalf("expected scoped triage center 200, got %d %s", triageResponse.Code, triageResponse.Body.String())
	}
	triageBody := triageResponse.Body.String()
	if !strings.Contains(triageBody, "task-scope-1") || strings.Contains(triageBody, "task-scope-2") {
		t.Fatalf("expected triage center to be scoped to platform team, got %s", triageBody)
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

	forbiddenTriageRequest := httptest.NewRequest(http.MethodGet, "/v2/triage/center?team=growth&limit=10", nil)
	forbiddenTriageRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	forbiddenTriageRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	forbiddenTriageRequest.Header.Set("X-BigClaw-Team", "platform")
	forbiddenTriageResponse := httptest.NewRecorder()
	handler.ServeHTTP(forbiddenTriageResponse, forbiddenTriageRequest)
	if forbiddenTriageResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden triage center for mismatched team, got %d %s", forbiddenTriageResponse.Code, forbiddenTriageResponse.Body.String())
	}

	regressionRequest := httptest.NewRequest(http.MethodGet, "/v2/regression/center?limit=10", nil)
	regressionRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	regressionRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	regressionRequest.Header.Set("X-BigClaw-Team", "platform")
	regressionResponse := httptest.NewRecorder()
	handler.ServeHTTP(regressionResponse, regressionRequest)
	if regressionResponse.Code != http.StatusOK {
		t.Fatalf("expected scoped regression center 200, got %d %s", regressionResponse.Code, regressionResponse.Body.String())
	}
	regressionBody := regressionResponse.Body.String()
	if !strings.Contains(regressionBody, "task-scope-1") || strings.Contains(regressionBody, "task-scope-2") {
		t.Fatalf("expected regression center to be scoped to platform team, got %s", regressionBody)
	}

	forbiddenRegressionRequest := httptest.NewRequest(http.MethodGet, "/v2/regression/center?team=growth&limit=10", nil)
	forbiddenRegressionRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	forbiddenRegressionRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	forbiddenRegressionRequest.Header.Set("X-BigClaw-Team", "platform")
	forbiddenRegressionResponse := httptest.NewRecorder()
	handler.ServeHTTP(forbiddenRegressionResponse, forbiddenRegressionRequest)
	if forbiddenRegressionResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden regression center for mismatched team, got %d %s", forbiddenRegressionResponse.Code, forbiddenRegressionResponse.Body.String())
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
