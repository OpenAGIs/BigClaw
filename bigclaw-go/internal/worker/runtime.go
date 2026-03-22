package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/workflow"
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
	WorkpadDir  string
	Now         func() time.Time
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
	LeaseRenewalFailures      int                 `json:"lease_renewal_failures"`
	LeaseLostRuns             int                 `json:"lease_lost_runs"`
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

	if r.Queue == nil {
		return false
	}
	if r.LeaseTTL <= 0 {
		r.LeaseTTL = 2 * time.Minute
	}
	if r.TaskTimeout <= 0 {
		r.TaskTimeout = 30 * time.Second
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
	leasedAt := time.Now()
	runID := runtimeRunID(*task, leasedAt)
	journal := r.newWorkpadJournal(*task, lease)
	r.recordWorkpad(journal, "intake", "recorded", map[string]any{
		"source":   task.Source,
		"trace_id": task.TraceID,
		"attempt":  lease.Attempt,
		"worker":   r.WorkerID,
	})
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
	if r.Control != nil {
		takeover, ok := r.Control.TakeoverStatus(task.ID)
		if ok && takeover.Active {
			_ = r.Queue.Requeue(ctx, lease, time.Now().Add(250*time.Millisecond))
			task.State = domain.TaskBlocked
			task.UpdatedAt = r.now()
			r.recordWorkpad(journal, "control-takeover", "blocked", map[string]any{
				"owner":    takeover.Owner,
				"reviewer": takeover.Reviewer,
				"reason":   "task under human takeover",
			})
			r.persistWorkpad(task, lease, journal)
			if r.Recorder != nil {
				r.Recorder.StoreTask(*task)
			}
			r.finishStatus(string(domain.EventRunTakeover), "task under human takeover", nil)
			r.publish(domain.Event{
				ID:        eventID(task.ID, "takeover-hold"),
				Type:      domain.EventRunAnnotated,
				TaskID:    task.ID,
				TraceID:   task.TraceID,
				RunID:     runID,
				Timestamp: leasedAt,
				Payload: map[string]any{
					"message":  "automation deferred while human takeover is active",
					"owner":    takeover.Owner,
					"reviewer": takeover.Reviewer,
				},
			})
			return true
		}
	}

	r.updateStatus(func(status *Status) {
		status.WorkerID = r.WorkerID
		status.State = "leased"
		status.CurrentTaskID = task.ID
		status.CurrentTraceID = task.TraceID
		status.LastTransition = string(domain.EventTaskLeased)
		status.LastHeartbeatAt = leasedAt
	})

	r.publish(domain.Event{ID: eventID(task.ID, "leased"), Type: domain.EventTaskLeased, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: leasedAt})
	if r.Scheduler == nil || r.Registry == nil {
		reasonParts := make([]string, 0, 2)
		if r.Scheduler == nil {
			reasonParts = append(reasonParts, "scheduler not configured")
		}
		if r.Registry == nil {
			reasonParts = append(reasonParts, "executor registry not configured")
		}
		reason := strings.Join(reasonParts, "; ")

		_ = r.Queue.DeadLetter(ctx, lease, reason)
		r.recordWorkpad(journal, "execution", "dead-letter", map[string]any{"reason": reason})
		r.persistWorkpad(task, lease, journal)

		finishedAt := time.Now()
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *task,
			RunID:       runID,
			Status:      workflow.WorkflowRunFailed,
			Executor:    task.RequiredExecutor,
			Message:     reason,
			StartedAt:   leasedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskDeadLetter), reason, func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "deadletter"),
			Type:      domain.EventTaskDeadLetter,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, task.RequiredExecutor, reason, nil, finishedAt, closeout, false),
		})
		return true
	}
	assessment := r.Scheduler.Assess(*task, quota)
	decision := assessment.Decision
	r.recordWorkpad(journal, "scheduler", runtimeAssessmentStatus(decision), map[string]any{
		"executor":           decision.Assignment.Executor,
		"reason":             decision.Reason,
		"risk_level":         assessment.Risk.Level,
		"risk_score":         assessment.Risk.Total,
		"requires_approval":  assessment.Risk.RequiresApproval,
		"collaboration_mode": assessment.OrchestrationPlan.CollaborationMode,
		"departments":        assessment.OrchestrationPlan.Departments(),
		"required_approvals": assessment.OrchestrationPlan.RequiredApprovals(),
		"policy_tier":        assessment.OrchestrationPolicy.Tier,
		"upgrade_required":   assessment.OrchestrationPolicy.UpgradeRequired,
	})
	if assessment.HandoffRequest != nil {
		r.recordWorkpad(journal, "handoff", assessment.HandoffRequest.Status, map[string]any{
			"target_team":        assessment.HandoffRequest.TargetTeam,
			"reason":             assessment.HandoffRequest.Reason,
			"required_approvals": assessment.HandoffRequest.RequiredApprovals,
		})
	}
	if !decision.Accepted {
		if assessment.HandoffRequest != nil {
			r.publish(domain.Event{ID: eventID(task.ID, "handoff"), Type: domain.EventRunTakeover, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: time.Now(), Payload: runtimeHandoffPayload(assessment)})
		}
		r.recordWorkpad(journal, "execution", "retried", map[string]any{"reason": decision.Reason})
		r.persistWorkpad(task, lease, journal)
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(100*time.Millisecond))
		finishedAt := time.Now()
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:           *task,
			RunID:          runID,
			Status:         workflow.WorkflowRunFailed,
			Executor:       decision.Assignment.Executor,
			Message:        decision.Reason,
			StartedAt:      leasedAt,
			CompletedAt:    finishedAt,
			RetryScheduled: true,
		})
		r.finishStatus(string(domain.EventTaskRetried), decision.Reason, func(status *Status) {
			status.RetriedRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "requeued"),
			Type:      domain.EventTaskRetried,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, decision.Assignment.Executor, decision.Reason, nil, finishedAt, closeout, true),
		})
		return true
	}
	r.updateStatus(func(status *Status) {
		status.CurrentExecutor = decision.Assignment.Executor
		status.LastTransition = string(domain.EventSchedulerRouted)
	})
	routedPayload := map[string]any{
		"executor":          decision.Assignment.Executor,
		"reason":            decision.Reason,
		"quota":             quota,
		"risk_level":        assessment.Risk.Level,
		"risk_score":        assessment.Risk.Total,
		"requires_approval": assessment.Risk.RequiresApproval,
		"orchestration": map[string]any{
			"collaboration_mode": assessment.OrchestrationPlan.CollaborationMode,
			"departments":        assessment.OrchestrationPlan.Departments(),
			"required_approvals": assessment.OrchestrationPlan.RequiredApprovals(),
		},
		"policy": map[string]any{
			"tier":                assessment.OrchestrationPolicy.Tier,
			"upgrade_required":    assessment.OrchestrationPolicy.UpgradeRequired,
			"reason":              assessment.OrchestrationPolicy.Reason,
			"blocked_departments": assessment.OrchestrationPolicy.BlockedDepartments,
			"entitlement_status":  assessment.OrchestrationPolicy.EntitlementStatus,
			"billing_model":       assessment.OrchestrationPolicy.BillingModel,
			"estimated_cost_usd":  assessment.OrchestrationPolicy.EstimatedCostUSD,
		},
	}
	if decision.Preemption.Required {
		routedPayload["preemption"] = decision.Preemption
	}
	r.publish(domain.Event{ID: eventID(task.ID, "routed"), Type: domain.EventSchedulerRouted, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: time.Now(), Payload: routedPayload})
	if assessment.HandoffRequest != nil {
		r.publish(domain.Event{ID: eventID(task.ID, "handoff"), Type: domain.EventRunTakeover, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: time.Now(), Payload: runtimeHandoffPayload(assessment)})
	}

	runner, ok := r.Registry.Get(decision.Assignment.Executor)
	if !ok {
		_ = r.Queue.DeadLetter(ctx, lease, "executor not registered")
		r.recordWorkpad(journal, "execution", "dead-letter", map[string]any{
			"executor": decision.Assignment.Executor,
			"reason":   "executor not registered",
		})
		r.persistWorkpad(task, lease, journal)
		finishedAt := time.Now()
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *task,
			RunID:       runID,
			Status:      workflow.WorkflowRunFailed,
			Executor:    decision.Assignment.Executor,
			Message:     "executor not registered",
			StartedAt:   leasedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskDeadLetter), "executor not registered", func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "deadletter"),
			Type:      domain.EventTaskDeadLetter,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, decision.Assignment.Executor, "executor not registered", nil, finishedAt, closeout, false),
		})
		return true
	}

	preemptedSnapshot, err := r.dispatchPreemption(ctx, *task, decision)
	if err != nil {
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(100*time.Millisecond))
		r.recordWorkpad(journal, "execution", "retried", map[string]any{"reason": err.Error()})
		r.persistWorkpad(task, lease, journal)
		finishedAt := time.Now()
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:           *task,
			RunID:          runID,
			Status:         workflow.WorkflowRunFailed,
			Executor:       decision.Assignment.Executor,
			Message:        err.Error(),
			StartedAt:      leasedAt,
			CompletedAt:    finishedAt,
			RetryScheduled: true,
		})
		r.finishStatus(string(domain.EventTaskRetried), err.Error(), func(status *Status) {
			status.RetriedRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "preemption-requeued"),
			Type:      domain.EventTaskRetried,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, decision.Assignment.Executor, err.Error(), nil, finishedAt, closeout, true),
		})
		return true
	}

	startedAt := leasedAt
	task.State = domain.TaskRunning
	task.UpdatedAt = r.now()
	r.recordWorkpad(journal, "execution", "started", map[string]any{
		"executor":       runner.Kind(),
		"required_tools": append([]string(nil), task.RequiredTools...),
	})
	r.persistWorkpad(task, lease, journal)
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
	startedAt = time.Now()
	r.updateStatus(func(status *Status) {
		status.State = "running"
		status.CurrentExecutor = runner.Kind()
		status.LastStartedAt = startedAt
		status.LastTransition = string(domain.EventTaskStarted)
	})
	startedPayload := map[string]any{"executor": runner.Kind(), "required_tools": task.RequiredTools}
	if preemptedSnapshot != nil {
		startedPayload["preempted_task_id"] = preemptedSnapshot.Task.ID
		startedPayload["preempted_worker_id"] = preemptedSnapshot.LeaseWorker
		startedPayload["preemption_reason"] = preemptionReason(*task, preemptedSnapshot.Task)
	}
	r.publish(domain.Event{ID: eventID(task.ID, "started"), Type: domain.EventTaskStarted, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: startedAt, Payload: startedPayload})

	execCtx, cancel := context.WithTimeout(ctx, r.TaskTimeout)
	leaseLost := make(chan error, 1)
	stopHeartbeat := r.startHeartbeat(execCtx, lease, cancel, leaseLost)
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
		finishedAt := time.Now()
		if r.Recorder != nil {
			r.Recorder.StoreTask(cancelled.Task)
		}
		r.recordWorkpad(journal, "execution", "cancelled", map[string]any{
			"executor": runner.Kind(),
			"reason":   message,
		})
		r.persistWorkpad(&cancelled.Task, lease, journal)
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        cancelled.Task,
			RunID:       runID,
			Status:      workflow.WorkflowRunCanceled,
			Executor:    runner.Kind(),
			Message:     message,
			StartedAt:   startedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskCancelled), message, func(status *Status) {
			status.CancelledRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "cancelled"),
			Type:      domain.EventTaskCancelled,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, runner.Kind(), message, nil, finishedAt, closeout, false),
		})
		return true
	}
	if cancelled, ok := r.cancelledSnapshot(ctx, task.ID); ok {
		_ = r.Queue.Ack(ctx, lease)
		message := cancelled.Task.Metadata["cancel_reason"]
		if message == "" {
			message = "task cancelled by control center"
		}
		finishedAt := time.Now()
		if r.Recorder != nil {
			r.Recorder.StoreTask(cancelled.Task)
		}
		r.recordWorkpad(journal, "execution", "cancelled", map[string]any{
			"executor": runner.Kind(),
			"reason":   message,
		})
		r.persistWorkpad(&cancelled.Task, lease, journal)
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        cancelled.Task,
			RunID:       runID,
			Status:      workflow.WorkflowRunCanceled,
			Executor:    runner.Kind(),
			Message:     message,
			StartedAt:   startedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskCancelled), message, func(status *Status) {
			status.CancelledRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "cancelled"),
			Type:      domain.EventTaskCancelled,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, runner.Kind(), message, nil, finishedAt, closeout, false),
		})
		return true
	}

	switch {
	case result.Success:
		_ = r.Queue.Ack(ctx, lease)
		task.State = domain.TaskSucceeded
		task.UpdatedAt = r.now()
		r.recordWorkpad(journal, "execution", "completed", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		acceptance, hasAcceptance := r.evaluateAcceptance(*task)
		if hasAcceptance {
			r.recordWorkpad(journal, "acceptance", acceptance.Status, map[string]any{
				"passed":                      acceptance.Passed,
				"summary":                     acceptance.Summary,
				"approvals":                   acceptance.Approvals,
				"missing_acceptance_criteria": acceptance.MissingAcceptanceCriteria,
				"missing_validation_steps":    acceptance.MissingValidationSteps,
			})
			applyAcceptanceMetadata(task, acceptance)
			if r.Recorder != nil {
				r.Recorder.StoreTask(*task)
			}
		}
		r.persistWorkpad(task, lease, journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *task,
			RunID:       runID,
			Status:      workflow.WorkflowRunSucceeded,
			Executor:    runner.Kind(),
			Message:     result.Message,
			Artifacts:   result.Artifacts,
			StartedAt:   startedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskCompleted), result.Message, func(status *Status) {
			status.SuccessfulRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "completed"),
			Type:      domain.EventTaskCompleted,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, false),
		})
		if hasAcceptance {
			r.publish(domain.Event{ID: eventID(task.ID, "acceptance"), Type: domain.EventRunAnnotated, TaskID: task.ID, TraceID: task.TraceID, RunID: runID, Timestamp: finishedAt, Payload: runtimeAcceptancePayload(acceptance)})
		}
	case result.DeadLetter:
		_ = r.Queue.DeadLetter(ctx, lease, result.Message)
		task.State = domain.TaskDeadLetter
		task.UpdatedAt = r.now()
		r.recordWorkpad(journal, "execution", "dead-letter", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		r.persistWorkpad(task, lease, journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *task,
			RunID:       runID,
			Status:      workflow.WorkflowRunFailed,
			Executor:    runner.Kind(),
			Message:     result.Message,
			Artifacts:   result.Artifacts,
			StartedAt:   startedAt,
			CompletedAt: finishedAt,
		})
		r.finishStatus(string(domain.EventTaskDeadLetter), result.Message, func(status *Status) {
			status.DeadLetterRuns++
		})
		r.publish(domain.Event{
			ID:        eventID(task.ID, "deadletter"),
			Type:      domain.EventTaskDeadLetter,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, false),
		})
	default:
		_ = r.Queue.Requeue(ctx, lease, time.Now().Add(200*time.Millisecond))
		transition := string(domain.EventTaskRetried)
		extra := func(status *Status) {
			status.RetriedRuns++
		}
		var leaseLostErr error
		select {
		case leaseLostErr = <-leaseLost:
		default:
		}
		if leaseLostErr != nil {
			transition = "lease.lost"
			if result.Message == "" || result.Message == context.Canceled.Error() {
				result.Message = fmt.Sprintf("lost lease: %v", leaseLostErr)
			}
			extra = func(status *Status) {
				status.LeaseLostRuns++
			}
		} else if ctx.Err() != nil || execCtx.Err() == context.Canceled {
			transition = "context.cancelled"
			extra = func(status *Status) {
				status.CancelledRuns++
			}
		}
		task.State = domain.TaskRetrying
		task.UpdatedAt = r.now()
		r.recordWorkpad(journal, "execution", "retried", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		r.persistWorkpad(task, lease, journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:           *task,
			RunID:          runID,
			Status:         workflow.WorkflowRunFailed,
			Executor:       runner.Kind(),
			Message:        result.Message,
			Artifacts:      result.Artifacts,
			StartedAt:      startedAt,
			CompletedAt:    finishedAt,
			RetryScheduled: true,
		})
		r.finishStatus(transition, result.Message, extra)
		r.publish(domain.Event{
			ID:        eventID(task.ID, "retry"),
			Type:      domain.EventTaskRetried,
			TaskID:    task.ID,
			TraceID:   task.TraceID,
			RunID:     runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, true),
		})
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

