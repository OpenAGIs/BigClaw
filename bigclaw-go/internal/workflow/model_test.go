package workflow

import (
	"encoding/json"
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestWorkflowTemplateAndRunRoundTripPreserveStepsAndOutputs(t *testing.T) {
	template := WorkflowTemplate{
		TemplateID:  "flow-template-1",
		Name:        "Risk Triage Flow",
		Version:     "v1",
		Description: "Routes risky work through triage and approval.",
		Trigger:     WorkflowTriggerEvent,
		DefaultRisk: domain.RiskMedium,
		Steps: []WorkflowTemplateStep{
			{
				StepID:        "triage",
				Name:          "Triage",
				Kind:          "review",
				RequiredTools: []string{"browser"},
				Approvals:     []string{"ops"},
				Metadata:      map[string]any{"lane": "risk"},
			},
			{
				StepID:    "approve",
				Name:      "Approval",
				Kind:      "approval",
				Approvals: []string{"security"},
			},
		},
		Tags:   []string{"risk", "triage"},
		Active: true,
	}
	run := WorkflowRun{
		RunID:       "flow-run-1",
		TemplateID:  template.TemplateID,
		TaskID:      "OPE-130",
		Status:      WorkflowRunRunning,
		TriggeredBy: "scheduler",
		StartedAt:   "2026-03-11T10:00:00Z",
		Steps: []WorkflowStepRun{
			{
				StepID:      "triage",
				Status:      WorkflowStepSucceeded,
				Actor:       "ops",
				StartedAt:   "2026-03-11T10:00:00Z",
				CompletedAt: "2026-03-11T10:02:00Z",
				Output:      map[string]any{"decision": "escalate"},
			},
			{
				StepID: "approve",
				Status: WorkflowStepRunning,
				Actor:  "security",
			},
		},
		Outputs:      map[string]any{"ticket": "SEC-42"},
		ApprovalRefs: []string{"security-review"},
	}

	templatePayload, err := json.Marshal(template)
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}
	runPayload, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("marshal run: %v", err)
	}

	var restoredTemplate WorkflowTemplate
	if err := json.Unmarshal(templatePayload, &restoredTemplate); err != nil {
		t.Fatalf("unmarshal template: %v", err)
	}
	var restoredRun WorkflowRun
	if err := json.Unmarshal(runPayload, &restoredRun); err != nil {
		t.Fatalf("unmarshal run: %v", err)
	}

	if !reflect.DeepEqual(restoredTemplate, template) {
		t.Fatalf("template round trip mismatch: restored=%+v want=%+v", restoredTemplate, template)
	}
	if !reflect.DeepEqual(restoredRun, run) {
		t.Fatalf("run round trip mismatch: restored=%+v want=%+v", restoredRun, run)
	}
	if restoredRun.Steps[0].Status != WorkflowStepSucceeded {
		t.Fatalf("expected succeeded step, got %+v", restoredRun)
	}
	if restoredTemplate.Trigger != WorkflowTriggerEvent {
		t.Fatalf("expected event trigger, got %+v", restoredTemplate)
	}
}

func TestWorkflowTemplateAndRunDefaultCompatibilityValues(t *testing.T) {
	var template FlowTemplate
	if err := json.Unmarshal([]byte(`{"template_id":"flow-template-2","name":"Closeout","version":"v2"}`), &template); err != nil {
		t.Fatalf("unmarshal template defaults: %v", err)
	}
	if template.Trigger != FlowTriggerManual {
		t.Fatalf("expected default trigger %q, got %q", FlowTriggerManual, template.Trigger)
	}
	if template.DefaultRisk != domain.RiskLow {
		t.Fatalf("expected default risk %q, got %q", domain.RiskLow, template.DefaultRisk)
	}
	if !template.Active {
		t.Fatalf("expected template to default active")
	}

	var stepRun FlowStepRun
	if err := json.Unmarshal([]byte(`{"step_id":"review"}`), &stepRun); err != nil {
		t.Fatalf("unmarshal step run defaults: %v", err)
	}
	if stepRun.Status != FlowStepPending {
		t.Fatalf("expected step default status %q, got %q", FlowStepPending, stepRun.Status)
	}

	var run FlowRun
	if err := json.Unmarshal([]byte(`{"run_id":"flow-run-2","template_id":"flow-template-2","task_id":"BIG-401"}`), &run); err != nil {
		t.Fatalf("unmarshal run defaults: %v", err)
	}
	if run.Status != FlowRunQueued {
		t.Fatalf("expected run default status %q, got %q", FlowRunQueued, run.Status)
	}
}
