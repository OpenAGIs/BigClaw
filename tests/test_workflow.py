import json
from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.workflow import AcceptanceGate, WorkflowEngine


def test_acceptance_gate_rejects_missing_evidence(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-403",
        source="linear",
        title="Close acceptance gate",
        description="need validation evidence",
        priority=Priority.P0,
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )

    execution = WorkflowEngine().scheduler.execute(task, run_id="run-gate-1", ledger=ledger)
    decision = AcceptanceGate().evaluate(task, execution, validation_evidence=["pytest"])

    assert decision.passed is False
    assert decision.status == "rejected"
    assert decision.missing_acceptance_criteria == ["report-shared"]
    assert decision.missing_validation_steps == []


def test_workflow_engine_records_journal_and_accepts_complete_evidence(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-402",
        source="linear",
        title="Record workflow journal",
        description="capture execution closure",
        priority=Priority.P0,
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
        required_tools=["browser"],
    )

    result = WorkflowEngine().run(
        task,
        run_id="run-wf-1",
        ledger=ledger,
        report_path=str(tmp_path / "reports" / "run-wf-1.md"),
        journal_path=str(tmp_path / "journals" / "run-wf-1.json"),
        validation_evidence=["pytest", "report-shared"],
    )

    assert result.execution.decision.medium == "browser"
    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.journal_path is not None

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "acceptance"]
    assert journal["entries"][-1]["status"] == "accepted"


def test_workflow_engine_keeps_high_risk_task_pending_manual_approval(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-403-risk",
        source="linear",
        title="Approve prod change",
        description="manual gate",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )

    result = WorkflowEngine().run(
        task,
        run_id="run-wf-2",
        ledger=ledger,
        validation_evidence=["rollback-plan", "integration-test"],
    )

    assert result.execution.run.status == "needs-approval"
    assert result.acceptance.passed is False
    assert result.acceptance.status == "needs-approval"