func runtimeHandoffPayload(assessment scheduler.Assessment) map[string]any {
	payload := map[string]any{
		"reason":             assessment.Decision.Reason,
		"risk_level":         assessment.Risk.Level,
		"risk_score":         assessment.Risk.Total,
		"collaboration_mode": assessment.OrchestrationPlan.CollaborationMode,
		"departments":        assessment.OrchestrationPlan.Departments(),
		"policy_tier":        assessment.OrchestrationPolicy.Tier,
		"upgrade_required":   assessment.OrchestrationPolicy.UpgradeRequired,
	}
	if len(assessment.OrchestrationPolicy.BlockedDepartments) > 0 {
		payload["blocked_departments"] = assessment.OrchestrationPolicy.BlockedDepartments
	}
	if assessment.HandoffRequest != nil {
		payload["target_team"] = assessment.HandoffRequest.TargetTeam
		payload["handoff_status"] = assessment.HandoffRequest.Status
		payload["handoff_reason"] = assessment.HandoffRequest.Reason
		payload["required_approvals"] = assessment.HandoffRequest.RequiredApprovals
	}
	return payload
}

func (r *Runtime) startHeartbeat(ctx context.Context, lease *queue.Lease, cancelExec context.CancelFunc, leaseLost chan<- error) func() {
	childCtx, cancelHeartbeat := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(r.LeaseTTL / 2)
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				if err := r.Queue.RenewLease(context.Background(), lease, r.LeaseTTL); err != nil {
					r.updateStatus(func(status *Status) {
						status.LastHeartbeatAt = time.Now()
						status.LeaseRenewalFailures++
						status.LastResult = fmt.Sprintf("lease_renew_failed: %v", err)
					})
					if errors.Is(err, queue.ErrLeaseExpired) || errors.Is(err, queue.ErrLeaseNotOwned) {
						select {
						case leaseLost <- err:
						default:
						}
						cancelExec()
						cancelHeartbeat()
					}
					continue
				}
				r.updateStatus(func(status *Status) {
					status.LastHeartbeatAt = time.Now()
					status.LeaseRenewals++
				})
			}
		}
	}()
	return cancelHeartbeat
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

