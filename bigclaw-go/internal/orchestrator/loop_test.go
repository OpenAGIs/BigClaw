package orchestrator

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

type stubRuntime struct {
	runOnce func(context.Context, scheduler.QuotaSnapshot) bool
}

func (s stubRuntime) RunOnce(ctx context.Context, quota scheduler.QuotaSnapshot) bool {
	return s.runOnce(ctx, quota)
}

func TestLoopRunsWorkImmediatelyBeforeFirstTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{}, 1)
	loop := &Loop{
		Runtime: stubRuntime{runOnce: func(ctx context.Context, quota scheduler.QuotaSnapshot) bool {
			if quota.ConcurrentLimit != 3 || quota.BudgetRemaining != 500 {
				t.Fatalf("unexpected quota passed to runtime: %+v", quota)
			}
			select {
			case done <- struct{}{}:
			default:
			}
			cancel()
			return true
		}},
		Quota:        scheduler.QuotaSnapshot{ConcurrentLimit: 3, BudgetRemaining: 500},
		PollInterval: time.Hour,
	}

	go loop.Run(ctx)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected loop to run work immediately")
	}
}

func TestLoopUsesDefaultPollIntervalWhenUnset(t *testing.T) {
	loop := &Loop{}
	if got := loop.pollInterval(); got != 100*time.Millisecond {
		t.Fatalf("expected default poll interval, got %s", got)
	}
}

func TestQueueQuotaSourceCapturesQueueDepthAndRunningCount(t *testing.T) {
	q := queue.NewMemoryQueue()
	ctx := context.Background()
	base := time.Now()
	if err := q.Enqueue(ctx, domain.Task{ID: "queued-1", Priority: 3, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue queued-1: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "queued-2", Priority: 1, CreatedAt: base.Add(time.Second)}); err != nil {
		t.Fatalf("enqueue queued-2: %v", err)
	}
	if _, _, err := q.LeaseNext(ctx, "worker-1", time.Minute); err != nil {
		t.Fatalf("lease first task: %v", err)
	}
	if _, _, err := q.LeaseNext(ctx, "worker-2", time.Minute); err != nil {
		t.Fatalf("lease second task: %v", err)
	}

	source := QueueQuotaSource{Queue: q, Scheduler: scheduler.New()}
	quota := source.Snapshot(ctx, scheduler.QuotaSnapshot{ConcurrentLimit: 5, BudgetRemaining: 900})
	if quota.ConcurrentLimit != 5 || quota.BudgetRemaining != 900 {
		t.Fatalf("expected base quota preserved, got %+v", quota)
	}
	if quota.QueueDepth != 2 {
		t.Fatalf("expected queue depth 2, got %+v", quota)
	}
	if quota.CurrentRunning != 2 {
		t.Fatalf("expected two leased tasks counted as running, got %+v", quota)
	}
	if quota.PreemptibleExecutions != 1 {
		t.Fatalf("expected one lower-priority leased task to count as preemptible, got %+v", quota)
	}
}
