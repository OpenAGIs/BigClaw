import json
import threading
import urllib.request
import warnings
from pathlib import Path

from bigclaw import deprecation, orchestration, queue, runtime, scheduler, service, workflow
from bigclaw.memory import TaskMemoryStore
from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.cost_control import CostController
from bigclaw.risk import RiskScorer
from bigclaw.runtime import ClawWorkerRuntime, SandboxRouter, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler, SchedulerDecision
from bigclaw.service import RepoGovernanceEnforcer, RepoGovernancePolicy, create_server


def _get(url: str) -> str:
    with urllib.request.urlopen(url, timeout=3) as resp:
        return resp.read().decode("utf-8")


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


def test_task_memory_store_reuses_history_and_injects_rules(tmp_path: Path) -> None:
    store = TaskMemoryStore(str(tmp_path / "memory" / "task-patterns.json"))

    previous = Task(
        task_id="BIG-501-prev",
        source="github",
        title="Previous queue rollout",
        description="",
        labels=["queue", "platform"],
        required_tools=["github", "browser"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest", "smoke-test"],
    )
    store.remember_success(previous, summary="queue migration done")

    current = Task(
        task_id="BIG-501-new",
        source="github",
        title="New queue hardening",
        description="",
        labels=["queue"],
        required_tools=["github"],
        acceptance_criteria=["unit-tests"],
        validation_plan=["pytest"],
    )

    suggestion = store.suggest_rules(current)

    assert "BIG-501-prev" in suggestion["matched_task_ids"]
    assert "report-shared" in suggestion["acceptance_criteria"]
    assert "smoke-test" in suggestion["validation_plan"]
    assert "unit-tests" in suggestion["acceptance_criteria"]


def test_repo_governance_enforcer_blocks_quota_and_sidecar_failures():
    enforcer = RepoGovernanceEnforcer(
        RepoGovernancePolicy(max_bundle_bytes=10, max_push_per_hour=1, max_diff_per_hour=1, sidecar_required=True)
    )

    ok = enforcer.evaluate(action="push", bundle_bytes=8, sidecar_available=True)
    assert ok.allowed is True

    too_large = enforcer.evaluate(action="push", bundle_bytes=12, sidecar_available=True)
    assert too_large.allowed is False
    assert too_large.mode == "blocked"

    over_quota = enforcer.evaluate(action="push", bundle_bytes=8, sidecar_available=True)
    assert over_quota.allowed is False
    assert "quota" in over_quota.reason

    degraded = enforcer.evaluate(action="diff", sidecar_available=False)
    assert degraded.allowed is False
    assert degraded.mode == "degraded"


def test_server_entry_health_metrics(tmp_path: Path):
    (tmp_path / "index.html").write_text("<h1>ok</h1>", encoding="utf-8")

    server, monitoring = create_server(host="127.0.0.1", port=0, directory=str(tmp_path))
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        host, port = server.server_address
        body = _get(f"http://{host}:{port}/")
        assert "ok" in body

        health = json.loads(_get(f"http://{host}:{port}/health"))
        assert health["status"] == "ok"
        assert health["request_total"] >= 1

        metrics = _get(f"http://{host}:{port}/metrics")
        assert "bigclaw_http_requests_total" in metrics
        assert "bigclaw_uptime_seconds" in metrics

        metrics_json = json.loads(_get(f"http://{host}:{port}/metrics.json"))
        assert "bigclaw_http_requests_total" in metrics_json
        assert "recent_requests" in metrics_json
        assert "rolling_5m" in metrics_json
        assert "bigclaw_http_error_rate" in metrics_json
        assert "health_summary" in metrics_json

        alerts = json.loads(_get(f"http://{host}:{port}/alerts"))
        assert alerts["level"] in {"ok", "warn", "critical"}
        assert "error_rate" in alerts

        monitor_html = _get(f"http://{host}:{port}/monitor")
        assert "BigClaw Monitor" in monitor_html
        assert "Requests" in monitor_html
        assert "Error Rate" in monitor_html
        assert "Auto refresh every 5s" in monitor_html

        assert monitoring.request_total >= 6
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)


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


def test_legacy_service_surface_emits_go_first_warning():
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        message = service.warn_legacy_service_surface()

    assert "go run ./bigclaw-go/cmd/bigclawd" in message
    assert caught
    assert issubclass(caught[0].category, DeprecationWarning)


