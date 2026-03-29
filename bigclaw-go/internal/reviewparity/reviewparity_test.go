package reviewparity

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	return root
}

func runPythonContract(t *testing.T, script string) {
	t.Helper()
	cmd := exec.Command("python3", "-c", script)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "PYTHONPATH=src")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python contract failed: %v\n%s", err, string(output))
	}
}

const reviewPreamble = `
from pathlib import Path
import tempfile

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
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_audit_density_board,
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
    render_ui_review_persona_readiness_board,
    render_ui_review_review_summary_board,
    render_ui_review_owner_review_queue,
    render_ui_review_owner_workload_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_role_coverage_board,
    render_ui_review_wireframe_readiness_board,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_decision_log,
    render_ui_review_pack_html,
    render_ui_review_pack_report,
    render_ui_review_role_matrix,
    render_ui_review_signoff_log,
    write_ui_review_pack_bundle,
)

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
`

func TestUIReviewPackRoundTripAndBasicAudit(t *testing.T) {
	runPythonContract(t, reviewPreamble+`
pack = build_review_pack()
restored = UIReviewPack.from_dict(pack.to_dict())
assert restored == pack

incomplete = UIReviewPack(
    issue_id="BIG-4204",
    title="UI评审包输出",
    version="v4.0-review-pack",
    objectives=[
        ReviewObjective(
            objective_id="obj-incomplete",
            title="Incomplete objective",
            persona="product-experience",
            outcome="Create a frame for review.",
            success_signal="",
        )
    ],
    wireframes=[
        WireframeSurface(
            surface_id="wf-empty",
            name="Empty frame",
            device="desktop",
            entry_point="Review hub",
        )
    ],
    interactions=[
        InteractionFlow(
            flow_id="flow-empty",
            name="Unspecified interaction",
            trigger="Reviewer opens the page",
            system_response="The system loads the frame.",
        )
    ],
)

audit = UIReviewPackAuditor().audit(incomplete)
assert audit.ready is False
assert audit.missing_sections == ["open_questions"]
assert audit.objectives_missing_signals == ["obj-incomplete"]
assert audit.wireframes_missing_blocks == ["wf-empty"]
assert audit.interactions_missing_states == ["flow-empty"]

audit2 = UIReviewPackAuditor().audit(pack)
assert audit2.ready is True
assert audit2.unresolved_question_ids == ["oq-mobile-depth"]

report = render_ui_review_pack_report(pack, audit2)
assert "# UI Review Pack" in report
assert "- Issue: BIG-4204 UI评审包输出" in report
assert "- Unresolved questions: oq-mobile-depth" in report
`)
}

func TestBuildBig4204ReviewPackReadyAndCoreBoards(t *testing.T) {
	runPythonContract(t, reviewPreamble+`
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
for fragment in [
    "## Review Summary Board",
    "summary-objectives: category=objectives total=4 blocked=1 at-risk=1 covered=2",
    "## Objective Coverage Board",
    "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail",
    "## Persona Readiness Board",
    "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1",
    "## Wireframe Readiness Board",
    "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail",
    "## Interaction Coverage Board",
    "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2",
    "## Open Question Tracker",
    "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue",
]:
    assert fragment in report
`)
}

