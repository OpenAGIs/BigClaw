import importlib
from pathlib import Path

from bigclaw.models import RiskLevel, Task
from bigclaw.observability import TaskRun
from bigclaw import runtime as legacy_runtime
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
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


def test_big_go_1003_runtime_batch_inventory_is_single_file_residual():
    inventory = legacy_runtime.legacy_runtime_batch_inventory()

    assert [item["logical_surface"] for item in inventory] == [
        "runtime",
        "queue",
        "orchestration",
        "scheduler",
        "workflow",
        "service",
    ]
    assert {item["physical_python_file"] for item in inventory} == {"src/bigclaw/runtime.py"}
    assert Path("src/bigclaw/runtime.py").exists()
    assert inventory[0]["status"] == "kept"
    assert all(item["go_replacement"].startswith("bigclaw-go/") for item in inventory)


def test_big_go_1003_legacy_surface_modules_reexport_runtime_objects():
    queue = importlib.import_module("bigclaw.queue")
    orchestration = importlib.import_module("bigclaw.orchestration")
    scheduler = importlib.import_module("bigclaw.scheduler")
    workflow = importlib.import_module("bigclaw.workflow")
    service = importlib.import_module("bigclaw.service")

    assert queue.PersistentTaskQueue is legacy_runtime.PersistentTaskQueue
    assert orchestration.CrossDepartmentOrchestrator is legacy_runtime.CrossDepartmentOrchestrator
    assert scheduler.Scheduler is legacy_runtime.Scheduler
    assert workflow.WorkflowEngine is legacy_runtime.WorkflowEngine
    assert service.create_server is legacy_runtime.create_server
