package uireview_test

import (
	"testing"

	"bigclaw-go/internal/testharness"
)

func runLegacyUIReviewPython(t *testing.T, script string) {
	t.Helper()
	cmd := testharness.PythonCommand(t, "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("legacy ui_review python check failed: %v\n%s", err, string(output))
	}
}

func TestLegacyPythonUIReviewPackAuditAndRoundTrip(t *testing.T) {
	runLegacyUIReviewPython(t, `
from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewObjective,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
)

pack = UIReviewPack(
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

restored = UIReviewPack.from_dict(pack.to_dict())
assert restored == pack

audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is True
assert audit.unresolved_question_ids == ["oq-mobile-depth"]
assert audit.missing_sections == []

big = build_big_4204_review_pack()
big_audit = UIReviewPackAuditor().audit(big)
assert big_audit.ready is True
assert len(big.objectives) == 4
assert len(big.wireframes) == 4
assert len(big.interactions) == 4
assert len(big.open_questions) == 3
assert len(big.reviewer_checklist) == 8
assert len(big.decision_log) == 4
assert len(big.role_matrix) == 8
assert len(big.signoff_log) == 4
assert len(big.blocker_log) == 1
assert len(big.blocker_timeline) == 2
`)
}

func TestLegacyPythonUIReviewAuditFailureModes(t *testing.T) {
	runLegacyUIReviewPython(t, `
from bigclaw.ui_review import (
    ReviewBlocker,
    ReviewBlockerEvent,
    ReviewDecision,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPackAuditor,
    build_big_4204_review_pack,
)

pack = build_big_4204_review_pack()
pack.reviewer_checklist = [
    ReviewerChecklistItem(
        item_id="chk-overview-kpi-scan",
        surface_id="wf-overview",
        prompt="Verify the KPI strip still supports one-screen executive scanning before drill-down.",
        owner="VP Eng",
        status="ready",
        evidence_links=[],
    )
]
audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is False
assert audit.wireframes_missing_checklists == ["wf-queue", "wf-run-detail", "wf-triage"]
assert audit.checklist_items_missing_evidence == ["chk-overview-kpi-scan"]

pack = build_big_4204_review_pack()
pack.decision_log = [
    ReviewDecision(
        decision_id="dec-overview-alert-stack",
        surface_id="wf-overview",
        owner="product-experience",
        summary="Keep approval and regression alerts in one stacked priority rail.",
        rationale="Reviewers need one comparison lane before jumping into queue or triage surfaces.",
        status="accepted",
    )
]
audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is False
assert audit.wireframes_missing_decisions == ["wf-queue", "wf-run-detail", "wf-triage"]

pack = build_big_4204_review_pack()
pack.role_matrix = [
    ReviewRoleAssignment(
        assignment_id="role-overview-vp-eng",
        surface_id="wf-overview",
        role="VP Eng",
        responsibilities=["approve overview scan path"],
        checklist_item_ids=["chk-overview-kpi-scan"],
        decision_ids=["dec-overview-alert-stack"],
        status="ready",
    )
]
audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is False
assert audit.wireframes_missing_role_assignments == ["wf-queue", "wf-run-detail", "wf-triage"]
assert "chk-queue-batch-approval" in audit.checklist_items_missing_role_links
assert "dec-queue-vp-summary" in audit.decisions_missing_role_links

pack = build_big_4204_review_pack()
pack.signoff_log[2] = ReviewSignoff(
    signoff_id="sig-run-detail-eng-lead",
    assignment_id="role-run-detail-eng-lead",
    surface_id="wf-run-detail",
    role="Eng Lead",
    status="pending",
    evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
    notes="Waiting for final replay-state copy review.",
    requested_at="",
    due_at="",
    escalation_owner="",
    sla_status="breached",
    reminder_owner="",
    reminder_channel="slack",
    last_reminder_at="2026-03-14T09:45:00Z",
    next_reminder_at="",
)
audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is False
assert audit.signoffs_missing_requested_dates == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_due_dates == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_escalation_owners == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_reminder_owners == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_next_reminders == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_reminder_cadence == ["sig-run-detail-eng-lead"]
assert audit.signoffs_with_breached_sla == ["sig-run-detail-eng-lead"]

pack = build_big_4204_review_pack()
pack.blocker_log[0] = ReviewBlocker(
    blocker_id="blk-run-detail-copy-final",
    surface_id="wf-run-detail",
    signoff_id="sig-run-detail-eng-lead",
    owner="product-experience",
    summary="Replay-state copy still needs final wording review before Eng Lead signoff can close.",
    status="open",
    severity="medium",
    escalation_owner="design-program-manager",
    next_action="Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",
    freeze_exception=True,
    freeze_owner="",
    freeze_until="",
    freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
    freeze_approved_by="",
    freeze_approved_at="",
)
pack.blocker_timeline[1] = ReviewBlockerEvent(
    event_id="evt-run-detail-copy-escalated",
    blocker_id="blk-run-detail-copy-final",
    actor="design-program-manager",
    status="escalated",
    summary="Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
    timestamp="2026-03-14T09:30:00Z",
    next_action="Refresh the run-detail frame annotations after the wording review completes.",
    handoff_from="product-experience",
    handoff_to="",
    channel="design-critique",
    artifact_ref="",
)
audit = UIReviewPackAuditor().audit(pack)
assert audit.ready is False
assert audit.freeze_exceptions_missing_owners == ["blk-run-detail-copy-final"]
assert audit.freeze_exceptions_missing_until == ["blk-run-detail-copy-final"]
assert audit.freeze_exceptions_missing_approvers == ["blk-run-detail-copy-final"]
assert audit.freeze_exceptions_missing_approval_dates == ["blk-run-detail-copy-final"]
assert audit.freeze_exceptions_missing_renewal_owners == ["blk-run-detail-copy-final"]
assert audit.freeze_exceptions_missing_renewal_dates == ["blk-run-detail-copy-final"]
assert audit.handoff_events_missing_targets == ["evt-run-detail-copy-escalated"]
assert audit.handoff_events_missing_artifacts == ["evt-run-detail-copy-escalated"]
assert audit.handoff_events_missing_ack_owners == ["evt-run-detail-copy-escalated"]
assert audit.handoff_events_missing_ack_dates == ["evt-run-detail-copy-escalated"]
`)
}

