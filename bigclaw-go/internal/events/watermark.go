package events

type RetentionWatermark struct {
	Backend          string `json:"backend,omitempty"`
	OldestEventID    string `json:"oldest_event_id,omitempty"`
	NewestEventID    string `json:"newest_event_id,omitempty"`
	OldestSequence   int64  `json:"oldest_sequence,omitempty"`
	NewestSequence   int64  `json:"newest_sequence,omitempty"`
	EventCount       int    `json:"event_count"`
	HistoryTruncated bool   `json:"history_truncated,omitempty"`
}

type RetentionWatermarkProvider interface {
	RetentionWatermark() (RetentionWatermark, error)
}
