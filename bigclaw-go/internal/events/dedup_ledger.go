package events

import (
	"context"
	"errors"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

var ErrConsumerDedupConflict = errors.New("consumer dedup ledger conflict")

type ConsumerDedupLedger interface {
	Get(context.Context, domain.ConsumerDedupKey) (domain.ConsumerDedupRecord, bool, error)
	Reserve(context.Context, domain.ConsumerDedupKey, time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error)
	MarkApplied(context.Context, domain.ConsumerDedupKey, domain.ConsumerDedupResult, time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error)
}

type MemoryConsumerDedupLedger struct {
	mu      sync.Mutex
	records map[string]domain.ConsumerDedupRecord
}

func NewMemoryConsumerDedupLedger() *MemoryConsumerDedupLedger {
	return &MemoryConsumerDedupLedger{records: make(map[string]domain.ConsumerDedupRecord)}
}

func (l *MemoryConsumerDedupLedger) Get(_ context.Context, key domain.ConsumerDedupKey) (domain.ConsumerDedupRecord, bool, error) {
	normalized, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, false, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	record, ok := l.records[storageKey]
	if !ok {
		return domain.ConsumerDedupRecord{}, false, nil
	}
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, false, ErrConsumerDedupConflict
	}
	record.Key = normalized
	return record, true, nil
}

func (l *MemoryConsumerDedupLedger) Reserve(_ context.Context, key domain.ConsumerDedupKey, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	normalized, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	record, ok := l.records[storageKey]
	if !ok {
		record = domain.ConsumerDedupRecord{
			Key:           normalized,
			StorageKey:    storageKey,
			Fingerprint:   fingerprint,
			State:         domain.ConsumerDedupStatePending,
			CreatedAt:     now,
			UpdatedAt:     now,
			LastAttemptAt: now,
			AttemptCount:  1,
		}
		l.records[storageKey] = record
		return record, domain.ConsumerDedupOutcomeReserved, nil
	}
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
	}

	record.LastAttemptAt = now
	record.UpdatedAt = now
	record.AttemptCount++
	l.records[storageKey] = record
	if record.State == domain.ConsumerDedupStateApplied {
		return record, domain.ConsumerDedupOutcomeAlreadyApplied, nil
	}
	return record, domain.ConsumerDedupOutcomeDuplicate, nil
}

func (l *MemoryConsumerDedupLedger) MarkApplied(_ context.Context, key domain.ConsumerDedupKey, result domain.ConsumerDedupResult, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	normalized, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	normalizedResult := result.Normalize()
	if normalizedResult.AppliedAt.IsZero() {
		normalizedResult.AppliedAt = now
	}
	resultFingerprint := normalizedResult.StableFingerprint()

	l.mu.Lock()
	defer l.mu.Unlock()

	record, ok := l.records[storageKey]
	if !ok {
		record = domain.ConsumerDedupRecord{
			Key:          normalized,
			StorageKey:   storageKey,
			Fingerprint:  fingerprint,
			State:        domain.ConsumerDedupStatePending,
			CreatedAt:    now,
			AttemptCount: 0,
		}
	}
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
	}
	if record.State == domain.ConsumerDedupStateApplied {
		if record.ResultFingerprint != resultFingerprint {
			return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
		}
		return record, domain.ConsumerDedupOutcomeAlreadyApplied, nil
	}

	record.Key = normalized
	record.State = domain.ConsumerDedupStateApplied
	record.AppliedResult = normalizedResult
	record.ResultFingerprint = resultFingerprint
	record.LastAttemptAt = now
	record.UpdatedAt = now
	record.CompletedAt = normalizedResult.AppliedAt
	if record.AttemptCount == 0 {
		record.AttemptCount = 1
	}
	l.records[storageKey] = record
	return record, domain.ConsumerDedupOutcomeReserved, nil
}

func normalizeConsumerDedupKey(key domain.ConsumerDedupKey) (domain.ConsumerDedupKey, string, string, error) {
	normalized := key.Normalize()
	if err := normalized.Validate(); err != nil {
		return domain.ConsumerDedupKey{}, "", "", err
	}
	return normalized, normalized.StorageKey(), normalized.Fingerprint(), nil
}
