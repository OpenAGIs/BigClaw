package orchestrator

import (
	"context"
	"time"

	"bigclaw-go/internal/scheduler"
)

type Runtime interface {
	RunOnce(context.Context, scheduler.QuotaSnapshot) bool
}

type Loop struct {
	Runtime      Runtime
	Quota        scheduler.QuotaSnapshot
	PollInterval time.Duration
}

func (l *Loop) Run(ctx context.Context) {
	if l.Runtime == nil {
		<-ctx.Done()
		return
	}

	l.runCycle(ctx)
	ticker := time.NewTicker(l.pollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.runCycle(ctx)
		}
	}
}

func (l *Loop) pollInterval() time.Duration {
	if l.PollInterval <= 0 {
		return 100 * time.Millisecond
	}
	return l.PollInterval
}

func (l *Loop) runCycle(ctx context.Context) {
	for ctx.Err() == nil {
		if !l.Runtime.RunOnce(ctx, l.Quota) {
			return
		}
	}
}
