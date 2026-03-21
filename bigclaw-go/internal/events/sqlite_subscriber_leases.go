package events

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

type SQLiteSubscriberLeaseStore struct {
	db   *sql.DB
	path string
	mu   sync.Mutex
}

func NewSQLiteSubscriberLeaseStore(path string) (*SQLiteSubscriberLeaseStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &SQLiteSubscriberLeaseStore{db: db, path: path}
	if err := store.initWithRetry(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteSubscriberLeaseStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLiteSubscriberLeaseStore) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS subscriber_group_lease (
			group_id TEXT NOT NULL,
			subscriber_id TEXT NOT NULL,
			consumer_id TEXT NOT NULL,
			lease_token TEXT NOT NULL,
			lease_epoch INTEGER NOT NULL,
			checkpoint_offset INTEGER NOT NULL,
			checkpoint_event_id TEXT NOT NULL,
			expires_at_ns INTEGER NOT NULL,
			updated_at_ns INTEGER NOT NULL,
			PRIMARY KEY(group_id, subscriber_id)
		);`,
		`CREATE TABLE IF NOT EXISTS subscriber_group_lease_token_seq (
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

func (s *SQLiteSubscriberLeaseStore) initWithRetry() error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if err := s.init(); err == nil {
			return nil
		} else if !isSQLiteBusy(err) {
			return err
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return lastErr
}

func (s *SQLiteSubscriberLeaseStore) Acquire(request LeaseRequest) (lease SubscriberLease, err error) {
	if request.GroupID == "" || request.SubscriberID == "" || request.ConsumerID == "" {
		return SubscriberLease{}, fmt.Errorf("group_id, subscriber_id, and consumer_id are required")
	}
	if request.TTL <= 0 {
		return SubscriberLease{}, fmt.Errorf("ttl must be positive")
	}
	if request.Now.IsZero() {
		request.Now = time.Now()
	}

	key := SubscriberLeaseKey{GroupID: request.GroupID, SubscriberID: request.SubscriberID}
	err = s.withImmediateTx(func() error {
		current, ok, err := s.get(key)
		if err != nil {
			return err
		}
		if ok && !leaseExpired(current, request.Now) && current.ConsumerID != request.ConsumerID {
			lease = current
			return ErrLeaseHeld
		}
		if ok && !leaseExpired(current, request.Now) && current.ConsumerID == request.ConsumerID {
			current.ExpiresAt = request.Now.Add(request.TTL)
			current.UpdatedAt = request.Now
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
		next := SubscriberLease{
			GroupID:          request.GroupID,
			SubscriberID:     request.SubscriberID,
			ConsumerID:       request.ConsumerID,
			LeaseToken:       token,
			LeaseEpoch:       current.LeaseEpoch + 1,
			CheckpointOffset: current.CheckpointOffset,
			CheckpointEvent:  current.CheckpointEvent,
			ExpiresAt:        request.Now.Add(request.TTL),
			UpdatedAt:        request.Now,
		}
		if err := s.save(next); err != nil {
			return err
		}
		lease = next
		return nil
	})
	return lease, err
}

func (s *SQLiteSubscriberLeaseStore) Commit(request CheckpointCommit) (lease SubscriberLease, err error) {
	if request.Now.IsZero() {
		request.Now = time.Now()
	}
	key := SubscriberLeaseKey{GroupID: request.GroupID, SubscriberID: request.SubscriberID}
	err = s.withImmediateTx(func() error {
		current, ok, err := s.get(key)
		if err != nil {
			return err
		}
		if !ok {
			return ErrLeaseExpired
		}
		if leaseExpired(current, request.Now) {
			lease = current
			return ErrLeaseExpired
		}
		if current.ConsumerID != request.ConsumerID || current.LeaseToken != request.LeaseToken || current.LeaseEpoch != request.LeaseEpoch {
			lease = current
			return ErrLeaseFence
		}
		if request.CheckpointOffset < current.CheckpointOffset {
			lease = current
			return ErrCheckpointRollback
		}

		current.CheckpointOffset = request.CheckpointOffset
		current.CheckpointEvent = request.CheckpointEvent
		current.UpdatedAt = request.Now
		if err := s.save(current); err != nil {
			return err
		}
		lease = current
		return nil
	})
	return lease, err
}

func (s *SQLiteSubscriberLeaseStore) Get(groupID string, subscriberID string) (SubscriberLease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lease, ok, err := s.get(SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID})
	if err != nil {
		return SubscriberLease{}, false
	}
	return lease, ok
}

func (s *SQLiteSubscriberLeaseStore) Release(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64) error {
	key := SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID}
	return s.withImmediateTx(func() error {
		current, ok, err := s.get(key)
		if err != nil {
			return err
		}
		if !ok {
			return ErrLeaseExpired
		}
		if current.ConsumerID != consumerID || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
			return ErrLeaseFence
		}
		_, err = s.db.Exec(`DELETE FROM subscriber_group_lease WHERE group_id = ? AND subscriber_id = ?`, groupID, subscriberID)
		return err
	})
}