func runtimeTerminalPayload(runID string, executorKind domain.ExecutorKind, message string, artifacts []string, finishedAt time.Time, closeout workflow.Closeout, retryScheduled bool) map[string]any {
	payload := map[string]any{"message": message, "run_id": runID}
	if executor := strings.TrimSpace(string(executorKind)); executor != "" {
		payload["executor"] = executorKind
	}
	if len(artifacts) > 0 {
		payload["artifacts"] = append([]string(nil), artifacts...)
	}
	if !finishedAt.IsZero() {
		payload["finished_at"] = finishedAt.UTC().Format(time.RFC3339)
	}
	if retryScheduled {
		payload["retry_scheduled"] = true
	}
	if closeout.Run.RunID != "" {
		payload["workflow_run"] = closeout.Run
	}
	if closeout.ReportPath != "" {
		payload["report_path"] = closeout.ReportPath
	}
	if closeout.JournalPath != "" {
		payload["journal_path"] = closeout.JournalPath
	}
	if len(closeout.ValidationEvidence) > 0 {
		payload["validation_evidence"] = append([]string(nil), closeout.ValidationEvidence...)
	}
	if len(closeout.RequiredApprovals) > 0 {
		payload["required_approvals"] = append([]string(nil), closeout.RequiredApprovals...)
	}
	return payload
}

