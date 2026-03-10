from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.risk import RiskScorer
from bigclaw.scheduler import Scheduler


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
