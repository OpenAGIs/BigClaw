package events

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type BrokerStubEventLog struct {
	mu          sync.RWMutex
	history     []domain.Event
	checkpoints map[string]SubscriberCheckpoint
	nextSeq     int64
}

func NewBrokerStubEventLog() *BrokerStubEventLog {
	return &BrokerStubEventLog{
		checkpoints: make(map[string]SubscriberCheckpoint),
	}
}

func (s *BrokerStubEventLog) Backend() string {
	return "broker_stub"
}

func (s *BrokerStubEventLog) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		Backend: "broker_stub",
		Scope:   "process_local_stub",
		Publish: FeatureSupport{
			Supported: true,
			Mode:      "append_only_stub",
			Detail:    "Local deterministic broker stub publishes into a repo-native in-process event log.",
		},
		Replay: FeatureSupport{
			Supported: true,
			Mode:      "ordered_stub_replay",
			Detail:    "Replay is served from the stub history so broker append and resume flows can be validated locally.",
		},
		Checkpoint: FeatureSupport{
			Supported: true,
			Mode:      "subscriber_ack_stub",
			Detail:    "Subscriber checkpoints are stored by the local stub for contract validation only.",
		},
		Dedup: FeatureSupport{
			Supported: false,
			Detail:    "The broker stub does not model durable deduplication.",
		},
		Filtering: FeatureSupport{
			Supported: true,
			Mode:      "server_side",
			Detail:    "Task and trace filters are applied against stub replay history.",
		},
		Retention: FeatureSupport{
			Supported: true,
			Mode:      "process_memory_stub",
			Detail:    "History lasts only for the current process and is not a real broker durability boundary.",
		},
	}
}

func (s *BrokerStubEventLog) Write(ctx context.Context, event domain.Event) error {
	_, err := s.Publish(ctx, event)
	return err
}

func (s *BrokerStubEventLog) Publish(_ context.Context, event domain.Event) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recorded := cloneEvent(event)
	if recorded.Timestamp.IsZero() {
		recorded.Timestamp = time.Now().UTC()
	}
	s.nextSeq++
	s.history = append(s.history, recorded)
	return Record{
		Event:       cloneEvent(recorded),
		Position:    Position{Sequence: s.nextSeq, Partition: "stub", Offset: stubOffset(s.nextSeq)},
		PublishedAt: recorded.Timestamp,
	}, nil
}

func (s *BrokerStubEventLog) Replay(limit int) ([]domain.Event, error) {
	return s.queryEvents("", "", "", limit), nil
}

func (s *BrokerStubEventLog) ReplayAfter(afterID string, limit int) ([]domain.Event, error) {
	return s.queryEvents("", "", strings.TrimSpace(afterID), limit), nil
}

func (s *BrokerStubEventLog) EventsByTask(taskID string, limit int) ([]domain.Event, error) {
	return s.queryEvents(strings.TrimSpace(taskID), "", "", limit), nil
}

func (s *BrokerStubEventLog) EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error) {
	return s.queryEvents(strings.TrimSpace(taskID), "", strings.TrimSpace(afterID), limit), nil
}

func (s *BrokerStubEventLog) EventsByTrace(traceID string, limit int) ([]domain.Event, error) {
	return s.queryEvents("", strings.TrimSpace(traceID), "", limit), nil
}

func (s *BrokerStubEventLog) EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error) {
	return s.queryEvents("", strings.TrimSpace(traceID), strings.TrimSpace(afterID), limit), nil
}

func (s *BrokerStubEventLog) Acknowledge(subscriberID string, eventID string, at time.Time) (SubscriberCheckpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscriberID = strings.TrimSpace(subscriberID)
	eventID = strings.TrimSpace(eventID)
	if subscriberID == "" || eventID == "" {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	sequence := int64(0)
	for index, event := range s.history {
		if event.ID == eventID {
			sequence = int64(index + 1)
			break
		}
	}
	if sequence == 0 {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	checkpoint := SubscriberCheckpoint{
		SubscriberID:  subscriberID,
		EventID:       eventID,
		EventSequence: sequence,
		UpdatedAt:     at.UTC(),
	}
	s.checkpoints[subscriberID] = checkpoint
	return checkpoint, nil
}

func (s *BrokerStubEventLog) Checkpoint(subscriberID string) (SubscriberCheckpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checkpoint, ok := s.checkpoints[strings.TrimSpace(subscriberID)]
	if !ok {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	return checkpoint, nil
}

func (s *BrokerStubEventLog) RetentionWatermark() (RetentionWatermark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	watermark := RetentionWatermark{
		Backend:           "broker_stub",
		Policy:            "process_memory_stub",
		EventCount:        len(s.history),
		PersistedBoundary: false,
	}
	if len(s.history) == 0 {
		return watermark, nil
	}
	watermark.OldestEventID = s.history[0].ID
	watermark.NewestEventID = s.history[len(s.history)-1].ID
	watermark.OldestSequence = 1
	watermark.NewestSequence = int64(len(s.history))
	return watermark, nil
}

func (s *BrokerStubEventLog) Path() string {
	return ""
}

func (s *BrokerStubEventLog) Close() error {
	return nil
}

func (s *BrokerStubEventLog) queryEvents(taskID, traceID, afterID string, limit int) []domain.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := 0
	if afterID != "" {
		for index, event := range s.history {
			if event.ID == afterID {
				start = index + 1
				break
			}
		}
	}
	filtered := make([]domain.Event, 0, len(s.history)-start)
	for _, event := range s.history[start:] {
		if taskID != "" && event.TaskID != taskID {
			continue
		}
		if traceID != "" && event.TraceID != traceID {
			continue
		}
		filtered = append(filtered, cloneEvent(event))
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

func cloneEvent(event domain.Event) domain.Event {
	if event.Payload == nil {
		return event
	}
	copied := make(map[string]any, len(event.Payload))
	for key, value := range event.Payload {
		copied[key] = value
	}
	event.Payload = copied
	return event
}

func stubOffset(sequence int64) string {
	return fmt.Sprintf("stub-%d", sequence)
}
