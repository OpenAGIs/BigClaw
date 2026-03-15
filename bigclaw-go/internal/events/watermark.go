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

type RetentionBootstrap struct {
	Backend       string `json:"backend,omitempty"`
	Durable       bool   `json:"durable"`
	Shared        bool   `json:"shared,omitempty"`
	LogDSN        string `json:"log_dsn,omitempty"`
	CheckpointDSN string `json:"checkpoint_dsn,omitempty"`
	RetentionMode string `json:"retention_mode,omitempty"`
	Detail        string `json:"detail,omitempty"`
}

type RetentionBootstrapProvider interface {
	RetentionBootstrap() RetentionBootstrap
}
