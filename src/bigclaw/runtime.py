"""Legacy Python runtime surface frozen after Go mainline cutover."""

import warnings
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional
from typing import Iterable, Sequence

from .models import Task
from .observability import TaskRun


LEGACY_RUNTIME_GUIDANCE = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "the legacy Python runtime surface remains migration-only."
)


def legacy_runtime_message(surface: str, replacement: str) -> str:
    return f"{surface} is frozen for migration-only use. {LEGACY_RUNTIME_GUIDANCE} Use {replacement} instead."


def warn_legacy_runtime_surface(surface: str, replacement: str) -> str:
    message = legacy_runtime_message(surface, replacement)
    warnings.warn(message, DeprecationWarning, stacklevel=2)
    return message


LEGACY_PYTHON_WRAPPER_NOTICE = (
    "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. "
    "This Python path remains only as a compatibility shim during migration."
)


def append_missing_flag(args: Sequence[str], flag: str, value: str) -> List[str]:
    flag_prefix = flag + "="
    if any(arg == flag or arg.startswith(flag_prefix) for arg in args):
        return list(args)
    return [*args, flag, value]


def build_bigclawctl_exec_args(repo_root: Path, command: Iterable[str], forwarded: Sequence[str]) -> List[str]:
    return ["bash", str(repo_root / "scripts/ops/bigclawctl"), *command, *forwarded]


def build_workspace_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    args = list(forwarded)
    args = append_missing_flag(args, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
    args = append_missing_flag(args, "--cache-key", "openagis-bigclaw")
    return build_bigclawctl_exec_args(repo_root, ["workspace", "bootstrap"], args)


def translate_workspace_validate_args(forwarded: Sequence[str]) -> List[str]:
    translated: List[str] = []
    i = 0
    while i < len(forwarded):
        arg = forwarded[i]
        if arg == "--report-file":
            translated.extend(["--report", forwarded[i + 1]])
            i += 2
            continue
        if arg.startswith("--report-file="):
            translated.append("--report=" + arg.split("=", 1)[1])
            i += 1
            continue
        if arg == "--no-cleanup":
            translated.append("--cleanup=false")
            i += 1
            continue
        if arg == "--issues":
            issues: List[str] = []
            i += 1
            while i < len(forwarded) and not forwarded[i].startswith("-"):
                issues.append(forwarded[i])
                i += 1
            translated.extend(["--issues", ",".join(issues)])
            continue
        translated.append(arg)
        i += 1
    return translated


def build_workspace_validate_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace", "validate"], translate_workspace_validate_args(forwarded))


def build_github_sync_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["github-sync"], list(forwarded))


def build_refill_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["refill"], list(forwarded))


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
