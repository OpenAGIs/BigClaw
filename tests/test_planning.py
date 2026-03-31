import subprocess
from pathlib import Path

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
    build_pilot_rollout_scorecard,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    evaluate_candidate_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
    render_pilot_rollout_gate_report,
)
from bigclaw.governance import ScopeFreezeAudit
from bigclaw.governance import (
    FreezeException,
    GovernanceBacklogItem,
    ScopeFreezeBoard,
    ScopeFreezeGovernance,
    render_scope_freeze_report,
)
from bigclaw.workspace_bootstrap import (
    bootstrap_workspace,
    cache_root_for_repo,
    cleanup_workspace,
    repo_cache_key,
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
                validation_command="python3 -m pytest tests/test_runtime_matrix.py -q",
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
                validation_command="python3 -m pytest tests/test_runtime_matrix.py -q",
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
        validation_command="python3 -m pytest tests/test_operations.py -q && (cd bigclaw-go && go test ./internal/product ./internal/api)",
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
                target="bigclaw-go/internal/product/saved_views.go",
                capability="saved-views",
                note="Go-owned saved views and digest evidence",
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
        "tests/test_operations.py",
        "src/bigclaw/execution_contract.py",
        "src/bigclaw/workflow.py",
        "bigclaw-go/internal/product/saved_views.go",
        "bigclaw-go/internal/product/saved_views_test.go",
        "bigclaw-go/internal/workflow/engine_test.go",
        "bigclaw-go/internal/worker/runtime_test.go",
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


def test_repo_cache_key_derives_from_repo_locator() -> None:
    assert repo_cache_key("git@github.com:OpenAGIs/BigClaw.git") == "github.com-openagis-bigclaw"
    assert repo_cache_key("https://github.com/OpenAGIs/BigClaw.git") == "github.com-openagis-bigclaw"
    assert repo_cache_key("git@github.com:OpenAGIs/BigClaw.git", cache_key="Team/BigClaw") == "team-bigclaw"


def test_cache_root_for_repo_uses_repo_specific_directory(tmp_path: Path) -> None:
    cache_root = cache_root_for_repo(
        "git@github.com:OpenAGIs/BigClaw.git",
        cache_base=tmp_path / "repos",
    )

    assert cache_root == tmp_path / "repos" / "github.com-openagis-bigclaw"


def test_bootstrap_workspace_creates_shared_worktree_from_local_seed(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-321"

    status = bootstrap_workspace(workspace, "OPE-321", str(remote), cache_base=cache_base)

    expected_cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)
    assert status.reused is False
    assert status.branch == "symphony/OPE-321"
    assert status.workspace_mode == "worktree_created"
    assert status.cache_reused is False
    assert status.clone_suppressed is False
    assert status.mirror_created is True
    assert status.seed_created is True
    assert Path(status.cache_root) == expected_cache_root
    assert (expected_cache_root / "mirror.git" / "HEAD").exists()
    assert (expected_cache_root / "seed" / ".git").exists()
    assert workspace.exists()
    assert (workspace / ".git").exists()
    assert git(workspace, "branch", "--show-current") == "symphony/OPE-321"
    assert (workspace / "README.md").read_text() == "hello\n"
    assert Path(git(workspace, "rev-parse", "--git-common-dir")).resolve() == (expected_cache_root / "seed" / ".git").resolve()
    assert git(expected_cache_root / "seed", "remote", "get-url", "origin") == str(remote)
    assert git(expected_cache_root / "seed", "remote", "get-url", "cache") == str((expected_cache_root / "mirror.git").resolve())


def test_second_workspace_reuses_warm_cache_without_full_clone(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"

    first = bootstrap_workspace(tmp_path / "workspaces" / "OPE-322", "OPE-322", str(remote), cache_base=cache_base)
    second = bootstrap_workspace(tmp_path / "workspaces" / "OPE-323", "OPE-323", str(remote), cache_base=cache_base)

    assert first.cache_root == second.cache_root
    assert second.cache_reused is True
    assert second.clone_suppressed is True
    assert second.mirror_created is False
    assert second.seed_created is False
    assert second.workspace_mode == "worktree_created"


def test_bootstrap_workspace_reuses_existing_issue_worktree(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-324"

    first = bootstrap_workspace(workspace, "OPE-324", str(remote), cache_base=cache_base)
    second = bootstrap_workspace(workspace, "OPE-324", str(remote), cache_base=cache_base)

    assert first.reused is False
    assert second.reused is True
    assert second.workspace_mode == "workspace_reused"
    assert second.cache_reused is True
    assert second.clone_suppressed is True
    assert second.branch == "symphony/OPE-324"


def test_cleanup_workspace_preserves_shared_cache_for_future_reuse(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)
    workspace = tmp_path / "workspaces" / "OPE-325"

    bootstrap_workspace(workspace, "OPE-325", str(remote), cache_base=cache_base)
    status = cleanup_workspace(workspace, "OPE-325", str(remote), cache_base=cache_base)
    follow_up = bootstrap_workspace(tmp_path / "workspaces" / "OPE-326", "OPE-326", str(remote), cache_base=cache_base)

    assert status.removed is True
    assert not workspace.exists()
    assert (cache_root / "mirror.git" / "HEAD").exists()
    assert (cache_root / "seed" / ".git").exists()
    assert follow_up.cache_reused is True
    assert follow_up.clone_suppressed is True
    assert follow_up.mirror_created is False
    assert follow_up.seed_created is False


def test_bootstrap_recovers_from_stale_seed_directory_without_remote_reclone(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    first = bootstrap_workspace(tmp_path / "workspaces" / "OPE-327", "OPE-327", str(remote), cache_base=cache_base)
    cache_root = Path(first.cache_root)

    cleanup_workspace(tmp_path / "workspaces" / "OPE-327", "OPE-327", str(remote), cache_base=cache_base)
    seed_path = cache_root / "seed"
    if seed_path.exists():
        subprocess.run(["rm", "-rf", str(seed_path)], check=True)
    seed_path.mkdir(parents=True, exist_ok=True)
    (seed_path / "stale.txt").write_text("stale\n")

    recovered = bootstrap_workspace(tmp_path / "workspaces" / "OPE-328", "OPE-328", str(remote), cache_base=cache_base)

    assert recovered.cache_reused is False
    assert recovered.clone_suppressed is True
    assert recovered.mirror_created is False
    assert recovered.seed_created is True
    assert (cache_root / "mirror.git" / "HEAD").exists()
    assert (cache_root / "seed" / ".git").exists()


def test_cleanup_workspace_prunes_worktree_and_bootstrap_branch(tmp_path: Path) -> None:
    remote = init_remote_with_main(tmp_path)
    cache_base = tmp_path / "repos"
    workspace = tmp_path / "workspaces" / "OPE-329"
    cache_root = cache_root_for_repo(str(remote), cache_base=cache_base)

    bootstrap_workspace(workspace, "OPE-329", str(remote), cache_base=cache_base)
    status = cleanup_workspace(workspace, "OPE-329", str(remote), cache_base=cache_base)

    assert status.removed is True
    assert not workspace.exists()
    assert "symphony/OPE-329" not in git(cache_root / "seed", "branch", "--format", "%(refname:short)").splitlines()
    assert str(workspace.resolve()) not in git(cache_root / "seed", "worktree", "list", "--porcelain")


def test_scope_freeze_board_round_trip_preserves_manifest_shape():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="pm-director",
                status="ready",
                scope_status="frozen",
                acceptance_criteria=["Epic scope frozen", "Backlog sequencing documented"],
                validation_plan=["governance-audit", "report-shared"],
            )
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-121",
                reason="Compliance requirement discovered after freeze",
                approved_by="cto",
                decision_note="Allow as approved exception.",
            )
        ],
    )

    restored = ScopeFreezeBoard.from_dict(board.to_dict())

    assert restored == board


