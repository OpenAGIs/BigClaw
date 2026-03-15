package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/internal/domain"
	_ "modernc.org/sqlite"
)

type EventLog interface {
	Sink
	Replay(limit int) ([]domain.Event, error)
	ReplayAfter(afterID string, limit int) ([]domain.Event, error)
	EventsByTask(taskID string, limit int) ([]domain.Event, error)
	EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error)
	EventsByTrace(traceID string, limit int) ([]domain.Event, error)
	EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error)
	Path() string
	Close() error
}

type SQLiteEventLog struct {
	db   *sql.DB
	path string
}

func NewSQLiteEventLog(path string) (*SQLiteEventLog, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLiteEventLog{db: db, path: path}
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
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteEventLog) Write(_ context.Context, event domain.Event) error {
	if s == nil || s.db == nil {
		return nil
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO event_log(event_id, event_type, task_id, trace_id, run_id, timestamp_ns, raw) VALUES(?, ?, ?, ?, ?, ?, ?)`, event.ID, string(event.Type), event.TaskID, event.TraceID, event.RunID, event.Timestamp.UnixNano(), raw)
	return err
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
		at = time.Now().UTC()
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
	row := s.db.QueryRow(`SELECT subscriber_id, event_id, updated_at_ns FROM subscriber_checkpoint WHERE subscriber_id = ?`, subscriberID)
	var checkpoint SubscriberCheckpoint
	var updatedAtNS int64
	if err := row.Scan(&checkpoint.SubscriberID, &checkpoint.EventID, &updatedAtNS); err != nil {
		return SubscriberCheckpoint{}, err
	}
	checkpoint.UpdatedAt = time.Unix(0, updatedAtNS).UTC()
	return checkpoint, nil
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
