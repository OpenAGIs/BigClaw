package events

import (
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBusReplayAndSubscribe(t *testing.T) {
	bus := NewBus()
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	event := domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, Timestamp: time.Now()}
	bus.Publish(event)

	got := <-ch
	if got.ID != event.ID {
		t.Fatalf("expected %s, got %s", event.ID, got.ID)
	}
	if got.Delivery == nil || got.Delivery.Mode != domain.EventDeliveryModeLive || got.Delivery.IdempotencyKey != event.ID {
		t.Fatalf("expected live delivery metadata with stable idempotency key, got %+v", got.Delivery)
	}
	if len(bus.Replay()) != 1 {
		t.Fatalf("expected 1 event in replay")
	}
}

func TestBusSubscribeReplayReturnsHistoryAndLiveEvents(t *testing.T) {
	bus := NewBus()
	first := domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	second := domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	bus.Publish(first)
	bus.Publish(second)

	ch, cancel := bus.SubscribeReplay(4, 1)
	defer cancel()

	replayed := <-ch
	if replayed.ID != second.ID {
		t.Fatalf("expected replayed event %s, got %s", second.ID, replayed.ID)
	}
	if replayed.Delivery == nil || replayed.Delivery.Mode != domain.EventDeliveryModeReplay || !replayed.Delivery.Replay || replayed.Delivery.IdempotencyKey != second.ID {
		t.Fatalf("expected replay delivery metadata with stable idempotency key, got %+v", replayed.Delivery)
	}

	third := domain.Event{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	bus.Publish(third)
	live := <-ch
	if live.ID != third.ID {
		t.Fatalf("expected live event %s, got %s", third.ID, live.ID)
	}
	if live.Delivery == nil || live.Delivery.Mode != domain.EventDeliveryModeLive || live.Delivery.IdempotencyKey != third.ID {
		t.Fatalf("expected live delivery metadata after replay handoff, got %+v", live.Delivery)
	}
}
