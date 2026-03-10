from bigclaw.connectors import GitHubConnector, JiraConnector, LinearConnector


def test_connectors_fetch_minimum_issue():
    connectors = [GitHubConnector(), LinearConnector(), JiraConnector()]
    for c in connectors:
        issues = c.fetch_issues("OpenAGIs/BigClaw", ["Todo"])
        assert len(issues) >= 1
        assert issues[0].source == c.name
        assert issues[0].title
