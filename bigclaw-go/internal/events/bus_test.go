package events

import (
	"context"
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
		Filtering: FeatureSupport{Supported: true, Mode: "server_side"},
		Retention: FeatureSupport{Supported: true, Mode: "ttl"},
	}
	bus.SetCapabilityProvider(staticCapabilityProvider{capability: override})
	if got := bus.Capabilities(context.Background()); got.Backend != "broker_adapter" || !got.Checkpoint.Supported || got.Retention.Mode != "ttl" {
		t.Fatalf("expected provider override, got %+v", got)
	}
}
