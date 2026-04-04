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


def test_queue_to_scheduler_execution_records_full_chain(tmp_path: Path) -> None:
    from bigclaw.queue import PersistentTaskQueue

    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    report_path = tmp_path / "reports" / "run-1.md"

    queue.enqueue(
        Task(
            task_id="BIG-502",
            source="linear",
            title="Record execution",
            description="full chain",
            priority=Priority.P0,
            risk_level=RiskLevel.MEDIUM,
            required_tools=["browser"],
        )
    )

    task = queue.dequeue_task()
    assert task is not None

    record = Scheduler().execute(task, run_id="run-1", ledger=ledger, report_path=str(report_path))
    entries = ledger.load()

    assert record.decision.medium == "browser"
    assert record.decision.approved is True
    assert record.run.status == "approved"
    assert report_path.exists()
    assert report_path.with_suffix(".html").exists()
    assert "Status: approved" in report_path.read_text()
    assert len(entries) == 1
    assert entries[0]["traces"][0]["span"] == "scheduler.decide"
    assert entries[0]["artifacts"][0]["kind"] == "page"
    assert entries[0]["artifacts"][1]["kind"] == "report"
    assert entries[0]["audits"][0]["details"]["reason"] == "browser automation task"


def test_high_risk_execution_records_pending_approval(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-502-risk",
        source="jira",
        title="Prod change",
        description="manual review",
        risk_level=RiskLevel.HIGH,
    )

    record = Scheduler().execute(task, run_id="run-2", ledger=ledger)

    assert record.decision.approved is False
    assert record.run.status == "needs-approval"
    assert ledger.load()[0]["audits"][0]["outcome"] == "pending"
