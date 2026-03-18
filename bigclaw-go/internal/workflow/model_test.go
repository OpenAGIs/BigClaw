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
