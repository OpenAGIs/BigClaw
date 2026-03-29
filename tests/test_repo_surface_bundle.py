from bigclaw.collaboration import (
    CollaborationComment,
    DecisionNote,
    build_collaboration_thread,
    merge_collaboration_threads,
)
from bigclaw.models import Task
from bigclaw.observability import TaskRun
from bigclaw.planning import (
    EntryGateDecision,
    build_pilot_rollout_scorecard,
    evaluate_candidate_gate,
    render_pilot_rollout_gate_report,
)
from bigclaw.repo_board import RepoDiscussionBoard
from bigclaw.repo_commits import CommitDiff, CommitLineage, RepoCommit
from bigclaw.repo_gateway import (
    normalize_commit,
    normalize_diff,
    normalize_gateway_error,
    normalize_lineage,
    repo_audit_payload,
)
from bigclaw.repo_governance import RepoPermissionContract, missing_repo_audit_fields
from bigclaw.repo_links import bind_run_commits
from bigclaw.repo_plane import RepoSpace, RunCommitLink
from bigclaw.repo_registry import RepoRegistry
from bigclaw.repo_triage import (
    LineageEvidence,
    approval_evidence_packet,
    recommend_triage_action,
)
from bigclaw.reports import render_repo_narrative_exports, render_weekly_repo_evidence_section


def test_repo_board_create_reply_and_target_filtering():
    board = RepoDiscussionBoard()
    post = board.create_post(
        channel="bigclaw-ope-164",
        author="agent-a",
        body="Need reviewer on commit lineage",
        target_surface="run",
        target_id="run-164",
        metadata={"severity": "p1"},
    )
    reply = board.reply(parent_post_id=post.post_id, author="reviewer", body="I will review this now")

    assert post.post_id == "post-1"
    assert reply.parent_post_id == "post-1"

    run_posts = board.list_posts(target_surface="run", target_id="run-164")
    assert len(run_posts) == 2
    assert run_posts[0].channel == "bigclaw-ope-164"

    comment = run_posts[0].to_collaboration_comment()
    assert comment.anchor == "run:run-164"
    assert comment.body.startswith("Need reviewer")


def test_merge_collaboration_threads_combines_native_and_repo_surfaces():
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


def test_repo_gateway_normalization_and_audit_payload():
    commit = normalize_commit({"commit_hash": "abc123", "title": "feat: add repo plane", "author": "bot"})
    assert isinstance(commit, RepoCommit)
    assert commit.commit_hash == "abc123"

    lineage = normalize_lineage(
        {
            "root_hash": "abc123",
            "lineage": [commit.to_dict()],
            "children": {"abc123": ["def456"]},
            "leaves": ["def456"],
        }
    )
    assert isinstance(lineage, CommitLineage)
    assert lineage.leaves == ["def456"]

    diff = normalize_diff(
        {
            "left_hash": "abc123",
            "right_hash": "def456",
            "files_changed": 3,
            "insertions": 20,
            "deletions": 4,
            "summary": "3 files changed",
        }
    )
    assert isinstance(diff, CommitDiff)
    assert diff.files_changed == 3

    payload = repo_audit_payload(
        actor="native cloud",
        action="repo.diff",
        outcome="success",
        commit_hash="def456",
        repo_space_id="space-1",
    )
    assert payload["actor"] == "native cloud"
    assert payload["commit_hash"] == "def456"


def test_repo_gateway_error_normalization_is_deterministic():
    timeout = normalize_gateway_error(RuntimeError("gateway timeout while fetching lineage"))
    assert timeout.code == "timeout"
    assert timeout.retryable is True

    missing = normalize_gateway_error(RuntimeError("commit not found"))
    assert missing.code == "not_found"
    assert missing.retryable is False


def test_repo_permission_matrix_resolves_roles():
    contract = RepoPermissionContract()

    assert contract.check(action_permission="repo.push", actor_roles=["eng-lead"]) is True
    assert contract.check(action_permission="repo.accept", actor_roles=["reviewer"]) is True
    assert contract.check(action_permission="repo.push", actor_roles=["execution-agent"]) is False


