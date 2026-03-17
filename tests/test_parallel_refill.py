from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-1", "OPE-2", "OPE-3", "OPE-4"]
    assert queue.issue_order() == ["OPE-1", "OPE-2", "OPE-3", "OPE-4"]


def test_parallel_refill_queue_has_no_remaining_candidates_when_all_four_are_active() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-1", "state": {"name": "Done"}},
            {"identifier": "OPE-2", "state": {"name": "Done"}},
            {"identifier": "OPE-3", "state": {"name": "In Progress"}},
            {"identifier": "OPE-4", "state": {"name": "In Progress"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-3", "OPE-4"}, issue_states, target_in_progress=2)

    assert candidates == []
