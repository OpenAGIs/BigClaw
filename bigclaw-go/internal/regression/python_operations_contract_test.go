package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonOperationsContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "operations_contract.py")
	script := `import json
import tempfile
import sys

from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.evaluation import (
    BenchmarkResult,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
)
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import TaskRun
from bigclaw.operations import (
    DashboardBuilder,
    DashboardLayout,
    DashboardWidgetPlacement,
    DashboardWidgetSpec,
    OperationsAnalytics,
    VersionedArtifact,
    build_repo_collaboration_metrics,
    render_dashboard_builder_report,
    render_engineering_overview,
    render_operations_dashboard,
    render_operations_metric_spec,
    render_policy_prompt_version_center,
    render_regression_center,
    render_weekly_operations_report,
    write_dashboard_builder_bundle,
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
    errors = None,
    partial_data = None,
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
    change_ticket = None,
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


analytics = OperationsAnalytics()
snapshot = analytics.summarize_runs(
    [
        make_run("run-1", "BIG-901-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
        make_run("run-2", "BIG-901-2", "approved", "2026-03-10T11:00:00Z", "2026-03-10T12:30:00Z", "slow", "browser automation task"),
        make_run("run-3", "BIG-901-3", "needs-approval", "2026-03-10T13:00:00Z", "2026-03-10T13:45:00Z", "approval", "requires approval for high-risk task"),
    ],
    sla_target_minutes=60,
)

runs_for_metrics = [
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
metric_spec = analytics.build_metric_spec(
    runs_for_metrics,
    period_start="2026-03-11T00:00:00Z",
    period_end="2026-03-11T23:59:59Z",
    timezone_name="UTC",
    generated_at="2026-03-11T09:00:00Z",
    sla_target_minutes=60,
    current_suite=current_suite,
    baseline_suite=baseline_suite,
)
metric_values = {value.metric_id: value for value in metric_spec.values}

repo_metrics = build_repo_collaboration_metrics(
    [
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
)

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    report = analytics.build_weekly_report(name="Ops Weekly", period="2026-W11", runs=[runs_for_metrics[0]])
    rendered_metric_spec = render_operations_metric_spec(metric_spec)
    artifacts = write_weekly_operations_bundle(str(td), report, metric_spec=metric_spec)
    metric_bundle = {
        "has_title": "# Operations Metric Spec" in rendered_metric_spec,
        "has_runs_today": "### Runs Today" in rendered_metric_spec,
        "has_spend": "Spend" in rendered_metric_spec,
        "metric_spec_exists": artifacts.metric_spec_path is not None and Path(artifacts.metric_spec_path).exists(),
        "metric_spec_text": "Intervention Rate" in Path(artifacts.metric_spec_path).read_text(),
    }

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
builder_restored = DashboardBuilder.from_dict(builder.to_dict())

normalized = analytics.normalize_dashboard_layout(
    DashboardLayout(
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
    ),
    [
        DashboardWidgetSpec(
            widget_id="success-rate",
            title="Success Rate",
            module="kpis",
            data_source="operations.snapshot",
            min_width=3,
            max_width=6,
        )
    ],
)

audit = analytics.audit_dashboard_builder(
    DashboardBuilder(
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
)

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    ready_builder = analytics.build_dashboard_builder(
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
    ready_audit = analytics.audit_dashboard_builder(ready_builder)
    report = render_dashboard_builder_report(ready_builder, ready_audit, view=make_shared_view(2))
    report_path = write_dashboard_builder_bundle(str(td / "dashboard"), ready_builder, ready_audit, view=make_shared_view(2))
    ready_builder_payload = {
        "release_ready": ready_audit.release_ready,
        "has_title": "# Dashboard Builder" in report,
        "has_ready": "- Release Ready: True" in report,
        "has_layout": "- desktop: name=Desktop columns=12 placements=2" in report,
        "has_filter": "filters=team=engineering" in report,
        "report_exists": Path(report_path).exists(),
        "report_text": "# Dashboard Builder" in Path(report_path).read_text(),
    }

clusters = analytics.build_triage_clusters(
    [
        make_run("run-1", "BIG-903-1", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:05:00Z", "hold", "requires approval for high-risk task"),
        make_run("run-2", "BIG-903-2", "failed", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "tool fail", "browser automation task"),
        make_run("run-3", "BIG-903-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:15:00Z", "hold", "requires approval for high-risk task"),
    ]
)

baseline = BenchmarkSuiteResult(
    version="v0.1",
    results=[make_result("case-stable", 100, True), make_result("case-drop", 100, True)],
)
current = BenchmarkSuiteResult(
    version="v0.2",
    results=[make_result("case-stable", 100, True), make_result("case-drop", 60, False)],
)
regressions = analytics.analyze_regressions(current, baseline)

weekly_report = analytics.build_weekly_report(
    name="Engineering Ops",
    period="2026-W11",
    runs=[
        make_run("run-1", "BIG-905-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
        make_run("run-2", "BIG-905-2", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:50:00Z", "hold", "requires approval for high-risk task"),
    ],
    current_suite=BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)]),
    baseline_suite=BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)]),
)
dashboard = render_operations_dashboard(weekly_report.snapshot)
weekly = render_weekly_operations_report(weekly_report)
loading_dashboard = render_operations_dashboard(analytics.summarize_runs([]), view=make_shared_view(0, loading=True))

center = analytics.build_regression_center(
    BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False), make_result("case-up", 100, True), make_result("case-stable", 100, True)]),
    BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True), make_result("case-up", 60, False), make_result("case-stable", 100, True)]),
)
regression_report = render_regression_center(center)
partial_regression_report = render_regression_center(
    analytics.build_regression_center(
        BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)]),
        BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)]),
    ),
    view=make_shared_view(1, partial_data=["Historical baseline fetch is delayed."]),
)

version_center = analytics.build_policy_prompt_version_center(
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
workflow_history = next(history for history in version_center.histories if history.artifact_id == "deploy-prod")

version_center_report = render_policy_prompt_version_center(
    analytics.build_policy_prompt_version_center(
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
    ),
    view=make_shared_view(1, partial_data=["Rollback simulation still running."]),
)

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    weekly_bundle = write_weekly_operations_bundle(
        str(td / "weekly"),
        weekly_report,
        regression_center=analytics.build_regression_center(
            BenchmarkSuiteResult(version="v0.2", results=[make_result("case-drop", 70, False)]),
            BenchmarkSuiteResult(version="v0.1", results=[make_result("case-drop", 100, True)]),
        ),
        version_center=analytics.build_policy_prompt_version_center(
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
        ),
    )
    weekly_bundle_payload = {
        "weekly_exists": Path(weekly_bundle.weekly_report_path).exists(),
        "dashboard_exists": Path(weekly_bundle.dashboard_path).exists(),
        "regression_exists": weekly_bundle.regression_center_path is not None and Path(weekly_bundle.regression_center_path).exists(),
        "version_exists": weekly_bundle.version_center_path is not None and Path(weekly_bundle.version_center_path).exists(),
        "weekly_text": "# Weekly Operations Report" in Path(weekly_bundle.weekly_report_path).read_text(),
        "dashboard_text": "# Operations Dashboard" in Path(weekly_bundle.dashboard_path).read_text(),
    }

engineering_overview = analytics.build_engineering_overview(
    name="Core Product",
    period="2026-W11",
    runs=[
        make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
        make_run("run-2", "BIG-1401-2", "running", "2026-03-10T10:00:00Z", "2026-03-10T10:30:00Z", "in flight", "long running implementation"),
        make_run("run-3", "BIG-1401-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T12:10:00Z", "approval", "requires approval for prod deploy"),
        make_run("run-4", "BIG-1401-4", "failed", "2026-03-10T12:00:00Z", "2026-03-10T12:45:00Z", "regression", "security scan failed"),
    ],
    viewer_role="engineering-manager",
    sla_target_minutes=60,
)

executive_report = render_engineering_overview(
    analytics.build_engineering_overview(
        name="Executive View",
        period="2026-W11",
        runs=[
            make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
            make_run("run-2", "BIG-1401-2", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "approval", "requires approval for prod deploy"),
        ],
        viewer_role="executive",
    )
)
contributor_report = render_engineering_overview(
    analytics.build_engineering_overview(
        name="Contributor View",
        period="2026-W11",
        runs=[
            make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
            make_run("run-2", "BIG-1401-2", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "approval", "requires approval for prod deploy"),
        ],
        viewer_role="contributor",
    )
)

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    overview_path = write_engineering_overview_bundle(
        str(td / "overview"),
        analytics.build_engineering_overview(
            name="Core Product",
            period="2026-W11",
            runs=[
                make_run("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
                make_run("run-2", "BIG-1401-2", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "approval", "requires approval for prod deploy"),
            ],
            viewer_role="operations",
        ),
    )
    overview_bundle = {
        "exists": Path(overview_path).exists(),
        "has_title": "# Engineering Overview" in Path(overview_path).read_text(),
        "has_viewer": "Viewer Role: operations" in Path(overview_path).read_text(),
        "has_activity": "## Activity Modules" in Path(overview_path).read_text(),
    }

print(json.dumps({
    "snapshot": {
        "total_runs": snapshot.total_runs,
        "status_counts": snapshot.status_counts,
        "success_rate": snapshot.success_rate,
        "approval_queue_depth": snapshot.approval_queue_depth,
        "sla_breach_count": snapshot.sla_breach_count,
        "average_cycle_minutes": snapshot.average_cycle_minutes,
    },
    "metric_spec": {
        "definitions": [definition.metric_id for definition in metric_spec.definitions],
        "runs_today": metric_values["runs-today"].value,
        "avg_lead_time": metric_values["avg-lead-time"].value,
        "intervention_rate": metric_values["intervention-rate"].value,
        "sla": metric_values["sla"].value,
        "regression": metric_values["regression"].value,
        "risk": metric_values["risk"].value,
        "spend": metric_values["spend"].value,
    },
    "repo_metrics": repo_metrics,
    "metric_bundle": metric_bundle,
    "builder_round_trip": builder_restored == builder,
    "normalized_layout": {
        "ids": [placement.placement_id for placement in normalized.placements],
        "early_column": normalized.placements[0].column,
        "early_row": normalized.placements[0].row,
        "early_width": normalized.placements[0].width,
        "late_column": normalized.placements[1].column,
        "late_width": normalized.placements[1].width,
        "late_height": normalized.placements[1].height,
    },
    "builder_audit": {
        "duplicate_ids": audit.duplicate_placement_ids,
        "missing_widget_defs": audit.missing_widget_defs,
        "inaccessible_widgets": audit.inaccessible_widgets,
        "overlapping": audit.overlapping_placements,
        "out_of_bounds": audit.out_of_bounds_placements,
        "empty_layouts": audit.empty_layouts,
        "release_ready": audit.release_ready,
    },
    "ready_builder": ready_builder_payload,
    "clusters": {
        "first_reason": clusters[0].reason,
        "first_occurrences": clusters[0].occurrences,
        "first_task_ids": clusters[0].task_ids,
        "second_reason": clusters[1].reason,
    },
    "regressions": {
        "count": len(regressions),
        "case_id": regressions[0].case_id,
        "delta": regressions[0].delta,
        "severity": regressions[0].severity,
    },
    "weekly": {
        "dashboard_title": "# Operations Dashboard" in dashboard,
        "dashboard_queue": "- Approval Queue Depth: 1" in dashboard,
        "dashboard_reason": "requires approval for high-risk task" in dashboard,
        "weekly_title": "# Weekly Operations Report" in weekly,
        "weekly_case": "case-drop" in weekly,
        "weekly_severity": "severity=high" in weekly,
        "loading_state": "- State: loading" in loading_dashboard and "- Summary: Loading data for the current filters." in loading_dashboard,
    },
    "regression_center": {
        "regression_count": center.regression_count,
        "regression_case": center.regressions[0].case_id,
        "improved_cases": center.improved_cases,
        "unchanged_cases": center.unchanged_cases,
        "report_title": "# Regression Analysis Center" in regression_report,
        "report_drop": "case-drop" in regression_report,
        "report_up": "case-up" in regression_report,
        "partial_state": "- State: partial-data" in partial_regression_report and "Historical baseline fetch is delayed." in partial_regression_report,
    },
    "version_center": {
        "artifact_count": version_center.artifact_count,
        "rollback_ready_count": version_center.rollback_ready_count,
        "current_version": workflow_history.current_version,
        "rollback_version": workflow_history.rollback_version,
        "rollback_ready": workflow_history.rollback_ready,
        "additions": workflow_history.change_summary.additions if workflow_history.change_summary else None,
        "deletions": workflow_history.change_summary.deletions if workflow_history.change_summary else None,
        "preview": workflow_history.change_summary.preview[:2] if workflow_history.change_summary else [],
        "report_title": "# Policy/Prompt Version Center" in version_center_report,
        "report_section": "### prompt / triage-system" in version_center_report,
        "report_rollback": "- Rollback Version: v1" in version_center_report,
        "report_diff": (chr(96) * 3 + "diff") in version_center_report,
        "report_partial": "Rollback simulation still running." in version_center_report,
    },
    "weekly_bundle": weekly_bundle_payload,
    "engineering_overview": {
        "allowed_modules": engineering_overview.permissions.allowed_modules,
        "kpi_names": [kpi.name for kpi in engineering_overview.kpis],
        "funnel": [(stage.name, stage.count) for stage in engineering_overview.funnel],
        "first_blocker_owner": engineering_overview.blockers[0].owner,
        "first_blocker_severity": engineering_overview.blockers[0].severity,
        "second_blocker_owner": engineering_overview.blockers[1].owner,
        "second_blocker_severity": engineering_overview.blockers[1].severity,
        "first_activity": engineering_overview.activities[0].run_id,
        "executive_modules": {
            "kpis": "## KPI Modules" in executive_report,
            "funnel": "## Funnel Modules" in executive_report,
            "blockers": "## Blocker Modules" in executive_report,
            "activity": "## Activity Modules" in executive_report,
        },
        "contributor_modules": {
            "kpis": "## KPI Modules" in contributor_report,
            "funnel": "## Funnel Modules" in contributor_report,
            "blockers": "## Blocker Modules" in contributor_report,
            "activity": "## Activity Modules" in contributor_report,
        },
        "bundle": overview_bundle,
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write operations contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run operations contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Snapshot struct {
			TotalRuns           int            `json:"total_runs"`
			StatusCounts        map[string]int `json:"status_counts"`
			SuccessRate         float64        `json:"success_rate"`
			ApprovalQueueDepth  int            `json:"approval_queue_depth"`
			SLABreachCount      int            `json:"sla_breach_count"`
			AverageCycleMinutes float64        `json:"average_cycle_minutes"`
		} `json:"snapshot"`
		MetricSpec struct {
			Definitions      []string `json:"definitions"`
			RunsToday        float64  `json:"runs_today"`
			AvgLeadTime      float64  `json:"avg_lead_time"`
			InterventionRate float64  `json:"intervention_rate"`
			SLA              float64  `json:"sla"`
			Regression       float64  `json:"regression"`
			Risk             float64  `json:"risk"`
			Spend            float64  `json:"spend"`
		} `json:"metric_spec"`
		RepoMetrics struct {
			RepoLinkCoverage        float64 `json:"repo_link_coverage"`
			AcceptedCommitRate      float64 `json:"accepted_commit_rate"`
			DiscussionDensity       float64 `json:"discussion_density"`
			AcceptedLineageDepthAvg float64 `json:"accepted_lineage_depth_avg"`
		} `json:"repo_metrics"`
		MetricBundle     map[string]bool `json:"metric_bundle"`
		BuilderRoundTrip bool            `json:"builder_round_trip"`
		NormalizedLayout struct {
			IDs         []string `json:"ids"`
			EarlyColumn int      `json:"early_column"`
			EarlyRow    int      `json:"early_row"`
			EarlyWidth  int      `json:"early_width"`
			LateColumn  int      `json:"late_column"`
			LateWidth   int      `json:"late_width"`
			LateHeight  int      `json:"late_height"`
		} `json:"normalized_layout"`
		BuilderAudit struct {
			DuplicateIDs        []string `json:"duplicate_ids"`
			MissingWidgetDefs   []string `json:"missing_widget_defs"`
			InaccessibleWidgets []string `json:"inaccessible_widgets"`
			Overlapping         []string `json:"overlapping"`
			OutOfBounds         []string `json:"out_of_bounds"`
			EmptyLayouts        []string `json:"empty_layouts"`
			ReleaseReady        bool     `json:"release_ready"`
		} `json:"builder_audit"`
		ReadyBuilder map[string]bool `json:"ready_builder"`
		Clusters     struct {
			FirstReason      string   `json:"first_reason"`
			FirstOccurrences int      `json:"first_occurrences"`
			FirstTaskIDs     []string `json:"first_task_ids"`
			SecondReason     string   `json:"second_reason"`
		} `json:"clusters"`
		Regressions struct {
			Count    int    `json:"count"`
			CaseID   string `json:"case_id"`
			Delta    int    `json:"delta"`
			Severity string `json:"severity"`
		} `json:"regressions"`
		Weekly           map[string]bool `json:"weekly"`
		RegressionCenter struct {
			RegressionCount int      `json:"regression_count"`
			RegressionCase  string   `json:"regression_case"`
			ImprovedCases   []string `json:"improved_cases"`
			UnchangedCases  []string `json:"unchanged_cases"`
			ReportTitle     bool     `json:"report_title"`
			ReportDrop      bool     `json:"report_drop"`
			ReportUp        bool     `json:"report_up"`
			PartialState    bool     `json:"partial_state"`
		} `json:"regression_center"`
		VersionCenter struct {
			ArtifactCount      int      `json:"artifact_count"`
			RollbackReadyCount int      `json:"rollback_ready_count"`
			CurrentVersion     string   `json:"current_version"`
			RollbackVersion    string   `json:"rollback_version"`
			RollbackReady      bool     `json:"rollback_ready"`
			Additions          int      `json:"additions"`
			Deletions          int      `json:"deletions"`
			Preview            []string `json:"preview"`
			ReportTitle        bool     `json:"report_title"`
			ReportSection      bool     `json:"report_section"`
			ReportRollback     bool     `json:"report_rollback"`
			ReportDiff         bool     `json:"report_diff"`
			ReportPartial      bool     `json:"report_partial"`
		} `json:"version_center"`
		WeeklyBundle        map[string]bool `json:"weekly_bundle"`
		EngineeringOverview struct {
			AllowedModules        []string        `json:"allowed_modules"`
			KPINames              []string        `json:"kpi_names"`
			Funnel                [][]any         `json:"funnel"`
			FirstBlockerOwner     string          `json:"first_blocker_owner"`
			FirstBlockerSeverity  string          `json:"first_blocker_severity"`
			SecondBlockerOwner    string          `json:"second_blocker_owner"`
			SecondBlockerSeverity string          `json:"second_blocker_severity"`
			FirstActivity         string          `json:"first_activity"`
			ExecutiveModules      map[string]bool `json:"executive_modules"`
			ContributorModules    map[string]bool `json:"contributor_modules"`
			Bundle                map[string]bool `json:"bundle"`
		} `json:"engineering_overview"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode operations contract output: %v\n%s", err, string(output))
	}

	if decoded.Snapshot.TotalRuns != 3 || decoded.Snapshot.StatusCounts["approved"] != 2 || decoded.Snapshot.StatusCounts["needs-approval"] != 1 || decoded.Snapshot.SuccessRate != 66.7 || decoded.Snapshot.ApprovalQueueDepth != 1 || decoded.Snapshot.SLABreachCount != 1 || decoded.Snapshot.AverageCycleMinutes != 51.7 {
		t.Fatalf("unexpected operations snapshot payload: %+v", decoded.Snapshot)
	}
	if len(decoded.MetricSpec.Definitions) != 7 || decoded.MetricSpec.RunsToday != 2 || decoded.MetricSpec.AvgLeadTime != 56.7 || decoded.MetricSpec.InterventionRate != 33.3 || decoded.MetricSpec.SLA != 66.7 || decoded.MetricSpec.Regression != 1 || decoded.MetricSpec.Risk != 57.7 || decoded.MetricSpec.Spend != 14.75 {
		t.Fatalf("unexpected metric spec payload: %+v", decoded.MetricSpec)
	}
	if decoded.RepoMetrics.RepoLinkCoverage != 50 || decoded.RepoMetrics.AcceptedCommitRate != 50 || decoded.RepoMetrics.DiscussionDensity != 2 || decoded.RepoMetrics.AcceptedLineageDepthAvg != 3 {
		t.Fatalf("unexpected repo metrics payload: %+v", decoded.RepoMetrics)
	}
	for name, ok := range decoded.MetricBundle {
		if !ok {
			t.Fatalf("expected metric bundle check %s to pass", name)
		}
	}
	if !decoded.BuilderRoundTrip {
		t.Fatal("expected dashboard builder round trip to remain stable")
	}
	if len(decoded.NormalizedLayout.IDs) != 2 || decoded.NormalizedLayout.IDs[0] != "early" || decoded.NormalizedLayout.IDs[1] != "late" || decoded.NormalizedLayout.EarlyColumn != 0 || decoded.NormalizedLayout.EarlyRow != 0 || decoded.NormalizedLayout.EarlyWidth != 3 || decoded.NormalizedLayout.LateColumn != 6 || decoded.NormalizedLayout.LateWidth != 6 || decoded.NormalizedLayout.LateHeight != 1 {
		t.Fatalf("unexpected normalized layout payload: %+v", decoded.NormalizedLayout)
	}
	if len(decoded.BuilderAudit.DuplicateIDs) != 1 || decoded.BuilderAudit.DuplicateIDs[0] != "dup" || len(decoded.BuilderAudit.MissingWidgetDefs) != 1 || decoded.BuilderAudit.MissingWidgetDefs[0] != "missing-widget" || len(decoded.BuilderAudit.InaccessibleWidgets) != 1 || decoded.BuilderAudit.InaccessibleWidgets[0] != "approval-queue" || len(decoded.BuilderAudit.Overlapping) != 1 || len(decoded.BuilderAudit.OutOfBounds) != 1 || decoded.BuilderAudit.OutOfBounds[0] != "ghost" || len(decoded.BuilderAudit.EmptyLayouts) != 1 || decoded.BuilderAudit.EmptyLayouts[0] != "tablet" || decoded.BuilderAudit.ReleaseReady {
		t.Fatalf("unexpected builder audit payload: %+v", decoded.BuilderAudit)
	}
	for name, ok := range decoded.ReadyBuilder {
		if !ok {
			t.Fatalf("expected ready builder check %s to pass", name)
		}
	}
	if decoded.Clusters.FirstReason != "requires approval for high-risk task" || decoded.Clusters.FirstOccurrences != 2 || len(decoded.Clusters.FirstTaskIDs) != 2 || decoded.Clusters.FirstTaskIDs[0] != "BIG-903-1" || decoded.Clusters.SecondReason != "browser automation task" {
		t.Fatalf("unexpected clusters payload: %+v", decoded.Clusters)
	}
	if decoded.Regressions.Count != 1 || decoded.Regressions.CaseID != "case-drop" || decoded.Regressions.Delta != -40 || decoded.Regressions.Severity != "high" {
		t.Fatalf("unexpected regressions payload: %+v", decoded.Regressions)
	}
	for name, ok := range decoded.Weekly {
		if !ok {
			t.Fatalf("expected weekly check %s to pass", name)
		}
	}
	if decoded.RegressionCenter.RegressionCount != 1 || decoded.RegressionCenter.RegressionCase != "case-drop" || len(decoded.RegressionCenter.ImprovedCases) != 1 || decoded.RegressionCenter.ImprovedCases[0] != "case-up" || len(decoded.RegressionCenter.UnchangedCases) != 1 || decoded.RegressionCenter.UnchangedCases[0] != "case-stable" || !decoded.RegressionCenter.ReportTitle || !decoded.RegressionCenter.ReportDrop || !decoded.RegressionCenter.ReportUp || !decoded.RegressionCenter.PartialState {
		t.Fatalf("unexpected regression center payload: %+v", decoded.RegressionCenter)
	}
	if decoded.VersionCenter.ArtifactCount != 2 || decoded.VersionCenter.RollbackReadyCount != 1 || decoded.VersionCenter.CurrentVersion != "v2" || decoded.VersionCenter.RollbackVersion != "v1" || !decoded.VersionCenter.RollbackReady || decoded.VersionCenter.Additions != 1 || decoded.VersionCenter.Deletions != 0 || len(decoded.VersionCenter.Preview) != 2 || decoded.VersionCenter.Preview[0] != "--- v1" || decoded.VersionCenter.Preview[1] != "+++ v2" || !decoded.VersionCenter.ReportTitle || !decoded.VersionCenter.ReportSection || !decoded.VersionCenter.ReportRollback || !decoded.VersionCenter.ReportDiff || !decoded.VersionCenter.ReportPartial {
		t.Fatalf("unexpected version center payload: %+v", decoded.VersionCenter)
	}
	for name, ok := range decoded.WeeklyBundle {
		if !ok {
			t.Fatalf("expected weekly bundle check %s to pass", name)
		}
	}
	if len(decoded.EngineeringOverview.AllowedModules) != 4 || decoded.EngineeringOverview.AllowedModules[0] != "kpis" || len(decoded.EngineeringOverview.KPINames) != 4 || decoded.EngineeringOverview.KPINames[0] != "success-rate" || len(decoded.EngineeringOverview.Funnel) != 4 || decoded.EngineeringOverview.FirstBlockerOwner != "operations" || decoded.EngineeringOverview.FirstBlockerSeverity != "medium" || decoded.EngineeringOverview.SecondBlockerOwner != "security" || decoded.EngineeringOverview.SecondBlockerSeverity != "high" || decoded.EngineeringOverview.FirstActivity != "run-4" {
		t.Fatalf("unexpected engineering overview payload: %+v", decoded.EngineeringOverview)
	}
	if !decoded.EngineeringOverview.ExecutiveModules["kpis"] || !decoded.EngineeringOverview.ExecutiveModules["funnel"] || !decoded.EngineeringOverview.ExecutiveModules["blockers"] || decoded.EngineeringOverview.ExecutiveModules["activity"] {
		t.Fatalf("unexpected executive module visibility: %+v", decoded.EngineeringOverview.ExecutiveModules)
	}
	if !decoded.EngineeringOverview.ContributorModules["kpis"] || !decoded.EngineeringOverview.ContributorModules["activity"] || decoded.EngineeringOverview.ContributorModules["funnel"] || decoded.EngineeringOverview.ContributorModules["blockers"] {
		t.Fatalf("unexpected contributor module visibility: %+v", decoded.EngineeringOverview.ContributorModules)
	}
	for name, ok := range decoded.EngineeringOverview.Bundle {
		if !ok {
			t.Fatalf("expected engineering overview bundle check %s to pass", name)
		}
	}
}
