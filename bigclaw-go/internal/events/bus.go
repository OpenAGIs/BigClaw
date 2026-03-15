package events

import (
	"context"
	"sync"

	"bigclaw-go/internal/domain"
)

type Sink interface {
	Write(context.Context, domain.Event) error
}

type Bus struct {
	mu          sync.RWMutex
	history     []domain.Event
	subscribers map[int]chan domain.Event
	sinks       []Sink
	nextID      int
	provider    CapabilityProvider
	capability  BackendCapabilities
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[int]chan domain.Event),
		capability:  defaultBusCapabilities(),
	}
}

func (b *Bus) AddSink(sink Sink) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks = append(b.sinks, sink)
}

func (b *Bus) SetCapabilityProvider(provider CapabilityProvider) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.provider = provider
}

func (b *Bus) SetCapabilities(capability BackendCapabilities) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.capability = capability
}

func (b *Bus) Capabilities(ctx context.Context) BackendCapabilities {
	b.mu.RLock()
	provider := b.provider
	capability := b.capability
	b.mu.RUnlock()
	if provider != nil {
		return provider.Capabilities(ctx)
	}
	return capability
}

func (b *Bus) Publish(event domain.Event) {
	b.mu.Lock()
	b.history = append(b.history, event)
	subs := make([]chan domain.Event, 0, len(b.subscribers))
	for _, ch := range b.subscribers {
		subs = append(subs, ch)
	}
	sinks := append([]Sink(nil), b.sinks...)
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
	for _, sink := range sinks {
		_ = sink.Write(context.Background(), event)
	}
}

func (b *Bus) Subscribe(buffer int) (<-chan domain.Event, func()) {
	return b.subscribe(buffer, nil)
}

func (b *Bus) SubscribeReplay(buffer int, limit int) (<-chan domain.Event, func()) {
	b.mu.RLock()
	replay := lastEvents(b.history, limit)
	b.mu.RUnlock()
	return b.subscribe(buffer, replay)
}

func (b *Bus) subscribe(buffer int, replay []domain.Event) (<-chan domain.Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++
	capacity := buffer
	if len(replay) > capacity {
		capacity = len(replay)
	}
	if capacity <= 0 {
		capacity = 1
	}
	ch := make(chan domain.Event, capacity)
	for _, event := range replay {
		ch <- event
	}
	b.subscribers[id] = ch
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if current, ok := b.subscribers[id]; ok {
			close(current)
			delete(b.subscribers, id)
		}
	}
}

func lastEvents(events []domain.Event, limit int) []domain.Event {
	if limit <= 0 || len(events) <= limit {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	start := len(events) - limit
	out := make([]domain.Event, limit)
	copy(out, events[start:])
	return out
}

func (b *Bus) Replay() []domain.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]domain.Event, len(b.history))
	copy(out, b.history)
	return out
}
