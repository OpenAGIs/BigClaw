from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 6
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order()[:3] == ["OPE-264", "OPE-265", "OPE-266"]


def test_parallel_refill_queue_selects_next_backlog_issue() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-264", "state": {"name": "In Progress"}},
            {"identifier": "OPE-265", "state": {"name": "In Progress"}},
            {"identifier": "OPE-266", "state": {"name": "In Progress"}},
            {"identifier": "OPE-267", "state": {"name": "In Progress"}},
            {"identifier": "OPE-268", "state": {"name": "In Progress"}},
            {"identifier": "OPE-269", "state": {"name": "Todo"}},
            {"identifier": "OPE-270", "state": {"name": "Backlog"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-264", "OPE-265", "OPE-266", "OPE-267", "OPE-268"}, issue_states)

    assert candidates == ["OPE-269"]
