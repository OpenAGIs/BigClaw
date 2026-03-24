"""Legacy Python orchestration surface frozen after Go mainline cutover."""

from dataclasses import dataclass, field
from typing import Dict, List, Optional, Sequence, Tuple

from .deprecation import LEGACY_RUNTIME_GUIDANCE
from .models import RiskLevel, Task


LEGACY_MAINLINE_STATUS = LEGACY_RUNTIME_GUIDANCE
GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/workflow/orchestration.go"


@dataclass
class DepartmentHandoff:
    department: str
    reason: str
    required_tools: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "department": self.department,
            "reason": self.reason,
            "required_tools": self.required_tools,
            "approvals": self.approvals,
        }


@dataclass
class OrchestrationPlan:
    task_id: str
    collaboration_mode: str
    handoffs: List[DepartmentHandoff] = field(default_factory=list)

    @property
    def departments(self) -> List[str]:
        return [handoff.department for handoff in self.handoffs]

    @property
    def department_count(self) -> int:
        return len(self.handoffs)

    @property
    def required_approvals(self) -> List[str]:
        approvals: List[str] = []
        for handoff in self.handoffs:
            for approval in handoff.approvals:
                if approval not in approvals:
                    approvals.append(approval)
        return approvals

    def to_dict(self) -> Dict[str, object]:
        return {
            "task_id": self.task_id,
            "collaboration_mode": self.collaboration_mode,
            "departments": self.departments,
            "required_approvals": self.required_approvals,
            "handoffs": [handoff.to_dict() for handoff in self.handoffs],
        }


@dataclass
class HandoffRequest:
    target_team: str
    reason: str
    status: str = "pending"
    required_approvals: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "target_team": self.target_team,
            "reason": self.reason,
            "status": self.status,
            "required_approvals": self.required_approvals,
        }


@dataclass
class OrchestrationPolicyDecision:
    tier: str
    upgrade_required: bool
    reason: str
    blocked_departments: List[str] = field(default_factory=list)
    entitlement_status: str = "included"
    billing_model: str = "standard-included"
    estimated_cost_usd: float = 0.0
    included_usage_units: int = 0
    overage_usage_units: int = 0
    overage_cost_usd: float = 0.0

    def to_dict(self) -> Dict[str, object]:
        return {
            "tier": self.tier,
            "upgrade_required": self.upgrade_required,
            "reason": self.reason,
            "blocked_departments": self.blocked_departments,
            "entitlement_status": self.entitlement_status,
            "billing_model": self.billing_model,
            "estimated_cost_usd": self.estimated_cost_usd,
            "included_usage_units": self.included_usage_units,
            "overage_usage_units": self.overage_usage_units,
            "overage_cost_usd": self.overage_cost_usd,
        }


class CrossDepartmentOrchestrator:
    def plan(self, task: Task) -> OrchestrationPlan:
        labels = {label.lower() for label in task.labels}
        tools = {tool.lower() for tool in task.required_tools}
        text = " ".join(
            [task.title.lower(), task.description.lower(), *task.acceptance_criteria, *task.validation_plan]
        )

        handoffs: List[DepartmentHandoff] = []
        self._append_unique(handoffs, "operations", self._operations_reason(task, labels, text))

        if task.required_tools or "github" in task.source.lower() or {"repo", "browser", "terminal"} & tools:
            self._append_unique(
                handoffs,
                "engineering",
                "implements automation and tool-driven execution",
                required_tools=sorted(tools),
            )

        if task.risk_level == RiskLevel.HIGH or labels & {"security", "compliance"} or "approval" in text:
            approvals = ["security-review"] if task.risk_level == RiskLevel.HIGH else []
            self._append_unique(
                handoffs,
                "security",
                "reviews elevated risk, compliance, or approval-sensitive work",
                approvals=approvals,
            )

        if labels & {"data", "analytics"} or tools & {"sql", "warehouse", "bi"}:
            self._append_unique(
                handoffs,
                "data",
                "validates analytics, warehouse, or measurement dependencies",
                required_tools=sorted(tools & {"sql", "warehouse", "bi"}),
            )

        if labels & {"customer", "support", "success"} or "customer" in text or "stakeholder" in text:
            self._append_unique(
                handoffs,
                "customer-success",
                "coordinates customer communication and rollout readiness",
            )

        collaboration_mode = "cross-functional" if len(handoffs) > 1 else "single-team"
        return OrchestrationPlan(task_id=task.task_id, collaboration_mode=collaboration_mode, handoffs=handoffs)

    def _operations_reason(self, task: Task, labels: set[str], text: str) -> str:
        if labels & {"program", "ops", "release"} or "rollout" in text or task.source.lower() in {"linear", "jira"}:
            return "coordinates issue intake, handoffs, and completion tracking"
        return "owns task intake and delivery coordination"

    def _append_unique(
        self,
        handoffs: List[DepartmentHandoff],
        department: str,
        reason: str,
        required_tools: Optional[Sequence[str]] = None,
        approvals: Optional[Sequence[str]] = None,
    ) -> None:
        for handoff in handoffs:
            if handoff.department == department:
                for tool_name in required_tools or []:
                    if tool_name not in handoff.required_tools:
                        handoff.required_tools.append(tool_name)
                for approval in approvals or []:
                    if approval not in handoff.approvals:
                        handoff.approvals.append(approval)
                return

        handoffs.append(
            DepartmentHandoff(
                department=department,
                reason=reason,
                required_tools=list(required_tools or []),
                approvals=list(approvals or []),
            )
        )


