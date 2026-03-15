package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"bigclaw-go/internal/domain"
	_ "modernc.org/sqlite"
)

type EventLog interface {
	Sink
	Replay(limit int) ([]domain.Event, error)
	EventsByTask(taskID string, limit int) ([]domain.Event, error)
	EventsByTrace(traceID string, limit int) ([]domain.Event, error)
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

func (s *SQLiteEventLog) EventsByTask(taskID string, limit int) ([]domain.Event, error) {
	return s.query(`SELECT raw FROM event_log WHERE task_id = ? ORDER BY seq DESC`, []any{taskID}, limit)
}

func (s *SQLiteEventLog) EventsByTrace(traceID string, limit int) ([]domain.Event, error) {
	return s.query(`SELECT raw FROM event_log WHERE trace_id = ? ORDER BY seq DESC`, []any{traceID}, limit)
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

func IsNoEventLog(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
