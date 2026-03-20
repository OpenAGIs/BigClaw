package worker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

type Runtime struct {
	WorkerID    string
	Queue       queue.Queue
	Scheduler   *scheduler.Scheduler
	Registry    *executor.Registry
	Bus         *events.Bus
	Recorder    *observability.Recorder
	Control     *control.Controller
	LeaseTTL    time.Duration
	TaskTimeout time.Duration
	statusMu    sync.Mutex
	status      Status
}

type Status struct {
	WorkerID                  string              `json:"worker_id"`
	State                     string              `json:"state"`
	CurrentTaskID             string              `json:"current_task_id,omitempty"`
	CurrentTraceID            string              `json:"current_trace_id,omitempty"`
	CurrentExecutor           domain.ExecutorKind `json:"current_executor,omitempty"`
	LastHeartbeatAt           time.Time           `json:"last_heartbeat_at,omitempty"`
	LastStartedAt             time.Time           `json:"last_started_at,omitempty"`
	LastFinishedAt            time.Time           `json:"last_finished_at,omitempty"`
	LastResult                string              `json:"last_result,omitempty"`
	LeaseRenewals             int                 `json:"lease_renewals"`
	SuccessfulRuns            int                 `json:"successful_runs"`
	RetriedRuns               int                 `json:"retried_runs"`
	DeadLetterRuns            int                 `json:"dead_letter_runs"`
	CancelledRuns             int                 `json:"cancelled_runs"`
	PreemptionActive          bool                `json:"preemption_active,omitempty"`
	CurrentPreemptionTaskID   string              `json:"current_preemption_task_id,omitempty"`
	CurrentPreemptionWorkerID string              `json:"current_preemption_worker_id,omitempty"`
	LastPreemptedTaskID       string              `json:"last_preempted_task_id,omitempty"`
	LastPreemptionAt          time.Time           `json:"last_preemption_at,omitempty"`
	LastPreemptionReason      string              `json:"last_preemption_reason,omitempty"`
	PreemptionsIssued         int                 `json:"preemptions_issued,omitempty"`
	LastTransition            string              `json:"last_transition,omitempty"`
}

type cancellationWatcher struct {
	done      chan struct{}
	mu        sync.Mutex
	snapshot  queue.TaskSnapshot
	cancelled bool
}

func (r *Runtime) Snapshot() Status {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	snapshot := r.status
	if snapshot.WorkerID == "" {
		snapshot.WorkerID = r.WorkerID
	}
	if snapshot.State == "" {
		snapshot.State = "idle"
	}
	return snapshot
}

