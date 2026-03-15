package events

import (
	"context"
	"strings"
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

type staticCapabilityProvider struct {
	capability BackendCapabilities
}

func (p staticCapabilityProvider) Capabilities(context.Context) BackendCapabilities {
	return p.capability
}

func TestBusCapabilitiesDefaultAndOverride(t *testing.T) {
	bus := NewBus()
	capability := bus.Capabilities(context.Background())
	if capability.Backend != "in_memory_history" {
		t.Fatalf("expected default backend in_memory_history, got %s", capability.Backend)
	}
	if !capability.Replay.Supported || capability.Checkpoint.Supported {
		t.Fatalf("unexpected default capability set: %+v", capability)
	}

	override := BackendCapabilities{
		Backend: "broker_adapter",
		Scope:   "shared_cluster",
		Publish: FeatureSupport{Supported: true, Mode: "replicated"},
		Replay:  FeatureSupport{Supported: true, Mode: "durable"},
		Checkpoint: FeatureSupport{
			Supported: true,
			Mode:      "lease_aware",
		},
		Dedup:     FeatureSupport{Supported: true, Mode: "shared_store"},
		Filtering: FeatureSupport{Supported: true, Mode: "server_side"},
		Retention: FeatureSupport{Supported: true, Mode: "ttl"},
	}
	bus.SetCapabilityProvider(staticCapabilityProvider{capability: override})
	if got := bus.Capabilities(context.Background()); got.Backend != "broker_adapter" || !got.Checkpoint.Supported || got.Retention.Mode != "ttl" {
		t.Fatalf("expected provider override, got %+v", got)
	}
}

func TestBrokerBootstrapCapabilitiesAdvertiseConfiguredReadiness(t *testing.T) {
	capability := BrokerBootstrapCapabilities(BrokerRuntimeConfig{
		Driver: "kafka",
		Topic:  "bigclaw.events",
	})
	if capability.Backend != "broker" || capability.Scope != "broker_bootstrap" {
		t.Fatalf("unexpected broker bootstrap capability identity: %+v", capability)
	}
	if capability.Publish.Supported || capability.Replay.Supported || capability.Checkpoint.Supported {
		t.Fatalf("expected broker bootstrap to avoid claiming live support, got %+v", capability)
	}
	if capability.Filtering.Mode != "provider_defined" || capability.Retention.Mode != "broker_managed" {
		t.Fatalf("expected broker bootstrap filtering/retention guidance, got %+v", capability)
	}
	if !strings.Contains(capability.Publish.Detail, "driver=kafka") || !strings.Contains(capability.Publish.Detail, "topic=bigclaw.events") {
		t.Fatalf("expected broker bootstrap detail to include configured driver/topic, got %q", capability.Publish.Detail)
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
