from pathlib import Path
from typing import List, Optional

from bigclaw.evaluation import (
    BenchmarkCase,
    BenchmarkResult,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
    render_benchmark_suite_report,
    render_replay_detail_page,
    render_run_replay_index_page,
)
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.operations import (
    DashboardBuilder,
    DashboardLayout,
    DashboardWidgetPlacement,
    DashboardWidgetSpec,
    OperationsAnalytics,
    build_repo_collaboration_metrics,
    render_operations_metric_spec,
    render_dashboard_builder_report,
    VersionedArtifact,
    render_engineering_overview,
    render_operations_dashboard,
    render_policy_prompt_version_center,
    render_regression_center,
    render_weekly_operations_report,
    write_dashboard_builder_bundle,
    write_engineering_overview_bundle,
    write_weekly_operations_bundle,
)
from bigclaw.reports import SharedViewContext, SharedViewFilter
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import ExecutionRecord, Scheduler, SchedulerDecision



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


def make_versioned_artifact(
    artifact_type: str,
    artifact_id: str,
    version: str,
    updated_at: str,
    summary: str,
    content: str,
    *,
    author: str = "ops-bot",
    change_ticket: Optional[str] = None,
) -> VersionedArtifact:
    return VersionedArtifact(
        artifact_type=artifact_type,
        artifact_id=artifact_id,
        version=version,
        updated_at=updated_at,
        author=author,
        summary=summary,
        content=content,
        change_ticket=change_ticket,
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


def test_operations_metric_spec_defines_and_computes_operational_metrics() -> None:
    analytics = OperationsAnalytics()
    runs = [
        {
            **make_run(
                "run-1",
                "BIG-4305-1",
                "approved",
                "2026-03-11T00:10:00Z",
                "2026-03-11T00:40:00Z",
                "ok",
                "default low risk path",
            ),
            "risk_level": "low",
            "spend_usd": 4.25,
        },
        {
            **make_run(
                "run-2",
                "BIG-4305-2",
                "needs-approval",
                "2026-03-11T02:00:00Z",
                "2026-03-11T03:30:00Z",
                "manual",
                "requires approval for production rollout",
            ),
            "risk_score": {"total": 88},
            "cost_usd": 7.5,
        },
        {
            **make_run(
                "run-3",
                "BIG-4305-3",
                "approved",
                "2026-03-10T23:30:00Z",
                "2026-03-11T00:20:00Z",
                "overnight",
                "batch cleanup",
            ),
            "risk_level": "medium",
            "spend": 3,
        },
    ]

    baseline_suite = BenchmarkSuiteResult(version="v1.0.0", results=[make_result("case-1", 92, True), make_result("case-2", 88, True)])
    current_suite = BenchmarkSuiteResult(version="v1.1.0", results=[make_result("case-1", 70, False), make_result("case-2", 90, True)])

    spec = analytics.build_metric_spec(
        runs,
        period_start="2026-03-11T00:00:00Z",
        period_end="2026-03-11T23:59:59Z",
        timezone_name="UTC",
        generated_at="2026-03-11T09:00:00Z",
        sla_target_minutes=60,
        current_suite=current_suite,
        baseline_suite=baseline_suite,
    )

    values = {value.metric_id: value for value in spec.values}

    assert [definition.metric_id for definition in spec.definitions] == [
        "runs-today",
        "avg-lead-time",
        "intervention-rate",
        "sla",
        "regression",
        "risk",
        "spend",
    ]
    assert values["runs-today"].value == 2
    assert values["avg-lead-time"].value == 56.7
    assert values["intervention-rate"].value == 33.3
    assert values["sla"].value == 66.7
    assert values["regression"].value == 1
    assert values["risk"].value == 57.7
    assert values["spend"].value == 14.75


def test_build_repo_collaboration_metrics() -> None:
    runs = [
        {
            "run_id": "r1",
            "closeout": {
                "run_commit_links": [{"role": "candidate"}],
                "accepted_commit_hash": "abc123",
            },
            "repo_discussion_posts": 3,
            "accepted_lineage_depth": 2,
        },
        {
            "run_id": "r2",
            "closeout": {
                "run_commit_links": [],
                "accepted_commit_hash": "",
            },
            "repo_discussion_posts": 1,
            "accepted_lineage_depth": 4,
        },
    ]

    metrics = build_repo_collaboration_metrics(runs)

    assert metrics["repo_link_coverage"] == 50.0
    assert metrics["accepted_commit_rate"] == 50.0
    assert metrics["discussion_density"] == 2.0
    assert metrics["accepted_lineage_depth_avg"] == 3.0


def test_render_and_bundle_operations_metric_spec(tmp_path: Path) -> None:
    analytics = OperationsAnalytics()
    runs = [
        {
            **make_run(
                "run-1",
                "BIG-4305-1",
                "approved",
                "2026-03-11T00:10:00Z",
                "2026-03-11T00:40:00Z",
                "ok",
                "default low risk path",
            ),
            "risk_level": "low",
            "spend_usd": 4.25,
        }
    ]
    report = analytics.build_weekly_report(name="Ops Weekly", period="2026-W11", runs=runs)
    metric_spec = analytics.build_metric_spec(
        runs,
        period_start="2026-03-11T00:00:00Z",
        period_end="2026-03-11T23:59:59Z",
        generated_at="2026-03-11T09:00:00Z",
    )

    rendered = render_operations_metric_spec(metric_spec)
    artifacts = write_weekly_operations_bundle(str(tmp_path), report, metric_spec=metric_spec)

    assert "# Operations Metric Spec" in rendered
    assert "### Runs Today" in rendered
    assert "Spend" in rendered
    assert artifacts.metric_spec_path is not None
    assert Path(artifacts.metric_spec_path).exists()
    assert "Intervention Rate" in Path(artifacts.metric_spec_path).read_text()


def test_dashboard_builder_round_trip_preserves_manifest_shape() -> None:
    analytics = OperationsAnalytics()
    builder = DashboardBuilder(
        name="Exec Builder",
        period="2026-W11",
        owner="ops-lead",
        permissions=analytics._permissions_for_role("engineering-manager"),
        widgets=[
            DashboardWidgetSpec(
                widget_id="success-rate",
                title="Success Rate",
                module="kpis",
                data_source="operations.snapshot",
            )
        ],
        layouts=[
            DashboardLayout(
                layout_id="desktop",
                name="Desktop",
                placements=[
                    DashboardWidgetPlacement(
                        placement_id="success-rate-main",
                        widget_id="success-rate",
                        column=0,
                        row=0,
                        width=4,
                        height=2,
                    )
                ],
            )
        ],
        documentation_complete=True,
    )

    restored = DashboardBuilder.from_dict(builder.to_dict())

    assert restored == builder


def test_normalize_dashboard_layout_clamps_dimensions_and_sorts_placements() -> None:
    analytics = OperationsAnalytics()
    widgets = [
        DashboardWidgetSpec(
            widget_id="success-rate",
            title="Success Rate",
            module="kpis",
            data_source="operations.snapshot",
            min_width=3,
            max_width=6,
        )
    ]
    layout = DashboardLayout(
        layout_id="desktop",
        name="Desktop",
        placements=[
            DashboardWidgetPlacement(
                placement_id="late",
                widget_id="success-rate",
                column=8,
                row=4,
                width=8,
                height=0,
            ),
            DashboardWidgetPlacement(
                placement_id="early",
                widget_id="success-rate",
                column=-2,
                row=-1,
                width=1,
                height=2,
            ),
        ],
    )

    normalized = analytics.normalize_dashboard_layout(layout, widgets)

    assert [placement.placement_id for placement in normalized.placements] == ["early", "late"]
    assert normalized.placements[0].column == 0
    assert normalized.placements[0].row == 0
    assert normalized.placements[0].width == 3
    assert normalized.placements[1].column == 6
    assert normalized.placements[1].width == 6
    assert normalized.placements[1].height == 1


def test_dashboard_builder_audit_flags_governance_gaps() -> None:
    analytics = OperationsAnalytics()
    builder = DashboardBuilder(
        name="Contributor Builder",
        period="2026-W11",
        owner="ic-user",
        permissions=analytics._permissions_for_role("contributor"),
        widgets=[
            DashboardWidgetSpec(
                widget_id="success-rate",
                title="Success Rate",
                module="kpis",
                data_source="operations.snapshot",
            ),
            DashboardWidgetSpec(
                widget_id="approval-queue",
                title="Approval Queue",
                module="blockers",
                data_source="operations.snapshot",
            ),
        ],
        layouts=[
            DashboardLayout(
                layout_id="desktop",
                name="Desktop",
                placements=[
                    DashboardWidgetPlacement(
                        placement_id="dup",
                        widget_id="success-rate",
                        column=0,
                        row=0,
                        width=4,
                        height=2,
                    ),
                    DashboardWidgetPlacement(
                        placement_id="dup",
                        widget_id="approval-queue",
                        column=2,
                        row=1,
                        width=4,
                        height=2,
                    ),
                    DashboardWidgetPlacement(
                        placement_id="ghost",
                        widget_id="missing-widget",
                        column=10,
                        row=0,
                        width=4,
                        height=2,
                    ),
                ],
            ),
            DashboardLayout(layout_id="tablet", name="Tablet"),
        ],
        documentation_complete=False,
    )

    audit = analytics.audit_dashboard_builder(builder)

    assert audit.duplicate_placement_ids == ["dup"]
    assert audit.missing_widget_defs == ["missing-widget"]
    assert audit.inaccessible_widgets == ["approval-queue"]
    assert audit.overlapping_placements == ["desktop:dup<->dup"]
    assert audit.out_of_bounds_placements == ["ghost"]
    assert audit.empty_layouts == ["tablet"]
    assert audit.release_ready is False


def test_render_and_write_dashboard_builder_report(tmp_path: Path) -> None:
    analytics = OperationsAnalytics()
    builder = analytics.build_dashboard_builder(
        name="Manager Builder",
        period="2026-W11",
        owner="manager",
        viewer_role="engineering-manager",
        widgets=[
            DashboardWidgetSpec(
                widget_id="success-rate",
                title="Success Rate",
                module="kpis",
                data_source="operations.snapshot",
            ),
            DashboardWidgetSpec(
                widget_id="recent-activity",
                title="Recent Activity",
                module="activity",
                data_source="operations.runs",
            ),
        ],
        layouts=[
            DashboardLayout(
                layout_id="desktop",
                name="Desktop",
                placements=[
                    DashboardWidgetPlacement(
                        placement_id="kpi-main",
                        widget_id="success-rate",
                        column=0,
                        row=0,
                        width=4,
                        height=2,
                    ),
                    DashboardWidgetPlacement(
                        placement_id="activity-main",
                        widget_id="recent-activity",
                        column=4,
                        row=0,
                        width=8,
                        height=3,
                        filters=["team=engineering"],
                    ),
                ],
            )
        ],
        documentation_complete=True,
    )
    audit = analytics.audit_dashboard_builder(builder)

    report = render_dashboard_builder_report(builder, audit, view=make_shared_view(2))
    report_path = write_dashboard_builder_bundle(str(tmp_path / "dashboard"), builder, audit, view=make_shared_view(2))

    assert audit.release_ready is True
    assert "# Dashboard Builder" in report
    assert "- Release Ready: True" in report
    assert "- desktop: name=Desktop columns=12 placements=2" in report
    assert "filters=team=engineering" in report
    assert Path(report_path).exists()
    assert "# Dashboard Builder" in Path(report_path).read_text()



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


def test_build_policy_prompt_version_center_tracks_history_diff_and_rollback() -> None:
    analytics = OperationsAnalytics()
    center = analytics.build_policy_prompt_version_center(
        [
            make_versioned_artifact(
                "workflow",
                "deploy-prod",
                "v1",
                "2026-03-09T08:00:00Z",
                "initial rollout",
                "step: build\nstep: deploy\n",
                change_ticket="OPE-101",
            ),
            make_versioned_artifact(
                "workflow",
                "deploy-prod",
                "v2",
                "2026-03-10T08:00:00Z",
                "added verification gate",
                "step: build\nstep: verify\nstep: deploy\n",
                change_ticket="OPE-111",
            ),
            make_versioned_artifact(
                "policy",
                "prod-approval",
                "v3",
                "2026-03-10T10:00:00Z",
                "tighten reviewer quorum",
                "approvals: 2\nregions: us, eu\n",
                change_ticket="OPE-109",
            ),
        ],
        generated_at="2026-03-11T09:30:00Z",
    )

    assert center.artifact_count == 2
    assert center.rollback_ready_count == 1
    workflow_history = next(history for history in center.histories if history.artifact_id == "deploy-prod")
    assert workflow_history.current_version == "v2"
    assert workflow_history.rollback_version == "v1"
    assert workflow_history.rollback_ready is True
    assert workflow_history.change_summary is not None
    assert workflow_history.change_summary.additions == 1
    assert workflow_history.change_summary.deletions == 0
    assert workflow_history.change_summary.preview[:2] == ["--- v1", "+++ v2"]


def test_render_policy_prompt_version_center_supports_shared_view_context() -> None:
    analytics = OperationsAnalytics()
    center = analytics.build_policy_prompt_version_center(
        [
            make_versioned_artifact(
                "prompt",
                "triage-system",
                "v2",
                "2026-03-10T14:00:00Z",
                "reduce false escalations",
                "system: keep concise\nrubric: strict\n",
            ),
            make_versioned_artifact(
                "prompt",
                "triage-system",
                "v1",
                "2026-03-08T14:00:00Z",
                "initial prompt",
                "system: keep concise\n",
            ),
        ]
    )

    report = render_policy_prompt_version_center(
        center,
        view=make_shared_view(1, partial_data=["Rollback simulation still running."]),
    )

    assert "# Policy/Prompt Version Center" in report
    assert "### prompt / triage-system" in report
    assert "- Rollback Version: v1" in report
    assert "```diff" in report
    assert "- State: partial-data" in report
    assert "Rollback simulation still running." in report


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
    version_center = analytics.build_policy_prompt_version_center(
        [
            make_versioned_artifact(
                "policy",
                "release-approval",
                "v2",
                "2026-03-10T09:00:00Z",
                "add rollback owner",
                "approvals: 2\nrollback_owner: release-manager\n",
            ),
            make_versioned_artifact(
                "policy",
                "release-approval",
                "v1",
                "2026-03-08T09:00:00Z",
                "initial policy",
                "approvals: 2\n",
            ),
        ]
    )
    artifacts = write_weekly_operations_bundle(
        str(tmp_path / "weekly"),
        weekly_report,
        regression_center=regression_center,
        version_center=version_center,
    )

    assert Path(artifacts.weekly_report_path).exists()
    assert Path(artifacts.dashboard_path).exists()
    assert artifacts.regression_center_path is not None
    assert Path(artifacts.regression_center_path).exists()
    assert artifacts.version_center_path is not None
    assert Path(artifacts.version_center_path).exists()
    assert "# Weekly Operations Report" in Path(artifacts.weekly_report_path).read_text()
    assert "# Operations Dashboard" in Path(artifacts.dashboard_path).read_text()
    assert "# Regression Analysis Center" in Path(artifacts.regression_center_path).read_text()
    assert "# Policy/Prompt Version Center" in Path(artifacts.version_center_path).read_text()


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


def test_benchmark_runner_scores_and_replays_case(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="browser-low-risk",
        task=Task(
            task_id="BIG-601",
            source="linear",
            title="Run browser benchmark",
            description="validate routing",
            risk_level=RiskLevel.LOW,
            required_tools=["browser"],
        ),
        expected_medium="browser",
        expected_approved=True,
        expected_status="approved",
        require_report=True,
    )

    result = runner.run_case(case)

    assert result.score == 100
    assert result.passed is True
    assert result.replay.matched is True
    assert (tmp_path / "browser-low-risk" / "task-run.md").exists()
    assert (tmp_path / "benchmark-browser-low-risk" / "replay.html").exists()
    assert (tmp_path / "browser-low-risk" / "run-detail.html").exists()
    assert result.detail_page_path == str(tmp_path / "browser-low-risk" / "run-detail.html")


def test_benchmark_runner_detects_failed_expectation(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="high-risk-gate",
        task=Task(
            task_id="BIG-601-risk",
            source="jira",
            title="Prod change benchmark",
            description="must stop for approval",
            risk_level=RiskLevel.HIGH,
        ),
        expected_medium="docker",
        expected_approved=False,
        expected_status="needs-approval",
    )

    result = runner.run_case(case)

    assert result.passed is False
    assert result.score == 60
    assert any(item.name == "decision-medium" and item.passed is False for item in result.criteria)


def test_replay_outcome_reports_mismatch(tmp_path: Path):
    runner = BenchmarkRunner(scheduler=Scheduler(), storage_dir=str(tmp_path))
    replay_record = ReplayRecord(
        task=Task(
            task_id="BIG-601-replay",
            source="github",
            title="Replay browser route",
            description="compare deterministic scheduler behavior",
            required_tools=["browser"],
        ),
        run_id="run-1",
        medium="docker",
        approved=True,
        status="approved",
    )

    outcome = runner.replay(replay_record)

    assert outcome.matched is False
    assert outcome.mismatches == ["medium expected docker got browser"]
    assert outcome.report_path is not None
    assert Path(outcome.report_path).exists()


def test_suite_comparison_and_report(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    improved_suite = runner.run_suite(
        [
            BenchmarkCase(
                case_id="browser-low-risk",
                task=Task(
                    task_id="BIG-601-v2",
                    source="linear",
                    title="Run browser benchmark",
                    description="validate routing",
                    required_tools=["browser"],
                ),
                expected_medium="browser",
                expected_approved=True,
                expected_status="approved",
            )
        ],
        version="v0.2",
    )
    baseline_suite = BenchmarkSuiteResult(results=[], version="v0.1")

    comparison = improved_suite.compare(baseline_suite)
    report = render_benchmark_suite_report(improved_suite, baseline_suite)

    assert comparison[0].delta == 100
    assert improved_suite.score == 100
    assert "Version: v0.2" in report
    assert "Baseline Version: v0.1" in report
    assert "Score Delta: 100" in report


def test_render_replay_detail_page_lists_mismatches():
    task = Task(task_id="BIG-804", source="linear", title="Replay detail", description="")
    expected = ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved")
    observed = ReplayRecord(task=task, run_id="run-1", medium="browser", approved=False, status="needs-approval")

    page = render_replay_detail_page(
        expected,
        observed,
        ["medium expected docker got browser", "approved expected True got False"],
    )

    assert "Replay Detail" in page
    assert "Timeline / Log Sync" in page
    assert "Split View" in page
    assert "Reports" in page
    assert "medium expected docker got browser" in page
    assert "needs-approval" in page


def test_render_run_replay_index_page_links_outputs(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="big-804-index",
        task=Task(
            task_id="BIG-804",
            source="linear",
            title="Run detail index",
            description="single landing page",
            required_tools=["browser"],
        ),
        expected_medium="browser",
        expected_approved=True,
        expected_status="approved",
        require_report=True,
    )

    result = runner.run_case(case)
    page = Path(result.detail_page_path).read_text()

    assert "Run Detail Index" in page
    assert "Timeline / Log Sync" in page
    assert "Acceptance" in page
    assert "Reports" in page
    assert "task-run.md" in page
    assert "replay.html" in page
    assert "decision-medium" in page


def test_render_run_replay_index_page_without_report_path(tmp_path: Path):
    task = Task(task_id="BIG-804", source="linear", title="Run detail index", description="")
    replay = ReplayOutcome(
        matched=True,
        replay_record=ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved"),
        report_path=None,
    )
    record = Scheduler().execute(
        task,
        run_id="run-1",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    page = render_run_replay_index_page("big-804-index", record, replay, [])

    assert "n/a" in page
    assert "Replay" in page


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


def test_scheduler_high_risk_requires_approval():
    s = Scheduler()
    t = Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)
    d = s.decide(t)
    assert d.medium == "vm"
    assert d.approved is False


def test_scheduler_browser_task_routes_browser():
    s = Scheduler()
    t = Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"])
    d = s.decide(t)
    assert d.medium == "browser"
    assert d.approved is True


def test_scheduler_over_budget_degrades_browser_task_to_docker():
    s = Scheduler()
    t = Task(
        task_id="z",
        source="github",
        title="budgeted ui test",
        description="",
        required_tools=["browser"],
        budget=15.0,
    )

    d = s.decide(t)

    assert d.medium == "docker"
    assert d.approved is True
    assert "budget degraded browser route to docker" in d.reason


def test_scheduler_over_budget_pauses_task():
    s = Scheduler()
    t = Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0)

    d = s.decide(t)

    assert d.medium == "none"
    assert d.approved is False
    assert d.reason == "paused: budget 5.0 below required docker budget 10.0"


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
