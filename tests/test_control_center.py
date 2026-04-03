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
            {"task_id": "BIG-802-2", "status": "failed", "medium": "browser", "failure_category": "tool"},
            {
                "task_id": "BIG-802-3",
                "status": "failed",
                "medium": "docker",
                "repo_sync_audit": {"sync": {"failure_category": "repo-sync"}},
                "retry_count": 2,
            },
            {"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
        ],
    )

    report = render_queue_control_center(center)

    assert center.queue_depth == 3
    assert center.queued_by_priority == {"P0": 1, "P1": 1, "P2": 1}
    assert center.queued_by_risk == {"low": 1, "medium": 1, "high": 1}
    assert center.execution_media == {"vm": 1, "browser": 1, "docker": 2}
    assert center.waiting_approval_runs == 1
    assert center.blocked_tasks == ["BIG-802-1"]
    assert center.queued_tasks == ["BIG-802-1", "BIG-802-2", "BIG-802-3"]
    assert center.bulk_retry_tasks == ["BIG-802-1", "BIG-802-2"]
    assert center.bulk_retry_blockers == {"BIG-802-3": "manual takeover required before retry"}
    assert center.failure_attribution == {
        "approval": ["BIG-802-1"],
        "tool": ["BIG-802-2"],
        "repo-sync": ["BIG-802-3"],
    }
    assert center.failure_attribution_counts == {
        "approval": 1,
        "tool": 1,
        "repo-sync": 1,
    }
    assert center.manual_takeover_tasks == ["BIG-802-3"]
    assert center.manual_takeover_reasons == {
        "BIG-802-3": "retry budget exhausted and requires human ownership"
    }
    assert [action.action_id for action in center.actions["BIG-802-1"]] == [
        "drill-down",
        "export",
        "add-note",
        "escalate",
        "retry",
        "pause",
        "reassign",
        "audit",
    ]
    assert [action.action_id for action in center.actions["BIG-802-3"]] == [
        "drill-down",
        "export",
        "add-note",
        "escalate",
        "retry",
        "pause",
        "reassign",
        "audit",
        "manual-takeover",
    ]
    assert center.actions["BIG-802-1"][3].enabled is True
    assert center.actions["BIG-802-1"][4].enabled is True
    assert center.actions["BIG-802-1"][5].enabled is False
    assert "# Queue Control Center" in report
    assert "- Waiting Approval Runs: 1" in report
    assert "- BIG-802-1" in report
    assert "- Eligible Tasks: BIG-802-1, BIG-802-2" in report
    assert "- Entry: Bulk Retry [bulk-retry] targets=BIG-802-1, BIG-802-2" in report
    assert "- Blocked: BIG-802-3 reason=manual takeover required before retry" in report
    assert "- approval: count=1 tasks=BIG-802-1" in report
    assert "- repo-sync: count=1 tasks=BIG-802-3" in report
    assert "- BIG-802-3: Manual Takeover [manual-takeover] reason=retry budget exhausted and requires human ownership" in report
    assert "BIG-802-1: Drill Down [drill-down]" in report
    assert "Escalate [escalate] state=enabled" in report
    assert "Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused" in report
    assert "Manual Takeover [manual-takeover] state=enabled target=BIG-802-3" in report


def test_queue_control_center_groups_unknown_failures_and_explicit_takeover_policy(tmp_path: Path) -> None:
    queue = PersistentTaskQueue(str(tmp_path / "queue.json"))
    queue.enqueue(Task(task_id="BIG-901-1", source="linear", title="first", description="", priority=Priority.P1))
    queue.enqueue(Task(task_id="BIG-901-2", source="linear", title="second", description="", priority=Priority.P1))
    queue.enqueue(Task(task_id="BIG-901-3", source="linear", title="third", description="", priority=Priority.P1))

    center = OperationsAnalytics().build_queue_control_center(
        queue,
        runs=[
            {"task_id": "BIG-901-1", "status": "failed"},
            {"task_id": "BIG-901-2", "status": "failed", "manual_takeover_required": True},
            {"task_id": "BIG-901-3", "status": "failed", "failure_category": "tool"},
        ],
    )

    report = render_queue_control_center(center)

    assert center.bulk_retry_tasks == ["BIG-901-1", "BIG-901-3"]
    assert center.bulk_retry_blockers == {"BIG-901-2": "manual takeover required before retry"}
    assert center.failure_attribution == {
        "tool": ["BIG-901-3"],
        "unknown": ["BIG-901-1", "BIG-901-2"],
    }
    assert center.failure_attribution_counts == {"unknown": 2, "tool": 1}
    assert center.manual_takeover_tasks == ["BIG-901-2"]
    assert center.manual_takeover_reasons == {
        "BIG-901-2": "run policy explicitly requires manual takeover"
    }
    assert "- unknown: count=2 tasks=BIG-901-1, BIG-901-2" in report
    assert "- BIG-901-2: Manual Takeover [manual-takeover] reason=run policy explicitly requires manual takeover" in report


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
