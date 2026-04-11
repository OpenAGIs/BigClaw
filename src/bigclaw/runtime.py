"""Legacy Python runtime surface frozen after Go mainline cutover."""

from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional

from .deprecation import LEGACY_RUNTIME_GUIDANCE
from .models import Task
from .observability import TaskRun


LEGACY_MAINLINE_STATUS = LEGACY_RUNTIME_GUIDANCE
GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/worker/runtime.go"


@dataclass
class SandboxProfile:
    medium: str
    isolation: str
    network_access: str
    filesystem_access: str


class SandboxRouter:
    def __init__(self) -> None:
        self._profiles = {
            "docker": SandboxProfile(
                medium="docker",
                isolation="container",
                network_access="restricted",
                filesystem_access="workspace-write",
            ),
            "browser": SandboxProfile(
                medium="browser",
                isolation="browser-automation",
                network_access="enabled",
                filesystem_access="downloads-only",
            ),
            "vm": SandboxProfile(
                medium="vm",
                isolation="virtual-machine",
                network_access="restricted",
                filesystem_access="workspace-write",
            ),
            "none": SandboxProfile(
                medium="none",
                isolation="disabled",
                network_access="none",
                filesystem_access="none",
            ),
        }

    def profile_for(self, medium: str) -> SandboxProfile:
        return self._profiles.get(medium, self._profiles["none"])


@dataclass
class ToolPolicy:
    allowed_tools: List[str] = field(default_factory=list)
    blocked_tools: List[str] = field(default_factory=list)

    def allows(self, tool_name: str) -> bool:
        if tool_name in self.blocked_tools:
            return False
        if self.allowed_tools:
            return tool_name in self.allowed_tools
        return True


@dataclass
class ToolCallResult:
    tool_name: str
    action: str
    success: bool
    output: str = ""
    error: str = ""


@dataclass
class BudgetDecision:
    status: str
    estimated_cost: float
    remaining_budget: float
    reason: str


class CostController:
    def __init__(self, medium_hourly_costs: Optional[Dict[str, float]] = None):
        self.medium_hourly_costs = medium_hourly_costs or {
            "docker": 2.0,
            "browser": 4.0,
            "vm": 8.0,
            "none": 0.0,
        }

    def estimate_cost(self, medium: str, duration_minutes: int) -> float:
        hourly = self.medium_hourly_costs.get(medium, 0.0)
        return round(hourly * (max(0, duration_minutes) / 60.0), 2)

    def evaluate(self, task: Task, medium: str, duration_minutes: int, spent_so_far: float = 0.0) -> BudgetDecision:
        estimated = self.estimate_cost(medium, duration_minutes)
        effective_budget = float(task.budget + task.budget_override_amount)
        remaining = round(effective_budget - spent_so_far - estimated, 2)

        if effective_budget <= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="budget not set",
            )

        if remaining >= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="within budget",
            )

        downgraded_medium = "docker" if medium in {"browser", "vm"} else "none"
        if downgraded_medium != medium:
            downgraded_estimated = self.estimate_cost(downgraded_medium, duration_minutes)
            downgraded_remaining = round(effective_budget - spent_so_far - downgraded_estimated, 2)
            if downgraded_remaining >= 0:
                return BudgetDecision(
                    status="degrade",
                    estimated_cost=downgraded_estimated,
                    remaining_budget=downgraded_remaining,
                    reason=f"degrade to {downgraded_medium} to stay within budget",
                )

        return BudgetDecision(
            status="pause",
            estimated_cost=estimated,
            remaining_budget=remaining,
            reason="budget exceeded",
        )


