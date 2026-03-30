package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
	"bigclaw-go/internal/workflow"
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

type leaseLossQueue struct {
	queue.Queue
	mu       sync.Mutex
	lostOnce bool
}

func (q *leaseLossQueue) RenewLease(ctx context.Context, lease *queue.Lease, ttl time.Duration) error {
	q.mu.Lock()
	shouldLose := !q.lostOnce
	if shouldLose {
		q.lostOnce = true
	}
	q.mu.Unlock()
	if shouldLose {
		return queue.ErrLeaseNotOwned
	}
	return q.Queue.RenewLease(ctx, lease, ttl)
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
	task, ok := recorder.Task("task-1")
	if !ok || task.State != domain.TaskSucceeded {
		t.Fatalf("expected succeeded task snapshot, got %+v ok=%v", task, ok)
	}
}

func TestRuntimeSkipsCleanlyWhenQueueUnset(t *testing.T) {
	runtime := Runtime{WorkerID: "worker-no-queue"}
	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if processed {
		t.Fatal("expected runtime to skip when queue is unset")
	}
	snapshot := runtime.Snapshot()
	if snapshot.State != "idle" || snapshot.WorkerID != "worker-no-queue" {
		t.Fatalf("expected idle snapshot when queue is unset, got %+v", snapshot)
	}
}

func TestRuntimeUsesDefaultTimeoutAndLeaseWhenUnset(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-defaults", TraceID: "trace-defaults", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()

	runtime := Runtime{
		WorkerID:  "worker-defaults",
		Queue:     q,
		Scheduler: scheduler.New(),
		Registry: executor.NewRegistry(fakeRunner{
			kind:  domain.ExecutorLocal,
			delay: 20 * time.Millisecond,
			result: executor.Result{
				Success: true,
				Message: "ok",
			},
		}),
		Bus:      bus,
		Recorder: recorder,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatal("expected runtime to process task with default timeout and lease settings")
	}
	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after success, got %d", got)
	}
	latest, ok := recorder.LatestByTask("task-defaults")
	if !ok || latest.Type != domain.EventTaskCompleted {
		t.Fatalf("expected completed event with default runtime settings, got %+v", latest)
	}
	snapshot := runtime.Snapshot()
	if snapshot.SuccessfulRuns != 1 || snapshot.State != "idle" {
		t.Fatalf("expected successful idle snapshot, got %+v", snapshot)
	}
}

