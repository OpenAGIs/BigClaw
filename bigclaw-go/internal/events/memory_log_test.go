package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestMemoryLogReplayAndLiveSubscribe(t *testing.T) {
	log := NewMemoryLog()
	first, err := log.Publish(context.Background(), domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("publish first: %v", err)
	}
	second, err := log.Publish(context.Background(), domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("publish second: %v", err)
	}
	if first.Position.Sequence != 1 || second.Position.Sequence != 2 {
		t.Fatalf("expected monotonic sequences, got %d and %d", first.Position.Sequence, second.Position.Sequence)
	}

	ch, cancel, err := log.Subscribe(context.Background(), SubscriptionRequest{
		Buffer: 4,
		Replay: ReplayRequest{After: Position{Sequence: 1}},
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer cancel()

	replayed := <-ch
	if replayed.Event.ID != second.Event.ID {
		t.Fatalf("expected replayed %s, got %s", second.Event.ID, replayed.Event.ID)
	}

	third, err := log.Publish(context.Background(), domain.Event{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()})
	if err != nil {
		t.Fatalf("publish third: %v", err)
	}
	live := <-ch
	if live.Event.ID != third.Event.ID {
		t.Fatalf("expected live %s, got %s", third.Event.ID, live.Event.ID)
	}
}

func TestMemoryLogSubscribeRejectsUnsupportedPartitionRoute(t *testing.T) {
	log := NewMemoryLog()

	_, cancel, err := log.Subscribe(context.Background(), SubscriptionRequest{
		PartitionRoute: &PartitionRoute{
			Topic:        "tasks",
			PartitionKey: PartitionKeyTaskID,
		},
	})
	if cancel != nil {
		t.Fatal("expected no cancel function on validation failure")
	}
	if !errors.Is(err, ErrUnsupportedSubscriptionPartitionRoute) {
		t.Fatalf("expected unsupported partition route error, got %v", err)
	}
}

func TestMemoryLogSubscribeRejectsUnsupportedOwnershipContract(t *testing.T) {
	log := NewMemoryLog()

	_, cancel, err := log.Subscribe(context.Background(), SubscriptionRequest{
		OwnershipContract: &SubscriberOwnershipContract{
			SubscriberGroup: "workers",
			Consumer:        "consumer-a",
			Mode:            OwnershipModeExclusive,
		},
	})
	if cancel != nil {
		t.Fatal("expected no cancel function on validation failure")
	}
	if !errors.Is(err, ErrUnsupportedSubscriptionOwnershipContract) {
		t.Fatalf("expected unsupported ownership contract error, got %v", err)
	}
}

func TestMemoryLogPersistsCheckpoints(t *testing.T) {
	log := NewMemoryLog()
	checkpoint := Checkpoint{
		Consumer: "subscriber-a",
		Position: Position{Sequence: 42, Partition: "task", Offset: "42-0"},
		Metadata: map[string]string{"mode": "replay"},
	}
	if err := log.SaveCheckpoint(context.Background(), checkpoint); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	got, ok, err := log.GetCheckpoint(context.Background(), checkpoint.Consumer)
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if !ok {
		t.Fatal("expected checkpoint to exist")
	}
	if got.Position.Sequence != checkpoint.Position.Sequence || got.Metadata["mode"] != "replay" {
		t.Fatalf("unexpected checkpoint: %+v", got)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected checkpoint update time")
	}
}
