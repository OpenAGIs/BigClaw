from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    PremiumOrchestrationPolicy,
    render_orchestration_canvas,
    render_orchestration_plan,
)
from bigclaw.scheduler import Scheduler


def test_cross_department_orchestrator_routes_security_data_and_customer_work() -> None:
    task = Task(
        task_id="OPE-66",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer", "premium"],
        priority=Priority.P0,
        risk_level=RiskLevel.HIGH,
        required_tools=["browser", "sql"],
        acceptance_criteria=["approval recorded"],
        validation_plan=["customer signoff"],
    )

    plan = CrossDepartmentOrchestrator().plan(task)

    assert plan.collaboration_mode == "cross-functional"
    assert plan.departments == ["operations", "engineering", "security", "data", "customer-success"]
    assert plan.required_approvals == ["security-review"]


def test_standard_policy_limits_advanced_cross_department_routing() -> None:
    task = Task(
        task_id="OPE-66-standard",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer"],
        required_tools=["browser", "sql"],
        risk_level=RiskLevel.HIGH,
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)

    assert plan.collaboration_mode == "tier-limited"
    assert plan.departments == ["operations", "engineering"]
    assert policy.upgrade_required is True
    assert policy.blocked_departments == ["security", "data", "customer-success"]


def test_render_orchestration_plan_lists_handoffs_and_policy() -> None:
    task = Task(
        task_id="OPE-66-render",
        source="jira",
        title="Warehouse rollout",
        description="Customer-ready release",
        labels=["data", "customer"],
        required_tools=["sql"],
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
    content = render_orchestration_plan(plan, policy)

    assert "# Cross-Department Orchestration Plan" in content
    assert "- Departments: operations, engineering" in content
    assert "- Tier: standard" in content
    assert "- Blocked Departments: data, customer-success" in content
    assert "- Human Handoff Team:" not in content

def test_render_orchestration_canvas_highlights_delivery_lanes() -> None:
    task = Task(
        task_id="BIG-1001",
        source="linear",
        title="Launch orchestration canvas",
        description="Coordinate rollout with customer and analytics workstreams",
        labels=["customer", "data", "ops"],
        required_tools=["browser", "sql"],
        acceptance_criteria=["canvas-reviewed"],
        validation_plan=["pytest", "ux-walkthrough"],
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
    page = render_orchestration_canvas(task, plan, policy)

    assert "Orchestration Canvas" in page
    assert "Launch orchestration canvas" in page
    assert "Upgrade Required: True" in page
    assert "operations" in page
    assert "canvas-reviewed" in page
    assert "ux-walkthrough" in page



def test_scheduler_execution_records_orchestration_plan_and_policy(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-exec",
        source="linear",
        title="Cross-team browser change",
        description="Program-managed rollout",
        labels=["ops"],
        priority=Priority.P0,
        risk_level=RiskLevel.MEDIUM,
        required_tools=["browser"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66", ledger=ledger)
    entry = ledger.load()[0]

    assert record.orchestration_plan is not None
    assert record.orchestration_policy is not None
    assert record.orchestration_plan.departments == ["operations", "engineering"]
    assert record.orchestration_policy.upgrade_required is False
    assert any(trace["span"] == "orchestration.plan" for trace in entry["traces"])
    assert any(trace["span"] == "orchestration.policy" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.plan" for audit in entry["audits"])
    assert any(audit["action"] == "orchestration.policy" for audit in entry["audits"])


def test_scheduler_creates_handoff_for_policy_or_approval_blockers(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-handoff",
        source="linear",
        title="Customer analytics rollout",
        description="Need cross-team coordination",
        labels=["customer", "data"],
        required_tools=["browser", "sql"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66-handoff", ledger=ledger)
    entry = ledger.load()[0]

    assert record.handoff_request is not None
    assert record.handoff_request.target_team == "operations"
    assert any(trace["span"] == "orchestration.handoff" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.handoff" for audit in entry["audits"])
