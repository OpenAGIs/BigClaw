package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/risk"
)

type Decision struct {
	Medium   string     `json:"medium"`
	Approved bool       `json:"approved"`
	Reason   string     `json:"reason"`
	Risk     risk.Score `json:"risk"`
}

type ExecutionRecord struct {
	Decision Decision `json:"decision"`
	Status   string   `json:"status"`
}

type Executor interface {
	Execute(task domain.Task, runID string) (ExecutionRecord, error)
}

type DefaultExecutor struct{}

func (DefaultExecutor) Execute(task domain.Task, _ string) (ExecutionRecord, error) {
	score := risk.ScoreTask(task, nil)
	decision := Decision{
		Medium:   "docker",
		Approved: true,
		Reason:   "default low risk path",
		Risk:     score,
	}

	if task.BudgetCents < 0 {
		decision.Medium = "none"
		decision.Approved = false
		decision.Reason = "invalid budget"
	} else if score.Level == domain.RiskHigh {
		decision.Medium = "vm"
		decision.Approved = false
		decision.Reason = "requires approval for high-risk task"
	} else if requiresTool(task, "browser") {
		decision.Medium = "browser"
		decision.Reason = "browser automation task"
	} else if score.Level == domain.RiskMedium {
		decision.Reason = "medium risk in docker"
	}

	status := "approved"
	if !decision.Approved {
		status = "needs-approval"
	}
	return ExecutionRecord{Decision: decision, Status: status}, nil
}

