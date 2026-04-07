package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/workflow"
)

type leasedTaskRun struct {
	task     *domain.Task
	lease    *queue.Lease
	runID    string
	leasedAt time.Time
	journal  *workflow.WorkpadJournal
}

func (r *Runtime) ensureDefaults() {
	if r.LeaseTTL <= 0 {
		r.LeaseTTL = 2 * time.Minute
	}
	if r.TaskTimeout <= 0 {
		r.TaskTimeout = 30 * time.Second
	}
}

func (r *Runtime) acquireLeasedTask(ctx context.Context) (*leasedTaskRun, bool) {
	task, lease, err := r.Queue.LeaseNext(ctx, r.WorkerID, r.LeaseTTL)
	if err != nil || task == nil || lease == nil {
		r.updateStatus(func(status *Status) {
			status.WorkerID = r.WorkerID
			if status.State == "" || status.State == "paused" {
				status.State = "idle"
			}
		})
		return nil, false
	}
	if task.TraceID == "" {
		task.TraceID = task.ID
	}
	leasedAt := time.Now()
	run := &leasedTaskRun{
		task:     task,
		lease:    lease,
		runID:    runtimeRunID(*task, leasedAt),
		leasedAt: leasedAt,
		journal:  r.newWorkpadJournal(*task, lease),
	}
	r.recordWorkpad(run.journal, "intake", "recorded", map[string]any{
		"source":   task.Source,
		"trace_id": task.TraceID,
		"attempt":  lease.Attempt,
		"worker":   r.WorkerID,
	})
	if r.Recorder != nil {
		r.Recorder.StoreTask(*task)
	}
	return run, true
}

func (r *Runtime) handleTakeoverHold(ctx context.Context, run *leasedTaskRun) bool {
	if r.Control == nil {
		return false
	}
	takeover, ok := r.Control.TakeoverStatus(run.task.ID)
	if !ok || !takeover.Active {
		return false
	}
	_ = r.Queue.Requeue(ctx, run.lease, time.Now().Add(250*time.Millisecond))
	run.task.State = domain.TaskBlocked
	run.task.UpdatedAt = r.now()
	r.recordWorkpad(run.journal, "control-takeover", "blocked", map[string]any{
		"owner":    takeover.Owner,
		"reviewer": takeover.Reviewer,
		"reason":   "task under human takeover",
	})
	r.persistWorkpad(run.task, run.lease, run.journal)
	if r.Recorder != nil {
		r.Recorder.StoreTask(*run.task)
	}
	r.finishStatus(string(domain.EventRunTakeover), "task under human takeover", nil)
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "takeover-hold"),
		Type:      domain.EventRunAnnotated,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: run.leasedAt,
		Payload: map[string]any{
			"message":  "automation deferred while human takeover is active",
			"owner":    takeover.Owner,
			"reviewer": takeover.Reviewer,
		},
	})
	return true
}

func (r *Runtime) publishLease(run *leasedTaskRun) {
	r.updateStatus(func(status *Status) {
		status.WorkerID = r.WorkerID
		status.State = "leased"
		status.CurrentTaskID = run.task.ID
		status.CurrentTraceID = run.task.TraceID
		status.LastTransition = string(domain.EventTaskLeased)
		status.LastHeartbeatAt = run.leasedAt
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "leased"),
		Type:      domain.EventTaskLeased,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: run.leasedAt,
	})
}

func (r *Runtime) handleMissingRuntimeDependencies(ctx context.Context, run *leasedTaskRun) bool {
	if r.Scheduler != nil && r.Registry != nil {
		return false
	}
	reasonParts := make([]string, 0, 2)
	if r.Scheduler == nil {
		reasonParts = append(reasonParts, "scheduler not configured")
	}
	if r.Registry == nil {
		reasonParts = append(reasonParts, "executor registry not configured")
	}
	reason := strings.Join(reasonParts, "; ")

	_ = r.Queue.DeadLetter(ctx, run.lease, reason)
	r.recordWorkpad(run.journal, "execution", "dead-letter", map[string]any{"reason": reason})
	r.persistWorkpad(run.task, run.lease, run.journal)

	finishedAt := time.Now()
	closeout := workflow.BuildCloseout(workflow.CloseoutInput{
		Task:        *run.task,
		RunID:       run.runID,
		Status:      workflow.WorkflowRunFailed,
		Executor:    run.task.RequiredExecutor,
		Message:     reason,
		StartedAt:   run.leasedAt,
		CompletedAt: finishedAt,
	})
	r.finishStatus(string(domain.EventTaskDeadLetter), reason, func(status *Status) {
		status.DeadLetterRuns++
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "deadletter"),
		Type:      domain.EventTaskDeadLetter,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: finishedAt,
		Payload:   runtimeTerminalPayload(run.runID, run.task.RequiredExecutor, reason, nil, finishedAt, closeout, false),
	})
	return true
}

