from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.queue import PersistentTaskQueue
from bigclaw.scheduler import Scheduler


def test_queue_to_scheduler_execution_records_full_chain(tmp_path: Path):
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


def test_high_risk_execution_records_pending_approval(tmp_path: Path):
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
