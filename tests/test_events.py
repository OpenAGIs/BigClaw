from bigclaw.events import EventBus, WebhookDelivery, WebhookEndpoint
from bigclaw.models import Task, TaskState
from bigclaw.observability import ObservabilityLedger
from bigclaw.scheduler import Scheduler


class FakeWebhookTransport:
    def __init__(self):
        self.calls = []

    def post(self, endpoint, event):
        self.calls.append((endpoint, event))
        return WebhookDelivery(endpoint=endpoint.name, success=True, status_code=202)


def test_event_bus_broadcasts_state_change_to_subscribers_and_webhooks():
    transport = FakeWebhookTransport()
    bus = EventBus(transport=transport)
    received = []
    task = Task(task_id="BIG-203", source="linear", title="Broadcast state", description="")
    task.state = TaskState.IN_PROGRESS

    bus.subscribe("task.state.changed", received.append)
    bus.register_webhook(
        WebhookEndpoint(name="ops-webhook", url="https://example.com/hooks/task", event_types=["task.state.changed"])
    )

    result = bus.publish_task_state_changed(task, previous_state=TaskState.TODO.value, actor="scheduler")

    assert result.subscribers_notified == 1
    assert len(received) == 1
    assert received[0].payload["previous_state"] == TaskState.TODO.value
    assert received[0].payload["new_state"] == TaskState.IN_PROGRESS.value
    assert len(result.deliveries) == 1
    assert transport.calls[0][0].name == "ops-webhook"
    assert transport.calls[0][1].event_type == "task.state.changed"


def test_event_bus_processes_pr_comment_signal_into_state_transition():
    bus = EventBus(transport=FakeWebhookTransport())
    received = []
    task = Task(task_id="BIG-203", source="github", title="Review me", description="")

    bus.subscribe("task.state.changed", received.append)

    result = bus.process_task_signal(
        task,
        "pr.comment.created",
        {"body": "Looks good, /start this work"},
        actor="github-webhook",
    )

    assert result is not None
    assert task.state == TaskState.IN_PROGRESS
    assert received[0].payload["metadata"]["signal_type"] == "pr.comment.created"
    assert received[0].payload["new_state"] == TaskState.IN_PROGRESS.value


def test_event_bus_processes_ci_completed_signal_into_done_transition():
    bus = EventBus(transport=FakeWebhookTransport())
    received = []
    task = Task(task_id="BIG-203", source="github", title="Green build", description="")
    task.state = TaskState.IN_PROGRESS

    bus.subscribe("task.state.changed", received.append)

    result = bus.process_task_signal(
        task,
        "ci.completed",
        {"conclusion": "success", "workflow": "ci"},
        actor="github-actions",
    )

    assert result is not None
    assert task.state == TaskState.DONE
    assert received[0].payload["metadata"]["signal_payload"]["workflow"] == "ci"
    assert received[0].payload["new_state"] == TaskState.DONE.value


def test_event_bus_processes_task_failed_signal_into_failed_transition():
    bus = EventBus(transport=FakeWebhookTransport())
    received = []
    task = Task(task_id="BIG-203", source="linear", title="Runtime failure", description="")
    task.state = TaskState.IN_PROGRESS

    bus.subscribe("task.state.changed", received.append)

    result = bus.process_task_signal(
        task,
        "task.failed",
        {"failed": True, "reason": "worker crashed"},
        actor="worker-runtime",
    )

    assert result is not None
    assert task.state == TaskState.FAILED
    assert received[0].payload["metadata"]["signal_payload"]["reason"] == "worker crashed"
    assert received[0].payload["new_state"] == TaskState.FAILED.value


def test_scheduler_execute_emits_state_change_event(tmp_path):
    transport = FakeWebhookTransport()
    bus = EventBus(transport=transport)
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    received = []
    task = Task(task_id="BIG-203", source="linear", title="Execute", description="")

    bus.subscribe("task.state.changed", received.append)

    record = Scheduler().execute(task, run_id="run-events", ledger=ledger, event_bus=bus)

    assert record.decision.approved is True
    assert task.state == TaskState.IN_PROGRESS
    assert len(received) == 1
    assert received[0].payload["task_id"] == "BIG-203"
    assert received[0].payload["new_state"] == TaskState.IN_PROGRESS.value
