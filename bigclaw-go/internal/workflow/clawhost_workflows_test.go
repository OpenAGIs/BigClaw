package workflow

import (
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestClawHostSkillsWorkflowDefinition(t *testing.T) {
	definition := ClawHostSkillsWorkflowDefinition()
	if definition.Name != ClawHostSkillsWorkflowName {
		t.Fatalf("unexpected definition name: %+v", definition)
	}
	if len(definition.Steps) != 6 {
		t.Fatalf("expected 6 workflow steps, got %+v", definition.Steps)
	}
	if got := definition.Approvals; !reflect.DeepEqual(got, []string{"tenant-owner-review", "device-approval-review"}) {
		t.Fatalf("unexpected approvals: %+v", got)
	}
	task := domain.Task{ID: "BIG-PAR-289", Source: "local"}
	if got := definition.RenderReportPath(task, "run-1"); got != "docs/reports/clawhost-skills-channels-device-approval/BIG-PAR-289-run-1.json" {
		t.Fatalf("unexpected report path: %q", got)
	}
	if got := definition.RenderJournalPath(task, "run-1"); got != "docs/reports/clawhost-skills-channels-device-approval/BIG-PAR-289-run-1.journal.json" {
		t.Fatalf("unexpected journal path: %q", got)
	}
	stage := definition.Steps[2]
	if stage.Kind != "batch" || stage.Metadata["max_parallelism"] != 3 || stage.Metadata["takeover_enabled"] != true {
		t.Fatalf("unexpected batch stage metadata: %+v", stage)
	}
}

func TestClawHostSkillsWorkflowTemplate(t *testing.T) {
	template := ClawHostSkillsWorkflowTemplate()
	if template.TemplateID != "clawhost-skills-workflow" || template.Name == "" || template.DefaultRisk != domain.RiskHigh || !template.Active {
		t.Fatalf("unexpected template header: %+v", template)
	}
	if len(template.Steps) != 5 {
		t.Fatalf("expected 5 template steps, got %+v", template.Steps)
	}
	if template.Steps[0].Metadata["workflow_definition"] != ClawHostSkillsWorkflowName {
		t.Fatalf("expected workflow definition reference in first step, got %+v", template.Steps[0])
	}
	if !reflect.DeepEqual(template.Steps[3].Approvals, []string{"device-approval-review"}) {
		t.Fatalf("unexpected device approval step: %+v", template.Steps[3])
	}
	if !reflect.DeepEqual(template.Tags, []string{"clawhost", "workflow", "parallel", "approval"}) {
		t.Fatalf("unexpected template tags: %+v", template.Tags)
	}
}