func runtimeRunID(task domain.Task, startedAt time.Time) string {
	base := strings.TrimSpace(task.TraceID)
	if base == "" {
		base = strings.TrimSpace(task.ID)
	}
	if base == "" {
		base = "run"
	}
	return fmt.Sprintf("%s-%d", base, startedAt.UnixNano())
}

func runtimeResultPayload(executorKind domain.ExecutorKind, result executor.Result) map[string]any {
	payload := map[string]any{"message": result.Message, "executor": executorKind}
	if len(result.Artifacts) > 0 {
		payload["artifacts"] = append([]string(nil), result.Artifacts...)
	}
	if finishedAt := runtimeResultFinishedAt(result); finishedAt != "" {
		payload["finished_at"] = finishedAt
	}
	return payload
}

func runtimeResultFinishedAt(result executor.Result) string {
	if result.FinishedAt.IsZero() {
		return ""
	}
	return result.FinishedAt.UTC().Format(time.RFC3339)
}

func runtimeAcceptancePayload(decision workflow.AcceptanceDecision) map[string]any {
	payload := map[string]any{
		"acceptance_status": decision.Status,
		"passed":            decision.Passed,
		"summary":           decision.Summary,
	}
	if len(decision.Approvals) > 0 {
		payload["approvals"] = append([]string(nil), decision.Approvals...)
	}
	if len(decision.MissingAcceptanceCriteria) > 0 {
		payload["missing_acceptance_criteria"] = append([]string(nil), decision.MissingAcceptanceCriteria...)
	}
	if len(decision.MissingValidationSteps) > 0 {
		payload["missing_validation_steps"] = append([]string(nil), decision.MissingValidationSteps...)
	}
	return payload
}

