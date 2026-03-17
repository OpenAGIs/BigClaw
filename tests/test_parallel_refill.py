from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_tracks_the_current_parallel_batch() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert queue.issue_identifiers() == ["OPE-234", "OPE-231", "OPE-227", "OPE-230"]
    assert queue.issue_order() == ["OPE-234", "OPE-231", "OPE-227", "OPE-230"]


def test_parallel_refill_queue_promotes_remaining_todo_slots() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-234", "state": {"name": "In Progress"}},
            {"identifier": "OPE-231", "state": {"name": "In Progress"}},
            {"identifier": "OPE-227", "state": {"name": "Todo"}},
            {"identifier": "OPE-230", "state": {"name": "Todo"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-234", "OPE-231"}, issue_states)

    assert candidates == ["OPE-227", "OPE-230"]