def test_repo_audit_field_contract_is_deterministic():
    missing = missing_repo_audit_fields(
        "repo.accept",
        {
            "task_id": "OPE-172",
            "run_id": "run-172",
            "repo_space_id": "space-1",
            "actor": "reviewer",
        },
    )
    assert missing == ["accepted_commit_hash", "reviewer"]


def test_run_closeout_supports_commit_roles_and_accepted_hash():
    task = Task(task_id="OPE-143", source="linear", title="run links", description="")
    run = TaskRun.from_task(task, run_id="run-143", medium="docker")

    links = [
        RunCommitLink(run_id=run.run_id, commit_hash="aaa111", role="source", repo_space_id="space-1"),
        RunCommitLink(run_id=run.run_id, commit_hash="bbb222", role="candidate", repo_space_id="space-1"),
        RunCommitLink(run_id=run.run_id, commit_hash="ccc333", role="accepted", repo_space_id="space-1"),
    ]

    binding = bind_run_commits(links)
    assert binding.accepted_commit_hash == "ccc333"

    run.record_closeout(
        validation_evidence=["pytest tests/test_repo_links.py"],
        git_push_succeeded=True,
        git_log_stat_output="commit ccc333",
        run_commit_links=links,
    )

    assert run.closeout.accepted_commit_hash == "ccc333"
    restored = TaskRun.from_dict(run.to_dict())
    assert restored.closeout.accepted_commit_hash == "ccc333"
    assert restored.closeout.run_commit_links[1].role == "candidate"


def test_repo_registry_resolves_space_channel_and_agent_deterministically():
    registry = RepoRegistry()
    registry.register_space(
        RepoSpace(
            space_id="space-1",
            project_key="BIGCLAW",
            repo="OpenAGIs/BigClaw",
            sidecar_url="http://127.0.0.1:4041",
            health_state="healthy",
        )
    )

    task = Task(task_id="OPE-141", source="linear", title="repo registry", description="")

    resolved = registry.resolve_space("BIGCLAW")
    assert resolved is not None
    assert resolved.repo == "OpenAGIs/BigClaw"

    channel = registry.resolve_default_channel("BIGCLAW", task)
    assert channel == "bigclaw-ope-141"

    agent = registry.resolve_agent("native cloud", role="reviewer")
    assert agent.repo_agent_id == "agent-native-cloud"


def test_repo_registry_round_trip():
    registry = RepoRegistry()
    registry.register_space(RepoSpace(space_id="s-1", project_key="BIGCLAW", repo="OpenAGIs/BigClaw"))
    registry.resolve_agent("native cloud")

    serialized = registry.to_dict()
    restored = RepoRegistry.from_dict(serialized)

    assert restored.resolve_space("BIGCLAW") is not None
    assert restored.resolve_agent("native cloud").repo_agent_id == "agent-native-cloud"


def test_pilot_rollout_scorecard_and_candidate_gate():
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


def test_repo_weekly_narrative_exports_remain_consistent():
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


def test_lineage_aware_recommendations():
    recommendation = recommend_triage_action(
        status="needs-approval",
        evidence=LineageEvidence(candidate_commit="abc", accepted_ancestor="0001", similar_failure_count=0),
    )
    assert recommendation.action == "approve"

    recommendation = recommend_triage_action(
        status="failed",
        evidence=LineageEvidence(candidate_commit="abc", similar_failure_count=3),
    )
    assert recommendation.action == "replay"


def test_approval_evidence_packet_includes_candidate_and_accepted_hash():
    packet = approval_evidence_packet(
        run_id="run-170",
        links=[
            {"role": "candidate", "commit_hash": "abc111"},
            {"role": "accepted", "commit_hash": "def222"},
        ],
        lineage_summary="candidate descends from accepted baseline",
    )

    assert packet["accepted_commit_hash"] == "def222"
    assert packet["candidate_commit_hash"] == "abc111"
    assert "accepted baseline" in packet["lineage_summary"]
