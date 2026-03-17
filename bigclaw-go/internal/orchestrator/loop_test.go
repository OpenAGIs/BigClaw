package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"bigclaw-go/internal/coordination"
	"bigclaw-go/internal/scheduler"
)

type spyRunner struct {
	calls int
}

func (r *spyRunner) RunOnce(_ context.Context, _ scheduler.QuotaSnapshot) bool {
	r.calls++
	return true
}

type stubElection struct {
	lease coordination.LeaderLease
	err   error
}

func (e stubElection) Campaign(_ coordination.CampaignRequest) (coordination.LeaderLease, error) {
	return e.lease, e.err
}

func (e stubElection) Get(_ string) (coordination.LeaderLease, bool) {
	return e.lease, e.lease.Scope != ""
}

func (e stubElection) Resign(_ string, _ string, _ string, _ int64) error {
	return nil
}

func TestLoopRunStepRequiresCoordinatorLeadership(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()

	t.Run("runs when candidate holds active lease", func(t *testing.T) {
		runner := &spyRunner{}
		loop := &Loop{
			Runtime:         runner,
			LeaderElection:  stubElection{lease: coordination.LeaderLease{Scope: "scheduler", LeaderID: "node-a", LeaseToken: "leader-1", ExpiresAt: now.Add(5 * time.Second)}},
			LeaderScope:     "scheduler",
			LeaderCandidate: "node-a",
			LeaderTTL:       5 * time.Second,
			Now:             func() time.Time { return now },
		}
		loop.runStep(context.Background())
		if runner.calls != 1 {
			t.Fatalf("expected runner to execute once, got %d", runner.calls)
		}
	})

	t.Run("skips when lease is held by another node", func(t *testing.T) {
		runner := &spyRunner{}
		loop := &Loop{
			Runtime:         runner,
			LeaderElection:  stubElection{lease: coordination.LeaderLease{Scope: "scheduler", LeaderID: "node-b", LeaseToken: "leader-2", ExpiresAt: now.Add(5 * time.Second)}, err: coordination.ErrLeaderHeld},
			LeaderScope:     "scheduler",
			LeaderCandidate: "node-a",
			LeaderTTL:       5 * time.Second,
			Now:             func() time.Time { return now },
		}
		loop.runStep(context.Background())
		if runner.calls != 0 {
			t.Fatalf("expected runner to be skipped, got %d", runner.calls)
		}
	})

	t.Run("skips on election errors", func(t *testing.T) {
		runner := &spyRunner{}
		loop := &Loop{
			Runtime:         runner,
			LeaderElection:  stubElection{err: errors.New("sqlite unavailable")},
			LeaderScope:     "scheduler",
			LeaderCandidate: "node-a",
			LeaderTTL:       5 * time.Second,
			Now:             func() time.Time { return now },
		}
		loop.runStep(context.Background())
		if runner.calls != 0 {
			t.Fatalf("expected runner to be skipped on election error, got %d", runner.calls)
		}
	})
}