func TestRuntimeDefaultsNilSchedulerAndRegistryToSafeBehavior(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-nil-runtime", TraceID: "trace-nil-runtime", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()

	runtime := Runtime{
		WorkerID: "worker-nil-runtime",
		Queue:    q,
		Bus:      bus,
		Recorder: recorder,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatal("expected runtime to process task even when scheduler and registry are unset")
	}
	deadLetters, err := q.ListDeadLetters(context.Background(), 10)
	if err != nil {
		t.Fatalf("list dead letters: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-nil-runtime" {
		t.Fatalf("expected task to dead-letter cleanly, got %+v", deadLetters)
	}
	latest, ok := recorder.LatestByTask("task-nil-runtime")
	if !ok || latest.Type != domain.EventTaskDeadLetter {
		t.Fatalf("expected dead-letter event from safe defaults, got %+v", latest)
	}
	snapshot := runtime.Snapshot()
	if snapshot.DeadLetterRuns != 1 || snapshot.State != "idle" {
		t.Fatalf("expected idle dead-letter snapshot, got %+v", snapshot)
	}
}

func TestRuntimePublishesExecutorArtifacts(t *testing.T) {
	q := queue.NewMemoryQueue()
	if err := q.Enqueue(context.Background(), domain.Task{
		ID:               "task-artifacts",
		TraceID:          "trace-artifacts",
		Priority:         1,
		RequiredExecutor: domain.ExecutorLocal,
		RequiredTools:    []string{"browser", "git"},
		ValidationPlan:   []string{"capture trace"},
		Metadata: map[string]string{
			"workflow_definition": `{
				"name":"release-closeout",
				"report_path_template":"reports/{task_id}/{run_id}.md",
				"journal_path_template":"journals/{workflow}/{run_id}.json",
				"validation_evidence":["capture trace","download bundle"],
				"approvals":["ops-review"]
			}`,
		},
		CreatedAt: time.Now(),
	}); err != nil {
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
	if len(events) != 5 {
		t.Fatalf("expected 5 lifecycle events including acceptance annotation, got %d", len(events))
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
	if completed.RunID == "" {
		t.Fatalf("expected run_id on completed event, got %+v", completed)
	}
	if completed.Payload["run_id"] != completed.RunID {
		t.Fatalf("expected run_id in payload, got %+v", completed.Payload)
	}
	artifacts, ok := completed.Payload["artifacts"].([]string)
	if !ok || len(artifacts) != 2 {
		t.Fatalf("expected artifact list in completed payload, got %+v", completed.Payload)
	}
	if completed.Payload["executor"] != domain.ExecutorLocal {
		t.Fatalf("expected executor in completed payload, got %+v", completed.Payload)
	}
	if completed.Payload["report_path"] != "reports/task-artifacts/"+completed.RunID+".md" {
		t.Fatalf("expected report path in completed payload, got %+v", completed.Payload)
	}
	if completed.Payload["journal_path"] != "journals/release-closeout/"+completed.RunID+".json" {
		t.Fatalf("expected journal path in completed payload, got %+v", completed.Payload)
	}
	if validation, ok := completed.Payload["validation_evidence"].([]string); !ok || len(validation) != 2 || validation[0] != "capture trace" || validation[1] != "download bundle" {
		t.Fatalf("expected validation evidence in completed payload, got %+v", completed.Payload)
	}
	if approvals, ok := completed.Payload["required_approvals"].([]string); !ok || len(approvals) != 1 || approvals[0] != "ops-review" {
		t.Fatalf("expected approvals in completed payload, got %+v", completed.Payload)
	}
	runCloseout, ok := completed.Payload["workflow_run"].(workflow.WorkflowRun)
	if !ok {
		t.Fatalf("expected workflow_run in completed payload, got %+v", completed.Payload)
	}
	if runCloseout.RunID != completed.RunID || runCloseout.Status != workflow.WorkflowRunSucceeded {
		t.Fatalf("expected succeeded workflow closeout, got %+v", runCloseout)
	}
	annotated := events[4]
	if annotated.Type != domain.EventRunAnnotated {
		t.Fatalf("expected acceptance annotation event, got %+v", annotated)
	}
	if annotated.RunID != completed.RunID {
		t.Fatalf("expected acceptance annotation to share run_id, got %+v", annotated)
	}
}

func TestRuntimePublishesOrchestrationAssessmentOnRoutedEvent(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:            "task-assessment",
		TraceID:       "trace-assessment",
		Source:        "linear",
		Title:         "Customer analytics rollout",
		Description:   "Need customer stakeholder rollout and analytics validation.",
		Labels:        []string{"customer", "analytics"},
		RequiredTools: []string{"browser", "sql"},
		CreatedAt:     time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorKubernetes, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	events := recorder.EventsByTask("task-assessment", 10)
	if len(events) != 5 {
		t.Fatalf("expected 5 lifecycle events with handoff, got %d", len(events))
	}
	routed := events[1]
	if routed.Type != domain.EventSchedulerRouted {
		t.Fatalf("expected routed event, got %+v", routed)
	}
	orchestration, ok := routed.Payload["orchestration"].(map[string]any)
	if !ok || orchestration["collaboration_mode"] != "tier-limited" {
		t.Fatalf("expected orchestration payload on routed event, got %+v", routed.Payload)
	}
	policy, ok := routed.Payload["policy"].(map[string]any)
	if !ok || policy["upgrade_required"] != true || policy["tier"] != "standard" || policy["entitlement_status"] != "upgrade-required" || policy["billing_model"] != "standard-blocked" {
		t.Fatalf("expected policy payload on routed event, got %+v", routed.Payload)
	}
	handoff := events[2]
	if handoff.Type != domain.EventRunTakeover || handoff.Payload["target_team"] != "operations" || handoff.Payload["handoff_status"] != "blocked" {
		t.Fatalf("expected operations handoff event after routed event, got %+v", handoff)
	}
}

func TestRuntimePublishesRejectedDecisionHandoffBeforeRetry(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:            "task-assessment-reject",
		TraceID:       "trace-assessment-reject",
		RiskLevel:     domain.RiskHigh,
		Labels:        []string{"security"},
		RequiredTools: []string{"deploy"},
		BudgetCents:   500,
		CreatedAt:     time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
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

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 100})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	events := recorder.EventsByTask("task-assessment-reject", 10)
	if len(events) != 3 {
		t.Fatalf("expected leased, handoff, retry events, got %+v", events)
	}
	if events[1].Type != domain.EventRunTakeover || events[1].Payload["target_team"] != "security" {
		t.Fatalf("expected security handoff event before retry, got %+v", events[1])
	}
	if events[2].Type != domain.EventTaskRetried {
		t.Fatalf("expected retry event after handoff, got %+v", events[2])
	}
}

func TestRuntimeWritesWorkpadJournalAndStoresArtifactPath(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:            "task-workpad",
		TraceID:       "trace-workpad",
		Source:        "linear",
		RequiredTools: []string{"git"},
		CreatedAt:     time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	dir := t.TempDir()
	runtime := Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
		WorkpadDir:  dir,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	stored, ok := recorder.Task("task-workpad")
	if !ok {
		t.Fatal("expected stored task snapshot")
	}
	workpadPath := stored.Metadata["workpad"]
	if workpadPath == "" {
		t.Fatalf("expected workpad path in task metadata, got %+v", stored.Metadata)
	}
	expectedPath := filepath.Join(dir, "task-workpad", "attempt-1.json")
	if workpadPath != expectedPath {
		t.Fatalf("expected workpad path %q, got %q", expectedPath, workpadPath)
	}
	body, err := os.ReadFile(workpadPath)
	if err != nil {
		t.Fatalf("read workpad journal: %v", err)
	}
	var journal workflow.WorkpadJournal
	if err := json.Unmarshal(body, &journal); err != nil {
		t.Fatalf("unmarshal workpad journal: %v", err)
	}
	if journal.TaskID != "task-workpad" || journal.RunID != "task-workpad-attempt-1" {
		t.Fatalf("unexpected journal identity: %+v", journal)
	}
	if len(journal.Entries) != 4 {
		t.Fatalf("expected intake, scheduler, execution started, execution completed entries, got %+v", journal.Entries)
	}
	if journal.Entries[0].Step != "intake" || journal.Entries[1].Step != "scheduler" || journal.Entries[2].Status != "started" || journal.Entries[3].Status != "completed" {
		t.Fatalf("unexpected journal entries: %+v", journal.Entries)
	}
}

func TestRuntimeAnnotatesAcceptanceOutcomeFromTaskMetadata(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:                 "task-acceptance",
		TraceID:            "trace-acceptance",
		AcceptanceCriteria: []string{"ship report"},
		ValidationPlan:     []string{"go test ./internal/worker"},
		Metadata: map[string]string{
			"validation_evidence": `["ship report","go test ./internal/worker"]`,
			"approvals":           `["ops-review"]`,
		},
		CreatedAt: time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
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
	stored, ok := recorder.Task("task-acceptance")
	if !ok {
		t.Fatal("expected stored task snapshot")
	}
	if stored.Metadata["acceptance_status"] != "accepted" || stored.Metadata["approval_status"] != "accepted" {
		t.Fatalf("expected accepted metadata, got %+v", stored.Metadata)
	}
	events := recorder.EventsByTask("task-acceptance", 10)
	if len(events) != 5 {
		t.Fatalf("expected acceptance annotation event, got %+v", events)
	}
	annotated := events[4]
	if annotated.Type != domain.EventRunAnnotated || annotated.Payload["acceptance_status"] != "accepted" || annotated.Payload["summary"] != "acceptance criteria and validation plan satisfied" {
		t.Fatalf("expected acceptance annotation payload, got %+v", annotated)
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

func TestRuntimeCancelsExecutionWhenLeaseIsLost(t *testing.T) {
	baseQueue := queue.NewMemoryQueue()
	if err := baseQueue.Enqueue(context.Background(), domain.Task{ID: "task-lease-lost", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	leaseLossQueue := &leaseLossQueue{Queue: baseQueue}

	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:  "worker-1",
		Queue:     leaseLossQueue,
		Scheduler: scheduler.New(),
		Registry: executor.NewRegistry(fakeRunner{
			kind:              domain.ExecutorLocal,
			blockUntilContext: true,
		}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    30 * time.Millisecond,
		TaskTimeout: 2 * time.Second,
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatalf("expected task to be processed")
	}
	// Renewals should fail once the lease TTL elapses; the runtime should treat this as a lease-loss retry.
	snapshot := runtime.Snapshot()
	if snapshot.LeaseRenewalFailures == 0 {
		t.Fatalf("expected lease renewal failures to be recorded, got %+v", snapshot)
	}
	if snapshot.LeaseLostRuns != 1 || snapshot.LastTransition != "lease.lost" {
		t.Fatalf("expected lease-lost transition, got %+v", snapshot)
	}
	if got := baseQueue.Size(context.Background()); got != 1 {
		t.Fatalf("expected task to be requeued after lease loss, got %d", got)
	}
	latest, ok := recorder.LatestByTask("task-lease-lost")
	if !ok || latest.Type != domain.EventTaskRetried {
		t.Fatalf("expected retried event, got %+v ok=%v", latest, ok)
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
	task, ok := recorder.Task("task-timeout")
	if !ok || task.State != domain.TaskRetrying {
		t.Fatalf("expected retrying task snapshot, got %+v ok=%v", task, ok)
	}
	if latest.TraceID != "task-timeout" {
		t.Fatalf("expected default trace id to match task id, got %+v", latest)
	}
	if latest.RunID == "" || latest.Payload["run_id"] != latest.RunID {
		t.Fatalf("expected retry run_id propagation, got %+v", latest)
	}
	if latest.Payload["retry_scheduled"] != true {
		t.Fatalf("expected retry_scheduled payload, got %+v", latest.Payload)
	}
	runCloseout, ok := latest.Payload["workflow_run"].(workflow.WorkflowRun)
	if !ok || runCloseout.Status != workflow.WorkflowRunFailed {
		t.Fatalf("expected failed workflow closeout for retry, got %+v", latest.Payload)
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
	task, ok := recorder.Task("task-missing")
	if !ok || task.State != domain.TaskDeadLetter {
		t.Fatalf("expected dead-letter task snapshot, got %+v ok=%v", task, ok)
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
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, blockUntilContext: true}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    80 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = q.CancelTask(context.Background(), "task-live-cancel", "manual stop")
	}()

	done := make(chan bool, 1)
	go func() {
		done <- runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	}()

	select {
	case processed := <-done:
		if !processed {
			t.Fatal("expected task to be processed")
		}
	case <-time.After(time.Second):
		t.Fatal("expected in-flight cancel to interrupt the running task")
	}
	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after in-flight cancel, got %d", got)
	}
	latest, ok := recorder.LatestByTask("task-live-cancel")
	if !ok || latest.Type != domain.EventTaskCancelled {
		t.Fatalf("expected latest cancel event, got %+v", latest)
	}
	if latest.RunID == "" || latest.Payload["run_id"] != latest.RunID {
		t.Fatalf("expected cancel run_id propagation, got %+v", latest)
	}
	runCloseout, ok := latest.Payload["workflow_run"].(workflow.WorkflowRun)
	if !ok || runCloseout.Status != workflow.WorkflowRunCanceled {
		t.Fatalf("expected canceled workflow closeout, got %+v", latest.Payload)
	}
	snapshot := runtime.Snapshot()
	if snapshot.CancelledRuns != 1 || snapshot.LastTransition != string(domain.EventTaskCancelled) {
		t.Fatalf("expected cancelled runtime snapshot, got %+v", snapshot)
	}
}

func TestRuntimeUrgentTaskPreemptsLowerPriorityRun(t *testing.T) {
	q := queue.NewMemoryQueue()
	base := time.Now()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-low", TraceID: "trace-low", Priority: 5, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue low priority: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	lowRuntime := Runtime{
		WorkerID:    "worker-low",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, blockUntilContext: true}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    80 * time.Millisecond,
		TaskTimeout: 2 * time.Second,
	}
	urgentRuntime := Runtime{
		WorkerID:    "worker-urgent",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "urgent ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    80 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	lowDone := make(chan bool, 1)
	go func() {
		lowDone <- lowRuntime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 1, BudgetRemaining: 1000})
	}()

	deadline := time.Now().Add(time.Second)
	for {
		if state := lowRuntime.Snapshot().State; state == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected low-priority runtime to start running")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-urgent", TraceID: "trace-urgent", Priority: 1, CreatedAt: base.Add(time.Second)}); err != nil {
		t.Fatalf("enqueue urgent task: %v", err)
	}
	processed := urgentRuntime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 1, CurrentRunning: 1, BudgetRemaining: 1000, PreemptibleExecutions: 1})
	if !processed {
		t.Fatal("expected urgent runtime to process task")
	}

	select {
	case processed := <-lowDone:
		if !processed {
			t.Fatal("expected low-priority runtime to finish after preemption")
		}
	case <-time.After(time.Second):
		t.Fatal("expected low-priority runtime to be cancelled by preemption")
	}

	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after urgent preemption, got %d", got)
	}
	lowEvents := recorder.EventsByTask("task-low", 10)
	foundPreempted := false
	for _, event := range lowEvents {
		if event.Type == domain.EventTaskPreempted {
			foundPreempted = true
			break
		}
	}
	if !foundPreempted {
		t.Fatalf("expected preemption event for low-priority task, got %+v", lowEvents)
	}
	lowLatest, ok := recorder.LatestByTask("task-low")
	if !ok || lowLatest.Type != domain.EventTaskCancelled {
		t.Fatalf("expected low-priority task latest event to be cancelled, got %+v", lowLatest)
	}
	urgentLatest, ok := recorder.LatestByTask("task-urgent")
	if !ok || urgentLatest.Type != domain.EventTaskCompleted {
		t.Fatalf("expected urgent task to complete, got %+v", urgentLatest)
	}
	snapshot := urgentRuntime.Snapshot()
	if snapshot.PreemptionsIssued != 1 || snapshot.LastPreemptedTaskID != "task-low" || snapshot.LastPreemptionReason == "" {
		t.Fatalf("expected urgent runtime preemption status, got %+v", snapshot)
	}
}
