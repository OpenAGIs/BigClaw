package events

type RetentionWatermark struct {
	Backend                string `json:"backend,omitempty"`
	Policy                 string `json:"policy,omitempty"`
	OldestEventID          string `json:"oldest_event_id,omitempty"`
	NewestEventID          string `json:"newest_event_id,omitempty"`
	OldestSequence         int64  `json:"oldest_sequence,omitempty"`
	NewestSequence         int64  `json:"newest_sequence,omitempty"`
	EventCount             int    `json:"event_count"`
	HistoryTruncated       bool   `json:"history_truncated,omitempty"`
	PersistedBoundary      bool   `json:"persisted_boundary,omitempty"`
	TrimmedThroughEventID  string `json:"trimmed_through_event_id,omitempty"`
	TrimmedThroughSequence int64  `json:"trimmed_through_sequence,omitempty"`
	RetentionWindowSeconds int64  `json:"retention_window_seconds,omitempty"`
}

type RetentionWatermarkProvider interface {
	RetentionWatermark() (RetentionWatermark, error)
}
