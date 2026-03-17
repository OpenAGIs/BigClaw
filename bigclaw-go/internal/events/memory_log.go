package events

import (
	"context"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type MemoryLog struct {
	mu          sync.RWMutex
	history     []Record
	subscribers map[int]chan Record
	checkpoints map[string]Checkpoint
	nextID      int
	nextSeq     int64
}

func NewMemoryLog() *MemoryLog {
	return &MemoryLog{
		subscribers: make(map[int]chan Record),
		checkpoints: make(map[string]Checkpoint),
	}
}

func (l *MemoryLog) Backend() EventLogBackend {
	return EventLogBackendMemory
}

func (l *MemoryLog) Capabilities() Capabilities {
	return Capabilities{
		Durable:             false,
		OrderedReplay:       true,
		LiveSubscriptions:   true,
		ConsumerCheckpoints: true,
		BrokerBacked:        false,
	}
}

func (l *MemoryLog) Write(ctx context.Context, event domain.Event) error {
	_, err := l.Publish(ctx, event)
	return err
}

func (l *MemoryLog) Publish(_ context.Context, event domain.Event) (Record, error) {
	l.mu.Lock()
	l.nextSeq++
	record := Record{
		Event:       event,
		Position:    Position{Sequence: l.nextSeq},
		PublishedAt: time.Now().UTC(),
	}
	l.history = append(l.history, record)
	subs := make([]chan Record, 0, len(l.subscribers))
	for _, ch := range l.subscribers {
		subs = append(subs, ch)
	}
	l.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- record:
		default:
		}
	}
	return record, nil
}

func (l *MemoryLog) Replay(_ context.Context, request ReplayRequest) ([]Record, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	filtered := filterRecords(l.history, request)
	if request.Limit > 0 && len(filtered) > request.Limit {
		filtered = filtered[len(filtered)-request.Limit:]
	}
	out := make([]Record, len(filtered))
	copy(out, filtered)
	return out, nil
}

func (l *MemoryLog) Subscribe(ctx context.Context, request SubscriptionRequest) (<-chan Record, func(), error) {
	_ = ctx
	if err := validateSubscriptionRequest(l.Backend(), request); err != nil {
		return nil, nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	replay := filterRecords(l.history, request.Replay)
	if request.Replay.Limit > 0 && len(replay) > request.Replay.Limit {
		replay = replay[len(replay)-request.Replay.Limit:]
	}
	id := l.nextID
	l.nextID++
	capacity := request.Buffer
	if len(replay) > capacity {
		capacity = len(replay)
	}
	if capacity <= 0 {
		capacity = 1
	}
	ch := make(chan Record, capacity)
	for _, record := range replay {
		ch <- record
	}
	l.subscribers[id] = ch
	return ch, func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if current, ok := l.subscribers[id]; ok {
			close(current)
			delete(l.subscribers, id)
		}
	}, nil
}

func (l *MemoryLog) GetCheckpoint(_ context.Context, consumer string) (Checkpoint, bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	checkpoint, ok := l.checkpoints[consumer]
	if !ok {
		return Checkpoint{}, false, nil
	}
	if checkpoint.Metadata != nil {
		copied := make(map[string]string, len(checkpoint.Metadata))
		for key, value := range checkpoint.Metadata {
			copied[key] = value
		}
		checkpoint.Metadata = copied
	}
	return checkpoint, true, nil
}

func (l *MemoryLog) SaveCheckpoint(_ context.Context, checkpoint Checkpoint) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	if checkpoint.Metadata != nil {
		copied := make(map[string]string, len(checkpoint.Metadata))
		for key, value := range checkpoint.Metadata {
			copied[key] = value
		}
		checkpoint.Metadata = copied
	}
	l.checkpoints[checkpoint.Consumer] = checkpoint
	return nil
}

func filterRecords(history []Record, request ReplayRequest) []Record {
	filtered := make([]Record, 0, len(history))
	for _, record := range history {
		if request.After.Sequence > 0 && record.Position.Sequence <= request.After.Sequence {
			continue
		}
		if request.TaskID != "" && record.Event.TaskID != request.TaskID {
			continue
		}
		if request.TraceID != "" && record.Event.TraceID != request.TraceID {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered
}
