package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type DefinitionExecutionRun struct {
	Status string `json:"status"`
}

type DefinitionExecutionResult struct {
	Approved bool                   `json:"approved"`
	Run      DefinitionExecutionRun `json:"run"`
}

type WorkflowEngine struct {
	Gate AcceptanceGate
	Now  func() time.Time
}

type WorkflowRunResult struct {
	Execution   DefinitionExecutionResult `json:"execution"`
	Acceptance  AcceptanceDecision        `json:"acceptance"`
	Journal     WorkpadJournal            `json:"journal"`
	JournalPath string                    `json:"journal_path,omitempty"`
	ReportPath  string                    `json:"report_path,omitempty"`
}

func (d Definition) Validate() error {
	for _, step := range d.Steps {
		switch strings.TrimSpace(step.Kind) {
		case "scheduler", "approval":
		default:
			return fmt.Errorf("invalid workflow step kind: %s", strings.TrimSpace(step.Kind))
		}
	}
	return nil
}

func (e WorkflowEngine) RunDefinition(task domain.Task, definition Definition, runID string) (WorkflowRunResult, error) {
	if err := definition.Validate(); err != nil {
		return WorkflowRunResult{}, err
	}
	now := e.Now
	if now == nil {
		now = time.Now
	}
	journal := WorkpadJournal{TaskID: task.ID, RunID: runID, Now: now}
	journal.Record("intake", "recorded", map[string]any{"source": task.Source})

	execution := DefinitionExecutionResult{
		Approved: true,
		Run:      DefinitionExecutionRun{Status: "approved"},
	}
	for _, step := range definition.Steps {
		if strings.TrimSpace(step.Kind) == "approval" {
			execution.Approved = false
			execution.Run.Status = "needs-approval"
			break
		}
	}
	journal.Record("execution", execution.Run.Status, map[string]any{"approved": execution.Approved})

	reportPath := definition.RenderReportPath(task, runID)
	if strings.TrimSpace(reportPath) != "" {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			return WorkflowRunResult{}, err
		}
		body := strings.Join([]string{
			"# Workflow Definition Run",
			"",
			fmt.Sprintf("- Task ID: %s", task.ID),
			fmt.Sprintf("- Run ID: %s", runID),
			fmt.Sprintf("- Status: %s", execution.Run.Status),
		}, "\n")
		if err := os.WriteFile(reportPath, []byte(body), 0o644); err != nil {
			return WorkflowRunResult{}, err
		}
	}

	acceptance := e.Gate.Evaluate(task, ExecutionOutcome{
		Approved: execution.Approved,
		Status:   execution.Run.Status,
	}, definition.ValidationEvidence, definition.Approvals, "")
	journal.Record("acceptance", acceptance.Status, map[string]any{
		"passed":                      acceptance.Passed,
		"missing_acceptance_criteria": acceptance.MissingAcceptanceCriteria,
		"missing_validation_steps":    acceptance.MissingValidationSteps,
	})

	journalPath := definition.RenderJournalPath(task, runID)
	if strings.TrimSpace(journalPath) != "" {
		var err error
		journalPath, err = journal.Write(journalPath)
		if err != nil {
			return WorkflowRunResult{}, err
		}
	}

	return WorkflowRunResult{
		Execution:   execution,
		Acceptance:  acceptance,
		Journal:     journal,
		JournalPath: journalPath,
		ReportPath:  reportPath,
	}, nil
}
