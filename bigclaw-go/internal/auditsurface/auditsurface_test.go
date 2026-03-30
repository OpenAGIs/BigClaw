package auditsurface

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestP0AuditEventSpecsDefineRequiredOperationalEvents(t *testing.T) {
	eventTypes := make([]string, 0, len(P0AuditEventSpecs))
	for _, spec := range P0AuditEventSpecs {
		eventTypes = append(eventTypes, spec.EventType)
	}
	if !reflect.DeepEqual(eventTypes, []string{
		SchedulerDecisionEvent,
		ManualTakeoverEvent,
		ApprovalRecordedEvent,
		BudgetOverrideEvent,
		FlowHandoffEvent,
	}) {
		t.Fatalf("unexpected event types: %+v", eventTypes)
	}
	missing := MissingRequiredFields(SchedulerDecisionEvent, map[string]any{
		"task_id": "OPE-134",
		"run_id":  "run-ope-134",
		"medium":  "docker",
	})
	if !reflect.DeepEqual(missing, []string{"approved", "reason", "risk_level", "risk_score"}) {
		t.Fatalf("unexpected missing required fields: %+v", missing)
	}
}

func TestTaskRunAuditSpecEventRequiresRequiredFields(t *testing.T) {
	run := NewTaskRun(domain.Task{ID: "OPE-134-spec", Source: "linear", Title: "Validate audit fields"}, "run-ope-134-spec", "docker")
	err := run.AuditSpecEvent(ManualTakeoverEvent, "scheduler", "pending", map[string]any{
		"task_id":     "OPE-134-spec",
		"run_id":      "run-ope-134-spec",
		"target_team": "security",
	})
	var missingErr *MissingFieldsError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected missing fields error, got %v", err)
	}
}

func TestSchedulerEmitsP0OperationalAuditEvents(t *testing.T) {
	ledger := NewObservabilityLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{
		ID:                   "OPE-134-scheduler",
		Source:               "linear",
		Title:                "Route cross-team rollout",
		Description:          "Needs coordinated release handling",
		Labels:               []string{"customer", "data"},
		RequiredTools:        []string{"browser", "sql"},
		BudgetCents:          12000,
		BudgetOverrideActor:  "finance-controller",
		BudgetOverrideReason: "approved additional analytics validation spend",
		BudgetOverrideAmount: 30.0,
	}

	record, err := (Scheduler{}).Execute(task, "run-ope-134-scheduler", ledger)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	audits := map[string]AuditEntry{}
	for _, audit := range entries[0].Audits {
		audits[audit.Action] = audit
	}
	if record.HandoffRequest == nil {
		t.Fatalf("expected handoff request, got %+v", record)
	}
	if riskScore, ok := audits[SchedulerDecisionEvent].Details["risk_score"].(float64); !ok || riskScore < 0 {
		t.Fatalf("expected non-negative risk score, got %+v", audits[SchedulerDecisionEvent])
	}
	if !reflect.DeepEqual(audits[BudgetOverrideEvent].Details, map[string]any{
		"task_id":          "OPE-134-scheduler",
		"run_id":           "run-ope-134-scheduler",
		"requested_budget": 120.0,
		"approved_budget":  150.0,
		"override_actor":   "finance-controller",
		"reason":           "approved additional analytics validation spend",
	}) {
		t.Fatalf("unexpected budget override audit: %+v", audits[BudgetOverrideEvent].Details)
	}
	if audits[ManualTakeoverEvent].Details["target_team"] != "operations" {
		t.Fatalf("unexpected takeover audit: %+v", audits[ManualTakeoverEvent])
	}
	if audits[FlowHandoffEvent].Details["source_stage"] != "scheduler" {
		t.Fatalf("unexpected handoff audit: %+v", audits[FlowHandoffEvent])
	}
}

func TestWorkflowRecordsCanonicalApprovalEvent(t *testing.T) {
	ledger := NewObservabilityLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{
		ID:                 "OPE-134-approval",
		Source:             "linear",
		Title:              "Approve production rollout",
		Description:        "Manual gate",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"rollback-plan"},
		ValidationPlan:     []string{"integration-test"},
	}

	if err := (WorkflowEngine{}).Run(task, "run-ope-134-approval", ledger, []string{"security-review"}, []string{"rollback-plan", "integration-test"}); err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	audits := map[string]AuditEntry{}
	for _, audit := range entries[0].Audits {
		audits[audit.Action] = audit
	}
	if !reflect.DeepEqual(audits[ApprovalRecordedEvent].Details, map[string]any{
		"task_id":           "OPE-134-approval",
		"run_id":            "run-ope-134-approval",
		"approvals":         []string{"security-review"},
		"approval_count":    1,
		"acceptance_status": "accepted",
	}) {
		t.Fatalf("unexpected approval audit: %+v", audits[ApprovalRecordedEvent].Details)
	}
}

func TestReportsAcceptCanonicalHandoffAndTakeoverEvents(t *testing.T) {
	entry := LedgerEntry{
		RunID:   "run-ope-134-canvas",
		TaskID:  "OPE-134-canvas",
		Source:  "linear",
		Summary: "handoff requested",
		Audits: []AuditEntry{
			{
				Action:  "orchestration.plan",
				Actor:   "scheduler",
				Outcome: "ready",
				Details: map[string]any{"collaboration_mode": "cross-functional", "departments": []string{"operations", "engineering"}, "approvals": []string{"security-review"}},
			},
			{
				Action:  ManualTakeoverEvent,
				Actor:   "scheduler",
				Outcome: "pending",
				Details: map[string]any{"task_id": "OPE-134-canvas", "run_id": "run-ope-134-canvas", "target_team": "security", "reason": "manual review required", "requested_by": "scheduler", "required_approvals": []string{"security-review"}},
			},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	queue := BuildTakeoverQueueFromLedger([]LedgerEntry{entry}, "2026-03-11")
	if canvas.HandoffTeam != "security" {
		t.Fatalf("unexpected canvas: %+v", canvas)
	}
	if !reflect.DeepEqual(queue.Requests[0].RequiredApprovals, []string{"security-review"}) {
		t.Fatalf("unexpected queue: %+v", queue)
	}
}
