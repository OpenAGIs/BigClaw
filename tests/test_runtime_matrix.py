from pathlib import Path

from bigclaw.models import RiskLevel, Task, WorkflowDefinition
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler
from bigclaw.workflow import WorkflowEngine


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


def test_workflow_definition_parses_and_renders_templates():
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
    task = Task(task_id="BIG-401", source="linear", title="DSL", description="")

    assert definition.steps[0].name == "execute"
    assert definition.render_report_path(task, "run-1") == "reports/BIG-401/run-1.md"
    assert definition.render_journal_path(task, "run-1") == "journals/release-closeout/run-1.json"


def test_workflow_engine_runs_definition_end_to_end(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "acceptance-closeout",
            "steps": [{"name": "execute", "kind": "scheduler"}],
            "report_path_template": str(tmp_path / "reports" / "{task_id}" / "{run_id}.md"),
            "journal_path_template": str(tmp_path / "journals" / "{workflow}" / "{run_id}.json"),
            "validation_evidence": ["pytest", "report-shared"],
        }
    )
    task = Task(
        task_id="BIG-401-flow",
        source="linear",
        title="Run workflow definition",
        description="dsl execution",
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )

    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-1",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    assert result.acceptance.status == "accepted"
    assert Path(definition.render_report_path(task, "run-dsl-1")).exists()
    assert Path(definition.render_journal_path(task, "run-dsl-1")).exists()


def test_workflow_definition_rejects_unknown_step_kind(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "broken-flow",
            "steps": [{"name": "hack", "kind": "unknown-kind"}],
        }
    )
    task = Task(task_id="BIG-401-invalid", source="local", title="invalid", description="")

    try:
        WorkflowEngine().run_definition(
            task,
            definition=definition,
            run_id="run-dsl-invalid",
            ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
        )
        assert False, "expected ValueError for invalid step kind"
    except ValueError as exc:
        assert "invalid workflow step kind" in str(exc)


def test_workflow_definition_manual_approval_closes_high_risk_task(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "prod-approval",
            "steps": [{"name": "review", "kind": "approval"}],
            "validation_evidence": ["rollback-plan", "integration-test"],
            "approvals": ["release-manager"],
        }
    )
    task = Task(
        task_id="BIG-403-dsl",
        source="linear",
        title="Prod rollout",
        description="needs manual closure",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )

    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-2",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    assert result.execution.run.status == "needs-approval"
    assert result.acceptance.status == "accepted"
    assert result.acceptance.approvals == ["release-manager"]
