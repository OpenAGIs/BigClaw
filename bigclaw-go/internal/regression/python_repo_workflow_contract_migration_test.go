package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPythonRepoWorkflowContractMigration(t *testing.T) {
	repoRoot := repoRoot(t)
	payload := runPythonRepoWorkflowContracts(t, repoRoot)

	control := payload.ControlCenter
	if control.PeekOrder[0] != "p0" || control.PeekOrder[1] != "p1" || control.PeekOrder[2] != "p2" {
		t.Fatalf("unexpected queue peek order: %+v", control.PeekOrder)
	}
	if control.QueueDepth != 3 || control.WaitingApprovalRuns != 1 {
		t.Fatalf("unexpected queue control center summary: %+v", control)
	}
	if control.PriorityP0 != 1 || control.PriorityP1 != 1 || control.PriorityP2 != 1 {
		t.Fatalf("unexpected queue priority counts: %+v", control)
	}
	if control.RiskLow != 1 || control.RiskMedium != 1 || control.RiskHigh != 1 {
		t.Fatalf("unexpected queue risk counts: %+v", control)
	}
	if control.MediumVM != 1 || control.MediumBrowser != 1 || control.MediumDocker != 1 {
		t.Fatalf("unexpected queue medium counts: %+v", control)
	}
	if len(control.BlockedTasks) != 1 || control.BlockedTasks[0] != "BIG-802-1" {
		t.Fatalf("unexpected blocked tasks: %+v", control.BlockedTasks)
	}
	if len(control.QueuedTasks) != 3 || control.QueuedTasks[0] != "BIG-802-1" || control.QueuedTasks[2] != "BIG-802-3" {
		t.Fatalf("unexpected queued tasks: %+v", control.QueuedTasks)
	}
	expectedActions := []string{"drill-down", "export", "add-note", "escalate", "retry", "pause", "reassign", "audit"}
	for i, actionID := range expectedActions {
		if control.BlockedActionIDs[i] != actionID {
			t.Fatalf("unexpected action order: %+v", control.BlockedActionIDs)
		}
	}
	if !control.EscalateEnabled || !control.RetryEnabled || control.PauseEnabled {
		t.Fatalf("unexpected action enablement: %+v", control)
	}
	if !control.ReportHasHeading || !control.ReportHasWaitingApproval || !control.ReportHasBlockedTask || !control.ReportHasDrillDown || !control.ReportHasEscalate || !control.ReportHasPauseReason {
		t.Fatalf("unexpected queue report rendering: %+v", control)
	}
	if !control.EmptyHasViewState || !control.EmptyHasEmptyState || !control.EmptyHasSummary || !control.EmptyHasTeamFilter {
		t.Fatalf("unexpected empty-state rendering: %+v", control)
	}

	eventBus := payload.EventBus
	if eventBus.PRStatus != "approved" || eventBus.PRSummary != "LGTM, merge when green." || len(eventBus.PRSeenStatuses) != 1 || eventBus.PRSeenStatuses[0] != "approved" {
		t.Fatalf("unexpected PR event bus contract: %+v", eventBus)
	}
	if !eventBus.PRHasCommentAudit || !eventBus.PRHasTransitionAudit || eventBus.CIStatus != "completed" || eventBus.CISummary != "CI workflow pytest completed with success" || !eventBus.CIHasEventAudit || eventBus.FailStatus != "failed" || eventBus.FailSummary != "sandbox command exited 137" || !eventBus.FailHasTransitionAudit {
		t.Fatalf("unexpected event bus contract: %+v", eventBus)
	}

	dsl := payload.DSL
	if dsl.FirstStepName != "execute" || dsl.ReportPath != "reports/BIG-401/run-1.md" || dsl.JournalPath != "journals/release-closeout/run-1.json" {
		t.Fatalf("unexpected DSL template rendering: %+v", dsl)
	}
	if dsl.EngineAcceptanceStatus != "accepted" || !dsl.EngineReportExists || !dsl.EngineJournalExists {
		t.Fatalf("unexpected DSL workflow engine result: %+v", dsl)
	}
	if dsl.InvalidError != "invalid workflow step kind(s): unknown-kind" {
		t.Fatalf("unexpected DSL invalid-step error: %+v", dsl.InvalidError)
	}
	if dsl.ApprovalRunStatus != "needs-approval" || dsl.ApprovalAcceptanceStatus != "accepted" || len(dsl.Approvals) != 1 || dsl.Approvals[0] != "release-manager" {
		t.Fatalf("unexpected DSL approval flow: %+v", dsl)
	}

	audit := payload.AuditEvents
	if len(audit.EventTypes) != 5 || audit.MissingSchedulerFields[0] != "approved" || audit.MissingSchedulerFields[3] != "risk_score" {
		t.Fatalf("unexpected audit event catalog: %+v", audit)
	}
	if audit.SpecError != "audit event execution.manual_takeover missing required fields: reason, requested_by, required_approvals" {
		t.Fatalf("unexpected audit spec error: %+v", audit.SpecError)
	}
	if !audit.HasHandoffRequest || audit.SchedulerRiskScore < 0 || !audit.BudgetOverrideMatches || audit.ManualTakeoverTarget != "operations" || audit.FlowHandoffSource != "scheduler" {
		t.Fatalf("unexpected scheduler audit event contract: %+v", audit)
	}
	if !audit.ApprovalRecordedMatches || audit.CanvasHandoffTeam != "security" || len(audit.QueueApprovals) != 1 || audit.QueueApprovals[0] != "security-review" {
		t.Fatalf("unexpected approval/canvas audit contract: %+v", audit)
	}
}

