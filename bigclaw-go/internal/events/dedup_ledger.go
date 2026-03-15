package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
	_ "modernc.org/sqlite"
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
		record = newPendingConsumerDedupRecord(normalized, storageKey, fingerprint, now, 1)
		l.records[storageKey] = record
		return record, domain.ConsumerDedupOutcomeReserved, nil
	}
	record, outcome, err := reserveExistingConsumerDedupRecord(record, normalized, fingerprint, now)
	if err != nil {
		return domain.ConsumerDedupRecord{}, outcome, err
	}
	l.records[storageKey] = record
	return record, outcome, nil
}

func (l *MemoryConsumerDedupLedger) MarkApplied(_ context.Context, key domain.ConsumerDedupKey, result domain.ConsumerDedupResult, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
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
		record = newPendingConsumerDedupRecord(normalized, storageKey, fingerprint, now, 0)
	}
	record, outcome, err := markAppliedConsumerDedupRecord(record, normalized, fingerprint, result, now)
	if err != nil {
		return domain.ConsumerDedupRecord{}, outcome, err
	}
	l.records[storageKey] = record
	return record, outcome, nil
}

func normalizeConsumerDedupKey(key domain.ConsumerDedupKey) (domain.ConsumerDedupKey, string, string, error) {
	normalized := key.Normalize()
	if err := normalized.Validate(); err != nil {
		return domain.ConsumerDedupKey{}, "", "", err
	}
	return normalized, normalized.StorageKey(), normalized.Fingerprint(), nil
}

func newPendingConsumerDedupRecord(key domain.ConsumerDedupKey, storageKey, fingerprint string, now time.Time, attemptCount int) domain.ConsumerDedupRecord {
	return domain.ConsumerDedupRecord{
		Key:           key,
		StorageKey:    storageKey,
		Fingerprint:   fingerprint,
		State:         domain.ConsumerDedupStatePending,
		CreatedAt:     now,
		UpdatedAt:     now,
		LastAttemptAt: now,
		AttemptCount:  attemptCount,
	}
}

func reserveExistingConsumerDedupRecord(record domain.ConsumerDedupRecord, key domain.ConsumerDedupKey, fingerprint string, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
	}
	record.Key = key
	record.LastAttemptAt = now
	record.UpdatedAt = now
	record.AttemptCount++
	if record.State == domain.ConsumerDedupStateApplied {
		return record, domain.ConsumerDedupOutcomeAlreadyApplied, nil
	}
	return record, domain.ConsumerDedupOutcomeDuplicate, nil
}

func markAppliedConsumerDedupRecord(record domain.ConsumerDedupRecord, key domain.ConsumerDedupKey, fingerprint string, result domain.ConsumerDedupResult, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
	}
	normalizedResult := result.Normalize()
	if normalizedResult.AppliedAt.IsZero() {
		normalizedResult.AppliedAt = now
	}
	resultFingerprint := normalizedResult.StableFingerprint()
	if record.State == domain.ConsumerDedupStateApplied {
		if record.ResultFingerprint != resultFingerprint {
			return domain.ConsumerDedupRecord{}, domain.ConsumerDedupOutcomeConflict, ErrConsumerDedupConflict
		}
		record.Key = key
		return record, domain.ConsumerDedupOutcomeAlreadyApplied, nil
	}
	record.Key = key
	record.State = domain.ConsumerDedupStateApplied
	record.AppliedResult = normalizedResult
	record.ResultFingerprint = resultFingerprint
	record.LastAttemptAt = now
	record.UpdatedAt = now
	record.CompletedAt = normalizedResult.AppliedAt
	if record.AttemptCount == 0 {
		record.AttemptCount = 1
	}
	return record, domain.ConsumerDedupOutcomeReserved, nil
}

type SQLiteConsumerDedupLedger struct {
	db   *sql.DB
	path string
}

func NewSQLiteConsumerDedupLedger(path string) (*SQLiteConsumerDedupLedger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ledger := &SQLiteConsumerDedupLedger{db: db, path: path}
	if err := ledger.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return ledger, nil
}

func (l *SQLiteConsumerDedupLedger) Close() error {
	if l == nil || l.db == nil {
		return nil
	}
	return l.db.Close()
}

func (l *SQLiteConsumerDedupLedger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *SQLiteConsumerDedupLedger) Get(_ context.Context, key domain.ConsumerDedupKey) (domain.ConsumerDedupRecord, bool, error) {
	_, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, false, err
	}
	record, ok, err := l.lookup(storageKey)
	if err != nil || !ok {
		return record, ok, err
	}
	if record.Fingerprint != fingerprint {
		return domain.ConsumerDedupRecord{}, false, ErrConsumerDedupConflict
	}
	return record, true, nil
}

