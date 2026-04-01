from __future__ import annotations

import argparse
import json
from pathlib import Path
import stat
import subprocess
import sys
import types
import warnings
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, Iterable, List, Optional, Sequence, Set

_models_module = types.ModuleType(f"{__name__}.models")
sys.modules[_models_module.__name__] = _models_module
exec(
    """
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List


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
        return {
            "name": self.name,
            "confidence": self.confidence,
            "source": self.source,
        }

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
""",
    _models_module.__dict__,
)
_models_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/domain/task.go"
globals()["models"] = _models_module

BillingInterval = _models_module.BillingInterval
BillingRate = _models_module.BillingRate
BillingSummary = _models_module.BillingSummary
FlowRun = _models_module.FlowRun
FlowRunStatus = _models_module.FlowRunStatus
FlowStepRun = _models_module.FlowStepRun
FlowStepStatus = _models_module.FlowStepStatus
FlowTemplate = _models_module.FlowTemplate
FlowTemplateStep = _models_module.FlowTemplateStep
FlowTrigger = _models_module.FlowTrigger
Priority = _models_module.Priority
RiskAssessment = _models_module.RiskAssessment
RiskLevel = _models_module.RiskLevel
RiskSignal = _models_module.RiskSignal
Task = _models_module.Task
TaskState = _models_module.TaskState
TriageLabel = _models_module.TriageLabel
TriageRecord = _models_module.TriageRecord
TriageStatus = _models_module.TriageStatus
UsageRecord = _models_module.UsageRecord


