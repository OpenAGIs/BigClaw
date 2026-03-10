from dataclasses import dataclass
from pathlib import Path
from typing import Optional

from .budget import BudgetGuard, BudgetPolicy
from .memory import MemoryStore
from .models import RiskLevel, Task
from .observability import ObservabilityLedger, TaskRun
from .reports import render_task_run_report, write_report


@dataclass
class SchedulerDecision:
    medium: str
    approved: bool
    reason: str


@dataclass
class ExecutionRecord:
    decision: SchedulerDecision
    run: TaskRun
    report_path: Optional[str]


class Scheduler:
    def __init__(
        self,
        budget_policy: Optional[BudgetPolicy] = None,
        memory_store: Optional[MemoryStore] = None,
    ):
        self.budget_guard = BudgetGuard(budget_policy)
        self.memory_store = memory_store

    def decide(self, task: Task) -> SchedulerDecision:
        budget_decision = self.budget_guard.evaluate(task)
        if not budget_decision.allowed:
            return SchedulerDecision("none", False, budget_decision.reason)

        if task.risk_level == RiskLevel.HIGH:
            return SchedulerDecision("vm", False, "requires approval for high-risk task")

        if "browser" in task.required_tools:
            return SchedulerDecision("browser", True, "browser automation task")

        if task.risk_level == RiskLevel.MEDIUM:
            return SchedulerDecision("docker", True, "medium risk in docker")

        return SchedulerDecision("docker", True, "default low risk path")

    def execute(
        self,
        task: Task,
        run_id: str,
        ledger: ObservabilityLedger,
        report_path: Optional[str] = None,
        actor: str = "scheduler",
    ) -> ExecutionRecord:
        decision = self.decide(task)
        run = TaskRun.from_task(task, run_id=run_id, medium=decision.medium)
        run.log(
            "info",
            "task received",
            source=task.source,
            priority=int(task.priority),
            estimated_tokens=task.estimated_tokens,
            estimated_runtime_seconds=task.estimated_runtime_seconds,
            required_workers=task.required_workers,
        )
        run.trace(
            "scheduler.decide",
            "ok" if decision.approved else "pending",
            approved=decision.approved,
            medium=decision.medium,
        )
        run.audit(
            "scheduler.decision",
            actor,
            "approved" if decision.approved else "pending",
            reason=decision.reason,
        )

        resolved_report_path = None
        if report_path:
            resolved_report_path = str(Path(report_path))
            report_content = render_task_run_report(run)
            write_report(resolved_report_path, report_content)
            run.register_artifact("task-run-report", "report", resolved_report_path, format="markdown")

        final_status = "approved" if decision.approved else "needs-approval"
        run.finalize(final_status, decision.reason)
        ledger.append(run)
        if self.memory_store is not None:
            self.memory_store.record_run(task, run, actor)
        return ExecutionRecord(decision=decision, run=run, report_path=resolved_report_path)