func (r *Runtime) RunOnce(ctx context.Context, quota scheduler.QuotaSnapshot) bool {
	if r.Control != nil && r.Control.IsPaused() {
		r.updateStatus(func(status *Status) {
			status.WorkerID = r.WorkerID
			status.State = "paused"
			status.LastTransition = string(domain.EventControlPaused)
		})
		return false
	}

	task, lease, err := r.Queue.LeaseNext(ctx, r.WorkerID, r.LeaseTTL)
	if err != nil || task == nil || lease == nil {
		r.updateStatus(func(status *Status) {
			status.WorkerID = r.WorkerID
			if status.State == "" || status.State == "paused" {
				status.State = "idle"
			}
		})
		return false
	}
	if task.TraceID == "" {
		task.TraceID = task.ID
	}
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
	if r.Control != nil {
		takeover, ok := r.Control.TakeoverStatus(task.ID)
		if ok && takeover.Active {
			blockedAt := time.Now()
			if controller, ok := r.Queue.(queue.TaskController); ok {
				snapshot, err := controller.UpdateTaskState(ctx, task.ID, domain.TaskBlocked, blockedAt, "automation deferred while human takeover is active")
				if err == nil {
					task = &snapshot.Task
				}
			} else {
				_ = r.Queue.Requeue(ctx, lease, blockedAt.Add(250*time.Millisecond))
				task.State = domain.TaskBlocked
				task.UpdatedAt = blockedAt
			}
			if r.Recorder != nil {
				r.Recorder.StoreTask(*task)
			}
			r.finishStatus(string(domain.EventRunTakeover), "task under human takeover", nil)
			r.publish(domain.Event{
				ID:        eventID(task.ID, "takeover-hold"),
				Type:      domain.EventRunAnnotated,
				TaskID:    task.ID,
				TraceID:   task.TraceID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"message":  "automation deferred while human takeover is active",
					"owner":    takeover.Owner,
					"reviewer": takeover.Reviewer,
				},
			})
			return true
		}
	}

	now := time.Now()
	r.updateStatus(func(status *Status) {
		status.WorkerID = r.WorkerID
		status.State = "leased"
		status.CurrentTaskID = task.ID
		status.CurrentTraceID = task.TraceID
		status.LastTransition = string(domain.EventTaskLeased)
		status.LastHeartbeatAt = now
	})

	r.publish(domain.Event{ID: eventID(task.ID, "leased"), Type: domain.EventTaskLeased, TaskID: task.ID, TraceID: task.TraceID, Timestamp: now})
	decision := r.Scheduler.Decide(*task, quota)
	if !decision.Accepted {
		if decision.ApprovalRequired {
			blockedAt := time.Now()
			if controller, ok := r.Queue.(queue.TaskController); ok {
				snapshot, err := controller.UpdateTaskState(ctx, task.ID, domain.TaskBlocked, blockedAt, decision.Reason)
				if err == nil {
					task = &snapshot.Task
				}
			} else {
				task.State = domain.TaskBlocked
				task.UpdatedAt = blockedAt
			}
			if r.Recorder != nil {
				r.Recorder.StoreTask(*task)
			}
			r.finishStatus(string(domain.EventRunTakeover), decision.Reason, nil)
			r.publish(domain.Event{
				ID:        eventID(task.ID, "approval-required"),
				Type:      domain.EventRunTakeover,
				TaskID:    task.ID,
				TraceID:   task.TraceID,
				Timestamp: blockedAt,
				Payload: map[string]any{
					"message":  decision.Reason,
					"executor": decision.Assignment.Executor,
					"approval": "required",
				},
			})
			return true
		}
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(100*time.Millisecond))
		r.finishStatus(string(domain.EventTaskRetried), decision.Reason, func(status *Status) {
			status.RetriedRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "requeued"), Type: domain.EventTaskRetried, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"reason": decision.Reason}})
		return true
	}
	r.updateStatus(func(status *Status) {
		status.CurrentExecutor = decision.Assignment.Executor
		status.LastTransition = string(domain.EventSchedulerRouted)
	})
	routedPayload := map[string]any{"executor": decision.Assignment.Executor, "reason": decision.Reason, "quota": quota}
	if decision.Preemption.Required {
		routedPayload["preemption"] = decision.Preemption
	}
	r.publish(domain.Event{ID: eventID(task.ID, "routed"), Type: domain.EventSchedulerRouted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: routedPayload})

	runner, ok := r.Registry.Get(decision.Assignment.Executor)
	if !ok {
		_ = r.Queue.DeadLetter(ctx, lease, "executor not registered")
		r.finishStatus(string(domain.EventTaskDeadLetter), "executor not registered", func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "deadletter"), Type: domain.EventTaskDeadLetter, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"message": "executor not registered", "executor": decision.Assignment.Executor}})
		return true
	}

	preemptedSnapshot, err := r.dispatchPreemption(ctx, *task, decision)
	if err != nil {
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(100*time.Millisecond))
		r.finishStatus(string(domain.EventTaskRetried), err.Error(), func(status *Status) {
			status.RetriedRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "preemption-requeued"), Type: domain.EventTaskRetried, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"reason": err.Error()}})
		return true
	}

	task.State = domain.TaskRunning
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
	r.updateStatus(func(status *Status) {
		status.State = "running"
		status.CurrentExecutor = runner.Kind()
		status.LastStartedAt = time.Now()
		status.LastTransition = string(domain.EventTaskStarted)
	})
	startedPayload := map[string]any{"executor": runner.Kind(), "required_tools": task.RequiredTools}
	if preemptedSnapshot != nil {
		startedPayload["preempted_task_id"] = preemptedSnapshot.Task.ID
		startedPayload["preempted_worker_id"] = preemptedSnapshot.LeaseWorker
		startedPayload["preemption_reason"] = preemptionReason(*task, preemptedSnapshot.Task)
	}
	r.publish(domain.Event{ID: eventID(task.ID, "started"), Type: domain.EventTaskStarted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: startedPayload})

	execCtx, cancel := context.WithTimeout(ctx, r.TaskTimeout)
	stopHeartbeat := r.startHeartbeat(execCtx, lease)
	watcher := r.startCancellationWatcher(execCtx, task.ID, cancel)
	result := runner.Execute(execCtx, *task)
	stopHeartbeat()
	cancel()

	if cancelled, ok := watcher.Snapshot(); ok {
		_ = r.Queue.Ack(ctx, lease)
		message := cancelled.Task.Metadata["cancel_reason"]
		if message == "" {
			message = "task cancelled by control center"
		}
		if r.Recorder != nil {
			r.Recorder.StoreTask(cancelled.Task)
		}
		r.finishStatus(string(domain.EventTaskCancelled), message, func(status *Status) {
			status.CancelledRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "cancelled"), Type: domain.EventTaskCancelled, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"message": message, "executor": runner.Kind()}})
		return true
	}
	if cancelled, ok := r.cancelledSnapshot(ctx, task.ID); ok {
		_ = r.Queue.Ack(ctx, lease)
		message := cancelled.Task.Metadata["cancel_reason"]
		if message == "" {
			message = "task cancelled by control center"
		}
		if r.Recorder != nil {
			r.Recorder.StoreTask(cancelled.Task)
		}
		r.finishStatus(string(domain.EventTaskCancelled), message, func(status *Status) {
			status.CancelledRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "cancelled"), Type: domain.EventTaskCancelled, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"message": message, "executor": runner.Kind()}})
		return true
	}

	switch {
	case result.Success:
		_ = r.Queue.Ack(ctx, lease)
		r.finishStatus(string(domain.EventTaskCompleted), result.Message, func(status *Status) {
			status.SuccessfulRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "completed"), Type: domain.EventTaskCompleted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: runtimeResultPayload(runner.Kind(), result)})
	case result.DeadLetter:
		_ = r.Queue.DeadLetter(ctx, lease, result.Message)
		r.finishStatus(string(domain.EventTaskDeadLetter), result.Message, func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "deadletter"), Type: domain.EventTaskDeadLetter, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: runtimeResultPayload(runner.Kind(), result)})
	default:
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(200*time.Millisecond))
		transition := string(domain.EventTaskRetried)
		extra := func(status *Status) {
			status.RetriedRuns++
		}
		if ctx.Err() != nil || execCtx.Err() == context.Canceled {
			transition = "context.cancelled"
			extra = func(status *Status) {
				status.CancelledRuns++
			}
		}
		r.finishStatus(transition, result.Message, extra)
		r.publish(domain.Event{ID: eventID(task.ID, "retry"), Type: domain.EventTaskRetried, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: runtimeResultPayload(runner.Kind(), result)})
	}
	return true
}

