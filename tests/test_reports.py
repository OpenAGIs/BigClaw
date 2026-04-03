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
