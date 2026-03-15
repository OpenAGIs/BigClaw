package scheduler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteFairnessStore struct {
	db   *sql.DB
	path string
}

func NewSQLiteFairnessStore(path string) (*SQLiteFairnessStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLiteFairnessStore{db: db, path: path}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteFairnessStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteFairnessStore) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS scheduler_fairness_events (
			tenant_id TEXT NOT NULL,
			accepted_at_ns INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduler_fairness_events_tenant_time ON scheduler_fairness_events(tenant_id, accepted_at_ns);`,
		`CREATE INDEX IF NOT EXISTS idx_scheduler_fairness_events_time ON scheduler_fairness_events(accepted_at_ns);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteFairnessStore) ShouldThrottle(now time.Time, tenantID string, rules RoutingRules) bool {
	if s == nil || !fairnessEnabled(rules) || tenantID == "" {
		return false
	}
	if err := s.prune(now, rules); err != nil {
		return false
	}
	cutoff := fairnessCutoff(now, rules)
	rows, err := s.db.Query(`SELECT tenant_id, COUNT(*) FROM scheduler_fairness_events WHERE accepted_at_ns >= ? GROUP BY tenant_id`, cutoff.UnixNano())
	if err != nil {
		return false
	}
	defer rows.Close()
	currentCount := 0
	otherActive := false
	for rows.Next() {
		var rowTenant string
		var count int
		if err := rows.Scan(&rowTenant, &count); err != nil {
			return false
		}
		if rowTenant == tenantID {
			currentCount = count
			continue
		}
		if count > 0 {
			otherActive = true
		}
	}
	if err := rows.Err(); err != nil {
		return false
	}
	return otherActive && currentCount >= rules.Fairness.MaxRecentDecisionsPerTenant
}

func (s *SQLiteFairnessStore) RecordAccepted(now time.Time, tenantID string, rules RoutingRules) {
	if s == nil || !fairnessEnabled(rules) || tenantID == "" {
		return
	}
	_ = s.prune(now, rules)
	_, _ = s.db.Exec(`INSERT INTO scheduler_fairness_events(tenant_id, accepted_at_ns) VALUES(?, ?)`, tenantID, now.UnixNano())
}

func (s *SQLiteFairnessStore) Snapshot(now time.Time, rules RoutingRules) FairnessSnapshot {
	snapshot := fairnessBaseSnapshot(rules, "sqlite", true)
	if s == nil {
		return snapshot
	}
	if err := s.prune(now, rules); err != nil {
		return snapshot
	}
	cutoff := fairnessCutoff(now, rules)
	rows, err := s.db.Query(`SELECT tenant_id, COUNT(*), MIN(accepted_at_ns), MAX(accepted_at_ns) FROM scheduler_fairness_events WHERE accepted_at_ns >= ? GROUP BY tenant_id`, cutoff.UnixNano())
	if err != nil {
		return snapshot
	}
	defer rows.Close()
	for rows.Next() {
		var tenantID string
		var count int
		var oldestNS int64
		var latestNS int64
		if err := rows.Scan(&tenantID, &count, &oldestNS, &latestNS); err != nil {
			return snapshot
		}
		snapshot.Tenants = append(snapshot.Tenants, FairnessTenantSnapshot{
			TenantID:            tenantID,
			RecentAcceptedCount: count,
			OldestAcceptedAt:    time.Unix(0, oldestNS),
			LatestAcceptedAt:    time.Unix(0, latestNS),
		})
	}
	snapshot.ActiveTenants = len(snapshot.Tenants)
	return snapshot
}

func (s *SQLiteFairnessStore) prune(now time.Time, rules RoutingRules) error {
	if s == nil {
		return nil
	}
	if !fairnessEnabled(rules) {
		_, err := s.db.Exec(`DELETE FROM scheduler_fairness_events`)
		return err
	}
	cutoff := fairnessCutoff(now, rules)
	_, err := s.db.Exec(`DELETE FROM scheduler_fairness_events WHERE accepted_at_ns < ?`, cutoff.UnixNano())
	return err
}

func fairnessCutoff(now time.Time, rules RoutingRules) time.Time {
	return now.Add(-time.Duration(rules.Fairness.WindowSeconds) * time.Second)
}

func (s *SQLiteFairnessStore) String() string {
	return fmt.Sprintf("sqlite:%s", s.path)
}
