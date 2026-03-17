from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-5", "OPE-6", "OPE-12", "OPE-21"]
    assert queue.issue_order() == ["OPE-5", "OPE-6", "OPE-12", "OPE-21"]


def test_parallel_refill_queue_has_no_remaining_candidates_when_all_four_are_active() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-5", "state": {"name": "In Progress"}},
            {"identifier": "OPE-6", "state": {"name": "In Progress"}},
            {"identifier": "OPE-12", "state": {"name": "In Progress"}},
            {"identifier": "OPE-21", "state": {"name": "In Progress"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-5", "OPE-6", "OPE-12", "OPE-21"}, issue_states, target_in_progress=4)

    assert candidates == []
