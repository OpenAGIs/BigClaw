package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonAuditEventsContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "audit_events_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    missing_required_fields,
)
from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.reports import build_orchestration_canvas_from_ledger_entry, build_takeover_queue_from_ledger
from bigclaw.scheduler import Scheduler
from bigclaw.workflow import WorkflowEngine

event_types = sorted(spec.event_type for spec in P0_AUDIT_EVENT_SPECS)
specs = {
    "event_types": event_types,
    "missing_fields": missing_required_fields(
        SCHEDULER_DECISION_EVENT,
        {
            "task_id": "OPE-134",
            "run_id": "run-ope-134",
            "medium": "docker",
        },
    ),
}

run = TaskRun.from_task(
    Task(task_id="OPE-134-spec", source="linear", title="Validate audit fields", description=""),
    run_id="run-ope-134-spec",
    medium="docker",
)
audit_error = ""
try:
    run.audit_spec_event(
        MANUAL_TAKEOVER_EVENT,
        "scheduler",
        "pending",
        task_id="OPE-134-spec",
        run_id="run-ope-134-spec",
        target_team="security",
    )
except ValueError as exc:
    audit_error = str(exc)

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="OPE-134-scheduler",
        source="linear",
        title="Route cross-team rollout",
        description="Needs coordinated release handling",
        labels=["customer", "data"],
        priority=Priority.P0,
        required_tools=["browser", "sql"],
        budget=120.0,
        budget_override_actor="finance-controller",
        budget_override_reason="approved additional analytics validation spend",
        budget_override_amount=30.0,
    )
    record = Scheduler().execute(task, run_id="run-ope-134-scheduler", ledger=ledger)
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}
    scheduler = {
        "has_handoff": record.handoff_request is not None,
        "risk_score_non_negative": audits[SCHEDULER_DECISION_EVENT]["details"]["risk_score"] >= 0,
        "budget_override": audits[BUDGET_OVERRIDE_EVENT]["details"],
        "target_team": audits[MANUAL_TAKEOVER_EVENT]["details"]["target_team"],
        "source_stage": audits[FLOW_HANDOFF_EVENT]["details"]["source_stage"],
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="OPE-134-approval",
        source="linear",
        title="Approve production rollout",
        description="Manual gate",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )
    WorkflowEngine().run(
        task,
        run_id="run-ope-134-approval",
        ledger=ledger,
        approvals=["security-review"],
        validation_evidence=["rollback-plan", "integration-test"],
    )
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}
    approval = audits[APPROVAL_RECORDED_EVENT]["details"]

entry = {
    "run_id": "run-ope-134-canvas",
    "task_id": "OPE-134-canvas",
    "source": "linear",
    "summary": "handoff requested",
    "audits": [
        {
            "action": "orchestration.plan",
            "actor": "scheduler",
            "outcome": "ready",
            "details": {
                "collaboration_mode": "cross-functional",
                "departments": ["operations", "engineering"],
                "approvals": ["security-review"],
            },
        },
        {
            "action": MANUAL_TAKEOVER_EVENT,
            "actor": "scheduler",
            "outcome": "pending",
            "details": {
                "task_id": "OPE-134-canvas",
                "run_id": "run-ope-134-canvas",
                "target_team": "security",
                "reason": "manual review required",
                "requested_by": "scheduler",
                "required_approvals": ["security-review"],
            },
        },
    ],
}
canvas = build_orchestration_canvas_from_ledger_entry(entry)
queue = build_takeover_queue_from_ledger([entry], period="2026-03-11")
reports = {
    "handoff_team": canvas.handoff_team,
    "required_approvals": queue.requests[0].required_approvals,
}

print(json.dumps({
    "specs": specs,
    "audit_error": audit_error,
    "scheduler": scheduler,
    "approval": approval,
    "reports": reports,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write audit events contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run audit events contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Specs struct {
			EventTypes    []string `json:"event_types"`
			MissingFields []string `json:"missing_fields"`
		} `json:"specs"`
		AuditError string `json:"audit_error"`
		Scheduler  struct {
			HasHandoff           bool                   `json:"has_handoff"`
			RiskScoreNonNegative bool                   `json:"risk_score_non_negative"`
			BudgetOverride       map[string]interface{} `json:"budget_override"`
			TargetTeam           string                 `json:"target_team"`
			SourceStage          string                 `json:"source_stage"`
		} `json:"scheduler"`
		Approval struct {
			TaskID           string   `json:"task_id"`
			RunID            string   `json:"run_id"`
			Approvals        []string `json:"approvals"`
			ApprovalCount    int      `json:"approval_count"`
			AcceptanceStatus string   `json:"acceptance_status"`
		} `json:"approval"`
		Reports struct {
			HandoffTeam       string   `json:"handoff_team"`
			RequiredApprovals []string `json:"required_approvals"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode audit events contract output: %v\n%s", err, string(output))
	}

	wantTypes := []string{
		"execution.approval_recorded",
		"execution.budget_override",
		"execution.flow_handoff",
		"execution.manual_takeover",
		"execution.scheduler_decision",
	}
	if len(decoded.Specs.EventTypes) != len(wantTypes) {
		t.Fatalf("unexpected p0 event types: %+v", decoded.Specs.EventTypes)
	}
	for i, want := range wantTypes {
		if decoded.Specs.EventTypes[i] != want {
			t.Fatalf("unexpected p0 event types: %+v", decoded.Specs.EventTypes)
		}
	}
	if len(decoded.Specs.MissingFields) != 4 || decoded.Specs.MissingFields[0] != "approved" || decoded.Specs.MissingFields[1] != "reason" || decoded.Specs.MissingFields[2] != "risk_level" || decoded.Specs.MissingFields[3] != "risk_score" {
		t.Fatalf("unexpected missing required fields: %+v", decoded.Specs.MissingFields)
	}
	if decoded.AuditError == "" {
		t.Fatalf("expected missing required fields error, got empty string")
	}
	if !decoded.Scheduler.HasHandoff || !decoded.Scheduler.RiskScoreNonNegative || decoded.Scheduler.TargetTeam != "operations" || decoded.Scheduler.SourceStage != "scheduler" {
		t.Fatalf("unexpected scheduler audit payload: %+v", decoded.Scheduler)
	}
	if decoded.Scheduler.BudgetOverride["task_id"] != "OPE-134-scheduler" ||
		decoded.Scheduler.BudgetOverride["run_id"] != "run-ope-134-scheduler" ||
		decoded.Scheduler.BudgetOverride["requested_budget"] != float64(120) ||
		decoded.Scheduler.BudgetOverride["approved_budget"] != float64(150) ||
		decoded.Scheduler.BudgetOverride["override_actor"] != "finance-controller" ||
		decoded.Scheduler.BudgetOverride["reason"] != "approved additional analytics validation spend" {
		t.Fatalf("unexpected budget override payload: %+v", decoded.Scheduler.BudgetOverride)
	}
	if decoded.Approval.TaskID != "OPE-134-approval" || decoded.Approval.RunID != "run-ope-134-approval" || len(decoded.Approval.Approvals) != 1 || decoded.Approval.Approvals[0] != "security-review" || decoded.Approval.ApprovalCount != 1 || decoded.Approval.AcceptanceStatus != "accepted" {
		t.Fatalf("unexpected canonical approval payload: %+v", decoded.Approval)
	}
	if decoded.Reports.HandoffTeam != "security" || len(decoded.Reports.RequiredApprovals) != 1 || decoded.Reports.RequiredApprovals[0] != "security-review" {
		t.Fatalf("unexpected report handoff payload: %+v", decoded.Reports)
	}
}
