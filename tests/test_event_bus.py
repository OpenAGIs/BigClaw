from pathlib import Path

from bigclaw.event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
)
from bigclaw.models import Task
from bigclaw.observability import ObservabilityLedger, TaskRun


def test_event_bus_pr_comment_approves_waiting_run_and_persists_ledger(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-pr", source="github", title="PR approval", description="")
    run = TaskRun.from_task(task, run_id="run-pr-1", medium="vm")
    run.finalize("needs-approval", "waiting for reviewer comment")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    seen_statuses: list[str] = []
    bus.subscribe(PULL_REQUEST_COMMENT_EVENT, lambda _event, current: seen_statuses.append(current.status))

    updated = bus.publish(
        BusEvent(
            event_type=PULL_REQUEST_COMMENT_EVENT,
            run_id=run.run_id,
            actor="reviewer",
            details={
                "decision": "approved",
                "body": "LGTM, merge when green.",
                "mentions": ["ops"],
            },
        )
    )

    assert updated.status == "approved"
    assert updated.summary == "LGTM, merge when green."
    assert seen_statuses == ["approved"]

    persisted = ledger.load()[0]
    assert persisted["status"] == "approved"
    assert any(audit["action"] == "collaboration.comment" for audit in persisted["audits"])
    assert any(
        audit["action"] == "event_bus.transition" and audit["details"]["previous_status"] == "needs-approval"
        for audit in persisted["audits"]
    )


def test_event_bus_ci_completed_marks_run_completed(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-ci", source="github", title="CI completion", description="")
    run = TaskRun.from_task(task, run_id="run-ci-1", medium="docker")
    run.finalize("approved", "waiting for CI")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    updated = bus.publish(
        BusEvent(
            event_type=CI_COMPLETED_EVENT,
            run_id=run.run_id,
            actor="github-actions",
            details={"workflow": "pytest", "conclusion": "success"},
        )
    )

    assert updated.status == "completed"
    assert updated.summary == "CI workflow pytest completed with success"

    persisted = ledger.load()[0]
    assert persisted["status"] == "completed"
    assert any(
        audit["action"] == "event_bus.event" and audit["details"]["event_type"] == CI_COMPLETED_EVENT
        for audit in persisted["audits"]
    )


def test_event_bus_task_failed_marks_run_failed(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(task_id="BIG-203-fail", source="scheduler", title="Task failure", description="")
    run = TaskRun.from_task(task, run_id="run-fail-1", medium="docker")
    ledger.append(run)

    bus = EventBus(ledger=ledger)
    updated = bus.publish(
        BusEvent(
            event_type=TASK_FAILED_EVENT,
            run_id=run.run_id,
            actor="worker",
            details={"error": "sandbox command exited 137"},
        )
    )

    assert updated.status == "failed"
    assert updated.summary == "sandbox command exited 137"

    persisted = ledger.load()[0]
    assert persisted["status"] == "failed"
    assert any(
        audit["action"] == "event_bus.transition" and audit["details"]["status"] == "failed"
        for audit in persisted["audits"]
    )