func (r *Runtime) dispatchPreemption(ctx context.Context, task domain.Task, decision scheduler.Decision) (*queue.TaskSnapshot, error) {
	if !decision.Preemption.Required {
		return nil, nil
	}
	inspector, ok := r.Queue.(queue.TaskInspector)
	if !ok {
		return nil, fmt.Errorf("preemptible capacity requires queue task inspection")
	}
	controller, ok := r.Queue.(queue.TaskController)
	if !ok {
		return nil, fmt.Errorf("preemptible capacity requires queue task cancellation")
	}
	snapshots, err := inspector.ListTasks(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("list active tasks for preemption: %w", err)
	}
	candidates := preemptionCandidates(task, snapshots)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no lower-priority active task available for live preemption")
	}
	var lastErr error
	for _, candidate := range candidates {
		reason := preemptionReason(task, candidate.Task)
		snapshot, err := controller.CancelTask(ctx, candidate.Task.ID, reason)
		if err != nil {
			lastErr = err
			continue
		}
		if snapshot.Task.TraceID == "" {
			snapshot.Task.TraceID = snapshot.Task.ID
		}
		if r.Recorder != nil {
			r.Recorder.StoreTask(snapshot.Task)
		}
		now := time.Now()
		r.updateStatus(func(status *Status) {
			status.PreemptionActive = true
			status.CurrentPreemptionTaskID = snapshot.Task.ID
			status.CurrentPreemptionWorkerID = snapshot.LeaseWorker
			status.LastPreemptedTaskID = snapshot.Task.ID
			status.LastPreemptionAt = now
			status.LastPreemptionReason = reason
			status.PreemptionsIssued++
		})
		payload := map[string]any{
			"message":                reason,
			"preempted_by_task_id":   task.ID,
			"preempted_by_priority":  task.Priority,
			"preempted_by_worker_id": r.WorkerID,
			"target_priority":        snapshot.Task.Priority,
			"target_worker_id":       snapshot.LeaseWorker,
			"target_executor":        decision.Assignment.Executor,
		}
		if decision.Preemption.Reason != "" {
			payload["scheduler_reason"] = decision.Preemption.Reason
		}
		r.publish(domain.Event{ID: eventID(snapshot.Task.ID, "preempted"), Type: domain.EventTaskPreempted, TaskID: snapshot.Task.ID, TraceID: snapshot.Task.TraceID, Timestamp: now, Payload: payload})
		return &snapshot, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("cancel lower-priority task for preemption: %w", lastErr)
	}
	return nil, fmt.Errorf("no lower-priority active task available for live preemption")
}

