from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_a_drained_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == []
    assert queue.issue_order() == []


def test_parallel_refill_queue_returns_no_candidates_when_queue_is_empty() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-271", "state": {"name": "Done"}},
            {"identifier": "OPE-272", "state": {"name": "Done"}},
            {"identifier": "OPE-273", "state": {"name": "Done"}},
            {"identifier": "OPE-274", "state": {"name": "Done"}},
        ]
    )

    candidates = queue.select_candidates(set(), issue_states)

    assert candidates == []
