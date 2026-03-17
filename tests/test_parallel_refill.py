from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-266", "OPE-267", "OPE-269", "OPE-268"]
    assert queue.issue_order() == ["OPE-266", "OPE-267", "OPE-269", "OPE-268"]


def test_parallel_refill_queue_has_no_remaining_recycled_candidates() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-266", "state": {"name": "Done"}},
            {"identifier": "OPE-267", "state": {"name": "In Progress"}},
            {"identifier": "OPE-269", "state": {"name": "Done"}},
            {"identifier": "OPE-268", "state": {"name": "Done"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-267"}, issue_states)

    assert candidates == []
