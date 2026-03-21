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

func TestBuildEventLogUsesBrokerStubDriver(t *testing.T) {
	defaults := config.Default()
	eventLog, err := buildEventLog(config.Config{
		EventLogBackend:            string(events.EventLogBackendBroker),
		EventLogBrokerDriver:       events.BrokerDriverStub,
		EventLogBrokerURLs:         []string{"stub://broker-a"},
		EventLogBrokerTopic:        "bigclaw.events",
		EventLogPublishTimeout:     defaults.EventLogPublishTimeout,
		EventLogReplayLimit:        defaults.EventLogReplayLimit,
		EventLogCheckpointInterval: defaults.EventLogCheckpointInterval,
	})
	if err != nil {
		t.Fatalf("build event log: %v", err)
	}
	if eventLog == nil || eventLog.Backend() != "broker_stub" {
		t.Fatalf("expected broker stub backend, got %#v", eventLog)
	}
	if capability := eventLog.Capabilities(); capability.Backend != "broker_stub" || capability.Retention.Mode != "process_memory_stub" {
		t.Fatalf("unexpected broker stub capability payload: %+v", capability)
	}
}

func TestBuildEventLogRejectsUnimplementedBrokerDriver(t *testing.T) {
	cfg := config.Default()
	cfg.EventLogBackend = string(events.EventLogBackendBroker)
	cfg.EventLogBrokerDriver = "kafka"
	cfg.EventLogBrokerURLs = []string{"kafka-1:9092"}
	cfg.EventLogBrokerTopic = "bigclaw.events"

	eventLog, err := buildEventLog(cfg)
	if err == nil {
		t.Fatalf("expected unimplemented broker driver error, got event log %#v", eventLog)
	}
}

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