type pythonRepoWorkflowContractsPayload struct {
	ControlCenter struct {
		PeekOrder                []string `json:"peek_order"`
		QueueDepth               int      `json:"queue_depth"`
		PriorityP0               int      `json:"priority_p0"`
		PriorityP1               int      `json:"priority_p1"`
		PriorityP2               int      `json:"priority_p2"`
		RiskLow                  int      `json:"risk_low"`
		RiskMedium               int      `json:"risk_medium"`
		RiskHigh                 int      `json:"risk_high"`
		MediumVM                 int      `json:"medium_vm"`
		MediumBrowser            int      `json:"medium_browser"`
		MediumDocker             int      `json:"medium_docker"`
		WaitingApprovalRuns      int      `json:"waiting_approval_runs"`
		BlockedTasks             []string `json:"blocked_tasks"`
		QueuedTasks              []string `json:"queued_tasks"`
		BlockedActionIDs         []string `json:"blocked_action_ids"`
		EscalateEnabled          bool     `json:"escalate_enabled"`
		RetryEnabled             bool     `json:"retry_enabled"`
		PauseEnabled             bool     `json:"pause_enabled"`
		ReportHasHeading         bool     `json:"report_has_heading"`
		ReportHasWaitingApproval bool     `json:"report_has_waiting_approval"`
		ReportHasBlockedTask     bool     `json:"report_has_blocked_task"`
		ReportHasDrillDown       bool     `json:"report_has_drill_down"`
		ReportHasEscalate        bool     `json:"report_has_escalate"`
		ReportHasPauseReason     bool     `json:"report_has_pause_reason"`
		EmptyHasViewState        bool     `json:"empty_has_view_state"`
		EmptyHasEmptyState       bool     `json:"empty_has_empty_state"`
		EmptyHasSummary          bool     `json:"empty_has_summary"`
		EmptyHasTeamFilter       bool     `json:"empty_has_team_filter"`
	} `json:"control_center"`
	EventBus struct {
		PRStatus               string   `json:"pr_status"`
		PRSummary              string   `json:"pr_summary"`
		PRSeenStatuses         []string `json:"pr_seen_statuses"`
		PRHasCommentAudit      bool     `json:"pr_has_comment_audit"`
		PRHasTransitionAudit   bool     `json:"pr_has_transition_audit"`
		CIStatus               string   `json:"ci_status"`
		CISummary              string   `json:"ci_summary"`
		CIHasEventAudit        bool     `json:"ci_has_event_audit"`
		FailStatus             string   `json:"fail_status"`
		FailSummary            string   `json:"fail_summary"`
		FailHasTransitionAudit bool     `json:"fail_has_transition_audit"`
	} `json:"event_bus"`
	DSL struct {
		FirstStepName            string   `json:"first_step_name"`
		ReportPath               string   `json:"report_path"`
		JournalPath              string   `json:"journal_path"`
		EngineAcceptanceStatus   string   `json:"engine_acceptance_status"`
		EngineReportExists       bool     `json:"engine_report_exists"`
		EngineJournalExists      bool     `json:"engine_journal_exists"`
		InvalidError             string   `json:"invalid_error"`
		ApprovalRunStatus        string   `json:"approval_run_status"`
		ApprovalAcceptanceStatus string   `json:"approval_acceptance_status"`
		Approvals                []string `json:"approvals"`
	} `json:"dsl"`
	AuditEvents struct {
		EventTypes              []string `json:"event_types"`
		MissingSchedulerFields  []string `json:"missing_scheduler_fields"`
		SpecError               string   `json:"spec_error"`
		HasHandoffRequest       bool     `json:"has_handoff_request"`
		SchedulerRiskScore      int      `json:"scheduler_risk_score"`
		BudgetOverrideMatches   bool     `json:"budget_override_matches"`
		ManualTakeoverTarget    string   `json:"manual_takeover_target"`
		FlowHandoffSource       string   `json:"flow_handoff_source"`
		ApprovalRecordedMatches bool     `json:"approval_recorded_matches"`
		CanvasHandoffTeam       string   `json:"canvas_handoff_team"`
		QueueApprovals          []string `json:"queue_approvals"`
	} `json:"audit_events"`
}

