package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

type fakeRunner struct {
	kind              domain.ExecutorKind
	delay             time.Duration
	blockUntilContext bool
	result            executor.Result
}

func (r fakeRunner) Kind() domain.ExecutorKind { return r.kind }

func (r fakeRunner) Capability() executor.Capability {
	return executor.Capability{Kind: r.kind, MaxConcurrency: 1, SupportsShell: true}
}

func (r fakeRunner) Execute(ctx context.Context, task domain.Task) executor.Result {
	if r.blockUntilContext {
		<-ctx.Done()
		return executor.Result{ShouldRetry: true, Message: ctx.Err().Error(), FinishedAt: time.Now()}
	}
	if r.delay > 0 {
		select {
		case <-ctx.Done():
			return executor.Result{ShouldRetry: true, Message: ctx.Err().Error(), FinishedAt: time.Now()}
		case <-time.After(r.delay):
		}
	}
	result := r.result
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now()
	}
	return result
}

type renewSpyQueue struct {
	queue.Queue
	mu         sync.Mutex
	renewCalls int
}

func (q *renewSpyQueue) RenewLease(ctx context.Context, lease *queue.Lease, ttl time.Duration) error {
	q.mu.Lock()
	q.renewCalls++
	q.mu.Unlock()
	return q.Queue.RenewLease(ctx, lease, ttl)
}

func (q *renewSpyQueue) RenewCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.renewCalls
}

func newRuntimeRecorder() (*events.Bus, *observability.Recorder) {
	bus := events.NewBus()
	recorder := observability.NewRecorder()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	return bus, recorder
}

func TestRuntimeProcessesTask(t *testing.T) {
	q := queue.NewMemoryQueue()
	_ = q.Enqueue(context.Background(), domain.Task{ID: "task-1", TraceID: "trace-task-1", Priority: 1, CreatedAt: time.Now()})
	bus, recorder := newRuntimeRecorder()

	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after ack, got %d", got)
	}
	events := recorder.EventsByTask("task-1", 10)
	if len(events) != 4 {
		t.Fatalf("expected 4 lifecycle events, got %d", len(events))
	}
	if events[1].Type != domain.EventSchedulerRouted {
		t.Fatalf("expected routed event, got %+v", events)
	}
	for _, event := range events {
		if event.TraceID != "trace-task-1" {
			t.Fatalf("expected trace propagation, got %+v", event)
		}
	}
}

func TestRuntimePublishesExecutorArtifacts(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-artifacts", TraceID: "trace-artifacts", Priority: 1, RequiredExecutor: domain.ExecutorLocal, RequiredTools: []string{"browser", "git"}, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok", Artifacts: []string{"k8s://jobs/default/task-artifacts", "https://docs.example.com/reports/task-artifacts.md"}}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	events := recorder.EventsByTask("task-artifacts", 10)
	if len(events) != 4 {
		t.Fatalf("expected 4 lifecycle events, got %d", len(events))
	}
	started := events[2]
	if started.Type != domain.EventTaskStarted {
		t.Fatalf("expected started event, got %+v", started)
	}
	tools, ok := started.Payload["required_tools"].([]string)
	if !ok || len(tools) != 2 || tools[0] != "browser" || tools[1] != "git" {
		t.Fatalf("expected required tools in started payload, got %+v", started.Payload)
	}
	completed := events[3]
	if completed.Type != domain.EventTaskCompleted {
		t.Fatalf("expected completed event, got %+v", completed)
	}
	artifacts, ok := completed.Payload["artifacts"].([]string)
	if !ok || len(artifacts) != 2 {
		t.Fatalf("expected artifact list in completed payload, got %+v", completed.Payload)
	}
	if completed.Payload["executor"] != domain.ExecutorLocal {
		t.Fatalf("expected executor in completed payload, got %+v", completed.Payload)
	}
}

func TestRuntimeRenewsLeaseDuringExecution(t *testing.T) {
	baseQueue := queue.NewMemoryQueue()
	if err := baseQueue.Enqueue(context.Background(), domain.Task{ID: "task-renew", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	spyQueue := &renewSpyQueue{Queue: baseQueue}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       spyQueue,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, delay: 120 * time.Millisecond, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    40 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	if spyQueue.RenewCount() == 0 {
		t.Fatal("expected lease renewals during execution")
	}
	snapshot := runtime.Snapshot()
	if snapshot.LeaseRenewals == 0 {
		t.Fatalf("expected snapshot lease renewals, got %+v", snapshot)
	}
	if snapshot.SuccessfulRuns != 1 || snapshot.State != "idle" {
		t.Fatalf("expected successful idle snapshot, got %+v", snapshot)
	}
}

func TestRuntimeTimeoutRequeuesTask(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-timeout", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, blockUntilContext: true}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: 30 * time.Millisecond,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected queue size 1 after timeout requeue, got %d", got)
	}
	time.Sleep(250 * time.Millisecond)
	requeuedTask, requeuedLease, err := q.LeaseNext(context.Background(), "worker-2", time.Second)
	if err != nil || requeuedTask == nil || requeuedLease == nil {
		t.Fatalf("lease requeued task: %v task=%v lease=%v", err, requeuedTask, requeuedLease)
	}
	if requeuedTask.ID != "task-timeout" {
		t.Fatalf("expected task-timeout, got %s", requeuedTask.ID)
	}
	latest, ok := recorder.LatestByTask("task-timeout")
	if !ok || latest.Type != domain.EventTaskRetried {
		t.Fatalf("expected latest event retried, got %+v", latest)
	}
	if latest.TraceID != "task-timeout" {
		t.Fatalf("expected default trace id to match task id, got %+v", latest)
	}
	snapshot := runtime.Snapshot()
	if snapshot.RetriedRuns != 1 || snapshot.State != "idle" {
		t.Fatalf("expected retry snapshot after timeout, got %+v", snapshot)
	}
}

