package orchestrator

import (
	"context"
	"testing"
	"time"

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
