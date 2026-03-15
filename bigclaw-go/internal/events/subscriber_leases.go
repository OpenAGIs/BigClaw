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

type SubscriberLeaseCoordinator struct {
	mu      sync.Mutex
	leases  map[SubscriberLeaseKey]SubscriberLease
	counter uint64
}

func NewSubscriberLeaseCoordinator() *SubscriberLeaseCoordinator {
	return &SubscriberLeaseCoordinator{
		leases: make(map[SubscriberLeaseKey]SubscriberLease),
	}
}

func (c *SubscriberLeaseCoordinator) Acquire(request LeaseRequest) (SubscriberLease, error) {
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

func (c *SubscriberLeaseCoordinator) Commit(request CheckpointCommit) (SubscriberLease, error) {
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

func (c *SubscriberLeaseCoordinator) Get(groupID string, subscriberID string) (SubscriberLease, bool) {
	key := SubscriberLeaseKey{GroupID: groupID, SubscriberID: subscriberID}
	c.mu.Lock()
	defer c.mu.Unlock()
	lease, ok := c.leases[key]
	return lease, ok
}

func (c *SubscriberLeaseCoordinator) Release(groupID string, subscriberID string, consumerID string, leaseToken string, leaseEpoch int64) error {
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

func (c *SubscriberLeaseCoordinator) nextToken() string {
	c.counter++
	return fmt.Sprintf("lease-%d", c.counter)
}

func leaseExpired(lease SubscriberLease, now time.Time) bool {
	return !lease.ExpiresAt.IsZero() && !now.Before(lease.ExpiresAt)
}
