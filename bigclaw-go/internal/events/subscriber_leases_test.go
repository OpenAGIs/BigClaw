package events

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSubscriberLeaseCoordinatorRejectsActiveConflictAndAllowsExpiryTakeover(t *testing.T) {
	for _, store := range testSubscriberLeaseStores(t) {
		t.Run(store.name, func(t *testing.T) {
			now := time.Unix(1700000000, 0)

			lease, err := store.primary.Acquire(LeaseRequest{
				GroupID:      "group-a",
				SubscriberID: "sub-a",
				ConsumerID:   "consumer-a",
				TTL:          30 * time.Second,
				Now:          now,
			})
			if err != nil {
				t.Fatalf("acquire initial lease: %v", err)
			}

			conflict, err := store.secondary.Acquire(LeaseRequest{
				GroupID:      "group-a",
				SubscriberID: "sub-a",
				ConsumerID:   "consumer-b",
				TTL:          30 * time.Second,
				Now:          now.Add(10 * time.Second),
			})
			if !errors.Is(err, ErrLeaseHeld) {
				t.Fatalf("expected ErrLeaseHeld, got %v", err)
			}
			if conflict.ConsumerID != lease.ConsumerID {
				t.Fatalf("expected conflict to expose current owner %q, got %q", lease.ConsumerID, conflict.ConsumerID)
			}

			takeover, err := store.secondary.Acquire(LeaseRequest{
				GroupID:      "group-a",
				SubscriberID: "sub-a",
				ConsumerID:   "consumer-b",
				TTL:          30 * time.Second,
				Now:          now.Add(31 * time.Second),
			})
			if err != nil {
				t.Fatalf("acquire after expiry: %v", err)
			}
			if takeover.ConsumerID != "consumer-b" {
				t.Fatalf("expected takeover by consumer-b, got %q", takeover.ConsumerID)
			}
			if takeover.LeaseEpoch != lease.LeaseEpoch+1 {
				t.Fatalf("expected epoch to advance from %d to %d, got %d", lease.LeaseEpoch, lease.LeaseEpoch+1, takeover.LeaseEpoch)
			}
		})
	}
}

func TestSubscriberLeaseCoordinatorFencesStaleWriterAndRollback(t *testing.T) {
	for _, store := range testSubscriberLeaseStores(t) {
		t.Run(store.name, func(t *testing.T) {
			now := time.Unix(1700000000, 0)

			lease, err := store.primary.Acquire(LeaseRequest{
				GroupID:      "group-a",
				SubscriberID: "sub-a",
				ConsumerID:   "consumer-a",
				TTL:          20 * time.Second,
				Now:          now,
			})
			if err != nil {
				t.Fatalf("acquire lease: %v", err)
			}
			lease, err = store.primary.Commit(CheckpointCommit{
				GroupID:          "group-a",
				SubscriberID:     "sub-a",
				ConsumerID:       "consumer-a",
				LeaseToken:       lease.LeaseToken,
				LeaseEpoch:       lease.LeaseEpoch,
				CheckpointOffset: 11,
				CheckpointEvent:  "evt-11",
				Now:              now.Add(5 * time.Second),
			})
			if err != nil {
				t.Fatalf("commit checkpoint: %v", err)
			}

			if _, err := store.primary.Commit(CheckpointCommit{
				GroupID:          "group-a",
				SubscriberID:     "sub-a",
				ConsumerID:       "consumer-a",
				LeaseToken:       lease.LeaseToken,
				LeaseEpoch:       lease.LeaseEpoch,
				CheckpointOffset: 10,
				CheckpointEvent:  "evt-10",
				Now:              now.Add(6 * time.Second),
			}); !errors.Is(err, ErrCheckpointRollback) {
				t.Fatalf("expected ErrCheckpointRollback, got %v", err)
			}

			takeover, err := store.secondary.Acquire(LeaseRequest{
				GroupID:      "group-a",
				SubscriberID: "sub-a",
				ConsumerID:   "consumer-b",
				TTL:          20 * time.Second,
				Now:          now.Add(21 * time.Second),
			})
			if err != nil {
				t.Fatalf("acquire takeover lease: %v", err)
			}

			if _, err := store.primary.Commit(CheckpointCommit{
				GroupID:          "group-a",
				SubscriberID:     "sub-a",
				ConsumerID:       "consumer-a",
				LeaseToken:       lease.LeaseToken,
				LeaseEpoch:       lease.LeaseEpoch,
				CheckpointOffset: 12,
				CheckpointEvent:  "evt-12",
				Now:              now.Add(22 * time.Second),
			}); !errors.Is(err, ErrLeaseFence) {
				t.Fatalf("expected stale writer to be fenced, got %v", err)
			}

			current, err := store.secondary.Commit(CheckpointCommit{
				GroupID:          "group-a",
				SubscriberID:     "sub-a",
				ConsumerID:       "consumer-b",
				LeaseToken:       takeover.LeaseToken,
				LeaseEpoch:       takeover.LeaseEpoch,
				CheckpointOffset: 15,
				CheckpointEvent:  "evt-15",
				Now:              now.Add(23 * time.Second),
			})
			if err != nil {
				t.Fatalf("commit with takeover lease: %v", err)
			}
			if current.CheckpointOffset != 15 {
				t.Fatalf("expected checkpoint offset 15, got %d", current.CheckpointOffset)
			}
		})
	}
}

type testSubscriberLeaseStorePair struct {
	name      string
	primary   SubscriberLeaseStore
	secondary SubscriberLeaseStore
}

func testSubscriberLeaseStores(t *testing.T) []testSubscriberLeaseStorePair {
	t.Helper()

	memoryStore := newMemorySubscriberLeaseStore()
	sqlitePath := filepath.Join(t.TempDir(), "subscriber-leases.db")
	sqlitePrimary, err := NewSQLiteSubscriberLeaseStore(sqlitePath)
	if err != nil {
		t.Fatalf("create sqlite subscriber lease store: %v", err)
	}
	t.Cleanup(func() { _ = sqlitePrimary.Close() })
	sqliteSecondary, err := NewSQLiteSubscriberLeaseStore(sqlitePath)
	if err != nil {
		t.Fatalf("reopen sqlite subscriber lease store: %v", err)
	}
	t.Cleanup(func() { _ = sqliteSecondary.Close() })

	return []testSubscriberLeaseStorePair{
		{
			name:      "memory",
			primary:   NewSubscriberLeaseCoordinatorWithStore(memoryStore),
			secondary: NewSubscriberLeaseCoordinatorWithStore(memoryStore),
		},
		{
			name:      "sqlite_shared_durable",
			primary:   sqlitePrimary,
			secondary: sqliteSecondary,
		},
	}
}