func TestUIReviewAuditFlagsGovernanceGaps(t *testing.T) {
	runPythonContract(t, reviewPreamble+`
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
assert audit.wireframes_missing_role_assignments == ["wf-queue", "wf-run-detail", "wf-triage"]
assert audit.decisions_missing_role_links == [
    "dec-queue-vp-summary",
    "dec-run-detail-audit-rail",
    "dec-triage-handoff-density",
]

pack = build_big_4204_review_pack()
pack.signoff_log = [
    ReviewSignoff(
        signoff_id="sig-overview-vp-eng",
        assignment_id="role-overview-missing",
        surface_id="wf-overview",
        role="VP Eng",
        status="approved",
        evidence_links=["chk-overview-kpi-scan"],
    )
]
audit = UIReviewPackAuditor().audit(pack)
assert audit.wireframes_missing_signoffs == ["wf-queue", "wf-run-detail", "wf-triage"]
assert audit.signoffs_missing_assignments == ["sig-overview-vp-eng"]

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
assert audit.signoffs_missing_requested_dates == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_due_dates == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_escalation_owners == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_reminder_owners == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_next_reminders == ["sig-run-detail-eng-lead"]
assert audit.signoffs_missing_reminder_cadence == ["sig-run-detail-eng-lead"]
assert audit.signoffs_with_breached_sla == ["sig-run-detail-eng-lead"]

pack = build_big_4204_review_pack()
pack.decision_log[1] = ReviewDecision(
    decision_id="dec-queue-vp-summary",
    surface_id="wf-queue",
    owner="VP Eng",
    summary="Route VP Eng to a summary-first queue state instead of operator controls.",
    rationale="The VP Eng persona needs queue visibility without accidental action affordances.",
    status="proposed",
    follow_up="",
)
audit = UIReviewPackAuditor().audit(pack)
assert audit.unresolved_decision_ids == ["dec-queue-vp-summary"]
assert audit.unresolved_decisions_missing_follow_ups == ["dec-queue-vp-summary"]

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

pack = build_big_4204_review_pack()
pack.blocker_log = []
audit = UIReviewPackAuditor().audit(pack)
assert audit.unresolved_required_signoff_ids == ["sig-run-detail-eng-lead"]
assert audit.unresolved_required_signoffs_without_blockers == ["sig-run-detail-eng-lead"]

pack = build_big_4204_review_pack()
pack.signoff_log[2] = ReviewSignoff(
    signoff_id="sig-run-detail-eng-lead",
    assignment_id="role-run-detail-eng-lead",
    surface_id="wf-run-detail",
    role="Eng Lead",
    status="waived",
    evidence_links=[],
    notes="Design review accepted a temporary waiver pending copy cleanup.",
)
pack.blocker_log = []
pack.blocker_timeline = []
audit = UIReviewPackAuditor().audit(pack)
assert audit.waived_signoffs_missing_metadata == ["sig-run-detail-eng-lead"]

pack = build_big_4204_review_pack()
pack.blocker_timeline = []
audit = UIReviewPackAuditor().audit(pack)
assert audit.blockers_missing_timeline_events == ["blk-run-detail-copy-final"]

pack = build_big_4204_review_pack()
pack.blocker_log[0] = ReviewBlocker(
    blocker_id="blk-run-detail-copy-final",
    surface_id="wf-run-detail",
    signoff_id="sig-run-detail-eng-lead",
    owner="product-experience",
    summary="Replay-state copy review is closed pending audit trail confirmation.",
    status="closed",
    severity="medium",
    escalation_owner="design-program-manager",
    next_action="Archive the blocker after the final review sync.",
)
pack.blocker_timeline = [
    ReviewBlockerEvent(
        event_id="evt-run-detail-copy-opened",
        blocker_id="blk-run-detail-copy-final",
        actor="product-experience",
        status="opened",
        summary="Tracked the replay-state wording gap during review prep.",
        timestamp="2026-03-13T10:00:00Z",
        next_action="Review wording changes with Eng Lead.",
    ),
    ReviewBlockerEvent(
        event_id="evt-orphan-blocker",
        blocker_id="blk-missing",
        actor="design-program-manager",
        status="escalated",
        summary="Unexpected timeline entry remained after blocker merge cleanup.",
        timestamp="2026-03-14T11:00:00Z",
        next_action="Remove orphaned timeline entry from the bundle.",
    ),
]
audit = UIReviewPackAuditor().audit(pack)
assert audit.closed_blockers_missing_resolution_events == ["blk-run-detail-copy-final"]
assert audit.orphan_blocker_timeline_blocker_ids == ["blk-missing"]
`)
}

