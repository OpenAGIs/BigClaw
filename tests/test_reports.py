import hashlib
import subprocess
import sys
from pathlib import Path
from typing import List, Optional

import pytest

ROOT = Path(__file__).resolve().parents[1]
SRC = ROOT / "src"
if str(SRC) not in sys.path:
    sys.path.insert(0, str(SRC))

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
from bigclaw.console_ia import (
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ConsoleInteractionAudit,
    ConsoleInteractionAuditor,
    ConsoleInteractionDraft,
    ConsoleSurface,
    FilterDefinition,
    GlobalAction,
    NavigationItem,
    SurfaceInteractionContract,
    SurfacePermissionRule,
    SurfaceState,
    build_big_4203_console_interaction_draft,
    render_console_interaction_report,
    render_console_ia_report,
)
from bigclaw.design_system import (
    AuditRequirement,
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DataAccuracyCheck,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationNode,
    NavigationRoute,
    PerformanceBudget,
    RolePermissionScenario,
    UIAcceptanceAudit,
    UIAcceptanceLibrary,
    UIAcceptanceSuite,
    UsabilityJourney,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
    render_ui_acceptance_report,
)
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
from bigclaw.governance import ScopeFreezeAudit
from bigclaw.memory import TaskMemoryStore
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
from bigclaw.planning import (
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EntryGate,
    EntryGateDecision,
    EvidenceLink,
    FourWeekExecutionPlan,
    WeeklyExecutionPlan,
    WeeklyGoal,
    build_big_4701_execution_plan,
    build_pilot_rollout_scorecard,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    evaluate_candidate_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
    render_pilot_rollout_gate_report,
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
from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewBlocker,
    ReviewBlockerEvent,
    ReviewDecision,
    ReviewObjective,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_audit_density_board,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_checklist_traceability_board,
    render_ui_review_decision_followup_tracker,
    render_ui_review_decision_log,
    render_ui_review_escalation_dashboard,
    render_ui_review_escalation_handoff_ledger,
    render_ui_review_exception_log,
    render_ui_review_exception_matrix,
    render_ui_review_freeze_approval_trail,
    render_ui_review_freeze_exception_board,
    render_ui_review_freeze_renewal_tracker,
    render_ui_review_handoff_ack_ledger,
    render_ui_review_interaction_coverage_board,
    render_ui_review_objective_coverage_board,
    render_ui_review_open_question_tracker,
    render_ui_review_owner_escalation_digest,
    render_ui_review_owner_review_queue,
    render_ui_review_owner_workload_board,
    render_ui_review_pack_html,
    render_ui_review_pack_report,
    render_ui_review_persona_readiness_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_review_summary_board,
    render_ui_review_role_coverage_board,
    render_ui_review_role_matrix,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_log,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_wireframe_readiness_board,
    write_ui_review_pack_bundle,
)
from bigclaw.workspace_bootstrap import (
    bootstrap_workspace,
    cache_root_for_repo,
    cleanup_workspace,
    repo_cache_key,
)
from bigclaw.workspace_bootstrap_validation import build_validation_report

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



def test_candidate_backlog_round_trip_preserves_manifest_shape() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_reports.py",
                        capability="release-gate",
                        note="role-permission and audit readiness coverage",
                    )
                ],
            )
        ],
    )

    restored = CandidateBacklog.from_dict(backlog.to_dict())

    assert restored == backlog


def test_candidate_backlog_ranks_ready_items_ahead_of_blocked_work() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-risky",
                title="Risky migration",
                theme="runtime",
                priority="P0",
                owner="runtime",
                outcome="Move execution runtime to the next rollout ring.",
                validation_command="cd bigclaw-go && go test ./internal/worker ./internal/scheduler",
                capabilities=["runtime-hardening"],
                evidence=["benchmark"],
                blockers=["missing rollback plan"],
            ),
            CandidateEntry(
                candidate_id="candidate-ready",
                title="Release control center",
                theme="console-governance",
                priority="P1",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
            ),
        ],
    )

    ranked_ids = [candidate.candidate_id for candidate in backlog.ranked_candidates]

    assert ranked_ids == ["candidate-ready", "candidate-risky"]


def test_entry_gate_evaluation_requires_ready_candidates_capabilities_and_evidence() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Ops hardening",
                theme="ops-command-center",
                priority="P0",
                owner="ops-platform",
                outcome="Package the command-center rollout with weekly review evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["ops-control"],
                evidence=["weekly-review"],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration",
                title="Orchestration rollout",
                theme="agent-orchestration",
                priority="P1",
                owner="orchestration",
                outcome="Promote cross-team orchestration with commercialization visibility.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["commercialization", "handoff"],
                evidence=["pilot-evidence"],
            ),
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
    )
    baseline_audit = ScopeFreezeAudit(
        board_name="BigClaw v2.0 Freeze",
        version="v2.0",
        total_items=5,
    )

    decision = CandidatePlanner().evaluate_gate(backlog, gate, baseline_audit=baseline_audit)

    assert decision.passed is True
    assert set(decision.ready_candidate_ids) == {
        "candidate-release-control",
        "candidate-ops-hardening",
        "candidate-orchestration",
    }
    assert decision.missing_capabilities == []
    assert decision.missing_evidence == []
    assert decision.baseline_ready is True
    assert decision.baseline_findings == []


def test_entry_gate_holds_when_v2_baseline_is_missing_or_not_ready() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["release-gate"],
                evidence=["acceptance-suite", "validation-report"],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Ops hardening",
                theme="ops-command-center",
                priority="P0",
                owner="ops-platform",
                outcome="Package the command-center rollout with weekly review evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["ops-control"],
                evidence=["weekly-review"],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration",
                title="Orchestration rollout",
                theme="agent-orchestration",
                priority="P1",
                owner="orchestration",
                outcome="Promote cross-team orchestration with commercialization visibility.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["commercialization"],
                evidence=["pilot-evidence"],
            ),
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
    )

    missing_baseline = CandidatePlanner().evaluate_gate(backlog, gate)
    failed_baseline = CandidatePlanner().evaluate_gate(
        backlog,
        gate,
        baseline_audit=ScopeFreezeAudit(
            board_name="BigClaw v2.0 Freeze",
            version="v2.0",
            total_items=5,
            missing_validation=["OPE-116"],
        ),
    )

    assert missing_baseline.passed is False
    assert missing_baseline.baseline_ready is False
    assert missing_baseline.baseline_findings == ["missing baseline audit for v2.0"]
    assert failed_baseline.passed is False
    assert failed_baseline.baseline_ready is False
    assert failed_baseline.baseline_findings == ["baseline v2.0 is not release ready (87.5)"]


