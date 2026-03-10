from pathlib import Path

from bigclaw.memory import MemoryRecord, MemoryScope, MemoryStore
from bigclaw.models import Priority, Task
from bigclaw.observability import TaskRun


def test_memory_store_persists_and_filters_records(tmp_path: Path):
    store = MemoryStore(str(tmp_path / "memory.json"))
    store.remember(
        MemoryRecord(
            key="project-alpha",
            value="prioritize observability rollout",
            scope=MemoryScope.PROJECT,
            source="linear",
            tags=["planning", "epic-5"],
        )
    )
    store.remember(
        MemoryRecord(
            key="org-standard",
            value="always attach validation evidence",
            scope=MemoryScope.ORG,
            source="workflow",
            tags=["policy"],
        )
    )

    project_records = store.recall(scope=MemoryScope.PROJECT)
    tagged_records = store.recall(tag="policy")

    assert len(project_records) == 1
    assert project_records[0].value == "prioritize observability rollout"
    assert tagged_records[0].scope == MemoryScope.ORG


def test_memory_store_records_run_and_experience_entries(tmp_path: Path):
    store = MemoryStore(str(tmp_path / "memory.json"))
    task = Task(
        task_id="BIG-501",
        source="linear",
        title="Add memory foundation",
        description="capture reusable run knowledge",
        priority=Priority.P1,
    )
    run = TaskRun.from_task(task, run_id="run-memory-1", medium="docker")
    run.finalize("approved", "memory capture complete")

    store.record_run(task, run, actor="scheduler")

    run_entry = store.latest("run-memory-1", scope=MemoryScope.RUN)
    experience_entry = store.latest("BIG-501", scope=MemoryScope.EXPERIENCE)

    assert run_entry is not None
    assert run_entry.metadata["actor"] == "scheduler"
    assert experience_entry is not None
    assert experience_entry.value == "approved: memory capture complete"
