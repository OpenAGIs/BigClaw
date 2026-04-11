from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import BUDGET_OVERRIDE_EVENT, FLOW_HANDOFF_EVENT, MANUAL_TAKEOVER_EVENT, ObservabilityLedger, SCHEDULER_DECISION_EVENT
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
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}
    assert audits[SCHEDULER_DECISION_EVENT]["outcome"] == "pending"
    assert audits[MANUAL_TAKEOVER_EVENT]["details"]["target_team"] == "security"
    assert audits[FLOW_HANDOFF_EVENT]["details"]["source_stage"] == "scheduler"


def test_budget_override_execution_records_canonical_audit_payloads(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-502-budget",
        source="linear",
        title="Budget override",
        description="manual budget extension",
        labels=["customer", "data"],
        priority=Priority.P0,
        required_tools=["browser", "sql"],
        budget=120.0,
        budget_override_actor="finance-controller",
        budget_override_reason="approved additional analytics validation spend",
        budget_override_amount=30.0,
    )

    Scheduler().execute(task, run_id="run-budget", ledger=ledger)
    audits = {entry["action"]: entry for entry in ledger.load()[0]["audits"]}

    assert audits[SCHEDULER_DECISION_EVENT]["details"]["risk_score"] >= 0
    assert audits[BUDGET_OVERRIDE_EVENT]["details"] == {
        "task_id": "BIG-502-budget",
        "run_id": "run-budget",
        "requested_budget": 120.0,
        "approved_budget": 150.0,
        "override_actor": "finance-controller",
        "reason": "approved additional analytics validation spend",
    }