func runtimeAssessmentStatus(decision scheduler.Decision) string {
	if decision.Accepted {
		return "accepted"
	}
	return "blocked"
}

func (r *Runtime) evaluateAcceptance(task domain.Task) (workflow.AcceptanceDecision, bool) {
	if len(task.AcceptanceCriteria) == 0 && len(task.ValidationPlan) == 0 && strings.TrimSpace(task.Metadata["pilot_recommendation"]) == "" {
		return workflow.AcceptanceDecision{}, false
	}
	gate := workflow.AcceptanceGate{}
	return gate.Evaluate(
		task,
		workflow.ExecutionOutcome{Approved: true, Status: "completed"},
		runtimeMetadataStringSlice(task, "validation_evidence"),
		runtimeMetadataStringSlice(task, "approvals"),
		strings.TrimSpace(task.Metadata["pilot_recommendation"]),
	), true
}

func (r *Runtime) newWorkpadJournal(task domain.Task, lease *queue.Lease) *workflow.WorkpadJournal {
	if strings.TrimSpace(r.WorkpadDir) == "" || lease == nil {
		return nil
	}
	return &workflow.WorkpadJournal{
		TaskID: task.ID,
		RunID:  fmt.Sprintf("%s-attempt-%d", task.ID, lease.Attempt),
		Now:    r.now,
	}
}

