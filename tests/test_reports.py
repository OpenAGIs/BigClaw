import hashlib
from pathlib import Path
from typing import List, Optional

import pytest

from bigclaw.audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    missing_required_fields,
)
from bigclaw.collaboration import (
    CollaborationComment,
    DecisionNote,
    build_collaboration_thread,
    build_collaboration_thread_from_audits,
    merge_collaboration_threads,
)
from bigclaw.dsl import WorkflowDefinition
from bigclaw.event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
)
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
from bigclaw.observability import (
    GitSyncTelemetry,
    ObservabilityLedger,
    PullRequestFreshness,
    RepoSyncAudit,
    TaskRun,
)
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
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    PremiumOrchestrationPolicy,
    render_orchestration_plan,
)
from bigclaw.reports import (
    ConsoleAction,
    BillingEntitlementsPage,
    BillingRunCharge,
    DocumentationArtifact,
    FinalDeliveryChecklist,
    LaunchChecklistItem,
    NarrativeSection,
    OrchestrationCanvas,
    OrchestrationPortfolio,
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    ReportStudio,
    SharedViewContext,
    SharedViewFilter,
    TriageFeedbackRecord,
    build_auto_triage_center,
    build_billing_entitlements_page,
    build_billing_entitlements_page_from_ledger,
    build_final_delivery_checklist,
    build_launch_checklist,
    build_orchestration_canvas,
    build_orchestration_canvas_from_ledger_entry,
    build_orchestration_portfolio,
    build_orchestration_portfolio_from_ledger,
    build_takeover_queue_from_ledger,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_billing_entitlements_page,
    render_billing_entitlements_report,
    render_final_delivery_checklist_report,
    render_issue_validation_report,
    render_launch_checklist_report,
    render_orchestration_canvas,
    render_orchestration_overview_page,
    render_orchestration_portfolio_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_repo_narrative_exports,
    render_repo_sync_audit_report,
    render_report_studio_html,
    render_report_studio_plain_text,
    render_report_studio_report,
    render_shared_view_context,
    render_takeover_queue_report,
    render_task_run_detail_page,
    render_task_run_report,
    render_weekly_repo_evidence_section,
    validation_report_exists,
    write_report,
    write_report_studio_bundle,
)
from bigclaw.repo_board import RepoDiscussionBoard
from bigclaw.repo_plane import RunCommitLink
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import ExecutionRecord, Scheduler, SchedulerDecision
from bigclaw.workflow import WorkflowEngine

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
            SharedViewFilter(label="Window", value="2026-03-10"),
        ],
        result_count=result_count,
        loading=loading,
        errors=errors or [],
        partial_data=partial_data or [],
        last_updated="2026-03-11T09:00:00Z",
    )


def test_render_and_write_report(tmp_path: Path):
    content = render_issue_validation_report("BIG-101", "v0.1", "sandbox", "pass")
    out = tmp_path / "report.md"
    write_report(str(out), content)
    assert out.exists()
    text = out.read_text()
    assert "BIG-101" in text
    assert "pass" in text


def test_console_action_state_reflects_enabled_flag():
    enabled = ConsoleAction("retry", "Retry", "run-1")
    disabled = ConsoleAction("pause", "Pause", "run-1", enabled=False, reason="already completed")

    assert enabled.state == "enabled"
    assert disabled.state == "disabled"


def test_merge_collaboration_threads_combines_native_and_repo_surfaces() -> None:
    native = build_collaboration_thread(
        "run",
        "run-165",
        comments=[CollaborationComment(comment_id="c1", author="ops", body="native note", created_at="2026-03-12T10:00:00Z")],
        decisions=[DecisionNote(decision_id="d1", author="lead", outcome="approved", summary="native decision", recorded_at="2026-03-12T10:05:00Z")],
    )

    board = RepoDiscussionBoard()
    repo_post = board.create_post(
        channel="bigclaw-ope-165",
        author="repo-agent",
        body="repo board context",
        target_surface="run",
        target_id="run-165",
    )
    repo_thread = build_collaboration_thread(
        "repo-board",
        "run-165",
        comments=[repo_post.to_collaboration_comment()],
    )

    merged = merge_collaboration_threads(target_id="run-165", native_thread=native, repo_thread=repo_thread)

    assert merged is not None
    assert merged.surface == "merged"
    assert len(merged.comments) == 2
    assert len(merged.decisions) == 1
    assert merged.comments[1].body == "repo board context"


def test_repo_weekly_narrative_exports_remain_consistent() -> None:
    section = render_weekly_repo_evidence_section(
        experiment_volume=14,
        converged_tasks=9,
        accepted_commits=7,
        hottest_threads=["repo/ope-168", "repo/ope-170"],
    )
    exports = render_repo_narrative_exports(
        experiment_volume=14,
        converged_tasks=9,
        accepted_commits=7,
        hottest_threads=["repo/ope-168", "repo/ope-170"],
    )

    assert "Accepted Commits: 7" in section
    assert "Repo Evidence Summary" in exports["markdown"]
    assert "Accepted Commits: 7" in exports["text"]
    assert "<section><h2>Repo Evidence Summary</h2>" in exports["html"]


def test_report_studio_renders_narrative_sections_and_export_bundle(tmp_path: Path):
    studio = ReportStudio(
        name="Executive Weekly Narrative",
        issue_id="OPE-112",
        audience="executive",
        period="2026-W11",
        summary="Delivery recovered after approval bottlenecks were cleared in the second half of the week.",
        sections=[
            NarrativeSection(
                heading="What changed",
                body="Approval queue depth fell from 5 to 1 after moving browser-heavy runs onto the shared operations lane.",
                evidence=["queue-control-center", "weekly-operations"],
                callouts=["SLA risk contained", "No new regressions opened"],
            ),
            NarrativeSection(
                heading="What needs attention",
                body="Security takeover requests still cluster around data-export tasks and need a dedicated reviewer window.",
                evidence=["takeover-queue"],
                callouts=["Review staffing before Friday close"],
            ),
        ],
        action_items=["Publish the markdown export to leadership", "Review security handoff staffing"],
        source_reports=["reports/weekly-operations.md", "reports/takeover-queue.md"],
    )

    markdown = render_report_studio_report(studio)
    plain_text = render_report_studio_plain_text(studio)
    html = render_report_studio_html(studio)
    artifacts = write_report_studio_bundle(str(tmp_path / "studio"), studio)

    assert studio.ready is True
    assert studio.recommendation == "publish"
    assert "# Report Studio" in markdown
    assert "### What changed" in markdown
    assert "Recommendation: publish" in plain_text
    assert "<h1>Executive Weekly Narrative</h1>" in html
    assert Path(artifacts.markdown_path).exists()
    assert Path(artifacts.html_path).exists()
    assert Path(artifacts.text_path).exists()
    assert "executive-weekly-narrative.md" in artifacts.markdown_path


def test_report_studio_requires_summary_and_complete_sections():
    studio = ReportStudio(
        name="Draft Narrative",
        issue_id="OPE-112",
        audience="operations",
        period="2026-W11",
        summary="",
        sections=[NarrativeSection(heading="Open risks", body="")],
    )

    assert studio.ready is False
    assert studio.recommendation == "draft"


def test_render_pilot_scorecard_includes_roi_and_recommendation():
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner A",
        period="2026-Q2",
        metrics=[
            PilotMetric(
                name="Automation coverage",
                baseline=35,
                current=82,
                target=80,
                unit="%",
            ),
            PilotMetric(
                name="Manual review time",
                baseline=12,
                current=4,
                target=5,
                unit="h",
                higher_is_better=False,
            ),
        ],
        monthly_benefit=12000,
        monthly_cost=2500,
        implementation_cost=18000,
        benchmark_score=96,
        benchmark_passed=True,
    )

    content = render_pilot_scorecard(scorecard)

    assert scorecard.metrics_met == 2
    assert scorecard.recommendation == "go"
    assert scorecard.payback_months == 1.9
    assert "Annualized ROI: 200.0%" in content
    assert "Recommendation: go" in content
    assert "Benchmark Score: 96" in content
    assert "Automation coverage" in content


def test_pilot_scorecard_returns_hold_when_value_is_negative():
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner B",
        period="2026-Q2",
        metrics=[PilotMetric(name="Backlog aging", baseline=5, current=7, target=4, unit="d", higher_is_better=False)],
        monthly_benefit=1000,
        monthly_cost=3000,
        implementation_cost=12000,
        benchmark_passed=False,
    )

    assert scorecard.monthly_net_value == -2000
    assert scorecard.payback_months is None
    assert scorecard.recommendation == "hold"