class PremiumOrchestrationPolicy:
    def apply(self, task: Task, plan: OrchestrationPlan) -> Tuple[OrchestrationPlan, OrchestrationPolicyDecision]:
        requested_units = max(1, plan.department_count)
        if self._is_premium(task):
            estimated_cost = self._estimate_cost(requested_units)
            return (
                plan,
                OrchestrationPolicyDecision(
                    tier="premium",
                    upgrade_required=False,
                    reason="premium tier enables advanced cross-department orchestration",
                    entitlement_status="included",
                    billing_model="premium-included",
                    estimated_cost_usd=estimated_cost,
                    included_usage_units=requested_units,
                ),
            )

        blocked_departments = [
            department for department in plan.departments if department not in {"operations", "engineering"}
        ]
        if not blocked_departments:
            estimated_cost = self._estimate_cost(requested_units)
            return (
                plan,
                OrchestrationPolicyDecision(
                    tier="standard",
                    upgrade_required=False,
                    reason="standard tier supports baseline orchestration",
                    entitlement_status="included",
                    billing_model="standard-included",
                    estimated_cost_usd=estimated_cost,
                    included_usage_units=requested_units,
                ),
            )

        constrained_handoffs = [
            handoff for handoff in plan.handoffs if handoff.department in {"operations", "engineering"}
        ]
        constrained_plan = OrchestrationPlan(
            task_id=plan.task_id,
            collaboration_mode="tier-limited",
            handoffs=constrained_handoffs,
        )
        included_units = max(1, constrained_plan.department_count)
        overage_units = len(blocked_departments)
        estimated_cost = self._estimate_cost(included_units) + self._estimate_overage_cost(overage_units)
        return (
            constrained_plan,
            OrchestrationPolicyDecision(
                tier="standard",
                upgrade_required=True,
                reason="premium tier required for advanced cross-department orchestration",
                blocked_departments=blocked_departments,
                entitlement_status="upgrade-required",
                billing_model="standard-blocked",
                estimated_cost_usd=estimated_cost,
                included_usage_units=included_units,
                overage_usage_units=overage_units,
                overage_cost_usd=self._estimate_overage_cost(overage_units),
            ),
        )

    def _is_premium(self, task: Task) -> bool:
        return any(label.lower() in {"premium", "enterprise"} for label in task.labels)

    def _estimate_cost(self, usage_units: int) -> float:
        return round(1.5 * max(1, usage_units), 2)

    def _estimate_overage_cost(self, usage_units: int) -> float:
        return round(4.0 * max(0, usage_units), 2)


@dataclass
class LifecycleFanoutTarget:
    run_id: str
    task_id: str
    current_status: str = "unknown"
    owner: str = "automation"
    note: str = ""
    blocked: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "run_id": self.run_id,
            "task_id": self.task_id,
            "current_status": self.current_status,
            "owner": self.owner,
            "note": self.note,
            "blocked": self.blocked,
        }


@dataclass
class LifecycleFanoutOperation:
    action: str
    targets: List[LifecycleFanoutTarget] = field(default_factory=list)
    reason: str = ""
    concurrency: int = 1
    takeover_queue: str = ""

    @property
    def target_count(self) -> int:
        return len(self.targets)

    @property
    def blocked_target_count(self) -> int:
        return sum(1 for target in self.targets if target.blocked)

    def to_dict(self) -> Dict[str, object]:
        return {
            "action": self.action,
            "reason": self.reason,
            "concurrency": self.concurrency,
            "takeover_queue": self.takeover_queue,
            "target_count": self.target_count,
            "blocked_target_count": self.blocked_target_count,
            "targets": [target.to_dict() for target in self.targets],
        }


