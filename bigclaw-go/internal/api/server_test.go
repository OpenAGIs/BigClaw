package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
		{WorkerID: "worker-b", State: "leased", CurrentExecutor: domain.ExecutorKubernetes, SuccessfulRuns: 3, LeaseRenewals: 2, LeaseRenewalFailures: 1, LeaseLostRuns: 1, LastResult: "warming", PreemptionActive: true, CurrentPreemptionTaskID: "task-low", CurrentPreemptionWorkerID: "worker-low", LastPreemptedTaskID: "task-low", LastPreemptionAt: time.Unix(1700000100, 0), LastPreemptionReason: "preempted by urgent task task-urgent (priority=1)", PreemptionsIssued: 1},
		{WorkerID: "worker-c", State: "idle", SuccessfulRuns: 8, LeaseRenewals: 0, LastResult: "idle"},
	}
}

type fakeNodeAwareWorkerPoolStatus struct {
	now time.Time
}

func (f fakeNodeAwareWorkerPoolStatus) Snapshot() worker.Status {
	return f.Snapshots()[0]
}

func (f fakeNodeAwareWorkerPoolStatus) Snapshots() []worker.Status {
	base := f.now
	if base.IsZero() {
		base = time.Unix(1700003600, 0)
	}
	return []worker.Status{
		{WorkerID: "worker-node-a", NodeID: "node-a", State: "running", CurrentExecutor: domain.ExecutorLocal, LastHeartbeatAt: base.Add(-2 * time.Minute), SuccessfulRuns: 5, LeaseRenewals: 7, LastResult: "ok"},
		{WorkerID: "worker-node-b", NodeID: "node-b", State: "leased", CurrentExecutor: domain.ExecutorKubernetes, LastHeartbeatAt: base.Add(-9 * time.Minute), SuccessfulRuns: 3, LeaseRenewals: 2, LeaseRenewalFailures: 1, LeaseLostRuns: 1, LastResult: "warming"},
		{WorkerID: "worker-node-c", NodeID: "node-c", State: "idle", CurrentExecutor: domain.ExecutorRay, LastHeartbeatAt: base.Add(-time.Minute), SuccessfulRuns: 8, LeaseRenewals: 0, LastResult: "idle"},
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestWorkerPoolSummaryBuildsNodeAwareAggregates(t *testing.T) {
	base := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	server := &Server{
		Worker: fakeNodeAwareWorkerPoolStatus{now: base},
		Now:    func() time.Time { return base },
	}

	summary := server.workerPoolSummary()
	if summary == nil {
		t.Fatal("expected worker pool summary")
	}
	if summary.TotalWorkers != 3 || summary.ActiveWorkers != 2 || summary.IdleWorkers != 1 {
		t.Fatalf("unexpected worker counts: %+v", summary)
	}
	if summary.TotalNodes != 3 || summary.ActiveNodes != 1 || summary.IdleNodes != 1 || summary.DegradedNodes != 1 {
		t.Fatalf("unexpected node counts: %+v", summary)
	}
	if summary.CapacityUtilizationPercent != float64(2)/float64(3)*100 {
		t.Fatalf("unexpected capacity utilization: %+v", summary)
	}
	if len(summary.ExecutorDistribution) != 3 || summary.ExecutorDistribution[0].Count != 1 {
		t.Fatalf("unexpected executor distribution: %+v", summary.ExecutorDistribution)
	}
	if len(summary.NodeHealthDistribution) != 3 ||
		summary.NodeHealthDistribution[0].Key != "active" ||
		summary.NodeHealthDistribution[1].Key != "degraded" ||
		summary.NodeHealthDistribution[2].Key != "idle" {
		t.Fatalf("unexpected node health distribution: %+v", summary.NodeHealthDistribution)
	}
	if len(summary.Nodes) != 3 || summary.Nodes[1].NodeID != "node-b" || summary.Nodes[1].Health != "degraded" || summary.Nodes[1].StaleWorkers != 1 {
		t.Fatalf("unexpected node summaries: %+v", summary.Nodes)
	}
}

func TestQueryMemoryEventsAndEventStateHelpers(t *testing.T) {
	t.Run("nil recorder", func(t *testing.T) {
		server := &Server{}
		if got := server.queryMemoryEvents("", "", "", 5); got != nil {
			t.Fatalf("expected nil recorder to return nil events, got %+v", got)
		}
	})

	t.Run("memory event filtering", func(t *testing.T) {
		recorder := observability.NewRecorder()
		server := &Server{Recorder: recorder}
		base := time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC)
		for _, event := range []domain.Event{
			{ID: "evt-1", TaskID: "task-a", TraceID: "trace-a", Type: domain.EventTaskQueued, Timestamp: base},
			{ID: "evt-2", TaskID: "task-b", TraceID: "trace-b", Type: domain.EventTaskStarted, Timestamp: base.Add(time.Minute)},
			{ID: "evt-3", TaskID: "task-a", TraceID: "trace-a", Type: domain.EventTaskCompleted, Timestamp: base.Add(2 * time.Minute)},
		} {
			recorder.Record(event)
		}

		latestTwo := server.queryMemoryEvents("", "", "", 2)
		if len(latestTwo) != 2 || latestTwo[0].ID != "evt-2" || latestTwo[1].ID != "evt-3" {
			t.Fatalf("expected no-after query to keep the latest matching events, got %+v", latestTwo)
		}

		taskOnly := server.queryMemoryEvents("task-a", "", "", 0)
		if len(taskOnly) != 2 || taskOnly[0].ID != "evt-1" || taskOnly[1].ID != "evt-3" {
			t.Fatalf("expected task-only query to skip non-matching memory events, got %+v", taskOnly)
		}

		taskAfter := server.queryMemoryEvents("task-a", "", "evt-1", 5)
		if len(taskAfter) != 1 || taskAfter[0].ID != "evt-3" {
			t.Fatalf("expected after filter to return later task events only, got %+v", taskAfter)
		}

		traceLimited := server.queryMemoryEvents("", "trace-a", "evt-1", 1)
		if len(traceLimited) != 1 || traceLimited[0].ID != "evt-3" {
			t.Fatalf("expected trace filter with limit to stop at first matching event, got %+v", traceLimited)
		}
	})

	t.Run("event state", func(t *testing.T) {
		if got := eventState(domain.EventTaskCompleted); got != string(domain.TaskSucceeded) {
			t.Fatalf("expected succeeded event to map to task state, got %q", got)
		}
		if got := eventState(domain.EventType("custom.unknown")); got != "unknown" {
			t.Fatalf("expected unknown event type to fall back to unknown, got %q", got)
		}
	})
}

func TestCheckpointExpiredErrorString(t *testing.T) {
	err := checkpointExpiredError{
		Diagnostics: checkpointDiagnostics{
			SubscriberID:    "worker-a",
			Status:          "expired",
			SuggestedAction: "reset the checkpoint and replay retained events",
		},
	}

	if got := err.Error(); got != "checkpoint for subscriber worker-a expired: reset the checkpoint and replay retained events" {
		t.Fatalf("unexpected checkpoint expired error string: %q", got)
	}
}

type countingInspectorQueue struct {
	*queue.MemoryQueue
	listCalls int
}

func (q *countingInspectorQueue) GetTask(ctx context.Context, taskID string) (queue.TaskSnapshot, error) {
	return q.MemoryQueue.GetTask(ctx, taskID)
}

func (q *countingInspectorQueue) ListTasks(ctx context.Context, limit int) ([]queue.TaskSnapshot, error) {
	q.listCalls++
	return q.MemoryQueue.ListTasks(ctx, limit)
}

type blockingEventLog struct {
	history       []domain.Event
	replayStarted chan struct{}
	release       chan struct{}
}

func (l *blockingEventLog) Write(context.Context, domain.Event) error {
	return nil
}

func (l *blockingEventLog) Replay(limit int) ([]domain.Event, error) {
	return l.query("", "", "", limit), nil
}

func (l *blockingEventLog) ReplayAfter(afterID string, limit int) ([]domain.Event, error) {
	return l.query("", "", afterID, limit), nil
}

func (l *blockingEventLog) EventsByTask(taskID string, limit int) ([]domain.Event, error) {
	return l.query(taskID, "", "", limit), nil
}

func (l *blockingEventLog) EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error) {
	return l.query(taskID, "", afterID, limit), nil
}

func (l *blockingEventLog) EventsByTrace(traceID string, limit int) ([]domain.Event, error) {
	return l.query("", traceID, "", limit), nil
}

func (l *blockingEventLog) EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error) {
	return l.query("", traceID, afterID, limit), nil
}

func (l *blockingEventLog) Backend() string {
	return "memory"
}

func (l *blockingEventLog) Capabilities() events.BackendCapabilities {
	return events.BackendCapabilities{
		Backend:    "memory",
		Scope:      "test_double",
		Publish:    events.FeatureSupport{Supported: true, Mode: "append_only"},
		Replay:     events.FeatureSupport{Supported: true, Mode: "durable"},
		Checkpoint: events.FeatureSupport{Supported: true, Mode: "subscriber_ack"},
		Dedup:      events.FeatureSupport{Supported: true, Mode: "sqlite"},
		Filtering:  events.FeatureSupport{Supported: true, Mode: "server_side"},
		Retention:  events.FeatureSupport{Supported: true, Mode: "test_memory"},
	}
}

func (l *blockingEventLog) Path() string {
	return "blocking"
}

func (l *blockingEventLog) Close() error {
	return nil
}

type blockingRemoteServiceLog struct {
	*blockingEventLog
}

func (l *blockingRemoteServiceLog) Acknowledge(string, string, time.Time) (events.SubscriberCheckpoint, error) {
	return events.SubscriberCheckpoint{}, errors.New("checkpoint ack unavailable in blocking replay test double")
}

func (l *blockingRemoteServiceLog) Checkpoint(string) (events.SubscriberCheckpoint, error) {
	return events.SubscriberCheckpoint{}, errors.New("checkpoint unavailable in blocking replay test double")
}

func (l *blockingEventLog) query(taskID, traceID, afterID string, limit int) []domain.Event {
	select {
	case l.replayStarted <- struct{}{}:
	default:
	}
	<-l.release
	start := 0
	if afterID != "" {
		for index, event := range l.history {
			if event.ID == afterID {
				start = index + 1
			}
		}
	}
	filtered := make([]domain.Event, 0, len(l.history)-start)
	for _, event := range l.history[start:] {
		if taskID != "" && event.TaskID != taskID {
			continue
		}
		if traceID != "" && event.TraceID != traceID {
			continue
		}
		filtered = append(filtered, event)
		if afterID != "" && limit > 0 && len(filtered) >= limit {
			break
		}
	}
	if afterID == "" && limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	out := make([]domain.Event, len(filtered))
	copy(out, filtered)
	return out
}

func TestCreateTaskAcceptsLegacyTaskIDPayload(t *testing.T) {
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

	payload := map[string]any{"task_id": "task-legacy-1", "title": "legacy hello", "budget_override_actor": "lead", "budget_override_reason": "approved", "budget_override_amount": 7.5}
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}

	var decoded struct {
		Task domain.Task `json:"task"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Task.ID != "task-legacy-1" || decoded.Task.TraceID != "task-legacy-1" {
		t.Fatalf("expected legacy task id to hydrate task identity, got %+v", decoded.Task)
	}
	if decoded.Task.BudgetOverrideActor != "lead" || decoded.Task.BudgetOverrideReason != "approved" || decoded.Task.BudgetOverrideAmount != 7.5 {
		t.Fatalf("expected budget override fields in response, got %+v", decoded.Task)
	}
}

func TestCreateTaskAcceptsLegacyBudgetPayload(t *testing.T) {
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

	payload := map[string]any{"task_id": "task-legacy-budget-1", "title": "legacy budget", "budget": 12.34}
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}

	var decoded struct {
		Task domain.Task `json:"task"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Task.ID != "task-legacy-budget-1" || decoded.Task.BudgetCents != 1234 {
		t.Fatalf("expected legacy budget payload to hydrate canonical task budget, got %+v", decoded.Task)
	}
}

func TestCreateTaskAcceptsLegacyStatePayload(t *testing.T) {
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

	payload := map[string]any{"task_id": "task-legacy-state-1", "title": "legacy state", "state": "In Progress"}
	body, _ := json.Marshal(payload)
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", response.Code)
	}

	var decoded struct {
		Task domain.Task `json:"task"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if decoded.Task.State != domain.TaskQueued {
		t.Fatalf("expected create-task runtime to queue accepted legacy state payloads, got %+v", decoded.Task)
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
	var eventsDecoded struct {
		Events []domain.Event `json:"events"`
	}
	if err := json.Unmarshal(eventsResponse.Body.Bytes(), &eventsDecoded); err != nil {
		t.Fatalf("decode events response: %v", err)
	}
	if len(eventsDecoded.Events) != 1 || eventsDecoded.Events[0].Delivery == nil || eventsDecoded.Events[0].Delivery.Mode != domain.EventDeliveryModeReplay || eventsDecoded.Events[0].Delivery.IdempotencyKey != "task-api-1-queued" {
		t.Fatalf("expected replay-safe event metadata on history endpoint, got %+v", eventsDecoded.Events)
	}
}

func TestAuditAndReplayEndpoints(t *testing.T) {
	recorder := observability.NewRecorder()
	recorder.Record(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", Timestamp: time.Now()})
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		EventPlan: events.NewDurabilityPlan("http", "broker_replicated", 3),
		Now:       time.Now,
	}
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
	var replayDecoded struct {
		TaskID           string         `json:"task_id"`
		Timeline         []domain.Event `json:"timeline"`
		ConsumerContract struct {
			EventIDField         string `json:"event_id_field"`
			IdempotencyKeyField  string `json:"idempotency_key_field"`
			ReplayIndicatorField string `json:"replay_indicator_field"`
			DeliveryModeField    string `json:"delivery_mode_field"`
		} `json:"consumer_contract"`
	}
	if err := json.Unmarshal(replayResponse.Body.Bytes(), &replayDecoded); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if replayDecoded.TaskID != "task-1" || len(replayDecoded.Timeline) != 1 {
		t.Fatalf("expected replay timeline for task-1, got %+v", replayDecoded)
	}
	if replayDecoded.Timeline[0].Delivery == nil || replayDecoded.Timeline[0].Delivery.Mode != domain.EventDeliveryModeReplay || replayDecoded.Timeline[0].Delivery.IdempotencyKey != "evt-1" {
		t.Fatalf("expected replay delivery metadata, got %+v", replayDecoded.Timeline[0].Delivery)
	}
	if replayDecoded.ConsumerContract.IdempotencyKeyField != "delivery.idempotency_key" || replayDecoded.ConsumerContract.ReplayIndicatorField != "delivery.replay" || replayDecoded.ConsumerContract.DeliveryModeField != "delivery.mode" || replayDecoded.ConsumerContract.EventIDField != "id" {
		t.Fatalf("unexpected consumer contract: %+v", replayDecoded.ConsumerContract)
	}
}

func TestDebugStatusIncludesEventDurabilityPlan(t *testing.T) {
	recorder := observability.NewRecorder()
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		EventPlan: events.NewDurabilityPlan("http", "broker_replicated", 5),
		Now:       time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		EventDurability struct {
			Current struct {
				Backend string `json:"backend"`
			} `json:"current"`
			Target struct {
				Backend string `json:"backend"`
			} `json:"target"`
			ReplicationFactor int `json:"replication_factor"`
			RolloutChecks     []struct {
				Name string `json:"name"`
			} `json:"rollout_checks"`
			FailureDomains []struct {
				Name string `json:"name"`
			} `json:"failure_domains"`
			VerificationEvidence []struct {
				Name string `json:"name"`
			} `json:"verification_evidence"`
			RolloutScorecard struct {
				Status          string   `json:"status"`
				RolloutReady    bool     `json:"rollout_ready"`
				CurrentBackend  string   `json:"current_backend"`
				TargetBackend   string   `json:"target_backend"`
				ReadyEvidence   int      `json:"ready_evidence"`
				PartialEvidence int      `json:"partial_evidence"`
				BlockedEvidence int      `json:"blocked_evidence"`
				Blockers        []string `json:"blockers"`
			} `json:"rollout_scorecard"`
		} `json:"event_durability"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug status: %v", err)
	}
	if decoded.EventDurability.Current.Backend != "http" || decoded.EventDurability.Target.Backend != "broker_replicated" {
		t.Fatalf("unexpected event durability backends: %+v", decoded.EventDurability)
	}
	if decoded.EventDurability.ReplicationFactor != 5 {
		t.Fatalf("expected replication factor 5, got %+v", decoded.EventDurability)
	}
	if len(decoded.EventDurability.RolloutChecks) == 0 || len(decoded.EventDurability.FailureDomains) == 0 || len(decoded.EventDurability.VerificationEvidence) == 0 {
		t.Fatalf("expected rollout contract details in payload, got %+v", decoded.EventDurability)
	}
	if decoded.EventDurability.RolloutScorecard.Status != "blocked" || decoded.EventDurability.RolloutScorecard.RolloutReady {
		t.Fatalf("expected blocked rollout scorecard, got %+v", decoded.EventDurability.RolloutScorecard)
	}
	if decoded.EventDurability.RolloutScorecard.CurrentBackend != "http" || decoded.EventDurability.RolloutScorecard.TargetBackend != "broker_replicated" {
		t.Fatalf("unexpected rollout scorecard backend payload: %+v", decoded.EventDurability.RolloutScorecard)
	}
	if decoded.EventDurability.RolloutScorecard.ReadyEvidence != 3 || decoded.EventDurability.RolloutScorecard.PartialEvidence != 0 || decoded.EventDurability.RolloutScorecard.BlockedEvidence != 1 {
		t.Fatalf("unexpected rollout scorecard evidence counts: %+v", decoded.EventDurability.RolloutScorecard)
	}
	if len(decoded.EventDurability.RolloutScorecard.Blockers) != 2 {
		t.Fatalf("expected rollout blockers, got %+v", decoded.EventDurability.RolloutScorecard)
	}
	if !strings.Contains(response.Body.String(), "\"event_durability_rollout\"") {
		t.Fatalf("expected rollout scorecard in payload, got %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "\"rollout_ready\":false") {
		t.Fatalf("expected rollout readiness flag in payload, got %s", response.Body.String())
	}
}

func TestDebugStatusIncludesAdmissionPolicySummary(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		AdmissionPolicy struct {
			ReportPath string `json:"report_path"`
			MatrixPath string `json:"matrix_path"`
			Ticket     string `json:"ticket"`
			Status     string `json:"status"`
			PolicyMode string `json:"policy_mode"`
			Enforced   bool   `json:"enforced"`
			Summary    struct {
				OverallStatus                string `json:"overall_status"`
				PassedLanes                  int    `json:"passed_lanes"`
				TotalLanes                   int    `json:"total_lanes"`
				RecommendedSustainedEnvelope string `json:"recommended_sustained_envelope"`
				CeilingEnvelope              string `json:"ceiling_envelope"`
				AdvisoryNote                 string `json:"advisory_note"`
			} `json:"summary"`
			RecommendedLanes []struct {
				Name                  string   `json:"name"`
				Lane                  string   `json:"lane"`
				MaxQueuedTasks        int      `json:"max_queued_tasks"`
				SubmitWorkers         int      `json:"submit_workers"`
				ObservedThroughputTPS float64  `json:"observed_throughput_tasks_per_sec"`
				EvidenceLanes         []string `json:"evidence_lanes"`
				DefaultRecommendation bool     `json:"default_recommendation"`
				CeilingOnly           bool     `json:"ceiling_only"`
			} `json:"recommended_lanes"`
			SupportingEvidence []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"supporting_evidence"`
			Saturation struct {
				BaselineLane      string  `json:"baseline_lane"`
				CeilingLane       string  `json:"ceiling_lane"`
				ThroughputDropPct float64 `json:"throughput_drop_pct"`
				Status            string  `json:"status"`
			} `json:"saturation"`
		} `json:"admission_policy_summary"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug admission policy payload: %v", err)
	}
	if decoded.AdmissionPolicy.ReportPath != capacityCertificationReportPath || decoded.AdmissionPolicy.MatrixPath != capacityCertificationMatrixPath {
		t.Fatalf("unexpected admission policy report metadata: %+v", decoded.AdmissionPolicy)
	}
	if decoded.AdmissionPolicy.Ticket != "BIG-PAR-098" || decoded.AdmissionPolicy.Status != "repo-native-capacity-certification" {
		t.Fatalf("unexpected admission policy status metadata: %+v", decoded.AdmissionPolicy)
	}
	if decoded.AdmissionPolicy.PolicyMode != "advisory_only" || decoded.AdmissionPolicy.Enforced {
		t.Fatalf("expected advisory-only non-enforced admission policy, got %+v", decoded.AdmissionPolicy)
	}
	if decoded.AdmissionPolicy.Summary.OverallStatus != "pass" || decoded.AdmissionPolicy.Summary.PassedLanes != 9 || decoded.AdmissionPolicy.Summary.TotalLanes != 9 {
		t.Fatalf("unexpected admission policy summary counts: %+v", decoded.AdmissionPolicy.Summary)
	}
	if decoded.AdmissionPolicy.Summary.RecommendedSustainedEnvelope != "<=1000 tasks with 24 submit workers" || decoded.AdmissionPolicy.Summary.CeilingEnvelope != "<=2000 tasks with 24 submit workers" {
		t.Fatalf("unexpected admission policy envelope summary: %+v", decoded.AdmissionPolicy.Summary)
	}
	if !strings.Contains(decoded.AdmissionPolicy.Summary.AdvisoryNote, "not an automated runtime admission policy") {
		t.Fatalf("expected advisory note in admission policy summary, got %+v", decoded.AdmissionPolicy.Summary)
	}
	if len(decoded.AdmissionPolicy.RecommendedLanes) != 2 {
		t.Fatalf("expected 2 recommended lanes, got %+v", decoded.AdmissionPolicy.RecommendedLanes)
	}
	if decoded.AdmissionPolicy.RecommendedLanes[0].Name != "recommended-local-sustained" || decoded.AdmissionPolicy.RecommendedLanes[0].Lane != "1000x24" || decoded.AdmissionPolicy.RecommendedLanes[0].MaxQueuedTasks != 1000 || decoded.AdmissionPolicy.RecommendedLanes[0].SubmitWorkers != 24 || !decoded.AdmissionPolicy.RecommendedLanes[0].DefaultRecommendation || decoded.AdmissionPolicy.RecommendedLanes[0].CeilingOnly {
		t.Fatalf("unexpected default admission lane: %+v", decoded.AdmissionPolicy.RecommendedLanes[0])
	}
	if decoded.AdmissionPolicy.RecommendedLanes[1].Name != "recommended-local-ceiling" || decoded.AdmissionPolicy.RecommendedLanes[1].Lane != "2000x24" || decoded.AdmissionPolicy.RecommendedLanes[1].MaxQueuedTasks != 2000 || !decoded.AdmissionPolicy.RecommendedLanes[1].CeilingOnly {
		t.Fatalf("unexpected ceiling admission lane: %+v", decoded.AdmissionPolicy.RecommendedLanes[1])
	}
	if len(decoded.AdmissionPolicy.SupportingEvidence) != 1 || decoded.AdmissionPolicy.SupportingEvidence[0].Name != "mixed-workload-routing" || decoded.AdmissionPolicy.SupportingEvidence[0].Status != "pass" {
		t.Fatalf("unexpected supporting admission evidence: %+v", decoded.AdmissionPolicy.SupportingEvidence)
	}
	if decoded.AdmissionPolicy.Saturation.BaselineLane != "1000x24" || decoded.AdmissionPolicy.Saturation.CeilingLane != "2000x24" || decoded.AdmissionPolicy.Saturation.ThroughputDropPct != 5.02 || decoded.AdmissionPolicy.Saturation.Status != "pass" {
		t.Fatalf("unexpected admission policy saturation summary: %+v", decoded.AdmissionPolicy.Saturation)
	}
}

func TestDebugStatusIncludesCoordinationCapabilitySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		Coordination struct {
			ReportPath string `json:"report_path"`
			Status     string `json:"status"`
			Summary    struct {
				CapabilityCount        int            `json:"capability_count"`
				ContractOnlyCount      int            `json:"contract_only_count"`
				HarnessProvenCount     int            `json:"harness_proven_count"`
				LiveProvenCount        int            `json:"live_proven_count"`
				CurrentStateCounts     map[string]int `json:"current_state_counts"`
				RuntimeReadinessCounts map[string]int `json:"runtime_readiness_counts"`
			} `json:"summary"`
			Capabilities []struct {
				Name              string   `json:"name"`
				CurrentState      string   `json:"current_state"`
				RuntimeReadiness  string   `json:"runtime_readiness"`
				ContractOnly      bool     `json:"contract_only"`
				HarnessProven     bool     `json:"harness_proven"`
				LiveProven        bool     `json:"live_proven"`
				SourceReportLinks []string `json:"source_report_links"`
			} `json:"capabilities"`
		} `json:"coordination_capability_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug coordination payload: %v", err)
	}
	if decoded.Coordination.ReportPath != coordinationCapabilitySurfacePath || decoded.Coordination.Status != "local-capability-surface" {
		t.Fatalf("unexpected coordination report metadata: %+v", decoded.Coordination)
	}
	if decoded.Coordination.Summary.CapabilityCount != 7 || decoded.Coordination.Summary.ContractOnlyCount != 2 || decoded.Coordination.Summary.HarnessProvenCount != 1 || decoded.Coordination.Summary.LiveProvenCount != 3 {
		t.Fatalf("unexpected coordination summary: %+v", decoded.Coordination.Summary)
	}
	if decoded.Coordination.Summary.CurrentStateCounts["contract_defined"] != 1 || decoded.Coordination.Summary.CurrentStateCounts["not_available"] != 2 {
		t.Fatalf("unexpected coordination state counts: %+v", decoded.Coordination.Summary.CurrentStateCounts)
	}
	if decoded.Coordination.Summary.RuntimeReadinessCounts["supporting_surface"] != 1 || decoded.Coordination.Summary.RuntimeReadinessCounts["live_proven"] != 3 {
		t.Fatalf("unexpected coordination readiness counts: %+v", decoded.Coordination.Summary.RuntimeReadinessCounts)
	}
	if len(decoded.Coordination.Capabilities) == 0 {
		t.Fatalf("expected capability entries in debug status payload")
	}
	first := decoded.Coordination.Capabilities[0]
	if first.Name != "shared_queue_task_coordination" || first.CurrentState != "implemented" || first.RuntimeReadiness != "live_proven" || first.ContractOnly || !first.LiveProven || len(first.SourceReportLinks) == 0 {
		t.Fatalf("unexpected first capability payload: %+v", first)
	}
}

func TestDebugStatusIncludesCoordinationLeaderElectionSurface(t *testing.T) {
	server := &Server{
		Recorder:         observability.NewRecorder(),
		Queue:            queue.NewMemoryQueue(),
		Bus:              events.NewBus(),
		SubscriberLeases: events.NewSubscriberLeaseCoordinator(),
		Now:              time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		Leader struct {
			Endpoint      string `json:"endpoint"`
			GroupID       string `json:"group_id"`
			SubscriberID  string `json:"subscriber_id"`
			ElectionModel string `json:"election_model"`
			Status        string `json:"status"`
			LeaderPresent bool   `json:"leader_present"`
		} `json:"coordination_leader_election"`
		LeaderElectionCapability struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				BackendCount        int    `json:"backend_count"`
				CurrentProofBackend string `json:"current_proof_backend"`
			} `json:"summary"`
		} `json:"leader_election_capability"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug leader surface: %v", err)
	}
	if decoded.Leader.Endpoint != coordinationLeaderEndpoint || decoded.Leader.GroupID != coordinationLeaderGroupID || decoded.Leader.SubscriberID != coordinationLeaderSubscriberID {
		t.Fatalf("unexpected leader election identity: %+v", decoded.Leader)
	}
	if decoded.Leader.ElectionModel != "subscriber_lease" || decoded.Leader.Status != "idle" || decoded.Leader.LeaderPresent {
		t.Fatalf("unexpected leader election surface: %+v", decoded.Leader)
	}
	if decoded.LeaderElectionCapability.ReportPath != leaderElectionCapabilitySurfacePath || decoded.LeaderElectionCapability.Summary.BackendCount != 4 || decoded.LeaderElectionCapability.Summary.CurrentProofBackend != "shared_sqlite_subscriber_lease" {
		t.Fatalf("unexpected leader election capability payload: %+v", decoded.LeaderElectionCapability)
	}
}

func TestDebugStatusIncludesBrokerStubFanoutIsolationEvidencePack(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		BrokerStubFanoutIsolation struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Summary    struct {
				ScenarioCount          int  `json:"scenario_count"`
				IsolatedScenarios      int  `json:"isolated_scenarios"`
				StalledScenarios       int  `json:"stalled_scenarios"`
				ReplayBacklogEvents    int  `json:"replay_backlog_events"`
				ReplayStepDelayMS      int  `json:"replay_step_delay_ms"`
				ReplayWindowMS         int  `json:"replay_window_ms"`
				LiveDeliveryDeadlineMS int  `json:"live_delivery_deadline_ms"`
				IsolationMaintained    bool `json:"isolation_maintained"`
			} `json:"summary"`
			Scenarios []struct {
				Name                   string `json:"name"`
				Status                 string `json:"status"`
				ReplayBacklogEvents    int    `json:"replay_backlog_events"`
				ReplayStepDelayMS      int    `json:"replay_step_delay_ms"`
				ReplayWindowMS         int    `json:"replay_window_ms"`
				LiveDeliveryDeadlineMS int    `json:"live_delivery_deadline_ms"`
				ReplayDrainsAfterLive  bool   `json:"replay_drains_after_live"`
			} `json:"scenarios"`
		} `json:"broker_stub_fanout_isolation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug broker fanout payload: %v", err)
	}
	if decoded.BrokerStubFanoutIsolation.ReportPath != brokerStubFanoutIsolationEvidencePackPath || decoded.BrokerStubFanoutIsolation.Ticket != "OPE-261" {
		t.Fatalf("unexpected broker fanout report metadata: %+v", decoded.BrokerStubFanoutIsolation)
	}
	if decoded.BrokerStubFanoutIsolation.Summary.ScenarioCount != 1 || decoded.BrokerStubFanoutIsolation.Summary.IsolatedScenarios != 1 || decoded.BrokerStubFanoutIsolation.Summary.StalledScenarios != 0 || decoded.BrokerStubFanoutIsolation.Summary.ReplayBacklogEvents != 4 || decoded.BrokerStubFanoutIsolation.Summary.ReplayStepDelayMS != 30 || decoded.BrokerStubFanoutIsolation.Summary.ReplayWindowMS != 120 || decoded.BrokerStubFanoutIsolation.Summary.LiveDeliveryDeadlineMS != 50 || !decoded.BrokerStubFanoutIsolation.Summary.IsolationMaintained {
		t.Fatalf("unexpected broker fanout summary: %+v", decoded.BrokerStubFanoutIsolation.Summary)
	}
	if len(decoded.BrokerStubFanoutIsolation.Scenarios) != 1 {
		t.Fatalf("expected 1 broker fanout scenario, got %+v", decoded.BrokerStubFanoutIsolation.Scenarios)
	}
	if decoded.BrokerStubFanoutIsolation.Scenarios[0].Name != "replay_catchup_does_not_block_live_publish" || decoded.BrokerStubFanoutIsolation.Scenarios[0].Status != "isolated" || decoded.BrokerStubFanoutIsolation.Scenarios[0].ReplayBacklogEvents != 4 || decoded.BrokerStubFanoutIsolation.Scenarios[0].ReplayStepDelayMS != 30 || decoded.BrokerStubFanoutIsolation.Scenarios[0].ReplayWindowMS != 120 || decoded.BrokerStubFanoutIsolation.Scenarios[0].LiveDeliveryDeadlineMS != 50 || !decoded.BrokerStubFanoutIsolation.Scenarios[0].ReplayDrainsAfterLive {
		t.Fatalf("unexpected broker fanout scenario: %+v", decoded.BrokerStubFanoutIsolation.Scenarios[0])
	}
}

func TestDebugStatusIncludesLiveShadowMirrorScorecard(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		LiveShadow struct {
			ReportPath           string   `json:"report_path"`
			CanonicalSummaryPath string   `json:"canonical_summary_path"`
			Status               string   `json:"status"`
			Severity             string   `json:"severity"`
			BundlePath           string   `json:"bundle_path"`
			SummaryPath          string   `json:"summary_path"`
			ReviewerLinks        []string `json:"reviewer_links"`
			Summary              struct {
				TotalEvidenceRuns  int `json:"total_evidence_runs"`
				ParityOKCount      int `json:"parity_ok_count"`
				DriftDetectedCount int `json:"drift_detected_count"`
				MatrixMismatched   int `json:"matrix_mismatched"`
				FreshInputs        int `json:"fresh_inputs"`
				StaleInputs        int `json:"stale_inputs"`
			} `json:"summary"`
			CutoverCheckpoints []struct {
				Name   string `json:"name"`
				Passed bool   `json:"passed"`
			} `json:"cutover_checkpoints"`
			RollbackTriggerSurface struct {
				Status                   string `json:"status"`
				AutomationBoundary       string `json:"automation_boundary"`
				AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
				SummaryPath              string `json:"summary_path"`
			} `json:"rollback_trigger_surface"`
			Limitations []string `json:"limitations"`
			FutureWork  []string `json:"future_work"`
		} `json:"live_shadow_mirror_scorecard"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug status: %v", err)
	}
	if decoded.LiveShadow.ReportPath != liveShadowMirrorScorecardPath || decoded.LiveShadow.CanonicalSummaryPath != liveShadowSummaryPath {
		t.Fatalf("unexpected live shadow report paths: %+v", decoded.LiveShadow)
	}
	if decoded.LiveShadow.Status != "parity-ok" || decoded.LiveShadow.Severity != "none" {
		t.Fatalf("unexpected live shadow status: %+v", decoded.LiveShadow)
	}
	if decoded.LiveShadow.BundlePath != "docs/reports/live-shadow-runs/20260313T085655Z" || decoded.LiveShadow.SummaryPath != "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" {
		t.Fatalf("unexpected live shadow bundle payload: %+v", decoded.LiveShadow)
	}
	if decoded.LiveShadow.Summary.TotalEvidenceRuns != 4 || decoded.LiveShadow.Summary.ParityOKCount != 4 || decoded.LiveShadow.Summary.DriftDetectedCount != 0 || decoded.LiveShadow.Summary.MatrixMismatched != 0 || decoded.LiveShadow.Summary.FreshInputs != 2 || decoded.LiveShadow.Summary.StaleInputs != 0 {
		t.Fatalf("unexpected live shadow summary payload: %+v", decoded.LiveShadow.Summary)
	}
	if len(decoded.LiveShadow.CutoverCheckpoints) != 5 || !decoded.LiveShadow.CutoverCheckpoints[0].Passed {
		t.Fatalf("unexpected cutover checkpoint payload: %+v", decoded.LiveShadow.CutoverCheckpoints)
	}
	if decoded.LiveShadow.RollbackTriggerSurface.Status != "manual-review-required" || decoded.LiveShadow.RollbackTriggerSurface.AutomationBoundary != "manual_only" || decoded.LiveShadow.RollbackTriggerSurface.AutomatedRollbackTrigger || decoded.LiveShadow.RollbackTriggerSurface.SummaryPath != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected rollback trigger surface payload: %+v", decoded.LiveShadow.RollbackTriggerSurface)
	}
	if len(decoded.LiveShadow.ReviewerLinks) == 0 || decoded.LiveShadow.ReviewerLinks[0] != liveShadowSummaryPath || decoded.LiveShadow.ReviewerLinks[len(decoded.LiveShadow.ReviewerLinks)-1] != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected reviewer links: %+v", decoded.LiveShadow.ReviewerLinks)
	}
	if len(decoded.LiveShadow.Limitations) == 0 || len(decoded.LiveShadow.FutureWork) == 0 {
		t.Fatalf("expected scorecard caveats and future work, got %+v", decoded.LiveShadow)
	}
	if !strings.Contains(response.Body.String(), "\"live_shadow_mirror_scorecard\"") || !strings.Contains(response.Body.String(), "\"canonical_summary_path\":\"docs/reports/live-shadow-summary.json\"") {
		t.Fatalf("expected live shadow payload in response, got %s", response.Body.String())
	}
}

