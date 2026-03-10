from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .models import RiskLevel, Task
from .observability import ObservabilityLedger, TaskRun
from .orchestration import (
    CrossDepartmentOrchestrator,
    OrchestrationPlan,
    OrchestrationPolicyDecision,
    PremiumOrchestrationPolicy,
)
from .reports import render_task_run_detail_page, render_task_run_report, write_report
from .runtime import ClawWorkerRuntime, ToolCallResult


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
    orchestration_policy: Optional[OrchestrationPolicyDecision] = None
    tool_results: list[ToolCallResult] = field(default_factory=list)


class Scheduler:
    def __init__(
        self,
        worker_runtime: Optional[ClawWorkerRuntime] = None,
        orchestrator: Optional[CrossDepartmentOrchestrator] = None,
        orchestration_policy: Optional[PremiumOrchestrationPolicy] = None,
    ):
        self.worker_runtime = worker_runtime or ClawWorkerRuntime()
        self.orchestrator = orchestrator or CrossDepartmentOrchestrator()
        self.orchestration_policy = orchestration_policy or PremiumOrchestrationPolicy()

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
        raw_plan = self.orchestrator.plan(task)
        orchestration_plan, policy_decision = self.orchestration_policy.apply(task, raw_plan)
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
        run.trace(
            "orchestration.policy",
            "upgrade-required" if policy_decision.upgrade_required else "ok",
            tier=policy_decision.tier,
            blocked_departments=policy_decision.blocked_departments,
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
        run.audit(
            "orchestration.policy",
            actor,
            "upgrade-required" if policy_decision.upgrade_required else "enabled",
            tier=policy_decision.tier,
            reason=policy_decision.reason,
            blocked_departments=policy_decision.blocked_departments,
        )

        worker_execution = self.worker_runtime.execute(task, decision, run, actor=actor)

        final_status = "approved" if decision.approved else "needs-approval"
        run.finalize(final_status, decision.reason)

        resolved_report_path = None
        if report_path:
            resolved_report_path = str(Path(report_path))
            report_content = render_task_run_report(run)
            write_report(resolved_report_path, report_content)
            detail_page_path = str(Path(report_path).with_suffix(".html"))
            write_report(detail_page_path, render_task_run_detail_page(run))
            run.register_artifact("task-run-detail", "page", detail_page_path, format="html")
            run.register_artifact("task-run-report", "report", resolved_report_path, format="markdown")

        ledger.append(run)
        return ExecutionRecord(
            decision=decision,
            run=run,
            report_path=resolved_report_path,
            orchestration_plan=orchestration_plan,
            orchestration_policy=policy_decision,
            tool_results=worker_execution.tool_results,
        )
