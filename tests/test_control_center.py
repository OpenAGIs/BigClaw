from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.operations import OperationsAnalytics, render_queue_control_center
from bigclaw.queue import PersistentTaskQueue
from bigclaw.reports import SharedViewContext, SharedViewFilter



def test_queue_peek_tasks_returns_priority_order(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    queue.enqueue(Task(task_id="p2", source="linear", title="low", description="", priority=Priority.P2))
    queue.enqueue(Task(task_id="p0", source="linear", title="top", description="", priority=Priority.P0))
    queue.enqueue(Task(task_id="p1", source="linear", title="mid", description="", priority=Priority.P1))

    assert [task.task_id for task in queue.peek_tasks()] == ["p0", "p1", "p2"]



def test_queue_control_center_summarizes_queue_and_execution_media(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    queue.enqueue(
        Task(task_id="BIG-802-1", source="linear", title="top", description="", priority=Priority.P0, risk_level=RiskLevel.HIGH)
    )
    queue.enqueue(
        Task(task_id="BIG-802-2", source="linear", title="mid", description="", priority=Priority.P1, risk_level=RiskLevel.MEDIUM)
    )
    queue.enqueue(
        Task(task_id="BIG-802-3", source="linear", title="low", description="", priority=Priority.P2, risk_level=RiskLevel.LOW)
    )

    center = OperationsAnalytics().build_queue_control_center(
        queue,
        runs=[
            {"task_id": "BIG-802-1", "status": "needs-approval", "medium": "vm"},
            {"task_id": "BIG-802-2", "status": "approved", "medium": "browser"},
            {"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
        ],
    )

    report = render_queue_control_center(center)

    assert center.queue_depth == 3
    assert center.queued_by_priority == {"P0": 1, "P1": 1, "P2": 1}
    assert center.queued_by_risk == {"low": 1, "medium": 1, "high": 1}
    assert center.execution_media == {"vm": 1, "browser": 1, "docker": 1}
    assert center.waiting_approval_runs == 1
    assert center.blocked_tasks == ["BIG-802-1"]
    assert center.queued_tasks == ["BIG-802-1", "BIG-802-2", "BIG-802-3"]
    assert "# Queue Control Center" in report
    assert "- Waiting Approval Runs: 1" in report
    assert "- BIG-802-1" in report


def test_queue_control_center_renders_shared_view_empty_state(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    center = OperationsAnalytics().build_queue_control_center(queue, runs=[])

    report = render_queue_control_center(
        center,
        view=SharedViewContext(
            filters=[SharedViewFilter(label="Team", value="operations")],
            result_count=0,
            empty_message="No queued work for the selected team.",
        ),
    )

    assert "## View State" in report
    assert "- State: empty" in report
    assert "- Summary: No queued work for the selected team." in report
    assert "- Team: operations" in report