func (r *Runtime) recordWorkpad(journal *workflow.WorkpadJournal, step, status string, details map[string]any) {
	if journal == nil {
		return
	}
	journal.Record(step, status, details)
}

func applyAcceptanceMetadata(task *domain.Task, decision workflow.AcceptanceDecision) {
	if task == nil {
		return
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["acceptance_status"] = decision.Status
	task.Metadata["approval_status"] = decision.Status
	task.Metadata["acceptance_summary"] = decision.Summary
	if !decision.Passed {
		task.Metadata["blocked_reason"] = decision.Summary
		return
	}
	delete(task.Metadata, "blocked_reason")
}

func (r *Runtime) persistWorkpad(task *domain.Task, lease *queue.Lease, journal *workflow.WorkpadJournal) {
	if task == nil || lease == nil || journal == nil {
		return
	}
	path, err := journal.Write(filepath.Join(r.WorkpadDir, task.ID, fmt.Sprintf("attempt-%d.json", lease.Attempt)))
	if err != nil {
		return
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["workpad"] = path
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = r.now()
	}
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
}

func runtimeMetadataStringSlice(task domain.Task, key string) []string {
	raw := strings.TrimSpace(task.Metadata[key])
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			return values
		}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	if len(parts) == 1 && strings.Contains(parts[0], ",") {
		parts = strings.Split(parts[0], ",")
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func (r *Runtime) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func eventID(taskID, suffix string) string {
	return fmt.Sprintf("%s-%s", taskID, suffix)
}
