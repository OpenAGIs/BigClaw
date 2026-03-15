package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"
)

const ConsumerDedupKeyVersion = "v1"

type ConsumerDedupRecordState string

const (
	ConsumerDedupStatePending ConsumerDedupRecordState = "pending"
	ConsumerDedupStateApplied ConsumerDedupRecordState = "applied"
)

type ConsumerDedupReserveOutcome string

const (
	ConsumerDedupOutcomeReserved       ConsumerDedupReserveOutcome = "reserved"
	ConsumerDedupOutcomeDuplicate      ConsumerDedupReserveOutcome = "duplicate"
	ConsumerDedupOutcomeAlreadyApplied ConsumerDedupReserveOutcome = "already_applied"
	ConsumerDedupOutcomeConflict       ConsumerDedupReserveOutcome = "conflict"
)

type ConsumerDedupKey struct {
	Version    string    `json:"version,omitempty"`
	ConsumerID string    `json:"consumer_id"`
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type,omitempty"`
	TaskID     string    `json:"task_id,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	RunID      string    `json:"run_id,omitempty"`
}

func NewConsumerDedupKey(consumerID string, event Event) ConsumerDedupKey {
	return ConsumerDedupKey{
		Version:    ConsumerDedupKeyVersion,
		ConsumerID: consumerID,
		EventID:    event.ID,
		EventType:  event.Type,
		TaskID:     event.TaskID,
		TraceID:    event.TraceID,
		RunID:      event.RunID,
	}
}

func (k ConsumerDedupKey) Normalize() ConsumerDedupKey {
	k.Version = strings.TrimSpace(k.Version)
	if k.Version == "" {
		k.Version = ConsumerDedupKeyVersion
	}
	k.ConsumerID = strings.TrimSpace(k.ConsumerID)
	k.EventID = strings.TrimSpace(k.EventID)
	k.EventType = EventType(strings.TrimSpace(string(k.EventType)))
	k.TaskID = strings.TrimSpace(k.TaskID)
	k.TraceID = strings.TrimSpace(k.TraceID)
	k.RunID = strings.TrimSpace(k.RunID)
	return k
}

func (k ConsumerDedupKey) StorageKey() string {
	key := k.Normalize()
	return fmt.Sprintf("%s/%s/%s", key.Version, key.ConsumerID, key.EventID)
}

func (k ConsumerDedupKey) Fingerprint() string {
	key := k.Normalize()
	sum := sha256.Sum256([]byte(strings.Join([]string{
		key.Version,
		key.ConsumerID,
		key.EventID,
		string(key.EventType),
		key.TaskID,
		key.TraceID,
		key.RunID,
	}, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func (k ConsumerDedupKey) Validate() error {
	key := k.Normalize()
	if key.ConsumerID == "" {
		return fmt.Errorf("consumer dedup key requires consumer_id")
	}
	if key.EventID == "" {
		return fmt.Errorf("consumer dedup key requires event_id")
	}
	return nil
}

type ConsumerDedupResult struct {
	State             ConsumerDedupRecordState `json:"state"`
	Handler           string                   `json:"handler,omitempty"`
	AppliedAt         time.Time                `json:"applied_at,omitempty"`
	EffectID          string                   `json:"effect_id,omitempty"`
	EffectSequence    int64                    `json:"effect_sequence,omitempty"`
	EffectFingerprint string                   `json:"effect_fingerprint,omitempty"`
	Summary           string                   `json:"summary,omitempty"`
	Metadata          map[string]string        `json:"metadata,omitempty"`
}

func (r ConsumerDedupResult) Normalize() ConsumerDedupResult {
	r.State = ConsumerDedupRecordState(strings.TrimSpace(string(r.State)))
	if r.State == "" {
		r.State = ConsumerDedupStateApplied
	}
	r.Handler = strings.TrimSpace(r.Handler)
	r.EffectID = strings.TrimSpace(r.EffectID)
	r.EffectFingerprint = strings.TrimSpace(r.EffectFingerprint)
	r.Summary = strings.TrimSpace(r.Summary)
	if r.Metadata == nil {
		return r
	}
	cloned := make(map[string]string, len(r.Metadata))
	for key, value := range r.Metadata {
		cloned[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	r.Metadata = cloned
	return r
}

func (r ConsumerDedupResult) StableFingerprint() string {
	normalized := r.Normalize()
	pairs := make([]string, 0, len(normalized.Metadata))
	for key, value := range normalized.Metadata {
		pairs = append(pairs, key+"="+value)
	}
	if len(pairs) > 1 {
		slices.Sort(pairs)
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(normalized.State),
		normalized.Handler,
		normalized.EffectID,
		fmt.Sprintf("%d", normalized.EffectSequence),
		normalized.EffectFingerprint,
		normalized.Summary,
		normalized.AppliedAt.UTC().Format(time.RFC3339Nano),
		strings.Join(pairs, "\x1e"),
	}, "\x1f")))
	return hex.EncodeToString(sum[:])
}

type ConsumerDedupRecord struct {
	Key               ConsumerDedupKey         `json:"key"`
	StorageKey        string                   `json:"storage_key"`
	Fingerprint       string                   `json:"fingerprint"`
	State             ConsumerDedupRecordState `json:"state"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
	LastAttemptAt     time.Time                `json:"last_attempt_at"`
	AttemptCount      int                      `json:"attempt_count"`
	CompletedAt       time.Time                `json:"completed_at,omitempty"`
	AppliedResult     ConsumerDedupResult      `json:"applied_result,omitempty"`
	ResultFingerprint string                   `json:"result_fingerprint,omitempty"`
}
