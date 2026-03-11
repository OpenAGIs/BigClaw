from pathlib import Path
from typing import List, Optional

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
    render_engineering_overview,
    render_operations_dashboard,
    render_regression_center,
    render_weekly_operations_report,
    write_engineering_overview_bundle,
    write_weekly_operations_bundle,
)
from bigclaw.reports import SharedViewContext, SharedViewFilter
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


def make_shared_view(
    result_count: int,
    *,
    loading: bool = False,
    errors: Optional[List[str]] = None,
    partial_data: Optional[List[str]] = None,
) -> SharedViewContext:
    return SharedViewContext(
        filters=[
            SharedViewFilter(label="Team", value="engineering"),
            SharedViewFilter(label="Status", value="needs-approval"),
        ],
        result_count=result_count,
        loading=loading,
        errors=errors or [],
        partial_data=partial_data or [],
        last_updated="2026-03-11T09:00:00Z",
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


def test_operations_dashboard_renders_shared_view_loading_state() -> None:
    analytics = OperationsAnalytics()
    snapshot = analytics.summarize_runs([])

    dashboard = render_operations_dashboard(snapshot, view=make_shared_view(0, loading=True))

    assert "## View State" in dashboard
    assert "- State: loading" in dashboard
    assert "- Summary: Loading data for the current filters." in dashboard
    assert "- Team: engineering" in dashboard


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


def test_regression_center_renders_shared_view_partial_state() -> None:
    analytics = OperationsAnalytics()
    baseline = BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)])
    current = BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)])

    center = analytics.build_regression_center(current, baseline)
    report = render_regression_center(
        center,
        view=make_shared_view(1, partial_data=["Historical baseline fetch is delayed."]),
    )

    assert "- State: partial-data" in report
    assert "## Partial Data" in report
    assert "Historical baseline fetch is delayed." in report


def test_write_weekly_operations_bundle_emits_expected_reports(tmp_path: Path) -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-905-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
        make_run("run-2", "BIG-905-2", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:50:00Z", "hold", "requires approval for high-risk task"),
    ]
    baseline = BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)])
    current = BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)])

    weekly_report = analytics.build_weekly_report(
        name="Engineering Ops",
        period="2026-W11",
        runs=runs,
        current_suite=current,
        baseline_suite=baseline,
    )
    regression_center = analytics.build_regression_center(current, baseline)
    artifacts = write_weekly_operations_bundle(str(tmp_path / "weekly"), weekly_report, regression_center=regression_center)

    assert Path(artifacts.weekly_report_path).exists()
    assert Path(artifacts.dashboard_path).exists()
    assert artifacts.regression_center_path is not None
    assert Path(artifacts.regression_center_path).exists()
    assert "# Weekly Operations Report" in Path(artifacts.weekly_report_path).read_text()
    assert "# Operations Dashboard" in Path(artifacts.dashboard_path).read_text()
    assert "# Regression Analysis Center" in Path(artifacts.regression_center_path).read_text()


def test_build_engineering_overview_includes_kpis_funnel_blockers_and_activity() -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
        make_run("run-2", "BIG-1401-2", "running", "2026-03-10T10:00:00Z", "2026-03-10T10:30:00Z", "in flight", "long running implementation"),
        make_run("run-3", "BIG-1401-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T12:10:00Z", "approval", "requires approval for prod deploy"),
        make_run("run-4", "BIG-1401-4", "failed", "2026-03-10T12:00:00Z", "2026-03-10T12:45:00Z", "regression", "security scan failed"),
    ]

    overview = analytics.build_engineering_overview(
        name="Core Product",
        period="2026-W11",
        runs=runs,
        viewer_role="engineering-manager",
        sla_target_minutes=60,
    )

    assert overview.permissions.allowed_modules == ["kpis", "funnel", "blockers", "activity"]
    assert [kpi.name for kpi in overview.kpis] == [
        "success-rate",
        "approval-queue-depth",
        "sla-breaches",
        "average-cycle-minutes",
    ]
    assert [(stage.name, stage.count) for stage in overview.funnel] == [
        ("queued", 0),
        ("in-progress", 1),
        ("awaiting-approval", 1),
        ("completed", 1),
    ]
    assert overview.blockers[0].owner == "operations"
    assert overview.blockers[0].severity == "medium"
    assert overview.blockers[1].owner == "security"
    assert overview.blockers[1].severity == "high"
    assert overview.activities[0].run_id == "run-4"


def test_render_engineering_overview_hides_modules_without_permission() -> None:
    analytics = OperationsAnalytics()
    runs = [
        make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
        make_run("run-2", "BIG-1401-2", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "approval", "requires approval for prod deploy"),
    ]

    executive_view = analytics.build_engineering_overview(
        name="Executive View",
        period="2026-W11",
        runs=runs,
        viewer_role="executive",
    )
    contributor_view = analytics.build_engineering_overview(
        name="Contributor View",
        period="2026-W11",
        runs=runs,
        viewer_role="contributor",
    )

    executive_report = render_engineering_overview(executive_view)
    contributor_report = render_engineering_overview(contributor_view)

    assert "## KPI Modules" in executive_report
    assert "## Funnel Modules" in executive_report
    assert "## Blocker Modules" in executive_report
    assert "## Activity Modules" not in executive_report
    assert "## KPI Modules" in contributor_report
    assert "## Activity Modules" in contributor_report
    assert "## Funnel Modules" not in contributor_report
    assert "## Blocker Modules" not in contributor_report


def test_write_engineering_overview_bundle_emits_report(tmp_path: Path) -> None:
    analytics = OperationsAnalytics()
    overview = analytics.build_engineering_overview(
        name="Core Product",
        period="2026-W11",
        runs=[
            make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
            make_run("run-2", "BIG-1401-2", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "approval", "requires approval for prod deploy"),
        ],
        viewer_role="operations",
    )

    output_path = write_engineering_overview_bundle(str(tmp_path / "overview"), overview)

    assert Path(output_path).exists()
    content = Path(output_path).read_text()
    assert "# Engineering Overview" in content
    assert "Viewer Role: operations" in content
    assert "## Activity Modules" in content