def test_entry_gate_decision_round_trip_preserves_findings() -> None:
    decision = EntryGateDecision(
        gate_id="gate-v3-entry",
        passed=False,
        ready_candidate_ids=["candidate-release-control"],
        blocked_candidate_ids=["candidate-runtime"],
        missing_capabilities=["commercialization"],
        missing_evidence=["pilot-evidence"],
        baseline_ready=False,
        baseline_findings=["baseline v2.0 is not release ready (87.5)"],
        blocker_count=1,
    )

    restored = EntryGateDecision.from_dict(decision.to_dict())

    assert restored == decision


def test_render_candidate_backlog_report_summarizes_backlog_and_gate_findings() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_reports.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_reports.py",
                        capability="release-gate",
                    )
                ],
            )
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=1,
        required_capabilities=["release-gate"],
        required_evidence=["validation-report"],
        required_baseline_version="v2.0",
    )
    decision = CandidatePlanner().evaluate_gate(
        backlog,
        gate,
        baseline_audit=ScopeFreezeAudit(
            board_name="BigClaw v2.0 Freeze",
            version="v2.0",
            total_items=5,
        ),
    )

    report = render_candidate_backlog_report(backlog, gate, decision)

    assert "# V3 Candidate Backlog Report" in report
    assert "- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件" in report
    assert "- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0" in report
    assert (
        "- candidate-release-control: Release control center "
        "priority=P0 owner=platform-ui score=100 ready=True"
    ) in report
    assert "validation=python3 -m pytest tests/test_reports.py -q" in report
    assert "- ui-acceptance -> tests/test_reports.py capability=release-gate" in report
    assert "- Missing evidence: none" in report
    assert "- Baseline ready: True" in report
    assert "- Baseline findings: none" in report


def test_candidate_entry_round_trip_preserves_evidence_links() -> None:
    candidate = CandidateEntry(
        candidate_id="candidate-ops-hardening",
        title="Ops hardening",
        theme="ops-command-center",
        priority="P0",
        owner="ops-platform",
        outcome="Package command-center and approval surfaces with linked evidence.",
        validation_command="python3 -m pytest tests/test_reports.py -q && (cd bigclaw-go && go test ./internal/product)",
        capabilities=["ops-control", "saved-views"],
        evidence=["weekly-review", "validation-report"],
        evidence_links=[
            EvidenceLink(
                label="queue-control-center",
                target="src/bigclaw/operations.py",
                capability="ops-control",
                note="queue and approval command center",
            ),
            EvidenceLink(
                label="saved-view-report",
                target="src/bigclaw/saved_views.py",
                capability="saved-views",
                note="team saved views and digest evidence",
            ),
        ],
    )
    restored = CandidateEntry.from_dict(candidate.to_dict())

    assert restored == candidate


def test_four_week_execution_plan_round_trip_preserves_weeks_and_goals() -> None:
    plan = build_big_4701_execution_plan()

    restored = FourWeekExecutionPlan.from_dict(plan.to_dict())

    assert restored == plan


def test_four_week_execution_plan_rolls_up_progress_and_at_risk_weeks() -> None:
    plan = build_big_4701_execution_plan()

    assert plan.total_goals == 8
    assert plan.completed_goals == 2
    assert plan.overall_progress_percent == 25
    assert plan.at_risk_weeks == [2]
    assert plan.goal_status_counts() == {
        "done": 2,
        "on-track": 1,
        "at-risk": 1,
        "not-started": 4,
    }


def test_four_week_execution_plan_validate_rejects_missing_or_unordered_weeks() -> None:
    plan = FourWeekExecutionPlan(
        plan_id="BIG-4701",
        title="4周执行计划与周目标",
        owner="execution-office",
        start_date="2026-03-11",
        weeks=[
            WeeklyExecutionPlan(week_number=1, theme="One", objective="One"),
            WeeklyExecutionPlan(week_number=3, theme="Three", objective="Three"),
            WeeklyExecutionPlan(week_number=2, theme="Two", objective="Two"),
            WeeklyExecutionPlan(week_number=4, theme="Four", objective="Four"),
        ],
    )

    with pytest.raises(
        ValueError,
        match="Four-week execution plans must include weeks 1 through 4 in order",
    ):
        plan.validate()


def test_render_four_week_execution_report_summarizes_plan_status() -> None:
    report = render_four_week_execution_report(build_big_4701_execution_plan())

    assert "# Four-Week Execution Plan" in report
    assert "- Plan: BIG-4701 4周执行计划与周目标" in report
    assert "- Overall progress: 2/8 goals complete (25%)" in report
    assert "- At-risk weeks: 2" in report
    assert "- Week 2: Build and integration progress=0/2 (0%)" in report
    assert (
        "- w2-handoff-sync: Resolve orchestration and console handoff dependencies "
        "owner=orchestration-office status=at-risk"
    ) in report


def test_weekly_execution_plan_flags_at_risk_goal_ids() -> None:
    week = WeeklyExecutionPlan(
        week_number=2,
        theme="Build and integration",
        objective="Land high-risk integration work.",
        goals=[
            WeeklyGoal(
                goal_id="w2-green",
                title="Green goal",
                owner="eng",
                status="on-track",
                success_metric="merged PRs",
                target_value="2",
            ),
            WeeklyGoal(
                goal_id="w2-blocked",
                title="Blocked goal",
                owner="eng",
                status="blocked",
                success_metric="open blockers",
                target_value="0",
            ),
        ],
    )

    assert week.at_risk_goal_ids == ["w2-blocked"]


def test_pilot_rollout_scorecard_and_candidate_gate() -> None:
    scorecard = build_pilot_rollout_scorecard(
        adoption=84,
        convergence_improvement=78,
        review_efficiency=82,
        governance_incidents=1,
        evidence_completeness=88,
    )
    assert scorecard["recommendation"] == "go"

    gate_decision = EntryGateDecision(gate_id="gate-v3", passed=True)
    result = evaluate_candidate_gate(gate_decision=gate_decision, rollout_scorecard=scorecard)

    assert result["candidate_gate"] == "enable-by-default"
    report = render_pilot_rollout_gate_report(result)
    assert "Candidate gate" in report


