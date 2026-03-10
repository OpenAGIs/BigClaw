from dataclasses import dataclass, field
from enum import Enum
import json
from pathlib import Path
from typing import Any, Dict, List, Optional

from .models import Task
from .observability import TaskRun, utc_now


class MemoryScope(str, Enum):
    RUN = "run"
    PROJECT = "project"
    ORG = "org"
    EXPERIENCE = "experience"


@dataclass
class MemoryRecord:
    key: str
    value: str
    scope: MemoryScope
    source: str
    timestamp: str = field(default_factory=utc_now)
    run_id: str = ""
    task_id: str = ""
    tags: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "key": self.key,
            "value": self.value,
            "scope": self.scope.value,
            "source": self.source,
            "timestamp": self.timestamp,
            "run_id": self.run_id,
            "task_id": self.task_id,
            "tags": self.tags,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "MemoryRecord":
        return cls(
            key=data["key"],
            value=data["value"],
            scope=MemoryScope(data["scope"]),
            source=data["source"],
            timestamp=data.get("timestamp", utc_now()),
            run_id=data.get("run_id", ""),
            task_id=data.get("task_id", ""),
            tags=data.get("tags", []),
            metadata=data.get("metadata", {}),
        )


class MemoryStore:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def load(self) -> List[MemoryRecord]:
        if not self.storage_path.exists():
            return []
        return [MemoryRecord.from_dict(item) for item in json.loads(self.storage_path.read_text())]

    def remember(self, record: MemoryRecord) -> None:
        entries = [item.to_dict() for item in self.load()]
        entries.append(record.to_dict())
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps(entries, ensure_ascii=False, indent=2))

    def recall(
        self,
        *,
        scope: Optional[MemoryScope] = None,
        key: Optional[str] = None,
        tag: Optional[str] = None,
        source: Optional[str] = None,
        limit: Optional[int] = None,
    ) -> List[MemoryRecord]:
        records = self.load()
        filtered = [
            record
            for record in records
            if (scope is None or record.scope == scope)
            and (key is None or record.key == key)
            and (tag is None or tag in record.tags)
            and (source is None or record.source == source)
        ]
        if limit is not None:
            return filtered[-limit:]
        return filtered

    def latest(self, key: str, scope: Optional[MemoryScope] = None) -> Optional[MemoryRecord]:
        matches = self.recall(key=key, scope=scope, limit=1)
        return matches[-1] if matches else None

    def record_run(self, task: Task, run: TaskRun, actor: str) -> None:
        self.remember(
            MemoryRecord(
                key=run.run_id,
                value=run.summary,
                scope=MemoryScope.RUN,
                source=task.source,
                run_id=run.run_id,
                task_id=task.task_id,
                tags=[run.status, run.medium],
                metadata={
                    "actor": actor,
                    "title": task.title,
                    "risk_level": task.risk_level.value,
                },
            )
        )
        self.remember(
            MemoryRecord(
                key=task.task_id,
                value=f"{run.status}: {run.summary}",
                scope=MemoryScope.EXPERIENCE,
                source=task.source,
                run_id=run.run_id,
                task_id=task.task_id,
                tags=["execution", run.status],
                metadata={
                    "title": task.title,
                    "medium": run.medium,
                },
            )
        )
