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

func TestBusSubscribeTopicFiltersByEventType(t *testing.T) {
	bus := NewBus()
	ch, cancel := bus.SubscribeTopic(2, SubscriptionFilter{EventTypes: map[domain.EventType]struct{}{domain.EventTaskStarted: {}}})
	defer cancel()

	bus.Publish(domain.Event{ID: "evt-queued", Type: domain.EventTaskQueued, Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-started", Type: domain.EventTaskStarted, Timestamp: time.Now()})

	select {
	case got := <-ch:
		if got.ID != "evt-started" {
			t.Fatalf("expected filtered topic event evt-started, got %+v", got)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for filtered topic event")
	}
}

func TestBusSubscribeReplayTopicFiltersHistoryAndLiveEvents(t *testing.T) {
	bus := NewBus()
	bus.Publish(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-3", Type: domain.EventTaskStarted, TaskID: "task-b", TraceID: "trace-b", Timestamp: time.Now()})

	ch, cancel := bus.SubscribeReplayTopic(4, 10, SubscriptionFilter{TaskID: "task-a", EventTypes: map[domain.EventType]struct{}{domain.EventTaskStarted: {}}})
	defer cancel()

	select {
	case replayed := <-ch:
		if replayed.ID != "evt-2" {
			t.Fatalf("expected replayed filtered event evt-2, got %+v", replayed)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for replayed filtered event")
	}

	bus.Publish(domain.Event{ID: "evt-4", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})
	bus.Publish(domain.Event{ID: "evt-5", Type: domain.EventTaskStarted, TaskID: "task-a", TraceID: "trace-a", Timestamp: time.Now()})

	select {
	case live := <-ch:
		if live.ID != "evt-5" {
			t.Fatalf("expected live filtered event evt-5, got %+v", live)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for live filtered event")
	}
}
