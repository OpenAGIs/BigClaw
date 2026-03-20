package orchestrator

import (
	"context"
	"time"

	"bigclaw-go/internal/scheduler"
)

type runtimeRunner interface {
	RunOnce(context.Context, scheduler.QuotaSnapshot) bool
}

type Loop struct {
	Runtime      runtimeRunner
	Quota        scheduler.QuotaSnapshot
	PollInterval time.Duration
}

func (l *Loop) Run(ctx context.Context) {
	if l == nil || l.Runtime == nil {
		return
	}

	_ = l.Runtime.RunOnce(ctx, l.Quota)
	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(l.pollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = l.Runtime.RunOnce(ctx, l.Quota)
		}
	}
}

func (l *Loop) pollInterval() time.Duration {
	if l == nil || l.PollInterval <= 0 {
		return 100 * time.Millisecond
	}
	return l.PollInterval
}
