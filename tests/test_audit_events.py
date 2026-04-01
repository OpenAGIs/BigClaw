from pathlib import Path

import pytest

from bigclaw.observability import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    ObservabilityLedger,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    TaskRun,
    missing_required_fields,
)
from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.reports import build_orchestration_canvas_from_ledger_entry, build_takeover_queue_from_ledger
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler
from bigclaw.workflow import WorkflowEngine


def test_p0_audit_event_specs_define_required_operational_events() -> None:
    event_types = {spec.event_type for spec in P0_AUDIT_EVENT_SPECS}

    assert event_types == {
        SCHEDULER_DECISION_EVENT,
        MANUAL_TAKEOVER_EVENT,
        APPROVAL_RECORDED_EVENT,
        BUDGET_OVERRIDE_EVENT,
        FLOW_HANDOFF_EVENT,
    }
    assert missing_required_fields(
        SCHEDULER_DECISION_EVENT,
        {
            "task_id": "OPE-134",
            "run_id": "run-ope-134",
            "medium": "docker",
        },
    ) == ["approved", "reason", "risk_level", "risk_score"]


def test_task_run_audit_spec_event_requires_required_fields() -> None:
    run = TaskRun.from_task(
        Task(task_id="OPE-134-spec", source="linear", title="Validate audit fields", description=""),
        run_id="run-ope-134-spec",
        medium="docker",
    )

    with pytest.raises(ValueError, match="missing required fields"):
        run.audit_spec_event(
            MANUAL_TAKEOVER_EVENT,
            "scheduler",
            "pending",
            task_id="OPE-134-spec",
            run_id="run-ope-134-spec",
            target_team="security",
        )


def test_scheduler_emits_p0_operational_audit_events(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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

    assert record.handoff_request is not None
    assert audits[SCHEDULER_DECISION_EVENT]["details"]["risk_score"] >= 0
    assert audits[BUDGET_OVERRIDE_EVENT]["details"] == {
        "task_id": "OPE-134-scheduler",
        "run_id": "run-ope-134-scheduler",
        "requested_budget": 120.0,
        "approved_budget": 150.0,
        "override_actor": "finance-controller",
        "reason": "approved additional analytics validation spend",
    }
    assert audits[MANUAL_TAKEOVER_EVENT]["details"]["target_team"] == "operations"
    assert audits[FLOW_HANDOFF_EVENT]["details"]["source_stage"] == "scheduler"


def test_workflow_records_canonical_approval_event(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
    assert audits[APPROVAL_RECORDED_EVENT]["details"] == {
        "task_id": "OPE-134-approval",
        "run_id": "run-ope-134-approval",
        "approvals": ["security-review"],
        "approval_count": 1,
        "acceptance_status": "accepted",
    }


def test_reports_accept_canonical_handoff_and_takeover_events() -> None:
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

    assert canvas.handoff_team == "security"
    assert queue.requests[0].required_approvals == ["security-review"]


def test_big301_worker_lifecycle_is_stable_with_multiple_tools() -> None:
    task = Task(
        task_id="BIG-301-matrix",
        source="github",
        title="worker lifecycle matrix",
        description="validate stable lifecycle",
        required_tools=["github", "browser"],
    )
    run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
    runtime = ToolRuntime(
        handlers={
            "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
            "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
        }
    )
    worker = ClawWorkerRuntime(tool_runtime=runtime)

    result = worker.execute(
        task,
        decision=type("Decision", (), {"medium": "docker", "approved": True, "reason": "ok"})(),
        run=run,
        tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}, "browser": {"url": "https://example.com"}},
    )

    assert len(result.tool_results) == 2
    assert all(item.success for item in result.tool_results)
    assert run.audits[-1].action == "worker.lifecycle"
    assert run.audits[-1].outcome == "completed"


def test_big302_risk_routes_to_expected_sandbox_mediums() -> None:
    scheduler = Scheduler()

    low = Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW)
    high = Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH)
    browser = Task(
        task_id="browser",
        source="local",
        title="browser",
        description="",
        required_tools=["browser"],
        risk_level=RiskLevel.MEDIUM,
    )

    assert scheduler.decide(low).medium == "docker"
    assert scheduler.decide(high).medium == "vm"
    assert scheduler.decide(browser).medium in {"browser", "docker"}


def test_big303_tool_runtime_policy_and_audit_chain() -> None:
    task = Task(
        task_id="BIG-303-matrix",
        source="local",
        title="tool policy",
        description="",
        required_tools=["github", "browser"],
    )
    run = TaskRun.from_task(task, run_id="run-big303-matrix", medium="docker")

    runtime = ToolRuntime(
        policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
        handlers={"github": lambda action, payload: "ok"},
    )

    allow = runtime.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=run)
    block = runtime.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert allow.success is True
    assert block.success is False
    outcomes = [audit.outcome for audit in run.audits if audit.action == "tool.invoke"]
    assert "success" in outcomes
    assert "blocked" in outcomes
