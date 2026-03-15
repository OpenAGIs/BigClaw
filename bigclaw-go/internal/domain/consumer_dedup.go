package domain

import (
	"strings"
	"time"
)

type ConsumerLedgerState string

const (
	ConsumerLedgerStateReceived         ConsumerLedgerState = "received"
	ConsumerLedgerStateApplied          ConsumerLedgerState = "applied"
	ConsumerLedgerStateDuplicate        ConsumerLedgerState = "duplicate"
	ConsumerLedgerStateAlreadyApplied   ConsumerLedgerState = "already_applied"
	ConsumerLedgerStateRetryableFailure ConsumerLedgerState = "retryable_failure"
	ConsumerLedgerStateTerminalFailure  ConsumerLedgerState = "terminal_failure"
)

type ConsumerHandleResult struct {
	State             ConsumerLedgerState `json:"state"`
	Attempt           int                 `json:"attempt,omitempty"`
	CompletedAt       time.Time           `json:"completed_at,omitempty"`
	SideEffectKey     string              `json:"side_effect_key,omitempty"`
	SideEffectApplied bool                `json:"side_effect_applied,omitempty"`
	Outcome           string              `json:"outcome,omitempty"`
	Error             string              `json:"error,omitempty"`
}

type ConsumerDedupLedgerEntry struct {
	LedgerKey         string              `json:"ledger_key"`
	ConsumerGroup     string              `json:"consumer_group"`
	Handler           string              `json:"handler"`
	EventID           string              `json:"event_id"`
	IdempotencyKey    string              `json:"idempotency_key"`
	EventType         EventType           `json:"event_type"`
	TaskID            string              `json:"task_id,omitempty"`
	TraceID           string              `json:"trace_id,omitempty"`
	RunID             string              `json:"run_id,omitempty"`
	DeliveryMode      EventDeliveryMode   `json:"delivery_mode,omitempty"`
	FirstSeenAt       time.Time           `json:"first_seen_at"`
	LastSeenAt        time.Time           `json:"last_seen_at"`
	Attempt           int                 `json:"attempt"`
	State             ConsumerLedgerState `json:"state"`
	CompletedAt       time.Time           `json:"completed_at,omitempty"`
	SideEffectKey     string              `json:"side_effect_key,omitempty"`
	SideEffectApplied bool                `json:"side_effect_applied,omitempty"`
	Outcome           string              `json:"outcome,omitempty"`
	Error             string              `json:"error,omitempty"`
}

func NewConsumerDedupLedgerEntry(consumerGroup, handler string, event Event, seenAt time.Time) ConsumerDedupLedgerEntry {
	idempotencyKey := EventIdempotencyKey(event)
	return ConsumerDedupLedgerEntry{
		LedgerKey:      ConsumerDedupLedgerKey(consumerGroup, handler, idempotencyKey),
		ConsumerGroup:  consumerGroup,
		Handler:        handler,
		EventID:        event.ID,
		IdempotencyKey: idempotencyKey,
		EventType:      event.Type,
		TaskID:         event.TaskID,
		TraceID:        event.TraceID,
		RunID:          event.RunID,
		DeliveryMode:   EventDeliveryModeFor(event),
		FirstSeenAt:    seenAt,
		LastSeenAt:     seenAt,
		Attempt:        1,
		State:          ConsumerLedgerStateReceived,
	}
}

func ConsumerDedupLedgerKey(consumerGroup, handler, idempotencyKey string) string {
	parts := []string{
		strings.TrimSpace(consumerGroup),
		strings.TrimSpace(handler),
		strings.TrimSpace(idempotencyKey),
	}
	return strings.Join(parts, "::")
}

func EventIdempotencyKey(event Event) string {
	if event.Delivery != nil && strings.TrimSpace(event.Delivery.IdempotencyKey) != "" {
		return strings.TrimSpace(event.Delivery.IdempotencyKey)
	}
	return strings.TrimSpace(event.ID)
}

func EventDeliveryModeFor(event Event) EventDeliveryMode {
	if event.Delivery == nil {
		return ""
	}
	return event.Delivery.Mode
}

func (entry ConsumerDedupLedgerEntry) WithResult(result ConsumerHandleResult) ConsumerDedupLedgerEntry {
	updated := entry
	if result.Attempt > 0 {
		updated.Attempt = result.Attempt
	}
	if updated.Attempt <= 0 {
		updated.Attempt = 1
	}
	updated.State = result.State
	updated.CompletedAt = result.CompletedAt
	updated.LastSeenAt = timestampOrFallback(result.CompletedAt, updated.LastSeenAt)
	updated.SideEffectKey = result.SideEffectKey
	updated.SideEffectApplied = result.SideEffectApplied
	updated.Outcome = result.Outcome
	updated.Error = result.Error
	return updated
}

func (entry ConsumerDedupLedgerEntry) NextAttempt(seenAt time.Time) ConsumerDedupLedgerEntry {
	updated := entry
	updated.Attempt++
	updated.LastSeenAt = seenAt
	updated.State = ConsumerLedgerStateReceived
	updated.CompletedAt = time.Time{}
	updated.Outcome = ""
	updated.Error = ""
	return updated
}

func (entry ConsumerDedupLedgerEntry) IsTerminal() bool {
	switch entry.State {
	case ConsumerLedgerStateApplied, ConsumerLedgerStateDuplicate, ConsumerLedgerStateAlreadyApplied, ConsumerLedgerStateTerminalFailure:
		return true
	default:
		return false
	}
}

func timestampOrFallback(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
