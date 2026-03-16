from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 1
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order() == ["OPE-275"]


def test_parallel_refill_queue_selects_no_candidates_when_only_active_issue_exists() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map([{"identifier": "OPE-275", "state": {"name": "In Progress"}}])

    candidates = queue.select_candidates({"OPE-275"}, issue_states)

    assert candidates == []
