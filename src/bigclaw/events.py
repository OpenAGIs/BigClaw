import json
import uuid
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Protocol
from urllib import request
from urllib.error import HTTPError, URLError

from .models import Task, TaskState
from .observability import utc_now


@dataclass
class Event:
    event_id: str
    event_type: str
    occurred_at: str
    payload: Dict[str, Any]

    @classmethod
    def create(cls, event_type: str, payload: Dict[str, Any]) -> "Event":
        return cls(
            event_id=str(uuid.uuid4()),
            event_type=event_type,
            occurred_at=utc_now(),
            payload=payload,
        )

    def to_dict(self) -> Dict[str, Any]:
        return {
            "event_id": self.event_id,
            "event_type": self.event_type,
            "occurred_at": self.occurred_at,
            "payload": self.payload,
        }


EventHandler = Callable[[Event], None]
SignalHandler = Callable[[Task, Dict[str, Any]], Optional[TaskState]]


@dataclass
class WebhookEndpoint:
    name: str
    url: str
    event_types: List[str] = field(default_factory=list)
    headers: Dict[str, str] = field(default_factory=dict)

    def accepts(self, event_type: str) -> bool:
        return not self.event_types or event_type in self.event_types


@dataclass
class WebhookDelivery:
    endpoint: str
    success: bool
    status_code: int
    timestamp: str = field(default_factory=utc_now)
    error: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "endpoint": self.endpoint,
            "success": self.success,
            "status_code": self.status_code,
            "timestamp": self.timestamp,
            "error": self.error,
        }


@dataclass
class DispatchResult:
    event: Event
    subscribers_notified: int
    deliveries: List[WebhookDelivery]


class WebhookTransport(Protocol):
    def post(self, endpoint: WebhookEndpoint, event: Event) -> WebhookDelivery:
        ...


class UrllibWebhookTransport:
    def post(self, endpoint: WebhookEndpoint, event: Event) -> WebhookDelivery:
        body = json.dumps(event.to_dict()).encode("utf-8")
        headers = {"Content-Type": "application/json", **endpoint.headers}
        webhook_request = request.Request(endpoint.url, data=body, headers=headers, method="POST")

        try:
            with request.urlopen(webhook_request) as response:
                return WebhookDelivery(
                    endpoint=endpoint.name,
                    success=200 <= response.status < 300,
                    status_code=response.status,
                )
        except HTTPError as exc:
            return WebhookDelivery(
                endpoint=endpoint.name,
                success=False,
                status_code=exc.code,
                error=str(exc),
            )
        except URLError as exc:
            return WebhookDelivery(
                endpoint=endpoint.name,
                success=False,
                status_code=0,
                error=str(exc.reason),
            )


class EventBus:
    def __init__(self, transport: Optional[WebhookTransport] = None):
        self.transport = transport or UrllibWebhookTransport()
        self._subscribers: Dict[str, List[EventHandler]] = {}
        self._webhooks: List[WebhookEndpoint] = []
        self._signal_handlers: Dict[str, SignalHandler] = {
            "pr.comment.created": self._handle_pr_comment_created,
            "ci.completed": self._handle_ci_completed,
            "task.failed": self._handle_task_failed,
        }

    def subscribe(self, event_type: str, handler: EventHandler) -> None:
        self._subscribers.setdefault(event_type, []).append(handler)

    def register_webhook(self, endpoint: WebhookEndpoint) -> None:
        self._webhooks.append(endpoint)

    def register_signal_handler(self, signal_type: str, handler: SignalHandler) -> None:
        self._signal_handlers[signal_type] = handler

    def publish(self, event: Event) -> DispatchResult:
        handlers = self._subscribers.get(event.event_type, [])
        for handler in handlers:
            handler(event)

        deliveries = [
            self.transport.post(endpoint, event)
            for endpoint in self._webhooks
            if endpoint.accepts(event.event_type)
        ]
        return DispatchResult(
            event=event,
            subscribers_notified=len(handlers),
            deliveries=deliveries,
        )

    def process_task_signal(
        self,
        task: Task,
        signal_type: str,
        payload: Dict[str, Any],
        actor: str = "system",
    ) -> Optional[DispatchResult]:
        handler = self._signal_handlers.get(signal_type)
        if handler is None:
            return None

        next_state = handler(task, payload)
        if next_state is None or next_state == task.state:
            return None

        previous_state = task.state.value
        task.state = next_state
        metadata = {
            "signal_type": signal_type,
            "signal_payload": payload,
        }
        return self.publish_task_state_changed(task, previous_state=previous_state, actor=actor, **metadata)

    def publish_task_state_changed(
        self,
        task: Task,
        previous_state: str,
        actor: str = "system",
        **metadata: Any,
    ) -> DispatchResult:
        event = Event.create(
            "task.state.changed",
            {
                "task_id": task.task_id,
                "source": task.source,
                "title": task.title,
                "previous_state": previous_state,
                "new_state": task.state.value,
                "actor": actor,
                "metadata": metadata,
            },
        )
        return self.publish(event)

    def _handle_pr_comment_created(self, task: Task, payload: Dict[str, Any]) -> Optional[TaskState]:
        transition = payload.get("transition_to") or payload.get("state")
        if transition:
            return self._coerce_task_state(transition)

        comment_body = str(payload.get("body", "")).lower()
        command_map = {
            "/start": TaskState.IN_PROGRESS,
            "/block": TaskState.BLOCKED,
            "/done": TaskState.DONE,
            "/fail": TaskState.FAILED,
            "/todo": TaskState.TODO,
        }
        for command, state in command_map.items():
            if command in comment_body:
                return state
        return None

    def _handle_ci_completed(self, task: Task, payload: Dict[str, Any]) -> Optional[TaskState]:
        conclusion = str(payload.get("conclusion") or payload.get("status") or "").lower()
        if conclusion in {"success", "passed", "succeeded"}:
            return TaskState.DONE
        if conclusion in {"failure", "failed", "cancelled", "timed_out"}:
            return TaskState.FAILED
        return None

    def _handle_task_failed(self, task: Task, payload: Dict[str, Any]) -> Optional[TaskState]:
        if payload.get("failed") is False:
            return None
        return TaskState.FAILED

    def _coerce_task_state(self, value: str) -> TaskState:
        normalized = value.strip().lower().replace("_", " ")
        for state in TaskState:
            if state.value.lower() == normalized:
                return state
        raise ValueError(f"unsupported task state: {value}")