def test_scope_freeze_audit_flags_backlog_governance_and_closeout_gaps():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="",
                status="planned",
                scope_status="proposed",
                acceptance_criteria=[],
                validation_plan=[],
                required_closeout=["validation-evidence"],
            ),
            GovernanceBacklogItem(
                issue_id="OPE-118",
                title="Duplicate issue record",
                phase="step-1",
                owner="pm-director",
                status="planned",
                scope_status="unknown",
                acceptance_criteria=["Keep backlog aligned"],
                validation_plan=["audit"],
                required_closeout=["validation-evidence", "git-push"],
            ),
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Mirrored tracking entry",
                phase="step-1",
                owner="pm-director",
                status="planned",
                scope_status="frozen",
                acceptance_criteria=["Track freeze decisions"],
                validation_plan=["audit"],
            ),
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-130",
                reason="Pending steering review",
            )
        ],
    )

    audit = ScopeFreezeGovernance().audit(board)

    assert audit.duplicate_issue_ids == ["OPE-116"]
    assert audit.missing_owners == ["OPE-116"]
    assert audit.missing_acceptance == ["OPE-116"]
    assert audit.missing_validation == ["OPE-116"]
    assert audit.missing_closeout_requirements == {
        "OPE-116": ["git-push", "git-log-stat"],
        "OPE-118": ["git-log-stat"],
    }
    assert audit.unauthorized_scope_changes == ["OPE-116"]
    assert audit.invalid_scope_statuses == ["OPE-118"]
    assert audit.unapproved_exceptions == ["OPE-130"]
    assert audit.release_ready is False
    assert audit.readiness_score == 0.0


def test_scope_freeze_audit_round_trip_and_ready_state():
    audit = ScopeFreezeAudit(
        board_name="BigClaw v4.0 Freeze",
        version="v4.0",
        total_items=1,
    )

    restored = ScopeFreezeAudit.from_dict(audit.to_dict())

    assert restored == audit
    assert restored.release_ready is True
    assert restored.readiness_score == 100.0


def test_render_scope_freeze_report_summarizes_board_and_run_closeout_requirements():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="pm-director",
                status="ready",
                scope_status="frozen",
                acceptance_criteria=["Epic scope frozen"],
                validation_plan=["governance-audit"],
                required_closeout=["validation-evidence", "git-push", "git-log-stat"],
            )
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-121",
                reason="Approved scope exception",
                approved_by="cto",
            )
        ],
    )

    audit = ScopeFreezeGovernance().audit(board)
    report = render_scope_freeze_report(board, audit)

    assert "# Scope Freeze Governance Report" in report
    assert "- Readiness Score: 100.0" in report
    assert "- Release Ready: True" in report
    assert "- OPE-116: phase=step-1 owner=pm-director status=ready scope=frozen closeout=validation-evidence, git-push, git-log-stat" in report
    assert "- OPE-121: approved_by=cto reason=Approved scope exception" in report
    assert "- Missing closeout requirements: none" in report
