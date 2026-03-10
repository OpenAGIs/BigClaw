from pathlib import Path

from bigclaw.models import Priority, Task
from bigclaw.queue import PersistentTaskQueue


def test_queue_persistence_and_priority(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(
        Task(
            task_id="t2",
            source="linear",
            title="P1",
            description="",
            priority=Priority.P1,
            estimated_tokens=200,
        )
    )
    q.enqueue(Task(task_id="t1", source="linear", title="P0", description="", priority=Priority.P0))

    first = q.dequeue()
    second = q.dequeue_task()

    assert first["task_id"] == "t1"
    assert second is not None
    assert second.task_id == "t2"
    assert second.estimated_tokens == 200
    assert q.size() == 0
