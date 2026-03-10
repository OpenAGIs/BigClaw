import json
from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.reports import PilotMetric, PilotScorecard
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
        orchestration_report_path=str(tmp_path / "reports" / "run-wf-1-orchestration.md"),
    )

    assert result.execution.decision.medium == "browser"
    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.journal_path is not None
    assert result.orchestration_report_path is not None

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "orchestration", "acceptance"]
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


def test_workflow_engine_writes_pilot_scorecard_and_accepts_positive_roi(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-60",
        source="linear",
        title="Pilot closeout",
        description="capture KPI and ROI evidence",
        priority=Priority.P0,
        acceptance_criteria=["pilot-scorecard", "report-shared"],
        validation_plan=["pytest"],
    )
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner A",
        period="2026-Q2",
        metrics=[
            PilotMetric(name="Automation coverage", baseline=30, current=81, target=80, unit="%"),
            PilotMetric(name="Review cycle time", baseline=10, current=4, target=5, unit="h", higher_is_better=False),
        ],
        monthly_benefit=15000,
        monthly_cost=3000,
        implementation_cost=18000,
        benchmark_score=98,
        benchmark_passed=True,
    )

    result = WorkflowEngine().run(
        task,
        run_id="run-wf-pilot-1",
        ledger=ledger,
        journal_path=str(tmp_path / "journals" / "run-wf-pilot-1.json"),
        validation_evidence=["pytest", "report-shared", "pilot-scorecard"],
        pilot_scorecard=scorecard,
        pilot_report_path=str(tmp_path / "reports" / "pilot-scorecard.md"),
    )

    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.pilot_report_path is not None
    assert Path(result.pilot_report_path).exists()

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "pilot-scorecard", "acceptance"]
    assert journal["entries"][2]["status"] == "go"
    assert "Annualized ROI" in Path(result.pilot_report_path).read_text()


def test_acceptance_gate_rejects_hold_pilot_scorecard(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-60-hold",
        source="linear",
        title="Pilot hold decision",
        description="scorecard blocks closure",
        acceptance_criteria=["pilot-scorecard"],
        validation_plan=["pytest"],
    )
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner B",
        period="2026-Q2",
        metrics=[PilotMetric(name="Backlog aging", baseline=4, current=6, target=4, unit="d", higher_is_better=False)],
        monthly_benefit=1000,
        monthly_cost=2500,
        implementation_cost=8000,
        benchmark_passed=False,
    )

    execution = WorkflowEngine().scheduler.execute(task, run_id="run-gate-pilot-1", ledger=ledger)
    decision = AcceptanceGate().evaluate(
        task,
        execution,
        validation_evidence=["pytest", "pilot-scorecard"],
        pilot_scorecard=scorecard,
    )

    assert decision.passed is False
    assert decision.status == "rejected"
    assert decision.summary == "pilot scorecard indicates insufficient ROI or KPI progress"


def test_workflow_engine_writes_orchestration_report_without_duplicating_ledger_entries(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-workflow",
        source="linear",
        title="Coordinate customer rollout",
        description="Need browser and analytics support",
        labels=["customer", "data"],
        priority=Priority.P0,
        required_tools=["browser", "sql"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )

    result = WorkflowEngine().run(
        task,
        run_id="run-wf-ope-66",
        ledger=ledger,
        journal_path=str(tmp_path / "journals" / "run-wf-ope-66.json"),
        orchestration_report_path=str(tmp_path / "reports" / "run-wf-ope-66-orchestration.md"),
        validation_evidence=["pytest", "report-shared"],
    )

    assert result.orchestration_report_path is not None
    assert Path(result.orchestration_report_path).exists()
    report = Path(result.orchestration_report_path).read_text()
    assert "- customer-success:" not in report
    assert "Upgrade Required: True" in report

    entries = ledger.load()
    assert len(entries) == 1
    assert entries[0]["artifacts"][0]["name"] == "cross-department-orchestration"

    journal = json.loads(Path(result.journal_path).read_text())
    assert journal["entries"][2]["step"] == "orchestration"
