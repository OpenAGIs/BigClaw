from bigclaw.observability import (
    CollaborationComment,
    DecisionNote,
    build_collaboration_thread,
    merge_collaboration_threads,
)


def test_merge_collaboration_threads_combines_native_and_repo_surfaces():
    native = build_collaboration_thread(
        "run",
        "run-165",
        comments=[CollaborationComment(comment_id="c1", author="ops", body="native note", created_at="2026-03-12T10:00:00Z")],
        decisions=[DecisionNote(decision_id="d1", author="lead", outcome="approved", summary="native decision", recorded_at="2026-03-12T10:05:00Z")],
    )

    repo_thread = build_collaboration_thread(
        "repo-board",
        "run-165",
        comments=[
            CollaborationComment(
                comment_id="repo-post-1",
                author="repo-agent",
                body="repo board context",
                created_at="2026-03-12T10:03:00Z",
                anchor="run:run-165",
            )
        ],
    )

    merged = merge_collaboration_threads(target_id="run-165", native_thread=native, repo_thread=repo_thread)

    assert merged is not None
    assert merged.surface == "merged"
    assert len(merged.comments) == 2
    assert len(merged.decisions) == 1
    assert merged.comments[1].body == "repo board context"
