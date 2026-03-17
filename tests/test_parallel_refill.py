from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-260", "OPE-261", "OPE-262", "OPE-263"]
    assert queue.issue_order() == ["OPE-260", "OPE-261", "OPE-262", "OPE-263"]


def test_parallel_refill_queue_promotes_remaining_todo_slots() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-260", "state": {"name": "In Progress"}},
            {"identifier": "OPE-261", "state": {"name": "In Progress"}},
            {"identifier": "OPE-262", "state": {"name": "Todo"}},
            {"identifier": "OPE-263", "state": {"name": "Todo"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-260", "OPE-261"}, issue_states)

    assert candidates == ["OPE-262", "OPE-263"]
