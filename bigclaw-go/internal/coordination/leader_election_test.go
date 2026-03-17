package coordination

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryLeaderElectionFailoverAndFence(t *testing.T) {
	election := NewLeaderElection()
	now := time.Unix(1_700_000_000, 0).UTC()

	leaseA, err := election.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-a", TTL: 5 * time.Second, Now: now})
	if err != nil {
		t.Fatalf("campaign node-a: %v", err)
	}

	conflict, err := election.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-b", TTL: 5 * time.Second, Now: now.Add(time.Second)})
	if !errors.Is(err, ErrLeaderHeld) {
		t.Fatalf("expected leader held conflict, got lease=%+v err=%v", conflict, err)
	}
	if conflict.LeaderID != "node-a" {
		t.Fatalf("expected conflict to expose node-a, got %+v", conflict)
	}

	leaseB, err := election.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-b", TTL: 5 * time.Second, Now: now.Add(6 * time.Second)})
	if err != nil {
		t.Fatalf("campaign node-b after expiry: %v", err)
	}
	if leaseB.LeaseEpoch != leaseA.LeaseEpoch+1 {
		t.Fatalf("expected lease epoch advance from %d to %d, got %+v", leaseA.LeaseEpoch, leaseA.LeaseEpoch+1, leaseB)
	}
	if err := election.Resign("scheduler", "node-a", leaseA.LeaseToken, leaseA.LeaseEpoch); !errors.Is(err, ErrLeaderFence) {
		t.Fatalf("expected stale resign fence, got %v", err)
	}
	if err := election.Resign("scheduler", "node-b", leaseB.LeaseToken, leaseB.LeaseEpoch); err != nil {
		t.Fatalf("resign active leader: %v", err)
	}
}

func TestSQLiteLeaderElectionSharedStoreFailover(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leader-election.db")
	primary, err := NewSQLiteLeaderElection(path)
	if err != nil {
		t.Fatalf("new sqlite election primary: %v", err)
	}
	defer func() { _ = primary.Close() }()
	secondary, err := NewSQLiteLeaderElection(path)
	if err != nil {
		t.Fatalf("new sqlite election secondary: %v", err)
	}
	defer func() { _ = secondary.Close() }()

	now := time.Unix(1_700_000_000, 0).UTC()
	leaseA, err := primary.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-a", TTL: 5 * time.Second, Now: now})
	if err != nil {
		t.Fatalf("campaign node-a: %v", err)
	}
	conflict, err := secondary.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-b", TTL: 5 * time.Second, Now: now.Add(time.Second)})
	if !errors.Is(err, ErrLeaderHeld) {
		t.Fatalf("expected leader held conflict, got lease=%+v err=%v", conflict, err)
	}
	leaseB, err := secondary.Campaign(CampaignRequest{Scope: "scheduler", Candidate: "node-b", TTL: 5 * time.Second, Now: now.Add(6 * time.Second)})
	if err != nil {
		t.Fatalf("campaign node-b after expiry: %v", err)
	}
	if leaseB.LeaseEpoch != leaseA.LeaseEpoch+1 {
		t.Fatalf("expected epoch advance, got lease-a=%+v lease-b=%+v", leaseA, leaseB)
	}
	if err := primary.Resign("scheduler", "node-a", leaseA.LeaseToken, leaseA.LeaseEpoch); !errors.Is(err, ErrLeaderFence) {
		t.Fatalf("expected stale resign fence, got %v", err)
	}
}
