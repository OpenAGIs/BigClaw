from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-272", "OPE-273", "OPE-274", "OPE-271"]
    assert queue.issue_order() == ["OPE-272", "OPE-273", "OPE-274", "OPE-271"]


def test_parallel_refill_queue_promotes_remaining_todo_slots() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-272", "state": {"name": "In Progress"}},
            {"identifier": "OPE-273", "state": {"name": "In Progress"}},
            {"identifier": "OPE-274", "state": {"name": "Todo"}},
            {"identifier": "OPE-271", "state": {"name": "Todo"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-272", "OPE-273"}, issue_states)

    assert candidates == ["OPE-274", "OPE-271"]
