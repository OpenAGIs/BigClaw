package workflow

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestDefinitionParsesAndRendersTemplates(t *testing.T) {
	definition, err := ParseDefinition(
		`{"name":"release-closeout","steps":[{"name":"execute","kind":"scheduler"}],"report_path_template":"reports/{task_id}/{run_id}.md","journal_path_template":"journals/{workflow}/{run_id}.json","validation_evidence":["pytest"],"approvals":["ops-review"]}`,
	)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	task := domain.Task{ID: "BIG-401", Source: "linear", Title: "DSL"}
	if definition.Steps[0].Name != "execute" {
		t.Fatalf("expected first step name execute, got %q", definition.Steps[0].Name)
	}
	if got := definition.RenderReportPath(task, "run-1"); got != "reports/BIG-401/run-1.md" {
		t.Fatalf("unexpected report path: %q", got)
	}
	if got := definition.RenderJournalPath(task, "run-1"); got != "journals/release-closeout/run-1.json" {
		t.Fatalf("unexpected journal path: %q", got)
	}
	if !definition.Steps[0].Required {
		t.Fatalf("expected step required default true")
	}
}

func TestDefinitionRespectsExplicitOptionalStep(t *testing.T) {
	definition, err := ParseDefinition(`{"name":"optional-closeout","steps":[{"name":"review","kind":"approval","required":false,"metadata":{"lane":"risk"}}]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	if definition.Steps[0].Required {
		t.Fatalf("expected explicit required=false to be preserved")
	}
	if got := definition.Steps[0].Metadata["lane"]; got != "risk" {
		t.Fatalf("expected metadata lane risk, got %#v", got)
	}
}
