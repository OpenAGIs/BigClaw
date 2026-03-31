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


def test_legacy_cost_control_mapping_and_shim_surfaces_behave() -> None:
    cost_control = importlib.import_module("bigclaw.cost_control")
    mapping = importlib.import_module("bigclaw.mapping")
    legacy_shim = importlib.import_module("bigclaw.legacy_shim")

    task = mapping.map_source_issue_to_task(
        bigclaw.SourceIssue(
            source="github",
            source_id="OpenAGIs/BigClaw#42",
            title="prod incident",
            description="desc",
            labels=["bug"],
            priority="P1",
            state="In Progress",
            links={"issue": "https://github.com/OpenAGIs/BigClaw/issues/42"},
        )
    )
    task.budget = 7
    controller = cost_control.CostController({"browser": 6.0, "docker": 2.0, "none": 0.0})
    decision = controller.evaluate(task, "browser", 60, spent_so_far=4)

    assert task.risk_level is bigclaw.RiskLevel.HIGH
    assert task.state is bigclaw.TaskState.IN_PROGRESS
    assert decision.status == "degrade"
    assert legacy_shim.append_missing_flag(["--foo"], "--bar", "baz") == ["--foo", "--bar", "baz"]
    assert legacy_shim.translate_workspace_validate_args(
        ["--report-file", "out.json", "--no-cleanup", "--issues", "BIG-1", "BIG-2"]
    ) == ["--report", "out.json", "--cleanup=false", "--issues", "BIG-1,BIG-2"]


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


def test_legacy_workspace_bootstrap_cli_submodule_stays_importable(capsys: pytest.CaptureFixture[str]) -> None:
    workspace_bootstrap_cli = importlib.import_module("bigclaw.workspace_bootstrap_cli")

    parser = workspace_bootstrap_cli.build_parser(
        description="desc",
        default_repo_url="git@example.com:repo.git",
        default_branch="main",
        default_cache_root=None,
        default_cache_base="~/.cache/symphony/repos",
        default_cache_key=None,
    )
    args = parser.parse_args(["cleanup", "--workspace", ".", "--json"])
    workspace_bootstrap_cli.emit({"status": "ok", "workspace": "x"}, as_json=False)

    captured = capsys.readouterr()

    assert args.command == "cleanup"
    assert "status=ok" in captured.out
