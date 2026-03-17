package events

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBrokerStubEventLogSupportsReplayAndCheckpoints(t *testing.T) {
	log := NewBrokerStubEventLog()
	base := time.Unix(1_700_000_000, 0).UTC()
	for _, event := range []domain.Event{
		{ID: "evt-broker-stub-1", Type: domain.EventTaskQueued, TaskID: "task-broker-stub", TraceID: "trace-a", Timestamp: base},
		{ID: "evt-broker-stub-2", Type: domain.EventTaskStarted, TaskID: "task-broker-stub", TraceID: "trace-a", Timestamp: base.Add(time.Second)},
		{ID: "evt-broker-stub-3", Type: domain.EventTaskCompleted, TaskID: "task-broker-stub-2", TraceID: "trace-b", Timestamp: base.Add(2 * time.Second)},
	} {
		if err := log.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}

	replayed, err := log.ReplayAfter("evt-broker-stub-1", 10)
	if err != nil {
		t.Fatalf("replay after: %v", err)
	}
	if len(replayed) != 2 || replayed[0].ID != "evt-broker-stub-2" || replayed[1].ID != "evt-broker-stub-3" {
		t.Fatalf("unexpected replay after payload: %+v", replayed)
	}

	byTask, err := log.EventsByTask("task-broker-stub", 10)
	if err != nil {
		t.Fatalf("events by task: %v", err)
	}
	if len(byTask) != 2 {
		t.Fatalf("expected two task-filtered events, got %+v", byTask)
	}

	checkpoint, err := log.Acknowledge("subscriber-broker-stub", "evt-broker-stub-2", base.Add(3*time.Second))
	if err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if checkpoint.EventSequence != 2 {
		t.Fatalf("expected checkpoint sequence 2, got %+v", checkpoint)
	}
	stored, err := log.Checkpoint("subscriber-broker-stub")
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if stored.EventID != "evt-broker-stub-2" || stored.EventSequence != 2 {
		t.Fatalf("unexpected stored checkpoint: %+v", stored)
	}

	watermark, err := log.RetentionWatermark()
	if err != nil {
		t.Fatalf("retention watermark: %v", err)
	}
	if watermark.Backend != "broker_stub" || watermark.EventCount != 3 || watermark.NewestSequence != 3 {
		t.Fatalf("unexpected watermark: %+v", watermark)
	}
}

func TestBrokerStubEventLogCapabilitiesAdvertiseStubMode(t *testing.T) {
	capability := NewBrokerStubEventLog().Capabilities()
	if capability.Backend != "broker_stub" || capability.Scope != "process_local_stub" {
		t.Fatalf("unexpected capability header: %+v", capability)
	}
	if !capability.Publish.Supported || !capability.Replay.Supported || !capability.Checkpoint.Supported {
		t.Fatalf("expected publish/replay/checkpoint support, got %+v", capability)
	}
	if capability.Dedup.Supported {
		t.Fatalf("expected dedup to stay unsupported in stub mode, got %+v", capability)
	}
	if capability.Retention.Mode != "process_memory_stub" {
		t.Fatalf("expected stub retention mode, got %+v", capability.Retention)
	}
}
