from bigclaw.events import EventBus, WebhookDelivery, WebhookEndpoint
from bigclaw.models import Task, TaskState
from bigclaw.scheduler import Scheduler
from bigclaw.observability import ObservabilityLedger


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
