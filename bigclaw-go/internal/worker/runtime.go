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
	NodeID      string
	Queue       queue.Queue
	Scheduler   *scheduler.Scheduler
	Registry    *executor.Registry
	Bus         *events.Bus
	Recorder    *observability.Recorder
	Control     *control.Controller
	LeaseTTL    time.Duration
	TaskTimeout time.Duration
	Backoff     BackoffPolicy
	WorkpadDir  string
	Now         func() time.Time
	statusMu    sync.Mutex
	status      Status
}

type Status struct {
	WorkerID                  string              `json:"worker_id"`
	NodeID                    string              `json:"node_id,omitempty"`
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
	if snapshot.NodeID == "" {
		snapshot.NodeID = r.NodeID
	}
	if snapshot.State == "" {
		snapshot.State = "idle"
	}
	return snapshot
}

func (r *Runtime) retryAvailableAt(reason BackoffReason, attempt int) time.Time {
	return r.now().Add(r.Backoff.Resolve(reason, attempt))
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
	r.ensureDefaults()

	run, ok := r.acquireLeasedTask(ctx)
	if !ok {
		return false
	}
	if r.handleTakeoverHold(ctx, run) {
		return true
	}

	r.publishLease(run)
	if r.handleMissingRuntimeDependencies(ctx, run) {
		return true
	}

	assessment := r.assessTask(run, quota)
	if r.handleRejectedAssessment(ctx, run, assessment) {
		return true
	}

	r.publishRoutedDecision(run, assessment, quota)
	runner, preemptedSnapshot, ok := r.resolveRunner(ctx, run, assessment.Decision)
	if !ok {
		return true
	}

	startedAt := r.markExecutionStarted(run, runner, preemptedSnapshot)

	execCtx, cancel := context.WithTimeout(ctx, r.TaskTimeout)
	leaseLost := make(chan error, 1)
	stopHeartbeat := r.startHeartbeat(execCtx, run.lease, cancel, leaseLost)
	watcher := r.startCancellationWatcher(execCtx, run.task.ID, cancel)
	result := runner.Execute(execCtx, *run.task)
	stopHeartbeat()
	cancel()

	if r.handleCancellationIfPresent(ctx, run, watcher, runner.Kind(), startedAt) {
		return true
	}

	r.finalizeExecutionResult(ctx, execCtx, run, runner, result, leaseLost, startedAt)
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
	if r.status.NodeID == "" {
		r.status.NodeID = r.NodeID
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