func TestRuntimeDeadLettersWhenExecutorMissing(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-missing", Priority: 1, CreatedAt: time.Now(), RequiredExecutor: domain.ExecutorLocal}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	deadLetters, err := q.ListDeadLetters(context.Background(), 10)
	if err != nil {
		t.Fatalf("list dead letters: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-missing" {
		t.Fatalf("unexpected dead letters: %+v", deadLetters)
	}
	latest, ok := recorder.LatestByTask("task-missing")
	if !ok || latest.Type != domain.EventTaskDeadLetter {
		t.Fatalf("expected latest dead-letter event, got %+v", latest)
	}
	events := recorder.EventsByTask("task-missing", 10)
	if len(events) < 3 || events[1].Type != domain.EventSchedulerRouted {
		t.Fatalf("expected routed event before dead-letter, got %+v", events)
	}
	snapshot := runtime.Snapshot()
	if snapshot.DeadLetterRuns != 1 || snapshot.State != "idle" {
		t.Fatalf("expected dead-letter snapshot, got %+v", snapshot)
	}
}

func TestRuntimeCancellationSnapshot(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-cancel", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, blockUntilContext: true}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    time.Second,
		TaskTimeout: 3 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	processed := runtime.RunOnce(ctx, scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	snapshot := runtime.Snapshot()
	if snapshot.CancelledRuns != 1 || snapshot.LastTransition != "context.cancelled" {
		t.Fatalf("expected cancellation snapshot, got %+v", snapshot)
	}
	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected requeued task after cancellation, got %d", got)
	}
}

func TestRuntimeSkipsWhenControlPaused(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-paused", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	controller := control.New()
	controller.Pause("ops", "maintenance", time.Now())
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		Control:     controller,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if processed {
		t.Fatal("expected no processing while control plane is paused")
	}
	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected queue size 1 while paused, got %d", got)
	}
	snapshot := runtime.Snapshot()
	if snapshot.State != "paused" || snapshot.LastTransition != string(domain.EventControlPaused) {
		t.Fatalf("expected paused snapshot, got %+v", snapshot)
	}
}

func TestRuntimeDefersTakenOverTask(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-takeover", TraceID: "trace-takeover", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	controller := control.New()
	controller.Takeover("task-takeover", "alice", "bob", "manual review", time.Now())
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		Control:     controller,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatal("expected runtime to requeue taken-over task")
	}
	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected queue size 1 after takeover deferral, got %d", got)
	}
	latest, ok := recorder.LatestByTask("task-takeover")
	if !ok || latest.Type != domain.EventRunAnnotated {
		t.Fatalf("expected takeover audit event, got %+v", latest)
	}
	task, ok := recorder.Task("task-takeover")
	if !ok || task.State != domain.TaskBlocked {
		t.Fatalf("expected blocked task snapshot, got %+v ok=%v", task, ok)
	}
	snapshot := runtime.Snapshot()
	if snapshot.LastTransition != string(domain.EventRunTakeover) {
		t.Fatalf("expected takeover transition in snapshot, got %+v", snapshot)
	}
}

func TestRuntimeCompletesCancelledInFlightTaskAsCancelled(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-live-cancel", TraceID: "trace-live-cancel", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, delay: 120 * time.Millisecond, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = q.CancelTask(context.Background(), "task-live-cancel", "manual stop")
	}()

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatal("expected task to be processed")
	}
	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after in-flight cancel, got %d", got)
	}
	latest, ok := recorder.LatestByTask("task-live-cancel")
	if !ok || latest.Type != domain.EventTaskCancelled {
		t.Fatalf("expected latest cancel event, got %+v", latest)
	}
	snapshot := runtime.Snapshot()
	if snapshot.CancelledRuns != 1 || snapshot.LastTransition != string(domain.EventTaskCancelled) {
		t.Fatalf("expected cancelled runtime snapshot, got %+v", snapshot)
	}
}
