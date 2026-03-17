package coordination

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteLeaderElection struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

func NewSQLiteLeaderElection(path string) (*SQLiteLeaderElection, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLiteLeaderElection{db: db, path: path}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteLeaderElection) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteLeaderElection) Campaign(request CampaignRequest) (lease LeaderLease, err error) {
	if request.Scope == "" || request.Candidate == "" {
		return LeaderLease{}, fmt.Errorf("scope and candidate are required")
	}
	if request.TTL <= 0 {
		return LeaderLease{}, fmt.Errorf("ttl must be positive")
	}
	if request.Now.IsZero() {
		request.Now = time.Now().UTC()
	}

	err = s.withImmediateTx(func() error {
		current, ok, err := s.get(request.Scope)
		if err != nil {
			return err
		}
		if ok && IsLeaderActive(current, request.Now) && current.LeaderID != request.Candidate {
			lease = current
			return ErrLeaderHeld
		}
		if ok && IsLeaderActive(current, request.Now) && current.LeaderID == request.Candidate {
			current.ExpiresAt = request.Now.Add(request.TTL)
			current.RenewedAt = request.Now
			if err := s.save(current); err != nil {
				return err
			}
			lease = current
			return nil
		}

		token, err := s.nextToken()
		if err != nil {
			return err
		}
		next := LeaderLease{
			Scope:      request.Scope,
			LeaderID:   request.Candidate,
			LeaseToken: token,
			LeaseEpoch: current.LeaseEpoch + 1,
			ExpiresAt:  request.Now.Add(request.TTL),
			AcquiredAt: request.Now,
			RenewedAt:  request.Now,
		}
		if err := s.save(next); err != nil {
			return err
		}
		lease = next
		return nil
	})
	return lease, err
}

func (s *SQLiteLeaderElection) Get(scope string) (LeaderLease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lease, ok, err := s.get(scope)
	if err != nil {
		return LeaderLease{}, false
	}
	return lease, ok
}

func (s *SQLiteLeaderElection) Resign(scope string, candidate string, leaseToken string, leaseEpoch int64) error {
	return s.withImmediateTx(func() error {
		current, ok, err := s.get(scope)
		if err != nil {
			return err
		}
		if !ok {
			return ErrLeaderExpired
		}
		if current.LeaderID != candidate || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
			return ErrLeaderFence
		}
		_, err = s.db.Exec(`DELETE FROM coordinator_leader_lease WHERE scope = ?`, scope)
		return err
	})
}

func (s *SQLiteLeaderElection) String() string {
	return fmt.Sprintf("sqlite:%s", s.path)
}

func (s *SQLiteLeaderElection) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS coordinator_leader_lease (
			scope TEXT PRIMARY KEY,
			leader_id TEXT NOT NULL,
			lease_token TEXT NOT NULL,
			lease_epoch INTEGER NOT NULL,
			expires_at_ns INTEGER NOT NULL,
			acquired_at_ns INTEGER NOT NULL,
			renewed_at_ns INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS coordinator_leader_token_seq (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteLeaderElection) withImmediateTx(fn func() error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err = s.db.Exec(`BEGIN IMMEDIATE`); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_, _ = s.db.Exec(`ROLLBACK`)
		}
	}()
	if err = fn(); err != nil {
		return err
	}
	_, err = s.db.Exec(`COMMIT`)
	return err
}

func (s *SQLiteLeaderElection) get(scope string) (LeaderLease, bool, error) {
	row := s.db.QueryRow(`SELECT scope, leader_id, lease_token, lease_epoch, expires_at_ns, acquired_at_ns, renewed_at_ns
		FROM coordinator_leader_lease WHERE scope = ?`, scope)
	var lease LeaderLease
	var expiresAtNS int64
	var acquiredAtNS int64
	var renewedAtNS int64
	if err := row.Scan(
		&lease.Scope,
		&lease.LeaderID,
		&lease.LeaseToken,
		&lease.LeaseEpoch,
		&expiresAtNS,
		&acquiredAtNS,
		&renewedAtNS,
	); err != nil {
		if err == sql.ErrNoRows {
			return LeaderLease{}, false, nil
		}
		return LeaderLease{}, false, err
	}
	lease.ExpiresAt = time.Unix(0, expiresAtNS).UTC()
	lease.AcquiredAt = time.Unix(0, acquiredAtNS).UTC()
	lease.RenewedAt = time.Unix(0, renewedAtNS).UTC()
	return lease, true, nil
}

func (s *SQLiteLeaderElection) save(lease LeaderLease) error {
	_, err := s.db.Exec(`INSERT INTO coordinator_leader_lease(
			scope, leader_id, lease_token, lease_epoch, expires_at_ns, acquired_at_ns, renewed_at_ns
		) VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(scope) DO UPDATE SET
			leader_id = excluded.leader_id,
			lease_token = excluded.lease_token,
			lease_epoch = excluded.lease_epoch,
			expires_at_ns = excluded.expires_at_ns,
			acquired_at_ns = excluded.acquired_at_ns,
			renewed_at_ns = excluded.renewed_at_ns`,
		lease.Scope,
		lease.LeaderID,
		lease.LeaseToken,
		lease.LeaseEpoch,
		lease.ExpiresAt.UTC().UnixNano(),
		lease.AcquiredAt.UTC().UnixNano(),
		lease.RenewedAt.UTC().UnixNano(),
	)
	return err
}

func (s *SQLiteLeaderElection) nextToken() (string, error) {
	result, err := s.db.Exec(`INSERT INTO coordinator_leader_token_seq DEFAULT VALUES`)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("leader-%d", id), nil
}

func isSQLiteBusy(err error) bool {
	return err != nil && strings.Contains(err.Error(), "database is locked")
}
