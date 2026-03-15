package events

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestMemoryConsumerDedupLedgerReserveAndApply(t *testing.T) {
	assertConsumerDedupLedgerReserveAndApply(t, NewMemoryConsumerDedupLedger())
}

func TestMemoryConsumerDedupLedgerRejectsStorageKeyCollision(t *testing.T) {
	assertConsumerDedupLedgerRejectsStorageKeyCollision(t, NewMemoryConsumerDedupLedger())
}

func TestMemoryConsumerDedupLedgerRejectsOverwritingAppliedResult(t *testing.T) {
	assertConsumerDedupLedgerRejectsOverwritingAppliedResult(t, NewMemoryConsumerDedupLedger())
}

func TestSQLiteConsumerDedupLedgerReserveAndApply(t *testing.T) {
	ledger := newSQLiteConsumerDedupLedgerForTest(t)
	assertConsumerDedupLedgerReserveAndApply(t, ledger)
}

func TestSQLiteConsumerDedupLedgerRejectsStorageKeyCollision(t *testing.T) {
	ledger := newSQLiteConsumerDedupLedgerForTest(t)
	assertConsumerDedupLedgerRejectsStorageKeyCollision(t, ledger)
}

func TestSQLiteConsumerDedupLedgerRejectsOverwritingAppliedResult(t *testing.T) {
	ledger := newSQLiteConsumerDedupLedgerForTest(t)
	assertConsumerDedupLedgerRejectsOverwritingAppliedResult(t, ledger)
}

func TestSQLiteConsumerDedupLedgerPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "consumer-dedup.db")
	now := time.Unix(1700000000, 0).UTC()
	key := domain.NewConsumerDedupKey("consumer-a", domain.Event{
		ID:      "evt-1",
		Type:    domain.EventTaskCompleted,
		TaskID:  "task-1",
		TraceID: "trace-1",
		RunID:   "run-1",
	})

	first, err := NewSQLiteConsumerDedupLedger(path)
	if err != nil {
		t.Fatalf("new sqlite consumer dedup ledger: %v", err)
	}
	if _, outcome, err := first.MarkApplied(context.Background(), key, domain.ConsumerDedupResult{
		Handler:           "projection",
		AppliedAt:         now,
		EffectID:          "effect-1",
		EffectSequence:    7,
		EffectFingerprint: "fp-1",
		Summary:           "projection updated",
	}, now); err != nil {
		t.Fatalf("mark applied: %v", err)
	} else if outcome != domain.ConsumerDedupOutcomeReserved {
		t.Fatalf("expected reserved outcome, got %s", outcome)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first sqlite consumer dedup ledger: %v", err)
	}

	second, err := NewSQLiteConsumerDedupLedger(path)
	if err != nil {
		t.Fatalf("reopen sqlite consumer dedup ledger: %v", err)
	}
	defer func() {
		if err := second.Close(); err != nil {
			t.Fatalf("close reopened sqlite consumer dedup ledger: %v", err)
		}
	}()

	record, ok, err := second.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if !ok {
		t.Fatal("expected persisted dedup record after reopen")
	}
	if record.State != domain.ConsumerDedupStateApplied {
		t.Fatalf("expected applied state after reopen, got %s", record.State)
	}
	if record.AppliedResult.EffectID != "effect-1" {
		t.Fatalf("expected persisted effect id effect-1, got %s", record.AppliedResult.EffectID)
	}
}

