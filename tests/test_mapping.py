from bigclaw.mapping import GitHubConnector, JiraConnector, LinearConnector, SourceIssue, map_priority, map_source_issue_to_task
from bigclaw.models import Priority


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