func TestLegacyPythonUIReviewRenderSurfaces(t *testing.T) {
	runLegacyUIReviewPython(t, `
from bigclaw.ui_review import (
    UIReviewPackAuditor,
    build_big_4204_review_pack,
    render_ui_review_audit_density_board,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_checklist_traceability_board,
    render_ui_review_decision_followup_tracker,
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
    render_ui_review_decision_log,
)

pack = build_big_4204_review_pack()
audit = UIReviewPackAuditor().audit(pack)

report = render_ui_review_pack_report(pack, audit)
assert "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2" in report
assert "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in report
assert "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in report
assert "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in report
assert "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in report
assert "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in report
assert "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in report
assert "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1" in report
assert "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in report
assert "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director" in report
assert "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack" in report
assert "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in report
assert "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in report
assert "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in report
assert "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in report
assert "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z" in report
assert "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in report
assert "exc-blk-run-detail-copy-final: type=blocker source=blk-run-detail-copy-final surface=wf-run-detail owner=product-experience status=open severity=medium" in report
assert "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense" in report
assert "queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending" in report
assert "- escalated: 1" in report

assert "# UI Review Decision Log" in render_ui_review_decision_log(pack)
assert "# UI Review Role Matrix" in render_ui_review_role_matrix(pack)
assert "# UI Review Objective Coverage Board" in render_ui_review_objective_coverage_board(pack)
assert "# UI Review Wireframe Readiness Board" in render_ui_review_wireframe_readiness_board(pack)
assert "# UI Review Open Question Tracker" in render_ui_review_open_question_tracker(pack)
assert "# UI Review Review Summary Board" in render_ui_review_review_summary_board(pack)
assert "# UI Review Persona Readiness Board" in render_ui_review_persona_readiness_board(pack)
assert "# UI Review Interaction Coverage Board" in render_ui_review_interaction_coverage_board(pack)
assert "# UI Review Checklist Traceability Board" in render_ui_review_checklist_traceability_board(pack)
assert "# UI Review Decision Follow-up Tracker" in render_ui_review_decision_followup_tracker(pack)
assert "# UI Review Role Coverage Board" in render_ui_review_role_coverage_board(pack)
assert "# UI Review Signoff Dependency Board" in render_ui_review_signoff_dependency_board(pack)
assert "# UI Review Audit Density Board" in render_ui_review_audit_density_board(pack)
assert "# UI Review Sign-off Log" in render_ui_review_signoff_log(pack)
assert "# UI Review Sign-off SLA Dashboard" in render_ui_review_signoff_sla_dashboard(pack)
assert "# UI Review Sign-off Reminder Queue" in render_ui_review_signoff_reminder_queue(pack)
assert "# UI Review Reminder Cadence Board" in render_ui_review_reminder_cadence_board(pack)
assert "# UI Review Sign-off Breach Board" in render_ui_review_signoff_breach_board(pack)
assert "# UI Review Escalation Dashboard" in render_ui_review_escalation_dashboard(pack)
assert "# UI Review Escalation Handoff Ledger" in render_ui_review_escalation_handoff_ledger(pack)
assert "# UI Review Handoff Ack Ledger" in render_ui_review_handoff_ack_ledger(pack)
assert "# UI Review Owner Escalation Digest" in render_ui_review_owner_escalation_digest(pack)
assert "# UI Review Owner Workload Board" in render_ui_review_owner_workload_board(pack)
assert "# UI Review Freeze Exception Board" in render_ui_review_freeze_exception_board(pack)
assert "# UI Review Freeze Approval Trail" in render_ui_review_freeze_approval_trail(pack)
assert "# UI Review Freeze Renewal Tracker" in render_ui_review_freeze_renewal_tracker(pack)
assert "# UI Review Blocker Log" in render_ui_review_blocker_log(pack)
assert "# UI Review Blocker Timeline" in render_ui_review_blocker_timeline(pack)
assert "# UI Review Exception Log" in render_ui_review_exception_log(pack)
assert "# UI Review Exception Matrix" in render_ui_review_exception_matrix(pack)
assert "# UI Review Owner Review Queue" in render_ui_review_owner_review_queue(pack)
assert "# UI Review Blocker Timeline Summary" in render_ui_review_blocker_timeline_summary(pack)
`)
}

