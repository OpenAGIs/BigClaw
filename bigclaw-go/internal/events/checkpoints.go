package events

import "time"

type SubscriberCheckpoint struct {
	SubscriberID  string    `json:"subscriber_id"`
	EventID       string    `json:"event_id"`
	EventSequence int64     `json:"event_sequence,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CheckpointResetRequest struct {
	RequestedBy string `json:"requested_by,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Source      string `json:"source,omitempty"`
}

type CheckpointResetRecord struct {
	SubscriberID       string                `json:"subscriber_id"`
	ResetAt            time.Time             `json:"reset_at"`
	RequestedBy        string                `json:"requested_by,omitempty"`
	Reason             string                `json:"reason,omitempty"`
	Source             string                `json:"source,omitempty"`
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

type CheckpointResetManager interface {
	ResetCheckpointWithAudit(subscriberID string, request CheckpointResetRequest) (CheckpointResetRecord, error)
}

type CheckpointResetHistoryProvider interface {
	CheckpointResetHistory(subscriberID string, limit int) ([]CheckpointResetRecord, error)
}
