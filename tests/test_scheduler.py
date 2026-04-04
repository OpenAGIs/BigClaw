from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.scheduler import RiskScorer, Scheduler


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


def test_risk_scorer_keeps_simple_low_risk_work_low() -> None:
    score = RiskScorer().score_task(
        Task(task_id="BIG-902-low", source="linear", title="doc cleanup", description="")
    )

    assert score.total == 0
    assert score.level == RiskLevel.LOW
    assert score.requires_approval is False


def test_risk_scorer_elevates_prod_browser_work() -> None:
    score = RiskScorer().score_task(
        Task(
            task_id="BIG-902-mid",
            source="linear",
            title="release verification",
            description="prod browser change",
            labels=["prod"],
            priority=Priority.P0,
            required_tools=["browser"],
        )
    )

    assert score.total == 40
    assert score.level == RiskLevel.MEDIUM
    assert score.requires_approval is False


def test_scheduler_uses_risk_score_to_require_approval(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-902-high",
        source="linear",
        title="security deploy",
        description="prod deploy",
        labels=["security", "prod"],
        priority=Priority.P0,
        required_tools=["deploy"],
    )

    record = Scheduler().execute(task, run_id="run-risk", ledger=ledger)
    entry = ledger.load()[0]

    assert record.risk_score is not None
    assert record.risk_score.total == 70
    assert record.risk_score.level == RiskLevel.HIGH
    assert record.decision.medium == "vm"
    assert record.decision.approved is False
    assert any(trace["span"] == "risk.score" for trace in entry["traces"])
    assert any(audit["action"] == "risk.score" for audit in entry["audits"])
