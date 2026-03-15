from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 2
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order()[:3] == ["OPE-233", "OPE-234", "OPE-235"]


def test_parallel_refill_queue_selects_next_backlog_issue() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-233", "state": {"name": "In Progress"}},
            {"identifier": "OPE-234", "state": {"name": "Todo"}},
            {"identifier": "OPE-235", "state": {"name": "Backlog"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-233"}, issue_states)

    assert candidates == ["OPE-234"]