def _install_legacy_surface_module(name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    for export_name in export_names:
        module.__dict__[export_name] = getattr(_legacy_runtime_surface, export_name)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


@dataclass
class _RepoSpace:
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
    def from_dict(cls, data: Dict[str, Any]) -> "_RepoSpace":
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
class _RepoAgent:
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
    def from_dict(cls, data: Dict[str, Any]) -> "_RepoAgent":
        return cls(
            actor=str(data["actor"]),
            repo_agent_id=str(data["repo_agent_id"]),
            display_name=str(data.get("display_name", "")),
            roles=[str(item) for item in data.get("roles", [])],
        )


@dataclass
class _RunCommitLink:
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
    def from_dict(cls, data: Dict[str, Any]) -> "_RunCommitLink":
        return cls(
            run_id=str(data["run_id"]),
            commit_hash=str(data["commit_hash"]),
            role=str(data["role"]),
            repo_space_id=str(data["repo_space_id"]),
            actor=str(data.get("actor", "")),
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class _RunCommitBinding:
    links: List[_RunCommitLink]

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


_VALID_RUN_COMMIT_ROLES = {"source", "candidate", "closeout", "accepted"}


def _validate_run_commit_roles(links: Iterable[_RunCommitLink]) -> None:
    invalid = [link.role for link in links if link.role not in _VALID_RUN_COMMIT_ROLES]
    if invalid:
        invalid_text = ", ".join(sorted(set(invalid)))
        raise ValueError(f"unsupported run commit roles: {invalid_text}")


def _bind_run_commits(links: List[_RunCommitLink]) -> _RunCommitBinding:
    _validate_run_commit_roles(links)
    return _RunCommitBinding(links=list(links))


_repo_plane_module = types.ModuleType(f"{__name__}.repo_plane")
_repo_plane_module.RepoSpace = _RepoSpace
_repo_plane_module.RepoAgent = _RepoAgent
_repo_plane_module.RunCommitLink = _RunCommitLink
_repo_plane_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/repo/plane.go"
sys.modules[_repo_plane_module.__name__] = _repo_plane_module
globals()["repo_plane"] = _repo_plane_module

_repo_links_module = types.ModuleType(f"{__name__}.repo_links")
_repo_links_module.RunCommitBinding = _RunCommitBinding
_repo_links_module.VALID_ROLES = set(_VALID_RUN_COMMIT_ROLES)
_repo_links_module.validate_roles = _validate_run_commit_roles
_repo_links_module.bind_run_commits = _bind_run_commits
_repo_links_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/repo/links.go"
sys.modules[_repo_links_module.__name__] = _repo_links_module
globals()["repo_links"] = _repo_links_module


class _GitSyncError(RuntimeError):
    """Raised when repository sync automation cannot complete safely."""


@dataclass
class _RepoSyncStatus:
    branch: str
    local_sha: str
    remote_sha: str
    dirty: bool
    remote_exists: bool
    synced: bool
    pushed: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


@dataclass
class _GitCommandResult:
    stdout: str
    stderr: str
    returncode: int


_EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH


def _run_git_sync_command(command: Sequence[str], repo: Path) -> _GitCommandResult:
    completed = subprocess.run(
        list(command),
        cwd=repo,
        text=True,
        capture_output=True,
        check=False,
    )
    return _GitCommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git_sync_git(repo: Path, *args: str) -> _GitCommandResult:
    return _run_git_sync_command(["git", *args], repo)


def _require_git_sync(repo: Path, *args: str) -> str:
    result = _git_sync_git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise _GitSyncError(detail)
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
        raise _GitSyncError(detail)

    for line in symref_result.stdout.splitlines():
        if line.startswith("ref: ") and line.endswith("\tHEAD"):
            ref = line.split()[1]
            prefix = "refs/heads/"
            if ref.startswith(prefix):
                return ref[len(prefix) :]

    raise _GitSyncError(f"Could not determine default branch for remote {remote}")


def _git_sync_remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git_sync_git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git_sync_git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise _GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _git_sync_matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _git_sync_remote_default_branch(repo, remote)
        default_sha = _git_sync_remote_branch_sha(repo, remote, default_branch)
    except _GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


def _inspect_repo_sync(repo: Path | str, remote: str = "origin") -> _RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git_sync(repo_path, "branch", "--show-current")
    if not branch:
        raise _GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git_sync(repo_path, "rev-parse", "HEAD")
    remote_result = _git_sync_git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise _GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _git_sync_dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha

    if not remote_exists and _git_sync_matches_remote_default_branch(repo_path, remote, local_sha):
        synced = True

    return _RepoSyncStatus(
        branch=branch,
        local_sha=local_sha,
        remote_sha=remote_sha,
        dirty=dirty,
        remote_exists=remote_exists,
        synced=synced,
    )


def _install_git_hooks(repo: Path | str, hooks_path: str = ".githooks") -> Path:
    repo_path = Path(repo).resolve()
    hooks_dir = repo_path / hooks_path
    if not hooks_dir.is_dir():
        raise _GitSyncError(f"Missing hooks directory: {hooks_dir}")

    _require_git_sync(repo_path, "config", "core.hooksPath", hooks_path)
    for hook in hooks_dir.iterdir():
        if hook.is_file():
            hook.chmod(hook.stat().st_mode | _EXECUTABLE_BITS)
    return hooks_dir


def _ensure_repo_sync(
    repo: Path | str,
    remote: str = "origin",
    auto_push: bool = True,
    allow_dirty: bool = False,
) -> _RepoSyncStatus:
    repo_path = Path(repo).resolve()
    status = _inspect_repo_sync(repo_path, remote=remote)

    if status.dirty:
        if allow_dirty:
            return status
        raise _GitSyncError("Working tree is dirty; commit or stash changes before syncing")

    if status.remote_exists and not status.synced:
        fetch_result = _git_sync_git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise _GitSyncError(detail)

        ff_result = _git_sync_git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise _GitSyncError(detail)
        status = _inspect_repo_sync(repo_path, remote=remote)

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git_sync_git(repo_path, *push_args)
    if push_result.returncode != 0:
        detail = push_result.stderr or push_result.stdout or f"git {' '.join(push_args)} failed"
        raise _GitSyncError(detail)

    refreshed = _inspect_repo_sync(repo_path, remote=remote)
    refreshed.pushed = True
    if not refreshed.synced:
        raise _GitSyncError(
            f"Remote SHA mismatch after push: local={refreshed.local_sha} remote={refreshed.remote_sha or 'missing'}"
        )
    return refreshed


class _ParallelIssueQueue:
    def __init__(self, queue_path: str):
        self.queue_path = Path(queue_path)
        self.payload = json.loads(self.queue_path.read_text())

    def project_slug(self) -> str:
        return str(self.payload["project"]["slug_id"])

    def activate_state_id(self) -> str:
        return str(self.payload["policy"]["activate_state_id"])

    def target_in_progress(self) -> int:
        return int(self.payload["policy"]["target_in_progress"])

    def refill_states(self) -> Set[str]:
        return {str(name) for name in self.payload["policy"].get("refill_states", [])}

    def issue_order(self) -> List[str]:
        return [str(identifier) for identifier in self.payload.get("issue_order", [])]

    def issue_records(self) -> List[dict]:
        return list(self.payload.get("issues", []))

    def issue_identifiers(self) -> List[str]:
        return [str(record["identifier"]) for record in self.issue_records()]

    def select_candidates(
        self,
        active_identifiers: Set[str],
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


def _issue_state_map(issues: Sequence[dict]) -> Dict[str, str]:
    state_map: Dict[str, str] = {}
    for issue in issues:
        identifier = str(issue.get("identifier", "")).strip()
        state = issue.get("state") or {}
        state_name = str(state.get("name", issue.get("state_name", ""))).strip()
        if identifier and state_name:
            state_map[identifier] = state_name
    return state_map


_github_sync_module = types.ModuleType(f"{__name__}.github_sync")
_github_sync_module.GitSyncError = _GitSyncError
_github_sync_module.RepoSyncStatus = _RepoSyncStatus
_github_sync_module.CommandResult = _GitCommandResult
_github_sync_module.EXECUTABLE_BITS = _EXECUTABLE_BITS
_github_sync_module.inspect_repo_sync = _inspect_repo_sync
_github_sync_module.install_git_hooks = _install_git_hooks
_github_sync_module.ensure_repo_sync = _ensure_repo_sync
_github_sync_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/githubsync/sync.go"
sys.modules[_github_sync_module.__name__] = _github_sync_module
globals()["github_sync"] = _github_sync_module

_parallel_refill_module = types.ModuleType(f"{__name__}.parallel_refill")
_parallel_refill_module.ParallelIssueQueue = _ParallelIssueQueue
_parallel_refill_module.issue_state_map = _issue_state_map
_parallel_refill_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/refill/queue.go"
sys.modules[_parallel_refill_module.__name__] = _parallel_refill_module
globals()["parallel_refill"] = _parallel_refill_module


LEGACY_PYTHON_WRAPPER_NOTICE = (
    "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. "
    "This Python path remains only as a compatibility shim during migration."
)


def _append_missing_flag(args: Sequence[str], flag: str, value: str) -> List[str]:
    flag_prefix = flag + "="
    if any(arg == flag or arg.startswith(flag_prefix) for arg in args):
        return list(args)
    return [*args, flag, value]


def _build_bigclawctl_exec_args(repo_root: Path, command: Iterable[str], forwarded: Sequence[str]) -> List[str]:
    return ["bash", str(repo_root / "scripts/ops/bigclawctl"), *command, *forwarded]


def _repo_root_from_script(script_path: str) -> Path:
    return Path(script_path).resolve().parents[2]


def _run_bigclawctl_shim(script_path: str, command: Iterable[str], forwarded: Sequence[str]) -> int:
    repo_root = _repo_root_from_script(script_path)
    argv = _build_bigclawctl_exec_args(repo_root, command, forwarded)
    return subprocess.call(argv, cwd=repo_root)


def _build_workspace_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    args = list(forwarded)
    args = _append_missing_flag(args, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
    args = _append_missing_flag(args, "--cache-key", "openagis-bigclaw")
    return _build_bigclawctl_exec_args(repo_root, ["workspace", "bootstrap"], args)


def _translate_workspace_validate_args(forwarded: Sequence[str]) -> List[str]:
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


def _build_workspace_validate_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return _build_bigclawctl_exec_args(repo_root, ["workspace", "validate"], _translate_workspace_validate_args(forwarded))


def _build_github_sync_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return _build_bigclawctl_exec_args(repo_root, ["github-sync"], list(forwarded))


def _build_refill_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return _build_bigclawctl_exec_args(repo_root, ["refill"], list(forwarded))


def _build_workspace_runtime_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return _build_bigclawctl_exec_args(repo_root, ["workspace"], list(forwarded))


_legacy_shim_module = types.ModuleType(f"{__name__}.legacy_shim")
_legacy_shim_module.LEGACY_PYTHON_WRAPPER_NOTICE = LEGACY_PYTHON_WRAPPER_NOTICE
_legacy_shim_module.append_missing_flag = _append_missing_flag
_legacy_shim_module.build_bigclawctl_exec_args = _build_bigclawctl_exec_args
_legacy_shim_module.repo_root_from_script = _repo_root_from_script
_legacy_shim_module.run_bigclawctl_shim = _run_bigclawctl_shim
_legacy_shim_module.build_workspace_bootstrap_args = _build_workspace_bootstrap_args
_legacy_shim_module.translate_workspace_validate_args = _translate_workspace_validate_args
_legacy_shim_module.build_workspace_validate_args = _build_workspace_validate_args
_legacy_shim_module.build_github_sync_args = _build_github_sync_args
_legacy_shim_module.build_refill_args = _build_refill_args
_legacy_shim_module.build_workspace_runtime_bootstrap_args = _build_workspace_runtime_bootstrap_args
_legacy_shim_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/legacyshim/wrappers.go"
sys.modules[_legacy_shim_module.__name__] = _legacy_shim_module
globals()["legacy_shim"] = _legacy_shim_module


@dataclass
class _RiskFactor:
    name: str
    points: int
    reason: str


@dataclass
class _RiskScore:
    level: RiskLevel
    total: int
    requires_approval: bool
    factors: List[_RiskFactor] = field(default_factory=list)

    @property
    def summary(self) -> str:
        return ", ".join(f"{factor.name}={factor.points}" for factor in self.factors) or "baseline=0"


class _RiskScorer:
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

    def score_task(self, task: Task) -> _RiskScore:
        factors: List[_RiskFactor] = []
        total = 0

        risk_points = {
            RiskLevel.LOW: 0,
            RiskLevel.MEDIUM: 30,
            RiskLevel.HIGH: 60,
        }[task.risk_level]
        total += risk_points
        if risk_points:
            factors.append(_RiskFactor("risk-level", risk_points, f"declared risk level {task.risk_level.value}"))

        if task.priority == Priority.P0:
            total += 10
            factors.append(_RiskFactor("priority", 10, "p0 task needs tighter controls"))

        labels = {label.lower() for label in task.labels}
        for label in sorted(labels):
            points = self.LABEL_POINTS.get(label)
            if points:
                total += points
                factors.append(_RiskFactor(f"label:{label}", points, f"label {label} increases operational risk"))

        tools = {tool.lower() for tool in task.required_tools}
        for tool in sorted(tools):
            points = self.TOOL_POINTS.get(tool)
            if points:
                total += points
                factors.append(_RiskFactor(f"tool:{tool}", points, f"tool {tool} expands execution surface"))

        if task.budget < 0:
            total += 20
            factors.append(_RiskFactor("budget", 20, "invalid budget requires manual review"))

        level = self._level_for_total(total)
        return _RiskScore(level=level, total=total, requires_approval=(level == RiskLevel.HIGH), factors=factors)

    def _level_for_total(self, total: int) -> RiskLevel:
        if total >= 60:
            return RiskLevel.HIGH
        if total >= 25:
            return RiskLevel.MEDIUM
        return RiskLevel.LOW


_risk_module = types.ModuleType(f"{__name__}.risk")
_risk_module.RiskFactor = _RiskFactor
_risk_module.RiskScore = _RiskScore
_risk_module.RiskScorer = _RiskScorer
_risk_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/risk/risk.go"
sys.modules[_risk_module.__name__] = _risk_module
globals()["risk"] = _risk_module


REQUIRED_RUN_CLOSEOUTS = ("validation-evidence", "git-push", "git-log-stat")
ALLOWED_SCOPE_STATUSES = {"frozen", "approved-exception", "proposed"}


@dataclass(frozen=True)
class _FreezeException:
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
    def from_dict(cls, data: Dict[str, str]) -> "_FreezeException":
        return cls(
            issue_id=data["issue_id"],
            reason=data.get("reason", ""),
            approved_by=data.get("approved_by", ""),
            decision_note=data.get("decision_note", ""),
        )


@dataclass
class _GovernanceBacklogItem:
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
    def from_dict(cls, data: Dict[str, object]) -> "_GovernanceBacklogItem":
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
class _ScopeFreezeBoard:
    name: str
    version: str
    freeze_date: str
    freeze_owner: str
    backlog_items: List[_GovernanceBacklogItem] = field(default_factory=list)
    exceptions: List[_FreezeException] = field(default_factory=list)

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
    def from_dict(cls, data: Dict[str, object]) -> "_ScopeFreezeBoard":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            freeze_date=str(data.get("freeze_date", "")),
            freeze_owner=str(data.get("freeze_owner", "")),
            backlog_items=[_GovernanceBacklogItem.from_dict(item) for item in data.get("backlog_items", [])],
            exceptions=[_FreezeException.from_dict(item) for item in data.get("exceptions", [])],
        )


@dataclass
class _ScopeFreezeAudit:
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
    def from_dict(cls, data: Dict[str, object]) -> "_ScopeFreezeAudit":
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


class _ScopeFreezeGovernance:
    def audit(self, board: _ScopeFreezeBoard) -> _ScopeFreezeAudit:
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

        unapproved_exceptions = sorted(
            exception.issue_id for exception in board.exceptions if not exception.approved
        )

        return _ScopeFreezeAudit(
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


def _render_scope_freeze_report(board: _ScopeFreezeBoard, audit: _ScopeFreezeAudit) -> str:
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
    lines.append(
        f"- Duplicate issues: {', '.join(audit.duplicate_issue_ids) if audit.duplicate_issue_ids else 'none'}"
    )
    lines.append(f"- Missing owners: {', '.join(audit.missing_owners) if audit.missing_owners else 'none'}")
    lines.append(
        f"- Missing acceptance: {', '.join(audit.missing_acceptance) if audit.missing_acceptance else 'none'}"
    )
    lines.append(
        f"- Missing validation: {', '.join(audit.missing_validation) if audit.missing_validation else 'none'}"
    )
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
    lines.append(
        f"- Invalid scope statuses: {', '.join(audit.invalid_scope_statuses) if audit.invalid_scope_statuses else 'none'}"
    )
    lines.append(
        f"- Unapproved exceptions: {', '.join(audit.unapproved_exceptions) if audit.unapproved_exceptions else 'none'}"
    )
    return "\n".join(lines) + "\n"


_governance_module = types.ModuleType(f"{__name__}.governance")
_governance_module.REQUIRED_RUN_CLOSEOUTS = REQUIRED_RUN_CLOSEOUTS
_governance_module.ALLOWED_SCOPE_STATUSES = set(ALLOWED_SCOPE_STATUSES)
_governance_module.FreezeException = _FreezeException
_governance_module.GovernanceBacklogItem = _GovernanceBacklogItem
_governance_module.ScopeFreezeBoard = _ScopeFreezeBoard
_governance_module.ScopeFreezeAudit = _ScopeFreezeAudit
_governance_module.ScopeFreezeGovernance = _ScopeFreezeGovernance
_governance_module.render_scope_freeze_report = _render_scope_freeze_report
_governance_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/governance/freeze.go"
sys.modules[_governance_module.__name__] = _governance_module
globals()["governance"] = _governance_module

LEGACY_RUNTIME_GUIDANCE = (
    "bigclaw-go is the sole implementation mainline for active development; "
    "the legacy Python runtime surface remains migration-only."
)


def _legacy_runtime_message(surface: str, replacement: str) -> str:
    return f"{surface} is frozen for migration-only use. {LEGACY_RUNTIME_GUIDANCE} Use {replacement} instead."


def _warn_legacy_runtime_surface(surface: str, replacement: str) -> str:
    message = _legacy_runtime_message(surface, replacement)
    warnings.warn(message, DeprecationWarning, stacklevel=2)
    return message


_deprecation_module = types.ModuleType(f"{__name__}.deprecation")
_deprecation_module.LEGACY_RUNTIME_GUIDANCE = LEGACY_RUNTIME_GUIDANCE
_deprecation_module.legacy_runtime_message = _legacy_runtime_message
_deprecation_module.warn_legacy_runtime_surface = _warn_legacy_runtime_surface
_deprecation_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/regression/deprecation_contract_test.go"
sys.modules[_deprecation_module.__name__] = _deprecation_module
globals()["deprecation"] = _deprecation_module

_execution_contract_module = types.ModuleType(f"{__name__}.execution_contract")
sys.modules[_execution_contract_module.__name__] = _execution_contract_module
exec(
    """
from dataclasses import dataclass, field
from typing import Dict, List, Optional


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
        return {
            "name": self.name,
            "fields": [field.to_dict() for field in self.fields],
            "owner": self.owner,
        }

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
        return PermissionCheckResult(
            allowed=not missing,
            granted_permissions=sorted(granted_set),
            missing_permissions=missing,
        )

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
        return {
            "name": self.name,
            "unit": self.unit,
            "owner": self.owner,
            "description": self.description,
        }

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
            "models_missing_required_fields": {
                name: list(fields) for name, fields in self.models_missing_required_fields.items()
            },
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
            models_missing_required_fields={
                str(name): [str(field) for field in fields]
                for name, fields in dict(data.get("models_missing_required_fields", {})).items()
            },
            apis_missing_permissions=[str(item) for item in data.get("apis_missing_permissions", [])],
            apis_missing_audits=[str(item) for item in data.get("apis_missing_audits", [])],
            apis_missing_metrics=[str(item) for item in data.get("apis_missing_metrics", [])],
            undefined_model_refs={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_model_refs", {})).items()
            },
            undefined_permissions={str(name): str(value) for name, value in dict(data.get("undefined_permissions", {})).items()},
            missing_roles=[str(item) for item in data.get("missing_roles", [])],
            roles_missing_personas=[str(item) for item in data.get("roles_missing_personas", [])],
            roles_missing_scope_bindings=[str(item) for item in data.get("roles_missing_scope_bindings", [])],
            roles_missing_escalation_targets=[str(item) for item in data.get("roles_missing_escalation_targets", [])],
            roles_missing_permissions=[str(item) for item in data.get("roles_missing_permissions", [])],
            undefined_role_permissions={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_role_permissions", {})).items()
            },
            permissions_without_roles=[str(item) for item in data.get("permissions_without_roles", [])],
            apis_without_role_coverage=[str(item) for item in data.get("apis_without_role_coverage", [])],
            undefined_metrics={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_metrics", {})).items()
            },
            undefined_audit_events={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_audit_events", {})).items()
            },
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
            missing_models = [
                model_name
                for model_name in [api.request_model, api.response_model]
                if model_name and model_name not in model_names
            ]
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
            permissions_granted_by_roles.update(
                permission for permission in role.granted_permissions if permission in permission_names
            )

        for api in contract.apis:
            if api.required_permission and api.required_permission in permission_names and api.required_permission not in permissions_granted_by_roles:
                apis_without_role_coverage.append(api.name)

        permissions_without_roles = sorted(permission for permission in permission_names if permission not in permissions_granted_by_roles)

        audit_policies_below_retention = sorted(
            policy.event_type for policy in contract.audit_policies if policy.retention_days < 30
        )

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
            lines.append(
                f"- {role.name}: personas={personas} permissions={permissions} scopes={scopes} escalation={escalation_target}"
            )
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
    return "\\n".join(lines)


def build_operations_api_contract(contract_id: str = "OPE-131", version: str = "v4.0-draft1") -> ExecutionContract:
    return ExecutionContract(
        contract_id=contract_id,
        version=version,
        models=[
            ExecutionModel(
                name="OperationsDashboardResponse",
                owner="operations",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("total_runs", "int"),
                    ExecutionField("success_rate", "float"),
                    ExecutionField("approval_queue_depth", "int"),
                    ExecutionField("sla_breach_count", "int"),
                    ExecutionField("top_blockers", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="RunDetailResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("task_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("timeline_events", "RunDetailEvent[]"),
                    ExecutionField("resources", "RunDetailResource[]"),
                    ExecutionField("audit_count", "int"),
                ],
            ),
            ExecutionModel(
                name="RunReplayResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("replay_available", "bool"),
                    ExecutionField("replay_path", "string", required=False),
                    ExecutionField("benchmark_case_ids", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="QueueControlCenterResponse",
                owner="operations",
                fields=[
                    ExecutionField("queue_depth", "int"),
                    ExecutionField("queued_by_priority", "map<string,int>"),
                    ExecutionField("queued_by_risk", "map<string,int>"),
                    ExecutionField("waiting_approval_runs", "int"),
                    ExecutionField("blocked_tasks", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="QueueActionRequest",
                owner="operations",
                fields=[
                    ExecutionField("actor", "string"),
                    ExecutionField("reason", "string"),
                ],
            ),
            ExecutionModel(
                name="QueueActionResponse",
                owner="operations",
                fields=[
                    ExecutionField("task_id", "string"),
                    ExecutionField("action", "string"),
                    ExecutionField("accepted", "bool"),
                    ExecutionField("queue_depth", "int"),
                ],
            ),
            ExecutionModel(
                name="RunApprovalRequest",
                owner="operations",
                fields=[
                    ExecutionField("actor", "string"),
                    ExecutionField("approval_token", "string"),
                    ExecutionField("decision", "string"),
                    ExecutionField("reason", "string", required=False),
                ],
            ),
            ExecutionModel(
                name="RunApprovalResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("approved", "bool"),
                    ExecutionField("required_follow_up", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="RiskOverviewResponse",
                owner="risk",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("high_risk_runs", "int"),
                    ExecutionField("approval_required_runs", "int"),
                    ExecutionField("risk_factors", "string[]"),
                    ExecutionField("recommendation", "string"),
                ],
            ),
            ExecutionModel(
                name="SlaOverviewResponse",
                owner="operations",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("sla_target_minutes", "int"),
                    ExecutionField("average_cycle_minutes", "float"),
                    ExecutionField("sla_breach_count", "int"),
                    ExecutionField("approval_queue_depth", "int"),
                ],
            ),
            ExecutionModel(
                name="RegressionCenterResponse",
                owner="operations",
                fields=[
                    ExecutionField("baseline_version", "string"),
                    ExecutionField("current_version", "string"),
                    ExecutionField("regression_count", "int"),
                    ExecutionField("improved_cases", "string[]", required=False),
                    ExecutionField("regressions", "RegressionFinding[]", required=False),
                ],
            ),
            ExecutionModel(
                name="FlowCanvasResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("collaboration_mode", "string"),
                    ExecutionField("departments", "string[]"),
                    ExecutionField("required_approvals", "string[]", required=False),
                    ExecutionField("billing_model", "string"),
                    ExecutionField("recommendation", "string"),
                ],
            ),
            ExecutionModel(
                name="BillingEntitlementsResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("tier", "string"),
                    ExecutionField("billing_model_counts", "map<string,int>"),
                    ExecutionField("upgrade_required_runs", "int"),
                    ExecutionField("estimated_cost_usd", "float"),
                ],
            ),
            ExecutionModel(
                name="BillingRunChargeResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("billing_model", "string"),
                    ExecutionField("estimated_cost_usd", "float"),
                    ExecutionField("overage_cost_usd", "float"),
                    ExecutionField("upgrade_required", "bool"),
                ],
            ),
        ],
        apis=[
            ExecutionApiSpec(
                name="get_operations_dashboard",
                method="GET",
                path="/operations/dashboard",
                request_model="",
                response_model="OperationsDashboardResponse",
                required_permission="operations.dashboard.read",
                emitted_audits=["operations.dashboard.viewed"],
                emitted_metrics=["operations.dashboard.requests", "operations.dashboard.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_run_detail",
                method="GET",
                path="/operations/runs/{run_id}",
                request_model="",
                response_model="RunDetailResponse",
                required_permission="operations.run.read",
                emitted_audits=["operations.run_detail.viewed"],
                emitted_metrics=["operations.run_detail.requests", "operations.run_detail.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_run_replay",
                method="GET",
                path="/operations/runs/{run_id}/replay",
                request_model="",
                response_model="RunReplayResponse",
                required_permission="operations.run.read",
                emitted_audits=["operations.run_replay.viewed"],
                emitted_metrics=["operations.run_replay.requests", "operations.run_replay.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_queue_control_center",
                method="GET",
                path="/operations/queue/control-center",
                request_model="",
                response_model="QueueControlCenterResponse",
                required_permission="operations.queue.read",
                emitted_audits=["operations.queue.viewed"],
                emitted_metrics=["operations.queue.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="retry_queue_task",
                method="POST",
                path="/operations/queue/{task_id}/retry",
                request_model="QueueActionRequest",
                response_model="QueueActionResponse",
                required_permission="operations.queue.act",
                emitted_audits=["operations.queue.retry.requested"],
                emitted_metrics=["operations.queue.retry.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="approve_run_execution",
                method="POST",
                path="/operations/runs/{run_id}/approve",
                request_model="RunApprovalRequest",
                response_model="RunApprovalResponse",
                required_permission="operations.run.approve",
                emitted_audits=["operations.run.approval.recorded"],
                emitted_metrics=["operations.run.approval.requests", "operations.approval.queue.depth"],
            ),
            ExecutionApiSpec(
                name="get_risk_overview",
                method="GET",
                path="/operations/risk/overview",
                request_model="",
                response_model="RiskOverviewResponse",
                required_permission="operations.risk.read",
                emitted_audits=["operations.risk.viewed"],
                emitted_metrics=["operations.risk.requests", "operations.risk.high_runs"],
            ),
            ExecutionApiSpec(
                name="get_sla_overview",
                method="GET",
                path="/operations/sla/overview",
                request_model="",
                response_model="SlaOverviewResponse",
                required_permission="operations.sla.read",
                emitted_audits=["operations.sla.viewed"],
                emitted_metrics=["operations.sla.requests", "operations.sla.breaches"],
            ),
            ExecutionApiSpec(
                name="get_regression_center",
                method="GET",
                path="/operations/regressions",
                request_model="",
                response_model="RegressionCenterResponse",
                required_permission="operations.regression.read",
                emitted_audits=["operations.regression.viewed"],
                emitted_metrics=["operations.regression.requests", "operations.regression.count"],
            ),
            ExecutionApiSpec(
                name="get_flow_canvas",
                method="GET",
                path="/operations/flows/{run_id}",
                request_model="",
                response_model="FlowCanvasResponse",
                required_permission="operations.flow.read",
                emitted_audits=["operations.flow.viewed"],
                emitted_metrics=["operations.flow.requests", "operations.flow.handoff_count"],
            ),
            ExecutionApiSpec(
                name="get_billing_entitlements",
                method="GET",
                path="/operations/billing/entitlements",
                request_model="",
                response_model="BillingEntitlementsResponse",
                required_permission="operations.billing.read",
                emitted_audits=["operations.billing.viewed"],
                emitted_metrics=["operations.billing.requests", "operations.billing.estimated_cost_usd"],
            ),
            ExecutionApiSpec(
                name="get_billing_run_charge",
                method="GET",
                path="/operations/billing/runs/{run_id}",
                request_model="",
                response_model="BillingRunChargeResponse",
                required_permission="operations.billing.read",
                emitted_audits=["operations.billing.run_charge.viewed"],
                emitted_metrics=["operations.billing.run_charge.requests", "operations.billing.overage_cost_usd"],
            ),
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
            ExecutionRole(
                name="eng-lead",
                personas=["Eng Lead"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.run.approve",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                ],
                scope_bindings=["team", "workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="platform-admin",
                personas=["Platform Admin"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.queue.act",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                    "operations.flow.read",
                    "operations.billing.read",
                ],
                scope_bindings=["workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="vp-eng",
                personas=["VP Eng"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.run.approve",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                    "operations.billing.read",
                ],
                scope_bindings=["portfolio", "workspace"],
                escalation_target="none",
            ),
            ExecutionRole(
                name="cross-team-operator",
                personas=["Cross-Team Operator"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.queue.act",
                    "operations.flow.read",
                    "operations.billing.read",
                ],
                scope_bindings=["cross-team", "team", "workspace"],
                escalation_target="eng-lead",
            ),
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
""",
    _execution_contract_module.__dict__,
)
_execution_contract_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/contract/execution.go"
globals()["execution_contract"] = _execution_contract_module

_run_detail_module = types.ModuleType(f"{__name__}.run_detail")
sys.modules[_run_detail_module.__name__] = _run_detail_module
exec(
    '''
from dataclasses import dataclass, field
from html import escape
import json
from typing import List


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
''',
    _run_detail_module.__dict__,
)
_run_detail_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/product/dashboard_run_contract.go"
globals()["run_detail"] = _run_detail_module

SCHEDULER_DECISION_EVENT = "execution.scheduler_decision"
MANUAL_TAKEOVER_EVENT = "execution.manual_takeover"
APPROVAL_RECORDED_EVENT = "execution.approval_recorded"
BUDGET_OVERRIDE_EVENT = "execution.budget_override"
FLOW_HANDOFF_EVENT = "execution.flow_handoff"


@dataclass(frozen=True)
class _AuditEventSpec:
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
    def from_dict(cls, data: Dict[str, object]) -> "_AuditEventSpec":
        return cls(
            event_type=str(data["event_type"]),
            description=str(data["description"]),
            severity=str(data["severity"]),
            retention_days=int(data["retention_days"]),
            required_fields=[str(value) for value in data.get("required_fields", [])],
        )


_P0_AUDIT_EVENT_SPECS: List[_AuditEventSpec] = [
    _AuditEventSpec(
        event_type=SCHEDULER_DECISION_EVENT,
        description="Records the scheduler routing decision and risk context for a run.",
        severity="info",
        retention_days=180,
        required_fields=["task_id", "run_id", "medium", "approved", "reason", "risk_level", "risk_score"],
    ),
    _AuditEventSpec(
        event_type=MANUAL_TAKEOVER_EVENT,
        description="Captures escalation into a human takeover queue.",
        severity="warn",
        retention_days=365,
        required_fields=["task_id", "run_id", "target_team", "reason", "requested_by", "required_approvals"],
    ),
    _AuditEventSpec(
        event_type=APPROVAL_RECORDED_EVENT,
        description="Records explicit human approvals attached to a run or acceptance decision.",
        severity="info",
        retention_days=365,
        required_fields=["task_id", "run_id", "approvals", "approval_count", "acceptance_status"],
    ),
    _AuditEventSpec(
        event_type=BUDGET_OVERRIDE_EVENT,
        description="Captures a manual override to the run budget envelope.",
        severity="warn",
        retention_days=365,
        required_fields=["task_id", "run_id", "requested_budget", "approved_budget", "override_actor", "reason"],
    ),
    _AuditEventSpec(
        event_type=FLOW_HANDOFF_EVENT,
        description="Captures ownership transfer between automated flow stages and teams.",
        severity="info",
        retention_days=180,
        required_fields=["task_id", "run_id", "source_stage", "target_team", "reason", "collaboration_mode"],
    ),
]

_AUDIT_SPEC_BY_EVENT = {spec.event_type: spec for spec in _P0_AUDIT_EVENT_SPECS}


def _get_audit_event_spec(event_type: str) -> Optional[_AuditEventSpec]:
    return _AUDIT_SPEC_BY_EVENT.get(event_type)


def _missing_required_fields(event_type: str, details: Dict[str, object]) -> List[str]:
    spec = _get_audit_event_spec(event_type)
    if spec is None:
        return []
    return [field for field in spec.required_fields if field not in details]


_audit_events_module = types.ModuleType(f"{__name__}.audit_events")
_audit_events_module.SCHEDULER_DECISION_EVENT = SCHEDULER_DECISION_EVENT
_audit_events_module.MANUAL_TAKEOVER_EVENT = MANUAL_TAKEOVER_EVENT
_audit_events_module.APPROVAL_RECORDED_EVENT = APPROVAL_RECORDED_EVENT
_audit_events_module.BUDGET_OVERRIDE_EVENT = BUDGET_OVERRIDE_EVENT
_audit_events_module.FLOW_HANDOFF_EVENT = FLOW_HANDOFF_EVENT
_audit_events_module.AuditEventSpec = _AuditEventSpec
_audit_events_module.P0_AUDIT_EVENT_SPECS = list(_P0_AUDIT_EVENT_SPECS)
_audit_events_module.get_audit_event_spec = _get_audit_event_spec
_audit_events_module.missing_required_fields = _missing_required_fields
_audit_events_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/observability/audit_spec.go"
sys.modules[_audit_events_module.__name__] = _audit_events_module
globals()["audit_events"] = _audit_events_module

_workspace_bootstrap_module = types.ModuleType(f"{__name__}.workspace_bootstrap")
sys.modules[_workspace_bootstrap_module.__name__] = _workspace_bootstrap_module
exec(
    """
from __future__ import annotations

import json
import shutil
import subprocess
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Sequence
from urllib.parse import urlparse


class WorkspaceBootstrapError(RuntimeError):
    \"\"\"Raised when the shared-worktree bootstrap flow cannot complete.\"\"\"


@dataclass
class CacheBootstrapState:
    cache_root: str
    cache_key: str
    mirror_path: str
    seed_path: str
    mirror_created: bool
    seed_created: bool

    def to_dict(self) -> dict[str, Any]:
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

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")


def _run(command: Sequence[str], cwd: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=cwd,
        text=True,
        capture_output=True,
        check=False,
    )
    return CommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git(repo: Path, *args: str) -> CommandResult:
    return _run(["git", *args], repo)


def _require_git(repo: Path, *args: str) -> str:
    result = _git(repo, *args)
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
        _require_git(mirror_path, "remote", "set-url", "origin", repo_url)
        _require_git(mirror_path, "fetch", "--prune", "origin")

    return _cache_state(repo_url, repo_cache_root, cache_key, mirror_created=mirror_created)


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
    _require_git(seed_path, "fetch", "--prune", CACHE_REMOTE)
    _require_git(seed_path, "worktree", "prune")
    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
    return _cache_state(
        repo_url,
        Path(cache_state.cache_root),
        cache_key,
        mirror_created=cache_state.mirror_created,
        seed_created=seed_created,
    )


def configure_seed_remotes(seed_path: Path, repo_url: str, mirror_path: Path) -> None:
    remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes and "origin" in remotes:
        current_origin = _require_git(seed_path, "remote", "get-url", "origin")
        if Path(current_origin).expanduser().resolve() == mirror_path.resolve():
            _require_git(seed_path, "remote", "rename", "origin", CACHE_REMOTE)
            remotes = set(_require_git(seed_path, "remote").splitlines())

    if CACHE_REMOTE not in remotes:
        _require_git(seed_path, "remote", "add", CACHE_REMOTE, str(mirror_path))
    else:
        _require_git(seed_path, "remote", "set-url", CACHE_REMOTE, str(mirror_path))

    remotes = set(_require_git(seed_path, "remote").splitlines())
    if "origin" not in remotes:
        _require_git(seed_path, "remote", "add", "origin", repo_url)
    else:
        _require_git(seed_path, "remote", "set-url", "origin", repo_url)

    _require_git(seed_path, "config", "remote.pushDefault", "origin")


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
        current_branch = _require_git(workspace_path, "branch", "--show-current")
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

    _require_git(seed_path, "worktree", "add", "--force", "-B", branch, str(workspace_path), f"{CACHE_REMOTE}/{default_branch}")

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

    if _git(workspace_path, "rev-parse", "--git-dir").returncode == 0:
        current_branch = _require_git(workspace_path, "branch", "--show-current")
        branch = current_branch or branch

    worktree_list = _require_git(seed_path, "worktree", "list", "--porcelain")
    registered = f"worktree {workspace_path}" in worktree_list
    if registered:
        _require_git(seed_path, "worktree", "remove", "--force", str(workspace_path))
        _require_git(seed_path, "worktree", "prune")

    local_branches = set(_require_git(seed_path, "branch", "--format", "%(refname:short)").splitlines())
    if branch.startswith(f"{BOOTSTRAP_BRANCH_PREFIX}/") and branch in local_branches:
        _require_git(seed_path, "branch", "-D", branch)

    _require_git(seed_path, "checkout", "-B", default_branch, f"{CACHE_REMOTE}/{default_branch}")
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
""",
    _workspace_bootstrap_module.__dict__,
)
_workspace_bootstrap_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/bootstrap/bootstrap.go"
globals()["workspace_bootstrap"] = _workspace_bootstrap_module

_collaboration_module = types.ModuleType(f"{__name__}.collaboration")
sys.modules[_collaboration_module.__name__] = _collaboration_module
exec(
    """
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from html import escape
from typing import Any, Dict, List, Optional, Sequence


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
        participants = {comment.author for comment in self.comments} | {decision.author for decision in self.decisions}
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
        return f'''
        <section class="surface">
          <h2>{escape(title)}</h2>
          <p>{escape(description)}</p>
          <div class="empty">No collaboration recorded for this surface.</div>
        </section>
        '''

    comment_items = "".join(
        f'''
        <article class="resource-card">
          <span class="kicker">{escape(comment.comment_id)}</span>
          <strong>{escape(comment.author)}</strong>
          <p>{escape(comment.body)}</p>
          <span class="resource-meta">status={escape(comment.status)} | anchor={escape(comment.anchor or "none")} | mentions={escape(", ".join(comment.mentions) if comment.mentions else "none")}</span>
        </article>
        '''
        for comment in thread.comments
    ) or '<div class="empty">No comments recorded.</div>'
    decision_items = "".join(
        f'''
        <article class="resource-card" data-tone="report">
          <span class="kicker">{escape(decision.decision_id)}</span>
          <strong>{escape(decision.outcome)}</strong>
          <p>{escape(decision.summary)}</p>
          <span class="resource-meta">author={escape(decision.author)} | mentions={escape(", ".join(decision.mentions) if decision.mentions else "none")} | related={escape(", ".join(decision.related_comment_ids) if decision.related_comment_ids else "none")} | follow_up={escape(decision.follow_up or "none")}</span>
        </article>
        '''
        for decision in thread.decisions
    ) or '<div class="empty">No decision notes recorded.</div>'

    return f'''
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
    '''


GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/api/v2.go"
""",
    _collaboration_module.__dict__,
)
globals()["collaboration"] = _collaboration_module

_planning_module = types.ModuleType(f"{__name__}.planning")
sys.modules[_planning_module.__name__] = _planning_module
exec(
    """
from dataclasses import dataclass, field
from typing import Dict, List, Optional

from .governance import ScopeFreezeAudit


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
    return "\\n".join(lines)


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
                    "PYTHONPATH=src python3 -m pytest tests/test_ui_review.py -q && "
                    "(cd bigclaw-go && go test ./internal/product ./internal/api)"
                ),
                capabilities=["release-gate", "console-shell", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="design-system-audit",
                        target="bigclaw-go/internal/product/console.go",
                        capability="release-gate",
                        note="Go-owned navigation, home cards, and console design system defaults",
                    ),
                    EvidenceLink(
                        label="console-ia-contract",
                        target="bigclaw-go/internal/api/v2.go",
                        capability="release-gate",
                        note="Go-owned console navigation, saved-view, and operator API contract surfaces",
                    ),
                    EvidenceLink(
                        label="ui-review-pack",
                        target="src/bigclaw/ui_review.py",
                        capability="release-gate",
                        note="review objectives, wireframes, interaction coverage, and open questions",
                    ),
                    EvidenceLink(
                        label="ui-acceptance-tests",
                        target="bigclaw-go/internal/product/console_test.go",
                        capability="release-gate",
                        note="Go console design-system and navigation coverage",
                    ),
                    EvidenceLink(
                        label="console-shell-tests",
                        target="bigclaw-go/internal/api/expansion_test.go",
                        capability="release-gate",
                        note="Go console shell, saved-view, and operator endpoint coverage",
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
                    "tests/test_evaluation.py -q && "
                    "(cd bigclaw-go && go test ./internal/product ./internal/api ./internal/worker ./internal/workflow ./internal/scheduler)"
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
                        target="bigclaw-go/internal/product/saved_views.go",
                        capability="saved-views",
                        note="Go-owned saved views, digest subscriptions, and governed filters",
                    ),
                    EvidenceLink(
                        label="saved-views-tests",
                        target="bigclaw-go/internal/product/saved_views_test.go",
                        capability="saved-views",
                        note="Go saved-view audit coverage",
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
    return "\\n".join(lines)


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
    return "\\n".join(lines)
""",
    _planning_module.__dict__,
)
_planning_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/api/admission_policy_surface.go"
globals()["planning"] = _planning_module

_evaluation_module = types.ModuleType(f"{__name__}.evaluation")
sys.modules[_evaluation_module.__name__] = _evaluation_module
exec(
    """
from __future__ import annotations

from dataclasses import dataclass, field
from html import escape
from pathlib import Path
from typing import List, Optional

from .models import Task
from .run_detail import (
    RunDetailEvent,
    RunDetailResource,
    RunDetailStat,
    RunDetailTab,
    render_resource_grid,
    render_run_detail_console,
    render_timeline_panel,
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
    def from_execution(cls, task: Task, run_id: str, record: ExecutionRecord) -> "ReplayRecord":
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
    record: ExecutionRecord
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
    def __init__(self, scheduler: Optional[Scheduler] = None, storage_dir: Optional[str] = None):
        self.scheduler = scheduler or _scheduler_type()
        self.storage_dir = Path(storage_dir) if storage_dir else None

    def run_case(self, case: BenchmarkCase) -> BenchmarkResult:
        ledger = _observability_ledger(str(self._case_path(case.case_id, "ledger.json")))
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
            _write_report(detail_page_path, render_run_replay_index_page(case.case_id, record, replay, criteria))
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
        return BenchmarkSuiteResult(
            results=[self.run_case(case) for case in cases],
            version=version,
        )

    def replay(self, replay_record: ReplayRecord) -> ReplayOutcome:
        ledger = _observability_ledger(str(self._case_path(replay_record.run_id, "replay-ledger.json")))
        replayed = self.scheduler.execute(
            replay_record.task,
            run_id=f"{replay_record.run_id}-replay",
            ledger=ledger,
            actor="benchmark-replay",
        )
        observed = ReplayRecord.from_execution(
            replay_record.task,
            replay_record.run_id,
            replayed,
        )
        mismatches = []
        if observed.medium != replay_record.medium:
            mismatches.append(f"medium expected {replay_record.medium} got {observed.medium}")
        if observed.approved != replay_record.approved:
            mismatches.append(
                f"approved expected {replay_record.approved} got {observed.approved}"
            )
        if observed.status != replay_record.status:
            mismatches.append(f"status expected {replay_record.status} got {observed.status}")
        report_path = None
        if self.storage_dir is not None:
            report_path = str(self._case_path(replay_record.run_id, "replay.html"))
            _write_report(report_path, render_replay_detail_page(replay_record, observed, mismatches))
        return ReplayOutcome(
            matched=not mismatches,
            replay_record=observed,
            mismatches=mismatches,
            report_path=report_path,
        )

    def _evaluate(self, case: BenchmarkCase, record: ExecutionRecord) -> List[EvaluationCriterion]:
        return [
            self._criterion(
                name="decision-medium",
                weight=40,
                expected=case.expected_medium,
                actual=record.decision.medium,
            ),
            self._criterion(
                name="approval-gate",
                weight=30,
                expected=case.expected_approved,
                actual=record.decision.approved,
            ),
            self._criterion(
                name="final-status",
                weight=20,
                expected=case.expected_status,
                actual=record.run.status,
            ),
            EvaluationCriterion(
                name="report-artifact",
                weight=10,
                passed=(not case.require_report) or bool(record.report_path),
                detail=(
                    "report emitted"
                    if (not case.require_report) or bool(record.report_path)
                    else "report missing"
                ),
            ),
        ]

    def _criterion(self, name: str, weight: int, expected: Optional[object], actual: object) -> EvaluationCriterion:
        if expected is None:
            return EvaluationCriterion(name=name, weight=weight, passed=True, detail="not asserted")
        passed = expected == actual
        detail = f"expected {expected} got {actual}"
        return EvaluationCriterion(name=name, weight=weight, passed=passed, detail=detail)

    def _case_path(self, case_id: str, file_name: str) -> Path:
        if self.storage_dir is None:
            return Path(file_name)
        return self.storage_dir / case_id / file_name


def _write_report(path: str, content: str) -> None:
    from .reports import write_report

    write_report(path, content)


def _observability_ledger(path: str):
    from .observability import ObservabilityLedger

    return ObservabilityLedger(path)


def _scheduler_type():
    from .runtime import Scheduler

    return Scheduler()


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
        lines.extend(
            f"- {result.case_id}: score={result.score} passed={result.passed} replay={result.replay.matched}"
            for result in suite.results
        )
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

    return "\\n".join(lines) + "\\n"


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
    comparison_html = f'''
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
    '''
    mismatch_html = f'''
    <section class="surface">
      <h2>Replay Mismatches</h2>
      <p>Detailed mismatch list for the replay execution.</p>
      <ul>{''.join(f'<li>{escape(item)}</li>' for item in mismatches) or '<li>None</li>'}</ul>
    </section>
    '''
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
    record: ExecutionRecord,
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
        RunDetailResource(
            name="Markdown report",
            kind="report",
            path=report_path,
            meta=["execution report"],
            tone="report",
        ),
        RunDetailResource(
            name="Run detail page",
            kind="page",
            path=detail_path,
            meta=["task run detail"],
            tone="page",
        ),
        RunDetailResource(
            name="Replay page",
            kind="page",
            path=replay_path,
            meta=[f"matched={replay.matched}"],
            tone="page",
        ),
    ]

    overview_html = f'''
    <section class="surface">
      <h2>Overview</h2>
      <p>Benchmark case <strong>{escape(case_id)}</strong> executed task <strong>{escape(record.run.task_id)}</strong> with scheduler medium <strong>{escape(record.decision.medium)}</strong>.</p>
      <p class="meta">Replay matched={escape(str(replay.matched))} | mismatches={escape(str(len(replay.mismatches)))}</p>
    </section>
    '''
    acceptance_html = f'''
    <section class="surface">
      <h2>Acceptance Criteria</h2>
      <p>Scored checks used to grade the run detail and replay execution path.</p>
      <ul>
        {''.join(f'<li><strong>{escape(item.name)}</strong>: {escape(item.detail)} | weight={item.weight} | passed={item.passed}</li>' for item in criteria) or '<li>None</li>'}
      </ul>
    </section>
    '''
    replay_html = f'''
    <section class="surface">
      <h2>Replay</h2>
      <p>Replay status <strong>{escape('matched' if replay.matched else 'mismatch')}</strong> for baseline run <code>{escape(replay.replay_record.run_id)}</code>.</p>
      <ul>
        {''.join(f'<li>{escape(item)}</li>' for item in replay.mismatches) or '<li>None</li>'}
      </ul>
    </section>
    '''

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
""",
    _evaluation_module.__dict__,
)
_evaluation_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go"
globals()["evaluation"] = _evaluation_module

_observability_module = types.ModuleType(f"{__name__}.observability")
sys.modules[_observability_module.__name__] = _observability_module
exec(
    """
import hashlib
import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

from .audit_events import missing_required_fields
from .collaboration import CollaborationComment, DecisionNote
from .models import Task
from .repo_links import bind_run_commits
from .repo_plane import RunCommitLink


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
        resolved_id = comment_id or f"{self.run_id}-comment-{len([audit for audit in self.audits if audit.action == 'collaboration.comment']) + 1}"
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
        resolved_id = decision_id or f"{self.run_id}-decision-{len([audit for audit in self.audits if audit.action == 'collaboration.decision']) + 1}"
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
""",
    _observability_module.__dict__,
)
_observability_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/observability/recorder.go"
globals()["observability"] = _observability_module

from . import runtime as _legacy_runtime_surface

_WorkspaceBootstrapError = _workspace_bootstrap_module.WorkspaceBootstrapError
_bootstrap_workspace = _workspace_bootstrap_module.bootstrap_workspace
_cleanup_workspace = _workspace_bootstrap_module.cleanup_workspace
_cache_root_for_repo = _workspace_bootstrap_module.cache_root_for_repo
_repo_cache_key = _workspace_bootstrap_module.repo_cache_key


_WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE = "~/.cache/symphony/repos"


def _workspace_bootstrap_build_parser(
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


def _workspace_bootstrap_emit(payload: dict, as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


def _workspace_bootstrap_cli_main(
    argv: Sequence[str] | None = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: str | None = None,
    default_cache_base: str = _WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE,
    default_cache_key: str | None = None,
) -> int:
    parser = _workspace_bootstrap_build_parser(
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
            status = _bootstrap_workspace(**payload)
        else:
            status = _cleanup_workspace(**payload)
        _workspace_bootstrap_emit({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except _WorkspaceBootstrapError as exc:
        _workspace_bootstrap_emit({"status": "error", "workspace": str(workspace), "error": str(exc)}, args.json)
        return 1


def _build_workspace_validation_report(
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
        status = _bootstrap_workspace(
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
            status = _cleanup_workspace(
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


def _render_workspace_validation_markdown(report: dict[str, Any]) -> str:
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


def _write_workspace_validation_report(report: dict[str, Any], path: str | Path) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)

    if target.suffix.lower() == ".md":
        target.write_text(_render_workspace_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target


_workspace_bootstrap_cli_module = types.ModuleType(f"{__name__}.workspace_bootstrap_cli")
_workspace_bootstrap_cli_module.DEFAULT_CACHE_BASE = _WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE
_workspace_bootstrap_cli_module.build_parser = _workspace_bootstrap_build_parser
_workspace_bootstrap_cli_module.emit = _workspace_bootstrap_emit
_workspace_bootstrap_cli_module.main = _workspace_bootstrap_cli_main
_workspace_bootstrap_cli_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/cmd/bigclawctl/main.go"
sys.modules[_workspace_bootstrap_cli_module.__name__] = _workspace_bootstrap_cli_module
globals()["workspace_bootstrap_cli"] = _workspace_bootstrap_cli_module

_workspace_bootstrap_validation_module = types.ModuleType(f"{__name__}.workspace_bootstrap_validation")
_workspace_bootstrap_validation_module.build_validation_report = _build_workspace_validation_report
_workspace_bootstrap_validation_module.render_validation_markdown = _render_workspace_validation_markdown
_workspace_bootstrap_validation_module.write_validation_report = _write_workspace_validation_report
_workspace_bootstrap_validation_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/bootstrap/bootstrap.go"
sys.modules[_workspace_bootstrap_validation_module.__name__] = _workspace_bootstrap_validation_module
globals()["workspace_bootstrap_validation"] = _workspace_bootstrap_validation_module


_install_legacy_surface_module(
    "queue",
    ["DeadLetterEntry", "PersistentTaskQueue"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
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
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/orchestration.go",
)
_install_legacy_surface_module(
    "scheduler",
    ["ExecutionRecord", "Scheduler", "SchedulerDecision"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/scheduler/scheduler.go",
)
_install_legacy_surface_module(
    "workflow",
    ["AcceptanceDecision", "AcceptanceGate", "JournalEntry", "WorkflowEngine", "WorkflowRunResult", "WorkpadJournal"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
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

from .runtime import (
    AcceptanceDecision,
    AcceptanceGate,
    ClawWorkerRuntime,
    CrossDepartmentOrchestrator,
    DeadLetterEntry,
    DepartmentHandoff,
    ExecutionRecord,
    HandoffRequest,
    JournalEntry,
    OrchestrationPlan,
    OrchestrationPolicyDecision,
    PersistentTaskQueue,
    PremiumOrchestrationPolicy,
    RepoGovernanceEnforcer,
    RepoGovernancePolicy,
    RepoGovernanceResult,
    SandboxProfile,
    SandboxRouter,
    Scheduler,
    SchedulerDecision,
    ServerMonitoring,
    ToolCallResult,
    ToolPolicy,
    ToolRuntime,
    WorkerExecutionResult,
    WorkflowEngine,
    WorkflowRunResult,
    WorkpadJournal,
    create_server,
    render_orchestration_plan,
    run_server,
    warn_legacy_service_surface,
)
from .collaboration import (
    CollaborationComment,
    CollaborationThread,
    DecisionNote,
    build_collaboration_thread,
    build_collaboration_thread_from_audits,
)
from .governance import (
    FreezeException,
    GovernanceBacklogItem,
    ScopeFreezeAudit,
    ScopeFreezeBoard,
    ScopeFreezeGovernance,
    render_scope_freeze_report,
)
from .risk import RiskFactor, RiskScore, RiskScorer
from .audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    AuditEventSpec,
    get_audit_event_spec,
    missing_required_fields,
)
from .observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, RunCloseout, TaskRun
from .execution_contract import (
    AuditPolicy,
    build_operations_api_contract,
    ExecutionApiSpec,
    ExecutionContract,
    ExecutionContractAudit,
    ExecutionContractLibrary,
    ExecutionField,
    ExecutionModel,
    ExecutionPermission,
    ExecutionPermissionMatrix,
    ExecutionRole,
    MetricDefinition,
    PermissionCheckResult,
    render_execution_contract_report,
)
_reports_module = types.ModuleType(f"{__name__}.reports")
sys.modules[_reports_module.__name__] = _reports_module
exec(
r'''
from dataclasses import dataclass, field
from datetime import datetime, timezone
from difflib import SequenceMatcher
from html import escape
from pathlib import Path
from typing import List, Optional

from .audit_events import FLOW_HANDOFF_EVENT, MANUAL_TAKEOVER_EVENT
from .collaboration import (
    CollaborationThread,
    build_collaboration_thread_from_audits,
    render_collaboration_lines,
    render_collaboration_panel_html,
)
from .observability import RepoSyncAudit, TaskRun
from .orchestration import HandoffRequest, OrchestrationPlan, OrchestrationPolicyDecision
from .run_detail import (
    RunDetailEvent,
    RunDetailResource,
    RunDetailStat,
    RunDetailTab,
    render_resource_grid,
    render_run_detail_console,
    render_timeline_panel,
)


def _utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


@dataclass
class PilotMetric:
    name: str
    baseline: float
    current: float
    target: float
    unit: str = ""
    higher_is_better: bool = True

    @property
    def delta(self) -> float:
        return self.current - self.baseline

    @property
    def met_target(self) -> bool:
        if self.higher_is_better:
            return self.current >= self.target
        return self.current <= self.target


@dataclass
class PilotScorecard:
    issue_id: str
    customer: str
    period: str
    metrics: List[PilotMetric] = field(default_factory=list)
    monthly_benefit: float = 0.0
    monthly_cost: float = 0.0
    implementation_cost: float = 0.0
    benchmark_score: Optional[int] = None
    benchmark_passed: Optional[bool] = None

    @property
    def monthly_net_value(self) -> float:
        return self.monthly_benefit - self.monthly_cost

    @property
    def annualized_roi(self) -> float:
        total_cost = self.implementation_cost + (self.monthly_cost * 12)
        if total_cost <= 0:
            return 0.0
        annual_gain = (self.monthly_benefit * 12) - total_cost
        return (annual_gain / total_cost) * 100

    @property
    def payback_months(self) -> Optional[float]:
        if self.monthly_net_value <= 0:
            return None
        if self.implementation_cost <= 0:
            return 0.0
        return round(self.implementation_cost / self.monthly_net_value, 1)

    @property
    def metrics_met(self) -> int:
        return sum(1 for metric in self.metrics if metric.met_target)

    @property
    def recommendation(self) -> str:
        benchmark_ok = self.benchmark_passed is not False
        if self.metrics and self.metrics_met == len(self.metrics) and self.annualized_roi > 0 and benchmark_ok:
            return "go"
        if self.annualized_roi > 0 or self.metrics_met:
            return "iterate"
        return "hold"


@dataclass
class PilotPortfolio:
    name: str
    period: str
    scorecards: List[PilotScorecard] = field(default_factory=list)

    @property
    def total_monthly_net_value(self) -> float:
        return sum(scorecard.monthly_net_value for scorecard in self.scorecards)

    @property
    def average_roi(self) -> float:
        if not self.scorecards:
            return 0.0
        return round(
            sum(scorecard.annualized_roi for scorecard in self.scorecards) / len(self.scorecards),
            1,
        )

    @property
    def recommendation_counts(self) -> dict[str, int]:
        counts = {"go": 0, "iterate": 0, "hold": 0}
        for scorecard in self.scorecards:
            counts[scorecard.recommendation] += 1
        return counts

    @property
    def recommendation(self) -> str:
        counts = self.recommendation_counts
        if self.scorecards and counts["go"] == len(self.scorecards):
            return "scale"
        if counts["go"] or counts["iterate"]:
            return "continue"
        return "stop"


@dataclass
class IssueClosureDecision:
    issue_id: str
    allowed: bool
    reason: str
    report_path: str = ""


@dataclass
class DocumentationArtifact:
    name: str
    path: str

    @property
    def available(self) -> bool:
        return validation_report_exists(self.path)


@dataclass
class LaunchChecklistItem:
    name: str
    evidence: List[str] = field(default_factory=list)


@dataclass
class LaunchChecklist:
    issue_id: str
    documentation: List[DocumentationArtifact] = field(default_factory=list)
    items: List[LaunchChecklistItem] = field(default_factory=list)

    @property
    def documentation_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.documentation}

    @property
    def completed_items(self) -> int:
        return sum(1 for item in self.items if self.item_completed(item))

    @property
    def missing_documentation(self) -> List[str]:
        return [artifact.name for artifact in self.documentation if not artifact.available]

    @property
    def ready(self) -> bool:
        if self.missing_documentation:
            return False
        return all(self.item_completed(item) for item in self.items)

    def item_completed(self, item: LaunchChecklistItem) -> bool:
        status = self.documentation_status
        if not item.evidence:
            return True
        return all(status.get(name, False) for name in item.evidence)


@dataclass
class FinalDeliveryChecklist:
    issue_id: str
    required_outputs: List[DocumentationArtifact] = field(default_factory=list)
    recommended_documentation: List[DocumentationArtifact] = field(default_factory=list)

    @property
    def required_output_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.required_outputs}

    @property
    def recommended_documentation_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.recommended_documentation}

    @property
    def generated_required_outputs(self) -> int:
        return sum(1 for artifact in self.required_outputs if artifact.available)

    @property
    def generated_recommended_documentation(self) -> int:
        return sum(1 for artifact in self.recommended_documentation if artifact.available)

    @property
    def missing_required_outputs(self) -> List[str]:
        return [artifact.name for artifact in self.required_outputs if not artifact.available]

    @property
    def missing_recommended_documentation(self) -> List[str]:
        return [artifact.name for artifact in self.recommended_documentation if not artifact.available]

    @property
    def ready(self) -> bool:
        return not self.missing_required_outputs


@dataclass
class NarrativeSection:
    heading: str
    body: str
    evidence: List[str] = field(default_factory=list)
    callouts: List[str] = field(default_factory=list)

    @property
    def ready(self) -> bool:
        return bool(self.heading.strip()) and bool(self.body.strip())


@dataclass
class ReportStudio:
    name: str
    issue_id: str
    audience: str
    period: str
    summary: str
    sections: List[NarrativeSection] = field(default_factory=list)
    action_items: List[str] = field(default_factory=list)
    source_reports: List[str] = field(default_factory=list)

    @property
    def ready(self) -> bool:
        return bool(self.summary.strip()) and bool(self.sections) and all(section.ready for section in self.sections)

    @property
    def recommendation(self) -> str:
        return "publish" if self.ready else "draft"

    @property
    def export_slug(self) -> str:
        return _slugify(self.name) or "report-studio"


@dataclass
class ReportStudioArtifacts:
    root_dir: str
    markdown_path: str
    html_path: str
    text_path: str


@dataclass
class TriageFinding:
    run_id: str
    task_id: str
    source: str
    severity: str
    owner: str
    status: str
    reason: str
    next_action: str
    actions: List["ConsoleAction"] = field(default_factory=list)


@dataclass
class TriageSimilarityEvidence:
    related_run_id: str
    related_task_id: str
    score: float
    reason: str


@dataclass
class TriageSuggestion:
    label: str
    action: str
    owner: str
    confidence: float
    evidence: List[TriageSimilarityEvidence] = field(default_factory=list)
    feedback_status: str = "pending"


@dataclass
class TriageInboxItem:
    run_id: str
    task_id: str
    source: str
    status: str
    severity: str
    owner: str
    summary: str
    submitted_at: str
    suggestions: List[TriageSuggestion] = field(default_factory=list)


@dataclass
class TriageFeedbackRecord:
    run_id: str
    action: str
    decision: str
    actor: str
    notes: str = ""
    timestamp: str = field(default_factory=_utc_now_iso)


@dataclass
class AutoTriageCenter:
    name: str
    period: str
    findings: List[TriageFinding] = field(default_factory=list)
    inbox: List[TriageInboxItem] = field(default_factory=list)
    feedback: List[TriageFeedbackRecord] = field(default_factory=list)

    @property
    def flagged_runs(self) -> int:
        return len(self.findings)

    @property
    def severity_counts(self) -> dict[str, int]:
        counts = {"critical": 0, "high": 0, "medium": 0}
        for finding in self.findings:
            counts[finding.severity] += 1
        return counts

    @property
    def owner_counts(self) -> dict[str, int]:
        counts = {"security": 0, "engineering": 0, "operations": 0}
        for finding in self.findings:
            counts[finding.owner] = counts.get(finding.owner, 0) + 1
        return counts

    @property
    def recommendation(self) -> str:
        counts = self.severity_counts
        if counts["critical"]:
            return "immediate-attention"
        if self.feedback_counts["rejected"] > self.feedback_counts["accepted"]:
            return "retune-suggestions"
        if counts["high"]:
            return "review-queue"
        return "monitor"

    @property
    def inbox_size(self) -> int:
        return len(self.inbox)

    @property
    def feedback_counts(self) -> dict[str, int]:
        counts = {"accepted": 0, "rejected": 0, "pending": 0}
        for record in self.feedback:
            counts[record.decision] = counts.get(record.decision, 0) + 1
        pending = sum(
            1
            for item in self.inbox
            for suggestion in item.suggestions
            if suggestion.feedback_status == "pending"
        )
        counts["pending"] = pending
        return counts


@dataclass
class TakeoverRequest:
    run_id: str
    task_id: str
    source: str
    target_team: str
    status: str
    reason: str
    required_approvals: List[str] = field(default_factory=list)
    actions: List["ConsoleAction"] = field(default_factory=list)


@dataclass
class TakeoverQueue:
    name: str
    period: str
    requests: List[TakeoverRequest] = field(default_factory=list)

    @property
    def pending_requests(self) -> int:
        return len(self.requests)

    @property
    def team_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for request in self.requests:
            counts[request.target_team] = counts.get(request.target_team, 0) + 1
        return counts

    @property
    def approval_count(self) -> int:
        return sum(len(request.required_approvals) for request in self.requests)

    @property
    def recommendation(self) -> str:
        if any(request.target_team == "security" for request in self.requests):
            return "expedite-security-review"
        if self.requests:
            return "staff-takeover-queue"
        return "monitor"


@dataclass
class SharedViewFilter:
    label: str
    value: str


@dataclass
class SharedViewContext:
    filters: List[SharedViewFilter] = field(default_factory=list)
    result_count: Optional[int] = None
    loading: bool = False
    errors: List[str] = field(default_factory=list)
    partial_data: List[str] = field(default_factory=list)
    empty_message: str = "No records match the current filters."
    last_updated: str = ""
    collaboration: Optional[CollaborationThread] = None

    @property
    def state(self) -> str:
        if self.loading:
            return "loading"
        if self.errors and not self.result_count:
            return "error"
        if self.result_count == 0 and not self.partial_data:
            return "empty"
        if self.errors or self.partial_data:
            return "partial-data"
        return "ready"

    @property
    def summary(self) -> str:
        if self.state == "loading":
            return "Loading data for the current filters."
        if self.state == "error":
            return "Unable to load data for the current filters."
        if self.state == "empty":
            return self.empty_message
        if self.state == "partial-data":
            return "Showing partial data while one or more sources are unavailable."
        return "Data is current for the selected filters."



@dataclass
class OrchestrationCanvas:
    task_id: str
    run_id: str
    collaboration_mode: str
    departments: List[str] = field(default_factory=list)
    required_approvals: List[str] = field(default_factory=list)
    tier: str = "standard"
    upgrade_required: bool = False
    blocked_departments: List[str] = field(default_factory=list)
    handoff_team: str = "none"
    handoff_status: str = "none"
    handoff_reason: str = ""
    active_tools: List[str] = field(default_factory=list)
    entitlement_status: str = "included"
    billing_model: str = "standard-included"
    estimated_cost_usd: float = 0.0
    included_usage_units: int = 0
    overage_usage_units: int = 0
    overage_cost_usd: float = 0.0
    actions: List["ConsoleAction"] = field(default_factory=list)
    collaboration: Optional[CollaborationThread] = None

    @property
    def recommendation(self) -> str:
        if self.collaboration is not None and self.collaboration.open_comment_count:
            return "resolve-flow-comments"
        if self.handoff_team == "security":
            return "review-security-takeover"
        if self.upgrade_required:
            return "resolve-entitlement-gap"
        if self.overage_cost_usd > 0:
            return "review-billing-overage"
        if len(self.departments) > 1:
            return "continue-cross-team-execution"
        return "monitor"



@dataclass
class OrchestrationPortfolio:
    name: str
    period: str
    canvases: List[OrchestrationCanvas] = field(default_factory=list)
    takeover_queue: Optional[TakeoverQueue] = None

    @property
    def total_runs(self) -> int:
        return len(self.canvases)

    @property
    def collaboration_modes(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.collaboration_mode] = counts.get(canvas.collaboration_mode, 0) + 1
        return counts

    @property
    def tier_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.tier] = counts.get(canvas.tier, 0) + 1
        return counts

    @property
    def upgrade_required_count(self) -> int:
        return sum(1 for canvas in self.canvases if canvas.upgrade_required)

    @property
    def active_handoffs(self) -> int:
        return sum(1 for canvas in self.canvases if canvas.handoff_team != "none")

    @property
    def entitlement_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.entitlement_status] = counts.get(canvas.entitlement_status, 0) + 1
        return counts

    @property
    def billing_model_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.billing_model] = counts.get(canvas.billing_model, 0) + 1
        return counts

    @property
    def total_estimated_cost_usd(self) -> float:
        return round(sum(canvas.estimated_cost_usd for canvas in self.canvases), 2)

    @property
    def total_overage_cost_usd(self) -> float:
        return round(sum(canvas.overage_cost_usd for canvas in self.canvases), 2)

    @property
    def recommendation(self) -> str:
        if self.takeover_queue is not None and self.takeover_queue.recommendation == "expedite-security-review":
            return "stabilize-security-takeovers"
        if self.upgrade_required_count:
            return "close-entitlement-gaps"
        if self.active_handoffs:
            return "manage-cross-team-flow"
        return "monitor"

@dataclass(frozen=True)
class ConsoleAction:
    action_id: str
    label: str
    target: str
    enabled: bool = True
    reason: str = ""

    @property
    def state(self) -> str:
        return "enabled" if self.enabled else "disabled"


@dataclass
class BillingRunCharge:
    run_id: str
    task_id: str
    billing_model: str
    entitlement_status: str
    estimated_cost_usd: float
    included_usage_units: int = 0
    overage_usage_units: int = 0
    overage_cost_usd: float = 0.0
    blocked_capabilities: List[str] = field(default_factory=list)
    handoff_team: str = "none"
    recommendation: str = "monitor"


@dataclass
class BillingEntitlementsPage:
    workspace_name: str
    plan_name: str
    billing_period: str
    charges: List[BillingRunCharge] = field(default_factory=list)

    @property
    def run_count(self) -> int:
        return len(self.charges)

    @property
    def total_estimated_cost_usd(self) -> float:
        return round(sum(charge.estimated_cost_usd for charge in self.charges), 2)

    @property
    def total_included_usage_units(self) -> int:
        return sum(charge.included_usage_units for charge in self.charges)

    @property
    def total_overage_usage_units(self) -> int:
        return sum(charge.overage_usage_units for charge in self.charges)

    @property
    def total_overage_cost_usd(self) -> float:
        return round(sum(charge.overage_cost_usd for charge in self.charges), 2)

    @property
    def upgrade_required_count(self) -> int:
        return sum(1 for charge in self.charges if charge.entitlement_status == "upgrade-required")

    @property
    def billing_model_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for charge in self.charges:
            counts[charge.billing_model] = counts.get(charge.billing_model, 0) + 1
        return counts

    @property
    def entitlement_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for charge in self.charges:
            counts[charge.entitlement_status] = counts.get(charge.entitlement_status, 0) + 1
        return counts

    @property
    def blocked_capabilities(self) -> List[str]:
        capabilities: List[str] = []
        for charge in self.charges:
            for capability in charge.blocked_capabilities:
                if capability not in capabilities:
                    capabilities.append(capability)
        return capabilities

    @property
    def recommendation(self) -> str:
        if self.upgrade_required_count:
            return "resolve-plan-gaps"
        if self.total_overage_cost_usd > 0:
            return "optimize-billed-usage"
        if any(charge.handoff_team != "none" for charge in self.charges):
            return "monitor-shared-capacity"
        return "healthy"


def render_issue_validation_report(issue_id: str, version: str, environment: str, summary: str) -> str:
    return f"""# Issue Validation Report\n\n- Issue ID: {issue_id}\n- 版本号: {version}\n- 测试环境: {environment}\n- 生成时间: {_utc_now_iso()}\n\n## 结论\n\n{summary}\n"""


def render_report_studio_report(studio: ReportStudio) -> str:
    lines = [
        "# Report Studio",
        "",
        f"- Name: {studio.name}",
        f"- Issue ID: {studio.issue_id}",
        f"- Audience: {studio.audience}",
        f"- Period: {studio.period}",
        f"- Sections: {len(studio.sections)}",
        f"- Recommendation: {studio.recommendation}",
        "",
        "## Narrative Summary",
        "",
        studio.summary or "No summary drafted.",
        "",
        "## Sections",
        "",
    ]

    if studio.sections:
        for section in studio.sections:
            lines.append(f"### {section.heading}")
            lines.append("")
            lines.append(section.body or "No narrative drafted.")
            lines.append("")
            lines.append("- Evidence: " + (", ".join(section.evidence) if section.evidence else "None"))
            lines.append("- Callouts: " + (", ".join(section.callouts) if section.callouts else "None"))
            lines.append("")
    else:
        lines.append("- None")
        lines.append("")

    lines.append("## Action Items")
    lines.append("")
    if studio.action_items:
        lines.extend(f"- {item}" for item in studio.action_items)
    else:
        lines.append("- None")
    lines.append("")

    lines.append("## Sources")
    lines.append("")
    if studio.source_reports:
        lines.extend(f"- {path}" for path in studio.source_reports)
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_report_studio_plain_text(studio: ReportStudio) -> str:
    lines = [
        f"{studio.name} ({studio.issue_id})",
        f"Audience: {studio.audience}",
        f"Period: {studio.period}",
        f"Recommendation: {studio.recommendation}",
        "",
        studio.summary or "No summary drafted.",
        "",
    ]
    for section in studio.sections:
        lines.append(section.heading.upper())
        lines.append(section.body or "No narrative drafted.")
        if section.callouts:
            lines.append("Callouts: " + "; ".join(section.callouts))
        if section.evidence:
            lines.append("Evidence: " + "; ".join(section.evidence))
        lines.append("")

    if studio.action_items:
        lines.append("Action Items:")
        lines.extend(f"- {item}" for item in studio.action_items)
        lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def render_report_studio_html(studio: ReportStudio) -> str:
    section_html = "".join(
        f"""
        <section class="section">
          <h2>{escape(section.heading)}</h2>
          <p>{escape(section.body)}</p>
          <p class="meta">Evidence: {escape(', '.join(section.evidence) if section.evidence else 'None')}</p>
          <p class="meta">Callouts: {escape(', '.join(section.callouts) if section.callouts else 'None')}</p>
        </section>
        """
        for section in studio.sections
    )
    action_html = "".join(f"<li>{escape(item)}</li>" for item in studio.action_items) or "<li>None</li>"
    source_html = "".join(f"<li>{escape(path)}</li>" for path in studio.source_reports) or "<li>None</li>"
    return f"""<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{escape(studio.name)}</title>
    <style>
      body {{ font-family: Georgia, 'Times New Roman', serif; margin: 40px auto; max-width: 840px; color: #1f2933; line-height: 1.6; }}
      h1, h2 {{ font-family: 'Avenir Next', 'Segoe UI', sans-serif; }}
      .meta {{ color: #52606d; font-size: 0.95rem; }}
      .summary {{ padding: 16px 20px; background: #f7f3e8; border-left: 4px solid #c58b32; }}
      .section {{ margin-top: 28px; }}
    </style>
  </head>
  <body>
    <header>
      <p class="meta">{escape(studio.issue_id)} · {escape(studio.audience)} · {escape(studio.period)}</p>
      <h1>{escape(studio.name)}</h1>
      <p class="meta">Recommendation: {escape(studio.recommendation)}</p>
    </header>
    <section class="summary">
      <h2>Narrative Summary</h2>
      <p>{escape(studio.summary or 'No summary drafted.')}</p>
    </section>
    {section_html or '<section class="section"><p>No sections drafted.</p></section>'}
    <section class="section">
      <h2>Action Items</h2>
      <ul>{action_html}</ul>
    </section>
    <section class="section">
      <h2>Sources</h2>
      <ul>{source_html}</ul>
    </section>
  </body>
</html>
"""


def build_launch_checklist(
    issue_id: str,
    documentation: List[DocumentationArtifact],
    items: List[LaunchChecklistItem],
) -> LaunchChecklist:
    return LaunchChecklist(issue_id=issue_id, documentation=documentation, items=items)


def build_final_delivery_checklist(
    issue_id: str,
    required_outputs: List[DocumentationArtifact],
    recommended_documentation: List[DocumentationArtifact],
) -> FinalDeliveryChecklist:
    return FinalDeliveryChecklist(
        issue_id=issue_id,
        required_outputs=required_outputs,
        recommended_documentation=recommended_documentation,
    )


def render_launch_checklist_report(checklist: LaunchChecklist) -> str:
    lines = [
        "# Launch Checklist",
        "",
        f"- Issue ID: {checklist.issue_id}",
        f"- Linked Documentation: {len(checklist.documentation)}",
        f"- Completed Items: {checklist.completed_items}/{len(checklist.items)}",
        f"- Ready: {checklist.ready}",
        "",
        "## Documentation",
        "",
    ]

    if checklist.documentation:
        for artifact in checklist.documentation:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Checklist", ""])
    if checklist.items:
        for item in checklist.items:
            evidence = ", ".join(item.evidence) if item.evidence else "none"
            lines.append(
                f"- {item.name}: completed={checklist.item_completed(item)} evidence={evidence}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_final_delivery_checklist_report(checklist: FinalDeliveryChecklist) -> str:
    lines = [
        "# Final Delivery Checklist",
        "",
        f"- Issue ID: {checklist.issue_id}",
        f"- Required Outputs Generated: {checklist.generated_required_outputs}/{len(checklist.required_outputs)}",
        f"- Recommended Docs Generated: {checklist.generated_recommended_documentation}/{len(checklist.recommended_documentation)}",
        f"- Ready: {checklist.ready}",
        "",
        "## Required Outputs",
        "",
    ]

    if checklist.required_outputs:
        for artifact in checklist.required_outputs:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Recommended Documentation", ""])
    if checklist.recommended_documentation:
        for artifact in checklist.recommended_documentation:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_pilot_scorecard(scorecard: PilotScorecard) -> str:
    lines = [
        "# Pilot Scorecard",
        "",
        f"- Issue ID: {scorecard.issue_id}",
        f"- Customer: {scorecard.customer}",
        f"- Period: {scorecard.period}",
        f"- Recommendation: {scorecard.recommendation}",
        f"- Metrics Met: {scorecard.metrics_met}/{len(scorecard.metrics)}",
        f"- Monthly Net Value: {scorecard.monthly_net_value:.2f}",
        f"- Annualized ROI: {scorecard.annualized_roi:.1f}%",
    ]

    if scorecard.payback_months is None:
        lines.append("- Payback Months: n/a")
    else:
        lines.append(f"- Payback Months: {scorecard.payback_months:.1f}")

    if scorecard.benchmark_score is not None:
        lines.append(f"- Benchmark Score: {scorecard.benchmark_score}")
    if scorecard.benchmark_passed is not None:
        lines.append(f"- Benchmark Passed: {scorecard.benchmark_passed}")

    lines.extend(["", "## KPI Progress", ""])
    if scorecard.metrics:
        for metric in scorecard.metrics:
            comparator = ">=" if metric.higher_is_better else "<="
            unit_suffix = f" {metric.unit}" if metric.unit else ""
            lines.append(
                f"- {metric.name}: baseline={metric.baseline}{unit_suffix} current={metric.current}{unit_suffix} "
                f"target{comparator}{metric.target}{unit_suffix} delta={metric.delta:+.2f}{unit_suffix} met={metric.met_target}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_pilot_portfolio_report(portfolio: PilotPortfolio) -> str:
    counts = portfolio.recommendation_counts
    lines = [
        "# Pilot Portfolio Report",
        "",
        f"- Portfolio: {portfolio.name}",
        f"- Period: {portfolio.period}",
        f"- Scorecards: {len(portfolio.scorecards)}",
        f"- Recommendation: {portfolio.recommendation}",
        f"- Total Monthly Net Value: {portfolio.total_monthly_net_value:.2f}",
        f"- Average ROI: {portfolio.average_roi:.1f}%",
        f"- Recommendation Mix: go={counts['go']} iterate={counts['iterate']} hold={counts['hold']}",
        "",
        "## Customers",
        "",
    ]

    if portfolio.scorecards:
        for scorecard in portfolio.scorecards:
            lines.append(
                f"- {scorecard.customer}: recommendation={scorecard.recommendation} roi={scorecard.annualized_roi:.1f}% "
                f"monthly-net={scorecard.monthly_net_value:.2f} benchmark={scorecard.benchmark_score if scorecard.benchmark_score is not None else 'n/a'}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def write_report(path: str, content: str) -> None:
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)


def write_report_studio_bundle(root_dir: str, studio: ReportStudio) -> ReportStudioArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    markdown_path = str(base / f"{studio.export_slug}.md")
    html_path = str(base / f"{studio.export_slug}.html")
    text_path = str(base / f"{studio.export_slug}.txt")
    write_report(markdown_path, render_report_studio_report(studio))
    write_report(html_path, render_report_studio_html(studio))
    write_report(text_path, render_report_studio_plain_text(studio))
    return ReportStudioArtifacts(
        root_dir=str(base),
        markdown_path=markdown_path,
        html_path=html_path,
        text_path=text_path,
    )


def validation_report_exists(report_path: Optional[str]) -> bool:
    if not report_path:
        return False

    path = Path(report_path)
    if not path.exists() or not path.is_file():
        return False

    return bool(path.read_text().strip())


def evaluate_issue_closure(
    issue_id: str,
    report_path: Optional[str],
    validation_passed: bool = True,
    launch_checklist: Optional[LaunchChecklist] = None,
    final_delivery_checklist: Optional[FinalDeliveryChecklist] = None,
) -> IssueClosureDecision:
    resolved_path = str(Path(report_path)) if report_path else ""

    if not validation_report_exists(report_path):
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="validation report required before closing issue",
            report_path=resolved_path,
        )

    if not validation_passed:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="validation failed; issue must remain open",
            report_path=resolved_path,
        )

    if final_delivery_checklist is not None and not final_delivery_checklist.ready:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="final delivery checklist incomplete; required outputs missing",
            report_path=resolved_path,
        )

    if launch_checklist is not None and not launch_checklist.ready:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="launch checklist incomplete; linked documentation missing or empty",
            report_path=resolved_path,
        )

    if final_delivery_checklist is not None:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=True,
            reason="validation report and final delivery checklist requirements satisfied; issue can be closed",
            report_path=resolved_path,
        )

    return IssueClosureDecision(
        issue_id=issue_id,
        allowed=True,
        reason="validation report and launch checklist requirements satisfied; issue can be closed",
        report_path=resolved_path,
    )

def build_console_actions(
    target: str,
    *,
    allow_retry: bool = True,
    retry_reason: str = "",
    allow_pause: bool = True,
    pause_reason: str = "",
    allow_reassign: bool = True,
    reassign_reason: str = "",
    allow_escalate: bool = True,
    escalate_reason: str = "",
) -> List[ConsoleAction]:
    return [
        ConsoleAction("drill-down", "Drill Down", target),
        ConsoleAction("export", "Export", target),
        ConsoleAction("add-note", "Add Note", target),
        ConsoleAction("escalate", "Escalate", target, enabled=allow_escalate, reason=escalate_reason),
        ConsoleAction("retry", "Retry", target, enabled=allow_retry, reason=retry_reason),
        ConsoleAction("pause", "Pause", target, enabled=allow_pause, reason=pause_reason),
        ConsoleAction("reassign", "Reassign", target, enabled=allow_reassign, reason=reassign_reason),
        ConsoleAction("audit", "Audit Trail", target),
    ]


def render_console_actions(actions: List[ConsoleAction]) -> str:
    if not actions:
        return "none"

    rendered: List[str] = []
    for action in actions:
        detail = f"{action.label} [{action.action_id}] state={action.state} target={action.target}"
        if action.reason:
            detail += f" reason={action.reason}"
        rendered.append(detail)
    return "; ".join(rendered)


def _default_canvas_actions(canvas: OrchestrationCanvas) -> List[ConsoleAction]:
    return build_console_actions(
        canvas.run_id,
        allow_retry=canvas.handoff_status != "pending",
        retry_reason="" if canvas.handoff_status != "pending" else "pending handoff must be resolved before retry",
        allow_pause=canvas.handoff_status != "completed",
        pause_reason="" if canvas.handoff_status != "completed" else "completed handoff runs cannot be paused",
        allow_reassign=canvas.handoff_team != "none",
        reassign_reason="" if canvas.handoff_team != "none" else "reassign is available after a handoff exists",
        allow_escalate=canvas.upgrade_required,
        escalate_reason="" if canvas.upgrade_required else "escalate when policy requires an entitlement or approval upgrade",
    )


def build_auto_triage_center(
    runs: List[TaskRun],
    name: str = "Auto Triage Center",
    period: str = "current",
    feedback: Optional[List[TriageFeedbackRecord]] = None,
) -> AutoTriageCenter:
    findings: List[TriageFinding] = []
    inbox: List[TriageInboxItem] = []
    feedback = feedback or []
    for run in runs:
        if not _run_requires_triage(run):
            continue

        severity = _triage_severity(run)
        owner = _triage_owner(run)
        reason = _triage_reason(run)
        next_action = _triage_next_action(severity, owner)
        suggestions = _build_triage_suggestions(run, runs, severity, owner, feedback)
        findings.append(
            TriageFinding(
                run_id=run.run_id,
                task_id=run.task_id,
                source=run.source,
                severity=severity,
                owner=owner,
                status=run.status,
                reason=reason,
                next_action=next_action,
                actions=build_console_actions(
                    run.run_id,
                    allow_retry=severity == "critical" and owner != "security",
                    retry_reason="" if severity == "critical" and owner != "security" else "retry available after owner review",
                    allow_pause=run.status not in {"failed", "completed", "approved"},
                    pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
                    allow_reassign=owner != "security",
                    reassign_reason="" if owner != "security" else "security-owned findings stay with the security queue",
                ),
            )
        )
        inbox.append(
            TriageInboxItem(
                run_id=run.run_id,
                task_id=run.task_id,
                source=run.source,
                status=run.status,
                severity=severity,
                owner=owner,
                summary=reason,
                submitted_at=run.ended_at or run.started_at,
                suggestions=suggestions,
            )
        )

    severity_rank = {"critical": 0, "high": 1, "medium": 2}
    findings.sort(key=lambda finding: (severity_rank[finding.severity], finding.owner, finding.run_id))
    inbox.sort(key=lambda item: (severity_rank[item.severity], item.owner, item.run_id))
    return AutoTriageCenter(name=name, period=period, findings=findings, inbox=inbox, feedback=feedback)


def render_shared_view_context(view: Optional[SharedViewContext]) -> List[str]:
    if view is None:
        return []

    lines = [
        "## View State",
        "",
        f"- State: {view.state}",
        f"- Summary: {view.summary}",
    ]
    if view.result_count is not None:
        lines.append(f"- Result Count: {view.result_count}")
    if view.last_updated:
        lines.append(f"- Last Updated: {view.last_updated}")

    lines.extend(["", "## Filters", ""])
    if view.filters:
        lines.extend(f"- {item.label}: {item.value}" for item in view.filters)
    else:
        lines.append("- None")

    if view.errors:
        lines.extend(["", "## Errors", ""])
        lines.extend(f"- {message}" for message in view.errors)

    if view.partial_data:
        lines.extend(["", "## Partial Data", ""])
        lines.extend(f"- {message}" for message in view.partial_data)

    lines.extend(render_collaboration_lines(view.collaboration))
    lines.append("")
    return lines


def render_auto_triage_center_report(
    center: AutoTriageCenter,
    total_runs: Optional[int] = None,
    view: Optional[SharedViewContext] = None,
) -> str:
    severity = center.severity_counts
    owners = center.owner_counts
    feedback = center.feedback_counts
    lines = [
        "# Auto Triage Center",
        "",
        f"- Center: {center.name}",
        f"- Period: {center.period}",
        f"- Flagged Runs: {center.flagged_runs}",
        f"- Inbox Size: {center.inbox_size}",
        f"- Total Runs: {total_runs if total_runs is not None else center.flagged_runs}",
        f"- Recommendation: {center.recommendation}",
        f"- Severity Mix: critical={severity['critical']} high={severity['high']} medium={severity['medium']}",
        f"- Owner Mix: security={owners['security']} engineering={owners['engineering']} operations={owners['operations']}",
        f"- Feedback Loop: accepted={feedback['accepted']} rejected={feedback['rejected']} pending={feedback['pending']}",
        "",
        "## Queue",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if center.findings:
        for finding in center.findings:
            lines.append(
                f"- {finding.run_id}: severity={finding.severity} owner={finding.owner} status={finding.status} "
                f"task={finding.task_id} reason={finding.reason} next={finding.next_action} actions={render_console_actions(finding.actions)}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Inbox", ""])
    if center.inbox:
        for item in center.inbox:
            suggestion_summary = "; ".join(
                f"{suggestion.action}({suggestion.feedback_status}, confidence={suggestion.confidence:.2f})"
                for suggestion in item.suggestions
            ) or "none"
            evidence_summary = ", ".join(
                f"{e.related_run_id}:{e.score:.2f}" for suggestion in item.suggestions for e in suggestion.evidence
            ) or "none"
            lines.append(
                f"- {item.run_id}: severity={item.severity} owner={item.owner} status={item.status} "
                f"summary={item.summary} suggestions={suggestion_summary} similar={evidence_summary}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def build_orchestration_portfolio(
    canvases: List[OrchestrationCanvas],
    name: str = "Cross-Department Orchestration",
    period: str = "current",
    takeover_queue: Optional[TakeoverQueue] = None,
) -> OrchestrationPortfolio:
    normalized_canvases = [
        canvas
        if canvas.actions
        else OrchestrationCanvas(
            task_id=canvas.task_id,
            run_id=canvas.run_id,
            collaboration_mode=canvas.collaboration_mode,
            departments=canvas.departments,
            required_approvals=canvas.required_approvals,
            tier=canvas.tier,
            upgrade_required=canvas.upgrade_required,
            blocked_departments=canvas.blocked_departments,
            handoff_team=canvas.handoff_team,
            handoff_status=canvas.handoff_status,
            handoff_reason=canvas.handoff_reason,
            active_tools=canvas.active_tools,
            entitlement_status=canvas.entitlement_status,
            billing_model=canvas.billing_model,
            estimated_cost_usd=canvas.estimated_cost_usd,
            included_usage_units=canvas.included_usage_units,
            overage_usage_units=canvas.overage_usage_units,
            overage_cost_usd=canvas.overage_cost_usd,
            actions=_default_canvas_actions(canvas),
        )
        for canvas in canvases
    ]
    return OrchestrationPortfolio(
        name=name,
        period=period,
        canvases=sorted(normalized_canvases, key=lambda canvas: canvas.run_id),
        takeover_queue=takeover_queue,
    )


def build_billing_entitlements_page(
    portfolio: OrchestrationPortfolio,
    *,
    workspace_name: str = "BigClaw Cloud",
    plan_name: str = "Standard",
    billing_period: Optional[str] = None,
) -> BillingEntitlementsPage:
    return BillingEntitlementsPage(
        workspace_name=workspace_name,
        plan_name=plan_name,
        billing_period=billing_period or portfolio.period,
        charges=[
            BillingRunCharge(
                run_id=canvas.run_id,
                task_id=canvas.task_id,
                billing_model=canvas.billing_model,
                entitlement_status=canvas.entitlement_status,
                estimated_cost_usd=canvas.estimated_cost_usd,
                included_usage_units=canvas.included_usage_units,
                overage_usage_units=canvas.overage_usage_units,
                overage_cost_usd=canvas.overage_cost_usd,
                blocked_capabilities=list(canvas.blocked_departments),
                handoff_team=canvas.handoff_team,
                recommendation=canvas.recommendation,
            )
            for canvas in portfolio.canvases
        ],
    )


def render_orchestration_portfolio_report(
    portfolio: OrchestrationPortfolio,
    view: Optional[SharedViewContext] = None,
) -> str:
    collaboration = " ".join(
        f"{mode}={count}" for mode, count in sorted(portfolio.collaboration_modes.items())
    ) or "none"
    tiers = " ".join(
        f"{tier}={count}" for tier, count in sorted(portfolio.tier_counts.items())
    ) or "none"
    entitlements = " ".join(
        f"{status}={count}" for status, count in sorted(portfolio.entitlement_counts.items())
    ) or "none"
    billing_models = " ".join(
        f"{model}={count}" for model, count in sorted(portfolio.billing_model_counts.items())
    ) or "none"
    takeover_summary = (
        f"pending={portfolio.takeover_queue.pending_requests} recommendation={portfolio.takeover_queue.recommendation}"
        if portfolio.takeover_queue is not None
        else "none"
    )
    lines = [
        "# Orchestration Portfolio Report",
        "",
        f"- Portfolio: {portfolio.name}",
        f"- Period: {portfolio.period}",
        f"- Total Runs: {portfolio.total_runs}",
        f"- Recommendation: {portfolio.recommendation}",
        f"- Collaboration Mix: {collaboration}",
        f"- Tier Mix: {tiers}",
        f"- Entitlement Mix: {entitlements}",
        f"- Billing Models: {billing_models}",
        f"- Upgrade Required Count: {portfolio.upgrade_required_count}",
        f"- Estimated Cost (USD): {portfolio.total_estimated_cost_usd:.2f}",
        f"- Overage Cost (USD): {portfolio.total_overage_cost_usd:.2f}",
        f"- Active Handoffs: {portfolio.active_handoffs}",
        f"- Takeover Queue: {takeover_summary}",
        "",
        "## Runs",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if portfolio.canvases:
        for canvas in portfolio.canvases:
            collaboration_summary = (
                f"comments={len(canvas.collaboration.comments)} decisions={len(canvas.collaboration.decisions)}"
                if canvas.collaboration is not None
                else "comments=0 decisions=0"
            )
            lines.append(
                f"- {canvas.run_id}: mode={canvas.collaboration_mode} tier={canvas.tier} "
                f"entitlement={canvas.entitlement_status} billing={canvas.billing_model} "
                f"estimated_cost_usd={canvas.estimated_cost_usd:.2f} overage_cost_usd={canvas.overage_cost_usd:.2f} "
                f"upgrade_required={canvas.upgrade_required} handoff={canvas.handoff_team} "
                f"collaboration={collaboration_summary} recommendation={canvas.recommendation} "
                f"actions={render_console_actions(canvas.actions)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_billing_entitlements_report(
    page: BillingEntitlementsPage,
    view: Optional[SharedViewContext] = None,
) -> str:
    entitlements = " ".join(
        f"{status}={count}" for status, count in sorted(page.entitlement_counts.items())
    ) or "none"
    billing_models = " ".join(
        f"{model}={count}" for model, count in sorted(page.billing_model_counts.items())
    ) or "none"
    blocked = ", ".join(page.blocked_capabilities) if page.blocked_capabilities else "none"
    lines = [
        "# Billing & Entitlements Report",
        "",
        f"- Workspace: {page.workspace_name}",
        f"- Plan: {page.plan_name}",
        f"- Billing Period: {page.billing_period}",
        f"- Runs: {page.run_count}",
        f"- Recommendation: {page.recommendation}",
        f"- Entitlement Mix: {entitlements}",
        f"- Billing Models: {billing_models}",
        f"- Included Usage Units: {page.total_included_usage_units}",
        f"- Overage Usage Units: {page.total_overage_usage_units}",
        f"- Estimated Cost (USD): {page.total_estimated_cost_usd:.2f}",
        f"- Overage Cost (USD): {page.total_overage_cost_usd:.2f}",
        f"- Upgrade Required Count: {page.upgrade_required_count}",
        f"- Blocked Capabilities: {blocked}",
        "",
        "## Charges",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if page.charges:
        for charge in page.charges:
            blocked_capabilities = ", ".join(charge.blocked_capabilities) if charge.blocked_capabilities else "none"
            lines.append(
                f"- {charge.run_id}: task={charge.task_id} entitlement={charge.entitlement_status} "
                f"billing={charge.billing_model} included_units={charge.included_usage_units} "
                f"overage_units={charge.overage_usage_units} estimated_cost_usd={charge.estimated_cost_usd:.2f} "
                f"overage_cost_usd={charge.overage_cost_usd:.2f} blocked={blocked_capabilities} "
                f"handoff={charge.handoff_team} recommendation={charge.recommendation}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_orchestration_overview_page(portfolio: OrchestrationPortfolio) -> str:
    def render_items(items: List[str]) -> str:
        if not items:
            return "<li>None</li>"
        return "".join(f"<li>{item}</li>" for item in items)

    collaboration = render_items(
        [f"<strong>{escape(mode)}</strong>: {count}" for mode, count in sorted(portfolio.collaboration_modes.items())]
    )
    tiers = render_items(
        [f"<strong>{escape(tier)}</strong>: {count}" for tier, count in sorted(portfolio.tier_counts.items())]
    )
    entitlements = render_items(
        [
            f"<strong>{escape(status)}</strong>: {count}"
            for status, count in sorted(portfolio.entitlement_counts.items())
        ]
    )
    billing_models = render_items(
        [
            f"<strong>{escape(model)}</strong>: {count}"
            for model, count in sorted(portfolio.billing_model_counts.items())
        ]
    )
    runs = render_items(
        [
            f"<strong>{escape(canvas.run_id)}</strong> · mode={escape(canvas.collaboration_mode)} · tier={escape(canvas.tier)} · entitlement={escape(canvas.entitlement_status)} · billing={escape(canvas.billing_model)} · cost=${canvas.estimated_cost_usd:.2f} · handoff={escape(canvas.handoff_team)} · comments={len(canvas.collaboration.comments) if canvas.collaboration is not None else 0} · decisions={len(canvas.collaboration.decisions) if canvas.collaboration is not None else 0} · recommendation={escape(canvas.recommendation)} · actions={escape(render_console_actions(canvas.actions or _default_canvas_actions(canvas)))}"
            for canvas in portfolio.canvases
        ]
    )
    takeover = "none"
    if portfolio.takeover_queue is not None:
        takeover = (
            f"pending={portfolio.takeover_queue.pending_requests} recommendation={portfolio.takeover_queue.recommendation}"
        )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Orchestration Overview · {escape(portfolio.name)}</title>
  <style>
    :root {{ color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }}
    body {{ margin: 2rem auto; max-width: 1080px; padding: 0 1rem 3rem; line-height: 1.5; }}
    .grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; margin: 1rem 0 1.5rem; }}
    .card {{ border: 1px solid #cbd5e1; border-radius: 10px; padding: 0.9rem; background: rgba(148, 163, 184, 0.08); }}
    h1, h2 {{ margin-bottom: 0.5rem; }}
    ul {{ padding-left: 1.2rem; }}
    code {{ font-size: 0.95em; }}
  </style>
</head>
<body>
  <h1>Orchestration Overview</h1>
  <p>{escape(portfolio.name)} · {escape(portfolio.period)}</p>
    <div class="grid">
      <div class="card"><strong>Total Runs</strong><br>{portfolio.total_runs}</div>
      <div class="card"><strong>Recommendation</strong><br>{escape(portfolio.recommendation)}</div>
      <div class="card"><strong>Upgrade Required</strong><br>{portfolio.upgrade_required_count}</div>
      <div class="card"><strong>Estimated Cost</strong><br>${portfolio.total_estimated_cost_usd:.2f}</div>
      <div class="card"><strong>Overage Cost</strong><br>${portfolio.total_overage_cost_usd:.2f}</div>
      <div class="card"><strong>Active Handoffs</strong><br>{portfolio.active_handoffs}</div>
      <div class="card"><strong>Takeover Queue</strong><br>{escape(takeover)}</div>
    </div>
  <h2>Collaboration Mix</h2>
  <ul>{collaboration}</ul>
  <h2>Tier Mix</h2>
  <ul>{tiers}</ul>
  <h2>Entitlement Mix</h2>
  <ul>{entitlements}</ul>
  <h2>Billing Models</h2>
  <ul>{billing_models}</ul>
  <h2>Runs</h2>
  <ul>{runs}</ul>
</body>
</html>
"""


def render_billing_entitlements_page(page: BillingEntitlementsPage) -> str:
    def render_items(items: List[str]) -> str:
        if not items:
            return "<li>None</li>"
        return "".join(f"<li>{item}</li>" for item in items)

    entitlements = render_items(
        [f"<strong>{escape(status)}</strong>: {count}" for status, count in sorted(page.entitlement_counts.items())]
    )
    billing_models = render_items(
        [f"<strong>{escape(model)}</strong>: {count}" for model, count in sorted(page.billing_model_counts.items())]
    )
    blocked = render_items([escape(capability) for capability in page.blocked_capabilities])
    charges = render_items(
        [
            f"<strong>{escape(charge.run_id)}</strong> · task={escape(charge.task_id)} · entitlement={escape(charge.entitlement_status)} · billing={escape(charge.billing_model)} · included={charge.included_usage_units} · overage={charge.overage_usage_units} · cost=${charge.estimated_cost_usd:.2f} · recommendation={escape(charge.recommendation)}"
            for charge in page.charges
        ]
    )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Billing & Entitlements · {escape(page.workspace_name)}</title>
  <style>
    :root {{
      color-scheme: light;
      --ink: #102033;
      --muted: #5c6876;
      --canvas: #f6f0e4;
      --panel: rgba(255, 252, 246, 0.9);
      --line: rgba(16, 32, 51, 0.12);
      --accent: #b45309;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top right, rgba(22, 101, 52, 0.12), transparent 24%),
        radial-gradient(circle at left center, rgba(180, 83, 9, 0.12), transparent 28%),
        linear-gradient(180deg, #fffaf2 0%, var(--canvas) 100%);
    }}
    main {{ width: min(1180px, calc(100% - 2rem)); margin: 0 auto; padding: 2rem 0 3rem; }}
    .hero {{
      border: 1px solid var(--line);
      border-radius: 28px;
      background: linear-gradient(135deg, rgba(255,255,255,0.82), rgba(255,247,237,0.94));
      box-shadow: 0 20px 48px rgba(16, 32, 51, 0.08);
      padding: 1.5rem;
    }}
    .eyebrow {{
      display: inline-block;
      font: 600 0.75rem/1.2 "SFMono-Regular", Consolas, monospace;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 0.75rem;
    }}
    h1, h2, p {{ margin: 0; }}
    .hero p {{ color: var(--muted); line-height: 1.6; max-width: 70ch; margin-top: 0.55rem; }}
    .metrics {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 0.85rem;
      margin-top: 1.35rem;
    }}
    .card, .surface {{
      border: 1px solid var(--line);
      border-radius: 20px;
      background: var(--panel);
      padding: 1rem;
    }}
    .card strong {{ display: block; font-size: 1.2rem; margin-top: 0.35rem; }}
    .card span {{
      color: var(--muted);
      font: 500 0.78rem/1.4 "SFMono-Regular", Consolas, monospace;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }}
    .layout {{
      display: grid;
      grid-template-columns: minmax(0, 1.35fr) minmax(280px, 0.85fr);
      gap: 1rem;
      margin-top: 1rem;
    }}
    .stack {{ display: grid; gap: 1rem; }}
    ul {{ margin: 0; padding-left: 1.15rem; }}
    li {{ margin: 0.35rem 0; }}
    .section-title {{ margin-bottom: 0.7rem; }}
    @media (max-width: 860px) {{
      .layout {{ grid-template-columns: 1fr; }}
    }}
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <span class="eyebrow">Billing & Entitlements</span>
      <h1>{escape(page.workspace_name)}</h1>
      <p>{escape(page.plan_name)} plan for {escape(page.billing_period)}. Recommendation: {escape(page.recommendation)}.</p>
      <div class="metrics">
        <article class="card"><span>Runs</span><strong>{page.run_count}</strong></article>
        <article class="card"><span>Included Units</span><strong>{page.total_included_usage_units}</strong></article>
        <article class="card"><span>Overage Units</span><strong>{page.total_overage_usage_units}</strong></article>
        <article class="card"><span>Estimated Cost</span><strong>${page.total_estimated_cost_usd:.2f}</strong></article>
        <article class="card"><span>Overage Cost</span><strong>${page.total_overage_cost_usd:.2f}</strong></article>
        <article class="card"><span>Upgrade Required</span><strong>{page.upgrade_required_count}</strong></article>
      </div>
    </section>
    <section class="layout">
      <div class="stack">
        <section class="surface">
          <h2 class="section-title">Charge Feed</h2>
          <ul>{charges}</ul>
        </section>
      </div>
      <div class="stack">
        <section class="surface">
          <h2 class="section-title">Entitlement Mix</h2>
          <ul>{entitlements}</ul>
        </section>
        <section class="surface">
          <h2 class="section-title">Billing Models</h2>
          <ul>{billing_models}</ul>
        </section>
        <section class="surface">
          <h2 class="section-title">Blocked Capabilities</h2>
          <ul>{blocked}</ul>
        </section>
      </div>
    </section>
  </main>
</body>
</html>
"""


def build_orchestration_canvas_from_ledger_entry(entry: dict) -> OrchestrationCanvas:
    audits = entry.get("audits", [])

    plan_audit = _latest_named_audit(audits, "orchestration.plan")
    policy_audit = _latest_named_audit(audits, "orchestration.policy")
    handoff_audit = _latest_handoff_audit(audits)
    tool_audits = [audit for audit in audits if audit.get("action") == "tool.invoke"]

    plan_details = plan_audit.get("details", {}) if plan_audit is not None else {}
    policy_details = policy_audit.get("details", {}) if policy_audit is not None else {}
    handoff_details = handoff_audit.get("details", {}) if handoff_audit is not None else {}

    active_tools = sorted(
        {
            str(audit.get("details", {}).get("tool", ""))
            for audit in tool_audits
            if audit.get("details", {}).get("tool")
        }
    )

    return OrchestrationCanvas(
        task_id=str(entry.get("task_id", "")),
        run_id=str(entry.get("run_id", "")),
        collaboration_mode=str(plan_details.get("collaboration_mode", "single-team")),
        departments=[str(value) for value in plan_details.get("departments", [])],
        required_approvals=[str(value) for value in plan_details.get("approvals", [])],
        tier=str(policy_details.get("tier", "standard")),
        upgrade_required=bool(policy_details.get("tier") and policy_audit.get("outcome") == "upgrade-required") if policy_audit is not None else False,
        blocked_departments=[str(value) for value in policy_details.get("blocked_departments", [])],
        handoff_team=str(handoff_details.get("target_team", "none")) if handoff_audit is not None else "none",
        handoff_status=str(handoff_audit.get("outcome", "none")) if handoff_audit is not None else "none",
        handoff_reason=str(handoff_details.get("reason", "")),
        active_tools=active_tools,
        entitlement_status=str(policy_details.get("entitlement_status", "included")),
        billing_model=str(policy_details.get("billing_model", "standard-included")),
        estimated_cost_usd=float(policy_details.get("estimated_cost_usd", 0.0) or 0.0),
        included_usage_units=int(policy_details.get("included_usage_units", 0) or 0),
        overage_usage_units=int(policy_details.get("overage_usage_units", 0) or 0),
        overage_cost_usd=float(policy_details.get("overage_cost_usd", 0.0) or 0.0),
        actions=build_console_actions(
            str(entry.get("run_id", "")),
            allow_retry=bool(handoff_audit is None or handoff_audit.get("outcome") != "pending"),
            retry_reason="" if handoff_audit is None or handoff_audit.get("outcome") != "pending" else "pending handoff must be resolved before retry",
            allow_pause=bool(handoff_audit is None or handoff_audit.get("outcome") != "completed"),
            pause_reason="" if handoff_audit is None or handoff_audit.get("outcome") != "completed" else "completed handoff runs cannot be paused",
            allow_reassign=handoff_audit is not None,
            reassign_reason="" if handoff_audit is not None else "reassign is available after a handoff exists",
            allow_escalate=bool(policy_audit is not None and policy_audit.get("outcome") == "upgrade-required"),
            escalate_reason="" if policy_audit is not None and policy_audit.get("outcome") == "upgrade-required" else "escalate when policy requires an entitlement or approval upgrade",
        ),
        collaboration=build_collaboration_thread_from_audits(audits, surface="flow", target_id=str(entry.get("run_id", ""))),
    )


def build_orchestration_canvas(
    run: TaskRun,
    plan: OrchestrationPlan,
    policy: Optional[OrchestrationPolicyDecision] = None,
    handoff_request: Optional[HandoffRequest] = None,
) -> OrchestrationCanvas:
    return OrchestrationCanvas(
        task_id=run.task_id,
        run_id=run.run_id,
        collaboration_mode=plan.collaboration_mode,
        departments=plan.departments,
        required_approvals=plan.required_approvals,
        tier=policy.tier if policy is not None else "standard",
        upgrade_required=policy.upgrade_required if policy is not None else False,
        blocked_departments=policy.blocked_departments if policy is not None else [],
        handoff_team=handoff_request.target_team if handoff_request is not None else "none",
        handoff_status=handoff_request.status if handoff_request is not None else "none",
        handoff_reason=handoff_request.reason if handoff_request is not None else "",
        active_tools=sorted({str(entry.details.get("tool", "")) for entry in run.audits if entry.action == "tool.invoke" and entry.details.get("tool")}),
        entitlement_status=policy.entitlement_status if policy is not None else "included",
        billing_model=policy.billing_model if policy is not None else "standard-included",
        estimated_cost_usd=policy.estimated_cost_usd if policy is not None else 0.0,
        included_usage_units=policy.included_usage_units if policy is not None else 0,
        overage_usage_units=policy.overage_usage_units if policy is not None else 0,
        overage_cost_usd=policy.overage_cost_usd if policy is not None else 0.0,
        actions=build_console_actions(
            run.run_id,
            allow_retry=handoff_request is None or handoff_request.status != "pending",
            retry_reason="" if handoff_request is None or handoff_request.status != "pending" else "pending handoff must be resolved before retry",
            allow_pause=run.status not in {"failed", "completed", "approved"},
            pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
            allow_reassign=handoff_request is not None,
            reassign_reason="" if handoff_request is not None else "reassign is available after a handoff exists",
            allow_escalate=policy is not None and policy.upgrade_required,
            escalate_reason="" if policy is not None and policy.upgrade_required else "escalate when policy requires an entitlement or approval upgrade",
        ),
        collaboration=build_collaboration_thread_from_audits(
            [entry.to_dict() for entry in run.audits],
            surface="flow",
            target_id=run.run_id,
        ),
    )


def render_orchestration_canvas(canvas: OrchestrationCanvas) -> str:
    lines = [
        "# Orchestration Canvas",
        "",
        f"- Task ID: {canvas.task_id}",
        f"- Run ID: {canvas.run_id}",
        f"- Collaboration Mode: {canvas.collaboration_mode}",
        f"- Departments: {', '.join(canvas.departments) if canvas.departments else 'none'}",
        f"- Required Approvals: {', '.join(canvas.required_approvals) if canvas.required_approvals else 'none'}",
        f"- Tier: {canvas.tier}",
        f"- Upgrade Required: {canvas.upgrade_required}",
        f"- Entitlement Status: {canvas.entitlement_status}",
        f"- Billing Model: {canvas.billing_model}",
        f"- Blocked Departments: {', '.join(canvas.blocked_departments) if canvas.blocked_departments else 'none'}",
        f"- Handoff Team: {canvas.handoff_team}",
        f"- Handoff Status: {canvas.handoff_status}",
        f"- Recommendation: {canvas.recommendation}",
        "",
        "## Execution Context",
        "",
        f"- Active Tools: {', '.join(canvas.active_tools) if canvas.active_tools else 'none'}",
        f"- Estimated Cost (USD): {canvas.estimated_cost_usd:.2f}",
        f"- Included Usage Units: {canvas.included_usage_units}",
        f"- Overage Usage Units: {canvas.overage_usage_units}",
        f"- Overage Cost (USD): {canvas.overage_cost_usd:.2f}",
        f"- Handoff Reason: {canvas.handoff_reason or 'none'}",
        "",
        "## Actions",
        "",
        f"- {render_console_actions(canvas.actions)}",
    ]
    lines.extend(render_collaboration_lines(canvas.collaboration))
    return "\n".join(lines) + "\n"


def build_orchestration_portfolio_from_ledger(
    entries: List[dict],
    name: str = "Cross-Department Orchestration",
    period: str = "current",
) -> OrchestrationPortfolio:
    canvases = [
        build_orchestration_canvas_from_ledger_entry(entry)
        for entry in entries
        if _latest_named_audit(entry.get("audits", []), "orchestration.plan") is not None
    ]
    takeover_queue = build_takeover_queue_from_ledger(entries, name=f"{name} Takeovers", period=period)
    return build_orchestration_portfolio(
        canvases,
        name=name,
        period=period,
        takeover_queue=takeover_queue,
    )


def build_billing_entitlements_page_from_ledger(
    entries: List[dict],
    *,
    workspace_name: str = "BigClaw Cloud",
    plan_name: str = "Standard",
    billing_period: str = "current",
) -> BillingEntitlementsPage:
    portfolio = build_orchestration_portfolio_from_ledger(entries, name=workspace_name, period=billing_period)
    return build_billing_entitlements_page(
        portfolio,
        workspace_name=workspace_name,
        plan_name=plan_name,
        billing_period=billing_period,
    )


def build_takeover_queue_from_ledger(
    entries: List[dict],
    name: str = "Human Takeover Queue",
    period: str = "current",
) -> TakeoverQueue:
    requests: List[TakeoverRequest] = []
    for entry in entries:
        handoff_audit = _latest_handoff_audit(entry.get("audits", []))
        if handoff_audit is None:
            continue

        details = handoff_audit.get("details", {})
        requests.append(
            TakeoverRequest(
                run_id=str(entry.get("run_id", "")),
                task_id=str(entry.get("task_id", "")),
                source=str(entry.get("source", "")),
                target_team=str(details.get("target_team", "operations")),
                status=str(handoff_audit.get("outcome", "pending")),
                reason=str(details.get("reason", entry.get("summary", "handoff requested"))),
                required_approvals=[str(value) for value in details.get("required_approvals", [])],
                actions=build_console_actions(
                    str(entry.get("run_id", "")),
                    allow_retry=False,
                    retry_reason="retry is blocked while takeover is pending",
                    allow_pause=str(handoff_audit.get("outcome", "pending")) == "pending",
                    pause_reason="" if str(handoff_audit.get("outcome", "pending")) == "pending" else "only pending takeovers can be paused",
                    allow_reassign=True,
                    allow_escalate=str(details.get("target_team", "")) != "security",
                    escalate_reason="" if str(details.get("target_team", "")) != "security" else "security takeovers are already escalated",
                ),
            )
        )

    requests.sort(key=lambda request: (request.target_team, request.run_id))
    return TakeoverQueue(name=name, period=period, requests=requests)


def render_takeover_queue_report(
    queue: TakeoverQueue,
    total_runs: Optional[int] = None,
    view: Optional[SharedViewContext] = None,
) -> str:
    team_counts = queue.team_counts
    team_mix = " ".join(f"{team}={count}" for team, count in sorted(team_counts.items())) or "none"
    lines = [
        "# Human Takeover Queue",
        "",
        f"- Queue: {queue.name}",
        f"- Period: {queue.period}",
        f"- Pending Requests: {queue.pending_requests}",
        f"- Total Runs: {total_runs if total_runs is not None else queue.pending_requests}",
        f"- Recommendation: {queue.recommendation}",
        f"- Team Mix: {team_mix}",
        f"- Required Approvals: {queue.approval_count}",
        "",
        "## Requests",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if queue.requests:
        for request in queue.requests:
            approvals = ",".join(request.required_approvals) if request.required_approvals else "none"
            lines.append(
                f"- {request.run_id}: team={request.target_team} status={request.status} task={request.task_id} "
                f"approvals={approvals} reason={request.reason} actions={render_console_actions(request.actions)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def _latest_named_audit(audits: List[dict], action: str) -> Optional[dict]:
    for audit in reversed(audits):
        if audit.get("action") == action:
            return audit
    return None


def _latest_handoff_audit(audits: List[dict]) -> Optional[dict]:
    for action in (MANUAL_TAKEOVER_EVENT, FLOW_HANDOFF_EVENT, "orchestration.handoff"):
        audit = _latest_named_audit(audits, action)
        if audit is not None:
            return audit
    return None


def _run_requires_triage(run: TaskRun) -> bool:
    if run.status in {"failed", "needs-approval"}:
        return True
    if any(entry.status in {"pending", "error", "failed"} for entry in run.traces):
        return True
    return any(entry.outcome in {"pending", "failed", "rejected"} for entry in run.audits)


def _triage_severity(run: TaskRun) -> str:
    if run.status == "failed":
        return "critical"
    if any(entry.status in {"error", "failed"} for entry in run.traces):
        return "critical"
    if any(entry.outcome in {"failed", "rejected"} for entry in run.audits):
        return "critical"
    if run.status == "needs-approval":
        return "high"
    if any(entry.status == "pending" for entry in run.traces):
        return "high"
    if any(entry.outcome == "pending" for entry in run.audits):
        return "high"
    return "medium"


def _triage_owner(run: TaskRun) -> str:
    evidence = " ".join(
        [run.summary, run.title, run.source, run.medium]
        + [entry.status for entry in run.traces]
        + [entry.span for entry in run.traces]
        + [entry.outcome for entry in run.audits]
        + [str(entry.details.get("reason", "")) for entry in run.audits]
        + [str(entry.details.get("approvals", [])) for entry in run.audits]
    ).lower()

    if "security" in evidence or "high-risk" in evidence or "security-review" in evidence:
        return "security"
    if run.medium == "browser" or any(artifact.kind == "page" for artifact in run.artifacts):
        return "engineering"
    return "operations"


def _triage_reason(run: TaskRun) -> str:
    for audit in run.audits:
        if audit.outcome in {"failed", "rejected", "pending"} and audit.details.get("reason"):
            return str(audit.details["reason"])
    for trace in run.traces:
        if trace.status in {"error", "failed", "pending"}:
            return f"{trace.span} is {trace.status}"
    return run.summary or run.status


def _triage_next_action(severity: str, owner: str) -> str:
    if severity == "critical":
        if owner == "engineering":
            return "replay run and inspect tool failures"
        if owner == "security":
            return "page security reviewer and block rollout"
        return "open incident review and coordinate response"
    if owner == "security":
        return "request approval and queue security review"
    if owner == "engineering":
        return "inspect execution evidence and retry when safe"
    return "confirm owner and clear pending workflow gate"


def _build_triage_suggestions(
    run: TaskRun,
    runs: List[TaskRun],
    severity: str,
    owner: str,
    feedback: List[TriageFeedbackRecord],
) -> List[TriageSuggestion]:
    action = _triage_next_action(severity, owner)
    label = _triage_suggestion_label(run, severity, owner)
    evidence = _similarity_evidence(run, runs)
    confidence = _triage_suggestion_confidence(run, evidence)
    feedback_status = _feedback_status(run.run_id, action, feedback)
    return [
        TriageSuggestion(
            label=label,
            action=action,
            owner=owner,
            confidence=confidence,
            evidence=evidence,
            feedback_status=feedback_status,
        )
    ]


def _triage_suggestion_label(run: TaskRun, severity: str, owner: str) -> str:
    if severity == "critical" and owner == "engineering":
        return "replay candidate"
    if owner == "security":
        return "approval review"
    if run.status == "failed":
        return "incident review"
    return "workflow follow-up"


def _triage_suggestion_confidence(run: TaskRun, evidence: List[TriageSimilarityEvidence]) -> float:
    base = 0.55 if run.status in {"needs-approval", "failed"} else 0.45
    if evidence:
        base = max(base, min(0.95, 0.45 + evidence[0].score / 2))
    return round(base, 2)


def _feedback_status(run_id: str, action: str, feedback: List[TriageFeedbackRecord]) -> str:
    for record in reversed(feedback):
        if record.run_id == run_id and record.action == action:
            return record.decision
    return "pending"


def _slugify(value: str) -> str:
    normalized = "".join(char.lower() if char.isalnum() else "-" for char in value.strip())
    return "-".join(part for part in normalized.split("-") if part)


def _similarity_evidence(run: TaskRun, runs: List[TaskRun], limit: int = 2) -> List[TriageSimilarityEvidence]:
    scored_matches: List[tuple[float, TaskRun]] = []
    for candidate in runs:
        if candidate.run_id == run.run_id:
            continue
        score = _run_similarity_score(run, candidate)
        if score < 0.35:
            continue
        scored_matches.append((score, candidate))

    scored_matches.sort(key=lambda item: (-item[0], item[1].run_id))
    evidence: List[TriageSimilarityEvidence] = []
    for score, candidate in scored_matches[:limit]:
        evidence.append(
            TriageSimilarityEvidence(
                related_run_id=candidate.run_id,
                related_task_id=candidate.task_id,
                score=round(score, 2),
                reason=_similarity_reason(run, candidate),
            )
        )
    return evidence


def _run_similarity_score(run: TaskRun, candidate: TaskRun) -> float:
    haystack = " ".join(
        [
            run.title,
            run.summary,
            " ".join(trace.span for trace in run.traces),
            " ".join(audit.outcome for audit in run.audits),
        ]
    ).lower()
    needle = " ".join(
        [
            candidate.title,
            candidate.summary,
            " ".join(trace.span for trace in candidate.traces),
            " ".join(audit.outcome for audit in candidate.audits),
        ]
    ).lower()
    status_bonus = 0.15 if run.status == candidate.status else 0.0
    owner_bonus = 0.1 if _triage_owner(run) == _triage_owner(candidate) else 0.0
    return min(1.0, SequenceMatcher(a=haystack, b=needle).ratio() + status_bonus + owner_bonus)


def _similarity_reason(run: TaskRun, candidate: TaskRun) -> str:
    reasons: List[str] = []
    if run.status == candidate.status:
        reasons.append(f"shared status {run.status}")
    if _triage_owner(run) == _triage_owner(candidate):
        reasons.append(f"shared owner {_triage_owner(run)}")
    run_reason = _triage_reason(run)
    candidate_reason = _triage_reason(candidate)
    if run_reason == candidate_reason:
        reasons.append("matching failure reason")
    return ", ".join(reasons) or "similar execution trail"


def render_repo_sync_audit_report(audit: RepoSyncAudit) -> str:
    lines = [
        "# Repo Sync Audit",
        "",
        "## Sync Status",
        "",
        f"- Status: {audit.sync.status}",
        f"- Failure Category: {audit.sync.failure_category or 'none'}",
        f"- Summary: {audit.sync.summary or 'none'}",
        f"- Branch: {audit.sync.branch or 'unknown'}",
        f"- Remote: {audit.sync.remote}",
        f"- Remote Ref: {audit.sync.remote_ref or 'unknown'}",
        f"- Ahead By: {audit.sync.ahead_by}",
        f"- Behind By: {audit.sync.behind_by}",
        f"- Dirty Paths: {', '.join(audit.sync.dirty_paths) if audit.sync.dirty_paths else 'none'}",
        f"- Auth Target: {audit.sync.auth_target or 'none'}",
        f"- Checked At: {audit.sync.timestamp}",
        "",
        "## Pull Request Freshness",
        "",
        f"- PR Number: {audit.pull_request.pr_number if audit.pull_request.pr_number is not None else 'unknown'}",
        f"- PR URL: {audit.pull_request.pr_url or 'none'}",
        f"- Branch State: {audit.pull_request.branch_state}",
        f"- Body State: {audit.pull_request.body_state}",
        f"- Branch Head SHA: {audit.pull_request.branch_head_sha or 'unknown'}",
        f"- PR Head SHA: {audit.pull_request.pr_head_sha or 'unknown'}",
        f"- Expected Body Digest: {audit.pull_request.expected_body_digest or 'unknown'}",
        f"- Actual Body Digest: {audit.pull_request.actual_body_digest or 'unknown'}",
        f"- Checked At: {audit.pull_request.checked_at}",
        "",
        "## Summary",
        "",
        f"- {audit.summary}",
    ]
    return "\n".join(lines) + "\n"


def render_task_run_report(run: TaskRun) -> str:
    actions = build_console_actions(
        run.run_id,
        allow_retry=run.status in {"failed", "needs-approval"},
        retry_reason="" if run.status in {"failed", "needs-approval"} else "retry is available for failed or approval-blocked runs",
        allow_pause=run.status not in {"failed", "completed", "approved"},
        pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
    )
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in run.audits],
        surface="run",
        target_id=run.run_id,
    )
    lines = [
        "# Task Run Report",
        "",
        f"- Run ID: {run.run_id}",
        f"- Task ID: {run.task_id}",
        f"- Source: {run.source}",
        f"- Medium: {run.medium}",
        f"- Status: {run.status}",
        f"- Started At: {run.started_at}",
        f"- Ended At: {run.ended_at or 'n/a'}",
        "",
        "## Summary",
        "",
        run.summary or "No summary recorded.",
        "",
        "## Logs",
        "",
    ]

    if run.logs:
        lines.extend(
            f"- [{entry.level}] {entry.timestamp} {entry.message}" for entry in run.logs
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Trace", ""])
    if run.traces:
        lines.extend(
            f"- {entry.span}: {entry.status} @ {entry.timestamp}" for entry in run.traces
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Artifacts", ""])
    if run.artifacts:
        lines.extend(
            f"- {entry.name} ({entry.kind}): {entry.path}" for entry in run.artifacts
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    if run.audits:
        lines.extend(
            f"- {entry.action} by {entry.actor}: {entry.outcome}" for entry in run.audits
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Closeout", ""])
    lines.append(f"- Complete: {run.closeout.complete}")
    lines.append(
        "- Validation Evidence: "
        + (", ".join(run.closeout.validation_evidence) if run.closeout.validation_evidence else "None")
    )
    lines.append(f"- Git Push Succeeded: {run.closeout.git_push_succeeded}")
    lines.append(f"- Git Push Output: {run.closeout.git_push_output or 'None'}")
    lines.append(f"- Git Log -1 --stat Output: {run.closeout.git_log_stat_output or 'None'}")
    if run.closeout.repo_sync_audit is not None:
        lines.append(f"- Repo Sync Status: {run.closeout.repo_sync_audit.sync.status}")
        lines.append(
            "- Repo Sync Failure Category: "
            + (run.closeout.repo_sync_audit.sync.failure_category or "none")
        )
        lines.append(f"- PR Branch State: {run.closeout.repo_sync_audit.pull_request.branch_state}")
        lines.append(f"- PR Body State: {run.closeout.repo_sync_audit.pull_request.body_state}")
    lines.extend(["", "## Actions", "", f"- {render_console_actions(actions)}"])
    lines.extend(render_collaboration_lines(collaboration))

    return "\n".join(lines) + "\n"


def render_task_run_detail_page(run: TaskRun) -> str:
    status_tone = "accent" if run.status in {"approved", "completed", "succeeded"} else "warning"
    if run.status in {"failed", "rejected"}:
        status_tone = "danger"

    actions = build_console_actions(
        run.run_id,
        allow_retry=run.status in {"failed", "needs-approval"},
        retry_reason="" if run.status in {"failed", "needs-approval"} else "retry is available for failed or approval-blocked runs",
        allow_pause=run.status not in {"failed", "completed", "approved"},
        pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
    )
    timeline_events = sorted(
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
                for index, entry in enumerate(run.logs)
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
                for index, entry in enumerate(run.traces)
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
                for index, entry in enumerate(run.audits)
            ],
            *[
                RunDetailEvent(
                    event_id=f"artifact-{index}",
                    lane="artifact",
                    title=entry.name,
                    timestamp=entry.timestamp,
                    status=entry.kind,
                    summary=f"artifact emitted at {entry.path}",
                    details=[
                        f"path={entry.path}",
                        f"sha256={entry.sha256 or 'n/a'}",
                        *[f"{key}={value}" for key, value in sorted(entry.metadata.items())],
                    ],
                )
                for index, entry in enumerate(run.artifacts)
            ],
        ],
        key=lambda event: event.timestamp,
    )
    artifacts = [
        RunDetailResource(
            name=entry.name,
            kind=entry.kind,
            path=entry.path,
            meta=[f"sha256={entry.sha256 or 'n/a'}", *[f"{key}={value}" for key, value in sorted(entry.metadata.items())]],
            tone="report" if entry.kind == "report" else "page" if entry.kind == "page" else "default",
        )
        for entry in run.artifacts
    ]
    report_resources = [resource for resource in artifacts if resource.kind == "report"]
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in run.audits],
        surface="run",
        target_id=run.run_id,
    )

    repo_link_resources = [
        RunDetailResource(
            name=link.commit_hash,
            kind=link.role,
            path=f"repo:{link.repo_space_id}",
            meta=[f"actor={link.actor or 'unknown'}", *[f"{key}={value}" for key, value in sorted(link.metadata.items())]],
            tone="accent" if link.role == "accepted" else "default",
        )
        for link in run.closeout.run_commit_links
    ]

    overview_html = f"""
    <section class="surface">
      <h2>Overview</h2>
      <p>{escape(run.summary or 'No summary recorded.')}</p>
      <p class="meta">Task {escape(run.task_id)} from {escape(run.source)} started at {escape(run.started_at)} and ended at {escape(run.ended_at or 'n/a')}.</p>
    </section>
    <section class="surface">
      <h2>Closeout</h2>
      <p>Validation evidence: {escape(', '.join(run.closeout.validation_evidence) if run.closeout.validation_evidence else 'None recorded.')}</p>
      <p class="meta">git push succeeded={escape(str(run.closeout.git_push_succeeded))} | git log captured={escape(str(bool(run.closeout.git_log_stat_output.strip())))} | complete={escape(str(run.closeout.complete))}</p>
      <p class="meta">accepted_commit_hash={escape(run.closeout.accepted_commit_hash or 'none')} | commit_links={escape(str(len(run.closeout.run_commit_links)))}</p>
    </section>
    <section class="surface">
      <h2>Actions</h2>
      <p>{escape(render_console_actions(actions))}</p>
    </section>
    """

    return render_run_detail_console(
        page_title=f"Task Run Detail · {run.run_id}",
        eyebrow="Run Detail",
        hero_title=run.title,
        hero_summary=run.summary or "Operational detail page with synced logs, traces, audits, and artifacts.",
        stats=[
            RunDetailStat("Run ID", run.run_id),
            RunDetailStat("Task ID", run.task_id),
            RunDetailStat("Medium", run.medium, tone="accent" if run.medium == "browser" else "default"),
            RunDetailStat("Status", run.status, tone=status_tone),
            RunDetailStat("Artifacts", str(len(run.artifacts))),
            RunDetailStat("Reports", str(len(report_resources)), tone="accent" if report_resources else "default"),
            RunDetailStat("Closeout", "complete" if run.closeout.complete else "pending", tone="accent" if run.closeout.complete else "warning"),
            RunDetailStat("Repo Links", str(len(run.closeout.run_commit_links)), tone="accent" if run.closeout.run_commit_links else "default"),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", overview_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Unified execution timeline for logs, traces, audits, and emitted artifacts. Selecting an item updates the inspector in the split view.",
                    timeline_events,
                ),
            ),
            RunDetailTab(
                "artifacts",
                "Artifacts",
                render_resource_grid(
                    "Artifacts",
                    "Execution artifacts and generated outputs attached to this run.",
                    artifacts,
                ),
            ),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Report artifacts emitted for this run, including markdown summaries and linked detail pages when present.",
                    report_resources,
                ),
            ),
            RunDetailTab(
                "repo-evidence",
                "Repo Evidence",
                render_resource_grid(
                    "Repo Evidence",
                    "Commit links, roles, and accepted lineage hints bound at closeout.",
                    repo_link_resources,
                ),
            ),
            RunDetailTab(
                "collaboration",
                "Collaboration",
                render_collaboration_panel_html(
                    "Collaboration",
                    "Comments, mentions, and decision notes recorded against this run.",
                    collaboration,
                ),
            ),
        ],
        timeline_events=timeline_events,
    )


def render_weekly_repo_evidence_section(
    *,
    experiment_volume: int,
    converged_tasks: int,
    accepted_commits: int,
    hottest_threads: List[str],
) -> str:
    lines = [
        "## Repo Evidence Summary",
        f"- Experiment Volume: {experiment_volume}",
        f"- Converged Tasks: {converged_tasks}",
        f"- Accepted Commits: {accepted_commits}",
        f"- Hottest Threads: {', '.join(hottest_threads) if hottest_threads else 'none'}",
    ]
    return "\n".join(lines)


def render_repo_narrative_exports(
    *,
    experiment_volume: int,
    converged_tasks: int,
    accepted_commits: int,
    hottest_threads: List[str],
) -> dict:
    markdown_text = render_weekly_repo_evidence_section(
        experiment_volume=experiment_volume,
        converged_tasks=converged_tasks,
        accepted_commits=accepted_commits,
        hottest_threads=hottest_threads,
    )
    plain_text = markdown_text.replace("## ", "")
    html = (
        "<section><h2>Repo Evidence Summary</h2>"
        f"<p>Experiment Volume: {experiment_volume}</p>"
        f"<p>Converged Tasks: {converged_tasks}</p>"
        f"<p>Accepted Commits: {accepted_commits}</p>"
        f"<p>Hottest Threads: {escape(', '.join(hottest_threads) if hottest_threads else 'none')}</p>"
        "</section>"
    )
    return {"markdown": markdown_text, "text": plain_text, "html": html}

''',
    _reports_module.__dict__,
)
_reports_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/reporting/reporting.go"
globals()["reports"] = _reports_module

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
_operations_module = types.ModuleType(f"{__name__}.operations")
sys.modules[_operations_module.__name__] = _operations_module
exec(
'from dataclasses import dataclass, field\nfrom datetime import datetime, timezone\nfrom difflib import unified_diff\nfrom pathlib import Path\nfrom typing import Dict, List, Optional, Sequence\n\nfrom .models import Task\nfrom .queue import PersistentTaskQueue\n\nfrom .evaluation import BenchmarkSuiteResult\nfrom .reports import (\n    SharedViewContext,\n    build_console_actions,\n    render_console_actions,\n    render_shared_view_context,\n    write_report,\n)\n\n\nSTATUS_COMPLETE = {"approved", "accepted", "completed", "succeeded"}\nSTATUS_ACTIONABLE = {"needs-approval", "failed", "rejected"}\n\n\n@dataclass\nclass TriageCluster:\n    reason: str\n    run_ids: List[str] = field(default_factory=list)\n    task_ids: List[str] = field(default_factory=list)\n    statuses: List[str] = field(default_factory=list)\n\n    @property\n    def occurrences(self) -> int:\n        return len(self.run_ids)\n\n\n@dataclass\nclass RegressionFinding:\n    case_id: str\n    baseline_score: int\n    current_score: int\n    delta: int\n    severity: str\n    summary: str\n\n\n@dataclass\nclass OperationsSnapshot:\n    total_runs: int\n    status_counts: Dict[str, int]\n    success_rate: float\n    approval_queue_depth: int\n    sla_target_minutes: int\n    sla_breach_count: int\n    average_cycle_minutes: float\n    top_blockers: List[TriageCluster] = field(default_factory=list)\n\n\n@dataclass\nclass WeeklyOperationsReport:\n    name: str\n    period: str\n    snapshot: OperationsSnapshot\n    regressions: List[RegressionFinding] = field(default_factory=list)\n\n\n@dataclass\nclass RegressionCenter:\n    name: str\n    baseline_version: str\n    current_version: str\n    regressions: List[RegressionFinding] = field(default_factory=list)\n    improved_cases: List[str] = field(default_factory=list)\n    unchanged_cases: List[str] = field(default_factory=list)\n\n    @property\n    def regression_count(self) -> int:\n        return len(self.regressions)\n\n\n@dataclass\nclass VersionedArtifact:\n    artifact_type: str\n    artifact_id: str\n    version: str\n    updated_at: str\n    author: str\n    summary: str\n    content: str\n    change_ticket: Optional[str] = None\n\n\n@dataclass\nclass VersionChangeSummary:\n    from_version: str\n    to_version: str\n    additions: int\n    deletions: int\n    changed_lines: int\n    preview: List[str] = field(default_factory=list)\n\n    @property\n    def has_changes(self) -> bool:\n        return self.changed_lines > 0\n\n\n@dataclass\nclass VersionedArtifactHistory:\n    artifact_type: str\n    artifact_id: str\n    current_version: str\n    current_updated_at: str\n    current_author: str\n    current_summary: str\n    revision_count: int\n    revisions: List[VersionedArtifact] = field(default_factory=list)\n    rollback_version: Optional[str] = None\n    rollback_ready: bool = False\n    change_summary: Optional[VersionChangeSummary] = None\n\n\n@dataclass\nclass PolicyPromptVersionCenter:\n    name: str\n    generated_at: str\n    histories: List[VersionedArtifactHistory] = field(default_factory=list)\n\n    @property\n    def artifact_count(self) -> int:\n        return len(self.histories)\n\n    @property\n    def rollback_ready_count(self) -> int:\n        return sum(1 for history in self.histories if history.rollback_ready)\n\n\n@dataclass\nclass WeeklyOperationsArtifacts:\n    root_dir: str\n    weekly_report_path: str\n    dashboard_path: str\n    metric_spec_path: Optional[str] = None\n    regression_center_path: Optional[str] = None\n    queue_control_path: Optional[str] = None\n    version_center_path: Optional[str] = None\n\n\n@dataclass\nclass QueueControlCenter:\n    queue_depth: int\n    queued_by_priority: Dict[str, int]\n    queued_by_risk: Dict[str, int]\n    execution_media: Dict[str, int]\n    waiting_approval_runs: int\n    blocked_tasks: List[str] = field(default_factory=list)\n    queued_tasks: List[str] = field(default_factory=list)\n    actions: Dict[str, List] = field(default_factory=dict)\n\n\n@dataclass\nclass EngineeringOverviewKPI:\n    name: str\n    value: float\n    target: float\n    unit: str = ""\n    direction: str = "up"\n\n    @property\n    def healthy(self) -> bool:\n        if self.direction == "down":\n            return self.value <= self.target\n        return self.value >= self.target\n\n\n@dataclass\nclass EngineeringFunnelStage:\n    name: str\n    count: int\n    share: float\n\n\n@dataclass\nclass EngineeringOverviewBlocker:\n    summary: str\n    affected_runs: int\n    affected_tasks: List[str] = field(default_factory=list)\n    owner: str = "engineering"\n    severity: str = "medium"\n\n\n@dataclass\nclass EngineeringActivity:\n    timestamp: str\n    run_id: str\n    task_id: str\n    status: str\n    summary: str\n\n\n@dataclass\nclass EngineeringOverviewPermission:\n    viewer_role: str\n    allowed_modules: List[str] = field(default_factory=list)\n\n    def can_view(self, module: str) -> bool:\n        return module in self.allowed_modules\n\n\n@dataclass\nclass EngineeringOverview:\n    name: str\n    period: str\n    snapshot: OperationsSnapshot\n    permissions: EngineeringOverviewPermission\n    kpis: List[EngineeringOverviewKPI] = field(default_factory=list)\n    funnel: List[EngineeringFunnelStage] = field(default_factory=list)\n    blockers: List[EngineeringOverviewBlocker] = field(default_factory=list)\n    activities: List[EngineeringActivity] = field(default_factory=list)\n\n\n@dataclass(frozen=True)\nclass OperationsMetricDefinition:\n    metric_id: str\n    label: str\n    unit: str\n    direction: str\n    formula: str\n    description: str\n    source_fields: List[str] = field(default_factory=list)\n\n\n@dataclass(frozen=True)\nclass OperationsMetricValue:\n    metric_id: str\n    label: str\n    value: float\n    display_value: str\n    numerator: float\n    denominator: float\n    unit: str\n    evidence: List[str] = field(default_factory=list)\n\n\n@dataclass(frozen=True)\nclass OperationsMetricSpec:\n    name: str\n    generated_at: str\n    period_start: str\n    period_end: str\n    timezone_name: str\n    definitions: List[OperationsMetricDefinition] = field(default_factory=list)\n    values: List[OperationsMetricValue] = field(default_factory=list)\n\n\n@dataclass(frozen=True)\nclass DashboardWidgetSpec:\n    widget_id: str\n    title: str\n    module: str\n    data_source: str\n    default_width: int = 4\n    default_height: int = 3\n    min_width: int = 2\n    max_width: int = 12\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "widget_id": self.widget_id,\n            "title": self.title,\n            "module": self.module,\n            "data_source": self.data_source,\n            "default_width": self.default_width,\n            "default_height": self.default_height,\n            "min_width": self.min_width,\n            "max_width": self.max_width,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetSpec":\n        return cls(\n            widget_id=str(data["widget_id"]),\n            title=str(data["title"]),\n            module=str(data["module"]),\n            data_source=str(data["data_source"]),\n            default_width=int(data.get("default_width", 4)),\n            default_height=int(data.get("default_height", 3)),\n            min_width=int(data.get("min_width", 2)),\n            max_width=int(data.get("max_width", 12)),\n        )\n\n\n@dataclass(frozen=True)\nclass DashboardWidgetPlacement:\n    placement_id: str\n    widget_id: str\n    column: int\n    row: int\n    width: int\n    height: int\n    title_override: str = ""\n    filters: List[str] = field(default_factory=list)\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "placement_id": self.placement_id,\n            "widget_id": self.widget_id,\n            "column": self.column,\n            "row": self.row,\n            "width": self.width,\n            "height": self.height,\n            "title_override": self.title_override,\n            "filters": list(self.filters),\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetPlacement":\n        return cls(\n            placement_id=str(data["placement_id"]),\n            widget_id=str(data["widget_id"]),\n            column=int(data.get("column", 0)),\n            row=int(data.get("row", 0)),\n            width=int(data.get("width", 1)),\n            height=int(data.get("height", 1)),\n            title_override=str(data.get("title_override", "")),\n            filters=[str(item) for item in data.get("filters", [])],\n        )\n\n\n@dataclass\nclass DashboardLayout:\n    layout_id: str\n    name: str\n    columns: int = 12\n    placements: List[DashboardWidgetPlacement] = field(default_factory=list)\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "layout_id": self.layout_id,\n            "name": self.name,\n            "columns": self.columns,\n            "placements": [placement.to_dict() for placement in self.placements],\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "DashboardLayout":\n        return cls(\n            layout_id=str(data["layout_id"]),\n            name=str(data["name"]),\n            columns=int(data.get("columns", 12)),\n            placements=[DashboardWidgetPlacement.from_dict(item) for item in data.get("placements", [])],\n        )\n\n\n@dataclass\nclass DashboardBuilder:\n    name: str\n    period: str\n    owner: str\n    permissions: EngineeringOverviewPermission\n    widgets: List[DashboardWidgetSpec] = field(default_factory=list)\n    layouts: List[DashboardLayout] = field(default_factory=list)\n    documentation_complete: bool = False\n\n    @property\n    def widget_index(self) -> Dict[str, DashboardWidgetSpec]:\n        return {widget.widget_id: widget for widget in self.widgets}\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "name": self.name,\n            "period": self.period,\n            "owner": self.owner,\n            "permissions": {\n                "viewer_role": self.permissions.viewer_role,\n                "allowed_modules": list(self.permissions.allowed_modules),\n            },\n            "widgets": [widget.to_dict() for widget in self.widgets],\n            "layouts": [layout.to_dict() for layout in self.layouts],\n            "documentation_complete": self.documentation_complete,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilder":\n        permissions = dict(data.get("permissions", {}))\n        return cls(\n            name=str(data["name"]),\n            period=str(data["period"]),\n            owner=str(data["owner"]),\n            permissions=EngineeringOverviewPermission(\n                viewer_role=str(permissions.get("viewer_role", "contributor")),\n                allowed_modules=[str(item) for item in permissions.get("allowed_modules", [])],\n            ),\n            widgets=[DashboardWidgetSpec.from_dict(item) for item in data.get("widgets", [])],\n            layouts=[DashboardLayout.from_dict(item) for item in data.get("layouts", [])],\n            documentation_complete=bool(data.get("documentation_complete", False)),\n        )\n\n\n@dataclass\nclass DashboardBuilderAudit:\n    name: str\n    total_widgets: int\n    layout_count: int\n    placed_widgets: int\n    duplicate_placement_ids: List[str] = field(default_factory=list)\n    missing_widget_defs: List[str] = field(default_factory=list)\n    inaccessible_widgets: List[str] = field(default_factory=list)\n    overlapping_placements: List[str] = field(default_factory=list)\n    out_of_bounds_placements: List[str] = field(default_factory=list)\n    empty_layouts: List[str] = field(default_factory=list)\n    documentation_complete: bool = False\n\n    @property\n    def release_ready(self) -> bool:\n        return not (\n            self.duplicate_placement_ids\n            or self.missing_widget_defs\n            or self.inaccessible_widgets\n            or self.overlapping_placements\n            or self.out_of_bounds_placements\n            or self.empty_layouts\n            or not self.documentation_complete\n        )\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "name": self.name,\n            "total_widgets": self.total_widgets,\n            "layout_count": self.layout_count,\n            "placed_widgets": self.placed_widgets,\n            "duplicate_placement_ids": list(self.duplicate_placement_ids),\n            "missing_widget_defs": list(self.missing_widget_defs),\n            "inaccessible_widgets": list(self.inaccessible_widgets),\n            "overlapping_placements": list(self.overlapping_placements),\n            "out_of_bounds_placements": list(self.out_of_bounds_placements),\n            "empty_layouts": list(self.empty_layouts),\n            "documentation_complete": self.documentation_complete,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilderAudit":\n        return cls(\n            name=str(data["name"]),\n            total_widgets=int(data.get("total_widgets", 0)),\n            layout_count=int(data.get("layout_count", 0)),\n            placed_widgets=int(data.get("placed_widgets", 0)),\n            duplicate_placement_ids=[str(item) for item in data.get("duplicate_placement_ids", [])],\n            missing_widget_defs=[str(item) for item in data.get("missing_widget_defs", [])],\n            inaccessible_widgets=[str(item) for item in data.get("inaccessible_widgets", [])],\n            overlapping_placements=[str(item) for item in data.get("overlapping_placements", [])],\n            out_of_bounds_placements=[str(item) for item in data.get("out_of_bounds_placements", [])],\n            empty_layouts=[str(item) for item in data.get("empty_layouts", [])],\n            documentation_complete=bool(data.get("documentation_complete", False)),\n        )\n\n\nclass OperationsAnalytics:\n    METRIC_DEFINITIONS = (\n        OperationsMetricDefinition(\n            metric_id="runs-today",\n            label="Runs Today",\n            unit="runs",\n            direction="up",\n            formula="count(run.started_at within [period_start, period_end])",\n            description="Number of runs that started inside the reporting day window.",\n            source_fields=["started_at"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="avg-lead-time",\n            label="Avg Lead Time",\n            unit="m",\n            direction="down",\n            formula="sum(cycle_minutes for runs with started_at and ended_at) / measured_runs",\n            description="Average elapsed minutes from run start to run end for runs with complete timestamps.",\n            source_fields=["started_at", "ended_at"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="intervention-rate",\n            label="Intervention Rate",\n            unit="%",\n            direction="down",\n            formula="100 * actionable_runs / total_runs",\n            description="Share of runs that require operator intervention because they ended in an actionable status.",\n            source_fields=["status"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="sla",\n            label="SLA",\n            unit="%",\n            direction="up",\n            formula="100 * compliant_runs / measured_runs where compliant_runs have cycle_minutes <= sla_target_minutes",\n            description="Share of measured runs that met the SLA target.",\n            source_fields=["started_at", "ended_at"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="regression",\n            label="Regression",\n            unit="cases",\n            direction="down",\n            formula="count(current.compare(baseline) deltas < 0 or pass->fail transitions)",\n            description="Number of benchmark cases that regressed against the provided baseline suite.",\n            source_fields=["benchmark.current", "benchmark.baseline"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="risk",\n            label="Risk",\n            unit="score",\n            direction="down",\n            formula="sum(resolved_run_risk_score) / runs_with_risk where risk_score.total wins over risk_level mapping low=25, medium=60, high=90",\n            description="Average per-run risk score from explicit risk scores or normalized risk levels.",\n            source_fields=["risk_score.total", "risk_level"],\n        ),\n        OperationsMetricDefinition(\n            metric_id="spend",\n            label="Spend",\n            unit="USD",\n            direction="down",\n            formula="sum(first non-null of spend_usd, cost_usd, spend, cost across runs)",\n            description="Total reported run spend in USD over the reporting window.",\n            source_fields=["spend_usd", "cost_usd", "spend", "cost"],\n        ),\n    )\n\n    def summarize_runs(\n        self,\n        runs: Sequence[dict],\n        sla_target_minutes: int = 60,\n        top_n_blockers: int = 3,\n    ) -> OperationsSnapshot:\n        status_counts: Dict[str, int] = {}\n        total_cycle_minutes = 0.0\n        cycle_count = 0\n        completed = 0\n        approval_queue_depth = 0\n        sla_breach_count = 0\n\n        for run in runs:\n            status = str(run.get("status", "unknown"))\n            status_counts[status] = status_counts.get(status, 0) + 1\n\n            if status == "needs-approval":\n                approval_queue_depth += 1\n\n            cycle_minutes = self._cycle_minutes(run)\n            if cycle_minutes is not None:\n                total_cycle_minutes += cycle_minutes\n                cycle_count += 1\n                if cycle_minutes > sla_target_minutes:\n                    sla_breach_count += 1\n\n            if status in STATUS_COMPLETE:\n                completed += 1\n\n        success_rate = round((completed / len(runs)) * 100, 1) if runs else 0.0\n        average_cycle_minutes = round(total_cycle_minutes / cycle_count, 1) if cycle_count else 0.0\n        blockers = self.build_triage_clusters(runs)[:top_n_blockers]\n        return OperationsSnapshot(\n            total_runs=len(runs),\n            status_counts=status_counts,\n            success_rate=success_rate,\n            approval_queue_depth=approval_queue_depth,\n            sla_target_minutes=sla_target_minutes,\n            sla_breach_count=sla_breach_count,\n            average_cycle_minutes=average_cycle_minutes,\n            top_blockers=blockers,\n        )\n\n    def build_metric_spec(\n        self,\n        runs: Sequence[dict],\n        *,\n        period_start: str,\n        period_end: str,\n        timezone_name: str = "UTC",\n        generated_at: Optional[str] = None,\n        sla_target_minutes: int = 60,\n        current_suite: Optional[BenchmarkSuiteResult] = None,\n        baseline_suite: Optional[BenchmarkSuiteResult] = None,\n    ) -> OperationsMetricSpec:\n        period_start_dt = self._parse_ts(period_start)\n        period_end_dt = self._parse_ts(period_end)\n        if period_start_dt is None or period_end_dt is None or period_end_dt < period_start_dt:\n            raise ValueError("period_start and period_end must be valid ISO-8601 timestamps with period_end >= period_start")\n\n        runs_today = 0\n        lead_time_sum = 0.0\n        lead_time_count = 0\n        actionable_runs = 0\n        sla_compliant_runs = 0\n        risk_sum = 0.0\n        risk_count = 0\n        spend_total = 0.0\n\n        for run in runs:\n            started_at = self._parse_ts(str(run.get("started_at", "")))\n            if started_at is not None and period_start_dt <= started_at <= period_end_dt:\n                runs_today += 1\n\n            cycle_minutes = self._cycle_minutes(run)\n            if cycle_minutes is not None:\n                lead_time_sum += cycle_minutes\n                lead_time_count += 1\n                if cycle_minutes <= sla_target_minutes:\n                    sla_compliant_runs += 1\n\n            if str(run.get("status", "unknown")) in STATUS_ACTIONABLE:\n                actionable_runs += 1\n\n            risk_score = self._resolve_run_risk_score(run)\n            if risk_score is not None:\n                risk_sum += risk_score\n                risk_count += 1\n\n            spend_total += self._resolve_run_spend(run)\n\n        regression_findings = self.analyze_regressions(current_suite, baseline_suite) if current_suite is not None else []\n        total_runs = len(runs)\n        avg_lead = round(lead_time_sum / lead_time_count, 1) if lead_time_count else 0.0\n        intervention_rate = round((actionable_runs / total_runs) * 100, 1) if total_runs else 0.0\n        sla_value = round((sla_compliant_runs / lead_time_count) * 100, 1) if lead_time_count else 0.0\n        avg_risk = round(risk_sum / risk_count, 1) if risk_count else 0.0\n        spend_total = round(spend_total, 2)\n\n        values = [\n            OperationsMetricValue(\n                metric_id="runs-today",\n                label="Runs Today",\n                value=float(runs_today),\n                display_value=str(runs_today),\n                numerator=float(runs_today),\n                denominator=float(total_runs),\n                unit="runs",\n                evidence=[f"{runs_today} of {total_runs} runs started inside the reporting window."],\n            ),\n            OperationsMetricValue(\n                metric_id="avg-lead-time",\n                label="Avg Lead Time",\n                value=avg_lead,\n                display_value=f"{avg_lead:.1f}m",\n                numerator=round(lead_time_sum, 1),\n                denominator=float(lead_time_count),\n                unit="m",\n                evidence=[f"{lead_time_count} runs had valid start/end timestamps."],\n            ),\n            OperationsMetricValue(\n                metric_id="intervention-rate",\n                label="Intervention Rate",\n                value=intervention_rate,\n                display_value=f"{intervention_rate:.1f}%",\n                numerator=float(actionable_runs),\n                denominator=float(total_runs),\n                unit="%",\n                evidence=[f"Actionable statuses counted: {\', \'.join(sorted(STATUS_ACTIONABLE))}."],\n            ),\n            OperationsMetricValue(\n                metric_id="sla",\n                label="SLA",\n                value=sla_value,\n                display_value=f"{sla_value:.1f}%",\n                numerator=float(sla_compliant_runs),\n                denominator=float(lead_time_count),\n                unit="%",\n                evidence=[\n                    f"SLA target: {sla_target_minutes} minutes.",\n                    f"{sla_compliant_runs} of {lead_time_count} measured runs met target.",\n                ],\n            ),\n            OperationsMetricValue(\n                metric_id="regression",\n                label="Regression",\n                value=float(len(regression_findings)),\n                display_value=str(len(regression_findings)),\n                numerator=float(len(regression_findings)),\n                denominator=float(len(current_suite.results)) if current_suite is not None else 0.0,\n                unit="cases",\n                evidence=[\n                    f"Baseline provided: {baseline_suite is not None}.",\n                    f"Current suite provided: {current_suite is not None}.",\n                ],\n            ),\n            OperationsMetricValue(\n                metric_id="risk",\n                label="Risk",\n                value=avg_risk,\n                display_value=f"{avg_risk:.1f}",\n                numerator=round(risk_sum, 1),\n                denominator=float(risk_count),\n                unit="score",\n                evidence=["Risk score precedence: risk_score.total, then risk_level mapping low=25 medium=60 high=90."],\n            ),\n            OperationsMetricValue(\n                metric_id="spend",\n                label="Spend",\n                value=spend_total,\n                display_value=f"${spend_total:.2f}",\n                numerator=spend_total,\n                denominator=float(total_runs),\n                unit="USD",\n                evidence=["Spend field precedence: spend_usd, cost_usd, spend, cost."],\n            ),\n        ]\n\n        return OperationsMetricSpec(\n            name="Operations Metric Spec",\n            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),\n            period_start=period_start,\n            period_end=period_end,\n            timezone_name=timezone_name,\n            definitions=list(self.METRIC_DEFINITIONS),\n            values=values,\n        )\n\n    def build_triage_clusters(self, runs: Sequence[dict]) -> List[TriageCluster]:\n        clusters: Dict[str, TriageCluster] = {}\n        for run in runs:\n            status = str(run.get("status", "unknown"))\n            if status not in STATUS_ACTIONABLE:\n                continue\n\n            reason = self._primary_reason(run)\n            cluster = clusters.setdefault(reason, TriageCluster(reason=reason))\n            run_id = str(run.get("run_id", ""))\n            task_id = str(run.get("task_id", ""))\n            if run_id and run_id not in cluster.run_ids:\n                cluster.run_ids.append(run_id)\n            if task_id and task_id not in cluster.task_ids:\n                cluster.task_ids.append(task_id)\n            if status not in cluster.statuses:\n                cluster.statuses.append(status)\n\n        return sorted(\n            clusters.values(),\n            key=lambda cluster: (-cluster.occurrences, cluster.reason),\n        )\n\n    def analyze_regressions(\n        self,\n        current: BenchmarkSuiteResult,\n        baseline: Optional[BenchmarkSuiteResult] = None,\n    ) -> List[RegressionFinding]:\n        if baseline is None:\n            return []\n\n        baseline_results = {result.case_id: result for result in baseline.results}\n        findings: List[RegressionFinding] = []\n        for comparison in current.compare(baseline):\n            baseline_result = baseline_results.get(comparison.case_id)\n            current_result = next(result for result in current.results if result.case_id == comparison.case_id)\n            if comparison.delta >= 0 and not (baseline_result and baseline_result.passed and not current_result.passed):\n                continue\n\n            severity = "high" if comparison.delta <= -20 or (baseline_result and baseline_result.passed and not current_result.passed) else "medium"\n            summary = (\n                f"score dropped from {comparison.baseline_score} to {comparison.current_score}"\n                if comparison.delta < 0\n                else "case regressed from passing to failing"\n            )\n            findings.append(\n                RegressionFinding(\n                    case_id=comparison.case_id,\n                    baseline_score=comparison.baseline_score,\n                    current_score=comparison.current_score,\n                    delta=comparison.delta,\n                    severity=severity,\n                    summary=summary,\n                )\n            )\n\n        return sorted(findings, key=lambda finding: (finding.delta, finding.case_id))\n\n    def build_regression_center(\n        self,\n        current: BenchmarkSuiteResult,\n        baseline: BenchmarkSuiteResult,\n        name: str = "Regression Analysis Center",\n    ) -> RegressionCenter:\n        regressions = self.analyze_regressions(current, baseline)\n        comparisons = current.compare(baseline)\n        improved_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta > 0)\n        unchanged_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta == 0)\n        return RegressionCenter(\n            name=name,\n            baseline_version=baseline.version,\n            current_version=current.version,\n            regressions=regressions,\n            improved_cases=improved_cases,\n            unchanged_cases=unchanged_cases,\n        )\n\n    def build_queue_control_center(\n        self,\n        queue: PersistentTaskQueue,\n        runs: Sequence[dict],\n    ) -> QueueControlCenter:\n        queued_tasks = queue.peek_tasks()\n        queued_by_priority = {"P0": 0, "P1": 0, "P2": 0}\n        queued_by_risk = {"low": 0, "medium": 0, "high": 0}\n        for task in queued_tasks:\n            queued_by_priority[f"P{int(task.priority)}"] += 1\n            queued_by_risk[task.risk_level.value] += 1\n\n        execution_media: Dict[str, int] = {}\n        waiting_approval_runs = 0\n        blocked_tasks: List[str] = []\n        for run in runs:\n            medium = str(run.get("medium", "unknown"))\n            execution_media[medium] = execution_media.get(medium, 0) + 1\n            if run.get("status") == "needs-approval":\n                waiting_approval_runs += 1\n                task_id = str(run.get("task_id", ""))\n                if task_id and task_id not in blocked_tasks:\n                    blocked_tasks.append(task_id)\n\n        return QueueControlCenter(\n            queue_depth=queue.size(),\n            queued_by_priority=queued_by_priority,\n            queued_by_risk=queued_by_risk,\n            execution_media=execution_media,\n            waiting_approval_runs=waiting_approval_runs,\n            blocked_tasks=blocked_tasks,\n            queued_tasks=[task.task_id for task in queued_tasks],\n            actions={\n                task.task_id: build_console_actions(\n                    task.task_id,\n                    allow_retry=task.task_id in blocked_tasks,\n                    retry_reason="" if task.task_id in blocked_tasks else "retry is reserved for blocked queue items",\n                    allow_pause=task.task_id not in blocked_tasks,\n                    pause_reason="" if task.task_id not in blocked_tasks else "approval-blocked tasks should be escalated instead of paused",\n                    allow_escalate=task.task_id in blocked_tasks,\n                    escalate_reason="" if task.task_id in blocked_tasks else "escalate is reserved for blocked queue items",\n                )\n                for task in queued_tasks\n            },\n        )\n\n    def build_policy_prompt_version_center(\n        self,\n        artifacts: Sequence[VersionedArtifact],\n        name: str = "Policy/Prompt Version Center",\n        generated_at: Optional[str] = None,\n        diff_preview_lines: int = 8,\n    ) -> PolicyPromptVersionCenter:\n        grouped: Dict[tuple[str, str], List[VersionedArtifact]] = {}\n        for artifact in artifacts:\n            key = (artifact.artifact_type, artifact.artifact_id)\n            grouped.setdefault(key, []).append(artifact)\n\n        histories: List[VersionedArtifactHistory] = []\n        for artifact_type, artifact_id in sorted(grouped.keys()):\n            revisions = sorted(\n                grouped[(artifact_type, artifact_id)],\n                key=lambda artifact: self._parse_ts(artifact.updated_at) or datetime.min.replace(tzinfo=timezone.utc),\n                reverse=True,\n            )\n            current = revisions[0]\n            previous = revisions[1] if len(revisions) > 1 else None\n            change_summary = None\n            rollback_version = None\n            rollback_ready = False\n\n            if previous is not None:\n                change_summary = self._summarize_version_change(previous, current, preview_lines=diff_preview_lines)\n                rollback_version = previous.version\n                rollback_ready = bool(previous.content.strip())\n\n            histories.append(\n                VersionedArtifactHistory(\n                    artifact_type=artifact_type,\n                    artifact_id=artifact_id,\n                    current_version=current.version,\n                    current_updated_at=current.updated_at,\n                    current_author=current.author,\n                    current_summary=current.summary,\n                    revision_count=len(revisions),\n                    revisions=revisions,\n                    rollback_version=rollback_version,\n                    rollback_ready=rollback_ready,\n                    change_summary=change_summary,\n                )\n            )\n\n        return PolicyPromptVersionCenter(\n            name=name,\n            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),\n            histories=histories,\n        )\n\n    def build_engineering_overview(\n        self,\n        name: str,\n        period: str,\n        runs: Sequence[dict],\n        viewer_role: str,\n        sla_target_minutes: int = 60,\n        top_n_blockers: int = 3,\n        recent_activity_limit: int = 5,\n    ) -> EngineeringOverview:\n        snapshot = self.summarize_runs(\n            runs,\n            sla_target_minutes=sla_target_minutes,\n            top_n_blockers=top_n_blockers,\n        )\n        permissions = self._permissions_for_role(viewer_role)\n        kpis = [\n            EngineeringOverviewKPI(name="success-rate", value=snapshot.success_rate, target=90.0, unit="%"),\n            EngineeringOverviewKPI(\n                name="approval-queue-depth",\n                value=float(snapshot.approval_queue_depth),\n                target=2.0,\n                direction="down",\n            ),\n            EngineeringOverviewKPI(\n                name="sla-breaches",\n                value=float(snapshot.sla_breach_count),\n                target=0.0,\n                direction="down",\n            ),\n            EngineeringOverviewKPI(\n                name="average-cycle-minutes",\n                value=snapshot.average_cycle_minutes,\n                target=float(sla_target_minutes),\n                unit="m",\n                direction="down",\n            ),\n        ]\n        blockers = [\n            EngineeringOverviewBlocker(\n                summary=cluster.reason,\n                affected_runs=cluster.occurrences,\n                affected_tasks=cluster.task_ids,\n                owner=self._owner_for_cluster(cluster),\n                severity=self._severity_for_cluster(cluster),\n            )\n            for cluster in snapshot.top_blockers\n        ]\n        return EngineeringOverview(\n            name=name,\n            period=period,\n            snapshot=snapshot,\n            permissions=permissions,\n            kpis=kpis,\n            funnel=self._build_funnel(snapshot.status_counts, snapshot.total_runs),\n            blockers=blockers,\n            activities=self._build_recent_activities(runs, recent_activity_limit),\n        )\n\n    def build_weekly_report(\n        self,\n        name: str,\n        period: str,\n        runs: Sequence[dict],\n        current_suite: Optional[BenchmarkSuiteResult] = None,\n        baseline_suite: Optional[BenchmarkSuiteResult] = None,\n        sla_target_minutes: int = 60,\n    ) -> WeeklyOperationsReport:\n        snapshot = self.summarize_runs(runs, sla_target_minutes=sla_target_minutes)\n        regressions = []\n        if current_suite is not None:\n            regressions = self.analyze_regressions(current_suite, baseline_suite)\n        return WeeklyOperationsReport(\n            name=name,\n            period=period,\n            snapshot=snapshot,\n            regressions=regressions,\n        )\n\n    def build_dashboard_builder(\n        self,\n        name: str,\n        period: str,\n        owner: str,\n        viewer_role: str,\n        widgets: Sequence[DashboardWidgetSpec],\n        layouts: Sequence[DashboardLayout],\n        documentation_complete: bool = False,\n    ) -> DashboardBuilder:\n        return DashboardBuilder(\n            name=name,\n            period=period,\n            owner=owner,\n            permissions=self._permissions_for_role(viewer_role),\n            widgets=list(widgets),\n            layouts=[self.normalize_dashboard_layout(layout, widgets) for layout in layouts],\n            documentation_complete=documentation_complete,\n        )\n\n    def normalize_dashboard_layout(\n        self,\n        layout: DashboardLayout,\n        widgets: Sequence[DashboardWidgetSpec],\n    ) -> DashboardLayout:\n        widget_index = {widget.widget_id: widget for widget in widgets}\n        normalized: List[DashboardWidgetPlacement] = []\n        column_count = max(1, layout.columns)\n        for placement in layout.placements:\n            spec = widget_index.get(placement.widget_id)\n            min_width = spec.min_width if spec is not None else 1\n            max_width = min(spec.max_width, column_count) if spec is not None else column_count\n            width = max(min_width, min(placement.width, max_width))\n            column = max(0, placement.column)\n            if column + width > column_count:\n                column = max(0, column_count - width)\n            normalized.append(\n                DashboardWidgetPlacement(\n                    placement_id=placement.placement_id,\n                    widget_id=placement.widget_id,\n                    column=column,\n                    row=max(0, placement.row),\n                    width=width,\n                    height=max(1, placement.height),\n                    title_override=placement.title_override,\n                    filters=list(placement.filters),\n                )\n            )\n\n        normalized.sort(key=lambda item: (item.row, item.column, item.placement_id))\n        return DashboardLayout(\n            layout_id=layout.layout_id,\n            name=layout.name,\n            columns=column_count,\n            placements=normalized,\n        )\n\n    def audit_dashboard_builder(self, dashboard: DashboardBuilder) -> DashboardBuilderAudit:\n        widget_index = dashboard.widget_index\n        placement_counts: Dict[str, int] = {}\n        missing_widget_defs: set[str] = set()\n        inaccessible_widgets: set[str] = set()\n        overlapping_placements: set[str] = set()\n        out_of_bounds_placements: set[str] = set()\n        empty_layouts: List[str] = []\n        placed_widgets = 0\n\n        for layout in dashboard.layouts:\n            if not layout.placements:\n                empty_layouts.append(layout.layout_id)\n                continue\n\n            placed_widgets += len(layout.placements)\n            for placement in layout.placements:\n                placement_counts[placement.placement_id] = placement_counts.get(placement.placement_id, 0) + 1\n                spec = widget_index.get(placement.widget_id)\n                if spec is None:\n                    missing_widget_defs.add(placement.widget_id)\n                else:\n                    if not dashboard.permissions.can_view(spec.module):\n                        inaccessible_widgets.add(placement.widget_id)\n                if placement.column + placement.width > layout.columns:\n                    out_of_bounds_placements.add(placement.placement_id)\n\n            for index, placement in enumerate(layout.placements):\n                for other in layout.placements[index + 1 :]:\n                    if self._placements_overlap(placement, other):\n                        overlapping_placements.add(\n                            f"{layout.layout_id}:{placement.placement_id}<->{other.placement_id}"\n                        )\n\n        duplicate_ids = sorted(\n            placement_id for placement_id, count in placement_counts.items() if count > 1\n        )\n        return DashboardBuilderAudit(\n            name=dashboard.name,\n            total_widgets=len(dashboard.widgets),\n            layout_count=len(dashboard.layouts),\n            placed_widgets=placed_widgets,\n            duplicate_placement_ids=duplicate_ids,\n            missing_widget_defs=sorted(missing_widget_defs),\n            inaccessible_widgets=sorted(inaccessible_widgets),\n            overlapping_placements=sorted(overlapping_placements),\n            out_of_bounds_placements=sorted(out_of_bounds_placements),\n            empty_layouts=sorted(empty_layouts),\n            documentation_complete=dashboard.documentation_complete,\n        )\n\n    def _primary_reason(self, run: dict) -> str:\n        for audit in run.get("audits", []):\n            reason = audit.get("details", {}).get("reason")\n            if reason:\n                return str(reason)\n        summary = str(run.get("summary", "")).strip()\n        if summary:\n            return summary\n        return str(run.get("status", "unknown"))\n\n    def _cycle_minutes(self, run: dict) -> Optional[float]:\n        started_at = run.get("started_at")\n        ended_at = run.get("ended_at")\n        if not started_at or not ended_at:\n            return None\n        start = self._parse_ts(str(started_at))\n        end = self._parse_ts(str(ended_at))\n        if start is None or end is None or end < start:\n            return None\n        return round((end - start).total_seconds() / 60, 1)\n\n    def _parse_ts(self, value: str) -> Optional[datetime]:\n        try:\n            return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)\n        except ValueError:\n            return None\n\n    def _resolve_run_risk_score(self, run: dict) -> Optional[float]:\n        risk_score = run.get("risk_score")\n        if isinstance(risk_score, dict) and risk_score.get("total") is not None:\n            try:\n                return float(risk_score["total"])\n            except (TypeError, ValueError):\n                return None\n\n        risk_level = str(run.get("risk_level", "")).strip().lower()\n        risk_by_level = {"low": 25.0, "medium": 60.0, "high": 90.0}\n        return risk_by_level.get(risk_level)\n\n    def _resolve_run_spend(self, run: dict) -> float:\n        for key in ("spend_usd", "cost_usd", "spend", "cost"):\n            value = run.get(key)\n            if value is None:\n                continue\n            try:\n                return float(value)\n            except (TypeError, ValueError):\n                return 0.0\n        return 0.0\n\n    def _summarize_version_change(\n        self,\n        previous: VersionedArtifact,\n        current: VersionedArtifact,\n        preview_lines: int,\n    ) -> VersionChangeSummary:\n        diff_lines = list(\n            unified_diff(\n                previous.content.splitlines(),\n                current.content.splitlines(),\n                fromfile=previous.version,\n                tofile=current.version,\n                lineterm="",\n            )\n        )\n        additions = sum(1 for line in diff_lines if line.startswith("+") and not line.startswith("+++"))\n        deletions = sum(1 for line in diff_lines if line.startswith("-") and not line.startswith("---"))\n        preview = [line for line in diff_lines if not line.startswith("@@")][:preview_lines]\n        return VersionChangeSummary(\n            from_version=previous.version,\n            to_version=current.version,\n            additions=additions,\n            deletions=deletions,\n            changed_lines=additions + deletions,\n            preview=preview,\n        )\n\n    def _build_funnel(self, status_counts: Dict[str, int], total_runs: int) -> List[EngineeringFunnelStage]:\n        funnel_counts = [\n            ("queued", status_counts.get("queued", 0)),\n            ("in-progress", status_counts.get("running", 0) + status_counts.get("in-progress", 0)),\n            ("awaiting-approval", status_counts.get("needs-approval", 0)),\n            ("completed", sum(count for status, count in status_counts.items() if status in STATUS_COMPLETE)),\n        ]\n        return [\n            EngineeringFunnelStage(\n                name=name,\n                count=count,\n                share=round((count / total_runs) * 100, 1) if total_runs else 0.0,\n            )\n            for name, count in funnel_counts\n        ]\n\n    def _build_recent_activities(self, runs: Sequence[dict], limit: int) -> List[EngineeringActivity]:\n        dated_runs = []\n        for run in runs:\n            sort_key = self._parse_ts(str(run.get("ended_at", ""))) or self._parse_ts(str(run.get("started_at", "")))\n            if sort_key is None:\n                continue\n            dated_runs.append((sort_key, run))\n\n        activities: List[EngineeringActivity] = []\n        for _, run in sorted(dated_runs, key=lambda item: item[0], reverse=True)[:limit]:\n            activities.append(\n                EngineeringActivity(\n                    timestamp=str(run.get("ended_at") or run.get("started_at") or ""),\n                    run_id=str(run.get("run_id", "")),\n                    task_id=str(run.get("task_id", "")),\n                    status=str(run.get("status", "unknown")),\n                    summary=self._primary_reason(run),\n                )\n            )\n        return activities\n\n    def _permissions_for_role(self, viewer_role: str) -> EngineeringOverviewPermission:\n        role = viewer_role.strip().lower() or "contributor"\n        modules_by_role = {\n            "executive": ["kpis", "funnel", "blockers"],\n            "engineering-manager": ["kpis", "funnel", "blockers", "activity"],\n            "operations": ["kpis", "funnel", "blockers", "activity"],\n            "contributor": ["kpis", "activity"],\n        }\n        return EngineeringOverviewPermission(\n            viewer_role=role,\n            allowed_modules=modules_by_role.get(role, modules_by_role["contributor"]),\n        )\n\n    def _owner_for_cluster(self, cluster: TriageCluster) -> str:\n        details = " ".join([cluster.reason, " ".join(cluster.statuses)]).lower()\n        if "approval" in details:\n            return "operations"\n        if "security" in details:\n            return "security"\n        return "engineering"\n\n    def _severity_for_cluster(self, cluster: TriageCluster) -> str:\n        if cluster.occurrences >= 3 or "failed" in cluster.statuses:\n            return "high"\n        return "medium"\n\n    @staticmethod\n    def _placements_overlap(left: DashboardWidgetPlacement, right: DashboardWidgetPlacement) -> bool:\n        return not (\n            left.column + left.width <= right.column\n            or right.column + right.width <= left.column\n            or left.row + left.height <= right.row\n            or right.row + right.height <= left.row\n        )\n\n\ndef render_operations_dashboard(\n    snapshot: OperationsSnapshot,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    lines = [\n        "# Operations Dashboard",\n        "",\n        f"- Total Runs: {snapshot.total_runs}",\n        f"- Success Rate: {snapshot.success_rate:.1f}%",\n        f"- Approval Queue Depth: {snapshot.approval_queue_depth}",\n        f"- SLA Target: {snapshot.sla_target_minutes} minutes",\n        f"- SLA Breaches: {snapshot.sla_breach_count}",\n        f"- Average Cycle Time: {snapshot.average_cycle_minutes:.1f} minutes",\n        "",\n        "## Status Counts",\n        "",\n    ]\n    lines.extend(render_shared_view_context(view))\n\n    if snapshot.status_counts:\n        for status, count in sorted(snapshot.status_counts.items()):\n            lines.append(f"- {status}: {count}")\n    else:\n        lines.append("- None")\n\n    lines.extend(["", "## Top Blockers", ""])\n    if snapshot.top_blockers:\n        for cluster in snapshot.top_blockers:\n            statuses = ", ".join(cluster.statuses) if cluster.statuses else "unknown"\n            lines.append(\n                f"- {cluster.reason}: occurrences={cluster.occurrences} statuses={statuses} tasks={\', \'.join(cluster.task_ids)}"\n            )\n    else:\n        lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_weekly_operations_report(report: WeeklyOperationsReport) -> str:\n    lines = [\n        "# Weekly Operations Report",\n        "",\n        f"- Name: {report.name}",\n        f"- Period: {report.period}",\n        f"- Total Runs: {report.snapshot.total_runs}",\n        f"- Success Rate: {report.snapshot.success_rate:.1f}%",\n        f"- SLA Breaches: {report.snapshot.sla_breach_count}",\n        f"- Approval Queue Depth: {report.snapshot.approval_queue_depth}",\n        "",\n        "## Blockers",\n        "",\n    ]\n\n    if report.snapshot.top_blockers:\n        for cluster in report.snapshot.top_blockers:\n            lines.append(f"- {cluster.reason}: {cluster.occurrences} runs")\n    else:\n        lines.append("- None")\n\n    lines.extend(["", "## Regressions", ""])\n    if report.regressions:\n        for finding in report.regressions:\n            lines.append(\n                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"\n            )\n    else:\n        lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_operations_metric_spec(spec: OperationsMetricSpec) -> str:\n    lines = [\n        "# Operations Metric Spec",\n        "",\n        f"- Name: {spec.name}",\n        f"- Generated At: {spec.generated_at}",\n        f"- Period Start: {spec.period_start}",\n        f"- Period End: {spec.period_end}",\n        f"- Timezone: {spec.timezone_name}",\n        "",\n        "## Definitions",\n        "",\n    ]\n\n    for definition in spec.definitions:\n        lines.extend(\n            [\n                f"### {definition.label}",\n                "",\n                f"- Metric ID: {definition.metric_id}",\n                f"- Unit: {definition.unit}",\n                f"- Direction: {definition.direction}",\n                f"- Formula: {definition.formula}",\n                f"- Description: {definition.description}",\n                f"- Source Fields: {\', \'.join(definition.source_fields)}",\n                "",\n            ]\n        )\n\n    lines.extend(["## Values", ""])\n    for value in spec.values:\n        evidence = " | ".join(value.evidence) if value.evidence else "none"\n        lines.append(\n            f"- {value.label}: value={value.display_value} numerator={value.numerator:.1f} "\n            f"denominator={value.denominator:.1f} unit={value.unit} evidence={evidence}"\n        )\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_queue_control_center(\n    center: QueueControlCenter,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    lines = [\n        "# Queue Control Center",\n        "",\n        f"- Queue Depth: {center.queue_depth}",\n        f"- Waiting Approval Runs: {center.waiting_approval_runs}",\n        f"- Queued Tasks: {\', \'.join(center.queued_tasks) if center.queued_tasks else \'none\'}",\n        "",\n        "## Queue By Priority",\n        "",\n    ]\n    lines.extend(render_shared_view_context(view))\n\n    for priority, count in center.queued_by_priority.items():\n        lines.append(f"- {priority}: {count}")\n\n    lines.extend(["", "## Queue By Risk", ""])\n    for risk_level, count in center.queued_by_risk.items():\n        lines.append(f"- {risk_level}: {count}")\n\n    lines.extend(["", "## Execution Media", ""])\n    if center.execution_media:\n        for medium, count in sorted(center.execution_media.items()):\n            lines.append(f"- {medium}: {count}")\n    else:\n        lines.append("- None")\n\n    lines.extend(["", "## Blocked Tasks", ""])\n    if center.blocked_tasks:\n        for task_id in center.blocked_tasks:\n            lines.append(f"- {task_id}")\n    else:\n        lines.append("- None")\n\n    lines.extend(["", "## Actions", ""])\n    if center.actions:\n        for task_id in center.queued_tasks:\n            actions = center.actions.get(task_id, [])\n            lines.append(f"- {task_id}: {render_console_actions(actions)}")\n    else:\n        lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_policy_prompt_version_center(\n    center: PolicyPromptVersionCenter,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    lines = [\n        "# Policy/Prompt Version Center",\n        "",\n        f"- Name: {center.name}",\n        f"- Generated At: {center.generated_at}",\n        f"- Versioned Artifacts: {center.artifact_count}",\n        f"- Rollback Ready Artifacts: {center.rollback_ready_count}",\n        "",\n        "## Artifact Histories",\n        "",\n    ]\n    lines.extend(render_shared_view_context(view))\n\n    if not center.histories:\n        lines.append("- None")\n        return "\\n".join(lines) + "\\n"\n\n    for history in center.histories:\n        lines.extend(\n            [\n                f"### {history.artifact_type} / {history.artifact_id}",\n                "",\n                f"- Current Version: {history.current_version}",\n                f"- Updated At: {history.current_updated_at}",\n                f"- Updated By: {history.current_author}",\n                f"- Summary: {history.current_summary}",\n                f"- Revision Count: {history.revision_count}",\n                f"- Rollback Version: {history.rollback_version or \'none\'}",\n                f"- Rollback Ready: {history.rollback_ready}",\n            ]\n        )\n        if history.change_summary is not None:\n            lines.append(\n                f"- Diff Summary: {history.change_summary.additions} additions, "\n                f"{history.change_summary.deletions} deletions"\n            )\n        lines.extend(["", "#### Revision History", ""])\n        for revision in history.revisions:\n            ticket = revision.change_ticket or "none"\n            lines.append(\n                f"- {revision.version}: updated_at={revision.updated_at} author={revision.author} "\n                f"ticket={ticket} summary={revision.summary}"\n            )\n        lines.extend(["", "#### Diff Preview", ""])\n        if history.change_summary is not None and history.change_summary.preview:\n            lines.append("```diff")\n            lines.extend(history.change_summary.preview)\n            lines.append("```")\n        else:\n            lines.append("- None")\n        lines.append("")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_engineering_overview(overview: EngineeringOverview) -> str:\n    lines = [\n        "# Engineering Overview",\n        "",\n        f"- Name: {overview.name}",\n        f"- Period: {overview.period}",\n        f"- Viewer Role: {overview.permissions.viewer_role}",\n        f"- Visible Modules: {\', \'.join(overview.permissions.allowed_modules)}",\n    ]\n\n    if overview.permissions.can_view("kpis"):\n        lines.extend(["", "## KPI Modules", ""])\n        for kpi in overview.kpis:\n            lines.append(\n                f"- {kpi.name}: value={kpi.value:.1f}{kpi.unit} target={kpi.target:.1f}{kpi.unit} healthy={kpi.healthy}"\n            )\n\n    if overview.permissions.can_view("funnel"):\n        lines.extend(["", "## Funnel Modules", ""])\n        for stage in overview.funnel:\n            lines.append(f"- {stage.name}: count={stage.count} share={stage.share:.1f}%")\n\n    if overview.permissions.can_view("blockers"):\n        lines.extend(["", "## Blocker Modules", ""])\n        if overview.blockers:\n            for blocker in overview.blockers:\n                lines.append(\n                    f"- {blocker.summary}: severity={blocker.severity} owner={blocker.owner} "\n                    f"affected_runs={blocker.affected_runs} tasks={\', \'.join(blocker.affected_tasks)}"\n                )\n        else:\n            lines.append("- None")\n\n    if overview.permissions.can_view("activity"):\n        lines.extend(["", "## Activity Modules", ""])\n        if overview.activities:\n            for activity in overview.activities:\n                lines.append(\n                    f"- {activity.timestamp}: {activity.run_id} task={activity.task_id} "\n                    f"status={activity.status} summary={activity.summary}"\n                )\n        else:\n            lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef render_dashboard_builder_report(\n    dashboard: DashboardBuilder,\n    audit: DashboardBuilderAudit,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    lines = [\n        "# Dashboard Builder",\n        "",\n        f"- Name: {dashboard.name}",\n        f"- Period: {dashboard.period}",\n        f"- Owner: {dashboard.owner}",\n        f"- Viewer Role: {dashboard.permissions.viewer_role}",\n        f"- Available Widgets: {len(dashboard.widgets)}",\n        f"- Layouts: {len(dashboard.layouts)}",\n        f"- Release Ready: {audit.release_ready}",\n        "",\n        "## Governance",\n        "",\n        f"- Documentation Complete: {audit.documentation_complete}",\n        f"- Duplicate Placement IDs: {\', \'.join(audit.duplicate_placement_ids) if audit.duplicate_placement_ids else \'none\'}",\n        f"- Missing Widget Definitions: {\', \'.join(audit.missing_widget_defs) if audit.missing_widget_defs else \'none\'}",\n        f"- Inaccessible Widgets: {\', \'.join(audit.inaccessible_widgets) if audit.inaccessible_widgets else \'none\'}",\n        f"- Overlaps: {\', \'.join(audit.overlapping_placements) if audit.overlapping_placements else \'none\'}",\n        f"- Out Of Bounds: {\', \'.join(audit.out_of_bounds_placements) if audit.out_of_bounds_placements else \'none\'}",\n        f"- Empty Layouts: {\', \'.join(audit.empty_layouts) if audit.empty_layouts else \'none\'}",\n        "",\n        "## Layouts",\n        "",\n    ]\n    lines.extend(render_shared_view_context(view))\n\n    if dashboard.layouts:\n        for layout in dashboard.layouts:\n            lines.append(f"- {layout.layout_id}: name={layout.name} columns={layout.columns} placements={len(layout.placements)}")\n            for placement in layout.placements:\n                widget = dashboard.widget_index.get(placement.widget_id)\n                title = placement.title_override or (widget.title if widget is not None else placement.widget_id)\n                filters = ", ".join(placement.filters) if placement.filters else "none"\n                lines.append(\n                    f"- {placement.placement_id}: widget={placement.widget_id} title={title} "\n                    f"grid=({placement.column},{placement.row}) size={placement.width}x{placement.height} filters={filters}"\n                )\n    else:\n        lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n\n\ndef write_engineering_overview_bundle(root_dir: str, overview: EngineeringOverview) -> str:\n    base = Path(root_dir)\n    base.mkdir(parents=True, exist_ok=True)\n    overview_path = str(base / "engineering-overview.md")\n    write_report(overview_path, render_engineering_overview(overview))\n    return overview_path\n\n\ndef write_dashboard_builder_bundle(\n    root_dir: str,\n    dashboard: DashboardBuilder,\n    audit: DashboardBuilderAudit,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    base = Path(root_dir)\n    base.mkdir(parents=True, exist_ok=True)\n    dashboard_path = str(base / "dashboard-builder.md")\n    write_report(dashboard_path, render_dashboard_builder_report(dashboard, audit, view=view))\n    return dashboard_path\n\n\n\n\ndef build_repo_collaboration_metrics(runs: Sequence[dict]) -> Dict[str, float]:\n    total = len(runs)\n    linked = 0\n    accepted = 0\n    discussion_posts = 0\n    lineage_depth_sum = 0\n    lineage_depth_count = 0\n\n    for run in runs:\n        links = run.get("closeout", {}).get("run_commit_links", [])\n        if links:\n            linked += 1\n        if run.get("closeout", {}).get("accepted_commit_hash"):\n            accepted += 1\n        discussion_posts += int(run.get("repo_discussion_posts", 0))\n\n        depth = run.get("accepted_lineage_depth")\n        if depth is not None:\n            lineage_depth_sum += float(depth)\n            lineage_depth_count += 1\n\n    return {\n        "repo_link_coverage": round((linked / total) * 100, 1) if total else 0.0,\n        "accepted_commit_rate": round((accepted / total) * 100, 1) if total else 0.0,\n        "discussion_density": round(discussion_posts / total, 2) if total else 0.0,\n        "accepted_lineage_depth_avg": round(lineage_depth_sum / lineage_depth_count, 2) if lineage_depth_count else 0.0,\n    }\n\n\ndef write_weekly_operations_bundle(\n    root_dir: str,\n    report: WeeklyOperationsReport,\n    metric_spec: Optional[OperationsMetricSpec] = None,\n    regression_center: Optional[RegressionCenter] = None,\n    queue_control_center: Optional[QueueControlCenter] = None,\n    version_center: Optional[PolicyPromptVersionCenter] = None,\n) -> WeeklyOperationsArtifacts:\n    base = Path(root_dir)\n    base.mkdir(parents=True, exist_ok=True)\n\n    weekly_report_path = str(base / "weekly-operations.md")\n    dashboard_path = str(base / "operations-dashboard.md")\n    write_report(weekly_report_path, render_weekly_operations_report(report))\n    write_report(dashboard_path, render_operations_dashboard(report.snapshot))\n\n    metric_spec_path = None\n    if metric_spec is not None:\n        metric_spec_path = str(base / "operations-metric-spec.md")\n        write_report(metric_spec_path, render_operations_metric_spec(metric_spec))\n\n    regression_center_path = None\n    if regression_center is not None:\n        regression_center_path = str(base / "regression-center.md")\n        write_report(regression_center_path, render_regression_center(regression_center))\n\n    queue_control_path = None\n    if queue_control_center is not None:\n        queue_control_path = str(base / "queue-control-center.md")\n        write_report(queue_control_path, render_queue_control_center(queue_control_center))\n\n    version_center_path = None\n    if version_center is not None:\n        version_center_path = str(base / "policy-prompt-version-center.md")\n        write_report(version_center_path, render_policy_prompt_version_center(version_center))\n\n    return WeeklyOperationsArtifacts(\n        root_dir=str(base),\n        weekly_report_path=weekly_report_path,\n        dashboard_path=dashboard_path,\n        metric_spec_path=metric_spec_path,\n        regression_center_path=regression_center_path,\n        queue_control_path=queue_control_path,\n        version_center_path=version_center_path,\n    )\n\n\ndef render_regression_center(\n    center: RegressionCenter,\n    view: Optional[SharedViewContext] = None,\n) -> str:\n    lines = [\n        "# Regression Analysis Center",\n        "",\n        f"- Name: {center.name}",\n        f"- Baseline Version: {center.baseline_version}",\n        f"- Current Version: {center.current_version}",\n        f"- Regressions: {center.regression_count}",\n        f"- Improved Cases: {len(center.improved_cases)}",\n        f"- Unchanged Cases: {len(center.unchanged_cases)}",\n        "",\n        "## Regressions",\n        "",\n    ]\n    lines.extend(render_shared_view_context(view))\n\n    if center.regressions:\n        for finding in center.regressions:\n            lines.append(\n                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"\n            )\n    else:\n        lines.append("- None")\n\n    lines.extend(["", "## Improved Cases", ""])\n    if center.improved_cases:\n        for case_id in center.improved_cases:\n            lines.append(f"- {case_id}")\n    else:\n        lines.append("- None")\n\n    return "\\n".join(lines) + "\\n"\n',
    _operations_module.__dict__,
)
_operations_module.GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/reporting/reporting.go"
globals()["operations"] = _operations_module

from .operations import (
    DashboardBuilder,
    DashboardBuilderAudit,
    DashboardLayout,
    DashboardWidgetPlacement,
    DashboardWidgetSpec,
    EngineeringActivity,
    EngineeringFunnelStage,
    EngineeringOverview,
    EngineeringOverviewBlocker,
    EngineeringOverviewKPI,
    EngineeringOverviewPermission,
    OperationsAnalytics,
    OperationsMetricDefinition,
    OperationsMetricSpec,
    OperationsMetricValue,
    OperationsSnapshot,
    PolicyPromptVersionCenter,
    RegressionFinding,
    RegressionCenter,
    TriageCluster,
    QueueControlCenter,
    VersionChangeSummary,
    VersionedArtifact,
    VersionedArtifactHistory,
    WeeklyOperationsArtifacts,
    WeeklyOperationsReport,
    render_dashboard_builder_report,
    render_engineering_overview,
    render_operations_metric_spec,
    render_operations_dashboard,
    render_policy_prompt_version_center,
    render_queue_control_center,
    render_regression_center,
    render_weekly_operations_report,
    write_dashboard_builder_bundle,
    write_engineering_overview_bundle,
    write_weekly_operations_bundle,
)
from .evaluation import (
    BenchmarkCase,
    BenchmarkComparison,
    BenchmarkResult,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
    render_run_replay_index_page,
    render_replay_detail_page,
    render_benchmark_suite_report,
)
from .planning import (
    FourWeekExecutionPlan,
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EvidenceLink,
    EntryGate,
    EntryGateDecision,
    WeeklyExecutionPlan,
    WeeklyGoal,
    build_big_4701_execution_plan,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
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
    "APPROVAL_RECORDED_EVENT",
    "BUDGET_OVERRIDE_EVENT",
    "FLOW_HANDOFF_EVENT",
    "MANUAL_TAKEOVER_EVENT",
    "P0_AUDIT_EVENT_SPECS",
    "SCHEDULER_DECISION_EVENT",
    "AuditEventSpec",
    "get_audit_event_spec",
    "missing_required_fields",
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
