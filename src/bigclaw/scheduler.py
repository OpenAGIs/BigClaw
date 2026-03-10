from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .models import Task, RiskLevel
from .observability import ObservabilityLedger, TaskRun
from .runtime import ClawWorkerRuntime, ToolCallResult
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
    tool_results: list[ToolCallResult] = field(default_factory=list)


class Scheduler:
    def __init__(self, worker_runtime: Optional[ClawWorkerRuntime] = None):
        self.worker_runtime = worker_runtime or ClawWorkerRuntime()

    def decide(self, task: Task) -> SchedulerDecision:
        if task.budget < 0:
            return SchedulerDecision("none", False, "invalid budget")

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
        run.log("info", "task received", source=task.source, priority=int(task.priority))
        run.trace(
            "scheduler.decide",
            "ok" if decision.approved else "pending",
            approved=decision.approved,
            medium=decision.medium,
        )
        run.audit("scheduler.decision", actor, "approved" if decision.approved else "pending", reason=decision.reason)

        worker_execution = self.worker_runtime.execute(task, decision, run, actor=actor)

        resolved_report_path = None
        if report_path:
            resolved_report_path = str(Path(report_path))
            report_content = render_task_run_report(run)
            write_report(resolved_report_path, report_content)
            run.register_artifact("task-run-report", "report", resolved_report_path, format="markdown")

        final_status = "approved" if decision.approved else "needs-approval"
        run.finalize(final_status, decision.reason)
        ledger.append(run)
        return ExecutionRecord(
            decision=decision,
            run=run,
            report_path=resolved_report_path,
            tool_results=worker_execution.tool_results,
        )