func TestDebugStatusIncludesRollbackTriggerSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		RollbackTrigger struct {
			ReportPath string `json:"report_path"`
			Issue      struct {
				ID   string `json:"id"`
				Slug string `json:"slug"`
			} `json:"issue"`
			Summary struct {
				Status                   string `json:"status"`
				AutomationBoundary       string `json:"automation_boundary"`
				AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
				CutoverGate              string `json:"cutover_gate"`
				Distinctions             struct {
					Blockers        int `json:"blockers"`
					Warnings        int `json:"warnings"`
					ManualOnlyPaths int `json:"manual_only_paths"`
				} `json:"distinctions"`
			} `json:"summary"`
			SharedGuardrailSummary struct {
				DigestPath             string `json:"digest_path"`
				MigrationReadinessPath string `json:"migration_readiness_path"`
				LiveShadowIndexPath    string `json:"live_shadow_index_path"`
				LiveShadowRollupPath   string `json:"live_shadow_rollup_path"`
			} `json:"shared_guardrail_summary"`
			Warnings        []map[string]any `json:"warnings"`
			Blockers        []map[string]any `json:"blockers"`
			ManualOnlyPaths []map[string]any `json:"manual_only_paths"`
			ReviewerLinks   []string         `json:"reviewer_links"`
		} `json:"rollback_trigger_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug rollback payload: %v", err)
	}
	if decoded.RollbackTrigger.ReportPath != rollbackTriggerSurfacePath || decoded.RollbackTrigger.Issue.ID != "OPE-254" || decoded.RollbackTrigger.Issue.Slug != "BIG-PAR-088" {
		t.Fatalf("unexpected rollback trigger metadata: %+v", decoded.RollbackTrigger)
	}
	if decoded.RollbackTrigger.Summary.Status != "manual-review-required" || decoded.RollbackTrigger.Summary.AutomationBoundary != "manual_only" || decoded.RollbackTrigger.Summary.AutomatedRollbackTrigger || decoded.RollbackTrigger.Summary.CutoverGate != "reviewer_enforced" {
		t.Fatalf("unexpected rollback trigger summary: %+v", decoded.RollbackTrigger.Summary)
	}
	if decoded.RollbackTrigger.Summary.Distinctions.Blockers != 3 || decoded.RollbackTrigger.Summary.Distinctions.Warnings != 1 || decoded.RollbackTrigger.Summary.Distinctions.ManualOnlyPaths != 2 {
		t.Fatalf("unexpected rollback distinctions: %+v", decoded.RollbackTrigger.Summary.Distinctions)
	}
	if decoded.RollbackTrigger.SharedGuardrailSummary.DigestPath != "docs/reports/rollback-safeguard-follow-up-digest.md" || decoded.RollbackTrigger.SharedGuardrailSummary.MigrationReadinessPath != migrationReadinessReportPath || decoded.RollbackTrigger.SharedGuardrailSummary.LiveShadowIndexPath != liveShadowIndexPath || decoded.RollbackTrigger.SharedGuardrailSummary.LiveShadowRollupPath != "docs/reports/live-shadow-drift-rollup.json" {
		t.Fatalf("unexpected rollback guardrail summary: %+v", decoded.RollbackTrigger.SharedGuardrailSummary)
	}
	if len(decoded.RollbackTrigger.Warnings) != 1 || len(decoded.RollbackTrigger.Blockers) != 3 || len(decoded.RollbackTrigger.ManualOnlyPaths) != 2 {
		t.Fatalf("unexpected rollback collections: %+v", decoded.RollbackTrigger)
	}
	if len(decoded.RollbackTrigger.ReviewerLinks) == 0 || decoded.RollbackTrigger.ReviewerLinks[0] != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected rollback reviewer links: %+v", decoded.RollbackTrigger.ReviewerLinks)
	}
	if !strings.Contains(response.Body.String(), "\"rollback_trigger_surface\"") || !strings.Contains(response.Body.String(), "\"cutover_gate\":\"reviewer_enforced\"") {
		t.Fatalf("expected rollback trigger payload in response, got %s", response.Body.String())
	}
}

func TestDebugStatusIncludesSequenceBridgeSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		SequenceBridge struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Track      string `json:"track"`
			Summary    struct {
				BackendCount                 int `json:"backend_count"`
				LiveProvenBackends           int `json:"live_proven_backends"`
				HarnessProvenBackends        int `json:"harness_proven_backends"`
				ContractOnlyBackends         int `json:"contract_only_backends"`
				OneToOneMappings             int `json:"one_to_one_mappings"`
				ProviderEpochBridgedBackends int `json:"provider_epoch_bridged_backends"`
			} `json:"summary"`
			Backends []struct {
				Backend          string `json:"backend"`
				RuntimeReadiness string `json:"runtime_readiness"`
			} `json:"backends"`
		} `json:"sequence_bridge_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode sequence bridge payload: %v", err)
	}
	if decoded.SequenceBridge.ReportPath != sequenceBridgeSurfacePath || decoded.SequenceBridge.Ticket != "OPE-12" || decoded.SequenceBridge.Track != "BIG-DUR-102" {
		t.Fatalf("unexpected sequence bridge metadata: %+v", decoded.SequenceBridge)
	}
	if decoded.SequenceBridge.Summary.BackendCount != 5 || decoded.SequenceBridge.Summary.LiveProvenBackends != 3 || decoded.SequenceBridge.Summary.HarnessProvenBackends != 1 || decoded.SequenceBridge.Summary.ContractOnlyBackends != 1 || decoded.SequenceBridge.Summary.OneToOneMappings != 2 || decoded.SequenceBridge.Summary.ProviderEpochBridgedBackends != 3 {
		t.Fatalf("unexpected sequence bridge summary: %+v", decoded.SequenceBridge.Summary)
	}
	if len(decoded.SequenceBridge.Backends) != 5 || decoded.SequenceBridge.Backends[0].Backend != "memory" || decoded.SequenceBridge.Backends[4].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected sequence bridge backends: %+v", decoded.SequenceBridge.Backends)
	}
}

func TestDebugStatusIncludesPublishAckOutcomeSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		PublishAckOutcomes struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Track      string `json:"track"`
			Summary    struct {
				ScenarioID         string   `json:"scenario_id"`
				ProofStatus        string   `json:"proof_status"`
				RequiredOutcomes   []string `json:"required_outcomes"`
				CommittedCount     int      `json:"committed_count"`
				RejectedCount      int      `json:"rejected_count"`
				UnknownCommitCount int      `json:"unknown_commit_count"`
			} `json:"summary"`
			Outcomes []struct {
				Outcome string `json:"outcome"`
			} `json:"outcomes"`
		} `json:"publish_ack_outcomes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode publish ack surface payload: %v", err)
	}
	if decoded.PublishAckOutcomes.ReportPath != publishAckOutcomeSurfacePath || decoded.PublishAckOutcomes.Ticket != "OPE-5" || decoded.PublishAckOutcomes.Track != "BIG-DUR-101" {
		t.Fatalf("unexpected publish ack report metadata: %+v", decoded.PublishAckOutcomes)
	}
	if decoded.PublishAckOutcomes.Summary.ScenarioID != "BF-05" || decoded.PublishAckOutcomes.Summary.ProofStatus != "repo-proof-summary" {
		t.Fatalf("unexpected publish ack summary metadata: %+v", decoded.PublishAckOutcomes.Summary)
	}
	if decoded.PublishAckOutcomes.Summary.CommittedCount != 1 || decoded.PublishAckOutcomes.Summary.RejectedCount != 1 || decoded.PublishAckOutcomes.Summary.UnknownCommitCount != 1 {
		t.Fatalf("unexpected publish ack counts: %+v", decoded.PublishAckOutcomes.Summary)
	}
	if len(decoded.PublishAckOutcomes.Summary.RequiredOutcomes) != 3 || len(decoded.PublishAckOutcomes.Outcomes) != 3 {
		t.Fatalf("unexpected publish ack outcomes payload: %+v", decoded.PublishAckOutcomes)
	}
}

func TestDebugStatusIncludesDeliveryAckReadinessSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		DeliveryAckReadiness struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Summary    struct {
				BackendCount         int `json:"backend_count"`
				ExplicitAckBackends  int `json:"explicit_ack_backends"`
				DurableAckBackends   int `json:"durable_ack_backends"`
				BestEffortBackends   int `json:"best_effort_backends"`
				ContractOnlyBackends int `json:"contract_only_backends"`
			} `json:"summary"`
			Backends []struct {
				Backend                 string `json:"backend"`
				AcknowledgementClass    string `json:"acknowledgement_class"`
				ExplicitAcknowledgement bool   `json:"explicit_acknowledgement"`
				DurableAcknowledgement  bool   `json:"durable_acknowledgement"`
				RuntimeReadiness        string `json:"runtime_readiness"`
			} `json:"backends"`
		} `json:"delivery_ack_readiness"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug ack readiness payload: %v", err)
	}
	if decoded.DeliveryAckReadiness.ReportPath != deliveryAckReadinessSurfacePath || decoded.DeliveryAckReadiness.Ticket != "OPE-264" {
		t.Fatalf("unexpected delivery ack report metadata: %+v", decoded.DeliveryAckReadiness)
	}
	if decoded.DeliveryAckReadiness.Summary.BackendCount != 5 || decoded.DeliveryAckReadiness.Summary.ExplicitAckBackends != 3 || decoded.DeliveryAckReadiness.Summary.DurableAckBackends != 2 || decoded.DeliveryAckReadiness.Summary.BestEffortBackends != 1 || decoded.DeliveryAckReadiness.Summary.ContractOnlyBackends != 1 {
		t.Fatalf("unexpected delivery ack summary: %+v", decoded.DeliveryAckReadiness.Summary)
	}
	if len(decoded.DeliveryAckReadiness.Backends) != 5 {
		t.Fatalf("expected 5 ack readiness backends, got %+v", decoded.DeliveryAckReadiness.Backends)
	}
	if decoded.DeliveryAckReadiness.Backends[0].Backend != "memory" || decoded.DeliveryAckReadiness.Backends[0].AcknowledgementClass != "best_effort_only" || decoded.DeliveryAckReadiness.Backends[0].ExplicitAcknowledgement || decoded.DeliveryAckReadiness.Backends[0].DurableAcknowledgement {
		t.Fatalf("unexpected memory ack readiness payload: %+v", decoded.DeliveryAckReadiness.Backends[0])
	}
	if decoded.DeliveryAckReadiness.Backends[1].Backend != "sqlite" || !decoded.DeliveryAckReadiness.Backends[1].ExplicitAcknowledgement || !decoded.DeliveryAckReadiness.Backends[1].DurableAcknowledgement {
		t.Fatalf("unexpected sqlite ack readiness payload: %+v", decoded.DeliveryAckReadiness.Backends[1])
	}
	if decoded.DeliveryAckReadiness.Backends[4].Backend != "broker_replicated" || decoded.DeliveryAckReadiness.Backends[4].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected broker replicated ack readiness payload: %+v", decoded.DeliveryAckReadiness.Backends[4])
	}
}

func TestDebugStatusIncludesValidationBundleContinuationGate(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		ValidationBundleContinuation struct {
			ReportPath     string   `json:"report_path"`
			ScorecardPath  string   `json:"scorecard_path"`
			DigestPath     string   `json:"digest_path"`
			Ticket         string   `json:"ticket"`
			Status         string   `json:"status"`
			Recommendation string   `json:"recommendation"`
			ReviewerLinks  []string `json:"reviewer_links"`
			Summary        struct {
				LatestRunID                                 string  `json:"latest_run_id"`
				LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours"`
				RecentBundleCount                           int     `json:"recent_bundle_count"`
				AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
				SharedQueueCompanionAvailable               bool    `json:"shared_queue_companion_available"`
				CrossNodeCompletions                        int     `json:"cross_node_completions"`
				PassingCheckCount                           int     `json:"passing_check_count"`
				FailingCheckCount                           int     `json:"failing_check_count"`
			} `json:"summary"`
			ExecutorLanes []struct {
				Lane                   string `json:"lane"`
				LatestStatus           string `json:"latest_status"`
				LatestEnabled          bool   `json:"latest_enabled"`
				EnabledRuns            int    `json:"enabled_runs"`
				SucceededRuns          int    `json:"succeeded_runs"`
				ConsecutiveSuccesses   int    `json:"consecutive_successes"`
				AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
			} `json:"executor_lanes"`
			PolicyChecks []struct {
				Name   string `json:"name"`
				Passed bool   `json:"passed"`
			} `json:"policy_checks"`
			NextActions []string `json:"next_actions"`
		} `json:"validation_bundle_continuation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode continuation gate payload: %v", err)
	}
	if decoded.ValidationBundleContinuation.ReportPath != validationBundleContinuationGatePath ||
		decoded.ValidationBundleContinuation.ScorecardPath != validationBundleContinuationScorecardPath ||
		decoded.ValidationBundleContinuation.DigestPath != "docs/reports/validation-bundle-continuation-digest.md" ||
		decoded.ValidationBundleContinuation.Ticket != "OPE-262" ||
		decoded.ValidationBundleContinuation.Status != "policy-go" ||
		decoded.ValidationBundleContinuation.Recommendation != "go" {
		t.Fatalf("unexpected continuation gate metadata: %+v", decoded.ValidationBundleContinuation)
	}
	if len(decoded.ValidationBundleContinuation.ReviewerLinks) != 3 ||
		decoded.ValidationBundleContinuation.ReviewerLinks[0] != "docs/reports/live-validation-index.md" ||
		decoded.ValidationBundleContinuation.ReviewerLinks[1] != "docs/reports/validation-bundle-continuation-digest.md" ||
		decoded.ValidationBundleContinuation.ReviewerLinks[2] != validationBundleContinuationScorecardPath {
		t.Fatalf("unexpected continuation gate reviewer links: %+v", decoded.ValidationBundleContinuation.ReviewerLinks)
	}
	if decoded.ValidationBundleContinuation.Summary.LatestRunID != "20260316T140138Z" ||
		decoded.ValidationBundleContinuation.Summary.RecentBundleCount != 3 ||
		!decoded.ValidationBundleContinuation.Summary.AllExecutorTracksHaveRepeatedRecentCoverage ||
		!decoded.ValidationBundleContinuation.Summary.SharedQueueCompanionAvailable ||
		decoded.ValidationBundleContinuation.Summary.CrossNodeCompletions != 99 ||
		decoded.ValidationBundleContinuation.Summary.PassingCheckCount != 6 ||
		decoded.ValidationBundleContinuation.Summary.FailingCheckCount != 0 {
		t.Fatalf("unexpected continuation gate summary: %+v", decoded.ValidationBundleContinuation.Summary)
	}
	if len(decoded.ValidationBundleContinuation.PolicyChecks) != 6 ||
		decoded.ValidationBundleContinuation.PolicyChecks[0].Name != "latest_bundle_age_within_threshold" ||
		!decoded.ValidationBundleContinuation.PolicyChecks[0].Passed {
		t.Fatalf("unexpected continuation gate policy checks: %+v", decoded.ValidationBundleContinuation.PolicyChecks)
	}
	if len(decoded.ValidationBundleContinuation.ExecutorLanes) != 3 ||
		decoded.ValidationBundleContinuation.ExecutorLanes[0].Lane != "local" ||
		decoded.ValidationBundleContinuation.ExecutorLanes[0].LatestStatus != "succeeded" ||
		!decoded.ValidationBundleContinuation.ExecutorLanes[0].LatestEnabled ||
		decoded.ValidationBundleContinuation.ExecutorLanes[0].ConsecutiveSuccesses != 3 {
		t.Fatalf("unexpected continuation gate executor lanes: %+v", decoded.ValidationBundleContinuation.ExecutorLanes)
	}
	if len(decoded.ValidationBundleContinuation.NextActions) < 4 ||
		decoded.ValidationBundleContinuation.NextActions[0] != "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions" {
		t.Fatalf("unexpected continuation gate next actions: %+v", decoded.ValidationBundleContinuation.NextActions)
	}
}

func TestDebugStatusIncludesClawHostPolicySurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010000, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-debug-1",
			Source:   "clawhost",
			Title:    "review provider defaults",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":           "sales-app",
				"clawhost_default_provider": "openai",
				"clawhost_approval_flow":    "standard",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-debug-2",
			Labels:   []string{"clawhost"},
			Title:    "override provider for support",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":             "support-app",
				"clawhost_default_provider":   "anthropic",
				"clawhost_provider_mode":      "tenant_override",
				"clawhost_provider_allowlist": "openai,google",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHost struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies      int `json:"active_policies"`
				ReviewRequired      int `json:"review_required"`
				OutOfPolicyDefaults int `json:"out_of_policy_defaults"`
			} `json:"summary"`
			ObservedProviders []string `json:"observed_providers"`
			ReviewQueue       []struct {
				TaskID      string `json:"task_id"`
				DriftStatus string `json:"drift_status"`
			} `json:"review_queue"`
		} `json:"clawhost_policy_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug status response: %v", err)
	}
	if decoded.ClawHost.Status != "active" || decoded.ClawHost.Summary.ActivePolicies != 2 || decoded.ClawHost.Summary.ReviewRequired != 1 || decoded.ClawHost.Summary.OutOfPolicyDefaults != 1 {
		t.Fatalf("unexpected ClawHost debug surface: %+v", decoded.ClawHost)
	}
	if decoded.ClawHost.Filters["team"] != "" || decoded.ClawHost.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost policy filters, got %+v", decoded.ClawHost.Filters)
	}
	if len(decoded.ClawHost.ReviewQueue) != 2 || decoded.ClawHost.ReviewQueue[0].DriftStatus != "out_of_policy" {
		t.Fatalf("expected prioritized ClawHost review queue, got %+v", decoded.ClawHost.ReviewQueue)
	}
	for _, want := range []string{"anthropic", "openai"} {
		if !containsString(decoded.ClawHost.ObservedProviders, want) {
			t.Fatalf("expected provider %q in debug surface, got %+v", want, decoded.ClawHost.ObservedProviders)
		}
	}
}

func TestDebugStatusIncludesClawHostRolloutSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010200, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-rollout-1",
			Source:   "clawhost",
			Title:    "restart sales-west",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":                      "clawhost",
				"inventory_kind":                     "claw",
				"claw_id":                            "claw-sales-west",
				"claw_name":                          "sales-west",
				"provider":                           "hetzner",
				"provider_status":                    "running",
				"domain":                             "sales-west.clawhost.cloud",
				"agent_count":                        "2",
				"clawhost_rollout_action":            "restart",
				"clawhost_rollout_concurrency_limit": "2",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-rollout-2",
			Source:   "clawhost",
			Title:    "restart support-east",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"inventory_kind":             "claw",
				"claw_id":                    "claw-support-east",
				"claw_name":                  "support-east",
				"provider":                   "hetzner",
				"provider_status":            "running",
				"domain":                     "support-east.clawhost.cloud",
				"agent_count":                "1",
				"clawhost_rollout_action":    "restart",
				"clawhost_takeover_required": "true",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostRollout struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePlans      int `json:"active_plans"`
				TotalTargets     int `json:"total_targets"`
				CanaryTargets    int `json:"canary_targets"`
				TakeoverRequired int `json:"takeover_required"`
			} `json:"summary"`
			Plans []struct {
				Action       string `json:"action"`
				Concurrency  int    `json:"concurrency_limit"`
				TakeoverHook string `json:"takeover_hook"`
				WaveCount    int    `json:"wave_count"`
				CanaryCount  int    `json:"canary_count"`
			} `json:"plans"`
		} `json:"clawhost_rollout_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug status rollout response: %v", err)
	}
	if decoded.ClawHostRollout.Status != "active" || decoded.ClawHostRollout.Summary.ActivePlans != 1 || decoded.ClawHostRollout.Summary.TotalTargets != 2 || decoded.ClawHostRollout.Summary.CanaryTargets != 1 || decoded.ClawHostRollout.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected ClawHost rollout surface: %+v", decoded.ClawHostRollout)
	}
	if decoded.ClawHostRollout.Filters["team"] != "" || decoded.ClawHostRollout.Filters["project"] != "" {
		t.Fatalf("expected unscoped rollout surface filters, got %+v", decoded.ClawHostRollout.Filters)
	}
	if len(decoded.ClawHostRollout.Plans) != 1 || decoded.ClawHostRollout.Plans[0].Action != "restart" || decoded.ClawHostRollout.Plans[0].Concurrency != 2 || decoded.ClawHostRollout.Plans[0].TakeoverHook != "required" || decoded.ClawHostRollout.Plans[0].WaveCount != 2 || decoded.ClawHostRollout.Plans[0].CanaryCount != 1 {
		t.Fatalf("unexpected ClawHost rollout plan payload: %+v", decoded.ClawHostRollout.Plans)
	}
}

func TestDebugStatusIncludesClawHostWorkflowSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010300, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-workflow-1",
			Source:   "clawhost",
			Title:    "review channels",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"claw_id":                   "claw-a",
				"claw_name":                 "sales-west",
				"skill_count":               "3",
				"agent_skill_count":         "4",
				"channel_types":             "telegram,discord,whatsapp",
				"whatsapp_pairing_status":   "waiting",
				"admin_credentials_exposed": "true",
				"admin_surface_path":        "/credentials",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-workflow-2",
			Source:   "clawhost",
			Title:    "review skills",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_id":                 "claw-b",
				"claw_name":               "support-east",
				"skill_count":             "2",
				"agent_skill_count":       "2",
				"channel_types":           "telegram",
				"whatsapp_pairing_status": "paired",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Workflow struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				WorkflowItems     int `json:"workflow_items"`
				PairingApprovals  int `json:"pairing_approvals"`
				CredentialReviews int `json:"credential_reviews"`
				TakeoverRequired  int `json:"takeover_required"`
			} `json:"summary"`
			ReviewQueue []struct {
				ClawName           string   `json:"claw_name"`
				WhatsAppPairing    string   `json:"whatsapp_pairing"`
				CredentialsExposed bool     `json:"credentials_exposed"`
				TakeoverRequired   bool     `json:"takeover_required"`
				Channels           []string `json:"channels"`
			} `json:"review_queue"`
		} `json:"clawhost_workflow_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug workflow payload: %v", err)
	}
	if decoded.Workflow.Status != "active" || decoded.Workflow.Summary.WorkflowItems != 2 || decoded.Workflow.Summary.PairingApprovals != 1 || decoded.Workflow.Summary.CredentialReviews != 1 || decoded.Workflow.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected workflow surface: %+v", decoded.Workflow)
	}
	if decoded.Workflow.Filters["team"] != "" || decoded.Workflow.Filters["project"] != "" || decoded.Workflow.Filters["actor"] != "workflow-operator" {
		t.Fatalf("expected unscoped workflow filters, got %+v", decoded.Workflow.Filters)
	}
	if len(decoded.Workflow.ReviewQueue) != 2 || !decoded.Workflow.ReviewQueue[0].TakeoverRequired {
		t.Fatalf("expected takeover-required item first, got %+v", decoded.Workflow.ReviewQueue)
	}
}

func TestDebugStatusIncludesClawHostReadinessSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010350, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-ready-1",
			Source:   "clawhost",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":       "clawhost",
				"claw_id":             "claw-a",
				"claw_name":           "sales-west",
				"domain":              "sales-west.clawhost.cloud",
				"proxy_mode":          "http_ws_gateway",
				"gateway_port":        "18789",
				"reachable":           "true",
				"admin_ui_enabled":    "true",
				"websocket_reachable": "true",
				"subdomain_ready":     "true",
				"version_status":      "current",
				"version_current":     "0.0.31",
				"version_latest":      "0.0.31",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-ready-2",
			Source:   "clawhost",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":       "clawhost",
				"claw_id":             "claw-b",
				"claw_name":           "support-east",
				"domain":              "support-east.clawhost.cloud",
				"proxy_mode":          "http_ws_gateway",
				"gateway_port":        "18789",
				"reachable":           "false",
				"admin_ui_enabled":    "true",
				"websocket_reachable": "false",
				"subdomain_ready":     "false",
				"version_status":      "upgrade_available",
				"version_current":     "0.0.30",
				"version_latest":      "0.0.31",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Readiness struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets                 int `json:"targets"`
				ReadyTargets            int `json:"ready_targets"`
				DegradedTargets         int `json:"degraded_targets"`
				AdminReadyTargets       int `json:"admin_ready_targets"`
				WebSocketReadyTargets   int `json:"websocket_ready_targets"`
				SubdomainReadyTargets   int `json:"subdomain_ready_targets"`
				UpgradeAvailableTargets int `json:"upgrade_available_targets"`
			} `json:"summary"`
			Targets []struct {
				ClawName     string   `json:"claw_name"`
				ReviewStatus string   `json:"review_status"`
				Warnings     []string `json:"warnings"`
			} `json:"targets"`
		} `json:"clawhost_readiness_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug readiness payload: %v", err)
	}
	if decoded.Readiness.Status != "active" || decoded.Readiness.Summary.Targets != 2 || decoded.Readiness.Summary.ReadyTargets != 1 || decoded.Readiness.Summary.DegradedTargets != 1 || decoded.Readiness.Summary.AdminReadyTargets != 1 || decoded.Readiness.Summary.WebSocketReadyTargets != 1 || decoded.Readiness.Summary.SubdomainReadyTargets != 1 || decoded.Readiness.Summary.UpgradeAvailableTargets != 1 {
		t.Fatalf("unexpected readiness summary: %+v", decoded.Readiness)
	}
	if decoded.Readiness.Filters["team"] != "" || decoded.Readiness.Filters["project"] != "" {
		t.Fatalf("expected unscoped readiness filters, got %+v", decoded.Readiness.Filters)
	}
	if len(decoded.Readiness.Targets) != 2 || decoded.Readiness.Targets[0].ReviewStatus != "degraded" {
		t.Fatalf("expected degraded target sorted first, got %+v", decoded.Readiness.Targets)
	}
}

func TestDebugStatusIncludesClawHostRecoverySurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010360, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-recovery-1",
			Source:   "clawhost",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_id":                    "claw-a",
				"claw_name":                  "sales-west",
				"clawhost_lifecycle_actions": "start,restart,upgrade",
				"clawhost_pod_isolation":     "true",
				"clawhost_service_isolation": "true",
				"clawhost_takeover_required": "true",
				"clawhost_takeover_triggers": "proxy health regresses,session restore fails",
				"clawhost_recovery_evidence": "GET /status,/v2/reports/distributed",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-recovery-2",
			Source:   "clawhost",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_id":                    "claw-b",
				"claw_name":                  "support-east",
				"clawhost_lifecycle_actions": "restart",
				"clawhost_pod_isolation":     "false",
				"clawhost_service_isolation": "true",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Recovery struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets            int `json:"targets"`
				RecoverableTargets int `json:"recoverable_targets"`
				DegradedTargets    int `json:"degraded_targets"`
				IsolatedTargets    int `json:"isolated_targets"`
				TakeoverRequired   int `json:"takeover_required"`
				EvidenceArtifacts  int `json:"evidence_artifacts"`
			} `json:"summary"`
			Targets []struct {
				ClawName       string   `json:"claw_name"`
				RecoveryStatus string   `json:"recovery_status"`
				Warnings       []string `json:"warnings"`
			} `json:"targets"`
		} `json:"clawhost_recovery_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode debug recovery payload: %v", err)
	}
	if decoded.Recovery.Status != "active" || decoded.Recovery.Summary.Targets != 2 || decoded.Recovery.Summary.RecoverableTargets != 1 || decoded.Recovery.Summary.DegradedTargets != 1 || decoded.Recovery.Summary.IsolatedTargets != 1 || decoded.Recovery.Summary.TakeoverRequired != 1 || decoded.Recovery.Summary.EvidenceArtifacts != 2 {
		t.Fatalf("unexpected recovery summary: %+v", decoded.Recovery)
	}
	if decoded.Recovery.Filters["team"] != "" || decoded.Recovery.Filters["project"] != "" {
		t.Fatalf("expected unscoped recovery filters, got %+v", decoded.Recovery.Filters)
	}
	if len(decoded.Recovery.Targets) != 2 || decoded.Recovery.Targets[0].RecoveryStatus != "degraded" {
		t.Fatalf("expected degraded recovery target sorted first, got %+v", decoded.Recovery.Targets)
	}
}

func TestDebugStatusReusesSingleClawHostTaskSnapshot(t *testing.T) {
	now := time.Unix(1700010365, 0)
	taskQueue := &countingInspectorQueue{MemoryQueue: queue.NewMemoryQueue()}
	task := domain.Task{
		ID:       "clawhost-debug-shared-1",
		Source:   "clawhost",
		TenantID: "tenant-a",
		State:    domain.TaskQueued,
		Metadata: map[string]string{
			"control_plane":               "clawhost",
			"inventory_kind":              "claw",
			"claw_id":                     "claw-a",
			"claw_name":                   "sales-west",
			"provider":                    "openai",
			"provider_status":             "running",
			"clawhost_app_id":             "sales-app",
			"clawhost_default_provider":   "openai",
			"clawhost_provider_mode":      "app_default",
			"clawhost_provider_allowlist": "anthropic,openai",
			"skill_count":                 "3",
			"agent_skill_count":           "4",
			"channel_types":               "discord,telegram",
			"whatsapp_pairing_status":     "waiting",
			"domain":                      "sales-west.clawhost.cloud",
			"proxy_mode":                  "http_ws_gateway",
			"gateway_port":                "18789",
			"reachable":                   "true",
			"admin_ui_enabled":            "true",
			"websocket_reachable":         "true",
			"subdomain_ready":             "true",
			"version_status":              "current",
			"version_current":             "0.0.31",
			"version_latest":              "0.0.31",
			"clawhost_update_available":   "true",
			"clawhost_takeover_required":  "true",
			"clawhost_lifecycle_actions":  "start,restart,upgrade",
			"clawhost_pod_isolation":      "true",
			"clawhost_service_isolation":  "true",
			"clawhost_takeover_triggers":  "proxy health regresses,session restore fails",
			"clawhost_recovery_evidence":  "GET /status,/v2/reports/distributed",
		},
		UpdatedAt: now,
	}
	if err := taskQueue.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	if taskQueue.listCalls != 1 {
		t.Fatalf("expected a single ClawHost task snapshot, got %d list calls", taskQueue.listCalls)
	}
}

