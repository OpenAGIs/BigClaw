from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    LifecycleFanoutOperation,
    LifecycleFanoutPlan,
    LifecycleFanoutTarget,
    PremiumOrchestrationPolicy,
    render_lifecycle_fanout_plan,
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
    assert policy.entitlement_status == "upgrade-required"
    assert policy.billing_model == "standard-blocked"
    assert policy.included_usage_units == 2
    assert policy.overage_usage_units == 3
    assert policy.overage_cost_usd == 12.0
    assert policy.estimated_cost_usd == 15.0
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
    assert "- Entitlement Status: upgrade-required" in content
    assert "- Billing Model: standard-blocked" in content
    assert "- Estimated Cost (USD): 11.00" in content
    assert "- Blocked Departments: data, customer-success" in content
    assert "- Human Handoff Team:" not in content


def test_render_lifecycle_fanout_plan_summarizes_batch_operations() -> None:
    plan = LifecycleFanoutPlan(
        name="BIGCLAW-176 lifecycle batch",
        requested_by="operations-control",
        operations=[
            LifecycleFanoutOperation(
                action="start",
                reason="resume queued bots after approval window opens",
                concurrency=2,
                targets=[
                    LifecycleFanoutTarget(run_id="run-start-a", task_id="BOT-101", current_status="stopped"),
                    LifecycleFanoutTarget(run_id="run-start-b", task_id="BOT-102", current_status="stopped"),
                ],
            ),
            LifecycleFanoutOperation(
                action="restart",
                reason="clear degraded browser workers",
                concurrency=1,
                takeover_queue="manual-takeovers",
                targets=[
                    LifecycleFanoutTarget(
                        run_id="run-restart-a",
                        task_id="BOT-201",
                        current_status="degraded",
                        owner="security",
                        blocked=True,
                        note="waiting for manual takeover slot",
                    )
                ],
            ),
            LifecycleFanoutOperation(
                action="upgrade",
                reason="roll forward to lifecycle v2",
                concurrency=1,
                targets=[LifecycleFanoutTarget(run_id="run-upgrade-a", task_id="BOT-301", current_status="running")],
            ),
            LifecycleFanoutOperation(
                action="stop",
                reason="drain unhealthy shards",
                concurrency=1,
                targets=[LifecycleFanoutTarget(run_id="run-stop-a", task_id="BOT-401", current_status="running")],
            ),
        ],
    )

    content = render_lifecycle_fanout_plan(plan)

    assert "# Lifecycle Fanout Plan" in content
    assert "- Name: BIGCLAW-176 lifecycle batch" in content
    assert "- Requested By: operations-control" in content
    assert "- Operations: 4" in content
    assert "- Target Count: 5" in content
    assert "- Blocked Targets: 1" in content
    assert "- Action Mix: restart=1 start=2 stop=1 upgrade=1" in content
    assert "- start: targets=2 blocked=0 concurrency=2 takeover_queue=none reason=resume queued bots after approval window opens" in content
    assert "run-restart-a: task=BOT-201 status=degraded owner=security blocked=True note=waiting for manual takeover slot" in content


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
    assert record.orchestration_policy.entitlement_status == "included"
    assert record.orchestration_policy.billing_model == "standard-included"
    assert record.orchestration_policy.estimated_cost_usd == 3.0
    assert any(trace["span"] == "orchestration.plan" for trace in entry["traces"])
    assert any(trace["span"] == "orchestration.policy" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.plan" for audit in entry["audits"])
    assert any(audit["action"] == "orchestration.policy" for audit in entry["audits"])
    policy_audit = next(audit for audit in entry["audits"] if audit["action"] == "orchestration.policy")
    assert policy_audit["details"]["entitlement_status"] == "included"
    assert policy_audit["details"]["billing_model"] == "standard-included"


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