func (r *Runtime) assessTask(run *leasedTaskRun, quota scheduler.QuotaSnapshot) scheduler.Assessment {
	assessment := r.Scheduler.Assess(*run.task, quota)
	decision := assessment.Decision
	r.recordWorkpad(run.journal, "scheduler", runtimeAssessmentStatus(decision), map[string]any{
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
		r.recordWorkpad(run.journal, "handoff", assessment.HandoffRequest.Status, map[string]any{
			"target_team":        assessment.HandoffRequest.TargetTeam,
			"reason":             assessment.HandoffRequest.Reason,
			"required_approvals": assessment.HandoffRequest.RequiredApprovals,
		})
	}
	return assessment
}

func (r *Runtime) handleRejectedAssessment(ctx context.Context, run *leasedTaskRun, assessment scheduler.Assessment) bool {
	if assessment.Decision.Accepted {
		return false
	}
	if assessment.HandoffRequest != nil {
		r.publish(domain.Event{
			ID:        eventID(run.task.ID, "handoff"),
			Type:      domain.EventRunTakeover,
			TaskID:    run.task.ID,
			TraceID:   run.task.TraceID,
			RunID:     run.runID,
			Timestamp: time.Now(),
			Payload:   runtimeHandoffPayload(assessment),
		})
	}
	r.recordWorkpad(run.journal, "execution", "retried", map[string]any{"reason": assessment.Decision.Reason})
	r.persistWorkpad(run.task, run.lease, run.journal)
	_ = r.Queue.Requeue(ctx, run.lease, time.Now().Add(100*time.Millisecond))
	finishedAt := time.Now()
	closeout := workflow.BuildCloseout(workflow.CloseoutInput{
		Task:           *run.task,
		RunID:          run.runID,
		Status:         workflow.WorkflowRunFailed,
		Executor:       assessment.Decision.Assignment.Executor,
		Message:        assessment.Decision.Reason,
		StartedAt:      run.leasedAt,
		CompletedAt:    finishedAt,
		RetryScheduled: true,
	})
	r.finishStatus(string(domain.EventTaskRetried), assessment.Decision.Reason, func(status *Status) {
		status.RetriedRuns++
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "requeued"),
		Type:      domain.EventTaskRetried,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: finishedAt,
		Payload: runtimeTerminalPayload(
			run.runID,
			assessment.Decision.Assignment.Executor,
			assessment.Decision.Reason,
			nil,
			finishedAt,
			closeout,
			true,
		),
	})
	return true
}

func (r *Runtime) publishRoutedDecision(run *leasedTaskRun, assessment scheduler.Assessment, quota scheduler.QuotaSnapshot) {
	decision := assessment.Decision
	r.updateStatus(func(status *Status) {
		status.CurrentExecutor = decision.Assignment.Executor
		status.LastTransition = string(domain.EventSchedulerRouted)
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "routed"),
		Type:      domain.EventSchedulerRouted,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: time.Now(),
		Payload:   runtimeRoutedPayload(assessment, quota),
	})
	if assessment.HandoffRequest != nil {
		r.publish(domain.Event{
			ID:        eventID(run.task.ID, "handoff"),
			Type:      domain.EventRunTakeover,
			TaskID:    run.task.ID,
			TraceID:   run.task.TraceID,
			RunID:     run.runID,
			Timestamp: time.Now(),
			Payload:   runtimeHandoffPayload(assessment),
		})
	}
}

func runtimeRoutedPayload(assessment scheduler.Assessment, quota scheduler.QuotaSnapshot) map[string]any {
	decision := assessment.Decision
	payload := map[string]any{
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
		payload["preemption"] = decision.Preemption
	}
	return payload
}

func (r *Runtime) resolveRunner(ctx context.Context, run *leasedTaskRun, decision scheduler.Decision) (executor.Runner, *queue.TaskSnapshot, bool) {
	runner, ok := r.Registry.Get(decision.Assignment.Executor)
	if !ok {
		r.handleMissingRunner(ctx, run, decision.Assignment.Executor)
		return nil, nil, false
	}
	preemptedSnapshot, err := r.dispatchPreemption(ctx, *run.task, decision)
	if err != nil {
		r.handlePreemptionRetry(ctx, run, decision.Assignment.Executor, err)
		return nil, nil, false
	}
	return runner, preemptedSnapshot, true
}

