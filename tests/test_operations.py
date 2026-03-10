from bigclaw.evaluation import (
    BenchmarkResult,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
)
from bigclaw.models import Task
from bigclaw.observability import TaskRun
from bigclaw.operations import (
    OperationsAnalytics,
    render_operations_dashboard,
    render_regression_center,
    render_weekly_operations_report,
)
from bigclaw.scheduler import ExecutionRecord, SchedulerDecision



def make_run(
    run_id: str,
    task_id: str,
    status: str,
    started_at: str,
    ended_at: str,
    summary: str,
    reason: str,
) -> dict:
    return {
        "run_id": run_id,
        "task_id": task_id,
        "status": status,
        "started_at": started_at,
        "ended_at": ended_at,
        "summary": summary,
        "audits": [{"details": {"reason": reason}}],
    }



def make_result(case_id: str, score: int, passed: bool) -> BenchmarkResult:
    task = Task(task_id=case_id, source="linear", title=case_id, description="")
    run = TaskRun.from_task(task, run_id=f"run-{case_id}", medium="docker")
    run.finalize("approved" if passed else "needs-approval", "summary")
    record = ExecutionRecord(
        decision=SchedulerDecision(medium="docker", approved=passed, reason="reason"),
        run=run,
        report_path=None,
    )
    replay = ReplayOutcome(
        matched=passed,
        replay_record=ReplayRecord(task=task, run_id=f"run-{case_id}", medium="docker", approved=passed, status=run.status),
    )
    return BenchmarkResult(
        case_id=case_id,
        score=score,
        passed=passed,
        criteria=[EvaluationCriterion(name="score", weight=100, passed=passed, detail="detail")],
        record=record,
        replay=replay,
    )



def test_operations_snapshot_tracks_sla_and_success_rate() -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-901-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
        make_run("run-2", "BIG-901-2", "approved", "2026-03-10T11:00:00Z", "2026-03-10T12:30:00Z", "slow", "browser automation task"),
        make_run("run-3", "BIG-901-3", "needs-approval", "2026-03-10T13:00:00Z", "2026-03-10T13:45:00Z", "approval", "requires approval for high-risk task"),
    ]

    snapshot = analytics.summarize_runs(runs, sla_target_minutes=60)

    assert snapshot.total_runs == 3
    assert snapshot.status_counts == {"approved": 2, "needs-approval": 1}
    assert snapshot.success_rate == 66.7
    assert snapshot.approval_queue_depth == 1
    assert snapshot.sla_breach_count == 1
    assert snapshot.average_cycle_minutes == 51.7



def test_triage_clusters_group_actionable_runs_by_reason() -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-903-1", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:05:00Z", "hold", "requires approval for high-risk task"),
        make_run("run-2", "BIG-903-2", "failed", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "tool fail", "browser automation task"),
        make_run("run-3", "BIG-903-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:15:00Z", "hold", "requires approval for high-risk task"),
    ]

    clusters = analytics.build_triage_clusters(runs)

    assert clusters[0].reason == "requires approval for high-risk task"
    assert clusters[0].occurrences == 2
    assert clusters[0].task_ids == ["BIG-903-1", "BIG-903-3"]
    assert clusters[1].reason == "browser automation task"



def test_regression_analysis_flags_score_drop_and_pass_failure() -> None:
    analytics = OperationsAnalytics()
    baseline = BenchmarkSuiteResult(
        version="v0.1",
        results=[make_result("case-stable", 100, True), make_result("case-drop", 100, True)],
    )
    current = BenchmarkSuiteResult(
        version="v0.2",
        results=[make_result("case-stable", 100, True), make_result("case-drop", 60, False)],
    )

    regressions = analytics.analyze_regressions(current, baseline)

    assert len(regressions) == 1
    assert regressions[0].case_id == "case-drop"
    assert regressions[0].delta == -40
    assert regressions[0].severity == "high"



def test_render_weekly_operations_report_includes_blockers_and_regressions() -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-905-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
        make_run("run-2", "BIG-905-2", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:50:00Z", "hold", "requires approval for high-risk task"),
    ]
    baseline = BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)])
    current = BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)])

    report = analytics.build_weekly_report(
        name="Engineering Ops",
        period="2026-W11",
        runs=runs,
        current_suite=current,
        baseline_suite=baseline,
    )

    dashboard = render_operations_dashboard(report.snapshot)
    weekly = render_weekly_operations_report(report)

    assert "# Operations Dashboard" in dashboard
    assert "- Approval Queue Depth: 1" in dashboard
    assert "requires approval for high-risk task" in dashboard
    assert "# Weekly Operations Report" in weekly
    assert "case-drop" in weekly
    assert "severity=high" in weekly


def test_build_regression_center_separates_regressions_and_improvements() -> None:
    analytics = OperationsAnalytics()
    baseline = BenchmarkSuiteResult(
        version="v0.1",
        results=[make_result("case-drop", 100, True), make_result("case-up", 60, False), make_result("case-stable", 100, True)],
    )
    current = BenchmarkSuiteResult(
        version="v0.2",
        results=[make_result("case-drop", 70, False), make_result("case-up", 100, True), make_result("case-stable", 100, True)],
    )

    center = analytics.build_regression_center(current, baseline)
    report = render_regression_center(center)

    assert center.regression_count == 1
    assert center.regressions[0].case_id == "case-drop"
    assert center.improved_cases == ["case-up"]
    assert center.unchanged_cases == ["case-stable"]
    assert "# Regression Analysis Center" in report
    assert "case-drop" in report
    assert "case-up" in report
