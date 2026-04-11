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

type LeaseLifecycleState string

const (
	LeaseLifecycleIdle     LeaseLifecycleState = "idle"
	LeaseLifecycleHeld     LeaseLifecycleState = "held"
	LeaseLifecycleExpired  LeaseLifecycleState = "expired"
	LeaseLifecycleReleased LeaseLifecycleState = "released"
	LeaseLifecycleFenced   LeaseLifecycleState = "fenced"
)

type LeaseLifecycleTransition struct {
	From        LeaseLifecycleState `json:"from"`
	Action      string              `json:"action"`
	To          LeaseLifecycleState `json:"to"`
	Guard       string              `json:"guard,omitempty"`
	Description string              `json:"description,omitempty"`
}

type LeaseLifecycleModel struct {
	RenewalMechanism string                     `json:"renewal_mechanism"`
	ReleaseMechanism string                     `json:"release_mechanism"`
	States           []LeaseLifecycleState      `json:"states"`
	Transitions      []LeaseLifecycleTransition `json:"transitions"`
}

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

func SubscriberLeaseLifecycleModelSpec() LeaseLifecycleModel {
	return LeaseLifecycleModel{
		RenewalMechanism: "same consumer reacquires the lease before expiry to extend ttl without advancing epoch",
		ReleaseMechanism: "active owner must present matching consumer_id, lease_token, and lease_epoch",
		States: []LeaseLifecycleState{
			LeaseLifecycleIdle,
			LeaseLifecycleHeld,
			LeaseLifecycleExpired,
			LeaseLifecycleReleased,
			LeaseLifecycleFenced,
		},
		Transitions: []LeaseLifecycleTransition{
			{
				From:        LeaseLifecycleIdle,
				Action:      "acquire",
				To:          LeaseLifecycleHeld,
				Guard:       "group_id, subscriber_id, consumer_id present and ttl > 0",
				Description: "first owner acquires a fresh lease token and epoch",
			},
			{
				From:        LeaseLifecycleHeld,
				Action:      "renew",
				To:          LeaseLifecycleHeld,
				Guard:       "same consumer reacquires before ttl expiry",
				Description: "ttl extends in place while lease token and epoch remain stable",
			},
			{
				From:        LeaseLifecycleHeld,
				Action:      "commit",
				To:          LeaseLifecycleHeld,
				Guard:       "matching token/epoch and monotonic checkpoint offset",
				Description: "checkpoint advances while the same owner still holds the lease",
			},
			{
				From:        LeaseLifecycleHeld,
				Action:      "release",
				To:          LeaseLifecycleReleased,
				Guard:       "matching consumer_id, lease_token, and lease_epoch",
				Description: "active owner releases the lease cleanly",
			},
			{
				From:        LeaseLifecycleHeld,
				Action:      "expire",
				To:          LeaseLifecycleExpired,
				Guard:       "ttl elapses before successful renew or release",
				Description: "ownership window closes and the lease becomes takeover-eligible",
			},
			{
				From:        LeaseLifecycleExpired,
				Action:      "takeover",
				To:          LeaseLifecycleHeld,
				Guard:       "different consumer acquires after expiry",
				Description: "new owner receives a fresh token and incremented epoch",
			},
			{
				From:        LeaseLifecycleHeld,
				Action:      "stale_release_or_commit",
				To:          LeaseLifecycleFenced,
				Guard:       "consumer_id, token, or epoch no longer matches the active owner",
				Description: "stale writer is rejected to prevent dual execution or checkpoint rollback",
			},
			{
				From:        LeaseLifecycleReleased,
				Action:      "acquire",
				To:          LeaseLifecycleHeld,
				Guard:       "a consumer requests a new lease after release",
				Description: "released slots can be reacquired without waiting for ttl expiry",
			},
		},
	}
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

	current, ok := c.leases[key]
	if ok && !leaseExpired(current, request.Now) && current.ConsumerID != request.ConsumerID {
		return current, ErrLeaseHeld
	}
	if ok && leaseExpired(current, request.Now) && current.ConsumerID != request.ConsumerID {
		current.LeaseToken = ""
	}

	if ok && !leaseExpired(current, request.Now) && current.ConsumerID == request.ConsumerID {
		current.ExpiresAt = request.Now.Add(request.TTL)
		current.UpdatedAt = request.Now
		c.leases[key] = current
		return current, nil
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
	if leaseExpired(current, request.Now) {
		return current, ErrLeaseExpired
	}
	if current.ConsumerID != request.ConsumerID || current.LeaseToken != request.LeaseToken || current.LeaseEpoch != request.LeaseEpoch {
		return current, ErrLeaseFence
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
	key := SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID}
	c.mu.Lock()
	defer c.mu.Unlock()
	current, ok := c.leases[key]
	if !ok {
		return ErrLeaseExpired
	}
	if current.ConsumerID != consumerID || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
		return ErrLeaseFence
	}
	delete(c.leases, key)
	return nil
}

func (c *memorySubscriberLeaseStore) nextToken() string {
	c.counter++
	return fmt.Sprintf("lease-%d", c.counter)
}

func leaseExpired(lease SubscriberLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}
