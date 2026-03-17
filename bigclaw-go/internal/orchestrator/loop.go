package orchestrator

import (
	"context"
	"time"

	"bigclaw-go/internal/coordination"
	"bigclaw-go/internal/scheduler"
)

type Runner interface {
	RunOnce(context.Context, scheduler.QuotaSnapshot) bool
}

type Loop struct {
	Runtime         Runner
	Quota           scheduler.QuotaSnapshot
	PollInterval    time.Duration
	LeaderElection  coordination.LeaderElection
	LeaderScope     string
	LeaderCandidate string
	LeaderTTL       time.Duration
	Now             func() time.Time
}

func (l *Loop) Run(ctx context.Context) {
	ticker := time.NewTicker(l.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.runStep(ctx)
		}
	}
}

func (l *Loop) runStep(ctx context.Context) {
	if l.Runtime == nil {
		return
	}
	if !l.shouldRun(ctx) {
		return
	}
	_ = l.Runtime.RunOnce(ctx, l.Quota)
}

func (l *Loop) shouldRun(_ context.Context) bool {
	if l.LeaderElection == nil {
		return true
	}
	now := time.Now().UTC()
	if l.Now != nil {
		now = l.Now().UTC()
	}
	lease, err := l.LeaderElection.Campaign(coordination.CampaignRequest{
		Scope:     l.LeaderScope,
		Candidate: l.LeaderCandidate,
		TTL:       l.LeaderTTL,
		Now:       now,
	})
	if err != nil {
		return false
	}
	return lease.LeaderID == l.LeaderCandidate && coordination.IsLeaderActive(lease, now)
}
