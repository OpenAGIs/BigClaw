"""Legacy Python queue surface frozen after Go mainline cutover."""

import heapq
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Tuple

from .models import Task
from .service import LEGACY_RUNTIME_GUIDANCE


LEGACY_MAINLINE_STATUS = LEGACY_RUNTIME_GUIDANCE
GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/queue/queue.go"

@dataclass(frozen=True)
class DeadLetterEntry:
    task: Task
    reason: str = ""

    def to_dict(self) -> dict:
        return {"task_id": self.task.task_id, "task": self.task.to_dict(), "reason": self.reason}

    @classmethod
    def from_dict(cls, data: dict) -> "DeadLetterEntry":
        return cls(
            task=Task.from_dict(data["task"]),
            reason=str(data.get("reason", "")),
        )


class PersistentTaskQueue:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)
        self._heap: List[Tuple[int, str, dict]] = []
        self._dead_letters: Dict[str, DeadLetterEntry] = {}
        self._load()

    def _load(self) -> None:
        self._heap = []
        self._dead_letters = {}
        if not self.storage_path.exists():
            return
        data = json.loads(self.storage_path.read_text(encoding="utf-8"))

        if isinstance(data, list):
            queue_items = data
            dead_letter_items: List[dict] = []
        else:
            queue_items = data.get("queue", [])
            dead_letter_items = data.get("dead_letters", [])

        for item in queue_items:
            heapq.heappush(self._heap, (item["priority"], item["task_id"], item["task"]))
        for item in dead_letter_items:
            entry = DeadLetterEntry.from_dict(item)
            self._dead_letters[entry.task.task_id] = entry

    def _save(self) -> None:
        data = {
            "queue": [
                {"priority": p, "task_id": tid, "task": task}
                for (p, tid, task) in sorted(self._heap)
            ],
            "dead_letters": [
                entry.to_dict()
                for _, entry in sorted(self._dead_letters.items())
            ],
        }
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        tmp_path = self.storage_path.with_name(f"{self.storage_path.name}.tmp")
        tmp_path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        tmp_path.replace(self.storage_path)

    def _drop_queued_task(self, task_id: str) -> None:
        if not any(queued_task_id == task_id for _, queued_task_id, _ in self._heap):
            return
        self._heap = [
            (priority, queued_task_id, task)
            for priority, queued_task_id, task in self._heap
            if queued_task_id != task_id
        ]
        heapq.heapify(self._heap)

    def enqueue(self, task: Task) -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters.pop(task.task_id, None)
        heapq.heappush(self._heap, (int(task.priority), task.task_id, task.to_dict()))
        self._save()

    def dequeue(self) -> Optional[dict]:
        if not self._heap:
            return None
        _p, _tid, task = heapq.heappop(self._heap)
        self._save()
        return task

    def dequeue_task(self) -> Optional[Task]:
        task = self.dequeue()
        if task is None:
            return None
        return Task.from_dict(task)

    def dead_letter(self, task: Task, reason: str = "") -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters[task.task_id] = DeadLetterEntry(task=task, reason=reason)
        self._save()

    def list_dead_letters(self) -> List[DeadLetterEntry]:
        return [entry for _, entry in sorted(self._dead_letters.items())]

    def retry_dead_letter(self, task_id: str) -> bool:
        entry = self._dead_letters.pop(task_id, None)
        if entry is None:
            return False
        self._drop_queued_task(task_id)
        heapq.heappush(self._heap, (int(entry.task.priority), entry.task.task_id, entry.task.to_dict()))
        self._save()
        return True

    def size(self) -> int:
        return len(self._heap)

    def peek_tasks(self) -> List[Task]:
        return [Task.from_dict(task) for (_p, _tid, task) in sorted(self._heap)]