func runPythonRepoWorkflowContracts(t *testing.T, repoRoot string) pythonRepoWorkflowContractsPayload {
	t.Helper()

	code := `
import json
import tempfile
from pathlib import Path

from bigclaw.audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    missing_required_fields,
)
from bigclaw.dsl import WorkflowDefinition
from bigclaw.event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
)
from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.operations import OperationsAnalytics, render_queue_control_center
from bigclaw.queue import PersistentTaskQueue
from bigclaw.reports import (
    SharedViewContext,
    SharedViewFilter,
    build_orchestration_canvas_from_ledger_entry,
    build_takeover_queue_from_ledger,
)
from bigclaw.scheduler import Scheduler
from bigclaw.workflow import WorkflowEngine

with tempfile.TemporaryDirectory() as tmpdir:
    tmp = Path(tmpdir)

    peek_queue = PersistentTaskQueue(str(tmp / "peek-queue.json"))
    peek_queue.enqueue(Task(task_id="p2", source="linear", title="low", description="", priority=Priority.P2))
    peek_queue.enqueue(Task(task_id="p0", source="linear", title="top", description="", priority=Priority.P0))
    peek_queue.enqueue(Task(task_id="p1", source="linear", title="mid", description="", priority=Priority.P1))
    peek_order = [task.task_id for task in peek_queue.peek_tasks()]

    queue = PersistentTaskQueue(str(tmp / "queue.json"))
    queue.enqueue(Task(task_id="BIG-802-1", source="linear", title="top", description="", priority=Priority.P0, risk_level=RiskLevel.HIGH))
    queue.enqueue(Task(task_id="BIG-802-2", source="linear", title="mid", description="", priority=Priority.P1, risk_level=RiskLevel.MEDIUM))
    queue.enqueue(Task(task_id="BIG-802-3", source="linear", title="low", description="", priority=Priority.P2, risk_level=RiskLevel.LOW))
    center = OperationsAnalytics().build_queue_control_center(
        queue,
        runs=[
            {"task_id": "BIG-802-1", "status": "needs-approval", "medium": "vm"},
            {"task_id": "BIG-802-2", "status": "approved", "medium": "browser"},
            {"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
        ],
    )
    report = render_queue_control_center(center)
    empty_report = render_queue_control_center(
        OperationsAnalytics().build_queue_control_center(PersistentTaskQueue(str(tmp / "empty-queue.json")), runs=[]),
        view=SharedViewContext(
            filters=[SharedViewFilter(label="Team", value="operations")],
            result_count=0,
            empty_message="No queued work for the selected team.",
        ),
    )

    ledger = ObservabilityLedger(str(tmp / "ledger-pr.json"))
    task = Task(task_id="BIG-203-pr", source="github", title="PR approval", description="")
    run = TaskRun.from_task(task, run_id="run-pr-1", medium="vm")
    run.finalize("needs-approval", "waiting for reviewer comment")
    ledger.append(run)
    bus = EventBus(ledger=ledger)
    seen_statuses = []
    bus.subscribe(PULL_REQUEST_COMMENT_EVENT, lambda _event, current: seen_statuses.append(current.status))
    updated_pr = bus.publish(BusEvent(
        event_type=PULL_REQUEST_COMMENT_EVENT,
        run_id=run.run_id,
        actor="reviewer",
        details={"decision": "approved", "body": "LGTM, merge when green.", "mentions": ["ops"]},
    ))
    persisted_pr = ledger.load()[0]

    ledger_ci = ObservabilityLedger(str(tmp / "ledger-ci.json"))
    task_ci = Task(task_id="BIG-203-ci", source="github", title="CI completion", description="")
    run_ci = TaskRun.from_task(task_ci, run_id="run-ci-1", medium="docker")
    run_ci.finalize("approved", "waiting for CI")
    ledger_ci.append(run_ci)
    updated_ci = EventBus(ledger=ledger_ci).publish(BusEvent(
        event_type=CI_COMPLETED_EVENT,
        run_id=run_ci.run_id,
        actor="github-actions",
        details={"workflow": "pytest", "conclusion": "success"},
    ))
    persisted_ci = ledger_ci.load()[0]

    ledger_fail = ObservabilityLedger(str(tmp / "ledger-fail.json"))
    task_fail = Task(task_id="BIG-203-fail", source="scheduler", title="Task failure", description="")
    run_fail = TaskRun.from_task(task_fail, run_id="run-fail-1", medium="docker")
    ledger_fail.append(run_fail)
    updated_fail = EventBus(ledger=ledger_fail).publish(BusEvent(
        event_type=TASK_FAILED_EVENT,
        run_id=run_fail.run_id,
        actor="worker",
        details={"error": "sandbox command exited 137"},
    ))
    persisted_fail = ledger_fail.load()[0]

    definition = WorkflowDefinition.from_json(
        '{'
        '"name": "release-closeout", '
        '"steps": [{"name": "execute", "kind": "scheduler"}], '
        '"report_path_template": "reports/{task_id}/{run_id}.md", '
        '"journal_path_template": "journals/{workflow}/{run_id}.json", '
        '"validation_evidence": ["pytest"], '
        '"approvals": ["ops-review"]'
        '}'
    )
    task_dsl = Task(task_id="BIG-401", source="linear", title="DSL", description="")

    engine_definition = WorkflowDefinition.from_dict(
        {
            "name": "acceptance-closeout",
            "steps": [{"name": "execute", "kind": "scheduler"}],
            "report_path_template": str(tmp / "reports" / "{task_id}" / "{run_id}.md"),
            "journal_path_template": str(tmp / "journals" / "{workflow}" / "{run_id}.json"),
            "validation_evidence": ["pytest", "report-shared"],
        }
    )
    engine_task = Task(
        task_id="BIG-401-flow",
        source="linear",
        title="Run workflow definition",
        description="dsl execution",
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )
    engine_result = WorkflowEngine().run_definition(
        engine_task,
        definition=engine_definition,
        run_id="run-dsl-1",
        ledger=ObservabilityLedger(str(tmp / "ledger-dsl.json")),
    )

    invalid_error = ""
    try:
        WorkflowEngine().run_definition(
            Task(task_id="BIG-401-invalid", source="local", title="invalid", description=""),
            definition=WorkflowDefinition.from_dict({"name": "broken-flow", "steps": [{"name": "hack", "kind": "unknown-kind"}]}),
            run_id="run-dsl-invalid",
            ledger=ObservabilityLedger(str(tmp / "ledger-dsl-invalid.json")),
        )
    except ValueError as exc:
        invalid_error = str(exc)

    approval_definition = WorkflowDefinition.from_dict(
        {
            "name": "prod-approval",
            "steps": [{"name": "review", "kind": "approval"}],
            "validation_evidence": ["rollback-plan", "integration-test"],
            "approvals": ["release-manager"],
        }
    )
    approval_task = Task(
        task_id="BIG-403-dsl",
        source="linear",
        title="Prod rollout",
        description="needs manual closure",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )
    approval_result = WorkflowEngine().run_definition(
        approval_task,
        definition=approval_definition,
        run_id="run-dsl-2",
        ledger=ObservabilityLedger(str(tmp / "ledger-dsl-approval.json")),
    )

    event_types = sorted(spec.event_type for spec in P0_AUDIT_EVENT_SPECS)
    scheduler_missing = missing_required_fields(
        SCHEDULER_DECISION_EVENT,
        {"task_id": "OPE-134", "run_id": "run-ope-134", "medium": "docker"},
    )

    spec_error = ""
    try:
        TaskRun.from_task(
            Task(task_id="OPE-134-spec", source="linear", title="Validate audit fields", description=""),
            run_id="run-ope-134-spec",
            medium="docker",
        ).audit_spec_event(
            MANUAL_TAKEOVER_EVENT,
            "scheduler",
            "pending",
            task_id="OPE-134-spec",
            run_id="run-ope-134-spec",
            target_team="security",
        )
    except ValueError as exc:
        spec_error = str(exc)

    scheduler_ledger = ObservabilityLedger(str(tmp / "ledger-audit.json"))
    scheduler_task = Task(
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
    scheduler_record = Scheduler().execute(scheduler_task, run_id="run-ope-134-scheduler", ledger=scheduler_ledger)
    scheduler_audits = {entry["action"]: entry for entry in scheduler_ledger.load()[0]["audits"]}

    workflow_ledger = ObservabilityLedger(str(tmp / "ledger-approval.json"))
    workflow_task = Task(
        task_id="OPE-134-approval",
        source="linear",
        title="Approve production rollout",
        description="Manual gate",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )
    WorkflowEngine().run(
        workflow_task,
        run_id="run-ope-134-approval",
        ledger=workflow_ledger,
        approvals=["security-review"],
        validation_evidence=["rollback-plan", "integration-test"],
    )
    workflow_audits = {entry["action"]: entry for entry in workflow_ledger.load()[0]["audits"]}

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
    queue_takeover = build_takeover_queue_from_ledger([entry], period="2026-03-11")

    print(json.dumps({
        "control_center": {
            "peek_order": peek_order,
            "queue_depth": center.queue_depth,
            "priority_p0": center.queued_by_priority["P0"],
            "priority_p1": center.queued_by_priority["P1"],
            "priority_p2": center.queued_by_priority["P2"],
            "risk_low": center.queued_by_risk["low"],
            "risk_medium": center.queued_by_risk["medium"],
            "risk_high": center.queued_by_risk["high"],
            "medium_vm": center.execution_media["vm"],
            "medium_browser": center.execution_media["browser"],
            "medium_docker": center.execution_media["docker"],
            "waiting_approval_runs": center.waiting_approval_runs,
            "blocked_tasks": center.blocked_tasks,
            "queued_tasks": center.queued_tasks,
            "blocked_action_ids": [action.action_id for action in center.actions["BIG-802-1"]],
            "escalate_enabled": center.actions["BIG-802-1"][3].enabled,
            "retry_enabled": center.actions["BIG-802-1"][4].enabled,
            "pause_enabled": center.actions["BIG-802-1"][5].enabled,
            "report_has_heading": "# Queue Control Center" in report,
            "report_has_waiting_approval": "- Waiting Approval Runs: 1" in report,
            "report_has_blocked_task": "- BIG-802-1" in report,
            "report_has_drill_down": "BIG-802-1: Drill Down [drill-down]" in report,
            "report_has_escalate": "Escalate [escalate] state=enabled" in report,
            "report_has_pause_reason": "Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused" in report,
            "empty_has_view_state": "## View State" in empty_report,
            "empty_has_empty_state": "- State: empty" in empty_report,
            "empty_has_summary": "- Summary: No queued work for the selected team." in empty_report,
            "empty_has_team_filter": "- Team: operations" in empty_report,
        },
        "event_bus": {
            "pr_status": updated_pr.status,
            "pr_summary": updated_pr.summary,
            "pr_seen_statuses": seen_statuses,
            "pr_has_comment_audit": any(audit["action"] == "collaboration.comment" for audit in persisted_pr["audits"]),
            "pr_has_transition_audit": any(audit["action"] == "event_bus.transition" and audit["details"]["previous_status"] == "needs-approval" for audit in persisted_pr["audits"]),
            "ci_status": updated_ci.status,
            "ci_summary": updated_ci.summary,
            "ci_has_event_audit": any(audit["action"] == "event_bus.event" and audit["details"]["event_type"] == CI_COMPLETED_EVENT for audit in persisted_ci["audits"]),
            "fail_status": updated_fail.status,
            "fail_summary": updated_fail.summary,
            "fail_has_transition_audit": any(audit["action"] == "event_bus.transition" and audit["details"]["status"] == "failed" for audit in persisted_fail["audits"]),
        },
        "dsl": {
            "first_step_name": definition.steps[0].name,
            "report_path": definition.render_report_path(task_dsl, "run-1"),
            "journal_path": definition.render_journal_path(task_dsl, "run-1"),
            "engine_acceptance_status": engine_result.acceptance.status,
            "engine_report_exists": Path(engine_definition.render_report_path(engine_task, "run-dsl-1")).exists(),
            "engine_journal_exists": Path(engine_definition.render_journal_path(engine_task, "run-dsl-1")).exists(),
            "invalid_error": invalid_error,
            "approval_run_status": approval_result.execution.run.status,
            "approval_acceptance_status": approval_result.acceptance.status,
            "approvals": approval_result.acceptance.approvals,
        },
        "audit_events": {
            "event_types": event_types,
            "missing_scheduler_fields": scheduler_missing,
            "spec_error": spec_error,
            "has_handoff_request": scheduler_record.handoff_request is not None,
            "scheduler_risk_score": scheduler_audits[SCHEDULER_DECISION_EVENT]["details"]["risk_score"],
            "budget_override_matches": scheduler_audits[BUDGET_OVERRIDE_EVENT]["details"] == {
                "task_id": "OPE-134-scheduler",
                "run_id": "run-ope-134-scheduler",
                "requested_budget": 120.0,
                "approved_budget": 150.0,
                "override_actor": "finance-controller",
                "reason": "approved additional analytics validation spend",
            },
            "manual_takeover_target": scheduler_audits[MANUAL_TAKEOVER_EVENT]["details"]["target_team"],
            "flow_handoff_source": scheduler_audits[FLOW_HANDOFF_EVENT]["details"]["source_stage"],
            "approval_recorded_matches": workflow_audits[APPROVAL_RECORDED_EVENT]["details"] == {
                "task_id": "OPE-134-approval",
                "run_id": "run-ope-134-approval",
                "approvals": ["security-review"],
                "approval_count": 1,
                "acceptance_status": "accepted",
            },
            "canvas_handoff_team": canvas.handoff_team,
            "queue_approvals": queue_takeover.requests[0].required_approvals,
        },
    }))
`

	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(repoRoot, "src"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run python repo workflow contracts: %v\n%s", err, output)
	}

	var payload pythonRepoWorkflowContractsPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode python repo workflow contracts payload: %v\n%s", err, output)
	}
	return payload
}
