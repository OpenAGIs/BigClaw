import pytest

from bigclaw.planning import (
    FourWeekExecutionPlan,
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EvidenceLink,
    EntryGate,
    EntryGateDecision,
    WeeklyExecutionPlan,
    WeeklyGoal,
    build_big_4701_execution_plan,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
)
from bigclaw.governance import ScopeFreezeAudit


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
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_design_system.py",
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
                validation_command="python3 -m pytest tests/test_runtime.py -q",
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
                validation_command="python3 -m pytest tests/test_design_system.py -q",
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
                validation_command="python3 -m pytest tests/test_design_system.py -q",
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
                validation_command="python3 -m pytest tests/test_operations.py -q",
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
                validation_command="python3 -m pytest tests/test_orchestration.py -q",
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
                validation_command="python3 -m pytest tests/test_design_system.py -q",
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
                validation_command="python3 -m pytest tests/test_operations.py -q",
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
                validation_command="python3 -m pytest tests/test_orchestration.py -q",
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
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_design_system.py",
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
    assert "validation=python3 -m pytest tests/test_design_system.py -q" in report
    assert "- ui-acceptance -> tests/test_design_system.py capability=release-gate" in report
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
        validation_command="python3 -m pytest tests/test_operations.py tests/test_saved_views.py -q",
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
        "tests/test_control_center.py",
        "tests/test_operations.py",
        "src/bigclaw/repo_governance.py",
        "src/bigclaw/workflow.py",
        "tests/test_workflow.py",
        "tests/test_execution_flow.py",
        "src/bigclaw/saved_views.py",
        "tests/test_saved_views.py",
        "src/bigclaw/evaluation.py",
        "tests/test_evaluation.py",
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
