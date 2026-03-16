from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 4
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order() == ["OPE-271", "OPE-270", "OPE-269", "OPE-268"]


def test_parallel_refill_queue_selects_next_continuation_slices_in_order() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-271", "state": {"name": "In Progress"}},
            {"identifier": "OPE-270", "state": {"name": "In Progress"}},
            {"identifier": "OPE-269", "state": {"name": "Todo"}},
            {"identifier": "OPE-268", "state": {"name": "Backlog"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-271", "OPE-270"}, issue_states)

    assert candidates == ["OPE-269", "OPE-268"]
