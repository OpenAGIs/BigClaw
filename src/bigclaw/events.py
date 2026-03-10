import json
import uuid
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Protocol
from urllib import request
from urllib.error import HTTPError, URLError

from .models import Task
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

    def subscribe(self, event_type: str, handler: EventHandler) -> None:
        self._subscribers.setdefault(event_type, []).append(handler)

    def register_webhook(self, endpoint: WebhookEndpoint) -> None:
        self._webhooks.append(endpoint)

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