func TestDebugStatusScopesClawHostSurfacesByFilters(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010367, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-debug-filtered-1",
			Source:   "clawhost",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":               "clawhost",
				"team":                        "platform",
				"project":                     "sales",
				"inventory_kind":              "claw",
				"claw_id":                     "claw-a",
				"claw_name":                   "sales-west",
				"provider":                    "openai",
				"provider_status":             "running",
				"clawhost_app_id":             "sales-app",
				"clawhost_default_provider":   "openai",
				"clawhost_provider_mode":      "app_default",
				"clawhost_provider_allowlist": "anthropic,openai",
				"skill_count":                 "3",
				"agent_skill_count":           "4",
				"channel_types":               "discord,telegram",
				"whatsapp_pairing_status":     "waiting",
				"domain":                      "sales-west.clawhost.cloud",
				"proxy_mode":                  "http_ws_gateway",
				"gateway_port":                "18789",
				"reachable":                   "true",
				"admin_ui_enabled":            "true",
				"websocket_reachable":         "true",
				"subdomain_ready":             "true",
				"version_status":              "current",
				"version_current":             "0.0.31",
				"version_latest":              "0.0.31",
				"clawhost_update_available":   "true",
				"clawhost_takeover_required":  "true",
				"clawhost_lifecycle_actions":  "start,restart,upgrade",
				"clawhost_pod_isolation":      "true",
				"clawhost_service_isolation":  "true",
				"clawhost_takeover_triggers":  "proxy health regresses,session restore fails",
				"clawhost_recovery_evidence":  "GET /status,/v2/reports/distributed",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-debug-filtered-2",
			Source:   "clawhost",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":               "clawhost",
				"team":                        "support",
				"project":                     "care",
				"inventory_kind":              "claw",
				"claw_id":                     "claw-b",
				"claw_name":                   "support-east",
				"provider":                    "anthropic",
				"provider_status":             "running",
				"clawhost_app_id":             "support-app",
				"clawhost_default_provider":   "anthropic",
				"clawhost_provider_mode":      "tenant_override",
				"clawhost_provider_allowlist": "openai,google",
				"skill_count":                 "2",
				"agent_skill_count":           "2",
				"channel_types":               "telegram",
				"whatsapp_pairing_status":     "paired",
				"domain":                      "support-east.clawhost.cloud",
				"proxy_mode":                  "http_ws_gateway",
				"gateway_port":                "18790",
				"reachable":                   "true",
				"admin_ui_enabled":            "true",
				"websocket_reachable":         "true",
				"subdomain_ready":             "true",
				"version_status":              "current",
				"version_current":             "0.0.31",
				"version_latest":              "0.0.31",
				"clawhost_update_available":   "false",
				"clawhost_takeover_required":  "false",
				"clawhost_lifecycle_actions":  "restart",
				"clawhost_pod_isolation":      "true",
				"clawhost_service_isolation":  "true",
				"clawhost_takeover_triggers":  "session restore fails",
				"clawhost_recovery_evidence":  "GET /status",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status?team=platform&project=sales", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Filters struct {
			Team    string `json:"team"`
			Project string `json:"project"`
		} `json:"filters"`
		Policy struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies int `json:"active_policies"`
			} `json:"summary"`
			ObservedProviders []string `json:"observed_providers"`
			ReviewQueue       []struct {
				TaskID string `json:"task_id"`
			} `json:"review_queue"`
		} `json:"clawhost_policy_surface"`
		Workflow struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				WorkflowItems int `json:"workflow_items"`
			} `json:"summary"`
		} `json:"clawhost_workflow_surface"`
		Rollout struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePlans int `json:"active_plans"`
			} `json:"summary"`
		} `json:"clawhost_rollout_surface"`
		Readiness struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
		} `json:"clawhost_readiness_surface"`
		Recovery struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
		} `json:"clawhost_recovery_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode filtered debug payload: %v", err)
	}
	if decoded.Filters.Team != "platform" || decoded.Filters.Project != "sales" {
		t.Fatalf("expected debug filters to echo request scope, got %+v", decoded.Filters)
	}
	if decoded.Policy.Summary.ActivePolicies != 1 || len(decoded.Policy.ReviewQueue) != 1 || decoded.Policy.ReviewQueue[0].TaskID != "clawhost-debug-filtered-1" {
		t.Fatalf("expected scoped debug policy surface, got %+v", decoded.Policy)
	}
	if decoded.Policy.Filters["team"] != "platform" || decoded.Policy.Filters["project"] != "sales" {
		t.Fatalf("expected scoped debug policy filters, got %+v", decoded.Policy.Filters)
	}
	if containsString(decoded.Policy.ObservedProviders, "anthropic") || !containsString(decoded.Policy.ObservedProviders, "openai") {
		t.Fatalf("expected scoped debug policy providers, got %+v", decoded.Policy.ObservedProviders)
	}
	if decoded.Workflow.Filters["team"] != "platform" || decoded.Workflow.Filters["project"] != "sales" || decoded.Workflow.Filters["actor"] != "workflow-operator" {
		t.Fatalf("expected scoped debug workflow filters, got %+v", decoded.Workflow.Filters)
	}
	if decoded.Rollout.Filters["team"] != "platform" || decoded.Rollout.Filters["project"] != "sales" {
		t.Fatalf("expected scoped debug rollout filters, got %+v", decoded.Rollout.Filters)
	}
	if decoded.Readiness.Filters["team"] != "platform" || decoded.Readiness.Filters["project"] != "sales" {
		t.Fatalf("expected scoped debug readiness filters, got %+v", decoded.Readiness.Filters)
	}
	if decoded.Recovery.Filters["team"] != "platform" || decoded.Recovery.Filters["project"] != "sales" {
		t.Fatalf("expected scoped debug recovery filters, got %+v", decoded.Recovery.Filters)
	}
	if decoded.Workflow.Summary.WorkflowItems != 1 || decoded.Rollout.Summary.ActivePlans != 1 || decoded.Readiness.Summary.Targets != 1 || decoded.Recovery.Summary.Targets != 1 {
		t.Fatalf("expected scoped debug ClawHost surfaces, got workflow=%+v rollout=%+v readiness=%+v recovery=%+v", decoded.Workflow, decoded.Rollout, decoded.Readiness, decoded.Recovery)
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
		var envelope domain.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &envelope); err != nil {
			t.Fatalf("decode replayed stream event: %v", err)
		}
		if envelope.Delivery == nil || envelope.Delivery.Mode != domain.EventDeliveryModeReplay || !envelope.Delivery.Replay || envelope.Delivery.IdempotencyKey != "evt-old-1" {
			t.Fatalf("expected replay delivery contract in stream event, got %+v", envelope.Delivery)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for replayed event")
	}
}

func TestSubscriberGroupLeaseEndpointsFenceConflictsAndRollback(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	now := time.Unix(1700000000, 0)
	server := &Server{
		Recorder:         recorder,
		Queue:            queue.NewMemoryQueue(),
		Bus:              bus,
		SubscriberLeases: events.NewSubscriberLeaseCoordinator(),
		Now: func() time.Time {
			current := now
			now = now.Add(time.Second)
			return current
		},
	}
	handler := server.Handler()

	acquireBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-a","ttl_seconds":30}`))
	acquireRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/leases", acquireBody)
	acquireResponse := httptest.NewRecorder()
	handler.ServeHTTP(acquireResponse, acquireRequest)
	if acquireResponse.Code != http.StatusOK {
		t.Fatalf("expected acquire 200, got %d with %s", acquireResponse.Code, acquireResponse.Body.String())
	}

	var acquired struct {
		Lease events.SubscriberLease `json:"lease"`
	}
	if err := json.Unmarshal(acquireResponse.Body.Bytes(), &acquired); err != nil {
		t.Fatalf("decode acquire response: %v", err)
	}

	conflictBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-b","ttl_seconds":30}`))
	conflictRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/leases", conflictBody)
	conflictResponse := httptest.NewRecorder()
	handler.ServeHTTP(conflictResponse, conflictRequest)
	if conflictResponse.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d with %s", conflictResponse.Code, conflictResponse.Body.String())
	}
	if !strings.Contains(conflictResponse.Body.String(), "consumer-a") {
		t.Fatalf("expected current owner in conflict payload, got %s", conflictResponse.Body.String())
	}

	epoch := strconv.FormatInt(acquired.Lease.LeaseEpoch, 10)

	commitBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-a","lease_token":"` + acquired.Lease.LeaseToken + `","lease_epoch":` + epoch + `,"checkpoint_offset":7,"checkpoint_event_id":"evt-7"}`))
	commitRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/checkpoints", commitBody)
	commitResponse := httptest.NewRecorder()
	handler.ServeHTTP(commitResponse, commitRequest)
	if commitResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint commit 200, got %d with %s", commitResponse.Code, commitResponse.Body.String())
	}

	rollbackBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-a","lease_token":"` + acquired.Lease.LeaseToken + `","lease_epoch":` + epoch + `,"checkpoint_offset":6,"checkpoint_event_id":"evt-6"}`))
	rollbackRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/checkpoints", rollbackBody)
	rollbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(rollbackResponse, rollbackRequest)
	if rollbackResponse.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected rollback fence 412, got %d with %s", rollbackResponse.Code, rollbackResponse.Body.String())
	}

	statusRequest := httptest.NewRequest(http.MethodGet, "/subscriber-groups/group-a/subscribers/sub-a", nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("expected lease status 200, got %d with %s", statusResponse.Code, statusResponse.Body.String())
	}
	if !strings.Contains(statusResponse.Body.String(), "\"checkpoint_offset\":7") || !strings.Contains(statusResponse.Body.String(), "\"sequence_bridge\"") {
		t.Fatalf("expected sequence bridge in status payload, got %s", statusResponse.Body.String())
	}
	now = now.Add(30 * time.Second)

	takeoverBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-b","ttl_seconds":30}`))
	takeoverRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/leases", takeoverBody)
	takeoverResponse := httptest.NewRecorder()
	handler.ServeHTTP(takeoverResponse, takeoverRequest)
	if takeoverResponse.Code != http.StatusOK {
		t.Fatalf("expected takeover acquire 200, got %d with %s", takeoverResponse.Code, takeoverResponse.Body.String())
	}

	var takeover struct {
		Lease events.SubscriberLease `json:"lease"`
	}
	if err := json.Unmarshal(takeoverResponse.Body.Bytes(), &takeover); err != nil {
		t.Fatalf("decode takeover response: %v", err)
	}

	staleBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-a","lease_token":"` + acquired.Lease.LeaseToken + `","lease_epoch":` + epoch + `,"checkpoint_offset":8,"checkpoint_event_id":"evt-8"}`))
	staleRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/checkpoints", staleBody)
	staleResponse := httptest.NewRecorder()
	handler.ServeHTTP(staleResponse, staleRequest)
	if staleResponse.Code != http.StatusConflict {
		t.Fatalf("expected stale writer conflict 409, got %d with %s", staleResponse.Code, staleResponse.Body.String())
	}

	takeoverEpoch := strconv.FormatInt(takeover.Lease.LeaseEpoch, 10)
	takeoverCommitBody := bytes.NewReader([]byte(`{"group_id":"group-a","subscriber_id":"sub-a","consumer_id":"consumer-b","lease_token":"` + takeover.Lease.LeaseToken + `","lease_epoch":` + takeoverEpoch + `,"checkpoint_offset":9,"checkpoint_event_id":"evt-9"}`))
	takeoverCommitRequest := httptest.NewRequest(http.MethodPost, "/subscriber-groups/checkpoints", takeoverCommitBody)
	takeoverCommitResponse := httptest.NewRecorder()
	handler.ServeHTTP(takeoverCommitResponse, takeoverCommitRequest)
	if takeoverCommitResponse.Code != http.StatusOK {
		t.Fatalf("expected takeover checkpoint commit 200, got %d with %s", takeoverCommitResponse.Code, takeoverCommitResponse.Body.String())
	}

	auditEvents := recorder.EventsByTask("", 0)
	wantTypes := []domain.EventType{
		domain.EventSubscriberLeaseAcquired,
		domain.EventSubscriberLeaseRejected,
		domain.EventSubscriberCheckpointCommitted,
		domain.EventSubscriberCheckpointRejected,
		domain.EventSubscriberLeaseExpired,
		domain.EventSubscriberTakeoverSucceeded,
		domain.EventSubscriberCheckpointRejected,
		domain.EventSubscriberCheckpointCommitted,
	}
	filtered := make([]domain.Event, 0, len(wantTypes))
	for _, event := range auditEvents {
		switch event.Type {
		case domain.EventSubscriberLeaseAcquired,
			domain.EventSubscriberLeaseRejected,
			domain.EventSubscriberCheckpointCommitted,
			domain.EventSubscriberCheckpointRejected,
			domain.EventSubscriberLeaseExpired,
			domain.EventSubscriberTakeoverSucceeded:
			filtered = append(filtered, event)
		}
	}
	if len(filtered) != len(wantTypes) {
		t.Fatalf("expected %d subscriber audit events, got %d: %+v", len(wantTypes), len(filtered), filtered)
	}
	for i, want := range wantTypes {
		if filtered[i].Type != want {
			t.Fatalf("expected subscriber event %d to be %s, got %+v", i, want, filtered)
		}
	}
	takeoverEvent := filtered[5]
	for _, key := range []string{"group_id", "subscriber_id", "consumer_id", "previous_consumer_id", "lease_token", "lease_epoch", "checkpoint_offset", "sequence_bridge"} {
		if _, ok := takeoverEvent.Payload[key]; !ok {
			t.Fatalf("expected takeover payload field %q in %+v", key, takeoverEvent.Payload)
		}
	}
	if takeoverEvent.Payload["consumer_id"] != "consumer-b" || takeoverEvent.Payload["previous_consumer_id"] != "consumer-a" {
		t.Fatalf("unexpected takeover payload: %+v", takeoverEvent.Payload)
	}
	rejected := filtered[6]
	if rejected.Payload["reason"] != events.ErrLeaseFence.Error() {
		t.Fatalf("expected fenced checkpoint rejection payload, got %+v", rejected.Payload)
	}
	bridge, ok := rejected.Payload["sequence_bridge"].(map[string]any)
	if !ok || bridge["mapping_status"] != "lease_checkpoint_offset_mirrors_portable_sequence" || bridge["ownership_epoch"] == nil {
		t.Fatalf("expected sequence bridge metadata in rejection payload, got %+v", rejected.Payload["sequence_bridge"])
	}
}

