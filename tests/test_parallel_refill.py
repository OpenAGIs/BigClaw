from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == []
    assert queue.issue_order() == []


def test_parallel_refill_queue_promotes_remaining_todo_slots() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-260", "state": {"name": "Done"}},
            {"identifier": "OPE-261", "state": {"name": "Done"}},
            {"identifier": "OPE-263", "state": {"name": "Done"}},
            {"identifier": "OPE-264", "state": {"name": "Done"}},
        ]
    )

    candidates = queue.select_candidates(set(), issue_states)

    assert candidates == []
