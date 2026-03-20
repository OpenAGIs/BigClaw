package orchestrator

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type loopTestRunner struct {
	kind domain.ExecutorKind
}

func (runner loopTestRunner) Kind() domain.ExecutorKind { return runner.kind }

func (runner loopTestRunner) Capability() executor.Capability {
	return executor.Capability{Kind: runner.kind, MaxConcurrency: 1, SupportsShell: true}
}

func (runner loopTestRunner) Execute(_ context.Context, _ domain.Task) executor.Result {
	return executor.Result{Success: true, Message: "ok", FinishedAt: time.Unix(1700000002, 0).UTC()}
}

func TestLoopQuotaSnapshotUsesLiveQueueState(t *testing.T) {
	ctx := context.Background()
	q := queue.NewMemoryQueue()
	base := time.Unix(1700000000, 0).UTC()
	for _, task := range []domain.Task{
		{ID: "queued", Priority: 3, CreatedAt: base},
		{ID: "leased-nonurgent", Priority: 3, CreatedAt: base.Add(time.Second)},
		{ID: "leased-urgent", Priority: 1, CreatedAt: base.Add(2 * time.Second)},
		{ID: "blocked", Priority: 2, CreatedAt: base.Add(3 * time.Second), State: domain.TaskBlocked},
		{ID: "cancelled", Priority: 4, CreatedAt: base.Add(4 * time.Second), State: domain.TaskCancelled},
	} {
		if err := q.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue %s: %v", task.ID, err)
		}
	}
	if _, err := q.UpdateTaskState(ctx, "blocked", domain.TaskBlocked, base.Add(3*time.Second), "human review"); err != nil {
		t.Fatalf("block task: %v", err)
	}
	if _, _, err := q.LeaseNext(ctx, "worker-a", time.Minute); err != nil {
		t.Fatalf("lease first task: %v", err)
	}
	if _, _, err := q.LeaseNext(ctx, "worker-b", time.Minute); err != nil {
		t.Fatalf("lease second task: %v", err)
	}
	cancelled, err := q.CancelTask(ctx, "cancelled", "operator stop")
	if err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if cancelled.Task.State != domain.TaskCancelled {
		t.Fatalf("expected cancelled snapshot, got %+v", cancelled)
	}

	loop := &Loop{
		Runtime: &worker.Runtime{
			Queue:     q,
			Scheduler: scheduler.New(),
		},
		Quota: scheduler.QuotaSnapshot{
			ConcurrentLimit: 8,
			BudgetRemaining: 900,
			MaxQueueDepth:   12,
		},
	}

	quota := loop.quotaSnapshot(ctx)
	if quota.ConcurrentLimit != 8 || quota.BudgetRemaining != 900 || quota.MaxQueueDepth != 12 {
		t.Fatalf("expected base quota preserved, got %+v", quota)
	}
	if quota.QueueDepth != 3 {
		t.Fatalf("expected live queue depth 3, got %+v", quota)
	}
	if quota.CurrentRunning != 2 {
		t.Fatalf("expected two live leased tasks, got %+v", quota)
	}
	if quota.PreemptibleExecutions != 1 {
		t.Fatalf("expected one non-urgent leased task to be preemptible, got %+v", quota)
	}
}

func TestLoopRunTickDrainsRunnableQueueWork(t *testing.T) {
	ctx := context.Background()
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	for _, task := range []domain.Task{
		{ID: "task-1", TraceID: "trace-1", Priority: 3, CreatedAt: time.Unix(1700000000, 0).UTC()},
		{ID: "task-2", TraceID: "trace-2", Priority: 2, CreatedAt: time.Unix(1700000001, 0).UTC()},
	} {
		if err := q.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue %s: %v", task.ID, err)
		}
	}

	loop := &Loop{
		Runtime: &worker.Runtime{
			WorkerID:    "worker-loop",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    executor.NewRegistry(loopTestRunner{kind: domain.ExecutorLocal}),
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
		Quota: scheduler.QuotaSnapshot{
			ConcurrentLimit: 4,
			BudgetRemaining: 1000,
			MaxQueueDepth:   10,
		},
	}

	processed := loop.RunTick(ctx)
	if processed != 2 {
		t.Fatalf("expected tick to drain both queued tasks, got %d", processed)
	}
	if got := q.Size(ctx); got != 0 {
		t.Fatalf("expected drained queue, got actionable size %d", got)
	}
	for _, taskID := range []string{"task-1", "task-2"} {
		task, ok := recorder.Task(taskID)
		if !ok || task.State != domain.TaskSucceeded {
			t.Fatalf("expected succeeded task snapshot for %s, got %+v ok=%v", taskID, task, ok)
		}
	}
}

func TestLoopRunTickStopsAfterDeferredRequeue(t *testing.T) {
	ctx := context.Background()
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	if err := q.Enqueue(ctx, domain.Task{
		ID:          "budget-blocked",
		TraceID:     "trace-budget",
		Priority:    3,
		BudgetCents: 500,
		CreatedAt:   time.Unix(1700000010, 0).UTC(),
	}); err != nil {
		t.Fatalf("enqueue blocked task: %v", err)
	}

	loop := &Loop{
		Runtime: &worker.Runtime{
			WorkerID:    "worker-loop",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    executor.NewRegistry(loopTestRunner{kind: domain.ExecutorLocal}),
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
		Quota: scheduler.QuotaSnapshot{
			ConcurrentLimit: 4,
			BudgetRemaining: 100,
		},
	}

	processed := loop.RunTick(ctx)
	if processed != 1 {
		t.Fatalf("expected single deferred processing attempt, got %d", processed)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected deferred task to remain queued, got actionable size %d", got)
	}
	snapshot, err := q.GetTask(ctx, "budget-blocked")
	if err != nil {
		t.Fatalf("get deferred task: %v", err)
	}
	if snapshot.Task.State != domain.TaskQueued || snapshot.Leased {
		t.Fatalf("expected task requeued and unleased, got %+v", snapshot)
	}
}
