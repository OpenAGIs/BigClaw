package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	_ "modernc.org/sqlite"
)

type EventLog interface {
	Sink
	Backend() string
	Capabilities() BackendCapabilities
	Replay(limit int) ([]domain.Event, error)
	ReplayAfter(afterID string, limit int) ([]domain.Event, error)
	EventsByTask(taskID string, limit int) ([]domain.Event, error)
	EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error)
	EventsByTrace(traceID string, limit int) ([]domain.Event, error)
	EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error)
	Path() string
	Close() error
}

type SQLiteEventLogOptions struct {
	Retention time.Duration
	Now       func() time.Time
}

type SQLiteEventLog struct {
	db              *sql.DB
	path            string
	retentionWindow time.Duration
	now             func() time.Time
}

func NewSQLiteEventLog(path string) (*SQLiteEventLog, error) {
	return NewSQLiteEventLogWithOptions(path, SQLiteEventLogOptions{})
}

func NewSQLiteEventLogWithOptions(path string, options SQLiteEventLogOptions) (*SQLiteEventLog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLiteEventLog{db: db, path: path, retentionWindow: options.Retention, now: options.Now}
	if store.now == nil {
		store.now = time.Now
	}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteEventLog) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteEventLog) Backend() string {
	return "sqlite"
}

func (s *SQLiteEventLog) Capabilities() BackendCapabilities {
	retention := FeatureSupport{Supported: true, Mode: "sqlite_wal", Detail: "Replay history persists in SQLite and survives process restarts."}
	if s != nil && s.retentionWindow > 0 {
		retention.Mode = "sqlite_retention_window"
		retention.Detail = fmt.Sprintf("Retention boundaries persist across restarts with a configured %s replay window.", s.retentionWindow)
	}
	return BackendCapabilities{
		Backend:    "sqlite",
		Scope:      "shared_node",
		Publish:    FeatureSupport{Supported: true, Mode: "append_only"},
		Replay:     FeatureSupport{Supported: true, Mode: "durable"},
		Checkpoint: FeatureSupport{Supported: true, Mode: "subscriber_ack"},
		Dedup:      FeatureSupport{Supported: true, Mode: "sqlite", Detail: "Consumer dedup records persist across process restarts."},
		Filtering:  FeatureSupport{Supported: true, Mode: "server_side"},
		Retention:  retention,
	}
}

func (s *SQLiteEventLog) RetentionWatermark() (RetentionWatermark, error) {
	watermark := RetentionWatermark{Backend: "sqlite"}
	if s == nil || s.db == nil {
		return watermark, nil
	}
	state, err := s.readRetentionState()
	if err != nil {
		return RetentionWatermark{}, err
	}
	watermark.Policy = state.Policy
	if state.RetentionWindow > 0 {
		watermark.RetentionWindowSeconds = int64(state.RetentionWindow / time.Second)
	}
	if state.TrimmedThroughSequence > 0 {
		watermark.HistoryTruncated = true
		watermark.PersistedBoundary = true
		watermark.TrimmedThroughSequence = state.TrimmedThroughSequence
		watermark.TrimmedThroughEventID = state.TrimmedThroughEventID
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM event_log`).Scan(&watermark.EventCount); err != nil {
		return RetentionWatermark{}, err
	}
	if watermark.EventCount == 0 {
		return watermark, nil
	}
	if err := s.db.QueryRow(`SELECT seq, event_id FROM event_log ORDER BY seq ASC LIMIT 1`).Scan(&watermark.OldestSequence, &watermark.OldestEventID); err != nil {
		return RetentionWatermark{}, err
	}
	if err := s.db.QueryRow(`SELECT seq, event_id FROM event_log ORDER BY seq DESC LIMIT 1`).Scan(&watermark.NewestSequence, &watermark.NewestEventID); err != nil {
		return RetentionWatermark{}, err
	}
	return watermark, nil
}

func (s *SQLiteEventLog) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *SQLiteEventLog) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS event_log (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			task_id TEXT,
			trace_id TEXT,
			run_id TEXT,
			timestamp_ns INTEGER NOT NULL,
			raw BLOB NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_event_log_task ON event_log(task_id, seq);`,
		`CREATE INDEX IF NOT EXISTS idx_event_log_trace ON event_log(trace_id, seq);`,
		`CREATE INDEX IF NOT EXISTS idx_event_log_type ON event_log(event_type, seq);`,
		`CREATE TABLE IF NOT EXISTS subscriber_checkpoint (
			subscriber_id TEXT PRIMARY KEY,
			event_id TEXT NOT NULL,
			event_seq INTEGER NOT NULL,
			updated_at_ns INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS checkpoint_reset_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			subscriber_id TEXT NOT NULL,
			reset_at_ns INTEGER NOT NULL,
			prior_event_id TEXT,
			prior_event_seq INTEGER,
			prior_updated_at_ns INTEGER,
			retention_watermark_json BLOB NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_reset_history_subscriber ON checkpoint_reset_history(subscriber_id, reset_at_ns DESC, id DESC);`,
		`CREATE TABLE IF NOT EXISTS event_log_retention_state (
			singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
			policy TEXT NOT NULL,
			retention_window_ns INTEGER NOT NULL,
			trimmed_through_seq INTEGER NOT NULL,
			trimmed_through_event_id TEXT NOT NULL,
			updated_at_ns INTEGER NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(`
		INSERT OR IGNORE INTO event_log_retention_state(singleton, policy, retention_window_ns, trimmed_through_seq, trimmed_through_event_id, updated_at_ns)
		VALUES(1, 'disabled', 0, 0, '', 0)
	`); err != nil {
		return err
	}
	if err := s.updateRetentionConfig(); err != nil {
		return err
	}
	return s.applyRetention(s.nowUTC())
}

func (s *SQLiteEventLog) Write(_ context.Context, event domain.Event) error {
	if s == nil || s.db == nil {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = s.nowUTC()
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO event_log(event_id, event_type, task_id, trace_id, run_id, timestamp_ns, raw) VALUES(?, ?, ?, ?, ?, ?, ?)`, event.ID, string(event.Type), event.TaskID, event.TraceID, event.RunID, event.Timestamp.UnixNano(), raw)
	if err != nil {
		return err
	}
	return s.applyRetention(s.nowUTC())
}