def test_issue_closure_requires_non_empty_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"

    decision = evaluate_issue_closure("BIG-602", str(report_path))

    assert decision.allowed is False
    assert decision.reason == "validation report required before closing issue"
    assert validation_report_exists(str(report_path)) is False


def test_issue_closure_blocks_failed_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    write_report(str(report_path), "# Validation\n\nfailed")

    decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=False)

    assert decision.allowed is False
    assert decision.reason == "validation failed; issue must remain open"
    assert validation_report_exists(str(report_path)) is True


def test_issue_closure_allows_completed_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    content = render_issue_validation_report("BIG-602", "v0.1", "sandbox", "pass")
    write_report(str(report_path), content)

    decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=True)

    assert decision.allowed is True
    assert decision.reason == "validation report and launch checklist requirements satisfied; issue can be closed"
    assert decision.report_path == str(report_path)


def test_launch_checklist_auto_links_documentation_status(tmp_path: Path):
    runbook = tmp_path / "runbook.md"
    faq = tmp_path / "faq.md"
    write_report(str(runbook), "# Runbook\n\nready")

    checklist = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="faq", path=str(faq)),
        ],
        items=[
            LaunchChecklistItem(name="Operations handoff", evidence=["runbook"]),
            LaunchChecklistItem(name="Support handoff", evidence=["faq"]),
        ],
    )

    report = render_launch_checklist_report(checklist)

    assert checklist.documentation_status == {"runbook": True, "faq": False}
    assert checklist.completed_items == 1
    assert checklist.missing_documentation == ["faq"]
    assert checklist.ready is False
    assert "runbook: available=True" in report
    assert "faq: available=False" in report
    assert "Support handoff: completed=False evidence=faq" in report


def test_final_delivery_checklist_tracks_required_outputs_and_recommended_docs(tmp_path: Path):
    validation_bundle = tmp_path / "validation-bundle.md"
    release_notes = tmp_path / "release-notes.md"
    write_report(str(validation_bundle), "# Validation Bundle\n\nready")

    checklist = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[
            DocumentationArtifact(name="validation-bundle", path=str(validation_bundle)),
            DocumentationArtifact(name="release-notes", path=str(release_notes)),
        ],
        recommended_documentation=[
            DocumentationArtifact(name="runbook", path=str(tmp_path / "runbook.md")),
            DocumentationArtifact(name="faq", path=str(tmp_path / "faq.md")),
        ],
    )

    report = render_final_delivery_checklist_report(checklist)

    assert checklist.required_output_status == {
        "validation-bundle": True,
        "release-notes": False,
    }
    assert checklist.recommended_documentation_status == {
        "runbook": False,
        "faq": False,
    }
    assert checklist.generated_required_outputs == 1
    assert checklist.generated_recommended_documentation == 0
    assert checklist.missing_required_outputs == ["release-notes"]
    assert checklist.missing_recommended_documentation == ["runbook", "faq"]
    assert checklist.ready is False
    assert "Required Outputs Generated: 1/2" in report
    assert "Recommended Docs Generated: 0/2" in report
    assert "release-notes: available=False" in report
    assert "runbook: available=False" in report


def test_issue_closure_blocks_incomplete_linked_launch_checklist(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    runbook = tmp_path / "runbook.md"
    write_report(str(report_path), render_issue_validation_report("BIG-1003", "v0.2", "staging", "pass"))
    write_report(str(runbook), "# Runbook\n\nready")

    checklist = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="launch-faq", path=str(tmp_path / "launch-faq.md")),
        ],
        items=[LaunchChecklistItem(name="Launch comms", evidence=["runbook", "launch-faq"])],
    )

    decision = evaluate_issue_closure(
        "BIG-1003",
        str(report_path),
        validation_passed=True,
        launch_checklist=checklist,
    )

    assert decision.allowed is False
    assert decision.reason == "launch checklist incomplete; linked documentation missing or empty"


def test_issue_closure_blocks_missing_required_final_delivery_outputs(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    write_report(str(report_path), render_issue_validation_report("BIG-4702", "v0.3", "staging", "pass"))

    checklist = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[
            DocumentationArtifact(name="validation-bundle", path=str(tmp_path / "validation-bundle.md")),
        ],
        recommended_documentation=[
            DocumentationArtifact(name="runbook", path=str(tmp_path / "runbook.md")),
        ],
    )

    decision = evaluate_issue_closure(
        "BIG-4702",
        str(report_path),
        validation_passed=True,
        final_delivery_checklist=checklist,
    )

    assert decision.allowed is False
    assert decision.reason == "final delivery checklist incomplete; required outputs missing"


