import importlib
import json
from pathlib import Path

import bigclaw
import pytest


def test_legacy_connectors_submodule_stays_importable() -> None:
    connectors = importlib.import_module("bigclaw.connectors")

    issue = connectors.GitHubConnector().fetch_issues("OpenAGIs/BigClaw", ["Todo"])[0]

    assert issue.source == "github"
    assert issue.state == "Todo"
    assert bigclaw.SourceIssue is connectors.SourceIssue


def test_legacy_deprecation_submodule_warns() -> None:
    deprecation = importlib.import_module("bigclaw.deprecation")

    message = deprecation.legacy_runtime_message("python -m bigclaw", "bigclawctl")

    assert "migration-only" in message
    with pytest.warns(DeprecationWarning):
        assert deprecation.warn_legacy_runtime_surface("python -m bigclaw", "bigclawctl") == message


def test_legacy_issue_archive_and_pilot_surfaces_behave() -> None:
    issue_archive = importlib.import_module("bigclaw.issue_archive")
    pilot = importlib.import_module("bigclaw.pilot")

    archive = issue_archive.IssuePriorityArchive(
        issue_id="BIG-1",
        title="Archive",
        version="v1",
        findings=[
            issue_archive.ArchivedIssue(
                finding_id="F-1",
                summary="Missing owner",
                category="ui",
                priority="P0",
                owner="",
            )
        ],
    )
    audit = issue_archive.IssuePriorityArchivist().audit(archive)
    result = pilot.PilotImplementationResult(
        customer="ACME",
        environment="prod",
        kpis=[pilot.PilotKPI(name="success", target=95, actual=99)],
        production_runs=3,
        incidents=0,
    )

    assert audit.ready is False
    assert "F-1" in issue_archive.render_issue_priority_archive_report(archive, audit)
    assert result.ready is True
    assert "KPI Pass Rate: 100.0%" in pilot.render_pilot_implementation_report(result)


def test_legacy_roadmap_and_parallel_refill_surfaces_behave(tmp_path: Path) -> None:
    roadmap = importlib.import_module("bigclaw.roadmap")
    parallel_refill = importlib.import_module("bigclaw.parallel_refill")

    built = roadmap.build_execution_pack_roadmap()
    queue_path = tmp_path / "queue.json"
    queue_path.write_text(
        json.dumps(
            {
                "project": {"slug_id": "bigclaw"},
                "policy": {
                    "activate_state_id": "state-1",
                    "target_in_progress": 2,
                    "refill_states": ["Todo", "Backlog"],
                },
                "issue_order": ["BIG-1", "BIG-2", "BIG-3"],
                "issues": [
                    {"identifier": "BIG-1", "state": {"name": "Todo"}},
                    {"identifier": "BIG-2", "state_name": "Backlog"},
                    {"identifier": "BIG-3", "state": {"name": "Done"}},
                ],
            }
        )
    )
    queue = parallel_refill.ParallelIssueQueue(str(queue_path))

    assert built.epic_map()["BIG-EPIC-8"].owner == "engineering-platform"
    assert queue.select_candidates(set(), {"BIG-1": "Todo", "BIG-2": "Backlog"}) == ["BIG-1", "BIG-2"]
    assert parallel_refill.issue_state_map(queue.issue_records()) == {
        "BIG-1": "Todo",
        "BIG-2": "Backlog",
        "BIG-3": "Done",
    }