func TestSubscriberGroupLeaseReleaseFencesStaleToken(t *testing.T) {
	coordinator := events.NewSubscriberLeaseCoordinator()
	now := time.Unix(1700000000, 0)
	lease, err := coordinator.Acquire(events.LeaseRequest{
		GroupID:      "group-a",
		SubscriberID: "sub-a",
		ConsumerID:   "consumer-a",
		TTL:          5 * time.Second,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("acquire initial lease: %v", err)
	}
	takeover, err := coordinator.Acquire(events.LeaseRequest{
		GroupID:      "group-a",
		SubscriberID: "sub-a",
		ConsumerID:   "consumer-b",
		TTL:          5 * time.Second,
		Now:          now.Add(6 * time.Second),
	})
	if err != nil {
		t.Fatalf("acquire takeover lease: %v", err)
	}
	if err := coordinator.Release("group-a", "sub-a", "consumer-a", lease.LeaseToken, lease.LeaseEpoch); !errors.Is(err, events.ErrLeaseFence) {
		t.Fatalf("expected stale release to be fenced, got %v", err)
	}
	if err := coordinator.Release("group-a", "sub-a", "consumer-b", takeover.LeaseToken, takeover.LeaseEpoch); err != nil {
		t.Fatalf("release active lease: %v", err)
	}
}

func TestEventsEndpointReturnsCursorFallbackMetadataWhenReplayWindowExpired(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBusWithHistoryLimit(2)
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	bus.Publish(domain.Event{ID: "evt-old-1", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-old-2", Type: domain.EventTaskStarted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-old-3", Type: domain.EventTaskCompleted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Now: time.Now}

	request := httptest.NewRequest(http.MethodGet, "/events?task_id=task-a&after_id=evt-old-1&limit=10", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if got := response.Header().Get("X-Replay-Cursor-Status"); got != "expired" {
		t.Fatalf("expected expired cursor header, got %q", got)
	}
	if got := response.Header().Get("X-Replay-Fallback"); got != "resume_from_oldest" {
		t.Fatalf("expected fallback header, got %q", got)
	}
	if !strings.Contains(response.Body.String(), "\"status\":\"expired\"") || !strings.Contains(response.Body.String(), "\"evt-old-2\"") {
		t.Fatalf("expected cursor metadata and fallback events, got %s", response.Body.String())
	}
}

func TestStreamEventsUsesLastEventIDForExpiredCursorFallback(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBusWithHistoryLimit(2)
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	bus.Publish(domain.Event{ID: "evt-old-1", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-old-2", Type: domain.EventTaskStarted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-old-3", Type: domain.EventTaskCompleted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?limit=10&task_id=task-a", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Last-Event-ID", "evt-old-1")

	type streamResult struct {
		line   string
		header http.Header
		err    error
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- streamResult{err: err}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line, err := reader.ReadString('\n')
		resultCh <- streamResult{line: strings.TrimSpace(line), header: response.Header.Clone(), err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("stream read: %v", result.err)
		}
		if got := result.header.Get("X-Replay-Cursor-Status"); got != "expired" {
			t.Fatalf("expected expired cursor header, got %q", got)
		}
		if got := result.header.Get("X-Replay-Fallback"); got != "resume_from_oldest" {
			t.Fatalf("expected fallback header, got %q", got)
		}
		if !strings.Contains(result.line, "evt-old-2") {
			t.Fatalf("expected fallback replay to start at oldest available event, got %q", result.line)
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
	for _, want := range []string{"worker_pool", "worker_pool_health", "workers_missing_heartbeat", "total_workers", "3", "active_workers", "2", "idle_workers", "1", "worker-b", "leased", "preemption_active", "last_preempted_task_id", "task-low"} {
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
	if !strings.Contains(response.Body.String(), "worker-a") || !strings.Contains(response.Body.String(), "successful_runs") || !strings.Contains(response.Body.String(), "event_log") || !strings.Contains(response.Body.String(), "in_memory_history") || !strings.Contains(response.Body.String(), "checkpoint") || !strings.Contains(response.Body.String(), "dedup") {
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

func TestMetricsJSONIncludesDurabilityRolloutScorecard(t *testing.T) {
	recorder := observability.NewRecorder()
	server := &Server{
		Recorder:  recorder,
		Queue:     queue.NewMemoryQueue(),
		Bus:       events.NewBus(),
		EventPlan: events.NewDurabilityPlan("memory", "broker_replicated", 3),
		Now:       time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "\"event_durability_rollout\"") {
		t.Fatalf("expected durability rollout scorecard in metrics payload, got %s", body)
	}
	if !strings.Contains(body, "\"rollout_ready\":false") {
		t.Fatalf("expected rollout readiness flag in metrics payload, got %s", body)
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
		"bigclaw_worker_lease_renewal_failures_total{worker_id=\"worker-b\"} 1",
		"bigclaw_worker_lease_lost_runs_total{worker_id=\"worker-b\"} 1",
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

func TestV2RunsListsScopedRunIndex(t *testing.T) {
	recorder := observability.NewRecorder()
	controller := control.New()
	base := time.Unix(1700000900, 0)
	premium := domain.Task{ID: "task-runs-1", TraceID: "trace-runs-1", Title: "Premium blocked run", State: domain.TaskBlocked, BudgetCents: 1200, Metadata: map[string]string{"team": "platform", "project": "alpha", "plan": "premium"}, CreatedAt: base, UpdatedAt: base.Add(2 * time.Minute)}
	dead := domain.Task{ID: "task-runs-2", TraceID: "trace-runs-2", Title: "Dead letter run", State: domain.TaskDeadLetter, BudgetCents: 300, Metadata: map[string]string{"team": "platform", "project": "alpha"}, CreatedAt: base, UpdatedAt: base.Add(4 * time.Minute)}
	other := domain.Task{ID: "task-runs-3", TraceID: "trace-runs-3", Title: "Other team run", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "growth", "project": "beta"}, CreatedAt: base, UpdatedAt: base.Add(3 * time.Minute)}
	for _, task := range []domain.Task{premium, dead, other} {
		recorder.StoreTask(task)
	}
	recorder.Record(domain.Event{ID: "evt-runs-1", Type: domain.EventRunTakeover, TaskID: premium.ID, TraceID: premium.TraceID, Timestamp: base.Add(2 * time.Minute), Payload: map[string]any{"message": "awaiting review"}})
	recorder.Record(domain.Event{ID: "evt-runs-2", Type: domain.EventTaskDeadLetter, TaskID: dead.ID, TraceID: dead.TraceID, Timestamp: base.Add(4 * time.Minute), Payload: map[string]any{"message": "replay required"}})
	controller.Takeover(premium.ID, "alice", "bob", "manual review", base.Add(5*time.Minute))
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: controller, Now: func() time.Time { return base.Add(6 * time.Minute) }}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/runs?team=platform&project=alpha&limit=10", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected run index 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Summary struct {
			TotalRuns         int            `json:"total_runs"`
			ActiveRuns        int            `json:"active_runs"`
			BlockedRuns       int            `json:"blocked_runs"`
			PremiumRuns       int            `json:"premium_runs"`
			DeadLetters       int            `json:"dead_letters"`
			BudgetCentsTotal  int64          `json:"budget_cents_total"`
			StateDistribution map[string]int `json:"state_distribution"`
		} `json:"summary"`
		Runs []struct {
			Task struct {
				ID string `json:"id"`
			} `json:"task"`
			Policy struct {
				Plan string `json:"plan"`
			} `json:"policy"`
			Drilldown struct {
				Run string `json:"run"`
			} `json:"drilldown"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode run index: %v", err)
	}
	if decoded.Summary.TotalRuns != 2 || decoded.Summary.ActiveRuns != 1 || decoded.Summary.BlockedRuns != 1 || decoded.Summary.PremiumRuns != 1 || decoded.Summary.DeadLetters != 1 || decoded.Summary.BudgetCentsTotal != 1500 {
		t.Fatalf("unexpected run index summary: %+v", decoded.Summary)
	}
	if len(decoded.Runs) != 2 || decoded.Runs[0].Task.ID != "task-runs-2" || decoded.Runs[1].Task.ID != "task-runs-1" {
		t.Fatalf("unexpected run ordering: %+v", decoded.Runs)
	}
	if decoded.Runs[1].Policy.Plan != "premium" || decoded.Runs[1].Drilldown.Run != "/v2/runs/task-runs-1" {
		t.Fatalf("unexpected run payload: %+v", decoded.Runs[1])
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
	recorder.Record(domain.Event{
		ID:        "evt-dead",
		Type:      domain.EventTaskDeadLetter,
		TaskID:    task.ID,
		TraceID:   task.TraceID,
		RunID:     "run-report-1",
		Timestamp: base.Add(3 * time.Second),
		Payload: map[string]any{
			"executor":     domain.ExecutorKubernetes,
			"message":      "pod crashed during validation",
			"artifacts":    []string{"k8s://jobs/bigclaw/run-report", "k8s://pods/bigclaw/run-report-0"},
			"report_path":  "reports/task-run-report/run-report-1.md",
			"journal_path": "journals/platform/run-report-1.json",
		},
	})
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
		Closeout struct {
			ValidationEvidence []string `json:"validation_evidence"`
			GitPushSucceeded   bool     `json:"git_push_succeeded"`
			GitPushOutput      string   `json:"git_push_output"`
			GitLogStatOutput   string   `json:"git_log_stat_output"`
			RemoteSynced       bool     `json:"remote_synced"`
			LocalSHA           string   `json:"local_sha"`
			RemoteSHA          string   `json:"remote_sha"`
			Complete           bool     `json:"complete"`
		} `json:"closeout"`
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
	if len(decoded.ArtifactRefs) < 6 {
		t.Fatalf("expected artifact refs for executor, workpad, and linked records, got %+v", decoded.ArtifactRefs)
	}
	artifactKinds := make(map[string]string, len(decoded.ArtifactRefs))
	for _, ref := range decoded.ArtifactRefs {
		artifactKinds[ref.Kind] = ref.URI
	}
	if artifactKinds["workflow_report"] != "reports/task-run-report/run-report-1.md" || artifactKinds["workflow_journal"] != "journals/platform/run-report-1.json" {
		t.Fatalf("expected workflow artifact refs in run detail, got %+v", decoded.ArtifactRefs)
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
	if len(decoded.Closeout.ValidationEvidence) != 0 || decoded.Closeout.GitPushSucceeded || decoded.Closeout.GitLogStatOutput != "" || decoded.Closeout.RemoteSynced || decoded.Closeout.Complete {
		t.Fatalf("expected empty closeout summary when metadata is absent, got %+v", decoded.Closeout)
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
	if !strings.Contains(reportResponse.Body.String(), "## Closeout") || !strings.Contains(reportResponse.Body.String(), "Complete: false") {
		t.Fatalf("expected closeout section in run report, got %s", reportResponse.Body.String())
	}
}

func TestV2RunDetailCloseoutSummaryFromMetadata(t *testing.T) {
	recorder := observability.NewRecorder()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Control: control.New(), Now: func() time.Time { return time.Unix(1700006000, 0) }}
	handler := server.Handler()
	task := domain.Task{
		ID:      "task-run-closeout",
		TraceID: "trace-run-closeout",
		Title:   "Close out release",
		State:   domain.TaskSucceeded,
		Metadata: map[string]string{
			"team":                "platform",
			"project":             "alpha",
			"validation_evidence": `["go test ./internal/api","bash scripts/ops/bigclawctl github-sync status --json"]`,
			"git_push_succeeded":  "true",
			"git_push_output":     "To github.com:OpenAGIs/BigClaw.git",
			"git_log_stat_output": "commit abc123\n bigclaw-go/internal/api/v2.go | 10 ++++++++++",
			"remote_synced":       "true",
			"local_sha":           "abc123",
			"remote_sha":          "abc123",
		},
	}
	recorder.StoreTask(task)

	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-closeout?limit=20", nil))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected run detail 200, got %d %s", runResponse.Code, runResponse.Body.String())
	}
	var decoded struct {
		Closeout struct {
			ValidationEvidence []string `json:"validation_evidence"`
			GitPushSucceeded   bool     `json:"git_push_succeeded"`
			GitPushOutput      string   `json:"git_push_output"`
			GitLogStatOutput   string   `json:"git_log_stat_output"`
			RemoteSynced       bool     `json:"remote_synced"`
			LocalSHA           string   `json:"local_sha"`
			RemoteSHA          string   `json:"remote_sha"`
			Complete           bool     `json:"complete"`
		} `json:"closeout"`
	}
	if err := json.Unmarshal(runResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode run detail closeout: %v", err)
	}
	if len(decoded.Closeout.ValidationEvidence) != 2 || !decoded.Closeout.GitPushSucceeded || decoded.Closeout.GitPushOutput == "" || decoded.Closeout.GitLogStatOutput == "" || !decoded.Closeout.RemoteSynced || decoded.Closeout.LocalSHA != "abc123" || decoded.Closeout.RemoteSHA != "abc123" || !decoded.Closeout.Complete {
		t.Fatalf("unexpected closeout payload: %+v", decoded.Closeout)
	}
}

func TestV2RunReportSanitizesAttachmentFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "Ops / Alert @ Night",
		TraceID:   "trace-run-report-special",
		Title:     "Investigate overnight alert storm",
		State:     domain.TaskBlocked,
		CreatedAt: time.Date(2026, 3, 25, 1, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 1, 30, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/runs/Ops%20%2F%20Alert%20@%20Night/report?limit=20", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected sanitized run report 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="ops-alert-night-run-report.md"` {
		t.Fatalf("expected sanitized attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "Task ID: Ops / Alert @ Night") {
		t.Fatalf("expected markdown to preserve original task id, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSanitizesAttachmentFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "dist-export-1",
		TraceID:   "trace-dist-export-1",
		Title:     "Distributed export coverage",
		State:     domain.TaskSucceeded,
		Metadata:  map[string]string{"team": "Platform / Ops @ Night", "project": "apollo/mobile"},
		CreatedAt: time.Date(2026, 3, 25, 2, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 2, 30, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=Platform%20%2F%20Ops%20%40%20Night", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected distributed export markdown content type, got %q", contentType)
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-platform-ops-night.md"` {
		t.Fatalf("expected sanitized distributed attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2RunReportSanitizationFallsBackForPunctuationOnlyTaskID(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        " / @ ",
		TraceID:   "trace-run-report-fallback",
		Title:     "Fallback filename coverage",
		State:     domain.TaskBlocked,
		CreatedAt: time.Date(2026, 3, 25, 3, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 3, 15, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/runs/%20%2F%20%40%20/report?limit=20", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected fallback-sanitized run report 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="all-run-report.md"` {
		t.Fatalf("expected fallback attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "Task ID:  / @ ") {
		t.Fatalf("expected markdown to preserve original punctuation-only task id, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSanitizationFallsBackForPunctuationOnlyTeam(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "dist-export-fallback",
		TraceID:   "trace-dist-export-fallback",
		Title:     "Distributed fallback filename coverage",
		State:     domain.TaskSucceeded,
		Metadata:  map[string]string{"team": " / @ ", "project": "apollo/mobile"},
		CreatedAt: time.Date(2026, 3, 25, 4, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 4, 15, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=%20%2F%20%40%20", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected fallback-sanitized distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-all.md"` {
		t.Fatalf("expected fallback distributed attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSanitizesProjectScopedAttachmentFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "dist-export-project-scope",
		TraceID:   "trace-dist-export-project-scope",
		Title:     "Distributed project-scope filename coverage",
		State:     domain.TaskSucceeded,
		Metadata:  map[string]string{"project": "Apollo / Mobile @ Core"},
		CreatedAt: time.Date(2026, 3, 25, 4, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 4, 45, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?project=Apollo%20%2F%20Mobile%20%40%20Core", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected project-scoped distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-apollo-mobile-core.md"` {
		t.Fatalf("expected sanitized project-scoped attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSanitizesTaskScopedAttachmentFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "Task / Scope @ Edge",
		TraceID:   "trace-dist-export-task-scope",
		Title:     "Distributed task-scope filename coverage",
		State:     domain.TaskSucceeded,
		CreatedAt: time.Date(2026, 3, 25, 5, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 5, 15, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?task_id=Task%20%2F%20Scope%20%40%20Edge", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected task-scoped distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-task-scope-edge.md"` {
		t.Fatalf("expected sanitized task-scoped attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportFilenamePrefersTeamScopeWhenMultipleFiltersExist(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:      "Task / Scope @ Edge",
		TraceID: "trace-dist-export-scope-precedence",
		Title:   "Distributed scope precedence filename coverage",
		State:   domain.TaskSucceeded,
		Metadata: map[string]string{
			"team":    "Platform / Ops @ Night",
			"project": "Apollo / Mobile @ Core",
		},
		CreatedAt: time.Date(2026, 3, 25, 5, 20, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 5, 35, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=Platform%20%2F%20Ops%20%40%20Night&project=Apollo%20%2F%20Mobile%20%40%20Core&task_id=Task%20%2F%20Scope%20%40%20Edge", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected multi-scope distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-platform-ops-night.md"` {
		t.Fatalf("expected team-scoped distributed attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportProjectFallbacksToAllFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "dist-export-project-fallback",
		TraceID:   "trace-dist-export-project-fallback",
		Title:     "Distributed project fallback filename coverage",
		State:     domain.TaskSucceeded,
		Metadata:  map[string]string{"project": " / @ "},
		CreatedAt: time.Date(2026, 3, 25, 5, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 5, 45, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?project=%20%2F%20%40%20", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected project-fallback distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-all.md"` {
		t.Fatalf("expected project-fallback distributed attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportTaskFallbacksToAllFilename(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "dist-export-task-fallback",
		TraceID:   "trace-dist-export-task-fallback",
		Title:     "Distributed task fallback filename coverage",
		State:     domain.TaskSucceeded,
		CreatedAt: time.Date(2026, 3, 25, 6, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 6, 15, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?task_id=%20%2F%20%40%20", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected task-fallback distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-all.md"` {
		t.Fatalf("expected task-fallback distributed attachment filename, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSkipsPunctuationOnlyTeamForFilenameScope(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:      "dist-export-scope-fallback",
		TraceID: "trace-dist-export-scope-fallback",
		Title:   "Distributed filename scope fallback coverage",
		State:   domain.TaskSucceeded,
		Metadata: map[string]string{
			"team":    " / @ ",
			"project": "Apollo / Mobile @ Core",
		},
		CreatedAt: time.Date(2026, 3, 25, 6, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 6, 45, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=%20%2F%20%40%20&project=Apollo%20%2F%20Mobile%20%40%20Core", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected punctuation-team distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-apollo-mobile-core.md"` {
		t.Fatalf("expected project-scoped distributed attachment filename after punctuation-only team, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestV2DistributedExportSkipsPunctuationOnlyTeamAndUsesTaskIDForFilenameScope(t *testing.T) {
	recorder := observability.NewRecorder()
	task := domain.Task{
		ID:        "Task / Scope @ Edge",
		TraceID:   "trace-dist-export-task-fallback-after-team",
		Title:     "Distributed task filename fallback after punctuation-only team",
		State:     domain.TaskSucceeded,
		Metadata:  map[string]string{"team": " / @ "},
		CreatedAt: time.Date(2026, 3, 25, 7, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 3, 25, 7, 15, 0, 0, time.UTC),
	}
	recorder.StoreTask(task)
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed/export?team=%20%2F%20%40%20&task_id=Task%20%2F%20Scope%20%40%20Edge", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected punctuation-team/task-scope distributed export 200, got %d %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); disposition != `attachment; filename="bigclaw-distributed-diagnostics-task-scope-edge.md"` {
		t.Fatalf("expected task-scoped distributed attachment filename after punctuation-only team, got %q", disposition)
	}
	if !strings.Contains(response.Body.String(), "# BigClaw Distributed Diagnostics") {
		t.Fatalf("expected distributed diagnostics markdown body, got %s", response.Body.String())
	}
}

func TestSanitizeReportNameNormalizesMixedSeparatorInputs(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "Platform / Ops @ Night", want: "platform-ops-night"},
		{input: "  Apollo___Mobile---Core  ", want: "apollo-mobile-core"},
		{input: " / @ ", want: "all"},
		{input: "", want: "all"},
	} {
		if got := sanitizeReportName(tc.input); got != tc.want {
			t.Fatalf("sanitizeReportName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFirstMeaningfulReportNameSkipsEmptyAndPunctuationOnlyScopes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs []string
		want   string
	}{
		{
			name:   "project after punctuation-only team",
			inputs: []string{" / @ ", "Apollo / Mobile @ Core", "Task / Scope @ Edge"},
			want:   "Apollo / Mobile @ Core",
		},
		{
			name:   "task after whitespace team and project",
			inputs: []string{"   ", " / @ ", "Task / Scope @ Edge"},
			want:   "Task / Scope @ Edge",
		},
		{
			name:   "all when every scope collapses",
			inputs: []string{"   ", " / @ ", ""},
			want:   "all",
		},
	} {
		if got := firstMeaningfulReportName(tc.inputs...); got != tc.want {
			t.Fatalf("%s: firstMeaningfulReportName(%q) = %q, want %q", tc.name, tc.inputs, got, tc.want)
		}
	}
}

func TestV2RunDetailIncludesRepoTriagePacket(t *testing.T) {
	recorder := observability.NewRecorder()
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Control: control.New(), Now: func() time.Time { return time.Unix(1700006100, 0) }}
	handler := server.Handler()
	task := domain.Task{
		ID:      "task-run-repo-triage",
		TraceID: "trace-run-repo-triage",
		Title:   "Review repo lineage",
		State:   domain.TaskBlocked,
		Metadata: map[string]string{
			"team":                  "platform",
			"project":               "alpha",
			"repo_triage_status":    "needs-approval",
			"candidate_commit_hash": "abc123",
			"accepted_commit_hash":  "def456",
			"similar_failure_count": "1",
			"discussion_open":       "1",
			"lineage_summary":       "candidate abc123 descends from accepted commit def456 after reviewer fixes",
			"run_commit_links":      `[{"run_id":"task-run-repo-triage","commit_hash":"abc123","role":"candidate","repo_space_id":"space-1"},{"run_id":"task-run-repo-triage","commit_hash":"def456","role":"accepted","repo_space_id":"space-1"}]`,
		},
	}
	recorder.StoreTask(task)

	runResponse := httptest.NewRecorder()
	handler.ServeHTTP(runResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-repo-triage?limit=20", nil))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("expected run detail 200, got %d %s", runResponse.Code, runResponse.Body.String())
	}
	var decoded struct {
		RepoTriage struct {
			Status   string `json:"status"`
			Evidence struct {
				CandidateCommit     string `json:"candidate_commit"`
				AcceptedAncestor    string `json:"accepted_ancestor"`
				SimilarFailureCount int    `json:"similar_failure_count"`
				DiscussionOpen      int    `json:"discussion_open"`
			} `json:"evidence"`
			Recommendation struct {
				Action string `json:"action"`
				Reason string `json:"reason"`
			} `json:"recommendation"`
			ApprovalPacket struct {
				RunID               string `json:"run_id"`
				CandidateCommitHash string `json:"candidate_commit_hash"`
				AcceptedCommitHash  string `json:"accepted_commit_hash"`
				LineageSummary      string `json:"lineage_summary"`
			} `json:"approval_packet"`
		} `json:"repo_triage"`
	}
	if err := json.Unmarshal(runResponse.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode repo triage run detail: %v", err)
	}
	if decoded.RepoTriage.Status != "needs-approval" || decoded.RepoTriage.Recommendation.Action != "approve" || decoded.RepoTriage.Recommendation.Reason != "accepted ancestor exists" {
		t.Fatalf("unexpected repo triage recommendation: %+v", decoded.RepoTriage)
	}
	if decoded.RepoTriage.Evidence.CandidateCommit != "abc123" || decoded.RepoTriage.Evidence.AcceptedAncestor != "def456" || decoded.RepoTriage.Evidence.SimilarFailureCount != 1 || decoded.RepoTriage.Evidence.DiscussionOpen != 1 {
		t.Fatalf("unexpected repo triage evidence: %+v", decoded.RepoTriage.Evidence)
	}
	if decoded.RepoTriage.ApprovalPacket.RunID != "task-run-repo-triage" || decoded.RepoTriage.ApprovalPacket.CandidateCommitHash != "abc123" || decoded.RepoTriage.ApprovalPacket.AcceptedCommitHash != "def456" || decoded.RepoTriage.ApprovalPacket.LineageSummary == "" {
		t.Fatalf("unexpected repo triage approval packet: %+v", decoded.RepoTriage.ApprovalPacket)
	}

	reportResponse := httptest.NewRecorder()
	handler.ServeHTTP(reportResponse, httptest.NewRequest(http.MethodGet, "/v2/runs/task-run-repo-triage/report?limit=20", nil))
	if reportResponse.Code != http.StatusOK {
		t.Fatalf("expected run report 200, got %d %s", reportResponse.Code, reportResponse.Body.String())
	}
	for _, want := range []string{"## Repo Triage", "Recommendation: approve (accepted ancestor exists)", "Candidate Commit: abc123", "Accepted Ancestor: def456"} {
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
	for _, want := range []string{"queue_budget_cents_total", "600", "high_risk_runs", "premium_runs", "active_takeovers", "worker_pool", "idle_workers", "task-control-1", "effective_state", "blocked", "Manual validation required", "queue_by_project", "queue_by_team", "alpha", "platform", "recent_actions", "audit_summary", "notes_timeline", "event_log", "in_memory_history", "process_local", "\"checkpoint\":{\"supported\":false", "\"dedup\":{\"supported\":false", "\"filtering\":{\"supported\":true"} {
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
	for _, want := range []string{"worker_pool", "worker_pool_health", "workers_missing_heartbeat", "total_workers", "3", "active_workers", "2", "idle_workers", "1", "worker-c", "idle", "preemption_active", "task-low"} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("expected %q in control center payload, got %s", want, bodyText)
		}
	}
}

func TestV2ControlCenterAppliesTimeWindowAndReturnsNodeAwareWorkerPoolSummary(t *testing.T) {
	base := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	recorder := observability.NewRecorder()
	server := &Server{
		Recorder: recorder,
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Worker:   fakeNodeAwareWorkerPoolStatus{now: base},
		Control:  control.New(),
		Now:      func() time.Time { return base },
	}
	for _, task := range []domain.Task{
		{ID: "task-window-old", TraceID: "trace-window-old", Title: "Old task", State: domain.TaskSucceeded, Metadata: map[string]string{"team": "platform", "project": "alpha"}, UpdatedAt: base.Add(-2 * time.Hour)},
		{ID: "task-window-current", TraceID: "trace-window-current", Title: "Current task", State: domain.TaskRunning, Metadata: map[string]string{"team": "platform", "project": "alpha"}, UpdatedAt: base.Add(-15 * time.Minute)},
		{ID: "task-window-other-team", TraceID: "trace-window-other-team", Title: "Other team", State: domain.TaskRunning, Metadata: map[string]string{"team": "growth", "project": "alpha"}, UpdatedAt: base.Add(-10 * time.Minute)},
	} {
		recorder.StoreTask(task)
	}

	since := base.Add(-30 * time.Minute)
	requestURL := fmt.Sprintf(
		"/v2/control-center?team=platform&project=alpha&since=%s&until=%s&limit=10&audit_limit=10",
		url.QueryEscape(since.Format(time.RFC3339)),
		url.QueryEscape(base.Format(time.RFC3339)),
	)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestURL, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}

	var decoded struct {
		Filters struct {
			Since time.Time `json:"since"`
			Until time.Time `json:"until"`
		} `json:"filters"`
		RecentTasks []struct {
			Task struct {
				ID string `json:"id"`
			} `json:"task"`
		} `json:"recent_tasks"`
		WorkerPool struct {
			TotalNodes                 int     `json:"total_nodes"`
			ActiveNodes                int     `json:"active_nodes"`
			IdleNodes                  int     `json:"idle_nodes"`
			DegradedNodes              int     `json:"degraded_nodes"`
			CapacityUtilizationPercent float64 `json:"capacity_utilization_percent"`
			NodeHealthDistribution     []struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			} `json:"node_health_distribution"`
			Nodes []struct {
				NodeID                     string         `json:"node_id"`
				Health                     string         `json:"health"`
				ActiveWorkers              int            `json:"active_workers"`
				IdleWorkers                int            `json:"idle_workers"`
				StaleWorkers               int            `json:"stale_workers"`
				CapacityUtilizationPercent float64        `json:"capacity_utilization_percent"`
				WorkerStates               map[string]int `json:"worker_states"`
			} `json:"nodes"`
			ExecutorDistribution []struct {
				Key   string `json:"key"`
				Count int    `json:"count"`
			} `json:"executor_distribution"`
		} `json:"worker_pool"`
		DistributedDiagnostics struct {
			Summary struct {
				TotalTasks int `json:"total_tasks"`
			} `json:"summary"`
			RolloutReport struct {
				ExportURL string `json:"export_url"`
			} `json:"rollout_report"`
		} `json:"distributed_diagnostics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center payload: %v", err)
	}
	if !decoded.Filters.Since.Equal(since) || !decoded.Filters.Until.Equal(base) {
		t.Fatalf("expected control center filters to preserve time window, got %+v", decoded.Filters)
	}
	if len(decoded.RecentTasks) != 1 || decoded.RecentTasks[0].Task.ID != "task-window-current" {
		t.Fatalf("expected time-window filtering to keep only the current platform task, got %+v", decoded.RecentTasks)
	}
	if decoded.WorkerPool.TotalNodes != 3 || decoded.WorkerPool.ActiveNodes != 1 || decoded.WorkerPool.IdleNodes != 1 || decoded.WorkerPool.DegradedNodes != 1 {
		t.Fatalf("unexpected node-aware worker pool summary: %+v", decoded.WorkerPool)
	}
	if len(decoded.WorkerPool.ExecutorDistribution) != 3 {
		t.Fatalf("expected executor distribution across worker nodes, got %+v", decoded.WorkerPool.ExecutorDistribution)
	}
	if len(decoded.WorkerPool.NodeHealthDistribution) != 3 ||
		decoded.WorkerPool.NodeHealthDistribution[0].Key != "active" ||
		decoded.WorkerPool.NodeHealthDistribution[1].Key != "degraded" ||
		decoded.WorkerPool.NodeHealthDistribution[2].Key != "idle" {
		t.Fatalf("expected node health distribution across worker nodes, got %+v", decoded.WorkerPool.NodeHealthDistribution)
	}
	nodesByID := make(map[string]struct {
		Health                     string
		ActiveWorkers              int
		IdleWorkers                int
		StaleWorkers               int
		CapacityUtilizationPercent float64
		WorkerStates               map[string]int
	}, len(decoded.WorkerPool.Nodes))
	for _, node := range decoded.WorkerPool.Nodes {
		nodesByID[node.NodeID] = struct {
			Health                     string
			ActiveWorkers              int
			IdleWorkers                int
			StaleWorkers               int
			CapacityUtilizationPercent float64
			WorkerStates               map[string]int
		}{
			Health:                     node.Health,
			ActiveWorkers:              node.ActiveWorkers,
			IdleWorkers:                node.IdleWorkers,
			StaleWorkers:               node.StaleWorkers,
			CapacityUtilizationPercent: node.CapacityUtilizationPercent,
			WorkerStates:               node.WorkerStates,
		}
	}
	if node, ok := nodesByID["node-a"]; !ok || node.Health != "active" || node.ActiveWorkers != 1 || node.StaleWorkers != 0 || node.WorkerStates["running"] != 1 {
		t.Fatalf("unexpected node-a summary: %+v", node)
	}
	if node, ok := nodesByID["node-b"]; !ok || node.Health != "degraded" || node.ActiveWorkers != 1 || node.StaleWorkers != 1 || node.WorkerStates["leased"] != 1 {
		t.Fatalf("unexpected node-b summary: %+v", node)
	}
	if node, ok := nodesByID["node-c"]; !ok || node.Health != "idle" || node.IdleWorkers != 1 || node.CapacityUtilizationPercent != 0 || node.WorkerStates["idle"] != 1 {
		t.Fatalf("unexpected node-c summary: %+v", node)
	}
	if decoded.DistributedDiagnostics.Summary.TotalTasks != 1 {
		t.Fatalf("expected distributed diagnostics to honor the same time window, got %+v", decoded.DistributedDiagnostics.Summary)
	}
	if !strings.Contains(decoded.DistributedDiagnostics.RolloutReport.ExportURL, "since=2026-03-23T09%3A30%3A00Z") || !strings.Contains(decoded.DistributedDiagnostics.RolloutReport.ExportURL, "until=2026-03-23T10%3A00%3A00Z") {
		t.Fatalf("expected distributed export url to retain the time window, got %s", decoded.DistributedDiagnostics.RolloutReport.ExportURL)
	}
}

func TestBuildDistributedDiagnosticsIncludesWorkerPoolSummary(t *testing.T) {
	base := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)
	server := &Server{
		Recorder:  observability.NewRecorder(),
		Queue:     queue.NewMemoryQueue(),
		Executors: []domain.ExecutorKind{domain.ExecutorLocal, domain.ExecutorKubernetes, domain.ExecutorRay},
		Control:   control.New(),
		Worker:    fakeNodeAwareWorkerPoolStatus{now: base},
		Now:       func() time.Time { return base },
	}

	diagnostics := server.buildDistributedDiagnostics(controlCenterFilters{})
	if diagnostics.WorkerPool == nil || diagnostics.WorkerPoolHealth == nil {
		t.Fatalf("expected worker pool diagnostics, got %+v %+v", diagnostics.WorkerPool, diagnostics.WorkerPoolHealth)
	}
	if diagnostics.WorkerPool.TotalNodes != 3 || diagnostics.WorkerPool.DegradedNodes != 1 {
		t.Fatalf("unexpected worker pool summary: %+v", diagnostics.WorkerPool)
	}
	if len(diagnostics.WorkerPool.ExecutorDistribution) != 3 || len(diagnostics.WorkerPool.NodeHealthDistribution) != 3 {
		t.Fatalf("unexpected worker pool distributions: %+v %+v", diagnostics.WorkerPool.ExecutorDistribution, diagnostics.WorkerPool.NodeHealthDistribution)
	}
	if diagnostics.WorkerPoolHealth.StaleWorkers != 1 || diagnostics.WorkerPoolHealth.WorkersMissingHeartbeat != 0 {
		t.Fatalf("unexpected worker pool health: %+v", diagnostics.WorkerPoolHealth)
	}
}

func TestRenderDistributedDiagnosticsMarkdownIncludesWorkerPoolSummary(t *testing.T) {
	markdown := renderDistributedDiagnosticsMarkdown(distributedDiagnostics{
		Summary: distributedDiagnosticsSummary{
			RegisteredExecutors: 2,
			ActiveExecutors:     1,
			TotalTasks:          3,
			ActiveRuns:          2,
			ActiveWorkers:       2,
			IdleWorkers:         1,
		},
		WorkerPool: &workerPoolSummary{
			TotalWorkers:               3,
			ActiveWorkers:              2,
			IdleWorkers:                1,
			TotalNodes:                 2,
			CapacityUtilizationPercent: 66.7,
			ExecutorDistribution: []auditFacetCount{
				{Key: "local", Count: 2},
				{Key: "ray", Count: 1},
			},
			NodeHealthDistribution: []auditFacetCount{
				{Key: "active", Count: 1},
				{Key: "degraded", Count: 1},
			},
			Nodes: []workerPoolNodeView{
				{
					NodeID:                     "node-a",
					TotalWorkers:               2,
					ActiveWorkers:              2,
					Health:                     "active",
					CapacityUtilizationPercent: 100,
					ExecutorDistribution: []auditFacetCount{
						{Key: "local", Count: 2},
					},
					WorkerStates: map[string]int{"running": 1, "leased": 1},
				},
				{
					NodeID:                     "node-b",
					TotalWorkers:               1,
					IdleWorkers:                1,
					StaleWorkers:               1,
					MissingHeartbeatWorkers:    1,
					Health:                     "degraded",
					CapacityUtilizationPercent: 0,
					ExecutorDistribution: []auditFacetCount{
						{Key: "ray", Count: 1},
					},
					WorkerStates: map[string]int{"idle": 1},
				},
			},
		},
		WorkerPoolHealth: &workerPoolHealthSummary{
			WorkersWithHeartbeat:    2,
			WorkersMissingHeartbeat: 1,
			StaleWorkers:            1,
		},
	}, controlCenterFilters{})

	for _, want := range []string{
		"## Worker Pool",
		"Capacity utilization: 66.7%",
		"Executor distribution: local=2, ray=1",
		"Node health: active=1, degraded=1",
		"## Worker Pool Nodes",
		"node-a: health=active workers=2 active=2 idle=0 stale=0 missing_heartbeat=0 capacity=100.0%",
		"executors: local=2",
		"worker states: leased=1, running=1",
		"node-b: health=degraded workers=1 active=0 idle=1 stale=1 missing_heartbeat=1 capacity=0.0%",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got %s", want, markdown)
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
		EventPlan: events.NewDurabilityPlanWithBrokerConfig("http", "broker_replicated", 3, events.BrokerRuntimeConfig{
			Driver:             "kafka",
			URLs:               []string{"kafka-1:9092"},
			Topic:              "bigclaw.events",
			ConsumerGroup:      "bigclaw-reviewers",
			PublishTimeout:     5 * time.Second,
			ReplayLimit:        1024,
			CheckpointInterval: 10 * time.Second,
		}),
		Control: controller,
		Worker:  fakeWorkerPoolStatus{},
		Now:     func() time.Time { return time.Unix(1700007200, 0) },
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
		{ID: "evt-replayed", Type: domain.EventTaskQueued, TaskID: "diag-local", TraceID: "trace-local", Timestamp: base.Add(8 * time.Second), Payload: map[string]any{"replayed": true}},
		{ID: "evt-dead-untracked", Type: domain.EventTaskDeadLetter, TaskID: "diag-untracked", TraceID: "trace-untracked", Timestamp: base.Add(9 * time.Second), Payload: map[string]any{"message": "manual dead letter"}},
		{ID: "evt-lease-acquired", Type: domain.EventSubscriberLeaseAcquired, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(10 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a", "consumer_id": "node-a"}},
		{ID: "evt-lease-rejected", Type: domain.EventSubscriberLeaseRejected, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(11 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a", "reason": "lease held"}},
		{ID: "evt-lease-expired", Type: domain.EventSubscriberLeaseExpired, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(12 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a"}},
		{ID: "evt-takeover-succeeded", Type: domain.EventSubscriberTakeoverSucceeded, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(13 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a"}},
		{ID: "evt-checkpoint-committed", Type: domain.EventSubscriberCheckpointCommitted, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(14 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a"}},
		{ID: "evt-checkpoint-rejected", Type: domain.EventSubscriberCheckpointRejected, TaskID: "diag-k8s", TraceID: "trace-k8s", Timestamp: base.Add(15 * time.Second), Payload: map[string]any{"group_id": "live-g1", "subscriber_id": "sub-a", "reason": "subscriber checkpoint lease fenced"}},
	} {
		recorder.Record(event)
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v2/control-center?team=platform&project=alpha&limit=10&audit_limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		EventDurability struct {
			RolloutScorecard struct {
				Status string `json:"status"`
			} `json:"rollout_scorecard"`
		} `json:"event_durability"`
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
			CoordinationLeader struct {
				Endpoint      string `json:"endpoint"`
				ElectionModel string `json:"election_model"`
				Status        string `json:"status"`
				LeaderPresent bool   `json:"leader_present"`
			} `json:"coordination_leader_election"`
			SharedQueueDiagnostics struct {
				DeadLetterBacklog         int   `json:"dead_letter_backlog"`
				DeadLetterEvents          int   `json:"dead_letter_events"`
				ReplayedQueueEvents       int   `json:"replayed_queue_events"`
				LeaseAcquiredEvents       int   `json:"lease_acquired_events"`
				LeaseRejectedEvents       int   `json:"lease_rejected_events"`
				LeaseExpiredEvents        int   `json:"lease_expired_events"`
				TakeoverSucceededEvents   int   `json:"takeover_succeeded_events"`
				CheckpointCommittedEvents int   `json:"checkpoint_committed_events"`
				CheckpointRejectedEvents  int   `json:"checkpoint_rejected_events"`
				LeaseFencedEvents         int   `json:"lease_fenced_events"`
				CheckpointResetsRecent    int   `json:"checkpoint_resets_recent"`
				RetentionWatermarkVisible bool  `json:"retention_watermark_available"`
				RetentionTrimmedThrough   int64 `json:"retention_trimmed_through_sequence"`
			} `json:"shared_queue_diagnostics"`
			LiveShadowMirror struct {
				ReportPath           string   `json:"report_path"`
				CanonicalSummaryPath string   `json:"canonical_summary_path"`
				SummaryPath          string   `json:"summary_path"`
				Status               string   `json:"status"`
				Severity             string   `json:"severity"`
				ReviewerLinks        []string `json:"reviewer_links"`
				Summary              struct {
					ParityOKCount      int `json:"parity_ok_count"`
					DriftDetectedCount int `json:"drift_detected_count"`
					FreshInputs        int `json:"fresh_inputs"`
				} `json:"summary"`
				RollbackTriggerSurface struct {
					Status string `json:"status"`
				} `json:"rollback_trigger_surface"`
			} `json:"live_shadow_mirror_scorecard"`
			BrokerReviewPack struct {
				Status                string   `json:"status"`
				SummaryPath           string   `json:"summary_path"`
				ReportPath            string   `json:"report_path"`
				ValidationPackPath    string   `json:"validation_pack_path"`
				ArtifactDirectory     string   `json:"artifact_directory"`
				ReviewerLinks         []string `json:"reviewer_links"`
				AmbiguousPublishProof struct {
					Path       string   `json:"path"`
					ScenarioID string   `json:"scenario_id"`
					Outcomes   []string `json:"outcomes"`
				} `json:"ambiguous_publish_proof"`
			} `json:"broker_review_pack"`
			TraceBundle struct {
				AmbiguousPublishProof struct {
					Path       string   `json:"path"`
					ScenarioID string   `json:"scenario_id"`
					Outcomes   []string `json:"outcomes"`
				} `json:"ambiguous_publish_proof"`
			} `json:"trace_export_bundle"`
			MigrationReviewPack struct {
				Status                 string   `json:"status"`
				ReadinessReportPath    string   `json:"readiness_report_path"`
				ScorecardPath          string   `json:"scorecard_path"`
				CanonicalSummaryPath   string   `json:"canonical_summary_path"`
				SummaryPath            string   `json:"summary_path"`
				IndexPath              string   `json:"index_path"`
				FollowUpDigestPath     string   `json:"follow_up_digest_path"`
				RollbackTriggerPath    string   `json:"rollback_trigger_path"`
				ReviewerLinks          []string `json:"reviewer_links"`
				RollbackTriggerSurface struct {
					ReportPath string `json:"report_path"`
					Issue      struct {
						ID   string `json:"id"`
						Slug string `json:"slug"`
					} `json:"issue"`
					Summary struct {
						Status                   string `json:"status"`
						AutomationBoundary       string `json:"automation_boundary"`
						AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
						CutoverGate              string `json:"cutover_gate"`
						Distinctions             struct {
							Blockers        int `json:"blockers"`
							Warnings        int `json:"warnings"`
							ManualOnlyPaths int `json:"manual_only_paths"`
						} `json:"distinctions"`
					} `json:"summary"`
					ReviewerLinks []string `json:"reviewer_links"`
				} `json:"rollback_trigger_surface"`
				LiveShadowMirrorScorecard struct {
					ReportPath           string `json:"report_path"`
					CanonicalSummaryPath string `json:"canonical_summary_path"`
					Summary              struct {
						ParityOKCount         int  `json:"parity_ok_count"`
						DriftDetectedCount    int  `json:"drift_detected_count"`
						MatrixMismatched      int  `json:"matrix_mismatched"`
						CorpusCoveragePresent bool `json:"corpus_coverage_present"`
					} `json:"summary"`
					RollbackTriggerSurface struct {
						AutomationBoundary string `json:"automation_boundary"`
						SummaryPath        string `json:"summary_path"`
					} `json:"rollback_trigger_surface"`
				} `json:"live_shadow_mirror_scorecard"`
			} `json:"migration_review_pack"`
			BrokerStubFanoutIsolation struct {
				ReportPath string `json:"report_path"`
				Summary    struct {
					ScenarioCount          int  `json:"scenario_count"`
					IsolatedScenarios      int  `json:"isolated_scenarios"`
					StalledScenarios       int  `json:"stalled_scenarios"`
					ReplayBacklogEvents    int  `json:"replay_backlog_events"`
					ReplayStepDelayMS      int  `json:"replay_step_delay_ms"`
					LiveDeliveryDeadlineMS int  `json:"live_delivery_deadline_ms"`
					IsolationMaintained    bool `json:"isolation_maintained"`
				} `json:"summary"`
				Scenarios []struct {
					Name                  string `json:"name"`
					Status                string `json:"status"`
					ReplayBacklogEvents   int    `json:"replay_backlog_events"`
					ReplayStepDelayMS     int    `json:"replay_step_delay_ms"`
					ReplayDrainsAfterLive bool   `json:"replay_drains_after_live"`
				} `json:"scenarios"`
			} `json:"broker_stub_fanout_isolation"`
			DeliveryAckReadiness struct {
				ReportPath string `json:"report_path"`
				Summary    struct {
					ExplicitAckBackends  int `json:"explicit_ack_backends"`
					DurableAckBackends   int `json:"durable_ack_backends"`
					BestEffortBackends   int `json:"best_effort_backends"`
					ContractOnlyBackends int `json:"contract_only_backends"`
				} `json:"summary"`
				Backends []struct {
					Backend              string `json:"backend"`
					AcknowledgementClass string `json:"acknowledgement_class"`
				} `json:"backends"`
			} `json:"delivery_ack_readiness"`
			PublishAckOutcomes struct {
				ReportPath string `json:"report_path"`
				Summary    struct {
					ScenarioID         string `json:"scenario_id"`
					CommittedCount     int    `json:"committed_count"`
					RejectedCount      int    `json:"rejected_count"`
					UnknownCommitCount int    `json:"unknown_commit_count"`
				} `json:"summary"`
				Outcomes []struct {
					Outcome string `json:"outcome"`
				} `json:"outcomes"`
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
				Backends []struct {
					Backend          string `json:"backend"`
					RuntimeReadiness string `json:"runtime_readiness"`
				} `json:"backends"`
			} `json:"sequence_bridge_surface"`
			ValidationBundleContinuation struct {
				ReportPath     string   `json:"report_path"`
				ScorecardPath  string   `json:"scorecard_path"`
				DigestPath     string   `json:"digest_path"`
				Recommendation string   `json:"recommendation"`
				ReviewerLinks  []string `json:"reviewer_links"`
				Summary        struct {
					RecentBundleCount                           int  `json:"recent_bundle_count"`
					AllExecutorTracksHaveRepeatedRecentCoverage bool `json:"all_executor_tracks_have_repeated_recent_coverage"`
					SharedQueueCompanionAvailable               bool `json:"shared_queue_companion_available"`
					CrossNodeCompletions                        int  `json:"cross_node_completions"`
				} `json:"summary"`
				ExecutorLanes []struct {
					Lane                   string `json:"lane"`
					LatestStatus           string `json:"latest_status"`
					LatestEnabled          bool   `json:"latest_enabled"`
					ConsecutiveSuccesses   int    `json:"consecutive_successes"`
					AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
				} `json:"executor_lanes"`
				NextActions []string `json:"next_actions"`
			} `json:"validation_bundle_continuation"`
			RolloutReport struct {
				Markdown  string `json:"markdown"`
				ExportURL string `json:"export_url"`
			} `json:"rollout_report"`
		} `json:"distributed_diagnostics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed diagnostics: %v", err)
	}
	if decoded.EventDurability.RolloutScorecard.Status != "blocked" {
		t.Fatalf("expected control center event durability scorecard, got %+v", decoded.EventDurability)
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
	if decoded.Diagnostics.CoordinationLeader.Endpoint != coordinationLeaderEndpoint ||
		decoded.Diagnostics.CoordinationLeader.ElectionModel != "subscriber_lease" ||
		decoded.Diagnostics.CoordinationLeader.Status != "unavailable" ||
		decoded.Diagnostics.CoordinationLeader.LeaderPresent {
		t.Fatalf("unexpected coordination leader diagnostics payload: %+v", decoded.Diagnostics.CoordinationLeader)
	}
	if decoded.Diagnostics.SharedQueueDiagnostics.DeadLetterBacklog != 0 ||
		decoded.Diagnostics.SharedQueueDiagnostics.DeadLetterEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.ReplayedQueueEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.LeaseAcquiredEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.LeaseRejectedEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.LeaseExpiredEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.TakeoverSucceededEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.CheckpointCommittedEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.CheckpointRejectedEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.LeaseFencedEvents != 1 ||
		decoded.Diagnostics.SharedQueueDiagnostics.CheckpointResetsRecent != 0 ||
		decoded.Diagnostics.SharedQueueDiagnostics.RetentionWatermarkVisible ||
		decoded.Diagnostics.SharedQueueDiagnostics.RetentionTrimmedThrough != 0 {
		t.Fatalf("unexpected shared queue diagnostics payload: %+v", decoded.Diagnostics.SharedQueueDiagnostics)
	}
	if decoded.Diagnostics.LiveShadowMirror.ReportPath != liveShadowMirrorScorecardPath ||
		decoded.Diagnostics.LiveShadowMirror.CanonicalSummaryPath != liveShadowSummaryPath ||
		decoded.Diagnostics.LiveShadowMirror.SummaryPath != "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" ||
		decoded.Diagnostics.LiveShadowMirror.Status != "parity-ok" ||
		decoded.Diagnostics.LiveShadowMirror.Severity != "none" ||
		decoded.Diagnostics.LiveShadowMirror.Summary.ParityOKCount != 4 ||
		decoded.Diagnostics.LiveShadowMirror.Summary.DriftDetectedCount != 0 ||
		decoded.Diagnostics.LiveShadowMirror.Summary.FreshInputs != 2 ||
		decoded.Diagnostics.LiveShadowMirror.RollbackTriggerSurface.Status != "manual-review-required" {
		t.Fatalf("unexpected live shadow diagnostics payload: %+v", decoded.Diagnostics.LiveShadowMirror)
	}
	if len(decoded.Diagnostics.LiveShadowMirror.ReviewerLinks) == 0 {
		t.Fatalf("expected live shadow reviewer links, got %+v", decoded.Diagnostics.LiveShadowMirror)
	}
	if decoded.Diagnostics.BrokerReviewPack.Status != "checked_in_stub_evidence" ||
		decoded.Diagnostics.BrokerReviewPack.SummaryPath != "docs/reports/broker-validation-summary.json" ||
		decoded.Diagnostics.BrokerReviewPack.ReportPath != "docs/reports/broker-failover-stub-report.json" ||
		decoded.Diagnostics.BrokerReviewPack.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" ||
		decoded.Diagnostics.BrokerReviewPack.ArtifactDirectory != "docs/reports/broker-failover-stub-artifacts" {
		t.Fatalf("unexpected broker review pack payload: %+v", decoded.Diagnostics.BrokerReviewPack)
	}
	if len(decoded.Diagnostics.BrokerReviewPack.ReviewerLinks) != 2 ||
		decoded.Diagnostics.BrokerReviewPack.ReviewerLinks[0] != "docs/reports/live-validation-index.json" ||
		decoded.Diagnostics.BrokerReviewPack.ReviewerLinks[1] != "docs/reports/review-readiness.md" {
		t.Fatalf("unexpected broker review pack reviewer links: %+v", decoded.Diagnostics.BrokerReviewPack.ReviewerLinks)
	}
	if decoded.Diagnostics.BrokerReviewPack.AmbiguousPublishProof.Path != "docs/reports/ambiguous-publish-outcome-proof-summary.json" ||
		decoded.Diagnostics.BrokerReviewPack.AmbiguousPublishProof.ScenarioID != "BF-05" ||
		len(decoded.Diagnostics.BrokerReviewPack.AmbiguousPublishProof.Outcomes) != 3 {
		t.Fatalf("unexpected broker review pack ambiguous publish proof: %+v", decoded.Diagnostics.BrokerReviewPack.AmbiguousPublishProof)
	}
	if decoded.Diagnostics.TraceBundle.AmbiguousPublishProof.Path != "docs/reports/ambiguous-publish-outcome-proof-summary.json" ||
		decoded.Diagnostics.TraceBundle.AmbiguousPublishProof.ScenarioID != "BF-05" ||
		len(decoded.Diagnostics.TraceBundle.AmbiguousPublishProof.Outcomes) != 3 {
		t.Fatalf("unexpected trace bundle ambiguous publish proof: %+v", decoded.Diagnostics.TraceBundle.AmbiguousPublishProof)
	}
	if decoded.Diagnostics.MigrationReviewPack.Status != "parity-ok" ||
		decoded.Diagnostics.MigrationReviewPack.ReadinessReportPath != migrationReadinessReportPath ||
		decoded.Diagnostics.MigrationReviewPack.ScorecardPath != liveShadowMirrorScorecardPath ||
		decoded.Diagnostics.MigrationReviewPack.CanonicalSummaryPath != liveShadowSummaryPath ||
		decoded.Diagnostics.MigrationReviewPack.SummaryPath != "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" ||
		decoded.Diagnostics.MigrationReviewPack.IndexPath != liveShadowIndexPath ||
		decoded.Diagnostics.MigrationReviewPack.FollowUpDigestPath != liveShadowComparisonFollowUpDigestPath ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerPath != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected migration review pack payload: %+v", decoded.Diagnostics.MigrationReviewPack)
	}
	if decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReportPath != rollbackTriggerSurfacePath ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Issue.ID != "OPE-254" ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Issue.Slug != "BIG-PAR-088" ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Status != "manual-review-required" ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.AutomationBoundary != "manual_only" ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.AutomatedRollbackTrigger ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.CutoverGate != "reviewer_enforced" ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.Blockers != 3 ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.Warnings != 1 ||
		decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.Summary.Distinctions.ManualOnlyPaths != 2 {
		t.Fatalf("unexpected migration rollback trigger payload: %+v", decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface)
	}
	if len(decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReviewerLinks) == 0 || decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReviewerLinks[0] != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected migration rollback reviewer links: %+v", decoded.Diagnostics.MigrationReviewPack.RollbackTriggerSurface.ReviewerLinks)
	}
	if decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.ReportPath != liveShadowMirrorScorecardPath ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.CanonicalSummaryPath != liveShadowSummaryPath ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.ParityOKCount != 4 ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.DriftDetectedCount != 0 ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.MatrixMismatched != 0 ||
		!decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.Summary.CorpusCoveragePresent ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.RollbackTriggerSurface.AutomationBoundary != "manual_only" ||
		decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard.RollbackTriggerSurface.SummaryPath != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected migration live shadow scorecard payload: %+v", decoded.Diagnostics.MigrationReviewPack.LiveShadowMirrorScorecard)
	}
	if len(decoded.Diagnostics.MigrationReviewPack.ReviewerLinks) == 0 ||
		decoded.Diagnostics.MigrationReviewPack.ReviewerLinks[0] != liveShadowSummaryPath ||
		decoded.Diagnostics.MigrationReviewPack.ReviewerLinks[len(decoded.Diagnostics.MigrationReviewPack.ReviewerLinks)-1] != rollbackTriggerSurfacePath {
		t.Fatalf("unexpected migration review links: %+v", decoded.Diagnostics.MigrationReviewPack.ReviewerLinks)
	}
	if decoded.Diagnostics.BrokerStubFanoutIsolation.ReportPath != brokerStubFanoutIsolationEvidencePackPath ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.ScenarioCount != 1 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.IsolatedScenarios != 1 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.StalledScenarios != 0 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.ReplayBacklogEvents != 4 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.ReplayStepDelayMS != 30 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.LiveDeliveryDeadlineMS != 50 ||
		!decoded.Diagnostics.BrokerStubFanoutIsolation.Summary.IsolationMaintained {
		t.Fatalf("unexpected broker fanout isolation payload: %+v", decoded.Diagnostics.BrokerStubFanoutIsolation)
	}
	if len(decoded.Diagnostics.BrokerStubFanoutIsolation.Scenarios) != 1 ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Scenarios[0].Name != "replay_catchup_does_not_block_live_publish" ||
		decoded.Diagnostics.BrokerStubFanoutIsolation.Scenarios[0].Status != "isolated" ||
		!decoded.Diagnostics.BrokerStubFanoutIsolation.Scenarios[0].ReplayDrainsAfterLive {
		t.Fatalf("unexpected broker fanout isolation backend detail: %+v", decoded.Diagnostics.BrokerStubFanoutIsolation.Scenarios)
	}
	if decoded.Diagnostics.DeliveryAckReadiness.ReportPath != deliveryAckReadinessSurfacePath ||
		decoded.Diagnostics.DeliveryAckReadiness.Summary.ExplicitAckBackends != 3 ||
		decoded.Diagnostics.DeliveryAckReadiness.Summary.DurableAckBackends != 2 ||
		decoded.Diagnostics.DeliveryAckReadiness.Summary.BestEffortBackends != 1 ||
		decoded.Diagnostics.DeliveryAckReadiness.Summary.ContractOnlyBackends != 1 {
		t.Fatalf("unexpected delivery ack readiness payload: %+v", decoded.Diagnostics.DeliveryAckReadiness)
	}
	if len(decoded.Diagnostics.DeliveryAckReadiness.Backends) != 5 ||
		decoded.Diagnostics.DeliveryAckReadiness.Backends[0].Backend != "memory" ||
		decoded.Diagnostics.DeliveryAckReadiness.Backends[0].AcknowledgementClass != "best_effort_only" {
		t.Fatalf("unexpected delivery ack readiness backend detail: %+v", decoded.Diagnostics.DeliveryAckReadiness.Backends)
	}
	if decoded.Diagnostics.PublishAckOutcomes.ReportPath != publishAckOutcomeSurfacePath ||
		decoded.Diagnostics.PublishAckOutcomes.Summary.ScenarioID != "BF-05" ||
		decoded.Diagnostics.PublishAckOutcomes.Summary.CommittedCount != 1 ||
		decoded.Diagnostics.PublishAckOutcomes.Summary.RejectedCount != 1 ||
		decoded.Diagnostics.PublishAckOutcomes.Summary.UnknownCommitCount != 1 ||
		len(decoded.Diagnostics.PublishAckOutcomes.Outcomes) != 3 {
		t.Fatalf("unexpected publish ack outcomes payload: %+v", decoded.Diagnostics.PublishAckOutcomes)
	}
	if decoded.Diagnostics.SequenceBridge.ReportPath != sequenceBridgeSurfacePath ||
		decoded.Diagnostics.SequenceBridge.Summary.BackendCount != 5 ||
		decoded.Diagnostics.SequenceBridge.Summary.LiveProvenBackends != 3 ||
		decoded.Diagnostics.SequenceBridge.Summary.HarnessProvenBackends != 1 ||
		decoded.Diagnostics.SequenceBridge.Summary.ContractOnlyBackends != 1 ||
		decoded.Diagnostics.SequenceBridge.Summary.OneToOneMappings != 2 ||
		decoded.Diagnostics.SequenceBridge.Summary.ProviderEpochBridgedBackends != 3 ||
		len(decoded.Diagnostics.SequenceBridge.Backends) != 5 {
		t.Fatalf("unexpected sequence bridge payload: %+v", decoded.Diagnostics.SequenceBridge)
	}
	if decoded.Diagnostics.ValidationBundleContinuation.ReportPath != validationBundleContinuationGatePath ||
		decoded.Diagnostics.ValidationBundleContinuation.ScorecardPath != validationBundleContinuationScorecardPath ||
		decoded.Diagnostics.ValidationBundleContinuation.DigestPath != "docs/reports/validation-bundle-continuation-digest.md" ||
		decoded.Diagnostics.ValidationBundleContinuation.Recommendation != "go" {
		t.Fatalf("unexpected continuation gate payload: %+v", decoded.Diagnostics.ValidationBundleContinuation)
	}
	if decoded.Diagnostics.ValidationBundleContinuation.Summary.RecentBundleCount != 3 ||
		!decoded.Diagnostics.ValidationBundleContinuation.Summary.AllExecutorTracksHaveRepeatedRecentCoverage ||
		!decoded.Diagnostics.ValidationBundleContinuation.Summary.SharedQueueCompanionAvailable ||
		decoded.Diagnostics.ValidationBundleContinuation.Summary.CrossNodeCompletions != 99 {
		t.Fatalf("unexpected continuation gate summary: %+v", decoded.Diagnostics.ValidationBundleContinuation.Summary)
	}
	if len(decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes) != 3 ||
		decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes[0].Lane != "local" ||
		decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes[0].LatestStatus != "succeeded" ||
		!decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes[0].LatestEnabled ||
		decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes[0].ConsecutiveSuccesses != 3 ||
		!decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes[0].AllRecentRunsSucceeded {
		t.Fatalf("unexpected continuation executor lanes: %+v", decoded.Diagnostics.ValidationBundleContinuation.ExecutorLanes)
	}
	if len(decoded.Diagnostics.ValidationBundleContinuation.ReviewerLinks) != 3 ||
		decoded.Diagnostics.ValidationBundleContinuation.ReviewerLinks[0] != "docs/reports/live-validation-index.md" {
		t.Fatalf("unexpected continuation gate reviewer links: %+v", decoded.Diagnostics.ValidationBundleContinuation.ReviewerLinks)
	}
	if len(decoded.Diagnostics.ValidationBundleContinuation.NextActions) < 4 {
		t.Fatalf("expected merged continuation next actions, got %+v", decoded.Diagnostics.ValidationBundleContinuation.NextActions)
	}
	if !strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "# BigClaw Distributed Diagnostics Report") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Worker Pool") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Executor distribution") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Worker Pool Nodes") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Takeover owners") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Coordination Leader Election") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Shared Queue Coordination") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Lease fenced events: 1") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Replayed queue events: 1") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Live Shadow Mirror Scorecard") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Broker Failover Review Pack") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Migration Readiness Review Pack") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Rollback Trigger Surface") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Broker Stub Live Fanout Isolation") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Delivery Acknowledgement Readiness") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Publish Acknowledgement Outcome Ledger") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Durable Sequence Bridge") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "## Validation Bundle Continuation Gate") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, liveShadowSummaryPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, liveShadowMirrorScorecardPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, migrationReadinessReportPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, liveShadowComparisonFollowUpDigestPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, rollbackTriggerSurfacePath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "reviewer_enforced") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "docs/reports/broker-validation-summary.json") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "docs/reports/ambiguous-publish-outcome-proof-summary.json") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, brokerStubFanoutIsolationEvidencePackPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, deliveryAckReadinessSurfacePath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, publishAckOutcomeSurfacePath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, sequenceBridgeSurfacePath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, validationBundleContinuationGatePath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, validationBundleContinuationScorecardPath) ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Lane local: latest_status=succeeded") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "docs/reports/validation-bundle-continuation-digest.md") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "docs/reports/broker-failover-stub-artifacts") ||
		!strings.Contains(decoded.Diagnostics.RolloutReport.ExportURL, "/v2/reports/distributed/export") {
		t.Fatalf("unexpected rollout report payload: %+v", decoded.Diagnostics.RolloutReport)
	}
}

func TestV2ControlCenterIncludesCoordinationCapabilitySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Coordination struct {
			Ticket           string   `json:"ticket"`
			CurrentCeiling   []string `json:"current_ceiling"`
			NextRuntimeHooks []string `json:"next_runtime_hooks"`
			EvidenceSources  struct {
				SharedQueueReport     string   `json:"shared_queue_report"`
				TakeoverHarnessReport string   `json:"takeover_harness_report"`
				SupportingDocs        []string `json:"supporting_docs"`
			} `json:"evidence_sources"`
			Capabilities []struct {
				Name              string   `json:"name"`
				ContractOnly      bool     `json:"contract_only"`
				HarnessProven     bool     `json:"harness_proven"`
				LiveProven        bool     `json:"live_proven"`
				SourceReportLinks []string `json:"source_report_links"`
			} `json:"capabilities"`
		} `json:"coordination_capability_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center coordination payload: %v", err)
	}
	if decoded.Coordination.Ticket != "BIG-PAR-085-local-prework" {
		t.Fatalf("unexpected coordination ticket payload: %+v", decoded.Coordination)
	}
	if decoded.Coordination.EvidenceSources.SharedQueueReport == "" || decoded.Coordination.EvidenceSources.TakeoverHarnessReport == "" || len(decoded.Coordination.EvidenceSources.SupportingDocs) != 3 {
		t.Fatalf("unexpected coordination evidence sources: %+v", decoded.Coordination.EvidenceSources)
	}
	if len(decoded.Coordination.CurrentCeiling) != 3 || len(decoded.Coordination.NextRuntimeHooks) != 3 {
		t.Fatalf("unexpected coordination summary detail: %+v", decoded.Coordination)
	}
	if len(decoded.Coordination.Capabilities) < 3 {
		t.Fatalf("expected capability surface in control center payload, got %+v", decoded.Coordination)
	}
	contractOnly := decoded.Coordination.Capabilities[4]
	if contractOnly.Name != "partitioned_topic_routing" || !contractOnly.ContractOnly || contractOnly.LiveProven || len(contractOnly.SourceReportLinks) == 0 {
		t.Fatalf("unexpected contract-only coordination entry: %+v", contractOnly)
	}
}

func TestV2ControlCenterIncludesAdmissionPolicySummary(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		AdmissionPolicy struct {
			PolicyMode       string   `json:"policy_mode"`
			Enforced         bool     `json:"enforced"`
			EvidenceSources  []string `json:"evidence_sources"`
			RecommendedLanes []struct {
				Name                  string  `json:"name"`
				MaxQueuedTasks        int     `json:"max_queued_tasks"`
				SubmitWorkers         int     `json:"submit_workers"`
				ObservedThroughputTPS float64 `json:"observed_throughput_tasks_per_sec"`
			} `json:"recommended_lanes"`
		} `json:"admission_policy_summary"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center admission policy payload: %v", err)
	}
	if decoded.AdmissionPolicy.PolicyMode != "advisory_only" || decoded.AdmissionPolicy.Enforced {
		t.Fatalf("expected advisory-only admission policy in control center payload, got %+v", decoded.AdmissionPolicy)
	}
	if len(decoded.AdmissionPolicy.EvidenceSources) < 4 ||
		!containsString(decoded.AdmissionPolicy.EvidenceSources, capacityCertificationMatrixPath) ||
		!containsString(decoded.AdmissionPolicy.EvidenceSources, capacityCertificationReportPath) {
		t.Fatalf("unexpected admission policy evidence sources: %+v", decoded.AdmissionPolicy.EvidenceSources)
	}
	if len(decoded.AdmissionPolicy.RecommendedLanes) != 2 {
		t.Fatalf("expected recommended admission lanes in control center payload, got %+v", decoded.AdmissionPolicy.RecommendedLanes)
	}
	if decoded.AdmissionPolicy.RecommendedLanes[0].Name != "recommended-local-sustained" || decoded.AdmissionPolicy.RecommendedLanes[0].MaxQueuedTasks != 1000 || decoded.AdmissionPolicy.RecommendedLanes[0].SubmitWorkers != 24 || decoded.AdmissionPolicy.RecommendedLanes[0].ObservedThroughputTPS != 9.607 {
		t.Fatalf("unexpected sustained admission lane in control center payload: %+v", decoded.AdmissionPolicy.RecommendedLanes[0])
	}
}

func TestV2ControlCenterIncludesCoordinationLeaderElectionSurface(t *testing.T) {
	now := time.Unix(1700000000, 0)
	coordinator := events.NewSubscriberLeaseCoordinator()
	lease, err := coordinator.Acquire(events.LeaseRequest{
		GroupID:      coordinationLeaderGroupID,
		SubscriberID: coordinationLeaderSubscriberID,
		ConsumerID:   "node-a",
		TTL:          30 * time.Second,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("seed leader lease: %v", err)
	}
	server := &Server{
		Recorder:         observability.NewRecorder(),
		Queue:            queue.NewMemoryQueue(),
		Bus:              events.NewBus(),
		Control:          control.New(),
		SubscriberLeases: coordinator,
		Now:              func() time.Time { return now.Add(5 * time.Second) },
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Leader struct {
			Endpoint            string `json:"endpoint"`
			Status              string `json:"status"`
			LeaderPresent       bool   `json:"leader_present"`
			RemainingTTLSeconds int64  `json:"remaining_ttl_seconds"`
			Lease               struct {
				ConsumerID string `json:"consumer_id"`
				LeaseToken string `json:"lease_token"`
				LeaseEpoch int64  `json:"lease_epoch"`
			} `json:"lease"`
		} `json:"coordination_leader_election"`
		LeaderElectionCapability struct {
			Summary struct {
				BackendCount          int `json:"backend_count"`
				LiveProvenBackends    int `json:"live_proven_backends"`
				HarnessProvenBackends int `json:"harness_proven_backends"`
				ContractOnlyBackends  int `json:"contract_only_backends"`
			} `json:"summary"`
		} `json:"leader_election_capability"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center leader surface: %v", err)
	}
	if decoded.Leader.Endpoint != coordinationLeaderEndpoint || decoded.Leader.Status != "active" || !decoded.Leader.LeaderPresent {
		t.Fatalf("unexpected leader surface: %+v", decoded.Leader)
	}
	if decoded.Leader.Lease.ConsumerID != "node-a" || decoded.Leader.Lease.LeaseToken != lease.LeaseToken || decoded.Leader.Lease.LeaseEpoch != lease.LeaseEpoch {
		t.Fatalf("unexpected leader lease payload: %+v", decoded.Leader)
	}
	if decoded.Leader.RemainingTTLSeconds <= 0 {
		t.Fatalf("expected positive ttl remaining, got %+v", decoded.Leader)
	}
	if decoded.LeaderElectionCapability.Summary.BackendCount != 4 || decoded.LeaderElectionCapability.Summary.LiveProvenBackends != 1 || decoded.LeaderElectionCapability.Summary.HarnessProvenBackends != 1 || decoded.LeaderElectionCapability.Summary.ContractOnlyBackends != 2 {
		t.Fatalf("unexpected control center leader election capability summary: %+v", decoded.LeaderElectionCapability.Summary)
	}
}

func TestCoordinationLeaderEndpointsAcquireStatusAndTakeover(t *testing.T) {
	now := time.Unix(1700000000, 0)
	current := now
	server := &Server{
		Recorder:         observability.NewRecorder(),
		Queue:            queue.NewMemoryQueue(),
		Bus:              events.NewBus(),
		SubscriberLeases: events.NewSubscriberLeaseCoordinator(),
		Now:              func() time.Time { return current },
	}
	handler := server.Handler()

	acquire := httptest.NewRecorder()
	handler.ServeHTTP(acquire, httptest.NewRequest(http.MethodPost, coordinationLeaderEndpoint, strings.NewReader(`{"consumer_id":"node-a","ttl_seconds":30}`)))
	if acquire.Code != http.StatusOK {
		t.Fatalf("expected first acquire 200, got %d %s", acquire.Code, acquire.Body.String())
	}
	var acquired struct {
		Lease struct {
			ConsumerID string `json:"consumer_id"`
			LeaseToken string `json:"lease_token"`
			LeaseEpoch int64  `json:"lease_epoch"`
		} `json:"lease"`
		LeaderElectionCapability struct {
			Summary struct {
				CurrentProofBackend string `json:"current_proof_backend"`
			} `json:"summary"`
		} `json:"leader_election_capability"`
	}
	if err := json.Unmarshal(acquire.Body.Bytes(), &acquired); err != nil {
		t.Fatalf("decode acquire response: %v", err)
	}
	if acquired.Lease.ConsumerID != "node-a" || acquired.Lease.LeaseToken == "" || acquired.Lease.LeaseEpoch != 1 {
		t.Fatalf("unexpected acquired lease: %+v", acquired.Lease)
	}
	if acquired.LeaderElectionCapability.Summary.CurrentProofBackend != "shared_sqlite_subscriber_lease" {
		t.Fatalf("unexpected leader election capability on acquire: %+v", acquired.LeaderElectionCapability)
	}

	status := httptest.NewRecorder()
	handler.ServeHTTP(status, httptest.NewRequest(http.MethodGet, coordinationLeaderEndpoint, nil))
	if status.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d %s", status.Code, status.Body.String())
	}
	var statusDecoded struct {
		Leader struct {
			Status        string `json:"status"`
			LeaderPresent bool   `json:"leader_present"`
			Lease         struct {
				ConsumerID string `json:"consumer_id"`
			} `json:"lease"`
		} `json:"leader"`
		LeaderElectionCapability struct {
			ReportPath string `json:"report_path"`
			Backends   []struct {
				Name             string `json:"name"`
				RuntimeReadiness string `json:"runtime_readiness"`
			} `json:"backends"`
		} `json:"leader_election_capability"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &statusDecoded); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusDecoded.Leader.Status != "active" || !statusDecoded.Leader.LeaderPresent || statusDecoded.Leader.Lease.ConsumerID != "node-a" {
		t.Fatalf("unexpected leader status payload: %+v", statusDecoded.Leader)
	}
	if statusDecoded.LeaderElectionCapability.ReportPath != leaderElectionCapabilitySurfacePath || len(statusDecoded.LeaderElectionCapability.Backends) != 4 || statusDecoded.LeaderElectionCapability.Backends[0].Name != "shared_sqlite_subscriber_lease" {
		t.Fatalf("unexpected leader capability status payload: %+v", statusDecoded.LeaderElectionCapability)
	}

	current = now.Add(10 * time.Second)
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPost, coordinationLeaderEndpoint, strings.NewReader(`{"consumer_id":"node-b","ttl_seconds":30}`)))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected conflicting acquire 409, got %d %s", conflict.Code, conflict.Body.String())
	}

	current = now.Add(31 * time.Second)
	takeover := httptest.NewRecorder()
	handler.ServeHTTP(takeover, httptest.NewRequest(http.MethodPost, coordinationLeaderEndpoint, strings.NewReader(`{"consumer_id":"node-b","ttl_seconds":30}`)))
	if takeover.Code != http.StatusOK {
		t.Fatalf("expected takeover acquire 200, got %d %s", takeover.Code, takeover.Body.String())
	}
	var takeoverDecoded struct {
		Lease struct {
			ConsumerID string `json:"consumer_id"`
			LeaseEpoch int64  `json:"lease_epoch"`
		} `json:"lease"`
	}
	if err := json.Unmarshal(takeover.Body.Bytes(), &takeoverDecoded); err != nil {
		t.Fatalf("decode takeover response: %v", err)
	}
	if takeoverDecoded.Lease.ConsumerID != "node-b" || takeoverDecoded.Lease.LeaseEpoch != 2 {
		t.Fatalf("unexpected takeover lease: %+v", takeoverDecoded.Lease)
	}
}

func TestV2ControlCenterAuditIncludesCheckpointResetSummary(t *testing.T) {
	store, err := events.NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	base := time.Now()
	if err := store.Write(context.Background(), domain.Event{ID: "evt-cc-reset-1", Type: domain.EventTaskQueued, TaskID: "task-cc-reset", TraceID: "trace-cc-reset", Timestamp: base}); err != nil {
		t.Fatalf("write control-center reset event: %v", err)
	}
	if _, err := store.Acknowledge("subscriber-cc-reset", "evt-cc-reset-1", base.Add(time.Second)); err != nil {
		t.Fatalf("ack control-center reset checkpoint: %v", err)
	}
	if err := store.ResetCheckpoint("subscriber-cc-reset"); err != nil {
		t.Fatalf("reset control-center checkpoint: %v", err)
	}
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, EventLog: store, Control: control.New(), Now: time.Now}
	handler := server.Handler()

	auditResponse := httptest.NewRecorder()
	auditRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/audit?audit_limit=10", nil)
	auditRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	auditRequest.Header.Set("X-BigClaw-Actor", "ops-1")
	handler.ServeHTTP(auditResponse, auditRequest)
	if auditResponse.Code != http.StatusOK {
		t.Fatalf("expected control center audit 200, got %d %s", auditResponse.Code, auditResponse.Body.String())
	}
	body := auditResponse.Body.String()
	if !strings.Contains(body, "checkpoint_resets") || !strings.Contains(body, "subscriber-cc-reset") || !strings.Contains(body, "operator_reset") {
		t.Fatalf("expected checkpoint reset summary in control center audit payload, got %s", body)
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
	taskQueue := queue.NewMemoryQueue()
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-policy-endpoint-1",
			Source:   "clawhost",
			Title:    "align sales provider defaults",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":           "sales-app",
				"clawhost_default_provider": "openai",
				"clawhost_approval_flow":    "standard",
				"team":                      "platform",
				"project":                   "sales",
			},
		},
		{
			ID:       "clawhost-policy-endpoint-2",
			Labels:   []string{"clawhost"},
			Title:    "review support tenant override",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":             "support-app",
				"clawhost_default_provider":   "anthropic",
				"clawhost_provider_mode":      "tenant_override",
				"clawhost_provider_allowlist": "openai,google",
				"clawhost_takeover_required":  "true",
				"team":                        "support",
				"project":                     "care",
			},
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue policy task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), SchedulerPolicy: store, SchedulerRuntime: schedulerRuntime, Now: time.Now}
	handler := server.Handler()

	policyResponse := httptest.NewRecorder()
	policyRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/policy?team=platform&project=sales", nil)
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
		Filters          struct {
			Team    string `json:"team"`
			Project string `json:"project"`
		} `json:"filters"`
		Policy struct {
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
		ClawHost struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies      int `json:"active_policies"`
				ActiveTenants       int `json:"active_tenants"`
				ActiveApps          int `json:"active_apps"`
				ReviewRequired      int `json:"review_required"`
				TakeoverRequired    int `json:"takeover_required"`
				OutOfPolicyDefaults int `json:"out_of_policy_defaults"`
			} `json:"summary"`
			ObservedProviders []string `json:"observed_providers"`
			ReviewQueue       []struct {
				TaskID      string `json:"task_id"`
				DriftStatus string `json:"drift_status"`
			} `json:"review_queue"`
		} `json:"clawhost"`
		Report struct {
			Markdown  string `json:"markdown"`
			ExportURL string `json:"export_url"`
		} `json:"report"`
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
	if policyDecoded.Filters.Team != "platform" || policyDecoded.Filters.Project != "sales" {
		t.Fatalf("expected scoped filters in policy response, got %+v", policyDecoded.Filters)
	}
	if policyDecoded.ClawHost.Status != "active" || policyDecoded.ClawHost.Summary.ActivePolicies != 1 || policyDecoded.ClawHost.Summary.ActiveTenants != 1 || policyDecoded.ClawHost.Summary.ActiveApps != 1 || policyDecoded.ClawHost.Summary.ReviewRequired != 0 || policyDecoded.ClawHost.Summary.TakeoverRequired != 0 || policyDecoded.ClawHost.Summary.OutOfPolicyDefaults != 0 {
		t.Fatalf("unexpected ClawHost policy payload: %+v", policyDecoded.ClawHost)
	}
	if policyDecoded.ClawHost.Filters["team"] != "platform" || policyDecoded.ClawHost.Filters["project"] != "sales" {
		t.Fatalf("expected scoped ClawHost surface filters, got %+v", policyDecoded.ClawHost.Filters)
	}
	if len(policyDecoded.ClawHost.ReviewQueue) != 1 || policyDecoded.ClawHost.ReviewQueue[0].TaskID != "clawhost-policy-endpoint-1" || policyDecoded.ClawHost.ReviewQueue[0].DriftStatus != "aligned" {
		t.Fatalf("expected scoped ClawHost review queue, got %+v", policyDecoded.ClawHost.ReviewQueue)
	}
	for _, want := range []string{"openai"} {
		if !containsString(policyDecoded.ClawHost.ObservedProviders, want) {
			t.Fatalf("expected ClawHost observed provider %q, got %+v", want, policyDecoded.ClawHost.ObservedProviders)
		}
	}
	if containsString(policyDecoded.ClawHost.ObservedProviders, "anthropic") {
		t.Fatalf("did not expect unscoped provider in scoped response, got %+v", policyDecoded.ClawHost.ObservedProviders)
	}
	if policyDecoded.Report.ExportURL != "/v2/control-center/policy/export?project=sales&team=platform" ||
		!strings.Contains(policyDecoded.Report.Markdown, "# ClawHost Policy Surface") ||
		!strings.Contains(policyDecoded.Report.Markdown, "## Filters") ||
		!strings.Contains(policyDecoded.Report.Markdown, "- Team: `platform`") ||
		!strings.Contains(policyDecoded.Report.Markdown, "- Project: `sales`") ||
		!strings.Contains(policyDecoded.Report.Markdown, "clawhost-policy-endpoint-1") ||
		strings.Contains(policyDecoded.Report.Markdown, "clawhost-policy-endpoint-2") {
		t.Fatalf("expected policy report metadata in response, got %+v", policyDecoded.Report)
	}

	exportResponse := httptest.NewRecorder()
	exportRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/policy/export?team=platform&project=sales", nil)
	exportRequest.Header.Set("X-BigClaw-Role", "platform_admin")
	handler.ServeHTTP(exportResponse, exportRequest)
	if exportResponse.Code != http.StatusOK {
		t.Fatalf("expected policy export 200, got %d %s", exportResponse.Code, exportResponse.Body.String())
	}
	if contentType := exportResponse.Header().Get("Content-Type"); !strings.Contains(contentType, "text/markdown") {
		t.Fatalf("expected markdown content type, got %q", contentType)
	}
	if disposition := exportResponse.Header().Get("Content-Disposition"); !strings.Contains(disposition, "clawhost-policy-surface.md") {
		t.Fatalf("expected attachment filename, got %q", disposition)
	}
	for _, want := range []string{"# ClawHost Policy Surface", "## Filters", "- Team: `platform`", "- Project: `sales`", "tenant `tenant-a`", "provider `openai`", "Reason: provider default remains aligned with the shared app policy"} {
		if !strings.Contains(exportResponse.Body.String(), want) {
			t.Fatalf("expected %q in policy export, got %s", want, exportResponse.Body.String())
		}
	}
	if strings.Contains(exportResponse.Body.String(), "tenant `tenant-b`") {
		t.Fatalf("did not expect unscoped tenant in policy export, got %s", exportResponse.Body.String())
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

	scopedForbidden := httptest.NewRecorder()
	scopedForbiddenRequest := httptest.NewRequest(http.MethodGet, "/v2/control-center/policy?team=support", nil)
	scopedForbiddenRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	scopedForbiddenRequest.Header.Set("X-BigClaw-Team", "platform")
	handler.ServeHTTP(scopedForbidden, scopedForbiddenRequest)
	if scopedForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden policy scope for eng lead, got %d %s", scopedForbidden.Code, scopedForbidden.Body.String())
	}
}

func TestV2ControlCenterIncludesClawHostRolloutSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010400, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-upgrade-1",
			Source:   "clawhost",
			Title:    "upgrade sales-west",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"inventory_kind":            "claw",
				"claw_id":                   "claw-sales-west",
				"claw_name":                 "sales-west",
				"provider":                  "hetzner",
				"provider_status":           "running",
				"clawhost_update_available": "true",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-upgrade-2",
			Source:   "clawhost",
			Title:    "upgrade support-east",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"inventory_kind":             "claw",
				"claw_id":                    "claw-support-east",
				"claw_name":                  "support-east",
				"provider":                   "digitalocean",
				"provider_status":            "running",
				"clawhost_update_available":  "true",
				"clawhost_takeover_required": "true",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue control center rollout task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostRollout struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePlans      int `json:"active_plans"`
				TotalTargets     int `json:"total_targets"`
				TakeoverRequired int `json:"takeover_required"`
			} `json:"summary"`
			Plans []struct {
				Action       string `json:"action"`
				TargetCount  int    `json:"target_count"`
				TakeoverHook string `json:"takeover_hook"`
			} `json:"plans"`
		} `json:"clawhost_rollout_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center rollout response: %v", err)
	}
	if decoded.ClawHostRollout.Status != "active" || decoded.ClawHostRollout.Summary.ActivePlans != 2 || decoded.ClawHostRollout.Summary.TotalTargets != 2 || decoded.ClawHostRollout.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected ClawHost rollout control center surface: %+v", decoded.ClawHostRollout)
	}
	if decoded.ClawHostRollout.Filters["team"] != "" || decoded.ClawHostRollout.Filters["project"] != "" {
		t.Fatalf("expected unscoped rollout surface filters in control-center bundle, got %+v", decoded.ClawHostRollout.Filters)
	}
	if len(decoded.ClawHostRollout.Plans) != 2 || decoded.ClawHostRollout.Plans[0].Action != "upgrade" {
		t.Fatalf("expected upgrade rollout plans, got %+v", decoded.ClawHostRollout.Plans)
	}
}

func TestV2ControlCenterIncludesClawHostRecoverySurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010410, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-recovery-center-1",
			Source:   "clawhost",
			Title:    "recover sales-west",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_id":                    "claw-sales-west",
				"claw_name":                  "sales-west",
				"clawhost_lifecycle_actions": "start,restart,upgrade",
				"clawhost_pod_isolation":     "true",
				"clawhost_service_isolation": "true",
				"clawhost_takeover_required": "true",
				"clawhost_takeover_triggers": "proxy health regresses,session restore fails",
				"clawhost_recovery_evidence": "GET /status,/v2/reports/distributed",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-recovery-center-2",
			Source:   "clawhost",
			Title:    "recover support-east",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_id":                    "claw-support-east",
				"claw_name":                  "support-east",
				"clawhost_lifecycle_actions": "restart",
				"clawhost_pod_isolation":     "false",
				"clawhost_service_isolation": "true",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue control center recovery task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostRecovery struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets            int `json:"targets"`
				RecoverableTargets int `json:"recoverable_targets"`
				DegradedTargets    int `json:"degraded_targets"`
				IsolatedTargets    int `json:"isolated_targets"`
				TakeoverRequired   int `json:"takeover_required"`
				EvidenceArtifacts  int `json:"evidence_artifacts"`
			} `json:"summary"`
			Targets []struct {
				ClawName       string   `json:"claw_name"`
				RecoveryStatus string   `json:"recovery_status"`
				Warnings       []string `json:"warnings"`
			} `json:"targets"`
		} `json:"clawhost_recovery_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center recovery response: %v", err)
	}
	if decoded.ClawHostRecovery.Status != "active" || decoded.ClawHostRecovery.Summary.Targets != 2 || decoded.ClawHostRecovery.Summary.RecoverableTargets != 1 || decoded.ClawHostRecovery.Summary.DegradedTargets != 1 || decoded.ClawHostRecovery.Summary.IsolatedTargets != 1 || decoded.ClawHostRecovery.Summary.TakeoverRequired != 1 || decoded.ClawHostRecovery.Summary.EvidenceArtifacts != 2 {
		t.Fatalf("unexpected ClawHost recovery control center surface: %+v", decoded.ClawHostRecovery)
	}
	if decoded.ClawHostRecovery.Filters["team"] != "" || decoded.ClawHostRecovery.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost recovery filters, got %+v", decoded.ClawHostRecovery.Filters)
	}
	if len(decoded.ClawHostRecovery.Targets) != 2 || decoded.ClawHostRecovery.Targets[0].RecoveryStatus != "degraded" || decoded.ClawHostRecovery.Targets[0].ClawName != "support-east" {
		t.Fatalf("expected degraded ClawHost recovery target first, got %+v", decoded.ClawHostRecovery.Targets)
	}
}

func TestV2ControlCenterIncludesClawHostWorkflowSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010415, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-workflow-center-1",
			Source:   "clawhost",
			Title:    "review channels",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"claw_id":                   "claw-a",
				"claw_name":                 "sales-west",
				"skill_count":               "3",
				"agent_skill_count":         "4",
				"channel_types":             "telegram,discord,whatsapp",
				"whatsapp_pairing_status":   "waiting",
				"admin_credentials_exposed": "true",
				"admin_surface_path":        "/credentials",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-workflow-center-2",
			Source:   "clawhost",
			Title:    "review skills",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_id":                 "claw-b",
				"claw_name":               "support-east",
				"skill_count":             "2",
				"agent_skill_count":       "2",
				"channel_types":           "telegram",
				"whatsapp_pairing_status": "paired",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue control center workflow task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostWorkflow struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				WorkflowItems     int `json:"workflow_items"`
				PairingApprovals  int `json:"pairing_approvals"`
				CredentialReviews int `json:"credential_reviews"`
				TakeoverRequired  int `json:"takeover_required"`
			} `json:"summary"`
			ReviewQueue []struct {
				ClawName           string   `json:"claw_name"`
				WhatsAppPairing    string   `json:"whatsapp_pairing"`
				CredentialsExposed bool     `json:"credentials_exposed"`
				TakeoverRequired   bool     `json:"takeover_required"`
				Channels           []string `json:"channels"`
			} `json:"review_queue"`
		} `json:"clawhost_workflow_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center workflow response: %v", err)
	}
	if decoded.ClawHostWorkflow.Status != "active" || decoded.ClawHostWorkflow.Summary.WorkflowItems != 2 || decoded.ClawHostWorkflow.Summary.PairingApprovals != 1 || decoded.ClawHostWorkflow.Summary.CredentialReviews != 1 || decoded.ClawHostWorkflow.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected ClawHost workflow control center surface: %+v", decoded.ClawHostWorkflow)
	}
	if decoded.ClawHostWorkflow.Filters["team"] != "" || decoded.ClawHostWorkflow.Filters["project"] != "" || decoded.ClawHostWorkflow.Filters["actor"] != "workflow-operator" {
		t.Fatalf("expected unscoped ClawHost workflow filters, got %+v", decoded.ClawHostWorkflow.Filters)
	}
	if len(decoded.ClawHostWorkflow.ReviewQueue) != 2 || !decoded.ClawHostWorkflow.ReviewQueue[0].TakeoverRequired || decoded.ClawHostWorkflow.ReviewQueue[0].ClawName != "sales-west" {
		t.Fatalf("expected takeover-required ClawHost workflow item first, got %+v", decoded.ClawHostWorkflow.ReviewQueue)
	}
}

func TestV2ControlCenterIncludesClawHostReadinessSurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010418, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-ready-center-1",
			Source:   "clawhost",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":       "clawhost",
				"claw_id":             "claw-a",
				"claw_name":           "sales-west",
				"domain":              "sales-west.clawhost.cloud",
				"proxy_mode":          "http_ws_gateway",
				"gateway_port":        "18789",
				"reachable":           "true",
				"admin_ui_enabled":    "true",
				"websocket_reachable": "true",
				"subdomain_ready":     "true",
				"version_status":      "current",
				"version_current":     "0.0.31",
				"version_latest":      "0.0.31",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-ready-center-2",
			Source:   "clawhost",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":       "clawhost",
				"claw_id":             "claw-b",
				"claw_name":           "support-east",
				"domain":              "support-east.clawhost.cloud",
				"proxy_mode":          "http_ws_gateway",
				"gateway_port":        "18789",
				"reachable":           "false",
				"admin_ui_enabled":    "true",
				"websocket_reachable": "false",
				"subdomain_ready":     "false",
				"version_status":      "upgrade_available",
				"version_current":     "0.0.30",
				"version_latest":      "0.0.31",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue control center readiness task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostReadiness struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets                 int `json:"targets"`
				ReadyTargets            int `json:"ready_targets"`
				DegradedTargets         int `json:"degraded_targets"`
				AdminReadyTargets       int `json:"admin_ready_targets"`
				WebSocketReadyTargets   int `json:"websocket_ready_targets"`
				SubdomainReadyTargets   int `json:"subdomain_ready_targets"`
				UpgradeAvailableTargets int `json:"upgrade_available_targets"`
			} `json:"summary"`
			Targets []struct {
				ClawName     string   `json:"claw_name"`
				ReviewStatus string   `json:"review_status"`
				Warnings     []string `json:"warnings"`
			} `json:"targets"`
		} `json:"clawhost_readiness_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center readiness response: %v", err)
	}
	if decoded.ClawHostReadiness.Status != "active" || decoded.ClawHostReadiness.Summary.Targets != 2 || decoded.ClawHostReadiness.Summary.ReadyTargets != 1 || decoded.ClawHostReadiness.Summary.DegradedTargets != 1 || decoded.ClawHostReadiness.Summary.AdminReadyTargets != 1 || decoded.ClawHostReadiness.Summary.WebSocketReadyTargets != 1 || decoded.ClawHostReadiness.Summary.SubdomainReadyTargets != 1 || decoded.ClawHostReadiness.Summary.UpgradeAvailableTargets != 1 {
		t.Fatalf("unexpected ClawHost readiness control center surface: %+v", decoded.ClawHostReadiness)
	}
	if decoded.ClawHostReadiness.Filters["team"] != "" || decoded.ClawHostReadiness.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost readiness filters, got %+v", decoded.ClawHostReadiness.Filters)
	}
	if len(decoded.ClawHostReadiness.Targets) != 2 || decoded.ClawHostReadiness.Targets[0].ReviewStatus != "degraded" || decoded.ClawHostReadiness.Targets[0].ClawName != "support-east" {
		t.Fatalf("expected degraded ClawHost readiness target first, got %+v", decoded.ClawHostReadiness.Targets)
	}
}

func TestV2ControlCenterIncludesClawHostPolicySurface(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010422, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-policy-center-1",
			Source:   "clawhost",
			Title:    "review tenant provider defaults",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":             "sales-app",
				"clawhost_default_provider":   "openai",
				"clawhost_provider_mode":      "app_default",
				"clawhost_provider_allowlist": "anthropic,openai",
				"clawhost_review_required":    "true",
				"clawhost_drift_status":       "out_of_policy",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-policy-center-2",
			Source:   "clawhost",
			Title:    "review support tenant override",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"clawhost_app_id":             "support-app",
				"clawhost_default_provider":   "anthropic",
				"clawhost_provider_mode":      "tenant_override",
				"clawhost_provider_allowlist": "openai,google",
				"clawhost_takeover_required":  "true",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue control center policy task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHostPolicy struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies      int `json:"active_policies"`
				ActiveTenants       int `json:"active_tenants"`
				ActiveApps          int `json:"active_apps"`
				ReviewRequired      int `json:"review_required"`
				TakeoverRequired    int `json:"takeover_required"`
				OutOfPolicyDefaults int `json:"out_of_policy_defaults"`
			} `json:"summary"`
			ObservedProviders []string `json:"observed_providers"`
			ReviewQueue       []struct {
				TaskID      string `json:"task_id"`
				DriftStatus string `json:"drift_status"`
			} `json:"review_queue"`
		} `json:"clawhost_policy_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center policy response: %v", err)
	}
	if decoded.ClawHostPolicy.Status != "active" || decoded.ClawHostPolicy.Summary.ActivePolicies != 2 || decoded.ClawHostPolicy.Summary.ActiveTenants != 2 || decoded.ClawHostPolicy.Summary.ActiveApps != 2 || decoded.ClawHostPolicy.Summary.ReviewRequired != 2 || decoded.ClawHostPolicy.Summary.TakeoverRequired != 1 || decoded.ClawHostPolicy.Summary.OutOfPolicyDefaults != 1 {
		t.Fatalf("unexpected ClawHost policy control center surface: %+v", decoded.ClawHostPolicy)
	}
	if decoded.ClawHostPolicy.Filters["team"] != "" || decoded.ClawHostPolicy.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost policy filters, got %+v", decoded.ClawHostPolicy.Filters)
	}
	if len(decoded.ClawHostPolicy.ReviewQueue) != 2 || decoded.ClawHostPolicy.ReviewQueue[0].TaskID != "clawhost-policy-center-2" || decoded.ClawHostPolicy.ReviewQueue[0].DriftStatus != "out_of_policy" || decoded.ClawHostPolicy.ReviewQueue[1].TaskID != "clawhost-policy-center-1" || decoded.ClawHostPolicy.ReviewQueue[1].DriftStatus != "review_required" {
		t.Fatalf("expected prioritized ClawHost policy review queue, got %+v", decoded.ClawHostPolicy.ReviewQueue)
	}
	for _, want := range []string{"anthropic", "openai"} {
		if !containsString(decoded.ClawHostPolicy.ObservedProviders, want) {
			t.Fatalf("expected ClawHost observed provider %q, got %+v", want, decoded.ClawHostPolicy.ObservedProviders)
		}
	}
}

func TestV2ControlCenterIncludesCompleteClawHostSurfaceBundle(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010430, 0)
	task := domain.Task{
		ID:       "clawhost-bundle-1",
		Source:   "clawhost",
		Title:    "bundle verification target",
		TenantID: "tenant-a",
		State:    domain.TaskQueued,
		Metadata: map[string]string{
			"control_plane":               "clawhost",
			"inventory_kind":              "claw",
			"claw_id":                     "claw-sales-west",
			"claw_name":                   "sales-west",
			"provider":                    "openai",
			"provider_status":             "running",
			"clawhost_app_id":             "sales-app",
			"clawhost_default_provider":   "openai",
			"clawhost_provider_mode":      "app_default",
			"clawhost_provider_allowlist": "anthropic,openai",
			"skill_count":                 "3",
			"agent_skill_count":           "4",
			"channel_types":               "discord,telegram,whatsapp",
			"whatsapp_pairing_status":     "waiting",
			"admin_credentials_exposed":   "true",
			"admin_surface_path":          "/credentials",
			"domain":                      "sales-west.clawhost.cloud",
			"proxy_mode":                  "http_ws_gateway",
			"gateway_port":                "18789",
			"reachable":                   "true",
			"admin_ui_enabled":            "true",
			"websocket_reachable":         "true",
			"subdomain_ready":             "true",
			"version_status":              "current",
			"version_current":             "0.0.31",
			"version_latest":              "0.0.31",
			"clawhost_update_available":   "true",
			"clawhost_takeover_required":  "true",
			"clawhost_lifecycle_actions":  "start,restart,upgrade",
			"clawhost_pod_isolation":      "true",
			"clawhost_service_isolation":  "true",
			"clawhost_takeover_triggers":  "proxy health regresses,session restore fails",
			"clawhost_recovery_evidence":  "GET /status,/v2/reports/distributed",
		},
		UpdatedAt: now,
	}
	if err := taskQueue.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue control center bundle task %s: %v", task.ID, err)
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Policy struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies int `json:"active_policies"`
			} `json:"summary"`
		} `json:"clawhost_policy_surface"`
		Workflow struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				WorkflowItems int `json:"workflow_items"`
			} `json:"summary"`
		} `json:"clawhost_workflow_surface"`
		Rollout struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePlans int `json:"active_plans"`
			} `json:"summary"`
		} `json:"clawhost_rollout_surface"`
		Readiness struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
		} `json:"clawhost_readiness_surface"`
		Recovery struct {
			Status  string            `json:"status"`
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
		} `json:"clawhost_recovery_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center bundle response: %v", err)
	}
	if decoded.Policy.Status != "active" || decoded.Policy.Summary.ActivePolicies != 1 {
		t.Fatalf("expected active ClawHost policy surface in bundle, got %+v", decoded.Policy)
	}
	if decoded.Policy.Filters["team"] != "" || decoded.Policy.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost policy filters in bundle, got %+v", decoded.Policy.Filters)
	}
	if decoded.Workflow.Status != "active" || decoded.Workflow.Summary.WorkflowItems != 1 {
		t.Fatalf("expected active ClawHost workflow surface in bundle, got %+v", decoded.Workflow)
	}
	if decoded.Workflow.Filters["team"] != "" || decoded.Workflow.Filters["project"] != "" || decoded.Workflow.Filters["actor"] != "workflow-operator" {
		t.Fatalf("expected unscoped ClawHost workflow filters in bundle, got %+v", decoded.Workflow.Filters)
	}
	if decoded.Rollout.Status != "active" || decoded.Rollout.Summary.ActivePlans != 1 {
		t.Fatalf("expected active ClawHost rollout surface in bundle, got %+v", decoded.Rollout)
	}
	if decoded.Rollout.Filters["team"] != "" || decoded.Rollout.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost rollout filters in bundle, got %+v", decoded.Rollout.Filters)
	}
	if decoded.Readiness.Status != "active" || decoded.Readiness.Summary.Targets != 1 {
		t.Fatalf("expected active ClawHost readiness surface in bundle, got %+v", decoded.Readiness)
	}
	if decoded.Readiness.Filters["team"] != "" || decoded.Readiness.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost readiness filters in bundle, got %+v", decoded.Readiness.Filters)
	}
	if decoded.Recovery.Status != "active" || decoded.Recovery.Summary.Targets != 1 {
		t.Fatalf("expected active ClawHost recovery surface in bundle, got %+v", decoded.Recovery)
	}
	if decoded.Recovery.Filters["team"] != "" || decoded.Recovery.Filters["project"] != "" {
		t.Fatalf("expected unscoped ClawHost recovery filters in bundle, got %+v", decoded.Recovery.Filters)
	}
}

func TestV2ControlCenterScopesClawHostSurfaceBundleByFilters(t *testing.T) {
	recorder := observability.NewRecorder()
	taskQueue := queue.NewMemoryQueue()
	now := time.Unix(1700010440, 0)
	for _, task := range []domain.Task{
		{
			ID:       "clawhost-filtered-1",
			Source:   "clawhost",
			Title:    "platform sales bundle target",
			TenantID: "tenant-a",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":               "clawhost",
				"team":                        "platform",
				"project":                     "sales",
				"inventory_kind":              "claw",
				"claw_id":                     "claw-sales-west",
				"claw_name":                   "sales-west",
				"provider":                    "openai",
				"provider_status":             "running",
				"clawhost_app_id":             "sales-app",
				"clawhost_default_provider":   "openai",
				"clawhost_provider_mode":      "app_default",
				"clawhost_provider_allowlist": "anthropic,openai",
				"skill_count":                 "3",
				"agent_skill_count":           "4",
				"channel_types":               "discord,telegram",
				"whatsapp_pairing_status":     "waiting",
				"admin_credentials_exposed":   "true",
				"admin_surface_path":          "/credentials",
				"domain":                      "sales-west.clawhost.cloud",
				"proxy_mode":                  "http_ws_gateway",
				"gateway_port":                "18789",
				"reachable":                   "true",
				"admin_ui_enabled":            "true",
				"websocket_reachable":         "true",
				"subdomain_ready":             "true",
				"version_status":              "current",
				"version_current":             "0.0.31",
				"version_latest":              "0.0.31",
				"clawhost_update_available":   "true",
				"clawhost_takeover_required":  "true",
				"clawhost_lifecycle_actions":  "start,restart,upgrade",
				"clawhost_pod_isolation":      "true",
				"clawhost_service_isolation":  "true",
				"clawhost_takeover_triggers":  "proxy health regresses,session restore fails",
				"clawhost_recovery_evidence":  "GET /status,/v2/reports/distributed",
			},
			UpdatedAt: now,
		},
		{
			ID:       "clawhost-filtered-2",
			Source:   "clawhost",
			Title:    "support care bundle target",
			TenantID: "tenant-b",
			State:    domain.TaskQueued,
			Metadata: map[string]string{
				"control_plane":               "clawhost",
				"team":                        "support",
				"project":                     "care",
				"inventory_kind":              "claw",
				"claw_id":                     "claw-support-east",
				"claw_name":                   "support-east",
				"provider":                    "anthropic",
				"provider_status":             "running",
				"clawhost_app_id":             "support-app",
				"clawhost_default_provider":   "anthropic",
				"clawhost_provider_mode":      "tenant_override",
				"clawhost_provider_allowlist": "openai,google",
				"skill_count":                 "5",
				"agent_skill_count":           "6",
				"channel_types":               "slack,whatsapp",
				"whatsapp_pairing_status":     "approved",
				"admin_credentials_exposed":   "false",
				"domain":                      "support-east.clawhost.cloud",
				"proxy_mode":                  "http_ws_gateway",
				"gateway_port":                "18790",
				"reachable":                   "true",
				"admin_ui_enabled":            "true",
				"websocket_reachable":         "true",
				"subdomain_ready":             "true",
				"version_status":              "current",
				"version_current":             "0.0.31",
				"version_latest":              "0.0.31",
				"clawhost_update_available":   "false",
				"clawhost_takeover_required":  "false",
				"clawhost_lifecycle_actions":  "start,restart",
				"clawhost_pod_isolation":      "true",
				"clawhost_service_isolation":  "true",
				"clawhost_takeover_triggers":  "session restore fails",
				"clawhost_recovery_evidence":  "GET /status",
			},
			UpdatedAt: now.Add(time.Second),
		},
	} {
		if err := taskQueue.Enqueue(context.Background(), task); err != nil {
			t.Fatalf("enqueue filtered control center task %s: %v", task.ID, err)
		}
	}
	server := &Server{Recorder: recorder, Queue: taskQueue, Bus: events.NewBus(), Control: control.New(), Now: func() time.Time { return now }}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?team=platform&project=sales", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected scoped control center 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Policy struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePolicies int `json:"active_policies"`
			} `json:"summary"`
			ObservedProviders []string `json:"observed_providers"`
			ReviewQueue       []struct {
				TaskID string `json:"task_id"`
			} `json:"review_queue"`
		} `json:"clawhost_policy_surface"`
		Workflow struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				WorkflowItems int `json:"workflow_items"`
			} `json:"summary"`
			ReviewQueue []struct {
				TaskID string `json:"task_id"`
			} `json:"review_queue"`
		} `json:"clawhost_workflow_surface"`
		Rollout struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				ActivePlans int `json:"active_plans"`
			} `json:"summary"`
			Plans []struct {
				Waves []struct {
					Targets []struct {
						TaskID string `json:"task_id"`
					} `json:"targets"`
				} `json:"waves"`
			} `json:"plans"`
		} `json:"clawhost_rollout_surface"`
		Readiness struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
			Targets []struct {
				TaskID string `json:"task_id"`
			} `json:"targets"`
		} `json:"clawhost_readiness_surface"`
		Recovery struct {
			Filters map[string]string `json:"filters"`
			Summary struct {
				Targets int `json:"targets"`
			} `json:"summary"`
			Targets []struct {
				TaskID string `json:"task_id"`
			} `json:"targets"`
		} `json:"clawhost_recovery_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode scoped control center response: %v", err)
	}
	if decoded.Policy.Summary.ActivePolicies != 1 || len(decoded.Policy.ReviewQueue) != 1 || decoded.Policy.ReviewQueue[0].TaskID != "clawhost-filtered-1" {
		t.Fatalf("expected scoped policy surface, got %+v", decoded.Policy)
	}
	if decoded.Policy.Filters["team"] != "platform" || decoded.Policy.Filters["project"] != "sales" {
		t.Fatalf("expected scoped policy filters, got %+v", decoded.Policy.Filters)
	}
	if containsString(decoded.Policy.ObservedProviders, "anthropic") || !containsString(decoded.Policy.ObservedProviders, "openai") {
		t.Fatalf("expected scoped policy providers, got %+v", decoded.Policy.ObservedProviders)
	}
	if decoded.Workflow.Summary.WorkflowItems != 1 || len(decoded.Workflow.ReviewQueue) != 1 || decoded.Workflow.ReviewQueue[0].TaskID != "clawhost-filtered-1" {
		t.Fatalf("expected scoped workflow surface, got %+v", decoded.Workflow)
	}
	if decoded.Workflow.Filters["team"] != "platform" || decoded.Workflow.Filters["project"] != "sales" || decoded.Workflow.Filters["actor"] != "workflow-operator" {
		t.Fatalf("expected scoped workflow filters, got %+v", decoded.Workflow.Filters)
	}
	if decoded.Rollout.Summary.ActivePlans != 1 || len(decoded.Rollout.Plans) != 1 || len(decoded.Rollout.Plans[0].Waves) != 1 || len(decoded.Rollout.Plans[0].Waves[0].Targets) != 1 || decoded.Rollout.Plans[0].Waves[0].Targets[0].TaskID != "clawhost-filtered-1" {
		t.Fatalf("expected scoped rollout surface, got %+v", decoded.Rollout)
	}
	if decoded.Rollout.Filters["team"] != "platform" || decoded.Rollout.Filters["project"] != "sales" {
		t.Fatalf("expected scoped rollout filters, got %+v", decoded.Rollout.Filters)
	}
	if decoded.Readiness.Summary.Targets != 1 || len(decoded.Readiness.Targets) != 1 || decoded.Readiness.Targets[0].TaskID != "clawhost-filtered-1" {
		t.Fatalf("expected scoped readiness surface, got %+v", decoded.Readiness)
	}
	if decoded.Readiness.Filters["team"] != "platform" || decoded.Readiness.Filters["project"] != "sales" {
		t.Fatalf("expected scoped readiness filters, got %+v", decoded.Readiness.Filters)
	}
	if decoded.Recovery.Summary.Targets != 1 || len(decoded.Recovery.Targets) != 1 || decoded.Recovery.Targets[0].TaskID != "clawhost-filtered-1" {
		t.Fatalf("expected scoped recovery surface, got %+v", decoded.Recovery)
	}
	if decoded.Recovery.Filters["team"] != "platform" || decoded.Recovery.Filters["project"] != "sales" {
		t.Fatalf("expected scoped recovery filters, got %+v", decoded.Recovery.Filters)
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

	runsRequest := httptest.NewRequest(http.MethodGet, "/v2/runs?limit=10", nil)
	runsRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	runsRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	runsRequest.Header.Set("X-BigClaw-Team", "platform")
	runsResponse := httptest.NewRecorder()
	handler.ServeHTTP(runsResponse, runsRequest)
	if runsResponse.Code != http.StatusOK {
		t.Fatalf("expected scoped runs 200, got %d %s", runsResponse.Code, runsResponse.Body.String())
	}
	runsBody := runsResponse.Body.String()
	if !strings.Contains(runsBody, "task-scope-1") || strings.Contains(runsBody, "task-scope-2") {
		t.Fatalf("expected run index to be scoped to platform team, got %s", runsBody)
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

	forbiddenRunsRequest := httptest.NewRequest(http.MethodGet, "/v2/runs?team=growth&limit=10", nil)
	forbiddenRunsRequest.Header.Set("X-BigClaw-Role", "eng_lead")
	forbiddenRunsRequest.Header.Set("X-BigClaw-Actor", "lead-2")
	forbiddenRunsRequest.Header.Set("X-BigClaw-Team", "platform")
	forbiddenRunsResponse := httptest.NewRecorder()
	handler.ServeHTTP(forbiddenRunsResponse, forbiddenRunsRequest)
	if forbiddenRunsResponse.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden run index for mismatched team, got %d %s", forbiddenRunsResponse.Code, forbiddenRunsResponse.Body.String())
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

func TestV2ControlCenterPolicyEndpointShowsRemoteFairnessHealth(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "scheduler-policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"default_executor":"ray","fairness":{"window_seconds":30,"max_recent_decisions_per_tenant":1}}`), 0o644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
	store, err := scheduler.NewPolicyStore(policyPath)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	serviceStore, err := scheduler.NewFairnessStore("")
	if err != nil {
		t.Fatalf("new service fairness store: %v", err)
	}
	service := httptest.NewServer(scheduler.NewFairnessServiceHandler(serviceStore))
	defer service.Close()
	remoteFairness, err := scheduler.NewFairnessStoreWithRemote("", service.URL, "")
	if err != nil {
		t.Fatalf("new remote fairness store: %v", err)
	}
	schedulerRuntime := scheduler.NewWithStores(store, remoteFairness)
	schedulerRuntime.Decide(domain.Task{ID: "remote-fair-1", TenantID: "tenant-a", Priority: 3}, scheduler.QuotaSnapshot{})
	schedulerRuntime.Decide(domain.Task{ID: "remote-fair-2", TenantID: "tenant-b", Priority: 3}, scheduler.QuotaSnapshot{})
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), SchedulerPolicy: store, SchedulerRuntime: schedulerRuntime, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center/policy", nil)
	request.Header.Set("X-BigClaw-Role", "platform_admin")
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected scheduler policy 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Fairness struct {
			Backend       string `json:"backend"`
			Healthy       bool   `json:"healthy"`
			Endpoint      string `json:"endpoint"`
			ActiveTenants int    `json:"active_tenants"`
		} `json:"fairness"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode remote fairness policy response: %v", err)
	}
	if decoded.Fairness.Backend != "http" || !decoded.Fairness.Healthy || decoded.Fairness.Endpoint != service.URL || decoded.Fairness.ActiveTenants != 2 {
		t.Fatalf("unexpected remote fairness metadata: %+v", decoded.Fairness)
	}
	body := response.Body.String()
	for _, want := range []string{"tenant-a", "tenant-b"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in remote fairness policy payload, got %s", want, body)
		}
	}
}

func TestEventsEndpointUsesDurableEventLogAcrossInstances(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = log1.Close() }()
	recorder1 := observability.NewRecorder()
	bus1 := events.NewBus()
	bus1.AddSink(events.RecorderSink{Recorder: recorder1})
	bus1.AddSink(log1)
	bus1.Publish(domain.Event{ID: "evt-durable-1", Type: domain.EventTaskQueued, TaskID: "task-durable", TraceID: "trace-durable", Timestamp: time.Now()})

	log2, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-durable&limit=10", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{"evt-durable-1", `"backend":"sqlite"`, `"durable":true`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in durable events response, got %s", want, body)
		}
	}
}

func TestStreamEventsReplayCanUseDurableEventLog(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = log1.Close() }()
	bus1 := events.NewBus()
	bus1.AddSink(log1)
	bus1.Publish(domain.Event{ID: "evt-durable-stream", Type: domain.EventTaskQueued, TaskID: "task-stream", TraceID: "trace-stream", Timestamp: time.Now()})

	log2, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?replay=1&trace_id=trace-stream&limit=10", nil)
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
		if !strings.Contains(line, "evt-durable-stream") {
			t.Fatalf("expected durable replayed event in stream, got %q", line)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for durable replay stream event")
	}
}

func TestEventsEndpointSupportsAfterIDCursor(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	base := time.Now()
	for index, event := range []domain.Event{
		{ID: "evt-durable-1", Type: domain.EventTaskQueued, TaskID: "task-durable", TraceID: "trace-durable", Timestamp: base},
		{ID: "evt-durable-2", Type: domain.EventTaskStarted, TaskID: "task-durable", TraceID: "trace-durable", Timestamp: base.Add(time.Second)},
		{ID: "evt-durable-3", Type: domain.EventTaskCompleted, TaskID: "task-durable", TraceID: "trace-durable", Timestamp: base.Add(2 * time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write durable event %d: %v", index, err)
		}
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}
	log2, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-durable&after_id=evt-durable-1&limit=10", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Events      []domain.Event `json:"events"`
		AfterID     string         `json:"after_id"`
		NextAfterID string         `json:"next_after_id"`
		Backend     string         `json:"backend"`
		Durable     bool           `json:"durable"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode cursor events response: %v", err)
	}
	if len(decoded.Events) != 2 || decoded.Events[0].ID != "evt-durable-2" || decoded.Events[1].ID != "evt-durable-3" {
		t.Fatalf("unexpected cursor events: %+v", decoded.Events)
	}
	if decoded.AfterID != "evt-durable-1" || decoded.NextAfterID != "evt-durable-3" {
		t.Fatalf("unexpected cursor metadata: after=%q next=%q", decoded.AfterID, decoded.NextAfterID)
	}
	if decoded.Backend != "sqlite" || !decoded.Durable {
		t.Fatalf("unexpected durable metadata: %+v", decoded)
	}
}

func TestStreamEventsResumeUsesLastEventID(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-stream-1", Type: domain.EventTaskQueued, TaskID: "task-stream", TraceID: "trace-stream", Timestamp: base},
		{ID: "evt-stream-2", Type: domain.EventTaskStarted, TaskID: "task-stream", TraceID: "trace-stream", Timestamp: base.Add(time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write stream event %s: %v", event.ID, err)
		}
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}
	log2, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?trace_id=trace-stream&limit=10", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Last-Event-ID", "evt-stream-1")
	resultCh := make(chan [3]string, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line1, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		line2, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		resultCh <- [3]string{response.Status, strings.TrimSpace(line1), strings.TrimSpace(line2)}
	}()
	select {
	case result := <-resultCh:
		if result[0] == "ERROR" {
			t.Fatalf("stream request failed: %s", result[1])
		}
		if !strings.Contains(result[1], "evt-stream-2") || strings.Contains(result[1], "evt-stream-1") {
			t.Fatalf("expected replayed cursor event in first SSE line, got %q", result[1])
		}
		if result[2] != "id: evt-stream-2" {
			t.Fatalf("expected SSE id line for resumed event, got %q", result[2])
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for resumed SSE event")
	}
}

func TestStreamEventsReplayLiveHandoffDeduplicatesOverlap(t *testing.T) {
	evt1 := domain.Event{ID: "evt-stream-1", Type: domain.EventTaskQueued, TaskID: "task-stream", TraceID: "trace-stream", Timestamp: time.Now()}
	evt2 := domain.Event{ID: "evt-stream-2", Type: domain.EventTaskStarted, TaskID: "task-stream", TraceID: "trace-stream", Timestamp: time.Now().Add(time.Second)}
	log := &blockingEventLog{
		history:       []domain.Event{evt1, evt2},
		replayStarted: make(chan struct{}, 1),
		release:       make(chan struct{}),
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?trace_id=trace-stream&after_id=evt-before&limit=10", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	type streamResult struct {
		lines []string
		err   string
	}
	resultCh := make(chan streamResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- streamResult{err: err.Error()}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		lines := make([]string, 0)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				resultCh <- streamResult{lines: lines, err: err.Error()}
				return
			}
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}()

	select {
	case <-log.replayStarted:
	case <-ctx.Done():
		t.Fatal("timed out waiting for replay to start")
	}
	server.Bus.Publish(evt2)
	close(log.release)

	select {
	case result := <-resultCh:
		if result.err == "" {
			t.Fatal("expected request to end after context cancellation")
		}
		if len(result.lines) != 4 {
			t.Fatalf("expected exactly two SSE events without duplicates, got %d lines: %+v", len(result.lines), result.lines)
		}
		if !strings.Contains(result.lines[0], "evt-stream-1") || result.lines[1] != "id: evt-stream-1" {
			t.Fatalf("unexpected first replayed event lines: %+v", result.lines)
		}
		if !strings.Contains(result.lines[2], "evt-stream-2") || result.lines[3] != "id: evt-stream-2" {
			t.Fatalf("unexpected second replayed event lines: %+v", result.lines)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for replay/live handoff result")
	}
}

func TestEventsEndpointFiltersByEventType(t *testing.T) {
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	base := time.Now()
	bus.Publish(domain.Event{ID: "evt-filter-1", Type: domain.EventTaskQueued, TaskID: "task-filter", TraceID: "trace-filter", Timestamp: base})
	bus.Publish(domain.Event{ID: "evt-filter-2", Type: domain.EventTaskStarted, TaskID: "task-filter", TraceID: "trace-filter", Timestamp: base.Add(time.Second)})
	bus.Publish(domain.Event{ID: "evt-filter-3", Type: domain.EventTaskCompleted, TaskID: "task-filter", TraceID: "trace-filter", Timestamp: base.Add(2 * time.Second)})
	server := &Server{Recorder: recorder, Queue: queue.NewMemoryQueue(), Bus: bus, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-filter&event_type=task.started&limit=10", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Events     []domain.Event     `json:"events"`
		EventTypes []domain.EventType `json:"event_types"`
		Backend    string             `json:"backend"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode filtered events response: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].ID != "evt-filter-2" || decoded.Events[0].Type != domain.EventTaskStarted {
		t.Fatalf("unexpected filtered events: %+v", decoded.Events)
	}
	if len(decoded.EventTypes) != 1 || decoded.EventTypes[0] != domain.EventTaskStarted || decoded.Backend != "memory" {
		t.Fatalf("unexpected filter metadata: %+v", decoded)
	}
}

func TestStreamEventCheckpointEndpointAndResume(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-check-1", Type: domain.EventTaskQueued, TaskID: "task-check", TraceID: "trace-check", Timestamp: base},
		{ID: "evt-check-2", Type: domain.EventTaskStarted, TaskID: "task-check", TraceID: "trace-check", Timestamp: base.Add(time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write checkpoint event %s: %v", event.ID, err)
		}
	}
	server1 := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log1, Now: time.Now}
	ackBody := bytes.NewBufferString(`{"event_id":"evt-check-1"}`)
	ackRequest := httptest.NewRequest(http.MethodPost, "/stream/events/checkpoints/subscriber-a", ackBody)
	ackResponse := httptest.NewRecorder()
	server1.Handler().ServeHTTP(ackResponse, ackRequest)
	if ackResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint ack 200, got %d %s", ackResponse.Code, ackResponse.Body.String())
	}
	checkpointRequest := httptest.NewRequest(http.MethodGet, "/stream/events/checkpoints/subscriber-a", nil)
	checkpointResponse := httptest.NewRecorder()
	server1.Handler().ServeHTTP(checkpointResponse, checkpointRequest)
	if checkpointResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint get 200, got %d %s", checkpointResponse.Code, checkpointResponse.Body.String())
	}
	var checkpointDecoded struct {
		Checkpoint events.SubscriberCheckpoint `json:"checkpoint"`
	}
	if err := json.Unmarshal(checkpointResponse.Body.Bytes(), &checkpointDecoded); err != nil {
		t.Fatalf("decode checkpoint response: %v", err)
	}
	if checkpointDecoded.Checkpoint.EventID != "evt-check-1" || checkpointDecoded.Checkpoint.SubscriberID != "subscriber-a" {
		t.Fatalf("unexpected checkpoint payload: %+v", checkpointDecoded.Checkpoint)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}

	log2, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server2 := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: time.Now}
	ts := httptest.NewServer(server2.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream/events?subscriber_id=subscriber-a&trace_id=trace-check&event_type=task.started&limit=10", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resultCh := make(chan [3]string, 1)
	go func() {
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line1, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		line2, err := reader.ReadString('\n')
		if err != nil {
			resultCh <- [3]string{"ERROR", err.Error(), ""}
			return
		}
		resultCh <- [3]string{response.Status, strings.TrimSpace(line1), strings.TrimSpace(line2)}
	}()
	select {
	case result := <-resultCh:
		if result[0] == "ERROR" {
			t.Fatalf("stream request failed: %s", result[1])
		}
		if !strings.Contains(result[1], "evt-check-2") || strings.Contains(result[1], "evt-check-1") {
			t.Fatalf("expected resumed filtered event in first SSE line, got %q", result[1])
		}
		if result[2] != "id: evt-check-2" {
			t.Fatalf("expected SSE id line for resumed event, got %q", result[2])
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for checkpoint-resumed SSE event")
	}
}

func TestEventsEndpointUsesRemoteEventLogBackend(t *testing.T) {
	store, err := events.NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	serviceServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: store, Now: time.Now}
	serviceTS := httptest.NewServer(serviceServer.Handler())
	defer serviceTS.Close()
	remoteLog, err := events.NewHTTPEventLog(serviceTS.URL+"/internal/events/log", "")
	if err != nil {
		t.Fatalf("new remote event log: %v", err)
	}
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-http-1", Type: domain.EventTaskQueued, TaskID: "task-http", TraceID: "trace-http", Timestamp: base},
		{ID: "evt-http-2", Type: domain.EventTaskStarted, TaskID: "task-http", TraceID: "trace-http", Timestamp: base.Add(time.Second)},
	} {
		if err := remoteLog.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote-backed event %s: %v", event.ID, err)
		}
	}
	apiServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: remoteLog, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-http&after_id=evt-http-1&limit=10", nil)
	apiServer.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected remote events 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Events  []domain.Event `json:"events"`
		Backend string         `json:"backend"`
		Durable bool           `json:"durable"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode remote backend events response: %v", err)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].ID != "evt-http-2" {
		t.Fatalf("unexpected remote backend events: %+v", decoded.Events)
	}
	if decoded.Backend != "http" || !decoded.Durable {
		t.Fatalf("unexpected remote backend metadata: %+v", decoded)
	}
}

func TestEventsEndpointIncludesRetentionWatermarkForRemoteEventLog(t *testing.T) {
	store, err := events.NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	serviceServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: store, Now: time.Now}
	serviceTS := httptest.NewServer(serviceServer.Handler())
	defer serviceTS.Close()
	remoteLog, err := events.NewHTTPEventLog(serviceTS.URL+"/internal/events/log", "")
	if err != nil {
		t.Fatalf("new remote event log: %v", err)
	}
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-http-watermark-1", Type: domain.EventTaskQueued, TaskID: "task-http-watermark", TraceID: "trace-http-watermark", Timestamp: base},
		{ID: "evt-http-watermark-2", Type: domain.EventTaskStarted, TaskID: "task-http-watermark", TraceID: "trace-http-watermark", Timestamp: base.Add(time.Second)},
	} {
		if err := remoteLog.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote-backed event %s: %v", event.ID, err)
		}
	}
	apiServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: remoteLog, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-http-watermark&limit=10", nil)
	apiServer.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected remote events 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		RetentionWatermark events.RetentionWatermark `json:"retention_watermark"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode remote backend events response: %v", err)
	}
	if decoded.RetentionWatermark.Backend != "sqlite" || decoded.RetentionWatermark.EventCount != 2 || decoded.RetentionWatermark.OldestEventID != "evt-http-watermark-1" || decoded.RetentionWatermark.NewestEventID != "evt-http-watermark-2" {
		t.Fatalf("unexpected retention watermark in events response: %+v", decoded.RetentionWatermark)
	}
}

func TestDebugStatusIncludesRetentionWatermark(t *testing.T) {
	store, err := events.NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-debug-watermark-1", Type: domain.EventTaskQueued, TaskID: "task-debug-watermark", TraceID: "trace-debug-watermark", Timestamp: base},
		{ID: "evt-debug-watermark-2", Type: domain.EventTaskStarted, TaskID: "task-debug-watermark", TraceID: "trace-debug-watermark", Timestamp: base.Add(time.Second)},
	} {
		if err := store.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: store, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "retention_watermark") || !strings.Contains(response.Body.String(), "evt-debug-watermark-1") || !strings.Contains(response.Body.String(), "evt-debug-watermark-2") {
		t.Fatalf("expected retention watermark in debug payload, got %s", response.Body.String())
	}
}

func TestDebugStatusIncludesCheckpointResetSummary(t *testing.T) {
	store, err := events.NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-debug-reset-1", Type: domain.EventTaskQueued, TaskID: "task-debug-reset", TraceID: "trace-debug-reset", Timestamp: base},
		{ID: "evt-debug-reset-2", Type: domain.EventTaskStarted, TaskID: "task-debug-reset", TraceID: "trace-debug-reset", Timestamp: base.Add(time.Second)},
	} {
		if err := store.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	if _, err := store.Acknowledge("subscriber-debug-reset", "evt-debug-reset-1", base.Add(2*time.Second)); err != nil {
		t.Fatalf("ack debug reset checkpoint: %v", err)
	}
	if err := store.ResetCheckpoint("subscriber-debug-reset"); err != nil {
		t.Fatalf("reset debug checkpoint: %v", err)
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: store, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "checkpoint_resets") || !strings.Contains(body, "subscriber-debug-reset") || !strings.Contains(body, "operator_reset") {
		t.Fatalf("expected checkpoint reset summary in debug payload, got %s", body)
	}
}

func TestDebugStatusDistinguishesBrokerStubEventLog(t *testing.T) {
	store := events.NewBrokerStubEventLog()
	if err := store.Write(context.Background(), domain.Event{
		ID:        "evt-broker-stub-debug-1",
		Type:      domain.EventTaskQueued,
		TaskID:    "task-broker-stub-debug",
		TraceID:   "trace-broker-stub-debug",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("write broker stub event: %v", err)
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: store, Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d", response.Code)
	}
	body := response.Body.String()
	for _, want := range []string{"broker_stub", "process_local_stub", "append_only_stub", "process_memory_stub"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in broker stub debug payload, got %s", want, body)
		}
	}
}

func TestStreamEventCheckpointExpiredDiagnosticsAndReset(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log.db")
	base := time.Unix(1_700_000_000, 0).UTC()
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	for _, event := range []domain.Event{
		{ID: "evt-expired-1", Type: domain.EventTaskQueued, TaskID: "task-expired", TraceID: "trace-expired", Timestamp: base},
		{ID: "evt-expired-2", Type: domain.EventTaskStarted, TaskID: "task-expired", TraceID: "trace-expired", Timestamp: base.Add(3 * time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write checkpoint event %s: %v", event.ID, err)
		}
	}
	if _, err := log1.Acknowledge("subscriber-expired", "evt-expired-1", base.Add(time.Second)); err != nil {
		t.Fatalf("ack expired checkpoint: %v", err)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}

	log2, err := events.NewSQLiteEventLogWithOptions(logPath, events.SQLiteEventLogOptions{
		Retention: 2 * time.Second,
		Now:       func() time.Time { return base.Add(4 * time.Second) },
	})
	if err != nil {
		t.Fatalf("reopen sqlite event log with retention: %v", err)
	}
	defer func() { _ = log2.Close() }()
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: func() time.Time { return base.Add(4 * time.Second) }}

	checkpointResponse := httptest.NewRecorder()
	checkpointRequest := httptest.NewRequest(http.MethodGet, "/stream/events/checkpoints/subscriber-expired", nil)
	server.Handler().ServeHTTP(checkpointResponse, checkpointRequest)
	if checkpointResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint diagnostics 200, got %d %s", checkpointResponse.Code, checkpointResponse.Body.String())
	}
	if !strings.Contains(checkpointResponse.Body.String(), "\"status\":\"expired\"") || !strings.Contains(checkpointResponse.Body.String(), "checkpoint_before_retention_boundary") || !strings.Contains(checkpointResponse.Body.String(), "evt-expired-1") {
		t.Fatalf("expected expired checkpoint diagnostics, got %s", checkpointResponse.Body.String())
	}

	eventsResponse := httptest.NewRecorder()
	eventsRequest := httptest.NewRequest(http.MethodGet, "/events?subscriber_id=subscriber-expired&trace_id=trace-expired&limit=10", nil)
	server.Handler().ServeHTTP(eventsResponse, eventsRequest)
	if eventsResponse.Code != http.StatusConflict {
		t.Fatalf("expected expired checkpoint conflict, got %d %s", eventsResponse.Code, eventsResponse.Body.String())
	}
	if !strings.Contains(eventsResponse.Body.String(), "checkpoint_expired") || !strings.Contains(eventsResponse.Body.String(), "DELETE /stream/events/checkpoints/{subscriber_id}") {
		t.Fatalf("expected checkpoint expired payload, got %s", eventsResponse.Body.String())
	}

	resetResponse := httptest.NewRecorder()
	resetRequest := httptest.NewRequest(http.MethodDelete, "/stream/events/checkpoints/subscriber-expired", nil)
	server.Handler().ServeHTTP(resetResponse, resetRequest)
	if resetResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint reset 200, got %d %s", resetResponse.Code, resetResponse.Body.String())
	}
	if !strings.Contains(resetResponse.Body.String(), "reset_audit") || !strings.Contains(resetResponse.Body.String(), "evt-expired-1") {
		t.Fatalf("expected reset audit payload, got %s", resetResponse.Body.String())
	}

	historyResponse := httptest.NewRecorder()
	historyRequest := httptest.NewRequest(http.MethodGet, "/stream/events/checkpoints/subscriber-expired/history?limit=10", nil)
	server.Handler().ServeHTTP(historyResponse, historyRequest)
	if historyResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint history 200, got %d %s", historyResponse.Code, historyResponse.Body.String())
	}
	if !strings.Contains(historyResponse.Body.String(), "operator_reset") || !strings.Contains(historyResponse.Body.String(), "trimmed_through_event_id") || !strings.Contains(historyResponse.Body.String(), "evt-expired-1") {
		t.Fatalf("expected checkpoint reset history payload, got %s", historyResponse.Body.String())
	}

	recoveredResponse := httptest.NewRecorder()
	recoveredRequest := httptest.NewRequest(http.MethodGet, "/events?subscriber_id=subscriber-expired&trace_id=trace-expired&limit=10", nil)
	server.Handler().ServeHTTP(recoveredResponse, recoveredRequest)
	if recoveredResponse.Code != http.StatusOK {
		t.Fatalf("expected events after reset 200, got %d %s", recoveredResponse.Code, recoveredResponse.Body.String())
	}
	var recovered struct {
		Events []domain.Event `json:"events"`
	}
	if err := json.Unmarshal(recoveredResponse.Body.Bytes(), &recovered); err != nil {
		t.Fatalf("decode recovered events: %v", err)
	}
	if len(recovered.Events) != 1 || recovered.Events[0].ID != "evt-expired-2" {
		t.Fatalf("expected replay from earliest retained event after reset, got %+v", recovered.Events)
	}

	reackResponse := httptest.NewRecorder()
	reackRequest := httptest.NewRequest(http.MethodPost, "/stream/events/checkpoints/subscriber-expired", strings.NewReader(`{"event_id":"evt-expired-2"}`))
	server.Handler().ServeHTTP(reackResponse, reackRequest)
	if reackResponse.Code != http.StatusOK {
		t.Fatalf("expected checkpoint re-ack 200, got %d %s", reackResponse.Code, reackResponse.Body.String())
	}

	historyAfterResume := httptest.NewRecorder()
	historyAfterResumeRequest := httptest.NewRequest(http.MethodGet, "/stream/events/checkpoints/subscriber-expired/history?limit=10", nil)
	server.Handler().ServeHTTP(historyAfterResume, historyAfterResumeRequest)
	if historyAfterResume.Code != http.StatusOK {
		t.Fatalf("expected checkpoint history after resume 200, got %d %s", historyAfterResume.Code, historyAfterResume.Body.String())
	}
	if !strings.Contains(historyAfterResume.Body.String(), "evt-expired-1") || !strings.Contains(historyAfterResume.Body.String(), "operator_reset") {
		t.Fatalf("expected checkpoint reset history to remain visible, got %s", historyAfterResume.Body.String())
	}
}

func TestDebugStatusIncludesRetentionExpirySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		RetentionExpiry struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Track      string `json:"track"`
			Summary    struct {
				BackendCount              int `json:"backend_count"`
				RuntimeVisibleBackends    int `json:"runtime_visible_backends"`
				PersistedBoundaryBackends int `json:"persisted_boundary_backends"`
				FailClosedExpiryBackends  int `json:"fail_closed_expiry_backends"`
				ContractOnlyBackends      int `json:"contract_only_backends"`
			} `json:"summary"`
			Backends []struct {
				Backend                 string `json:"backend"`
				RuntimeReadiness        string `json:"runtime_readiness"`
				RetainedBoundaryVisible bool   `json:"retained_boundary_visible"`
				PersistedBoundaries     bool   `json:"persisted_boundaries"`
				FailClosedExpiry        bool   `json:"fail_closed_expiry"`
			} `json:"backends"`
			PolicySplit []string `json:"policy_split"`
		} `json:"retention_expiry_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode retention expiry payload: %v", err)
	}
	if decoded.RetentionExpiry.ReportPath != retentionExpirySurfacePath || decoded.RetentionExpiry.Ticket != "OPE-21" || decoded.RetentionExpiry.Track != "BIG-DUR-103" {
		t.Fatalf("unexpected retention expiry metadata: %+v", decoded.RetentionExpiry)
	}
	if decoded.RetentionExpiry.Summary.BackendCount != 5 || decoded.RetentionExpiry.Summary.RuntimeVisibleBackends != 4 || decoded.RetentionExpiry.Summary.PersistedBoundaryBackends != 2 || decoded.RetentionExpiry.Summary.FailClosedExpiryBackends != 3 || decoded.RetentionExpiry.Summary.ContractOnlyBackends != 1 {
		t.Fatalf("unexpected retention expiry summary: %+v", decoded.RetentionExpiry.Summary)
	}
	if len(decoded.RetentionExpiry.Backends) != 5 || decoded.RetentionExpiry.Backends[1].Backend != "sqlite" || !decoded.RetentionExpiry.Backends[1].PersistedBoundaries || !decoded.RetentionExpiry.Backends[2].FailClosedExpiry {
		t.Fatalf("unexpected retention expiry backends: %+v", decoded.RetentionExpiry.Backends)
	}
	if len(decoded.RetentionExpiry.PolicySplit) != 3 {
		t.Fatalf("expected retention policy split guidance, got %+v", decoded.RetentionExpiry.PolicySplit)
	}
}

func TestV2ControlCenterIncludesRetentionExpirySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		RetentionExpiry struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				PersistedBoundaryBackends int `json:"persisted_boundary_backends"`
			} `json:"summary"`
		} `json:"retention_expiry_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center retention expiry payload: %v", err)
	}
	if decoded.RetentionExpiry.ReportPath != retentionExpirySurfacePath || decoded.RetentionExpiry.Summary.PersistedBoundaryBackends != 2 {
		t.Fatalf("unexpected control center retention expiry payload: %+v", decoded.RetentionExpiry)
	}
}

func TestV2DistributedReportIncludesRetentionExpirySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		RetentionExpiry struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				RuntimeVisibleBackends   int `json:"runtime_visible_backends"`
				FailClosedExpiryBackends int `json:"fail_closed_expiry_backends"`
			} `json:"summary"`
		} `json:"retention_expiry_surface"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed retention expiry payload: %v", err)
	}
	if decoded.RetentionExpiry.ReportPath != retentionExpirySurfacePath || decoded.RetentionExpiry.Summary.RuntimeVisibleBackends != 4 || decoded.RetentionExpiry.Summary.FailClosedExpiryBackends != 3 {
		t.Fatalf("unexpected distributed retention expiry payload: %+v", decoded.RetentionExpiry)
	}
	if !strings.Contains(decoded.Report.Markdown, "## Retention Watermark & Expiry") || !strings.Contains(decoded.Report.Markdown, "Fail-closed expiry backends: 3") {
		t.Fatalf("expected retention expiry markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestRemoteEventLogExpiredCheckpointFailsClosedWithRetentionGuidance(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "event-log-remote.db")
	base := time.Unix(1_700_100_000, 0).UTC()
	log1, err := events.NewSQLiteEventLog(logPath)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	for _, event := range []domain.Event{
		{ID: "evt-remote-expired-1", Type: domain.EventTaskQueued, TaskID: "task-remote-expired", TraceID: "trace-remote-expired", Timestamp: base},
		{ID: "evt-remote-expired-2", Type: domain.EventTaskStarted, TaskID: "task-remote-expired", TraceID: "trace-remote-expired", Timestamp: base.Add(3 * time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote checkpoint event %s: %v", event.ID, err)
		}
	}
	if _, err := log1.Acknowledge("subscriber-remote-expired", "evt-remote-expired-1", base.Add(time.Second)); err != nil {
		t.Fatalf("ack remote expired checkpoint: %v", err)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}

	log2, err := events.NewSQLiteEventLogWithOptions(logPath, events.SQLiteEventLogOptions{
		Retention: 2 * time.Second,
		Now:       func() time.Time { return base.Add(4 * time.Second) },
	})
	if err != nil {
		t.Fatalf("reopen sqlite event log with retention: %v", err)
	}
	defer func() { _ = log2.Close() }()
	serviceServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: log2, Now: func() time.Time { return base.Add(4 * time.Second) }}
	serviceTS := httptest.NewServer(serviceServer.Handler())
	defer serviceTS.Close()

	remoteLog, err := events.NewHTTPEventLog(serviceTS.URL+"/internal/events/log", "")
	if err != nil {
		t.Fatalf("new remote event log: %v", err)
	}
	apiServer := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: remoteLog, Now: func() time.Time { return base.Add(4 * time.Second) }}

	checkpointResponse := httptest.NewRecorder()
	checkpointRequest := httptest.NewRequest(http.MethodGet, "/stream/events/checkpoints/subscriber-remote-expired", nil)
	apiServer.Handler().ServeHTTP(checkpointResponse, checkpointRequest)
	if checkpointResponse.Code != http.StatusOK {
		t.Fatalf("expected remote checkpoint diagnostics 200, got %d %s", checkpointResponse.Code, checkpointResponse.Body.String())
	}
	if !strings.Contains(checkpointResponse.Body.String(), "\"status\":\"expired\"") || !strings.Contains(checkpointResponse.Body.String(), "trimmed_through_sequence") || !strings.Contains(checkpointResponse.Body.String(), "evt-remote-expired-1") {
		t.Fatalf("expected remote expired checkpoint diagnostics, got %s", checkpointResponse.Body.String())
	}

	eventsResponse := httptest.NewRecorder()
	eventsRequest := httptest.NewRequest(http.MethodGet, "/events?subscriber_id=subscriber-remote-expired&trace_id=trace-remote-expired&limit=10", nil)
	apiServer.Handler().ServeHTTP(eventsResponse, eventsRequest)
	if eventsResponse.Code != http.StatusConflict {
		t.Fatalf("expected remote expired checkpoint conflict, got %d %s", eventsResponse.Code, eventsResponse.Body.String())
	}
	body := eventsResponse.Body.String()
	if !strings.Contains(body, "checkpoint_expired") || !strings.Contains(body, "DELETE /stream/events/checkpoints/{subscriber_id}") || !strings.Contains(body, "trimmed_through_sequence") {
		t.Fatalf("expected remote checkpoint expired payload with retention guidance, got %s", body)
	}
}

