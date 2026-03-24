package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"bigclaw-go/internal/queue"
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
	p.runHealthProbe(ctx)

	results := make(chan bool, len(p.workers))
	var group sync.WaitGroup
	group.Add(len(p.workers))

	for _, runtime := range p.workers {
		go func(runtime *Runtime) {
			defer group.Done()
			results <- runtime.RunOnce(ctx, quota)
		}(runtime)
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

func (p *Pool) Snapshot() Status {
	snapshots := p.Snapshots()
	if len(snapshots) == 0 {
		return Status{WorkerID: "worker-pool", State: "idle"}
	}

	summary := Status{WorkerID: "worker-pool", State: "idle"}
	for _, snapshot := range snapshots {
		summary.HealthProbeRuns += snapshot.HealthProbeRuns
		summary.HealthProbeRecoveries += snapshot.HealthProbeRecoveries
		summary.HealthProbePurges += snapshot.HealthProbePurges
		summary.HealthProbeFailures += snapshot.HealthProbeFailures
		if snapshot.StaleHeartbeatWorkers > summary.StaleHeartbeatWorkers {
			summary.StaleHeartbeatWorkers = snapshot.StaleHeartbeatWorkers
		}
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
		if snapshot.LastHealthProbeAt.After(summary.LastHealthProbeAt) {
			summary.LastHealthProbeAt = snapshot.LastHealthProbeAt
			summary.LastHealthProbeResult = snapshot.LastHealthProbeResult
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

func (p *Pool) runHealthProbe(ctx context.Context) {
	now := time.Now()
	staleWorkers := 0
	for _, runtime := range p.workers {
		if runtime == nil {
			continue
		}
		snapshot := runtime.Snapshot()
		if snapshot.LastHeartbeatAt.IsZero() {
			continue
		}
		if age := now.Sub(snapshot.LastHeartbeatAt); age >= 5*time.Minute {
			staleWorkers++
		}
	}

	result := queue.LeaseRecoveryResult{}
	var probeErr error
	if recoverer, ok := p.sharedLeaseRecoverer(); ok {
		result, probeErr = recoverer.RecoverExpiredLeases(ctx, now)
	}

	message := "ok"
	switch {
	case probeErr != nil:
		message = fmt.Sprintf("probe_failed: %v", probeErr)
	case result.Recovered > 0 || result.Purged > 0:
		message = fmt.Sprintf("redispatched=%d purged=%d stale_workers=%d", result.Recovered, result.Purged, staleWorkers)
	default:
		message = fmt.Sprintf("healthy stale_workers=%d", staleWorkers)
	}

	for index, runtime := range p.workers {
		if runtime == nil {
			continue
		}
		runtime.updateStatus(func(status *Status) {
			status.LastHealthProbeAt = now
			status.LastHealthProbeResult = message
			status.StaleHeartbeatWorkers = staleWorkers
			if index != 0 {
				return
			}
			status.HealthProbeRuns++
			if probeErr != nil {
				status.HealthProbeFailures++
				return
			}
			status.HealthProbeRecoveries += result.Recovered
			status.HealthProbePurges += result.Purged
		})
	}
}

func (p *Pool) sharedLeaseRecoverer() (queue.LeaseRecoverer, bool) {
	for _, runtime := range p.workers {
		if runtime == nil || runtime.Queue == nil {
			continue
		}
		recoverer, ok := runtime.Queue.(queue.LeaseRecoverer)
		if ok {
			return recoverer, true
		}
		break
	}
	return nil, false
}
