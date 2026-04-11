package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"

	"bigclaw-go/internal/scheduler"
)

type fakeRuntime struct {
	mu       sync.Mutex
	results  []bool
	calls    int
	calledCh chan int
}

func (r *fakeRuntime) RunOnce(_ context.Context, _ scheduler.QuotaSnapshot) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.calledCh != nil {
		r.calledCh <- r.calls
	}
	index := r.calls - 1
	if index >= len(r.results) {
		return false
	}
	return r.results[index]
}

func TestLoopRunsImmediatelyBeforeFirstTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime := &fakeRuntime{
		results:  []bool{false},
		calledCh: make(chan int, 1),
	}
	loop := &Loop{Runtime: runtime, PollInterval: time.Hour}

	done := make(chan struct{})
	go func() {
		defer close(done)
		loop.Run(ctx)
	}()

	select {
	case <-runtime.calledCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected runtime to run before the first poll tick")
	}

	cancel()
	<-done
}

func TestLoopDrainsAvailableWorkWithinSingleCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime := &fakeRuntime{
		results:  []bool{true, true, false},
		calledCh: make(chan int, 3),
	}
	loop := &Loop{Runtime: runtime, PollInterval: time.Hour}

	done := make(chan struct{})
	go func() {
		defer close(done)
		loop.Run(ctx)
	}()

	for want := 1; want <= 3; want++ {
		select {
		case got := <-runtime.calledCh:
			if got != want {
				t.Fatalf("expected call %d, got %d", want, got)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("expected call %d during initial drain cycle", want)
		}
	}

	cancel()
	<-done
}

func TestLoopUsesDefaultPollIntervalWhenUnset(t *testing.T) {
	loop := &Loop{}
	if got := loop.pollInterval(); got != 100*time.Millisecond {
		t.Fatalf("expected default poll interval, got %s", got)
	}
}