func TestDebugStatusIncludesProviderLiveHandoffIsolationSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		ProviderLiveHandoff struct {
			ReportPath     string `json:"report_path"`
			Ticket         string `json:"ticket"`
			Track          string `json:"track"`
			Backend        string `json:"backend"`
			ValidationLane string `json:"validation_lane"`
			Summary        struct {
				ScenarioCount          int  `json:"scenario_count"`
				IsolatedScenarios      int  `json:"isolated_scenarios"`
				StalledScenarios       int  `json:"stalled_scenarios"`
				ReplayBacklogEvents    int  `json:"replay_backlog_events"`
				LiveDeliveryDeadlineMS int  `json:"live_delivery_deadline_ms"`
				IsolationMaintained    bool `json:"isolation_maintained"`
			} `json:"summary"`
			Scenarios []struct {
				Name                  string `json:"name"`
				Status                string `json:"status"`
				ReplayDrainsAfterLive bool   `json:"replay_drains_after_live"`
			} `json:"scenarios"`
		} `json:"provider_live_handoff_isolation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode provider handoff payload: %v", err)
	}
	if decoded.ProviderLiveHandoff.ReportPath != providerLiveHandoffIsolationEvidencePackPath || decoded.ProviderLiveHandoff.Ticket != "OPE-225" || decoded.ProviderLiveHandoff.Track != "BIG-DUR-104" {
		t.Fatalf("unexpected provider handoff metadata: %+v", decoded.ProviderLiveHandoff)
	}
	if decoded.ProviderLiveHandoff.Backend != "http_remote_service" || decoded.ProviderLiveHandoff.ValidationLane != "external_store_validation" {
		t.Fatalf("unexpected provider handoff backend posture: %+v", decoded.ProviderLiveHandoff)
	}
	if decoded.ProviderLiveHandoff.Summary.ScenarioCount != 1 || decoded.ProviderLiveHandoff.Summary.IsolatedScenarios != 1 || decoded.ProviderLiveHandoff.Summary.StalledScenarios != 0 || decoded.ProviderLiveHandoff.Summary.ReplayBacklogEvents != 4 || decoded.ProviderLiveHandoff.Summary.LiveDeliveryDeadlineMS != 200 || !decoded.ProviderLiveHandoff.Summary.IsolationMaintained {
		t.Fatalf("unexpected provider handoff summary: %+v", decoded.ProviderLiveHandoff.Summary)
	}
	if len(decoded.ProviderLiveHandoff.Scenarios) != 1 || decoded.ProviderLiveHandoff.Scenarios[0].Name != "http_remote_service_replay_handoff_keeps_live_lane_unblocked" || decoded.ProviderLiveHandoff.Scenarios[0].Status != "isolated" || !decoded.ProviderLiveHandoff.Scenarios[0].ReplayDrainsAfterLive {
		t.Fatalf("unexpected provider handoff scenarios: %+v", decoded.ProviderLiveHandoff.Scenarios)
	}
}

func TestV2ControlCenterIncludesProviderLiveHandoffIsolationSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ProviderLiveHandoff struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				IsolatedScenarios int `json:"isolated_scenarios"`
			} `json:"summary"`
		} `json:"provider_live_handoff_isolation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center provider handoff payload: %v", err)
	}
	if decoded.ProviderLiveHandoff.ReportPath != providerLiveHandoffIsolationEvidencePackPath || decoded.ProviderLiveHandoff.Summary.IsolatedScenarios != 1 {
		t.Fatalf("unexpected control center provider handoff payload: %+v", decoded.ProviderLiveHandoff)
	}
}

