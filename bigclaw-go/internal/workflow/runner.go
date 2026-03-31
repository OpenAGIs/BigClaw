package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/domain"
)

type DefinitionExecutionRun struct {
	Status string `json:"status"`
}

type DefinitionExecution struct {
	Run DefinitionExecutionRun `json:"run"`
}

type DefinitionRunResult struct {
	Execution   DefinitionExecution `json:"execution"`
	Acceptance  AcceptanceDecision  `json:"acceptance"`
	Journal     WorkpadJournal      `json:"journal"`
	JournalPath string              `json:"journal_path,omitempty"`
	ReportPath  string              `json:"report_path,omitempty"`
}

func RunDefinition(task domain.Task, definition Definition, runID string) (DefinitionRunResult, error) {
	if err := definition.Validate(); err != nil {
		return DefinitionRunResult{}, err
	}

	journal := WorkpadJournal{TaskID: task.ID, RunID: runID}
	journal.Record("intake", "recorded", map[string]any{"source": task.Source})

	status := "completed"
	approved := true
	if task.RiskLevel == domain.RiskHigh || definitionNeedsApproval(definition) {
		status = "needs-approval"
		approved = false
	}
	journal.Record("execution", status, map[string]any{"approved": approved})

	reportPath := definition.RenderReportPath(task, runID)
	if err := writeDefinitionArtifact(reportPath, renderDefinitionReport(task, definition, runID, status)); err != nil {
		return DefinitionRunResult{}, err
	}

	acceptance := AcceptanceGate{}.Evaluate(
		task,
		ExecutionOutcome{Approved: approved, Status: status},
		definition.ValidationEvidence,
		definition.Approvals,
		"",
	)
	journal.Record("acceptance", acceptance.Status, map[string]any{
		"passed":                      acceptance.Passed,
		"missing_acceptance_criteria": acceptance.MissingAcceptanceCriteria,
		"missing_validation_steps":    acceptance.MissingValidationSteps,
	})

	journalPath := definition.RenderJournalPath(task, runID)
	resolvedJournalPath := ""
	if strings.TrimSpace(journalPath) != "" {
		path, err := journal.Write(journalPath)
		if err != nil {
			return DefinitionRunResult{}, err
		}
		resolvedJournalPath = path
	}

	return DefinitionRunResult{
		Execution:   DefinitionExecution{Run: DefinitionExecutionRun{Status: status}},
		Acceptance:  acceptance,
		Journal:     journal,
		JournalPath: resolvedJournalPath,
		ReportPath:  reportPath,
	}, nil
}

func definitionNeedsApproval(definition Definition) bool {
	for _, step := range definition.Steps {
		if strings.EqualFold(strings.TrimSpace(step.Kind), "approval") {
			return true
		}
	}
	return false
}

func writeDefinitionArtifact(path string, content string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func renderDefinitionReport(task domain.Task, definition Definition, runID string, status string) string {
	return fmt.Sprintf(
		"# Workflow Definition Report\n\n- Workflow: %s\n- Task: %s\n- Run: %s\n- Status: %s\n",
		definition.Name,
		task.ID,
		runID,
		status,
	)
}
