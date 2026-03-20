"""Legacy Python scheduler surface frozen after Go mainline cutover."""

from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

from .deprecation import LEGACY_RUNTIME_GUIDANCE
from .audit_events import (
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    SCHEDULER_DECISION_EVENT,
)
from .models import RiskLevel, Task
from .observability import ObservabilityLedger, TaskRun
from .orchestration import (
    CrossDepartmentOrchestrator,
    HandoffRequest,
    OrchestrationPlan,
    OrchestrationPolicyDecision,
    PremiumOrchestrationPolicy,
)
from .reports import render_task_run_detail_page, render_task_run_report, write_report
from .risk import RiskScore, RiskScorer
from .runtime import ClawWorkerRuntime, ToolCallResult


LEGACY_MAINLINE_STATUS = LEGACY_RUNTIME_GUIDANCE
GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/scheduler/scheduler.go"


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
    handoff_request: Optional[HandoffRequest] = None
    risk_score: Optional[RiskScore] = None
    tool_results: list[ToolCallResult] = field(default_factory=list)


class Scheduler:
    def __init__(
        self,
        worker_runtime: Optional[ClawWorkerRuntime] = None,
        orchestrator: Optional[CrossDepartmentOrchestrator] = None,
        orchestration_policy: Optional[PremiumOrchestrationPolicy] = None,
        risk_scorer: Optional[RiskScorer] = None,
    ):
        self.worker_runtime = worker_runtime or ClawWorkerRuntime()
        self.orchestrator = orchestrator or CrossDepartmentOrchestrator()
        self.orchestration_policy = orchestration_policy or PremiumOrchestrationPolicy()
        self.risk_scorer = risk_scorer or RiskScorer()

    def decide(self, task: Task, risk_score: Optional[RiskScore] = None) -> SchedulerDecision:
        resolved_risk = risk_score or self.risk_scorer.score_task(task)
        if task.budget < 0:
            return SchedulerDecision("none", False, "invalid budget")

        if resolved_risk.level == RiskLevel.HIGH:
            return SchedulerDecision("vm", False, "requires approval for high-risk task")

        if "browser" in task.required_tools:
            return SchedulerDecision("browser", True, "browser automation task")

        if resolved_risk.level == RiskLevel.MEDIUM:
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
        risk_score = self.risk_scorer.score_task(task)
        decision = self.decide(task, risk_score=risk_score)
        raw_plan = self.orchestrator.plan(task)
        orchestration_plan, policy_decision = self.orchestration_policy.apply(task, raw_plan)
        handoff_request = self._build_handoff_request(decision, orchestration_plan, policy_decision)
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
            "risk.score",
            risk_score.level.value,
            total=risk_score.total,
            requires_approval=risk_score.requires_approval,
            factors=[factor.name for factor in risk_score.factors],
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
            entitlement_status=policy_decision.entitlement_status,
            billing_model=policy_decision.billing_model,
            estimated_cost_usd=policy_decision.estimated_cost_usd,
            included_usage_units=policy_decision.included_usage_units,
            overage_usage_units=policy_decision.overage_usage_units,
            overage_cost_usd=policy_decision.overage_cost_usd,
            blocked_departments=policy_decision.blocked_departments,
        )
        run.audit("scheduler.decision", actor, "approved" if decision.approved else "pending", reason=decision.reason)
        run.audit_spec_event(
            SCHEDULER_DECISION_EVENT,
            actor,
            "approved" if decision.approved else "pending",
            task_id=task.task_id,
            run_id=run_id,
            medium=decision.medium,
            approved=decision.approved,
            reason=decision.reason,
            risk_level=risk_score.level.value,
            risk_score=risk_score.total,
        )
        run.audit(
            "risk.score",
            actor,
            risk_score.level.value,
            total=risk_score.total,
            requires_approval=risk_score.requires_approval,
            summary=risk_score.summary,
        )
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
            entitlement_status=policy_decision.entitlement_status,
            billing_model=policy_decision.billing_model,
            estimated_cost_usd=policy_decision.estimated_cost_usd,
            included_usage_units=policy_decision.included_usage_units,
            overage_usage_units=policy_decision.overage_usage_units,
            overage_cost_usd=policy_decision.overage_cost_usd,
            blocked_departments=policy_decision.blocked_departments,
        )
        if task.budget_override_reason.strip():
            run.audit_spec_event(
                BUDGET_OVERRIDE_EVENT,
                task.budget_override_actor or actor,
                "recorded",
                task_id=task.task_id,
                run_id=run_id,
                requested_budget=task.budget,
                approved_budget=task.budget + task.budget_override_amount,
                override_actor=task.budget_override_actor or actor,
                reason=task.budget_override_reason,
            )
        if handoff_request is not None:
            run.trace(
                "orchestration.handoff",
                handoff_request.status,
                target_team=handoff_request.target_team,
                required_approvals=handoff_request.required_approvals,
            )
            run.audit(
                "orchestration.handoff",
                actor,
                handoff_request.status,
                target_team=handoff_request.target_team,
                reason=handoff_request.reason,
                required_approvals=handoff_request.required_approvals,
            )
            run.audit_spec_event(
                MANUAL_TAKEOVER_EVENT,
                actor,
                handoff_request.status,
                task_id=task.task_id,
                run_id=run_id,
                target_team=handoff_request.target_team,
                reason=handoff_request.reason,
                requested_by=actor,
                required_approvals=handoff_request.required_approvals,
            )
            run.audit_spec_event(
                FLOW_HANDOFF_EVENT,
                actor,
                handoff_request.status,
                task_id=task.task_id,
                run_id=run_id,
                source_stage="scheduler",
                target_team=handoff_request.target_team,
                reason=handoff_request.reason,
                collaboration_mode=orchestration_plan.collaboration_mode,
                required_approvals=handoff_request.required_approvals,
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
            handoff_request=handoff_request,
            risk_score=risk_score,
            tool_results=worker_execution.tool_results,
        )

    def _build_handoff_request(
        self,
        decision: SchedulerDecision,
        plan: OrchestrationPlan,
        policy_decision: OrchestrationPolicyDecision,
    ) -> Optional[HandoffRequest]:
        if not decision.approved:
            return HandoffRequest(
                target_team="security",
                reason=decision.reason,
                required_approvals=plan.required_approvals or ["security-review"],
            )
        if policy_decision.upgrade_required:
            return HandoffRequest(
                target_team="operations",
                reason=policy_decision.reason,
                required_approvals=["ops-manager"],
            )
        return None
