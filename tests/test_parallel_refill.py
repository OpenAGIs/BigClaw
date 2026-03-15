from pathlib import Path

from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_refill_queue_records_unique_identifiers() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")

    identifiers = queue.issue_identifiers()

    assert queue.project_slug() == "8a198fec793e"
    assert queue.target_in_progress() == 2
    assert len(identifiers) == len(set(identifiers))
    assert queue.issue_order()[:4] == ["OPE-210", "OPE-211", "OPE-212", "OPE-213"]


def test_parallel_refill_queue_selects_next_backlog_issue() -> None:
    queue = ParallelIssueQueue("docs/parallel-refill-queue.json")
    issue_states = issue_state_map(
        [
            {"identifier": "OPE-210", "state": {"name": "Done"}},
            {"identifier": "OPE-211", "state": {"name": "In Progress"}},
            {"identifier": "OPE-212", "state": {"name": "Backlog"}},
            {"identifier": "OPE-213", "state": {"name": "Backlog"}},
        ]
    )

    candidates = queue.select_candidates({"OPE-211"}, issue_states)

    assert candidates == ["OPE-212"]