func (l *SQLiteConsumerDedupLedger) Reserve(_ context.Context, key domain.ConsumerDedupKey, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	normalized, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := l.db.BeginTx(context.Background(), nil)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	record, ok, err := l.lookupTx(tx, storageKey)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	var outcome domain.ConsumerDedupReserveOutcome
	if !ok {
		record = newPendingConsumerDedupRecord(normalized, storageKey, fingerprint, now, 1)
		outcome = domain.ConsumerDedupOutcomeReserved
	} else {
		record, outcome, err = reserveExistingConsumerDedupRecord(record, normalized, fingerprint, now)
		if err != nil {
			return domain.ConsumerDedupRecord{}, outcome, err
		}
	}
	if err := l.upsertTx(tx, record); err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	return record, outcome, nil
}

func (l *SQLiteConsumerDedupLedger) MarkApplied(_ context.Context, key domain.ConsumerDedupKey, result domain.ConsumerDedupResult, now time.Time) (domain.ConsumerDedupRecord, domain.ConsumerDedupReserveOutcome, error) {
	normalized, storageKey, fingerprint, err := normalizeConsumerDedupKey(key)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := l.db.BeginTx(context.Background(), nil)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	record, ok, err := l.lookupTx(tx, storageKey)
	if err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if !ok {
		record = newPendingConsumerDedupRecord(normalized, storageKey, fingerprint, now, 0)
	}
	record, outcome, err := markAppliedConsumerDedupRecord(record, normalized, fingerprint, result, now)
	if err != nil {
		return domain.ConsumerDedupRecord{}, outcome, err
	}
	if err := l.upsertTx(tx, record); err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return domain.ConsumerDedupRecord{}, "", err
	}
	return record, outcome, nil
}

func (l *SQLiteConsumerDedupLedger) init() error {
	if l == nil || l.db == nil {
		return nil
	}
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS consumer_dedup_ledger (
			storage_key TEXT PRIMARY KEY,
			fingerprint TEXT NOT NULL,
			record_json BLOB NOT NULL,
			state TEXT NOT NULL,
			updated_at_ns INTEGER NOT NULL,
			completed_at_ns INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_consumer_dedup_state_updated ON consumer_dedup_ledger(state, updated_at_ns);`,
		`CREATE INDEX IF NOT EXISTS idx_consumer_dedup_completed ON consumer_dedup_ledger(completed_at_ns);`,
	}
	for _, stmt := range stmts {
		if _, err := l.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (l *SQLiteConsumerDedupLedger) lookup(storageKey string) (domain.ConsumerDedupRecord, bool, error) {
	if l == nil || l.db == nil {
		return domain.ConsumerDedupRecord{}, false, nil
	}
	row := l.db.QueryRow(`SELECT record_json FROM consumer_dedup_ledger WHERE storage_key = ?`, storageKey)
	return scanConsumerDedupRecord(row)
}

func (l *SQLiteConsumerDedupLedger) lookupTx(tx *sql.Tx, storageKey string) (domain.ConsumerDedupRecord, bool, error) {
	row := tx.QueryRow(`SELECT record_json FROM consumer_dedup_ledger WHERE storage_key = ?`, storageKey)
	return scanConsumerDedupRecord(row)
}

func (l *SQLiteConsumerDedupLedger) upsertTx(tx *sql.Tx, record domain.ConsumerDedupRecord) error {
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	completedAtNS := int64(0)
	if !record.CompletedAt.IsZero() {
		completedAtNS = record.CompletedAt.UnixNano()
	}
	_, err = tx.Exec(`
		INSERT INTO consumer_dedup_ledger(storage_key, fingerprint, record_json, state, updated_at_ns, completed_at_ns)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(storage_key) DO UPDATE SET
			fingerprint = excluded.fingerprint,
			record_json = excluded.record_json,
			state = excluded.state,
			updated_at_ns = excluded.updated_at_ns,
			completed_at_ns = excluded.completed_at_ns
	`, record.StorageKey, record.Fingerprint, payload, string(record.State), record.UpdatedAt.UnixNano(), completedAtNS)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanConsumerDedupRecord(row scanner) (domain.ConsumerDedupRecord, bool, error) {
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ConsumerDedupRecord{}, false, nil
		}
		return domain.ConsumerDedupRecord{}, false, err
	}
	var record domain.ConsumerDedupRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return domain.ConsumerDedupRecord{}, false, err
	}
	return record, true, nil
}
