from pathlib import Path

from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.workflow import WorkflowDefinition, WorkflowEngine


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