def test_issue_closure_allows_when_required_final_delivery_outputs_exist(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    validation_bundle = tmp_path / "validation-bundle.md"
    release_notes = tmp_path / "release-notes.md"
    write_report(str(report_path), render_issue_validation_report("BIG-4702", "v0.3", "staging", "pass"))
    write_report(str(validation_bundle), "# Validation Bundle\n\nready")
    write_report(str(release_notes), "# Release Notes\n\nready")

    checklist = build_final_delivery_checklist(
        "BIG-4702",
        required_outputs=[
            DocumentationArtifact(name="validation-bundle", path=str(validation_bundle)),
            DocumentationArtifact(name="release-notes", path=str(release_notes)),
        ],
        recommended_documentation=[
            DocumentationArtifact(name="runbook", path=str(tmp_path / "runbook.md")),
        ],
    )

    decision = evaluate_issue_closure(
        "BIG-4702",
        str(report_path),
        validation_passed=True,
        final_delivery_checklist=checklist,
    )

    assert decision.allowed is True
    assert decision.reason == "validation report and final delivery checklist requirements satisfied; issue can be closed"


def test_issue_closure_allows_when_linked_launch_checklist_is_ready(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    runbook = tmp_path / "runbook.md"
    faq = tmp_path / "launch-faq.md"
    write_report(str(report_path), render_issue_validation_report("BIG-1003", "v0.2", "staging", "pass"))
    write_report(str(runbook), "# Runbook\n\nready")
    write_report(str(faq), "# FAQ\n\nready")

    checklist = build_launch_checklist(
        "BIG-1003",
        documentation=[
            DocumentationArtifact(name="runbook", path=str(runbook)),
            DocumentationArtifact(name="launch-faq", path=str(faq)),
        ],
        items=[LaunchChecklistItem(name="Launch comms", evidence=["runbook", "launch-faq"])],
    )

    decision = evaluate_issue_closure(
        "BIG-1003",
        str(report_path),
        validation_passed=True,
        launch_checklist=checklist,
    )

    assert decision.allowed is True
    assert decision.reason == "validation report and launch checklist requirements satisfied; issue can be closed"


def test_render_pilot_portfolio_report_summarizes_commercial_readiness():
    portfolio = PilotPortfolio(
        name="Design Partners",
        period="2026-H1",
        scorecards=[
            PilotScorecard(
                issue_id="OPE-60",
                customer="Partner A",
                period="2026-Q2",
                metrics=[PilotMetric(name="Coverage", baseline=40, current=85, target=80, unit="%")],
                monthly_benefit=15000,
                monthly_cost=3000,
                implementation_cost=18000,
                benchmark_score=97,
                benchmark_passed=True,
            ),
            PilotScorecard(
                issue_id="OPE-61",
                customer="Partner B",
                period="2026-Q2",
                metrics=[PilotMetric(name="Cycle time", baseline=12, current=7, target=5, unit="h", higher_is_better=False)],
                monthly_benefit=9000,
                monthly_cost=2500,
                implementation_cost=12000,
                benchmark_score=88,
                benchmark_passed=True,
            ),
        ],
    )

    content = render_pilot_portfolio_report(portfolio)

    assert portfolio.total_monthly_net_value == 18500
    assert portfolio.average_roi == 195.2
    assert portfolio.recommendation_counts == {"go": 1, "iterate": 1, "hold": 0}
    assert portfolio.recommendation == "continue"
    assert "Recommendation Mix: go=1 iterate=1 hold=0" in content
    assert "Partner A: recommendation=go" in content
    assert "Partner B: recommendation=iterate" in content


def test_render_shared_view_context_includes_collaboration_annotations():
    view = SharedViewContext(
        filters=[SharedViewFilter(label="Team", value="ops")],
        result_count=4,
        collaboration=build_collaboration_thread(
            "dashboard",
            "ops-overview",
            comments=[
                CollaborationComment(
                    comment_id="dashboard-comment-1",
                    author="pm",
                    body="Please review blocker copy with @ops and @eng.",
                    mentions=["ops", "eng"],
                    anchor="blockers",
                )
            ],
            decisions=[
                DecisionNote(
                    decision_id="dashboard-decision-1",
                    author="ops",
                    outcome="approved",
                    summary="Keep the blocker module visible for managers.",
                    mentions=["pm"],
                    follow_up="Recheck after next data refresh.",
                )
            ],
        ),
    )

    lines = render_shared_view_context(view)
    content = "\n".join(lines)

    assert "## Collaboration" in content
    assert "Surface: dashboard" in content
    assert "Please review blocker copy with @ops and @eng." in content
    assert "Keep the blocker module visible for managers." in content


def test_auto_triage_center_prioritizes_failed_and_pending_runs():
    approval_task = Task(task_id="OPE-76-risk", source="linear", title="Prod approval", description="")
    approval_run = TaskRun.from_task(approval_task, run_id="run-risk", medium="vm")
    approval_run.trace("scheduler.decide", "pending")
    approval_run.audit(
        "scheduler.decision",
        "scheduler",
        "pending",
        reason="requires approval for high-risk task",
    )
    approval_run.finalize("needs-approval", "requires approval for high-risk task")

    failed_task = Task(task_id="OPE-76-browser", source="linear", title="Replay browser task", description="")
    failed_run = TaskRun.from_task(failed_task, run_id="run-browser", medium="browser")
    failed_run.trace("runtime.execute", "failed")
    failed_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
    failed_run.finalize("failed", "browser session crashed")

    healthy_task = Task(task_id="OPE-76-ok", source="linear", title="Healthy run", description="")
    healthy_run = TaskRun.from_task(healthy_task, run_id="run-ok", medium="docker")
    healthy_run.trace("scheduler.decide", "ok")
    healthy_run.audit("scheduler.decision", "scheduler", "approved", reason="default low risk path")
    healthy_run.finalize("approved", "default low risk path")

    center = build_auto_triage_center(
        [healthy_run, approval_run, failed_run],
        name="Engineering Ops",
        period="2026-03-10",
    )
    report = render_auto_triage_center_report(center, total_runs=3)

    assert center.flagged_runs == 2
    assert center.inbox_size == 2
    assert center.severity_counts == {"critical": 1, "high": 1, "medium": 0}
    assert center.owner_counts == {"security": 1, "engineering": 1, "operations": 0}
    assert center.recommendation == "immediate-attention"
    assert [finding.run_id for finding in center.findings] == ["run-browser", "run-risk"]
    assert [item.run_id for item in center.inbox] == ["run-browser", "run-risk"]
    assert center.inbox[0].suggestions[0].label == "replay candidate"
    assert center.inbox[0].suggestions[0].confidence >= 0.55
    assert center.findings[0].next_action == "replay run and inspect tool failures"
    assert center.findings[1].next_action == "request approval and queue security review"
    assert center.findings[0].actions[4].enabled is True
    assert center.findings[1].actions[4].enabled is False
    assert center.findings[1].actions[6].enabled is False
    assert "Flagged Runs: 2" in report
    assert "Inbox Size: 2" in report
    assert "Severity Mix: critical=1 high=1 medium=0" in report
    assert "Feedback Loop: accepted=0 rejected=0 pending=2" in report
    assert "run-browser: severity=critical owner=engineering status=failed" in report
    assert "run-risk: severity=high owner=security status=needs-approval" in report
    assert "actions=Drill Down [drill-down]" in report
    assert "Retry [retry] state=disabled target=run-risk reason=retry available after owner review" in report

def test_auto_triage_center_report_renders_shared_view_partial_state():
    task = Task(task_id="OPE-94-risk", source="linear", title="Prod approval", description="")
    run = TaskRun.from_task(task, run_id="run-risk", medium="vm")
    run.audit("scheduler.decision", "scheduler", "pending", reason="requires approval for high-risk task")
    run.finalize("needs-approval", "requires approval for high-risk task")

    center = build_auto_triage_center([run], name="Engineering Ops", period="2026-03-10")
    report = render_auto_triage_center_report(
        center,
        total_runs=1,
        view=make_shared_view(1, partial_data=["Replay ledger data is still backfilling."]),
    )

    assert "## View State" in report
    assert "- State: partial-data" in report
    assert "- Team: engineering" in report
    assert "## Partial Data" in report
    assert "Replay ledger data is still backfilling." in report


def test_auto_triage_center_builds_similarity_evidence_and_feedback_loop():
    failed_browser_task = Task(task_id="OPE-100-browser-a", source="linear", title="Browser replay failure", description="")
    failed_browser_run = TaskRun.from_task(failed_browser_task, run_id="run-browser-a", medium="browser")
    failed_browser_run.trace("runtime.execute", "failed")
    failed_browser_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
    failed_browser_run.finalize("failed", "browser session crashed")

    similar_browser_task = Task(task_id="OPE-100-browser-b", source="linear", title="Browser replay failure", description="")
    similar_browser_run = TaskRun.from_task(similar_browser_task, run_id="run-browser-b", medium="browser")
    similar_browser_run.trace("runtime.execute", "failed")
    similar_browser_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
    similar_browser_run.finalize("failed", "browser session crashed")

    approval_task = Task(task_id="OPE-100-security", source="linear", title="Security approval", description="")
    approval_run = TaskRun.from_task(approval_task, run_id="run-security", medium="vm")
    approval_run.trace("scheduler.decide", "pending")
    approval_run.audit("scheduler.decision", "scheduler", "pending", reason="requires approval for high-risk task")
    approval_run.finalize("needs-approval", "requires approval for high-risk task")

    feedback = [
        TriageFeedbackRecord(
            run_id="run-browser-a",
            action="replay run and inspect tool failures",
            decision="accepted",
            actor="ops-lead",
            notes="matched previous recovery path",
        ),
        TriageFeedbackRecord(
            run_id="run-security",
            action="request approval and queue security review",
            decision="rejected",
            actor="sec-reviewer",
            notes="approval already in flight",
        ),
    ]

    center = build_auto_triage_center(
        [failed_browser_run, similar_browser_run, approval_run],
        name="Auto Triage Center",
        period="2026-03-11",
        feedback=feedback,
    )
    report = render_auto_triage_center_report(center, total_runs=3)

    browser_item = next(item for item in center.inbox if item.run_id == "run-browser-a")
    approval_item = next(item for item in center.inbox if item.run_id == "run-security")

    assert center.feedback_counts == {"accepted": 1, "rejected": 1, "pending": 1}
    assert browser_item.suggestions[0].feedback_status == "accepted"
    assert approval_item.suggestions[0].feedback_status == "rejected"
    assert browser_item.suggestions[0].evidence[0].related_run_id == "run-browser-b"
    assert browser_item.suggestions[0].evidence[0].score >= 0.8
    assert "## Inbox" in report
    assert "run-browser-a: severity=critical owner=engineering status=failed" in report
    assert "similar=run-browser-b:" in report
    assert "Feedback Loop: accepted=1 rejected=1 pending=1" in report


def test_takeover_queue_from_ledger_groups_pending_handoffs():
    entries = [
        {
            "run_id": "run-sec",
            "task_id": "OPE-66-sec",
            "source": "linear",
            "summary": "requires approval for high-risk task",
            "audits": [
                {
                    "action": "orchestration.handoff",
                    "outcome": "pending",
                    "details": {
                        "target_team": "security",
                        "reason": "requires approval for high-risk task",
                        "required_approvals": ["security-review"],
                    },
                }
            ],
        },
        {
            "run_id": "run-ops",
            "task_id": "OPE-66-ops",
            "source": "linear",
            "summary": "premium tier required for advanced cross-department orchestration",
            "audits": [
                {
                    "action": "orchestration.handoff",
                    "outcome": "pending",
                    "details": {
                        "target_team": "operations",
                        "reason": "premium tier required for advanced cross-department orchestration",
                        "required_approvals": ["ops-manager"],
                    },
                }
            ],
        },
        {
            "run_id": "run-ok",
            "task_id": "OPE-66-ok",
            "source": "linear",
            "summary": "default low risk path",
            "audits": [
                {"action": "scheduler.decision", "outcome": "approved", "details": {"reason": "default low risk path"}}
            ],
        },
    ]

    queue = build_takeover_queue_from_ledger(entries, name="Cross-Team Takeovers", period="2026-03-10")
    report = render_takeover_queue_report(queue, total_runs=3)

    assert queue.pending_requests == 2
    assert queue.team_counts == {"operations": 1, "security": 1}
    assert queue.approval_count == 2
    assert queue.recommendation == "expedite-security-review"
    assert [request.run_id for request in queue.requests] == ["run-ops", "run-sec"]
    assert queue.requests[0].actions[3].enabled is True
    assert queue.requests[1].actions[3].enabled is False
    assert "Pending Requests: 2" in report
    assert "Team Mix: operations=1 security=1" in report
    assert "run-sec: team=security status=pending task=OPE-66-sec approvals=security-review" in report
    assert "run-ops: team=operations status=pending task=OPE-66-ops approvals=ops-manager" in report
    assert "Escalate [escalate] state=disabled target=run-sec reason=security takeovers are already escalated" in report


def test_takeover_queue_report_renders_shared_view_error_state():
    queue = build_takeover_queue_from_ledger([], name="Cross-Team Takeovers", period="2026-03-10")
    report = render_takeover_queue_report(
        queue,
        total_runs=0,
        view=make_shared_view(0, errors=["Takeover approvals service timed out."]),
    )

    assert "- State: error" in report
    assert "- Summary: Unable to load data for the current filters." in report
    assert "## Errors" in report
    assert "Takeover approvals service timed out." in report


def test_orchestration_canvas_summarizes_policy_and_handoff():
    task = Task(task_id="OPE-66-canvas", source="linear", title="Canvas run", description="")
    run = TaskRun.from_task(task, run_id="run-canvas", medium="browser")
    run.audit("tool.invoke", "worker", "success", tool="browser")

    from bigclaw.orchestration import DepartmentHandoff, HandoffRequest, OrchestrationPlan, OrchestrationPolicyDecision

    plan = OrchestrationPlan(
        task_id="OPE-66-canvas",
        collaboration_mode="tier-limited",
        handoffs=[
            DepartmentHandoff("operations", "coordinate"),
            DepartmentHandoff("engineering", "execute", required_tools=["browser"]),
        ],
    )
    policy = OrchestrationPolicyDecision(
        tier="standard",
        upgrade_required=True,
        reason="premium tier required for advanced cross-department orchestration",
        blocked_departments=["customer-success"],
        entitlement_status="upgrade-required",
        billing_model="standard-blocked",
        estimated_cost_usd=7.0,
        included_usage_units=2,
        overage_usage_units=1,
        overage_cost_usd=4.0,
    )
    handoff = HandoffRequest(target_team="operations", reason=policy.reason, required_approvals=["ops-manager"])

    canvas = build_orchestration_canvas(run, plan, policy, handoff)
    report = render_orchestration_canvas(canvas)

    assert isinstance(canvas, OrchestrationCanvas)
    assert canvas.recommendation == "resolve-entitlement-gap"
    assert canvas.active_tools == ["browser"]
    assert canvas.actions[3].enabled is True
    assert canvas.actions[4].enabled is False
    assert "# Orchestration Canvas" in report
    assert "- Tier: standard" in report
    assert "- Entitlement Status: upgrade-required" in report
    assert "- Billing Model: standard-blocked" in report
    assert "- Estimated Cost (USD): 7.00" in report
    assert "- Handoff Team: operations" in report
    assert "- Recommendation: resolve-entitlement-gap" in report
    assert "## Actions" in report
    assert "Escalate [escalate] state=enabled target=run-canvas" in report


def test_orchestration_canvas_reconstructs_flow_collaboration_from_ledger():
    entry = {
        "run_id": "run-flow-1",
        "task_id": "OPE-113",
        "audits": [
            {
                "action": "orchestration.plan",
                "actor": "scheduler",
                "outcome": "enabled",
                "timestamp": "2026-03-11T11:00:00Z",
                "details": {
                    "collaboration_mode": "cross-functional",
                    "departments": ["operations", "engineering"],
                    "approvals": [],
                },
            },
            {
                "action": "orchestration.policy",
                "actor": "scheduler",
                "outcome": "enabled",
                "timestamp": "2026-03-11T11:01:00Z",
                "details": {
                    "tier": "premium",
                    "entitlement_status": "included",
                    "billing_model": "premium-included",
                },
            },
            {
                "action": "collaboration.comment",
                "actor": "ops-lead",
                "outcome": "recorded",
                "timestamp": "2026-03-11T11:02:00Z",
                "details": {
                    "surface": "flow",
                    "comment_id": "flow-comment-1",
                    "body": "Route @eng once the dashboard note is resolved.",
                    "mentions": ["eng"],
                    "anchor": "handoff-lane",
                    "status": "open",
                },
            },
            {
                "action": "collaboration.decision",
                "actor": "eng-manager",
                "outcome": "accepted",
                "timestamp": "2026-03-11T11:03:00Z",
                "details": {
                    "surface": "flow",
                    "decision_id": "flow-decision-1",
                    "summary": "Engineering owns the next flow handoff.",
                    "mentions": ["ops-lead"],
                    "related_comment_ids": ["flow-comment-1"],
                    "follow_up": "Post in the shared channel after deploy.",
                },
            },
        ],
    }

    canvas = build_orchestration_canvas_from_ledger_entry(entry)
    report = render_orchestration_canvas(canvas)

    assert canvas.collaboration is not None
    assert canvas.recommendation == "resolve-flow-comments"
    assert "## Collaboration" in report
    assert "Route @eng once the dashboard note is resolved." in report
    assert "Engineering owns the next flow handoff." in report


def test_orchestration_portfolio_rolls_up_canvas_and_takeover_state():
    canvases = [
        OrchestrationCanvas(
            task_id="OPE-66-a",
            run_id="run-a",
            collaboration_mode="cross-functional",
            departments=["operations", "engineering", "security"],
            tier="premium",
            entitlement_status="included",
            billing_model="premium-included",
            estimated_cost_usd=4.5,
            included_usage_units=3,
            handoff_team="security",
            handoff_status="pending",
        ),
        OrchestrationCanvas(
            task_id="OPE-66-b",
            run_id="run-b",
            collaboration_mode="tier-limited",
            departments=["operations", "engineering"],
            tier="standard",
            upgrade_required=True,
            entitlement_status="upgrade-required",
            billing_model="standard-blocked",
            estimated_cost_usd=7.0,
            included_usage_units=2,
            overage_usage_units=1,
            overage_cost_usd=4.0,
            blocked_departments=["customer-success"],
            handoff_team="operations",
            handoff_status="pending",
        ),
    ]
    queue = build_takeover_queue_from_ledger(
        [
            {
                "run_id": "run-a",
                "task_id": "OPE-66-a",
                "source": "linear",
                "audits": [
                    {
                        "action": "orchestration.handoff",
                        "outcome": "pending",
                        "details": {"target_team": "security", "reason": "risk", "required_approvals": ["security-review"]},
                    }
                ],
            },
            {
                "run_id": "run-b",
                "task_id": "OPE-66-b",
                "source": "linear",
                "audits": [
                    {
                        "action": "orchestration.handoff",
                        "outcome": "pending",
                        "details": {"target_team": "operations", "reason": "entitlement", "required_approvals": ["ops-manager"]},
                    }
                ],
            },
        ],
        name="Cross-Team Takeovers",
        period="2026-03-10",
    )

    portfolio = build_orchestration_portfolio(
        canvases,
        name="Cross-Team Portfolio",
        period="2026-03-10",
        takeover_queue=queue,
    )
    report = render_orchestration_portfolio_report(portfolio)

    assert isinstance(portfolio, OrchestrationPortfolio)
    assert portfolio.total_runs == 2
    assert portfolio.collaboration_modes == {"cross-functional": 1, "tier-limited": 1}
    assert portfolio.tier_counts == {"premium": 1, "standard": 1}
    assert portfolio.entitlement_counts == {"included": 1, "upgrade-required": 1}
    assert portfolio.billing_model_counts == {"premium-included": 1, "standard-blocked": 1}
    assert portfolio.total_estimated_cost_usd == 11.5
    assert portfolio.total_overage_cost_usd == 4.0
    assert portfolio.upgrade_required_count == 1
    assert portfolio.active_handoffs == 2
    assert portfolio.recommendation == "stabilize-security-takeovers"
    assert "# Orchestration Portfolio Report" in report
    assert "- Collaboration Mix: cross-functional=1 tier-limited=1" in report
    assert "- Tier Mix: premium=1 standard=1" in report
    assert "- Entitlement Mix: included=1 upgrade-required=1" in report
    assert "- Billing Models: premium-included=1 standard-blocked=1" in report
    assert "- Estimated Cost (USD): 11.50" in report
    assert "- Overage Cost (USD): 4.00" in report
    assert "- Takeover Queue: pending=2 recommendation=expedite-security-review" in report
    assert "- run-a: mode=cross-functional tier=premium entitlement=included billing=premium-included estimated_cost_usd=4.50 overage_cost_usd=0.00 upgrade_required=False handoff=security" in report
    assert "actions=Drill Down [drill-down]" in report


def test_orchestration_portfolio_report_renders_shared_view_empty_state():
    portfolio = build_orchestration_portfolio([], name="Cross-Team Portfolio", period="2026-03-10")
    report = render_orchestration_portfolio_report(
        portfolio,
        view=make_shared_view(0),
    )

    assert "- State: empty" in report
    assert "- Summary: No records match the current filters." in report
    assert "## Filters" in report


def test_render_orchestration_overview_page():
    portfolio = OrchestrationPortfolio(
        name="Cross-Team Portfolio",
        period="2026-03-10",
        canvases=[
            OrchestrationCanvas(
                task_id="OPE-66-a",
                run_id="run-a",
                collaboration_mode="cross-functional",
                departments=["operations", "engineering"],
                tier="premium",
                entitlement_status="included",
                billing_model="premium-included",
                estimated_cost_usd=3.0,
                handoff_team="security",
            )
        ],
        takeover_queue=build_takeover_queue_from_ledger(
            [
                {
                    "run_id": "run-a",
                    "task_id": "OPE-66-a",
                    "source": "linear",
                    "audits": [
                        {
                            "action": "orchestration.handoff",
                            "outcome": "pending",
                            "details": {"target_team": "security", "reason": "risk", "required_approvals": ["security-review"]},
                        }
                    ],
                }
            ],
            name="Cross-Team Takeovers",
            period="2026-03-10",
        ),
    )

    page = render_orchestration_overview_page(portfolio)

    assert "<title>Orchestration Overview" in page
    assert "Cross-Team Portfolio" in page
    assert "review-security-takeover" in page
    assert "Estimated Cost" in page
    assert "premium-included" in page
    assert "pending=1 recommendation=expedite-security-review" in page
    assert "run-a" in page
    assert "actions=Drill Down [drill-down]" in page


def test_build_orchestration_canvas_from_ledger_entry_extracts_audit_state():
    entry = {
        "run_id": "run-ledger",
        "task_id": "OPE-66-ledger",
        "audits": [
            {
                "action": "orchestration.plan",
                "outcome": "ready",
                "details": {
                    "collaboration_mode": "tier-limited",
                    "departments": ["operations", "engineering"],
                    "approvals": ["security-review"],
                },
            },
            {
                "action": "orchestration.policy",
                "outcome": "upgrade-required",
                "details": {
                    "tier": "standard",
                    "entitlement_status": "upgrade-required",
                    "billing_model": "standard-blocked",
                    "estimated_cost_usd": 7.0,
                    "included_usage_units": 2,
                    "overage_usage_units": 1,
                    "overage_cost_usd": 4.0,
                    "blocked_departments": ["security", "customer-success"],
                },
            },
            {
                "action": "orchestration.handoff",
                "outcome": "pending",
                "details": {
                    "target_team": "operations",
                    "reason": "premium tier required for advanced cross-department orchestration",
                },
            },
            {"action": "tool.invoke", "outcome": "success", "details": {"tool": "browser"}},
        ],
    }

    canvas = build_orchestration_canvas_from_ledger_entry(entry)

    assert canvas.run_id == "run-ledger"
    assert canvas.collaboration_mode == "tier-limited"
    assert canvas.departments == ["operations", "engineering"]
    assert canvas.required_approvals == ["security-review"]
    assert canvas.tier == "standard"
    assert canvas.upgrade_required is True
    assert canvas.entitlement_status == "upgrade-required"
    assert canvas.billing_model == "standard-blocked"
    assert canvas.estimated_cost_usd == 7.0
    assert canvas.included_usage_units == 2
    assert canvas.overage_usage_units == 1
    assert canvas.overage_cost_usd == 4.0
    assert canvas.blocked_departments == ["security", "customer-success"]
    assert canvas.handoff_team == "operations"
    assert canvas.active_tools == ["browser"]
    assert canvas.actions[3].enabled is True
    assert canvas.actions[4].enabled is False


def test_build_orchestration_portfolio_from_ledger_rolls_up_entries():
    entries = [
        {
            "run_id": "run-a",
            "task_id": "OPE-66-a",
            "audits": [
                {
                    "action": "orchestration.plan",
                    "outcome": "ready",
                    "details": {
                        "collaboration_mode": "cross-functional",
                        "departments": ["operations", "engineering", "security"],
                        "approvals": ["security-review"],
                    },
                },
                {
                    "action": "orchestration.policy",
                    "outcome": "enabled",
                    "details": {
                        "tier": "premium",
                        "entitlement_status": "included",
                        "billing_model": "premium-included",
                        "estimated_cost_usd": 4.5,
                        "included_usage_units": 3,
                        "blocked_departments": [],
                    },
                },
                {
                    "action": "orchestration.handoff",
                    "outcome": "pending",
                    "details": {"target_team": "security", "reason": "approval required", "required_approvals": ["security-review"]},
                },
            ],
        },
        {
            "run_id": "run-b",
            "task_id": "OPE-66-b",
            "audits": [
                {
                    "action": "orchestration.plan",
                    "outcome": "ready",
                    "details": {
                        "collaboration_mode": "tier-limited",
                        "departments": ["operations", "engineering"],
                        "approvals": [],
                    },
                },
                {
                    "action": "orchestration.policy",
                    "outcome": "upgrade-required",
                    "details": {
                        "tier": "standard",
                        "entitlement_status": "upgrade-required",
                        "billing_model": "standard-blocked",
                        "estimated_cost_usd": 7.0,
                        "included_usage_units": 2,
                        "overage_usage_units": 1,
                        "overage_cost_usd": 4.0,
                        "blocked_departments": ["customer-success"],
                    },
                },
                {
                    "action": "orchestration.handoff",
                    "outcome": "pending",
                    "details": {"target_team": "operations", "reason": "entitlement gap", "required_approvals": ["ops-manager"]},
                },
            ],
        },
    ]

    portfolio = build_orchestration_portfolio_from_ledger(entries, name="Ledger Portfolio", period="2026-03-10")

    assert portfolio.total_runs == 2
    assert portfolio.collaboration_modes == {"cross-functional": 1, "tier-limited": 1}
    assert portfolio.tier_counts == {"premium": 1, "standard": 1}
    assert portfolio.entitlement_counts == {"included": 1, "upgrade-required": 1}
    assert portfolio.total_estimated_cost_usd == 11.5
    assert portfolio.takeover_queue is not None
    assert portfolio.takeover_queue.pending_requests == 2
    assert portfolio.recommendation == "stabilize-security-takeovers"


def test_build_billing_entitlements_page_rolls_up_orchestration_costs():
    portfolio = OrchestrationPortfolio(
        name="Revenue Ops",
        period="2026-03",
        canvases=[
            OrchestrationCanvas(
                task_id="OPE-104-a",
                run_id="run-billing-a",
                collaboration_mode="cross-functional",
                departments=["operations", "engineering", "security"],
                tier="premium",
                entitlement_status="included",
                billing_model="premium-included",
                estimated_cost_usd=4.5,
                included_usage_units=3,
                handoff_team="security",
            ),
            OrchestrationCanvas(
                task_id="OPE-104-b",
                run_id="run-billing-b",
                collaboration_mode="tier-limited",
                departments=["operations", "engineering"],
                tier="standard",
                upgrade_required=True,
                entitlement_status="upgrade-required",
                billing_model="standard-blocked",
                estimated_cost_usd=7.0,
                included_usage_units=2,
                overage_usage_units=1,
                overage_cost_usd=4.0,
                blocked_departments=["customer-success"],
                handoff_team="operations",
            ),
        ],
    )

    page = build_billing_entitlements_page(
        portfolio,
        workspace_name="OpenAGI Revenue Cloud",
        plan_name="Standard",
        billing_period="2026-03",
    )
    report = render_billing_entitlements_report(page)

    assert isinstance(page, BillingEntitlementsPage)
    assert page.run_count == 2
    assert page.total_included_usage_units == 5
    assert page.total_overage_usage_units == 1
    assert page.total_estimated_cost_usd == 11.5
    assert page.total_overage_cost_usd == 4.0
    assert page.upgrade_required_count == 1
    assert page.entitlement_counts == {"included": 1, "upgrade-required": 1}
    assert page.billing_model_counts == {"premium-included": 1, "standard-blocked": 1}
    assert page.blocked_capabilities == ["customer-success"]
    assert page.recommendation == "resolve-plan-gaps"
    assert "# Billing & Entitlements Report" in report
    assert "- Workspace: OpenAGI Revenue Cloud" in report
    assert "- Overage Cost (USD): 4.00" in report
    assert "- run-billing-b: task=OPE-104-b entitlement=upgrade-required billing=standard-blocked" in report


def test_render_billing_entitlements_page_outputs_html_dashboard():
    page = BillingEntitlementsPage(
        workspace_name="OpenAGI Revenue Cloud",
        plan_name="Premium",
        billing_period="2026-03",
        charges=[
            BillingRunCharge(
                run_id="run-billing-a",
                task_id="OPE-104-a",
                entitlement_status="included",
                billing_model="premium-included",
                estimated_cost_usd=4.5,
                included_usage_units=3,
                recommendation="review-security-takeover",
            )
        ],
    )

    page_html = render_billing_entitlements_page(page)

    assert "<title>Billing & Entitlements" in page_html
    assert "OpenAGI Revenue Cloud" in page_html
    assert "Premium plan for 2026-03" in page_html
    assert "Charge Feed" in page_html
    assert "premium-included" in page_html


def test_build_billing_entitlements_page_from_ledger_extracts_upgrade_signals():
    entries = [
        {
            "run_id": "run-ledger-a",
            "task_id": "OPE-104-a",
            "audits": [
                {
                    "action": "orchestration.plan",
                    "outcome": "ready",
                    "details": {
                        "collaboration_mode": "cross-functional",
                        "departments": ["operations", "engineering", "security"],
                        "approvals": ["security-review"],
                    },
                },
                {
                    "action": "orchestration.policy",
                    "outcome": "enabled",
                    "details": {
                        "tier": "premium",
                        "entitlement_status": "included",
                        "billing_model": "premium-included",
                        "estimated_cost_usd": 4.5,
                        "included_usage_units": 3,
                        "blocked_departments": [],
                    },
                },
            ],
        },
        {
            "run_id": "run-ledger-b",
            "task_id": "OPE-104-b",
            "audits": [
                {
                    "action": "orchestration.plan",
                    "outcome": "ready",
                    "details": {
                        "collaboration_mode": "tier-limited",
                        "departments": ["operations", "engineering"],
                        "approvals": [],
                    },
                },
                {
                    "action": "orchestration.policy",
                    "outcome": "upgrade-required",
                    "details": {
                        "tier": "standard",
                        "entitlement_status": "upgrade-required",
                        "billing_model": "standard-blocked",
                        "estimated_cost_usd": 7.0,
                        "included_usage_units": 2,
                        "overage_usage_units": 1,
                        "overage_cost_usd": 4.0,
                        "blocked_departments": ["customer-success"],
                    },
                },
                {
                    "action": "orchestration.handoff",
                    "outcome": "pending",
                    "details": {"target_team": "operations", "reason": "entitlement gap", "required_approvals": ["ops-manager"]},
                },
            ],
        },
    ]

    page = build_billing_entitlements_page_from_ledger(
        entries,
        workspace_name="OpenAGI Revenue Cloud",
        plan_name="Standard",
        billing_period="2026-03",
    )

    assert page.run_count == 2
    assert page.recommendation == "resolve-plan-gaps"
    assert page.total_overage_cost_usd == 4.0
    assert page.charges[1].blocked_capabilities == ["customer-success"]
    assert page.charges[1].handoff_team == "operations"


def test_triage_feedback_record_uses_timezone_aware_utc_timestamp():
    record = TriageFeedbackRecord(run_id="run-1", action="classify", decision="accepted", actor="ops")

    assert record.timestamp.endswith("Z")
    parsed = __import__("datetime").datetime.fromisoformat(record.timestamp.replace("Z", "+00:00"))
    assert parsed.tzinfo is not None
    assert parsed.utcoffset().total_seconds() == 0


def test_issue_validation_report_uses_timezone_aware_utc_timestamp():
    content = render_issue_validation_report("BIG-900", "v1", "repo", "pass")

    timestamp_line = next(line for line in content.splitlines() if line.startswith("- 生成时间:"))
    timestamp_value = timestamp_line.split(": ", 1)[1]
    assert timestamp_value.endswith("Z")
    parsed = __import__("datetime").datetime.fromisoformat(timestamp_value.replace("Z", "+00:00"))
    assert parsed.tzinfo is not None
    assert parsed.utcoffset().total_seconds() == 0


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


def test_reports_accept_canonical_handoff_and_takeover_events() -> None:
    entry = {
        "run_id": "run-ope-134-canvas",
        "task_id": "OPE-134-canvas",
        "source": "linear",
        "summary": "handoff requested",
        "audits": [
            {
                "action": "orchestration.plan",
                "actor": "scheduler",
                "outcome": "ready",
                "details": {
                    "collaboration_mode": "cross-functional",
                    "departments": ["operations", "engineering"],
                    "approvals": ["security-review"],
                },
            },
            {
                "action": MANUAL_TAKEOVER_EVENT,
                "actor": "scheduler",
                "outcome": "pending",
                "details": {
                    "task_id": "OPE-134-canvas",
                    "run_id": "run-ope-134-canvas",
                    "target_team": "security",
                    "reason": "manual review required",
                    "requested_by": "scheduler",
                    "required_approvals": ["security-review"],
                },
            },
        ],
    }

    canvas = build_orchestration_canvas_from_ledger_entry(entry)
    queue = build_takeover_queue_from_ledger([entry], period="2026-03-11")

    assert canvas.handoff_team == "security"
    assert queue.requests[0].required_approvals == ["security-review"]

def test_cross_department_orchestrator_routes_security_data_and_customer_work() -> None:
    task = Task(
        task_id="OPE-66",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer", "premium"],
        priority=Priority.P0,
        risk_level=RiskLevel.HIGH,
        required_tools=["browser", "sql"],
        acceptance_criteria=["approval recorded"],
        validation_plan=["customer signoff"],
    )

    plan = CrossDepartmentOrchestrator().plan(task)

    assert plan.collaboration_mode == "cross-functional"
    assert plan.departments == ["operations", "engineering", "security", "data", "customer-success"]
    assert plan.required_approvals == ["security-review"]


def test_standard_policy_limits_advanced_cross_department_routing() -> None:
    task = Task(
        task_id="OPE-66-standard",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer"],
        required_tools=["browser", "sql"],
        risk_level=RiskLevel.HIGH,
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)

    assert plan.collaboration_mode == "tier-limited"
    assert plan.departments == ["operations", "engineering"]
    assert policy.upgrade_required is True
    assert policy.entitlement_status == "upgrade-required"
    assert policy.billing_model == "standard-blocked"
    assert policy.included_usage_units == 2
    assert policy.overage_usage_units == 3
    assert policy.overage_cost_usd == 12.0
    assert policy.estimated_cost_usd == 15.0
    assert policy.blocked_departments == ["security", "data", "customer-success"]


def test_render_orchestration_plan_lists_handoffs_and_policy() -> None:
    task = Task(
        task_id="OPE-66-render",
        source="jira",
        title="Warehouse rollout",
        description="Customer-ready release",
        labels=["data", "customer"],
        required_tools=["sql"],
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
    content = render_orchestration_plan(plan, policy)

    assert "# Cross-Department Orchestration Plan" in content
    assert "- Departments: operations, engineering" in content
    assert "- Tier: standard" in content
    assert "- Entitlement Status: upgrade-required" in content
    assert "- Billing Model: standard-blocked" in content
    assert "- Estimated Cost (USD): 11.00" in content
    assert "- Blocked Departments: data, customer-success" in content
    assert "- Human Handoff Team:" not in content


def test_scheduler_execution_records_orchestration_plan_and_policy(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-exec",
        source="linear",
        title="Cross-team browser change",
        description="Program-managed rollout",
        labels=["ops"],
        priority=Priority.P0,
        risk_level=RiskLevel.MEDIUM,
        required_tools=["browser"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66", ledger=ledger)
    entry = ledger.load()[0]

    assert record.orchestration_plan is not None
    assert record.orchestration_policy is not None
    assert record.orchestration_plan.departments == ["operations", "engineering"]
    assert record.orchestration_policy.upgrade_required is False
    assert record.orchestration_policy.entitlement_status == "included"
    assert record.orchestration_policy.billing_model == "standard-included"
    assert record.orchestration_policy.estimated_cost_usd == 3.0
    assert any(trace["span"] == "orchestration.plan" for trace in entry["traces"])
    assert any(trace["span"] == "orchestration.policy" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.plan" for audit in entry["audits"])
    assert any(audit["action"] == "orchestration.policy" for audit in entry["audits"])
    policy_audit = next(audit for audit in entry["audits"] if audit["action"] == "orchestration.policy")
    assert policy_audit["details"]["entitlement_status"] == "included"
    assert policy_audit["details"]["billing_model"] == "standard-included"


def test_scheduler_creates_handoff_for_policy_or_approval_blockers(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-handoff",
        source="linear",
        title="Customer analytics rollout",
        description="Need cross-team coordination",
        labels=["customer", "data"],
        required_tools=["browser", "sql"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66-handoff", ledger=ledger)
    entry = ledger.load()[0]

    assert record.handoff_request is not None
    assert record.handoff_request.target_team == "operations"
    assert any(trace["span"] == "orchestration.handoff" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.handoff" for audit in entry["audits"])


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


def test_scheduler_emits_p0_operational_audit_events(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-134-scheduler",
        source="linear",
        title="Route cross-team rollout",
        description="Needs coordinated release handling",
        labels=["customer", "data"],
        priority=Priority.P0,
        required_tools=["browser", "sql"],
        budget=120.0,
        budget_override_actor="finance-controller",
        budget_override_reason="approved additional analytics validation spend",
        budget_override_amount=30.0,
    )

    record = Scheduler().execute(task, run_id="run-ope-134-scheduler", ledger=ledger)
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}

    assert record.handoff_request is not None
    assert audits[SCHEDULER_DECISION_EVENT]["details"]["risk_score"] >= 0
    assert audits[BUDGET_OVERRIDE_EVENT]["details"] == {
        "task_id": "OPE-134-scheduler",
        "run_id": "run-ope-134-scheduler",
        "requested_budget": 120.0,
        "approved_budget": 150.0,
        "override_actor": "finance-controller",
        "reason": "approved additional analytics validation spend",
    }
    assert audits[MANUAL_TAKEOVER_EVENT]["details"]["target_team"] == "operations"
    assert audits[FLOW_HANDOFF_EVENT]["details"]["source_stage"] == "scheduler"


def test_workflow_records_canonical_approval_event(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-134-approval",
        source="linear",
        title="Approve production rollout",
        description="Manual gate",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )

    WorkflowEngine().run(
        task,
        run_id="run-ope-134-approval",
        ledger=ledger,
        approvals=["security-review"],
        validation_evidence=["rollback-plan", "integration-test"],
    )

    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}
    assert audits[APPROVAL_RECORDED_EVENT]["details"] == {
        "task_id": "OPE-134-approval",
        "run_id": "run-ope-134-approval",
        "approvals": ["security-review"],
        "approval_count": 1,
        "acceptance_status": "accepted",
    }

def test_task_run_captures_logs_trace_artifacts_and_audits(tmp_path: Path):
    artifact = tmp_path / "validation.md"
    artifact.write_text("validation ok")
    expected_digest = hashlib.sha256(artifact.read_bytes()).hexdigest()

    task = Task(
        task_id="BIG-502",
        source="linear",
        title="Add observability",
        description="full chain",
        priority=Priority.P0,
    )
    run = TaskRun.from_task(task, run_id="run-1", medium="docker")
    run.log("info", "task accepted", queue="primary")
    run.trace("scheduler.decide", "ok", approved=True)
    run.register_artifact("validation-report", "report", str(artifact), environment="sandbox")
    run.audit("scheduler.approved", "system", "success", reason="default low risk path")
    run.record_closeout(
        validation_evidence=["pytest", "validation-report"],
        git_push_succeeded=True,
        git_push_output="Everything up-to-date",
        git_log_stat_output="commit abc123\n 1 file changed, 2 insertions(+)",
    )
    run.finalize("succeeded", "validation passed")

    ledger = ObservabilityLedger(str(tmp_path / "observability.json"))
    ledger.append(run)
    entries = ledger.load()

    assert len(entries) == 1
    assert entries[0]["status"] == "succeeded"
    assert entries[0]["logs"][0]["context"]["queue"] == "primary"
    assert entries[0]["traces"][0]["attributes"]["approved"] is True
    assert entries[0]["artifacts"][0]["sha256"] == expected_digest
    actions = [item["action"] for item in entries[0]["audits"]]
    assert "artifact.registered" in actions
    assert "closeout.recorded" in actions
    assert "scheduler.approved" in actions
    assert entries[0]["closeout"]["complete"] is True


def test_task_run_closeout_serializes_repo_sync_audit(tmp_path: Path):
    task = Task(task_id="BIG-sync", source="linear", title="Repo sync closeout", description="")
    run = TaskRun.from_task(task, run_id="run-sync", medium="docker")
    repo_sync_audit = RepoSyncAudit(
        sync=GitSyncTelemetry(
            status="failed",
            failure_category="dirty",
            summary="worktree has local changes",
            branch="feature/OPE-219",
            remote_ref="origin/feature/OPE-219",
            dirty_paths=["src/bigclaw/workflow.py"],
        ),
        pull_request=PullRequestFreshness(
            pr_number=219,
            pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
            branch_state="out-of-sync",
            body_state="drifted",
            branch_head_sha="abc123",
            pr_head_sha="def456",
            expected_body_digest="body-expected",
            actual_body_digest="body-actual",
        ),
    )
    run.record_closeout(
        validation_evidence=["pytest"],
        git_push_succeeded=False,
        git_push_output="push rejected",
        git_log_stat_output="commit abc123\n 1 file changed, 2 insertions(+)",
        repo_sync_audit=repo_sync_audit,
    )

    ledger = ObservabilityLedger(str(tmp_path / "observability.json"))
    ledger.append(run)
    loaded_run = ledger.load_runs()[0]

    assert loaded_run.closeout.repo_sync_audit is not None
    assert loaded_run.closeout.repo_sync_audit.sync.failure_category == "dirty"
    assert loaded_run.closeout.repo_sync_audit.pull_request.body_state == "drifted"


def test_render_task_run_report(tmp_path: Path):
    artifact = tmp_path / "artifact.txt"
    artifact.write_text("audit trail")

    task = Task(task_id="BIG-502", source="linear", title="Observe execution", description="")
    run = TaskRun.from_task(task, run_id="run-2", medium="vm")
    run.log("warn", "approval required")
    run.trace("risk.review", "pending")
    run.register_artifact("approval-note", "note", str(artifact))
    run.audit("risk.review", "reviewer", "approved")
    comment = run.add_comment(
        author="ops-lead",
        body="Need @security sign-off before we clear this run.",
        mentions=["security"],
        anchor="closeout",
    )
    run.add_decision_note(
        author="security-reviewer",
        summary="Approved release after manual review.",
        outcome="approved",
        mentions=["ops-lead"],
        related_comment_ids=[comment.comment_id],
        follow_up="Share decision in the weekly review.",
    )
    run.record_closeout(
        validation_evidence=["pytest"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit def456\n 1 file changed, 3 insertions(+)",
    )
    run.finalize("completed", "manual approval granted")

    report = render_task_run_report(run)

    assert "Run ID: run-2" in report
    assert "## Logs" in report
    assert "## Trace" in report
    assert "## Artifacts" in report
    assert "## Audit" in report
    assert "## Closeout" in report
    assert "Git Push Succeeded: True" in report
    assert "## Actions" in report
    assert "Retry [retry] state=disabled target=run-2 reason=retry is available for failed or approval-blocked runs" in report
    assert "## Collaboration" in report
    assert "Need @security sign-off before we clear this run." in report
    assert "Approved release after manual review." in report


def test_render_repo_sync_audit_report():
    audit = RepoSyncAudit(
        sync=GitSyncTelemetry(
            status="failed",
            failure_category="auth",
            summary="github token expired",
            branch="dcjcloud/ope-219",
            remote_ref="origin/dcjcloud/ope-219",
            auth_target="github.com/OpenAGIs/BigClaw.git",
        ),
        pull_request=PullRequestFreshness(
            pr_number=219,
            pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
            branch_state="in-sync",
            body_state="drifted",
            branch_head_sha="abc123",
            pr_head_sha="abc123",
            expected_body_digest="expected",
            actual_body_digest="actual",
        ),
    )

    report = render_repo_sync_audit_report(audit)

    assert "# Repo Sync Audit" in report
    assert "Failure Category: auth" in report
    assert "Branch State: in-sync" in report
    assert "Body State: drifted" in report
    assert "sync=failed, failure=auth, pr-branch=in-sync, pr-body=drifted" in report


def test_render_task_run_detail_page(tmp_path: Path):
    artifact = tmp_path / "artifact.txt"
    artifact.write_text("audit trail")

    task = Task(task_id="BIG-502", source="linear", title="Observe execution", description="")
    run = TaskRun.from_task(task, run_id="run-3", medium="browser")
    run.log("info", "opened detail page")
    run.trace("playback.render", "ok")
    run.register_artifact("approval-note", "note", str(artifact))
    run.audit("playback.render", "reviewer", "success")
    run.add_comment(
        author="pm",
        body="Loop in @design before we publish the replay.",
        mentions=["design"],
        anchor="overview",
    )
    run.add_decision_note(
        author="design",
        summary="Replay copy approved for external review.",
        outcome="approved",
        mentions=["pm"],
    )
    run.record_closeout(
        validation_evidence=["pytest", "playback-smoke"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit fedcba\n 1 file changed, 1 insertion(+)",
        run_commit_links=[
            RunCommitLink(run_id="run-3", commit_hash="abc111", role="candidate", repo_space_id="space-1"),
            RunCommitLink(run_id="run-3", commit_hash="fedcba", role="accepted", repo_space_id="space-1"),
        ],
    )
    run.finalize("approved", "detail page ready")

    page = render_task_run_detail_page(run)

    assert "<title>Task Run Detail" in page
    assert "Timeline / Log Sync" in page
    assert "data-detail=\"title\"" in page
    assert "Reports" in page
    assert "opened detail page" in page
    assert "playback.render" in page
    assert str(artifact) in page
    assert "detail page ready" in page
    assert "Closeout" in page
    assert "complete" in page
    assert "Repo Evidence" in page
    assert "fedcba" in page
    assert "Actions" in page
    assert "Pause [pause] state=disabled target=run-3 reason=completed or failed runs cannot be paused" in page
    assert "Collaboration" in page
    assert "Loop in @design before we publish the replay." in page
    assert "Replay copy approved for external review." in page


def test_render_task_run_detail_page_escapes_timeline_json_script_breakout():
    task = Task(task_id="BIG-escape", source="linear", title="Escape check", description="")
    run = TaskRun.from_task(task, run_id="run-escape", medium="browser")
    run.log("info", "contains </script> marker")
    run.finalize("approved", "ok")

    page = render_task_run_detail_page(run)

    assert "contains <\\/script> marker" in page


def test_observability_ledger_load_runs_round_trips_entries(tmp_path: Path):
    task = Task(task_id="BIG-502-roundtrip", source="linear", title="Round trip", description="")
    run = TaskRun.from_task(task, run_id="run-roundtrip", medium="docker")
    run.log("info", "persisted")
    run.trace("scheduler.decide", "ok")
    run.audit("scheduler.decision", "scheduler", "approved", reason="default low risk path")
    run.add_comment(
        author="ops",
        body="Need @eng confirmation on the retry plan.",
        mentions=["eng"],
        anchor="timeline",
    )
    run.finalize("approved", "default low risk path")

    ledger = ObservabilityLedger(str(tmp_path / "observability.json"))
    ledger.append(run)

    loaded_runs = ledger.load_runs()

    assert len(loaded_runs) == 1
    assert loaded_runs[0].run_id == "run-roundtrip"
    assert loaded_runs[0].logs[0].message == "persisted"
    assert loaded_runs[0].traces[0].span == "scheduler.decide"
    assert loaded_runs[0].audits[0].details["reason"] == "default low risk path"
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in loaded_runs[0].audits],
        surface="run",
        target_id=loaded_runs[0].run_id,
    )
    assert collaboration is not None
    assert collaboration.mention_count == 1
    assert collaboration.comments[0].body == "Need @eng confirmation on the retry plan."


def test_event_bus_pr_comment_approves_waiting_run_and_persists_ledger(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-pr", source="github", title="PR approval", description="")
    run = TaskRun.from_task(task, run_id="run-pr-1", medium="vm")
    run.finalize("needs-approval", "waiting for reviewer comment")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    seen_statuses: list[str] = []
    bus.subscribe(PULL_REQUEST_COMMENT_EVENT, lambda _event, current: seen_statuses.append(current.status))

    updated = bus.publish(
        BusEvent(
            event_type=PULL_REQUEST_COMMENT_EVENT,
            run_id=run.run_id,
            actor="reviewer",
            details={
                "decision": "approved",
                "body": "LGTM, merge when green.",
                "mentions": ["ops"],
            },
        )
    )

    assert updated.status == "approved"
    assert updated.summary == "LGTM, merge when green."
    assert seen_statuses == ["approved"]

    persisted = ledger.load()[0]
    assert persisted["status"] == "approved"
    assert any(audit["action"] == "collaboration.comment" for audit in persisted["audits"])
    assert any(
        audit["action"] == "event_bus.transition" and audit["details"]["previous_status"] == "needs-approval"
        for audit in persisted["audits"]
    )


def test_event_bus_ci_completed_marks_run_completed(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-ci", source="github", title="CI completion", description="")
    run = TaskRun.from_task(task, run_id="run-ci-1", medium="docker")
    run.finalize("approved", "waiting for CI")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    updated = bus.publish(
        BusEvent(
            event_type=CI_COMPLETED_EVENT,
            run_id=run.run_id,
            actor="github-actions",
            details={"workflow": "pytest", "conclusion": "success"},
        )
    )

    assert updated.status == "completed"
    assert updated.summary == "CI workflow pytest completed with success"

    persisted = ledger.load()[0]
    assert persisted["status"] == "completed"
    assert any(
        audit["action"] == "event_bus.event" and audit["details"]["event_type"] == CI_COMPLETED_EVENT
        for audit in persisted["audits"]
    )


def test_event_bus_task_failed_marks_run_failed(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-fail", source="scheduler", title="Task failure", description="")
    run = TaskRun.from_task(task, run_id="run-fail-1", medium="docker")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    updated = bus.publish(
        BusEvent(
            event_type=TASK_FAILED_EVENT,
            run_id=run.run_id,
            actor="worker",
            details={"error": "sandbox command exited 137"},
        )
    )

    assert updated.status == "failed"
    assert updated.summary == "sandbox command exited 137"

    persisted = ledger.load()[0]
    assert persisted["status"] == "failed"
    assert any(
        audit["action"] == "event_bus.transition" and audit["details"]["status"] == "failed"
        for audit in persisted["audits"]
    )


def test_p0_audit_event_specs_define_required_operational_events() -> None:
    event_types = {spec.event_type for spec in P0_AUDIT_EVENT_SPECS}

    assert event_types == {
        SCHEDULER_DECISION_EVENT,
        MANUAL_TAKEOVER_EVENT,
        APPROVAL_RECORDED_EVENT,
        BUDGET_OVERRIDE_EVENT,
        FLOW_HANDOFF_EVENT,
    }
    assert missing_required_fields(
        SCHEDULER_DECISION_EVENT,
        {
            "task_id": "OPE-134",
            "run_id": "run-ope-134",
            "medium": "docker",
        },
    ) == ["approved", "reason", "risk_level", "risk_score"]


def test_task_run_audit_spec_event_requires_required_fields() -> None:
    run = TaskRun.from_task(
        Task(task_id="OPE-134-spec", source="linear", title="Validate audit fields", description=""),
        run_id="run-ope-134-spec",
        medium="docker",
    )

    with pytest.raises(ValueError, match="missing required fields"):
        run.audit_spec_event(
            MANUAL_TAKEOVER_EVENT,
            "scheduler",
            "pending",
            task_id="OPE-134-spec",
            run_id="run-ope-134-spec",
            target_team="security",
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
