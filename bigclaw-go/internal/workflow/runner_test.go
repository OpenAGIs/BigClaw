package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestWorkflowEngineRunsDefinitionEndToEnd(t *testing.T) {
	dir := t.TempDir()
	definition, err := ParseDefinition(`{"name":"acceptance-closeout","steps":[{"name":"execute","kind":"scheduler"}],"report_path_template":"` + filepath.Join(dir, "reports", "{task_id}", "{run_id}.md") + `","journal_path_template":"` + filepath.Join(dir, "journals", "{workflow}", "{run_id}.json") + `","validation_evidence":["pytest","report-shared"]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	task := domain.Task{
		ID:                 "BIG-401-flow",
		Source:             "linear",
		Title:              "Run workflow definition",
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
	}

	result, err := (WorkflowEngine{}).RunDefinition(task, definition, "run-dsl-1")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted result, got %+v", result)
	}
	if _, err := os.Stat(definition.RenderReportPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected report path to exist: %v", err)
	}
	if _, err := os.Stat(definition.RenderJournalPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected journal path to exist: %v", err)
	}
}

func TestWorkflowDefinitionRejectsUnknownStepKind(t *testing.T) {
	definition := Definition{
		Name:  "broken-flow",
		Steps: []Step{{Name: "hack", Kind: "unknown-kind"}},
	}
	task := domain.Task{ID: "BIG-401-invalid", Source: "local", Title: "invalid"}

	if _, err := (WorkflowEngine{}).RunDefinition(task, definition, "run-dsl-invalid"); err == nil || !strings.Contains(err.Error(), "invalid workflow step kind") {
		t.Fatalf("expected invalid workflow step kind error, got %v", err)
	}
}

func TestWorkflowDefinitionManualApprovalClosesHighRiskTask(t *testing.T) {
	definition := Definition{
		Name:               "prod-approval",
		Steps:              []Step{{Name: "review", Kind: "approval"}},
		ValidationEvidence: []string{"rollback-plan", "integration-test"},
		Approvals:          []string{"release-manager"},
	}
	task := domain.Task{
		ID:                 "BIG-403-dsl",
		Source:             "linear",
		Title:              "Prod rollout",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"rollback-plan"},
		ValidationPlan:     []string{"integration-test"},
	}

	result, err := (WorkflowEngine{}).RunDefinition(task, definition, "run-dsl-2")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Execution.Run.Status != "needs-approval" {
		t.Fatalf("expected needs-approval execution status, got %+v", result.Execution)
	}
	if result.Acceptance.Status != "accepted" || len(result.Acceptance.Approvals) != 1 || result.Acceptance.Approvals[0] != "release-manager" {
		t.Fatalf("unexpected acceptance result: %+v", result.Acceptance)
	}
}