func (r *Runtime) handleMissingRunner(ctx context.Context, run *leasedTaskRun, executorKind domain.ExecutorKind) {
	_ = r.Queue.DeadLetter(ctx, run.lease, "executor not registered")
	r.recordWorkpad(run.journal, "execution", "dead-letter", map[string]any{
		"executor": executorKind,
		"reason":   "executor not registered",
	})
	r.persistWorkpad(run.task, run.lease, run.journal)
	finishedAt := time.Now()
	closeout := workflow.BuildCloseout(workflow.CloseoutInput{
		Task:        *run.task,
		RunID:       run.runID,
		Status:      workflow.WorkflowRunFailed,
		Executor:    executorKind,
		Message:     "executor not registered",
		StartedAt:   run.leasedAt,
		CompletedAt: finishedAt,
	})
	r.finishStatus(string(domain.EventTaskDeadLetter), "executor not registered", func(status *Status) {
		status.DeadLetterRuns++
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "deadletter"),
		Type:      domain.EventTaskDeadLetter,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: finishedAt,
		Payload:   runtimeTerminalPayload(run.runID, executorKind, "executor not registered", nil, finishedAt, closeout, false),
	})
}

func (r *Runtime) handlePreemptionRetry(ctx context.Context, run *leasedTaskRun, executorKind domain.ExecutorKind, err error) {
	_ = r.Queue.Requeue(ctx, run.lease, time.Now().Add(100*time.Millisecond))
	r.recordWorkpad(run.journal, "execution", "retried", map[string]any{"reason": err.Error()})
	r.persistWorkpad(run.task, run.lease, run.journal)
	finishedAt := time.Now()
	closeout := workflow.BuildCloseout(workflow.CloseoutInput{
		Task:           *run.task,
		RunID:          run.runID,
		Status:         workflow.WorkflowRunFailed,
		Executor:       executorKind,
		Message:        err.Error(),
		StartedAt:      run.leasedAt,
		CompletedAt:    finishedAt,
		RetryScheduled: true,
	})
	r.finishStatus(string(domain.EventTaskRetried), err.Error(), func(status *Status) {
		status.RetriedRuns++
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "preemption-requeued"),
		Type:      domain.EventTaskRetried,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: finishedAt,
		Payload:   runtimeTerminalPayload(run.runID, executorKind, err.Error(), nil, finishedAt, closeout, true),
	})
}

func (r *Runtime) markExecutionStarted(run *leasedTaskRun, runner executor.Runner, preemptedSnapshot *queue.TaskSnapshot) time.Time {
	run.task.State = domain.TaskRunning
	run.task.UpdatedAt = r.now()
	r.recordWorkpad(run.journal, "execution", "started", map[string]any{
		"executor":       runner.Kind(),
		"required_tools": append([]string(nil), run.task.RequiredTools...),
	})
	r.persistWorkpad(run.task, run.lease, run.journal)
	if r.Recorder != nil {
		r.Recorder.StoreTask(*run.task)
	}
	startedAt := time.Now()
	r.updateStatus(func(status *Status) {
		status.State = "running"
		status.CurrentExecutor = runner.Kind()
		status.LastStartedAt = startedAt
		status.LastTransition = string(domain.EventTaskStarted)
	})
	startedPayload := map[string]any{"executor": runner.Kind(), "required_tools": run.task.RequiredTools}
	if preemptedSnapshot != nil {
		startedPayload["preempted_task_id"] = preemptedSnapshot.Task.ID
		startedPayload["preempted_worker_id"] = preemptedSnapshot.LeaseWorker
		startedPayload["preemption_reason"] = preemptionReason(*run.task, preemptedSnapshot.Task)
	}
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "started"),
		Type:      domain.EventTaskStarted,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: startedAt,
		Payload:   startedPayload,
	})
	return startedAt
}

func (r *Runtime) handleCancellationIfPresent(ctx context.Context, run *leasedTaskRun, watcher *cancellationWatcher, executorKind domain.ExecutorKind, startedAt time.Time) bool {
	if cancelled, ok := watcher.Snapshot(); ok {
		return r.finalizeCancellation(ctx, run, cancelled, executorKind, startedAt)
	}
	if cancelled, ok := r.cancelledSnapshot(ctx, run.task.ID); ok {
		return r.finalizeCancellation(ctx, run, cancelled, executorKind, startedAt)
	}
	return false
}

