from pathlib import Path
from typing import List, Optional
import json

from bigclaw.evaluation import (
    BenchmarkCase,
    BenchmarkResult,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
    render_benchmark_suite_report,
    render_run_replay_index_page,
    render_replay_detail_page,
)
from bigclaw.execution_contract import (
    AuditPolicy,
    build_operations_api_contract,
    ExecutionApiSpec,
    ExecutionContract,
    ExecutionContractAudit,
    ExecutionContractLibrary,
    ExecutionField,
    ExecutionModel,
    ExecutionPermission,
    ExecutionPermissionMatrix,
    ExecutionRole,
    MetricDefinition,
    render_execution_contract_report,
)
from bigclaw.models import (
    BillingInterval,
    BillingRate,
    BillingSummary,
    FlowRun,
    FlowRunStatus,
    FlowStepRun,
    FlowStepStatus,
    FlowTemplate,
    FlowTemplateStep,
    FlowTrigger,
    Priority,
    RiskAssessment,
    RiskLevel,
    RiskSignal,
    Task,
    TriageLabel,
    TriageRecord,
    TriageStatus,
    UsageRecord,
)
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
    render_queue_control_center,
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
from bigclaw.queue import PersistentTaskQueue
from bigclaw.scheduler import ExecutionRecord, Scheduler, SchedulerDecision



