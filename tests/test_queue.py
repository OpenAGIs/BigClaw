import json
from pathlib import Path

from bigclaw.models import Task, Priority
from bigclaw.queue import PersistentTaskQueue


def test_queue_persistence_and_priority(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(Task(task_id="t2", source="linear", title="P1", description="", priority=Priority.P1))
    q.enqueue(Task(task_id="t1", source="linear", title="P0", description="", priority=Priority.P0))

    assert q.size() == 2
    first = q.dequeue()
    assert first["task_id"] == "t1"

    q2 = PersistentTaskQueue(str(qfile))
    assert q2.size() == 1


def test_queue_creates_parent_directory_and_preserves_task_payload(tmp_path: Path):
    qfile = tmp_path / "state" / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(
        Task(
            task_id="t-meta",
            source="linear",
            title="Persist metadata",
            description="keep fields",
            labels=["platform"],
            required_tools=["browser"],
            acceptance_criteria=["queue survives restart"],
            validation_plan=["pytest tests/test_queue.py"],
            priority=Priority.P1,
        )
    )

    reloaded = PersistentTaskQueue(str(qfile))
    task = reloaded.dequeue_task()

    assert qfile.exists()
    assert task is not None
    assert task.labels == ["platform"]
    assert task.required_tools == ["browser"]
    assert task.acceptance_criteria == ["queue survives restart"]
    assert task.validation_plan == ["pytest tests/test_queue.py"]


def test_queue_dead_letter_and_retry_persist_across_reload(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))
    q.enqueue(Task(task_id="t-dead", source="linear", title="dead", description="", priority=Priority.P0))

    task = q.dequeue_task()
    assert task is not None

    q.dead_letter(task, reason="executor crashed")

    dead_letters = q.list_dead_letters()
    assert q.size() == 0
    assert len(dead_letters) == 1
    assert dead_letters[0].task.task_id == "t-dead"
    assert dead_letters[0].reason == "executor crashed"

    reloaded = PersistentTaskQueue(str(qfile))
    persisted_dead_letters = reloaded.list_dead_letters()
    assert len(persisted_dead_letters) == 1
    assert persisted_dead_letters[0].task.task_id == "t-dead"

    assert reloaded.retry_dead_letter("t-dead") is True
    assert reloaded.retry_dead_letter("missing") is False
    assert reloaded.list_dead_letters() == []
    assert reloaded.size() == 1

    replayed = PersistentTaskQueue(str(qfile)).dequeue_task()
    assert replayed is not None
    assert replayed.task_id == "t-dead"


def test_queue_loads_legacy_list_storage(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    qfile.write_text(
        json.dumps(
            [
                {
                    "priority": 0,
                    "task_id": "legacy",
                    "task": Task(
                        task_id="legacy",
                        source="linear",
                        title="legacy",
                        description="legacy payload",
                        priority=Priority.P0,
                    ).to_dict(),
                }
            ]
        ),
        encoding="utf-8",
    )

    queue = PersistentTaskQueue(str(qfile))

    assert queue.size() == 1
    task = queue.dequeue_task()
    assert task is not None
    assert task.task_id == "legacy"
