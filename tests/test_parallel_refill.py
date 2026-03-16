from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 0
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order()[:3] == ["OPE-264", "OPE-265", "OPE-266"]


def test_parallel_refill_queue_selects_no_candidates_when_queue_is_drained() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-270", "state": {"name": "Done"}},
            {"identifier": "OPE-271", "state": {"name": "Done"}},
        ]
    )

    candidates = queue.select_candidates(set(), issue_states)

    assert candidates == []