func TestV2DistributedReportIncludesProviderLiveHandoffIsolationSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ProviderLiveHandoff struct {
			ReportPath string `json:"report_path"`
			Backend    string `json:"backend"`
			Summary    struct {
				IsolatedScenarios      int `json:"isolated_scenarios"`
				LiveDeliveryDeadlineMS int `json:"live_delivery_deadline_ms"`
			} `json:"summary"`
		} `json:"provider_live_handoff_isolation"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed provider handoff payload: %v", err)
	}
	if decoded.ProviderLiveHandoff.ReportPath != providerLiveHandoffIsolationEvidencePackPath || decoded.ProviderLiveHandoff.Backend != "http_remote_service" || decoded.ProviderLiveHandoff.Summary.IsolatedScenarios != 1 || decoded.ProviderLiveHandoff.Summary.LiveDeliveryDeadlineMS != 200 {
		t.Fatalf("unexpected distributed provider handoff payload: %+v", decoded.ProviderLiveHandoff)
	}
	if !strings.Contains(decoded.Report.Markdown, "## Provider-backed Live Handoff Isolation") || !strings.Contains(decoded.Report.Markdown, "Backend: http_remote_service") {
		t.Fatalf("expected provider handoff markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestDebugStatusIncludesClawHostProxyAdminValidationLane(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		ClawHost struct {
			ReportPath     string `json:"report_path"`
			Ticket         string `json:"ticket"`
			Provider       string `json:"provider"`
			ValidationLane string `json:"validation_lane"`
			Summary        struct {
				AppCount               int    `json:"app_count"`
				BotCount               int    `json:"bot_count"`
				HTTPReachableBots      int    `json:"http_reachable_bots"`
				WebsocketReachableBots int    `json:"websocket_reachable_bots"`
				SubdomainReadyBots     int    `json:"subdomain_ready_bots"`
				AdminReadyBots         int    `json:"admin_ready_bots"`
				DegradedBots           int    `json:"degraded_bots"`
				ParallelProbeWidth     int    `json:"parallel_probe_width"`
				ReviewerExportStatus   string `json:"reviewer_export_status"`
			} `json:"summary"`
			Bots []struct {
				AppID            string `json:"app_id"`
				BotID            string `json:"bot_id"`
				WebsocketStatus  string `json:"websocket_status"`
				ValidationStatus string `json:"validation_status"`
			} `json:"bots"`
		} `json:"clawhost_proxy_admin_validation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode clawhost validation payload: %v", err)
	}
	if decoded.ClawHost.ReportPath != clawHostProxyAdminValidationLanePath || decoded.ClawHost.Ticket != "BIG-PAR-291" || decoded.ClawHost.Provider != "clawhost" || decoded.ClawHost.ValidationLane != "clawhost_proxy_admin_parallel_probe" {
		t.Fatalf("unexpected clawhost validation metadata: %+v", decoded.ClawHost)
	}
	if decoded.ClawHost.Summary.AppCount != 2 || decoded.ClawHost.Summary.BotCount != 3 || decoded.ClawHost.Summary.HTTPReachableBots != 3 || decoded.ClawHost.Summary.WebsocketReachableBots != 2 || decoded.ClawHost.Summary.SubdomainReadyBots != 3 || decoded.ClawHost.Summary.AdminReadyBots != 2 || decoded.ClawHost.Summary.DegradedBots != 1 || decoded.ClawHost.Summary.ParallelProbeWidth != 3 || decoded.ClawHost.Summary.ReviewerExportStatus != "ready" {
		t.Fatalf("unexpected clawhost validation summary: %+v", decoded.ClawHost.Summary)
	}
	if len(decoded.ClawHost.Bots) != 3 || decoded.ClawHost.Bots[2].BotID != "bot-support-a" || decoded.ClawHost.Bots[2].WebsocketStatus != "degraded" || decoded.ClawHost.Bots[2].ValidationStatus != "degraded" {
		t.Fatalf("unexpected clawhost validation bots: %+v", decoded.ClawHost.Bots)
	}
}

