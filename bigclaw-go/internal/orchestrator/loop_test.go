package orchestrator

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

func TestLoopQuotaSnapshotUsesLiveQueueState(t *testing.T) {
	ctx := context.Background()
	q := queue.NewMemoryQueue()
	base := time.Unix(1700000000, 0).UTC()
	for _, task := range []domain.Task{
		{ID: "queued", Priority: 3, CreatedAt: base},
		{ID: "leased-nonurgent", Priority: 3, CreatedAt: base.Add(time.Second)},
		{ID: "leased-urgent", Priority: 1, CreatedAt: base.Add(2 * time.Second)},
		{ID: "cancelled", Priority: 4, CreatedAt: base.Add(3 * time.Second), State: domain.TaskCancelled},
	} {
		if err := q.Enqueue(ctx, task); err != nil {
			t.Fatalf("enqueue %s: %v", task.ID, err)
		}
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
