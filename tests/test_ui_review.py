from pathlib import Path

from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewDecision,
    ReviewObjective,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
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


def test_ui_review_pack_round_trip_preserves_manifest_shape() -> None:
    pack = build_review_pack()

    restored = UIReviewPack.from_dict(pack.to_dict())

    assert restored == pack


def test_ui_review_pack_audit_flags_missing_sections_and_coverage_gaps() -> None:
    pack = UIReviewPack(
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

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is False
    assert audit.missing_sections == ["open_questions"]
    assert audit.objectives_missing_signals == ["obj-incomplete"]
    assert audit.wireframes_missing_blocks == ["wf-empty"]
    assert audit.interactions_missing_states == ["flow-empty"]


def test_ui_review_pack_audit_allows_open_questions_while_marking_pack_ready() -> None:
    pack = build_review_pack()

    audit = UIReviewPackAuditor().audit(pack)

    assert audit.ready is True
    assert audit.unresolved_question_ids == ["oq-mobile-depth"]
    assert audit.missing_sections == []


def test_render_ui_review_pack_report_summarizes_review_shape_and_findings() -> None:
    pack = build_review_pack()
    audit = UIReviewPackAuditor().audit(pack)

    report = render_ui_review_pack_report(pack, audit)

    assert "# UI Review Pack" in report
    assert "- Issue: BIG-4204 UI评审包输出" in report
    assert "- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1 checklist=0 decisions=0 role_assignments=0 signoffs=0" in report
    assert (
        "- obj-alignment: Align reviewers on the release-control story "
        "persona=product-experience priority=P0"
    ) in report
    assert "- Unresolved questions: oq-mobile-depth" in report


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
    assert pack.requires_reviewer_checklist is True
    assert pack.requires_decision_log is True
    assert pack.requires_role_matrix is True
    assert pack.requires_signoff_log is True
    assert "obj-queue-governance" in report
    assert "wf-triage: Triage and handoff board" in report
    assert "flow-run-replay: Run replay with evidence audit" in report
    assert "chk-queue-batch-approval: surface=wf-queue owner=Platform Admin status=ready" in report
    assert "dec-queue-vp-summary: surface=wf-queue owner=VP Eng status=proposed" in report
    assert "role-queue-platform-admin: surface=wf-queue role=Platform Admin status=ready" in report
    assert "sig-run-detail-eng-lead: surface=wf-run-detail role=Eng Lead assignment=role-run-detail-eng-lead status=pending" in report
    assert "- Wireframes missing checklist coverage: none" in report
    assert "- Wireframes missing decision coverage: none" in report
    assert "- Wireframes missing role assignments: none" in report
    assert "- Wireframes missing signoff coverage: none" in report
    assert "- Unresolved decision ids: dec-queue-vp-summary" in report
    assert "- Unresolved required signoff ids: sig-run-detail-eng-lead" in report
    assert "- Unresolved questions: oq-role-density, oq-alert-priority, oq-handoff-evidence" in report


def test_ui_review_pack_audit_flags_missing_checklist_coverage_and_evidence() -> None:
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
    assert audit.orphan_checklist_surfaces == []


def test_ui_review_pack_audit_flags_missing_decision_coverage() -> None:
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
    assert audit.orphan_decision_surfaces == []
    assert audit.unresolved_decision_ids == []


def test_ui_review_pack_audit_flags_missing_role_matrix_coverage() -> None:
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
    assert audit.orphan_role_assignment_surfaces == []
    assert audit.role_assignments_missing_responsibilities == []
    assert audit.role_assignments_missing_checklist_links == []
    assert audit.role_assignments_missing_decision_links == []


def test_ui_review_pack_audit_flags_missing_signoff_coverage_and_assignment_links() -> None:
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

    assert audit.ready is False
    assert audit.wireframes_missing_signoffs == ["wf-queue", "wf-run-detail", "wf-triage"]
    assert audit.orphan_signoff_surfaces == []
    assert audit.signoffs_missing_assignments == ["sig-overview-vp-eng"]
    assert audit.signoffs_missing_evidence == []
    assert audit.unresolved_required_signoff_ids == []


def test_render_ui_review_html_and_bundle_export(tmp_path) -> None:
    pack = build_big_4204_review_pack()
    audit = UIReviewPackAuditor().audit(pack)

    html = render_ui_review_pack_html(pack, audit)
    decision_log = render_ui_review_decision_log(pack)
    role_matrix = render_ui_review_role_matrix(pack)
    signoff_log = render_ui_review_signoff_log(pack)
    artifacts = write_ui_review_pack_bundle(str(tmp_path), pack)

    assert "<h2>Decision Log</h2>" in html
    assert "<h2>Role Matrix</h2>" in html
    assert "<h2>Sign-off Log</h2>" in html
    assert "dec-queue-vp-summary" in html
    assert "# UI Review Decision Log" in decision_log
    assert "dec-run-detail-audit-rail" in decision_log
    assert "# UI Review Role Matrix" in role_matrix
    assert "role-triage-platform-admin" in role_matrix
    assert "# UI Review Sign-off Log" in signoff_log
    assert "sig-run-detail-eng-lead" in signoff_log
    assert Path(artifacts.markdown_path).exists()
    assert Path(artifacts.html_path).exists()
    assert Path(artifacts.decision_log_path).exists()
    assert Path(artifacts.role_matrix_path).exists()
    assert Path(artifacts.signoff_log_path).exists()
    assert "Decision Log" in Path(artifacts.html_path).read_text()
    assert "Role Matrix" in Path(artifacts.html_path).read_text()
    assert "Sign-off Log" in Path(artifacts.html_path).read_text()
    assert "dec-triage-handoff-density" in Path(artifacts.decision_log_path).read_text()
    assert "role-run-detail-eng-lead" in Path(artifacts.role_matrix_path).read_text()
    assert "sig-queue-platform-admin" in Path(artifacts.signoff_log_path).read_text()
