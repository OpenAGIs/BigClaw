package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
		{WorkerID: "worker-b", State: "leased", CurrentExecutor: domain.ExecutorKubernetes, SuccessfulRuns: 3, LeaseRenewals: 2, LastResult: "warming", PreemptionActive: true, CurrentPreemptionTaskID: "task-low", CurrentPreemptionWorkerID: "worker-low", LastPreemptedTaskID: "task-low", LastPreemptionAt: time.Unix(1700000100, 0), LastPreemptionReason: "preempted by urgent task task-urgent (priority=1)", PreemptionsIssued: 1},
		{WorkerID: "worker-c", State: "idle", SuccessfulRuns: 8, LeaseRenewals: 0, LastResult: "idle"},
	}
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
	if decoded.EventDurability.RolloutScorecard.ReadyEvidence != 2 || decoded.EventDurability.RolloutScorecard.PartialEvidence != 1 || decoded.EventDurability.RolloutScorecard.BlockedEvidence != 1 {
		t.Fatalf("unexpected rollout scorecard evidence counts: %+v", decoded.EventDurability.RolloutScorecard)
	}
	if len(decoded.EventDurability.RolloutScorecard.Blockers) != 3 {
		t.Fatalf("expected rollout blockers, got %+v", decoded.EventDurability.RolloutScorecard)
	}
	if !strings.Contains(response.Body.String(), "\"event_durability_rollout\"") {
		t.Fatalf("expected rollout scorecard in payload, got %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "\"rollout_ready\":false") {
		t.Fatalf("expected rollout readiness flag in payload, got %s", response.Body.String())
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
	if !strings.Contains(statusResponse.Body.String(), "\"checkpoint_offset\":7") {
		t.Fatalf("expected checkpoint offset in status payload, got %s", statusResponse.Body.String())
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
	for _, key := range []string{"group_id", "subscriber_id", "consumer_id", "previous_consumer_id", "lease_token", "lease_epoch", "checkpoint_offset"} {
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
	if !strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "# BigClaw Distributed Diagnostics Report") || !strings.Contains(decoded.Diagnostics.RolloutReport.Markdown, "Takeover owners") || !strings.Contains(decoded.Diagnostics.RolloutReport.ExportURL, "/v2/reports/distributed/export") {
		t.Fatalf("unexpected rollout report payload: %+v", decoded.Diagnostics.RolloutReport)
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
