from bigclaw.validation_policy import enforce_validation_report_policy


def test_big602_validation_policy_blocks_issue_close_without_required_reports():
    decision = enforce_validation_report_policy(["task-run", "replay"])

    assert decision.allowed_to_close is False
    assert decision.status == "blocked"
    assert "benchmark-suite" in decision.missing_reports


def test_big602_validation_policy_allows_issue_close_when_reports_complete():
    decision = enforce_validation_report_policy(["task-run", "replay", "benchmark-suite"])

    assert decision.allowed_to_close is True
    assert decision.status == "ready"
