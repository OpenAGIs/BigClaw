package workflow

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestDefinitionEngineRunsDefinitionEndToEnd(t *testing.T) {
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

	result, err := (DefinitionEngine{Now: func() time.Time { return time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC) }}).RunDefinition(task, definition, "run-dsl-1")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "accepted" {
		t.Fatalf("expected accepted result, got %+v", result)
	}
	if result.ReportPath == "" || result.JournalPath == "" {
		t.Fatalf("expected rendered paths, got %+v", result)
	}
	if _, err := ReadWorkpadJournal(result.JournalPath); err != nil {
		t.Fatalf("expected written journal, got %v", err)
	}
}

func TestDefinitionEngineRejectsUnknownStepKind(t *testing.T) {
	_, err := DefinitionEngine{}.RunDefinition(domain.Task{ID: "BIG-401-invalid"}, Definition{
		Name:  "broken-flow",
		Steps: []Step{{Name: "hack", Kind: "unknown-kind", Required: true}},
	}, "run-dsl-invalid")
	if err == nil || !strings.Contains(err.Error(), "invalid workflow step kind") {
		t.Fatalf("expected invalid workflow step kind error, got %v", err)
	}
}

func TestDefinitionEngineManualApprovalClosesHighRiskTask(t *testing.T) {
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

	result, err := DefinitionEngine{}.RunDefinition(task, definition, "run-dsl-2")
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Execution.Run.Status != WorkflowRunQueued {
		t.Fatalf("expected queued workflow run for manual approval, got %+v", result.Execution.Run)
	}
	if result.Acceptance.Status != "accepted" || len(result.Acceptance.Approvals) != 1 || result.Acceptance.Approvals[0] != "release-manager" {
		t.Fatalf("expected accepted approval result, got %+v", result.Acceptance)
	}
}