func TestV2ControlCenterIncludesClawHostProxyAdminValidationLane(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHost struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				BotCount           int `json:"bot_count"`
				AdminReadyBots     int `json:"admin_ready_bots"`
				SubdomainReadyBots int `json:"subdomain_ready_bots"`
			} `json:"summary"`
		} `json:"clawhost_proxy_admin_validation"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center clawhost payload: %v", err)
	}
	if decoded.ClawHost.ReportPath != clawHostProxyAdminValidationLanePath || decoded.ClawHost.Summary.BotCount != 3 || decoded.ClawHost.Summary.AdminReadyBots != 2 || decoded.ClawHost.Summary.SubdomainReadyBots != 3 {
		t.Fatalf("unexpected control center clawhost payload: %+v", decoded.ClawHost)
	}
}

func TestV2DistributedReportIncludesClawHostProxyAdminValidationLane(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		ClawHost struct {
			ReportPath     string `json:"report_path"`
			Provider       string `json:"provider"`
			ValidationLane string `json:"validation_lane"`
			Summary        struct {
				BotCount               int `json:"bot_count"`
				HTTPReachableBots      int `json:"http_reachable_bots"`
				WebsocketReachableBots int `json:"websocket_reachable_bots"`
				DegradedBots           int `json:"degraded_bots"`
			} `json:"summary"`
		} `json:"clawhost_proxy_admin_validation"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed clawhost payload: %v", err)
	}
	if decoded.ClawHost.ReportPath != clawHostProxyAdminValidationLanePath || decoded.ClawHost.Provider != "clawhost" || decoded.ClawHost.ValidationLane != "clawhost_proxy_admin_parallel_probe" || decoded.ClawHost.Summary.BotCount != 3 || decoded.ClawHost.Summary.HTTPReachableBots != 3 || decoded.ClawHost.Summary.WebsocketReachableBots != 2 || decoded.ClawHost.Summary.DegradedBots != 1 {
		t.Fatalf("unexpected distributed clawhost payload: %+v", decoded.ClawHost)
	}
	if !strings.Contains(decoded.Report.Markdown, "## ClawHost Proxy and Admin Validation") || !strings.Contains(decoded.Report.Markdown, "Provider: clawhost") || !strings.Contains(decoded.Report.Markdown, "bot-support-a: validation=degraded") {
		t.Fatalf("expected clawhost markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestDebugStatusIncludesClawHostFleetInventorySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		Fleet struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Provider   string `json:"provider"`
			SourceKind string `json:"source_kind"`
			Summary    struct {
				AppCount          int `json:"app_count"`
				BotCount          int `json:"bot_count"`
				ActiveBots        int `json:"active_bots"`
				SuspendedBots     int `json:"suspended_bots"`
				DegradedBots      int `json:"degraded_bots"`
				TenantCount       int `json:"tenant_count"`
				DomainCount       int `json:"domain_count"`
				OwnershipTeams    int `json:"ownership_teams"`
				ReviewerReadyApps int `json:"reviewer_ready_apps"`
			} `json:"summary"`
			Apps []struct {
				AppID    string `json:"app_id"`
				BotCount int    `json:"bot_count"`
			} `json:"apps"`
			Bots []struct {
				BotID          string `json:"bot_id"`
				LifecycleState string `json:"lifecycle_state"`
			} `json:"bots"`
		} `json:"clawhost_fleet_inventory"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode clawhost fleet payload: %v", err)
	}
	if decoded.Fleet.ReportPath != clawHostFleetInventorySurfacePath || decoded.Fleet.Ticket != "BIG-PAR-287" || decoded.Fleet.Provider != "clawhost" || decoded.Fleet.SourceKind != "app_bot_control_plane_inventory" {
		t.Fatalf("unexpected clawhost fleet metadata: %+v", decoded.Fleet)
	}
	if decoded.Fleet.Summary.AppCount != 2 || decoded.Fleet.Summary.BotCount != 5 || decoded.Fleet.Summary.ActiveBots != 3 || decoded.Fleet.Summary.SuspendedBots != 1 || decoded.Fleet.Summary.DegradedBots != 1 || decoded.Fleet.Summary.TenantCount != 2 || decoded.Fleet.Summary.DomainCount != 5 || decoded.Fleet.Summary.OwnershipTeams != 3 || decoded.Fleet.Summary.ReviewerReadyApps != 2 {
		t.Fatalf("unexpected clawhost fleet summary: %+v", decoded.Fleet.Summary)
	}
	if len(decoded.Fleet.Apps) != 2 || decoded.Fleet.Apps[0].AppID != "clawhost-sales" || len(decoded.Fleet.Bots) != 5 || decoded.Fleet.Bots[2].LifecycleState != "degraded" {
		t.Fatalf("unexpected clawhost fleet entries: %+v %+v", decoded.Fleet.Apps, decoded.Fleet.Bots)
	}
}

func TestV2ControlCenterIncludesClawHostFleetInventorySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Fleet struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				BotCount     int `json:"bot_count"`
				ActiveBots   int `json:"active_bots"`
				DegradedBots int `json:"degraded_bots"`
				TenantCount  int `json:"tenant_count"`
				DomainCount  int `json:"domain_count"`
			} `json:"summary"`
		} `json:"clawhost_fleet_inventory"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center clawhost fleet payload: %v", err)
	}
	if decoded.Fleet.ReportPath != clawHostFleetInventorySurfacePath || decoded.Fleet.Summary.BotCount != 5 || decoded.Fleet.Summary.ActiveBots != 3 || decoded.Fleet.Summary.DegradedBots != 1 || decoded.Fleet.Summary.TenantCount != 2 || decoded.Fleet.Summary.DomainCount != 5 {
		t.Fatalf("unexpected control center clawhost fleet payload: %+v", decoded.Fleet)
	}
}

