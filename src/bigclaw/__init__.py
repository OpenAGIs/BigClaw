from __future__ import annotations

import argparse
import hashlib
import importlib
import json
import shutil
import stat
import subprocess
import sys
import types
import warnings
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from enum import Enum
from html import escape
from pathlib import Path
from typing import Any, Callable, Dict, Iterable, List, Optional, Protocol, Sequence
from urllib.parse import urlparse

def _install_support_module(name: str, **attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__.update(attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


def _install_exec_support_module(name: str, source: str, export_names: List[str]) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__["__file__"] = f"<generated {module.__name__}>"
    module.__dict__["__package__"] = __name__
    sys.modules[module.__name__] = module
    exec(compile(source, module.__dict__["__file__"], "exec"), module.__dict__)
    globals()[name] = module
    for export_name in export_names:
        globals()[export_name] = module.__dict__[export_name]


def _install_exec_support_module_from_asset(name: str, asset_name: str, export_names: List[str]) -> None:
    source = Path(__file__).with_name(asset_name).read_text(encoding="utf-8")
    _install_exec_support_module(name, source, export_names)


class TaskState(str, Enum):
    TODO = "Todo"
    IN_PROGRESS = "In Progress"
    BLOCKED = "Blocked"
    DONE = "Done"
    FAILED = "Failed"


class RiskLevel(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"


class Priority(int, Enum):
    P0 = 0
    P1 = 1
    P2 = 2


class TriageStatus(str, Enum):
    OPEN = "open"
    IN_PROGRESS = "in-progress"
    ESCALATED = "escalated"
    RESOLVED = "resolved"


class FlowTrigger(str, Enum):
    MANUAL = "manual"
    SCHEDULED = "scheduled"
    EVENT = "event"


class FlowRunStatus(str, Enum):
    QUEUED = "queued"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"
    CANCELED = "canceled"


class FlowStepStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCEEDED = "succeeded"
    FAILED = "failed"
    SKIPPED = "skipped"


class BillingInterval(str, Enum):
    MONTHLY = "monthly"
    ANNUAL = "annual"
    USAGE = "usage"


@dataclass
class Task:
    task_id: str
    source: str
    title: str
    description: str
    labels: List[str] = field(default_factory=list)
    priority: Priority = Priority.P2
    state: TaskState = TaskState.TODO
    risk_level: RiskLevel = RiskLevel.LOW
    budget: float = 0.0
    budget_override_actor: str = ""
    budget_override_reason: str = ""
    budget_override_amount: float = 0.0
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict:
        return {
            "task_id": self.task_id,
            "source": self.source,
            "title": self.title,
            "description": self.description,
            "labels": self.labels,
            "priority": int(self.priority),
            "state": self.state.value,
            "risk_level": self.risk_level.value,
            "budget": self.budget,
            "budget_override_actor": self.budget_override_actor,
            "budget_override_reason": self.budget_override_reason,
            "budget_override_amount": self.budget_override_amount,
            "required_tools": self.required_tools,
            "acceptance_criteria": self.acceptance_criteria,
            "validation_plan": self.validation_plan,
        }

    @classmethod
    def from_dict(cls, data: Dict) -> "Task":
        return cls(
            task_id=data["task_id"],
            source=data["source"],
            title=data["title"],
            description=data.get("description", ""),
            labels=data.get("labels", []),
            priority=Priority(data.get("priority", Priority.P2)),
            state=TaskState(data.get("state", TaskState.TODO.value)),
            risk_level=RiskLevel(data.get("risk_level", RiskLevel.LOW.value)),
            budget=data.get("budget", 0.0),
            budget_override_actor=str(data.get("budget_override_actor", "")),
            budget_override_reason=str(data.get("budget_override_reason", "")),
            budget_override_amount=float(data.get("budget_override_amount", 0.0)),
            required_tools=data.get("required_tools", []),
            acceptance_criteria=data.get("acceptance_criteria", []),
            validation_plan=data.get("validation_plan", []),
        )


@dataclass(frozen=True)
class RiskSignal:
    name: str
    score: int
    reason: str
    source: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "score": self.score,
            "reason": self.reason,
            "source": self.source,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RiskSignal":
        return cls(
            name=str(data["name"]),
            score=int(data.get("score", 0)),
            reason=str(data.get("reason", "")),
            source=str(data.get("source", "")),
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class RiskAssessment:
    assessment_id: str
    task_id: str
    level: RiskLevel
    total_score: int
    requires_approval: bool = False
    signals: List[RiskSignal] = field(default_factory=list)
    mitigations: List[str] = field(default_factory=list)
    reviewer: str = ""
    notes: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "assessment_id": self.assessment_id,
            "task_id": self.task_id,
            "level": self.level.value,
            "total_score": self.total_score,
            "requires_approval": self.requires_approval,
            "signals": [signal.to_dict() for signal in self.signals],
            "mitigations": list(self.mitigations),
            "reviewer": self.reviewer,
            "notes": self.notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RiskAssessment":
        return cls(
            assessment_id=str(data["assessment_id"]),
            task_id=str(data["task_id"]),
            level=RiskLevel(data.get("level", RiskLevel.LOW.value)),
            total_score=int(data.get("total_score", 0)),
            requires_approval=bool(data.get("requires_approval", False)),
            signals=[RiskSignal.from_dict(item) for item in data.get("signals", [])],
            mitigations=[str(item) for item in data.get("mitigations", [])],
            reviewer=str(data.get("reviewer", "")),
            notes=str(data.get("notes", "")),
        )


@dataclass(frozen=True)
class TriageLabel:
    name: str
    confidence: float = 1.0
    source: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {"name": self.name, "confidence": self.confidence, "source": self.source}

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TriageLabel":
        return cls(
            name=str(data["name"]),
            confidence=float(data.get("confidence", 1.0)),
            source=str(data.get("source", "")),
        )


@dataclass
class TriageRecord:
    triage_id: str
    task_id: str
    status: TriageStatus = TriageStatus.OPEN
    queue: str = "default"
    owner: str = ""
    summary: str = ""
    labels: List[TriageLabel] = field(default_factory=list)
    related_run_id: str = ""
    escalation_target: str = ""
    actions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "triage_id": self.triage_id,
            "task_id": self.task_id,
            "status": self.status.value,
            "queue": self.queue,
            "owner": self.owner,
            "summary": self.summary,
            "labels": [label.to_dict() for label in self.labels],
            "related_run_id": self.related_run_id,
            "escalation_target": self.escalation_target,
            "actions": list(self.actions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TriageRecord":
        return cls(
            triage_id=str(data["triage_id"]),
            task_id=str(data["task_id"]),
            status=TriageStatus(data.get("status", TriageStatus.OPEN.value)),
            queue=str(data.get("queue", "default")),
            owner=str(data.get("owner", "")),
            summary=str(data.get("summary", "")),
            labels=[TriageLabel.from_dict(item) for item in data.get("labels", [])],
            related_run_id=str(data.get("related_run_id", "")),
            escalation_target=str(data.get("escalation_target", "")),
            actions=[str(item) for item in data.get("actions", [])],
        )


@dataclass(frozen=True)
class FlowTemplateStep:
    step_id: str
    name: str
    kind: str
    required_tools: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "step_id": self.step_id,
            "name": self.name,
            "kind": self.kind,
            "required_tools": list(self.required_tools),
            "approvals": list(self.approvals),
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FlowTemplateStep":
        return cls(
            step_id=str(data["step_id"]),
            name=str(data["name"]),
            kind=str(data.get("kind", "")),
            required_tools=[str(item) for item in data.get("required_tools", [])],
            approvals=[str(item) for item in data.get("approvals", [])],
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class FlowTemplate:
    template_id: str
    name: str
    version: str
    description: str = ""
    trigger: FlowTrigger = FlowTrigger.MANUAL
    default_risk: RiskLevel = RiskLevel.LOW
    steps: List[FlowTemplateStep] = field(default_factory=list)
    tags: List[str] = field(default_factory=list)
    active: bool = True

    def to_dict(self) -> Dict[str, Any]:
        return {
            "template_id": self.template_id,
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "trigger": self.trigger.value,
            "default_risk": self.default_risk.value,
            "steps": [step.to_dict() for step in self.steps],
            "tags": list(self.tags),
            "active": self.active,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FlowTemplate":
        return cls(
            template_id=str(data["template_id"]),
            name=str(data["name"]),
            version=str(data["version"]),
            description=str(data.get("description", "")),
            trigger=FlowTrigger(data.get("trigger", FlowTrigger.MANUAL.value)),
            default_risk=RiskLevel(data.get("default_risk", RiskLevel.LOW.value)),
            steps=[FlowTemplateStep.from_dict(item) for item in data.get("steps", [])],
            tags=[str(item) for item in data.get("tags", [])],
            active=bool(data.get("active", True)),
        )


@dataclass(frozen=True)
class FlowStepRun:
    step_id: str
    status: FlowStepStatus = FlowStepStatus.PENDING
    actor: str = ""
    started_at: str = ""
    completed_at: str = ""
    output: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "step_id": self.step_id,
            "status": self.status.value,
            "actor": self.actor,
            "started_at": self.started_at,
            "completed_at": self.completed_at,
            "output": dict(self.output),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FlowStepRun":
        return cls(
            step_id=str(data["step_id"]),
            status=FlowStepStatus(data.get("status", FlowStepStatus.PENDING.value)),
            actor=str(data.get("actor", "")),
            started_at=str(data.get("started_at", "")),
            completed_at=str(data.get("completed_at", "")),
            output=dict(data.get("output", {})),
        )


@dataclass
class FlowRun:
    run_id: str
    template_id: str
    task_id: str
    status: FlowRunStatus = FlowRunStatus.QUEUED
    triggered_by: str = ""
    started_at: str = ""
    completed_at: str = ""
    steps: List[FlowStepRun] = field(default_factory=list)
    outputs: Dict[str, Any] = field(default_factory=dict)
    approval_refs: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "run_id": self.run_id,
            "template_id": self.template_id,
            "task_id": self.task_id,
            "status": self.status.value,
            "triggered_by": self.triggered_by,
            "started_at": self.started_at,
            "completed_at": self.completed_at,
            "steps": [step.to_dict() for step in self.steps],
            "outputs": dict(self.outputs),
            "approval_refs": list(self.approval_refs),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FlowRun":
        return cls(
            run_id=str(data["run_id"]),
            template_id=str(data["template_id"]),
            task_id=str(data["task_id"]),
            status=FlowRunStatus(data.get("status", FlowRunStatus.QUEUED.value)),
            triggered_by=str(data.get("triggered_by", "")),
            started_at=str(data.get("started_at", "")),
            completed_at=str(data.get("completed_at", "")),
            steps=[FlowStepRun.from_dict(item) for item in data.get("steps", [])],
            outputs=dict(data.get("outputs", {})),
            approval_refs=[str(item) for item in data.get("approval_refs", [])],
        )


@dataclass(frozen=True)
class BillingRate:
    metric: str
    interval: BillingInterval = BillingInterval.USAGE
    included_units: int = 0
    unit_price_usd: float = 0.0
    overage_price_usd: float = 0.0

    def to_dict(self) -> Dict[str, Any]:
        return {
            "metric": self.metric,
            "interval": self.interval.value,
            "included_units": self.included_units,
            "unit_price_usd": self.unit_price_usd,
            "overage_price_usd": self.overage_price_usd,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "BillingRate":
        return cls(
            metric=str(data["metric"]),
            interval=BillingInterval(data.get("interval", BillingInterval.USAGE.value)),
            included_units=int(data.get("included_units", 0)),
            unit_price_usd=float(data.get("unit_price_usd", 0.0)),
            overage_price_usd=float(data.get("overage_price_usd", 0.0)),
        )


@dataclass(frozen=True)
class UsageRecord:
    record_id: str
    account_id: str
    metric: str
    quantity: float
    period: str
    run_id: str = ""
    unit: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "record_id": self.record_id,
            "account_id": self.account_id,
            "metric": self.metric,
            "quantity": self.quantity,
            "period": self.period,
            "run_id": self.run_id,
            "unit": self.unit,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "UsageRecord":
        return cls(
            record_id=str(data["record_id"]),
            account_id=str(data["account_id"]),
            metric=str(data["metric"]),
            quantity=float(data.get("quantity", 0.0)),
            period=str(data.get("period", "")),
            run_id=str(data.get("run_id", "")),
            unit=str(data.get("unit", "")),
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class BillingSummary:
    statement_id: str
    account_id: str
    billing_period: str
    currency: str = "USD"
    rates: List[BillingRate] = field(default_factory=list)
    usage: List[UsageRecord] = field(default_factory=list)
    subtotal_usd: float = 0.0
    overage_usd: float = 0.0
    total_usd: float = 0.0

    def to_dict(self) -> Dict[str, Any]:
        return {
            "statement_id": self.statement_id,
            "account_id": self.account_id,
            "billing_period": self.billing_period,
            "currency": self.currency,
            "rates": [rate.to_dict() for rate in self.rates],
            "usage": [record.to_dict() for record in self.usage],
            "subtotal_usd": self.subtotal_usd,
            "overage_usd": self.overage_usd,
            "total_usd": self.total_usd,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "BillingSummary":
        return cls(
            statement_id=str(data["statement_id"]),
            account_id=str(data["account_id"]),
            billing_period=str(data["billing_period"]),
            currency=str(data.get("currency", "USD")),
            rates=[BillingRate.from_dict(item) for item in data.get("rates", [])],
            usage=[UsageRecord.from_dict(item) for item in data.get("usage", [])],
            subtotal_usd=float(data.get("subtotal_usd", 0.0)),
            overage_usd=float(data.get("overage_usd", 0.0)),
            total_usd=float(data.get("total_usd", 0.0)),
        )


_install_support_module(
    "models",
    TaskState=TaskState,
    RiskLevel=RiskLevel,
    Priority=Priority,
    TriageStatus=TriageStatus,
    FlowTrigger=FlowTrigger,
    FlowRunStatus=FlowRunStatus,
    FlowStepStatus=FlowStepStatus,
    BillingInterval=BillingInterval,
    Task=Task,
    RiskSignal=RiskSignal,
    RiskAssessment=RiskAssessment,
    TriageLabel=TriageLabel,
    TriageRecord=TriageRecord,
    FlowTemplateStep=FlowTemplateStep,
    FlowTemplate=FlowTemplate,
    FlowStepRun=FlowStepRun,
    FlowRun=FlowRun,
    BillingRate=BillingRate,
    UsageRecord=UsageRecord,
    BillingSummary=BillingSummary,
)


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


_install_support_module(
    "deprecation",
    LEGACY_RUNTIME_GUIDANCE=LEGACY_RUNTIME_GUIDANCE,
    legacy_runtime_message=legacy_runtime_message,
    warn_legacy_runtime_surface=warn_legacy_runtime_surface,
)


@dataclass
class SourceIssue:
    source: str
    source_id: str
    title: str
    description: str
    labels: List[str]
    priority: str
    state: str
    links: Dict[str, str]


class Connector(Protocol):
    name: str

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        ...


class GitHubConnector:
    name = "github"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="github",
                source_id=f"{project}#1",
                title="Fix flaky test",
                description="CI flaky on macOS",
                labels=["bug", "ci"],
                priority="P1",
                state=states[0] if states else "Todo",
                links={"issue": f"https://github.com/{project}/issues/1"},
            )
        ]


class LinearConnector:
    name = "linear"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="linear",
                source_id=f"{project}-101",
                title="Implement queue persistence",
                description="Need restart-safe queue",
                labels=["platform"],
                priority="P0",
                state=states[0] if states else "Todo",
                links={"issue": f"https://linear.app/{project}/issue/{project}-101"},
            )
        ]


class JiraConnector:
    name = "jira"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="jira",
                source_id=f"{project}-23",
                title="Runbook automation",
                description="Automate oncall runbook",
                labels=["ops"],
                priority="P2",
                state=states[0] if states else "Todo",
                links={"issue": f"https://jira.example.com/browse/{project}-23"},
            )
        ]


_install_support_module(
    "connectors",
    SourceIssue=SourceIssue,
    Connector=Connector,
    GitHubConnector=GitHubConnector,
    LinearConnector=LinearConnector,
    JiraConnector=JiraConnector,
)


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
            return BudgetDecision("allow", estimated, remaining, "budget not set")
        if remaining >= 0:
            return BudgetDecision("allow", estimated, remaining, "within budget")

        downgraded_medium = "docker" if medium in {"browser", "vm"} else "none"
        if downgraded_medium != medium:
            downgraded_estimated = self.estimate_cost(downgraded_medium, duration_minutes)
            downgraded_remaining = round(effective_budget - spent_so_far - downgraded_estimated, 2)
            if downgraded_remaining >= 0:
                return BudgetDecision(
                    "degrade",
                    downgraded_estimated,
                    downgraded_remaining,
                    f"degrade to {downgraded_medium} to stay within budget",
                )

        return BudgetDecision("pause", estimated, remaining, "budget exceeded")


_install_support_module(
    "cost_control",
    BudgetDecision=BudgetDecision,
    CostController=CostController,
)


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


def repo_root_from_script(script_path: str) -> Path:
    return Path(script_path).resolve().parents[2]


def run_bigclawctl_shim(script_path: str, command: Iterable[str], forwarded: Sequence[str]) -> int:
    repo_root = repo_root_from_script(script_path)
    argv = build_bigclawctl_exec_args(repo_root, command, forwarded)
    return subprocess.call(argv, cwd=repo_root)


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


def build_workspace_runtime_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace"], list(forwarded))


_install_support_module(
    "legacy_shim",
    LEGACY_PYTHON_WRAPPER_NOTICE=LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag=append_missing_flag,
    build_bigclawctl_exec_args=build_bigclawctl_exec_args,
    repo_root_from_script=repo_root_from_script,
    run_bigclawctl_shim=run_bigclawctl_shim,
    build_workspace_bootstrap_args=build_workspace_bootstrap_args,
    translate_workspace_validate_args=translate_workspace_validate_args,
    build_workspace_validate_args=build_workspace_validate_args,
    build_github_sync_args=build_github_sync_args,
    build_refill_args=build_refill_args,
    build_workspace_runtime_bootstrap_args=build_workspace_runtime_bootstrap_args,
)


class WorkspaceBootstrapError(RuntimeError):
    """Raised when the shared-worktree bootstrap flow cannot complete."""


@dataclass
class CacheBootstrapState:
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    mirror_created: bool
    seed_created: bool

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


@dataclass
class WorkspaceBootstrapStatus:
    workspace: str
    branch: str
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    reused: bool
    cache_reused: bool
    clone_suppressed: bool
    mirror_created: bool = False
    seed_created: bool = False
    workspace_mode: str = "worktree_created"
    removed: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


@dataclass
class WorkspaceBootstrapCommandResult:
    stdout: str
    stderr: str
    returncode: int


WORKSPACE_CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")


def _workspace_bootstrap_run(command: Sequence[str], cwd: Path) -> WorkspaceBootstrapCommandResult:
    completed = subprocess.run(
        list(command),
        cwd=cwd,
        text=True,
        capture_output=True,
        check=False,
    )
    return WorkspaceBootstrapCommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _workspace_bootstrap_git(repo: Path, *args: str) -> WorkspaceBootstrapCommandResult:
    return _workspace_bootstrap_run(["git", *args], repo)


def _workspace_bootstrap_require_git(repo: Path, *args: str) -> str:
    result = _workspace_bootstrap_git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise WorkspaceBootstrapError(detail)
    return result.stdout


def sanitize_issue_identifier(identifier: str | None) -> str:
    raw = (identifier or "issue").strip() or "issue"
    return "".join(character if character.isalnum() or character in ".-_" else "_" for character in raw)


def bootstrap_branch_name(identifier: str | None) -> str:
    return f"{BOOTSTRAP_BRANCH_PREFIX}/{sanitize_issue_identifier(identifier)}"


def default_cache_base(path: str | Path | None = None) -> Path:
    if path is None:
        return DEFAULT_CACHE_BASE.expanduser().resolve()
    return Path(path).expanduser().resolve()


def normalize_repo_locator(repo_url: str) -> str:
    raw = repo_url.strip()

    if "://" in raw:
        parsed = urlparse(raw)
        locator = f"{parsed.netloc}{parsed.path}"
    elif ":" in raw and "@" in raw.split(":", 1)[0]:
        user_host, repo_path = raw.split(":", 1)
        host = user_host.split("@", 1)[-1]
        locator = f"{host}/{repo_path}"
    else:
        locator = raw

    return locator.strip().rstrip("/").removesuffix(".git")


def repo_cache_key(repo_url: str, cache_key: str | None = None) -> str:
    raw = (cache_key or normalize_repo_locator(repo_url)).strip().lower()
    sanitized = "".join(character if character.isalnum() or character in ".-_" else "-" for character in raw)
    compact = "-".join(segment for segment in sanitized.split("-") if segment)
    return compact or "repo"


def cache_root_for_repo(
    repo_url: str,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    return default_cache_base(cache_base) / repo_cache_key(repo_url, cache_key)


def resolve_cache_root(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> Path:
    if cache_root is not None:
        return Path(cache_root).expanduser().resolve()
    return cache_root_for_repo(repo_url, cache_base=cache_base, cache_key=cache_key)


def default_cache_root(path: str | Path | None = None) -> Path:
    return default_cache_base(path)


def _remove_path(path: Path) -> None:
    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    elif path.exists() or path.is_symlink():
        path.unlink()


def _cache_state(
    repo_url: str,
    repo_cache_root: Path,
    cache_key: str | None = None,
    *,
    mirror_created: bool = False,
    seed_created: bool = False,
) -> CacheBootstrapState:
    return CacheBootstrapState(
        cache_root=str(repo_cache_root),
        cache_key=repo_cache_key(repo_url, cache_key),
        mirror_path=str(repo_cache_root / "mirror.git"),
        seed_path=str(repo_cache_root / "seed"),
        mirror_created=mirror_created,
        seed_created=seed_created,
    )


def ensure_mirror(
    repo_url: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    mirror_path = repo_cache_root / "mirror.git"
    mirror_path.parent.mkdir(parents=True, exist_ok=True)
    mirror_created = False

    if not (mirror_path / "HEAD").exists():
        if mirror_path.exists():
            _remove_path(mirror_path)
        result = subprocess.run(
            ["git", "clone", "--mirror", repo_url, str(mirror_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone --mirror failed"
            raise WorkspaceBootstrapError(detail)
        mirror_created = True
    else:
        _workspace_bootstrap_require_git(mirror_path, "remote", "set-url", "origin", repo_url)
        _workspace_bootstrap_require_git(mirror_path, "fetch", "--prune", "origin")

    return _cache_state(repo_url, repo_cache_root, cache_key, mirror_created=mirror_created)


def configure_seed_remotes(seed_path: Path, repo_url: str, mirror_path: Path) -> None:
    remotes = set(_workspace_bootstrap_require_git(seed_path, "remote").splitlines())

    if WORKSPACE_CACHE_REMOTE not in remotes and "origin" in remotes:
        current_origin = _workspace_bootstrap_require_git(seed_path, "remote", "get-url", "origin")
        if Path(current_origin).expanduser().resolve() == mirror_path.resolve():
            _workspace_bootstrap_require_git(seed_path, "remote", "rename", "origin", WORKSPACE_CACHE_REMOTE)
            remotes = set(_workspace_bootstrap_require_git(seed_path, "remote").splitlines())

    if WORKSPACE_CACHE_REMOTE not in remotes:
        _workspace_bootstrap_require_git(seed_path, "remote", "add", WORKSPACE_CACHE_REMOTE, str(mirror_path))
    else:
        _workspace_bootstrap_require_git(seed_path, "remote", "set-url", WORKSPACE_CACHE_REMOTE, str(mirror_path))

    remotes = set(_workspace_bootstrap_require_git(seed_path, "remote").splitlines())
    if "origin" not in remotes:
        _workspace_bootstrap_require_git(seed_path, "remote", "add", "origin", repo_url)
    else:
        _workspace_bootstrap_require_git(seed_path, "remote", "set-url", "origin", repo_url)

    _workspace_bootstrap_require_git(seed_path, "config", "remote.pushDefault", "origin")


def ensure_seed(
    repo_url: str,
    default_branch: str,
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> CacheBootstrapState:
    cache_state = ensure_mirror(
        repo_url,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    seed_path = Path(cache_state.seed_path)
    seed_created = False

    if not (seed_path / ".git").exists():
        if seed_path.exists():
            _remove_path(seed_path)
        result = subprocess.run(
            ["git", "clone", cache_state.mirror_path, str(seed_path)],
            text=True,
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = result.stderr.strip() or result.stdout.strip() or "git clone seed failed"
            raise WorkspaceBootstrapError(detail)
        seed_created = True

    configure_seed_remotes(seed_path, repo_url, Path(cache_state.mirror_path))
    _workspace_bootstrap_require_git(seed_path, "fetch", "--prune", WORKSPACE_CACHE_REMOTE)
    _workspace_bootstrap_require_git(seed_path, "worktree", "prune")
    _workspace_bootstrap_require_git(
        seed_path,
        "checkout",
        "-B",
        default_branch,
        f"{WORKSPACE_CACHE_REMOTE}/{default_branch}",
    )
    return _cache_state(
        repo_url,
        Path(cache_state.cache_root),
        cache_key,
        mirror_created=cache_state.mirror_created,
        seed_created=seed_created,
    )


def bootstrap_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    cache_state = ensure_seed(
        repo_url,
        default_branch,
        cache_root=cache_root,
        cache_base=cache_base,
        cache_key=cache_key,
    )
    seed_path = Path(cache_state.seed_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)
    cache_reused = not cache_state.mirror_created and not cache_state.seed_created
    clone_suppressed = not cache_state.mirror_created

    git_dir = workspace_path / ".git"
    if git_dir.exists():
        current_branch = _workspace_bootstrap_require_git(workspace_path, "branch", "--show-current")
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=current_branch or branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=True,
            cache_reused=cache_reused,
            clone_suppressed=clone_suppressed,
            mirror_created=cache_state.mirror_created,
            seed_created=cache_state.seed_created,
            workspace_mode="workspace_reused",
        )

    parent = workspace_path.parent
    parent.mkdir(parents=True, exist_ok=True)

    if workspace_path.exists() and workspace_path.is_dir() and any(workspace_path.iterdir()):
        raise WorkspaceBootstrapError(f"Workspace is not empty: {workspace_path}")

    _workspace_bootstrap_require_git(
        seed_path,
        "worktree",
        "add",
        "--force",
        "-B",
        branch,
        str(workspace_path),
        f"{WORKSPACE_CACHE_REMOTE}/{default_branch}",
    )

    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=cache_reused,
        clone_suppressed=clone_suppressed,
        mirror_created=cache_state.mirror_created,
        seed_created=cache_state.seed_created,
        workspace_mode="worktree_created",
    )


def cleanup_workspace(
    workspace: str | Path,
    issue_identifier: str | None,
    repo_url: str,
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
) -> WorkspaceBootstrapStatus:
    workspace_path = Path(workspace).expanduser().resolve()
    repo_cache_root = resolve_cache_root(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
    cache_state = _cache_state(repo_url, repo_cache_root, cache_key)
    seed_path = Path(cache_state.seed_path)
    mirror_path = Path(cache_state.mirror_path)
    branch = bootstrap_branch_name(issue_identifier or workspace_path.name)

    if not (seed_path / ".git").exists() or not workspace_path.exists():
        return WorkspaceBootstrapStatus(
            workspace=str(workspace_path),
            branch=branch,
            cache_root=cache_state.cache_root,
            cache_key=cache_state.cache_key,
            mirror_path=cache_state.mirror_path,
            seed_path=cache_state.seed_path,
            reused=False,
            cache_reused=seed_path.exists() or mirror_path.exists(),
            clone_suppressed=True,
            workspace_mode="cleanup",
            removed=False,
        )

    configure_seed_remotes(seed_path, repo_url, mirror_path)

    if _workspace_bootstrap_git(workspace_path, "rev-parse", "--git-dir").returncode == 0:
        current_branch = _workspace_bootstrap_require_git(workspace_path, "branch", "--show-current")
        branch = current_branch or branch

    worktree_list = _workspace_bootstrap_require_git(seed_path, "worktree", "list", "--porcelain")
    registered = f"worktree {workspace_path}" in worktree_list
    if registered:
        _workspace_bootstrap_require_git(seed_path, "worktree", "remove", "--force", str(workspace_path))
        _workspace_bootstrap_require_git(seed_path, "worktree", "prune")

    local_branches = set(
        _workspace_bootstrap_require_git(seed_path, "branch", "--format", "%(refname:short)").splitlines()
    )
    if branch.startswith(f"{BOOTSTRAP_BRANCH_PREFIX}/") and branch in local_branches:
        _workspace_bootstrap_require_git(seed_path, "branch", "-D", branch)

    _workspace_bootstrap_require_git(
        seed_path,
        "checkout",
        "-B",
        default_branch,
        f"{WORKSPACE_CACHE_REMOTE}/{default_branch}",
    )
    return WorkspaceBootstrapStatus(
        workspace=str(workspace_path),
        branch=branch,
        cache_root=cache_state.cache_root,
        cache_key=cache_state.cache_key,
        mirror_path=cache_state.mirror_path,
        seed_path=cache_state.seed_path,
        reused=False,
        cache_reused=True,
        clone_suppressed=True,
        workspace_mode="cleanup",
        removed=registered,
    )


def status_as_json(status: WorkspaceBootstrapStatus) -> str:
    return json.dumps(status.to_dict(), ensure_ascii=False, indent=2)


_install_support_module(
    "workspace_bootstrap",
    WorkspaceBootstrapError=WorkspaceBootstrapError,
    CacheBootstrapState=CacheBootstrapState,
    WorkspaceBootstrapStatus=WorkspaceBootstrapStatus,
    CommandResult=WorkspaceBootstrapCommandResult,
    CACHE_REMOTE=WORKSPACE_CACHE_REMOTE,
    BOOTSTRAP_BRANCH_PREFIX=BOOTSTRAP_BRANCH_PREFIX,
    DEFAULT_CACHE_BASE=DEFAULT_CACHE_BASE,
    sanitize_issue_identifier=sanitize_issue_identifier,
    bootstrap_branch_name=bootstrap_branch_name,
    default_cache_base=default_cache_base,
    normalize_repo_locator=normalize_repo_locator,
    repo_cache_key=repo_cache_key,
    cache_root_for_repo=cache_root_for_repo,
    resolve_cache_root=resolve_cache_root,
    default_cache_root=default_cache_root,
    ensure_mirror=ensure_mirror,
    ensure_seed=ensure_seed,
    configure_seed_remotes=configure_seed_remotes,
    bootstrap_workspace=bootstrap_workspace,
    cleanup_workspace=cleanup_workspace,
    status_as_json=status_as_json,
)


def map_priority(p: str) -> Priority:
    p = (p or "").upper()
    if p == "P0":
        return Priority.P0
    if p == "P1":
        return Priority.P1
    return Priority.P2


def map_state(s: str) -> TaskState:
    normalized = (s or "").lower()
    if "progress" in normalized:
        return TaskState.IN_PROGRESS
    if "done" in normalized or "closed" in normalized:
        return TaskState.DONE
    if "block" in normalized:
        return TaskState.BLOCKED
    return TaskState.TODO


def map_source_issue_to_task(issue: SourceIssue) -> Task:
    risk = RiskLevel.HIGH if "prod" in issue.title.lower() else RiskLevel.LOW
    return Task(
        task_id=issue.source_id,
        source=issue.source,
        title=issue.title,
        description=issue.description,
        labels=issue.labels,
        priority=map_priority(issue.priority),
        state=map_state(issue.state),
        risk_level=risk,
        required_tools=["github" if issue.source == "github" else "connector"],
        acceptance_criteria=["Synced from source issue"],
        validation_plan=["mapping-test"],
    )


_install_support_module(
    "mapping",
    map_priority=map_priority,
    map_state=map_state,
    map_source_issue_to_task=map_source_issue_to_task,
)


ALLOWED_ISSUE_CATEGORIES = {"ui", "ia", "permission", "metric"}
ALLOWED_ISSUE_PRIORITIES = {"P0", "P1", "P2"}


@dataclass(frozen=True)
class ArchivedIssue:
    finding_id: str
    summary: str
    category: str
    priority: str
    owner: str
    surface: str = ""
    impact: str = ""
    status: str = "open"
    evidence: List[str] = field(default_factory=list)

    @property
    def normalized_category(self) -> str:
        return self.category.strip().lower()

    @property
    def normalized_priority(self) -> str:
        return self.priority.strip().upper()

    @property
    def resolved(self) -> bool:
        return self.status.strip().lower() == "resolved"

    def to_dict(self) -> Dict[str, object]:
        return {
            "finding_id": self.finding_id,
            "summary": self.summary,
            "category": self.category,
            "priority": self.priority,
            "owner": self.owner,
            "surface": self.surface,
            "impact": self.impact,
            "status": self.status,
            "evidence": list(self.evidence),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ArchivedIssue":
        return cls(
            finding_id=str(data["finding_id"]),
            summary=str(data["summary"]),
            category=str(data["category"]),
            priority=str(data["priority"]),
            owner=str(data.get("owner", "")),
            surface=str(data.get("surface", "")),
            impact=str(data.get("impact", "")),
            status=str(data.get("status", "open")),
            evidence=[str(item) for item in data.get("evidence", [])],
        )


@dataclass
class IssuePriorityArchive:
    issue_id: str
    title: str
    version: str
    findings: List[ArchivedIssue] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "version": self.version,
            "findings": [finding.to_dict() for finding in self.findings],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "IssuePriorityArchive":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            findings=[ArchivedIssue.from_dict(item) for item in data.get("findings", [])],
        )


@dataclass(frozen=True)
class IssuePriorityArchiveAudit:
    ready: bool
    finding_count: int
    priority_counts: Dict[str, int] = field(default_factory=dict)
    category_counts: Dict[str, int] = field(default_factory=dict)
    missing_owners: List[str] = field(default_factory=list)
    invalid_priorities: List[str] = field(default_factory=list)
    invalid_categories: List[str] = field(default_factory=list)
    unresolved_p0_findings: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: findings={self.finding_count} "
            f"missing_owners={len(self.missing_owners)} "
            f"invalid_priorities={len(self.invalid_priorities)} "
            f"invalid_categories={len(self.invalid_categories)} "
            f"unresolved_p0={len(self.unresolved_p0_findings)}"
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "ready": self.ready,
            "finding_count": self.finding_count,
            "priority_counts": dict(self.priority_counts),
            "category_counts": dict(self.category_counts),
            "missing_owners": list(self.missing_owners),
            "invalid_priorities": list(self.invalid_priorities),
            "invalid_categories": list(self.invalid_categories),
            "unresolved_p0_findings": list(self.unresolved_p0_findings),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "IssuePriorityArchiveAudit":
        return cls(
            ready=bool(data["ready"]),
            finding_count=int(data.get("finding_count", 0)),
            priority_counts={str(priority): int(count) for priority, count in dict(data.get("priority_counts", {})).items()},
            category_counts={str(category): int(count) for category, count in dict(data.get("category_counts", {})).items()},
            missing_owners=[str(item) for item in data.get("missing_owners", [])],
            invalid_priorities=[str(item) for item in data.get("invalid_priorities", [])],
            invalid_categories=[str(item) for item in data.get("invalid_categories", [])],
            unresolved_p0_findings=[str(item) for item in data.get("unresolved_p0_findings", [])],
        )


class IssuePriorityArchivist:
    def audit(self, archive: IssuePriorityArchive) -> IssuePriorityArchiveAudit:
        priority_counts = {priority: 0 for priority in sorted(ALLOWED_ISSUE_PRIORITIES)}
        category_counts = {category: 0 for category in sorted(ALLOWED_ISSUE_CATEGORIES)}
        missing_owners: List[str] = []
        invalid_priorities: List[str] = []
        invalid_categories: List[str] = []
        unresolved_p0_findings: List[str] = []

        for finding in archive.findings:
            if not finding.owner.strip():
                missing_owners.append(finding.finding_id)
            if finding.normalized_priority in ALLOWED_ISSUE_PRIORITIES:
                priority_counts[finding.normalized_priority] += 1
            else:
                invalid_priorities.append(finding.finding_id)
            if finding.normalized_category in ALLOWED_ISSUE_CATEGORIES:
                category_counts[finding.normalized_category] += 1
            else:
                invalid_categories.append(finding.finding_id)
            if finding.normalized_priority == "P0" and not finding.resolved:
                unresolved_p0_findings.append(finding.finding_id)

        ready = bool(archive.findings) and not (
            missing_owners or invalid_priorities or invalid_categories or unresolved_p0_findings
        )
        return IssuePriorityArchiveAudit(
            ready=ready,
            finding_count=len(archive.findings),
            priority_counts=priority_counts,
            category_counts=category_counts,
            missing_owners=sorted(missing_owners),
            invalid_priorities=sorted(invalid_priorities),
            invalid_categories=sorted(invalid_categories),
            unresolved_p0_findings=sorted(unresolved_p0_findings),
        )


def render_issue_priority_archive_report(archive: IssuePriorityArchive, audit: IssuePriorityArchiveAudit) -> str:
    lines = [
        "# Issue Priority Archive",
        "",
        f"- Issue: {archive.issue_id} {archive.title}",
        f"- Version: {archive.version}",
        f"- Audit: {audit.summary}",
        (
            "- Priority Counts: "
            f"P0={audit.priority_counts.get('P0', 0)} "
            f"P1={audit.priority_counts.get('P1', 0)} "
            f"P2={audit.priority_counts.get('P2', 0)}"
        ),
        (
            "- Category Counts: "
            f"ui={audit.category_counts.get('ui', 0)} "
            f"ia={audit.category_counts.get('ia', 0)} "
            f"permission={audit.category_counts.get('permission', 0)} "
            f"metric={audit.category_counts.get('metric', 0)}"
        ),
        "",
        "## Findings",
    ]
    for finding in archive.findings:
        lines.append(
            "- "
            f"{finding.finding_id}: {finding.summary} "
            f"category={finding.normalized_category} priority={finding.normalized_priority} "
            f"owner={finding.owner or 'none'} status={finding.status}"
        )
        lines.append(
            "  "
            f"surface={finding.surface or 'none'} impact={finding.impact or 'none'} "
            f"evidence={','.join(finding.evidence) or 'none'}"
        )
    lines.extend(
        [
            "",
            "## Audit Findings",
            f"- Missing owners: {', '.join(audit.missing_owners) or 'none'}",
            f"- Invalid priorities: {', '.join(audit.invalid_priorities) or 'none'}",
            f"- Invalid categories: {', '.join(audit.invalid_categories) or 'none'}",
            f"- Unresolved P0 findings: {', '.join(audit.unresolved_p0_findings) or 'none'}",
        ]
    )
    return "\n".join(lines)


_install_support_module(
    "issue_archive",
    ALLOWED_ISSUE_CATEGORIES=ALLOWED_ISSUE_CATEGORIES,
    ALLOWED_ISSUE_PRIORITIES=ALLOWED_ISSUE_PRIORITIES,
    ArchivedIssue=ArchivedIssue,
    IssuePriorityArchive=IssuePriorityArchive,
    IssuePriorityArchiveAudit=IssuePriorityArchiveAudit,
    IssuePriorityArchivist=IssuePriorityArchivist,
    render_issue_priority_archive_report=render_issue_priority_archive_report,
)


@dataclass(frozen=True)
class EpicMilestone:
    epic_id: str
    title: str
    phase: str
    owner: str
    milestone: str


@dataclass
class ExecutionPackRoadmap:
    name: str
    epics: List[EpicMilestone] = field(default_factory=list)

    def phase_map(self) -> Dict[str, List[EpicMilestone]]:
        phases: Dict[str, List[EpicMilestone]] = {}
        for epic in self.epics:
            phases.setdefault(epic.phase, []).append(epic)
        return phases

    def epic_map(self) -> Dict[str, EpicMilestone]:
        return {epic.epic_id: epic for epic in self.epics}

    def validate_unique_owners(self) -> None:
        seen: Dict[str, str] = {}
        for epic in self.epics:
            owner = epic.owner.strip().lower()
            if owner in seen:
                raise ValueError(f"Owner '{epic.owner}' is assigned to both {seen[owner]} and {epic.epic_id}")
            seen[owner] = epic.epic_id


def build_execution_pack_roadmap() -> ExecutionPackRoadmap:
    roadmap = ExecutionPackRoadmap(
        name="BigClaw v4.0 Execution Pack",
        epics=[
            EpicMilestone("BIG-EPIC-8", "研发自治执行平台增强", "Phase 1", "engineering-platform", "M1 Foundation uplift"),
            EpicMilestone("BIG-EPIC-9", "工程运营系统", "Phase 2", "engineering-operations", "M2 Operations control plane"),
            EpicMilestone("BIG-EPIC-10", "跨部门 Agent Orchestration", "Phase 3", "orchestration-office", "M3 Cross-team orchestration"),
            EpicMilestone("BIG-EPIC-11", "产品化 UI 与控制台", "Phase 4", "product-experience", "M4 Productized console"),
            EpicMilestone("BIG-EPIC-12", "计费、套餐与商业化控制", "Phase 5", "commercialization", "M5 Billing and packaging"),
        ],
    )
    roadmap.validate_unique_owners()
    return roadmap


_install_support_module(
    "roadmap",
    EpicMilestone=EpicMilestone,
    ExecutionPackRoadmap=ExecutionPackRoadmap,
    build_execution_pack_roadmap=build_execution_pack_roadmap,
)


@dataclass
class PilotKPI:
    name: str
    target: float
    actual: float
    higher_is_better: bool = True

    @property
    def met(self) -> bool:
        return self.actual >= self.target if self.higher_is_better else self.actual <= self.target


@dataclass
class PilotImplementationResult:
    customer: str
    environment: str
    kpis: List[PilotKPI] = field(default_factory=list)
    production_runs: int = 0
    incidents: int = 0

    @property
    def kpi_pass_rate(self) -> float:
        if not self.kpis:
            return 0.0
        passed = len([kpi for kpi in self.kpis if kpi.met])
        return round((passed / len(self.kpis)) * 100, 1)

    @property
    def ready(self) -> bool:
        return self.production_runs > 0 and self.incidents == 0 and self.kpi_pass_rate >= 80.0


def render_pilot_implementation_report(result: PilotImplementationResult) -> str:
    lines = [
        "# Pilot Implementation Report",
        "",
        f"- Customer: {result.customer}",
        f"- Environment: {result.environment}",
        f"- Production Runs: {result.production_runs}",
        f"- Incidents: {result.incidents}",
        f"- KPI Pass Rate: {result.kpi_pass_rate}%",
        f"- Ready: {result.ready}",
        "",
        "## KPI Details",
        "",
    ]
    if result.kpis:
        for kpi in result.kpis:
            lines.append(f"- {kpi.name}: target={kpi.target} actual={kpi.actual} met={kpi.met}")
    else:
        lines.append("- none")
    return "\n".join(lines) + "\n"


_install_support_module(
    "pilot",
    PilotKPI=PilotKPI,
    PilotImplementationResult=PilotImplementationResult,
    render_pilot_implementation_report=render_pilot_implementation_report,
)


class ParallelIssueQueue:
    def __init__(self, queue_path: str):
        self.queue_path = Path(queue_path)
        self.payload = json.loads(self.queue_path.read_text())

    def project_slug(self) -> str:
        return str(self.payload["project"]["slug_id"])

    def activate_state_id(self) -> str:
        return str(self.payload["policy"]["activate_state_id"])

    def target_in_progress(self) -> int:
        return int(self.payload["policy"]["target_in_progress"])

    def refill_states(self) -> set[str]:
        return {str(name) for name in self.payload["policy"].get("refill_states", [])}

    def issue_order(self) -> List[str]:
        return [str(identifier) for identifier in self.payload.get("issue_order", [])]

    def issue_records(self) -> List[dict]:
        return list(self.payload.get("issues", []))

    def issue_identifiers(self) -> List[str]:
        return [str(record["identifier"]) for record in self.issue_records()]

    def select_candidates(
        self,
        active_identifiers: set[str],
        issue_states: Dict[str, str],
        target_in_progress: Optional[int] = None,
    ) -> List[str]:
        target = self.target_in_progress() if target_in_progress is None else int(target_in_progress)
        needed = max(target - len(active_identifiers), 0)
        if needed == 0:
            return []
        candidates: List[str] = []
        refill_states = self.refill_states()
        for identifier in self.issue_order():
            if needed == 0:
                break
            if identifier in active_identifiers:
                continue
            if issue_states.get(identifier) in refill_states:
                candidates.append(identifier)
                needed -= 1
        return candidates


def issue_state_map(issues: Sequence[dict]) -> Dict[str, str]:
    state_map: Dict[str, str] = {}
    for issue in issues:
        identifier = str(issue.get("identifier", "")).strip()
        state = issue.get("state") or {}
        state_name = str(state.get("name", issue.get("state_name", ""))).strip()
        if identifier and state_name:
            state_map[identifier] = state_name
    return state_map


_install_support_module(
    "parallel_refill",
    ParallelIssueQueue=ParallelIssueQueue,
    issue_state_map=issue_state_map,
)


WORKSPACE_BOOTSTRAP_CLI_DEFAULT_CACHE_BASE = "~/.cache/symphony/repos"


def build_workspace_bootstrap_parser(
    description: str,
    default_repo_url: str,
    default_branch: str,
    default_cache_root: str | None,
    default_cache_base: str,
    default_cache_key: str | None,
) -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument("command", choices=["bootstrap", "cleanup"])
    parser.add_argument("--workspace", default=".", help="Workspace path managed by Symphony.")
    parser.add_argument("--issue", default="", help="Linear issue identifier used for the bootstrap branch.")
    parser.add_argument("--repo-url", default=default_repo_url, help="Canonical remote repository URL.")
    parser.add_argument("--default-branch", default=default_branch, help="Default branch used as the bootstrap base.")
    parser.add_argument(
        "--cache-root",
        default=default_cache_root,
        help="Full cache root that contains mirror.git and seed. Overrides --cache-base/--cache-key.",
    )
    parser.add_argument(
        "--cache-base",
        default=default_cache_base,
        help="Base directory that stores per-repo cache roots.",
    )
    parser.add_argument(
        "--cache-key",
        default=default_cache_key,
        help="Optional stable key for the per-repo cache directory.",
    )
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser


def emit_workspace_bootstrap_payload(payload: dict, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


def run_workspace_bootstrap_cli(
    argv: Sequence[str] | None = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: str | None = None,
    default_cache_base: str = WORKSPACE_BOOTSTRAP_CLI_DEFAULT_CACHE_BASE,
    default_cache_key: str | None = None,
) -> int:
    parser = build_workspace_bootstrap_parser(
        description=description,
        default_repo_url=default_repo_url,
        default_branch=default_branch,
        default_cache_root=default_cache_root,
        default_cache_base=default_cache_base,
        default_cache_key=default_cache_key,
    )
    args = parser.parse_args(argv)
    workspace = Path(args.workspace).expanduser().resolve()

    try:
        payload = dict(
            workspace=workspace,
            issue_identifier=args.issue,
            repo_url=args.repo_url,
            default_branch=args.default_branch,
            cache_root=args.cache_root,
            cache_base=args.cache_base,
            cache_key=args.cache_key,
        )
        if args.command == "bootstrap":
            status = bootstrap_workspace(**payload)
        else:
            status = cleanup_workspace(**payload)
        emit_workspace_bootstrap_payload({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except WorkspaceBootstrapError as exc:
        emit_workspace_bootstrap_payload(
            {"status": "error", "workspace": str(workspace), "error": str(exc)},
            args.json,
        )
        return 1


_install_support_module(
    "workspace_bootstrap_cli",
    DEFAULT_CACHE_BASE=WORKSPACE_BOOTSTRAP_CLI_DEFAULT_CACHE_BASE,
    build_parser=build_workspace_bootstrap_parser,
    emit=emit_workspace_bootstrap_payload,
    main=run_workspace_bootstrap_cli,
)


class ValidationReportDecision:
    def __init__(
        self,
        allowed_to_close: bool,
        status: str,
        summary: str,
        missing_reports: Optional[List[str]] = None,
    ) -> None:
        self.allowed_to_close = allowed_to_close
        self.status = status
        self.summary = summary
        self.missing_reports = list(missing_reports or [])


REQUIRED_REPORT_ARTIFACTS = [
    "task-run",
    "replay",
    "benchmark-suite",
]


def enforce_validation_report_policy(artifacts: List[str]) -> ValidationReportDecision:
    existing = set(artifacts)
    missing = [name for name in REQUIRED_REPORT_ARTIFACTS if name not in existing]
    if missing:
        return ValidationReportDecision(
            allowed_to_close=False,
            status="blocked",
            summary="validation report policy not satisfied",
            missing_reports=missing,
        )
    return ValidationReportDecision(
        allowed_to_close=True,
        status="ready",
        summary="validation report policy satisfied",
    )


_install_support_module(
    "validation_policy",
    ValidationReportDecision=ValidationReportDecision,
    REQUIRED_REPORT_ARTIFACTS=REQUIRED_REPORT_ARTIFACTS,
    enforce_validation_report_policy=enforce_validation_report_policy,
)


def build_workspace_validation_report(
    *,
    repo_url: str,
    workspace_root: str | Path,
    issue_identifiers: Sequence[str],
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    cleanup: bool = True,
) -> dict[str, Any]:
    workspace_root_path = Path(workspace_root).expanduser().resolve()
    workspace_root_path.mkdir(parents=True, exist_ok=True)

    bootstrap_results = []
    for issue_identifier in issue_identifiers:
        workspace_path = workspace_root_path / issue_identifier
        status = bootstrap_workspace(
            workspace=workspace_path,
            issue_identifier=issue_identifier,
            repo_url=repo_url,
            default_branch=default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
        bootstrap_results.append(status.to_dict())

    cache_roots = sorted({result["cache_root"] for result in bootstrap_results})
    mirror_paths = sorted({result["mirror_path"] for result in bootstrap_results})
    seed_paths = sorted({result["seed_path"] for result in bootstrap_results})
    cleanup_results = []

    if cleanup:
        for issue_identifier in issue_identifiers:
            workspace_path = workspace_root_path / issue_identifier
            status = cleanup_workspace(
                workspace=workspace_path,
                issue_identifier=issue_identifier,
                repo_url=repo_url,
                default_branch=default_branch,
                cache_root=cache_root,
                cache_base=cache_base,
                cache_key=cache_key,
            )
            cleanup_results.append(status.to_dict())

    return {
        "repo_url": repo_url,
        "default_branch": default_branch,
        "workspace_root": str(workspace_root_path),
        "issue_identifiers": list(issue_identifiers),
        "bootstrap_results": bootstrap_results,
        "cleanup_results": cleanup_results,
        "summary": {
            "workspace_count": len(bootstrap_results),
            "unique_cache_roots": cache_roots,
            "unique_mirror_paths": mirror_paths,
            "unique_seed_paths": seed_paths,
            "single_cache_root_reused": len(cache_roots) == 1,
            "single_mirror_reused": len(mirror_paths) == 1,
            "single_seed_reused": len(seed_paths) == 1,
            "mirror_creations": sum(1 for result in bootstrap_results if result["mirror_created"]),
            "seed_creations": sum(1 for result in bootstrap_results if result["seed_created"]),
            "clone_suppressed_after_first": all(result["clone_suppressed"] for result in bootstrap_results[1:]),
            "cache_reused_after_first": all(result["cache_reused"] for result in bootstrap_results[1:]),
            "all_workspaces_created_via_worktree": all(
                result["workspace_mode"] in {"worktree_created", "workspace_reused"}
                for result in bootstrap_results
            ),
            "cleanup_preserved_cache": bool(bootstrap_results)
            and Path(bootstrap_results[0]["mirror_path"]).exists()
            and Path(bootstrap_results[0]["seed_path"]).joinpath(".git").exists(),
        },
    }


def render_workspace_validation_markdown(report: dict[str, Any]) -> str:
    summary = report["summary"]
    lines = [
        "# Symphony bootstrap cache validation",
        "",
        f"- Repo: `{report['repo_url']}`",
        f"- Workspace root: `{report['workspace_root']}`",
        f"- Workspaces: `{summary['workspace_count']}`",
        f"- Single cache root reused: `{summary['single_cache_root_reused']}`",
        f"- Mirror creations: `{summary['mirror_creations']}`",
        f"- Seed creations: `{summary['seed_creations']}`",
        f"- Clone suppressed after first workspace: `{summary['clone_suppressed_after_first']}`",
        f"- Cleanup preserved cache: `{summary['cleanup_preserved_cache']}`",
        "",
        "## Bootstrap Results",
        "",
    ]

    for result in report["bootstrap_results"]:
        lines.extend(
            [
                f"- `{result['workspace']}`",
                f"  - `cache_root={result['cache_root']}`",
                f"  - `cache_key={result['cache_key']}`",
                f"  - `workspace_mode={result['workspace_mode']}`",
                f"  - `cache_reused={result['cache_reused']}`",
                f"  - `clone_suppressed={result['clone_suppressed']}`",
                f"  - `mirror_created={result['mirror_created']}`",
                f"  - `seed_created={result['seed_created']}`",
            ]
        )

    return "\n".join(lines) + "\n"


def write_workspace_validation_report(report: dict[str, Any], path: str | Path) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)

    if target.suffix.lower() == ".md":
        target.write_text(render_workspace_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target


_install_support_module(
    "workspace_bootstrap_validation",
    build_validation_report=build_workspace_validation_report,
    render_validation_markdown=render_workspace_validation_markdown,
    write_validation_report=write_workspace_validation_report,
)


class GitSyncError(RuntimeError):
    """Raised when repository sync automation cannot complete safely."""


@dataclass
class RepoSyncStatus:
    branch: str
    local_sha: str
    remote_sha: str
    dirty: bool
    remote_exists: bool
    synced: bool
    pushed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class GitSyncCommandResult:
    stdout: str
    stderr: str
    returncode: int


EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH


def _run_git_sync_command(command: Sequence[str], repo: Path) -> GitSyncCommandResult:
    completed = subprocess.run(
        list(command),
        cwd=repo,
        text=True,
        capture_output=True,
        check=False,
    )
    return GitSyncCommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git_sync_git(repo: Path, *args: str) -> GitSyncCommandResult:
    return _run_git_sync_command(["git", *args], repo)


def _require_git_sync(repo: Path, *args: str) -> str:
    result = _git_sync_git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise GitSyncError(detail)
    return result.stdout


def _git_sync_dirty(repo: Path) -> bool:
    return bool(_require_git_sync(repo, "status", "--porcelain"))


def _git_sync_remote_default_branch(repo: Path, remote: str) -> str:
    symbolic_ref = _git_sync_git(repo, "symbolic-ref", "--quiet", f"refs/remotes/{remote}/HEAD")
    if symbolic_ref.returncode == 0 and symbolic_ref.stdout:
        prefix = f"refs/remotes/{remote}/"
        if symbolic_ref.stdout.startswith(prefix):
            return symbolic_ref.stdout[len(prefix) :]

    symref_result = _git_sync_git(repo, "ls-remote", "--symref", remote, "HEAD")
    if symref_result.returncode != 0:
        detail = symref_result.stderr or symref_result.stdout or f"git ls-remote --symref failed for {remote}/HEAD"
        raise GitSyncError(detail)

    for line in symref_result.stdout.splitlines():
        if line.startswith("ref: ") and line.endswith("\tHEAD"):
            ref = line.split()[1]
            prefix = "refs/heads/"
            if ref.startswith(prefix):
                return ref[len(prefix) :]

    raise GitSyncError(f"Could not determine default branch for remote {remote}")


def _git_sync_remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git_sync_git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git_sync_git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _git_sync_matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _git_sync_remote_default_branch(repo, remote)
        default_sha = _git_sync_remote_branch_sha(repo, remote, default_branch)
    except GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


def inspect_repo_sync(repo: Path | str, remote: str = "origin") -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git_sync(repo_path, "branch", "--show-current")
    if not branch:
        raise GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git_sync(repo_path, "rev-parse", "HEAD")
    remote_result = _git_sync_git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _git_sync_dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha

    if not remote_exists and _git_sync_matches_remote_default_branch(repo_path, remote, local_sha):
        synced = True

    return RepoSyncStatus(
        branch=branch,
        local_sha=local_sha,
        remote_sha=remote_sha,
        dirty=dirty,
        remote_exists=remote_exists,
        synced=synced,
    )


def install_git_hooks(repo: Path | str, hooks_path: str = ".githooks") -> Path:
    repo_path = Path(repo).resolve()
    hooks_dir = repo_path / hooks_path
    if not hooks_dir.is_dir():
        raise GitSyncError(f"Missing hooks directory: {hooks_dir}")

    _require_git_sync(repo_path, "config", "core.hooksPath", hooks_path)
    for hook in hooks_dir.iterdir():
        if hook.is_file():
            hook.chmod(hook.stat().st_mode | EXECUTABLE_BITS)
    return hooks_dir


def ensure_repo_sync(
    repo: Path | str,
    remote: str = "origin",
    auto_push: bool = True,
    allow_dirty: bool = False,
) -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    status = inspect_repo_sync(repo_path, remote=remote)

    if status.dirty:
        if allow_dirty:
            return status
        raise GitSyncError("Working tree is dirty; commit or stash changes before syncing")

    if status.remote_exists and not status.synced:
        fetch_result = _git_sync_git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise GitSyncError(detail)

        ff_result = _git_sync_git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise GitSyncError(detail)
        status = inspect_repo_sync(repo_path, remote=remote)

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git_sync_git(repo_path, *push_args)
    if push_result.returncode != 0:
        detail = push_result.stderr or push_result.stdout or f"git {' '.join(push_args)} failed"
        raise GitSyncError(detail)

    refreshed = inspect_repo_sync(repo_path, remote=remote)
    refreshed.pushed = True
    if not refreshed.synced:
        raise GitSyncError(
            f"Remote SHA mismatch after push: local={refreshed.local_sha} remote={refreshed.remote_sha or 'missing'}"
        )
    return refreshed


_install_support_module(
    "github_sync",
    GitSyncError=GitSyncError,
    RepoSyncStatus=RepoSyncStatus,
    CommandResult=GitSyncCommandResult,
    EXECUTABLE_BITS=EXECUTABLE_BITS,
    inspect_repo_sync=inspect_repo_sync,
    install_git_hooks=install_git_hooks,
    ensure_repo_sync=ensure_repo_sync,
)


PULL_REQUEST_COMMENT_EVENT = "pull_request.comment"
CI_COMPLETED_EVENT = "ci.completed"
TASK_FAILED_EVENT = "task.failed"


def _event_bus_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


EventSubscriber = Callable[["BusEvent", "TaskRun"], None]


@dataclass(frozen=True)
class BusEvent:
    event_type: str
    run_id: str
    actor: str
    details: Dict[str, Any] = field(default_factory=dict)
    timestamp: str = field(default_factory=_event_bus_now)


class EventBus:
    def __init__(self, ledger: Optional[ObservabilityLedger] = None):
        self.ledger = ledger
        self._runs: Dict[str, TaskRun] = {}
        self._subscribers: dict[str, List[EventSubscriber]] = {}

    def register_run(self, run: TaskRun) -> None:
        self._runs[run.run_id] = run

    def subscribe(self, event_type: str, handler: EventSubscriber) -> None:
        self._subscribers.setdefault(event_type, []).append(handler)

    def publish(self, event: BusEvent) -> TaskRun:
        run = self._resolve_run(event.run_id)
        previous_status = run.status
        self._record_event(run, event)

        next_status, summary = self._resolve_transition(run, event)
        if next_status:
            run.finalize(next_status, summary)
            run.audit(
                "event_bus.transition",
                "event-bus",
                next_status,
                event_type=event.event_type,
                previous_status=previous_status,
                status=next_status,
                summary=summary,
                event_timestamp=event.timestamp,
            )

        for handler in self._subscribers.get(event.event_type, []):
            handler(event, run)

        if self.ledger is not None:
            self.ledger.upsert(run)
        return run

    def _resolve_run(self, run_id: str) -> TaskRun:
        registered = self._runs.get(run_id)
        if registered is not None:
            return registered
        if self.ledger is not None:
            for run in self.ledger.load_runs():
                if run.run_id == run_id:
                    self._runs[run_id] = run
                    return run
        raise KeyError(f"run {run_id!r} is not registered with the event bus")

    def _record_event(self, run: TaskRun, event: BusEvent) -> None:
        run.audit(
            "event_bus.event",
            event.actor,
            "received",
            event_type=event.event_type,
            event_timestamp=event.timestamp,
            **event.details,
        )
        if event.event_type != PULL_REQUEST_COMMENT_EVENT:
            return
        body = str(event.details.get("body", "")).strip()
        if not body:
            return
        mentions = [str(item) for item in event.details.get("mentions", [])]
        run.add_comment(
            author=event.actor,
            body=body,
            mentions=mentions,
            anchor="pull-request",
            surface="pull-request",
        )

    def _resolve_transition(self, run: TaskRun, event: BusEvent) -> tuple[str, str]:
        explicit_status = str(event.details.get("target_status", "")).strip()
        if explicit_status:
            return explicit_status, self._build_summary(event, explicit_status)

        if event.event_type == PULL_REQUEST_COMMENT_EVENT:
            decision = str(event.details.get("decision", "")).strip().lower()
            if decision in {"approved", "accept", "accepted", "lgtm"}:
                return "approved", self._build_summary(event, "approved")
            if decision in {"blocked", "changes-requested", "rejected"}:
                return "needs-approval", self._build_summary(event, "needs-approval")
        elif event.event_type == CI_COMPLETED_EVENT:
            conclusion = str(event.details.get("conclusion", "")).strip().lower()
            if conclusion in {"success", "passed", "green"}:
                return "completed", self._build_summary(event, "completed")
            if conclusion in {"cancelled", "canceled", "error", "failed", "failure", "timed_out"}:
                return "failed", self._build_summary(event, "failed")
        elif event.event_type == TASK_FAILED_EVENT:
            return "failed", self._build_summary(event, "failed")

        return "", run.summary

    def _build_summary(self, event: BusEvent, status: str) -> str:
        summary = str(event.details.get("summary", "")).strip()
        if summary:
            return summary
        if event.event_type == PULL_REQUEST_COMMENT_EVENT:
            body = str(event.details.get("body", "")).strip()
            if body:
                return body
            return f"pull request comment set run to {status}"
        if event.event_type == CI_COMPLETED_EVENT:
            workflow = str(event.details.get("workflow", "")).strip()
            conclusion = str(event.details.get("conclusion", "")).strip() or status
            if workflow:
                return f"CI workflow {workflow} completed with {conclusion}"
            return f"CI completed with {conclusion}"
        if event.event_type == TASK_FAILED_EVENT:
            reason = str(event.details.get("error", "")).strip() or str(event.details.get("reason", "")).strip()
            if reason:
                return reason
            return "task failed"
        return status


_install_support_module(
    "event_bus",
    PULL_REQUEST_COMMENT_EVENT=PULL_REQUEST_COMMENT_EVENT,
    CI_COMPLETED_EVENT=CI_COMPLETED_EVENT,
    TASK_FAILED_EVENT=TASK_FAILED_EVENT,
    BusEvent=BusEvent,
    EventBus=EventBus,
)


@dataclass
class MemoryPattern:
    task_id: str
    title: str
    labels: List[str] = field(default_factory=list)
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    summary: str = ""

    def to_dict(self) -> Dict:
        return {
            "task_id": self.task_id,
            "title": self.title,
            "labels": list(self.labels),
            "required_tools": list(self.required_tools),
            "acceptance_criteria": list(self.acceptance_criteria),
            "validation_plan": list(self.validation_plan),
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict) -> "MemoryPattern":
        return cls(
            task_id=str(data.get("task_id", "")),
            title=str(data.get("title", "")),
            labels=[str(item) for item in data.get("labels", [])],
            required_tools=[str(item) for item in data.get("required_tools", [])],
            acceptance_criteria=[str(item) for item in data.get("acceptance_criteria", [])],
            validation_plan=[str(item) for item in data.get("validation_plan", [])],
            summary=str(data.get("summary", "")),
        )


class TaskMemoryStore:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def _load_patterns(self) -> List[MemoryPattern]:
        if not self.storage_path.exists():
            return []
        payload = json.loads(self.storage_path.read_text())
        return [MemoryPattern.from_dict(item) for item in payload]

    def _write_patterns(self, patterns: List[MemoryPattern]) -> None:
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps([item.to_dict() for item in patterns], ensure_ascii=False, indent=2))

    def remember_success(self, task: Task, summary: str = "") -> None:
        patterns = [item for item in self._load_patterns() if item.task_id != task.task_id]
        patterns.append(
            MemoryPattern(
                task_id=task.task_id,
                title=task.title,
                labels=list(task.labels),
                required_tools=list(task.required_tools),
                acceptance_criteria=list(task.acceptance_criteria),
                validation_plan=list(task.validation_plan),
                summary=summary,
            )
        )
        self._write_patterns(patterns)

    def suggest_rules(self, task: Task, limit: int = 3) -> Dict[str, List[str]]:
        ranked: List[tuple[float, MemoryPattern]] = []
        for pattern in self._load_patterns():
            score = self._score(task, pattern)
            if score <= 0:
                continue
            ranked.append((score, pattern))

        ranked.sort(key=lambda item: item[0], reverse=True)
        selected = [item[1] for item in ranked[: max(1, limit)]]

        acceptance = list(task.acceptance_criteria)
        validation = list(task.validation_plan)
        for pattern in selected:
            for item in pattern.acceptance_criteria:
                if item not in acceptance:
                    acceptance.append(item)
            for item in pattern.validation_plan:
                if item not in validation:
                    validation.append(item)

        return {
            "acceptance_criteria": acceptance,
            "validation_plan": validation,
            "matched_task_ids": [item.task_id for item in selected],
        }

    @staticmethod
    def _score(task: Task, pattern: MemoryPattern) -> float:
        label_overlap = len(set(task.labels) & set(pattern.labels))
        tool_overlap = len(set(task.required_tools) & set(pattern.required_tools))
        return float(label_overlap * 2 + tool_overlap)


_install_support_module(
    "memory",
    MemoryPattern=MemoryPattern,
    TaskMemoryStore=TaskMemoryStore,
)


VALID_WORKFLOW_STEP_KINDS = {
    "scheduler",
    "approval",
    "orchestration",
    "report",
    "closeout",
}


@dataclass
class WorkflowStep:
    name: str
    kind: str
    required: bool = True
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "kind": self.kind,
            "required": self.required,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkflowStep":
        return cls(
            name=data["name"],
            kind=data["kind"],
            required=data.get("required", True),
            metadata=data.get("metadata", {}),
        )


@dataclass
class WorkflowDefinition:
    name: str
    steps: List[WorkflowStep] = field(default_factory=list)
    report_path_template: Optional[str] = None
    journal_path_template: Optional[str] = None
    validation_evidence: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "steps": [step.to_dict() for step in self.steps],
            "report_path_template": self.report_path_template,
            "journal_path_template": self.journal_path_template,
            "validation_evidence": self.validation_evidence,
            "approvals": self.approvals,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkflowDefinition":
        return cls(
            name=data["name"],
            steps=[WorkflowStep.from_dict(item) for item in data.get("steps", [])],
            report_path_template=data.get("report_path_template"),
            journal_path_template=data.get("journal_path_template"),
            validation_evidence=data.get("validation_evidence", []),
            approvals=data.get("approvals", []),
        )

    @classmethod
    def from_json(cls, text: str) -> "WorkflowDefinition":
        return cls.from_dict(json.loads(text))

    def render_path(self, template: Optional[str], task: Task, run_id: str) -> Optional[str]:
        if template is None:
            return None
        return template.format(
            workflow=self.name,
            task_id=task.task_id,
            source=task.source,
            run_id=run_id,
        )

    def render_report_path(self, task: Task, run_id: str) -> Optional[str]:
        return self.render_path(self.report_path_template, task, run_id)

    def render_journal_path(self, task: Task, run_id: str) -> Optional[str]:
        return self.render_path(self.journal_path_template, task, run_id)

    def validate(self) -> None:
        invalid_steps = [step.kind for step in self.steps if step.kind not in VALID_WORKFLOW_STEP_KINDS]
        if invalid_steps:
            joined = ", ".join(sorted(set(invalid_steps)))
            raise ValueError(f"invalid workflow step kind(s): {joined}")


_install_support_module(
    "dsl",
    WorkflowStep=WorkflowStep,
    WorkflowDefinition=WorkflowDefinition,
)


@dataclass
class RiskFactor:
    name: str
    points: int
    reason: str


@dataclass
class RiskScore:
    level: RiskLevel
    total: int
    requires_approval: bool
    factors: List[RiskFactor] = field(default_factory=list)

    @property
    def summary(self) -> str:
        return ", ".join(f"{factor.name}={factor.points}" for factor in self.factors) or "baseline=0"


class RiskScorer:
    TOOL_POINTS = {
        "browser": 10,
        "terminal": 15,
        "github": 10,
        "deploy": 20,
        "sql": 15,
        "warehouse": 15,
        "bi": 10,
    }
    LABEL_POINTS = {
        "security": 20,
        "compliance": 20,
        "prod": 20,
        "release": 15,
        "ops": 10,
    }

    def score_task(self, task: Task) -> RiskScore:
        factors: List[RiskFactor] = []
        total = 0

        risk_points = {
            RiskLevel.LOW: 0,
            RiskLevel.MEDIUM: 30,
            RiskLevel.HIGH: 60,
        }[task.risk_level]
        total += risk_points
        if risk_points:
            factors.append(RiskFactor("risk-level", risk_points, f"declared risk level {task.risk_level.value}"))

        if task.priority == Priority.P0:
            total += 10
            factors.append(RiskFactor("priority", 10, "p0 task needs tighter controls"))

        labels = {label.lower() for label in task.labels}
        for label in sorted(labels):
            points = self.LABEL_POINTS.get(label)
            if points:
                total += points
                factors.append(RiskFactor(f"label:{label}", points, f"label {label} increases operational risk"))

        tools = {tool.lower() for tool in task.required_tools}
        for tool in sorted(tools):
            points = self.TOOL_POINTS.get(tool)
            if points:
                total += points
                factors.append(RiskFactor(f"tool:{tool}", points, f"tool {tool} expands execution surface"))

        if task.budget < 0:
            total += 20
            factors.append(RiskFactor("budget", 20, "invalid budget requires manual review"))

        level = self._level_for_total(total)
        return RiskScore(level=level, total=total, requires_approval=(level == RiskLevel.HIGH), factors=factors)

    def _level_for_total(self, total: int) -> RiskLevel:
        if total >= 60:
            return RiskLevel.HIGH
        if total >= 25:
            return RiskLevel.MEDIUM
        return RiskLevel.LOW


_install_support_module(
    "risk",
    RiskFactor=RiskFactor,
    RiskScore=RiskScore,
    RiskScorer=RiskScorer,
)


@dataclass
class RunDetailStat:
    label: str
    value: str
    tone: str = "default"


@dataclass
class RunDetailResource:
    name: str
    kind: str
    path: str
    meta: List[str] = field(default_factory=list)
    tone: str = "default"


@dataclass
class RunDetailEvent:
    event_id: str
    lane: str
    title: str
    timestamp: str
    status: str
    summary: str
    details: List[str] = field(default_factory=list)


@dataclass
class RunDetailTab:
    tab_id: str
    label: str
    body_html: str


def render_run_detail_console(
    *,
    page_title: str,
    eyebrow: str,
    hero_title: str,
    hero_summary: str,
    stats: List[RunDetailStat],
    tabs: List[RunDetailTab],
    timeline_events: List[RunDetailEvent],
) -> str:
    stat_cards = "".join(
        f"""
        <article class="stat-card" data-tone="{escape(stat.tone)}">
          <span>{escape(stat.label)}</span>
          <strong>{escape(stat.value)}</strong>
        </article>
        """
        for stat in stats
    )
    tab_buttons = "".join(
        f'<button class="tab-button" type="button" data-tab="{escape(tab.tab_id)}">{escape(tab.label)}</button>'
        for tab in tabs
    )
    tab_panels = "".join(
        f'<section class="tab-panel" data-panel="{escape(tab.tab_id)}">{tab.body_html}</section>'
        for tab in tabs
    )
    event_payload = [
        {
            "id": event.event_id,
            "lane": event.lane,
            "title": event.title,
            "timestamp": event.timestamp,
            "status": event.status,
            "summary": event.summary,
            "details": event.details,
        }
        for event in timeline_events
    ]
    timeline_json = json.dumps(event_payload).replace("</", "<\\/")

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{escape(page_title)}</title>
  <style>
    :root {{
      color-scheme: light;
      --paper: #f7f1e3;
      --ink: #13212f;
      --muted: #5f6c79;
      --line: rgba(19, 33, 47, 0.14);
      --panel: rgba(255, 251, 245, 0.92);
      --accent: #0f766e;
      --accent-soft: rgba(15, 118, 110, 0.14);
      --alert: #b45309;
      --danger: #b91c1c;
      --shadow: 0 18px 40px rgba(19, 33, 47, 0.12);
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.12), transparent 28%),
        linear-gradient(180deg, #fcfaf4 0%, var(--paper) 100%);
    }}
    main {{
      width: min(1160px, calc(100% - 2rem));
      margin: 0 auto;
      padding: 2rem 0 3rem;
    }}
    .shell {{
      border: 1px solid var(--line);
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.7);
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(10px);
    }}
    .hero {{
      padding: 1.5rem;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(255, 255, 255, 0.45));
    }}
    .eyebrow {{
      display: inline-block;
      margin-bottom: 0.65rem;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      font: 600 0.72rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
      color: var(--muted);
    }}
    h1, h2, h3, p {{ margin: 0; }}
    .hero p {{
      margin-top: 0.6rem;
      max-width: 70ch;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.6;
    }}
    .stats {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
      gap: 0.8rem;
      margin-top: 1.2rem;
    }}
    .stat-card, .surface {{
      border: 1px solid var(--line);
      border-radius: 18px;
      background: var(--panel);
      padding: 1rem;
    }}
    .stat-card span, .meta, .resource-meta, .timeline-meta, .detail-list {{
      display: block;
      color: var(--muted);
      font: 500 0.78rem/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .stat-card strong {{
      display: block;
      margin-top: 0.45rem;
      font-size: 1.15rem;
    }}
    .stat-card[data-tone="danger"] strong {{ color: var(--danger); }}
    .stat-card[data-tone="warning"] strong {{ color: var(--alert); }}
    .stat-card[data-tone="accent"] strong {{ color: var(--accent); }}
    .tabs {{
      display: flex;
      flex-wrap: wrap;
      gap: 0.6rem;
      padding: 1rem 1.5rem 0;
    }}
    .tab-button {{
      border: 1px solid var(--line);
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.72);
      color: var(--ink);
      padding: 0.55rem 0.95rem;
      cursor: pointer;
      font: 600 0.83rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .tab-button.active {{
      background: var(--accent);
      border-color: var(--accent);
      color: #f8fffd;
    }}
    .panel-stack {{
      padding: 1rem 1.5rem 1.5rem;
    }}
    .tab-panel {{
      display: none;
    }}
    .tab-panel.active {{
      display: block;
    }}
    .surface h2, .surface h3 {{
      margin-bottom: 0.6rem;
    }}
    .surface p {{
      color: var(--muted);
      line-height: 1.6;
    }}
    .resource-grid {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 0.8rem;
    }}
    .resource-card {{
      border: 1px solid var(--line);
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.8);
      padding: 0.95rem;
    }}
    .resource-card[data-tone="report"] {{
      background: linear-gradient(180deg, rgba(15, 118, 110, 0.08), rgba(255, 255, 255, 0.9));
    }}
    .resource-card[data-tone="page"] {{
      background: linear-gradient(180deg, rgba(180, 83, 9, 0.08), rgba(255, 255, 255, 0.92));
    }}
    .resource-card code, .detail-pane code {{
      font: 500 0.78rem/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
      word-break: break-all;
    }}
    .timeline-shell {{
      display: grid;
      grid-template-columns: minmax(280px, 0.95fr) minmax(0, 1.35fr);
      gap: 1rem;
    }}
    .timeline-list {{
      display: grid;
      gap: 0.65rem;
      max-height: 560px;
      overflow: auto;
      padding-right: 0.2rem;
    }}
    .timeline-item {{
      border: 1px solid var(--line);
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.82);
      text-align: left;
      padding: 0.9rem;
      cursor: pointer;
    }}
    .timeline-item.active {{
      border-color: var(--accent);
      box-shadow: inset 0 0 0 1px var(--accent);
      background: var(--accent-soft);
    }}
    .timeline-item strong {{
      display: block;
      margin: 0.25rem 0 0.45rem;
    }}
    .timeline-item p {{
      margin-top: 0.35rem;
      color: var(--muted);
      line-height: 1.5;
    }}
    .detail-pane {{
      min-height: 420px;
    }}
    .detail-pane ul {{
      margin: 0.8rem 0 0;
      padding-left: 1.2rem;
      line-height: 1.6;
    }}
    .kicker {{
      display: inline-block;
      border-radius: 999px;
      padding: 0.25rem 0.6rem;
      background: rgba(19, 33, 47, 0.06);
      font: 600 0.72rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .empty {{
      padding: 1rem;
      border: 1px dashed var(--line);
      border-radius: 14px;
      color: var(--muted);
      background: rgba(255, 255, 255, 0.62);
    }}
    @media (max-width: 860px) {{
      .timeline-shell {{
        grid-template-columns: 1fr;
      }}
      main {{
        width: min(100%, calc(100% - 1rem));
      }}
    }}
  </style>
</head>
<body>
  <main>
    <section class="shell">
      <header class="hero">
        <span class="eyebrow">{escape(eyebrow)}</span>
        <h1>{escape(hero_title)}</h1>
        <p>{escape(hero_summary)}</p>
        <div class="stats">{stat_cards}</div>
      </header>
      <nav class="tabs" aria-label="Run detail tabs">{tab_buttons}</nav>
      <div class="panel-stack">{tab_panels}</div>
    </section>
  </main>
  <script id="timeline-data" type="application/json">{timeline_json}</script>
  <script>
    const tabs = Array.from(document.querySelectorAll(".tab-button"));
    const panels = Array.from(document.querySelectorAll(".tab-panel"));
    const activateTab = (tabId) => {{
      tabs.forEach((button) => button.classList.toggle("active", button.dataset.tab === tabId));
      panels.forEach((panel) => panel.classList.toggle("active", panel.dataset.panel === tabId));
    }};
    if (tabs.length > 0) {{
      activateTab(tabs[0].dataset.tab);
      tabs.forEach((button) => button.addEventListener("click", () => activateTab(button.dataset.tab)));
    }}

    const timelineData = JSON.parse(document.getElementById("timeline-data").textContent);
    const timelineButtons = Array.from(document.querySelectorAll(".timeline-item"));
    const detailTitle = document.querySelector("[data-detail='title']");
    const detailMeta = document.querySelector("[data-detail='meta']");
    const detailSummary = document.querySelector("[data-detail='summary']");
    const detailList = document.querySelector("[data-detail='list']");
    const renderEvent = (eventId) => {{
      const match = timelineData.find((item) => item.id === eventId);
      if (!match || !detailTitle || !detailMeta || !detailSummary || !detailList) {{
        return;
      }}
      timelineButtons.forEach((button) => button.classList.toggle("active", button.dataset.eventId === eventId));
      detailTitle.textContent = match.title;
      detailMeta.textContent = `${{match.lane}} / ${{match.status}} / ${{match.timestamp}}`;
      detailSummary.textContent = match.summary;
      detailList.innerHTML = "";
      const items = match.details.length ? match.details : ["No additional details."];
      items.forEach((detail) => {{
        const li = document.createElement("li");
        li.textContent = detail;
        detailList.appendChild(li);
      }});
    }};
    if (timelineButtons.length > 0) {{
      renderEvent(timelineButtons[0].dataset.eventId);
      timelineButtons.forEach((button) => button.addEventListener("click", () => renderEvent(button.dataset.eventId)));
    }}
  </script>
</body>
</html>
"""


def render_resource_grid(title: str, description: str, resources: List[RunDetailResource]) -> str:
    if resources:
        cards = "".join(
            f"""
            <article class="resource-card" data-tone="{escape(resource.tone)}">
              <span class="kicker">{escape(resource.kind)}</span>
              <h3>{escape(resource.name)}</h3>
              <p><code>{escape(resource.path)}</code></p>
              <span class="resource-meta">{escape(" | ".join(resource.meta) if resource.meta else "No extra metadata")}</span>
            </article>
            """
            for resource in resources
        )
        body = f'<div class="resource-grid">{cards}</div>'
    else:
        body = '<div class="empty">No resources recorded.</div>'
    return f'<section class="surface"><h2>{escape(title)}</h2><p>{escape(description)}</p>{body}</section>'


def render_timeline_panel(title: str, description: str, timeline_events: List[RunDetailEvent]) -> str:
    if timeline_events:
        items = "".join(
            f"""
            <button class="timeline-item" type="button" data-event-id="{escape(event.event_id)}">
              <span class="kicker">{escape(event.lane)}</span>
              <strong>{escape(event.title)}</strong>
              <span class="timeline-meta">{escape(event.timestamp)} | {escape(event.status)}</span>
              <p>{escape(event.summary)}</p>
            </button>
            """
            for event in timeline_events
        )
    else:
        items = '<div class="empty">No timeline events recorded.</div>'
    return f"""
    <section class="surface">
      <h2>{escape(title)}</h2>
      <p>{escape(description)}</p>
      <div class="timeline-shell">
        <div class="timeline-list">{items}</div>
        <aside class="surface detail-pane">
          <span class="kicker">Inspector</span>
          <h3 data-detail="title">No event selected</h3>
          <span class="meta" data-detail="meta">timeline / idle / n/a</span>
          <p data-detail="summary">Select a timeline item to inspect the synced log, trace, audit, or artifact details.</p>
          <ul class="detail-list" data-detail="list"><li>No additional details.</li></ul>
        </aside>
      </div>
    </section>
    """


_install_support_module(
    "run_detail",
    RunDetailStat=RunDetailStat,
    RunDetailResource=RunDetailResource,
    RunDetailEvent=RunDetailEvent,
    RunDetailTab=RunDetailTab,
    render_run_detail_console=render_run_detail_console,
    render_resource_grid=render_resource_grid,
    render_timeline_panel=render_timeline_panel,
)


@dataclass(frozen=True)
class SchemaField:
    name: str
    field_type: str
    required: bool = True
    description: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field_type": self.field_type,
            "required": self.required,
            "description": self.description,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SchemaField":
        return cls(
            name=str(data["name"]),
            field_type=str(data["field_type"]),
            required=bool(data.get("required", True)),
            description=str(data.get("description", "")),
        )


@dataclass
class SurfaceSchema:
    name: str
    owner: str
    description: str = ""
    fields: List[SchemaField] = field(default_factory=list)
    sample: Dict[str, object] = field(default_factory=dict)

    @property
    def required_fields(self) -> List[str]:
        return [field.name for field in self.fields if field.required]

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "owner": self.owner,
            "description": self.description,
            "fields": [field.to_dict() for field in self.fields],
            "sample": self.sample,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceSchema":
        return cls(
            name=str(data["name"]),
            owner=str(data.get("owner", "")),
            description=str(data.get("description", "")),
            fields=[SchemaField.from_dict(field) for field in data.get("fields", [])],
            sample=dict(data.get("sample", {})),
        )


@dataclass
class DashboardRunContract:
    contract_id: str
    version: str
    dashboard_schema: SurfaceSchema
    run_detail_schema: SurfaceSchema

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_schema": self.dashboard_schema.to_dict(),
            "run_detail_schema": self.run_detail_schema.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContract":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_schema=SurfaceSchema.from_dict(dict(data["dashboard_schema"])),
            run_detail_schema=SurfaceSchema.from_dict(dict(data["run_detail_schema"])),
        )


@dataclass
class DashboardRunContractAudit:
    contract_id: str
    version: str
    dashboard_missing_fields: List[str] = field(default_factory=list)
    dashboard_sample_gaps: List[str] = field(default_factory=list)
    run_detail_missing_fields: List[str] = field(default_factory=list)
    run_detail_sample_gaps: List[str] = field(default_factory=list)

    @property
    def release_ready(self) -> bool:
        return not (
            self.dashboard_missing_fields
            or self.dashboard_sample_gaps
            or self.run_detail_missing_fields
            or self.run_detail_sample_gaps
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_missing_fields": list(self.dashboard_missing_fields),
            "dashboard_sample_gaps": list(self.dashboard_sample_gaps),
            "run_detail_missing_fields": list(self.run_detail_missing_fields),
            "run_detail_sample_gaps": list(self.run_detail_sample_gaps),
            "release_ready": self.release_ready,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContractAudit":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_missing_fields=[str(item) for item in data.get("dashboard_missing_fields", [])],
            dashboard_sample_gaps=[str(item) for item in data.get("dashboard_sample_gaps", [])],
            run_detail_missing_fields=[str(item) for item in data.get("run_detail_missing_fields", [])],
            run_detail_sample_gaps=[str(item) for item in data.get("run_detail_sample_gaps", [])],
        )


class DashboardRunContractLibrary:
    DASHBOARD_REQUIRED_FIELDS = [
        "dashboard_id",
        "generated_at",
        "period.label",
        "period.start",
        "period.end",
        "filters.team",
        "summary.total_runs",
        "summary.success_rate",
        "summary.approval_queue_depth",
        "summary.sla_breach_count",
        "kpis",
        "kpis[].name",
        "kpis[].value",
        "kpis[].target",
        "funnel",
        "blockers",
        "activity",
    ]
    RUN_DETAIL_REQUIRED_FIELDS = [
        "run_id",
        "task_id",
        "status",
        "started_at",
        "ended_at",
        "summary",
        "timeline",
        "timeline[].event_id",
        "timeline[].status",
        "artifacts",
        "closeout.validation_evidence",
        "closeout.git_push_succeeded",
        "closeout.git_log_stat_output",
    ]

    def audit(self, contract: DashboardRunContract) -> DashboardRunContractAudit:
        return DashboardRunContractAudit(
            contract_id=contract.contract_id,
            version=contract.version,
            dashboard_missing_fields=self._missing_field_defs(self.DASHBOARD_REQUIRED_FIELDS, contract.dashboard_schema.fields),
            dashboard_sample_gaps=self._missing_sample_paths(self.DASHBOARD_REQUIRED_FIELDS, contract.dashboard_schema.sample),
            run_detail_missing_fields=self._missing_field_defs(self.RUN_DETAIL_REQUIRED_FIELDS, contract.run_detail_schema.fields),
            run_detail_sample_gaps=self._missing_sample_paths(self.RUN_DETAIL_REQUIRED_FIELDS, contract.run_detail_schema.sample),
        )

    def build_default_contract(self) -> DashboardRunContract:
        return DashboardRunContract(
            contract_id="BIG-4301",
            version="v1",
            dashboard_schema=SurfaceSchema(
                name="DashboardKpiAggregate",
                owner="operations",
                description="Team dashboard response for KPI cards, funnel, blockers, and recent activity.",
                fields=[
                    SchemaField("dashboard_id", "string", description="Stable dashboard identifier."),
                    SchemaField("generated_at", "datetime", description="UTC generation timestamp."),
                    SchemaField("period.label", "string", description="Human-readable period label."),
                    SchemaField("period.start", "date", description="Inclusive reporting window start."),
                    SchemaField("period.end", "date", description="Inclusive reporting window end."),
                    SchemaField("filters.team", "string", description="Team or org filter applied."),
                    SchemaField("filters.viewer_role", "string", required=False, description="Persona used for visibility rules."),
                    SchemaField("summary.total_runs", "integer", description="Total runs in the period."),
                    SchemaField("summary.success_rate", "number", description="Completed-success ratio in percent."),
                    SchemaField("summary.approval_queue_depth", "integer", description="Runs waiting for approval."),
                    SchemaField("summary.sla_breach_count", "integer", description="Runs over SLA."),
                    SchemaField("kpis", "DashboardKpi[]", description="Ordered KPI cards shown in the hero grid."),
                    SchemaField("kpis[].name", "string", description="Metric identifier."),
                    SchemaField("kpis[].value", "number", description="Observed metric value."),
                    SchemaField("kpis[].target", "number", description="Target threshold."),
                    SchemaField("kpis[].unit", "string", required=False, description="Display suffix or unit."),
                    SchemaField("kpis[].direction", "string", required=False, description="Healthy trend direction."),
                    SchemaField("kpis[].healthy", "boolean", required=False, description="Precomputed health state."),
                    SchemaField("funnel", "DashboardFunnelStage[]", description="Pipeline distribution."),
                    SchemaField("blockers", "DashboardBlocker[]", description="Highest impact blockers."),
                    SchemaField("activity", "DashboardActivity[]", description="Most recent run activity."),
                ],
                sample={
                    "dashboard_id": "eng-overview-core-product",
                    "generated_at": "2026-03-11T09:30:00Z",
                    "period": {"label": "2026-W11", "start": "2026-03-09", "end": "2026-03-15"},
                    "filters": {"team": "core-product", "viewer_role": "engineering-manager"},
                    "summary": {"total_runs": 42, "success_rate": 88.1, "approval_queue_depth": 3, "sla_breach_count": 2},
                    "kpis": [
                        {"name": "success-rate", "value": 88.1, "target": 90.0, "unit": "%", "direction": "up", "healthy": False},
                        {"name": "average-cycle-minutes", "value": 47.3, "target": 60.0, "unit": "m", "direction": "down", "healthy": True},
                    ],
                    "funnel": [
                        {"name": "queued", "count": 5, "share": 11.9},
                        {"name": "in-progress", "count": 7, "share": 16.7},
                        {"name": "awaiting-approval", "count": 3, "share": 7.1},
                        {"name": "completed", "count": 27, "share": 64.3},
                    ],
                    "blockers": [
                        {
                            "summary": "Security scan failures on release branch",
                            "affected_runs": 2,
                            "affected_tasks": ["OPE-121", "OPE-127"],
                            "owner": "security",
                            "severity": "high",
                        }
                    ],
                    "activity": [
                        {
                            "timestamp": "2026-03-11T09:20:00Z",
                            "run_id": "run-204",
                            "task_id": "OPE-127",
                            "status": "failed",
                            "summary": "Security scan failed after dependency bump",
                        }
                    ],
                },
            ),
            run_detail_schema=SurfaceSchema(
                name="RunDetail",
                owner="runtime",
                description="Canonical run detail payload for replay, timeline inspection, artifacts, and closeout evidence.",
                fields=[
                    SchemaField("run_id", "string", description="Stable run identifier."),
                    SchemaField("task_id", "string", description="Parent task identifier."),
                    SchemaField("status", "string", description="Current run status."),
                    SchemaField("started_at", "datetime", description="Run start timestamp."),
                    SchemaField("ended_at", "datetime", description="Run end timestamp."),
                    SchemaField("summary", "string", description="Operator-readable summary."),
                    SchemaField("medium", "string", required=False, description="Execution medium."),
                    SchemaField("timeline", "RunTimelineEvent[]", description="Chronological execution events."),
                    SchemaField("timeline[].event_id", "string", description="Stable event identifier."),
                    SchemaField("timeline[].lane", "string", required=False, description="UI lane or track."),
                    SchemaField("timeline[].timestamp", "datetime", required=False, description="Event timestamp."),
                    SchemaField("timeline[].status", "string", description="Event outcome."),
                    SchemaField("artifacts", "RunArtifact[]", description="Artifacts emitted by the run."),
                    SchemaField("closeout.validation_evidence", "string[]", description="Validation proof captured before finish."),
                    SchemaField("closeout.git_push_succeeded", "boolean", description="Whether push completed successfully."),
                    SchemaField("closeout.git_push_output", "string", required=False, description="Push command output."),
                    SchemaField("closeout.git_log_stat_output", "string", description="Captured git log --stat output."),
                ],
                sample={
                    "run_id": "run-204",
                    "task_id": "OPE-127",
                    "status": "completed",
                    "started_at": "2026-03-11T08:58:00Z",
                    "ended_at": "2026-03-11T09:24:00Z",
                    "medium": "codex-cli",
                    "summary": "Shipped release-branch scan hardening and captured closeout evidence.",
                    "timeline": [
                        {
                            "event_id": "evt-1",
                            "lane": "analysis",
                            "timestamp": "2026-03-11T08:59:00Z",
                            "status": "completed",
                            "title": "Inspected release branch failures",
                            "summary": "Reviewed the failing dependency diff and scan output.",
                            "details": ["Focused on lockfile drift and transitive CVE triage."],
                        },
                        {
                            "event_id": "evt-2",
                            "lane": "validation",
                            "timestamp": "2026-03-11T09:22:00Z",
                            "status": "completed",
                            "title": "Validated fix",
                            "summary": "Executed targeted tests and recorded the validation report.",
                            "details": ["pytest passed", "report stored under reports/OPE-127-validation.md"],
                        },
                    ],
                    "artifacts": [
                        {
                            "name": "validation-report",
                            "kind": "report",
                            "path": "reports/OPE-127-validation.md",
                            "timestamp": "2026-03-11T09:23:00Z",
                            "sha256": "abc123",
                            "metadata": {"ticket": "OPE-127"},
                        }
                    ],
                    "closeout": {
                        "validation_evidence": [
                            "python3 -m pytest tests/test_security.py -> 7 passed",
                            "python3 -m pytest -> 126 passed",
                        ],
                        "git_push_succeeded": True,
                        "git_push_output": "To github.com:OpenAGIs/BigClaw.git\\n   abc123..def456  main -> main",
                        "git_log_stat_output": "commit def456\\n src/bigclaw/security.py | 12 ++++++++++--",
                    },
                },
            ),
        )

    def _missing_field_defs(self, required_fields: Sequence[str], fields: Sequence[SchemaField]) -> List[str]:
        defined = {field.name for field in fields}
        return [field_name for field_name in required_fields if field_name not in defined]

    def _missing_sample_paths(self, required_fields: Sequence[str], payload: Dict[str, object]) -> List[str]:
        return [field_name for field_name in required_fields if not self._path_exists(payload, field_name)]

    def _path_exists(self, payload: object, path: str) -> bool:
        current_items = [payload]
        for part in path.split("."):
            next_items: List[object] = []
            is_list = part.endswith("[]")
            key = part[:-2] if is_list else part
            for item in current_items:
                if not isinstance(item, dict) or key not in item:
                    continue
                value = item[key]
                if is_list:
                    if isinstance(value, list) and value:
                        next_items.extend(value)
                    else:
                        continue
                else:
                    next_items.append(value)
            if not next_items:
                return False
            current_items = next_items
        return True


def render_dashboard_run_contract_report(contract: DashboardRunContract, audit: DashboardRunContractAudit) -> str:
    sections = [
        "# Dashboard and Run Contract",
        "",
        f"- Contract ID: {contract.contract_id}",
        f"- Version: {contract.version}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Dashboard KPI Aggregate",
        f"- Name: {contract.dashboard_schema.name}",
        f"- Owner: {contract.dashboard_schema.owner}",
        f"- Required Fields: {', '.join(contract.dashboard_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.dashboard_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.dashboard_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.dashboard_schema.sample, indent=2, sort_keys=True),
        "```",
        "",
        "## Run Detail",
        f"- Name: {contract.run_detail_schema.name}",
        f"- Owner: {contract.run_detail_schema.owner}",
        f"- Required Fields: {', '.join(contract.run_detail_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.run_detail_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.run_detail_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.run_detail_schema.sample, indent=2, sort_keys=True),
        "```",
    ]
    return "\n".join(sections)


_install_support_module(
    "dashboard_run_contract",
    SchemaField=SchemaField,
    SurfaceSchema=SurfaceSchema,
    DashboardRunContract=DashboardRunContract,
    DashboardRunContractAudit=DashboardRunContractAudit,
    DashboardRunContractLibrary=DashboardRunContractLibrary,
    render_dashboard_run_contract_report=render_dashboard_run_contract_report,
)


REQUIRED_RUN_CLOSEOUTS = ("validation-evidence", "git-push", "git-log-stat")
ALLOWED_SCOPE_STATUSES = {"frozen", "approved-exception", "proposed"}


@dataclass(frozen=True)
class FreezeException:
    issue_id: str
    reason: str
    approved_by: str = ""
    decision_note: str = ""

    @property
    def approved(self) -> bool:
        return bool(self.approved_by.strip())

    def to_dict(self) -> Dict[str, str]:
        return {
            "issue_id": self.issue_id,
            "reason": self.reason,
            "approved_by": self.approved_by,
            "decision_note": self.decision_note,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "FreezeException":
        return cls(
            issue_id=data["issue_id"],
            reason=data.get("reason", ""),
            approved_by=data.get("approved_by", ""),
            decision_note=data.get("decision_note", ""),
        )


@dataclass
class GovernanceBacklogItem:
    issue_id: str
    title: str
    phase: str
    owner: str = ""
    status: str = "planned"
    scope_status: str = "frozen"
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    required_closeout: List[str] = field(default_factory=lambda: list(REQUIRED_RUN_CLOSEOUTS))
    linked_epics: List[str] = field(default_factory=list)
    notes: str = ""

    @property
    def missing_closeout_requirements(self) -> List[str]:
        present = {item.strip().lower() for item in self.required_closeout if item.strip()}
        return [item for item in REQUIRED_RUN_CLOSEOUTS if item not in present]

    @property
    def governance_ready(self) -> bool:
        return (
            bool(self.owner.strip())
            and self.scope_status in ALLOWED_SCOPE_STATUSES
            and bool(self.acceptance_criteria)
            and bool(self.validation_plan)
            and not self.missing_closeout_requirements
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "phase": self.phase,
            "owner": self.owner,
            "status": self.status,
            "scope_status": self.scope_status,
            "acceptance_criteria": list(self.acceptance_criteria),
            "validation_plan": list(self.validation_plan),
            "required_closeout": list(self.required_closeout),
            "linked_epics": list(self.linked_epics),
            "notes": self.notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "GovernanceBacklogItem":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            phase=str(data.get("phase", "")),
            owner=str(data.get("owner", "")),
            status=str(data.get("status", "planned")),
            scope_status=str(data.get("scope_status", "frozen")),
            acceptance_criteria=[str(item) for item in data.get("acceptance_criteria", [])],
            validation_plan=[str(item) for item in data.get("validation_plan", [])],
            required_closeout=[str(item) for item in data.get("required_closeout", [])],
            linked_epics=[str(item) for item in data.get("linked_epics", [])],
            notes=str(data.get("notes", "")),
        )


@dataclass
class ScopeFreezeBoard:
    name: str
    version: str
    freeze_date: str
    freeze_owner: str
    backlog_items: List[GovernanceBacklogItem] = field(default_factory=list)
    exceptions: List[FreezeException] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "freeze_date": self.freeze_date,
            "freeze_owner": self.freeze_owner,
            "backlog_items": [item.to_dict() for item in self.backlog_items],
            "exceptions": [exception.to_dict() for exception in self.exceptions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ScopeFreezeBoard":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            freeze_date=str(data.get("freeze_date", "")),
            freeze_owner=str(data.get("freeze_owner", "")),
            backlog_items=[GovernanceBacklogItem.from_dict(item) for item in data.get("backlog_items", [])],
            exceptions=[FreezeException.from_dict(item) for item in data.get("exceptions", [])],
        )


@dataclass
class ScopeFreezeAudit:
    board_name: str
    version: str
    total_items: int
    duplicate_issue_ids: List[str] = field(default_factory=list)
    missing_owners: List[str] = field(default_factory=list)
    missing_acceptance: List[str] = field(default_factory=list)
    missing_validation: List[str] = field(default_factory=list)
    missing_closeout_requirements: Dict[str, List[str]] = field(default_factory=dict)
    unauthorized_scope_changes: List[str] = field(default_factory=list)
    invalid_scope_statuses: List[str] = field(default_factory=list)
    unapproved_exceptions: List[str] = field(default_factory=list)

    @property
    def release_ready(self) -> bool:
        return not (
            self.duplicate_issue_ids
            or self.missing_owners
            or self.missing_acceptance
            or self.missing_validation
            or self.missing_closeout_requirements
            or self.unauthorized_scope_changes
            or self.invalid_scope_statuses
            or self.unapproved_exceptions
        )

    @property
    def readiness_score(self) -> float:
        checks = [
            not self.duplicate_issue_ids,
            not self.missing_owners,
            not self.missing_acceptance,
            not self.missing_validation,
            not self.missing_closeout_requirements,
            not self.unauthorized_scope_changes,
            not self.invalid_scope_statuses,
            not self.unapproved_exceptions,
        ]
        passed = sum(1 for item in checks if item)
        return round((passed / len(checks)) * 100, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "board_name": self.board_name,
            "version": self.version,
            "total_items": self.total_items,
            "duplicate_issue_ids": list(self.duplicate_issue_ids),
            "missing_owners": list(self.missing_owners),
            "missing_acceptance": list(self.missing_acceptance),
            "missing_validation": list(self.missing_validation),
            "missing_closeout_requirements": {
                issue_id: list(requirements)
                for issue_id, requirements in self.missing_closeout_requirements.items()
            },
            "unauthorized_scope_changes": list(self.unauthorized_scope_changes),
            "invalid_scope_statuses": list(self.invalid_scope_statuses),
            "unapproved_exceptions": list(self.unapproved_exceptions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ScopeFreezeAudit":
        return cls(
            board_name=str(data["board_name"]),
            version=str(data["version"]),
            total_items=int(data.get("total_items", 0)),
            duplicate_issue_ids=[str(item) for item in data.get("duplicate_issue_ids", [])],
            missing_owners=[str(item) for item in data.get("missing_owners", [])],
            missing_acceptance=[str(item) for item in data.get("missing_acceptance", [])],
            missing_validation=[str(item) for item in data.get("missing_validation", [])],
            missing_closeout_requirements={
                str(issue_id): [str(requirement) for requirement in requirements]
                for issue_id, requirements in dict(data.get("missing_closeout_requirements", {})).items()
            },
            unauthorized_scope_changes=[str(item) for item in data.get("unauthorized_scope_changes", [])],
            invalid_scope_statuses=[str(item) for item in data.get("invalid_scope_statuses", [])],
            unapproved_exceptions=[str(item) for item in data.get("unapproved_exceptions", [])],
        )


class ScopeFreezeGovernance:
    def audit(self, board: ScopeFreezeBoard) -> ScopeFreezeAudit:
        counts: Dict[str, int] = {}
        exception_index = {exception.issue_id: exception for exception in board.exceptions}

        for item in board.backlog_items:
            counts[item.issue_id] = counts.get(item.issue_id, 0) + 1
        duplicate_issue_ids = sorted(issue_id for issue_id, count in counts.items() if count > 1)

        missing_owners = sorted(item.issue_id for item in board.backlog_items if not item.owner.strip())
        missing_acceptance = sorted(item.issue_id for item in board.backlog_items if not item.acceptance_criteria)
        missing_validation = sorted(item.issue_id for item in board.backlog_items if not item.validation_plan)
        missing_closeout_requirements = {
            item.issue_id: item.missing_closeout_requirements
            for item in board.backlog_items
            if item.missing_closeout_requirements
        }
        invalid_scope_statuses = sorted(
            item.issue_id for item in board.backlog_items if item.scope_status not in ALLOWED_SCOPE_STATUSES
        )

        unauthorized_scope_changes: List[str] = []
        for item in board.backlog_items:
            if item.scope_status != "proposed":
                continue
            exception = exception_index.get(item.issue_id)
            if exception is None or not exception.approved:
                unauthorized_scope_changes.append(item.issue_id)

        unapproved_exceptions = sorted(exception.issue_id for exception in board.exceptions if not exception.approved)

        return ScopeFreezeAudit(
            board_name=board.name,
            version=board.version,
            total_items=len(board.backlog_items),
            duplicate_issue_ids=duplicate_issue_ids,
            missing_owners=missing_owners,
            missing_acceptance=missing_acceptance,
            missing_validation=missing_validation,
            missing_closeout_requirements=missing_closeout_requirements,
            unauthorized_scope_changes=sorted(unauthorized_scope_changes),
            invalid_scope_statuses=invalid_scope_statuses,
            unapproved_exceptions=unapproved_exceptions,
        )


def render_scope_freeze_report(board: ScopeFreezeBoard, audit: ScopeFreezeAudit) -> str:
    lines = [
        "# Scope Freeze Governance Report",
        "",
        f"- Name: {board.name}",
        f"- Version: {board.version}",
        f"- Freeze Date: {board.freeze_date}",
        f"- Freeze Owner: {board.freeze_owner}",
        f"- Backlog Items: {len(board.backlog_items)}",
        f"- Exceptions: {len(board.exceptions)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Backlog",
        "",
    ]

    if board.backlog_items:
        for item in board.backlog_items:
            closeout = ", ".join(item.required_closeout) or "none"
            lines.append(
                f"- {item.issue_id}: phase={item.phase} owner={item.owner or 'none'} "
                f"status={item.status} scope={item.scope_status} closeout={closeout}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Freeze Exceptions", ""])
    if board.exceptions:
        for exception in board.exceptions:
            lines.append(
                f"- {exception.issue_id}: approved_by={exception.approved_by or 'pending'} reason={exception.reason or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    lines.append(f"- Duplicate issues: {', '.join(audit.duplicate_issue_ids) if audit.duplicate_issue_ids else 'none'}")
    lines.append(f"- Missing owners: {', '.join(audit.missing_owners) if audit.missing_owners else 'none'}")
    lines.append(f"- Missing acceptance: {', '.join(audit.missing_acceptance) if audit.missing_acceptance else 'none'}")
    lines.append(f"- Missing validation: {', '.join(audit.missing_validation) if audit.missing_validation else 'none'}")
    if audit.missing_closeout_requirements:
        missing_closeout = "; ".join(
            f"{issue_id}={', '.join(requirements)}"
            for issue_id, requirements in sorted(audit.missing_closeout_requirements.items())
        )
    else:
        missing_closeout = "none"
    lines.append(f"- Missing closeout requirements: {missing_closeout}")
    lines.append(
        "- Unauthorized scope changes: "
        f"{', '.join(audit.unauthorized_scope_changes) if audit.unauthorized_scope_changes else 'none'}"
    )
    lines.append(f"- Invalid scope statuses: {', '.join(audit.invalid_scope_statuses) if audit.invalid_scope_statuses else 'none'}")
    lines.append(f"- Unapproved exceptions: {', '.join(audit.unapproved_exceptions) if audit.unapproved_exceptions else 'none'}")
    return "\n".join(lines) + "\n"


_install_support_module(
    "governance",
    REQUIRED_RUN_CLOSEOUTS=REQUIRED_RUN_CLOSEOUTS,
    ALLOWED_SCOPE_STATUSES=ALLOWED_SCOPE_STATUSES,
    FreezeException=FreezeException,
    GovernanceBacklogItem=GovernanceBacklogItem,
    ScopeFreezeBoard=ScopeFreezeBoard,
    ScopeFreezeAudit=ScopeFreezeAudit,
    ScopeFreezeGovernance=ScopeFreezeGovernance,
    render_scope_freeze_report=render_scope_freeze_report,
)


@dataclass
class RepoCommit:
    commit_hash: str
    title: str
    author: str = ""
    parent_hashes: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "commit_hash": self.commit_hash,
            "title": self.title,
            "author": self.author,
            "parent_hashes": list(self.parent_hashes),
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoCommit":
        return cls(
            commit_hash=str(data["commit_hash"]),
            title=str(data.get("title", "")),
            author=str(data.get("author", "")),
            parent_hashes=[str(item) for item in data.get("parent_hashes", [])],
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class CommitLineage:
    root_hash: str
    lineage: List[RepoCommit] = field(default_factory=list)
    children: Dict[str, List[str]] = field(default_factory=dict)
    leaves: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "root_hash": self.root_hash,
            "lineage": [item.to_dict() for item in self.lineage],
            "children": {key: list(value) for key, value in self.children.items()},
            "leaves": list(self.leaves),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitLineage":
        return cls(
            root_hash=str(data.get("root_hash", "")),
            lineage=[RepoCommit.from_dict(item) for item in data.get("lineage", [])],
            children={str(key): [str(v) for v in value] for key, value in dict(data.get("children", {})).items()},
            leaves=[str(item) for item in data.get("leaves", [])],
        )


@dataclass
class CommitDiff:
    left_hash: str
    right_hash: str
    files_changed: int
    insertions: int
    deletions: int
    summary: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "left_hash": self.left_hash,
            "right_hash": self.right_hash,
            "files_changed": self.files_changed,
            "insertions": self.insertions,
            "deletions": self.deletions,
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitDiff":
        return cls(
            left_hash=str(data.get("left_hash", "")),
            right_hash=str(data.get("right_hash", "")),
            files_changed=int(data.get("files_changed", 0)),
            insertions=int(data.get("insertions", 0)),
            deletions=int(data.get("deletions", 0)),
            summary=str(data.get("summary", "")),
        )


_install_support_module(
    "repo_commits",
    RepoCommit=RepoCommit,
    CommitLineage=CommitLineage,
    CommitDiff=CommitDiff,
)


@dataclass(frozen=True)
class RepoGatewayError:
    code: str
    message: str
    retryable: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return {"code": self.code, "message": self.message, "retryable": self.retryable}


def normalize_gateway_error(error: Exception) -> RepoGatewayError:
    message = str(error).lower()
    if "timeout" in message:
        return RepoGatewayError(code="timeout", message=str(error), retryable=True)
    if "not found" in message:
        return RepoGatewayError(code="not_found", message=str(error), retryable=False)
    return RepoGatewayError(code="gateway_error", message=str(error), retryable=False)


def normalize_commit(payload: Dict[str, Any]) -> RepoCommit:
    return RepoCommit.from_dict(payload)


def normalize_lineage(payload: Dict[str, Any]) -> CommitLineage:
    return CommitLineage.from_dict(payload)


def normalize_diff(payload: Dict[str, Any]) -> CommitDiff:
    return CommitDiff.from_dict(payload)


def repo_audit_payload(*, actor: str, action: str, outcome: str, commit_hash: str, repo_space_id: str) -> Dict[str, Any]:
    return {
        "actor": actor,
        "action": action,
        "outcome": outcome,
        "commit_hash": commit_hash,
        "repo_space_id": repo_space_id,
    }


_install_support_module(
    "repo_gateway",
    RepoGatewayError=RepoGatewayError,
    normalize_gateway_error=normalize_gateway_error,
    normalize_commit=normalize_commit,
    normalize_lineage=normalize_lineage,
    normalize_diff=normalize_diff,
    repo_audit_payload=repo_audit_payload,
)


@dataclass(frozen=True)
class LineageEvidence:
    candidate_commit: str
    accepted_ancestor: str = ""
    similar_failure_count: int = 0
    discussion_open: int = 0


@dataclass(frozen=True)
class TriageRecommendation:
    action: str
    reason: str


def recommend_triage_action(*, status: str, evidence: LineageEvidence) -> TriageRecommendation:
    if status in {"failed", "rejected"} and evidence.similar_failure_count >= 2:
        return TriageRecommendation(action="replay", reason="similar lineage failures detected")
    if status == "needs-approval" and evidence.accepted_ancestor:
        return TriageRecommendation(action="approve", reason="accepted ancestor exists")
    if evidence.discussion_open > 0:
        return TriageRecommendation(action="handoff", reason="open repo discussion requires reviewer")
    return TriageRecommendation(action="retry", reason="default retry path")


def approval_evidence_packet(*, run_id: str, links: List[Dict[str, str]], lineage_summary: str) -> Dict[str, object]:
    accepted = next((link.get("commit_hash", "") for link in links if link.get("role") == "accepted"), "")
    candidate = next((link.get("commit_hash", "") for link in links if link.get("role") == "candidate"), "")
    return {
        "run_id": run_id,
        "accepted_commit_hash": accepted,
        "candidate_commit_hash": candidate,
        "lineage_summary": lineage_summary,
        "links": links,
    }


_install_support_module(
    "repo_triage",
    LineageEvidence=LineageEvidence,
    TriageRecommendation=TriageRecommendation,
    recommend_triage_action=recommend_triage_action,
    approval_evidence_packet=approval_evidence_packet,
)


def collaboration_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


@dataclass
class CollaborationComment:
    comment_id: str
    author: str
    body: str
    created_at: str = field(default_factory=collaboration_now)
    mentions: List[str] = field(default_factory=list)
    anchor: str = ""
    status: str = "open"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "comment_id": self.comment_id,
            "author": self.author,
            "body": self.body,
            "created_at": self.created_at,
            "mentions": self.mentions,
            "anchor": self.anchor,
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CollaborationComment":
        return cls(
            comment_id=str(data.get("comment_id", "")),
            author=str(data.get("author", "")),
            body=str(data.get("body", "")),
            created_at=str(data.get("created_at", collaboration_now())),
            mentions=[str(value) for value in data.get("mentions", [])],
            anchor=str(data.get("anchor", "")),
            status=str(data.get("status", "open")),
        )


@dataclass
class DecisionNote:
    decision_id: str
    author: str
    outcome: str
    summary: str
    recorded_at: str = field(default_factory=collaboration_now)
    mentions: List[str] = field(default_factory=list)
    related_comment_ids: List[str] = field(default_factory=list)
    follow_up: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "decision_id": self.decision_id,
            "author": self.author,
            "outcome": self.outcome,
            "summary": self.summary,
            "recorded_at": self.recorded_at,
            "mentions": self.mentions,
            "related_comment_ids": self.related_comment_ids,
            "follow_up": self.follow_up,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "DecisionNote":
        return cls(
            decision_id=str(data.get("decision_id", "")),
            author=str(data.get("author", "")),
            outcome=str(data.get("outcome", "")),
            summary=str(data.get("summary", "")),
            recorded_at=str(data.get("recorded_at", collaboration_now())),
            mentions=[str(value) for value in data.get("mentions", [])],
            related_comment_ids=[str(value) for value in data.get("related_comment_ids", [])],
            follow_up=str(data.get("follow_up", "")),
        )


@dataclass
class CollaborationThread:
    surface: str
    target_id: str
    comments: List[CollaborationComment] = field(default_factory=list)
    decisions: List[DecisionNote] = field(default_factory=list)

    @property
    def participant_count(self) -> int:
        participants = {comment.author for comment in self.comments} | {
            decision.author for decision in self.decisions
        }
        return len({value for value in participants if value})

    @property
    def mention_count(self) -> int:
        return sum(len(comment.mentions) for comment in self.comments) + sum(
            len(decision.mentions) for decision in self.decisions
        )

    @property
    def open_comment_count(self) -> int:
        return sum(1 for comment in self.comments if comment.status != "resolved")

    @property
    def recommendation(self) -> str:
        if self.decisions:
            return "share-latest-decision"
        if self.open_comment_count:
            return "resolve-open-comments"
        if self.comments:
            return "monitor-collaboration"
        return "no-collaboration-recorded"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "surface": self.surface,
            "target_id": self.target_id,
            "comments": [comment.to_dict() for comment in self.comments],
            "decisions": [decision.to_dict() for decision in self.decisions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CollaborationThread":
        return cls(
            surface=str(data.get("surface", "")),
            target_id=str(data.get("target_id", "")),
            comments=[CollaborationComment.from_dict(value) for value in data.get("comments", [])],
            decisions=[DecisionNote.from_dict(value) for value in data.get("decisions", [])],
        )


def build_collaboration_thread(
    surface: str,
    target_id: str,
    *,
    comments: Optional[Sequence[CollaborationComment]] = None,
    decisions: Optional[Sequence[DecisionNote]] = None,
) -> CollaborationThread:
    return CollaborationThread(
        surface=surface,
        target_id=target_id,
        comments=list(comments or []),
        decisions=list(decisions or []),
    )


def merge_collaboration_threads(
    *,
    target_id: str,
    native_thread: Optional[CollaborationThread],
    repo_thread: Optional[CollaborationThread],
) -> Optional[CollaborationThread]:
    if native_thread is None and repo_thread is None:
        return None

    merged_comments: List[CollaborationComment] = []
    merged_decisions: List[DecisionNote] = []
    for thread in [native_thread, repo_thread]:
        if thread is None:
            continue
        merged_comments.extend(thread.comments)
        merged_decisions.extend(thread.decisions)

    merged_comments.sort(key=lambda item: item.created_at)
    merged_decisions.sort(key=lambda item: item.recorded_at)
    return CollaborationThread(
        surface="merged",
        target_id=target_id,
        comments=merged_comments,
        decisions=merged_decisions,
    )


def build_collaboration_thread_from_audits(
    audits: Sequence[Dict[str, Any]],
    *,
    surface: str,
    target_id: str,
) -> Optional[CollaborationThread]:
    comments: List[CollaborationComment] = []
    decisions: List[DecisionNote] = []

    for audit in audits:
        details = audit.get("details", {})
        audit_surface = str(details.get("surface", "run"))
        if audit_surface != surface:
            continue

        action = audit.get("action")
        if action == "collaboration.comment":
            comments.append(
                CollaborationComment(
                    comment_id=str(details.get("comment_id", "")),
                    author=str(audit.get("actor", "")),
                    body=str(details.get("body", "")),
                    created_at=str(audit.get("timestamp", collaboration_now())),
                    mentions=[str(value) for value in details.get("mentions", [])],
                    anchor=str(details.get("anchor", "")),
                    status=str(details.get("status", "open")),
                )
            )
        if action == "collaboration.decision":
            decisions.append(
                DecisionNote(
                    decision_id=str(details.get("decision_id", "")),
                    author=str(audit.get("actor", "")),
                    outcome=str(audit.get("outcome", "")),
                    summary=str(details.get("summary", "")),
                    recorded_at=str(audit.get("timestamp", collaboration_now())),
                    mentions=[str(value) for value in details.get("mentions", [])],
                    related_comment_ids=[str(value) for value in details.get("related_comment_ids", [])],
                    follow_up=str(details.get("follow_up", "")),
                )
            )

    if not comments and not decisions:
        return None
    return CollaborationThread(surface=surface, target_id=target_id, comments=comments, decisions=decisions)


def render_collaboration_lines(thread: Optional[CollaborationThread]) -> List[str]:
    if thread is None:
        return []

    lines = [
        "## Collaboration",
        "",
        f"- Surface: {thread.surface}",
        f"- Target: {thread.target_id}",
        f"- Participants: {thread.participant_count}",
        f"- Comments: {len(thread.comments)}",
        f"- Open Comments: {thread.open_comment_count}",
        f"- Mentions: {thread.mention_count}",
        f"- Decision Notes: {len(thread.decisions)}",
        f"- Recommendation: {thread.recommendation}",
    ]
    lines.extend(["", "## Comments", ""])
    if thread.comments:
        for comment in thread.comments:
            mentions = ", ".join(comment.mentions) if comment.mentions else "none"
            lines.append(
                f"- {comment.comment_id}: author={comment.author} status={comment.status} "
                f"anchor={comment.anchor or 'none'} mentions={mentions} body={comment.body}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Decision Notes", ""])
    if thread.decisions:
        for decision in thread.decisions:
            mentions = ", ".join(decision.mentions) if decision.mentions else "none"
            related = ", ".join(decision.related_comment_ids) if decision.related_comment_ids else "none"
            lines.append(
                f"- {decision.decision_id}: outcome={decision.outcome} author={decision.author} "
                f"mentions={mentions} related={related} summary={decision.summary} follow_up={decision.follow_up or 'none'}"
            )
    else:
        lines.append("- None")

    lines.append("")
    return lines


def render_collaboration_panel_html(title: str, description: str, thread: Optional[CollaborationThread]) -> str:
    if thread is None:
        return f"""
        <section class="surface">
          <h2>{escape(title)}</h2>
          <p>{escape(description)}</p>
          <div class="empty">No collaboration recorded for this surface.</div>
        </section>
        """

    comment_items = "".join(
        f"""
        <article class="resource-card">
          <span class="kicker">{escape(comment.comment_id)}</span>
          <strong>{escape(comment.author)}</strong>
          <p>{escape(comment.body)}</p>
          <span class="resource-meta">status={escape(comment.status)} | anchor={escape(comment.anchor or 'none')} | mentions={escape(', '.join(comment.mentions) if comment.mentions else 'none')}</span>
        </article>
        """
        for comment in thread.comments
    ) or '<div class="empty">No comments recorded.</div>'
    decision_items = "".join(
        f"""
        <article class="resource-card" data-tone="report">
          <span class="kicker">{escape(decision.decision_id)}</span>
          <strong>{escape(decision.outcome)}</strong>
          <p>{escape(decision.summary)}</p>
          <span class="resource-meta">author={escape(decision.author)} | mentions={escape(', '.join(decision.mentions) if decision.mentions else 'none')} | related={escape(', '.join(decision.related_comment_ids) if decision.related_comment_ids else 'none')} | follow_up={escape(decision.follow_up or 'none')}</span>
        </article>
        """
        for decision in thread.decisions
    ) or '<div class="empty">No decision notes recorded.</div>'

    return f"""
    <section class="surface">
      <h2>{escape(title)}</h2>
      <p>{escape(description)}</p>
      <p class="meta">participants={thread.participant_count} | comments={len(thread.comments)} | open={thread.open_comment_count} | mentions={thread.mention_count} | decisions={len(thread.decisions)}</p>
    </section>
    <section class="surface">
      <h3>Comments</h3>
      <div class="resource-grid">{comment_items}</div>
    </section>
    <section class="surface">
      <h3>Decision Notes</h3>
      <div class="resource-grid">{decision_items}</div>
    </section>
    """


_install_support_module(
    "collaboration",
    collaboration_now=collaboration_now,
    CollaborationComment=CollaborationComment,
    DecisionNote=DecisionNote,
    CollaborationThread=CollaborationThread,
    build_collaboration_thread=build_collaboration_thread,
    merge_collaboration_threads=merge_collaboration_threads,
    build_collaboration_thread_from_audits=build_collaboration_thread_from_audits,
    render_collaboration_lines=render_collaboration_lines,
    render_collaboration_panel_html=render_collaboration_panel_html,
)


@dataclass
class RepoSpace:
    space_id: str
    project_key: str
    repo: str
    default_branch: str = "main"
    sidecar_url: str = ""
    sidecar_enabled: bool = True
    health_state: str = "unknown"
    default_channel_strategy: str = "task"
    metadata: Dict[str, Any] = field(default_factory=dict)

    def default_channel_for_task(self, task_id: str) -> str:
        normalized = "".join(ch.lower() if ch.isalnum() else "-" for ch in task_id).strip("-")
        normalized = "-".join(part for part in normalized.split("-") if part)
        return f"{self.project_key.lower()}-{normalized}"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "space_id": self.space_id,
            "project_key": self.project_key,
            "repo": self.repo,
            "default_branch": self.default_branch,
            "sidecar_url": self.sidecar_url,
            "sidecar_enabled": self.sidecar_enabled,
            "health_state": self.health_state,
            "default_channel_strategy": self.default_channel_strategy,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoSpace":
        return cls(
            space_id=str(data["space_id"]),
            project_key=str(data["project_key"]),
            repo=str(data["repo"]),
            default_branch=str(data.get("default_branch", "main")),
            sidecar_url=str(data.get("sidecar_url", "")),
            sidecar_enabled=bool(data.get("sidecar_enabled", True)),
            health_state=str(data.get("health_state", "unknown")),
            default_channel_strategy=str(data.get("default_channel_strategy", "task")),
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class RepoAgent:
    actor: str
    repo_agent_id: str
    display_name: str = ""
    roles: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "actor": self.actor,
            "repo_agent_id": self.repo_agent_id,
            "display_name": self.display_name,
            "roles": list(self.roles),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoAgent":
        return cls(
            actor=str(data["actor"]),
            repo_agent_id=str(data["repo_agent_id"]),
            display_name=str(data.get("display_name", "")),
            roles=[str(item) for item in data.get("roles", [])],
        )


@dataclass
class RunCommitLink:
    run_id: str
    commit_hash: str
    role: str
    repo_space_id: str
    actor: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "run_id": self.run_id,
            "commit_hash": self.commit_hash,
            "role": self.role,
            "repo_space_id": self.repo_space_id,
            "actor": self.actor,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RunCommitLink":
        return cls(
            run_id=str(data["run_id"]),
            commit_hash=str(data["commit_hash"]),
            role=str(data["role"]),
            repo_space_id=str(data["repo_space_id"]),
            actor=str(data.get("actor", "")),
            metadata=dict(data.get("metadata", {})),
        )


_install_support_module(
    "repo_plane",
    RepoSpace=RepoSpace,
    RepoAgent=RepoAgent,
    RunCommitLink=RunCommitLink,
)

VALID_RUN_COMMIT_ROLES = {"source", "candidate", "closeout", "accepted"}


@dataclass
class RunCommitBinding:
    links: List[RunCommitLink]

    @property
    def accepted_commit_hash(self) -> str:
        for link in self.links:
            if link.role == "accepted":
                return link.commit_hash
        return ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "links": [link.to_dict() for link in self.links],
            "accepted_commit_hash": self.accepted_commit_hash,
        }


def validate_run_commit_roles(links: Iterable[RunCommitLink]) -> None:
    invalid = [link.role for link in links if link.role not in VALID_RUN_COMMIT_ROLES]
    if invalid:
        invalid_text = ", ".join(sorted(set(invalid)))
        raise ValueError(f"unsupported run commit roles: {invalid_text}")


def bind_run_commits(links: List[RunCommitLink]) -> RunCommitBinding:
    validate_run_commit_roles(links)
    return RunCommitBinding(links=list(links))


_install_support_module(
    "repo_links",
    VALID_ROLES=VALID_RUN_COMMIT_ROLES,
    RunCommitBinding=RunCommitBinding,
    validate_roles=validate_run_commit_roles,
    bind_run_commits=bind_run_commits,
)


def _repo_board_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


@dataclass
class RepoPost:
    post_id: str
    channel: str
    author: str
    body: str
    target_surface: str = "task"
    target_id: str = ""
    parent_post_id: str = ""
    created_at: str = field(default_factory=_repo_board_now)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "post_id": self.post_id,
            "channel": self.channel,
            "author": self.author,
            "body": self.body,
            "target_surface": self.target_surface,
            "target_id": self.target_id,
            "parent_post_id": self.parent_post_id,
            "created_at": self.created_at,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoPost":
        return cls(
            post_id=str(data.get("post_id", "")),
            channel=str(data.get("channel", "")),
            author=str(data.get("author", "")),
            body=str(data.get("body", "")),
            target_surface=str(data.get("target_surface", "task")),
            target_id=str(data.get("target_id", "")),
            parent_post_id=str(data.get("parent_post_id", "")),
            created_at=str(data.get("created_at", _repo_board_now())),
            metadata=dict(data.get("metadata", {})),
        )

    def to_collaboration_comment(self) -> CollaborationComment:
        return CollaborationComment(
            comment_id=f"repo-{self.post_id}",
            author=self.author,
            body=self.body,
            created_at=self.created_at,
            anchor=f"{self.target_surface}:{self.target_id}",
            status="resolved" if self.metadata.get("resolved") else "open",
        )


@dataclass
class RepoDiscussionBoard:
    posts: List[RepoPost] = field(default_factory=list)

    def create_post(
        self,
        *,
        channel: str,
        author: str,
        body: str,
        target_surface: str,
        target_id: str,
        metadata: Dict[str, Any] | None = None,
    ) -> RepoPost:
        post = RepoPost(
            post_id=f"post-{len(self.posts) + 1}",
            channel=channel,
            author=author,
            body=body,
            target_surface=target_surface,
            target_id=target_id,
            metadata=dict(metadata or {}),
        )
        self.posts.append(post)
        return post

    def reply(self, *, parent_post_id: str, author: str, body: str) -> RepoPost:
        parent = next((post for post in self.posts if post.post_id == parent_post_id), None)
        if not parent:
            raise ValueError(f"unknown parent post: {parent_post_id}")
        post = RepoPost(
            post_id=f"post-{len(self.posts) + 1}",
            channel=parent.channel,
            author=author,
            body=body,
            target_surface=parent.target_surface,
            target_id=parent.target_id,
            parent_post_id=parent_post_id,
        )
        self.posts.append(post)
        return post

    def list_posts(self, *, channel: str = "", target_surface: str = "", target_id: str = "") -> List[RepoPost]:
        result = self.posts
        if channel:
            result = [post for post in result if post.channel == channel]
        if target_surface:
            result = [post for post in result if post.target_surface == target_surface]
        if target_id:
            result = [post for post in result if post.target_id == target_id]
        return list(result)


_install_support_module(
    "repo_board",
    RepoPost=RepoPost,
    RepoDiscussionBoard=RepoDiscussionBoard,
)


def _repo_registry_slug(value: str) -> str:
    cleaned = "".join(ch.lower() if ch.isalnum() else "-" for ch in value)
    return "-".join(part for part in cleaned.split("-") if part) or "agent"


@dataclass
class RepoRegistry:
    spaces_by_project: Dict[str, RepoSpace] = field(default_factory=dict)
    agents_by_actor: Dict[str, RepoAgent] = field(default_factory=dict)

    def register_space(self, space: RepoSpace) -> None:
        self.spaces_by_project[space.project_key] = space

    def resolve_space(self, project_key: str) -> Optional[RepoSpace]:
        return self.spaces_by_project.get(project_key)

    def resolve_default_channel(self, project_key: str, task: Task) -> str:
        space = self.resolve_space(project_key)
        if not space:
            return f"{project_key.lower()}-{_repo_registry_slug(task.task_id)}"
        return space.default_channel_for_task(task.task_id)

    def resolve_agent(self, actor: str, role: str = "executor") -> RepoAgent:
        if actor in self.agents_by_actor:
            return self.agents_by_actor[actor]
        agent = RepoAgent(
            actor=actor,
            repo_agent_id=f"agent-{_repo_registry_slug(actor)}",
            display_name=actor,
            roles=[role],
        )
        self.agents_by_actor[actor] = agent
        return agent

    def to_dict(self) -> Dict[str, object]:
        return {
            "spaces_by_project": {key: value.to_dict() for key, value in self.spaces_by_project.items()},
            "agents_by_actor": {key: value.to_dict() for key, value in self.agents_by_actor.items()},
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "RepoRegistry":
        registry = cls()
        for key, value in dict(data.get("spaces_by_project", {})).items():
            registry.spaces_by_project[str(key)] = RepoSpace.from_dict(dict(value))
        for key, value in dict(data.get("agents_by_actor", {})).items():
            registry.agents_by_actor[str(key)] = RepoAgent.from_dict(dict(value))
        return registry


_install_support_module(
    "repo_registry",
    RepoRegistry=RepoRegistry,
)


def _install_legacy_surface_module(name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__.update(extra_attrs)

    def __getattr__(attr_name: str) -> object:
        if attr_name in extra_attrs:
            return extra_attrs[attr_name]
        if attr_name not in export_names:
            raise AttributeError(f"module {module.__name__!r} has no attribute {attr_name!r}")
        runtime_module = importlib.import_module(f"{__name__}.runtime")
        value = getattr(runtime_module, attr_name)
        module.__dict__[attr_name] = value
        return value

    module.__dict__["__getattr__"] = __getattr__
    sys.modules[module.__name__] = module
    globals()[name] = module


_install_legacy_surface_module(
    "queue",
    ["DeadLetterEntry", "PersistentTaskQueue"],
    LEGACY_MAINLINE_STATUS="bigclaw-go is the sole implementation mainline for active development.",
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/queue/queue.go",
)
_install_legacy_surface_module(
    "orchestration",
    [
        "CrossDepartmentOrchestrator",
        "DepartmentHandoff",
        "HandoffRequest",
        "OrchestrationPlan",
        "OrchestrationPolicyDecision",
        "PremiumOrchestrationPolicy",
        "render_orchestration_plan",
    ],
    LEGACY_MAINLINE_STATUS="bigclaw-go is the sole implementation mainline for active development.",
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/orchestration.go",
)
_install_legacy_surface_module(
    "scheduler",
    ["ExecutionRecord", "Scheduler", "SchedulerDecision"],
    LEGACY_MAINLINE_STATUS="bigclaw-go is the sole implementation mainline for active development.",
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/scheduler/scheduler.go",
)
_install_legacy_surface_module(
    "workflow",
    ["AcceptanceDecision", "AcceptanceGate", "JournalEntry", "WorkflowEngine", "WorkflowRunResult", "WorkpadJournal"],
    LEGACY_MAINLINE_STATUS="bigclaw-go is the sole implementation mainline for active development.",
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/engine.go",
)
_install_legacy_surface_module(
    "service",
    [
        "RepoGovernanceEnforcer",
        "RepoGovernancePolicy",
        "RepoGovernanceResult",
        "ServerMonitoring",
        "create_server",
        "run_server",
        "warn_legacy_service_surface",
    ],
    LEGACY_MAINLINE_STATUS=(
        "bigclaw-go is the sole implementation mainline for active development; "
        "service.py remains migration-only compatibility scaffolding."
    ),
    GO_MAINLINE_REPLACEMENT="bigclaw-go/cmd/bigclawd/main.go",
)

FOUNDATION_CATEGORIES = ("color", "spacing", "typography", "motion", "radius")
COMPONENT_READINESS_ORDER = {"draft": 0, "alpha": 1, "beta": 2, "stable": 3}
REQUIRED_INTERACTION_STATES = {"default", "hover", "disabled"}


@dataclass(frozen=True)
class DesignToken:
    name: str
    category: str
    value: str
    semantic_role: str = ""
    theme: str = "core"

    def to_dict(self) -> Dict[str, str]:
        return {
            "name": self.name,
            "category": self.category,
            "value": self.value,
            "semantic_role": self.semantic_role,
            "theme": self.theme,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "DesignToken":
        return cls(
            name=data["name"],
            category=data["category"],
            value=data["value"],
            semantic_role=data.get("semantic_role", ""),
            theme=data.get("theme", "core"),
        )


@dataclass
class ComponentVariant:
    name: str
    tokens: List[str] = field(default_factory=list)
    states: List[str] = field(default_factory=list)
    usage_notes: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "tokens": list(self.tokens),
            "states": list(self.states),
            "usage_notes": self.usage_notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ComponentVariant":
        return cls(
            name=str(data["name"]),
            tokens=[str(token) for token in data.get("tokens", [])],
            states=[str(state) for state in data.get("states", [])],
            usage_notes=str(data.get("usage_notes", "")),
        )


@dataclass
class ComponentSpec:
    name: str
    readiness: str = "draft"
    slots: List[str] = field(default_factory=list)
    variants: List[ComponentVariant] = field(default_factory=list)
    accessibility_requirements: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def token_names(self) -> List[str]:
        names: List[str] = []
        for variant in self.variants:
            for token in variant.tokens:
                if token not in names:
                    names.append(token)
        return names

    @property
    def state_coverage(self) -> List[str]:
        coverage: List[str] = []
        for variant in self.variants:
            for state in variant.states:
                if state not in coverage:
                    coverage.append(state)
        return coverage

    @property
    def missing_required_states(self) -> List[str]:
        return sorted(REQUIRED_INTERACTION_STATES.difference(self.state_coverage))

    @property
    def release_ready(self) -> bool:
        return (
            COMPONENT_READINESS_ORDER.get(self.readiness, -1) >= COMPONENT_READINESS_ORDER["beta"]
            and self.documentation_complete
            and bool(self.accessibility_requirements)
            and not self.missing_required_states
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "readiness": self.readiness,
            "slots": list(self.slots),
            "variants": [variant.to_dict() for variant in self.variants],
            "accessibility_requirements": list(self.accessibility_requirements),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ComponentSpec":
        return cls(
            name=str(data["name"]),
            readiness=str(data.get("readiness", "draft")),
            slots=[str(slot) for slot in data.get("slots", [])],
            variants=[ComponentVariant.from_dict(variant) for variant in data.get("variants", [])],
            accessibility_requirements=[str(requirement) for requirement in data.get("accessibility_requirements", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class DesignSystemAudit:
    system_name: str
    version: str
    token_counts: Dict[str, int]
    component_count: int
    release_ready_components: List[str] = field(default_factory=list)
    components_missing_docs: List[str] = field(default_factory=list)
    components_missing_accessibility: List[str] = field(default_factory=list)
    components_missing_states: List[str] = field(default_factory=list)
    undefined_token_refs: Dict[str, List[str]] = field(default_factory=dict)
    token_orphans: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.component_count == 0:
            return 0.0
        ready = len(self.release_ready_components)
        penalties = len(self.components_missing_docs) + len(self.components_missing_accessibility) + len(self.components_missing_states)
        score = max(0.0, ((ready * 100) - (penalties * 10)) / self.component_count)
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "system_name": self.system_name,
            "version": self.version,
            "token_counts": dict(self.token_counts),
            "component_count": self.component_count,
            "release_ready_components": list(self.release_ready_components),
            "components_missing_docs": list(self.components_missing_docs),
            "components_missing_accessibility": list(self.components_missing_accessibility),
            "components_missing_states": list(self.components_missing_states),
            "undefined_token_refs": {name: list(tokens) for name, tokens in self.undefined_token_refs.items()},
            "token_orphans": list(self.token_orphans),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DesignSystemAudit":
        return cls(
            system_name=str(data["system_name"]),
            version=str(data["version"]),
            token_counts={str(name): int(count) for name, count in dict(data.get("token_counts", {})).items()},
            component_count=int(data.get("component_count", 0)),
            release_ready_components=[str(name) for name in data.get("release_ready_components", [])],
            components_missing_docs=[str(name) for name in data.get("components_missing_docs", [])],
            components_missing_accessibility=[str(name) for name in data.get("components_missing_accessibility", [])],
            components_missing_states=[str(name) for name in data.get("components_missing_states", [])],
            undefined_token_refs={str(name): [str(token) for token in tokens] for name, tokens in dict(data.get("undefined_token_refs", {})).items()},
            token_orphans=[str(token) for token in data.get("token_orphans", [])],
        )


def _normalize_route_path(path: str) -> str:
    stripped = path.strip("/")
    return f"/{stripped}" if stripped else "/"


@dataclass(frozen=True)
class NavigationRoute:
    path: str
    screen_id: str
    title: str
    nav_node_id: str = ""
    layout: str = "workspace"

    def __post_init__(self) -> None:
        object.__setattr__(self, "path", _normalize_route_path(self.path))

    def to_dict(self) -> Dict[str, str]:
        return {
            "path": self.path,
            "screen_id": self.screen_id,
            "title": self.title,
            "nav_node_id": self.nav_node_id,
            "layout": self.layout,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "NavigationRoute":
        return cls(
            path=data["path"],
            screen_id=data["screen_id"],
            title=data["title"],
            nav_node_id=data.get("nav_node_id", ""),
            layout=data.get("layout", "workspace"),
        )


@dataclass
class NavigationNode:
    node_id: str
    title: str
    segment: str
    screen_id: str = ""
    children: List["NavigationNode"] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "node_id": self.node_id,
            "title": self.title,
            "segment": self.segment,
            "screen_id": self.screen_id,
            "children": [child.to_dict() for child in self.children],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "NavigationNode":
        return cls(
            node_id=str(data["node_id"]),
            title=str(data["title"]),
            segment=str(data.get("segment", "")),
            screen_id=str(data.get("screen_id", "")),
            children=[cls.from_dict(child) for child in data.get("children", [])],
        )


@dataclass(frozen=True)
class NavigationEntry:
    node_id: str
    title: str
    path: str
    depth: int
    parent_id: str = ""
    screen_id: str = ""


@dataclass
class InformationArchitectureAudit:
    total_navigation_nodes: int
    total_routes: int
    duplicate_routes: List[str] = field(default_factory=list)
    missing_route_nodes: Dict[str, str] = field(default_factory=dict)
    secondary_nav_gaps: Dict[str, List[str]] = field(default_factory=dict)
    orphan_routes: List[str] = field(default_factory=list)

    @property
    def healthy(self) -> bool:
        return not (self.duplicate_routes or self.missing_route_nodes or self.secondary_nav_gaps or self.orphan_routes)

    def to_dict(self) -> Dict[str, object]:
        return {
            "total_navigation_nodes": self.total_navigation_nodes,
            "total_routes": self.total_routes,
            "duplicate_routes": list(self.duplicate_routes),
            "missing_route_nodes": dict(self.missing_route_nodes),
            "secondary_nav_gaps": {section: list(paths) for section, paths in self.secondary_nav_gaps.items()},
            "orphan_routes": list(self.orphan_routes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitectureAudit":
        return cls(
            total_navigation_nodes=int(data.get("total_navigation_nodes", 0)),
            total_routes=int(data.get("total_routes", 0)),
            duplicate_routes=[str(path) for path in data.get("duplicate_routes", [])],
            missing_route_nodes={str(node_id): str(path) for node_id, path in dict(data.get("missing_route_nodes", {})).items()},
            secondary_nav_gaps={str(section): [str(path) for path in paths] for section, paths in dict(data.get("secondary_nav_gaps", {})).items()},
            orphan_routes=[str(path) for path in data.get("orphan_routes", [])],
        )


@dataclass
class InformationArchitecture:
    global_nav: List[NavigationNode] = field(default_factory=list)
    routes: List[NavigationRoute] = field(default_factory=list)

    @property
    def route_index(self) -> Dict[str, NavigationRoute]:
        index: Dict[str, NavigationRoute] = {}
        for route in self.routes:
            if route.path not in index:
                index[route.path] = route
        return index

    @property
    def navigation_entries(self) -> List[NavigationEntry]:
        entries: List[NavigationEntry] = []
        for node in self.global_nav:
            entries.extend(self._flatten_node(node=node, parent_path="", depth=0, parent_id=""))
        return entries

    def resolve_route(self, path: str) -> Optional[NavigationRoute]:
        return self.route_index.get(_normalize_route_path(path))

    def audit(self) -> InformationArchitectureAudit:
        entries = self.navigation_entries
        route_counts: Dict[str, int] = {}
        for route in self.routes:
            route_counts[route.path] = route_counts.get(route.path, 0) + 1
        duplicate_routes = sorted(path for path, count in route_counts.items() if count > 1)
        route_index = self.route_index
        missing_route_nodes = {entry.node_id: entry.path for entry in entries if entry.path not in route_index}
        secondary_nav_gaps: Dict[str, List[str]] = {}
        for root in self.global_nav:
            gaps = sorted(self._missing_paths_for_descendants(root, parent_path=""))
            if gaps:
                secondary_nav_gaps[root.title] = gaps
        nav_paths = {entry.path for entry in entries}
        orphan_routes = sorted(route.path for route in self.routes if route.path not in nav_paths)
        return InformationArchitectureAudit(
            total_navigation_nodes=len(entries),
            total_routes=len(self.routes),
            duplicate_routes=duplicate_routes,
            missing_route_nodes=missing_route_nodes,
            secondary_nav_gaps=secondary_nav_gaps,
            orphan_routes=orphan_routes,
        )

    def to_dict(self) -> Dict[str, object]:
        return {"global_nav": [node.to_dict() for node in self.global_nav], "routes": [route.to_dict() for route in self.routes]}

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitecture":
        return cls(
            global_nav=[NavigationNode.from_dict(node) for node in data.get("global_nav", [])],
            routes=[NavigationRoute.from_dict(route) for route in data.get("routes", [])],
        )

    def _flatten_node(self, node: NavigationNode, parent_path: str, depth: int, parent_id: str) -> List[NavigationEntry]:
        path = self._join_path(parent_path, node.segment)
        entries = [NavigationEntry(node_id=node.node_id, title=node.title, path=path, depth=depth, parent_id=parent_id, screen_id=node.screen_id)]
        for child in node.children:
            entries.extend(self._flatten_node(child, parent_path=path, depth=depth + 1, parent_id=node.node_id))
        return entries

    def _missing_paths_for_descendants(self, node: NavigationNode, parent_path: str) -> List[str]:
        path = self._join_path(parent_path, node.segment)
        missing: List[str] = []
        if node.children and path not in self.route_index:
            missing.append(path)
        for child in node.children:
            missing.extend(self._missing_paths_for_descendants(child, parent_path=path))
        return missing

    @staticmethod
    def _join_path(parent_path: str, segment: str) -> str:
        base = _normalize_route_path(parent_path)
        part = segment.strip("/")
        if not part:
            return base
        if base == "/":
            return f"/{part}"
        return f"{base}/{part}"


@dataclass
class CommandAction:
    id: str
    title: str
    section: str
    shortcut: str = ""

    def to_dict(self) -> Dict[str, str]:
        return {"id": self.id, "title": self.title, "section": self.section, "shortcut": self.shortcut}

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "CommandAction":
        return cls(id=data["id"], title=data["title"], section=data["section"], shortcut=data.get("shortcut", ""))


@dataclass
class ConsoleCommandEntry:
    trigger_label: str
    placeholder: str
    shortcut: str
    commands: List[CommandAction] = field(default_factory=list)
    recent_queries_enabled: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "trigger_label": self.trigger_label,
            "placeholder": self.placeholder,
            "shortcut": self.shortcut,
            "commands": [command.to_dict() for command in self.commands],
            "recent_queries_enabled": self.recent_queries_enabled,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleCommandEntry":
        return cls(
            trigger_label=str(data["trigger_label"]),
            placeholder=str(data["placeholder"]),
            shortcut=str(data["shortcut"]),
            commands=[CommandAction.from_dict(command) for command in data.get("commands", [])],
            recent_queries_enabled=bool(data.get("recent_queries_enabled", False)),
        )


@dataclass
class ConsoleTopBar:
    name: str
    search_placeholder: str
    environment_options: List[str] = field(default_factory=list)
    time_range_options: List[str] = field(default_factory=list)
    alert_channels: List[str] = field(default_factory=list)
    command_entry: ConsoleCommandEntry = field(default_factory=lambda: ConsoleCommandEntry(trigger_label="", placeholder="", shortcut=""))
    documentation_complete: bool = False
    accessibility_requirements: List[str] = field(default_factory=list)

    @property
    def has_global_search(self) -> bool:
        return bool(self.search_placeholder.strip())

    @property
    def has_environment_switch(self) -> bool:
        return len(self.environment_options) >= 2

    @property
    def has_time_range_switch(self) -> bool:
        return len(self.time_range_options) >= 2

    @property
    def has_alert_entry(self) -> bool:
        return bool(self.alert_channels)

    @property
    def has_command_shell(self) -> bool:
        return bool(self.command_entry.trigger_label.strip()) and bool(self.command_entry.commands)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "search_placeholder": self.search_placeholder,
            "environment_options": list(self.environment_options),
            "time_range_options": list(self.time_range_options),
            "alert_channels": list(self.alert_channels),
            "command_entry": self.command_entry.to_dict(),
            "documentation_complete": self.documentation_complete,
            "accessibility_requirements": list(self.accessibility_requirements),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleTopBar":
        return cls(
            name=str(data["name"]),
            search_placeholder=str(data.get("search_placeholder", "")),
            environment_options=[str(option) for option in data.get("environment_options", [])],
            time_range_options=[str(option) for option in data.get("time_range_options", [])],
            alert_channels=[str(channel) for channel in data.get("alert_channels", [])],
            command_entry=ConsoleCommandEntry.from_dict(dict(data.get("command_entry", {}))),
            documentation_complete=bool(data.get("documentation_complete", False)),
            accessibility_requirements=[str(requirement) for requirement in data.get("accessibility_requirements", [])],
        )


@dataclass
class ConsoleTopBarAudit:
    name: str
    missing_capabilities: List[str] = field(default_factory=list)
    documentation_complete: bool = False
    accessibility_complete: bool = False
    command_shortcut_supported: bool = False
    command_count: int = 0

    @property
    def release_ready(self) -> bool:
        return not self.missing_capabilities and self.documentation_complete and self.accessibility_complete and self.command_shortcut_supported

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "missing_capabilities": list(self.missing_capabilities),
            "documentation_complete": self.documentation_complete,
            "accessibility_complete": self.accessibility_complete,
            "command_shortcut_supported": self.command_shortcut_supported,
            "command_count": self.command_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleTopBarAudit":
        return cls(
            name=str(data["name"]),
            missing_capabilities=[str(item) for item in data.get("missing_capabilities", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
            accessibility_complete=bool(data.get("accessibility_complete", False)),
            command_shortcut_supported=bool(data.get("command_shortcut_supported", False)),
            command_count=int(data.get("command_count", 0)),
        )


@dataclass
class DesignSystem:
    name: str
    version: str
    tokens: List[DesignToken] = field(default_factory=list)
    components: List[ComponentSpec] = field(default_factory=list)

    @property
    def token_counts(self) -> Dict[str, int]:
        counts = {category: 0 for category in FOUNDATION_CATEGORIES}
        for token in self.tokens:
            counts[token.category] = counts.get(token.category, 0) + 1
        return counts

    @property
    def token_index(self) -> Dict[str, DesignToken]:
        return {token.name: token for token in self.tokens}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "tokens": [token.to_dict() for token in self.tokens],
            "components": [component.to_dict() for component in self.components],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DesignSystem":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            tokens=[DesignToken.from_dict(token) for token in data.get("tokens", [])],
            components=[ComponentSpec.from_dict(component) for component in data.get("components", [])],
        )


class ComponentLibrary:
    def audit(self, system: DesignSystem) -> DesignSystemAudit:
        used_tokens = set()
        release_ready_components: List[str] = []
        components_missing_docs: List[str] = []
        components_missing_accessibility: List[str] = []
        components_missing_states: List[str] = []
        undefined_token_refs: Dict[str, List[str]] = {}
        token_index = system.token_index
        for component in system.components:
            used_tokens.update(component.token_names)
            missing_tokens = sorted(token for token in component.token_names if token not in token_index)
            if missing_tokens:
                undefined_token_refs[component.name] = missing_tokens
            if component.release_ready and not missing_tokens:
                release_ready_components.append(component.name)
            if not component.documentation_complete:
                components_missing_docs.append(component.name)
            if not component.accessibility_requirements:
                components_missing_accessibility.append(component.name)
            if component.missing_required_states:
                components_missing_states.append(component.name)
        token_orphans = sorted(token.name for token in system.tokens if token.name not in used_tokens)
        return DesignSystemAudit(
            system_name=system.name,
            version=system.version,
            token_counts=system.token_counts,
            component_count=len(system.components),
            release_ready_components=sorted(release_ready_components),
            components_missing_docs=sorted(components_missing_docs),
            components_missing_accessibility=sorted(components_missing_accessibility),
            components_missing_states=sorted(components_missing_states),
            undefined_token_refs=undefined_token_refs,
            token_orphans=token_orphans,
        )


class ConsoleChromeLibrary:
    REQUIRED_SHORTCUTS = {"cmd+k", "ctrl+k"}
    REQUIRED_ACCESSIBILITY = {"keyboard-navigation", "screen-reader-label", "focus-visible"}

    def audit_top_bar(self, top_bar: ConsoleTopBar) -> ConsoleTopBarAudit:
        missing_capabilities: List[str] = []
        if not top_bar.has_global_search:
            missing_capabilities.append("global-search")
        if not top_bar.has_time_range_switch:
            missing_capabilities.append("time-range-switch")
        if not top_bar.has_environment_switch:
            missing_capabilities.append("environment-switch")
        if not top_bar.has_alert_entry:
            missing_capabilities.append("alert-entry")
        if not top_bar.has_command_shell:
            missing_capabilities.append("command-shell")
        normalized_shortcuts = {item.strip().lower().replace(" ", "") for item in top_bar.command_entry.shortcut.split("/") if item.strip()}
        accessibility_complete = self.REQUIRED_ACCESSIBILITY.issubset(set(top_bar.accessibility_requirements))
        return ConsoleTopBarAudit(
            name=top_bar.name,
            missing_capabilities=missing_capabilities,
            documentation_complete=top_bar.documentation_complete,
            accessibility_complete=accessibility_complete,
            command_shortcut_supported=self.REQUIRED_SHORTCUTS.issubset(normalized_shortcuts),
            command_count=len(top_bar.command_entry.commands),
        )


def render_design_system_report(system: DesignSystem, audit: DesignSystemAudit) -> str:
    lines = [
        "# Design System Report",
        "",
        f"- Name: {system.name}",
        f"- Version: {system.version}",
        f"- Components: {audit.component_count}",
        f"- Release Ready Components: {len(audit.release_ready_components)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Token Foundations",
        "",
    ]
    for category, count in audit.token_counts.items():
        lines.append(f"- {category}: {count}")
    lines.extend(["", "## Component Status", ""])
    if system.components:
        for component in system.components:
            states = ", ".join(component.state_coverage) or "none"
            missing_states = ", ".join(component.missing_required_states) or "none"
            undefined_tokens = ", ".join(audit.undefined_token_refs.get(component.name, [])) or "none"
            lines.append(
                f"- {component.name}: readiness={component.readiness} docs={component.documentation_complete} "
                f"a11y={bool(component.accessibility_requirements)} states={states} missing_states={missing_states} "
                f"undefined_tokens={undefined_tokens}"
            )
    else:
        lines.append("- None")
    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Missing docs: {', '.join(audit.components_missing_docs) if audit.components_missing_docs else 'none'}")
    lines.append(f"- Missing accessibility: {', '.join(audit.components_missing_accessibility) if audit.components_missing_accessibility else 'none'}")
    lines.append(f"- Missing interaction states: {', '.join(audit.components_missing_states) if audit.components_missing_states else 'none'}")
    undefined_refs = "; ".join(f"{component}={', '.join(tokens)}" for component, tokens in sorted(audit.undefined_token_refs.items())) if audit.undefined_token_refs else "none"
    lines.append(f"- Undefined token refs: {undefined_refs}")
    lines.append(f"- Orphan tokens: {', '.join(audit.token_orphans) if audit.token_orphans else 'none'}")
    return "\n".join(lines) + "\n"


def render_console_top_bar_report(top_bar: ConsoleTopBar, audit: ConsoleTopBarAudit) -> str:
    lines = [
        "# Console Top Bar Report",
        "",
        f"- Name: {top_bar.name}",
        f"- Global Search: {top_bar.has_global_search}",
        f"- Environment Switch: {', '.join(top_bar.environment_options) if top_bar.environment_options else 'none'}",
        f"- Time Range Switch: {', '.join(top_bar.time_range_options) if top_bar.time_range_options else 'none'}",
        f"- Alert Entry: {', '.join(top_bar.alert_channels) if top_bar.alert_channels else 'none'}",
        f"- Command Trigger: {top_bar.command_entry.trigger_label or 'none'}",
        f"- Command Shortcut: {top_bar.command_entry.shortcut or 'none'}",
        f"- Command Count: {audit.command_count}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Command Palette",
        "",
    ]
    if top_bar.command_entry.commands:
        for command in top_bar.command_entry.commands:
            shortcut = command.shortcut or "none"
            lines.append(f"- {command.id}: {command.title} [{command.section}] shortcut={shortcut}")
    else:
        lines.append("- None")
    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Missing capabilities: {', '.join(audit.missing_capabilities) if audit.missing_capabilities else 'none'}")
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    lines.append(f"- Accessibility complete: {audit.accessibility_complete}")
    lines.append(f"- Cmd/Ctrl+K supported: {audit.command_shortcut_supported}")
    return "\n".join(lines) + "\n"


def render_information_architecture_report(architecture: InformationArchitecture, audit: InformationArchitectureAudit) -> str:
    lines = [
        "# Information Architecture Report",
        "",
        f"- Navigation Nodes: {audit.total_navigation_nodes}",
        f"- Routes: {audit.total_routes}",
        f"- Healthy: {audit.healthy}",
        "",
        "## Navigation Tree",
        "",
    ]
    if architecture.navigation_entries:
        for entry in architecture.navigation_entries:
            indent = "  " * entry.depth
            lines.append(f"- {indent}{entry.title} ({entry.path}) screen={entry.screen_id or 'none'}")
    else:
        lines.append("- None")
    lines.extend(["", "## Route Registry", ""])
    if architecture.routes:
        for route in architecture.routes:
            lines.append(f"- {route.path}: screen={route.screen_id} title={route.title} nav_node={route.nav_node_id or 'none'}")
    else:
        lines.append("- None")
    lines.extend(["", "## Audit", ""])
    lines.append(f"- Duplicate routes: {', '.join(audit.duplicate_routes) if audit.duplicate_routes else 'none'}")
    missing = ", ".join(f"{node_id}={path}" for node_id, path in sorted(audit.missing_route_nodes.items())) if audit.missing_route_nodes else "none"
    lines.append(f"- Missing route nodes: {missing}")
    gaps = "; ".join(f"{section}={', '.join(paths)}" for section, paths in sorted(audit.secondary_nav_gaps.items())) if audit.secondary_nav_gaps else "none"
    lines.append(f"- Secondary nav gaps: {gaps}")
    lines.append(f"- Orphan routes: {', '.join(audit.orphan_routes) if audit.orphan_routes else 'none'}")
    return "\n".join(lines) + "\n"


@dataclass(frozen=True)
class RolePermissionScenario:
    screen_id: str
    allowed_roles: List[str] = field(default_factory=list)
    denied_roles: List[str] = field(default_factory=list)
    audit_event: str = ""

    @property
    def missing_coverage(self) -> List[str]:
        missing: List[str] = []
        if not self.allowed_roles:
            missing.append("allowed-roles")
        if not self.denied_roles:
            missing.append("denied-roles")
        if not self.audit_event.strip():
            missing.append("audit-event")
        return missing

    def to_dict(self) -> Dict[str, object]:
        return {
            "screen_id": self.screen_id,
            "allowed_roles": list(self.allowed_roles),
            "denied_roles": list(self.denied_roles),
            "audit_event": self.audit_event,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "RolePermissionScenario":
        return cls(
            screen_id=str(data["screen_id"]),
            allowed_roles=[str(role) for role in data.get("allowed_roles", [])],
            denied_roles=[str(role) for role in data.get("denied_roles", [])],
            audit_event=str(data.get("audit_event", "")),
        )


@dataclass(frozen=True)
class DataAccuracyCheck:
    screen_id: str
    metric_id: str
    source_of_truth: str
    rendered_value: str
    tolerance: float = 0.0
    observed_delta: float = 0.0
    freshness_slo_seconds: int = 0
    observed_freshness_seconds: int = 0

    @property
    def within_tolerance(self) -> bool:
        return abs(self.observed_delta) <= self.tolerance

    @property
    def within_freshness_slo(self) -> bool:
        if self.freshness_slo_seconds <= 0:
            return True
        return self.observed_freshness_seconds <= self.freshness_slo_seconds

    @property
    def passes(self) -> bool:
        return self.within_tolerance and self.within_freshness_slo

    def to_dict(self) -> Dict[str, object]:
        return {
            "screen_id": self.screen_id,
            "metric_id": self.metric_id,
            "source_of_truth": self.source_of_truth,
            "rendered_value": self.rendered_value,
            "tolerance": self.tolerance,
            "observed_delta": self.observed_delta,
            "freshness_slo_seconds": self.freshness_slo_seconds,
            "observed_freshness_seconds": self.observed_freshness_seconds,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DataAccuracyCheck":
        return cls(
            screen_id=str(data["screen_id"]),
            metric_id=str(data["metric_id"]),
            source_of_truth=str(data.get("source_of_truth", "")),
            rendered_value=str(data.get("rendered_value", "")),
            tolerance=float(data.get("tolerance", 0.0)),
            observed_delta=float(data.get("observed_delta", 0.0)),
            freshness_slo_seconds=int(data.get("freshness_slo_seconds", 0)),
            observed_freshness_seconds=int(data.get("observed_freshness_seconds", 0)),
        )


@dataclass(frozen=True)
class PerformanceBudget:
    surface_id: str
    interaction: str
    target_p95_ms: int
    observed_p95_ms: int
    target_tti_ms: int = 0
    observed_tti_ms: int = 0

    @property
    def within_budget(self) -> bool:
        p95_ok = self.observed_p95_ms <= self.target_p95_ms
        tti_ok = self.target_tti_ms <= 0 or self.observed_tti_ms <= self.target_tti_ms
        return p95_ok and tti_ok

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_id": self.surface_id,
            "interaction": self.interaction,
            "target_p95_ms": self.target_p95_ms,
            "observed_p95_ms": self.observed_p95_ms,
            "target_tti_ms": self.target_tti_ms,
            "observed_tti_ms": self.observed_tti_ms,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "PerformanceBudget":
        return cls(
            surface_id=str(data["surface_id"]),
            interaction=str(data["interaction"]),
            target_p95_ms=int(data.get("target_p95_ms", 0)),
            observed_p95_ms=int(data.get("observed_p95_ms", 0)),
            target_tti_ms=int(data.get("target_tti_ms", 0)),
            observed_tti_ms=int(data.get("observed_tti_ms", 0)),
        )


@dataclass(frozen=True)
class UsabilityJourney:
    journey_id: str
    personas: List[str] = field(default_factory=list)
    critical_steps: List[str] = field(default_factory=list)
    expected_max_steps: int = 0
    observed_steps: int = 0
    keyboard_accessible: bool = False
    empty_state_guidance: bool = False
    recovery_support: bool = False

    @property
    def passes(self) -> bool:
        return bool(self.personas) and bool(self.critical_steps) and self.expected_max_steps > 0 and self.observed_steps <= self.expected_max_steps and self.keyboard_accessible and self.empty_state_guidance and self.recovery_support

    def to_dict(self) -> Dict[str, object]:
        return {
            "journey_id": self.journey_id,
            "personas": list(self.personas),
            "critical_steps": list(self.critical_steps),
            "expected_max_steps": self.expected_max_steps,
            "observed_steps": self.observed_steps,
            "keyboard_accessible": self.keyboard_accessible,
            "empty_state_guidance": self.empty_state_guidance,
            "recovery_support": self.recovery_support,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UsabilityJourney":
        return cls(
            journey_id=str(data["journey_id"]),
            personas=[str(persona) for persona in data.get("personas", [])],
            critical_steps=[str(step) for step in data.get("critical_steps", [])],
            expected_max_steps=int(data.get("expected_max_steps", 0)),
            observed_steps=int(data.get("observed_steps", 0)),
            keyboard_accessible=bool(data.get("keyboard_accessible", False)),
            empty_state_guidance=bool(data.get("empty_state_guidance", False)),
            recovery_support=bool(data.get("recovery_support", False)),
        )


@dataclass(frozen=True)
class AuditRequirement:
    event_type: str
    required_fields: List[str] = field(default_factory=list)
    emitted_fields: List[str] = field(default_factory=list)
    retention_days: int = 0
    observed_retention_days: int = 0

    @property
    def missing_fields(self) -> List[str]:
        emitted = set(self.emitted_fields)
        return sorted(field for field in self.required_fields if field not in emitted)

    @property
    def retention_met(self) -> bool:
        if self.retention_days <= 0:
            return True
        return self.observed_retention_days >= self.retention_days

    @property
    def complete(self) -> bool:
        return not self.missing_fields and self.retention_met

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_type": self.event_type,
            "required_fields": list(self.required_fields),
            "emitted_fields": list(self.emitted_fields),
            "retention_days": self.retention_days,
            "observed_retention_days": self.observed_retention_days,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AuditRequirement":
        return cls(
            event_type=str(data["event_type"]),
            required_fields=[str(field_name) for field_name in data.get("required_fields", [])],
            emitted_fields=[str(field_name) for field_name in data.get("emitted_fields", [])],
            retention_days=int(data.get("retention_days", 0)),
            observed_retention_days=int(data.get("observed_retention_days", 0)),
        )


@dataclass
class UIAcceptanceSuite:
    name: str
    version: str
    role_permissions: List[RolePermissionScenario] = field(default_factory=list)
    data_accuracy_checks: List[DataAccuracyCheck] = field(default_factory=list)
    performance_budgets: List[PerformanceBudget] = field(default_factory=list)
    usability_journeys: List[UsabilityJourney] = field(default_factory=list)
    audit_requirements: List[AuditRequirement] = field(default_factory=list)
    documentation_complete: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "role_permissions": [scenario.to_dict() for scenario in self.role_permissions],
            "data_accuracy_checks": [check.to_dict() for check in self.data_accuracy_checks],
            "performance_budgets": [budget.to_dict() for budget in self.performance_budgets],
            "usability_journeys": [journey.to_dict() for journey in self.usability_journeys],
            "audit_requirements": [requirement.to_dict() for requirement in self.audit_requirements],
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIAcceptanceSuite":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            role_permissions=[RolePermissionScenario.from_dict(scenario) for scenario in data.get("role_permissions", [])],
            data_accuracy_checks=[DataAccuracyCheck.from_dict(check) for check in data.get("data_accuracy_checks", [])],
            performance_budgets=[PerformanceBudget.from_dict(budget) for budget in data.get("performance_budgets", [])],
            usability_journeys=[UsabilityJourney.from_dict(journey) for journey in data.get("usability_journeys", [])],
            audit_requirements=[AuditRequirement.from_dict(requirement) for requirement in data.get("audit_requirements", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class UIAcceptanceAudit:
    name: str
    version: str
    permission_gaps: List[str] = field(default_factory=list)
    failing_data_checks: List[str] = field(default_factory=list)
    failing_performance_budgets: List[str] = field(default_factory=list)
    failing_usability_journeys: List[str] = field(default_factory=list)
    incomplete_audit_trails: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def release_ready(self) -> bool:
        return not self.permission_gaps and not self.failing_data_checks and not self.failing_performance_budgets and not self.failing_usability_journeys and not self.incomplete_audit_trails and self.documentation_complete

    @property
    def readiness_score(self) -> float:
        checks = [not self.permission_gaps, not self.failing_data_checks, not self.failing_performance_budgets, not self.failing_usability_journeys, not self.incomplete_audit_trails, self.documentation_complete]
        passed = sum(1 for item in checks if item)
        return round((passed / len(checks)) * 100, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "permission_gaps": list(self.permission_gaps),
            "failing_data_checks": list(self.failing_data_checks),
            "failing_performance_budgets": list(self.failing_performance_budgets),
            "failing_usability_journeys": list(self.failing_usability_journeys),
            "incomplete_audit_trails": list(self.incomplete_audit_trails),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIAcceptanceAudit":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            permission_gaps=[str(item) for item in data.get("permission_gaps", [])],
            failing_data_checks=[str(item) for item in data.get("failing_data_checks", [])],
            failing_performance_budgets=[str(item) for item in data.get("failing_performance_budgets", [])],
            failing_usability_journeys=[str(item) for item in data.get("failing_usability_journeys", [])],
            incomplete_audit_trails=[str(item) for item in data.get("incomplete_audit_trails", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


class UIAcceptanceLibrary:
    def audit(self, suite: UIAcceptanceSuite) -> UIAcceptanceAudit:
        permission_gaps = [f"{scenario.screen_id}: missing={', '.join(scenario.missing_coverage)}" for scenario in suite.role_permissions if scenario.missing_coverage]
        failing_data_checks = [f"{check.screen_id}.{check.metric_id}: delta={check.observed_delta} freshness={check.observed_freshness_seconds}s" for check in suite.data_accuracy_checks if not check.passes]
        failing_performance_budgets = [
            f"{budget.surface_id}.{budget.interaction}: p95={budget.observed_p95_ms}ms" + (f" tti={budget.observed_tti_ms}ms" if budget.target_tti_ms > 0 else "")
            for budget in suite.performance_budgets
            if not budget.within_budget
        ]
        failing_usability_journeys = [f"{journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps}" for journey in suite.usability_journeys if not journey.passes]
        incomplete_audit_trails = []
        for requirement in suite.audit_requirements:
            if requirement.complete:
                continue
            gaps = requirement.missing_fields
            parts: List[str] = []
            if gaps:
                parts.append(f"missing_fields={', '.join(gaps)}")
            if not requirement.retention_met:
                parts.append(f"retention={requirement.observed_retention_days}/{requirement.retention_days}d")
            incomplete_audit_trails.append(f"{requirement.event_type}: {' '.join(parts)}")
        return UIAcceptanceAudit(
            name=suite.name,
            version=suite.version,
            permission_gaps=permission_gaps,
            failing_data_checks=failing_data_checks,
            failing_performance_budgets=failing_performance_budgets,
            failing_usability_journeys=failing_usability_journeys,
            incomplete_audit_trails=incomplete_audit_trails,
            documentation_complete=suite.documentation_complete,
        )


def render_ui_acceptance_report(suite: UIAcceptanceSuite, audit: UIAcceptanceAudit) -> str:
    lines = [
        "# UI Acceptance Report",
        "",
        f"- Name: {suite.name}",
        f"- Version: {suite.version}",
        f"- Role/Permission Scenarios: {len(suite.role_permissions)}",
        f"- Data Accuracy Checks: {len(suite.data_accuracy_checks)}",
        f"- Performance Budgets: {len(suite.performance_budgets)}",
        f"- Usability Journeys: {len(suite.usability_journeys)}",
        f"- Audit Requirements: {len(suite.audit_requirements)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Coverage",
        "",
    ]
    if suite.role_permissions:
        for scenario in suite.role_permissions:
            denied = ", ".join(scenario.denied_roles) or "none"
            lines.append(f"- Role/Permission {scenario.screen_id}: allow={', '.join(scenario.allowed_roles) or 'none'} deny={denied} audit_event={scenario.audit_event or 'none'}")
    else:
        lines.append("- Role/Permission: none")
    if suite.data_accuracy_checks:
        for check in suite.data_accuracy_checks:
            lines.append(f"- Data Accuracy {check.screen_id}.{check.metric_id}: delta={check.observed_delta} tolerance={check.tolerance} freshness={check.observed_freshness_seconds}/{check.freshness_slo_seconds}s")
    else:
        lines.append("- Data Accuracy: none")
    if suite.performance_budgets:
        for budget in suite.performance_budgets:
            tti_text = f" tti={budget.observed_tti_ms}/{budget.target_tti_ms}ms" if budget.target_tti_ms > 0 else ""
            lines.append(f"- Performance {budget.surface_id}.{budget.interaction}: p95={budget.observed_p95_ms}/{budget.target_p95_ms}ms{tti_text}")
    else:
        lines.append("- Performance: none")
    if suite.usability_journeys:
        for journey in suite.usability_journeys:
            lines.append(f"- Usability {journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps} keyboard={journey.keyboard_accessible} empty_state={journey.empty_state_guidance} recovery={journey.recovery_support}")
    else:
        lines.append("- Usability: none")
    if suite.audit_requirements:
        for requirement in suite.audit_requirements:
            lines.append(f"- Audit {requirement.event_type}: fields={len(requirement.emitted_fields)}/{len(requirement.required_fields)} retention={requirement.observed_retention_days}/{requirement.retention_days}d")
    else:
        lines.append("- Audit: none")
    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Role/Permission gaps: {', '.join(audit.permission_gaps) if audit.permission_gaps else 'none'}")
    lines.append(f"- Data accuracy failures: {', '.join(audit.failing_data_checks) if audit.failing_data_checks else 'none'}")
    lines.append(f"- Performance budget failures: {', '.join(audit.failing_performance_budgets) if audit.failing_performance_budgets else 'none'}")
    lines.append(f"- Usability journey failures: {', '.join(audit.failing_usability_journeys) if audit.failing_usability_journeys else 'none'}")
    lines.append(f"- Audit completeness gaps: {', '.join(audit.incomplete_audit_trails) if audit.incomplete_audit_trails else 'none'}")
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    return "\n".join(lines) + "\n"


_install_support_module(
    "design_system",
    FOUNDATION_CATEGORIES=FOUNDATION_CATEGORIES,
    COMPONENT_READINESS_ORDER=COMPONENT_READINESS_ORDER,
    REQUIRED_INTERACTION_STATES=REQUIRED_INTERACTION_STATES,
    DesignToken=DesignToken,
    ComponentVariant=ComponentVariant,
    ComponentSpec=ComponentSpec,
    DesignSystemAudit=DesignSystemAudit,
    NavigationRoute=NavigationRoute,
    NavigationNode=NavigationNode,
    NavigationEntry=NavigationEntry,
    InformationArchitectureAudit=InformationArchitectureAudit,
    InformationArchitecture=InformationArchitecture,
    CommandAction=CommandAction,
    ConsoleCommandEntry=ConsoleCommandEntry,
    ConsoleTopBar=ConsoleTopBar,
    ConsoleTopBarAudit=ConsoleTopBarAudit,
    DesignSystem=DesignSystem,
    ComponentLibrary=ComponentLibrary,
    ConsoleChromeLibrary=ConsoleChromeLibrary,
    render_design_system_report=render_design_system_report,
    render_console_top_bar_report=render_console_top_bar_report,
    render_information_architecture_report=render_information_architecture_report,
    RolePermissionScenario=RolePermissionScenario,
    DataAccuracyCheck=DataAccuracyCheck,
    PerformanceBudget=PerformanceBudget,
    UsabilityJourney=UsabilityJourney,
    AuditRequirement=AuditRequirement,
    UIAcceptanceSuite=UIAcceptanceSuite,
    UIAcceptanceAudit=UIAcceptanceAudit,
    UIAcceptanceLibrary=UIAcceptanceLibrary,
    render_ui_acceptance_report=render_ui_acceptance_report,
)

REQUIRED_SURFACE_STATES = {"default", "loading", "empty", "error"}


@dataclass(frozen=True)
class NavigationItem:
    name: str
    route: str
    section: str
    icon: str = ""
    badge_count: int = 0

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "route": self.route,
            "section": self.section,
            "icon": self.icon,
            "badge_count": self.badge_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "NavigationItem":
        return cls(
            name=str(data["name"]),
            route=str(data["route"]),
            section=str(data["section"]),
            icon=str(data.get("icon", "")),
            badge_count=int(data.get("badge_count", 0)),
        )


@dataclass(frozen=True)
class GlobalAction:
    action_id: str
    label: str
    placement: str
    requires_selection: bool = False
    intent: str = "default"

    def to_dict(self) -> Dict[str, object]:
        return {
            "action_id": self.action_id,
            "label": self.label,
            "placement": self.placement,
            "requires_selection": self.requires_selection,
            "intent": self.intent,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "GlobalAction":
        return cls(
            action_id=str(data["action_id"]),
            label=str(data["label"]),
            placement=str(data["placement"]),
            requires_selection=bool(data.get("requires_selection", False)),
            intent=str(data.get("intent", "default")),
        )


@dataclass(frozen=True)
class FilterDefinition:
    name: str
    field: str
    control: str
    options: List[str] = field(default_factory=list)
    default_value: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field": self.field,
            "control": self.control,
            "options": list(self.options),
            "default_value": self.default_value,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "FilterDefinition":
        return cls(
            name=str(data["name"]),
            field=str(data["field"]),
            control=str(data["control"]),
            options=[str(option) for option in data.get("options", [])],
            default_value=str(data.get("default_value", "")),
        )


@dataclass(frozen=True)
class SurfaceState:
    name: str
    message: str = ""
    allowed_actions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "message": self.message,
            "allowed_actions": list(self.allowed_actions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceState":
        return cls(
            name=str(data["name"]),
            message=str(data.get("message", "")),
            allowed_actions=[str(action_id) for action_id in data.get("allowed_actions", [])],
        )


@dataclass
class ConsoleSurface:
    name: str
    route: str
    navigation_section: str
    top_bar_actions: List[GlobalAction] = field(default_factory=list)
    filters: List[FilterDefinition] = field(default_factory=list)
    states: List[SurfaceState] = field(default_factory=list)
    supports_bulk_actions: bool = False

    @property
    def action_ids(self) -> List[str]:
        return [action.action_id for action in self.top_bar_actions]

    @property
    def state_names(self) -> List[str]:
        return [state.name for state in self.states]

    @property
    def missing_required_states(self) -> List[str]:
        return sorted(REQUIRED_SURFACE_STATES.difference(self.state_names))

    @property
    def unresolved_state_actions(self) -> Dict[str, List[str]]:
        available = set(self.action_ids)
        unresolved: Dict[str, List[str]] = {}
        for state in self.states:
            missing = sorted(action_id for action_id in state.allowed_actions if action_id not in available)
            if missing:
                unresolved[state.name] = missing
        return unresolved

    @property
    def states_missing_actions(self) -> List[str]:
        missing: List[str] = []
        for state in self.states:
            if state.name != "default" and not state.allowed_actions:
                missing.append(state.name)
        return missing

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "route": self.route,
            "navigation_section": self.navigation_section,
            "top_bar_actions": [action.to_dict() for action in self.top_bar_actions],
            "filters": [surface_filter.to_dict() for surface_filter in self.filters],
            "states": [state.to_dict() for state in self.states],
            "supports_bulk_actions": self.supports_bulk_actions,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleSurface":
        return cls(
            name=str(data["name"]),
            route=str(data["route"]),
            navigation_section=str(data["navigation_section"]),
            top_bar_actions=[GlobalAction.from_dict(item) for item in data.get("top_bar_actions", [])],
            filters=[FilterDefinition.from_dict(item) for item in data.get("filters", [])],
            states=[SurfaceState.from_dict(item) for item in data.get("states", [])],
            supports_bulk_actions=bool(data.get("supports_bulk_actions", False)),
        )


@dataclass
class ConsoleIA:
    name: str
    version: str
    navigation: List[NavigationItem] = field(default_factory=list)
    surfaces: List[ConsoleSurface] = field(default_factory=list)
    top_bar: ConsoleTopBar = field(default_factory=lambda: ConsoleTopBar(name="", search_placeholder=""))

    @property
    def route_index(self) -> Dict[str, ConsoleSurface]:
        return {surface.route: surface for surface in self.surfaces}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "navigation": [item.to_dict() for item in self.navigation],
            "surfaces": [surface.to_dict() for surface in self.surfaces],
            "top_bar": self.top_bar.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleIA":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            navigation=[NavigationItem.from_dict(item) for item in data.get("navigation", [])],
            surfaces=[ConsoleSurface.from_dict(item) for item in data.get("surfaces", [])],
            top_bar=ConsoleTopBar.from_dict(dict(data.get("top_bar", {}))),
        )


@dataclass(frozen=True)
class SurfacePermissionRule:
    allowed_roles: List[str] = field(default_factory=list)
    denied_roles: List[str] = field(default_factory=list)
    audit_event: str = ""

    @property
    def missing_coverage(self) -> List[str]:
        missing: List[str] = []
        if not self.allowed_roles:
            missing.append("allowed-roles")
        if not self.denied_roles:
            missing.append("denied-roles")
        if not self.audit_event.strip():
            missing.append("audit-event")
        return missing

    @property
    def complete(self) -> bool:
        return not self.missing_coverage

    def to_dict(self) -> Dict[str, object]:
        return {
            "allowed_roles": list(self.allowed_roles),
            "denied_roles": list(self.denied_roles),
            "audit_event": self.audit_event,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfacePermissionRule":
        return cls(
            allowed_roles=[str(role) for role in data.get("allowed_roles", [])],
            denied_roles=[str(role) for role in data.get("denied_roles", [])],
            audit_event=str(data.get("audit_event", "")),
        )


@dataclass
class SurfaceInteractionContract:
    surface_name: str
    required_action_ids: List[str] = field(default_factory=list)
    requires_filters: bool = True
    requires_batch_actions: bool = False
    required_states: List[str] = field(default_factory=lambda: sorted(REQUIRED_SURFACE_STATES))
    permission_rule: SurfacePermissionRule = field(default_factory=SurfacePermissionRule)
    primary_persona: str = ""
    linked_wireframe_id: str = ""
    review_focus_areas: List[str] = field(default_factory=list)
    decision_prompts: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_name": self.surface_name,
            "required_action_ids": list(self.required_action_ids),
            "requires_filters": self.requires_filters,
            "requires_batch_actions": self.requires_batch_actions,
            "required_states": list(self.required_states),
            "permission_rule": self.permission_rule.to_dict(),
            "primary_persona": self.primary_persona,
            "linked_wireframe_id": self.linked_wireframe_id,
            "review_focus_areas": list(self.review_focus_areas),
            "decision_prompts": list(self.decision_prompts),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceInteractionContract":
        return cls(
            surface_name=str(data["surface_name"]),
            required_action_ids=[str(action_id) for action_id in data.get("required_action_ids", [])],
            requires_filters=bool(data.get("requires_filters", True)),
            requires_batch_actions=bool(data.get("requires_batch_actions", False)),
            required_states=[str(state) for state in data.get("required_states", sorted(REQUIRED_SURFACE_STATES))],
            permission_rule=SurfacePermissionRule.from_dict(dict(data.get("permission_rule", {}))),
            primary_persona=str(data.get("primary_persona", "")),
            linked_wireframe_id=str(data.get("linked_wireframe_id", "")),
            review_focus_areas=[str(item) for item in data.get("review_focus_areas", [])],
            decision_prompts=[str(item) for item in data.get("decision_prompts", [])],
        )


@dataclass
class ConsoleInteractionDraft:
    name: str
    version: str
    architecture: ConsoleIA
    contracts: List[SurfaceInteractionContract] = field(default_factory=list)
    required_roles: List[str] = field(default_factory=list)
    requires_frame_contracts: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "architecture": self.architecture.to_dict(),
            "contracts": [contract.to_dict() for contract in self.contracts],
            "required_roles": list(self.required_roles),
            "requires_frame_contracts": self.requires_frame_contracts,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleInteractionDraft":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            architecture=ConsoleIA.from_dict(dict(data["architecture"])),
            contracts=[SurfaceInteractionContract.from_dict(item) for item in data.get("contracts", [])],
            required_roles=[str(role) for role in data.get("required_roles", [])],
            requires_frame_contracts=bool(data.get("requires_frame_contracts", False)),
        )


@dataclass
class ConsoleInteractionAudit:
    name: str
    version: str
    contract_count: int
    missing_surfaces: List[str] = field(default_factory=list)
    surfaces_missing_filters: List[str] = field(default_factory=list)
    surfaces_missing_actions: Dict[str, List[str]] = field(default_factory=dict)
    surfaces_missing_batch_actions: List[str] = field(default_factory=list)
    surfaces_missing_states: Dict[str, List[str]] = field(default_factory=dict)
    permission_gaps: Dict[str, List[str]] = field(default_factory=dict)
    uncovered_roles: List[str] = field(default_factory=list)
    surfaces_missing_primary_personas: List[str] = field(default_factory=list)
    surfaces_missing_wireframe_links: List[str] = field(default_factory=list)
    surfaces_missing_review_focus: List[str] = field(default_factory=list)
    surfaces_missing_decision_prompts: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.contract_count == 0:
            return 0.0
        penalties = (
            len(self.missing_surfaces)
            + len(self.surfaces_missing_filters)
            + len(self.surfaces_missing_actions)
            + len(self.surfaces_missing_batch_actions)
            + len(self.surfaces_missing_states)
            + len(self.permission_gaps)
            + len(self.uncovered_roles)
            + len(self.surfaces_missing_primary_personas)
            + len(self.surfaces_missing_wireframe_links)
            + len(self.surfaces_missing_review_focus)
            + len(self.surfaces_missing_decision_prompts)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.contract_count))
        return round(score, 1)

    @property
    def release_ready(self) -> bool:
        return (
            not self.missing_surfaces
            and not self.surfaces_missing_filters
            and not self.surfaces_missing_actions
            and not self.surfaces_missing_batch_actions
            and not self.surfaces_missing_states
            and not self.permission_gaps
            and not self.uncovered_roles
            and not self.surfaces_missing_primary_personas
            and not self.surfaces_missing_wireframe_links
            and not self.surfaces_missing_review_focus
            and not self.surfaces_missing_decision_prompts
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "contract_count": self.contract_count,
            "missing_surfaces": list(self.missing_surfaces),
            "surfaces_missing_filters": list(self.surfaces_missing_filters),
            "surfaces_missing_actions": {name: list(actions) for name, actions in self.surfaces_missing_actions.items()},
            "surfaces_missing_batch_actions": list(self.surfaces_missing_batch_actions),
            "surfaces_missing_states": {name: list(states) for name, states in self.surfaces_missing_states.items()},
            "permission_gaps": {name: list(gaps) for name, gaps in self.permission_gaps.items()},
            "uncovered_roles": list(self.uncovered_roles),
            "surfaces_missing_primary_personas": list(self.surfaces_missing_primary_personas),
            "surfaces_missing_wireframe_links": list(self.surfaces_missing_wireframe_links),
            "surfaces_missing_review_focus": list(self.surfaces_missing_review_focus),
            "surfaces_missing_decision_prompts": list(self.surfaces_missing_decision_prompts),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleInteractionAudit":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            contract_count=int(data.get("contract_count", 0)),
            missing_surfaces=[str(name) for name in data.get("missing_surfaces", [])],
            surfaces_missing_filters=[str(name) for name in data.get("surfaces_missing_filters", [])],
            surfaces_missing_actions={
                str(name): [str(action_id) for action_id in actions]
                for name, actions in dict(data.get("surfaces_missing_actions", {})).items()
            },
            surfaces_missing_batch_actions=[str(name) for name in data.get("surfaces_missing_batch_actions", [])],
            surfaces_missing_states={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("surfaces_missing_states", {})).items()
            },
            permission_gaps={str(name): [str(gap) for gap in gaps] for name, gaps in dict(data.get("permission_gaps", {})).items()},
            uncovered_roles=[str(role) for role in data.get("uncovered_roles", [])],
            surfaces_missing_primary_personas=[str(name) for name in data.get("surfaces_missing_primary_personas", [])],
            surfaces_missing_wireframe_links=[str(name) for name in data.get("surfaces_missing_wireframe_links", [])],
            surfaces_missing_review_focus=[str(name) for name in data.get("surfaces_missing_review_focus", [])],
            surfaces_missing_decision_prompts=[str(name) for name in data.get("surfaces_missing_decision_prompts", [])],
        )


@dataclass
class ConsoleIAAudit:
    system_name: str
    version: str
    surface_count: int
    navigation_count: int
    top_bar_audit: ConsoleTopBarAudit = field(default_factory=lambda: ConsoleTopBarAudit(name=""))
    surfaces_missing_filters: List[str] = field(default_factory=list)
    surfaces_missing_actions: List[str] = field(default_factory=list)
    surfaces_missing_states: Dict[str, List[str]] = field(default_factory=dict)
    states_missing_actions: Dict[str, List[str]] = field(default_factory=dict)
    unresolved_state_actions: Dict[str, Dict[str, List[str]]] = field(default_factory=dict)
    orphan_navigation_routes: List[str] = field(default_factory=list)
    unnavigable_surfaces: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.surface_count == 0:
            return 0.0
        penalties = (
            (0 if self.top_bar_audit.release_ready else 1)
            + len(self.surfaces_missing_filters)
            + len(self.surfaces_missing_actions)
            + len(self.surfaces_missing_states)
            + len(self.states_missing_actions)
            + len(self.unresolved_state_actions)
            + len(self.orphan_navigation_routes)
            + len(self.unnavigable_surfaces)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.surface_count))
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "system_name": self.system_name,
            "version": self.version,
            "surface_count": self.surface_count,
            "navigation_count": self.navigation_count,
            "top_bar_audit": self.top_bar_audit.to_dict(),
            "surfaces_missing_filters": list(self.surfaces_missing_filters),
            "surfaces_missing_actions": list(self.surfaces_missing_actions),
            "surfaces_missing_states": {name: list(states) for name, states in self.surfaces_missing_states.items()},
            "states_missing_actions": {name: list(states) for name, states in self.states_missing_actions.items()},
            "unresolved_state_actions": {
                name: {state: list(actions) for state, actions in states.items()}
                for name, states in self.unresolved_state_actions.items()
            },
            "orphan_navigation_routes": list(self.orphan_navigation_routes),
            "unnavigable_surfaces": list(self.unnavigable_surfaces),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ConsoleIAAudit":
        return cls(
            system_name=str(data["system_name"]),
            version=str(data["version"]),
            surface_count=int(data.get("surface_count", 0)),
            navigation_count=int(data.get("navigation_count", 0)),
            top_bar_audit=ConsoleTopBarAudit.from_dict(dict(data.get("top_bar_audit", {}))),
            surfaces_missing_filters=[str(name) for name in data.get("surfaces_missing_filters", [])],
            surfaces_missing_actions=[str(name) for name in data.get("surfaces_missing_actions", [])],
            surfaces_missing_states={str(name): [str(state) for state in states] for name, states in dict(data.get("surfaces_missing_states", {})).items()},
            states_missing_actions={str(name): [str(state) for state in states] for name, states in dict(data.get("states_missing_actions", {})).items()},
            unresolved_state_actions={
                str(name): {str(state): [str(action_id) for action_id in actions] for state, actions in dict(states).items()}
                for name, states in dict(data.get("unresolved_state_actions", {})).items()
            },
            orphan_navigation_routes=[str(route) for route in data.get("orphan_navigation_routes", [])],
            unnavigable_surfaces=[str(name) for name in data.get("unnavigable_surfaces", [])],
        )


class ConsoleIAAuditor:
    def audit(self, architecture: ConsoleIA) -> ConsoleIAAudit:
        top_bar_audit = ConsoleChromeLibrary().audit_top_bar(architecture.top_bar)
        route_index = architecture.route_index
        navigation_routes = {item.route for item in architecture.navigation}
        surfaces_missing_filters: List[str] = []
        surfaces_missing_actions: List[str] = []
        surfaces_missing_states: Dict[str, List[str]] = {}
        states_missing_actions: Dict[str, List[str]] = {}
        unresolved_state_actions: Dict[str, Dict[str, List[str]]] = {}

        for surface in architecture.surfaces:
            if not surface.filters:
                surfaces_missing_filters.append(surface.name)
            if not surface.top_bar_actions:
                surfaces_missing_actions.append(surface.name)
            if surface.missing_required_states:
                surfaces_missing_states[surface.name] = surface.missing_required_states
            if surface.states_missing_actions:
                states_missing_actions[surface.name] = surface.states_missing_actions
            if surface.unresolved_state_actions:
                unresolved_state_actions[surface.name] = surface.unresolved_state_actions

        orphan_navigation_routes = sorted(route for route in navigation_routes if route not in route_index)
        unnavigable_surfaces = sorted(surface.name for surface in architecture.surfaces if surface.route not in navigation_routes)

        return ConsoleIAAudit(
            system_name=architecture.name,
            version=architecture.version,
            surface_count=len(architecture.surfaces),
            navigation_count=len(architecture.navigation),
            top_bar_audit=top_bar_audit,
            surfaces_missing_filters=sorted(surfaces_missing_filters),
            surfaces_missing_actions=sorted(surfaces_missing_actions),
            surfaces_missing_states=dict(sorted(surfaces_missing_states.items())),
            states_missing_actions=dict(sorted(states_missing_actions.items())),
            unresolved_state_actions=dict(sorted(unresolved_state_actions.items())),
            orphan_navigation_routes=orphan_navigation_routes,
            unnavigable_surfaces=unnavigable_surfaces,
        )


class ConsoleInteractionAuditor:
    def audit(self, draft: ConsoleInteractionDraft) -> ConsoleInteractionAudit:
        route_index = draft.architecture.route_index
        missing_surfaces: List[str] = []
        surfaces_missing_filters: List[str] = []
        surfaces_missing_actions: Dict[str, List[str]] = {}
        surfaces_missing_batch_actions: List[str] = []
        surfaces_missing_states: Dict[str, List[str]] = {}
        permission_gaps: Dict[str, List[str]] = {}
        referenced_roles = set()
        surfaces_missing_primary_personas: List[str] = []
        surfaces_missing_wireframe_links: List[str] = []
        surfaces_missing_review_focus: List[str] = []
        surfaces_missing_decision_prompts: List[str] = []

        for contract in draft.contracts:
            surface: Optional[ConsoleSurface] = route_index.get(contract.surface_name)
            if surface is None:
                surface = next((candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name), None)
            if surface is None:
                missing_surfaces.append(contract.surface_name)
                continue
            if contract.requires_filters and not surface.filters:
                surfaces_missing_filters.append(contract.surface_name)
            available_action_ids = set(surface.action_ids)
            missing_action_ids = sorted(action_id for action_id in contract.required_action_ids if action_id not in available_action_ids)
            if missing_action_ids:
                surfaces_missing_actions[contract.surface_name] = missing_action_ids
            if contract.requires_batch_actions and not any(action.requires_selection for action in surface.top_bar_actions):
                surfaces_missing_batch_actions.append(contract.surface_name)
            missing_state_ids = sorted(state_name for state_name in contract.required_states if state_name not in surface.state_names)
            if missing_state_ids:
                surfaces_missing_states[contract.surface_name] = missing_state_ids
            referenced_roles.update(contract.permission_rule.allowed_roles)
            referenced_roles.update(contract.permission_rule.denied_roles)
            if contract.permission_rule.missing_coverage:
                permission_gaps[contract.surface_name] = contract.permission_rule.missing_coverage
            if draft.requires_frame_contracts:
                if not contract.primary_persona.strip():
                    surfaces_missing_primary_personas.append(contract.surface_name)
                if not contract.linked_wireframe_id.strip():
                    surfaces_missing_wireframe_links.append(contract.surface_name)
                if not contract.review_focus_areas:
                    surfaces_missing_review_focus.append(contract.surface_name)
                if not contract.decision_prompts:
                    surfaces_missing_decision_prompts.append(contract.surface_name)

        uncovered_roles = sorted(role for role in draft.required_roles if role not in referenced_roles)

        return ConsoleInteractionAudit(
            name=draft.name,
            version=draft.version,
            contract_count=len(draft.contracts),
            missing_surfaces=sorted(missing_surfaces),
            surfaces_missing_filters=sorted(surfaces_missing_filters),
            surfaces_missing_actions=dict(sorted(surfaces_missing_actions.items())),
            surfaces_missing_batch_actions=sorted(surfaces_missing_batch_actions),
            surfaces_missing_states=dict(sorted(surfaces_missing_states.items())),
            permission_gaps=dict(sorted(permission_gaps.items())),
            uncovered_roles=uncovered_roles,
            surfaces_missing_primary_personas=sorted(surfaces_missing_primary_personas),
            surfaces_missing_wireframe_links=sorted(surfaces_missing_wireframe_links),
            surfaces_missing_review_focus=sorted(surfaces_missing_review_focus),
            surfaces_missing_decision_prompts=sorted(surfaces_missing_decision_prompts),
        )


def render_console_ia_report(architecture: ConsoleIA, audit: ConsoleIAAudit) -> str:
    lines = [
        "# Console Information Architecture Report",
        "",
        f"- Name: {architecture.name}",
        f"- Version: {architecture.version}",
        f"- Navigation Items: {audit.navigation_count}",
        f"- Surfaces: {audit.surface_count}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Global Header",
        "",
        f"- Name: {architecture.top_bar.name or 'none'}",
        f"- Release Ready: {audit.top_bar_audit.release_ready}",
        f"- Missing capabilities: {', '.join(audit.top_bar_audit.missing_capabilities) if audit.top_bar_audit.missing_capabilities else 'none'}",
        f"- Command Count: {audit.top_bar_audit.command_count}",
        f"- Cmd/Ctrl+K supported: {audit.top_bar_audit.command_shortcut_supported}",
        "",
        "## Navigation",
        "",
    ]
    if architecture.navigation:
        for item in architecture.navigation:
            lines.append(f"- {item.section} / {item.name}: route={item.route} badge={item.badge_count} icon={item.icon or 'none'}")
    else:
        lines.append("- None")
    lines.extend(["", "## Surface Coverage", ""])
    if architecture.surfaces:
        for surface in architecture.surfaces:
            filters = ", ".join(surface_filter.name for surface_filter in surface.filters) or "none"
            actions = ", ".join(action.label for action in surface.top_bar_actions) or "none"
            states = ", ".join(surface.state_names) or "none"
            missing_states = ", ".join(surface.missing_required_states) or "none"
            unresolved = audit.unresolved_state_actions.get(surface.name, {})
            unresolved_text = "; ".join(f"{state}={', '.join(action_ids)}" for state, action_ids in sorted(unresolved.items())) if unresolved else "none"
            state_actions_missing = ", ".join(audit.states_missing_actions.get(surface.name, [])) or "none"
            lines.append(
                f"- {surface.name}: route={surface.route} filters={filters} actions={actions} states={states} "
                f"missing_states={missing_states} states_without_actions={state_actions_missing} "
                f"unresolved_state_actions={unresolved_text}"
            )
    else:
        lines.append("- None")
    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Surfaces missing filters: {', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}")
    lines.append(f"- Surfaces missing top-bar actions: {', '.join(audit.surfaces_missing_actions) if audit.surfaces_missing_actions else 'none'}")
    missing_states_text = "; ".join(f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())) if audit.surfaces_missing_states else "none"
    lines.append(f"- Surfaces missing required states: {missing_states_text}")
    states_without_actions_text = "; ".join(f"{name}={', '.join(states)}" for name, states in sorted(audit.states_missing_actions.items())) if audit.states_missing_actions else "none"
    lines.append(f"- States without recovery actions: {states_without_actions_text}")
    unresolved_text = "; ".join(
        f"{name}=" + ", ".join(f"{state}:{'/'.join(actions)}" for state, actions in sorted(states.items()))
        for name, states in sorted(audit.unresolved_state_actions.items())
    ) if audit.unresolved_state_actions else "none"
    lines.append(f"- Undefined state actions: {unresolved_text}")
    lines.append(f"- Orphan navigation routes: {', '.join(audit.orphan_navigation_routes) if audit.orphan_navigation_routes else 'none'}")
    lines.append(f"- Unnavigable surfaces: {', '.join(audit.unnavigable_surfaces) if audit.unnavigable_surfaces else 'none'}")
    return "\n".join(lines) + "\n"


def render_console_interaction_report(draft: ConsoleInteractionDraft, audit: ConsoleInteractionAudit) -> str:
    route_index = draft.architecture.route_index
    lines = [
        "# Console Interaction Draft Report",
        "",
        f"- Name: {draft.name}",
        f"- Version: {draft.version}",
        f"- Critical Pages: {len(draft.contracts)}",
        f"- Required Roles: {', '.join(draft.required_roles) if draft.required_roles else 'none'}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Page Coverage",
        "",
    ]
    if draft.contracts:
        for contract in draft.contracts:
            surface = route_index.get(contract.surface_name)
            if surface is None:
                surface = next((candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name), None)
            if surface is None:
                lines.append(f"- {contract.surface_name}: missing surface definition")
                continue
            required_actions = ", ".join(contract.required_action_ids) or "none"
            available_actions = ", ".join(surface.action_ids) or "none"
            batch_mode = "required" if contract.requires_batch_actions else "optional"
            permission_state = "complete" if contract.permission_rule.complete else "incomplete"
            lines.append(
                f"- {contract.surface_name}: route={surface.route} required_actions={required_actions} "
                f"available_actions={available_actions} filters={len(surface.filters)} "
                f"states={', '.join(surface.state_names) or 'none'} batch={batch_mode} permissions={permission_state}"
            )
            lines.append(
                "  "
                f"persona={contract.primary_persona or 'none'} "
                f"wireframe={contract.linked_wireframe_id or 'none'} "
                f"review_focus={','.join(contract.review_focus_areas) or 'none'} "
                f"decision_prompts={','.join(contract.decision_prompts) or 'none'}"
            )
    else:
        lines.append("- None")
    lines.extend(["", "## Gaps", ""])
    lines.append(f"- Missing surfaces: {', '.join(audit.missing_surfaces) if audit.missing_surfaces else 'none'}")
    lines.append(f"- Pages missing filters: {', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}")
    action_gap_text = "; ".join(f"{name}={', '.join(actions)}" for name, actions in sorted(audit.surfaces_missing_actions.items())) if audit.surfaces_missing_actions else "none"
    lines.append(f"- Pages missing actions: {action_gap_text}")
    lines.append(f"- Pages missing batch actions: {', '.join(audit.surfaces_missing_batch_actions) if audit.surfaces_missing_batch_actions else 'none'}")
    state_gap_text = "; ".join(f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())) if audit.surfaces_missing_states else "none"
    lines.append(f"- Pages missing states: {state_gap_text}")
    permission_gap_text = "; ".join(f"{name}={', '.join(gaps)}" for name, gaps in sorted(audit.permission_gaps.items())) if audit.permission_gaps else "none"
    lines.append(f"- Permission gaps: {permission_gap_text}")
    lines.append(f"- Uncovered roles: {', '.join(audit.uncovered_roles) if audit.uncovered_roles else 'none'}")
    lines.append(f"- Pages missing personas: {', '.join(audit.surfaces_missing_primary_personas) if audit.surfaces_missing_primary_personas else 'none'}")
    lines.append(f"- Pages missing wireframe links: {', '.join(audit.surfaces_missing_wireframe_links) if audit.surfaces_missing_wireframe_links else 'none'}")
    lines.append(f"- Pages missing review focus: {', '.join(audit.surfaces_missing_review_focus) if audit.surfaces_missing_review_focus else 'none'}")
    lines.append(f"- Pages missing decision prompts: {', '.join(audit.surfaces_missing_decision_prompts) if audit.surfaces_missing_decision_prompts else 'none'}")
    return "\n".join(lines) + "\n"


def build_big_4203_console_interaction_draft() -> ConsoleInteractionDraft:
    return ConsoleInteractionDraft(
        name="BIG-4203 Four Critical Pages",
        version="v4.0-design-sprint",
        required_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
        requires_frame_contracts=True,
        architecture=ConsoleIA(
            name="BigClaw Console IA",
            version="v4.0-design-sprint",
            top_bar=ConsoleTopBar(
                name="BigClaw Global Header",
                search_placeholder="Search runs, queues, prompts, and commands",
                environment_options=["Production", "Staging", "Shadow"],
                time_range_options=["24h", "7d", "30d"],
                alert_channels=["approvals", "sla", "regressions"],
                documentation_complete=True,
                accessibility_requirements=["keyboard-navigation", "screen-reader-label", "focus-visible"],
                command_entry=ConsoleCommandEntry(
                    trigger_label="Command Menu",
                    placeholder="Jump to a run, queue, or release control action",
                    shortcut="Cmd+K / Ctrl+K",
                    commands=[
                        CommandAction(id="search-runs", title="Search runs", section="Navigate", shortcut="/"),
                        CommandAction(id="open-queue", title="Open queue control", section="Operate"),
                        CommandAction(id="open-triage", title="Open triage center", section="Operate"),
                    ],
                ),
            ),
            navigation=[
                NavigationItem(name="Overview", route="/overview", section="Operate", icon="dashboard"),
                NavigationItem(name="Queue", route="/queue", section="Operate", icon="queue"),
                NavigationItem(name="Run Detail", route="/runs/detail", section="Operate", icon="activity"),
                NavigationItem(name="Triage", route="/triage", section="Operate", icon="alert"),
            ],
            surfaces=[
                ConsoleSurface(
                    name="Overview",
                    route="/overview",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                    ],
                    filters=[
                        FilterDefinition(name="Team", field="team", control="select", options=["all", "platform", "product"]),
                        FilterDefinition(name="Time", field="time_range", control="segmented", options=["24h", "7d", "30d"], default_value="7d"),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["drill-down"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                ),
                ConsoleSurface(
                    name="Queue",
                    route="/queue",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                        GlobalAction(action_id="bulk-approve", label="Bulk Approve", placement="topbar", requires_selection=True),
                    ],
                    filters=[
                        FilterDefinition(name="Status", field="status", control="select", options=["all", "queued", "approval"]),
                        FilterDefinition(name="Owner", field="owner", control="search"),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                    supports_bulk_actions=True,
                ),
                ConsoleSurface(
                    name="Run Detail",
                    route="/runs/detail",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                    ],
                    filters=[
                        FilterDefinition(name="Run", field="run_id", control="search"),
                        FilterDefinition(name="Replay Mode", field="replay_mode", control="select", options=["latest", "failure-only"]),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["drill-down"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                ),
                ConsoleSurface(
                    name="Triage",
                    route="/triage",
                    navigation_section="Operate",
                    top_bar_actions=[
                        GlobalAction(action_id="drill-down", label="Drill Down", placement="topbar"),
                        GlobalAction(action_id="export", label="Export", placement="topbar"),
                        GlobalAction(action_id="audit", label="Audit Trail", placement="topbar"),
                        GlobalAction(action_id="bulk-assign", label="Bulk Assign", placement="topbar", requires_selection=True),
                    ],
                    filters=[
                        FilterDefinition(name="Severity", field="severity", control="select", options=["all", "high", "critical"]),
                        FilterDefinition(name="Workflow", field="workflow", control="select", options=["all", "triage", "handoff"]),
                    ],
                    states=[
                        SurfaceState(name="default"),
                        SurfaceState(name="loading", allowed_actions=["export"]),
                        SurfaceState(name="empty", allowed_actions=["audit"]),
                        SurfaceState(name="error", allowed_actions=["audit"]),
                    ],
                    supports_bulk_actions=True,
                ),
            ],
        ),
        contracts=[
            SurfaceInteractionContract(
                surface_name="Overview",
                required_action_ids=["drill-down", "export", "audit"],
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
                    denied_roles=["guest"],
                    audit_event="overview.access.denied",
                ),
                primary_persona="VP Eng",
                linked_wireframe_id="wf-overview",
                review_focus_areas=["metric hierarchy", "drill-down posture", "alert prioritization"],
                decision_prompts=[
                    "Is the executive KPI density still scannable within one screen?",
                    "Do risk and blocker cards point to the correct downstream investigation surface?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Queue",
                required_action_ids=["drill-down", "export", "audit"],
                requires_batch_actions=True,
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "cross-team-operator"],
                    denied_roles=["vp-eng", "guest"],
                    audit_event="queue.access.denied",
                ),
                primary_persona="Platform Admin",
                linked_wireframe_id="wf-queue",
                review_focus_areas=["batch approvals", "denied-role state", "audit rail"],
                decision_prompts=[
                    "Does the queue clearly separate selection, confirmation, and audit outcomes?",
                    "Is the denied-role treatment explicit enough for VP Eng and guest personas?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Run Detail",
                required_action_ids=["drill-down", "export", "audit"],
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"],
                    denied_roles=["guest"],
                    audit_event="run-detail.access.denied",
                ),
                primary_persona="Eng Lead",
                linked_wireframe_id="wf-run-detail",
                review_focus_areas=["replay context", "artifact evidence", "escalation path"],
                decision_prompts=[
                    "Can reviewers distinguish replay, compare, and escalated states without narration?",
                    "Is the audit trail visible at the moment an escalation decision is made?",
                ],
            ),
            SurfaceInteractionContract(
                surface_name="Triage",
                required_action_ids=["drill-down", "export", "audit"],
                requires_batch_actions=True,
                permission_rule=SurfacePermissionRule(
                    allowed_roles=["eng-lead", "platform-admin", "cross-team-operator"],
                    denied_roles=["vp-eng", "guest"],
                    audit_event="triage.access.denied",
                ),
                primary_persona="Cross-Team Operator",
                linked_wireframe_id="wf-triage",
                review_focus_areas=["handoff path", "bulk assignment", "ownership history"],
                decision_prompts=[
                    "Does the triage frame explain handoff consequences before ownership changes commit?",
                    "Is bulk assignment discoverable without overpowering the audit context?",
                ],
            ),
        ],
    )


_install_support_module(
    "console_ia",
    REQUIRED_SURFACE_STATES=REQUIRED_SURFACE_STATES,
    NavigationItem=NavigationItem,
    GlobalAction=GlobalAction,
    FilterDefinition=FilterDefinition,
    SurfaceState=SurfaceState,
    ConsoleSurface=ConsoleSurface,
    ConsoleIA=ConsoleIA,
    SurfacePermissionRule=SurfacePermissionRule,
    SurfaceInteractionContract=SurfaceInteractionContract,
    ConsoleInteractionDraft=ConsoleInteractionDraft,
    ConsoleInteractionAudit=ConsoleInteractionAudit,
    ConsoleIAAudit=ConsoleIAAudit,
    ConsoleIAAuditor=ConsoleIAAuditor,
    ConsoleInteractionAuditor=ConsoleInteractionAuditor,
    render_console_ia_report=render_console_ia_report,
    render_console_interaction_report=render_console_interaction_report,
    build_big_4203_console_interaction_draft=build_big_4203_console_interaction_draft,
)

from .workspace_bootstrap import WorkspaceBootstrapError, bootstrap_workspace, cleanup_workspace


@dataclass(frozen=True)
class ExecutionField:
    name: str
    field_type: str
    required: bool = True
    description: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field_type": self.field_type,
            "required": self.required,
            "description": self.description,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionField":
        return cls(
            name=str(data["name"]),
            field_type=str(data["field_type"]),
            required=bool(data.get("required", True)),
            description=str(data.get("description", "")),
        )


@dataclass
class ExecutionModel:
    name: str
    fields: List[ExecutionField] = field(default_factory=list)
    owner: str = ""

    @property
    def required_fields(self) -> List[str]:
        return [field.name for field in self.fields if field.required]

    def to_dict(self) -> Dict[str, object]:
        return {"name": self.name, "fields": [field.to_dict() for field in self.fields], "owner": self.owner}

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionModel":
        return cls(
            name=str(data["name"]),
            fields=[ExecutionField.from_dict(field) for field in data.get("fields", [])],
            owner=str(data.get("owner", "")),
        )


@dataclass
class ExecutionApiSpec:
    name: str
    method: str
    path: str
    request_model: str
    response_model: str
    required_permission: str
    emitted_audits: List[str] = field(default_factory=list)
    emitted_metrics: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "method": self.method,
            "path": self.path,
            "request_model": self.request_model,
            "response_model": self.response_model,
            "required_permission": self.required_permission,
            "emitted_audits": list(self.emitted_audits),
            "emitted_metrics": list(self.emitted_metrics),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionApiSpec":
        return cls(
            name=str(data["name"]),
            method=str(data["method"]),
            path=str(data["path"]),
            request_model=str(data.get("request_model", "")),
            response_model=str(data.get("response_model", "")),
            required_permission=str(data.get("required_permission", "")),
            emitted_audits=[str(item) for item in data.get("emitted_audits", [])],
            emitted_metrics=[str(item) for item in data.get("emitted_metrics", [])],
        )


@dataclass(frozen=True)
class ExecutionPermission:
    name: str
    resource: str
    actions: List[str] = field(default_factory=list)
    scopes: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "resource": self.resource,
            "actions": list(self.actions),
            "scopes": list(self.scopes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionPermission":
        return cls(
            name=str(data["name"]),
            resource=str(data.get("resource", "")),
            actions=[str(item) for item in data.get("actions", [])],
            scopes=[str(item) for item in data.get("scopes", [])],
        )


@dataclass(frozen=True)
class ExecutionRole:
    name: str
    personas: List[str] = field(default_factory=list)
    granted_permissions: List[str] = field(default_factory=list)
    scope_bindings: List[str] = field(default_factory=list)
    escalation_target: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "personas": list(self.personas),
            "granted_permissions": list(self.granted_permissions),
            "scope_bindings": list(self.scope_bindings),
            "escalation_target": self.escalation_target,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionRole":
        return cls(
            name=str(data["name"]),
            personas=[str(item) for item in data.get("personas", [])],
            granted_permissions=[str(item) for item in data.get("granted_permissions", [])],
            scope_bindings=[str(item) for item in data.get("scope_bindings", [])],
            escalation_target=str(data.get("escalation_target", "")),
        )


@dataclass
class PermissionCheckResult:
    allowed: bool
    granted_permissions: List[str] = field(default_factory=list)
    missing_permissions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "allowed": self.allowed,
            "granted_permissions": list(self.granted_permissions),
            "missing_permissions": list(self.missing_permissions),
        }


class ExecutionPermissionMatrix:
    def __init__(self, permissions: List[ExecutionPermission], roles: Optional[List[ExecutionRole]] = None) -> None:
        self.permissions = {permission.name: permission for permission in permissions}
        self.roles = {role.name: role for role in roles or []}

    def evaluate(self, required_permissions: List[str], granted_permissions: List[str]) -> PermissionCheckResult:
        granted_set = {permission for permission in granted_permissions if permission in self.permissions}
        missing = [permission for permission in required_permissions if permission not in granted_set]
        return PermissionCheckResult(allowed=not missing, granted_permissions=sorted(granted_set), missing_permissions=missing)

    def evaluate_roles(self, required_permissions: List[str], actor_roles: List[str]) -> PermissionCheckResult:
        granted_permissions = {
            permission
            for role_name in actor_roles
            for permission in self.roles.get(role_name, ExecutionRole(name=role_name)).granted_permissions
            if permission in self.permissions
        }
        return self.evaluate(required_permissions=required_permissions, granted_permissions=sorted(granted_permissions))


@dataclass(frozen=True)
class MetricDefinition:
    name: str
    unit: str
    owner: str
    description: str = ""

    def to_dict(self) -> Dict[str, str]:
        return {"name": self.name, "unit": self.unit, "owner": self.owner, "description": self.description}

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "MetricDefinition":
        return cls(
            name=str(data["name"]),
            unit=str(data.get("unit", "")),
            owner=str(data.get("owner", "")),
            description=str(data.get("description", "")),
        )


@dataclass(frozen=True)
class AuditPolicy:
    event_type: str
    required_fields: List[str] = field(default_factory=list)
    retention_days: int = 30
    severity: str = "info"

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_type": self.event_type,
            "required_fields": list(self.required_fields),
            "retention_days": self.retention_days,
            "severity": self.severity,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AuditPolicy":
        return cls(
            event_type=str(data["event_type"]),
            required_fields=[str(item) for item in data.get("required_fields", [])],
            retention_days=int(data.get("retention_days", 30)),
            severity=str(data.get("severity", "info")),
        )


@dataclass
class ExecutionContract:
    contract_id: str
    version: str
    models: List[ExecutionModel] = field(default_factory=list)
    apis: List[ExecutionApiSpec] = field(default_factory=list)
    permissions: List[ExecutionPermission] = field(default_factory=list)
    roles: List[ExecutionRole] = field(default_factory=list)
    metrics: List[MetricDefinition] = field(default_factory=list)
    audit_policies: List[AuditPolicy] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "models": [model.to_dict() for model in self.models],
            "apis": [api.to_dict() for api in self.apis],
            "permissions": [permission.to_dict() for permission in self.permissions],
            "roles": [role.to_dict() for role in self.roles],
            "metrics": [metric.to_dict() for metric in self.metrics],
            "audit_policies": [policy.to_dict() for policy in self.audit_policies],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionContract":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            models=[ExecutionModel.from_dict(model) for model in data.get("models", [])],
            apis=[ExecutionApiSpec.from_dict(api) for api in data.get("apis", [])],
            permissions=[ExecutionPermission.from_dict(permission) for permission in data.get("permissions", [])],
            roles=[ExecutionRole.from_dict(role) for role in data.get("roles", [])],
            metrics=[MetricDefinition.from_dict(metric) for metric in data.get("metrics", [])],
            audit_policies=[AuditPolicy.from_dict(policy) for policy in data.get("audit_policies", [])],
        )


@dataclass
class ExecutionContractAudit:
    contract_id: str
    version: str
    models_missing_required_fields: Dict[str, List[str]] = field(default_factory=dict)
    apis_missing_permissions: List[str] = field(default_factory=list)
    apis_missing_audits: List[str] = field(default_factory=list)
    apis_missing_metrics: List[str] = field(default_factory=list)
    undefined_model_refs: Dict[str, List[str]] = field(default_factory=dict)
    undefined_permissions: Dict[str, str] = field(default_factory=dict)
    missing_roles: List[str] = field(default_factory=list)
    roles_missing_personas: List[str] = field(default_factory=list)
    roles_missing_scope_bindings: List[str] = field(default_factory=list)
    roles_missing_escalation_targets: List[str] = field(default_factory=list)
    roles_missing_permissions: List[str] = field(default_factory=list)
    undefined_role_permissions: Dict[str, List[str]] = field(default_factory=dict)
    permissions_without_roles: List[str] = field(default_factory=list)
    apis_without_role_coverage: List[str] = field(default_factory=list)
    undefined_metrics: Dict[str, List[str]] = field(default_factory=dict)
    undefined_audit_events: Dict[str, List[str]] = field(default_factory=dict)
    audit_policies_below_retention: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        api_count = max(1, len(self.apis_missing_permissions) + len(self.apis_missing_audits) + len(self.apis_missing_metrics))
        issue_count = (
            len(self.models_missing_required_fields)
            + len(self.apis_missing_permissions)
            + len(self.apis_missing_audits)
            + len(self.apis_missing_metrics)
            + len(self.undefined_model_refs)
            + len(self.undefined_permissions)
            + len(self.missing_roles)
            + len(self.roles_missing_personas)
            + len(self.roles_missing_scope_bindings)
            + len(self.roles_missing_escalation_targets)
            + len(self.roles_missing_permissions)
            + len(self.undefined_role_permissions)
            + len(self.permissions_without_roles)
            + len(self.apis_without_role_coverage)
            + len(self.undefined_metrics)
            + len(self.undefined_audit_events)
            + len(self.audit_policies_below_retention)
        )
        if issue_count == 0:
            return 100.0
        penalty = min(100.0, issue_count * (100.0 / api_count))
        return round(max(0.0, 100.0 - penalty), 1)

    @property
    def release_ready(self) -> bool:
        return self.readiness_score == 100.0

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "models_missing_required_fields": {name: list(fields) for name, fields in self.models_missing_required_fields.items()},
            "apis_missing_permissions": list(self.apis_missing_permissions),
            "apis_missing_audits": list(self.apis_missing_audits),
            "apis_missing_metrics": list(self.apis_missing_metrics),
            "undefined_model_refs": {name: list(values) for name, values in self.undefined_model_refs.items()},
            "undefined_permissions": dict(self.undefined_permissions),
            "missing_roles": list(self.missing_roles),
            "roles_missing_personas": list(self.roles_missing_personas),
            "roles_missing_scope_bindings": list(self.roles_missing_scope_bindings),
            "roles_missing_escalation_targets": list(self.roles_missing_escalation_targets),
            "roles_missing_permissions": list(self.roles_missing_permissions),
            "undefined_role_permissions": {name: list(values) for name, values in self.undefined_role_permissions.items()},
            "permissions_without_roles": list(self.permissions_without_roles),
            "apis_without_role_coverage": list(self.apis_without_role_coverage),
            "undefined_metrics": {name: list(values) for name, values in self.undefined_metrics.items()},
            "undefined_audit_events": {name: list(values) for name, values in self.undefined_audit_events.items()},
            "audit_policies_below_retention": list(self.audit_policies_below_retention),
            "readiness_score": self.readiness_score,
            "release_ready": self.release_ready,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionContractAudit":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            models_missing_required_fields={str(name): [str(field) for field in fields] for name, fields in dict(data.get("models_missing_required_fields", {})).items()},
            apis_missing_permissions=[str(item) for item in data.get("apis_missing_permissions", [])],
            apis_missing_audits=[str(item) for item in data.get("apis_missing_audits", [])],
            apis_missing_metrics=[str(item) for item in data.get("apis_missing_metrics", [])],
            undefined_model_refs={str(name): [str(value) for value in values] for name, values in dict(data.get("undefined_model_refs", {})).items()},
            undefined_permissions={str(name): str(value) for name, value in dict(data.get("undefined_permissions", {})).items()},
            missing_roles=[str(item) for item in data.get("missing_roles", [])],
            roles_missing_personas=[str(item) for item in data.get("roles_missing_personas", [])],
            roles_missing_scope_bindings=[str(item) for item in data.get("roles_missing_scope_bindings", [])],
            roles_missing_escalation_targets=[str(item) for item in data.get("roles_missing_escalation_targets", [])],
            roles_missing_permissions=[str(item) for item in data.get("roles_missing_permissions", [])],
            undefined_role_permissions={str(name): [str(value) for value in values] for name, values in dict(data.get("undefined_role_permissions", {})).items()},
            permissions_without_roles=[str(item) for item in data.get("permissions_without_roles", [])],
            apis_without_role_coverage=[str(item) for item in data.get("apis_without_role_coverage", [])],
            undefined_metrics={str(name): [str(value) for value in values] for name, values in dict(data.get("undefined_metrics", {})).items()},
            undefined_audit_events={str(name): [str(value) for value in values] for name, values in dict(data.get("undefined_audit_events", {})).items()},
            audit_policies_below_retention=[str(item) for item in data.get("audit_policies_below_retention", [])],
        )


class ExecutionContractLibrary:
    REQUIRED_MODEL_FIELDS = {
        "ExecutionRequest": ["task_id", "actor", "requested_tools"],
        "ExecutionResponse": ["run_id", "status", "sandbox_profile"],
    }
    REQUIRED_ROLES = ["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"]

    def audit(self, contract: ExecutionContract) -> ExecutionContractAudit:
        model_names = {model.name for model in contract.models}
        permission_names = {permission.name for permission in contract.permissions}
        metric_names = {metric.name for metric in contract.metrics}
        audit_events = {policy.event_type for policy in contract.audit_policies}
        role_names = {role.name for role in contract.roles}
        models_missing_required_fields: Dict[str, List[str]] = {}
        for model in contract.models:
            expected_fields = self.REQUIRED_MODEL_FIELDS.get(model.name, [])
            missing = [field for field in expected_fields if field not in model.required_fields]
            if missing:
                models_missing_required_fields[model.name] = missing
        undefined_model_refs: Dict[str, List[str]] = {}
        undefined_permissions: Dict[str, str] = {}
        missing_roles = sorted(role for role in self.REQUIRED_ROLES if role not in role_names)
        roles_missing_personas: List[str] = []
        roles_missing_scope_bindings: List[str] = []
        roles_missing_escalation_targets: List[str] = []
        roles_missing_permissions: List[str] = []
        undefined_role_permissions: Dict[str, List[str]] = {}
        permissions_granted_by_roles: set[str] = set()
        apis_without_role_coverage: List[str] = []
        undefined_metrics: Dict[str, List[str]] = {}
        undefined_audit_events: Dict[str, List[str]] = {}
        apis_missing_permissions: List[str] = []
        apis_missing_audits: List[str] = []
        apis_missing_metrics: List[str] = []
        for api in contract.apis:
            missing_models = [model_name for model_name in [api.request_model, api.response_model] if model_name and model_name not in model_names]
            if missing_models:
                undefined_model_refs[api.name] = missing_models
            if not api.required_permission:
                apis_missing_permissions.append(api.name)
            elif api.required_permission not in permission_names:
                undefined_permissions[api.name] = api.required_permission
            if not api.emitted_audits:
                apis_missing_audits.append(api.name)
            else:
                missing_events = [event for event in api.emitted_audits if event not in audit_events]
                if missing_events:
                    undefined_audit_events[api.name] = missing_events
            if not api.emitted_metrics:
                apis_missing_metrics.append(api.name)
            else:
                missing_metric_defs = [metric for metric in api.emitted_metrics if metric not in metric_names]
                if missing_metric_defs:
                    undefined_metrics[api.name] = missing_metric_defs
        for role in contract.roles:
            if not role.personas:
                roles_missing_personas.append(role.name)
            if not role.scope_bindings:
                roles_missing_scope_bindings.append(role.name)
            if not role.escalation_target.strip():
                roles_missing_escalation_targets.append(role.name)
            if not role.granted_permissions:
                roles_missing_permissions.append(role.name)
                continue
            missing_permissions = [permission for permission in role.granted_permissions if permission not in permission_names]
            if missing_permissions:
                undefined_role_permissions[role.name] = missing_permissions
            permissions_granted_by_roles.update(permission for permission in role.granted_permissions if permission in permission_names)
        for api in contract.apis:
            if api.required_permission and api.required_permission in permission_names and api.required_permission not in permissions_granted_by_roles:
                apis_without_role_coverage.append(api.name)
        permissions_without_roles = sorted(permission for permission in permission_names if permission not in permissions_granted_by_roles)
        audit_policies_below_retention = sorted(policy.event_type for policy in contract.audit_policies if policy.retention_days < 30)
        return ExecutionContractAudit(
            contract_id=contract.contract_id,
            version=contract.version,
            models_missing_required_fields=models_missing_required_fields,
            apis_missing_permissions=sorted(apis_missing_permissions),
            apis_missing_audits=sorted(apis_missing_audits),
            apis_missing_metrics=sorted(apis_missing_metrics),
            undefined_model_refs=undefined_model_refs,
            undefined_permissions=undefined_permissions,
            missing_roles=missing_roles,
            roles_missing_personas=sorted(roles_missing_personas),
            roles_missing_scope_bindings=sorted(roles_missing_scope_bindings),
            roles_missing_escalation_targets=sorted(roles_missing_escalation_targets),
            roles_missing_permissions=sorted(roles_missing_permissions),
            undefined_role_permissions=undefined_role_permissions,
            permissions_without_roles=permissions_without_roles,
            apis_without_role_coverage=sorted(apis_without_role_coverage),
            undefined_metrics=undefined_metrics,
            undefined_audit_events=undefined_audit_events,
            audit_policies_below_retention=audit_policies_below_retention,
        )


def render_execution_contract_report(contract: ExecutionContract, audit: ExecutionContractAudit) -> str:
    lines = [
        "# Execution Layer Technical Contract",
        "",
        f"- Contract ID: {contract.contract_id}",
        f"- Version: {contract.version}",
        f"- Models: {len(contract.models)}",
        f"- APIs: {len(contract.apis)}",
        f"- Permissions: {len(contract.permissions)}",
        f"- Roles: {len(contract.roles)}",
        f"- Metrics: {len(contract.metrics)}",
        f"- Audit Policies: {len(contract.audit_policies)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## APIs",
        "",
    ]
    if contract.apis:
        for api in contract.apis:
            audits = ", ".join(api.emitted_audits) if api.emitted_audits else "none"
            metrics = ", ".join(api.emitted_metrics) if api.emitted_metrics else "none"
            permission = api.required_permission or "none"
            lines.append(
                f"- {api.method} {api.path}: request={api.request_model or 'none'} "
                f"response={api.response_model or 'none'} permission={permission} audits={audits} metrics={metrics}"
            )
    else:
        lines.append("- APIs: none")
    lines.extend(["", "## Roles", ""])
    if contract.roles:
        for role in contract.roles:
            personas = ", ".join(role.personas) if role.personas else "none"
            permissions = ", ".join(role.granted_permissions) if role.granted_permissions else "none"
            scopes = ", ".join(role.scope_bindings) if role.scope_bindings else "none"
            escalation_target = role.escalation_target or "none"
            lines.append(f"- {role.name}: personas={personas} permissions={permissions} scopes={scopes} escalation={escalation_target}")
    else:
        lines.append("- Roles: none")
    lines.extend(
        [
            "",
            "## Audit",
            "",
            f"- Models missing required fields: {', '.join(f'{name}={fields}' for name, fields in sorted(audit.models_missing_required_fields.items())) if audit.models_missing_required_fields else 'none'}",
            f"- APIs missing permissions: {', '.join(audit.apis_missing_permissions) if audit.apis_missing_permissions else 'none'}",
            f"- APIs missing audits: {', '.join(audit.apis_missing_audits) if audit.apis_missing_audits else 'none'}",
            f"- APIs missing metrics: {', '.join(audit.apis_missing_metrics) if audit.apis_missing_metrics else 'none'}",
            f"- Undefined model refs: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_model_refs.items())) if audit.undefined_model_refs else 'none'}",
            f"- Undefined permissions: {', '.join(f'{name}={value}' for name, value in sorted(audit.undefined_permissions.items())) if audit.undefined_permissions else 'none'}",
            f"- Missing roles: {', '.join(audit.missing_roles) if audit.missing_roles else 'none'}",
            f"- Roles missing personas: {', '.join(audit.roles_missing_personas) if audit.roles_missing_personas else 'none'}",
            f"- Roles missing scope bindings: {', '.join(audit.roles_missing_scope_bindings) if audit.roles_missing_scope_bindings else 'none'}",
            f"- Roles missing escalation targets: {', '.join(audit.roles_missing_escalation_targets) if audit.roles_missing_escalation_targets else 'none'}",
            f"- Roles missing permissions: {', '.join(audit.roles_missing_permissions) if audit.roles_missing_permissions else 'none'}",
            f"- Undefined role permissions: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_role_permissions.items())) if audit.undefined_role_permissions else 'none'}",
            f"- Permissions without roles: {', '.join(audit.permissions_without_roles) if audit.permissions_without_roles else 'none'}",
            f"- APIs without role coverage: {', '.join(audit.apis_without_role_coverage) if audit.apis_without_role_coverage else 'none'}",
            f"- Undefined metrics: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_metrics.items())) if audit.undefined_metrics else 'none'}",
            f"- Undefined audit events: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_audit_events.items())) if audit.undefined_audit_events else 'none'}",
            f"- Audit retention gaps: {', '.join(audit.audit_policies_below_retention) if audit.audit_policies_below_retention else 'none'}",
        ]
    )
    return "\n".join(lines)


def build_operations_api_contract(contract_id: str = "OPE-131", version: str = "v4.0-draft1") -> ExecutionContract:
    return ExecutionContract(
        contract_id=contract_id,
        version=version,
        models=[
            ExecutionModel(name="OperationsDashboardResponse", owner="operations", fields=[ExecutionField("period", "string"), ExecutionField("total_runs", "int"), ExecutionField("success_rate", "float"), ExecutionField("approval_queue_depth", "int"), ExecutionField("sla_breach_count", "int"), ExecutionField("top_blockers", "string[]", required=False)]),
            ExecutionModel(name="RunDetailResponse", owner="operations", fields=[ExecutionField("run_id", "string"), ExecutionField("task_id", "string"), ExecutionField("status", "string"), ExecutionField("timeline_events", "RunDetailEvent[]"), ExecutionField("resources", "RunDetailResource[]"), ExecutionField("audit_count", "int")]),
            ExecutionModel(name="RunReplayResponse", owner="operations", fields=[ExecutionField("run_id", "string"), ExecutionField("replay_available", "bool"), ExecutionField("replay_path", "string", required=False), ExecutionField("benchmark_case_ids", "string[]", required=False)]),
            ExecutionModel(name="QueueControlCenterResponse", owner="operations", fields=[ExecutionField("queue_depth", "int"), ExecutionField("queued_by_priority", "map<string,int>"), ExecutionField("queued_by_risk", "map<string,int>"), ExecutionField("waiting_approval_runs", "int"), ExecutionField("blocked_tasks", "string[]", required=False)]),
            ExecutionModel(name="QueueActionRequest", owner="operations", fields=[ExecutionField("actor", "string"), ExecutionField("reason", "string")]),
            ExecutionModel(name="QueueActionResponse", owner="operations", fields=[ExecutionField("task_id", "string"), ExecutionField("action", "string"), ExecutionField("accepted", "bool"), ExecutionField("queue_depth", "int")]),
            ExecutionModel(name="RunApprovalRequest", owner="operations", fields=[ExecutionField("actor", "string"), ExecutionField("approval_token", "string"), ExecutionField("decision", "string"), ExecutionField("reason", "string", required=False)]),
            ExecutionModel(name="RunApprovalResponse", owner="operations", fields=[ExecutionField("run_id", "string"), ExecutionField("status", "string"), ExecutionField("approved", "bool"), ExecutionField("required_follow_up", "string[]", required=False)]),
            ExecutionModel(name="RiskOverviewResponse", owner="risk", fields=[ExecutionField("period", "string"), ExecutionField("high_risk_runs", "int"), ExecutionField("approval_required_runs", "int"), ExecutionField("risk_factors", "string[]"), ExecutionField("recommendation", "string")]),
            ExecutionModel(name="SlaOverviewResponse", owner="operations", fields=[ExecutionField("period", "string"), ExecutionField("sla_target_minutes", "int"), ExecutionField("average_cycle_minutes", "float"), ExecutionField("sla_breach_count", "int"), ExecutionField("approval_queue_depth", "int")]),
            ExecutionModel(name="RegressionCenterResponse", owner="operations", fields=[ExecutionField("baseline_version", "string"), ExecutionField("current_version", "string"), ExecutionField("regression_count", "int"), ExecutionField("improved_cases", "string[]", required=False), ExecutionField("regressions", "RegressionFinding[]", required=False)]),
            ExecutionModel(name="FlowCanvasResponse", owner="orchestration", fields=[ExecutionField("run_id", "string"), ExecutionField("collaboration_mode", "string"), ExecutionField("departments", "string[]"), ExecutionField("required_approvals", "string[]", required=False), ExecutionField("billing_model", "string"), ExecutionField("recommendation", "string")]),
            ExecutionModel(name="BillingEntitlementsResponse", owner="orchestration", fields=[ExecutionField("period", "string"), ExecutionField("tier", "string"), ExecutionField("billing_model_counts", "map<string,int>"), ExecutionField("upgrade_required_runs", "int"), ExecutionField("estimated_cost_usd", "float")]),
            ExecutionModel(name="BillingRunChargeResponse", owner="orchestration", fields=[ExecutionField("run_id", "string"), ExecutionField("billing_model", "string"), ExecutionField("estimated_cost_usd", "float"), ExecutionField("overage_cost_usd", "float"), ExecutionField("upgrade_required", "bool")]),
        ],
        apis=[
            ExecutionApiSpec(name="get_operations_dashboard", method="GET", path="/operations/dashboard", request_model="", response_model="OperationsDashboardResponse", required_permission="operations.dashboard.read", emitted_audits=["operations.dashboard.viewed"], emitted_metrics=["operations.dashboard.requests", "operations.dashboard.latency.ms"]),
            ExecutionApiSpec(name="get_run_detail", method="GET", path="/operations/runs/{run_id}", request_model="", response_model="RunDetailResponse", required_permission="operations.run.read", emitted_audits=["operations.run_detail.viewed"], emitted_metrics=["operations.run_detail.requests", "operations.run_detail.latency.ms"]),
            ExecutionApiSpec(name="get_run_replay", method="GET", path="/operations/runs/{run_id}/replay", request_model="", response_model="RunReplayResponse", required_permission="operations.run.read", emitted_audits=["operations.run_replay.viewed"], emitted_metrics=["operations.run_replay.requests", "operations.run_replay.latency.ms"]),
            ExecutionApiSpec(name="get_queue_control_center", method="GET", path="/operations/queue/control-center", request_model="", response_model="QueueControlCenterResponse", required_permission="operations.queue.read", emitted_audits=["operations.queue.viewed"], emitted_metrics=["operations.queue.requests", "operations.queue.depth"]),
            ExecutionApiSpec(name="retry_queue_task", method="POST", path="/operations/queue/{task_id}/retry", request_model="QueueActionRequest", response_model="QueueActionResponse", required_permission="operations.queue.act", emitted_audits=["operations.queue.retry.requested"], emitted_metrics=["operations.queue.retry.requests", "operations.queue.depth"]),
            ExecutionApiSpec(name="approve_run_execution", method="POST", path="/operations/runs/{run_id}/approve", request_model="RunApprovalRequest", response_model="RunApprovalResponse", required_permission="operations.run.approve", emitted_audits=["operations.run.approval.recorded"], emitted_metrics=["operations.run.approval.requests", "operations.approval.queue.depth"]),
            ExecutionApiSpec(name="get_risk_overview", method="GET", path="/operations/risk/overview", request_model="", response_model="RiskOverviewResponse", required_permission="operations.risk.read", emitted_audits=["operations.risk.viewed"], emitted_metrics=["operations.risk.requests", "operations.risk.high_runs"]),
            ExecutionApiSpec(name="get_sla_overview", method="GET", path="/operations/sla/overview", request_model="", response_model="SlaOverviewResponse", required_permission="operations.sla.read", emitted_audits=["operations.sla.viewed"], emitted_metrics=["operations.sla.requests", "operations.sla.breaches"]),
            ExecutionApiSpec(name="get_regression_center", method="GET", path="/operations/regressions", request_model="", response_model="RegressionCenterResponse", required_permission="operations.regression.read", emitted_audits=["operations.regression.viewed"], emitted_metrics=["operations.regression.requests", "operations.regression.count"]),
            ExecutionApiSpec(name="get_flow_canvas", method="GET", path="/operations/flows/{run_id}", request_model="", response_model="FlowCanvasResponse", required_permission="operations.flow.read", emitted_audits=["operations.flow.viewed"], emitted_metrics=["operations.flow.requests", "operations.flow.handoff_count"]),
            ExecutionApiSpec(name="get_billing_entitlements", method="GET", path="/operations/billing/entitlements", request_model="", response_model="BillingEntitlementsResponse", required_permission="operations.billing.read", emitted_audits=["operations.billing.viewed"], emitted_metrics=["operations.billing.requests", "operations.billing.estimated_cost_usd"]),
            ExecutionApiSpec(name="get_billing_run_charge", method="GET", path="/operations/billing/runs/{run_id}", request_model="", response_model="BillingRunChargeResponse", required_permission="operations.billing.read", emitted_audits=["operations.billing.run_charge.viewed"], emitted_metrics=["operations.billing.run_charge.requests", "operations.billing.overage_cost_usd"]),
        ],
        permissions=[
            ExecutionPermission("operations.dashboard.read", "operations-dashboard", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.run.read", "run-detail", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.queue.read", "queue-control-center", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.queue.act", "queue-control-center", actions=["retry", "escalate"], scopes=["team"]),
            ExecutionPermission("operations.run.approve", "run-approval", actions=["approve"], scopes=["workspace"]),
            ExecutionPermission("operations.risk.read", "risk-overview", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.sla.read", "sla-overview", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.regression.read", "regression-center", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.flow.read", "flow-canvas", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.billing.read", "billing-entitlements", actions=["read"], scopes=["workspace"]),
        ],
        roles=[
            ExecutionRole(name="eng-lead", personas=["Eng Lead"], granted_permissions=["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read"], scope_bindings=["team", "workspace"], escalation_target="vp-eng"),
            ExecutionRole(name="platform-admin", personas=["Platform Admin"], granted_permissions=["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.billing.read"], scope_bindings=["workspace"], escalation_target="vp-eng"),
            ExecutionRole(name="vp-eng", personas=["VP Eng"], granted_permissions=["operations.dashboard.read", "operations.run.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.billing.read"], scope_bindings=["portfolio", "workspace"], escalation_target="none"),
            ExecutionRole(name="cross-team-operator", personas=["Cross-Team Operator"], granted_permissions=["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.flow.read", "operations.billing.read"], scope_bindings=["cross-team", "team", "workspace"], escalation_target="eng-lead"),
        ],
        metrics=[
            MetricDefinition("operations.dashboard.requests", "count", owner="operations"),
            MetricDefinition("operations.dashboard.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.run_detail.requests", "count", owner="operations"),
            MetricDefinition("operations.run_detail.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.run_replay.requests", "count", owner="operations"),
            MetricDefinition("operations.run_replay.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.queue.requests", "count", owner="operations"),
            MetricDefinition("operations.queue.depth", "count", owner="operations"),
            MetricDefinition("operations.queue.retry.requests", "count", owner="operations"),
            MetricDefinition("operations.run.approval.requests", "count", owner="operations"),
            MetricDefinition("operations.approval.queue.depth", "count", owner="operations"),
            MetricDefinition("operations.risk.requests", "count", owner="risk"),
            MetricDefinition("operations.risk.high_runs", "count", owner="risk"),
            MetricDefinition("operations.sla.requests", "count", owner="operations"),
            MetricDefinition("operations.sla.breaches", "count", owner="operations"),
            MetricDefinition("operations.regression.requests", "count", owner="operations"),
            MetricDefinition("operations.regression.count", "count", owner="operations"),
            MetricDefinition("operations.flow.requests", "count", owner="orchestration"),
            MetricDefinition("operations.flow.handoff_count", "count", owner="orchestration"),
            MetricDefinition("operations.billing.requests", "count", owner="finance"),
            MetricDefinition("operations.billing.estimated_cost_usd", "usd", owner="finance"),
            MetricDefinition("operations.billing.run_charge.requests", "count", owner="finance"),
            MetricDefinition("operations.billing.overage_cost_usd", "usd", owner="finance"),
        ],
        audit_policies=[
            AuditPolicy("operations.dashboard.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.run_detail.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.run_replay.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.queue.viewed", required_fields=["actor", "queue_depth"], retention_days=180, severity="info"),
            AuditPolicy("operations.queue.retry.requested", required_fields=["actor", "task_id", "reason"], retention_days=180, severity="warning"),
            AuditPolicy("operations.run.approval.recorded", required_fields=["actor", "run_id", "decision"], retention_days=365, severity="warning"),
            AuditPolicy("operations.risk.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.sla.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.regression.viewed", required_fields=["actor", "current_version"], retention_days=180, severity="info"),
            AuditPolicy("operations.flow.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.billing.viewed", required_fields=["actor", "period", "tier"], retention_days=365, severity="info"),
            AuditPolicy("operations.billing.run_charge.viewed", required_fields=["actor", "run_id", "billing_model"], retention_days=365, severity="info"),
        ],
    )


_install_support_module(
    "execution_contract",
    ExecutionField=ExecutionField,
    ExecutionModel=ExecutionModel,
    ExecutionApiSpec=ExecutionApiSpec,
    ExecutionPermission=ExecutionPermission,
    ExecutionRole=ExecutionRole,
    PermissionCheckResult=PermissionCheckResult,
    ExecutionPermissionMatrix=ExecutionPermissionMatrix,
    MetricDefinition=MetricDefinition,
    AuditPolicy=AuditPolicy,
    ExecutionContract=ExecutionContract,
    ExecutionContractAudit=ExecutionContractAudit,
    ExecutionContractLibrary=ExecutionContractLibrary,
    render_execution_contract_report=render_execution_contract_report,
    build_operations_api_contract=build_operations_api_contract,
)

REPO_ACTION_PERMISSIONS = [
    ExecutionPermission(name="repo.push", resource="repo", actions=["push"], scopes=["project"]),
    ExecutionPermission(name="repo.fetch", resource="repo", actions=["fetch"], scopes=["project"]),
    ExecutionPermission(name="repo.diff", resource="repo", actions=["diff"], scopes=["project"]),
    ExecutionPermission(name="repo.post", resource="repo-board", actions=["create"], scopes=["channel"]),
    ExecutionPermission(name="repo.reply", resource="repo-board", actions=["reply"], scopes=["channel"]),
    ExecutionPermission(name="repo.accept", resource="repo", actions=["approve"], scopes=["run"]),
    ExecutionPermission(name="repo.inspect", resource="repo", actions=["inspect"], scopes=["project"]),
]

REPO_ROLE_POLICIES = [
    ExecutionRole(
        name="platform-admin",
        personas=["Platform Admin"],
        granted_permissions=[permission.name for permission in REPO_ACTION_PERMISSIONS],
        scope_bindings=["workspace"],
        escalation_target="security",
    ),
    ExecutionRole(
        name="eng-lead",
        personas=["Eng Lead"],
        granted_permissions=["repo.push", "repo.fetch", "repo.diff", "repo.post", "repo.reply", "repo.accept", "repo.inspect"],
        scope_bindings=["project"],
        escalation_target="platform-admin",
    ),
    ExecutionRole(
        name="reviewer",
        personas=["Reviewer"],
        granted_permissions=["repo.fetch", "repo.diff", "repo.reply", "repo.inspect", "repo.accept"],
        scope_bindings=["project"],
        escalation_target="eng-lead",
    ),
    ExecutionRole(
        name="execution-agent",
        personas=["Execution Agent"],
        granted_permissions=["repo.fetch", "repo.diff", "repo.post", "repo.reply"],
        scope_bindings=["run"],
        escalation_target="reviewer",
    ),
]


@dataclass
class RepoPermissionContract:
    matrix: ExecutionPermissionMatrix = field(
        default_factory=lambda: ExecutionPermissionMatrix(REPO_ACTION_PERMISSIONS, REPO_ROLE_POLICIES)
    )

    def check(self, *, action_permission: str, actor_roles: List[str]) -> bool:
        result = self.matrix.evaluate_roles([action_permission], actor_roles)
        return result.allowed


def repo_required_audit_fields(action: str) -> List[str]:
    common = ["task_id", "run_id", "repo_space_id", "actor"]
    if action == "repo.accept":
        return [*common, "accepted_commit_hash", "reviewer"]
    if action in {"repo.push", "repo.fetch", "repo.diff"}:
        return [*common, "commit_hash", "outcome"]
    if action in {"repo.post", "repo.reply"}:
        return [*common, "channel", "post_id", "outcome"]
    return common


def missing_repo_audit_fields(action: str, payload: Dict[str, object]) -> List[str]:
    required = repo_required_audit_fields(action)
    return [field for field in required if field not in payload]


_install_support_module(
    "repo_governance",
    REPO_ACTION_PERMISSIONS=REPO_ACTION_PERMISSIONS,
    REPO_ROLE_POLICIES=REPO_ROLE_POLICIES,
    RepoPermissionContract=RepoPermissionContract,
    repo_required_audit_fields=repo_required_audit_fields,
    missing_repo_audit_fields=missing_repo_audit_fields,
)

VALID_VIEW_VISIBILITY = {"private", "team", "organization"}
VALID_DIGEST_CHANNELS = {"email", "slack", "webhook"}
VALID_DIGEST_CADENCES = {"hourly", "daily", "weekly"}


@dataclass(frozen=True)
class SavedViewFilter:
    field: str
    operator: str
    value: str

    def to_dict(self) -> Dict[str, str]:
        return {"field": self.field, "operator": self.operator, "value": self.value}

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewFilter":
        return cls(field=str(data["field"]), operator=str(data["operator"]), value=str(data["value"]))


@dataclass
class SavedView:
    view_id: str
    name: str
    route: str
    owner: str
    visibility: str = "private"
    filters: List[SavedViewFilter] = field(default_factory=list)
    sort_by: str = ""
    pinned: bool = False
    is_default: bool = False

    @property
    def filter_count(self) -> int:
        return len(self.filters)

    def to_dict(self) -> Dict[str, object]:
        return {
            "view_id": self.view_id,
            "name": self.name,
            "route": self.route,
            "owner": self.owner,
            "visibility": self.visibility,
            "filters": [view_filter.to_dict() for view_filter in self.filters],
            "sort_by": self.sort_by,
            "pinned": self.pinned,
            "is_default": self.is_default,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedView":
        return cls(
            view_id=str(data["view_id"]),
            name=str(data["name"]),
            route=str(data["route"]),
            owner=str(data["owner"]),
            visibility=str(data.get("visibility", "private")),
            filters=[SavedViewFilter.from_dict(item) for item in data.get("filters", [])],
            sort_by=str(data.get("sort_by", "")),
            pinned=bool(data.get("pinned", False)),
            is_default=bool(data.get("is_default", False)),
        )


@dataclass
class AlertDigestSubscription:
    subscription_id: str
    saved_view_id: str
    channel: str
    cadence: str
    recipients: List[str] = field(default_factory=list)
    include_empty_results: bool = False
    muted: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "subscription_id": self.subscription_id,
            "saved_view_id": self.saved_view_id,
            "channel": self.channel,
            "cadence": self.cadence,
            "recipients": list(self.recipients),
            "include_empty_results": self.include_empty_results,
            "muted": self.muted,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AlertDigestSubscription":
        return cls(
            subscription_id=str(data["subscription_id"]),
            saved_view_id=str(data["saved_view_id"]),
            channel=str(data["channel"]),
            cadence=str(data["cadence"]),
            recipients=[str(recipient) for recipient in data.get("recipients", [])],
            include_empty_results=bool(data.get("include_empty_results", False)),
            muted=bool(data.get("muted", False)),
        )


@dataclass
class SavedViewCatalog:
    name: str
    version: str
    views: List[SavedView] = field(default_factory=list)
    subscriptions: List[AlertDigestSubscription] = field(default_factory=list)

    @property
    def view_index(self) -> Dict[str, SavedView]:
        return {view.view_id: view for view in self.views}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "views": [view.to_dict() for view in self.views],
            "subscriptions": [subscription.to_dict() for subscription in self.subscriptions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalog":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            views=[SavedView.from_dict(item) for item in data.get("views", [])],
            subscriptions=[AlertDigestSubscription.from_dict(item) for item in data.get("subscriptions", [])],
        )


@dataclass
class SavedViewCatalogAudit:
    catalog_name: str
    version: str
    view_count: int
    subscription_count: int
    duplicate_view_names: Dict[str, List[str]] = field(default_factory=dict)
    invalid_visibility_views: List[str] = field(default_factory=list)
    views_missing_filters: List[str] = field(default_factory=list)
    duplicate_default_views: Dict[str, List[str]] = field(default_factory=dict)
    orphan_subscriptions: List[str] = field(default_factory=list)
    subscriptions_missing_recipients: List[str] = field(default_factory=list)
    subscriptions_with_invalid_channel: List[str] = field(default_factory=list)
    subscriptions_with_invalid_cadence: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.view_count == 0:
            return 0.0
        penalties = (
            len(self.duplicate_view_names)
            + len(self.invalid_visibility_views)
            + len(self.views_missing_filters)
            + len(self.duplicate_default_views)
            + len(self.orphan_subscriptions)
            + len(self.subscriptions_missing_recipients)
            + len(self.subscriptions_with_invalid_channel)
            + len(self.subscriptions_with_invalid_cadence)
        )
        return round(max(0.0, 100 - ((penalties * 100) / self.view_count)), 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "catalog_name": self.catalog_name,
            "version": self.version,
            "view_count": self.view_count,
            "subscription_count": self.subscription_count,
            "duplicate_view_names": {
                key: list(values) for key, values in self.duplicate_view_names.items()
            },
            "invalid_visibility_views": list(self.invalid_visibility_views),
            "views_missing_filters": list(self.views_missing_filters),
            "duplicate_default_views": {
                key: list(values) for key, values in self.duplicate_default_views.items()
            },
            "orphan_subscriptions": list(self.orphan_subscriptions),
            "subscriptions_missing_recipients": list(self.subscriptions_missing_recipients),
            "subscriptions_with_invalid_channel": list(self.subscriptions_with_invalid_channel),
            "subscriptions_with_invalid_cadence": list(self.subscriptions_with_invalid_cadence),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalogAudit":
        return cls(
            catalog_name=str(data["catalog_name"]),
            version=str(data["version"]),
            view_count=int(data.get("view_count", 0)),
            subscription_count=int(data.get("subscription_count", 0)),
            duplicate_view_names={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_view_names", {})).items()
            },
            invalid_visibility_views=[str(name) for name in data.get("invalid_visibility_views", [])],
            views_missing_filters=[str(name) for name in data.get("views_missing_filters", [])],
            duplicate_default_views={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_default_views", {})).items()
            },
            orphan_subscriptions=[str(name) for name in data.get("orphan_subscriptions", [])],
            subscriptions_missing_recipients=[
                str(name) for name in data.get("subscriptions_missing_recipients", [])
            ],
            subscriptions_with_invalid_channel=[
                str(name) for name in data.get("subscriptions_with_invalid_channel", [])
            ],
            subscriptions_with_invalid_cadence=[
                str(name) for name in data.get("subscriptions_with_invalid_cadence", [])
            ],
        )


class SavedViewLibrary:
    def audit(self, catalog: SavedViewCatalog) -> SavedViewCatalogAudit:
        duplicate_view_names: Dict[str, List[str]] = {}
        invalid_visibility_views: List[str] = []
        views_missing_filters: List[str] = []
        duplicate_default_views: Dict[str, List[str]] = {}
        orphan_subscriptions: List[str] = []
        subscriptions_missing_recipients: List[str] = []
        subscriptions_with_invalid_channel: List[str] = []
        subscriptions_with_invalid_cadence: List[str] = []
        names_by_scope: Dict[str, List[str]] = {}
        defaults_by_scope: Dict[str, List[str]] = {}

        for view in catalog.views:
            scope = f"{view.route}:{view.owner}"
            names_by_scope.setdefault(scope, []).append(view.name)
            if view.is_default:
                defaults_by_scope.setdefault(scope, []).append(view.name)
            if view.visibility not in VALID_VIEW_VISIBILITY:
                invalid_visibility_views.append(view.name)
            if not view.filters:
                views_missing_filters.append(view.name)

        for scope, names in sorted(names_by_scope.items()):
            unique_names = sorted({name for name in names if names.count(name) > 1})
            if unique_names:
                duplicate_view_names[scope] = unique_names

        for scope, names in sorted(defaults_by_scope.items()):
            if len(names) > 1:
                duplicate_default_views[scope] = sorted(names)

        view_index = catalog.view_index
        for subscription in catalog.subscriptions:
            if subscription.saved_view_id not in view_index:
                orphan_subscriptions.append(subscription.subscription_id)
            if not subscription.recipients:
                subscriptions_missing_recipients.append(subscription.subscription_id)
            if subscription.channel not in VALID_DIGEST_CHANNELS:
                subscriptions_with_invalid_channel.append(subscription.subscription_id)
            if subscription.cadence not in VALID_DIGEST_CADENCES:
                subscriptions_with_invalid_cadence.append(subscription.subscription_id)

        return SavedViewCatalogAudit(
            catalog_name=catalog.name,
            version=catalog.version,
            view_count=len(catalog.views),
            subscription_count=len(catalog.subscriptions),
            duplicate_view_names=duplicate_view_names,
            invalid_visibility_views=sorted(invalid_visibility_views),
            views_missing_filters=sorted(views_missing_filters),
            duplicate_default_views=duplicate_default_views,
            orphan_subscriptions=sorted(orphan_subscriptions),
            subscriptions_missing_recipients=sorted(subscriptions_missing_recipients),
            subscriptions_with_invalid_channel=sorted(subscriptions_with_invalid_channel),
            subscriptions_with_invalid_cadence=sorted(subscriptions_with_invalid_cadence),
        )


def render_saved_view_report(catalog: SavedViewCatalog, audit: SavedViewCatalogAudit) -> str:
    lines = [
        "# Saved Views & Alert Digests Report",
        "",
        f"- Name: {catalog.name}",
        f"- Version: {catalog.version}",
        f"- Saved Views: {audit.view_count}",
        f"- Alert Subscriptions: {audit.subscription_count}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Saved Views",
        "",
    ]
    if catalog.views:
        for view in catalog.views:
            filters = ", ".join(
                f"{view_filter.field}{view_filter.operator}{view_filter.value}"
                for view_filter in view.filters
            ) or "none"
            lines.append(
                f"- {view.name}: route={view.route} owner={view.owner} visibility={view.visibility} "
                f"filters={filters} sort={view.sort_by or 'none'} pinned={view.pinned} default={view.is_default}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Alert Digests", ""])
    if catalog.subscriptions:
        for subscription in catalog.subscriptions:
            recipients = ", ".join(subscription.recipients) or "none"
            lines.append(
                f"- {subscription.subscription_id}: view={subscription.saved_view_id} channel={subscription.channel} "
                f"cadence={subscription.cadence} recipients={recipients} "
                f"include_empty={subscription.include_empty_results} muted={subscription.muted}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    duplicate_names = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_view_names.items())
        if audit.duplicate_view_names
        else "none"
    )
    duplicate_defaults = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_default_views.items())
        if audit.duplicate_default_views
        else "none"
    )
    lines.append(f"- Duplicate view names: {duplicate_names}")
    lines.append(
        f"- Invalid view visibility: {', '.join(audit.invalid_visibility_views) if audit.invalid_visibility_views else 'none'}"
    )
    lines.append(
        f"- Views missing filters: {', '.join(audit.views_missing_filters) if audit.views_missing_filters else 'none'}"
    )
    lines.append(f"- Duplicate default views: {duplicate_defaults}")
    lines.append(
        f"- Orphan subscriptions: {', '.join(audit.orphan_subscriptions) if audit.orphan_subscriptions else 'none'}"
    )
    lines.append(
        "- Subscriptions missing recipients: "
        f"{', '.join(audit.subscriptions_missing_recipients) if audit.subscriptions_missing_recipients else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid channel: "
        f"{', '.join(audit.subscriptions_with_invalid_channel) if audit.subscriptions_with_invalid_channel else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid cadence: "
        f"{', '.join(audit.subscriptions_with_invalid_cadence) if audit.subscriptions_with_invalid_cadence else 'none'}"
    )
    return "\n".join(lines) + "\n"


_install_support_module(
    "saved_views",
    VALID_VIEW_VISIBILITY=VALID_VIEW_VISIBILITY,
    VALID_DIGEST_CHANNELS=VALID_DIGEST_CHANNELS,
    VALID_DIGEST_CADENCES=VALID_DIGEST_CADENCES,
    SavedViewFilter=SavedViewFilter,
    SavedView=SavedView,
    AlertDigestSubscription=AlertDigestSubscription,
    SavedViewCatalog=SavedViewCatalog,
    SavedViewCatalogAudit=SavedViewCatalogAudit,
    SavedViewLibrary=SavedViewLibrary,
    render_saved_view_report=render_saved_view_report,
)

SCHEDULER_DECISION_EVENT = "execution.scheduler_decision"
MANUAL_TAKEOVER_EVENT = "execution.manual_takeover"
APPROVAL_RECORDED_EVENT = "execution.approval_recorded"
BUDGET_OVERRIDE_EVENT = "execution.budget_override"
FLOW_HANDOFF_EVENT = "execution.flow_handoff"


@dataclass(frozen=True)
class AuditEventSpec:
    event_type: str
    description: str
    severity: str
    retention_days: int
    required_fields: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_type": self.event_type,
            "description": self.description,
            "severity": self.severity,
            "retention_days": self.retention_days,
            "required_fields": list(self.required_fields),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AuditEventSpec":
        return cls(
            event_type=str(data["event_type"]),
            description=str(data["description"]),
            severity=str(data["severity"]),
            retention_days=int(data["retention_days"]),
            required_fields=[str(value) for value in data.get("required_fields", [])],
        )


P0_AUDIT_EVENT_SPECS: List[AuditEventSpec] = [
    AuditEventSpec(
        event_type=SCHEDULER_DECISION_EVENT,
        description="Records the scheduler routing decision and risk context for a run.",
        severity="info",
        retention_days=180,
        required_fields=["task_id", "run_id", "medium", "approved", "reason", "risk_level", "risk_score"],
    ),
    AuditEventSpec(
        event_type=MANUAL_TAKEOVER_EVENT,
        description="Captures escalation into a human takeover queue.",
        severity="warn",
        retention_days=365,
        required_fields=["task_id", "run_id", "target_team", "reason", "requested_by", "required_approvals"],
    ),
    AuditEventSpec(
        event_type=APPROVAL_RECORDED_EVENT,
        description="Records explicit human approvals attached to a run or acceptance decision.",
        severity="info",
        retention_days=365,
        required_fields=["task_id", "run_id", "approvals", "approval_count", "acceptance_status"],
    ),
    AuditEventSpec(
        event_type=BUDGET_OVERRIDE_EVENT,
        description="Captures a manual override to the run budget envelope.",
        severity="warn",
        retention_days=365,
        required_fields=["task_id", "run_id", "requested_budget", "approved_budget", "override_actor", "reason"],
    ),
    AuditEventSpec(
        event_type=FLOW_HANDOFF_EVENT,
        description="Captures ownership transfer between automated flow stages and teams.",
        severity="info",
        retention_days=180,
        required_fields=["task_id", "run_id", "source_stage", "target_team", "reason", "collaboration_mode"],
    ),
]

_SPEC_BY_EVENT = {spec.event_type: spec for spec in P0_AUDIT_EVENT_SPECS}


def get_audit_event_spec(event_type: str) -> Optional[AuditEventSpec]:
    return _SPEC_BY_EVENT.get(event_type)


def missing_required_fields(event_type: str, details: Dict[str, object]) -> List[str]:
    spec = get_audit_event_spec(event_type)
    if spec is None:
        return []
    return [field_name for field_name in spec.required_fields if field_name not in details]


_install_support_module(
    "audit_events",
    SCHEDULER_DECISION_EVENT=SCHEDULER_DECISION_EVENT,
    MANUAL_TAKEOVER_EVENT=MANUAL_TAKEOVER_EVENT,
    APPROVAL_RECORDED_EVENT=APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT=BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT=FLOW_HANDOFF_EVENT,
    AuditEventSpec=AuditEventSpec,
    P0_AUDIT_EVENT_SPECS=P0_AUDIT_EVENT_SPECS,
    get_audit_event_spec=get_audit_event_spec,
    missing_required_fields=missing_required_fields,
)


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def sha256_file(path: str) -> str:
    file_path = Path(path)
    if not file_path.exists() or not file_path.is_file():
        return ""

    digest = hashlib.sha256()
    with file_path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(8192), b""):
            digest.update(chunk)
    return digest.hexdigest()


@dataclass
class LogEntry:
    level: str
    message: str
    timestamp: str = field(default_factory=utc_now)
    context: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "level": self.level,
            "message": self.message,
            "timestamp": self.timestamp,
            "context": self.context,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "LogEntry":
        return cls(
            level=data["level"],
            message=data["message"],
            timestamp=data.get("timestamp", utc_now()),
            context=data.get("context", {}),
        )


@dataclass
class TraceEntry:
    span: str
    status: str
    timestamp: str = field(default_factory=utc_now)
    attributes: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "span": self.span,
            "status": self.status,
            "timestamp": self.timestamp,
            "attributes": self.attributes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TraceEntry":
        return cls(
            span=data["span"],
            status=data["status"],
            timestamp=data.get("timestamp", utc_now()),
            attributes=data.get("attributes", {}),
        )


@dataclass
class ArtifactRecord:
    name: str
    kind: str
    path: str
    timestamp: str = field(default_factory=utc_now)
    sha256: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "kind": self.kind,
            "path": self.path,
            "timestamp": self.timestamp,
            "sha256": self.sha256,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ArtifactRecord":
        return cls(
            name=data["name"],
            kind=data["kind"],
            path=data["path"],
            timestamp=data.get("timestamp", utc_now()),
            sha256=data.get("sha256", ""),
            metadata=data.get("metadata", {}),
        )


@dataclass
class AuditEntry:
    action: str
    actor: str
    outcome: str
    timestamp: str = field(default_factory=utc_now)
    details: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "action": self.action,
            "actor": self.actor,
            "outcome": self.outcome,
            "timestamp": self.timestamp,
            "details": self.details,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "AuditEntry":
        return cls(
            action=data["action"],
            actor=data["actor"],
            outcome=data["outcome"],
            timestamp=data.get("timestamp", utc_now()),
            details=data.get("details", {}),
        )


@dataclass
class GitSyncTelemetry:
    status: str = "unknown"
    failure_category: str = ""
    summary: str = ""
    branch: str = ""
    remote: str = "origin"
    remote_ref: str = ""
    ahead_by: int = 0
    behind_by: int = 0
    dirty_paths: List[str] = field(default_factory=list)
    auth_target: str = ""
    timestamp: str = field(default_factory=utc_now)

    @property
    def ok(self) -> bool:
        return self.status == "synced"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "status": self.status,
            "failure_category": self.failure_category,
            "summary": self.summary,
            "branch": self.branch,
            "remote": self.remote,
            "remote_ref": self.remote_ref,
            "ahead_by": self.ahead_by,
            "behind_by": self.behind_by,
            "dirty_paths": list(self.dirty_paths),
            "auth_target": self.auth_target,
            "timestamp": self.timestamp,
            "ok": self.ok,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "GitSyncTelemetry":
        return cls(
            status=str(data.get("status", "unknown")),
            failure_category=str(data.get("failure_category", "")),
            summary=str(data.get("summary", "")),
            branch=str(data.get("branch", "")),
            remote=str(data.get("remote", "origin")),
            remote_ref=str(data.get("remote_ref", "")),
            ahead_by=int(data.get("ahead_by", 0)),
            behind_by=int(data.get("behind_by", 0)),
            dirty_paths=[str(item) for item in data.get("dirty_paths", [])],
            auth_target=str(data.get("auth_target", "")),
            timestamp=data.get("timestamp", utc_now()),
        )


@dataclass
class PullRequestFreshness:
    pr_number: Optional[int] = None
    pr_url: str = ""
    branch_state: str = "unknown"
    body_state: str = "unknown"
    branch_head_sha: str = ""
    pr_head_sha: str = ""
    expected_body_digest: str = ""
    actual_body_digest: str = ""
    checked_at: str = field(default_factory=utc_now)

    @property
    def fresh(self) -> bool:
        return self.branch_state == "in-sync" and self.body_state == "fresh"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "pr_number": self.pr_number,
            "pr_url": self.pr_url,
            "branch_state": self.branch_state,
            "body_state": self.body_state,
            "branch_head_sha": self.branch_head_sha,
            "pr_head_sha": self.pr_head_sha,
            "expected_body_digest": self.expected_body_digest,
            "actual_body_digest": self.actual_body_digest,
            "checked_at": self.checked_at,
            "fresh": self.fresh,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "PullRequestFreshness":
        pr_number = data.get("pr_number")
        return cls(
            pr_number=int(pr_number) if pr_number is not None else None,
            pr_url=str(data.get("pr_url", "")),
            branch_state=str(data.get("branch_state", "unknown")),
            body_state=str(data.get("body_state", "unknown")),
            branch_head_sha=str(data.get("branch_head_sha", "")),
            pr_head_sha=str(data.get("pr_head_sha", "")),
            expected_body_digest=str(data.get("expected_body_digest", "")),
            actual_body_digest=str(data.get("actual_body_digest", "")),
            checked_at=data.get("checked_at", utc_now()),
        )


@dataclass
class RepoSyncAudit:
    sync: GitSyncTelemetry = field(default_factory=GitSyncTelemetry)
    pull_request: PullRequestFreshness = field(default_factory=PullRequestFreshness)

    @property
    def summary(self) -> str:
        parts = [f"sync={self.sync.status}"]
        if self.sync.failure_category:
            parts.append(f"failure={self.sync.failure_category}")
        parts.append(f"pr-branch={self.pull_request.branch_state}")
        parts.append(f"pr-body={self.pull_request.body_state}")
        return ", ".join(parts)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "sync": self.sync.to_dict(),
            "pull_request": self.pull_request.to_dict(),
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoSyncAudit":
        return cls(
            sync=GitSyncTelemetry.from_dict(data.get("sync", {})),
            pull_request=PullRequestFreshness.from_dict(data.get("pull_request", {})),
        )


@dataclass
class RunCloseout:
    validation_evidence: List[str] = field(default_factory=list)
    git_push_succeeded: bool = False
    git_push_output: str = ""
    git_log_stat_output: str = ""
    repo_sync_audit: Optional[RepoSyncAudit] = None
    run_commit_links: List[RunCommitLink] = field(default_factory=list)
    accepted_commit_hash: str = ""
    timestamp: str = field(default_factory=utc_now)

    @property
    def complete(self) -> bool:
        return bool(self.validation_evidence) and self.git_push_succeeded and bool(self.git_log_stat_output.strip())

    def to_dict(self) -> Dict[str, Any]:
        return {
            "validation_evidence": self.validation_evidence,
            "git_push_succeeded": self.git_push_succeeded,
            "git_push_output": self.git_push_output,
            "git_log_stat_output": self.git_log_stat_output,
            "repo_sync_audit": self.repo_sync_audit.to_dict() if self.repo_sync_audit else None,
            "run_commit_links": [link.to_dict() for link in self.run_commit_links],
            "accepted_commit_hash": self.accepted_commit_hash,
            "timestamp": self.timestamp,
            "complete": self.complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RunCloseout":
        return cls(
            validation_evidence=data.get("validation_evidence", []),
            git_push_succeeded=data.get("git_push_succeeded", False),
            git_push_output=data.get("git_push_output", ""),
            git_log_stat_output=data.get("git_log_stat_output", ""),
            repo_sync_audit=RepoSyncAudit.from_dict(data["repo_sync_audit"]) if data.get("repo_sync_audit") else None,
            run_commit_links=[RunCommitLink.from_dict(item) for item in data.get("run_commit_links", [])],
            accepted_commit_hash=str(data.get("accepted_commit_hash", "")),
            timestamp=data.get("timestamp", utc_now()),
        )


@dataclass
class TaskRun:
    run_id: str
    task_id: str
    source: str
    title: str
    medium: str
    started_at: str = field(default_factory=utc_now)
    ended_at: str = ""
    status: str = "running"
    summary: str = ""
    logs: List[LogEntry] = field(default_factory=list)
    traces: List[TraceEntry] = field(default_factory=list)
    artifacts: List[ArtifactRecord] = field(default_factory=list)
    audits: List[AuditEntry] = field(default_factory=list)
    closeout: RunCloseout = field(default_factory=RunCloseout)

    @classmethod
    def from_task(cls, task: Task, run_id: str, medium: str) -> "TaskRun":
        return cls(
            run_id=run_id,
            task_id=task.task_id,
            source=task.source,
            title=task.title,
            medium=medium,
        )

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TaskRun":
        return cls(
            run_id=data["run_id"],
            task_id=data["task_id"],
            source=data["source"],
            title=data["title"],
            medium=data["medium"],
            started_at=data.get("started_at", utc_now()),
            ended_at=data.get("ended_at", ""),
            status=data.get("status", "running"),
            summary=data.get("summary", ""),
            logs=[LogEntry.from_dict(entry) for entry in data.get("logs", [])],
            traces=[TraceEntry.from_dict(entry) for entry in data.get("traces", [])],
            artifacts=[ArtifactRecord.from_dict(entry) for entry in data.get("artifacts", [])],
            audits=[AuditEntry.from_dict(entry) for entry in data.get("audits", [])],
            closeout=RunCloseout.from_dict(data.get("closeout", {})),
        )

    def log(self, level: str, message: str, **context: Any) -> None:
        self.logs.append(LogEntry(level=level, message=message, context=context))

    def trace(self, span: str, status: str, **attributes: Any) -> None:
        self.traces.append(TraceEntry(span=span, status=status, attributes=attributes))

    def register_artifact(self, name: str, kind: str, path: str, **metadata: Any) -> None:
        digest = sha256_file(path)
        self.artifacts.append(
            ArtifactRecord(
                name=name,
                kind=kind,
                path=path,
                sha256=digest,
                metadata=metadata,
            )
        )
        self.audit(
            "artifact.registered",
            "task-run",
            "recorded",
            artifact_name=name,
            artifact_kind=kind,
            path=path,
            sha256=digest,
        )

    def audit(self, action: str, actor: str, outcome: str, **details: Any) -> None:
        self.audits.append(AuditEntry(action=action, actor=actor, outcome=outcome, details=details))

    def audit_spec_event(self, action: str, actor: str, outcome: str, **details: Any) -> None:
        missing = missing_required_fields(action, details)
        if missing:
            missing_text = ", ".join(missing)
            raise ValueError(f"audit event {action} missing required fields: {missing_text}")
        self.audit(action, actor, outcome, **details)

    def add_comment(
        self,
        *,
        author: str,
        body: str,
        mentions: Optional[List[str]] = None,
        anchor: str = "",
        status: str = "open",
        comment_id: str = "",
        surface: str = "run",
    ) -> CollaborationComment:
        resolved_id = comment_id or (
            f"{self.run_id}-comment-{len([audit for audit in self.audits if audit.action == 'collaboration.comment']) + 1}"
        )
        comment = CollaborationComment(
            comment_id=resolved_id,
            author=author,
            body=body,
            mentions=list(mentions or []),
            anchor=anchor,
            status=status,
        )
        self.audit(
            "collaboration.comment",
            author,
            "recorded",
            surface=surface,
            comment_id=comment.comment_id,
            body=comment.body,
            mentions=comment.mentions,
            anchor=comment.anchor,
            status=comment.status,
        )
        return comment

    def add_decision_note(
        self,
        *,
        author: str,
        summary: str,
        outcome: str,
        mentions: Optional[List[str]] = None,
        related_comment_ids: Optional[List[str]] = None,
        follow_up: str = "",
        decision_id: str = "",
        surface: str = "run",
    ) -> DecisionNote:
        resolved_id = decision_id or (
            f"{self.run_id}-decision-{len([audit for audit in self.audits if audit.action == 'collaboration.decision']) + 1}"
        )
        decision = DecisionNote(
            decision_id=resolved_id,
            author=author,
            outcome=outcome,
            summary=summary,
            mentions=list(mentions or []),
            related_comment_ids=list(related_comment_ids or []),
            follow_up=follow_up,
        )
        self.audit(
            "collaboration.decision",
            author,
            outcome,
            surface=surface,
            decision_id=decision.decision_id,
            summary=decision.summary,
            mentions=decision.mentions,
            related_comment_ids=decision.related_comment_ids,
            follow_up=decision.follow_up,
        )
        return decision

    def record_closeout(
        self,
        *,
        validation_evidence: List[str],
        git_push_succeeded: bool,
        git_push_output: str = "",
        git_log_stat_output: str = "",
        repo_sync_audit: Optional[RepoSyncAudit] = None,
        run_commit_links: Optional[List[RunCommitLink]] = None,
    ) -> None:
        links = list(run_commit_links or [])
        binding = bind_run_commits(links) if links else None
        self.closeout = RunCloseout(
            validation_evidence=list(validation_evidence),
            git_push_succeeded=git_push_succeeded,
            git_push_output=git_push_output,
            git_log_stat_output=git_log_stat_output,
            repo_sync_audit=repo_sync_audit,
            run_commit_links=links,
            accepted_commit_hash=binding.accepted_commit_hash if binding else "",
        )
        self.audit(
            "closeout.recorded",
            "task-run",
            "recorded",
            validation_evidence_count=len(validation_evidence),
            git_push_succeeded=git_push_succeeded,
            git_log_stat_captured=bool(git_log_stat_output.strip()),
            has_repo_sync_audit=repo_sync_audit is not None,
        )

    def finalize(self, status: str, summary: str) -> None:
        self.status = status
        self.summary = summary
        self.ended_at = utc_now()

    def to_dict(self) -> Dict[str, Any]:
        return {
            "run_id": self.run_id,
            "task_id": self.task_id,
            "source": self.source,
            "title": self.title,
            "medium": self.medium,
            "started_at": self.started_at,
            "ended_at": self.ended_at,
            "status": self.status,
            "summary": self.summary,
            "logs": [entry.to_dict() for entry in self.logs],
            "traces": [entry.to_dict() for entry in self.traces],
            "artifacts": [entry.to_dict() for entry in self.artifacts],
            "audits": [entry.to_dict() for entry in self.audits],
            "closeout": self.closeout.to_dict(),
        }


class ObservabilityLedger:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def load(self) -> List[Dict[str, Any]]:
        if not self.storage_path.exists():
            return []
        return json.loads(self.storage_path.read_text())

    def _write_entries(self, entries: List[Dict[str, Any]]) -> None:
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps(entries, ensure_ascii=False, indent=2))

    def append(self, run: TaskRun) -> None:
        self.upsert(run)

    def upsert(self, run: TaskRun) -> None:
        entries = self.load()
        serialized = run.to_dict()
        for index, entry in enumerate(entries):
            if entry.get("run_id") == run.run_id:
                entries[index] = serialized
                self._write_entries(entries)
                return
        entries.append(serialized)
        self._write_entries(entries)

    def load_runs(self) -> List[TaskRun]:
        return [TaskRun.from_dict(entry) for entry in self.load()]


_install_support_module(
    "observability",
    utc_now=utc_now,
    sha256_file=sha256_file,
    LogEntry=LogEntry,
    TraceEntry=TraceEntry,
    ArtifactRecord=ArtifactRecord,
    AuditEntry=AuditEntry,
    GitSyncTelemetry=GitSyncTelemetry,
    PullRequestFreshness=PullRequestFreshness,
    RepoSyncAudit=RepoSyncAudit,
    RunCloseout=RunCloseout,
    TaskRun=TaskRun,
    ObservabilityLedger=ObservabilityLedger,
)

_install_exec_support_module(
    "runtime",
    r'''from __future__ import annotations

import heapq
import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, List, Optional, Tuple

from .models import Task


@dataclass(frozen=True)
class DeadLetterEntry:
    task: Task
    reason: str = ""

    def to_dict(self) -> dict:
        return {"task_id": self.task.task_id, "task": self.task.to_dict(), "reason": self.reason}

    @classmethod
    def from_dict(cls, data: dict) -> "DeadLetterEntry":
        return cls(
            task=Task.from_dict(data["task"]),
            reason=str(data.get("reason", "")),
        )


class PersistentTaskQueue:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)
        self._heap: List[Tuple[int, str, dict]] = []
        self._dead_letters: Dict[str, DeadLetterEntry] = {}
        self._load()

    def _load(self) -> None:
        self._heap = []
        self._dead_letters = {}
        if not self.storage_path.exists():
            return
        data = json.loads(self.storage_path.read_text(encoding="utf-8"))

        if isinstance(data, list):
            queue_items = data
            dead_letter_items: List[dict] = []
        else:
            queue_items = data.get("queue", [])
            dead_letter_items = data.get("dead_letters", [])

        for item in queue_items:
            heapq.heappush(self._heap, (item["priority"], item["task_id"], item["task"]))
        for item in dead_letter_items:
            entry = DeadLetterEntry.from_dict(item)
            self._dead_letters[entry.task.task_id] = entry

    def _save(self) -> None:
        data = {
            "queue": [{"priority": p, "task_id": tid, "task": task} for (p, tid, task) in sorted(self._heap)],
            "dead_letters": [entry.to_dict() for _, entry in sorted(self._dead_letters.items())],
        }
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        tmp_path = self.storage_path.with_name(f"{self.storage_path.name}.tmp")
        tmp_path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        tmp_path.replace(self.storage_path)

    def _drop_queued_task(self, task_id: str) -> None:
        if not any(queued_task_id == task_id for _, queued_task_id, _ in self._heap):
            return
        self._heap = [
            (priority, queued_task_id, task)
            for priority, queued_task_id, task in self._heap
            if queued_task_id != task_id
        ]
        heapq.heapify(self._heap)

    def enqueue(self, task: Task) -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters.pop(task.task_id, None)
        heapq.heappush(self._heap, (int(task.priority), task.task_id, task.to_dict()))
        self._save()

    def dequeue(self) -> Optional[dict]:
        if not self._heap:
            return None
        _priority, _task_id, task = heapq.heappop(self._heap)
        self._save()
        return task

    def dequeue_task(self) -> Optional[Task]:
        task = self.dequeue()
        if task is None:
            return None
        return Task.from_dict(task)

    def dead_letter(self, task: Task, reason: str = "") -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters[task.task_id] = DeadLetterEntry(task=task, reason=reason)
        self._save()

    def list_dead_letters(self) -> List[DeadLetterEntry]:
        return [entry for _, entry in sorted(self._dead_letters.items())]

    def retry_dead_letter(self, task_id: str) -> bool:
        entry = self._dead_letters.pop(task_id, None)
        if entry is None:
            return False
        self._drop_queued_task(task_id)
        heapq.heappush(self._heap, (int(entry.task.priority), entry.task.task_id, entry.task.to_dict()))
        self._save()
        return True

    def size(self) -> int:
        return len(self._heap)

    def peek_tasks(self) -> List[Task]:
        return [Task.from_dict(task) for (_priority, _task_id, task) in sorted(self._heap)]


@dataclass
class DepartmentHandoff:
    department: str
    reason: str
    required_tools: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)


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


@dataclass
class HandoffRequest:
    target_team: str
    reason: str
    status: str = "pending"
    required_approvals: List[str] = field(default_factory=list)


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
''',
    [
        "DeadLetterEntry",
        "PersistentTaskQueue",
        "DepartmentHandoff",
        "OrchestrationPlan",
        "HandoffRequest",
        "OrchestrationPolicyDecision",
    ],
)

from .reports import (
    AutoTriageCenter,
    ConsoleAction,
    BillingEntitlementsPage,
    BillingRunCharge,
    DocumentationArtifact,
    FinalDeliveryChecklist,
    LaunchChecklist,
    LaunchChecklistItem,
    NarrativeSection,
    IssueClosureDecision,
    OrchestrationCanvas,
    OrchestrationPortfolio,
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    ReportStudio,
    ReportStudioArtifacts,
    SharedViewContext,
    SharedViewFilter,
    TakeoverQueue,
    TakeoverRequest,
    TriageFeedbackRecord,
    TriageFinding,
    TriageInboxItem,
    TriageSimilarityEvidence,
    TriageSuggestion,
    build_auto_triage_center,
    build_console_actions,
    build_billing_entitlements_page,
    build_billing_entitlements_page_from_ledger,
    build_final_delivery_checklist,
    build_launch_checklist,
    build_orchestration_canvas,
    build_orchestration_canvas_from_ledger_entry,
    build_orchestration_portfolio,
    build_orchestration_portfolio_from_ledger,
    build_takeover_queue_from_ledger,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_console_actions,
    render_billing_entitlements_page,
    render_billing_entitlements_report,
    render_final_delivery_checklist_report,
    render_launch_checklist_report,
    render_orchestration_canvas,
    render_orchestration_overview_page,
    render_orchestration_portfolio_report,
    render_issue_validation_report,
    render_report_studio_html,
    render_report_studio_plain_text,
    render_report_studio_report,
    render_takeover_queue_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_repo_sync_audit_report,
    render_task_run_detail_page,
    render_task_run_report,
    validation_report_exists,
    write_report,
    write_report_studio_bundle,
)


@dataclass
class EvaluationCriterion:
    name: str
    weight: int
    passed: bool
    detail: str


@dataclass
class BenchmarkCase:
    case_id: str
    task: Task
    expected_medium: Optional[str] = None
    expected_approved: Optional[bool] = None
    expected_status: Optional[str] = None
    require_report: bool = False


@dataclass
class ReplayRecord:
    task: Task
    run_id: str
    medium: str
    approved: bool
    status: str

    @classmethod
    def from_execution(cls, task: Task, run_id: str, record: "ExecutionRecord") -> "ReplayRecord":
        return cls(
            task=task,
            run_id=run_id,
            medium=record.decision.medium,
            approved=record.decision.approved,
            status=record.run.status,
        )


@dataclass
class ReplayOutcome:
    matched: bool
    replay_record: ReplayRecord
    mismatches: List[str] = field(default_factory=list)
    report_path: Optional[str] = None


@dataclass
class BenchmarkResult:
    case_id: str
    score: int
    passed: bool
    criteria: List[EvaluationCriterion]
    record: "ExecutionRecord"
    replay: ReplayOutcome
    detail_page_path: Optional[str] = None


@dataclass
class BenchmarkComparison:
    case_id: str
    baseline_score: int
    current_score: int
    delta: int
    changed: bool


@dataclass
class BenchmarkSuiteResult:
    results: List[BenchmarkResult]
    version: str = "current"

    @property
    def score(self) -> int:
        if not self.results:
            return 0
        return round(sum(result.score for result in self.results) / len(self.results))

    @property
    def passed(self) -> bool:
        return all(result.passed for result in self.results)

    def compare(self, baseline: "BenchmarkSuiteResult") -> List[BenchmarkComparison]:
        baseline_by_case = {result.case_id: result for result in baseline.results}
        comparisons = []
        for result in self.results:
            baseline_result = baseline_by_case.get(result.case_id)
            baseline_score = baseline_result.score if baseline_result else 0
            delta = result.score - baseline_score
            comparisons.append(
                BenchmarkComparison(
                    case_id=result.case_id,
                    baseline_score=baseline_score,
                    current_score=result.score,
                    delta=delta,
                    changed=delta != 0,
                )
            )
        return comparisons


class BenchmarkRunner:
    def __init__(self, scheduler: Optional["Scheduler"] = None, storage_dir: Optional[str] = None):
        self.scheduler = scheduler or Scheduler()
        self.storage_dir = Path(storage_dir) if storage_dir else None

    def run_case(self, case: BenchmarkCase) -> BenchmarkResult:
        ledger = ObservabilityLedger(str(self._case_path(case.case_id, "ledger.json")))
        report_path = None
        if case.require_report:
            report_path = str(self._case_path(case.case_id, "task-run.md"))

        run_id = f"benchmark-{case.case_id}"
        record = self.scheduler.execute(
            case.task,
            run_id=run_id,
            ledger=ledger,
            report_path=report_path,
            actor="benchmark-runner",
        )
        criteria = self._evaluate(case, record)
        replay = self.replay(ReplayRecord.from_execution(case.task, run_id, record))
        total_weight = sum(item.weight for item in criteria)
        earned_weight = sum(item.weight for item in criteria if item.passed)
        score = round((earned_weight / total_weight) * 100) if total_weight else 0
        passed = all(item.passed for item in criteria) and replay.matched
        detail_page_path = None
        if self.storage_dir is not None:
            detail_page_path = str(self._case_path(case.case_id, "run-detail.html"))
            write_report(detail_page_path, render_run_replay_index_page(case.case_id, record, replay, criteria))
        return BenchmarkResult(
            case_id=case.case_id,
            score=score,
            passed=passed,
            criteria=criteria,
            record=record,
            replay=replay,
            detail_page_path=detail_page_path,
        )

    def run_suite(self, cases: List[BenchmarkCase], version: str = "current") -> BenchmarkSuiteResult:
        return BenchmarkSuiteResult(results=[self.run_case(case) for case in cases], version=version)

    def replay(self, replay_record: ReplayRecord) -> ReplayOutcome:
        ledger = ObservabilityLedger(str(self._case_path(replay_record.run_id, "replay-ledger.json")))
        replayed = self.scheduler.execute(
            replay_record.task,
            run_id=f"{replay_record.run_id}-replay",
            ledger=ledger,
            actor="benchmark-replay",
        )
        observed = ReplayRecord.from_execution(replay_record.task, replay_record.run_id, replayed)
        mismatches = []
        if observed.medium != replay_record.medium:
            mismatches.append(f"medium expected {replay_record.medium} got {observed.medium}")
        if observed.approved != replay_record.approved:
            mismatches.append(f"approved expected {replay_record.approved} got {observed.approved}")
        if observed.status != replay_record.status:
            mismatches.append(f"status expected {replay_record.status} got {observed.status}")
        report_path = None
        if self.storage_dir is not None:
            report_path = str(self._case_path(replay_record.run_id, "replay.html"))
            write_report(report_path, render_replay_detail_page(replay_record, observed, mismatches))
        return ReplayOutcome(
            matched=not mismatches,
            replay_record=observed,
            mismatches=mismatches,
            report_path=report_path,
        )

    def _evaluate(self, case: BenchmarkCase, record: "ExecutionRecord") -> List[EvaluationCriterion]:
        return [
            self._criterion(name="decision-medium", weight=40, expected=case.expected_medium, actual=record.decision.medium),
            self._criterion(name="approval-gate", weight=30, expected=case.expected_approved, actual=record.decision.approved),
            self._criterion(name="final-status", weight=20, expected=case.expected_status, actual=record.run.status),
            EvaluationCriterion(
                name="report-artifact",
                weight=10,
                passed=(not case.require_report) or bool(record.report_path),
                detail=("report emitted" if (not case.require_report) or bool(record.report_path) else "report missing"),
            ),
        ]

    def _criterion(self, name: str, weight: int, expected: Optional[object], actual: object) -> EvaluationCriterion:
        if expected is None:
            return EvaluationCriterion(name=name, weight=weight, passed=True, detail="not asserted")
        return EvaluationCriterion(name=name, weight=weight, passed=(expected == actual), detail=f"expected {expected} got {actual}")

    def _case_path(self, case_id: str, file_name: str) -> Path:
        if self.storage_dir is None:
            return Path(file_name)
        return self.storage_dir / case_id / file_name


def render_benchmark_suite_report(
    suite: BenchmarkSuiteResult,
    baseline: Optional[BenchmarkSuiteResult] = None,
) -> str:
    lines = [
        "# Benchmark Suite Report",
        "",
        f"- Version: {suite.version}",
        f"- Cases: {len(suite.results)}",
        f"- Passed: {suite.passed}",
        f"- Score: {suite.score}",
        "",
        "## Cases",
        "",
    ]

    if suite.results:
        lines.extend(f"- {result.case_id}: score={result.score} passed={result.passed} replay={result.replay.matched}" for result in suite.results)
    else:
        lines.append("- None")

    lines.extend(["", "## Comparison", ""])
    if baseline is None:
        lines.append("- No baseline provided")
    else:
        lines.append(f"- Baseline Version: {baseline.version}")
        lines.append(f"- Score Delta: {suite.score - baseline.score}")
        comparisons = suite.compare(baseline)
        if comparisons:
            lines.extend(
                f"- {comparison.case_id}: baseline={comparison.baseline_score} current={comparison.current_score} delta={comparison.delta}"
                for comparison in comparisons
            )
        else:
            lines.append("- No comparable cases")

    return "\n".join(lines) + "\n"


def render_replay_detail_page(expected: ReplayRecord, observed: ReplayRecord, mismatches: List[str]) -> str:
    tone = "accent" if not mismatches else "danger"
    timeline_events = [
        RunDetailEvent(
            event_id="compare-medium",
            lane="comparison",
            title="Medium",
            timestamp="compare-1",
            status="matched" if expected.medium == observed.medium else "mismatch",
            summary=f"expected {expected.medium} | observed {observed.medium}",
            details=[f"expected={expected.medium}", f"observed={observed.medium}"],
        ),
        RunDetailEvent(
            event_id="compare-approved",
            lane="comparison",
            title="Approval",
            timestamp="compare-2",
            status="matched" if expected.approved == observed.approved else "mismatch",
            summary=f"expected {expected.approved} | observed {observed.approved}",
            details=[f"expected={expected.approved}", f"observed={observed.approved}"],
        ),
        RunDetailEvent(
            event_id="compare-status",
            lane="comparison",
            title="Status",
            timestamp="compare-3",
            status="matched" if expected.status == observed.status else "mismatch",
            summary=f"expected {expected.status} | observed {observed.status}",
            details=[f"expected={expected.status}", f"observed={observed.status}"],
        ),
        *[
            RunDetailEvent(
                event_id=f"mismatch-{index}",
                lane="replay",
                title=f"Mismatch {index + 1}",
                timestamp=f"compare-{index + 4}",
                status="mismatch",
                summary=item,
                details=[item],
            )
            for index, item in enumerate(mismatches)
        ],
    ]
    comparison_html = f"""
    <section class="surface">
      <h2>Split Comparison</h2>
      <p>Side-by-side replay comparison for task <strong>{escape(expected.task.task_id)}</strong> against baseline run <code>{escape(expected.run_id)}</code>.</p>
      <div class="resource-grid">
        <article class="resource-card">
          <span class="kicker">Baseline</span>
          <h3>Expected</h3>
          <p><code>medium={escape(expected.medium)}</code></p>
          <span class="resource-meta">approved={escape(str(expected.approved))} | status={escape(expected.status)}</span>
        </article>
        <article class="resource-card">
          <span class="kicker">Replay</span>
          <h3>Observed</h3>
          <p><code>medium={escape(observed.medium)}</code></p>
          <span class="resource-meta">approved={escape(str(observed.approved))} | status={escape(observed.status)}</span>
        </article>
      </div>
    </section>
    """
    mismatch_html = f"""
    <section class="surface">
      <h2>Replay Mismatches</h2>
      <p>Detailed mismatch list for the replay execution.</p>
      <ul>{''.join(f'<li>{escape(item)}</li>' for item in mismatches) or '<li>None</li>'}</ul>
    </section>
    """
    return render_run_detail_console(
        page_title=f"Replay Detail · {expected.run_id}",
        eyebrow="Replay Detail",
        hero_title=f"Replay Detail · {expected.task.task_id}",
        hero_summary="High-fidelity replay inspection with synced comparison timeline and split-view baseline versus observed execution state.",
        stats=[
            RunDetailStat("Run ID", expected.run_id),
            RunDetailStat("Task ID", expected.task.task_id),
            RunDetailStat("Expected Medium", expected.medium),
            RunDetailStat("Observed Medium", observed.medium, tone=tone),
            RunDetailStat("Replay", "matched" if not mismatches else "mismatch", tone=tone),
            RunDetailStat("Mismatches", str(len(mismatches)), tone=tone),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", comparison_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Field-by-field replay comparison with a synced inspector for each expectation and mismatch.",
                    timeline_events,
                ),
            ),
            RunDetailTab("comparison", "Split View", comparison_html),
            RunDetailTab("replay", "Replay", mismatch_html),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Replay detail pages do not emit standalone report files beyond the generated HTML page unless the caller persists additional artifacts.",
                    [],
                ),
            ),
        ],
        timeline_events=timeline_events,
    )


def render_run_replay_index_page(
    case_id: str,
    record: "ExecutionRecord",
    replay: ReplayOutcome,
    criteria: List[EvaluationCriterion],
) -> str:
    status_tone = "accent" if record.run.status == "approved" else "warning"
    if replay.mismatches:
        status_tone = "danger"

    report_path = record.report_path or "n/a"
    detail_path = str(Path(record.report_path).with_suffix(".html")) if record.report_path else "n/a"
    replay_path = replay.report_path or "n/a"

    criteria_events = [
        RunDetailEvent(
            event_id=f"criterion-{index}",
            lane="acceptance",
            title=item.name,
            timestamp=f"step-{index + 1}",
            status="passed" if item.passed else "failed",
            summary=item.detail,
            details=[f"weight={item.weight}", f"passed={item.passed}"],
        )
        for index, item in enumerate(criteria)
    ]
    mismatch_events = [
        RunDetailEvent(
            event_id=f"mismatch-{index}",
            lane="replay",
            title=f"Replay mismatch {index + 1}",
            timestamp=f"replay-{index + 1}",
            status="mismatch",
            summary=item,
            details=[item],
        )
        for index, item in enumerate(replay.mismatches)
    ]
    run_events = sorted(
        [
            *[
                RunDetailEvent(
                    event_id=f"log-{index}",
                    lane="log",
                    title=entry.message,
                    timestamp=entry.timestamp,
                    status=entry.level,
                    summary=f"log entry at {entry.timestamp}",
                    details=[f"{key}={value}" for key, value in sorted(entry.context.items())] or ["No structured context recorded."],
                )
                for index, entry in enumerate(record.run.logs)
            ],
            *[
                RunDetailEvent(
                    event_id=f"trace-{index}",
                    lane="trace",
                    title=entry.span,
                    timestamp=entry.timestamp,
                    status=entry.status,
                    summary=f"trace span {entry.span}",
                    details=[f"{key}={value}" for key, value in sorted(entry.attributes.items())] or ["No trace attributes recorded."],
                )
                for index, entry in enumerate(record.run.traces)
            ],
            *[
                RunDetailEvent(
                    event_id=f"audit-{index}",
                    lane="audit",
                    title=entry.action,
                    timestamp=entry.timestamp,
                    status=entry.outcome,
                    summary=f"audit by {entry.actor}",
                    details=[f"actor={entry.actor}", *[f"{key}={value}" for key, value in sorted(entry.details.items())]] or ["No audit details recorded."],
                )
                for index, entry in enumerate(record.run.audits)
            ],
            *criteria_events,
            *mismatch_events,
        ],
        key=lambda event: event.timestamp,
    )

    execution_resources = [
        RunDetailResource(name="Markdown report", kind="report", path=report_path, meta=["execution report"], tone="report"),
        RunDetailResource(name="Run detail page", kind="page", path=detail_path, meta=["task run detail"], tone="page"),
        RunDetailResource(name="Replay page", kind="page", path=replay_path, meta=[f"matched={replay.matched}"], tone="page"),
    ]

    overview_html = f"""
    <section class="surface">
      <h2>Overview</h2>
      <p>Benchmark case <strong>{escape(case_id)}</strong> executed task <strong>{escape(record.run.task_id)}</strong> with scheduler medium <strong>{escape(record.decision.medium)}</strong>.</p>
      <p class="meta">Replay matched={escape(str(replay.matched))} | mismatches={escape(str(len(replay.mismatches)))}</p>
    </section>
    """
    acceptance_html = f"""
    <section class="surface">
      <h2>Acceptance Criteria</h2>
      <p>Scored checks used to grade the run detail and replay execution path.</p>
      <ul>
        {''.join(f'<li><strong>{escape(item.name)}</strong>: {escape(item.detail)} | weight={item.weight} | passed={item.passed}</li>' for item in criteria) or '<li>None</li>'}
      </ul>
    </section>
    """
    replay_html = f"""
    <section class="surface">
      <h2>Replay</h2>
      <p>Replay status <strong>{escape('matched' if replay.matched else 'mismatch')}</strong> for baseline run <code>{escape(replay.replay_record.run_id)}</code>.</p>
      <ul>
        {''.join(f'<li>{escape(item)}</li>' for item in replay.mismatches) or '<li>None</li>'}
      </ul>
    </section>
    """

    return render_run_detail_console(
        page_title=f"Run Detail Index · {case_id}",
        eyebrow="Replay Console",
        hero_title=f"Run Detail Index · {case_id}",
        hero_summary="Benchmark execution, replay evidence, and acceptance criteria in a single operator-facing run console.",
        stats=[
            RunDetailStat("Task ID", record.run.task_id),
            RunDetailStat("Status", record.run.status, tone=status_tone),
            RunDetailStat("Medium", record.decision.medium, tone="accent" if record.decision.medium == "browser" else "default"),
            RunDetailStat("Replay", "matched" if replay.matched else "mismatch", tone="accent" if replay.matched else "danger"),
            RunDetailStat("Criteria", str(len(criteria))),
            RunDetailStat("Mismatches", str(len(replay.mismatches)), tone="danger" if replay.mismatches else "default"),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", overview_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Run logs, trace spans, audits, acceptance checks, and replay mismatches are merged into one synced timeline inspector.",
                    run_events,
                ),
            ),
            RunDetailTab("acceptance", "Acceptance", acceptance_html),
            RunDetailTab(
                "artifacts",
                "Artifacts",
                render_resource_grid(
                    "Artifacts",
                    "Generated reports and pages emitted for benchmark review and replay inspection.",
                    execution_resources,
                ),
            ),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Report-first view for markdown output and linked run/replay pages.",
                    [resource for resource in execution_resources if resource.kind == "report" or resource.name.endswith("page")],
                ),
            ),
            RunDetailTab("replay", "Replay", replay_html),
        ],
        timeline_events=run_events,
    )


_install_support_module(
    "evaluation",
    EvaluationCriterion=EvaluationCriterion,
    BenchmarkCase=BenchmarkCase,
    ReplayRecord=ReplayRecord,
    ReplayOutcome=ReplayOutcome,
    BenchmarkResult=BenchmarkResult,
    BenchmarkComparison=BenchmarkComparison,
    BenchmarkSuiteResult=BenchmarkSuiteResult,
    BenchmarkRunner=BenchmarkRunner,
    render_benchmark_suite_report=render_benchmark_suite_report,
    render_replay_detail_page=render_replay_detail_page,
    render_run_replay_index_page=render_run_replay_index_page,
)

PRIORITY_WEIGHTS = {"P0": 4, "P1": 3, "P2": 2, "P3": 1}
GOAL_STATUS_ORDER = {
    "done": 4,
    "on-track": 3,
    "at-risk": 2,
    "blocked": 1,
    "not-started": 0,
}


@dataclass(frozen=True)
class EvidenceLink:
    label: str
    target: str
    capability: str = ""
    note: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "label": self.label,
            "target": self.target,
            "capability": self.capability,
            "note": self.note,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EvidenceLink":
        return cls(
            label=str(data["label"]),
            target=str(data["target"]),
            capability=str(data.get("capability", "")),
            note=str(data.get("note", "")),
        )


@dataclass(frozen=True)
class CandidateEntry:
    candidate_id: str
    title: str
    theme: str
    priority: str
    owner: str
    outcome: str
    validation_command: str
    capabilities: List[str] = field(default_factory=list)
    evidence: List[str] = field(default_factory=list)
    evidence_links: List[EvidenceLink] = field(default_factory=list)
    dependencies: List[str] = field(default_factory=list)
    blockers: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> int:
        base = PRIORITY_WEIGHTS.get(self.priority.upper(), 0) * 25
        dependency_penalty = len(self.dependencies) * 10
        blocker_penalty = len(self.blockers) * 20
        evidence_bonus = min(len(self.evidence), 3) * 5
        return max(0, min(100, base + evidence_bonus - dependency_penalty - blocker_penalty))

    @property
    def ready(self) -> bool:
        return bool(self.capabilities) and bool(self.evidence) and not self.blockers

    def to_dict(self) -> Dict[str, object]:
        return {
            "candidate_id": self.candidate_id,
            "title": self.title,
            "theme": self.theme,
            "priority": self.priority,
            "owner": self.owner,
            "outcome": self.outcome,
            "validation_command": self.validation_command,
            "capabilities": list(self.capabilities),
            "evidence": list(self.evidence),
            "evidence_links": [link.to_dict() for link in self.evidence_links],
            "dependencies": list(self.dependencies),
            "blockers": list(self.blockers),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "CandidateEntry":
        return cls(
            candidate_id=str(data["candidate_id"]),
            title=str(data["title"]),
            theme=str(data["theme"]),
            priority=str(data["priority"]),
            owner=str(data["owner"]),
            outcome=str(data["outcome"]),
            validation_command=str(data["validation_command"]),
            capabilities=[str(item) for item in data.get("capabilities", [])],
            evidence=[str(item) for item in data.get("evidence", [])],
            evidence_links=[EvidenceLink.from_dict(item) for item in data.get("evidence_links", [])],
            dependencies=[str(item) for item in data.get("dependencies", [])],
            blockers=[str(item) for item in data.get("blockers", [])],
        )


@dataclass
class CandidateBacklog:
    epic_id: str
    title: str
    version: str
    candidates: List[CandidateEntry] = field(default_factory=list)

    @property
    def ranked_candidates(self) -> List[CandidateEntry]:
        return sorted(
            self.candidates,
            key=lambda candidate: (-candidate.readiness_score, candidate.candidate_id),
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "epic_id": self.epic_id,
            "title": self.title,
            "version": self.version,
            "candidates": [candidate.to_dict() for candidate in self.candidates],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "CandidateBacklog":
        return cls(
            epic_id=str(data["epic_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            candidates=[CandidateEntry.from_dict(item) for item in data.get("candidates", [])],
        )


@dataclass(frozen=True)
class EntryGate:
    gate_id: str
    name: str
    min_ready_candidates: int
    required_capabilities: List[str] = field(default_factory=list)
    required_evidence: List[str] = field(default_factory=list)
    required_baseline_version: str = ""
    max_blockers: int = 0

    def to_dict(self) -> Dict[str, object]:
        return {
            "gate_id": self.gate_id,
            "name": self.name,
            "min_ready_candidates": self.min_ready_candidates,
            "required_capabilities": list(self.required_capabilities),
            "required_evidence": list(self.required_evidence),
            "required_baseline_version": self.required_baseline_version,
            "max_blockers": self.max_blockers,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EntryGate":
        return cls(
            gate_id=str(data["gate_id"]),
            name=str(data["name"]),
            min_ready_candidates=int(data["min_ready_candidates"]),
            required_capabilities=[str(item) for item in data.get("required_capabilities", [])],
            required_evidence=[str(item) for item in data.get("required_evidence", [])],
            required_baseline_version=str(data.get("required_baseline_version", "")),
            max_blockers=int(data.get("max_blockers", 0)),
        )


@dataclass
class EntryGateDecision:
    gate_id: str
    passed: bool
    ready_candidate_ids: List[str] = field(default_factory=list)
    blocked_candidate_ids: List[str] = field(default_factory=list)
    missing_capabilities: List[str] = field(default_factory=list)
    missing_evidence: List[str] = field(default_factory=list)
    baseline_ready: bool = True
    baseline_findings: List[str] = field(default_factory=list)
    blocker_count: int = 0

    @property
    def summary(self) -> str:
        status = "PASS" if self.passed else "HOLD"
        return (
            f"{status}: ready={len(self.ready_candidate_ids)} "
            f"blocked={self.blocker_count} "
            f"missing_capabilities={len(self.missing_capabilities)} "
            f"missing_evidence={len(self.missing_evidence)} "
            f"baseline_findings={len(self.baseline_findings)}"
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "gate_id": self.gate_id,
            "passed": self.passed,
            "ready_candidate_ids": list(self.ready_candidate_ids),
            "blocked_candidate_ids": list(self.blocked_candidate_ids),
            "missing_capabilities": list(self.missing_capabilities),
            "missing_evidence": list(self.missing_evidence),
            "baseline_ready": self.baseline_ready,
            "baseline_findings": list(self.baseline_findings),
            "blocker_count": self.blocker_count,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "EntryGateDecision":
        return cls(
            gate_id=str(data["gate_id"]),
            passed=bool(data["passed"]),
            ready_candidate_ids=[str(item) for item in data.get("ready_candidate_ids", [])],
            blocked_candidate_ids=[str(item) for item in data.get("blocked_candidate_ids", [])],
            missing_capabilities=[str(item) for item in data.get("missing_capabilities", [])],
            missing_evidence=[str(item) for item in data.get("missing_evidence", [])],
            baseline_ready=bool(data.get("baseline_ready", True)),
            baseline_findings=[str(item) for item in data.get("baseline_findings", [])],
            blocker_count=int(data.get("blocker_count", 0)),
        )


class CandidatePlanner:
    def evaluate_gate(
        self,
        backlog: CandidateBacklog,
        gate: EntryGate,
        baseline_audit: Optional[ScopeFreezeAudit] = None,
    ) -> EntryGateDecision:
        ready_candidates = [candidate for candidate in backlog.ranked_candidates if candidate.ready]
        blocked_candidates = [candidate for candidate in backlog.candidates if candidate.blockers]
        provided_capabilities = {capability for candidate in ready_candidates for capability in candidate.capabilities}
        provided_evidence = {item for candidate in ready_candidates for item in candidate.evidence}
        missing_capabilities = [
            capability
            for capability in gate.required_capabilities
            if capability not in provided_capabilities
        ]
        missing_evidence = [
            item for item in gate.required_evidence if item not in provided_evidence
        ]
        baseline_findings = self._baseline_findings(gate, baseline_audit)
        baseline_ready = not baseline_findings
        passed = (
            len(ready_candidates) >= gate.min_ready_candidates
            and len(blocked_candidates) <= gate.max_blockers
            and not missing_capabilities
            and not missing_evidence
            and baseline_ready
        )
        return EntryGateDecision(
            gate_id=gate.gate_id,
            passed=passed,
            ready_candidate_ids=[candidate.candidate_id for candidate in ready_candidates],
            blocked_candidate_ids=[candidate.candidate_id for candidate in blocked_candidates],
            missing_capabilities=missing_capabilities,
            missing_evidence=missing_evidence,
            baseline_ready=baseline_ready,
            baseline_findings=baseline_findings,
            blocker_count=len(blocked_candidates),
        )

    def _baseline_findings(
        self,
        gate: EntryGate,
        baseline_audit: Optional[ScopeFreezeAudit],
    ) -> List[str]:
        if not gate.required_baseline_version:
            return []
        if baseline_audit is None:
            return [f"missing baseline audit for {gate.required_baseline_version}"]
        findings: List[str] = []
        if baseline_audit.version != gate.required_baseline_version:
            findings.append(
                f"baseline version mismatch: expected {gate.required_baseline_version}, got {baseline_audit.version}"
            )
        if not baseline_audit.release_ready:
            findings.append(
                f"baseline {baseline_audit.version} is not release ready ({baseline_audit.readiness_score:.1f})"
            )
        return findings


def render_candidate_backlog_report(
    backlog: CandidateBacklog,
    gate: EntryGate,
    decision: EntryGateDecision,
) -> str:
    lines = [
        "# V3 Candidate Backlog Report",
        "",
        f"- Epic: {backlog.epic_id} {backlog.title}",
        f"- Version: {backlog.version}",
        f"- Gate: {gate.name}",
        f"- Decision: {decision.summary}",
        "",
        "## Candidates",
    ]
    for candidate in backlog.ranked_candidates:
        lines.append(
            "- "
            f"{candidate.candidate_id}: {candidate.title} "
            f"priority={candidate.priority} owner={candidate.owner} "
            f"score={candidate.readiness_score} ready={candidate.ready}"
        )
        lines.append(
            "  "
            f"theme={candidate.theme} outcome={candidate.outcome} "
            f"capabilities={','.join(candidate.capabilities) or 'none'} "
            f"evidence={','.join(candidate.evidence) or 'none'} "
            f"blockers={','.join(candidate.blockers) or 'none'}"
        )
        lines.append(f"  validation={candidate.validation_command}")
        if candidate.dependencies:
            lines.append(f"  dependencies={','.join(candidate.dependencies)}")
        if candidate.evidence_links:
            lines.append("  evidence-links:")
            for link in candidate.evidence_links:
                qualifier = f" capability={link.capability}" if link.capability else ""
                note = f" note={link.note}" if link.note else ""
                lines.append(f"  - {link.label} -> {link.target}{qualifier}{note}")
    lines.extend(
        [
            "",
            "## Gate Findings",
            f"- Ready candidates: {', '.join(decision.ready_candidate_ids) or 'none'}",
            f"- Blocked candidates: {', '.join(decision.blocked_candidate_ids) or 'none'}",
            f"- Missing capabilities: {', '.join(decision.missing_capabilities) or 'none'}",
            f"- Missing evidence: {', '.join(decision.missing_evidence) or 'none'}",
            f"- Baseline ready: {decision.baseline_ready}",
            f"- Baseline findings: {', '.join(decision.baseline_findings) or 'none'}",
        ]
    )
    return "\n".join(lines)


def build_v3_candidate_backlog() -> CandidateBacklog:
    return CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Console release control center",
                theme="console-governance",
                priority="P0",
                owner="product-experience",
                outcome="Converge console shell governance, UI acceptance, and review-pack evidence into one release-control candidate.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_design_system.py "
                    "tests/test_console_ia.py tests/test_ui_review.py -q"
                ),
                capabilities=["release-gate", "console-shell", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="design-system-audit",
                        target="src/bigclaw/design_system.py",
                        capability="release-gate",
                        note="component inventory, accessibility, and UI acceptance coverage",
                    ),
                    EvidenceLink(
                        label="console-ia-contract",
                        target="src/bigclaw/console_ia.py",
                        capability="release-gate",
                        note="global navigation, top bar, filters, and state contracts",
                    ),
                    EvidenceLink(
                        label="ui-review-pack",
                        target="src/bigclaw/ui_review.py",
                        capability="release-gate",
                        note="review objectives, wireframes, interaction coverage, and open questions",
                    ),
                    EvidenceLink(
                        label="ui-acceptance-tests",
                        target="tests/test_design_system.py",
                        capability="release-gate",
                        note="role-permission, data accuracy, and performance audits",
                    ),
                    EvidenceLink(
                        label="console-shell-tests",
                        target="tests/test_console_ia.py",
                        capability="release-gate",
                        note="console shell and interaction draft release readiness",
                    ),
                    EvidenceLink(
                        label="review-pack-tests",
                        target="tests/test_ui_review.py",
                        capability="release-gate",
                        note="deterministic review packet validation",
                    ),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Operations command-center hardening",
                theme="ops-command-center",
                priority="P0",
                owner="engineering-operations",
                outcome="Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_operations.py "
                    "tests/test_saved_views.py tests/test_evaluation.py -q && "
                    "(cd bigclaw-go && go test ./internal/worker ./internal/workflow ./internal/scheduler)"
                ),
                capabilities=["ops-control", "saved-views", "rollback-simulation"],
                evidence=["weekly-review", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="command-center-src",
                        target="src/bigclaw/operations.py",
                        capability="ops-control",
                        note="queue control center, dashboard builder, weekly review, and regression surfaces",
                    ),
                    EvidenceLink(
                        label="command-center-tests",
                        target="tests/test_control_center.py",
                        capability="ops-control",
                        note="queue control center validation",
                    ),
                    EvidenceLink(
                        label="operations-tests",
                        target="tests/test_operations.py",
                        capability="ops-control",
                        note="dashboard, weekly report, regression, and version-center coverage",
                    ),
                    EvidenceLink(
                        label="approval-contract",
                        target="src/bigclaw/execution_contract.py",
                        capability="ops-control",
                        note="approval permission and API role coverage contract",
                    ),
                    EvidenceLink(
                        label="approval-workflow",
                        target="src/bigclaw/workflow.py",
                        capability="ops-control",
                        note="approval workflow and closeout flow wiring",
                    ),
                    EvidenceLink(
                        label="workflow-tests",
                        target="bigclaw-go/internal/workflow/engine_test.go",
                        capability="ops-control",
                        note="acceptance gate and workpad journal validation",
                    ),
                    EvidenceLink(
                        label="execution-flow-tests",
                        target="bigclaw-go/internal/worker/runtime_test.go",
                        capability="ops-control",
                        note="execution handoff, closeout, and routed runtime evidence",
                    ),
                    EvidenceLink(
                        label="saved-views-src",
                        target="src/bigclaw/__init__.py",
                        capability="saved-views",
                        note="saved views, digest subscriptions, governed filters, and compatibility submodule",
                    ),
                    EvidenceLink(
                        label="saved-views-tests",
                        target="tests/test_saved_views.py",
                        capability="saved-views",
                        note="saved-view audit coverage",
                    ),
                    EvidenceLink(
                        label="simulation-src",
                        target="src/bigclaw/evaluation.py",
                        capability="rollback-simulation",
                        note="simulation, replay, and comparison evidence",
                    ),
                    EvidenceLink(
                        label="simulation-tests",
                        target="tests/test_evaluation.py",
                        capability="rollback-simulation",
                        note="replay and benchmark validation",
                    ),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration-rollout",
                title="Agent orchestration rollout",
                theme="agent-orchestration",
                priority="P0",
                owner="orchestration-office",
                outcome="Carry entitlement-aware orchestration, handoff visibility, and commercialization proof into a candidate ready for release review.",
                validation_command=(
                    "PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_reports.py -q"
                ),
                capabilities=["commercialization", "handoff", "pilot-rollout"],
                evidence=["pilot-evidence", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="orchestration-plan-src",
                        target="src/bigclaw/orchestration.py",
                        capability="commercialization",
                        note="cross-team orchestration, entitlement-aware policy, and handoff decisions",
                    ),
                    EvidenceLink(
                        label="orchestration-report-src",
                        target="src/bigclaw/reports.py",
                        capability="commercialization",
                        note="orchestration canvas, portfolio rollups, and narrative exports",
                    ),
                    EvidenceLink(
                        label="orchestration-tests",
                        target="tests/test_orchestration.py",
                        capability="commercialization",
                        note="handoff and policy decision validation",
                    ),
                    EvidenceLink(
                        label="report-studio-tests",
                        target="tests/test_reports.py",
                        capability="commercialization",
                        note="report exports and downstream evidence sharing",
                    ),
                ],
            ),
        ],
    )


def build_v3_entry_gate() -> EntryGate:
    return EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
        max_blockers=0,
    )


@dataclass(frozen=True)
class WeeklyGoal:
    goal_id: str
    title: str
    owner: str
    status: str
    success_metric: str
    target_value: str
    current_value: str = ""
    dependencies: List[str] = field(default_factory=list)
    risks: List[str] = field(default_factory=list)

    @property
    def status_rank(self) -> int:
        return GOAL_STATUS_ORDER.get(self.status.strip().lower(), -1)

    @property
    def is_complete(self) -> bool:
        return self.status.strip().lower() == "done"

    @property
    def is_at_risk(self) -> bool:
        return self.status.strip().lower() in {"at-risk", "blocked"}

    def to_dict(self) -> Dict[str, object]:
        return {
            "goal_id": self.goal_id,
            "title": self.title,
            "owner": self.owner,
            "status": self.status,
            "success_metric": self.success_metric,
            "target_value": self.target_value,
            "current_value": self.current_value,
            "dependencies": list(self.dependencies),
            "risks": list(self.risks),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WeeklyGoal":
        return cls(
            goal_id=str(data["goal_id"]),
            title=str(data["title"]),
            owner=str(data["owner"]),
            status=str(data["status"]),
            success_metric=str(data["success_metric"]),
            target_value=str(data["target_value"]),
            current_value=str(data.get("current_value", "")),
            dependencies=[str(item) for item in data.get("dependencies", [])],
            risks=[str(item) for item in data.get("risks", [])],
        )


@dataclass(frozen=True)
class WeeklyExecutionPlan:
    week_number: int
    theme: str
    objective: str
    exit_criteria: List[str] = field(default_factory=list)
    deliverables: List[str] = field(default_factory=list)
    goals: List[WeeklyGoal] = field(default_factory=list)

    @property
    def completed_goals(self) -> int:
        return sum(goal.is_complete for goal in self.goals)

    @property
    def total_goals(self) -> int:
        return len(self.goals)

    @property
    def progress_percent(self) -> int:
        if not self.goals:
            return 0
        return int((self.completed_goals / len(self.goals)) * 100)

    @property
    def at_risk_goal_ids(self) -> List[str]:
        return [goal.goal_id for goal in self.goals if goal.is_at_risk]

    def to_dict(self) -> Dict[str, object]:
        return {
            "week_number": self.week_number,
            "theme": self.theme,
            "objective": self.objective,
            "exit_criteria": list(self.exit_criteria),
            "deliverables": list(self.deliverables),
            "goals": [goal.to_dict() for goal in self.goals],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WeeklyExecutionPlan":
        return cls(
            week_number=int(data["week_number"]),
            theme=str(data["theme"]),
            objective=str(data["objective"]),
            exit_criteria=[str(item) for item in data.get("exit_criteria", [])],
            deliverables=[str(item) for item in data.get("deliverables", [])],
            goals=[WeeklyGoal.from_dict(item) for item in data.get("goals", [])],
        )


@dataclass
class FourWeekExecutionPlan:
    plan_id: str
    title: str
    owner: str
    start_date: str
    weeks: List[WeeklyExecutionPlan] = field(default_factory=list)

    @property
    def total_goals(self) -> int:
        return sum(week.total_goals for week in self.weeks)

    @property
    def completed_goals(self) -> int:
        return sum(week.completed_goals for week in self.weeks)

    @property
    def overall_progress_percent(self) -> int:
        if self.total_goals == 0:
            return 0
        return int((self.completed_goals / self.total_goals) * 100)

    @property
    def at_risk_weeks(self) -> List[int]:
        return [week.week_number for week in self.weeks if week.at_risk_goal_ids]

    def goal_status_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for week in self.weeks:
            for goal in week.goals:
                counts[goal.status] = counts.get(goal.status, 0) + 1
        return counts

    def validate(self) -> None:
        week_numbers = [week.week_number for week in self.weeks]
        if week_numbers != [1, 2, 3, 4]:
            raise ValueError("Four-week execution plans must include weeks 1 through 4 in order")

    def to_dict(self) -> Dict[str, object]:
        return {
            "plan_id": self.plan_id,
            "title": self.title,
            "owner": self.owner,
            "start_date": self.start_date,
            "weeks": [week.to_dict() for week in self.weeks],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "FourWeekExecutionPlan":
        return cls(
            plan_id=str(data["plan_id"]),
            title=str(data["title"]),
            owner=str(data["owner"]),
            start_date=str(data["start_date"]),
            weeks=[WeeklyExecutionPlan.from_dict(item) for item in data.get("weeks", [])],
        )


def build_big_4701_execution_plan() -> FourWeekExecutionPlan:
    plan = FourWeekExecutionPlan(
        plan_id="BIG-4701",
        title="4周执行计划与周目标",
        owner="execution-office",
        start_date="2026-03-11",
        weeks=[
            WeeklyExecutionPlan(
                week_number=1,
                theme="Scope freeze and operating baseline",
                objective="Freeze scope, align owners, and establish validation and reporting cadence.",
                exit_criteria=[
                    "Scope freeze board published",
                    "Owners and validation commands assigned for all streams",
                ],
                deliverables=[
                    "Execution baseline report",
                    "Scope freeze audit snapshot",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w1-scope-freeze",
                        title="Lock the v4.0 scope and escalation path",
                        owner="program-office",
                        status="done",
                        success_metric="frozen backlog items",
                        target_value="5 epics aligned",
                        current_value="5 epics aligned",
                    ),
                    WeeklyGoal(
                        goal_id="w1-validation-matrix",
                        title="Assign validation commands and evidence owners",
                        owner="engineering-ops",
                        status="done",
                        success_metric="streams with validation owners",
                        target_value="5/5 streams",
                        current_value="5/5 streams",
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=2,
                theme="Build and integration",
                objective="Land the highest-risk implementation slices and wire cross-team dependencies.",
                exit_criteria=[
                    "P0 build items merged",
                    "Cross-team dependency review completed",
                ],
                deliverables=[
                    "Integrated build checkpoint",
                    "Dependency burn-down",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w2-p0-burndown",
                        title="Close the top P0 implementation gaps",
                        owner="engineering-platform",
                        status="on-track",
                        success_metric="P0 items merged",
                        target_value=">=3 merged",
                        current_value="2 merged",
                    ),
                    WeeklyGoal(
                        goal_id="w2-handoff-sync",
                        title="Resolve orchestration and console handoff dependencies",
                        owner="orchestration-office",
                        status="at-risk",
                        success_metric="open handoff blockers",
                        target_value="0 blockers",
                        current_value="1 blocker",
                        dependencies=["w2-p0-burndown"],
                        risks=["console entitlement contract is pending"],
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=3,
                theme="Stabilization and validation",
                objective="Drive regression triage, benchmark replay, and release-readiness evidence.",
                exit_criteria=[
                    "Regression backlog under control threshold",
                    "Benchmark comparison published",
                ],
                deliverables=[
                    "Stabilization report",
                    "Benchmark replay pack",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w3-regression-triage",
                        title="Reduce critical regressions before release gate",
                        owner="quality-ops",
                        status="not-started",
                        success_metric="critical regressions",
                        target_value="<=2 open",
                    ),
                    WeeklyGoal(
                        goal_id="w3-benchmark-pack",
                        title="Publish replay and weighted benchmark evidence",
                        owner="evaluation-lab",
                        status="not-started",
                        success_metric="benchmark evidence bundle",
                        target_value="1 bundle published",
                    ),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=4,
                theme="Launch decision and weekly operating rhythm",
                objective="Convert validation evidence into launch readiness and the post-launch weekly review cadence.",
                exit_criteria=[
                    "Launch decision signed off",
                    "Weekly operating review template adopted",
                ],
                deliverables=[
                    "Launch readiness packet",
                    "Weekly review operating template",
                ],
                goals=[
                    WeeklyGoal(
                        goal_id="w4-launch-decision",
                        title="Complete launch readiness review",
                        owner="release-governance",
                        status="not-started",
                        success_metric="required sign-offs",
                        target_value="all sign-offs complete",
                    ),
                    WeeklyGoal(
                        goal_id="w4-weekly-rhythm",
                        title="Roll out the weekly KPI and issue review cadence",
                        owner="engineering-operations",
                        status="not-started",
                        success_metric="weekly review adoption",
                        target_value="1 recurring cadence active",
                    ),
                ],
            ),
        ],
    )
    plan.validate()
    return plan


def build_pilot_rollout_scorecard(
    *,
    adoption: float,
    convergence_improvement: float,
    review_efficiency: float,
    governance_incidents: int,
    evidence_completeness: float,
) -> Dict[str, object]:
    score = (
        adoption * 0.25
        + convergence_improvement * 0.25
        + review_efficiency * 0.2
        + evidence_completeness * 0.2
        + max(0.0, 100.0 - (governance_incidents * 20.0)) * 0.1
    )
    passed = score >= 75 and governance_incidents <= 2 and evidence_completeness >= 70
    return {
        "adoption": round(adoption, 1),
        "convergence_improvement": round(convergence_improvement, 1),
        "review_efficiency": round(review_efficiency, 1),
        "governance_incidents": int(governance_incidents),
        "evidence_completeness": round(evidence_completeness, 1),
        "rollout_score": round(score, 1),
        "recommendation": "go" if passed else "hold",
    }


def evaluate_candidate_gate(
    *,
    gate_decision: EntryGateDecision,
    rollout_scorecard: Dict[str, object],
) -> Dict[str, object]:
    readiness = bool(gate_decision.passed)
    rollout_ready = rollout_scorecard.get("recommendation") == "go"
    recommendation = "enable-by-default" if readiness and rollout_ready else "pilot-only"
    findings: List[str] = []
    if not readiness:
        findings.append(gate_decision.summary)
    if not rollout_ready:
        findings.append(
            "rollout score below threshold"
            f" ({rollout_scorecard.get('rollout_score', 'n/a')})"
        )
    return {
        "gate_passed": readiness,
        "rollout_recommendation": str(rollout_scorecard.get("recommendation", "hold")),
        "candidate_gate": recommendation,
        "findings": findings,
    }


def render_pilot_rollout_gate_report(result: Dict[str, object]) -> str:
    findings = result.get("findings") or []
    lines = [
        "# Pilot Rollout Candidate Gate",
        "",
        f"- Gate passed: {result.get('gate_passed')}",
        f"- Rollout recommendation: {result.get('rollout_recommendation')}",
        f"- Candidate gate: {result.get('candidate_gate')}",
    ]
    lines.append(f"- Findings: {', '.join(findings) if findings else 'none'}")
    return "\n".join(lines)


def render_four_week_execution_report(plan: FourWeekExecutionPlan) -> str:
    plan.validate()
    status_counts = plan.goal_status_counts()
    lines = [
        "# Four-Week Execution Plan",
        "",
        f"- Plan: {plan.plan_id} {plan.title}",
        f"- Owner: {plan.owner}",
        f"- Start date: {plan.start_date}",
        f"- Overall progress: {plan.completed_goals}/{plan.total_goals} goals complete ({plan.overall_progress_percent}%)",
        f"- At-risk weeks: {', '.join(str(week_number) for week_number in plan.at_risk_weeks) or 'none'}",
        (
            "- Goal status counts: "
            f"done={status_counts.get('done', 0)} "
            f"on-track={status_counts.get('on-track', 0)} "
            f"at-risk={status_counts.get('at-risk', 0)} "
            f"blocked={status_counts.get('blocked', 0)} "
            f"not-started={status_counts.get('not-started', 0)}"
        ),
        "",
        "## Weekly Plans",
    ]
    for week in plan.weeks:
        lines.extend(
            [
                (
                    f"- Week {week.week_number}: {week.theme} "
                    f"progress={week.completed_goals}/{week.total_goals} ({week.progress_percent}%)"
                ),
                f"  objective={week.objective}",
                f"  exit_criteria={', '.join(week.exit_criteria) or 'none'}",
                f"  deliverables={', '.join(week.deliverables) or 'none'}",
            ]
        )
        for goal in week.goals:
            lines.append(
                "  "
                f"- {goal.goal_id}: {goal.title} owner={goal.owner} status={goal.status} "
                f"metric={goal.success_metric} current={goal.current_value or 'n/a'} "
                f"target={goal.target_value}"
            )
            lines.append(
                "    "
                f"dependencies={','.join(goal.dependencies) or 'none'} "
                f"risks={','.join(goal.risks) or 'none'}"
            )
    return "\n".join(lines)


_install_support_module(
    "planning",
    PRIORITY_WEIGHTS=PRIORITY_WEIGHTS,
    GOAL_STATUS_ORDER=GOAL_STATUS_ORDER,
    EvidenceLink=EvidenceLink,
    CandidateEntry=CandidateEntry,
    CandidateBacklog=CandidateBacklog,
    EntryGate=EntryGate,
    EntryGateDecision=EntryGateDecision,
    CandidatePlanner=CandidatePlanner,
    render_candidate_backlog_report=render_candidate_backlog_report,
    build_v3_candidate_backlog=build_v3_candidate_backlog,
    build_v3_entry_gate=build_v3_entry_gate,
    WeeklyGoal=WeeklyGoal,
    WeeklyExecutionPlan=WeeklyExecutionPlan,
    FourWeekExecutionPlan=FourWeekExecutionPlan,
    build_big_4701_execution_plan=build_big_4701_execution_plan,
    build_pilot_rollout_scorecard=build_pilot_rollout_scorecard,
    evaluate_candidate_gate=evaluate_candidate_gate,
    render_pilot_rollout_gate_report=render_pilot_rollout_gate_report,
    render_four_week_execution_report=render_four_week_execution_report,
)

_install_exec_support_module_from_asset(
    "operations",
    "_operations_support.txt",
    [
        "STATUS_COMPLETE",
        "STATUS_ACTIONABLE",
        "TriageCluster",
        "RegressionFinding",
        "OperationsSnapshot",
        "WeeklyOperationsReport",
        "RegressionCenter",
        "VersionedArtifact",
        "VersionChangeSummary",
        "VersionedArtifactHistory",
        "PolicyPromptVersionCenter",
        "WeeklyOperationsArtifacts",
        "QueueControlCenter",
        "EngineeringOverviewKPI",
        "EngineeringFunnelStage",
        "EngineeringOverviewBlocker",
        "EngineeringActivity",
        "EngineeringOverviewPermission",
        "EngineeringOverview",
        "OperationsMetricDefinition",
        "OperationsMetricValue",
        "OperationsMetricSpec",
        "DashboardWidgetSpec",
        "DashboardWidgetPlacement",
        "DashboardLayout",
        "DashboardBuilder",
        "DashboardBuilderAudit",
        "OperationsAnalytics",
        "render_operations_dashboard",
        "render_weekly_operations_report",
        "render_operations_metric_spec",
        "render_queue_control_center",
        "render_policy_prompt_version_center",
        "render_engineering_overview",
        "render_dashboard_builder_report",
        "write_engineering_overview_bundle",
        "write_dashboard_builder_bundle",
        "build_repo_collaboration_metrics",
        "write_weekly_operations_bundle",
        "render_regression_center",
    ],
)
from .ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewBlocker,
    ReviewBlockerEvent,
    ReviewDecision,
    ReviewObjective,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPack,
    UIReviewPackArtifacts,
    UIReviewPackAudit,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_escalation_dashboard,
    render_ui_review_escalation_handoff_ledger,
    render_ui_review_exception_log,
    render_ui_review_exception_matrix,
    render_ui_review_freeze_approval_trail,
    render_ui_review_freeze_exception_board,
    render_ui_review_freeze_renewal_tracker,
    render_ui_review_handoff_ack_ledger,
    render_ui_review_interaction_coverage_board,
    render_ui_review_objective_coverage_board,
    render_ui_review_open_question_tracker,
    render_ui_review_owner_escalation_digest,
    render_ui_review_persona_readiness_board,
    render_ui_review_review_summary_board,
    render_ui_review_owner_review_queue,
    render_ui_review_owner_workload_board,
    render_ui_review_checklist_traceability_board,
    render_ui_review_decision_followup_tracker,
    render_ui_review_audit_density_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_role_coverage_board,
    render_ui_review_wireframe_readiness_board,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_decision_log,
    render_ui_review_pack_html,
    render_ui_review_role_matrix,
    render_ui_review_signoff_log,
    render_ui_review_pack_report,
    write_ui_review_pack_bundle,
)
_install_exec_support_module(
    "runtime",
    r'''from __future__ import annotations

"""Legacy Python runtime and workflow surfaces frozen after Go mainline cutover."""

import heapq
import json
import os
import threading
import time
from collections import deque
from dataclasses import dataclass, field
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any, Callable, Deque, Dict, List, Optional, Sequence, Tuple

from .audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    SCHEDULER_DECISION_EVENT,
)
from .deprecation import LEGACY_RUNTIME_GUIDANCE
from .models import RiskLevel, Task
from .observability import ObservabilityLedger, RepoSyncAudit, TaskRun, utc_now
from .risk import RiskScore, RiskScorer


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


@dataclass(frozen=True)
class DeadLetterEntry:
    task: Task
    reason: str = ""

    def to_dict(self) -> dict:
        return {"task_id": self.task.task_id, "task": self.task.to_dict(), "reason": self.reason}

    @classmethod
    def from_dict(cls, data: dict) -> "DeadLetterEntry":
        return cls(
            task=Task.from_dict(data["task"]),
            reason=str(data.get("reason", "")),
        )


class PersistentTaskQueue:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)
        self._heap: List[Tuple[int, str, dict]] = []
        self._dead_letters: Dict[str, DeadLetterEntry] = {}
        self._load()

    def _load(self) -> None:
        self._heap = []
        self._dead_letters = {}
        if not self.storage_path.exists():
            return
        data = json.loads(self.storage_path.read_text(encoding="utf-8"))

        if isinstance(data, list):
            queue_items = data
            dead_letter_items: List[dict] = []
        else:
            queue_items = data.get("queue", [])
            dead_letter_items = data.get("dead_letters", [])

        for item in queue_items:
            heapq.heappush(self._heap, (item["priority"], item["task_id"], item["task"]))
        for item in dead_letter_items:
            entry = DeadLetterEntry.from_dict(item)
            self._dead_letters[entry.task.task_id] = entry

    def _save(self) -> None:
        data = {
            "queue": [{"priority": p, "task_id": tid, "task": task} for (p, tid, task) in sorted(self._heap)],
            "dead_letters": [entry.to_dict() for _, entry in sorted(self._dead_letters.items())],
        }
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        tmp_path = self.storage_path.with_name(f"{self.storage_path.name}.tmp")
        tmp_path.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        tmp_path.replace(self.storage_path)

    def _drop_queued_task(self, task_id: str) -> None:
        if not any(queued_task_id == task_id for _, queued_task_id, _ in self._heap):
            return
        self._heap = [
            (priority, queued_task_id, task)
            for priority, queued_task_id, task in self._heap
            if queued_task_id != task_id
        ]
        heapq.heapify(self._heap)

    def enqueue(self, task: Task) -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters.pop(task.task_id, None)
        heapq.heappush(self._heap, (int(task.priority), task.task_id, task.to_dict()))
        self._save()

    def dequeue(self) -> Optional[dict]:
        if not self._heap:
            return None
        _priority, _task_id, task = heapq.heappop(self._heap)
        self._save()
        return task

    def dequeue_task(self) -> Optional[Task]:
        task = self.dequeue()
        if task is None:
            return None
        return Task.from_dict(task)

    def dead_letter(self, task: Task, reason: str = "") -> None:
        self._drop_queued_task(task.task_id)
        self._dead_letters[task.task_id] = DeadLetterEntry(task=task, reason=reason)
        self._save()

    def list_dead_letters(self) -> List[DeadLetterEntry]:
        return [entry for _, entry in sorted(self._dead_letters.items())]

    def retry_dead_letter(self, task_id: str) -> bool:
        entry = self._dead_letters.pop(task_id, None)
        if entry is None:
            return False
        self._drop_queued_task(task_id)
        heapq.heappush(self._heap, (int(entry.task.priority), entry.task.task_id, entry.task.to_dict()))
        self._save()
        return True

    def size(self) -> int:
        return len(self._heap)

    def peek_tasks(self) -> List[Task]:
        return [Task.from_dict(task) for (_priority, _task_id, task) in sorted(self._heap)]


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
            lines.append(f"- {handoff.department}: reason={handoff.reason} tools={tools} approvals={approvals}")

    return "\n".join(lines) + "\n"


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
    _MEDIUM_BUDGET_FLOORS = {
        "docker": 10.0,
        "browser": 20.0,
        "vm": 40.0,
    }

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
            decision = SchedulerDecision("vm", False, "requires approval for high-risk task")
            return self._apply_budget_policy(task, decision, resolved_risk)

        if "browser" in task.required_tools:
            decision = SchedulerDecision("browser", True, "browser automation task")
            return self._apply_budget_policy(task, decision, resolved_risk)

        if resolved_risk.level == RiskLevel.MEDIUM:
            decision = SchedulerDecision("docker", True, "medium risk in docker")
            return self._apply_budget_policy(task, decision, resolved_risk)

        decision = SchedulerDecision("docker", True, "default low risk path")
        return self._apply_budget_policy(task, decision, resolved_risk)

    def execute(
        self,
        task: Task,
        run_id: str,
        ledger: ObservabilityLedger,
        report_path: Optional[str] = None,
        actor: str = "scheduler",
    ) -> ExecutionRecord:
        from .reports import render_task_run_detail_page, render_task_run_report, write_report

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

        if decision.approved:
            final_status = "approved"
        elif decision.medium == "none":
            final_status = "paused"
        else:
            final_status = "needs-approval"
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

    def _apply_budget_policy(
        self,
        task: Task,
        decision: SchedulerDecision,
        risk_score: RiskScore,
    ) -> SchedulerDecision:
        effective_budget = self._effective_budget(task)
        if effective_budget is None:
            return decision

        required_budget = self._MEDIUM_BUDGET_FLOORS.get(decision.medium, 0.0)
        if effective_budget >= required_budget:
            return decision

        if (
            decision.medium == "browser"
            and risk_score.level != RiskLevel.HIGH
            and effective_budget >= self._MEDIUM_BUDGET_FLOORS["docker"]
        ):
            return SchedulerDecision(
                "docker",
                True,
                (
                    "budget degraded browser route to docker "
                    f"(budget {effective_budget:.1f} < required {required_budget:.1f})"
                ),
            )

        return SchedulerDecision(
            "none",
            False,
            f"paused: budget {effective_budget:.1f} below required {decision.medium} budget {required_budget:.1f}",
        )

    def _effective_budget(self, task: Task) -> Optional[float]:
        effective_budget = task.budget + task.budget_override_amount
        if effective_budget <= 0:
            return None
        return effective_budget


@dataclass
class JournalEntry:
    step: str
    status: str
    timestamp: str = field(default_factory=utc_now)
    details: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "step": self.step,
            "status": self.status,
            "timestamp": self.timestamp,
            "details": self.details,
        }


@dataclass
class WorkpadJournal:
    task_id: str
    run_id: str
    entries: List[JournalEntry] = field(default_factory=list)

    def record(self, step: str, status: str, **details: Any) -> None:
        self.entries.append(JournalEntry(step=step, status=status, details=details))

    def replay(self) -> List[str]:
        return [f"{entry.step}:{entry.status}" for entry in self.entries]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "task_id": self.task_id,
            "run_id": self.run_id,
            "entries": [entry.to_dict() for entry in self.entries],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkpadJournal":
        entries = [
            JournalEntry(
                step=item["step"],
                status=item["status"],
                timestamp=item.get("timestamp", utc_now()),
                details=item.get("details", {}),
            )
            for item in data.get("entries", [])
        ]
        return cls(task_id=data["task_id"], run_id=data["run_id"], entries=entries)

    @classmethod
    def read(cls, path: str) -> "WorkpadJournal":
        payload = json.loads(Path(path).read_text())
        return cls.from_dict(payload)

    def write(self, path: str) -> str:
        output = Path(path)
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(self.to_dict(), ensure_ascii=False, indent=2))
        return str(output)


@dataclass
class AcceptanceDecision:
    passed: bool
    status: str
    summary: str
    missing_acceptance_criteria: List[str] = field(default_factory=list)
    missing_validation_steps: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)


class AcceptanceGate:
    def evaluate(
        self,
        task: Task,
        record: ExecutionRecord,
        validation_evidence: Optional[Sequence[str]] = None,
        approvals: Optional[Sequence[str]] = None,
        pilot_scorecard: Optional[Any] = None,
    ) -> AcceptanceDecision:
        evidence = set(validation_evidence or [])
        approval_list = list(approvals or [])

        missing_acceptance = [item for item in task.acceptance_criteria if item not in evidence]
        missing_validation = [item for item in task.validation_plan if item not in evidence]

        needs_manual_approval = (
            task.risk_level == RiskLevel.HIGH or not record.decision.approved or record.run.status == "needs-approval"
        )
        if needs_manual_approval and not approval_list:
            return AcceptanceDecision(
                passed=False,
                status="needs-approval",
                summary="manual approval required before acceptance closure",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        if pilot_scorecard is not None and pilot_scorecard.recommendation == "hold":
            return AcceptanceDecision(
                passed=False,
                status="rejected",
                summary="pilot scorecard indicates insufficient ROI or KPI progress",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        if missing_acceptance or missing_validation:
            return AcceptanceDecision(
                passed=False,
                status="rejected",
                summary="acceptance evidence incomplete",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        return AcceptanceDecision(
            passed=True,
            status="accepted",
            summary="acceptance criteria and validation plan satisfied",
            approvals=approval_list,
        )


@dataclass
class WorkflowRunResult:
    execution: ExecutionRecord
    acceptance: AcceptanceDecision
    journal: WorkpadJournal
    journal_path: Optional[str]
    orchestration_report_path: Optional[str] = None
    orchestration_canvas_path: Optional[str] = None
    pilot_report_path: Optional[str] = None
    repo_sync_report_path: Optional[str] = None


class WorkflowEngine:
    def __init__(self, scheduler: Optional[Scheduler] = None, gate: Optional[AcceptanceGate] = None):
        self.scheduler = scheduler or Scheduler()
        self.gate = gate or AcceptanceGate()

    def run(
        self,
        task: Task,
        run_id: str,
        ledger: ObservabilityLedger,
        report_path: Optional[str] = None,
        journal_path: Optional[str] = None,
        validation_evidence: Optional[Sequence[str]] = None,
        approvals: Optional[Sequence[str]] = None,
        pilot_scorecard: Optional[Any] = None,
        pilot_report_path: Optional[str] = None,
        orchestration_report_path: Optional[str] = None,
        orchestration_canvas_path: Optional[str] = None,
        repo_sync_audit: Optional[RepoSyncAudit] = None,
        repo_sync_report_path: Optional[str] = None,
        git_push_succeeded: bool = False,
        git_push_output: str = "",
        git_log_stat_output: str = "",
    ) -> WorkflowRunResult:
        from .reports import (
            build_orchestration_canvas,
            render_orchestration_canvas,
            render_pilot_scorecard,
            render_repo_sync_audit_report,
            write_report,
        )

        journal = WorkpadJournal(task_id=task.task_id, run_id=run_id)
        journal.record("intake", "recorded", source=task.source)

        execution = self.scheduler.execute(
            task,
            run_id=run_id,
            ledger=ledger,
            report_path=report_path,
            actor="workflow-engine",
        )
        journal.record(
            "execution",
            execution.run.status,
            medium=execution.decision.medium,
            approved=execution.decision.approved,
        )

        resolved_orchestration_report_path = None
        resolved_orchestration_canvas_path = None
        if execution.orchestration_plan is not None and orchestration_report_path:
            resolved_orchestration_report_path = str(Path(orchestration_report_path))
            write_report(
                resolved_orchestration_report_path,
                render_orchestration_plan(
                    execution.orchestration_plan,
                    execution.orchestration_policy,
                    execution.handoff_request,
                ),
            )
            execution.run.register_artifact(
                "cross-department-orchestration",
                "report",
                resolved_orchestration_report_path,
                format="markdown",
                collaboration_mode=execution.orchestration_plan.collaboration_mode,
                departments=execution.orchestration_plan.departments,
            )
            ledger.upsert(execution.run)
            if orchestration_canvas_path:
                resolved_orchestration_canvas_path = str(Path(orchestration_canvas_path))
                canvas = build_orchestration_canvas(
                    execution.run,
                    execution.orchestration_plan,
                    execution.orchestration_policy,
                    execution.handoff_request,
                )
                write_report(resolved_orchestration_canvas_path, render_orchestration_canvas(canvas))
                execution.run.register_artifact(
                    "orchestration-canvas",
                    "report",
                    resolved_orchestration_canvas_path,
                    format="markdown",
                    recommendation=canvas.recommendation,
                )
                ledger.upsert(execution.run)
            journal.record(
                "orchestration",
                execution.orchestration_plan.collaboration_mode,
                departments=execution.orchestration_plan.departments,
                approvals=execution.orchestration_plan.required_approvals,
                tier=execution.orchestration_policy.tier if execution.orchestration_policy else "standard",
                upgrade_required=(
                    execution.orchestration_policy.upgrade_required if execution.orchestration_policy else False
                ),
                handoff_team=execution.handoff_request.target_team if execution.handoff_request else "none",
            )

        resolved_pilot_report_path = None
        if pilot_scorecard is not None and pilot_report_path:
            resolved_pilot_report_path = str(Path(pilot_report_path))
            write_report(resolved_pilot_report_path, render_pilot_scorecard(pilot_scorecard))
            execution.run.register_artifact(
                "pilot-scorecard",
                "report",
                resolved_pilot_report_path,
                format="markdown",
                recommendation=pilot_scorecard.recommendation,
            )
            ledger.upsert(execution.run)
            journal.record(
                "pilot-scorecard",
                pilot_scorecard.recommendation,
                metrics_met=pilot_scorecard.metrics_met,
                metrics_total=len(pilot_scorecard.metrics),
                annualized_roi=round(pilot_scorecard.annualized_roi, 1),
            )

        resolved_repo_sync_report_path = None
        if repo_sync_audit is not None:
            execution.run.audit(
                "repo.sync",
                "workflow-engine",
                repo_sync_audit.sync.status,
                failure_category=repo_sync_audit.sync.failure_category,
                summary=repo_sync_audit.sync.summary,
                branch=repo_sync_audit.sync.branch,
                remote_ref=repo_sync_audit.sync.remote_ref,
                ahead_by=repo_sync_audit.sync.ahead_by,
                behind_by=repo_sync_audit.sync.behind_by,
                dirty_paths=repo_sync_audit.sync.dirty_paths,
                auth_target=repo_sync_audit.sync.auth_target,
            )
            execution.run.audit(
                "repo.pr-freshness",
                "workflow-engine",
                "fresh" if repo_sync_audit.pull_request.fresh else "stale",
                pr_number=repo_sync_audit.pull_request.pr_number,
                pr_url=repo_sync_audit.pull_request.pr_url,
                branch_state=repo_sync_audit.pull_request.branch_state,
                body_state=repo_sync_audit.pull_request.body_state,
                branch_head_sha=repo_sync_audit.pull_request.branch_head_sha,
                pr_head_sha=repo_sync_audit.pull_request.pr_head_sha,
            )
            journal.record(
                "repo-sync",
                repo_sync_audit.sync.status,
                failure_category=repo_sync_audit.sync.failure_category,
                branch_state=repo_sync_audit.pull_request.branch_state,
                body_state=repo_sync_audit.pull_request.body_state,
            )
            if repo_sync_report_path:
                resolved_repo_sync_report_path = str(Path(repo_sync_report_path))
                write_report(resolved_repo_sync_report_path, render_repo_sync_audit_report(repo_sync_audit))
                execution.run.register_artifact(
                    "repo-sync-audit",
                    "report",
                    resolved_repo_sync_report_path,
                    format="markdown",
                    sync_status=repo_sync_audit.sync.status,
                    pr_branch_state=repo_sync_audit.pull_request.branch_state,
                    pr_body_state=repo_sync_audit.pull_request.body_state,
                )
                ledger.upsert(execution.run)

        acceptance = self.gate.evaluate(
            task,
            execution,
            validation_evidence=validation_evidence,
            approvals=approvals,
            pilot_scorecard=pilot_scorecard,
        )
        if acceptance.approvals:
            execution.run.audit_spec_event(
                APPROVAL_RECORDED_EVENT,
                "workflow-engine",
                "recorded",
                task_id=task.task_id,
                run_id=run_id,
                approvals=list(acceptance.approvals),
                approval_count=len(acceptance.approvals),
                acceptance_status=acceptance.status,
            )
        journal.record(
            "acceptance",
            acceptance.status,
            passed=acceptance.passed,
            missing_acceptance_criteria=acceptance.missing_acceptance_criteria,
            missing_validation_steps=acceptance.missing_validation_steps,
        )
        execution.run.record_closeout(
            validation_evidence=list(validation_evidence or []),
            git_push_succeeded=git_push_succeeded,
            git_push_output=git_push_output,
            git_log_stat_output=git_log_stat_output,
            repo_sync_audit=repo_sync_audit,
        )
        ledger.upsert(execution.run)
        journal.record(
            "closeout",
            "complete" if execution.run.closeout.complete else "pending",
            validation_evidence=list(validation_evidence or []),
            git_push_succeeded=git_push_succeeded,
            git_log_stat_captured=bool(git_log_stat_output.strip()),
            repo_sync_status=repo_sync_audit.sync.status if repo_sync_audit else "none",
            repo_sync_failure_category=repo_sync_audit.sync.failure_category if repo_sync_audit else "",
        )

        resolved_journal_path = None
        if journal_path:
            resolved_journal_path = journal.write(journal_path)

        return WorkflowRunResult(
            execution=execution,
            acceptance=acceptance,
            journal=journal,
            journal_path=resolved_journal_path,
            orchestration_report_path=resolved_orchestration_report_path,
            orchestration_canvas_path=resolved_orchestration_canvas_path,
            pilot_report_path=resolved_pilot_report_path,
            repo_sync_report_path=resolved_repo_sync_report_path,
        )

    def run_definition(
        self,
        task: Task,
        definition: Any,
        run_id: str,
        ledger: ObservabilityLedger,
    ) -> WorkflowRunResult:
        definition.validate()
        return self.run(
            task,
            run_id=run_id,
            ledger=ledger,
            report_path=definition.render_report_path(task, run_id),
            journal_path=definition.render_journal_path(task, run_id),
            validation_evidence=definition.validation_evidence,
            approvals=definition.approvals,
        )


def warn_legacy_service_surface(surface: str = "python -m bigclaw serve") -> str:
    from .deprecation import warn_legacy_runtime_surface

    return warn_legacy_runtime_surface(surface, "go run ./bigclaw-go/cmd/bigclawd")


@dataclass
class ServerMonitoring:
    start_time: float = field(default_factory=time.time)
    request_total: int = 0
    error_total: int = 0
    recent_requests: Deque[Dict[str, str]] = field(default_factory=lambda: deque(maxlen=20))
    minute_buckets: Deque[Dict[str, int]] = field(default_factory=lambda: deque(maxlen=5))
    _lock: threading.Lock = field(default_factory=threading.Lock)

    def _ensure_bucket(self, minute: int) -> Dict[str, int]:
        if not self.minute_buckets or self.minute_buckets[-1]["minute"] != minute:
            self.minute_buckets.append({"minute": minute, "requests": 0, "errors": 0})
        return self.minute_buckets[-1]

    def record(self, path: str, status: int) -> None:
        ts = time.time()
        minute = int(ts // 60)
        with self._lock:
            self.request_total += 1
            if status >= 400:
                self.error_total += 1
            self.recent_requests.append({"path": path, "status": str(status), "ts": f"{ts:.3f}"})
            bucket = self._ensure_bucket(minute)
            bucket["requests"] += 1
            if status >= 400:
                bucket["errors"] += 1

    def _rolling(self) -> List[Dict[str, int]]:
        return [dict(bucket) for bucket in self.minute_buckets]

    def health_payload(self) -> Dict[str, object]:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            return {
                "status": "ok",
                "uptime_seconds": round(uptime, 3),
                "request_total": self.request_total,
                "error_total": self.error_total,
                "recent_requests": list(self.recent_requests),
                "rolling_5m": self._rolling(),
            }

    def metrics_payload(self) -> Dict[str, object]:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            error_rate = (self.error_total / self.request_total) if self.request_total else 0.0
            summary = "healthy"
            if error_rate >= 0.2:
                summary = "critical"
            elif error_rate >= 0.05:
                summary = "degraded"
            return {
                "bigclaw_uptime_seconds": round(uptime, 3),
                "bigclaw_http_requests_total": self.request_total,
                "bigclaw_http_errors_total": self.error_total,
                "bigclaw_http_error_rate": round(error_rate, 4),
                "health_summary": summary,
                "recent_requests": list(self.recent_requests),
                "rolling_5m": self._rolling(),
            }

    def alerts_payload(self) -> Dict[str, object]:
        metrics = self.metrics_payload()
        error_rate = float(metrics["bigclaw_http_error_rate"])
        level = "ok"
        message = "System healthy"
        if error_rate >= 0.2:
            level = "critical"
            message = "High HTTP error rate detected"
        elif error_rate >= 0.05:
            level = "warn"
            message = "Elevated HTTP error rate detected"
        return {
            "level": level,
            "message": message,
            "error_rate": error_rate,
            "request_total": metrics["bigclaw_http_requests_total"],
            "error_total": metrics["bigclaw_http_errors_total"],
        }

    def metrics_text(self) -> str:
        uptime = max(0.0, time.time() - self.start_time)
        with self._lock:
            lines = [
                "# HELP bigclaw_uptime_seconds process uptime in seconds",
                "# TYPE bigclaw_uptime_seconds gauge",
                f"bigclaw_uptime_seconds {uptime:.3f}",
                "# HELP bigclaw_http_requests_total total HTTP requests",
                "# TYPE bigclaw_http_requests_total counter",
                f"bigclaw_http_requests_total {self.request_total}",
                "# HELP bigclaw_http_errors_total total HTTP error responses (>=400)",
                "# TYPE bigclaw_http_errors_total counter",
                f"bigclaw_http_errors_total {self.error_total}",
            ]
            for bucket in self.minute_buckets:
                lines.append(f"bigclaw_http_requests_minute{{minute=\"{bucket['minute']}\"}} {bucket['requests']}")
                lines.append(f"bigclaw_http_errors_minute{{minute=\"{bucket['minute']}\"}} {bucket['errors']}")
            return "\n".join(lines) + "\n"


def _monitor_page(stats: Dict[str, object]) -> str:
    rows = "".join(
        f"<tr><td>{item['ts']}</td><td>{item['path']}</td><td>{item['status']}</td></tr>"
        for item in stats["recent_requests"]
    ) or "<tr><td colspan='3'>No requests yet</td></tr>"

    rolling_rows = "".join(
        f"<tr><td>{bucket['minute']}</td><td>{bucket['requests']}</td><td>{bucket['errors']}</td></tr>"
        for bucket in stats.get("rolling_5m", [])
    ) or "<tr><td colspan='3'>No rolling data yet</td></tr>"

    return f"""<!doctype html>
<html>
<head>
  <meta charset='utf-8'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>BigClaw Monitor</title>
  <style>
    body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; background:#f6f7fb; color:#0f172a; }}
    .container {{ max-width: 1040px; margin: 24px auto; padding: 0 16px; }}
    .cards {{ display:grid; grid-template-columns: repeat(auto-fit,minmax(180px,1fr)); gap:12px; }}
    .card {{ background:#fff; border:1px solid #e2e8f0; border-radius:12px; padding:12px; }}
    .label {{ color:#64748b; font-size:12px; }}
    .value {{ font-size:24px; font-weight:700; margin-top:4px; }}
    table {{ width:100%; border-collapse: collapse; background:#fff; border:1px solid #e2e8f0; border-radius:12px; overflow:hidden; }}
    th,td {{ border-bottom:1px solid #e2e8f0; padding:8px 10px; text-align:left; font-size:13px; }}
    h1,h2 {{ margin: 0 0 10px; }}
    section {{ margin-top: 16px; }}
    .muted {{ color:#64748b; font-size:12px; }}
  </style>
</head>
<body>
  <div class='container'>
    <h1>BigClaw Monitor</h1>
    <p class='muted'>Auto refresh every 5s · endpoint: /metrics.json</p>
    <div class='cards'>
      <div class='card'><div class='label'>Uptime (s)</div><div class='value' id='uptime'>{stats['bigclaw_uptime_seconds']}</div></div>
      <div class='card'><div class='label'>Requests</div><div class='value' id='requests'>{stats['bigclaw_http_requests_total']}</div></div>
      <div class='card'><div class='label'>Errors</div><div class='value' id='errors'>{stats['bigclaw_http_errors_total']}</div></div>
      <div class='card'><div class='label'>Error Rate</div><div class='value' id='error-rate'>{stats['bigclaw_http_error_rate']}</div></div>
      <div class='card'><div class='label'>Health</div><div class='value' id='health-summary'>{stats['health_summary']}</div></div>
    </div>

    <section>
      <h2>Rolling 5m</h2>
      <table id='rolling-table'>
        <thead><tr><th>minute</th><th>requests</th><th>errors</th></tr></thead>
        <tbody>{rolling_rows}</tbody>
      </table>
    </section>

    <section>
      <h2>Recent Requests</h2>
      <table id='recent-table'>
        <thead><tr><th>ts</th><th>path</th><th>status</th></tr></thead>
        <tbody>{rows}</tbody>
      </table>
    </section>
  </div>
  <script>
    async function refreshMonitor() {{
      try {{
        const res = await fetch('/metrics.json', {{ cache: 'no-store' }});
        const data = await res.json();
        document.getElementById('uptime').textContent = data.bigclaw_uptime_seconds;
        document.getElementById('requests').textContent = data.bigclaw_http_requests_total;
        document.getElementById('errors').textContent = data.bigclaw_http_errors_total;
        document.getElementById('error-rate').textContent = data.bigclaw_http_error_rate;
        document.getElementById('health-summary').textContent = data.health_summary;

        const rollingBody = document.querySelector('#rolling-table tbody');
        rollingBody.innerHTML = (data.rolling_5m || []).map((b) =>
          `<tr><td>${{b.minute}}</td><td>${{b.requests}}</td><td>${{b.errors}}</td></tr>`
        ).join('') || "<tr><td colspan='3'>No rolling data yet</td></tr>";

        const recentBody = document.querySelector('#recent-table tbody');
        recentBody.innerHTML = (data.recent_requests || []).map((r) =>
          `<tr><td>${{r.ts}}</td><td>${{r.path}}</td><td>${{r.status}}</td></tr>`
        ).join('') || "<tr><td colspan='3'>No requests yet</td></tr>";
      }} catch (e) {{
        console.error('monitor refresh failed', e);
      }}
    }}
    setInterval(refreshMonitor, 5000);
  </script>
</body>
</html>"""


def _handler_factory(*, directory: str, monitoring: ServerMonitoring):
    class BigClawHandler(SimpleHTTPRequestHandler):
        def __init__(self, *args, **kwargs):
            super().__init__(*args, directory=directory, **kwargs)

        def do_GET(self) -> None:  # noqa: N802
            if self.path == "/health":
                payload = monitoring.health_payload()
                body = json.dumps(payload).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/metrics":
                body = monitoring.metrics_text().encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "text/plain; version=0.0.4")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/metrics.json":
                body = json.dumps(monitoring.metrics_payload()).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/monitor":
                html = _monitor_page(monitoring.metrics_payload())
                body = html.encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            if self.path == "/alerts":
                body = json.dumps(monitoring.alerts_payload()).encode("utf-8")
                self.send_response(HTTPStatus.OK)
                self.send_header("Content-Type", "application/json; charset=utf-8")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return

            super().do_GET()

        def log_message(self, format: str, *args) -> None:
            super().log_message(format, *args)

        def send_response(self, code: int, message=None):  # type: ignore[override]
            super().send_response(code, message)
            path = getattr(self, "path", "-")
            monitoring.record(path, int(code))

    return BigClawHandler


def create_server(host: str = "127.0.0.1", port: int = 8008, directory: str = "."):
    directory = os.path.abspath(directory)
    monitoring = ServerMonitoring()
    handler = _handler_factory(directory=directory, monitoring=monitoring)
    server = ThreadingHTTPServer((host, port), handler)
    return server, monitoring


def run_server(host: str = "127.0.0.1", port: int = 8008, directory: str = ".") -> None:
    warn_legacy_service_surface()
    server, _ = create_server(host=host, port=port, directory=directory)
    print(f"BigClaw server running at http://{host}:{port} (dir={os.path.abspath(directory)})")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


@dataclass
class RepoGovernancePolicy:
    max_bundle_bytes: int = 50 * 1024 * 1024
    max_push_per_hour: int = 20
    max_diff_per_hour: int = 120
    sidecar_required: bool = True


@dataclass
class RepoGovernanceResult:
    allowed: bool
    mode: str
    reason: str = ""


class RepoGovernanceEnforcer:
    def __init__(self, policy: RepoGovernancePolicy):
        self.policy = policy
        self.push_count = 0
        self.diff_count = 0

    def evaluate(self, *, action: str, bundle_bytes: int = 0, sidecar_available: bool = True) -> RepoGovernanceResult:
        if self.policy.sidecar_required and not sidecar_available:
            return RepoGovernanceResult(allowed=False, mode="degraded", reason="repo sidecar unavailable")

        if action == "push":
            if bundle_bytes > self.policy.max_bundle_bytes:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="bundle exceeds max size")
            if self.push_count >= self.policy.max_push_per_hour:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="push quota exceeded")
            self.push_count += 1
            return RepoGovernanceResult(allowed=True, mode="allow")

        if action == "diff":
            if self.diff_count >= self.policy.max_diff_per_hour:
                return RepoGovernanceResult(allowed=False, mode="blocked", reason="diff quota exceeded")
            self.diff_count += 1
            return RepoGovernanceResult(allowed=True, mode="allow")

        return RepoGovernanceResult(allowed=True, mode="allow")
''',
    [
        "LEGACY_MAINLINE_STATUS",
        "GO_MAINLINE_REPLACEMENT",
        "SandboxProfile",
        "SandboxRouter",
        "ToolPolicy",
        "ToolCallResult",
        "ToolRuntime",
        "WorkerExecutionResult",
        "ClawWorkerRuntime",
        "DeadLetterEntry",
        "PersistentTaskQueue",
        "DepartmentHandoff",
        "OrchestrationPlan",
        "HandoffRequest",
        "OrchestrationPolicyDecision",
        "CrossDepartmentOrchestrator",
        "PremiumOrchestrationPolicy",
        "render_orchestration_plan",
        "SchedulerDecision",
        "ExecutionRecord",
        "Scheduler",
        "JournalEntry",
        "WorkpadJournal",
        "AcceptanceDecision",
        "AcceptanceGate",
        "WorkflowRunResult",
        "WorkflowEngine",
        "warn_legacy_service_surface",
        "ServerMonitoring",
        "create_server",
        "run_server",
        "RepoGovernancePolicy",
        "RepoGovernanceResult",
        "RepoGovernanceEnforcer",
    ],
)
from .observability import (
    GitSyncTelemetry,
    ObservabilityLedger,
    PullRequestFreshness,
    RepoSyncAudit,
    RunCloseout,
    TaskRun,
    utc_now,
)

__all__ = [
    "Task",
    "TaskState",
    "RiskLevel",
    "Priority",
    "RiskSignal",
    "RiskAssessment",
    "TriageStatus",
    "TriageLabel",
    "TriageRecord",
    "FlowTrigger",
    "FlowRunStatus",
    "FlowStepStatus",
    "FlowTemplateStep",
    "FlowTemplate",
    "FlowStepRun",
    "FlowRun",
    "BillingInterval",
    "BillingRate",
    "UsageRecord",
    "BillingSummary",
    "PersistentTaskQueue",
    "Scheduler",
    "SchedulerDecision",
    "ExecutionRecord",
    "SourceIssue",
    "GitHubConnector",
    "LinearConnector",
    "JiraConnector",
    "CommandAction",
    "AuditRequirement",
    "ComponentLibrary",
    "ComponentSpec",
    "ComponentVariant",
    "ConsoleChromeLibrary",
    "ConsoleCommandEntry",
    "ConsoleTopBar",
    "ConsoleTopBarAudit",
    "DataAccuracyCheck",
    "DesignSystem",
    "DesignSystemAudit",
    "DesignToken",
    "InformationArchitecture",
    "InformationArchitectureAudit",
    "NavigationEntry",
    "NavigationNode",
    "NavigationRoute",
    "PerformanceBudget",
    "RolePermissionScenario",
    "UIAcceptanceAudit",
    "UIAcceptanceLibrary",
    "UIAcceptanceSuite",
    "UsabilityJourney",
    "render_console_top_bar_report",
    "render_design_system_report",
    "render_information_architecture_report",
    "render_ui_acceptance_report",
    "ConsoleIA",
    "ConsoleIAAudit",
    "ConsoleIAAuditor",
    "ConsoleInteractionAudit",
    "ConsoleInteractionAuditor",
    "ConsoleInteractionDraft",
    "ConsoleSurface",
    "FilterDefinition",
    "GlobalAction",
    "NavigationItem",
    "SurfaceInteractionContract",
    "SurfacePermissionRule",
    "SurfaceState",
    "build_big_4203_console_interaction_draft",
    "render_console_interaction_report",
    "render_console_ia_report",
    "AlertDigestSubscription",
    "SavedView",
    "SavedViewCatalog",
    "SavedViewCatalogAudit",
    "SavedViewFilter",
    "SavedViewLibrary",
    "render_saved_view_report",
    "FreezeException",
    "GovernanceBacklogItem",
    "ScopeFreezeAudit",
    "ScopeFreezeBoard",
    "ScopeFreezeGovernance",
    "render_scope_freeze_report",
    "RiskFactor",
    "RiskScore",
    "RiskScorer",
    "CollaborationComment",
    "CollaborationThread",
    "DecisionNote",
    "build_collaboration_thread",
    "build_collaboration_thread_from_audits",
    "WorkflowDefinition",
    "WorkflowStep",
    "map_source_issue_to_task",
    "EpicMilestone",
    "ExecutionPackRoadmap",
    "build_execution_pack_roadmap",
    "APPROVAL_RECORDED_EVENT",
    "BUDGET_OVERRIDE_EVENT",
    "FLOW_HANDOFF_EVENT",
    "MANUAL_TAKEOVER_EVENT",
    "P0_AUDIT_EVENT_SPECS",
    "SCHEDULER_DECISION_EVENT",
    "AuditEventSpec",
    "get_audit_event_spec",
    "missing_required_fields",
    "BusEvent",
    "EventBus",
    "PULL_REQUEST_COMMENT_EVENT",
    "CI_COMPLETED_EVENT",
    "TASK_FAILED_EVENT",
    "ObservabilityLedger",
    "GitSyncTelemetry",
    "PullRequestFreshness",
    "RepoSyncAudit",
    "RunCloseout",
    "TaskRun",
    "CrossDepartmentOrchestrator",
    "DepartmentHandoff",
    "HandoffRequest",
    "OrchestrationPlan",
    "OrchestrationPolicyDecision",
    "PremiumOrchestrationPolicy",
    "render_orchestration_plan",
    "ClawWorkerRuntime",
    "SandboxProfile",
    "SandboxRouter",
    "ToolCallResult",
    "ToolPolicy",
    "ToolRuntime",
    "WorkerExecutionResult",
    "AuditPolicy",
    "ExecutionApiSpec",
    "ExecutionContract",
    "ExecutionContractAudit",
    "ExecutionContractLibrary",
    "ExecutionField",
    "ExecutionModel",
    "ExecutionPermission",
    "ExecutionPermissionMatrix",
    "ExecutionRole",
    "MetricDefinition",
    "PermissionCheckResult",
    "render_execution_contract_report",
    "build_operations_api_contract",
    "SchemaField",
    "SurfaceSchema",
    "DashboardRunContract",
    "DashboardRunContractAudit",
    "DashboardRunContractLibrary",
    "render_dashboard_run_contract_report",
    "AutoTriageCenter",
    "ConsoleAction",
    "BillingEntitlementsPage",
    "BillingRunCharge",
    "DocumentationArtifact",
    "FinalDeliveryChecklist",
    "LaunchChecklist",
    "LaunchChecklistItem",
    "NarrativeSection",
    "IssueClosureDecision",
    "OrchestrationCanvas",
    "OrchestrationPortfolio",
    "PilotMetric",
    "PilotPortfolio",
    "PilotScorecard",
    "ReportStudio",
    "ReportStudioArtifacts",
    "SharedViewContext",
    "SharedViewFilter",
    "TakeoverQueue",
    "TakeoverRequest",
    "TriageFeedbackRecord",
    "TriageFinding",
    "TriageInboxItem",
    "TriageSimilarityEvidence",
    "TriageSuggestion",
    "build_auto_triage_center",
    "build_console_actions",
    "build_billing_entitlements_page",
    "build_billing_entitlements_page_from_ledger",
    "build_final_delivery_checklist",
    "build_launch_checklist",
    "build_orchestration_canvas",
    "build_orchestration_canvas_from_ledger_entry",
    "build_orchestration_portfolio",
    "build_orchestration_portfolio_from_ledger",
    "build_takeover_queue_from_ledger",
    "evaluate_issue_closure",
    "render_auto_triage_center_report",
    "render_console_actions",
    "render_billing_entitlements_page",
    "render_billing_entitlements_report",
    "render_final_delivery_checklist_report",
    "render_launch_checklist_report",
    "render_orchestration_canvas",
    "render_orchestration_overview_page",
    "render_orchestration_portfolio_report",
    "render_issue_validation_report",
    "render_report_studio_html",
    "render_report_studio_plain_text",
    "render_report_studio_report",
    "render_takeover_queue_report",
    "render_pilot_portfolio_report",
    "render_pilot_scorecard",
    "render_repo_sync_audit_report",
    "render_task_run_detail_page",
    "render_task_run_report",
    "validation_report_exists",
    "write_report",
    "write_report_studio_bundle",
    "AcceptanceDecision",
    "AcceptanceGate",
    "WorkflowEngine",
    "WorkflowRunResult",
    "WorkpadJournal",
    "DashboardBuilder",
    "DashboardBuilderAudit",
    "DashboardLayout",
    "DashboardWidgetPlacement",
    "DashboardWidgetSpec",
    "EngineeringActivity",
    "EngineeringFunnelStage",
    "EngineeringOverview",
    "EngineeringOverviewBlocker",
    "EngineeringOverviewKPI",
    "EngineeringOverviewPermission",
    "OperationsAnalytics",
    "OperationsMetricDefinition",
    "OperationsMetricSpec",
    "OperationsMetricValue",
    "OperationsSnapshot",
    "PolicyPromptVersionCenter",
    "RegressionCenter",
    "RegressionFinding",
    "TriageCluster",
    "WeeklyOperationsArtifacts",
    "WeeklyOperationsReport",
    "QueueControlCenter",
    "render_dashboard_builder_report",
    "VersionChangeSummary",
    "VersionedArtifact",
    "VersionedArtifactHistory",
    "render_engineering_overview",
    "render_operations_metric_spec",
    "render_operations_dashboard",
    "render_policy_prompt_version_center",
    "render_queue_control_center",
    "render_regression_center",
    "render_weekly_operations_report",
    "write_dashboard_builder_bundle",
    "write_engineering_overview_bundle",
    "write_weekly_operations_bundle",
    "BenchmarkCase",
    "BenchmarkComparison",
    "BenchmarkResult",
    "BenchmarkRunner",
    "BenchmarkSuiteResult",
    "EvaluationCriterion",
    "ReplayOutcome",
    "ReplayRecord",
    "render_run_replay_index_page",
    "render_replay_detail_page",
    "render_benchmark_suite_report",
    "CandidateBacklog",
    "CandidateEntry",
    "CandidatePlanner",
    "EvidenceLink",
    "EntryGate",
    "EntryGateDecision",
    "FourWeekExecutionPlan",
    "WeeklyExecutionPlan",
    "WeeklyGoal",
    "build_big_4701_execution_plan",
    "build_v3_candidate_backlog",
    "build_v3_entry_gate",
    "render_candidate_backlog_report",
    "render_four_week_execution_report",
    "InteractionFlow",
    "OpenQuestion",
    "ReviewBlocker",
    "ReviewBlockerEvent",
    "ReviewDecision",
    "ReviewObjective",
    "ReviewRoleAssignment",
    "ReviewSignoff",
    "ReviewerChecklistItem",
    "UIReviewPack",
    "UIReviewPackArtifacts",
    "UIReviewPackAudit",
    "UIReviewPackAuditor",
    "WireframeSurface",
    "build_big_4204_review_pack",
    "render_ui_review_blocker_log",
    "render_ui_review_blocker_timeline",
    "render_ui_review_blocker_timeline_summary",
    "render_ui_review_escalation_dashboard",
    "render_ui_review_escalation_handoff_ledger",
    "render_ui_review_exception_log",
    "render_ui_review_exception_matrix",
    "render_ui_review_freeze_approval_trail",
    "render_ui_review_freeze_exception_board",
    "render_ui_review_freeze_renewal_tracker",
    "render_ui_review_handoff_ack_ledger",
    "render_ui_review_interaction_coverage_board",
    "render_ui_review_objective_coverage_board",
    "render_ui_review_open_question_tracker",
    "render_ui_review_owner_escalation_digest",
    "render_ui_review_persona_readiness_board",
    "render_ui_review_review_summary_board",
    "render_ui_review_owner_review_queue",
    "render_ui_review_owner_workload_board",
    "render_ui_review_checklist_traceability_board",
    "render_ui_review_decision_followup_tracker",
    "render_ui_review_audit_density_board",
    "render_ui_review_reminder_cadence_board",
    "render_ui_review_role_coverage_board",
    "render_ui_review_wireframe_readiness_board",
    "render_ui_review_signoff_breach_board",
    "render_ui_review_signoff_dependency_board",
    "render_ui_review_signoff_reminder_queue",
    "render_ui_review_signoff_sla_dashboard",
    "render_ui_review_decision_log",
    "render_ui_review_pack_html",
    "render_ui_review_role_matrix",
    "render_ui_review_signoff_log",
    "render_ui_review_pack_report",
    "write_ui_review_pack_bundle",
]