def test_big501_memory_store_reuses_history_and_injects_rules(tmp_path) -> None:
    store = TaskMemoryStore(str(tmp_path / "memory" / "task-patterns.json"))

    previous = Task(
        task_id="BIG-501-prev",
        source="github",
        title="Previous queue rollout",
        description="",
        labels=["queue", "platform"],
        required_tools=["github", "browser"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest", "smoke-test"],
    )
    store.remember_success(previous, summary="queue migration done")

    current = Task(
        task_id="BIG-501-new",
        source="github",
        title="New queue hardening",
        description="",
        labels=["queue"],
        required_tools=["github"],
        acceptance_criteria=["unit-tests"],
        validation_plan=["pytest"],
    )

    suggestion = store.suggest_rules(current)

    assert "BIG-501-prev" in suggestion["matched_task_ids"]
    assert "report-shared" in suggestion["acceptance_criteria"]
    assert "smoke-test" in suggestion["validation_plan"]
    assert "unit-tests" in suggestion["acceptance_criteria"]


def test_build_v3_candidate_backlog_matches_issue_plan_traceability() -> None:
    backlog = build_v3_candidate_backlog()

    assert backlog.epic_id == "BIG-EPIC-20"
    assert backlog.title == "v4.0 v3候选与进入条件"
    assert [candidate.candidate_id for candidate in backlog.ranked_candidates] == [
        "candidate-ops-hardening",
        "candidate-orchestration-rollout",
        "candidate-release-control",
    ]
    assert all(candidate.ready for candidate in backlog.candidates)

    ops_candidate = next(
        candidate for candidate in backlog.candidates if candidate.candidate_id == "candidate-ops-hardening"
    )
    assert {link.target for link in ops_candidate.evidence_links} >= {
        "src/bigclaw/operations.py",
        "bigclaw-go/internal/reporting/reporting_test.go",
        "bigclaw-go/internal/queue/memory_queue_test.go",
        "tests/test_reports.py",
        "src/bigclaw/execution_contract.py",
        "src/bigclaw/workflow.py",
        "bigclaw-go/internal/workflow/engine_test.go",
        "bigclaw-go/internal/worker/runtime_test.go",
        "src/bigclaw/saved_views.py",
        "bigclaw-go/internal/product/saved_views_test.go",
        "src/bigclaw/evaluation.py",
        "tests/test_reports.py",
    }


def test_build_v3_entry_gate_passes_built_candidate_backlog_against_v2_baseline() -> None:
    backlog = build_v3_candidate_backlog()
    gate = build_v3_entry_gate()

    decision = CandidatePlanner().evaluate_gate(
        backlog,
        gate,
        baseline_audit=ScopeFreezeAudit(
            board_name="BigClaw v2.0 Freeze",
            version="v2.0",
            total_items=25,
        ),
    )
    report = render_candidate_backlog_report(backlog, gate, decision)

    assert decision.passed is True
    assert decision.ready_candidate_ids == [
        "candidate-ops-hardening",
        "candidate-orchestration-rollout",
        "candidate-release-control",
    ]
    assert decision.missing_capabilities == []
    assert decision.missing_evidence == []
    assert "candidate-ops-hardening: Operations command-center hardening" in report
    assert "- command-center-src -> src/bigclaw/operations.py capability=ops-control" in report
    assert "- report-studio-tests -> tests/test_reports.py capability=commercialization" in report


def git(repo: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", *args],
        cwd=repo,
        text=True,
        capture_output=True,
        check=True,
    )
    return completed.stdout.strip()


def init_repo(repo: Path, branch: str = "main") -> None:
    git(repo, "init", "-b", branch)
    git(repo, "config", "user.email", "test@example.com")
    git(repo, "config", "user.name", "Test User")


def commit_file(repo: Path, name: str, content: str, message: str) -> str:
    (repo / name).write_text(content)
    git(repo, "add", name)
    git(repo, "commit", "-m", message)
    return git(repo, "rev-parse", "HEAD")


def init_remote_with_main(tmp_path: Path) -> Path:
    remote = tmp_path / "remote.git"
    subprocess.run(
        ["git", "init", "--bare", "--initial-branch=main", str(remote)],
        check=True,
        capture_output=True,
        text=True,
    )

    source = tmp_path / "source"
    source.mkdir()
    init_repo(source)
    git(source, "remote", "add", "origin", str(remote))
    commit_file(source, "README.md", "hello\n", "initial")
    git(source, "push", "-u", "origin", "main")
    return remote


def build_review_pack() -> UIReviewPack:
    return UIReviewPack(
        issue_id="BIG-4204",
        title="UI评审包输出",
        version="v4.0-review-pack",
        objectives=[
            ReviewObjective(
                objective_id="obj-alignment",
                title="Align reviewers on the release-control story",
                persona="product-experience",
                outcome="Reviewers see the scope, stakes, and success criteria before page-level critique.",
                success_signal="The kickoff frame is sufficient to decide whether the slice is review-ready.",
                priority="P0",
                dependencies=["BIG-1103", "BIG-1701"],
            )
        ],
        wireframes=[
            WireframeSurface(
                surface_id="wf-overview",
                name="Review overview board",
                device="desktop",
                entry_point="Epic 11 review hub",
                primary_blocks=["header", "objective strip", "wireframe rail", "decision log"],
                review_notes=["Highlight unresolved dependencies before approval."],
            )
        ],
        interactions=[
            InteractionFlow(
                flow_id="flow-frame-switch",
                name="Switch between wireframes and interaction notes",
                trigger="Reviewer selects a surface from the wireframe rail",
                system_response="The board swaps the focus frame and preserves reviewer comments.",
                states=["default", "focus", "with-comments"],
                exceptions=["Warn when leaving a frame with unsaved notes."],
            )
        ],
        open_questions=[
            OpenQuestion(
                question_id="oq-mobile-depth",
                theme="scope",
                question="Should the first review pack cover mobile wireframes or desktop only?",
                owner="product-experience",
                impact="Changes review breadth and the number of required surfaces.",
            )
        ],
    )

def test_build_big_4204_review_pack_is_ready_for_design_sprint_review() -> None:
    pack = build_big_4204_review_pack()

    audit = UIReviewPackAuditor().audit(pack)
    report = render_ui_review_pack_report(pack, audit)

    assert audit.ready is True
    assert len(pack.objectives) == 4
    assert len(pack.wireframes) == 4
    assert len(pack.interactions) == 4
    assert len(pack.open_questions) == 3
    assert len(pack.reviewer_checklist) == 8
    assert len(pack.decision_log) == 4
    assert len(pack.role_matrix) == 8
    assert len(pack.signoff_log) == 4
    assert len(pack.blocker_log) == 1
    assert len(pack.blocker_timeline) == 2
    assert pack.requires_reviewer_checklist is True
    assert pack.requires_decision_log is True
    assert pack.requires_role_matrix is True
    assert pack.requires_signoff_log is True
    assert pack.requires_blocker_log is True
    assert pack.requires_blocker_timeline is True
    assert "obj-queue-governance" in report
    assert "## Review Summary Board" in report
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in report
    assert "## Objective Coverage Board" in report
    assert "- covered: 2" in report
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in report
    assert "## Persona Readiness Board" in report
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in report
    assert "wf-triage: Triage and handoff board" in report
    assert "## Wireframe Readiness Board" in report
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in report
    assert "flow-run-replay: Run replay with evidence audit" in report
    assert "## Interaction Coverage Board" in report
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in report
    assert "## Open Question Tracker" in report
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in report
    assert "chk-queue-batch-approval: surface=wf-queue owner=Platform Admin status=ready" in report
    assert "dec-queue-vp-summary: surface=wf-queue owner=VP Eng status=proposed" in report
    assert "role-queue-platform-admin: surface=wf-queue role=Platform Admin status=ready" in report
    assert "## Checklist Traceability Board" in report
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in report
    assert "## Decision Follow-up Tracker" in report
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in report
    assert "## Role Coverage Board" in report
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in report
    assert "## Signoff Dependency Board" in report
    assert "- blocked: 1" in report
    assert "- clear: 3" in report
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in report
    assert "assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z sla=at-risk due_at=2026-03-15T18:00:00Z cadence=daily" in report
    assert "sig-run-detail-eng-lead: surface=wf-run-detail role=Eng Lead assignment=role-run-detail-eng-lead status=pending" in report
    assert "blk-run-detail-copy-final: surface=wf-run-detail signoff=sig-run-detail-eng-lead owner=product-experience status=open severity=medium" in report
    assert "evt-run-detail-copy-escalated: blocker=blk-run-detail-copy-final actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in report
    assert "## Review Exceptions" in report
    assert "exc-blk-run-detail-copy-final: type=blocker source=blk-run-detail-copy-final surface=wf-run-detail owner=product-experience status=open severity=medium" in report
    assert "## Sign-off SLA Dashboard" in report
    assert "- at-risk: 1" in report
    assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in report
    assert "## Sign-off Reminder Queue" in report
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in report
    assert "## Reminder Cadence Board" in report
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in report
    assert "## Sign-off Breach Board" in report
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in report
    assert "## Escalation Dashboard" in report
    assert "- engineering-director: blockers=0 signoffs=1 total=1" in report
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in report
    assert "## Escalation Handoff Ledger" in report
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in report
    assert "## Handoff Ack Ledger" in report
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in report
    assert "## Owner Escalation Digest" in report
    assert "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2" in report
    assert "## Owner Workload Board" in report
    assert "- Owners: 7" in report
    assert "- Items: 8" in report
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 reminders=0 renewals=0 total=2" in report
    assert "load-queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open lane=queue" in report
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in report
    assert "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal" in report
    assert "## Review Freeze Exception Board" in report
    assert "## Freeze Approval Trail" in report
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in report
    assert "## Freeze Renewal Tracker" in report
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in report
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in report
    assert "## Review Exception Matrix" in report
    assert "- product-experience: blockers=1 signoffs=0 total=1" in report
    assert "## Audit Density Board" in report
    assert "- Surfaces: 4" in report
    assert "- Load bands: 3" in report
    assert "- active: 2" in report
    assert "- dense: 1" in report
    assert "- light: 1" in report
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in report
    assert "checklist=2 decisions=1 assignments=2 signoffs=1 blockers=1 timeline=2 blocks=4 notes=2" in report
    assert "## Owner Review Queue" in report
    assert "queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in report
    assert "## Blocker Timeline Summary" in report
    assert "- escalated: 1" in report
    assert "- Wireframes missing checklist coverage: none" in report
    assert "- Checklist items missing role links: none" in report
    assert "- Wireframes missing decision coverage: none" in report
    assert "- Unresolved decisions missing follow-ups: none" in report
    assert "- Wireframes missing role assignments: none" in report
    assert "- Wireframes missing signoff coverage: none" in report
    assert "- Blockers missing signoff links: none" in report
    assert "- Freeze exceptions missing owners: none" in report
    assert "- Freeze exceptions missing windows: none" in report
    assert "- Freeze exceptions missing approvers: none" in report
    assert "- Freeze exceptions missing approval dates: none" in report
    assert "- Freeze exceptions missing renewal owners: none" in report
    assert "- Freeze exceptions missing renewal dates: none" in report
    assert "- Blockers missing timeline events: none" in report
    assert "- Closed blockers missing resolution events: none" in report
    assert "- Orphan blocker timeline blocker ids: none" in report
    assert "- Handoff events missing targets: none" in report
    assert "- Handoff events missing artifacts: none" in report
    assert "- Handoff events missing ack owners: none" in report
    assert "- Handoff events missing ack dates: none" in report
    assert "- Unresolved required signoffs without blockers: none" in report
    assert "- Unresolved decision ids: dec-queue-vp-summary" in report
    assert "- Decisions missing role links: none" in report
    assert "- Signoffs missing requested dates: none" in report
    assert "- Signoffs missing due dates: none" in report
    assert "- Signoffs missing escalation owners: none" in report
    assert "- Signoffs missing reminder owners: none" in report
    assert "- Signoffs missing next reminders: none" in report
    assert "- Signoffs missing reminder cadence: none" in report
    assert "- Signoffs with breached SLA: none" in report
    assert "- Unresolved required signoff ids: sig-run-detail-eng-lead" in report
    assert "- Unresolved questions: oq-role-density, oq-alert-priority, oq-handoff-evidence" in report

def test_render_ui_review_exception_matrix_includes_signoff_and_blocker_counts() -> None:
    pack = build_big_4204_review_pack()
    pack.signoff_log[2] = ReviewSignoff(
        signoff_id="sig-run-detail-eng-lead",
        assignment_id="role-run-detail-eng-lead",
        surface_id="wf-run-detail",
        role="Eng Lead",
        status="waived",
        evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
        notes="Temporary waiver approved pending copy lock.",
        waiver_owner="Eng Lead",
        waiver_reason="Copy review is deferred to the next wording pass.",
    )

    exception_matrix = render_ui_review_exception_matrix(pack)

    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- Exceptions: 2" in exception_matrix
    assert "- Owners: 2" in exception_matrix
    assert "- Surfaces: 1" in exception_matrix
    assert "- Eng Lead: blockers=0 signoffs=1 total=1" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "- open: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "- waived: blockers=0 signoffs=1 total=1" in exception_matrix
    assert "- wf-run-detail: blockers=1 signoffs=1 total=2" in exception_matrix



def test_render_ui_review_freeze_exception_board() -> None:
    pack = build_big_4204_review_pack()

    freeze_board = render_ui_review_freeze_exception_board(pack)

    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "- Exceptions: 1" in freeze_board
    assert "- release-director: blockers=1 signoffs=0 total=1" in freeze_board
    assert "- wf-run-detail: blockers=1 signoffs=0 total=1" in freeze_board
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in freeze_board


def test_render_ui_review_freeze_approval_trail() -> None:
    pack = build_big_4204_review_pack()

    freeze_trail = render_ui_review_freeze_approval_trail(pack)

    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "- Approvals: 1" in freeze_trail
    assert "- release-director: 1" in freeze_trail
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in freeze_trail




def test_render_ui_review_summary_persona_and_interaction_boards() -> None:
    pack = build_big_4204_review_pack()

    review_summary = render_ui_review_review_summary_board(pack)
    persona_readiness = render_ui_review_persona_readiness_board(pack)
    interaction_coverage = render_ui_review_interaction_coverage_board(pack)

    assert "# UI Review Review Summary Board" in review_summary
    assert "- Categories: 6" in review_summary
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in review_summary
    assert "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2" in review_summary
    assert "summary-interactions: category=interactions total=4 covered=4 watch=0 missing=0" in review_summary
    assert "summary-actions: category=actions total=8 queue=6 reminder=1 renewal=1" in review_summary
    assert "# UI Review Persona Readiness Board" in persona_readiness
    assert "- Personas: 4" in persona_readiness
    assert "- Objectives: 4" in persona_readiness
    assert "- blocked: 1" in persona_readiness
    assert "- at-risk: 1" in persona_readiness
    assert "- ready: 2" in persona_readiness
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in persona_readiness
    assert "objective_ids=obj-run-detail-investigation surfaces=wf-run-detail queue_ids=queue-sig-run-detail-eng-lead blocker_ids=blk-run-detail-copy-final" in persona_readiness
    assert "# UI Review Interaction Coverage Board" in interaction_coverage
    assert "- Interactions: 4" in interaction_coverage
    assert "- Surfaces: 4" in interaction_coverage
    assert "- covered: 4" in interaction_coverage
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in interaction_coverage
    assert "checklist=chk-triage-handoff-clarity,chk-triage-bulk-assign open_checklist=none trigger=Cross-Team Operator bulk-assigns a finding set or opens the handoff panel" in interaction_coverage


def test_render_ui_review_objective_wireframe_and_question_boards() -> None:
    pack = build_big_4204_review_pack()

    objective_coverage = render_ui_review_objective_coverage_board(pack)
    wireframe_readiness = render_ui_review_wireframe_readiness_board(pack)
    question_tracker = render_ui_review_open_question_tracker(pack)

    assert "# UI Review Objective Coverage Board" in objective_coverage
    assert "- Objectives: 4" in objective_coverage
    assert "- Personas: 4" in objective_coverage
    assert "- blocked: 1" in objective_coverage
    assert "- covered: 2" in objective_coverage
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in objective_coverage
    assert "dependency_ids=BIG-4203,OPE-72,OPE-73 assignments=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final" in objective_coverage
    assert "# UI Review Wireframe Readiness Board" in wireframe_readiness
    assert "- Wireframes: 4" in wireframe_readiness
    assert "- Devices: 1" in wireframe_readiness
    assert "- at-risk: 2" in wireframe_readiness
    assert "- blocked: 1" in wireframe_readiness
    assert "- ready: 1" in wireframe_readiness
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in wireframe_readiness
    assert "checklist_open=1 decisions_open=0 assignments_open=1 signoffs_open=1 blockers_open=1 signoffs=sig-run-detail-eng-lead blockers=blk-run-detail-copy-final blocks=4 notes=2" in wireframe_readiness
    assert "# UI Review Open Question Tracker" in question_tracker
    assert "- Questions: 3" in question_tracker
    assert "- Owners: 3" in question_tracker
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in question_tracker
    assert "checklist=chk-queue-role-density flows=none impact=Changes denial-path copy, button placement, and review criteria for queue and triage pages." in question_tracker


def test_render_ui_review_traceability_and_role_coverage_boards() -> None:
    pack = build_big_4204_review_pack()

    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)

    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "- Checklist items: 8" in checklist_traceability
    assert "- Owners: 7" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "- Decisions: 4" in decision_followup
    assert "- Owners: 4" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "linked_assignments=role-queue-platform-admin,role-queue-product-experience linked_checklists=chk-queue-batch-approval,chk-queue-role-density follow_up=Resolve after the next design critique with policy owners." in decision_followup
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "- Assignments: 8" in role_coverage
    assert "- Surfaces: 4" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "signoff=sig-run-detail-eng-lead signoff_status=pending" in role_coverage