func TestLegacyPythonUIReviewBundleWriter(t *testing.T) {
	runLegacyUIReviewPython(t, `
from pathlib import Path
from tempfile import TemporaryDirectory

from bigclaw.ui_review import (
    build_big_4204_review_pack,
    write_ui_review_pack_bundle,
)

pack = build_big_4204_review_pack()
with TemporaryDirectory() as tmp:
    artifacts = write_ui_review_pack_bundle(tmp, pack)
    expected = [
        artifacts.markdown_path,
        artifacts.html_path,
        artifacts.decision_log_path,
        artifacts.review_summary_board_path,
        artifacts.objective_coverage_board_path,
        artifacts.persona_readiness_board_path,
        artifacts.wireframe_readiness_board_path,
        artifacts.interaction_coverage_board_path,
        artifacts.open_question_tracker_path,
        artifacts.checklist_traceability_board_path,
        artifacts.decision_followup_tracker_path,
        artifacts.role_matrix_path,
        artifacts.role_coverage_board_path,
        artifacts.signoff_dependency_board_path,
        artifacts.signoff_log_path,
        artifacts.signoff_sla_dashboard_path,
        artifacts.signoff_reminder_queue_path,
        artifacts.reminder_cadence_board_path,
        artifacts.signoff_breach_board_path,
        artifacts.escalation_dashboard_path,
        artifacts.escalation_handoff_ledger_path,
        artifacts.handoff_ack_ledger_path,
        artifacts.owner_escalation_digest_path,
        artifacts.owner_workload_board_path,
        artifacts.blocker_log_path,
        artifacts.blocker_timeline_path,
        artifacts.freeze_exception_board_path,
        artifacts.freeze_approval_trail_path,
        artifacts.freeze_renewal_tracker_path,
        artifacts.exception_log_path,
        artifacts.exception_matrix_path,
        artifacts.audit_density_board_path,
        artifacts.owner_review_queue_path,
        artifacts.blocker_timeline_summary_path,
    ]
    for candidate in expected:
        assert Path(candidate).exists(), candidate

    html = Path(artifacts.html_path).read_text()
    assert "Checklist Traceability Board" in html
    assert "Decision Follow-up Tracker" in html
    assert "Review Summary Board" in html
    assert "Sign-off Breach Board" in html
    assert "Blocker Timeline Summary" in html
`)
}