func (s *SQLiteEventLog) Replay(limit int) ([]domain.Event, error) {
	return s.query(`SELECT raw FROM event_log ORDER BY seq DESC`, nil, limit)
}

func (s *SQLiteEventLog) ReplayAfter(afterID string, limit int) ([]domain.Event, error) {
	return s.queryAfter(`SELECT raw FROM event_log WHERE seq > ? ORDER BY seq ASC`, nil, afterID, limit)
}

func (s *SQLiteEventLog) EventsByTask(taskID string, limit int) ([]domain.Event, error) {
	return s.query(`SELECT raw FROM event_log WHERE task_id = ? ORDER BY seq DESC`, []any{taskID}, limit)
}

func (s *SQLiteEventLog) EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error) {
	return s.queryAfter(`SELECT raw FROM event_log WHERE task_id = ? AND seq > ? ORDER BY seq ASC`, []any{taskID}, afterID, limit)
}

func (s *SQLiteEventLog) EventsByTrace(traceID string, limit int) ([]domain.Event, error) {
	return s.query(`SELECT raw FROM event_log WHERE trace_id = ? ORDER BY seq DESC`, []any{traceID}, limit)
}

func (s *SQLiteEventLog) EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error) {
	return s.queryAfter(`SELECT raw FROM event_log WHERE trace_id = ? AND seq > ? ORDER BY seq ASC`, []any{traceID}, afterID, limit)
}