func TestUIReviewRendersOperationalBoards(t *testing.T) {
	runPythonContract(t, reviewPreamble+`
pack = build_big_4204_review_pack()
for content, fragments in [
    (render_ui_review_signoff_sla_dashboard(pack), ["# UI Review Sign-off SLA Dashboard", "- at-risk: 1", "sig-run-detail-eng-lead: role=Eng Lead surface=wf-run-detail status=pending sla=at-risk requested_at=2026-03-12T11:00:00Z due_at=2026-03-15T18:00:00Z escalation_owner=engineering-director"]),
    (render_ui_review_signoff_reminder_queue(pack), ["# UI Review Sign-off Reminder Queue", "- Reminders: 1", "rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk owner=design-program-manager channel=slack"]),
    (render_ui_review_signoff_breach_board(pack), ["# UI Review Sign-off Breach Board", "- Breach items: 1", "breach-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail status=pending sla=at-risk escalation_owner=engineering-director"]),
    (render_ui_review_escalation_dashboard(pack), ["# UI Review Escalation Dashboard", "- engineering-director: blockers=0 signoffs=1 total=1", "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead"]),
    (render_ui_review_escalation_handoff_ledger(pack), ["# UI Review Escalation Handoff Ledger", "- Handoffs: 1", "handoff-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z", "from=product-experience to=Eng Lead channel=design-critique artifact=wf-run-detail#copy-v5"]),
    (render_ui_review_owner_escalation_digest(pack), ["# UI Review Owner Escalation Digest", "- design-program-manager: blockers=1 signoffs=0 reminders=1 freezes=0 handoffs=0 total=2", "digest-rem-sig-run-detail-eng-lead: owner=design-program-manager type=reminder source=sig-run-detail-eng-lead surface=wf-run-detail status=pending"]),
    (render_ui_review_exception_matrix(pack), ["# UI Review Exception Matrix", "- product-experience: blockers=1 signoffs=0 total=1"]),
    (render_ui_review_freeze_exception_board(pack), ["# UI Review Freeze Exception Board", "freeze-blk-run-detail-copy-final: owner=release-director type=blocker source=blk-run-detail-copy-final surface=wf-run-detail status=open window=2026-03-18T18:00:00Z"]),
    (render_ui_review_freeze_approval_trail(pack), ["# UI Review Freeze Approval Trail", "freeze-approval-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open owner=release-director approved_by=release-director approved_at=2026-03-14T08:30:00Z window=2026-03-18T18:00:00Z"]),
    (render_ui_review_review_summary_board(pack), ["# UI Review Review Summary Board", "summary-personas: category=personas total=4 blocked=1 at-risk=1 ready=2", "summary-actions: category=actions total=8 queue=6 reminder=1 renewal=1"]),
    (render_ui_review_persona_readiness_board(pack), ["# UI Review Persona Readiness Board", "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1"]),
    (render_ui_review_interaction_coverage_board(pack), ["# UI Review Interaction Coverage Board", "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2"]),
    (render_ui_review_objective_coverage_board(pack), ["# UI Review Objective Coverage Board", "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail"]),
    (render_ui_review_wireframe_readiness_board(pack), ["# UI Review Wireframe Readiness Board", "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail"]),
    (render_ui_review_open_question_tracker(pack), ["# UI Review Open Question Tracker", "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue"]),
    (render_ui_review_checklist_traceability_board(pack), ["# UI Review Checklist Traceability Board", "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience"]),
    (render_ui_review_decision_followup_tracker(pack), ["# UI Review Decision Follow-up Tracker", "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience"]),
    (render_ui_review_role_coverage_board(pack), ["# UI Review Role Coverage Board", "cover-role-run-detail-eng-lead: assignment=role-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=ready responsibilities=2 checklist=1 decisions=1"]),
    (render_ui_review_signoff_dependency_board(pack), ["# UI Review Signoff Dependency Board", "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final"]),
    (render_ui_review_owner_workload_board(pack), ["# UI Review Owner Workload Board", "load-renew-blk-run-detail-copy-final: owner=release-director type=renewal source=blk-run-detail-copy-final surface=wf-run-detail status=review-needed lane=renewal"]),
    (render_ui_review_audit_density_board(pack), ["# UI Review Audit Density Board", "density-wf-run-detail: surface=wf-run-detail artifact_total=9 open_total=4 band=dense"]),
    (render_ui_review_owner_review_queue(pack), ["# UI Review Owner Review Queue", "- Queue items: 6", "- queue-sig-run-detail-eng-lead: owner=Eng Lead type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending"]),
    (render_ui_review_exception_log(pack), ["# UI Review Exception Log", "exc-blk-run-detail-copy-final", "evt-run-detail-copy-escalated/escalated/design-program-manager@2026-03-14T09:30:00Z"]),
    (render_ui_review_blocker_timeline_summary(pack), ["# UI Review Blocker Timeline Summary", "- Events: 2", "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z"]),
]:
    for fragment in fragments:
        assert fragment in content
`)
}

func TestUIReviewHTMLAndBundleExport(t *testing.T) {
	runPythonContract(t, reviewPreamble+`
pack = build_big_4204_review_pack()
audit = UIReviewPackAuditor().audit(pack)
html = render_ui_review_pack_html(pack, audit)
for fragment in [
    "<h2>Decision Log</h2>",
    "<h2>Checklist Traceability Board</h2>",
    "<h2>Decision Follow-up Tracker</h2>",
    "<h2>Review Summary Board</h2>",
    "<h2>Objective Coverage Board</h2>",
    "<h2>Persona Readiness Board</h2>",
    "<h2>Wireframe Readiness Board</h2>",
    "<h2>Interaction Coverage Board</h2>",
    "<h2>Open Question Tracker</h2>",
    "<h2>Role Matrix</h2>",
    "<h2>Role Coverage Board</h2>",
    "<h2>Signoff Dependency Board</h2>",
    "<h2>Sign-off Log</h2>",
    "<h2>Sign-off SLA Dashboard</h2>",
    "<h2>Sign-off Reminder Queue</h2>",
    "<h2>Reminder Cadence Board</h2>",
    "<h2>Sign-off Breach Board</h2>",
    "<h2>Escalation Dashboard</h2>",
    "<h2>Escalation Handoff Ledger</h2>",
    "<h2>Handoff Ack Ledger</h2>",
    "<h2>Owner Escalation Digest</h2>",
    "<h2>Owner Workload Board</h2>",
    "<h2>Blocker Log</h2>",
    "<h2>Blocker Timeline</h2>",
    "<h2>Review Freeze Exception Board</h2>",
    "<h2>Freeze Approval Trail</h2>",
    "<h2>Freeze Renewal Tracker</h2>",
    "<h2>Review Exceptions</h2>",
    "<h2>Review Exception Matrix</h2>",
    "<h2>Audit Density Board</h2>",
    "<h2>Owner Review Queue</h2>",
    "<h2>Blocker Timeline Summary</h2>",
]:
    assert fragment in html

with tempfile.TemporaryDirectory() as tmp:
    artifacts = write_ui_review_pack_bundle(str(Path(tmp)), pack)
    assert Path(artifacts.markdown_path).exists()
    assert Path(artifacts.html_path).exists()
    assert Path(artifacts.decision_log_path).exists()
    assert "review-pack" in artifacts.markdown_path
`)
}
