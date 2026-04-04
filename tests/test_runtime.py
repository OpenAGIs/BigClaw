import warnings
from pathlib import Path

from bigclaw import deprecation, orchestration, queue, workflow
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw import runtime, scheduler
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


def test_big301_worker_lifecycle_is_stable_with_multiple_tools() -> None:
    task = Task(
        task_id="BIG-301-matrix",
        source="github",
        title="worker lifecycle matrix",
        description="validate stable lifecycle",
        required_tools=["github", "browser"],
    )
    run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
    runtime_matrix = ToolRuntime(
        handlers={
            "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
            "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
        }
    )
    worker = ClawWorkerRuntime(tool_runtime=runtime_matrix)

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


def test_warn_legacy_runtime_surface_emits_deprecation_warning():
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        message = deprecation.warn_legacy_runtime_surface(
            "python -m bigclaw",
            "bash scripts/ops/bigclawctl",
        )

    assert "frozen for migration-only use" in message
    assert caught
    assert issubclass(caught[0].category, DeprecationWarning)
    assert "bash scripts/ops/bigclawctl" in str(caught[0].message)


def test_legacy_runtime_modules_expose_go_mainline_replacements():
    assert runtime.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/worker/runtime.go"
    assert scheduler.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/scheduler/scheduler.go"
    assert workflow.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/engine.go"
    assert orchestration.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/orchestration.go"
    assert queue.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/queue/queue.go"
    assert "sole implementation mainline" in runtime.LEGACY_MAINLINE_STATUS


def test_big302_risk_routes_to_expected_sandbox_mediums() -> None:
    scheduler_runtime = Scheduler()

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

    assert scheduler_runtime.decide(low).medium == "docker"
    assert scheduler_runtime.decide(high).medium == "vm"
    assert scheduler_runtime.decide(browser).medium in {"browser", "docker"}


def test_big303_tool_runtime_policy_and_audit_chain() -> None:
    task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
    run = TaskRun.from_task(task, run_id="run-big303-matrix", medium="docker")

    runtime_matrix = ToolRuntime(
        policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
        handlers={"github": lambda action, payload: "ok"},
    )

    allow = runtime_matrix.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=run)
    block = runtime_matrix.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert allow.success is True
    assert block.success is False
    outcomes = [audit.outcome for audit in run.audits if audit.action == "tool.invoke"]
    assert "success" in outcomes
    assert "blocked" in outcomes
