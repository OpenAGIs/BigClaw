package coordination

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrLeaderHeld    = errors.New("coordinator leader lease held by another candidate")
	ErrLeaderExpired = errors.New("coordinator leader lease expired")
	ErrLeaderFence   = errors.New("coordinator leader lease fenced")
)

type LeaderLease struct {
	Scope      string    `json:"scope"`
	LeaderID   string    `json:"leader_id"`
	LeaseToken string    `json:"lease_token"`
	LeaseEpoch int64     `json:"lease_epoch"`
	ExpiresAt  time.Time `json:"expires_at"`
	AcquiredAt time.Time `json:"acquired_at"`
	RenewedAt  time.Time `json:"renewed_at"`
}

type CampaignRequest struct {
	Scope     string
	Candidate string
	TTL       time.Duration
	Now       time.Time
}

type LeaderElection interface {
	Campaign(request CampaignRequest) (LeaderLease, error)
	Get(scope string) (LeaderLease, bool)
	Resign(scope string, candidate string, leaseToken string, leaseEpoch int64) error
}

func IsLeaderActive(lease LeaderLease, now time.Time) bool {
	if lease.Scope == "" || lease.LeaderID == "" || lease.LeaseToken == "" || lease.ExpiresAt.IsZero() {
		return false
	}
	return now.Before(lease.ExpiresAt)
}

type memoryLeaderElection struct {
	mu      sync.Mutex
	leases  map[string]LeaderLease
	counter uint64
}

func NewLeaderElection() LeaderElection {
	return &memoryLeaderElection{leases: make(map[string]LeaderLease)}
}

func (e *memoryLeaderElection) Campaign(request CampaignRequest) (LeaderLease, error) {
	if request.Scope == "" || request.Candidate == "" {
		return LeaderLease{}, fmt.Errorf("scope and candidate are required")
	}
	if request.TTL <= 0 {
		return LeaderLease{}, fmt.Errorf("ttl must be positive")
	}
	if request.Now.IsZero() {
		request.Now = time.Now().UTC()
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	current, ok := e.leases[request.Scope]
	if ok && IsLeaderActive(current, request.Now) && current.LeaderID != request.Candidate {
		return current, ErrLeaderHeld
	}
	if ok && IsLeaderActive(current, request.Now) && current.LeaderID == request.Candidate {
		current.ExpiresAt = request.Now.Add(request.TTL)
		current.RenewedAt = request.Now
		e.leases[request.Scope] = current
		return current, nil
	}

	next := LeaderLease{
		Scope:      request.Scope,
		LeaderID:   request.Candidate,
		LeaseToken: e.nextToken(),
		LeaseEpoch: current.LeaseEpoch + 1,
		ExpiresAt:  request.Now.Add(request.TTL),
		AcquiredAt: request.Now,
		RenewedAt:  request.Now,
	}
	e.leases[request.Scope] = next
	return next, nil
}

func (e *memoryLeaderElection) Get(scope string) (LeaderLease, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	lease, ok := e.leases[scope]
	return lease, ok
}

func (e *memoryLeaderElection) Resign(scope string, candidate string, leaseToken string, leaseEpoch int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	current, ok := e.leases[scope]
	if !ok {
		return ErrLeaderExpired
	}
	if current.LeaderID != candidate || current.LeaseToken != leaseToken || current.LeaseEpoch != leaseEpoch {
		return ErrLeaderFence
	}
	delete(e.leases, scope)
	return nil
}

func (e *memoryLeaderElection) nextToken() string {
	e.counter++
	return fmt.Sprintf("leader-%d", e.counter)
}
