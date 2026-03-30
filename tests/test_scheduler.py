from bigclaw.models import Task, RiskLevel
from bigclaw.scheduler import Scheduler


def test_scheduler_high_risk_requires_approval():
    s = Scheduler()
    t = Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)
    d = s.decide(t)
    assert d.medium == "vm"
    assert d.approved is False


def test_scheduler_browser_task_routes_browser():
    s = Scheduler()
    t = Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"])
    d = s.decide(t)
    assert d.medium == "browser"
    assert d.approved is True


def test_scheduler_over_budget_degrades_browser_task_to_docker():
    s = Scheduler()
    t = Task(
        task_id="z",
        source="github",
        title="budgeted ui test",
        description="",
        required_tools=["browser"],
        budget=15.0,
    )

    d = s.decide(t)

    assert d.medium == "docker"
    assert d.approved is True
    assert "budget degraded browser route to docker" in d.reason


def test_scheduler_over_budget_pauses_task():
    s = Scheduler()
    t = Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0)

    d = s.decide(t)

    assert d.medium == "none"
    assert d.approved is False
    assert d.reason == "paused: budget 5.0 below required docker budget 10.0"
