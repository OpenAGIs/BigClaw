import heapq
import json
from pathlib import Path
from typing import List, Optional, Tuple

from .models import Task


class PersistentTaskQueue:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)
        self._heap: List[Tuple[int, str, dict]] = []
        self._load()

    def _load(self) -> None:
        if not self.storage_path.exists():
            return
        data = json.loads(self.storage_path.read_text())
        for item in data:
            heapq.heappush(self._heap, (item["priority"], item["task_id"], item["task"]))

    def _save(self) -> None:
        data = [
            {"priority": p, "task_id": tid, "task": task}
            for (p, tid, task) in self._heap
        ]
        self.storage_path.write_text(json.dumps(data, ensure_ascii=False, indent=2))

    def enqueue(self, task: Task) -> None:
        heapq.heappush(self._heap, (int(task.priority), task.task_id, task.to_dict()))
        self._save()

    def dequeue(self) -> Optional[dict]:
        if not self._heap:
            return None
        _p, _tid, task = heapq.heappop(self._heap)
        self._save()
        return task

    def size(self) -> int:
        return len(self._heap)
