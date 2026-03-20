from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "53e33900c67e"
    assert queue.target_in_progress() == 2
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order()[:4] == [
        "BIG-GOM-301",
        "BIG-GOM-302",
        "BIG-GOM-303",
        "BIG-GOM-304",
    ]


def test_parallel_refill_queue_selects_first_runnable_draft_slices() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "BIG-GOM-301", "state": {"name": "Todo"}},
            {"identifier": "BIG-GOM-302", "state": {"name": "Todo"}},
            {"identifier": "BIG-GOM-303", "state": {"name": "Todo"}},
            {"identifier": "BIG-GOM-304", "state": {"name": "Todo"}},
            {"identifier": "BIG-GOM-305", "state": {"name": "Backlog"}},
            {"identifier": "BIG-GOM-306", "state": {"name": "Backlog"}},
        ]
    )

    candidates = queue.select_candidates(set(), issue_states)

    assert candidates == ["BIG-GOM-301", "BIG-GOM-302"]
