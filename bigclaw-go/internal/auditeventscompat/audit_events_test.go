package auditeventscompat

import (
	"path/filepath"
	"reflect"
	"testing"
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
	if missing := MissingRequiredFields(SchedulerDecisionEvent, map[string]any{
		"task_id": "OPE-134",
		"run_id":  "run-ope-134",
		"medium":  "docker",
	}); !reflect.DeepEqual(missing, []string{"approved", "reason", "risk_level", "risk_score"}) {
		t.Fatalf("unexpected missing fields: %+v", missing)
	}
}

func TestTaskRunAuditSpecEventRequiresRequiredFields(t *testing.T) {
	run := NewRun(Task{TaskID: "OPE-134-spec", Source: "linear", Title: "Validate audit fields"}, "run-ope-134-spec", "docker")
	err := run.AuditSpecEvent(ManualTakeoverEvent, "scheduler", "pending", map[string]any{
		"task_id":     "OPE-134-spec",
		"run_id":      "run-ope-134-spec",
		"target_team": "security",
	})
	if err == nil {
		t.Fatal("expected missing required fields error")
	}
}

func TestSchedulerEmitsP0OperationalAuditEvents(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := Task{
		TaskID:               "OPE-134-scheduler",
		Source:               "linear",
		Title:                "Route cross-team rollout",
		Description:          "Needs coordinated release handling",
		Labels:               []string{"customer", "data"},
		Priority:             0,
		RequiredTools:        []string{"browser", "sql"},
		Budget:               120,
		BudgetOverrideActor:  "finance-controller",
		BudgetOverrideReason: "approved additional analytics validation spend",
		BudgetOverrideAmount: 30,
	}

	record, err := Scheduler{}.Execute(task, "run-ope-134-scheduler", ledger)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	auditsRaw, ok := entries[0]["audits"].([]any)
	if !ok {
		t.Fatalf("expected audits slice, got %+v", entries[0]["audits"])
	}
	audits := map[string]map[string]any{}
	for _, raw := range auditsRaw {
		audit := raw.(map[string]any)
		audits[asString(audit["action"])] = audit
	}

	if record.HandoffRequest == nil {
		t.Fatal("expected handoff request")
	}
	if details := mapValue(audits[SchedulerDecisionEvent]["details"]); details["risk_score"] == nil {
		t.Fatalf("expected scheduler decision risk score, got %+v", details)
	}
	if got := mapValue(audits[BudgetOverrideEvent]["details"]); !reflect.DeepEqual(got, map[string]any{
		"task_id":          "OPE-134-scheduler",
		"run_id":           "run-ope-134-scheduler",
		"requested_budget": float64(120),
		"approved_budget":  float64(150),
		"override_actor":   "finance-controller",
		"reason":           "approved additional analytics validation spend",
	}) {
		t.Fatalf("unexpected budget override details: %+v", got)
	}
	if got := mapValue(audits[ManualTakeoverEvent]["details"])["target_team"]; got != "operations" {
		t.Fatalf("unexpected manual takeover target team: %+v", got)
	}
	if got := mapValue(audits[FlowHandoffEvent]["details"])["source_stage"]; got != "scheduler" {
		t.Fatalf("unexpected handoff source stage: %+v", got)
	}
}

func TestWorkflowRecordsCanonicalApprovalEvent(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	engine := WorkflowEngine{}
	task := Task{
		TaskID:             "OPE-134-approval",
		Source:             "linear",
		Title:              "Approve production rollout",
		RiskLevel:          "high",
		AcceptanceCriteria: []string{"rollback-plan"},
		ValidationPlan:     []string{"integration-test"},
	}
	if err := engine.Run(task, "run-ope-134-approval", ledger, []string{"security-review"}, []string{"rollback-plan", "integration-test"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	audits := entries[0]["audits"].([]any)
	got := mapValue(audits[0].(map[string]any)["details"])
	if !reflect.DeepEqual(got, map[string]any{
		"task_id":           "OPE-134-approval",
		"run_id":            "run-ope-134-approval",
		"approvals":         []any{"security-review"},
		"approval_count":    float64(1),
		"acceptance_status": "accepted",
	}) {
		t.Fatalf("unexpected approval details: %+v", got)
	}
}

func TestReportsAcceptCanonicalHandoffAndTakeoverEvents(t *testing.T) {
	entry := map[string]any{
		"run_id":  "run-ope-134-canvas",
		"task_id": "OPE-134-canvas",
		"source":  "linear",
		"summary": "handoff requested",
		"audits": []any{
			map[string]any{
				"action":  "orchestration.plan",
				"actor":   "scheduler",
				"outcome": "ready",
				"details": map[string]any{
					"collaboration_mode": "cross-functional",
					"departments":        []any{"operations", "engineering"},
					"approvals":          []any{"security-review"},
				},
			},
			map[string]any{
				"action":  ManualTakeoverEvent,
				"actor":   "scheduler",
				"outcome": "pending",
				"details": map[string]any{
					"task_id":            "OPE-134-canvas",
					"run_id":             "run-ope-134-canvas",
					"target_team":        "security",
					"reason":             "manual review required",
					"requested_by":       "scheduler",
					"required_approvals": []any{"security-review"},
				},
			},
		},
	}

	canvas := BuildOrchestrationCanvasFromLedgerEntry(entry)
	queue := BuildTakeoverQueueFromLedger([]map[string]any{entry}, "Human Takeover Queue", "2026-03-11")

	if canvas.HandoffTeam != "security" {
		t.Fatalf("unexpected handoff team: %+v", canvas)
	}
	if !reflect.DeepEqual(queue.Requests[0].RequiredApprovals, []string{"security-review"}) {
		t.Fatalf("unexpected queue approvals: %+v", queue.Requests)
	}
}