func assertConsumerDedupLedgerReserveAndApply(t *testing.T, ledger ConsumerDedupLedger) {
	t.Helper()

	key := domain.NewConsumerDedupKey("consumer-a", domain.Event{
		ID:      "evt-1",
		Type:    domain.EventTaskCompleted,
		TaskID:  "task-1",
		TraceID: "trace-1",
		RunID:   "run-1",
	})
	now := time.Unix(1700000000, 0).UTC()

	record, outcome, err := ledger.Reserve(context.Background(), key, now)
	if err != nil {
		t.Fatalf("reserve failed: %v", err)
	}
	if outcome != domain.ConsumerDedupOutcomeReserved {
		t.Fatalf("expected reserved outcome, got %s", outcome)
	}
	if record.State != domain.ConsumerDedupStatePending {
		t.Fatalf("expected pending record, got %s", record.State)
	}
	if record.AttemptCount != 1 {
		t.Fatalf("expected first attempt count, got %d", record.AttemptCount)
	}

	record, outcome, err = ledger.Reserve(context.Background(), key, now.Add(time.Second))
	if err != nil {
		t.Fatalf("duplicate reserve failed: %v", err)
	}
	if outcome != domain.ConsumerDedupOutcomeDuplicate {
		t.Fatalf("expected duplicate outcome, got %s", outcome)
	}
	if record.AttemptCount != 2 {
		t.Fatalf("expected duplicate attempt count, got %d", record.AttemptCount)
	}

	applied := domain.ConsumerDedupResult{
		Handler:           "projection",
		AppliedAt:         now.Add(2 * time.Second),
		EffectID:          "effect-1",
		EffectSequence:    7,
		EffectFingerprint: "fp-1",
		Summary:           "projection updated",
	}
	record, outcome, err = ledger.MarkApplied(context.Background(), key, applied, now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("mark applied failed: %v", err)
	}
	if outcome != domain.ConsumerDedupOutcomeReserved {
		t.Fatalf("expected reserved outcome on first apply, got %s", outcome)
	}
	if record.State != domain.ConsumerDedupStateApplied {
		t.Fatalf("expected applied state, got %s", record.State)
	}
	if record.ResultFingerprint == "" {
		t.Fatalf("expected recorded result fingerprint")
	}

	record, outcome, err = ledger.Reserve(context.Background(), key, now.Add(3*time.Second))
	if err != nil {
		t.Fatalf("already applied reserve failed: %v", err)
	}
	if outcome != domain.ConsumerDedupOutcomeAlreadyApplied {
		t.Fatalf("expected already_applied outcome, got %s", outcome)
	}
	if record.AppliedResult.EffectID != "effect-1" {
		t.Fatalf("expected stored result to survive duplicate read")
	}
}

func assertConsumerDedupLedgerRejectsStorageKeyCollision(t *testing.T, ledger ConsumerDedupLedger) {
	t.Helper()

	now := time.Unix(1700000000, 0).UTC()
	first := domain.ConsumerDedupKey{
		ConsumerID: "consumer-a",
		EventID:    "evt-1",
		EventType:  domain.EventTaskQueued,
		TaskID:     "task-1",
	}
	second := domain.ConsumerDedupKey{
		ConsumerID: "consumer-a",
		EventID:    "evt-1",
		EventType:  domain.EventTaskCompleted,
		TaskID:     "task-2",
	}

	if _, _, err := ledger.Reserve(context.Background(), first, now); err != nil {
		t.Fatalf("reserve failed: %v", err)
	}
	if _, outcome, err := ledger.Reserve(context.Background(), second, now.Add(time.Second)); !errors.Is(err, ErrConsumerDedupConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	} else if outcome != domain.ConsumerDedupOutcomeConflict {
		t.Fatalf("expected conflict outcome, got %s", outcome)
	}
}

func assertConsumerDedupLedgerRejectsOverwritingAppliedResult(t *testing.T, ledger ConsumerDedupLedger) {
	t.Helper()

	key := domain.NewConsumerDedupKey("consumer-a", domain.Event{
		ID:      "evt-1",
		Type:    domain.EventTaskCompleted,
		TaskID:  "task-1",
		TraceID: "trace-1",
	})
	now := time.Unix(1700000000, 0).UTC()

	if _, _, err := ledger.MarkApplied(context.Background(), key, domain.ConsumerDedupResult{
		Handler:        "projection",
		AppliedAt:      now,
		EffectID:       "effect-1",
		EffectSequence: 1,
		Summary:        "first",
	}, now); err != nil {
		t.Fatalf("initial apply failed: %v", err)
	}

	if _, outcome, err := ledger.MarkApplied(context.Background(), key, domain.ConsumerDedupResult{
		Handler:        "projection",
		AppliedAt:      now.Add(time.Second),
		EffectID:       "effect-2",
		EffectSequence: 2,
		Summary:        "second",
	}, now.Add(time.Second)); !errors.Is(err, ErrConsumerDedupConflict) {
		t.Fatalf("expected overwrite conflict, got %v", err)
	} else if outcome != domain.ConsumerDedupOutcomeConflict {
		t.Fatalf("expected conflict outcome, got %s", outcome)
	}
}

func newSQLiteConsumerDedupLedgerForTest(t *testing.T) *SQLiteConsumerDedupLedger {
	t.Helper()

	ledger, err := NewSQLiteConsumerDedupLedger(filepath.Join(t.TempDir(), "consumer-dedup.db"))
	if err != nil {
		t.Fatalf("new sqlite consumer dedup ledger: %v", err)
	}
	t.Cleanup(func() {
		if err := ledger.Close(); err != nil {
			t.Fatalf("close sqlite consumer dedup ledger: %v", err)
		}
	})
	return ledger
}
