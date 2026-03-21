package workflow

import (
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildCloseoutRendersWorkflowDefinitionArtifacts(t *testing.T) {
	startedAt := time.Date(2026, 3, 19, 2, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(2 * time.Minute)
	task := domain.Task{
		ID:        "task-closeout-1",
		Source:    "local",
		CreatedAt: startedAt,
		Metadata: map[string]string{
			"created_by": "pm-1",
			"workflow_definition": `{
				"name":"release-closeout",
				"report_path_template":"reports/{task_id}/{run_id}.md",
				"journal_path_template":"journals/{workflow}/{run_id}.json",
				"validation_evidence":["pytest","smoke"],
				"approvals":["ops-review","security-review"]
			}`,
		},
	}

	closeout := BuildCloseout(CloseoutInput{
		Task:        task,
		RunID:       "run-123",
		Status:      WorkflowRunSucceeded,
		Executor:    domain.ExecutorLocal,
		Message:     "ok",
		Artifacts:   []string{"artifact://bundle"},
		StartedAt:   startedAt,
		CompletedAt: completedAt,
	})

	if closeout.ReportPath != "reports/task-closeout-1/run-123.md" {
		t.Fatalf("expected rendered report path, got %+v", closeout)
	}
	if closeout.JournalPath != "journals/release-closeout/run-123.json" {
		t.Fatalf("expected rendered journal path, got %+v", closeout)
	}
	if len(closeout.ValidationEvidence) != 2 || closeout.ValidationEvidence[0] != "pytest" || closeout.ValidationEvidence[1] != "smoke" {
		t.Fatalf("expected validation evidence from workflow definition, got %+v", closeout.ValidationEvidence)
	}
	if len(closeout.RequiredApprovals) != 2 || closeout.RequiredApprovals[0] != "ops-review" || closeout.RequiredApprovals[1] != "security-review" {
		t.Fatalf("expected approvals from workflow definition, got %+v", closeout.RequiredApprovals)
	}
	if closeout.Run.RunID != "run-123" || closeout.Run.TemplateID != "release-closeout" || closeout.Run.TaskID != task.ID || closeout.Run.Status != WorkflowRunSucceeded {
		t.Fatalf("unexpected workflow run payload: %+v", closeout.Run)
	}
	if closeout.Run.TriggeredBy != "pm-1" || closeout.Run.StartedAt != startedAt.Format(time.RFC3339) || closeout.Run.CompletedAt != completedAt.Format(time.RFC3339) {
		t.Fatalf("expected timestamps and triggered_by in workflow run, got %+v", closeout.Run)
	}
	if closeout.Run.Outputs["report_path"] != closeout.ReportPath || closeout.Run.Outputs["journal_path"] != closeout.JournalPath {
		t.Fatalf("expected rendered paths in workflow outputs, got %+v", closeout.Run.Outputs)
	}
}

func TestBuildCloseoutFallsBackToTaskMetadataAndValidationPlans(t *testing.T) {
	startedAt := time.Date(2026, 3, 19, 3, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(30 * time.Second)
	task := domain.Task{
		ID:                 "task-closeout-2",
		Source:             "triage",
		AcceptanceCriteria: []string{"ship report", "capture evidence"},
		ValidationPlan:     []string{"replay trace", "capture evidence"},
		Metadata: map[string]string{
			"workflow":              "deploy",
			"template":              "release-template",
			"report_path_template":  "reports/{workflow}/{run_id}.md",
			"journal_path_template": "journals/{task_id}/{run_id}.json",
			"required_approvals":    `["ops-review","security-review"]`,
		},
	}

	closeout := BuildCloseout(CloseoutInput{
		Task:           task,
		RunID:          "run-456",
		Status:         WorkflowRunFailed,
		Executor:       domain.ExecutorKubernetes,
		Message:        "pod crashed",
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		RetryScheduled: true,
	})

	if closeout.ReportPath != "reports/deploy/run-456.md" {
		t.Fatalf("expected fallback report path, got %+v", closeout)
	}
	if closeout.JournalPath != "journals/task-closeout-2/run-456.json" {
		t.Fatalf("expected fallback journal path, got %+v", closeout)
	}
	if len(closeout.ValidationEvidence) != 3 || closeout.ValidationEvidence[0] != "replay trace" || closeout.ValidationEvidence[1] != "capture evidence" || closeout.ValidationEvidence[2] != "ship report" {
		t.Fatalf("expected merged validation evidence, got %+v", closeout.ValidationEvidence)
	}
	if len(closeout.RequiredApprovals) != 2 || closeout.RequiredApprovals[0] != "ops-review" || closeout.RequiredApprovals[1] != "security-review" {
		t.Fatalf("expected approvals from metadata, got %+v", closeout.RequiredApprovals)
	}
	if closeout.Run.TemplateID != "release-template" || closeout.Run.Status != WorkflowRunFailed {
		t.Fatalf("unexpected fallback workflow run payload: %+v", closeout.Run)
	}
	if closeout.Run.Outputs["retry_scheduled"] != true {
		t.Fatalf("expected retry_scheduled output, got %+v", closeout.Run.Outputs)
	}
}