func (s *SQLiteSubscriberLeaseStore) String() string {
	return fmt.Sprintf("sqlite:%s", s.path)
}

func isSQLiteBusy(err error) bool {
	return err != nil && strings.Contains(err.Error(), "database is locked")
}

func (s *SQLiteSubscriberLeaseStore) withImmediateTx(fn func() error) (err error) {
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

func (s *SQLiteSubscriberLeaseStore) get(key SubscriberLeaseKey) (SubscriberLease, bool, error) {
	row := s.db.QueryRow(`SELECT group_id, subscriber_id, consumer_id, lease_token, lease_epoch, checkpoint_offset, checkpoint_event_id, expires_at_ns, updated_at_ns
		FROM subscriber_group_lease WHERE group_id = ? AND subscriber_id = ?`, key.GroupID, key.SubscriberID)
	var lease SubscriberLease
	var checkpointEvent sql.NullString
	var expiresAtNS int64
	var updatedAtNS int64
	if err := row.Scan(
		&lease.GroupID,
		&lease.SubscriberID,
		&lease.ConsumerID,
		&lease.LeaseToken,
		&lease.LeaseEpoch,
		&lease.CheckpointOffset,
		&checkpointEvent,
		&expiresAtNS,
		&updatedAtNS,
	); err != nil {
		if err == sql.ErrNoRows {
			return SubscriberLease{}, false, nil
		}
		return SubscriberLease{}, false, err
	}
	lease.CheckpointEvent = checkpointEvent.String
	lease.ExpiresAt = time.Unix(0, expiresAtNS).UTC()
	lease.UpdatedAt = time.Unix(0, updatedAtNS).UTC()
	return lease, true, nil
}

func (s *SQLiteSubscriberLeaseStore) save(lease SubscriberLease) error {
	_, err := s.db.Exec(`INSERT INTO subscriber_group_lease(
			group_id, subscriber_id, consumer_id, lease_token, lease_epoch, checkpoint_offset, checkpoint_event_id, expires_at_ns, updated_at_ns
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(group_id, subscriber_id) DO UPDATE SET
			consumer_id = excluded.consumer_id,
			lease_token = excluded.lease_token,
			lease_epoch = excluded.lease_epoch,
			checkpoint_offset = excluded.checkpoint_offset,
			checkpoint_event_id = excluded.checkpoint_event_id,
			expires_at_ns = excluded.expires_at_ns,
			updated_at_ns = excluded.updated_at_ns`,
		lease.GroupID,
		lease.SubscriberID,
		lease.ConsumerID,
		lease.LeaseToken,
		lease.LeaseEpoch,
		lease.CheckpointOffset,
		lease.CheckpointEvent,
		lease.ExpiresAt.UTC().UnixNano(),
		lease.UpdatedAt.UTC().UnixNano(),
	)
	return err
}

func (s *SQLiteSubscriberLeaseStore) nextToken() (string, error) {
	result, err := s.db.Exec(`INSERT INTO subscriber_group_lease_token_seq DEFAULT VALUES`)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("lease-%d", id), nil
}
