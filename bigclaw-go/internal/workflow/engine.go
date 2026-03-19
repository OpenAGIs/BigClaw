package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type JournalEntry struct {
	Step      string         `json:"step"`
	Status    string         `json:"status"`
	Timestamp time.Time      `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type WorkpadJournal struct {
	TaskID  string         `json:"task_id"`
	RunID   string         `json:"run_id"`
	Entries []JournalEntry `json:"entries,omitempty"`
}

func (j *WorkpadJournal) Record(step, status string, timestamp time.Time, details map[string]any) {
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	entry := JournalEntry{
		Step:      step,
		Status:    status,
		Timestamp: timestamp.UTC(),
	}
	if len(details) > 0 {
		entry.Details = cloneMap(details)
	}
	j.Entries = append(j.Entries, entry)
}

func (j WorkpadJournal) Write(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	contents, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(contents, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

type AcceptanceDecision struct {
	Passed                    bool     `json:"passed"`
	Status                    string   `json:"status"`
	Summary                   string   `json:"summary"`
	MissingAcceptanceCriteria []string `json:"missing_acceptance_criteria,omitempty"`
	MissingValidationSteps    []string `json:"missing_validation_steps,omitempty"`
	Approvals                 []string `json:"approvals,omitempty"`
}

type AcceptanceGate struct{}

func (gate AcceptanceGate) Evaluate(task domain.Task, evidence, approvals []string) AcceptanceDecision {
	resolvedEvidence := toSet(evidence)
	resolvedApprovals := uniqueNonEmpty(approvals)
	missingAcceptance := missingItems(task.AcceptanceCriteria, resolvedEvidence)
	missingValidation := missingItems(task.ValidationPlan, resolvedEvidence)
	if requiresManualApproval(task) && len(resolvedApprovals) == 0 {
		return AcceptanceDecision{
			Passed:                    false,
			Status:                    "needs-approval",
			Summary:                   "manual approval required before acceptance closure",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 resolvedApprovals,
		}
	}
	if len(missingAcceptance) > 0 || len(missingValidation) > 0 {
		return AcceptanceDecision{
			Passed:                    false,
			Status:                    "rejected",
			Summary:                   "acceptance evidence incomplete",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 resolvedApprovals,
		}
	}
	return AcceptanceDecision{
		Passed:    true,
		Status:    "accepted",
		Summary:   "acceptance criteria and validation plan satisfied",
		Approvals: resolvedApprovals,
	}
}

type Closeout struct {
	ValidationEvidence []string                     `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool                         `json:"git_push_succeeded"`
	GitLogStatCaptured bool                         `json:"git_log_stat_captured"`
	GitLogStatOutput   string                       `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *observability.RepoSyncAudit `json:"repo_sync_audit,omitempty"`
	Complete           bool                         `json:"complete"`
}

type RunOptions struct {
	ValidationEvidence []string
	Approvals          []string
	GitPushSucceeded   bool
	GitLogStatOutput   string
	RepoSyncAudit      *observability.RepoSyncAudit
}

type RunResult struct {
	Task        domain.Task        `json:"task"`
	Events      []domain.Event     `json:"events,omitempty"`
	WorkflowRun WorkflowRun        `json:"workflow_run"`
	Acceptance  AcceptanceDecision `json:"acceptance"`
	Closeout    Closeout           `json:"closeout"`
	Journal     WorkpadJournal     `json:"journal"`
	JournalPath string             `json:"journal_path,omitempty"`
	ReportPath  string             `json:"report_path,omitempty"`
	Status      worker.Status      `json:"status"`
	RunID       string             `json:"run_id"`
	Definition  Definition         `json:"definition"`
}

type Engine struct {
	Runtime  *worker.Runtime
	Recorder *observability.Recorder
	Queue    queue.Queue
	Quota    scheduler.QuotaSnapshot
	Gate     AcceptanceGate
	Now      func() time.Time
}

func (engine *Engine) RunDefinition(ctx context.Context, task domain.Task, definition Definition, runID string, options RunOptions) (RunResult, error) {
	if engine == nil || engine.Runtime == nil || engine.Queue == nil || engine.Recorder == nil {
		return RunResult{}, fmt.Errorf("workflow engine requires runtime, queue, and recorder")
	}
	if strings.TrimSpace(task.ID) == "" {
		return RunResult{}, fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(runID) == "" {
		return RunResult{}, fmt.Errorf("run id is required")
	}
	now := engine.now()
	resolvedTask := task
	if resolvedTask.TraceID == "" {
		resolvedTask.TraceID = resolvedTask.ID
	}
	resolvedTask.Metadata = cloneStringMap(resolvedTask.Metadata)
	resolvedTask.Metadata["workflow"] = definition.Name
	resolvedTask.Metadata["workflow_run_id"] = runID
	if resolvedTask.CreatedAt.IsZero() {
		resolvedTask.CreatedAt = now
	}
	resolvedTask.UpdatedAt = now

	journal := WorkpadJournal{TaskID: resolvedTask.ID, RunID: runID}
	journal.Record("intake", "recorded", now, map[string]any{
		"source":   resolvedTask.Source,
		"workflow": definition.Name,
	})

	if err := engine.Queue.Enqueue(ctx, resolvedTask); err != nil {
		return RunResult{}, fmt.Errorf("enqueue task: %w", err)
	}
	journal.Record("queue", "queued", engine.now(), map[string]any{
		"task_id": resolvedTask.ID,
		"run_id":  runID,
	})

	processed := engine.Runtime.RunOnce(ctx, engine.Quota)
	if !processed {
		return RunResult{}, fmt.Errorf("runtime did not process queued task %s", resolvedTask.ID)
	}

	finalTask, ok := engine.Recorder.Task(resolvedTask.ID)
	if !ok {
		finalTask = resolvedTask
	}
	events := engine.Recorder.EventsByTask(resolvedTask.ID, 0)
	journal.Record("execution", string(finalTask.State), engine.now(), map[string]any{
		"event_count": len(events),
		"task_state":  finalTask.State,
	})

	evidence := uniqueNonEmpty(append(append([]string(nil), definition.ValidationEvidence...), options.ValidationEvidence...))
	approvals := uniqueNonEmpty(append(append([]string(nil), definition.Approvals...), options.Approvals...))
	acceptance := engine.Gate.Evaluate(finalTask, evidence, approvals)
	journal.Record("acceptance", acceptance.Status, engine.now(), map[string]any{
		"passed":                      acceptance.Passed,
		"missing_acceptance_criteria": append([]string(nil), acceptance.MissingAcceptanceCriteria...),
		"missing_validation_steps":    append([]string(nil), acceptance.MissingValidationSteps...),
	})

	closeout := Closeout{
		ValidationEvidence: append([]string(nil), evidence...),
		GitPushSucceeded:   options.GitPushSucceeded,
		GitLogStatCaptured: strings.TrimSpace(options.GitLogStatOutput) != "",
		GitLogStatOutput:   options.GitLogStatOutput,
		RepoSyncAudit:      cloneRepoSyncAudit(options.RepoSyncAudit),
	}
	closeout.Complete = len(closeout.ValidationEvidence) > 0 &&
		closeout.GitPushSucceeded &&
		closeout.GitLogStatCaptured &&
		closeout.repoSyncVerified()
	workflowRun := buildWorkflowRun(definition, finalTask, runID, journal, acceptance, closeout, now, engine.now())
	closeoutDetails := map[string]any{
		"validation_evidence":   append([]string(nil), closeout.ValidationEvidence...),
		"git_push_succeeded":    closeout.GitPushSucceeded,
		"git_log_stat_captured": closeout.GitLogStatCaptured,
	}
	if closeout.RepoSyncAudit != nil {
		closeoutDetails["repo_sync_status"] = closeout.RepoSyncAudit.Sync.Status
		closeoutDetails["repo_sync_summary"] = closeout.RepoSyncAudit.Summary()
		closeoutDetails["repo_sync_verified"] = closeout.RepoSyncAudit.Verified()
	}
	journal.Record("closeout", closeoutStatus(closeout.Complete), engine.now(), closeoutDetails)

	reportPath := definition.RenderReportPath(finalTask, runID)
	if err := writeReport(reportPath, renderRunReport(definition, runID, finalTask, acceptance, closeout, events)); err != nil {
		return RunResult{}, fmt.Errorf("write run report: %w", err)
	}
	journalPath, err := journal.Write(definition.RenderJournalPath(finalTask, runID))
	if err != nil {
		return RunResult{}, fmt.Errorf("write journal: %w", err)
	}

	return RunResult{
		Task:        finalTask,
		Events:      events,
		WorkflowRun: workflowRun,
		Acceptance:  acceptance,
		Closeout:    closeout,
		Journal:     journal,
		JournalPath: journalPath,
		ReportPath:  reportPath,
		Status:      engine.Runtime.Snapshot(),
		RunID:       runID,
		Definition:  definition,
	}, nil
}

func (engine *Engine) now() time.Time {
	if engine != nil && engine.Now != nil {
		return engine.Now().UTC()
	}
	return time.Now().UTC()
}

func renderRunReport(definition Definition, runID string, task domain.Task, acceptance AcceptanceDecision, closeout Closeout, events []domain.Event) string {
	lines := []string{
		"# Workflow Run Report",
		"",
		fmt.Sprintf("- Workflow: %s", firstNonEmpty(definition.Name, task.Metadata["workflow"], "unnamed")),
		fmt.Sprintf("- Run ID: %s", runID),
		fmt.Sprintf("- Task ID: %s", task.ID),
		fmt.Sprintf("- Task State: %s", task.State),
		fmt.Sprintf("- Acceptance: %s", acceptance.Status),
		fmt.Sprintf("- Closeout Complete: %t", closeout.Complete),
		fmt.Sprintf("- Event Count: %d", len(events)),
	}
	if len(closeout.ValidationEvidence) > 0 {
		lines = append(lines, fmt.Sprintf("- Validation Evidence: %s", strings.Join(closeout.ValidationEvidence, ", ")))
	}
	if closeout.RepoSyncAudit != nil {
		lines = append(lines,
			fmt.Sprintf("- Repo Sync Status: %s", firstNonEmpty(closeout.RepoSyncAudit.Sync.Status, "unknown")),
			fmt.Sprintf("- Repo Sync Summary: %s", closeout.RepoSyncAudit.Summary()),
			fmt.Sprintf("- Repo Sync Verified: %t", closeout.RepoSyncAudit.Verified()),
		)
	}
	if len(acceptance.Approvals) > 0 {
		lines = append(lines, fmt.Sprintf("- Approvals: %s", strings.Join(acceptance.Approvals, ", ")))
	}
	if acceptance.Summary != "" {
		lines = append(lines, "", "## Acceptance Summary", "", acceptance.Summary)
	}
	if len(events) > 0 {
		lines = append(lines, "", "## Timeline", "")
		for _, event := range events {
			lines = append(lines, fmt.Sprintf("- %s %s", event.Timestamp.UTC().Format(time.RFC3339), event.Type))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeReport(path string, contents string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}

func requiresManualApproval(task domain.Task) bool {
	return task.RiskLevel == domain.RiskHigh
}

func closeoutStatus(complete bool) string {
	if complete {
		return "complete"
	}
	return "pending"
}

func missingItems(required []string, present map[string]struct{}) []string {
	out := make([]string, 0)
	for _, item := range required {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := present[trimmed]; !ok {
			out = append(out, trimmed)
		}
	}
	return out
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out[trimmed] = struct{}{}
	}
	return out
}

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneRepoSyncAudit(audit *observability.RepoSyncAudit) *observability.RepoSyncAudit {
	if audit == nil {
		return nil
	}
	clone := *audit
	clone.Sync.DirtyPaths = append([]string(nil), audit.Sync.DirtyPaths...)
	return &clone
}

func (closeout Closeout) repoSyncVerified() bool {
	return closeout.RepoSyncAudit != nil && closeout.RepoSyncAudit.Verified()
}

func buildWorkflowRun(definition Definition, task domain.Task, runID string, journal WorkpadJournal, acceptance AcceptanceDecision, closeout Closeout, startedAt, completedAt time.Time) WorkflowRun {
	run := WorkflowRun{
		RunID:       runID,
		TemplateID:  firstNonEmpty(definition.Name, task.Metadata["workflow"], "workflow-run"),
		TaskID:      task.ID,
		TriggeredBy: "workflow-engine",
		StartedAt:   startedAt.UTC().Format(time.RFC3339),
		CompletedAt: completedAt.UTC().Format(time.RFC3339),
		Outputs: map[string]any{
			"acceptance_status": acceptance.Status,
			"closeout_complete": closeout.Complete,
			"journal_entries":   len(journal.Entries),
			"task_state":        task.State,
		},
		ApprovalRefs: append([]string(nil), acceptance.Approvals...),
	}
	run.Steps = buildWorkflowStepRuns(definition.Steps, acceptance, closeout, task, startedAt, completedAt)
	run.Status = resolveWorkflowRunStatus(run.Steps, task, acceptance)
	return run
}

func buildWorkflowStepRuns(steps []Step, acceptance AcceptanceDecision, closeout Closeout, task domain.Task, startedAt, completedAt time.Time) []WorkflowStepRun {
	if len(steps) == 0 {
		return nil
	}
	runs := make([]WorkflowStepRun, 0, len(steps))
	for index, step := range steps {
		stepID := firstNonEmpty(step.Name, step.Kind, fmt.Sprintf("step-%d", index+1))
		run := WorkflowStepRun{
			StepID:    stepID,
			Actor:     "workflow-engine",
			StartedAt: startedAt.UTC().Format(time.RFC3339),
		}
		status, output := resolveWorkflowStepStatus(step, acceptance, closeout, task)
		run.Status = status
		if status != WorkflowStepPending {
			run.CompletedAt = completedAt.UTC().Format(time.RFC3339)
		}
		if len(output) > 0 {
			run.Output = output
		}
		runs = append(runs, run)
	}
	return runs
}

func resolveWorkflowStepStatus(step Step, acceptance AcceptanceDecision, closeout Closeout, task domain.Task) (WorkflowStepStatus, map[string]any) {
	output := cloneMap(step.Metadata)
	if output == nil {
		output = make(map[string]any)
	}
	output["required"] = step.Required
	kind := strings.ToLower(strings.TrimSpace(step.Kind))
	switch kind {
	case "approval":
		output["approvals"] = append([]string(nil), acceptance.Approvals...)
		if len(acceptance.Approvals) > 0 {
			return WorkflowStepSucceeded, output
		}
		if step.Required {
			output["status"] = acceptance.Status
			return WorkflowStepPending, output
		}
		return WorkflowStepSkipped, output
	case "validation":
		output["missing_validation_steps"] = append([]string(nil), acceptance.MissingValidationSteps...)
		if len(acceptance.MissingValidationSteps) == 0 {
			return WorkflowStepSucceeded, output
		}
		if step.Required {
			return WorkflowStepFailed, output
		}
		return WorkflowStepSkipped, output
	case "closeout":
		output["complete"] = closeout.Complete
		if closeout.Complete {
			return WorkflowStepSucceeded, output
		}
		if step.Required {
			return WorkflowStepFailed, output
		}
		return WorkflowStepSkipped, output
	default:
		output["task_state"] = task.State
		switch task.State {
		case domain.TaskSucceeded:
			return WorkflowStepSucceeded, output
		case domain.TaskCancelled:
			return WorkflowStepSkipped, output
		case domain.TaskDeadLetter, domain.TaskBlocked:
			return WorkflowStepFailed, output
		default:
			if step.Required {
				return WorkflowStepFailed, output
			}
			return WorkflowStepSkipped, output
		}
	}
}

func resolveWorkflowRunStatus(steps []WorkflowStepRun, task domain.Task, acceptance AcceptanceDecision) WorkflowRunStatus {
	switch task.State {
	case domain.TaskCancelled:
		return WorkflowRunCanceled
	case domain.TaskDeadLetter, domain.TaskBlocked:
		return WorkflowRunFailed
	}
	for _, step := range steps {
		switch step.Status {
		case WorkflowStepFailed:
			return WorkflowRunFailed
		case WorkflowStepPending, WorkflowStepRunning:
			return WorkflowRunRunning
		}
	}
	if acceptance.Passed {
		return WorkflowRunSucceeded
	}
	if acceptance.Status == "needs-approval" {
		return WorkflowRunRunning
	}
	return WorkflowRunFailed
}
