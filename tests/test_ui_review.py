from bigclaw.ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewObjective,
    UIReviewPack,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_pack_report,
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
    assert "- Audit: READY: objectives=1 wireframes=1 interactions=1 open_questions=1" in report
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
    assert "obj-queue-governance" in report
    assert "wf-triage: Triage and handoff board" in report
    assert "flow-run-replay: Run replay with evidence audit" in report
    assert "- Unresolved questions: oq-role-density, oq-alert-priority, oq-handoff-evidence" in report