func (s *SQLiteEventLog) Acknowledge(subscriberID string, eventID string, at time.Time) (SubscriberCheckpoint, error) {
	if s == nil || s.db == nil {
		return SubscriberCheckpoint{}, nil
	}
	if subscriberID == "" || eventID == "" {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	seq, ok, err := s.lookupSequenceForEventID(eventID)
	if err != nil {
		return SubscriberCheckpoint{}, err
	}
	if !ok {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	if at.IsZero() {
		at = s.nowUTC()
	}
	_, err = s.db.Exec(`
		INSERT INTO subscriber_checkpoint(subscriber_id, event_id, event_seq, updated_at_ns)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(subscriber_id) DO UPDATE SET
			event_id = excluded.event_id,
			event_seq = excluded.event_seq,
			updated_at_ns = excluded.updated_at_ns
		WHERE excluded.event_seq >= subscriber_checkpoint.event_seq
	`, subscriberID, eventID, seq, at.UnixNano())
	if err != nil {
		return SubscriberCheckpoint{}, err
	}
	return s.Checkpoint(subscriberID)
}

func (s *SQLiteEventLog) Checkpoint(subscriberID string) (SubscriberCheckpoint, error) {
	if s == nil || s.db == nil {
		return SubscriberCheckpoint{}, sql.ErrNoRows
	}
	row := s.db.QueryRow(`SELECT subscriber_id, event_id, event_seq, updated_at_ns FROM subscriber_checkpoint WHERE subscriber_id = ?`, subscriberID)
	var checkpoint SubscriberCheckpoint
	var updatedAtNS int64
	if err := row.Scan(&checkpoint.SubscriberID, &checkpoint.EventID, &checkpoint.EventSequence, &updatedAtNS); err != nil {
		return SubscriberCheckpoint{}, err
	}
	checkpoint.UpdatedAt = time.Unix(0, updatedAtNS).UTC()
	return checkpoint, nil
}

func (s *SQLiteEventLog) ResetCheckpoint(subscriberID string) error {
	if s == nil || s.db == nil || subscriberID == "" {
		return sql.ErrNoRows
	}
	checkpoint, err := s.Checkpoint(subscriberID)
	if err != nil {
		return err
	}
	watermark, err := s.RetentionWatermark()
	if err != nil {
		return err
	}
	retentionWatermarkJSON, err := json.Marshal(watermark)
	if err != nil {
		return err
	}
	resetAt := s.nowUTC()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
		INSERT INTO checkpoint_reset_history(subscriber_id, reset_at_ns, prior_event_id, prior_event_seq, prior_updated_at_ns, retention_watermark_json)
		VALUES(?, ?, ?, ?, ?, ?)
	`, subscriberID, resetAt.UnixNano(), checkpoint.EventID, checkpoint.EventSequence, checkpoint.UpdatedAt.UnixNano(), retentionWatermarkJSON); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM subscriber_checkpoint WHERE subscriber_id = ?`, subscriberID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (s *SQLiteEventLog) CheckpointResetHistory(subscriberID string, limit int) ([]CheckpointResetRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	query := `
		SELECT subscriber_id, reset_at_ns, prior_event_id, prior_event_seq, prior_updated_at_ns, retention_watermark_json
		FROM checkpoint_reset_history
	`
	args := make([]any, 0, 2)
	if subscriberID = strings.TrimSpace(subscriberID); subscriberID != "" {
		query += ` WHERE subscriber_id = ?`
		args = append(args, subscriberID)
	}
	query += ` ORDER BY reset_at_ns DESC, id DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CheckpointResetRecord, 0)
	for rows.Next() {
		var (
			record                CheckpointResetRecord
			resetAtNS             int64
			priorEventID          sql.NullString
			priorEventSeq         sql.NullInt64
			priorUpdatedAtNS      sql.NullInt64
			retentionWatermarkRaw []byte
		)
		if err := rows.Scan(&record.SubscriberID, &resetAtNS, &priorEventID, &priorEventSeq, &priorUpdatedAtNS, &retentionWatermarkRaw); err != nil {
			return nil, err
		}
		record.ResetAt = time.Unix(0, resetAtNS).UTC()
		if priorEventID.Valid || priorEventSeq.Valid || priorUpdatedAtNS.Valid {
			priorCheckpoint := &SubscriberCheckpoint{SubscriberID: record.SubscriberID}
			if priorEventID.Valid {
				priorCheckpoint.EventID = priorEventID.String
			}
			if priorEventSeq.Valid {
				priorCheckpoint.EventSequence = priorEventSeq.Int64
			}
			if priorUpdatedAtNS.Valid {
				priorCheckpoint.UpdatedAt = time.Unix(0, priorUpdatedAtNS.Int64).UTC()
			}
			record.PriorCheckpoint = priorCheckpoint
		}
		if len(retentionWatermarkRaw) > 0 {
			watermark := &RetentionWatermark{}
			if err := json.Unmarshal(retentionWatermarkRaw, watermark); err != nil {
				return nil, err
			}
			record.RetentionWatermark = watermark
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteEventLog) queryAfter(base string, args []any, afterID string, limit int) ([]domain.Event, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	afterSeq, err := s.sequenceForEventID(afterID)
	if err != nil {
		return nil, err
	}
	params := append([]any(nil), args...)
	params = append(params, afterSeq)
	query := base
	if limit > 0 {
		query += ` LIMIT ?`
		params = append(params, limit)
	}
	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Event, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var event domain.Event
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteEventLog) sequenceForEventID(eventID string) (int64, error) {
	seq, _, err := s.lookupSequenceForEventID(eventID)
	return seq, err
}

func (s *SQLiteEventLog) lookupSequenceForEventID(eventID string) (int64, bool, error) {
	if s == nil || s.db == nil || eventID == "" {
		return 0, false, nil
	}
	row := s.db.QueryRow(`SELECT seq FROM event_log WHERE event_id = ? ORDER BY seq DESC LIMIT 1`, eventID)
	var seq int64
	if err := row.Scan(&seq); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return seq, true, nil
}

func (s *SQLiteEventLog) query(base string, args []any, limit int) ([]domain.Event, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	query := base
	params := append([]any(nil), args...)
	if limit > 0 {
		query += ` LIMIT ?`
		params = append(params, limit)
	}
	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Event, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var event domain.Event
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	reverseEvents(out)
	return out, nil
}

func (s *SQLiteEventLog) nowUTC() time.Time {
	if s == nil || s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

type retentionState struct {
	Policy                 string
	RetentionWindow        time.Duration
	TrimmedThroughSequence int64
	TrimmedThroughEventID  string
}

func (s *SQLiteEventLog) readRetentionState() (retentionState, error) {
	if s == nil || s.db == nil {
		return retentionState{}, nil
	}
	row := s.db.QueryRow(`SELECT policy, retention_window_ns, trimmed_through_seq, trimmed_through_event_id FROM event_log_retention_state WHERE singleton = 1`)
	var state retentionState
	var retentionWindowNS int64
	if err := row.Scan(&state.Policy, &retentionWindowNS, &state.TrimmedThroughSequence, &state.TrimmedThroughEventID); err != nil {
		return retentionState{}, err
	}
	state.RetentionWindow = time.Duration(retentionWindowNS)
	return state, nil
}

func (s *SQLiteEventLog) updateRetentionConfig() error {
	if s == nil || s.db == nil {
		return nil
	}
	policy := "disabled"
	retentionWindowNS := int64(0)
	if s.retentionWindow > 0 {
		policy = "time_window"
		retentionWindowNS = s.retentionWindow.Nanoseconds()
	}
	_, err := s.db.Exec(`UPDATE event_log_retention_state SET policy = ?, retention_window_ns = ? WHERE singleton = 1`, policy, retentionWindowNS)
	return err
}

func (s *SQLiteEventLog) applyRetention(now time.Time) error {
	if s == nil || s.db == nil || s.retentionWindow <= 0 {
		return nil
	}
	cutoff := now.Add(-s.retentionWindow).UnixNano()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	var trimmedSeq int64
	var trimmedEventID string
	scanErr := tx.QueryRow(`SELECT seq, event_id FROM event_log WHERE timestamp_ns < ? ORDER BY seq DESC LIMIT 1`, cutoff).Scan(&trimmedSeq, &trimmedEventID)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return tx.Commit()
		}
		err = scanErr
		return err
	}
	result, execErr := tx.Exec(`DELETE FROM event_log WHERE timestamp_ns < ?`, cutoff)
	if execErr != nil {
		err = execErr
		return err
	}
	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		err = rowsErr
		return err
	}
	if rowsAffected == 0 {
		return tx.Commit()
	}
	var existingSeq int64
	if err = tx.QueryRow(`SELECT trimmed_through_seq FROM event_log_retention_state WHERE singleton = 1`).Scan(&existingSeq); err != nil {
		return err
	}
	if trimmedSeq > existingSeq {
		_, err = tx.Exec(`
			UPDATE event_log_retention_state
			SET trimmed_through_seq = ?, trimmed_through_event_id = ?, updated_at_ns = ?
			WHERE singleton = 1
		`, trimmedSeq, trimmedEventID, now.UnixNano())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func reverseEvents(events []domain.Event) {
	for left, right := 0, len(events)-1; left < right; left, right = left+1, right-1 {
		events[left], events[right] = events[right], events[left]
	}
}

var _ EventLog = (*SQLiteEventLog)(nil)
var _ CheckpointStore = (*SQLiteEventLog)(nil)

func IsNoEventLog(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
