from dataclasses import dataclass, field
from html import escape
from typing import Dict, List, Optional, Sequence, Tuple

from .models import RiskLevel, Task


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

    def to_dict(self) -> Dict[str, object]:
        return {
            "tier": self.tier,
            "upgrade_required": self.upgrade_required,
            "reason": self.reason,
            "blocked_departments": self.blocked_departments,
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
        if self._is_premium(task):
            return (
                plan,
                OrchestrationPolicyDecision(
                    tier="premium",
                    upgrade_required=False,
                    reason="premium tier enables advanced cross-department orchestration",
                ),
            )

        blocked_departments = [
            department for department in plan.departments if department not in {"operations", "engineering"}
        ]
        if not blocked_departments:
            return (
                plan,
                OrchestrationPolicyDecision(
                    tier="standard",
                    upgrade_required=False,
                    reason="standard tier supports baseline orchestration",
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
        return (
            constrained_plan,
            OrchestrationPolicyDecision(
                tier="standard",
                upgrade_required=True,
                reason="premium tier required for advanced cross-department orchestration",
                blocked_departments=blocked_departments,
            ),
        )

    def _is_premium(self, task: Task) -> bool:
        return any(label.lower() in {"premium", "enterprise"} for label in task.labels)


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


def render_orchestration_canvas(
    task: Task,
    plan: OrchestrationPlan,
    policy_decision: Optional[OrchestrationPolicyDecision] = None,
    handoff_request: Optional[HandoffRequest] = None,
) -> str:
    handoff_cards = "".join(
        f"""
        <section class=\"card handoff\">
          <div class=\"eyebrow\">Lane {index}</div>
          <h2>{escape(handoff.department)}</h2>
          <p>{escape(handoff.reason)}</p>
          <dl>
            <div><dt>Tools</dt><dd>{escape(', '.join(handoff.required_tools) if handoff.required_tools else 'none')}</dd></div>
            <div><dt>Approvals</dt><dd>{escape(', '.join(handoff.approvals) if handoff.approvals else 'none')}</dd></div>
          </dl>
        </section>
        """
        for index, handoff in enumerate(plan.handoffs, start=1)
    ) or "<section class=\"card handoff\"><h2>No handoffs</h2><p>This task stays within a single delivery lane.</p></section>"

    acceptance_items = "".join(f"<li>{escape(item)}</li>" for item in task.acceptance_criteria) or "<li>none</li>"
    validation_items = "".join(f"<li>{escape(item)}</li>" for item in task.validation_plan) or "<li>none</li>"
    label_items = escape(", ".join(task.labels) if task.labels else "none")
    tool_items = escape(", ".join(task.required_tools) if task.required_tools else "none")
    approval_items = escape(", ".join(plan.required_approvals) if plan.required_approvals else "none")
    tier = policy_decision.tier if policy_decision is not None else "standard"
    upgrade_required = policy_decision.upgrade_required if policy_decision is not None else False
    blocked_departments = (
        escape(", ".join(policy_decision.blocked_departments))
        if policy_decision is not None and policy_decision.blocked_departments
        else "none"
    )
    handoff_team = handoff_request.target_team if handoff_request is not None else "none"
    handoff_reason = handoff_request.reason if handoff_request is not None else "No manual handoff required"

    return f"""<!doctype html>
<html lang=\"en\">
<head>
  <meta charset=\"utf-8\">
  <title>Orchestration Canvas · {escape(task.task_id)}</title>
  <style>
    :root {{ color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }}
    body {{ margin: 2rem auto; max-width: 1100px; padding: 0 1rem 3rem; line-height: 1.5; }}
    .grid {{ display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); }}
    .canvas {{ display: grid; gap: 1rem; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); margin-top: 1rem; }}
    .card {{ border: 1px solid #cbd5e1; border-radius: 14px; padding: 1rem; background: rgba(148, 163, 184, 0.08); }}
    .handoff {{ position: relative; }}
    .handoff::after {{ content: "→"; position: absolute; right: -0.7rem; top: 50%; transform: translateY(-50%); font-size: 1.4rem; opacity: 0.45; }}
    .handoff:last-child::after {{ content: ""; }}
    .eyebrow {{ text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.75rem; opacity: 0.7; }}
    dl {{ margin: 0; }}
    dt {{ font-weight: 600; margin-top: 0.75rem; }}
    dd {{ margin: 0.15rem 0 0; }}
    ul {{ padding-left: 1.2rem; }}
  </style>
</head>
<body>
  <h1>Orchestration Canvas</h1>
  <p><strong>{escape(task.task_id)}</strong> · {escape(task.title)}</p>
  <div class=\"grid\">
    <section class=\"card\"><div class=\"eyebrow\">Mode</div><strong>{escape(plan.collaboration_mode)}</strong></section>
    <section class=\"card\"><div class=\"eyebrow\">Departments</div><strong>{escape(str(plan.department_count))}</strong><br>{escape(', '.join(plan.departments) if plan.departments else 'none')}</section>
    <section class=\"card\"><div class=\"eyebrow\">Required approvals</div><strong>{approval_items}</strong></section>
    <section class=\"card\"><div class=\"eyebrow\">Task profile</div>Labels: {label_items}<br>Tools: {tool_items}</section>
    <section class=\"card\"><div class=\"eyebrow\">Tier</div><strong>{escape(tier)}</strong><br>Upgrade Required: {escape(str(upgrade_required))}</section>
    <section class=\"card\"><div class=\"eyebrow\">Blocked departments</div>{blocked_departments}</section>
    <section class=\"card\"><div class=\"eyebrow\">Human handoff</div><strong>{escape(handoff_team)}</strong><br>{escape(handoff_reason)}</section>
  </div>
  <h2>Delivery lanes</h2>
  <div class=\"canvas\">{handoff_cards}</div>
  <div class=\"grid\" style=\"margin-top: 1rem;\">
    <section class=\"card\">
      <h2>Acceptance</h2>
      <ul>{acceptance_items}</ul>
    </section>
    <section class=\"card\">
      <h2>Validation</h2>
      <ul>{validation_items}</ul>
    </section>
  </div>
</body>
</html>
"""
