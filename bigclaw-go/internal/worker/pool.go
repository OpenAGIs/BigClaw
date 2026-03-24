package worker

import (
	"context"
	"sync"

	"bigclaw-go/internal/scheduler"
)

type Pool struct {
	workers []*Runtime
}

func NewPool(workers ...*Runtime) *Pool {
	filtered := make([]*Runtime, 0, len(workers))
	for _, runtime := range workers {
		if runtime != nil {
			filtered = append(filtered, runtime)
		}
	}
	return &Pool{workers: filtered}
}

func (p *Pool) RunOnce(ctx context.Context, quota scheduler.QuotaSnapshot) bool {
	if len(p.workers) == 0 {
		return false
	}

	results := make(chan bool, len(p.workers))
	var group sync.WaitGroup
	remainingNormal := availableConcurrency(quota)
	remainingPreemptible := availablePreemptible(quota)
	dispatched := 0

	for _, runtime := range p.workers {
		if !canDispatchWorker(quota, remainingNormal, remainingPreemptible) {
			break
		}
		assignedQuota := quotaForWorker(quota, remainingNormal, remainingPreemptible)
		dispatched++
		group.Add(1)
		go func(runtime *Runtime) {
			defer group.Done()
			results <- runtime.RunOnce(ctx, assignedQuota)
		}(runtime)
		if quota.ConcurrentLimit > 0 {
			if remainingNormal > 0 {
				remainingNormal--
			} else if remainingPreemptible > 0 {
				remainingPreemptible--
			}
		}
	}

	if dispatched == 0 {
		return false
	}

	group.Wait()
	close(results)

	processed := false
	for result := range results {
		if result {
			processed = true
		}
	}

	return processed
}

func availableConcurrency(quota scheduler.QuotaSnapshot) int {
	if quota.ConcurrentLimit <= 0 {
		return -1
	}
	remaining := quota.ConcurrentLimit - quota.CurrentRunning
	if remaining < 0 {
		return 0
	}
	return remaining
}

func availablePreemptible(quota scheduler.QuotaSnapshot) int {
	if quota.PreemptibleExecutions < 0 {
		return 0
	}
	return quota.PreemptibleExecutions
}

func canDispatchWorker(quota scheduler.QuotaSnapshot, remainingNormal, remainingPreemptible int) bool {
	if quota.ConcurrentLimit <= 0 {
		return true
	}
	return remainingNormal > 0 || remainingPreemptible > 0
}

func quotaForWorker(quota scheduler.QuotaSnapshot, remainingNormal, remainingPreemptible int) scheduler.QuotaSnapshot {
	assigned := quota
	if quota.ConcurrentLimit <= 0 {
		return assigned
	}

	consumedNormal := quota.ConcurrentLimit - quota.CurrentRunning - remainingNormal
	if consumedNormal < 0 {
		consumedNormal = 0
	}
	assigned.CurrentRunning = quota.CurrentRunning + consumedNormal
	if assigned.CurrentRunning > quota.ConcurrentLimit {
		assigned.CurrentRunning = quota.ConcurrentLimit
	}
	assigned.PreemptibleExecutions = remainingPreemptible
	return assigned
}

func (p *Pool) Snapshot() Status {
	snapshots := p.Snapshots()
	if len(snapshots) == 0 {
		return Status{WorkerID: "worker-pool", State: "idle"}
	}

	summary := Status{WorkerID: "worker-pool", State: "idle"}
	for _, snapshot := range snapshots {
		summary.LeaseRenewals += snapshot.LeaseRenewals
		summary.LeaseRenewalFailures += snapshot.LeaseRenewalFailures
		summary.LeaseLostRuns += snapshot.LeaseLostRuns
		summary.SuccessfulRuns += snapshot.SuccessfulRuns
		summary.RetriedRuns += snapshot.RetriedRuns
		summary.DeadLetterRuns += snapshot.DeadLetterRuns
		summary.CancelledRuns += snapshot.CancelledRuns
		summary.PreemptionsIssued += snapshot.PreemptionsIssued

		if snapshot.LastStartedAt.After(summary.LastStartedAt) {
			summary.LastStartedAt = snapshot.LastStartedAt
		}
		if snapshot.LastFinishedAt.After(summary.LastFinishedAt) {
			summary.LastFinishedAt = snapshot.LastFinishedAt
			summary.LastResult = snapshot.LastResult
			summary.LastTransition = snapshot.LastTransition
		}
		if snapshot.LastHeartbeatAt.After(summary.LastHeartbeatAt) {
			summary.LastHeartbeatAt = snapshot.LastHeartbeatAt
		}
		if summary.CurrentTaskID == "" && (snapshot.State == "leased" || snapshot.State == "running") {
			summary.State = snapshot.State
			summary.CurrentTaskID = snapshot.CurrentTaskID
			summary.CurrentTraceID = snapshot.CurrentTraceID
			summary.CurrentExecutor = snapshot.CurrentExecutor
		}
		if !summary.PreemptionActive && snapshot.PreemptionActive {
			summary.PreemptionActive = true
			summary.CurrentPreemptionTaskID = snapshot.CurrentPreemptionTaskID
			summary.CurrentPreemptionWorkerID = snapshot.CurrentPreemptionWorkerID
			summary.LastPreemptedTaskID = snapshot.LastPreemptedTaskID
			summary.LastPreemptionAt = snapshot.LastPreemptionAt
			summary.LastPreemptionReason = snapshot.LastPreemptionReason
		}
		if snapshot.State == "paused" && summary.State == "idle" {
			summary.State = "paused"
		}
	}

	return summary
}

func (p *Pool) Snapshots() []Status {
	snapshots := make([]Status, 0, len(p.workers))
	for _, runtime := range p.workers {
		if runtime != nil {
			snapshots = append(snapshots, runtime.Snapshot())
		}
	}
	return snapshots
}