def build_contract() -> ExecutionContract:
    return ExecutionContract(
        contract_id="BIG-EPIC-18",
        version="v4.0",
        models=[
            ExecutionModel(
                name="ExecutionRequest",
                owner="runtime",
                fields=[
                    ExecutionField("task_id", "string"),
                    ExecutionField("actor", "string"),
                    ExecutionField("requested_tools", "string[]"),
                    ExecutionField("approval_token", "string", required=False),
                ],
            ),
            ExecutionModel(
                name="ExecutionResponse",
                owner="runtime",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("sandbox_profile", "string"),
                ],
            ),
        ],
        apis=[
            ExecutionApiSpec(
                name="start_execution",
                method="POST",
                path="/execution/runs",
                request_model="ExecutionRequest",
                response_model="ExecutionResponse",
                required_permission="execution.run.write",
                emitted_audits=["execution.run.started", "execution.permission.checked"],
                emitted_metrics=["execution.request.count", "execution.duration.ms"],
            )
        ],
        permissions=[
            ExecutionPermission(
                name="execution.run.write",
                resource="execution-run",
                actions=["create"],
                scopes=["project", "workspace"],
            ),
            ExecutionPermission(
                name="execution.run.approve",
                resource="execution-run",
                actions=["approve"],
                scopes=["workspace"],
            ),
            ExecutionPermission(
                name="execution.audit.read",
                resource="execution-audit",
                actions=["read"],
                scopes=["workspace", "portfolio"],
            ),
            ExecutionPermission(
                name="execution.orchestration.manage",
                resource="orchestration-plan",
                actions=["read", "update"],
                scopes=["cross-team"],
            ),
        ],
        roles=[
            ExecutionRole(
                name="eng-lead",
                personas=["Eng Lead"],
                granted_permissions=["execution.run.write", "execution.run.approve"],
                scope_bindings=["project"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="platform-admin",
                personas=["Platform Admin"],
                granted_permissions=["execution.run.write", "execution.audit.read"],
                scope_bindings=["workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="vp-eng",
                personas=["VP Eng"],
                granted_permissions=["execution.run.approve", "execution.audit.read"],
                scope_bindings=["portfolio", "workspace"],
                escalation_target="none",
            ),
            ExecutionRole(
                name="cross-team-operator",
                personas=["Cross-Team Operator"],
                granted_permissions=["execution.run.write", "execution.orchestration.manage"],
                scope_bindings=["cross-team", "project"],
                escalation_target="eng-lead",
            ),
        ],
        metrics=[
            MetricDefinition("execution.request.count", "count", owner="runtime"),
            MetricDefinition("execution.duration.ms", "ms", owner="runtime"),
        ],
        audit_policies=[
            AuditPolicy(
                event_type="execution.run.started",
                required_fields=["task_id", "run_id", "actor"],
                retention_days=180,
                severity="info",
            ),
            AuditPolicy(
                event_type="execution.permission.checked",
                required_fields=["task_id", "actor", "permission", "allowed"],
                retention_days=180,
                severity="info",
            ),
        ],
    )


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


def test_queue_peek_tasks_returns_priority_order(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    queue.enqueue(Task(task_id="p2", source="linear", title="low", description="", priority=Priority.P2))
    queue.enqueue(Task(task_id="p0", source="linear", title="top", description="", priority=Priority.P0))
    queue.enqueue(Task(task_id="p1", source="linear", title="mid", description="", priority=Priority.P1))

    assert [task.task_id for task in queue.peek_tasks()] == ["p0", "p1", "p2"]


def test_queue_control_center_summarizes_queue_and_execution_media(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    queue.enqueue(
        Task(task_id="BIG-802-1", source="linear", title="top", description="", priority=Priority.P0, risk_level=RiskLevel.HIGH)
    )
    queue.enqueue(
        Task(task_id="BIG-802-2", source="linear", title="mid", description="", priority=Priority.P1, risk_level=RiskLevel.MEDIUM)
    )
    queue.enqueue(
        Task(task_id="BIG-802-3", source="linear", title="low", description="", priority=Priority.P2, risk_level=RiskLevel.LOW)
    )

    center = OperationsAnalytics().build_queue_control_center(
        queue,
        runs=[
            {"task_id": "BIG-802-1", "status": "needs-approval", "medium": "vm"},
            {"task_id": "BIG-802-2", "status": "approved", "medium": "browser"},
            {"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
        ],
    )

    report = render_queue_control_center(center)

    assert center.queue_depth == 3
    assert center.queued_by_priority == {"P0": 1, "P1": 1, "P2": 1}
    assert center.queued_by_risk == {"low": 1, "medium": 1, "high": 1}
    assert center.execution_media == {"vm": 1, "browser": 1, "docker": 1}
    assert center.waiting_approval_runs == 1
    assert center.blocked_tasks == ["BIG-802-1"]
    assert center.queued_tasks == ["BIG-802-1", "BIG-802-2", "BIG-802-3"]
    assert [action.action_id for action in center.actions["BIG-802-1"]] == [
        "drill-down",
        "export",
        "add-note",
        "escalate",
        "retry",
        "pause",
        "reassign",
        "audit",
    ]
    assert center.actions["BIG-802-1"][3].enabled is True
    assert center.actions["BIG-802-1"][4].enabled is True
    assert center.actions["BIG-802-1"][5].enabled is False
    assert "# Queue Control Center" in report
    assert "- Waiting Approval Runs: 1" in report
    assert "- BIG-802-1" in report
    assert "BIG-802-1: Drill Down [drill-down]" in report
    assert "Escalate [escalate] state=enabled" in report
    assert "Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused" in report


def test_queue_control_center_renders_shared_view_empty_state(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    center = OperationsAnalytics().build_queue_control_center(queue, runs=[])

    report = render_queue_control_center(
        center,
        view=SharedViewContext(
            filters=[SharedViewFilter(label="Team", value="operations")],
            result_count=0,
            empty_message="No queued work for the selected team.",
        ),
    )

    assert "## View State" in report
    assert "- State: empty" in report
    assert "- Summary: No queued work for the selected team." in report
    assert "- Team: operations" in report


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


def test_queue_persistence_and_priority(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(Task(task_id="t2", source="linear", title="P1", description="", priority=Priority.P1))
    q.enqueue(Task(task_id="t1", source="linear", title="P0", description="", priority=Priority.P0))

    assert q.size() == 2
    first = q.dequeue()
    assert first["task_id"] == "t1"

    q2 = PersistentTaskQueue(str(qfile))
    assert q2.size() == 1


def test_queue_creates_parent_directory_and_preserves_task_payload(tmp_path: Path):
    qfile = tmp_path / "state" / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(
        Task(
            task_id="t-meta",
            source="linear",
            title="Persist metadata",
            description="keep fields",
            labels=["platform"],
            required_tools=["browser"],
            acceptance_criteria=["queue survives restart"],
            validation_plan=["pytest tests/test_queue.py"],
            priority=Priority.P1,
        )
    )

    reloaded = PersistentTaskQueue(str(qfile))
    task = reloaded.dequeue_task()

    assert qfile.exists()
    assert task is not None
    assert task.labels == ["platform"]
    assert task.required_tools == ["browser"]
    assert task.acceptance_criteria == ["queue survives restart"]
    assert task.validation_plan == ["pytest tests/test_queue.py"]


def test_queue_dead_letter_and_retry_persist_across_reload(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))
    q.enqueue(Task(task_id="t-dead", source="linear", title="dead", description="", priority=Priority.P0))

    task = q.dequeue_task()
    assert task is not None

    q.dead_letter(task, reason="executor crashed")

    dead_letters = q.list_dead_letters()
    assert q.size() == 0
    assert len(dead_letters) == 1
    assert dead_letters[0].task.task_id == "t-dead"
    assert dead_letters[0].reason == "executor crashed"

    reloaded = PersistentTaskQueue(str(qfile))
    persisted_dead_letters = reloaded.list_dead_letters()
    assert len(persisted_dead_letters) == 1
    assert persisted_dead_letters[0].task.task_id == "t-dead"

    assert reloaded.retry_dead_letter("t-dead") is True
    assert reloaded.retry_dead_letter("missing") is False
    assert reloaded.list_dead_letters() == []
    assert reloaded.size() == 1

    replayed = PersistentTaskQueue(str(qfile)).dequeue_task()
    assert replayed is not None
    assert replayed.task_id == "t-dead"


def test_queue_loads_legacy_list_storage(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    qfile.write_text(
        json.dumps(
            [
                {
                    "priority": 0,
                    "task_id": "legacy",
                    "task": Task(
                        task_id="legacy",
                        source="linear",
                        title="legacy",
                        description="legacy payload",
                        priority=Priority.P0,
                    ).to_dict(),
                }
            ]
        ),
        encoding="utf-8",
    )

    queue = PersistentTaskQueue(str(qfile))

    assert queue.size() == 1
    task = queue.dequeue_task()
    assert task is not None
    assert task.task_id == "legacy"


def test_risk_assessment_round_trip_preserves_signals_and_mitigations() -> None:
    assessment = RiskAssessment(
        assessment_id="risk-1",
        task_id="OPE-130",
        level=RiskLevel.HIGH,
        total_score=75,
        requires_approval=True,
        signals=[
            RiskSignal(
                name="prod-deploy",
                score=20,
                reason="production deployment surface",
                source="scheduler",
                metadata={"tool": "deploy"},
            )
        ],
        mitigations=["security review", "rollback plan"],
        reviewer="ops-review",
        notes="Requires explicit sign-off.",
    )

    payload = assessment.to_dict()
    restored = RiskAssessment.from_dict(payload)

    assert payload["level"] == "high"
    assert restored == assessment
    assert restored.signals[0].metadata == {"tool": "deploy"}


def test_triage_record_round_trip_preserves_queue_labels_and_actions() -> None:
    record = TriageRecord(
        triage_id="triage-1",
        task_id="OPE-130",
        status=TriageStatus.ESCALATED,
        queue="risk-review",
        owner="ops",
        summary="High-risk flow needs billing review",
        labels=[
            TriageLabel(name="risk", confidence=0.9, source="heuristic"),
            TriageLabel(name="billing", confidence=0.8, source="classifier"),
        ],
        related_run_id="run-1",
        escalation_target="finance",
        actions=["route-to-finance", "request-approval"],
    )

    payload = record.to_dict()
    restored = TriageRecord.from_dict(payload)

    assert payload["status"] == "escalated"
    assert restored == record
    assert [label.name for label in restored.labels] == ["risk", "billing"]


def test_flow_template_and_run_round_trip_preserve_steps_and_outputs() -> None:
    template = FlowTemplate(
        template_id="flow-template-1",
        name="Risk Triage Flow",
        version="v1",
        description="Routes risky work through triage and approval.",
        trigger=FlowTrigger.EVENT,
        default_risk=RiskLevel.MEDIUM,
        steps=[
            FlowTemplateStep(
                step_id="triage",
                name="Triage",
                kind="review",
                required_tools=["browser"],
                approvals=["ops"],
                metadata={"lane": "risk"},
            ),
            FlowTemplateStep(
                step_id="approve",
                name="Approval",
                kind="approval",
                approvals=["security"],
            ),
        ],
        tags=["risk", "triage"],
    )
    run = FlowRun(
        run_id="flow-run-1",
        template_id=template.template_id,
        task_id="OPE-130",
        status=FlowRunStatus.RUNNING,
        triggered_by="scheduler",
        started_at="2026-03-11T10:00:00Z",
        steps=[
            FlowStepRun(
                step_id="triage",
                status=FlowStepStatus.SUCCEEDED,
                actor="ops",
                started_at="2026-03-11T10:00:00Z",
                completed_at="2026-03-11T10:02:00Z",
                output={"decision": "escalate"},
            ),
            FlowStepRun(
                step_id="approve",
                status=FlowStepStatus.RUNNING,
                actor="security",
            ),
        ],
        outputs={"ticket": "SEC-42"},
        approval_refs=["security-review"],
    )

    template_payload = template.to_dict()
    run_payload = run.to_dict()

    assert FlowTemplate.from_dict(template_payload) == template
    assert FlowRun.from_dict(run_payload) == run
    assert run_payload["steps"][0]["status"] == "succeeded"
    assert template_payload["trigger"] == "event"


def test_billing_summary_round_trip_preserves_rates_and_usage() -> None:
    summary = BillingSummary(
        statement_id="bill-1",
        account_id="acct-1",
        billing_period="2026-03",
        rates=[
            BillingRate(
                metric="orchestration-run",
                interval=BillingInterval.MONTHLY,
                included_units=100,
                unit_price_usd=0.0,
                overage_price_usd=1.5,
            )
        ],
        usage=[
            UsageRecord(
                record_id="usage-1",
                account_id="acct-1",
                metric="orchestration-run",
                quantity=124,
                period="2026-03",
                run_id="flow-run-1",
                unit="run",
                metadata={"source": "workflow-engine"},
            )
        ],
        subtotal_usd=0.0,
        overage_usd=36.0,
        total_usd=36.0,
    )

    payload = summary.to_dict()
    restored = BillingSummary.from_dict(payload)

    assert payload["rates"][0]["interval"] == "monthly"
    assert restored == summary
    assert restored.usage[0].metadata["source"] == "workflow-engine"


def test_execution_contract_audit_accepts_well_formed_contract() -> None:
    contract = build_contract()

    audit = ExecutionContractLibrary().audit(contract)
    report = render_execution_contract_report(contract, audit)

    assert audit.release_ready is True
    assert audit.readiness_score == 100.0
    assert "- Release Ready: True" in report
    assert "POST /execution/runs" in report


def test_execution_contract_audit_surfaces_contract_gaps() -> None:
    contract = build_contract()
    contract.models[0] = ExecutionModel(
        name="ExecutionRequest",
        owner="runtime",
        fields=[ExecutionField("task_id", "string")],
    )
    contract.apis[0] = ExecutionApiSpec(
        name="start_execution",
        method="POST",
        path="/execution/runs",
        request_model="ExecutionRequest",
        response_model="MissingResponse",
        required_permission="execution.run.approve",
        emitted_audits=["execution.run.finished"],
        emitted_metrics=["execution.queue.depth"],
    )
    contract.audit_policies[0] = AuditPolicy(
        event_type="execution.run.started",
        required_fields=["task_id"],
        retention_days=7,
        severity="info",
    )
    contract.roles = [
        ExecutionRole(
            name="eng-lead",
            personas=[],
            granted_permissions=[],
            scope_bindings=[],
            escalation_target="",
        ),
        ExecutionRole(
            name="platform-admin",
            personas=["Platform Admin"],
            granted_permissions=["execution.audit.override"],
            scope_bindings=["workspace"],
            escalation_target="vp-eng",
        ),
    ]

    audit = ExecutionContractLibrary().audit(contract)

    assert audit.models_missing_required_fields == {
        "ExecutionRequest": ["actor", "requested_tools"]
    }
    assert audit.undefined_model_refs == {"start_execution": ["MissingResponse"]}
    assert audit.undefined_permissions == {}
    assert audit.missing_roles == ["cross-team-operator", "vp-eng"]
    assert audit.roles_missing_personas == ["eng-lead"]
    assert audit.roles_missing_scope_bindings == ["eng-lead"]
    assert audit.roles_missing_escalation_targets == ["eng-lead"]
    assert audit.roles_missing_permissions == ["eng-lead"]
    assert audit.undefined_role_permissions == {"platform-admin": ["execution.audit.override"]}
    assert audit.apis_without_role_coverage == ["start_execution"]
    assert audit.permissions_without_roles == [
        "execution.audit.read",
        "execution.orchestration.manage",
        "execution.run.approve",
        "execution.run.write",
    ]
    assert audit.undefined_metrics == {"start_execution": ["execution.queue.depth"]}
    assert audit.undefined_audit_events == {"start_execution": ["execution.run.finished"]}
    assert audit.audit_policies_below_retention == ["execution.run.started"]
    assert audit.release_ready is False


def test_execution_contract_round_trip_and_permission_matrix() -> None:
    contract = build_contract()
    audit = ExecutionContractAudit.from_dict(ExecutionContractLibrary().audit(contract).to_dict())
    restored = ExecutionContract.from_dict(contract.to_dict())
    matrix = ExecutionPermissionMatrix(restored.permissions, restored.roles)
    decision = matrix.evaluate(
        ["execution.run.write", "missing.permission"],
        ["execution.run.write", "unknown.permission"],
    )
    role_decision = matrix.evaluate_roles(
        ["execution.run.write", "execution.orchestration.manage"],
        ["cross-team-operator", "unknown-role"],
    )

    assert restored == contract
    assert audit.release_ready is True
    assert decision.allowed is False
    assert decision.granted_permissions == ["execution.run.write"]
    assert decision.missing_permissions == ["missing.permission"]
    assert role_decision.allowed is True
    assert role_decision.granted_permissions == ["execution.orchestration.manage", "execution.run.write"]
    assert role_decision.missing_permissions == []


def test_render_execution_contract_report_includes_role_matrix() -> None:
    contract = build_contract()

    report = render_execution_contract_report(contract, ExecutionContractLibrary().audit(contract))

    assert "- Roles: 4" in report
    assert "## Roles" in report
    assert "- eng-lead: personas=Eng Lead permissions=execution.run.write, execution.run.approve" in report
    assert "- Missing roles: none" in report
    assert "- Roles missing personas: none" in report
    assert "- Roles missing scope bindings: none" in report
    assert "- Roles missing escalation targets: none" in report


def test_operations_api_contract_draft_is_release_ready() -> None:
    contract = build_operations_api_contract()

    audit = ExecutionContractLibrary().audit(contract)
    report = render_execution_contract_report(contract, audit)

    assert contract.contract_id == "OPE-131"
    assert audit.release_ready is True
    assert len(contract.apis) == 12
    assert "GET /operations/dashboard" in report
    assert "GET /operations/runs/{run_id}" in report
    assert "GET /operations/queue/control-center" in report
    assert "GET /operations/risk/overview" in report
    assert "GET /operations/sla/overview" in report
    assert "GET /operations/regressions" in report
    assert "GET /operations/flows/{run_id}" in report
    assert "GET /operations/billing/entitlements" in report


def test_operations_api_contract_permissions_cover_read_and_action_paths() -> None:
    contract = build_operations_api_contract()
    matrix = ExecutionPermissionMatrix(contract.permissions)

    viewer = matrix.evaluate(
        ["operations.dashboard.read", "operations.queue.read", "operations.run.read"],
        ["operations.dashboard.read", "operations.queue.read", "operations.run.read"],
    )
    operator = matrix.evaluate(
        ["operations.queue.act", "operations.run.approve", "operations.billing.read"],
        ["operations.queue.act", "operations.billing.read"],
    )

    assert viewer.allowed is True
    assert operator.allowed is False
    assert operator.missing_permissions == ["operations.run.approve"]


def test_execution_contract_audit_requires_persona_scope_and_escalation_metadata() -> None:
    contract = build_contract()
    contract.roles[0] = ExecutionRole(
        name="eng-lead",
        personas=[],
        granted_permissions=["execution.run.write"],
        scope_bindings=[],
        escalation_target="",
    )

    audit = ExecutionContractLibrary().audit(contract)

    assert audit.roles_missing_personas == ["eng-lead"]
    assert audit.roles_missing_scope_bindings == ["eng-lead"]
    assert audit.roles_missing_escalation_targets == ["eng-lead"]
    assert audit.release_ready is False
