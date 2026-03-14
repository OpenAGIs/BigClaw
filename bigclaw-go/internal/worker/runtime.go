package worker

import (
	"context"
	"fmt"
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
	WorkerID        string              `json:"worker_id"`
	State           string              `json:"state"`
	CurrentTaskID   string              `json:"current_task_id,omitempty"`
	CurrentTraceID  string              `json:"current_trace_id,omitempty"`
	CurrentExecutor domain.ExecutorKind `json:"current_executor,omitempty"`
	LastHeartbeatAt time.Time           `json:"last_heartbeat_at,omitempty"`
	LastStartedAt   time.Time           `json:"last_started_at,omitempty"`
	LastFinishedAt  time.Time           `json:"last_finished_at,omitempty"`
	LastResult      string              `json:"last_result,omitempty"`
	LeaseRenewals   int                 `json:"lease_renewals"`
	SuccessfulRuns  int                 `json:"successful_runs"`
	RetriedRuns     int                 `json:"retried_runs"`
	DeadLetterRuns  int                 `json:"dead_letter_runs"`
	CancelledRuns   int                 `json:"cancelled_runs"`
	LastTransition  string              `json:"last_transition,omitempty"`
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
			_ = r.Queue.Requeue(ctx, lease, time.Now().Add(250*time.Millisecond))
			task.State = domain.TaskBlocked
			task.UpdatedAt = time.Now()
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
	r.publish(domain.Event{ID: eventID(task.ID, "routed"), Type: domain.EventSchedulerRouted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"executor": decision.Assignment.Executor, "reason": decision.Reason, "quota": quota}})

	runner, ok := r.Registry.Get(decision.Assignment.Executor)
	if !ok {
		_ = r.Queue.DeadLetter(ctx, lease, "executor not registered")
		r.finishStatus(string(domain.EventTaskDeadLetter), "executor not registered", func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{ID: eventID(task.ID, "deadletter"), Type: domain.EventTaskDeadLetter, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"message": "executor not registered", "executor": decision.Assignment.Executor}})
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
	r.publish(domain.Event{ID: eventID(task.ID, "started"), Type: domain.EventTaskStarted, TaskID: task.ID, TraceID: task.TraceID, Timestamp: time.Now(), Payload: map[string]any{"executor": runner.Kind(), "required_tools": task.RequiredTools}})

	execCtx, cancel := context.WithTimeout(ctx, r.TaskTimeout)
	defer cancel()
	stopHeartbeat := r.startHeartbeat(execCtx, lease)
	result := runner.Execute(execCtx, *task)
	stopHeartbeat()

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

func (r *Runtime) finishStatus(transition, result string, extra func(*Status)) {
	r.updateStatus(func(status *Status) {
		status.State = "idle"
		status.CurrentTaskID = ""
		status.CurrentTraceID = ""
		status.CurrentExecutor = ""
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
