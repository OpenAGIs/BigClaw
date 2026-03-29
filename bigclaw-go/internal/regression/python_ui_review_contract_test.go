package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonUIReviewContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "ui_review_contract.py")
	script := `import json
import tempfile
import sys

from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewObjective,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_audit_density_board,
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
    render_ui_review_pack_html,
    render_ui_review_pack_report,
    render_ui_review_persona_readiness_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_review_summary_board,
    render_ui_review_role_coverage_board,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_log,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_wireframe_readiness_board,
    render_ui_review_decision_log,
    render_ui_review_role_matrix,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    write_ui_review_pack_bundle,
)

minimal_pack = UIReviewPack(
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
minimal_restored = UIReviewPack.from_dict(minimal_pack.to_dict())
minimal_audit = UIReviewPackAuditor().audit(minimal_pack)
minimal_report = render_ui_review_pack_report(minimal_pack, minimal_audit)

incomplete_pack = UIReviewPack(
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
incomplete_audit = UIReviewPackAuditor().audit(incomplete_pack)

pack = build_big_4204_review_pack()
audit = UIReviewPackAuditor().audit(pack)
report = render_ui_review_pack_report(pack, audit)
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

with tempfile.TemporaryDirectory() as td:
    artifacts = write_ui_review_pack_bundle(td, pack)
    bundle_checks = {
        "markdown_exists": Path(artifacts.markdown_path).exists(),
        "html_exists": Path(artifacts.html_path).exists(),
        "decision_log_exists": Path(artifacts.decision_log_path).exists(),
        "summary_exists": Path(artifacts.review_summary_board_path).exists(),
        "timeline_summary_exists": Path(artifacts.blocker_timeline_summary_path).exists(),
    }

print(json.dumps({
    "minimal": {
        "round_trip_ok": minimal_restored == minimal_pack,
        "ready": minimal_audit.ready,
        "unresolved_question_ids": minimal_audit.unresolved_question_ids,
        "report_checks": {
            "title": "# UI Review Pack" in minimal_report,
            "issue": "- Issue: BIG-4204 UI评审包输出" in minimal_report,
            "summary": "- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0 blockers=0 timeline_events=0" in minimal_report,
            "objective": "- obj-alignment: Align reviewers on the release-control story persona=product-experience priority=P0" in minimal_report,
            "question": "- Unresolved questions: oq-mobile-depth" in minimal_report,
        },
    },
    "incomplete": {
        "ready": incomplete_audit.ready,
        "missing_sections": incomplete_audit.missing_sections,
        "objectives_missing_signals": incomplete_audit.objectives_missing_signals,
        "wireframes_missing_blocks": incomplete_audit.wireframes_missing_blocks,
        "interactions_missing_states": incomplete_audit.interactions_missing_states,
    },
    "release_pack": {
        "ready": audit.ready,
        "counts": {
            "objectives": len(pack.objectives),
            "wireframes": len(pack.wireframes),
            "interactions": len(pack.interactions),
            "open_questions": len(pack.open_questions),
            "reviewer_checklist": len(pack.reviewer_checklist),
            "decision_log": len(pack.decision_log),
            "role_matrix": len(pack.role_matrix),
            "signoff_log": len(pack.signoff_log),
            "blocker_log": len(pack.blocker_log),
            "blocker_timeline": len(pack.blocker_timeline),
        },
        "report_checks": {
            "review_summary": "## Review Summary Board" in report,
            "objective_coverage": "objcov-obj-run-detail-investigation: objective=obj-run-detail-investigation persona=Eng Lead priority=P0 coverage=blocked dependencies=3 surfaces=wf-run-detail" in report,
            "persona_readiness": "persona-eng-lead: persona=Eng Lead readiness=blocked objectives=1 assignments=1 signoffs=1 open_questions=0 queue_items=1 blockers=1" in report,
            "wireframe_readiness": "wire-wf-run-detail: surface=wf-run-detail device=desktop readiness=blocked open_total=4 entry=/runs/detail" in report,
            "interaction_coverage": "intcov-flow-triage-handoff: flow=flow-triage-handoff surfaces=wf-triage owners=Cross-Team Operator,Platform Admin coverage=covered states=4 exceptions=2" in report,
            "open_question_tracker": "qtrack-oq-role-density: question=oq-role-density owner=product-experience theme=role-matrix status=open link_status=linked surfaces=wf-queue" in report,
            "signoff_dependency": "dep-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead surface=wf-run-detail role=Eng Lead status=pending dependency_status=blocked blockers=blk-run-detail-copy-final" in report,
            "escalation_dashboard": "esc-sig-run-detail-eng-lead: owner=engineering-director type=signoff source=sig-run-detail-eng-lead surface=wf-run-detail status=pending priority=at-risk current_owner=Eng Lead" in report,
        },
        "surface_checks": {
            "checklist_traceability": "trace-chk-queue-role-density: item=chk-queue-role-density surface=wf-queue owner=product-experience status=open linked_roles=product-experience" in checklist_traceability,
            "decision_followup": "follow-dec-queue-vp-summary: decision=dec-queue-vp-summary surface=wf-queue owner=VP Eng status=proposed linked_roles=Platform Admin,product-experience" in decision_followup,
            "reminder_cadence": "cad-rem-sig-run-detail-eng-lead: signoff=sig-run-detail-eng-lead role=Eng Lead surface=wf-run-detail cadence=daily status=scheduled owner=design-program-manager" in reminder_cadence,
            "handoff_ack": "ack-evt-run-detail-copy-escalated: event=evt-run-detail-copy-escalated blocker=blk-run-detail-copy-final surface=wf-run-detail handoff_to=Eng Lead ack_owner=Eng Lead ack_status=acknowledged ack_at=2026-03-14T10:15:00Z" in handoff_ack,
            "freeze_renewal": "renew-blk-run-detail-copy-final: blocker=blk-run-detail-copy-final surface=wf-run-detail status=open renewal_owner=release-director renewal_by=2026-03-17T12:00:00Z renewal_status=review-needed" in freeze_renewal,
            "timeline_summary": "- blk-run-detail-copy-final: latest=evt-run-detail-copy-escalated actor=design-program-manager status=escalated at=2026-03-14T09:30:00Z" in timeline_summary,
        },
        "html_checks": {
            "decision_log": "<h2>Decision Log</h2>" in html,
            "checklist_traceability": "<h2>Checklist Traceability Board</h2>" in html,
            "timeline_summary": "<h2>Blocker Timeline Summary</h2>" in html,
            "decision_id": "dec-queue-vp-summary" in html,
        },
        "board_titles": {
            "decision_log": "# UI Review Decision Log" in decision_log,
            "review_summary": "# UI Review Review Summary Board" in review_summary,
            "objective_coverage": "# UI Review Objective Coverage Board" in objective_coverage,
            "persona_readiness": "# UI Review Persona Readiness Board" in persona_readiness,
            "wireframe_readiness": "# UI Review Wireframe Readiness Board" in wireframe_readiness,
            "interaction_coverage": "# UI Review Interaction Coverage Board" in interaction_coverage,
            "question_tracker": "# UI Review Open Question Tracker" in question_tracker,
            "role_matrix": "# UI Review Role Matrix" in role_matrix,
            "role_coverage": "# UI Review Role Coverage Board" in role_coverage,
            "signoff_dependency": "# UI Review Signoff Dependency Board" in signoff_dependency,
            "signoff_log": "# UI Review Sign-off Log" in signoff_log,
            "blocker_log": "# UI Review Blocker Log" in blocker_log,
            "blocker_timeline": "# UI Review Blocker Timeline" in blocker_timeline,
            "signoff_sla": "# UI Review Sign-off SLA Dashboard" in signoff_sla,
            "signoff_reminder": "# UI Review Sign-off Reminder Queue" in signoff_reminder,
            "signoff_breach": "# UI Review Sign-off Breach Board" in signoff_breach,
            "escalation_dashboard": "# UI Review Escalation Dashboard" in escalation_dashboard,
            "handoff_ledger": "# UI Review Escalation Handoff Ledger" in handoff_ledger,
            "owner_digest": "# UI Review Owner Escalation Digest" in owner_digest,
            "owner_workload": "# UI Review Owner Workload Board" in owner_workload,
            "freeze_board": "# UI Review Freeze Exception Board" in freeze_board,
            "freeze_trail": "# UI Review Freeze Approval Trail" in freeze_trail,
            "exception_log": "# UI Review Exception Log" in exception_log,
            "exception_matrix": "# UI Review Exception Matrix" in exception_matrix,
            "audit_density": "# UI Review Audit Density Board" in audit_density,
            "owner_review_queue": "# UI Review Owner Review Queue" in owner_review_queue,
        },
        "bundle_checks": bundle_checks,
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write ui review contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run ui review contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Minimal struct {
			RoundTripOK           bool     `json:"round_trip_ok"`
			Ready                 bool     `json:"ready"`
			UnresolvedQuestionIDs []string `json:"unresolved_question_ids"`
			ReportChecks          struct {
				Title     bool `json:"title"`
				Issue     bool `json:"issue"`
				Summary   bool `json:"summary"`
				Objective bool `json:"objective"`
				Question  bool `json:"question"`
			} `json:"report_checks"`
		} `json:"minimal"`
		Incomplete struct {
			Ready                     bool     `json:"ready"`
			MissingSections           []string `json:"missing_sections"`
			ObjectivesMissingSignals  []string `json:"objectives_missing_signals"`
			WireframesMissingBlocks   []string `json:"wireframes_missing_blocks"`
			InteractionsMissingStates []string `json:"interactions_missing_states"`
		} `json:"incomplete"`
		ReleasePack struct {
			Ready  bool `json:"ready"`
			Counts struct {
				Objectives        int `json:"objectives"`
				Wireframes        int `json:"wireframes"`
				Interactions      int `json:"interactions"`
				OpenQuestions     int `json:"open_questions"`
				ReviewerChecklist int `json:"reviewer_checklist"`
				DecisionLog       int `json:"decision_log"`
				RoleMatrix        int `json:"role_matrix"`
				SignoffLog        int `json:"signoff_log"`
				BlockerLog        int `json:"blocker_log"`
				BlockerTimeline   int `json:"blocker_timeline"`
			} `json:"counts"`
			ReportChecks  map[string]bool `json:"report_checks"`
			SurfaceChecks map[string]bool `json:"surface_checks"`
			HTMLChecks    map[string]bool `json:"html_checks"`
			BoardTitles   map[string]bool `json:"board_titles"`
			BundleChecks  map[string]bool `json:"bundle_checks"`
		} `json:"release_pack"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode ui review contract output: %v\n%s", err, string(output))
	}

	if !decoded.Minimal.RoundTripOK || !decoded.Minimal.Ready || len(decoded.Minimal.UnresolvedQuestionIDs) != 1 || decoded.Minimal.UnresolvedQuestionIDs[0] != "oq-mobile-depth" {
		t.Fatalf("unexpected minimal ui review payload: %+v", decoded.Minimal)
	}
	for key, ok := range map[string]bool{
		"title":     decoded.Minimal.ReportChecks.Title,
		"issue":     decoded.Minimal.ReportChecks.Issue,
		"summary":   decoded.Minimal.ReportChecks.Summary,
		"objective": decoded.Minimal.ReportChecks.Objective,
		"question":  decoded.Minimal.ReportChecks.Question,
	} {
		if !ok {
			t.Fatalf("expected minimal report check %s to pass", key)
		}
	}
	if decoded.Incomplete.Ready || len(decoded.Incomplete.MissingSections) != 1 || decoded.Incomplete.MissingSections[0] != "open_questions" ||
		len(decoded.Incomplete.ObjectivesMissingSignals) != 1 || decoded.Incomplete.ObjectivesMissingSignals[0] != "obj-incomplete" ||
		len(decoded.Incomplete.WireframesMissingBlocks) != 1 || decoded.Incomplete.WireframesMissingBlocks[0] != "wf-empty" ||
		len(decoded.Incomplete.InteractionsMissingStates) != 1 || decoded.Incomplete.InteractionsMissingStates[0] != "flow-empty" {
		t.Fatalf("unexpected incomplete ui review payload: %+v", decoded.Incomplete)
	}
	if !decoded.ReleasePack.Ready ||
		decoded.ReleasePack.Counts.Objectives != 4 ||
		decoded.ReleasePack.Counts.Wireframes != 4 ||
		decoded.ReleasePack.Counts.Interactions != 4 ||
		decoded.ReleasePack.Counts.OpenQuestions != 3 ||
		decoded.ReleasePack.Counts.ReviewerChecklist != 8 ||
		decoded.ReleasePack.Counts.DecisionLog != 4 ||
		decoded.ReleasePack.Counts.RoleMatrix != 8 ||
		decoded.ReleasePack.Counts.SignoffLog != 4 ||
		decoded.ReleasePack.Counts.BlockerLog != 1 ||
		decoded.ReleasePack.Counts.BlockerTimeline != 2 {
		t.Fatalf("unexpected release ui review counts: %+v", decoded.ReleasePack.Counts)
	}
	for name, ok := range decoded.ReleasePack.ReportChecks {
		if !ok {
			t.Fatalf("expected ui review report check %s to pass", name)
		}
	}
	for name, ok := range decoded.ReleasePack.SurfaceChecks {
		if !ok {
			t.Fatalf("expected ui review surface check %s to pass", name)
		}
	}
	for name, ok := range decoded.ReleasePack.HTMLChecks {
		if !ok {
			t.Fatalf("expected ui review html check %s to pass", name)
		}
	}
	for name, ok := range decoded.ReleasePack.BoardTitles {
		if !ok {
			t.Fatalf("expected ui review board title %s to pass", name)
		}
	}
	for name, ok := range decoded.ReleasePack.BundleChecks {
		if !ok {
			t.Fatalf("expected ui review bundle check %s to pass", name)
		}
	}
}
