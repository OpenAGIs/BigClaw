from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .models import Task, RiskLevel
from .observability import ObservabilityLedger, TaskRun
from .orchestration import CrossDepartmentOrchestrator, OrchestrationPlan
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
    orchestration_plan: Optional[OrchestrationPlan] = None
    tool_results: list[ToolCallResult] = field(default_factory=list)


class Scheduler:
    def __init__(
        self,
        worker_runtime: Optional[ClawWorkerRuntime] = None,
        orchestrator: Optional[CrossDepartmentOrchestrator] = None,
    ):
        self.worker_runtime = worker_runtime or ClawWorkerRuntime()
        self.orchestrator = orchestrator or CrossDepartmentOrchestrator()

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
        orchestration_plan = self.orchestrator.plan(task)
        run = TaskRun.from_task(task, run_id=run_id, medium=decision.medium)
        run.log("info", "task received", source=task.source, priority=int(task.priority))
        run.log(
            "info",
            "orchestration planned",
            collaboration_mode=orchestration_plan.collaboration_mode,
            departments=orchestration_plan.departments,
        )
        run.trace(
            "scheduler.decide",
            "ok" if decision.approved else "pending",
            approved=decision.approved,
            medium=decision.medium,
        )
        run.trace(
            "orchestration.plan",
            "ready",
            collaboration_mode=orchestration_plan.collaboration_mode,
            departments=orchestration_plan.departments,
            handoffs=orchestration_plan.department_count,
        )
        run.audit("scheduler.decision", actor, "approved" if decision.approved else "pending", reason=decision.reason)
        run.audit(
            "orchestration.plan",
            actor,
            "ready",
            collaboration_mode=orchestration_plan.collaboration_mode,
            departments=orchestration_plan.departments,
            approvals=orchestration_plan.required_approvals,
        )

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
            orchestration_plan=orchestration_plan,
            tool_results=worker_execution.tool_results,
        )
