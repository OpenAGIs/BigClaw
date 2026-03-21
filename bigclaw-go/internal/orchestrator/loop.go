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
	ticker := time.NewTicker(l.PollInterval)
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
