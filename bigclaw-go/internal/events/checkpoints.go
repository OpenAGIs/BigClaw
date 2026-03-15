package events

import (
	"errors"
	"fmt"
	"time"
)

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

type CheckpointDiagnosticProvider interface {
	CheckpointDiagnostic(subscriberID string) (CheckpointDiagnostic, error)
}

type CheckpointDiagnostic struct {
	SubscriberID       string                 `json:"subscriber_id"`
	Backend            string                 `json:"backend,omitempty"`
	Status             string                 `json:"status"`
	Reason             string                 `json:"reason,omitempty"`
	Checkpoint         *SubscriberCheckpoint  `json:"checkpoint,omitempty"`
	RetentionWatermark *RetentionWatermark    `json:"retention_watermark,omitempty"`
	ResetAction        *CheckpointResetAction `json:"reset_action,omitempty"`
}

type CheckpointResetAction struct {
	Action                  string `json:"action"`
	Scope                   string `json:"scope,omitempty"`
	EarliestRetainedEventID string `json:"earliest_retained_event_id,omitempty"`
	LatestRetainedEventID   string `json:"latest_retained_event_id,omitempty"`
	Message                 string `json:"message,omitempty"`
}

type CheckpointDiagnosticError struct {
	Diagnostic CheckpointDiagnostic
}

func (e *CheckpointDiagnosticError) Error() string {
	if e == nil {
		return ""
	}
	subscriberID := e.Diagnostic.SubscriberID
	if subscriberID == "" && e.Diagnostic.Checkpoint != nil {
		subscriberID = e.Diagnostic.Checkpoint.SubscriberID
	}
	if subscriberID == "" {
		subscriberID = "unknown"
	}
	switch e.Diagnostic.Status {
	case "expired":
		return fmt.Sprintf("subscriber checkpoint expired for %s", subscriberID)
	case "missing":
		return fmt.Sprintf("subscriber checkpoint missing retained event for %s", subscriberID)
	default:
		return fmt.Sprintf("subscriber checkpoint unavailable for %s", subscriberID)
	}
}

func IsCheckpointDiagnostic(err error) bool {
	var target *CheckpointDiagnosticError
	return errors.As(err, &target)
}

func IsExpiredCheckpoint(err error) bool {
	var target *CheckpointDiagnosticError
	return errors.As(err, &target) && target.Diagnostic.Status == "expired"
}
