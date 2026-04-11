package events

import "time"

type SubscriberCheckpoint struct {
	SubscriberID  string    `json:"subscriber_id"`
	EventID       string    `json:"event_id"`
	EventSequence int64     `json:"event_sequence,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CheckpointResetAudit struct {
	SubscriberID       string                `json:"subscriber_id"`
	ResetAt            time.Time             `json:"reset_at"`
	Reason             string                `json:"reason,omitempty"`
	PreviousCheckpoint *SubscriberCheckpoint `json:"previous_checkpoint,omitempty"`
	RetentionWatermark *RetentionWatermark   `json:"retention_watermark,omitempty"`
}

type CheckpointStore interface {
	Acknowledge(subscriberID string, eventID string, at time.Time) (SubscriberCheckpoint, error)
	Checkpoint(subscriberID string) (SubscriberCheckpoint, error)
}

type CheckpointResetter interface {
	ResetCheckpoint(subscriberID string) error
}

type CheckpointResetHistoryProvider interface {
	CheckpointResetHistory(subscriberID string, limit int) ([]CheckpointResetAudit, error)
}

type RecentCheckpointResetProvider interface {
	RecentCheckpointResets(limit int) ([]CheckpointResetAudit, error)
}
