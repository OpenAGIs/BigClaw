from bigclaw.repo_board import RepoDiscussionBoard


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
