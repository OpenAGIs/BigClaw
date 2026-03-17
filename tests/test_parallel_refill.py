from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_can_be_empty_when_no_next_slice_is_assigned() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 1
    assert identifiers == []
    assert queue.issue_order() == []


def test_parallel_refill_queue_selects_no_candidates_when_queue_is_empty() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map([{"identifier": "OPE-275", "state": {"name": "Done"}}])

    candidates = queue.select_candidates(set(), issue_states)

    assert candidates == []
