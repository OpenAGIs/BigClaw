package main

import (
	"fmt"
	"testing"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/control"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

func TestBuildWorkerPoolUsesMaxConcurrentRuns(t *testing.T) {
	cfg := config.Default()
	cfg.MaxConcurrentRuns = 3

	pool := buildWorkerPool(
		cfg,
		queue.NewMemoryQueue(),
		scheduler.New(),
		executor.NewRegistry(executor.LocalRunner{}),
		events.NewBus(),
		observability.NewRecorder(),
		control.New(),
	)

	snapshots := pool.Snapshots()
	if len(snapshots) != 3 {
		t.Fatalf("expected 3 workers, got %d", len(snapshots))
	}
	for index, snapshot := range snapshots {
		expectedID := fmt.Sprintf("worker-%d", index+1)
		if snapshot.WorkerID != expectedID {
			t.Fatalf("expected worker id %s, got %+v", expectedID, snapshot)
		}
	}
}

func TestBuildWorkerPoolClampsToAtLeastOneWorker(t *testing.T) {
	cfg := config.Default()
	cfg.MaxConcurrentRuns = 0

	pool := buildWorkerPool(
		cfg,
		queue.NewMemoryQueue(),
		scheduler.New(),
		executor.NewRegistry(executor.LocalRunner{}),
		events.NewBus(),
		observability.NewRecorder(),
		control.New(),
	)

	if len(pool.Snapshots()) != 1 {
		t.Fatalf("expected a single fallback worker, got %d", len(pool.Snapshots()))
	}
}