def test_render_ui_review_dependency_workload_and_density_boards() -> None:
    pack = build_big_4204_review_pack()

    signoff_dependency = render_ui_review_signoff_dependency_board(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    audit_density = render_ui_review_audit_density_board(pack)

    assert "# UI Review Signoff Dependency Board" in signoff_dependency
    assert "- Sign-offs: 4" in signoff_dependency
    assert "- blocked: 1" in signoff_dependency
    assert "- clear: 3" in signoff_dependency
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in signoff_dependency
    assert "assignment=role-run-detail-eng-lead checklist=chk-run-replay-context decisions=dec-run-detail-audit-rail latest_blocker_event=evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z sla=at-risk due_at=2026-03-15T18:00:00Z cadence=daily" in signoff_dependency
    assert "# UI Review Owner Workload Board" in owner_workload
    assert "- Owners: 7" in owner_workload
    assert "- Items: 8" in owner_workload
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 reminders=0 renewals=0 total=2" in owner_workload
    assert "load-queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open lane=queue" in owner_workload
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in owner_workload
    assert "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal" in owner_workload
    assert "# UI Review Audit Density Board" in audit_density
    assert "- Surfaces: 4" in audit_density
    assert "- Load bands: 3" in audit_density
    assert "- active: 2" in audit_density
    assert "- dense: 1" in audit_density
    assert "- light: 1" in audit_density
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in audit_density
    assert "checklist=2 decisions=1 assignments=2 signoffs=1 blockers=1 timeline=2 blocks=4 notes=2" in audit_density


def test_render_ui_review_owner_review_queue_groups_actionable_items() -> None:
    pack = build_big_4204_review_pack()

    owner_queue = render_ui_review_owner_review_queue(pack)

    assert "# UI Review Owner Review Queue" in owner_queue
    assert "- Owners: 5" in owner_queue
    assert "- Queue items: 6" in owner_queue
    assert "- engineering-operations: blockers=0 checklist=1 decisions=0 signoffs=0 total=1" in owner_queue
    assert "- product-experience: blockers=1 checklist=1 decisions=0 signoffs=0 total=2" in owner_queue
    assert "- queue-chk-queue-role-density: owner=product-experience type=checklist source=chk-queue-role-density surface=wf-queue status=open" in owner_queue
    assert "- queue-dec-queue-vp-summary: owner=VP Eng type=decision source=dec-queue-vp-summary surface=wf-queue status=proposed" in owner_queue
    assert "- queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in owner_queue
    assert "- queue-blk-run-detail-copy-final: owner=product-experience type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open" in owner_queue



def test_render_ui_review_exception_log_and_timeline_summary() -> None:
    pack = build_big_4204_review_pack()

    signoff_sla = render_ui_review_signoff_sla_dashboard(pack)
    signoff_reminder = render_ui_review_signoff_reminder_queue(pack)
    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    reminder_cadence = render_ui_review_reminder_cadence_board(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)
    signoff_breach = render_ui_review_signoff_breach_board(pack)
    escalation_dashboard = render_ui_review_escalation_dashboard(pack)
    handoff_ledger = render_ui_review_escalation_handoff_ledger(pack)
    handoff_ack = render_ui_review_handoff_ack_ledger(pack)
    owner_digest = render_ui_review_owner_escalation_digest(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    freeze_board = render_ui_review_freeze_exception_board(pack)
    freeze_trail = render_ui_review_freeze_approval_trail(pack)
    freeze_renewal = render_ui_review_freeze_renewal_tracker(pack)
    exception_log = render_ui_review_exception_log(pack)
    exception_matrix = render_ui_review_exception_matrix(pack)
    audit_density = render_ui_review_audit_density_board(pack)
    owner_review_queue = render_ui_review_owner_review_queue(pack)
    timeline_summary = render_ui_review_blocker_timeline_summary(pack)

    assert "# UI Review Sign-off SLA Dashboard" in signoff_sla
    assert "- at-risk: 1" in signoff_sla
    assert "# UI Review Sign-off Reminder Queue" in signoff_reminder
    assert "- Reminders: 1" in signoff_reminder
    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "# UI Review Reminder Cadence Board" in reminder_cadence
    assert "- Cadences: 1" in reminder_cadence
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in reminder_cadence
    assert "# UI Review Sign-off Breach Board" in signoff_breach
    assert "- Breach items: 1" in signoff_breach
    assert "# UI Review Escalation Dashboard" in escalation_dashboard
    assert "- engineering-director: blockers=0 signoffs=1 total=1" in escalation_dashboard
    assert "# UI Review Escalation Handoff Ledger" in handoff_ledger
    assert "- design-critique: 1" in handoff_ledger
    assert "# UI Review Handoff Ack Ledger" in handoff_ack
    assert "- Ack owners: 1" in handoff_ack
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in handoff_ack
    assert "# UI Review Owner Escalation Digest" in owner_digest
    assert "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2" in owner_digest
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "- release-director: blockers=1 signoffs=0 total=1" in freeze_board
    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "- Approvals: 1" in freeze_trail
    assert "# UI Review Freeze Renewal Tracker" in freeze_renewal
    assert "- Renewal owners: 1" in freeze_renewal
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in freeze_renewal
    assert "# UI Review Exception Log" in exception_log
    assert "- Exceptions: 1" in exception_log
    assert "exc-blk-run-detail-copy-final" in exception_log
    assert "evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z" in exception_log
    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "# UI Review Audit Density Board" in audit_density
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in audit_density
    assert "# UI Review Owner Review Queue" in owner_review_queue
    assert "- Queue items: 6" in owner_review_queue
    assert "# UI Review Blocker Timeline Summary" in timeline_summary
    assert "- Events: 2" in timeline_summary
    assert "- opened: 1" in timeline_summary
    assert "- escalated: 1" in timeline_summary
    assert "- design-program-manager: 1" in timeline_summary
    assert "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in timeline_summary


def test_render_ui_review_html_and_bundle_export(tmp_path) -> None:
    pack = build_big_4204_review_pack()
    audit = UIReviewPackAuditor().audit(pack)

    html = render_ui_review_pack_html(pack, audit)
    checklist_traceability = render_ui_review_checklist_traceability_board(pack)
    decision_log = render_ui_review_decision_log(pack)
    decision_followup = render_ui_review_decision_followup_tracker(pack)
    review_summary = render_ui_review_review_summary_board(pack)
    objective_coverage = render_ui_review_objective_coverage_board(pack)
    persona_readiness = render_ui_review_persona_readiness_board(pack)
    wireframe_readiness = render_ui_review_wireframe_readiness_board(pack)
    interaction_coverage = render_ui_review_interaction_coverage_board(pack)
    question_tracker = render_ui_review_open_question_tracker(pack)
    role_matrix = render_ui_review_role_matrix(pack)
    role_coverage = render_ui_review_role_coverage_board(pack)
    signoff_dependency = render_ui_review_signoff_dependency_board(pack)
    signoff_log = render_ui_review_signoff_log(pack)
    blocker_log = render_ui_review_blocker_log(pack)
    blocker_timeline = render_ui_review_blocker_timeline(pack)
    signoff_sla = render_ui_review_signoff_sla_dashboard(pack)
    signoff_reminder = render_ui_review_signoff_reminder_queue(pack)
    reminder_cadence = render_ui_review_reminder_cadence_board(pack)
    signoff_breach = render_ui_review_signoff_breach_board(pack)
    escalation_dashboard = render_ui_review_escalation_dashboard(pack)
    handoff_ledger = render_ui_review_escalation_handoff_ledger(pack)
    handoff_ack = render_ui_review_handoff_ack_ledger(pack)
    owner_digest = render_ui_review_owner_escalation_digest(pack)
    owner_workload = render_ui_review_owner_workload_board(pack)
    freeze_board = render_ui_review_freeze_exception_board(pack)
    freeze_trail = render_ui_review_freeze_approval_trail(pack)
    freeze_renewal = render_ui_review_freeze_renewal_tracker(pack)
    exception_log = render_ui_review_exception_log(pack)
    exception_matrix = render_ui_review_exception_matrix(pack)
    audit_density = render_ui_review_audit_density_board(pack)
    owner_review_queue = render_ui_review_owner_review_queue(pack)
    timeline_summary = render_ui_review_blocker_timeline_summary(pack)
    artifacts = write_ui_review_pack_bundle(str(tmp_path), pack)

    assert "<h2>Decision Log</h2>" in html
    assert "<h2>Checklist Traceability Board</h2>" in html
    assert "<h2>Decision Follow-up Tracker</h2>" in html
    assert "<h2>Review Summary Board</h2>" in html
    assert "<h2>Objective Coverage Board</h2>" in html
    assert "<h2>Persona Readiness Board</h2>" in html
    assert "<h2>Wireframe Readiness Board</h2>" in html
    assert "<h2>Interaction Coverage Board</h2>" in html
    assert "<h2>Open Question Tracker</h2>" in html
    assert "<h2>Role Matrix</h2>" in html
    assert "<h2>Role Coverage Board</h2>" in html
    assert "<h2>Signoff Dependency Board</h2>" in html
    assert "<h2>Sign-off Log</h2>" in html
    assert "<h2>Sign-off SLA Dashboard</h2>" in html
    assert "<h2>Sign-off Reminder Queue</h2>" in html
    assert "<h2>Reminder Cadence Board</h2>" in html
    assert "<h2>Sign-off Breach Board</h2>" in html
    assert "<h2>Escalation Dashboard</h2>" in html
    assert "<h2>Escalation Handoff Ledger</h2>" in html
    assert "<h2>Handoff Ack Ledger</h2>" in html
    assert "<h2>Owner Escalation Digest</h2>" in html
    assert "<h2>Owner Workload Board</h2>" in html
    assert "<h2>Blocker Log</h2>" in html
    assert "<h2>Blocker Timeline</h2>" in html
    assert "<h2>Review Freeze Exception Board</h2>" in html
    assert "<h2>Freeze Approval Trail</h2>" in html
    assert "<h2>Freeze Renewal Tracker</h2>" in html
    assert "<h2>Review Exceptions</h2>" in html
    assert "<h2>Review Exception Matrix</h2>" in html
    assert "<h2>Audit Density Board</h2>" in html
    assert "<h2>Owner Review Queue</h2>" in html
    assert "<h2>Blocker Timeline Summary</h2>" in html
    assert "dec-queue-vp-summary" in html
    assert "# UI Review Checklist Traceability Board" in checklist_traceability
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability
    assert "# UI Review Decision Log" in decision_log
    assert "dec-run-detail-audit-rail" in decision_log
    assert "# UI Review Decision Follow-up Tracker" in decision_followup
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup
    assert "# UI Review Review Summary Board" in review_summary
    assert "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2" in review_summary
    assert "# UI Review Objective Coverage Board" in objective_coverage
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in objective_coverage
    assert "# UI Review Persona Readiness Board" in persona_readiness
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in persona_readiness
    assert "# UI Review Wireframe Readiness Board" in wireframe_readiness
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in wireframe_readiness
    assert "# UI Review Interaction Coverage Board" in interaction_coverage
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in interaction_coverage
    assert "# UI Review Open Question Tracker" in question_tracker
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in question_tracker
    assert "# UI Review Role Matrix" in role_matrix
    assert "role-triage-platform-admin" in role_matrix
    assert "# UI Review Role Coverage Board" in role_coverage
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in role_coverage
    assert "# UI Review Signoff Dependency Board" in signoff_dependency
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in signoff_dependency
    assert "# UI Review Sign-off Log" in signoff_log
    assert "sig-run-detail-eng-lead" in signoff_log
    assert "# UI Review Sign-off SLA Dashboard" in signoff_sla
    assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in signoff_sla
    assert "# UI Review Sign-off Reminder Queue" in signoff_reminder
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in signoff_reminder
    assert "# UI Review Reminder Cadence Board" in reminder_cadence
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in reminder_cadence
    assert "# UI Review Sign-off Breach Board" in signoff_breach
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in signoff_breach
    assert "# UI Review Escalation Dashboard" in escalation_dashboard
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in escalation_dashboard
    assert "# UI Review Escalation Handoff Ledger" in handoff_ledger
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in handoff_ledger
    assert "# UI Review Handoff Ack Ledger" in handoff_ack
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in handoff_ack
    assert "# UI Review Owner Escalation Digest" in owner_digest
    assert "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in owner_digest
    assert "# UI Review Owner Workload Board" in owner_workload
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in owner_workload
    assert "# UI Review Freeze Exception Board" in freeze_board
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in freeze_board
    assert "# UI Review Freeze Approval Trail" in freeze_trail
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in freeze_trail
    assert "# UI Review Freeze Renewal Tracker" in freeze_renewal
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in freeze_renewal
    assert "# UI Review Blocker Log" in blocker_log
    assert "blk-run-detail-copy-final" in blocker_log
    assert "# UI Review Blocker Timeline" in blocker_timeline
    assert "evt-run-detail-copy-escalated" in blocker_timeline
    assert "# UI Review Exception Log" in exception_log
    assert "exc-blk-run-detail-copy-final" in exception_log
    assert "# UI Review Exception Matrix" in exception_matrix
    assert "- product-experience: blockers=1 signoffs=0 total=1" in exception_matrix
    assert "# UI Review Owner Review Queue" in owner_review_queue
    assert "- Queue items: 6" in owner_review_queue
    assert "# UI Review Blocker Timeline Summary" in timeline_summary
    assert "- escalated: 1" in timeline_summary
    assert Path(artifacts.markdown_path).exists()
    assert Path(artifacts.html_path).exists()
    assert Path(artifacts.decision_log_path).exists()
    assert Path(artifacts.review_summary_board_path).exists()
    assert Path(artifacts.objective_coverage_board_path).exists()
    assert Path(artifacts.persona_readiness_board_path).exists()
    assert Path(artifacts.wireframe_readiness_board_path).exists()
    assert Path(artifacts.interaction_coverage_board_path).exists()
    assert Path(artifacts.open_question_tracker_path).exists()
    assert Path(artifacts.checklist_traceability_board_path).exists()
    assert Path(artifacts.decision_followup_tracker_path).exists()
    assert Path(artifacts.role_matrix_path).exists()
    assert Path(artifacts.role_coverage_board_path).exists()
    assert Path(artifacts.signoff_dependency_board_path).exists()
    assert Path(artifacts.signoff_log_path).exists()
    assert Path(artifacts.signoff_sla_dashboard_path).exists()
    assert Path(artifacts.signoff_reminder_queue_path).exists()
    assert Path(artifacts.reminder_cadence_board_path).exists()
    assert Path(artifacts.signoff_breach_board_path).exists()
    assert Path(artifacts.escalation_dashboard_path).exists()
    assert Path(artifacts.escalation_handoff_ledger_path).exists()
    assert Path(artifacts.handoff_ack_ledger_path).exists()
    assert Path(artifacts.owner_escalation_digest_path).exists()
    assert Path(artifacts.owner_workload_board_path).exists()
    assert Path(artifacts.blocker_log_path).exists()
    assert Path(artifacts.blocker_timeline_path).exists()
    assert Path(artifacts.freeze_exception_board_path).exists()
    assert Path(artifacts.freeze_approval_trail_path).exists()
    assert Path(artifacts.freeze_renewal_tracker_path).exists()
    assert Path(artifacts.exception_log_path).exists()
    assert Path(artifacts.exception_matrix_path).exists()
    assert Path(artifacts.audit_density_board_path).exists()
    assert Path(artifacts.owner_review_queue_path).exists()
    assert Path(artifacts.blocker_timeline_summary_path).exists()
    assert "Decision Log" in Path(artifacts.html_path).read_text()
    assert "Checklist Traceability Board" in Path(artifacts.html_path).read_text()
    assert "Decision Follow-up Tracker" in Path(artifacts.html_path).read_text()
    assert "Review Summary Board" in Path(artifacts.html_path).read_text()
    assert "Objective Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Persona Readiness Board" in Path(artifacts.html_path).read_text()
    assert "Wireframe Readiness Board" in Path(artifacts.html_path).read_text()
    assert "Interaction Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Open Question Tracker" in Path(artifacts.html_path).read_text()
    assert "Role Matrix" in Path(artifacts.html_path).read_text()
    assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in Path(artifacts.review_summary_board_path).read_text()
    assert "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in Path(artifacts.persona_readiness_board_path).read_text()
    assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in Path(artifacts.interaction_coverage_board_path).read_text()
    assert "Role Coverage Board" in Path(artifacts.html_path).read_text()
    assert "Signoff Dependency Board" in Path(artifacts.html_path).read_text()
    assert "Sign-off Log" in Path(artifacts.html_path).read_text()
    assert "Sign-off SLA Dashboard" in Path(artifacts.html_path).read_text()
    assert "Sign-off Reminder Queue" in Path(artifacts.html_path).read_text()
    assert "Reminder Cadence Board" in Path(artifacts.html_path).read_text()
    assert "Sign-off Breach Board" in Path(artifacts.html_path).read_text()
    assert "Escalation Dashboard" in Path(artifacts.html_path).read_text()
    assert "Escalation Handoff Ledger" in Path(artifacts.html_path).read_text()
    assert "Handoff Ack Ledger" in Path(artifacts.html_path).read_text()
    assert "Owner Escalation Digest" in Path(artifacts.html_path).read_text()
    assert "Owner Workload Board" in Path(artifacts.html_path).read_text()
    assert "Blocker Log" in Path(artifacts.html_path).read_text()
    assert "Blocker Timeline" in Path(artifacts.html_path).read_text()
    assert "Review Freeze Exception Board" in Path(artifacts.html_path).read_text()
    assert "Freeze Approval Trail" in Path(artifacts.html_path).read_text()
    assert "Freeze Renewal Tracker" in Path(artifacts.html_path).read_text()
    assert "Review Exceptions" in Path(artifacts.html_path).read_text()
    assert "Review Exception Matrix" in Path(artifacts.html_path).read_text()
    assert "Audit Density Board" in Path(artifacts.html_path).read_text()
    assert "Owner Review Queue" in Path(artifacts.html_path).read_text()
    assert "Blocker Timeline Summary" in Path(artifacts.html_path).read_text()
    assert "dec-triage-handoff-density" in Path(artifacts.decision_log_path).read_text()
    assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in Path(artifacts.objective_coverage_board_path).read_text()
    assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in Path(artifacts.wireframe_readiness_board_path).read_text()
    assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in Path(artifacts.open_question_tracker_path).read_text()
    assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in Path(artifacts.checklist_traceability_board_path).read_text()
    assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in Path(artifacts.decision_followup_tracker_path).read_text()
    assert "role-run-detail-eng-lead" in Path(artifacts.role_matrix_path).read_text()
    assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in Path(artifacts.role_coverage_board_path).read_text()
    assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in Path(artifacts.signoff_dependency_board_path).read_text()
    assert "sig-queue-platform-admin" in Path(artifacts.signoff_log_path).read_text()
    assert "- at-risk: 1" in Path(artifacts.signoff_sla_dashboard_path).read_text()
    assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in Path(artifacts.signoff_reminder_queue_path).read_text()
    assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in Path(artifacts.reminder_cadence_board_path).read_text()
    assert "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director" in Path(artifacts.signoff_breach_board_path).read_text()
    assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in Path(artifacts.escalation_dashboard_path).read_text()
    assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in Path(artifacts.escalation_handoff_ledger_path).read_text()
    assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in Path(artifacts.handoff_ack_ledger_path).read_text()
    assert "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in Path(artifacts.owner_escalation_digest_path).read_text()
    assert "load-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending lane=reminder" in Path(artifacts.owner_workload_board_path).read_text()
    assert "blk-run-detail-copy-final" in Path(artifacts.blocker_log_path).read_text()
    assert "evt-run-detail-copy-opened" in Path(artifacts.blocker_timeline_path).read_text()
    assert "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z" in Path(artifacts.freeze_exception_board_path).read_text()
    assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in Path(artifacts.freeze_approval_trail_path).read_text()
    assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in Path(artifacts.freeze_renewal_tracker_path).read_text()
    assert "exc-blk-run-detail-copy-final" in Path(artifacts.exception_log_path).read_text()
    assert "- product-experience: blockers=1 signoffs=0 total=1" in Path(artifacts.exception_matrix_path).read_text()
    assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in Path(artifacts.audit_density_board_path).read_text()
    assert "- Queue items: 6" in Path(artifacts.owner_review_queue_path).read_text()
    assert "- escalated: 1" in Path(artifacts.blocker_timeline_summary_path).read_text()
