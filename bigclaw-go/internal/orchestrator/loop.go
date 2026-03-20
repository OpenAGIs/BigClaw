package orchestrator

import (
	"context"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type Loop struct {
	Runtime      *worker.Runtime
	Quota        scheduler.QuotaSnapshot
	PollInterval time.Duration
}

func (l *Loop) Run(ctx context.Context) {
	if l == nil {
		<-ctx.Done()
		return
	}

	interval := l.PollInterval
	if interval <= 0 {
		interval = time.Second
	}

	l.RunTick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.RunTick(ctx)
		}
	}
}

func (l *Loop) RunTick(ctx context.Context) int {
	if l == nil || l.Runtime == nil || l.Runtime.Queue == nil {
		return 0
	}

	iterations := l.Runtime.Queue.Size(ctx)
	if iterations < 1 {
		iterations = 1
	}

	processed := 0
	for attempts := 0; attempts < iterations; attempts++ {
		if ctx.Err() != nil {
			break
		}
		if !l.Runtime.RunOnce(ctx, l.quotaSnapshot(ctx)) {
			break
		}
		processed++
	}
	return processed
}

func (l *Loop) quotaSnapshot(ctx context.Context) scheduler.QuotaSnapshot {
	quota := l.Quota
	if l == nil || l.Runtime == nil || l.Runtime.Queue == nil {
		return quota
	}

	quota.QueueDepth = l.Runtime.Queue.Size(ctx)
	status := l.Runtime.Snapshot()
	if status.State == "leased" || status.State == "running" {
		quota.CurrentRunning = 1
	}

	inspector, ok := l.Runtime.Queue.(queue.TaskInspector)
	if !ok {
		return quota
	}

	snapshots, err := inspector.ListTasks(ctx, 0)
	if err != nil {
		return quota
	}

	quota.QueueDepth = 0
	quota.CurrentRunning = 0
	quota.PreemptibleExecutions = 0
	urgentThreshold := l.Runtime.Scheduler.Rules().UrgentPriorityThreshold
	for _, snapshot := range snapshots {
		if !isQuotaActionable(snapshot.Task.State) {
			continue
		}
		quota.QueueDepth++
		if !snapshot.Leased {
			continue
		}
		quota.CurrentRunning++
		if snapshot.Task.Priority > urgentThreshold {
			quota.PreemptibleExecutions++
		}
	}
	return quota
}

func isQuotaActionable(state domain.TaskState) bool {
	switch state {
	case domain.TaskBlocked, domain.TaskCancelled, domain.TaskDeadLetter:
		return false
	default:
		return true
	}
}