def test_service_module_exposes_go_mainline_replacement():
    assert service.GO_MAINLINE_REPLACEMENT == "bigclaw-go/cmd/bigclawd/main.go"
    assert "sole implementation mainline" in service.LEGACY_MAINLINE_STATUS


def test_risk_scorer_keeps_simple_low_risk_work_low() -> None:
    score = RiskScorer().score_task(
        Task(task_id="BIG-902-low", source="linear", title="doc cleanup", description="")
    )

    assert score.total == 0
    assert score.level == RiskLevel.LOW
    assert score.requires_approval is False


def test_risk_scorer_elevates_prod_browser_work() -> None:
    score = RiskScorer().score_task(
        Task(
            task_id="BIG-902-mid",
            source="linear",
            title="release verification",
            description="prod browser change",
            labels=["prod"],
            priority=Priority.P0,
            required_tools=["browser"],
        )
    )

    assert score.total == 40
    assert score.level == RiskLevel.MEDIUM
    assert score.requires_approval is False


def test_scheduler_uses_risk_score_to_require_approval(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-902-high",
        source="linear",
        title="security deploy",
        description="prod deploy",
        labels=["security", "prod"],
        priority=Priority.P0,
        required_tools=["deploy"],
    )

    record = Scheduler().execute(task, run_id="run-risk", ledger=ledger)
    entry = ledger.load()[0]

    assert record.risk_score is not None
    assert record.risk_score.total == 70
    assert record.risk_score.level == RiskLevel.HIGH
    assert record.decision.medium == "vm"
    assert record.decision.approved is False
    assert any(trace["span"] == "risk.score" for trace in entry["traces"])
    assert any(audit["action"] == "risk.score" for audit in entry["audits"])


def test_big301_worker_lifecycle_is_stable_with_multiple_tools():
    task = Task(
        task_id="BIG-301-matrix",
        source="github",
        title="worker lifecycle matrix",
        description="validate stable lifecycle",
        required_tools=["github", "browser"],
    )
    run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
    runtime_tool = ToolRuntime(
        handlers={
            "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
            "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
        }
    )
    worker = ClawWorkerRuntime(tool_runtime=runtime_tool)

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
    scheduler_obj = Scheduler()

    low = Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW)
    high = Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH)
    browser_task = Task(
        task_id="browser", source="local", title="browser", description="", required_tools=["browser"], risk_level=RiskLevel.MEDIUM
    )

    assert scheduler_obj.decide(low).medium == "docker"
    assert scheduler_obj.decide(high).medium == "vm"
    assert scheduler_obj.decide(browser_task).medium in {"browser", "docker"}


def test_big303_tool_runtime_policy_and_audit_chain():
    task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
    run = TaskRun.from_task(task, run_id="run-big303-matrix", medium="docker")

    runtime_tool = ToolRuntime(
        policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
        handlers={"github": lambda action, payload: "ok"},
    )

    allow = runtime_tool.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=run)
    block = runtime_tool.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert allow.success is True
    assert block.success is False
    outcomes = [audit.outcome for audit in run.audits if audit.action == "tool.invoke"]
    assert "success" in outcomes
    assert "blocked" in outcomes


def test_big503_cost_controller_degrades_when_high_medium_over_budget():
    controller = CostController()
    task = Task(task_id="BIG-503-1", source="local", title="budget", description="", budget=0.7)

    decision = controller.evaluate(task, medium="browser", duration_minutes=15)

    assert decision.status == "degrade"
    assert "degrade to docker" in decision.reason
    assert decision.remaining_budget >= 0


def test_big503_cost_controller_pauses_when_even_docker_exceeds_budget():
    controller = CostController()
    task = Task(task_id="BIG-503-2", source="local", title="budget", description="", budget=0.2)

    decision = controller.evaluate(task, medium="browser", duration_minutes=30)

    assert decision.status == "pause"
    assert decision.reason == "budget exceeded"


def test_big503_cost_controller_respects_budget_override_amount():
    controller = CostController()
    task = Task(
        task_id="BIG-503-3",
        source="local",
        title="budget",
        description="",
        budget=0.2,
        budget_override_amount=0.6,
    )

    decision = controller.evaluate(task, medium="browser", duration_minutes=10)

    assert decision.status in {"allow", "degrade"}
