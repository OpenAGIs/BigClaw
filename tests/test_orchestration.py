from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.orchestration import CrossDepartmentOrchestrator, render_orchestration_plan
from bigclaw.scheduler import Scheduler


def test_cross_department_orchestrator_routes_security_data_and_customer_work() -> None:
    task = Task(
        task_id="OPE-66",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer"],
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


def test_render_orchestration_plan_lists_handoffs_and_approvals() -> None:
    task = Task(
        task_id="OPE-66-render",
        source="jira",
        title="Warehouse rollout",
        description="Customer-ready release",
        labels=["data", "customer"],
        required_tools=["sql"],
    )

    content = render_orchestration_plan(CrossDepartmentOrchestrator().plan(task))

    assert "# Cross-Department Orchestration Plan" in content
    assert "- Departments: operations, engineering, data, customer-success" in content
    assert "- data: reason=validates analytics, warehouse, or measurement dependencies" in content


def test_scheduler_execution_records_orchestration_plan(tmp_path: Path) -> None:
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
    assert record.orchestration_plan.departments == ["operations", "engineering"]
    assert any(trace["span"] == "orchestration.plan" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.plan" for audit in entry["audits"])
