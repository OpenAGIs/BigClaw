from pathlib import Path

from bigclaw.budget import BudgetGuard, BudgetPolicy
from bigclaw.memory import MemoryScope, MemoryStore
from bigclaw.models import Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.scheduler import Scheduler


def test_budget_guard_rejects_token_overflow():
    decision = BudgetGuard(BudgetPolicy(max_tokens=1000)).evaluate(
        Task(
            task_id="BIG-503",
            source="linear",
            title="Large run",
            description="token heavy",
            estimated_tokens=1500,
        )
    )

    assert decision.allowed is False
    assert decision.dimension == "tokens"


def test_scheduler_records_memory_for_successful_execution(tmp_path: Path):
    memory_store = MemoryStore(str(tmp_path / "memory.json"))
    scheduler = Scheduler(
        budget_policy=BudgetPolicy(max_cost=5.0, max_tokens=5000, max_runtime_seconds=60, max_workers=2),
        memory_store=memory_store,
    )
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))

    record = scheduler.execute(
        Task(
            task_id="BIG-503",
            source="linear",
            title="Budget aware task",
            description="fits budget",
            budget=1.5,
            estimated_tokens=300,
            estimated_runtime_seconds=10,
            required_workers=1,
        ),
        run_id="budget-ok-1",
        ledger=ledger,
    )

    experience_entry = memory_store.latest("BIG-503", scope=MemoryScope.EXPERIENCE)

    assert record.decision.approved is True
    assert experience_entry is not None
    assert experience_entry.metadata["medium"] == "docker"


def test_scheduler_rejects_worker_budget():
    scheduler = Scheduler(budget_policy=BudgetPolicy(max_workers=1))

    decision = scheduler.decide(
        Task(
            task_id="BIG-503-workers",
            source="jira",
            title="Parallel rollout",
            description="needs more workers",
            required_workers=2,
        )
    )

    assert decision.approved is False
    assert decision.medium == "none"
    assert "worker budget exceeded" in decision.reason
