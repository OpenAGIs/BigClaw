from pathlib import Path
from typing import List, Optional

from bigclaw.collaboration import CollaborationComment, DecisionNote, build_collaboration_thread
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
    build_auto_triage_center,
    build_billing_entitlements_page,
    build_billing_entitlements_page_from_ledger,
    build_final_delivery_checklist,
    build_orchestration_canvas,
    build_orchestration_canvas_from_ledger_entry,
    build_orchestration_portfolio,
    build_orchestration_portfolio_from_ledger,
    build_launch_checklist,
    build_takeover_queue_from_ledger,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_billing_entitlements_page,
    render_billing_entitlements_report,
    render_final_delivery_checklist_report,
    render_orchestration_canvas,
    render_orchestration_overview_page,
    render_orchestration_portfolio_report,
    render_issue_validation_report,
    render_launch_checklist_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_repo_narrative_exports,
    render_report_studio_html,
    render_report_studio_plain_text,
    render_report_studio_report,
    render_shared_view_context,
    render_takeover_queue_report,
    render_weekly_repo_evidence_section,
    validation_report_exists,
    write_report,
    write_report_studio_bundle,
    TriageFeedbackRecord,
)

from bigclaw.observability import TaskRun
from bigclaw.models import Task


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
