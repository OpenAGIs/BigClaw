package events

import "time"

type SubscriberCheckpoint struct {
	SubscriberID  string    `json:"subscriber_id"`
	EventID       string    `json:"event_id"`
	EventSequence int64     `json:"event_sequence,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CheckpointStore interface {
	Acknowledge(subscriberID string, eventID string, at time.Time) (SubscriberCheckpoint, error)
	Checkpoint(subscriberID string) (SubscriberCheckpoint, error)
}

type CheckpointResetter interface {
	ResetCheckpoint(subscriberID string) error
}
