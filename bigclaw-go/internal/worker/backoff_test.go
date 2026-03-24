package worker

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

func TestBackoffPolicyResolve(t *testing.T) {
	t.Run("fixed", func(t *testing.T) {
		policy := BackoffPolicy{
			Strategy:            FixedBackoffStrategy{},
			SchedulerRetryDelay: 100 * time.Millisecond,
			MaxDelay:            time.Second,
		}
		if got := policy.Resolve(BackoffSchedulerRetry, 3); got != 100*time.Millisecond {
			t.Fatalf("expected fixed 100ms delay, got %s", got)
		}
	})

	t.Run("linear", func(t *testing.T) {
		policy := BackoffPolicy{
			Strategy:             LinearBackoffStrategy{},
			ExecutionRetryDelay:  50 * time.Millisecond,
			PreemptionRetryDelay: 75 * time.Millisecond,
			MaxDelay:             time.Second,
		}
		if got := policy.Resolve(BackoffExecutionRetry, 3); got != 150*time.Millisecond {
			t.Fatalf("expected linear 150ms delay, got %s", got)
		}
		if got := policy.Resolve(BackoffPreemption, 2); got != 150*time.Millisecond {
			t.Fatalf("expected linear 150ms preemption delay, got %s", got)
		}
	})

	t.Run("exponential with cap", func(t *testing.T) {
		policy := BackoffPolicy{
			Strategy:            ExponentialBackoffStrategy{},
			SchedulerRetryDelay: 80 * time.Millisecond,
			MaxDelay:            300 * time.Millisecond,
		}
		if got := policy.Resolve(BackoffSchedulerRetry, 1); got != 80*time.Millisecond {
			t.Fatalf("expected exponential 80ms delay on first attempt, got %s", got)
		}
		if got := policy.Resolve(BackoffSchedulerRetry, 3); got != 300*time.Millisecond {
			t.Fatalf("expected capped 300ms delay on third attempt, got %s", got)
		}
	})
}

func TestRuntimeUsesConfiguredBackoffForRetryScheduling(t *testing.T) {
	q := queue.NewMemoryQueue()
	base := time.Date(2026, time.March, 24, 11, 0, 0, 0, time.UTC)
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-backoff", Priority: 1, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	now := base
	runtime := Runtime{
		WorkerID:  "worker-backoff",
		Queue:     q,
		Scheduler: scheduler.New(),
		Registry: executor.NewRegistry(fakeRunner{
			kind: domain.ExecutorLocal,
			result: executor.Result{
				ShouldRetry: true,
				Message:     "retry me",
				FinishedAt:  base.Add(5 * time.Millisecond),
			},
		}),
		LeaseTTL:    time.Second,
		TaskTimeout: time.Second,
		Backoff: BackoffPolicy{
			Strategy:            LinearBackoffStrategy{},
			ExecutionRetryDelay: 40 * time.Millisecond,
			MaxDelay:            time.Second,
		},
		Now: func() time.Time { return now },
	}

	processed := runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000})
	if !processed {
		t.Fatal("expected task to be processed")
	}

	snapshot, err := q.GetTask(context.Background(), "task-backoff")
	if err != nil {
		t.Fatalf("get task snapshot: %v", err)
	}
	if snapshot.Task.State != domain.TaskQueued {
		t.Fatalf("expected queued task after requeue, got %+v", snapshot)
	}
	if snapshot.Attempt != 1 {
		t.Fatalf("expected first lease attempt to be recorded, got %+v", snapshot)
	}
	expectedAvailableAt := base.Add(40 * time.Millisecond)
	if !snapshot.AvailableAt.Equal(expectedAvailableAt) {
		t.Fatalf("expected available_at %s, got %s", expectedAvailableAt, snapshot.AvailableAt)
	}
}

func BenchmarkWorkerBackoffStrategy(b *testing.B) {
	strategies := []BackoffStrategy{
		FixedBackoffStrategy{},
		LinearBackoffStrategy{},
		ExponentialBackoffStrategy{},
	}
	reasons := []BackoffReason{
		BackoffTakeoverHold,
		BackoffSchedulerRetry,
		BackoffPreemption,
		BackoffExecutionRetry,
	}
	attempts := []int{1, 2, 4, 8}
	for _, strategy := range strategies {
		policy := BackoffPolicy{
			Strategy:             strategy,
			TakeoverHoldDelay:    250 * time.Millisecond,
			SchedulerRetryDelay:  100 * time.Millisecond,
			PreemptionRetryDelay: 100 * time.Millisecond,
			ExecutionRetryDelay:  200 * time.Millisecond,
			MaxDelay:             5 * time.Second,
		}
		b.Run(strategy.Name(), func(b *testing.B) {
			var total time.Duration
			for i := 0; i < b.N; i++ {
				total += policy.Resolve(reasons[i%len(reasons)], attempts[i%len(attempts)])
			}
			if total == 0 {
				b.Fatal("expected non-zero benchmark accumulator")
			}
		})
	}
}
