package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type DefinitionRunResult struct {
	Execution   DefinitionExecution `json:"execution"`
	Acceptance  AcceptanceDecision  `json:"acceptance"`
	ReportPath  string              `json:"report_path,omitempty"`
	JournalPath string              `json:"journal_path,omitempty"`
}

type DefinitionExecution struct {
	Run DefinitionRun `json:"run"`
}

type DefinitionRun struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

func RunDefinition(task domain.Task, definition Definition, runID string) (DefinitionRunResult, error) {
	for _, step := range definition.Steps {
		switch strings.TrimSpace(step.Kind) {
		case "scheduler", "approval":
		default:
			return DefinitionRunResult{}, fmt.Errorf("invalid workflow step kind: %s", step.Kind)
		}
	}

	status := "completed"
	approved := true
	if task.RiskLevel == domain.RiskHigh {
		status = "needs-approval"
		approved = false
	}

	reportPath := definition.RenderReportPath(task, runID)
	journalPath := definition.RenderJournalPath(task, runID)
	if err := writeDefinitionArtifacts(task, definition, runID, status, reportPath, journalPath); err != nil {
		return DefinitionRunResult{}, err
	}

	evidence := definition.ValidationEvidence
	if len(evidence) == 0 {
		evidence = append(append([]string(nil), task.ValidationPlan...), task.AcceptanceCriteria...)
	}
	acceptance := AcceptanceGate{}.Evaluate(task, ExecutionOutcome{
		Approved: approved,
		Status:   status,
	}, evidence, definition.Approvals, "")

	return DefinitionRunResult{
		Execution: DefinitionExecution{
			Run: DefinitionRun{
				RunID:  strings.TrimSpace(runID),
				Status: status,
			},
		},
		Acceptance:  acceptance,
		ReportPath:  reportPath,
		JournalPath: journalPath,
	}, nil
}

func writeDefinitionArtifacts(task domain.Task, definition Definition, runID, status, reportPath, journalPath string) error {
	if reportPath != "" {
		if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
			return err
		}
		report := strings.Join([]string{
			"# Workflow Definition Run",
			"",
			fmt.Sprintf("- Task: %s", task.ID),
			fmt.Sprintf("- Run ID: %s", runID),
			fmt.Sprintf("- Workflow: %s", definition.Name),
			fmt.Sprintf("- Status: %s", status),
			"",
		}, "\n")
		if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
			return err
		}
	}

	if journalPath != "" {
		journal := WorkpadJournal{
			TaskID: task.ID,
			RunID:  runID,
			Now:    func() time.Time { return time.Unix(0, 0).UTC() },
		}
		journal.Record("definition", status, map[string]any{"workflow": definition.Name})
		if _, err := journal.Write(journalPath); err != nil {
			return err
		}
	}

	return nil
}
