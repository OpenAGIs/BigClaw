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
	mu             sync.RWMutex
	history        []domain.Event
	subscribers    map[int]chan domain.Event
	sinks          []Sink
	nextID         int
	provider       CapabilityProvider
	capability     BackendCapabilities
	historyLimit   int
	historyDropped bool
}

func NewBus() *Bus {
	return NewBusWithHistoryLimit(0)
}

func NewBusWithHistoryLimit(limit int) *Bus {
	return &Bus{
		subscribers:  make(map[int]chan domain.Event),
		capability:   defaultBusCapabilities(),
		historyLimit: limit,
	}
}

type ReplayCursorStatus struct {
	RequestedAfterID string `json:"requested_after_id,omitempty"`
	OldestEventID    string `json:"oldest_event_id,omitempty"`
	NewestEventID    string `json:"newest_event_id,omitempty"`
	ReplayWindowSize int    `json:"replay_window_size"`
	Status           string `json:"status"`
	Fallback         string `json:"fallback"`
	HistoryTruncated bool   `json:"history_truncated"`
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
	if b.historyLimit > 0 && len(b.history) > b.historyLimit {
		drop := len(b.history) - b.historyLimit
		b.history = append([]domain.Event(nil), b.history[drop:]...)
		b.historyDropped = true
	}
	subs := make([]chan domain.Event, 0, len(b.subscribers))
	for _, ch := range b.subscribers {
		subs = append(subs, ch)
	}
	sinks := append([]Sink(nil), b.sinks...)
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- WithDelivery(event, domain.EventDeliveryModeLive):
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
	replay, _ := b.ReplayWindow(limit, "", "", "")
	return b.subscribe(buffer, replay)
}

func (b *Bus) SubscribeReplayWindow(buffer int, replay []domain.Event) (<-chan domain.Event, func()) {
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
		ch <- WithDelivery(event, domain.EventDeliveryModeReplay)
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

func leadingEvents(events []domain.Event, limit int) []domain.Event {
	if limit <= 0 || len(events) <= limit {
		out := make([]domain.Event, len(events))
		copy(out, events)
		return out
	}
	out := make([]domain.Event, limit)
	copy(out, events[:limit])
	return out
}

func (b *Bus) ReplayWindow(limit int, afterID string, taskID string, traceID string) ([]domain.Event, ReplayCursorStatus) {
	b.mu.RLock()
	filtered := make([]domain.Event, 0, len(b.history))
	for _, event := range b.history {
		if taskID != "" && event.TaskID != taskID {
			continue
		}
		if traceID != "" && event.TraceID != traceID {
			continue
		}
		filtered = append(filtered, event)
	}
	truncated := b.historyDropped
	b.mu.RUnlock()

	status := ReplayCursorStatus{
		RequestedAfterID: afterID,
		ReplayWindowSize: len(filtered),
		Status:           "ok",
		Fallback:         "none",
		HistoryTruncated: truncated,
	}
	if len(filtered) > 0 {
		status.OldestEventID = filtered[0].ID
		status.NewestEventID = filtered[len(filtered)-1].ID
	}
	if afterID == "" {
		return WithDeliveryBatch(lastEvents(filtered, limit), domain.EventDeliveryModeReplay), status
	}
	for index, event := range filtered {
		if event.ID != afterID {
			continue
		}
		return WithDeliveryBatch(leadingEvents(filtered[index+1:], limit), domain.EventDeliveryModeReplay), status
	}
	if len(filtered) == 0 {
		status.Status = "empty"
		status.Fallback = "empty"
		return nil, status
	}
	status.Fallback = "resume_from_oldest"
	if truncated {
		status.Status = "expired"
	} else {
		status.Status = "not_found"
	}
	return WithDeliveryBatch(leadingEvents(filtered, limit), domain.EventDeliveryModeReplay), status
}

func (b *Bus) Replay() []domain.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]domain.Event, len(b.history))
	copy(out, b.history)
	return out
}
