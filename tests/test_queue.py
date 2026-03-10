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
