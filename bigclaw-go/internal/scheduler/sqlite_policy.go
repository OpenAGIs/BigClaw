package scheduler

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLitePolicyStore struct {
	db   *sql.DB
	path string
}

func NewSQLitePolicyStore(path string) (*SQLitePolicyStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLitePolicyStore{db: db, path: path}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLitePolicyStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLitePolicyStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *SQLitePolicyStore) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS scheduler_policy_state (
			id INTEGER PRIMARY KEY CHECK(id = 1),
			payload BLOB NOT NULL,
			updated_at_ns INTEGER NOT NULL,
			source TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLitePolicyStore) Load() (RoutingRules, time.Time, bool, error) {
	if s == nil || s.db == nil {
		return RoutingRules{}, time.Time{}, false, nil
	}
	row := s.db.QueryRow(`SELECT payload, updated_at_ns FROM scheduler_policy_state WHERE id = 1`)
	var payload []byte
	var updatedAtNS int64
	if err := row.Scan(&payload, &updatedAtNS); err != nil {
		if err == sql.ErrNoRows {
			return RoutingRules{}, time.Time{}, false, nil
		}
		return RoutingRules{}, time.Time{}, false, err
	}
	rules, err := LoadRoutingRulesJSON(payload)
	if err != nil {
		return RoutingRules{}, time.Time{}, false, err
	}
	return rules, time.Unix(0, updatedAtNS), true, nil
}

func (s *SQLitePolicyStore) Save(rules RoutingRules, source string) (time.Time, error) {
	if s == nil || s.db == nil {
		return time.Time{}, nil
	}
	payload, err := json.Marshal(cloneRoutingRules(rules))
	if err != nil {
		return time.Time{}, err
	}
	updatedAt := time.Now()
	_, err = s.db.Exec(`INSERT INTO scheduler_policy_state(id, payload, updated_at_ns, source)
		VALUES(1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET payload=excluded.payload, updated_at_ns=excluded.updated_at_ns, source=excluded.source`, payload, updatedAt.UnixNano(), source)
	if err != nil {
		return time.Time{}, err
	}
	return updatedAt, nil
}