type JournalEntry struct {
	Step      string         `json:"step"`
	Status    string         `json:"status"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type WorkpadJournal struct {
	TaskID  string         `json:"task_id"`
	RunID   string         `json:"run_id"`
	Entries []JournalEntry `json:"entries,omitempty"`
}

func (j *WorkpadJournal) Record(step, status string, details map[string]any) {
	j.Entries = append(j.Entries, JournalEntry{
		Step:      strings.TrimSpace(step),
		Status:    strings.TrimSpace(status),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Details:   cloneAnyMap(details),
	})
}

func (j WorkpadJournal) Write(path string) (string, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	body, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, body, 0o644); err != nil {
		return "", err
	}
	return target, nil
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

func (AcceptanceGate) Evaluate(task domain.Task, execution ExecutionRecord, validationEvidence, approvals []string) AcceptanceDecision {
	evidence := make(map[string]struct{}, len(validationEvidence))
	for _, item := range validationEvidence {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			evidence[trimmed] = struct{}{}
		}
	}

	missingAcceptance := missingItems(task.AcceptanceCriteria, evidence)
	missingValidation := missingItems(task.ValidationPlan, evidence)
	resolvedApprovals := compactStrings(approvals)
	needsApproval := execution.Status == "needs-approval" || !execution.Decision.Approved || execution.Decision.Risk.RequiresApproval
	if needsApproval && len(resolvedApprovals) == 0 {
		return AcceptanceDecision{
			Status:                    "needs-approval",
			Summary:                   "manual approval required before acceptance closure",
			MissingAcceptanceCriteria: missingAcceptance,
			MissingValidationSteps:    missingValidation,
			Approvals:                 resolvedApprovals,
		}
	}
	if len(missingAcceptance) > 0 || len(missingValidation) > 0 {
		return AcceptanceDecision{
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

type RunResult struct {
	Execution   ExecutionRecord    `json:"execution"`
	Acceptance  AcceptanceDecision `json:"acceptance"`
	Journal     WorkpadJournal     `json:"journal"`
	JournalPath string             `json:"journal_path,omitempty"`
	ReportPath  string             `json:"report_path,omitempty"`
}

type Engine struct {
	Executor Executor
	Gate     AcceptanceGate
	Recorder *observability.Recorder
}

func NewEngine() *Engine {
	return &Engine{Executor: DefaultExecutor{}}
}

func (e *Engine) resolvedExecutor() Executor {
	if e != nil && e.Executor != nil {
		return e.Executor
	}
	return DefaultExecutor{}
}

func (e *Engine) Run(task domain.Task, runID, reportPath, journalPath string, validationEvidence, approvals []string) (RunResult, error) {
	execution, err := e.resolvedExecutor().Execute(task, runID)
	if err != nil {
		return RunResult{}, err
	}

	journal := WorkpadJournal{TaskID: task.ID, RunID: runID}
	journal.Record("intake", "recorded", map[string]any{"source": task.Source})
	journal.Record("execution", execution.Status, map[string]any{
		"medium":   execution.Decision.Medium,
		"approved": execution.Decision.Approved,
		"reason":   execution.Decision.Reason,
	})
	if e != nil && e.Recorder != nil {
		_ = e.Recorder.RecordSpecEvent(domain.Event{
			ID:        fmt.Sprintf("%s-%s-scheduler-decision", task.ID, runID),
			Type:      domain.EventType(observability.SchedulerDecisionEvent),
			TaskID:    task.ID,
			RunID:     runID,
			TraceID:   firstNonEmpty(task.TraceID, task.ID),
			Timestamp: time.Now().UTC(),
			Payload: map[string]any{
				"medium":     execution.Decision.Medium,
				"approved":   execution.Decision.Approved,
				"reason":     execution.Decision.Reason,
				"risk_level": execution.Decision.Risk.Level,
				"risk_score": execution.Decision.Risk.Total,
			},
		})
	}

	acceptance := e.Gate.Evaluate(task, execution, validationEvidence, approvals)
	if len(acceptance.Approvals) > 0 && e != nil && e.Recorder != nil {
		_ = e.Recorder.RecordSpecEvent(domain.Event{
			ID:        fmt.Sprintf("%s-%s-approval-recorded", task.ID, runID),
			Type:      domain.EventType(observability.ApprovalRecordedEvent),
			TaskID:    task.ID,
			RunID:     runID,
			TraceID:   firstNonEmpty(task.TraceID, task.ID),
			Timestamp: time.Now().UTC(),
			Payload: map[string]any{
				"approvals":         append([]string(nil), acceptance.Approvals...),
				"approval_count":    len(acceptance.Approvals),
				"acceptance_status": acceptance.Status,
			},
		})
	}

	journal.Record("acceptance", acceptance.Status, map[string]any{
		"passed":                      acceptance.Passed,
		"missing_acceptance_criteria": append([]string(nil), acceptance.MissingAcceptanceCriteria...),
		"missing_validation_steps":    append([]string(nil), acceptance.MissingValidationSteps...),
	})
	journal.Record("closeout", closeoutStatus(acceptance), map[string]any{
		"validation_evidence": append([]string(nil), validationEvidence...),
		"approval_count":      len(acceptance.Approvals),
	})

	resolvedReportPath := strings.TrimSpace(reportPath)
	if resolvedReportPath != "" {
		if err := writeFile(resolvedReportPath, renderRunReport(task, runID, execution, acceptance)); err != nil {
			return RunResult{}, err
		}
	}
	resolvedJournalPath, err := journal.Write(journalPath)
	if err != nil {
		return RunResult{}, err
	}

	return RunResult{
		Execution:   execution,
		Acceptance:  acceptance,
		Journal:     journal,
		JournalPath: resolvedJournalPath,
		ReportPath:  resolvedReportPath,
	}, nil
}

func (e *Engine) RunDefinition(task domain.Task, definition Definition, runID string) (RunResult, error) {
	return e.Run(
		task,
		runID,
		definition.RenderReportPath(task, runID),
		definition.RenderJournalPath(task, runID),
		definition.ValidationEvidence,
		definition.Approvals,
	)
}

func renderRunReport(task domain.Task, runID string, execution ExecutionRecord, acceptance AcceptanceDecision) string {
	lines := []string{
		"# BigClaw Workflow Run Report",
		"",
		fmt.Sprintf("- Task ID: %s", task.ID),
		fmt.Sprintf("- Run ID: %s", runID),
		fmt.Sprintf("- Medium: %s", execution.Decision.Medium),
		fmt.Sprintf("- Approved: %t", execution.Decision.Approved),
		fmt.Sprintf("- Execution Status: %s", execution.Status),
		fmt.Sprintf("- Acceptance Status: %s", acceptance.Status),
		fmt.Sprintf("- Acceptance Summary: %s", acceptance.Summary),
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeFile(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}

func missingItems(items []string, evidence map[string]struct{}) []string {
	out := make([]string, 0)
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := evidence[trimmed]; !ok {
			out = append(out, trimmed)
		}
	}
	return out
}

func compactStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func closeoutStatus(acceptance AcceptanceDecision) string {
	if acceptance.Passed && acceptance.Status == "accepted" {
		return "complete"
	}
	if strings.TrimSpace(acceptance.Status) != "" {
		return acceptance.Status
	}
	return "incomplete"
}

func requiresTool(task domain.Task, want string) bool {
	for _, tool := range task.RequiredTools {
		if strings.EqualFold(strings.TrimSpace(tool), want) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
