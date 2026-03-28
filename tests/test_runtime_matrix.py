from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.runtime import ClawWorkerRuntime, SandboxRouter, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler


def test_big301_worker_lifecycle_is_stable_with_multiple_tools():
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


def test_runtime_sandbox_router_profiles_are_stable():
    router = SandboxRouter()

    assert router.profile_for("docker").isolation == "container"
    assert router.profile_for("browser").network_access == "enabled"
    assert router.profile_for("vm").filesystem_access == "workspace-write"
    assert router.profile_for("unknown").medium == "none"


def test_big302_risk_routes_to_expected_sandbox_mediums():
    scheduler = Scheduler()

    low = Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW)
    high = Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH)
    browser = Task(
        task_id="browser", source="local", title="browser", description="", required_tools=["browser"], risk_level=RiskLevel.MEDIUM
    )

    assert scheduler.decide(low).medium == "docker"
    assert scheduler.decide(high).medium == "vm"
    assert scheduler.decide(browser).medium in {"browser", "docker"}


def test_big302_scheduler_execution_records_pending_and_budget_paused_states(tmp_path):
    scheduler = Scheduler()

    high_risk = Task(
        task_id="high-risk",
        source="jira",
        title="prod sandbox",
        description="needs approval",
        risk_level=RiskLevel.HIGH,
        required_tools=["browser"],
    )
    high_risk_record = scheduler.execute(
        high_risk,
        run_id="run-high-risk",
        ledger=ObservabilityLedger(str(tmp_path / "high-risk-ledger.json")),
    )
    assert high_risk_record.decision.medium == "vm"
    assert high_risk_record.run.status == "needs-approval"
    assert high_risk_record.tool_results == []

    budget_paused = Task(
        task_id="budget-pause",
        source="linear",
        title="budget pause",
        description="budget should stop execution",
        budget=5.0,
        required_tools=["github"],
    )
    budget_record = scheduler.execute(
        budget_paused,
        run_id="run-budget-pause",
        ledger=ObservabilityLedger(str(tmp_path / "budget-ledger.json")),
    )
    assert budget_record.decision.medium == "none"
    assert budget_record.decision.approved is False
    assert budget_record.run.status == "paused"
    assert budget_record.tool_results == []


def test_big303_tool_runtime_policy_and_audit_chain():
    task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
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
