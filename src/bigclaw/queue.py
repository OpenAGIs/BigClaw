"""Legacy Python queue surface frozen after Go mainline cutover."""

import heapq
import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional, Sequence, Set, Tuple

from .legacy_shim import LEGACY_RUNTIME_GUIDANCE
from .models import Task


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


class ParallelIssueQueue:
    def __init__(self, queue_path: str):
        self.queue_path = Path(queue_path)
        self.payload = json.loads(self.queue_path.read_text())

    def project_slug(self) -> str:
        return str(self.payload["project"]["slug_id"])

    def activate_state_id(self) -> str:
        return str(self.payload["policy"]["activate_state_id"])

    def target_in_progress(self) -> int:
        return int(self.payload["policy"]["target_in_progress"])

    def refill_states(self) -> Set[str]:
        return {str(name) for name in self.payload["policy"].get("refill_states", [])}

    def issue_order(self) -> List[str]:
        return [str(identifier) for identifier in self.payload.get("issue_order", [])]

    def issue_records(self) -> List[dict]:
        return list(self.payload.get("issues", []))

    def issue_identifiers(self) -> List[str]:
        return [str(record["identifier"]) for record in self.issue_records()]

    def select_candidates(
        self,
        active_identifiers: Set[str],
        issue_states: Dict[str, str],
        target_in_progress: Optional[int] = None,
    ) -> List[str]:
        target = self.target_in_progress() if target_in_progress is None else int(target_in_progress)
        needed = max(target - len(active_identifiers), 0)
        if needed == 0:
            return []
        candidates: List[str] = []
        refill_states = self.refill_states()
        for identifier in self.issue_order():
            if needed == 0:
                break
            if identifier in active_identifiers:
                continue
            if issue_states.get(identifier) in refill_states:
                candidates.append(identifier)
                needed -= 1
        return candidates


def issue_state_map(issues: Sequence[dict]) -> Dict[str, str]:
    state_map: Dict[str, str] = {}
    for issue in issues:
        identifier = str(issue.get("identifier", "")).strip()
        state = issue.get("state") or {}
        state_name = str(state.get("name", issue.get("state_name", ""))).strip()
        if identifier and state_name:
            state_map[identifier] = state_name
    return state_map


@dataclass
class MemoryPattern:
    task_id: str
    title: str
    labels: List[str] = field(default_factory=list)
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    summary: str = ""

    def to_dict(self) -> Dict:
        return {
            "task_id": self.task_id,
            "title": self.title,
            "labels": list(self.labels),
            "required_tools": list(self.required_tools),
            "acceptance_criteria": list(self.acceptance_criteria),
            "validation_plan": list(self.validation_plan),
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict) -> "MemoryPattern":
        return cls(
            task_id=str(data.get("task_id", "")),
            title=str(data.get("title", "")),
            labels=[str(item) for item in data.get("labels", [])],
            required_tools=[str(item) for item in data.get("required_tools", [])],
            acceptance_criteria=[str(item) for item in data.get("acceptance_criteria", [])],
            validation_plan=[str(item) for item in data.get("validation_plan", [])],
            summary=str(data.get("summary", "")),
        )


class TaskMemoryStore:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def _load_patterns(self) -> List[MemoryPattern]:
        if not self.storage_path.exists():
            return []
        payload = json.loads(self.storage_path.read_text())
        return [MemoryPattern.from_dict(item) for item in payload]

    def _write_patterns(self, patterns: List[MemoryPattern]) -> None:
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps([item.to_dict() for item in patterns], ensure_ascii=False, indent=2))

    def remember_success(self, task: Task, summary: str = "") -> None:
        patterns = [item for item in self._load_patterns() if item.task_id != task.task_id]
        patterns.append(
            MemoryPattern(
                task_id=task.task_id,
                title=task.title,
                labels=list(task.labels),
                required_tools=list(task.required_tools),
                acceptance_criteria=list(task.acceptance_criteria),
                validation_plan=list(task.validation_plan),
                summary=summary,
            )
        )
        self._write_patterns(patterns)

    def suggest_rules(self, task: Task, limit: int = 3) -> Dict[str, List[str]]:
        ranked: List[Tuple[float, MemoryPattern]] = []
        for pattern in self._load_patterns():
            score = self._score(task, pattern)
            if score <= 0:
                continue
            ranked.append((score, pattern))

        ranked.sort(key=lambda item: item[0], reverse=True)
        selected = [item[1] for item in ranked[: max(1, limit)]]

        acceptance = list(task.acceptance_criteria)
        validation = list(task.validation_plan)
        for pattern in selected:
            for item in pattern.acceptance_criteria:
                if item not in acceptance:
                    acceptance.append(item)
            for item in pattern.validation_plan:
                if item not in validation:
                    validation.append(item)

        return {
            "acceptance_criteria": acceptance,
            "validation_plan": validation,
            "matched_task_ids": [item.task_id for item in selected],
        }

    @staticmethod
    def _score(task: Task, pattern: MemoryPattern) -> float:
        label_overlap = len(set(task.labels) & set(pattern.labels))
        tool_overlap = len(set(task.required_tools) & set(pattern.required_tools))
        return float(label_overlap * 2 + tool_overlap)
