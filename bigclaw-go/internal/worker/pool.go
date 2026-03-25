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
	reassigned := p.reassignTasksFromDegradedNodes(ctx)
	runtimes := p.runnableWorkers()
	if len(runtimes) == 0 {
		return reassigned
	}

	results := make(chan bool, len(runtimes))
	var group sync.WaitGroup
	group.Add(len(runtimes))

	for _, runtime := range runtimes {
		go func(runtime *Runtime) {
			defer group.Done()
			results <- runtime.RunOnce(ctx, quota)
		}(runtime)
	}

	group.Wait()
	close(results)

	processed := reassigned
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

func (p *Pool) runnableWorkers() []*Runtime {
	degradedNodes := degradedNodeIDs(time.Now(), p.Snapshots())
	if len(degradedNodes) == 0 {
		return append([]*Runtime(nil), p.workers...)
	}
	runtimes := make([]*Runtime, 0, len(p.workers))
	for _, runtime := range p.workers {
		if runtime == nil {
			continue
		}
		if _, degraded := degradedNodes[runtime.NodeID]; degraded {
			runtime.updateStatus(func(status *Status) {
				status.WorkerID = runtime.WorkerID
				status.NodeID = runtime.NodeID
				if status.State == "" || status.State == "idle" {
					status.State = "idle"
				}
				status.LastTransition = "node.degraded"
			})
			continue
		}
		runtimes = append(runtimes, runtime)
	}
	return runtimes
}

func (p *Pool) reassignTasksFromDegradedNodes(ctx context.Context) bool {
	taskInspector, taskReallocator, ok := poolQueueCapabilities(p.workers)
	if !ok {
		return false
	}
	snapshots := p.Snapshots()
	degradedNodes := degradedNodeIDs(time.Now(), snapshots)
	if len(degradedNodes) == 0 {
		return false
	}
	degradedWorkers := make(map[string]Status)
	for _, snapshot := range snapshots {
		if snapshot.WorkerID == "" {
			continue
		}
		if _, degraded := degradedNodes[snapshot.NodeID]; degraded {
			degradedWorkers[snapshot.WorkerID] = snapshot
		}
	}
	if len(degradedWorkers) == 0 {
		return false
	}
	tasks, err := taskInspector.ListTasks(ctx, 0)
	if err != nil {
		return false
	}
	reassigned := false
	for _, snapshot := range tasks {
		if !snapshot.Leased || snapshot.LeaseWorker == "" {
			continue
		}
		workerStatus, degraded := degradedWorkers[snapshot.LeaseWorker]
		if !degraded {
			continue
		}
		if _, err := taskReallocator.ReassignTask(ctx, snapshot.Task.ID, snapshot.LeaseWorker, time.Now(), degradedNodeReassignReason(workerStatus)); err == nil {
			reassigned = true
		}
	}
	return reassigned
}

func poolQueueCapabilities(workers []*Runtime) (queue.TaskInspector, queue.TaskReallocator, bool) {
	for _, runtime := range workers {
		if runtime == nil || runtime.Queue == nil {
			continue
		}
		inspector, ok := runtime.Queue.(queue.TaskInspector)
		if !ok {
			continue
		}
		reallocator, ok := runtime.Queue.(queue.TaskReallocator)
		if !ok {
			continue
		}
		return inspector, reallocator, true
	}
	return nil, nil, false
}

func degradedNodeIDs(now time.Time, snapshots []Status) map[string]struct{} {
	degraded := make(map[string]struct{})
	for _, snapshot := range snapshots {
		nodeID := snapshot.NodeID
		if nodeID == "" {
			nodeID = "unassigned"
		}
		switch {
		case snapshot.LastHeartbeatAt.IsZero():
			degraded[nodeID] = struct{}{}
		case workerHeartbeatAge(now, snapshot) >= workerPoolStaleAfterDuration():
			degraded[nodeID] = struct{}{}
		}
	}
	return degraded
}

func degradedNodeReassignReason(status Status) string {
	nodeID := status.NodeID
	if nodeID == "" {
		nodeID = "unassigned"
	}
	if status.LastHeartbeatAt.IsZero() {
		return fmt.Sprintf("node %s degraded: worker %s is missing heartbeat", nodeID, status.WorkerID)
	}
	age := workerHeartbeatAge(time.Now(), status).Round(time.Second)
	return fmt.Sprintf("node %s degraded: worker %s heartbeat stale for %s", nodeID, status.WorkerID, age)
}

func workerPoolStaleAfterDuration() time.Duration {
	return 5 * time.Minute
}

func workerHeartbeatAge(now time.Time, status Status) time.Duration {
	if status.LastHeartbeatAt.IsZero() {
		return 0
	}
	age := now.Sub(status.LastHeartbeatAt)
	if age < 0 {
		return 0
	}
	return age
}
