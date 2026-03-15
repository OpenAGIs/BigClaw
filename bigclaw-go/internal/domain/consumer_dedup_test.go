package domain

import (
	"testing"
	"time"
)

func TestNewConsumerDedupLedgerEntryUsesDeliveryIdempotencyKey(t *testing.T) {
	seenAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	event := Event{
		ID:      "evt-1",
		Type:    EventTaskCompleted,
		TaskID:  "task-1",
		TraceID: "trace-1",
		RunID:   "run-1",
		Delivery: &EventDelivery{
			Mode:           EventDeliveryModeReplay,
			Replay:         true,
			IdempotencyKey: "task-1::task.completed::seq-9",
		},
	}

	entry := NewConsumerDedupLedgerEntry("billing-projection", "applyInvoice", event, seenAt)

	if entry.IdempotencyKey != "task-1::task.completed::seq-9" {
		t.Fatalf("expected delivery idempotency key, got %q", entry.IdempotencyKey)
	}
	if entry.LedgerKey != "billing-projection::applyInvoice::task-1::task.completed::seq-9" {
		t.Fatalf("unexpected ledger key %q", entry.LedgerKey)
	}
	if entry.DeliveryMode != EventDeliveryModeReplay {
		t.Fatalf("expected replay delivery mode, got %q", entry.DeliveryMode)
	}
	if entry.State != ConsumerLedgerStateReceived {
		t.Fatalf("expected received state, got %q", entry.State)
	}
	if entry.Attempt != 1 {
		t.Fatalf("expected first attempt, got %d", entry.Attempt)
	}
}

func TestConsumerDedupLedgerEntryTracksRetryableFailuresAndRetries(t *testing.T) {
	seenAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	entry := NewConsumerDedupLedgerEntry("search-index", "applyDocument", Event{
		ID:   "evt-2",
		Type: EventTaskStarted,
	}, seenAt)

	failedAt := seenAt.Add(2 * time.Minute)
	entry = entry.WithResult(ConsumerHandleResult{
		State:       ConsumerLedgerStateRetryableFailure,
		Attempt:     1,
		CompletedAt: failedAt,
		Error:       "transient upstream timeout",
	})
	if entry.State != ConsumerLedgerStateRetryableFailure {
		t.Fatalf("expected retryable failure state, got %q", entry.State)
	}
	if entry.IsTerminal() {
		t.Fatalf("retryable failure should not be terminal")
	}
	if entry.LastSeenAt != failedAt {
		t.Fatalf("expected last seen to follow completed time")
	}

	retriedAt := failedAt.Add(30 * time.Second)
	entry = entry.NextAttempt(retriedAt)
	if entry.Attempt != 2 {
		t.Fatalf("expected second attempt, got %d", entry.Attempt)
	}
	if entry.State != ConsumerLedgerStateReceived {
		t.Fatalf("expected retry to move back to received, got %q", entry.State)
	}
	if !entry.CompletedAt.IsZero() {
		t.Fatalf("retry should clear completed time")
	}
}

func TestConsumerDedupLedgerEntryRepresentsAlreadyAppliedSideEffects(t *testing.T) {
	seenAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	entry := NewConsumerDedupLedgerEntry("audit-fanout", "publishWebhook", Event{
		ID:   "evt-3",
		Type: EventTaskCompleted,
	}, seenAt)

	completedAt := seenAt.Add(time.Minute)
	entry = entry.WithResult(ConsumerHandleResult{
		State:             ConsumerLedgerStateAlreadyApplied,
		Attempt:           1,
		CompletedAt:       completedAt,
		SideEffectKey:     "webhook-delivery-9",
		SideEffectApplied: true,
		Outcome:           "remote sink already reflected this event",
	})
	if entry.State != ConsumerLedgerStateAlreadyApplied {
		t.Fatalf("expected already applied state, got %q", entry.State)
	}
	if !entry.SideEffectApplied {
		t.Fatalf("expected already applied side effect marker")
	}
	if entry.SideEffectKey != "webhook-delivery-9" {
		t.Fatalf("unexpected side effect key %q", entry.SideEffectKey)
	}
	if !entry.IsTerminal() {
		t.Fatalf("already applied should be terminal")
	}
}

func TestConsumerDedupLedgerEntryRepresentsDuplicateDelivery(t *testing.T) {
	seenAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	entry := NewConsumerDedupLedgerEntry("projection", "applySummary", Event{
		ID:   "evt-4",
		Type: EventTaskCompleted,
	}, seenAt)

	duplicateAt := seenAt.Add(10 * time.Second)
	entry = entry.WithResult(ConsumerHandleResult{
		State:       ConsumerLedgerStateDuplicate,
		Attempt:     1,
		CompletedAt: duplicateAt,
		Outcome:     "ledger entry already completed for this event",
	})
	if entry.State != ConsumerLedgerStateDuplicate {
		t.Fatalf("expected duplicate state, got %q", entry.State)
	}
	if !entry.IsTerminal() {
		t.Fatalf("duplicate should be terminal")
	}
	if entry.LastSeenAt != duplicateAt {
		t.Fatalf("expected duplicate timestamp to be persisted")
	}
}
