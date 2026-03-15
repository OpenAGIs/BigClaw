package events

import "time"

type CheckpointResetRecord struct {
	SubscriberID       string                `json:"subscriber_id"`
	ResetAt            time.Time             `json:"reset_at"`
	PriorCheckpoint    *SubscriberCheckpoint `json:"prior_checkpoint,omitempty"`
	RetentionWatermark *RetentionWatermark   `json:"retention_watermark,omitempty"`
}

type CheckpointResetHistoryProvider interface {
	CheckpointResetHistory(subscriberID string, limit int) ([]CheckpointResetRecord, error)
}
