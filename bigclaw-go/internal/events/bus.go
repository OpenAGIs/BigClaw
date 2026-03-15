package events

import (
	"context"
	"sync"

	"bigclaw-go/internal/domain"
)

type Sink interface {
	Write(context.Context, domain.Event) error
}

type SubscriptionFilter struct {
	TaskID     string
	TraceID    string
	EventTypes map[domain.EventType]struct{}
}

func (f SubscriptionFilter) Match(event domain.Event) bool {
	if f.TaskID != "" && event.TaskID != f.TaskID {
		return false
	}
	if f.TraceID != "" && event.TraceID != f.TraceID {
		return false
	}
	if len(f.EventTypes) > 0 {
		if _, ok := f.EventTypes[event.Type]; !ok {
			return false
		}
	}
	return true
}

type subscriber struct {
	ch     chan domain.Event
	filter SubscriptionFilter
}

type Bus struct {
	mu          sync.RWMutex
	history     []domain.Event
	subscribers map[int]subscriber
	sinks       []Sink
	nextID      int
}

func NewBus() *Bus {
	return &Bus{subscribers: make(map[int]subscriber)}
}

func (b *Bus) AddSink(sink Sink) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks = append(b.sinks, sink)
}

func (b *Bus) Publish(event domain.Event) {
	b.mu.Lock()
	b.history = append(b.history, event)
	subs := make([]subscriber, 0, len(b.subscribers))
	for _, sub := range b.subscribers {
		subs = append(subs, sub)
	}
	sinks := append([]Sink(nil), b.sinks...)
	b.mu.Unlock()

	for _, sub := range subs {
		if !sub.filter.Match(event) {
			continue
		}
		select {
		case sub.ch <- event:
		default:
		}
	}
	for _, sink := range sinks {
		_ = sink.Write(context.Background(), event)
	}
}

func (b *Bus) Subscribe(buffer int) (<-chan domain.Event, func()) {
	return b.subscribe(buffer, nil, SubscriptionFilter{})
}

func (b *Bus) SubscribeTopic(buffer int, filter SubscriptionFilter) (<-chan domain.Event, func()) {
	return b.subscribe(buffer, nil, filter)
}

func (b *Bus) SubscribeReplay(buffer int, limit int) (<-chan domain.Event, func()) {
	b.mu.RLock()
	replay := lastEvents(b.history, limit)
	b.mu.RUnlock()
	return b.subscribe(buffer, replay, SubscriptionFilter{})
}

func (b *Bus) SubscribeReplayTopic(buffer int, limit int, filter SubscriptionFilter) (<-chan domain.Event, func()) {
	b.mu.RLock()
	replay := filterEvents(lastEvents(b.history, limit), filter)
	b.mu.RUnlock()
	return b.subscribe(buffer, replay, filter)
}

func (b *Bus) subscribe(buffer int, replay []domain.Event, filter SubscriptionFilter) (<-chan domain.Event, func()) {
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
	b.subscribers[id] = subscriber{ch: ch, filter: filter}
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if current, ok := b.subscribers[id]; ok {
			close(current.ch)
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

func filterEvents(events []domain.Event, filter SubscriptionFilter) []domain.Event {
	if filter.TaskID == "" && filter.TraceID == "" && len(filter.EventTypes) == 0 {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	filtered := make([]domain.Event, 0, len(events))
	for _, event := range events {
		if filter.Match(event) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (b *Bus) Replay() []domain.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]domain.Event, len(b.history))
	copy(out, b.history)
	return out
}
