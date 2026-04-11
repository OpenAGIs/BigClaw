from pathlib import Path

from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.runtime import ClawWorkerRuntime, SandboxRouter, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler, SchedulerDecision


def test_sandbox_router_maps_execution_media():
    router = SandboxRouter()

    assert router.profile_for("docker").isolation == "container"
    assert router.profile_for("browser").network_access == "enabled"
    assert router.profile_for("vm").filesystem_access == "workspace-write"
    assert router.profile_for("unknown").medium == "none"


def test_tool_runtime_blocks_disallowed_tool_and_audits(tmp_path: Path):
    task = Task(task_id="BIG-303", source="linear", title="tool policy", description="")
    run = TaskRun.from_task(task, run_id="run-tool-1", medium="docker")
    runtime = ToolRuntime(policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]))

    result = runtime.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert result.success is False
    assert result.error == "tool blocked by policy"
    assert run.audits[-1].action == "tool.invoke"
    assert run.audits[-1].outcome == "blocked"


def test_worker_runtime_returns_tool_results_for_approved_task():
    task = Task(
        task_id="BIG-301",
        source="linear",
        title="worker runtime",
        description="return tool results",
        required_tools=["github"],
    )
    run = TaskRun.from_task(task, run_id="run-worker-1", medium="docker")
    tool_runtime = ToolRuntime(handlers={"github": lambda action, payload: f"{action}:{payload['repo']}"})
    worker = ClawWorkerRuntime(tool_runtime=tool_runtime)

    result = worker.execute(
        task,
        SchedulerDecision(medium="docker", approved=True, reason="default low risk path"),
        run,
        tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}},
    )

    assert result.sandbox_profile.medium == "docker"
    assert result.tool_results[0].success is True
    assert result.tool_results[0].output == "execute:OpenAGIs/BigClaw"
    assert run.audits[-1].action == "worker.lifecycle"
    assert run.audits[-1].outcome == "completed"


def test_scheduler_records_worker_runtime_results_and_waits_on_high_risk(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-302",
        source="jira",
        title="prod sandbox",
        description="needs approval",
        risk_level=RiskLevel.HIGH,
        required_tools=["browser"],
    )

    record = Scheduler().execute(task, run_id="run-worker-2", ledger=ledger)
    entry = ledger.load()[0]

    assert record.decision.medium == "vm"
    assert record.run.status == "needs-approval"
    assert record.tool_results == []
    assert entry["audits"][-1]["action"] == "worker.lifecycle"
    assert entry["audits"][-1]["outcome"] == "waiting-approval"


def test_scheduler_pauses_execution_when_budget_cannot_cover_docker(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-202-budget",
        source="linear",
        title="budget pause",
        description="budget should stop execution",
        budget=5.0,
        required_tools=["github"],
    )

    record = Scheduler().execute(task, run_id="run-budget-pause", ledger=ledger)
    entry = ledger.load()[0]

    assert record.decision.medium == "none"
    assert record.decision.approved is False
    assert record.run.status == "paused"
    assert record.tool_results == []
    assert entry["audits"][-1]["action"] == "worker.lifecycle"
    assert entry["audits"][-1]["outcome"] == "paused"


def test_scheduler_high_risk_requires_approval():
    scheduler = Scheduler()
    task = Task(task_id="sched-high", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)

    decision = scheduler.decide(task)

    assert decision.medium == "vm"
    assert decision.approved is False


def test_scheduler_browser_task_routes_browser():
    scheduler = Scheduler()
    task = Task(task_id="sched-browser", source="github", title="ui test", description="", required_tools=["browser"])

    decision = scheduler.decide(task)

    assert decision.medium == "browser"
    assert decision.approved is True


def test_scheduler_over_budget_degrades_browser_task_to_docker():
    scheduler = Scheduler()
    task = Task(
        task_id="sched-budget-browser",
        source="github",
        title="budgeted ui test",
        description="",
        required_tools=["browser"],
        budget=15.0,
    )

    decision = scheduler.decide(task)

    assert decision.medium == "docker"
    assert decision.approved is True
    assert "budget degraded browser route to docker" in decision.reason


def test_scheduler_over_budget_pauses_task():
    scheduler = Scheduler()
    task = Task(task_id="sched-budget-pause", source="linear", title="tiny budget", description="", budget=5.0)

    decision = scheduler.decide(task)

    assert decision.medium == "none"
    assert decision.approved is False
    assert decision.reason == "paused: budget 5.0 below required docker budget 10.0"
