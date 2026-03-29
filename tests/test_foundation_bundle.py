from pathlib import Path

from bigclaw.connectors import GitHubConnector, JiraConnector, LinearConnector, SourceIssue
from bigclaw.mapping import map_priority, map_source_issue_to_task
from bigclaw.memory import TaskMemoryStore
from bigclaw.models import Priority, Task
from bigclaw.validation_policy import enforce_validation_report_policy


def test_connectors_fetch_minimum_issue():
    connectors = [GitHubConnector(), LinearConnector(), JiraConnector()]
    for connector in connectors:
        issues = connector.fetch_issues("OpenAGIs/BigClaw", ["Todo"])
        assert len(issues) >= 1
        assert issues[0].source == connector.name
        assert issues[0].title


def test_map_priority():
    assert map_priority("P0") == Priority.P0
    assert map_priority("P1") == Priority.P1
    assert map_priority("UNKNOWN") == Priority.P2


def test_map_source_issue_to_task():
    issue = SourceIssue(
        source="linear",
        source_id="BIG-102",
        title="Implement model",
        description="desc",
        labels=["p0"],
        priority="P0",
        state="Todo",
        links={},
    )
    task = map_source_issue_to_task(issue)
    assert task.task_id == "BIG-102"
    assert task.priority == Priority.P0
    assert task.source == "linear"


def test_big501_memory_store_reuses_history_and_injects_rules(tmp_path: Path):
    store = TaskMemoryStore(str(tmp_path / "memory" / "task-patterns.json"))

    previous = Task(
        task_id="BIG-501-prev",
        source="github",
        title="Previous queue rollout",
        description="",
        labels=["queue", "platform"],
        required_tools=["github", "browser"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest", "smoke-test"],
    )
    store.remember_success(previous, summary="queue migration done")

    current = Task(
        task_id="BIG-501-new",
        source="github",
        title="New queue hardening",
        description="",
        labels=["queue"],
        required_tools=["github"],
        acceptance_criteria=["unit-tests"],
        validation_plan=["pytest"],
    )

    suggestion = store.suggest_rules(current)

    assert "BIG-501-prev" in suggestion["matched_task_ids"]
    assert "report-shared" in suggestion["acceptance_criteria"]
    assert "smoke-test" in suggestion["validation_plan"]
    assert "unit-tests" in suggestion["acceptance_criteria"]


def test_big602_validation_policy_blocks_issue_close_without_required_reports():
    decision = enforce_validation_report_policy(["task-run", "replay"])

    assert decision.allowed_to_close is False
    assert decision.status == "blocked"
    assert "benchmark-suite" in decision.missing_reports


def test_big602_validation_policy_allows_issue_close_when_reports_complete():
    decision = enforce_validation_report_policy(["task-run", "replay", "benchmark-suite"])

    assert decision.allowed_to_close is True
    assert decision.status == "ready"
