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

	third := domain.Event{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	bus.Publish(third)
	live := <-ch
	if live.ID != third.ID {
		t.Fatalf("expected live event %s, got %s", third.ID, live.ID)
	}
}

func TestBusReplayWindowFallsBackToOldestAvailableEventWhenCursorExpires(t *testing.T) {
	bus := NewBusWithHistoryLimit(2)
	first := domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	second := domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	third := domain.Event{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-1", TraceID: "trace-1", Timestamp: time.Now()}
	bus.Publish(first)
	bus.Publish(second)
	bus.Publish(third)

	replay, status := bus.ReplayWindow(10, first.ID, "task-1", "trace-1")
	if status.Status != "expired" {
		t.Fatalf("expected expired status, got %s", status.Status)
	}
	if status.Fallback != "resume_from_oldest" {
		t.Fatalf("expected resume_from_oldest fallback, got %s", status.Fallback)
	}
	if !status.HistoryTruncated {
		t.Fatal("expected history_truncated to be true")
	}
	if len(replay) != 2 || replay[0].ID != second.ID || replay[1].ID != third.ID {
		t.Fatalf("expected fallback replay window [evt-2 evt-3], got %#v", replay)
	}
}