@dataclass
class LifecycleFanoutPlan:
    name: str
    operations: List[LifecycleFanoutOperation] = field(default_factory=list)
    requested_by: str = "system"

    @property
    def total_operations(self) -> int:
        return len(self.operations)

    @property
    def total_targets(self) -> int:
        return sum(operation.target_count for operation in self.operations)

    @property
    def blocked_target_count(self) -> int:
        return sum(operation.blocked_target_count for operation in self.operations)

    @property
    def action_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for operation in self.operations:
            counts[operation.action] = counts.get(operation.action, 0) + operation.target_count
        return counts

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "requested_by": self.requested_by,
            "total_operations": self.total_operations,
            "total_targets": self.total_targets,
            "blocked_target_count": self.blocked_target_count,
            "action_counts": self.action_counts,
            "operations": [operation.to_dict() for operation in self.operations],
        }


def render_orchestration_plan(
    plan: OrchestrationPlan,
    policy_decision: Optional[OrchestrationPolicyDecision] = None,
    handoff_request: Optional[HandoffRequest] = None,
) -> str:
    lines = [
        "# Cross-Department Orchestration Plan",
        "",
        f"- Task ID: {plan.task_id}",
        f"- Collaboration Mode: {plan.collaboration_mode}",
        f"- Departments: {', '.join(plan.departments) if plan.departments else 'none'}",
        f"- Required Approvals: {', '.join(plan.required_approvals) if plan.required_approvals else 'none'}",
    ]

    if policy_decision is not None:
        blocked = ", ".join(policy_decision.blocked_departments) if policy_decision.blocked_departments else "none"
        lines.extend(
            [
                f"- Tier: {policy_decision.tier}",
                f"- Upgrade Required: {policy_decision.upgrade_required}",
                f"- Entitlement Status: {policy_decision.entitlement_status}",
                f"- Billing Model: {policy_decision.billing_model}",
                f"- Estimated Cost (USD): {policy_decision.estimated_cost_usd:.2f}",
                f"- Included Usage Units: {policy_decision.included_usage_units}",
                f"- Overage Usage Units: {policy_decision.overage_usage_units}",
                f"- Overage Cost (USD): {policy_decision.overage_cost_usd:.2f}",
                f"- Policy Reason: {policy_decision.reason}",
                f"- Blocked Departments: {blocked}",
            ]
        )

    if handoff_request is not None:
        approvals = ", ".join(handoff_request.required_approvals) if handoff_request.required_approvals else "none"
        lines.extend(
            [
                f"- Human Handoff Team: {handoff_request.target_team}",
                f"- Human Handoff Status: {handoff_request.status}",
                f"- Human Handoff Reason: {handoff_request.reason}",
                f"- Human Handoff Approvals: {approvals}",
            ]
        )

    lines.extend(["", "## Handoffs", ""])

    if not plan.handoffs:
        lines.append("- None")
    else:
        for handoff in plan.handoffs:
            tools = ", ".join(handoff.required_tools) if handoff.required_tools else "none"
            approvals = ", ".join(handoff.approvals) if handoff.approvals else "none"
            lines.append(
                f"- {handoff.department}: reason={handoff.reason} tools={tools} approvals={approvals}"
            )

    return "\n".join(lines) + "\n"


def render_lifecycle_fanout_plan(plan: LifecycleFanoutPlan) -> str:
    action_mix = " ".join(
        f"{action}={count}" for action, count in sorted(plan.action_counts.items())
    ) or "none"
    lines = [
        "# Lifecycle Fanout Plan",
        "",
        f"- Name: {plan.name}",
        f"- Requested By: {plan.requested_by}",
        f"- Operations: {plan.total_operations}",
        f"- Target Count: {plan.total_targets}",
        f"- Blocked Targets: {plan.blocked_target_count}",
        f"- Action Mix: {action_mix}",
        "",
        "## Operations",
        "",
    ]

    if not plan.operations:
        lines.append("- None")
        return "\n".join(lines) + "\n"

    for operation in plan.operations:
        queue_name = operation.takeover_queue or "none"
        reason = operation.reason or "none"
        lines.append(
            f"- {operation.action}: targets={operation.target_count} blocked={operation.blocked_target_count} "
            f"concurrency={operation.concurrency} takeover_queue={queue_name} reason={reason}"
        )
        for target in operation.targets:
            note = target.note or "none"
            lines.append(
                f"  - {target.run_id}: task={target.task_id} status={target.current_status} "
                f"owner={target.owner} blocked={target.blocked} note={note}"
            )

    return "\n".join(lines) + "\n"