func TestV2DistributedReportIncludesClawHostFleetInventorySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Fleet struct {
			ReportPath string `json:"report_path"`
			Provider   string `json:"provider"`
			SourceKind string `json:"source_kind"`
			Summary    struct {
				BotCount      int `json:"bot_count"`
				ActiveBots    int `json:"active_bots"`
				SuspendedBots int `json:"suspended_bots"`
				DegradedBots  int `json:"degraded_bots"`
				DomainCount   int `json:"domain_count"`
			} `json:"summary"`
		} `json:"clawhost_fleet_inventory"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed clawhost fleet payload: %v", err)
	}
	if decoded.Fleet.ReportPath != clawHostFleetInventorySurfacePath || decoded.Fleet.Provider != "clawhost" || decoded.Fleet.SourceKind != "app_bot_control_plane_inventory" || decoded.Fleet.Summary.BotCount != 5 || decoded.Fleet.Summary.ActiveBots != 3 || decoded.Fleet.Summary.SuspendedBots != 1 || decoded.Fleet.Summary.DegradedBots != 1 || decoded.Fleet.Summary.DomainCount != 5 {
		t.Fatalf("unexpected distributed clawhost fleet payload: %+v", decoded.Fleet)
	}
	if !strings.Contains(decoded.Report.Markdown, "## ClawHost Fleet Inventory") || !strings.Contains(decoded.Report.Markdown, "Source kind: app_bot_control_plane_inventory") || !strings.Contains(decoded.Report.Markdown, "bot bot-sales-c: app=clawhost-sales lifecycle=degraded") {
		t.Fatalf("expected clawhost fleet markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestDebugStatusIncludesClawHostRolloutPlannerSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		Rollout struct {
			ReportPath  string `json:"report_path"`
			Ticket      string `json:"ticket"`
			Provider    string `json:"provider"`
			PlannerMode string `json:"planner_mode"`
			Summary     struct {
				AppCount           int    `json:"app_count"`
				BotCount           int    `json:"bot_count"`
				TotalWaves         int    `json:"total_waves"`
				CanaryWaves        int    `json:"canary_waves"`
				MaxParallelism     int    `json:"max_parallelism"`
				TakeoverProtected  int    `json:"takeover_protected_waves"`
				EvidenceReadyWaves int    `json:"evidence_ready_waves"`
				BlockedWaves       int    `json:"blocked_waves"`
				ExecutionReadiness string `json:"execution_readiness"`
			} `json:"summary"`
			Waves []struct {
				WaveID           string `json:"wave_id"`
				Action           string `json:"action"`
				ValidationStatus string `json:"validation_status"`
				TakeoverRequired bool   `json:"takeover_required"`
			} `json:"waves"`
		} `json:"clawhost_rollout_planner"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode clawhost rollout payload: %v", err)
	}
	if decoded.Rollout.ReportPath != clawHostRolloutPlannerSurfacePath || decoded.Rollout.Ticket != "BIG-PAR-288" || decoded.Rollout.Provider != "clawhost" || decoded.Rollout.PlannerMode != "wave_canary_takeover_guarded" {
		t.Fatalf("unexpected clawhost rollout metadata: %+v", decoded.Rollout)
	}
	if decoded.Rollout.Summary.AppCount != 2 || decoded.Rollout.Summary.BotCount != 5 || decoded.Rollout.Summary.TotalWaves != 3 || decoded.Rollout.Summary.CanaryWaves != 1 || decoded.Rollout.Summary.MaxParallelism != 2 || decoded.Rollout.Summary.TakeoverProtected != 2 || decoded.Rollout.Summary.EvidenceReadyWaves != 2 || decoded.Rollout.Summary.BlockedWaves != 1 || decoded.Rollout.Summary.ExecutionReadiness != "guarded_ready" {
		t.Fatalf("unexpected clawhost rollout summary: %+v", decoded.Rollout.Summary)
	}
	if len(decoded.Rollout.Waves) != 3 || decoded.Rollout.Waves[0].WaveID != "wave-canary-restart-sales" || !decoded.Rollout.Waves[0].TakeoverRequired || decoded.Rollout.Waves[2].ValidationStatus != "blocked" {
		t.Fatalf("unexpected clawhost rollout waves: %+v", decoded.Rollout.Waves)
	}
}

func TestV2ControlCenterIncludesClawHostRolloutPlannerSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Rollout struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				TotalWaves         int `json:"total_waves"`
				CanaryWaves        int `json:"canary_waves"`
				TakeoverProtected  int `json:"takeover_protected_waves"`
				EvidenceReadyWaves int `json:"evidence_ready_waves"`
			} `json:"summary"`
		} `json:"clawhost_rollout_planner"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center clawhost rollout payload: %v", err)
	}
	if decoded.Rollout.ReportPath != clawHostRolloutPlannerSurfacePath || decoded.Rollout.Summary.TotalWaves != 3 || decoded.Rollout.Summary.CanaryWaves != 1 || decoded.Rollout.Summary.TakeoverProtected != 2 || decoded.Rollout.Summary.EvidenceReadyWaves != 2 {
		t.Fatalf("unexpected control center clawhost rollout payload: %+v", decoded.Rollout)
	}
}

func TestV2DistributedReportIncludesClawHostRolloutPlannerSurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Rollout struct {
			ReportPath  string `json:"report_path"`
			Provider    string `json:"provider"`
			PlannerMode string `json:"planner_mode"`
			Summary     struct {
				BotCount          int `json:"bot_count"`
				TotalWaves        int `json:"total_waves"`
				CanaryWaves       int `json:"canary_waves"`
				MaxParallelism    int `json:"max_parallelism"`
				TakeoverProtected int `json:"takeover_protected_waves"`
				BlockedWaves      int `json:"blocked_waves"`
			} `json:"summary"`
		} `json:"clawhost_rollout_planner"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed clawhost rollout payload: %v", err)
	}
	if decoded.Rollout.ReportPath != clawHostRolloutPlannerSurfacePath || decoded.Rollout.Provider != "clawhost" || decoded.Rollout.PlannerMode != "wave_canary_takeover_guarded" || decoded.Rollout.Summary.BotCount != 5 || decoded.Rollout.Summary.TotalWaves != 3 || decoded.Rollout.Summary.CanaryWaves != 1 || decoded.Rollout.Summary.MaxParallelism != 2 || decoded.Rollout.Summary.TakeoverProtected != 2 || decoded.Rollout.Summary.BlockedWaves != 1 {
		t.Fatalf("unexpected distributed clawhost rollout payload: %+v", decoded.Rollout)
	}
	if !strings.Contains(decoded.Report.Markdown, "## ClawHost Rollout Planner") || !strings.Contains(decoded.Report.Markdown, "Planner mode: wave_canary_takeover_guarded") || !strings.Contains(decoded.Report.Markdown, "wave-support-websocket-unblock: action=restart validation=blocked") {
		t.Fatalf("expected clawhost rollout markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestDebugStatusIncludesClawHostTenantPolicySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		Policy struct {
			ReportPath string `json:"report_path"`
			Ticket     string `json:"ticket"`
			Provider   string `json:"provider"`
			PolicyMode string `json:"policy_mode"`
			Summary    struct {
				TenantCount            int `json:"tenant_count"`
				AppDefaultCount        int `json:"app_default_count"`
				MultiProviderTenants   int `json:"multi_provider_tenants"`
				EntitlementGuardrails  int `json:"entitlement_guardrails"`
				RolloutBlockedDefaults int `json:"rollout_blocked_defaults"`
				ReviewerReadyTenants   int `json:"reviewer_ready_tenants"`
			} `json:"summary"`
			Tenants []struct {
				Tenant                string `json:"tenant"`
				DefaultProvider       string `json:"default_provider"`
				BlockedDefaultChanges int    `json:"blocked_default_changes"`
			} `json:"tenants"`
		} `json:"clawhost_tenant_policy"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode clawhost tenant policy payload: %v", err)
	}
	if decoded.Policy.ReportPath != clawHostTenantPolicySurfacePath || decoded.Policy.Ticket != "BIG-PAR-290" || decoded.Policy.Provider != "clawhost" || decoded.Policy.PolicyMode != "tenant_guarded_provider_defaults" {
		t.Fatalf("unexpected clawhost tenant policy metadata: %+v", decoded.Policy)
	}
	if decoded.Policy.Summary.TenantCount != 2 || decoded.Policy.Summary.AppDefaultCount != 3 || decoded.Policy.Summary.MultiProviderTenants != 2 || decoded.Policy.Summary.EntitlementGuardrails != 2 || decoded.Policy.Summary.RolloutBlockedDefaults != 1 || decoded.Policy.Summary.ReviewerReadyTenants != 2 {
		t.Fatalf("unexpected clawhost tenant policy summary: %+v", decoded.Policy.Summary)
	}
	if len(decoded.Policy.Tenants) != 2 || decoded.Policy.Tenants[0].Tenant != "tenant-acme" || decoded.Policy.Tenants[0].BlockedDefaultChanges != 1 {
		t.Fatalf("unexpected clawhost tenant policy tenants: %+v", decoded.Policy.Tenants)
	}
}

func TestV2ControlCenterIncludesClawHostTenantPolicySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Policy struct {
			ReportPath string `json:"report_path"`
			Summary    struct {
				TenantCount            int `json:"tenant_count"`
				AppDefaultCount        int `json:"app_default_count"`
				RolloutBlockedDefaults int `json:"rollout_blocked_defaults"`
			} `json:"summary"`
		} `json:"clawhost_tenant_policy"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center clawhost tenant policy payload: %v", err)
	}
	if decoded.Policy.ReportPath != clawHostTenantPolicySurfacePath || decoded.Policy.Summary.TenantCount != 2 || decoded.Policy.Summary.AppDefaultCount != 3 || decoded.Policy.Summary.RolloutBlockedDefaults != 1 {
		t.Fatalf("unexpected control center clawhost tenant policy payload: %+v", decoded.Policy)
	}
}

func TestV2DistributedReportIncludesClawHostTenantPolicySurface(t *testing.T) {
	server := &Server{
		Recorder: observability.NewRecorder(),
		Queue:    queue.NewMemoryQueue(),
		Bus:      events.NewBus(),
		Control:  control.New(),
		Now:      time.Now,
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)

	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Policy struct {
			ReportPath string `json:"report_path"`
			Provider   string `json:"provider"`
			PolicyMode string `json:"policy_mode"`
			Summary    struct {
				TenantCount            int `json:"tenant_count"`
				AppDefaultCount        int `json:"app_default_count"`
				MultiProviderTenants   int `json:"multi_provider_tenants"`
				EntitlementGuardrails  int `json:"entitlement_guardrails"`
				RolloutBlockedDefaults int `json:"rollout_blocked_defaults"`
			} `json:"summary"`
		} `json:"clawhost_tenant_policy"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed clawhost tenant policy payload: %v", err)
	}
	if decoded.Policy.ReportPath != clawHostTenantPolicySurfacePath || decoded.Policy.Provider != "clawhost" || decoded.Policy.PolicyMode != "tenant_guarded_provider_defaults" || decoded.Policy.Summary.TenantCount != 2 || decoded.Policy.Summary.AppDefaultCount != 3 || decoded.Policy.Summary.MultiProviderTenants != 2 || decoded.Policy.Summary.EntitlementGuardrails != 2 || decoded.Policy.Summary.RolloutBlockedDefaults != 1 {
		t.Fatalf("unexpected distributed clawhost tenant policy payload: %+v", decoded.Policy)
	}
	if !strings.Contains(decoded.Report.Markdown, "## ClawHost Tenant Policy") || !strings.Contains(decoded.Report.Markdown, "Policy mode: tenant_guarded_provider_defaults") || !strings.Contains(decoded.Report.Markdown, "tenant tenant-acme: default_provider=openai") {
		t.Fatalf("expected clawhost tenant policy markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestProviderBackedRemoteEventLogLiveHandoffIsolation(t *testing.T) {
	base := time.Unix(1_700_200_000, 0).UTC()
	backlog := []domain.Event{
		{ID: "evt-provider-backlog-1", Type: domain.EventTaskQueued, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base},
		{ID: "evt-provider-backlog-2", Type: domain.EventTaskStarted, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base.Add(time.Second)},
		{ID: "evt-provider-backlog-3", Type: domain.EventTaskCompleted, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base.Add(2 * time.Second)},
		{ID: "evt-provider-backlog-4", Type: domain.EventTaskQueued, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base.Add(3 * time.Second)},
	}
	liveEvent := domain.Event{ID: "evt-provider-live", Type: domain.EventTaskStarted, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base.Add(4 * time.Second)}
	handoffEvent := domain.Event{ID: "evt-provider-handoff", Type: domain.EventTaskCompleted, TaskID: "task-provider", TraceID: "trace-provider", Timestamp: base.Add(5 * time.Second)}
	remoteStore := &blockingRemoteServiceLog{blockingEventLog: &blockingEventLog{
		history:       backlog,
		replayStarted: make(chan struct{}, 1),
		release:       make(chan struct{}),
	}}
	remoteService := httptest.NewServer(events.NewEventLogServiceHandler(remoteStore))
	defer remoteService.Close()
	remoteLog, err := events.NewHTTPEventLog(remoteService.URL, "")
	if err != nil {
		t.Fatalf("new remote event log: %v", err)
	}
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), EventLog: remoteLog, Now: time.Now}
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	type sseResult struct {
		lines []string
		err   string
	}
	liveCtx, liveCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer liveCancel()
	liveReq, err := http.NewRequestWithContext(liveCtx, http.MethodGet, ts.URL+"/stream/events?trace_id=trace-provider&limit=10", nil)
	if err != nil {
		t.Fatalf("new live request: %v", err)
	}
	liveCh := make(chan sseResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(liveReq)
		if err != nil {
			liveCh <- sseResult{err: err.Error()}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		line1, err := reader.ReadString('\n')
		if err != nil {
			liveCh <- sseResult{err: err.Error()}
			return
		}
		line2, err := reader.ReadString('\n')
		if err != nil {
			liveCh <- sseResult{err: err.Error()}
			return
		}
		liveCh <- sseResult{lines: []string{strings.TrimSpace(line1), strings.TrimSpace(line2)}}
	}()

	replayCtx, replayCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer replayCancel()
	replayReq, err := http.NewRequestWithContext(replayCtx, http.MethodGet, ts.URL+"/stream/events?trace_id=trace-provider&after_id=evt-before&limit=10", nil)
	if err != nil {
		t.Fatalf("new replay request: %v", err)
	}
	replayCh := make(chan sseResult, 1)
	go func() {
		response, err := http.DefaultClient.Do(replayReq)
		if err != nil {
			replayCh <- sseResult{err: err.Error()}
			return
		}
		defer response.Body.Close()
		reader := bufio.NewReader(response.Body)
		lines := make([]string, 0, 16)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				replayCh <- sseResult{lines: lines, err: err.Error()}
				return
			}
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}()

	select {
	case <-remoteStore.replayStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for provider-backed replay to start")
	}

	publishStarted := time.Now()
	server.Bus.Publish(liveEvent)
	select {
	case result := <-liveCh:
		if result.err != "" {
			t.Fatalf("live subscriber failed: %s", result.err)
		}
		if time.Since(publishStarted) > 200*time.Millisecond {
			t.Fatalf("expected live subscriber delivery within 200ms while replay was blocked, took %s", time.Since(publishStarted))
		}
		if len(result.lines) != 2 || !strings.Contains(result.lines[0], "evt-provider-live") || result.lines[1] != "id: evt-provider-live" {
			t.Fatalf("unexpected live subscriber lines: %+v", result.lines)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live-only subscriber while replay was blocked")
	}

	close(remoteStore.release)
	server.Bus.Publish(handoffEvent)
	time.Sleep(150 * time.Millisecond)
	replayCancel()

	result := <-replayCh
	if len(result.lines) == 0 {
		t.Fatalf("expected replay subscriber lines, got err=%q", result.err)
	}
	body := strings.Join(result.lines, "\n")
	for _, want := range []string{"evt-provider-backlog-1", "evt-provider-backlog-2", "evt-provider-backlog-3", "evt-provider-backlog-4", "evt-provider-live", "evt-provider-handoff"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected replay/live handoff body to include %s, got %s", want, body)
		}
	}
}

func TestDebugStatusIncludesBrokerBootstrapSurface(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		BrokerBootstrap struct {
			ReportPath                    string `json:"report_path"`
			CanonicalSummaryPath          string `json:"canonical_summary_path"`
			CanonicalBootstrapSummaryPath string `json:"canonical_bootstrap_summary_path"`
			ValidationPackPath            string `json:"validation_pack_path"`
			ConfigurationState            string `json:"configuration_state"`
			BootstrapReady                bool   `json:"bootstrap_ready"`
			RuntimePosture                string `json:"runtime_posture"`
			LiveAdapterImplemented        bool   `json:"live_adapter_implemented"`
			ProofBoundary                 string `json:"proof_boundary"`
			ConfigCompleteness            struct {
				Driver        bool `json:"driver"`
				URLs          bool `json:"urls"`
				Topic         bool `json:"topic"`
				ConsumerGroup bool `json:"consumer_group"`
			} `json:"config_completeness"`
			ConfigDiagnostics struct {
				MissingFields      []string `json:"missing_fields"`
				RequiredEnv        []string `json:"required_env"`
				MissingRequiredEnv []string `json:"missing_required_env"`
				AdvisoryEnv        []string `json:"advisory_env"`
				MissingAdvisoryEnv []string `json:"missing_advisory_env"`
				NextActions        []string `json:"next_actions"`
				ReferenceDocs      []string `json:"reference_docs"`
				RuntimeKnobs       struct {
					PublishTimeout     string `json:"publish_timeout"`
					ReplayLimit        int    `json:"replay_limit"`
					CheckpointInterval string `json:"checkpoint_interval"`
					ConsumerGroup      string `json:"consumer_group"`
				} `json:"runtime_knobs"`
			} `json:"config_diagnostics"`
			RuntimeGate struct {
				Status             string `json:"status"`
				Requested          bool   `json:"requested"`
				FailClosed         bool   `json:"fail_closed"`
				ContractOnly       bool   `json:"contract_only"`
				StubDriverOnly     bool   `json:"stub_driver_only"`
				SafeForLiveTraffic bool   `json:"safe_for_live_traffic"`
				OperatorMessage    string `json:"operator_message"`
				ProofBoundary      string `json:"proof_boundary"`
			} `json:"runtime_gate"`
			ValidationErrors []string `json:"validation_errors"`
		} `json:"broker_bootstrap_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode broker bootstrap payload: %v", err)
	}
	if decoded.BrokerBootstrap.ReportPath != brokerBootstrapSurfacePath || decoded.BrokerBootstrap.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" || decoded.BrokerBootstrap.CanonicalBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" || decoded.BrokerBootstrap.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" {
		t.Fatalf("unexpected broker bootstrap paths: %+v", decoded.BrokerBootstrap)
	}
	if decoded.BrokerBootstrap.ConfigurationState != "not_configured" || decoded.BrokerBootstrap.BootstrapReady || decoded.BrokerBootstrap.RuntimePosture != "contract_only" || decoded.BrokerBootstrap.LiveAdapterImplemented {
		t.Fatalf("unexpected broker bootstrap posture: %+v", decoded.BrokerBootstrap)
	}
	if decoded.BrokerBootstrap.ProofBoundary == "" || len(decoded.BrokerBootstrap.ValidationErrors) == 0 {
		t.Fatalf("expected broker bootstrap boundary/errors, got %+v", decoded.BrokerBootstrap)
	}
	if decoded.BrokerBootstrap.ConfigCompleteness.Driver || decoded.BrokerBootstrap.ConfigCompleteness.URLs || decoded.BrokerBootstrap.ConfigCompleteness.Topic || decoded.BrokerBootstrap.ConfigCompleteness.ConsumerGroup {
		t.Fatalf("expected missing broker config completeness, got %+v", decoded.BrokerBootstrap.ConfigCompleteness)
	}
	if strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.MissingFields, ",") != "driver,urls,topic" {
		t.Fatalf("unexpected missing fields: %+v", decoded.BrokerBootstrap.ConfigDiagnostics)
	}
	if strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.MissingRequiredEnv, ",") != "BIGCLAW_EVENT_LOG_BROKER_DRIVER,BIGCLAW_EVENT_LOG_BROKER_URLS,BIGCLAW_EVENT_LOG_BROKER_TOPIC" {
		t.Fatalf("unexpected missing required env: %+v", decoded.BrokerBootstrap.ConfigDiagnostics)
	}
	if strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.MissingAdvisoryEnv, ",") != "BIGCLAW_EVENT_LOG_CONSUMER_GROUP" {
		t.Fatalf("unexpected missing advisory env: %+v", decoded.BrokerBootstrap.ConfigDiagnostics)
	}
	if decoded.BrokerBootstrap.ConfigDiagnostics.RuntimeKnobs.PublishTimeout != "5s" || decoded.BrokerBootstrap.ConfigDiagnostics.RuntimeKnobs.ReplayLimit != 500 || decoded.BrokerBootstrap.ConfigDiagnostics.RuntimeKnobs.CheckpointInterval != "5s" {
		t.Fatalf("unexpected runtime knobs: %+v", decoded.BrokerBootstrap.ConfigDiagnostics.RuntimeKnobs)
	}
	if len(decoded.BrokerBootstrap.ConfigDiagnostics.NextActions) == 0 || !strings.Contains(strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.NextActions, " | "), "BIGCLAW_EVENT_LOG_BROKER_DRIVER") {
		t.Fatalf("expected operator next actions, got %+v", decoded.BrokerBootstrap.ConfigDiagnostics.NextActions)
	}
	if !strings.Contains(strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.ReferenceDocs, " | "), "docs/reports/broker-event-log-adapter-contract.md") {
		t.Fatalf("expected broker operator guide reference, got %+v", decoded.BrokerBootstrap.ConfigDiagnostics.ReferenceDocs)
	}
	if decoded.BrokerBootstrap.RuntimeGate.Status != "contract_only" || !decoded.BrokerBootstrap.RuntimeGate.Requested || decoded.BrokerBootstrap.RuntimeGate.FailClosed || !decoded.BrokerBootstrap.RuntimeGate.ContractOnly || decoded.BrokerBootstrap.RuntimeGate.StubDriverOnly || decoded.BrokerBootstrap.RuntimeGate.SafeForLiveTraffic {
		t.Fatalf("unexpected broker runtime gate: %+v", decoded.BrokerBootstrap.RuntimeGate)
	}
	if !strings.Contains(decoded.BrokerBootstrap.RuntimeGate.OperatorMessage, "contract-only") || decoded.BrokerBootstrap.RuntimeGate.ProofBoundary == "" {
		t.Fatalf("expected runtime gate messaging, got %+v", decoded.BrokerBootstrap.RuntimeGate)
	}
}

func TestV2ControlCenterIncludesBrokerBootstrapSurface(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		BrokerBootstrap struct {
			ReportPath         string `json:"report_path"`
			ConfigurationState string `json:"configuration_state"`
			RuntimePosture     string `json:"runtime_posture"`
			ConfigDiagnostics  struct {
				MissingRequiredEnv []string `json:"missing_required_env"`
			} `json:"config_diagnostics"`
			RuntimeGate struct {
				Status        string `json:"status"`
				Requested     bool   `json:"requested"`
				FailClosed    bool   `json:"fail_closed"`
				ProofBoundary string `json:"proof_boundary"`
			} `json:"runtime_gate"`
		} `json:"broker_bootstrap_surface"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center broker bootstrap payload: %v", err)
	}
	if decoded.BrokerBootstrap.ReportPath != brokerBootstrapSurfacePath || decoded.BrokerBootstrap.ConfigurationState != "not_configured" || decoded.BrokerBootstrap.RuntimePosture != "contract_only" {
		t.Fatalf("unexpected control center broker bootstrap payload: %+v", decoded.BrokerBootstrap)
	}
	if strings.Join(decoded.BrokerBootstrap.ConfigDiagnostics.MissingRequiredEnv, ",") != "BIGCLAW_EVENT_LOG_BROKER_DRIVER,BIGCLAW_EVENT_LOG_BROKER_URLS,BIGCLAW_EVENT_LOG_BROKER_TOPIC" {
		t.Fatalf("unexpected control center config diagnostics: %+v", decoded.BrokerBootstrap.ConfigDiagnostics)
	}
	if decoded.BrokerBootstrap.RuntimeGate.Status != "contract_only" || !decoded.BrokerBootstrap.RuntimeGate.Requested || decoded.BrokerBootstrap.RuntimeGate.FailClosed || decoded.BrokerBootstrap.RuntimeGate.ProofBoundary == "" {
		t.Fatalf("unexpected control center runtime gate: %+v", decoded.BrokerBootstrap.RuntimeGate)
	}
}

func TestV2DistributedReportIncludesBrokerBootstrapSurface(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		BrokerBootstrap struct {
			ReportPath         string `json:"report_path"`
			ConfigurationState string `json:"configuration_state"`
			RuntimePosture     string `json:"runtime_posture"`
			BootstrapReady     bool   `json:"bootstrap_ready"`
		} `json:"broker_bootstrap_surface"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed broker bootstrap payload: %v", err)
	}
	if decoded.BrokerBootstrap.ReportPath != brokerBootstrapSurfacePath || decoded.BrokerBootstrap.ConfigurationState != "not_configured" || decoded.BrokerBootstrap.RuntimePosture != "contract_only" || decoded.BrokerBootstrap.BootstrapReady {
		t.Fatalf("unexpected distributed broker bootstrap payload: %+v", decoded.BrokerBootstrap)
	}
	if !strings.Contains(decoded.Report.Markdown, "## Broker Bootstrap Readiness") || !strings.Contains(decoded.Report.Markdown, "Configuration state: not_configured") || !strings.Contains(decoded.Report.Markdown, "Runtime posture: contract_only") || !strings.Contains(decoded.Report.Markdown, "Runtime gate: requested=true fail_closed=false contract_only=true stub_driver_only=false safe_for_live_traffic=false") || !strings.Contains(decoded.Report.Markdown, "Runtime gate message: Broker durability remains a contract-only target") || !strings.Contains(decoded.Report.Markdown, "Missing required env: BIGCLAW_EVENT_LOG_BROKER_DRIVER, BIGCLAW_EVENT_LOG_BROKER_URLS, BIGCLAW_EVENT_LOG_BROKER_TOPIC") || !strings.Contains(decoded.Report.Markdown, "Reference docs: docs/reports/broker-event-log-adapter-contract.md") {
		t.Fatalf("expected broker bootstrap markdown section, got %s", decoded.Report.Markdown)
	}
}

func TestDebugStatusIncludesBrokerReviewBundle(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/debug/status", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	var decoded struct {
		BrokerReviewBundle struct {
			CanonicalSummaryPath          string   `json:"canonical_summary_path"`
			CanonicalBootstrapSummaryPath string   `json:"canonical_bootstrap_summary_path"`
			ValidationPackPath            string   `json:"validation_pack_path"`
			ReviewReadinessPath           string   `json:"review_readiness_path"`
			OperatorGuidePath             string   `json:"operator_guide_path"`
			RuntimePosture                string   `json:"runtime_posture"`
			ReviewerLinks                 []string `json:"reviewer_links"`
		} `json:"broker_review_bundle"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode broker review bundle payload: %v", err)
	}
	if decoded.BrokerReviewBundle.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" || decoded.BrokerReviewBundle.CanonicalBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" || decoded.BrokerReviewBundle.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" || decoded.BrokerReviewBundle.ReviewReadinessPath != "docs/reports/review-readiness.md" || decoded.BrokerReviewBundle.OperatorGuidePath != "docs/reports/broker-event-log-adapter-contract.md" || decoded.BrokerReviewBundle.RuntimePosture != "contract_only" {
		t.Fatalf("unexpected broker review bundle payload: %+v", decoded.BrokerReviewBundle)
	}
	if !strings.Contains(strings.Join(decoded.BrokerReviewBundle.ReviewerLinks, " | "), "docs/reports/review-readiness.md") {
		t.Fatalf("expected bundle reviewer links, got %+v", decoded.BrokerReviewBundle.ReviewerLinks)
	}
}

func TestV2ControlCenterIncludesBrokerReviewBundle(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/control-center?limit=5&audit_limit=5", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		BrokerReviewBundle struct {
			LiveValidationIndexPath string `json:"live_validation_index_path"`
			ReviewReadinessPath     string `json:"review_readiness_path"`
			RuntimePosture          string `json:"runtime_posture"`
		} `json:"broker_review_bundle"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode control center broker review bundle payload: %v", err)
	}
	if decoded.BrokerReviewBundle.LiveValidationIndexPath != "docs/reports/live-validation-index.json" || decoded.BrokerReviewBundle.ReviewReadinessPath != "docs/reports/review-readiness.md" || decoded.BrokerReviewBundle.RuntimePosture != "contract_only" {
		t.Fatalf("unexpected control center broker review bundle payload: %+v", decoded.BrokerReviewBundle)
	}
}

func TestV2DistributedReportIncludesBrokerReviewBundle(t *testing.T) {
	server := &Server{Recorder: observability.NewRecorder(), Queue: queue.NewMemoryQueue(), Bus: events.NewBus(), Control: control.New(), Now: time.Now}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v2/reports/distributed?limit=5", nil)
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		BrokerReviewBundle struct {
			CanonicalSummaryPath string `json:"canonical_summary_path"`
			StubReportPath       string `json:"stub_report_path"`
		} `json:"broker_review_bundle"`
		Report struct {
			Markdown string `json:"markdown"`
		} `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode distributed broker review bundle payload: %v", err)
	}
	if decoded.BrokerReviewBundle.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" || decoded.BrokerReviewBundle.StubReportPath != "docs/reports/broker-failover-stub-report.json" {
		t.Fatalf("unexpected distributed broker review bundle payload: %+v", decoded.BrokerReviewBundle)
	}
	if !strings.Contains(decoded.Report.Markdown, "## Broker Review Bundle") || !strings.Contains(decoded.Report.Markdown, "Review readiness: docs/reports/review-readiness.md") || !strings.Contains(decoded.Report.Markdown, "Operator guide: docs/reports/broker-event-log-adapter-contract.md") || !strings.Contains(decoded.Report.Markdown, "Reviewer bundle links: docs/reports/broker-validation-summary.json") {
		t.Fatalf("expected broker review bundle markdown section, got %s", decoded.Report.Markdown)
	}
}
