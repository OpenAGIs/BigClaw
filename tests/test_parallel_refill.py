import json

from bigclaw.parallel_refill import ParallelIssueQueue, issue_state_map


def test_parallel_issue_queue_selects_candidates_in_order(tmp_path) -> None:
    payload = {
        "project": {"slug_id": "bigclaw"},
        "policy": {
            "activate_state_id": "state-1",
            "target_in_progress": 3,
            "refill_states": ["Todo", "Backlog"],
        },
        "issue_order": ["BIG-1", "BIG-2", "BIG-3", "BIG-4"],
        "issues": [
            {"identifier": "BIG-1"},
            {"identifier": "BIG-2"},
            {"identifier": "BIG-3"},
            {"identifier": "BIG-4"},
        ],
    }
    queue_path = tmp_path / "queue.json"
    queue_path.write_text(json.dumps(payload))

    queue = ParallelIssueQueue(str(queue_path))
    selected = queue.select_candidates(
        active_identifiers={"BIG-2"},
        issue_states={"BIG-1": "Todo", "BIG-2": "In Progress", "BIG-3": "Backlog", "BIG-4": "Done"},
    )

    assert queue.project_slug() == "bigclaw"
    assert selected == ["BIG-1", "BIG-3"]


def test_issue_state_map_reads_nested_and_flat_state_names() -> None:
    states = issue_state_map(
        [
            {"identifier": "BIG-1", "state": {"name": "Todo"}},
            {"identifier": "BIG-2", "state_name": "Blocked"},
            {"identifier": "", "state": {"name": "Ignored"}},
        ]
    )

    assert states == {"BIG-1": "Todo", "BIG-2": "Blocked"}