func preemptionCandidates(task domain.Task, snapshots []queue.TaskSnapshot) []queue.TaskSnapshot {
	candidates := make([]queue.TaskSnapshot, 0)
	for _, snapshot := range snapshots {
		if snapshot.Task.ID == task.ID || !snapshot.Leased {
			continue
		}
		if snapshot.Task.State == domain.TaskCancelled || snapshot.Task.State == domain.TaskDeadLetter {
			continue
		}
		if snapshot.Task.Priority <= task.Priority {
			continue
		}
		candidates = append(candidates, snapshot)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Task.Priority == candidates[j].Task.Priority {
			if candidates[i].Task.UpdatedAt.Equal(candidates[j].Task.UpdatedAt) {
				return candidates[i].Task.ID < candidates[j].Task.ID
			}
			return candidates[i].Task.UpdatedAt.After(candidates[j].Task.UpdatedAt)
		}
		return candidates[i].Task.Priority > candidates[j].Task.Priority
	})
	return candidates
}

func preemptionReason(task domain.Task, _ domain.Task) string {
	return fmt.Sprintf("preempted by urgent task %s (priority=%d)", task.ID, task.Priority)
}

func (r *Runtime) startHeartbeat(ctx context.Context, lease *queue.Lease) func() {
	childCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(r.LeaseTTL / 2)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				_ = r.Queue.RenewLease(context.Background(), lease, r.LeaseTTL)
				r.updateStatus(func(status *Status) {
					status.LastHeartbeatAt = time.Now()
					status.LeaseRenewals++
				})
			}
		}
	}()
	return cancel
}

func (r *Runtime) startCancellationWatcher(ctx context.Context, taskID string, cancel context.CancelFunc) *cancellationWatcher {
	watcher := &cancellationWatcher{done: make(chan struct{})}
	if _, ok := r.Queue.(queue.TaskInspector); !ok {
		close(watcher.done)
		return watcher
	}
	interval := cancellationPollInterval(r.LeaseTTL)
	go func() {
		defer close(watcher.done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snapshot, ok := r.cancelledSnapshot(context.Background(), taskID)
				if !ok {
					continue
				}
				watcher.record(snapshot)
				cancel()
				return
			}
		}
	}()
	return watcher
}

func cancellationPollInterval(leaseTTL time.Duration) time.Duration {
	if leaseTTL <= 0 {
		return 25 * time.Millisecond
	}
	interval := leaseTTL / 4
	if interval < 25*time.Millisecond {
		interval = 25 * time.Millisecond
	}
	if interval > 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	return interval
}

func (w *cancellationWatcher) record(snapshot queue.TaskSnapshot) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.snapshot = snapshot
	w.cancelled = true
}

func (w *cancellationWatcher) Snapshot() (queue.TaskSnapshot, bool) {
	<-w.done
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.snapshot, w.cancelled
}

func (r *Runtime) finishStatus(transition, result string, extra func(*Status)) {
	r.updateStatus(func(status *Status) {
		status.State = "idle"
		status.CurrentTaskID = ""
		status.CurrentTraceID = ""
		status.CurrentExecutor = ""
		status.PreemptionActive = false
		status.CurrentPreemptionTaskID = ""
		status.CurrentPreemptionWorkerID = ""
		status.LastFinishedAt = time.Now()
		status.LastResult = result
		status.LastTransition = transition
		if extra != nil {
			extra(status)
		}
	})
}

func (r *Runtime) updateStatus(apply func(*Status)) {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	if r.status.WorkerID == "" {
		r.status.WorkerID = r.WorkerID
	}
	if r.status.State == "" {
		r.status.State = "idle"
	}
	apply(&r.status)
}

func (r *Runtime) publish(event domain.Event) {
	if r.Bus != nil {
		r.Bus.Publish(event)
		return
	}
	if r.Recorder != nil {
		r.Recorder.Record(event)
	}
}

func (r *Runtime) cancelledSnapshot(ctx context.Context, taskID string) (queue.TaskSnapshot, bool) {
	inspector, ok := r.Queue.(queue.TaskInspector)
	if !ok {
		return queue.TaskSnapshot{}, false
	}
	snapshot, err := inspector.GetTask(ctx, taskID)
	if err != nil {
		return queue.TaskSnapshot{}, false
	}
	return snapshot, snapshot.Task.State == domain.TaskCancelled
}

func runtimeResultPayload(executorKind domain.ExecutorKind, result executor.Result) map[string]any {
	payload := map[string]any{"message": result.Message, "executor": executorKind}
	if len(result.Artifacts) > 0 {
		payload["artifacts"] = append([]string(nil), result.Artifacts...)
	}
	if !result.FinishedAt.IsZero() {
		payload["finished_at"] = result.FinishedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func eventID(taskID, suffix string) string {
	return fmt.Sprintf("%s-%s", taskID, suffix)
}