func (r *Runtime) finalizeCancellation(ctx context.Context, run *leasedTaskRun, cancelled queue.TaskSnapshot, executorKind domain.ExecutorKind, startedAt time.Time) bool {
	_ = r.Queue.Ack(ctx, run.lease)
	message := cancelled.Task.Metadata["cancel_reason"]
	if message == "" {
		message = "task cancelled by control center"
	}
	finishedAt := time.Now()
	if r.Recorder != nil {
		r.Recorder.StoreTask(cancelled.Task)
	}
	r.recordWorkpad(run.journal, "execution", "cancelled", map[string]any{
		"executor": executorKind,
		"reason":   message,
	})
	r.persistWorkpad(&cancelled.Task, run.lease, run.journal)
	closeout := workflow.BuildCloseout(workflow.CloseoutInput{
		Task:        cancelled.Task,
		RunID:       run.runID,
		Status:      workflow.WorkflowRunCanceled,
		Executor:    executorKind,
		Message:     message,
		StartedAt:   startedAt,
		CompletedAt: finishedAt,
	})
	r.finishStatus(string(domain.EventTaskCancelled), message, func(status *Status) {
		status.CancelledRuns++
	})
	r.publish(domain.Event{
		ID:        eventID(run.task.ID, "cancelled"),
		Type:      domain.EventTaskCancelled,
		TaskID:    run.task.ID,
		TraceID:   run.task.TraceID,
		RunID:     run.runID,
		Timestamp: finishedAt,
		Payload:   runtimeTerminalPayload(run.runID, executorKind, message, nil, finishedAt, closeout, false),
	})
	return true
}

func (r *Runtime) finalizeExecutionResult(ctx, execCtx context.Context, run *leasedTaskRun, runner executor.Runner, result executor.Result, leaseLost <-chan error, startedAt time.Time) {
	switch {
	case result.Success:
		_ = r.Queue.Ack(ctx, run.lease)
		run.task.State = domain.TaskSucceeded
		run.task.UpdatedAt = r.now()
		r.recordWorkpad(run.journal, "execution", "completed", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		acceptance, hasAcceptance := r.evaluateAcceptance(*run.task)
		if hasAcceptance {
			r.recordWorkpad(run.journal, "acceptance", acceptance.Status, map[string]any{
				"passed":                      acceptance.Passed,
				"summary":                     acceptance.Summary,
				"approvals":                   acceptance.Approvals,
				"missing_acceptance_criteria": acceptance.MissingAcceptanceCriteria,
				"missing_validation_steps":    acceptance.MissingValidationSteps,
			})
			applyAcceptanceMetadata(run.task, acceptance)
			if r.Recorder != nil {
				r.Recorder.StoreTask(*run.task)
			}
		}
		r.persistWorkpad(run.task, run.lease, run.journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *run.task,
			RunID:       run.runID,
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
			ID:        eventID(run.task.ID, "completed"),
			Type:      domain.EventTaskCompleted,
			TaskID:    run.task.ID,
			TraceID:   run.task.TraceID,
			RunID:     run.runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(run.runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, false),
		})
		if hasAcceptance {
			r.publish(domain.Event{
				ID:        eventID(run.task.ID, "acceptance"),
				Type:      domain.EventRunAnnotated,
				TaskID:    run.task.ID,
				TraceID:   run.task.TraceID,
				RunID:     run.runID,
				Timestamp: finishedAt,
				Payload:   runtimeAcceptancePayload(acceptance),
			})
		}
	case result.DeadLetter:
		_ = r.Queue.DeadLetter(ctx, run.lease, result.Message)
		run.task.State = domain.TaskDeadLetter
		run.task.UpdatedAt = r.now()
		r.recordWorkpad(run.journal, "execution", "dead-letter", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		r.persistWorkpad(run.task, run.lease, run.journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:        *run.task,
			RunID:       run.runID,
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
			ID:        eventID(run.task.ID, "deadletter"),
			Type:      domain.EventTaskDeadLetter,
			TaskID:    run.task.ID,
			TraceID:   run.task.TraceID,
			RunID:     run.runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(run.runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, false),
		})
	default:
		_ = r.Queue.Requeue(ctx, run.lease, time.Now().Add(200*time.Millisecond))
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
		run.task.State = domain.TaskRetrying
		run.task.UpdatedAt = r.now()
		r.recordWorkpad(run.journal, "execution", "retried", map[string]any{
			"executor":    runner.Kind(),
			"message":     result.Message,
			"artifacts":   append([]string(nil), result.Artifacts...),
			"finished_at": runtimeResultFinishedAt(result),
		})
		r.persistWorkpad(run.task, run.lease, run.journal)
		finishedAt := result.FinishedAt
		if finishedAt.IsZero() {
			finishedAt = time.Now()
		}
		closeout := workflow.BuildCloseout(workflow.CloseoutInput{
			Task:           *run.task,
			RunID:          run.runID,
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
			ID:        eventID(run.task.ID, "retry"),
			Type:      domain.EventTaskRetried,
			TaskID:    run.task.ID,
			TraceID:   run.task.TraceID,
			RunID:     run.runID,
			Timestamp: finishedAt,
			Payload:   runtimeTerminalPayload(run.runID, runner.Kind(), result.Message, result.Artifacts, finishedAt, closeout, true),
		})
	}
}
