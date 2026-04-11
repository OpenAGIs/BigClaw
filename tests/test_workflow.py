import json
from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import APPROVAL_RECORDED_EVENT, GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit
from bigclaw.reports import PilotMetric, PilotScorecard
from bigclaw.workflow import AcceptanceGate, WorkflowEngine, WorkpadJournal


def test_workpad_journal_can_replay_and_reload(tmp_path: Path):
    journal = WorkpadJournal(task_id="BIG-402-replay", run_id="run-journal-1")
    journal.record("intake", "recorded", source="local")
    journal.record("execution", "approved", medium="docker")
    output = journal.write(str(tmp_path / "journals" / "run-journal-1.json"))

    loaded = WorkpadJournal.read(output)

    assert loaded.task_id == "BIG-402-replay"
    assert loaded.run_id == "run-journal-1"
    assert loaded.replay() == ["intake:recorded", "execution:approved"]


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
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 123abc\n 3 files changed, 12 insertions(+)",
    )

    assert result.execution.decision.medium == "browser"
    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.journal_path is not None
    assert result.orchestration_report_path is not None

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "orchestration", "acceptance", "closeout"]
    assert journal["entries"][-2]["status"] == "accepted"
    assert journal["entries"][-1]["status"] == "complete"
    assert ledger.load()[0]["closeout"]["git_push_succeeded"] is True


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
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}
    assert APPROVAL_RECORDED_EVENT not in audits
    assert audits["execution.manual_takeover"]["details"]["target_team"] == "security"


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
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 456def\n 2 files changed, 9 insertions(+)",
    )

    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.pilot_report_path is not None
    assert Path(result.pilot_report_path).exists()

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "pilot-scorecard", "acceptance", "closeout"]
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
        orchestration_canvas_path=str(tmp_path / "reports" / "run-wf-ope-66-canvas.md"),
        validation_evidence=["pytest", "report-shared"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 789fed\n 4 files changed, 16 insertions(+)",
    )

    assert result.orchestration_report_path is not None
    assert Path(result.orchestration_report_path).exists()
    assert result.orchestration_canvas_path is not None
    assert Path(result.orchestration_canvas_path).exists()
    report = Path(result.orchestration_report_path).read_text()
    assert "- customer-success:" not in report
    assert "Upgrade Required: True" in report
    assert "Human Handoff Team: operations" in report
    canvas = Path(result.orchestration_canvas_path).read_text()
    assert "# Orchestration Canvas" in canvas
    assert "Recommendation: resolve-entitlement-gap" in canvas

    entries = ledger.load()
    assert len(entries) == 1
    assert entries[0]["artifacts"][0]["name"] == "cross-department-orchestration"
    assert entries[0]["artifacts"][1]["name"] == "orchestration-canvas"

    journal = json.loads(Path(result.journal_path).read_text())
    assert journal["entries"][2]["step"] == "orchestration"
    assert journal["entries"][-1]["step"] == "closeout"


def test_workflow_engine_writes_repo_sync_audit_report_and_records_failure_categories(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-219",
        source="linear",
        title="Audit repo sync",
        description="capture sync failures and pr freshness",
        priority=Priority.P1,
        acceptance_criteria=["repo-sync-audit", "report-shared"],
        validation_plan=["pytest"],
    )
    repo_sync_audit = RepoSyncAudit(
        sync=GitSyncTelemetry(
            status="failed",
            failure_category="divergence",
            summary="branch diverged from remote",
            branch="dcjcloud/ope-219",
            remote_ref="origin/dcjcloud/ope-219",
            ahead_by=2,
            behind_by=1,
        ),
        pull_request=PullRequestFreshness(
            pr_number=219,
            pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
            branch_state="out-of-sync",
            body_state="drifted",
            branch_head_sha="abc123",
            pr_head_sha="def456",
            expected_body_digest="expected",
            actual_body_digest="actual",
        ),
    )

    result = WorkflowEngine().run(
        task,
        run_id="run-wf-ope-219",
        ledger=ledger,
        journal_path=str(tmp_path / "journals" / "run-wf-ope-219.json"),
        validation_evidence=["pytest", "report-shared", "repo-sync-audit"],
        repo_sync_audit=repo_sync_audit,
        repo_sync_report_path=str(tmp_path / "reports" / "run-wf-ope-219-repo-sync.md"),
        git_push_succeeded=True,
        git_push_output="feature/OPE-219 -> origin/feature/OPE-219",
        git_log_stat_output="commit abc123\n 3 files changed, 18 insertions(+)",
    )

    assert result.acceptance.passed is True
    assert result.repo_sync_report_path is not None
    assert Path(result.repo_sync_report_path).exists()
    report = Path(result.repo_sync_report_path).read_text()
    assert "Failure Category: divergence" in report
    assert "Body State: drifted" in report

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == [
        "intake",
        "execution",
        "repo-sync",
        "acceptance",
        "closeout",
    ]
    assert journal["entries"][2]["details"]["failure_category"] == "divergence"
    entries = ledger.load()
    audit_actions = [entry["action"] for entry in entries[0]["audits"]]
    assert "repo.sync" in audit_actions
    assert "repo.pr-freshness" in audit_actions
    assert entries[0]["artifacts"][0]["name"] == "repo-sync-audit"
    assert entries[0]["closeout"]["repo_sync_audit"]["sync"]["failure_category"] == "divergence"
