package events

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrLeaseHeld          = errors.New("subscriber checkpoint lease held by another consumer")
	ErrLeaseExpired       = errors.New("subscriber checkpoint lease expired")
	ErrLeaseFence         = errors.New("subscriber checkpoint lease fenced")
	ErrCheckpointRollback = errors.New("subscriber checkpoint rollback rejected")
)

type SubscriberLeaseKey struct {
	GroupID      string
	SubscriberID string
}

type SubscriberLease struct {
	GroupID          string    `json:"group_id"`
	SubscriberID     string    `json:"subscriber_id"`
	ConsumerID       string    `json:"consumer_id"`
	LeaseToken       string    `json:"lease_token"`
	LeaseEpoch       int64     `json:"lease_epoch"`
	CheckpointOffset uint64    `json:"checkpoint_offset"`
	CheckpointEvent  string    `json:"checkpoint_event_id,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type subscriberLeaseState string

const (
	subscriberLeaseStateVacant  subscriberLeaseState = "vacant"
	subscriberLeaseStateActive  subscriberLeaseState = "active"
	subscriberLeaseStateExpired subscriberLeaseState = "expired"
)

type subscriberLeaseAction string

const (
	subscriberLeaseActionAcquire subscriberLeaseAction = "acquire"
	subscriberLeaseActionCommit  subscriberLeaseAction = "commit"
	subscriberLeaseActionRelease subscriberLeaseAction = "release"
)

type LeaseRequest struct {
	GroupID      string
	SubscriberID string
	ConsumerID   string
	TTL          time.Duration
	Now          time.Time
}

type CheckpointCommit struct {
	GroupID          string
	SubscriberID     string
	ConsumerID       string
	LeaseToken       string
	LeaseEpoch       int64
	CheckpointOffset uint64
	CheckpointEvent  string
	Now              time.Time
}

type SubscriberLeaseStore interface {
	Acquire(request LeaseRequest) (SubscriberLease, error)
	Commit(request CheckpointCommit) (SubscriberLease, error)
	Get(groupID string, subscriberID string) (SubscriberLease, bool)
	Release(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64) error
}

type SubscriberLeaseCoordinator struct {
	store SubscriberLeaseStore
}

func NewSubscriberLeaseCoordinator() *SubscriberLeaseCoordinator {
	return NewSubscriberLeaseCoordinatorWithStore(newMemorySubscriberLeaseStore())
}

func NewSubscriberLeaseCoordinatorWithStore(store SubscriberLeaseStore) *SubscriberLeaseCoordinator {
	if store == nil {
		store = newMemorySubscriberLeaseStore()
	}
	return &SubscriberLeaseCoordinator{store: store}
}

func (c *SubscriberLeaseCoordinator) Acquire(request LeaseRequest) (SubscriberLease, error) {
	return c.store.Acquire(request)
}

func (c *SubscriberLeaseCoordinator) Commit(request CheckpointCommit) (SubscriberLease, error) {
	return c.store.Commit(request)
}

func (c *SubscriberLeaseCoordinator) Get(groupID string, subscriberID string) (SubscriberLease, bool) {
	return c.store.Get(groupID, subscriberID)
}

func (c *SubscriberLeaseCoordinator) Release(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64) error {
	return c.store.Release(groupID, subscriberID, consumerID, leaseToken, leaseEpoch)
}

type memorySubscriberLeaseStore struct {
	mu      sync.Mutex
	leases  map[SubscriberLeaseKey]SubscriberLease
	counter uint64
}

func newMemorySubscriberLeaseStore() *memorySubscriberLeaseStore {
	return &memorySubscriberLeaseStore{
		leases: make(map[SubscriberLeaseKey]SubscriberLease),
	}
}

func (c *memorySubscriberLeaseStore) Acquire(request LeaseRequest) (SubscriberLease, error) {
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

	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.leases[key]
	switch current.stateAt(request.Now) {
	case subscriberLeaseStateActive:
		if current.ConsumerID == request.ConsumerID {
			current.ExpiresAt = request.Now.Add(request.TTL)
			current.UpdatedAt = request.Now
			c.leases[key] = current
			return current, nil
		}
		return current, ErrLeaseHeld
	}

	next := SubscriberLease{
		GroupID:          request.GroupID,
		SubscriberID:     request.SubscriberID,
		ConsumerID:       request.ConsumerID,
		LeaseToken:       c.nextToken(),
		LeaseEpoch:       current.LeaseEpoch + 1,
		CheckpointOffset: current.CheckpointOffset,
		CheckpointEvent:  current.CheckpointEvent,
		ExpiresAt:        request.Now.Add(request.TTL),
		UpdatedAt:        request.Now,
	}
	c.leases[key] = next
	return next, nil
}

func (c *memorySubscriberLeaseStore) Commit(request CheckpointCommit) (SubscriberLease, error) {
	if request.Now.IsZero() {
		request.Now = time.Now()
	}
	key := SubscriberLeaseKey{GroupID: request.GroupID, SubscriberID: request.SubscriberID}

	c.mu.Lock()
	defer c.mu.Unlock()

	current, ok := c.leases[key]
	if !ok {
		return SubscriberLease{}, ErrLeaseExpired
	}
	if err := current.validateAction(subscriberLeaseActionCommit, request.Now, request.ConsumerID, request.LeaseToken, request.LeaseEpoch); err != nil {
		return current, errors.Unwrap(err)
	}
	if request.CheckpointOffset < current.CheckpointOffset {
		return current, ErrCheckpointRollback
	}

	current.CheckpointOffset = request.CheckpointOffset
	current.CheckpointEvent = request.CheckpointEvent
	current.UpdatedAt = request.Now
	c.leases[key] = current
	return current, nil
}

func (c *memorySubscriberLeaseStore) Get(groupID string, subscriberID string) (SubscriberLease, bool) {
	key := SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID}
	c.mu.Lock()
	defer c.mu.Unlock()
	lease, ok := c.leases[key]
	return lease, ok
}

func (c *memorySubscriberLeaseStore) Release(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64) error {
	return c.releaseAt(groupID, subscriberID, consumerID, leaseToken, leaseEpoch, time.Now().UTC())
}

func (c *memorySubscriberLeaseStore) releaseAt(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64, now time.Time) error {
	key := SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID}
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.leases[key]
	if !ok {
		return ErrLeaseExpired
	}
	if err := current.validateAction(subscriberLeaseActionRelease, now, consumerID, leaseToken, leaseEpoch); err != nil {
		return errors.Unwrap(err)
	}
	current.ConsumerID = ""
	current.LeaseToken = ""
	current.ExpiresAt = time.Time{}
	current.UpdatedAt = now
	c.leases[key] = current
	return nil
}

func (c *memorySubscriberLeaseStore) nextToken() string {
	c.counter++
	return fmt.Sprintf("lease-%d", c.counter)
}

func leaseExpired(lease SubscriberLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}

func (lease SubscriberLease) stateAt(now time.Time) subscriberLeaseState {
	if lease.ConsumerID == "" || lease.LeaseToken == "" {
		return subscriberLeaseStateVacant
	}
	if leaseExpired(lease, now) {
		return subscriberLeaseStateExpired
	}
	return subscriberLeaseStateActive
}

func (lease SubscriberLease) validateAction(action subscriberLeaseAction, now time.Time, consumerID string, leaseToken string, leaseEpoch int64) error {
	if lease.stateAt(now) != subscriberLeaseStateActive {
		return fmt.Errorf("%s: %w", action, ErrLeaseExpired)
	}
	if lease.ConsumerID != consumerID || lease.LeaseToken != leaseToken || lease.LeaseEpoch != leaseEpoch {
		return fmt.Errorf("%s: %w", action, ErrLeaseFence)
	}
	return nil
}
