package orchestrator

import (
	"context"
	"time"

	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

type runtimeRunner interface {
	RunOnce(context.Context, scheduler.QuotaSnapshot) bool
}

type quotaSource interface {
	Snapshot(context.Context, scheduler.QuotaSnapshot) scheduler.QuotaSnapshot
}

type Loop struct {
	Runtime      runtimeRunner
	Quota        scheduler.QuotaSnapshot
	QuotaSource  quotaSource
	PollInterval time.Duration
}

func (l *Loop) Run(ctx context.Context) {
	if l == nil || l.Runtime == nil {
		return
	}

	_ = l.Runtime.RunOnce(ctx, l.currentQuota(ctx))
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
			_ = l.Runtime.RunOnce(ctx, l.currentQuota(ctx))
		}
	}
}

func (l *Loop) currentQuota(ctx context.Context) scheduler.QuotaSnapshot {
	if l == nil {
		return scheduler.QuotaSnapshot{}
	}
	if l.QuotaSource == nil {
		return l.Quota
	}
	return l.QuotaSource.Snapshot(ctx, l.Quota)
}

func (l *Loop) pollInterval() time.Duration {
	if l == nil || l.PollInterval <= 0 {
		return 100 * time.Millisecond
	}
	return l.PollInterval
}

type QueueQuotaSource struct {
	Queue queue.Queue
}

func (s QueueQuotaSource) Snapshot(ctx context.Context, base scheduler.QuotaSnapshot) scheduler.QuotaSnapshot {
	if s.Queue == nil {
		return base
	}
	quota := base
	quota.QueueDepth = s.Queue.Size(ctx)
	inspector, ok := s.Queue.(queue.TaskInspector)
	if !ok {
		return quota
	}
	snapshots, err := inspector.ListTasks(ctx, 0)
	if err != nil {
		return quota
	}
	running := 0
	for _, snapshot := range snapshots {
		if snapshot.Leased {
			running++
		}
	}
	quota.CurrentRunning = running
	return quota
}
