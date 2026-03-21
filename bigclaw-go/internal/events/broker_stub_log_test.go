package events

import (
	"context"
	"fmt"
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

func TestBrokerStubLiveFanoutStaysIsolatedFromReplayCatchUp(t *testing.T) {
	log := NewBrokerStubEventLog()
	bus := NewBus()
	ctx := context.Background()
	base := time.Unix(1_700_000_100, 0).UTC()
	for index := 0; index < 4; index++ {
		event := domain.Event{
			ID:        fmt.Sprintf("evt-broker-backlog-%d", index+1),
			Type:      domain.EventTaskQueued,
			TaskID:    "task-broker-fanout",
			TraceID:   "trace-broker-fanout",
			Timestamp: base.Add(time.Duration(index) * time.Second),
		}
		if err := log.Write(ctx, event); err != nil {
			t.Fatalf("seed backlog %s: %v", event.ID, err)
		}
	}

	replayed, err := log.Replay(10)
	if err != nil {
		t.Fatalf("replay backlog: %v", err)
	}
	if len(replayed) != 4 {
		t.Fatalf("expected 4 replay backlog events, got %+v", replayed)
	}

	liveCh, cancel := bus.Subscribe(1)
	defer cancel()

	replayDone := make(chan struct{})
	go func() {
		defer close(replayDone)
		for range replayed {
			time.Sleep(30 * time.Millisecond)
		}
	}()

	time.Sleep(10 * time.Millisecond)
	select {
	case <-replayDone:
		t.Fatal("replay catch-up finished before live publish drill began")
	default:
	}

	liveEvent := domain.Event{
		ID:        "evt-broker-live",
		Type:      domain.EventTaskStarted,
		TaskID:    "task-broker-fanout",
		TraceID:   "trace-broker-fanout",
		Timestamp: base.Add(5 * time.Second),
	}
	if err := log.Write(ctx, liveEvent); err != nil {
		t.Fatalf("write live event: %v", err)
	}
	publishedAt := time.Now()
	bus.Publish(liveEvent)

	select {
	case got := <-liveCh:
		if got.ID != liveEvent.ID {
			t.Fatalf("expected live event %s, got %+v", liveEvent.ID, got)
		}
		if got.Delivery == nil || got.Delivery.Mode != domain.EventDeliveryModeLive {
			t.Fatalf("expected live delivery metadata, got %+v", got.Delivery)
		}
		if elapsed := time.Since(publishedAt); elapsed > 50*time.Millisecond {
			t.Fatalf("live fanout delivery exceeded deadline: %s", elapsed)
		}
		select {
		case <-replayDone:
			t.Fatal("replay catch-up drained before live delivery; isolation drill no longer proves separation")
		default:
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timed out waiting for live publish while replay catch-up was active")
	}

	<-replayDone
	watermark, err := log.RetentionWatermark()
	if err != nil {
		t.Fatalf("retention watermark: %v", err)
	}
	if watermark.EventCount != 5 || watermark.NewestEventID != liveEvent.ID {
		t.Fatalf("unexpected watermark after live fanout drill: %+v", watermark)
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
