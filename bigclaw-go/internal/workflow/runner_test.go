package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestRunDefinitionWritesArtifactsAndAcceptsCompleteEvidence(t *testing.T) {
	dir := t.TempDir()
	definition := Definition{
		Name: "acceptance-closeout",
		Steps: []Step{
			{Name: "execute", Kind: "scheduler"},
		},
		ReportPathTemplate:  filepath.Join(dir, "reports", "{task_id}", "{run_id}.md"),
		JournalPathTemplate: filepath.Join(dir, "journals", "{workflow}", "{run_id}.json"),
		ValidationEvidence:  []string{"pytest", "report-shared"},
	}
	task := domain.Task{
		ID:                 "BIG-401-flow",
		Source:             "linear",
		Title:              "Run workflow definition",
		AcceptanceCriteria: []string{"report-shared"},
		ValidationPlan:     []string{"pytest"},
	}

	result, err := RunDefinition(task, definition, "run-dsl-1")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted result, got %+v", result.Acceptance)
	}
	if _, err := os.Stat(definition.RenderReportPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected report artifact: %v", err)
	}
	if _, err := os.Stat(definition.RenderJournalPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected journal artifact: %v", err)
	}
}

func TestRunDefinitionManualApprovalClosesHighRiskTask(t *testing.T) {
	definition := Definition{
		Name: "prod-approval",
		Steps: []Step{
			{Name: "review", Kind: "approval"},
		},
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

	result, err := RunDefinition(task, definition, "run-dsl-2")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Execution.Run.Status != "needs-approval" {
		t.Fatalf("expected needs-approval execution status, got %+v", result.Execution)
	}
	if result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted high-risk approval result, got %+v", result.Acceptance)
	}
	if len(result.Acceptance.Approvals) != 1 || result.Acceptance.Approvals[0] != "release-manager" {
		t.Fatalf("expected release-manager approval, got %+v", result.Acceptance.Approvals)
	}
}

func TestDefinitionValidateRejectsUnknownStepKinds(t *testing.T) {
	definition := Definition{
		Name: "broken-flow",
		Steps: []Step{
			{Name: "hack", Kind: "unknown-kind"},
		},
	}

	err := definition.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid workflow step kind") {
		t.Fatalf("expected invalid step kind error, got %v", err)
	}
}
