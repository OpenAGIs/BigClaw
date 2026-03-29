package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestRunDefinitionEndToEndWritesReportAndJournal(t *testing.T) {
	root := t.TempDir()
	definition := Definition{
		Name:                "acceptance-closeout",
		Steps:               []Step{{Name: "execute", Kind: "scheduler", Required: true}},
		ReportPathTemplate:  filepath.Join(root, "reports", "{task_id}", "{run_id}.md"),
		JournalPathTemplate: filepath.Join(root, "journals", "{workflow}", "{run_id}.json"),
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
		t.Fatalf("expected accepted result, got %+v", result)
	}
	if _, err := os.Stat(definition.RenderReportPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected report artifact: %v", err)
	}
	if _, err := os.Stat(definition.RenderJournalPath(task, "run-dsl-1")); err != nil {
		t.Fatalf("expected journal artifact: %v", err)
	}
}

func TestRunDefinitionRejectsUnknownStepKind(t *testing.T) {
	definition := Definition{
		Name:  "broken-flow",
		Steps: []Step{{Name: "hack", Kind: "unknown-kind", Required: true}},
	}
	task := domain.Task{ID: "BIG-401-invalid", Source: "local", Title: "invalid"}

	_, err := RunDefinition(task, definition, "run-dsl-invalid")
	if err == nil || !strings.Contains(err.Error(), "invalid workflow step kind") {
		t.Fatalf("expected invalid workflow step kind error, got %v", err)
	}
}

func TestRunDefinitionManualApprovalClosesHighRiskTask(t *testing.T) {
	definition := Definition{
		Name:               "prod-approval",
		Steps:              []Step{{Name: "review", Kind: "approval", Required: true}},
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
		t.Fatalf("expected needs-approval execution status, got %+v", result)
	}
	if result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted closeout with explicit approvals, got %+v", result)
	}
	if len(result.Acceptance.Approvals) != 1 || result.Acceptance.Approvals[0] != "release-manager" {
		t.Fatalf("unexpected approvals: %+v", result.Acceptance.Approvals)
	}
}
