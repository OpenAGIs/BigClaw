from bigclaw.connectors import GitHubConnector, JiraConnector, LinearConnector
from bigclaw.mapping import map_source_issue_to_task
from bigclaw.models import Priority, RiskLevel, TaskState


def test_connector_stubs_return_deterministic_source_issues() -> None:
    github_issue = GitHubConnector().fetch_issues("OpenAGIs/BigClaw", ["Todo"])[0]
    linear_issue = LinearConnector().fetch_issues("BIG", ["In Progress"])[0]
    jira_issue = JiraConnector().fetch_issues("OPS", ["Blocked"])[0]

    assert github_issue.source == "github"
    assert github_issue.source_id == "OpenAGIs/BigClaw#1"
    assert linear_issue.source_id == "BIG-101"
    assert jira_issue.links["issue"] == "https://jira.example.com/browse/OPS-23"


def test_map_source_issue_to_task_preserves_priority_state_and_risk_defaults() -> None:
    issue = GitHubConnector().fetch_issues("OpenAGIs/BigClaw", ["In Progress"])[0]
    task = map_source_issue_to_task(issue)

    assert task.task_id == "OpenAGIs/BigClaw#1"
    assert task.priority == Priority.P1
    assert task.state == TaskState.IN_PROGRESS
    assert task.risk_level == RiskLevel.LOW
    assert task.required_tools == ["github"]


def test_map_source_issue_to_task_marks_prod_titles_high_risk() -> None:
    issue = LinearConnector().fetch_issues("BIG", ["Done"])[0]
    issue.title = "Prod release verification"

    task = map_source_issue_to_task(issue)

    assert task.priority == Priority.P0
    assert task.state == TaskState.DONE
    assert task.risk_level == RiskLevel.HIGH