class ToolRuntime:
    def __init__(
        self,
        policy: Optional[ToolPolicy] = None,
        handlers: Optional[Dict[str, Callable[[str, Dict[str, Any]], str]]] = None,
    ) -> None:
        self.policy = policy or ToolPolicy()
        self.handlers = handlers or {}

    def register_handler(self, tool_name: str, handler: Callable[[str, Dict[str, Any]], str]) -> None:
        self.handlers[tool_name] = handler

    def invoke(
        self,
        tool_name: str,
        action: str,
        payload: Optional[Dict[str, Any]] = None,
        run: Optional[TaskRun] = None,
        actor: str = "tool-runtime",
    ) -> ToolCallResult:
        resolved_payload = payload or {}
        if not self.policy.allows(tool_name):
            if run is not None:
                run.log("error", "tool blocked", tool=tool_name, operation=action)
                run.audit("tool.invoke", actor, "blocked", tool=tool_name, operation=action)
                run.trace("tool.invoke", "blocked", tool=tool_name, operation=action)
            return ToolCallResult(
                tool_name=tool_name,
                action=action,
                success=False,
                error="tool blocked by policy",
            )

        handler = self.handlers.get(tool_name, self._default_handler)
        output = handler(action, resolved_payload)
        if run is not None:
            run.log("info", "tool executed", tool=tool_name, operation=action)
            run.audit("tool.invoke", actor, "success", tool=tool_name, operation=action)
            run.trace("tool.invoke", "ok", tool=tool_name, operation=action)
        return ToolCallResult(
            tool_name=tool_name,
            action=action,
            success=True,
            output=output,
        )

    def _default_handler(self, action: str, payload: Dict[str, Any]) -> str:
        if payload:
            return f"{action}:{sorted(payload.items())}"
        return action


@dataclass
class WorkerExecutionResult:
    run: TaskRun
    tool_results: List[ToolCallResult]
    sandbox_profile: SandboxProfile


class ClawWorkerRuntime:
    def __init__(
        self,
        tool_runtime: Optional[ToolRuntime] = None,
        sandbox_router: Optional[SandboxRouter] = None,
    ) -> None:
        self.tool_runtime = tool_runtime or ToolRuntime()
        self.sandbox_router = sandbox_router or SandboxRouter()

    def execute(
        self,
        task: Task,
        decision: Any,
        run: TaskRun,
        actor: str = "worker-runtime",
        tool_payloads: Optional[Dict[str, Dict[str, Any]]] = None,
    ) -> WorkerExecutionResult:
        profile = self.sandbox_router.profile_for(decision.medium)
        run.log(
            "info",
            "worker assigned sandbox",
            medium=profile.medium,
            isolation=profile.isolation,
            network_access=profile.network_access,
        )
        run.trace(
            "worker.sandbox",
            "ready",
            medium=profile.medium,
            isolation=profile.isolation,
            filesystem_access=profile.filesystem_access,
        )

        if not decision.approved:
            if profile.medium == "none":
                run.log("warn", "worker paused by scheduler budget policy", reason=decision.reason)
                run.audit(
                    "worker.lifecycle",
                    actor,
                    "paused",
                    medium=decision.medium,
                    required_tools=task.required_tools,
                    reason=decision.reason,
                )
                run.trace("worker.lifecycle", "blocked", medium=decision.medium, reason=decision.reason)
                return WorkerExecutionResult(run=run, tool_results=[], sandbox_profile=profile)
            run.log("warn", "worker waiting for approval", medium=decision.medium)
            run.audit(
                "worker.lifecycle",
                actor,
                "waiting-approval",
                medium=decision.medium,
                required_tools=task.required_tools,
            )
            run.trace("worker.lifecycle", "pending", medium=decision.medium)
            return WorkerExecutionResult(run=run, tool_results=[], sandbox_profile=profile)

        results: List[ToolCallResult] = []
        resolved_payloads = tool_payloads or {}
        run.log("info", "worker started", medium=decision.medium, tool_count=len(task.required_tools))
        run.trace("worker.lifecycle", "started", medium=decision.medium, tool_count=len(task.required_tools))
        for tool_name in task.required_tools:
            result = self.tool_runtime.invoke(
                tool_name,
                action="execute",
                payload=resolved_payloads.get(tool_name),
                run=run,
                actor=actor,
            )
            results.append(result)

        run.audit(
            "worker.lifecycle",
            actor,
            "completed",
            medium=decision.medium,
            successful_tools=[result.tool_name for result in results if result.success],
            blocked_tools=[result.tool_name for result in results if not result.success],
        )
        run.trace(
            "worker.lifecycle",
            "completed",
            medium=decision.medium,
            successful_tools=len([result for result in results if result.success]),
            blocked_tools=len([result for result in results if not result.success]),
        )
        return WorkerExecutionResult(run=run, tool_results=results, sandbox_profile=profile)
