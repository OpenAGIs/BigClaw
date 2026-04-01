from __future__ import annotations

import sys
import types
import shutil
import hashlib
import json
import stat
import subprocess
import warnings
import argparse
from collections import defaultdict
from pathlib import Path
from datetime import datetime, timezone
from difflib import SequenceMatcher, unified_diff
from html import escape
from dataclasses import asdict, dataclass, field
from enum import Enum
from typing import Any, Callable, DefaultDict, Dict, Iterable, List, Optional, Protocol, Sequence, Set, Tuple
from urllib.parse import urlparse


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
        from .support_surfaces import CollaborationComment

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
        from .support_surfaces import DecisionNote

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


STATUS_COMPLETE = {"approved", "accepted", "completed", "succeeded"}
STATUS_ACTIONABLE = {"needs-approval", "failed", "rejected"}


@dataclass
class TriageCluster:
    reason: str
    run_ids: List[str] = field(default_factory=list)
    task_ids: List[str] = field(default_factory=list)
    statuses: List[str] = field(default_factory=list)

    @property
    def occurrences(self) -> int:
        return len(self.run_ids)


@dataclass
class RegressionFinding:
    case_id: str
    baseline_score: int
    current_score: int
    delta: int
    severity: str
    summary: str


@dataclass
class OperationsSnapshot:
    total_runs: int
    status_counts: Dict[str, int]
    success_rate: float
    approval_queue_depth: int
    sla_target_minutes: int
    sla_breach_count: int
    average_cycle_minutes: float
    top_blockers: List[TriageCluster] = field(default_factory=list)


@dataclass
class WeeklyOperationsReport:
    name: str
    period: str
    snapshot: OperationsSnapshot
    regressions: List[RegressionFinding] = field(default_factory=list)


@dataclass
class RegressionCenter:
    name: str
    baseline_version: str
    current_version: str
    regressions: List[RegressionFinding] = field(default_factory=list)
    improved_cases: List[str] = field(default_factory=list)
    unchanged_cases: List[str] = field(default_factory=list)

    @property
    def regression_count(self) -> int:
        return len(self.regressions)


@dataclass
class VersionedArtifact:
    artifact_type: str
    artifact_id: str
    version: str
    updated_at: str
    author: str
    summary: str
    content: str
    change_ticket: Optional[str] = None


@dataclass
class VersionChangeSummary:
    from_version: str
    to_version: str
    additions: int
    deletions: int
    changed_lines: int
    preview: List[str] = field(default_factory=list)

    @property
    def has_changes(self) -> bool:
        return self.changed_lines > 0


@dataclass
class VersionedArtifactHistory:
    artifact_type: str
    artifact_id: str
    current_version: str
    current_updated_at: str
    current_author: str
    current_summary: str
    revision_count: int
    revisions: List[VersionedArtifact] = field(default_factory=list)
    rollback_version: Optional[str] = None
    rollback_ready: bool = False
    change_summary: Optional[VersionChangeSummary] = None


@dataclass
class PolicyPromptVersionCenter:
    name: str
    generated_at: str
    histories: List[VersionedArtifactHistory] = field(default_factory=list)

    @property
    def artifact_count(self) -> int:
        return len(self.histories)

    @property
    def rollback_ready_count(self) -> int:
        return sum(1 for history in self.histories if history.rollback_ready)


@dataclass
class WeeklyOperationsArtifacts:
    root_dir: str
    weekly_report_path: str
    dashboard_path: str
    metric_spec_path: Optional[str] = None
    regression_center_path: Optional[str] = None
    queue_control_path: Optional[str] = None
    version_center_path: Optional[str] = None


@dataclass
class QueueControlCenter:
    queue_depth: int
    queued_by_priority: Dict[str, int]
    queued_by_risk: Dict[str, int]
    execution_media: Dict[str, int]
    waiting_approval_runs: int
    blocked_tasks: List[str] = field(default_factory=list)
    queued_tasks: List[str] = field(default_factory=list)
    actions: Dict[str, List] = field(default_factory=dict)


@dataclass
class EngineeringOverviewKPI:
    name: str
    value: float
    target: float
    unit: str = ""
    direction: str = "up"

    @property
    def healthy(self) -> bool:
        if self.direction == "down":
            return self.value <= self.target
        return self.value >= self.target


@dataclass
class EngineeringFunnelStage:
    name: str
    count: int
    share: float


@dataclass
class EngineeringOverviewBlocker:
    summary: str
    affected_runs: int
    affected_tasks: List[str] = field(default_factory=list)
    owner: str = "engineering"
    severity: str = "medium"


@dataclass
class EngineeringActivity:
    timestamp: str
    run_id: str
    task_id: str
    status: str
    summary: str


@dataclass
class EngineeringOverviewPermission:
    viewer_role: str
    allowed_modules: List[str] = field(default_factory=list)

    def can_view(self, module: str) -> bool:
        return module in self.allowed_modules


@dataclass
class EngineeringOverview:
    name: str
    period: str
    snapshot: OperationsSnapshot
    permissions: EngineeringOverviewPermission
    kpis: List[EngineeringOverviewKPI] = field(default_factory=list)
    funnel: List[EngineeringFunnelStage] = field(default_factory=list)
    blockers: List[EngineeringOverviewBlocker] = field(default_factory=list)
    activities: List[EngineeringActivity] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricDefinition:
    metric_id: str
    label: str
    unit: str
    direction: str
    formula: str
    description: str
    source_fields: List[str] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricValue:
    metric_id: str
    label: str
    value: float
    display_value: str
    numerator: float
    denominator: float
    unit: str
    evidence: List[str] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricSpec:
    name: str
    generated_at: str
    period_start: str
    period_end: str
    timezone_name: str
    definitions: List[OperationsMetricDefinition] = field(default_factory=list)
    values: List[OperationsMetricValue] = field(default_factory=list)


@dataclass(frozen=True)
class DashboardWidgetSpec:
    widget_id: str
    title: str
    module: str
    data_source: str
    default_width: int = 4
    default_height: int = 3
    min_width: int = 2
    max_width: int = 12

    def to_dict(self) -> Dict[str, object]:
        return {
            "widget_id": self.widget_id,
            "title": self.title,
            "module": self.module,
            "data_source": self.data_source,
            "default_width": self.default_width,
            "default_height": self.default_height,
            "min_width": self.min_width,
            "max_width": self.max_width,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetSpec":
        return cls(
            widget_id=str(data["widget_id"]),
            title=str(data["title"]),
            module=str(data["module"]),
            data_source=str(data["data_source"]),
            default_width=int(data.get("default_width", 4)),
            default_height=int(data.get("default_height", 3)),
            min_width=int(data.get("min_width", 2)),
            max_width=int(data.get("max_width", 12)),
        )


@dataclass(frozen=True)
class DashboardWidgetPlacement:
    placement_id: str
    widget_id: str
    column: int
    row: int
    width: int
    height: int
    title_override: str = ""
    filters: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "placement_id": self.placement_id,
            "widget_id": self.widget_id,
            "column": self.column,
            "row": self.row,
            "width": self.width,
            "height": self.height,
            "title_override": self.title_override,
            "filters": list(self.filters),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetPlacement":
        return cls(
            placement_id=str(data["placement_id"]),
            widget_id=str(data["widget_id"]),
            column=int(data.get("column", 0)),
            row=int(data.get("row", 0)),
            width=int(data.get("width", 1)),
            height=int(data.get("height", 1)),
            title_override=str(data.get("title_override", "")),
            filters=[str(item) for item in data.get("filters", [])],
        )


@dataclass
class DashboardLayout:
    layout_id: str
    name: str
    columns: int = 12
    placements: List[DashboardWidgetPlacement] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "layout_id": self.layout_id,
            "name": self.name,
            "columns": self.columns,
            "placements": [placement.to_dict() for placement in self.placements],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardLayout":
        return cls(
            layout_id=str(data["layout_id"]),
            name=str(data["name"]),
            columns=int(data.get("columns", 12)),
            placements=[DashboardWidgetPlacement.from_dict(item) for item in data.get("placements", [])],
        )


@dataclass
class DashboardBuilder:
    name: str
    period: str
    owner: str
    permissions: EngineeringOverviewPermission
    widgets: List[DashboardWidgetSpec] = field(default_factory=list)
    layouts: List[DashboardLayout] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def widget_index(self) -> Dict[str, DashboardWidgetSpec]:
        return {widget.widget_id: widget for widget in self.widgets}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "period": self.period,
            "owner": self.owner,
            "permissions": {
                "viewer_role": self.permissions.viewer_role,
                "allowed_modules": list(self.permissions.allowed_modules),
            },
            "widgets": [widget.to_dict() for widget in self.widgets],
            "layouts": [layout.to_dict() for layout in self.layouts],
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilder":
        permissions = dict(data.get("permissions", {}))
        return cls(
            name=str(data["name"]),
            period=str(data["period"]),
            owner=str(data["owner"]),
            permissions=EngineeringOverviewPermission(
                viewer_role=str(permissions.get("viewer_role", "contributor")),
                allowed_modules=[str(item) for item in permissions.get("allowed_modules", [])],
            ),
            widgets=[DashboardWidgetSpec.from_dict(item) for item in data.get("widgets", [])],
            layouts=[DashboardLayout.from_dict(item) for item in data.get("layouts", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class DashboardBuilderAudit:
    name: str
    total_widgets: int
    layout_count: int
    placed_widgets: int
    duplicate_placement_ids: List[str] = field(default_factory=list)
    missing_widget_defs: List[str] = field(default_factory=list)
    inaccessible_widgets: List[str] = field(default_factory=list)
    overlapping_placements: List[str] = field(default_factory=list)
    out_of_bounds_placements: List[str] = field(default_factory=list)
    empty_layouts: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def release_ready(self) -> bool:
        return not (
            self.duplicate_placement_ids
            or self.missing_widget_defs
            or self.inaccessible_widgets
            or self.overlapping_placements
            or self.out_of_bounds_placements
            or self.empty_layouts
            or not self.documentation_complete
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "total_widgets": self.total_widgets,
            "layout_count": self.layout_count,
            "placed_widgets": self.placed_widgets,
            "duplicate_placement_ids": list(self.duplicate_placement_ids),
            "missing_widget_defs": list(self.missing_widget_defs),
            "inaccessible_widgets": list(self.inaccessible_widgets),
            "overlapping_placements": list(self.overlapping_placements),
            "out_of_bounds_placements": list(self.out_of_bounds_placements),
            "empty_layouts": list(self.empty_layouts),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilderAudit":
        return cls(
            name=str(data["name"]),
            total_widgets=int(data.get("total_widgets", 0)),
            layout_count=int(data.get("layout_count", 0)),
            placed_widgets=int(data.get("placed_widgets", 0)),
            duplicate_placement_ids=[str(item) for item in data.get("duplicate_placement_ids", [])],
            missing_widget_defs=[str(item) for item in data.get("missing_widget_defs", [])],
            inaccessible_widgets=[str(item) for item in data.get("inaccessible_widgets", [])],
            overlapping_placements=[str(item) for item in data.get("overlapping_placements", [])],
            out_of_bounds_placements=[str(item) for item in data.get("out_of_bounds_placements", [])],
            empty_layouts=[str(item) for item in data.get("empty_layouts", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


class OperationsAnalytics:
    METRIC_DEFINITIONS = (
        OperationsMetricDefinition(
            metric_id="runs-today",
            label="Runs Today",
            unit="runs",
            direction="up",
            formula="count(run.started_at within [period_start, period_end])",
            description="Number of runs that started inside the reporting day window.",
            source_fields=["started_at"],
        ),
        OperationsMetricDefinition(
            metric_id="avg-lead-time",
            label="Avg Lead Time",
            unit="m",
            direction="down",
            formula="sum(cycle_minutes for runs with started_at and ended_at) / measured_runs",
            description="Average elapsed minutes from run start to run end for runs with complete timestamps.",
            source_fields=["started_at", "ended_at"],
        ),
        OperationsMetricDefinition(
            metric_id="intervention-rate",
            label="Intervention Rate",
            unit="%",
            direction="down",
            formula="100 * actionable_runs / total_runs",
            description="Share of runs that require operator intervention because they ended in an actionable status.",
            source_fields=["status"],
        ),
        OperationsMetricDefinition(
            metric_id="sla",
            label="SLA",
            unit="%",
            direction="up",
            formula="100 * compliant_runs / measured_runs where compliant_runs have cycle_minutes <= sla_target_minutes",
            description="Share of measured runs that met the SLA target.",
            source_fields=["started_at", "ended_at"],
        ),
        OperationsMetricDefinition(
            metric_id="regression",
            label="Regression",
            unit="cases",
            direction="down",
            formula="count(current.compare(baseline) deltas < 0 or pass->fail transitions)",
            description="Number of benchmark cases that regressed against the provided baseline suite.",
            source_fields=["benchmark.current", "benchmark.baseline"],
        ),
        OperationsMetricDefinition(
            metric_id="risk",
            label="Risk",
            unit="score",
            direction="down",
            formula="sum(resolved_run_risk_score) / runs_with_risk where risk_score.total wins over risk_level mapping low=25, medium=60, high=90",
            description="Average per-run risk score from explicit risk scores or normalized risk levels.",
            source_fields=["risk_score.total", "risk_level"],
        ),
        OperationsMetricDefinition(
            metric_id="spend",
            label="Spend",
            unit="USD",
            direction="down",
            formula="sum(first non-null of spend_usd, cost_usd, spend, cost across runs)",
            description="Total reported run spend in USD over the reporting window.",
            source_fields=["spend_usd", "cost_usd", "spend", "cost"],
        ),
    )

    def summarize_runs(
        self,
        runs: Sequence[dict],
        sla_target_minutes: int = 60,
        top_n_blockers: int = 3,
    ) -> OperationsSnapshot:
        status_counts: Dict[str, int] = {}
        total_cycle_minutes = 0.0
        cycle_count = 0
        completed = 0
        approval_queue_depth = 0
        sla_breach_count = 0

        for run in runs:
            status = str(run.get("status", "unknown"))
            status_counts[status] = status_counts.get(status, 0) + 1

            if status == "needs-approval":
                approval_queue_depth += 1

            cycle_minutes = self._cycle_minutes(run)
            if cycle_minutes is not None:
                total_cycle_minutes += cycle_minutes
                cycle_count += 1
                if cycle_minutes > sla_target_minutes:
                    sla_breach_count += 1

            if status in STATUS_COMPLETE:
                completed += 1

        success_rate = round((completed / len(runs)) * 100, 1) if runs else 0.0
        average_cycle_minutes = round(total_cycle_minutes / cycle_count, 1) if cycle_count else 0.0
        blockers = self.build_triage_clusters(runs)[:top_n_blockers]
        return OperationsSnapshot(
            total_runs=len(runs),
            status_counts=status_counts,
            success_rate=success_rate,
            approval_queue_depth=approval_queue_depth,
            sla_target_minutes=sla_target_minutes,
            sla_breach_count=sla_breach_count,
            average_cycle_minutes=average_cycle_minutes,
            top_blockers=blockers,
        )

    def build_metric_spec(
        self,
        runs: Sequence[dict],
        *,
        period_start: str,
        period_end: str,
        timezone_name: str = "UTC",
        generated_at: Optional[str] = None,
        sla_target_minutes: int = 60,
        current_suite: Optional[BenchmarkSuiteResult] = None,
        baseline_suite: Optional[BenchmarkSuiteResult] = None,
    ) -> OperationsMetricSpec:
        period_start_dt = self._parse_ts(period_start)
        period_end_dt = self._parse_ts(period_end)
        if period_start_dt is None or period_end_dt is None or period_end_dt < period_start_dt:
            raise ValueError("period_start and period_end must be valid ISO-8601 timestamps with period_end >= period_start")

        runs_today = 0
        lead_time_sum = 0.0
        lead_time_count = 0
        actionable_runs = 0
        sla_compliant_runs = 0
        risk_sum = 0.0
        risk_count = 0
        spend_total = 0.0

        for run in runs:
            started_at = self._parse_ts(str(run.get("started_at", "")))
            if started_at is not None and period_start_dt <= started_at <= period_end_dt:
                runs_today += 1

            cycle_minutes = self._cycle_minutes(run)
            if cycle_minutes is not None:
                lead_time_sum += cycle_minutes
                lead_time_count += 1
                if cycle_minutes <= sla_target_minutes:
                    sla_compliant_runs += 1

            if str(run.get("status", "unknown")) in STATUS_ACTIONABLE:
                actionable_runs += 1

            risk_score = self._resolve_run_risk_score(run)
            if risk_score is not None:
                risk_sum += risk_score
                risk_count += 1

            spend_total += self._resolve_run_spend(run)

        regression_findings = self.analyze_regressions(current_suite, baseline_suite) if current_suite is not None else []
        total_runs = len(runs)
        avg_lead = round(lead_time_sum / lead_time_count, 1) if lead_time_count else 0.0
        intervention_rate = round((actionable_runs / total_runs) * 100, 1) if total_runs else 0.0
        sla_value = round((sla_compliant_runs / lead_time_count) * 100, 1) if lead_time_count else 0.0
        avg_risk = round(risk_sum / risk_count, 1) if risk_count else 0.0
        spend_total = round(spend_total, 2)

        values = [
            OperationsMetricValue(
                metric_id="runs-today",
                label="Runs Today",
                value=float(runs_today),
                display_value=str(runs_today),
                numerator=float(runs_today),
                denominator=float(total_runs),
                unit="runs",
                evidence=[f"{runs_today} of {total_runs} runs started inside the reporting window."],
            ),
            OperationsMetricValue(
                metric_id="avg-lead-time",
                label="Avg Lead Time",
                value=avg_lead,
                display_value=f"{avg_lead:.1f}m",
                numerator=round(lead_time_sum, 1),
                denominator=float(lead_time_count),
                unit="m",
                evidence=[f"{lead_time_count} runs had valid start/end timestamps."],
            ),
            OperationsMetricValue(
                metric_id="intervention-rate",
                label="Intervention Rate",
                value=intervention_rate,
                display_value=f"{intervention_rate:.1f}%",
                numerator=float(actionable_runs),
                denominator=float(total_runs),
                unit="%",
                evidence=[f"Actionable statuses counted: {', '.join(sorted(STATUS_ACTIONABLE))}."],
            ),
            OperationsMetricValue(
                metric_id="sla",
                label="SLA",
                value=sla_value,
                display_value=f"{sla_value:.1f}%",
                numerator=float(sla_compliant_runs),
                denominator=float(lead_time_count),
                unit="%",
                evidence=[
                    f"SLA target: {sla_target_minutes} minutes.",
                    f"{sla_compliant_runs} of {lead_time_count} measured runs met target.",
                ],
            ),
            OperationsMetricValue(
                metric_id="regression",
                label="Regression",
                value=float(len(regression_findings)),
                display_value=str(len(regression_findings)),
                numerator=float(len(regression_findings)),
                denominator=float(len(current_suite.results)) if current_suite is not None else 0.0,
                unit="cases",
                evidence=[
                    f"Baseline provided: {baseline_suite is not None}.",
                    f"Current suite provided: {current_suite is not None}.",
                ],
            ),
            OperationsMetricValue(
                metric_id="risk",
                label="Risk",
                value=avg_risk,
                display_value=f"{avg_risk:.1f}",
                numerator=round(risk_sum, 1),
                denominator=float(risk_count),
                unit="score",
                evidence=["Risk score precedence: risk_score.total, then risk_level mapping low=25 medium=60 high=90."],
            ),
            OperationsMetricValue(
                metric_id="spend",
                label="Spend",
                value=spend_total,
                display_value=f"${spend_total:.2f}",
                numerator=spend_total,
                denominator=float(total_runs),
                unit="USD",
                evidence=["Spend field precedence: spend_usd, cost_usd, spend, cost."],
            ),
        ]

        return OperationsMetricSpec(
            name="Operations Metric Spec",
            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            period_start=period_start,
            period_end=period_end,
            timezone_name=timezone_name,
            definitions=list(self.METRIC_DEFINITIONS),
            values=values,
        )

    def build_triage_clusters(self, runs: Sequence[dict]) -> List[TriageCluster]:
        clusters: Dict[str, TriageCluster] = {}
        for run in runs:
            status = str(run.get("status", "unknown"))
            if status not in STATUS_ACTIONABLE:
                continue

            reason = self._primary_reason(run)
            cluster = clusters.setdefault(reason, TriageCluster(reason=reason))
            run_id = str(run.get("run_id", ""))
            task_id = str(run.get("task_id", ""))
            if run_id and run_id not in cluster.run_ids:
                cluster.run_ids.append(run_id)
            if task_id and task_id not in cluster.task_ids:
                cluster.task_ids.append(task_id)
            if status not in cluster.statuses:
                cluster.statuses.append(status)

        return sorted(
            clusters.values(),
            key=lambda cluster: (-cluster.occurrences, cluster.reason),
        )

    def analyze_regressions(
        self,
        current: BenchmarkSuiteResult,
        baseline: Optional[BenchmarkSuiteResult] = None,
    ) -> List[RegressionFinding]:
        if baseline is None:
            return []

        baseline_results = {result.case_id: result for result in baseline.results}
        findings: List[RegressionFinding] = []
        for comparison in current.compare(baseline):
            baseline_result = baseline_results.get(comparison.case_id)
            current_result = next(result for result in current.results if result.case_id == comparison.case_id)
            if comparison.delta >= 0 and not (baseline_result and baseline_result.passed and not current_result.passed):
                continue

            severity = "high" if comparison.delta <= -20 or (baseline_result and baseline_result.passed and not current_result.passed) else "medium"
            summary = (
                f"score dropped from {comparison.baseline_score} to {comparison.current_score}"
                if comparison.delta < 0
                else "case regressed from passing to failing"
            )
            findings.append(
                RegressionFinding(
                    case_id=comparison.case_id,
                    baseline_score=comparison.baseline_score,
                    current_score=comparison.current_score,
                    delta=comparison.delta,
                    severity=severity,
                    summary=summary,
                )
            )

        return sorted(findings, key=lambda finding: (finding.delta, finding.case_id))

    def build_regression_center(
        self,
        current: BenchmarkSuiteResult,
        baseline: BenchmarkSuiteResult,
        name: str = "Regression Analysis Center",
    ) -> RegressionCenter:
        regressions = self.analyze_regressions(current, baseline)
        comparisons = current.compare(baseline)
        improved_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta > 0)
        unchanged_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta == 0)
        return RegressionCenter(
            name=name,
            baseline_version=baseline.version,
            current_version=current.version,
            regressions=regressions,
            improved_cases=improved_cases,
            unchanged_cases=unchanged_cases,
        )

    def build_queue_control_center(
        self,
        queue: PersistentTaskQueue,
        runs: Sequence[dict],
    ) -> QueueControlCenter:
        queued_tasks = queue.peek_tasks()
        queued_by_priority = {"P0": 0, "P1": 0, "P2": 0}
        queued_by_risk = {"low": 0, "medium": 0, "high": 0}
        for task in queued_tasks:
            queued_by_priority[f"P{int(task.priority)}"] += 1
            queued_by_risk[task.risk_level.value] += 1

        execution_media: Dict[str, int] = {}
        waiting_approval_runs = 0
        blocked_tasks: List[str] = []
        for run in runs:
            medium = str(run.get("medium", "unknown"))
            execution_media[medium] = execution_media.get(medium, 0) + 1
            if run.get("status") == "needs-approval":
                waiting_approval_runs += 1
                task_id = str(run.get("task_id", ""))
                if task_id and task_id not in blocked_tasks:
                    blocked_tasks.append(task_id)

        return QueueControlCenter(
            queue_depth=queue.size(),
            queued_by_priority=queued_by_priority,
            queued_by_risk=queued_by_risk,
            execution_media=execution_media,
            waiting_approval_runs=waiting_approval_runs,
            blocked_tasks=blocked_tasks,
            queued_tasks=[task.task_id for task in queued_tasks],
            actions={
                task.task_id: build_console_actions(
                    task.task_id,
                    allow_retry=task.task_id in blocked_tasks,
                    retry_reason="" if task.task_id in blocked_tasks else "retry is reserved for blocked queue items",
                    allow_pause=task.task_id not in blocked_tasks,
                    pause_reason="" if task.task_id not in blocked_tasks else "approval-blocked tasks should be escalated instead of paused",
                    allow_escalate=task.task_id in blocked_tasks,
                    escalate_reason="" if task.task_id in blocked_tasks else "escalate is reserved for blocked queue items",
                )
                for task in queued_tasks
            },
        )

    def build_policy_prompt_version_center(
        self,
        artifacts: Sequence[VersionedArtifact],
        name: str = "Policy/Prompt Version Center",
        generated_at: Optional[str] = None,
        diff_preview_lines: int = 8,
    ) -> PolicyPromptVersionCenter:
        grouped: Dict[tuple[str, str], List[VersionedArtifact]] = {}
        for artifact in artifacts:
            key = (artifact.artifact_type, artifact.artifact_id)
            grouped.setdefault(key, []).append(artifact)

        histories: List[VersionedArtifactHistory] = []
        for artifact_type, artifact_id in sorted(grouped.keys()):
            revisions = sorted(
                grouped[(artifact_type, artifact_id)],
                key=lambda artifact: self._parse_ts(artifact.updated_at) or datetime.min.replace(tzinfo=timezone.utc),
                reverse=True,
            )
            current = revisions[0]
            previous = revisions[1] if len(revisions) > 1 else None
            change_summary = None
            rollback_version = None
            rollback_ready = False

            if previous is not None:
                change_summary = self._summarize_version_change(previous, current, preview_lines=diff_preview_lines)
                rollback_version = previous.version
                rollback_ready = bool(previous.content.strip())

            histories.append(
                VersionedArtifactHistory(
                    artifact_type=artifact_type,
                    artifact_id=artifact_id,
                    current_version=current.version,
                    current_updated_at=current.updated_at,
                    current_author=current.author,
                    current_summary=current.summary,
                    revision_count=len(revisions),
                    revisions=revisions,
                    rollback_version=rollback_version,
                    rollback_ready=rollback_ready,
                    change_summary=change_summary,
                )
            )

        return PolicyPromptVersionCenter(
            name=name,
            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            histories=histories,
        )

    def build_engineering_overview(
        self,
        name: str,
        period: str,
        runs: Sequence[dict],
        viewer_role: str,
        sla_target_minutes: int = 60,
        top_n_blockers: int = 3,
        recent_activity_limit: int = 5,
    ) -> EngineeringOverview:
        snapshot = self.summarize_runs(
            runs,
            sla_target_minutes=sla_target_minutes,
            top_n_blockers=top_n_blockers,
        )
        permissions = self._permissions_for_role(viewer_role)
        kpis = [
            EngineeringOverviewKPI(name="success-rate", value=snapshot.success_rate, target=90.0, unit="%"),
            EngineeringOverviewKPI(
                name="approval-queue-depth",
                value=float(snapshot.approval_queue_depth),
                target=2.0,
                direction="down",
            ),
            EngineeringOverviewKPI(
                name="sla-breaches",
                value=float(snapshot.sla_breach_count),
                target=0.0,
                direction="down",
            ),
            EngineeringOverviewKPI(
                name="average-cycle-minutes",
                value=snapshot.average_cycle_minutes,
                target=float(sla_target_minutes),
                unit="m",
                direction="down",
            ),
        ]
        blockers = [
            EngineeringOverviewBlocker(
                summary=cluster.reason,
                affected_runs=cluster.occurrences,
                affected_tasks=cluster.task_ids,
                owner=self._owner_for_cluster(cluster),
                severity=self._severity_for_cluster(cluster),
            )
            for cluster in snapshot.top_blockers
        ]
        return EngineeringOverview(
            name=name,
            period=period,
            snapshot=snapshot,
            permissions=permissions,
            kpis=kpis,
            funnel=self._build_funnel(snapshot.status_counts, snapshot.total_runs),
            blockers=blockers,
            activities=self._build_recent_activities(runs, recent_activity_limit),
        )

    def build_weekly_report(
        self,
        name: str,
        period: str,
        runs: Sequence[dict],
        current_suite: Optional[BenchmarkSuiteResult] = None,
        baseline_suite: Optional[BenchmarkSuiteResult] = None,
        sla_target_minutes: int = 60,
    ) -> WeeklyOperationsReport:
        snapshot = self.summarize_runs(runs, sla_target_minutes=sla_target_minutes)
        regressions = []
        if current_suite is not None:
            regressions = self.analyze_regressions(current_suite, baseline_suite)
        return WeeklyOperationsReport(
            name=name,
            period=period,
            snapshot=snapshot,
            regressions=regressions,
        )

    def build_dashboard_builder(
        self,
        name: str,
        period: str,
        owner: str,
        viewer_role: str,
        widgets: Sequence[DashboardWidgetSpec],
        layouts: Sequence[DashboardLayout],
        documentation_complete: bool = False,
    ) -> DashboardBuilder:
        return DashboardBuilder(
            name=name,
            period=period,
            owner=owner,
            permissions=self._permissions_for_role(viewer_role),
            widgets=list(widgets),
            layouts=[self.normalize_dashboard_layout(layout, widgets) for layout in layouts],
            documentation_complete=documentation_complete,
        )

    def normalize_dashboard_layout(
        self,
        layout: DashboardLayout,
        widgets: Sequence[DashboardWidgetSpec],
    ) -> DashboardLayout:
        widget_index = {widget.widget_id: widget for widget in widgets}
        normalized: List[DashboardWidgetPlacement] = []
        column_count = max(1, layout.columns)
        for placement in layout.placements:
            spec = widget_index.get(placement.widget_id)
            min_width = spec.min_width if spec is not None else 1
            max_width = min(spec.max_width, column_count) if spec is not None else column_count
            width = max(min_width, min(placement.width, max_width))
            column = max(0, placement.column)
            if column + width > column_count:
                column = max(0, column_count - width)
            normalized.append(
                DashboardWidgetPlacement(
                    placement_id=placement.placement_id,
                    widget_id=placement.widget_id,
                    column=column,
                    row=max(0, placement.row),
                    width=width,
                    height=max(1, placement.height),
                    title_override=placement.title_override,
                    filters=list(placement.filters),
                )
            )

        normalized.sort(key=lambda item: (item.row, item.column, item.placement_id))
        return DashboardLayout(
            layout_id=layout.layout_id,
            name=layout.name,
            columns=column_count,
            placements=normalized,
        )

    def audit_dashboard_builder(self, dashboard: DashboardBuilder) -> DashboardBuilderAudit:
        widget_index = dashboard.widget_index
        placement_counts: Dict[str, int] = {}
        missing_widget_defs: set[str] = set()
        inaccessible_widgets: set[str] = set()
        overlapping_placements: set[str] = set()
        out_of_bounds_placements: set[str] = set()
        empty_layouts: List[str] = []
        placed_widgets = 0

        for layout in dashboard.layouts:
            if not layout.placements:
                empty_layouts.append(layout.layout_id)
                continue

            placed_widgets += len(layout.placements)
            for placement in layout.placements:
                placement_counts[placement.placement_id] = placement_counts.get(placement.placement_id, 0) + 1
                spec = widget_index.get(placement.widget_id)
                if spec is None:
                    missing_widget_defs.add(placement.widget_id)
                else:
                    if not dashboard.permissions.can_view(spec.module):
                        inaccessible_widgets.add(placement.widget_id)
                if placement.column + placement.width > layout.columns:
                    out_of_bounds_placements.add(placement.placement_id)

            for index, placement in enumerate(layout.placements):
                for other in layout.placements[index + 1 :]:
                    if self._placements_overlap(placement, other):
                        overlapping_placements.add(
                            f"{layout.layout_id}:{placement.placement_id}<->{other.placement_id}"
                        )

        duplicate_ids = sorted(
            placement_id for placement_id, count in placement_counts.items() if count > 1
        )
        return DashboardBuilderAudit(
            name=dashboard.name,
            total_widgets=len(dashboard.widgets),
            layout_count=len(dashboard.layouts),
            placed_widgets=placed_widgets,
            duplicate_placement_ids=duplicate_ids,
            missing_widget_defs=sorted(missing_widget_defs),
            inaccessible_widgets=sorted(inaccessible_widgets),
            overlapping_placements=sorted(overlapping_placements),
            out_of_bounds_placements=sorted(out_of_bounds_placements),
            empty_layouts=sorted(empty_layouts),
            documentation_complete=dashboard.documentation_complete,
        )

    def _primary_reason(self, run: dict) -> str:
        for audit in run.get("audits", []):
            reason = audit.get("details", {}).get("reason")
            if reason:
                return str(reason)
        summary = str(run.get("summary", "")).strip()
        if summary:
            return summary
        return str(run.get("status", "unknown"))

    def _cycle_minutes(self, run: dict) -> Optional[float]:
        started_at = run.get("started_at")
        ended_at = run.get("ended_at")
        if not started_at or not ended_at:
            return None
        start = self._parse_ts(str(started_at))
        end = self._parse_ts(str(ended_at))
        if start is None or end is None or end < start:
            return None
        return round((end - start).total_seconds() / 60, 1)

    def _parse_ts(self, value: str) -> Optional[datetime]:
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)
        except ValueError:
            return None

    def _resolve_run_risk_score(self, run: dict) -> Optional[float]:
        risk_score = run.get("risk_score")
        if isinstance(risk_score, dict) and risk_score.get("total") is not None:
            try:
                return float(risk_score["total"])
            except (TypeError, ValueError):
                return None

        risk_level = str(run.get("risk_level", "")).strip().lower()
        risk_by_level = {"low": 25.0, "medium": 60.0, "high": 90.0}
        return risk_by_level.get(risk_level)

    def _resolve_run_spend(self, run: dict) -> float:
        for key in ("spend_usd", "cost_usd", "spend", "cost"):
            value = run.get(key)
            if value is None:
                continue
            try:
                return float(value)
            except (TypeError, ValueError):
                return 0.0
        return 0.0

    def _summarize_version_change(
        self,
        previous: VersionedArtifact,
        current: VersionedArtifact,
        preview_lines: int,
    ) -> VersionChangeSummary:
        diff_lines = list(
            unified_diff(
                previous.content.splitlines(),
                current.content.splitlines(),
                fromfile=previous.version,
                tofile=current.version,
                lineterm="",
            )
        )
        additions = sum(1 for line in diff_lines if line.startswith("+") and not line.startswith("+++"))
        deletions = sum(1 for line in diff_lines if line.startswith("-") and not line.startswith("---"))
        preview = [line for line in diff_lines if not line.startswith("@@")][:preview_lines]
        return VersionChangeSummary(
            from_version=previous.version,
            to_version=current.version,
            additions=additions,
            deletions=deletions,
            changed_lines=additions + deletions,
            preview=preview,
        )

    def _build_funnel(self, status_counts: Dict[str, int], total_runs: int) -> List[EngineeringFunnelStage]:
        funnel_counts = [
            ("queued", status_counts.get("queued", 0)),
            ("in-progress", status_counts.get("running", 0) + status_counts.get("in-progress", 0)),
            ("awaiting-approval", status_counts.get("needs-approval", 0)),
            ("completed", sum(count for status, count in status_counts.items() if status in STATUS_COMPLETE)),
        ]
        return [
            EngineeringFunnelStage(
                name=name,
                count=count,
                share=round((count / total_runs) * 100, 1) if total_runs else 0.0,
            )
            for name, count in funnel_counts
        ]

    def _build_recent_activities(self, runs: Sequence[dict], limit: int) -> List[EngineeringActivity]:
        dated_runs = []
        for run in runs:
            sort_key = self._parse_ts(str(run.get("ended_at", ""))) or self._parse_ts(str(run.get("started_at", "")))
            if sort_key is None:
                continue
            dated_runs.append((sort_key, run))

        activities: List[EngineeringActivity] = []
        for _, run in sorted(dated_runs, key=lambda item: item[0], reverse=True)[:limit]:
            activities.append(
                EngineeringActivity(
                    timestamp=str(run.get("ended_at") or run.get("started_at") or ""),
                    run_id=str(run.get("run_id", "")),
                    task_id=str(run.get("task_id", "")),
                    status=str(run.get("status", "unknown")),
                    summary=self._primary_reason(run),
                )
            )
        return activities

    def _permissions_for_role(self, viewer_role: str) -> EngineeringOverviewPermission:
        role = viewer_role.strip().lower() or "contributor"
        modules_by_role = {
            "executive": ["kpis", "funnel", "blockers"],
            "engineering-manager": ["kpis", "funnel", "blockers", "activity"],
            "operations": ["kpis", "funnel", "blockers", "activity"],
            "contributor": ["kpis", "activity"],
        }
        return EngineeringOverviewPermission(
            viewer_role=role,
            allowed_modules=modules_by_role.get(role, modules_by_role["contributor"]),
        )

    def _owner_for_cluster(self, cluster: TriageCluster) -> str:
        details = " ".join([cluster.reason, " ".join(cluster.statuses)]).lower()
        if "approval" in details:
            return "operations"
        if "security" in details:
            return "security"
        return "engineering"

    def _severity_for_cluster(self, cluster: TriageCluster) -> str:
        if cluster.occurrences >= 3 or "failed" in cluster.statuses:
            return "high"
        return "medium"

    @staticmethod
    def _placements_overlap(left: DashboardWidgetPlacement, right: DashboardWidgetPlacement) -> bool:
        return not (
            left.column + left.width <= right.column
            or right.column + right.width <= left.column
            or left.row + left.height <= right.row
            or right.row + right.height <= left.row
        )


def render_operations_dashboard(
    snapshot: OperationsSnapshot,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Operations Dashboard",
        "",
        f"- Total Runs: {snapshot.total_runs}",
        f"- Success Rate: {snapshot.success_rate:.1f}%",
        f"- Approval Queue Depth: {snapshot.approval_queue_depth}",
        f"- SLA Target: {snapshot.sla_target_minutes} minutes",
        f"- SLA Breaches: {snapshot.sla_breach_count}",
        f"- Average Cycle Time: {snapshot.average_cycle_minutes:.1f} minutes",
        "",
        "## Status Counts",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if snapshot.status_counts:
        for status, count in sorted(snapshot.status_counts.items()):
            lines.append(f"- {status}: {count}")
    else:
        lines.append("- None")

    lines.extend(["", "## Top Blockers", ""])
    if snapshot.top_blockers:
        for cluster in snapshot.top_blockers:
            statuses = ", ".join(cluster.statuses) if cluster.statuses else "unknown"
            lines.append(
                f"- {cluster.reason}: occurrences={cluster.occurrences} statuses={statuses} tasks={', '.join(cluster.task_ids)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_weekly_operations_report(report: WeeklyOperationsReport) -> str:
    lines = [
        "# Weekly Operations Report",
        "",
        f"- Name: {report.name}",
        f"- Period: {report.period}",
        f"- Total Runs: {report.snapshot.total_runs}",
        f"- Success Rate: {report.snapshot.success_rate:.1f}%",
        f"- SLA Breaches: {report.snapshot.sla_breach_count}",
        f"- Approval Queue Depth: {report.snapshot.approval_queue_depth}",
        "",
        "## Blockers",
        "",
    ]

    if report.snapshot.top_blockers:
        for cluster in report.snapshot.top_blockers:
            lines.append(f"- {cluster.reason}: {cluster.occurrences} runs")
    else:
        lines.append("- None")

    lines.extend(["", "## Regressions", ""])
    if report.regressions:
        for finding in report.regressions:
            lines.append(
                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_operations_metric_spec(spec: OperationsMetricSpec) -> str:
    lines = [
        "# Operations Metric Spec",
        "",
        f"- Name: {spec.name}",
        f"- Generated At: {spec.generated_at}",
        f"- Period Start: {spec.period_start}",
        f"- Period End: {spec.period_end}",
        f"- Timezone: {spec.timezone_name}",
        "",
        "## Definitions",
        "",
    ]

    for definition in spec.definitions:
        lines.extend(
            [
                f"### {definition.label}",
                "",
                f"- Metric ID: {definition.metric_id}",
                f"- Unit: {definition.unit}",
                f"- Direction: {definition.direction}",
                f"- Formula: {definition.formula}",
                f"- Description: {definition.description}",
                f"- Source Fields: {', '.join(definition.source_fields)}",
                "",
            ]
        )

    lines.extend(["## Values", ""])
    for value in spec.values:
        evidence = " | ".join(value.evidence) if value.evidence else "none"
        lines.append(
            f"- {value.label}: value={value.display_value} numerator={value.numerator:.1f} "
            f"denominator={value.denominator:.1f} unit={value.unit} evidence={evidence}"
        )

    return "\n".join(lines) + "\n"


def render_queue_control_center(
    center: QueueControlCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Queue Control Center",
        "",
        f"- Queue Depth: {center.queue_depth}",
        f"- Waiting Approval Runs: {center.waiting_approval_runs}",
        f"- Queued Tasks: {', '.join(center.queued_tasks) if center.queued_tasks else 'none'}",
        "",
        "## Queue By Priority",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    for priority, count in center.queued_by_priority.items():
        lines.append(f"- {priority}: {count}")

    lines.extend(["", "## Queue By Risk", ""])
    for risk_level, count in center.queued_by_risk.items():
        lines.append(f"- {risk_level}: {count}")

    lines.extend(["", "## Execution Media", ""])
    if center.execution_media:
        for medium, count in sorted(center.execution_media.items()):
            lines.append(f"- {medium}: {count}")
    else:
        lines.append("- None")

    lines.extend(["", "## Blocked Tasks", ""])
    if center.blocked_tasks:
        for task_id in center.blocked_tasks:
            lines.append(f"- {task_id}")
    else:
        lines.append("- None")

    lines.extend(["", "## Actions", ""])
    if center.actions:
        for task_id in center.queued_tasks:
            actions = center.actions.get(task_id, [])
            lines.append(f"- {task_id}: {render_console_actions(actions)}")
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_policy_prompt_version_center(
    center: PolicyPromptVersionCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Policy/Prompt Version Center",
        "",
        f"- Name: {center.name}",
        f"- Generated At: {center.generated_at}",
        f"- Versioned Artifacts: {center.artifact_count}",
        f"- Rollback Ready Artifacts: {center.rollback_ready_count}",
        "",
        "## Artifact Histories",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if not center.histories:
        lines.append("- None")
        return "\n".join(lines) + "\n"

    for history in center.histories:
        lines.extend(
            [
                f"### {history.artifact_type} / {history.artifact_id}",
                "",
                f"- Current Version: {history.current_version}",
                f"- Updated At: {history.current_updated_at}",
                f"- Updated By: {history.current_author}",
                f"- Summary: {history.current_summary}",
                f"- Revision Count: {history.revision_count}",
                f"- Rollback Version: {history.rollback_version or 'none'}",
                f"- Rollback Ready: {history.rollback_ready}",
            ]
        )
        if history.change_summary is not None:
            lines.append(
                f"- Diff Summary: {history.change_summary.additions} additions, "
                f"{history.change_summary.deletions} deletions"
            )
        lines.extend(["", "#### Revision History", ""])
        for revision in history.revisions:
            ticket = revision.change_ticket or "none"
            lines.append(
                f"- {revision.version}: updated_at={revision.updated_at} author={revision.author} "
                f"ticket={ticket} summary={revision.summary}"
            )
        lines.extend(["", "#### Diff Preview", ""])
        if history.change_summary is not None and history.change_summary.preview:
            lines.append("```diff")
            lines.extend(history.change_summary.preview)
            lines.append("```")
        else:
            lines.append("- None")
        lines.append("")

    return "\n".join(lines) + "\n"


def render_engineering_overview(overview: EngineeringOverview) -> str:
    lines = [
        "# Engineering Overview",
        "",
        f"- Name: {overview.name}",
        f"- Period: {overview.period}",
        f"- Viewer Role: {overview.permissions.viewer_role}",
        f"- Visible Modules: {', '.join(overview.permissions.allowed_modules)}",
    ]

    if overview.permissions.can_view("kpis"):
        lines.extend(["", "## KPI Modules", ""])
        for kpi in overview.kpis:
            lines.append(
                f"- {kpi.name}: value={kpi.value:.1f}{kpi.unit} target={kpi.target:.1f}{kpi.unit} healthy={kpi.healthy}"
            )

    if overview.permissions.can_view("funnel"):
        lines.extend(["", "## Funnel Modules", ""])
        for stage in overview.funnel:
            lines.append(f"- {stage.name}: count={stage.count} share={stage.share:.1f}%")

    if overview.permissions.can_view("blockers"):
        lines.extend(["", "## Blocker Modules", ""])
        if overview.blockers:
            for blocker in overview.blockers:
                lines.append(
                    f"- {blocker.summary}: severity={blocker.severity} owner={blocker.owner} "
                    f"affected_runs={blocker.affected_runs} tasks={', '.join(blocker.affected_tasks)}"
                )
        else:
            lines.append("- None")

    if overview.permissions.can_view("activity"):
        lines.extend(["", "## Activity Modules", ""])
        if overview.activities:
            for activity in overview.activities:
                lines.append(
                    f"- {activity.timestamp}: {activity.run_id} task={activity.task_id} "
                    f"status={activity.status} summary={activity.summary}"
                )
        else:
            lines.append("- None")

    return "\n".join(lines) + "\n"


def render_dashboard_builder_report(
    dashboard: DashboardBuilder,
    audit: DashboardBuilderAudit,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Dashboard Builder",
        "",
        f"- Name: {dashboard.name}",
        f"- Period: {dashboard.period}",
        f"- Owner: {dashboard.owner}",
        f"- Viewer Role: {dashboard.permissions.viewer_role}",
        f"- Available Widgets: {len(dashboard.widgets)}",
        f"- Layouts: {len(dashboard.layouts)}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Governance",
        "",
        f"- Documentation Complete: {audit.documentation_complete}",
        f"- Duplicate Placement IDs: {', '.join(audit.duplicate_placement_ids) if audit.duplicate_placement_ids else 'none'}",
        f"- Missing Widget Definitions: {', '.join(audit.missing_widget_defs) if audit.missing_widget_defs else 'none'}",
        f"- Inaccessible Widgets: {', '.join(audit.inaccessible_widgets) if audit.inaccessible_widgets else 'none'}",
        f"- Overlaps: {', '.join(audit.overlapping_placements) if audit.overlapping_placements else 'none'}",
        f"- Out Of Bounds: {', '.join(audit.out_of_bounds_placements) if audit.out_of_bounds_placements else 'none'}",
        f"- Empty Layouts: {', '.join(audit.empty_layouts) if audit.empty_layouts else 'none'}",
        "",
        "## Layouts",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if dashboard.layouts:
        for layout in dashboard.layouts:
            lines.append(f"- {layout.layout_id}: name={layout.name} columns={layout.columns} placements={len(layout.placements)}")
            for placement in layout.placements:
                widget = dashboard.widget_index.get(placement.widget_id)
                title = placement.title_override or (widget.title if widget is not None else placement.widget_id)
                filters = ", ".join(placement.filters) if placement.filters else "none"
                lines.append(
                    f"- {placement.placement_id}: widget={placement.widget_id} title={title} "
                    f"grid=({placement.column},{placement.row}) size={placement.width}x{placement.height} filters={filters}"
                )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def write_engineering_overview_bundle(root_dir: str, overview: EngineeringOverview) -> str:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    overview_path = str(base / "engineering-overview.md")
    write_report(overview_path, render_engineering_overview(overview))
    return overview_path


def write_dashboard_builder_bundle(
    root_dir: str,
    dashboard: DashboardBuilder,
    audit: DashboardBuilderAudit,
    view: Optional[SharedViewContext] = None,
) -> str:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    dashboard_path = str(base / "dashboard-builder.md")
    write_report(dashboard_path, render_dashboard_builder_report(dashboard, audit, view=view))
    return dashboard_path




def build_repo_collaboration_metrics(runs: Sequence[dict]) -> Dict[str, float]:
    total = len(runs)
    linked = 0
    accepted = 0
    discussion_posts = 0
    lineage_depth_sum = 0
    lineage_depth_count = 0

    for run in runs:
        links = run.get("closeout", {}).get("run_commit_links", [])
        if links:
            linked += 1
        if run.get("closeout", {}).get("accepted_commit_hash"):
            accepted += 1
        discussion_posts += int(run.get("repo_discussion_posts", 0))

        depth = run.get("accepted_lineage_depth")
        if depth is not None:
            lineage_depth_sum += float(depth)
            lineage_depth_count += 1

    return {
        "repo_link_coverage": round((linked / total) * 100, 1) if total else 0.0,
        "accepted_commit_rate": round((accepted / total) * 100, 1) if total else 0.0,
        "discussion_density": round(discussion_posts / total, 2) if total else 0.0,
        "accepted_lineage_depth_avg": round(lineage_depth_sum / lineage_depth_count, 2) if lineage_depth_count else 0.0,
    }


def write_weekly_operations_bundle(
    root_dir: str,
    report: WeeklyOperationsReport,
    metric_spec: Optional[OperationsMetricSpec] = None,
    regression_center: Optional[RegressionCenter] = None,
    queue_control_center: Optional[QueueControlCenter] = None,
    version_center: Optional[PolicyPromptVersionCenter] = None,
) -> WeeklyOperationsArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)

    weekly_report_path = str(base / "weekly-operations.md")
    dashboard_path = str(base / "operations-dashboard.md")
    write_report(weekly_report_path, render_weekly_operations_report(report))
    write_report(dashboard_path, render_operations_dashboard(report.snapshot))

    metric_spec_path = None
    if metric_spec is not None:
        metric_spec_path = str(base / "operations-metric-spec.md")
        write_report(metric_spec_path, render_operations_metric_spec(metric_spec))

    regression_center_path = None
    if regression_center is not None:
        regression_center_path = str(base / "regression-center.md")
        write_report(regression_center_path, render_regression_center(regression_center))

    queue_control_path = None
    if queue_control_center is not None:
        queue_control_path = str(base / "queue-control-center.md")
        write_report(queue_control_path, render_queue_control_center(queue_control_center))

    version_center_path = None
    if version_center is not None:
        version_center_path = str(base / "policy-prompt-version-center.md")
        write_report(version_center_path, render_policy_prompt_version_center(version_center))

    return WeeklyOperationsArtifacts(
        root_dir=str(base),
        weekly_report_path=weekly_report_path,
        dashboard_path=dashboard_path,
        metric_spec_path=metric_spec_path,
        regression_center_path=regression_center_path,
        queue_control_path=queue_control_path,
        version_center_path=version_center_path,
    )


def render_regression_center(
    center: RegressionCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Regression Analysis Center",
        "",
        f"- Name: {center.name}",
        f"- Baseline Version: {center.baseline_version}",
        f"- Current Version: {center.current_version}",
        f"- Regressions: {center.regression_count}",
        f"- Improved Cases: {len(center.improved_cases)}",
        f"- Unchanged Cases: {len(center.unchanged_cases)}",
        "",
        "## Regressions",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if center.regressions:
        for finding in center.regressions:
            lines.append(
                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Improved Cases", ""])
    if center.improved_cases:
        for case_id in center.improved_cases:
            lines.append(f"- {case_id}")
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


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
            accessibility_requirements=[
                str(requirement) for requirement in data.get("accessibility_requirements", [])
            ],
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
        penalties = (
            len(self.components_missing_docs)
            + len(self.components_missing_accessibility)
            + len(self.components_missing_states)
        )
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
            components_missing_accessibility=[
                str(name) for name in data.get("components_missing_accessibility", [])
            ],
            components_missing_states=[str(name) for name in data.get("components_missing_states", [])],
            undefined_token_refs={
                str(name): [str(token) for token in tokens]
                for name, tokens in dict(data.get("undefined_token_refs", {})).items()
            },
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
        return not (
            self.duplicate_routes
            or self.missing_route_nodes
            or self.secondary_nav_gaps
            or self.orphan_routes
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "total_navigation_nodes": self.total_navigation_nodes,
            "total_routes": self.total_routes,
            "duplicate_routes": list(self.duplicate_routes),
            "missing_route_nodes": dict(self.missing_route_nodes),
            "secondary_nav_gaps": {
                section: list(paths) for section, paths in self.secondary_nav_gaps.items()
            },
            "orphan_routes": list(self.orphan_routes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitectureAudit":
        return cls(
            total_navigation_nodes=int(data.get("total_navigation_nodes", 0)),
            total_routes=int(data.get("total_routes", 0)),
            duplicate_routes=[str(path) for path in data.get("duplicate_routes", [])],
            missing_route_nodes={
                str(node_id): str(path) for node_id, path in dict(data.get("missing_route_nodes", {})).items()
            },
            secondary_nav_gaps={
                str(section): [str(path) for path in paths]
                for section, paths in dict(data.get("secondary_nav_gaps", {})).items()
            },
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
        missing_route_nodes = {
            entry.node_id: entry.path
            for entry in entries
            if entry.path not in route_index
        }

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
        return {
            "global_nav": [node.to_dict() for node in self.global_nav],
            "routes": [route.to_dict() for route in self.routes],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InformationArchitecture":
        return cls(
            global_nav=[NavigationNode.from_dict(node) for node in data.get("global_nav", [])],
            routes=[NavigationRoute.from_dict(route) for route in data.get("routes", [])],
        )

    def _flatten_node(
        self,
        node: NavigationNode,
        parent_path: str,
        depth: int,
        parent_id: str,
    ) -> List[NavigationEntry]:
        path = self._join_path(parent_path, node.segment)
        entries = [
            NavigationEntry(
                node_id=node.node_id,
                title=node.title,
                path=path,
                depth=depth,
                parent_id=parent_id,
                screen_id=node.screen_id,
            )
        ]
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
        return {
            "id": self.id,
            "title": self.title,
            "section": self.section,
            "shortcut": self.shortcut,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, str]) -> "CommandAction":
        return cls(
            id=data["id"],
            title=data["title"],
            section=data["section"],
            shortcut=data.get("shortcut", ""),
        )


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
    command_entry: ConsoleCommandEntry = field(
        default_factory=lambda: ConsoleCommandEntry(trigger_label="", placeholder="", shortcut="")
    )
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
            accessibility_requirements=[
                str(requirement) for requirement in data.get("accessibility_requirements", [])
            ],
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
        return (
            not self.missing_capabilities
            and self.documentation_complete
            and self.accessibility_complete
            and self.command_shortcut_supported
        )

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

        normalized_shortcuts = {
            item.strip().lower().replace(" ", "")
            for item in top_bar.command_entry.shortcut.split("/")
            if item.strip()
        }
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
    lines.append(
        f"- Missing docs: {', '.join(audit.components_missing_docs) if audit.components_missing_docs else 'none'}"
    )
    lines.append(
        "- Missing accessibility: "
        f"{', '.join(audit.components_missing_accessibility) if audit.components_missing_accessibility else 'none'}"
    )
    lines.append(
        f"- Missing interaction states: {', '.join(audit.components_missing_states) if audit.components_missing_states else 'none'}"
    )
    if audit.undefined_token_refs:
        undefined_refs = "; ".join(
            f"{component}={', '.join(tokens)}" for component, tokens in sorted(audit.undefined_token_refs.items())
        )
    else:
        undefined_refs = "none"
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
    lines.append(
        f"- Missing capabilities: {', '.join(audit.missing_capabilities) if audit.missing_capabilities else 'none'}"
    )
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    lines.append(f"- Accessibility complete: {audit.accessibility_complete}")
    lines.append(f"- Cmd/Ctrl+K supported: {audit.command_shortcut_supported}")
    return "\n".join(lines) + "\n"


def render_information_architecture_report(
    architecture: InformationArchitecture,
    audit: InformationArchitectureAudit,
) -> str:
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
            lines.append(
                f"- {route.path}: screen={route.screen_id} title={route.title} nav_node={route.nav_node_id or 'none'}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    lines.append(f"- Duplicate routes: {', '.join(audit.duplicate_routes) if audit.duplicate_routes else 'none'}")
    if audit.missing_route_nodes:
        missing = ", ".join(f"{node_id}={path}" for node_id, path in sorted(audit.missing_route_nodes.items()))
    else:
        missing = "none"
    lines.append(f"- Missing route nodes: {missing}")
    if audit.secondary_nav_gaps:
        gaps = "; ".join(
            f"{section}={', '.join(paths)}" for section, paths in sorted(audit.secondary_nav_gaps.items())
        )
    else:
        gaps = "none"
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
        return (
            bool(self.personas)
            and bool(self.critical_steps)
            and self.expected_max_steps > 0
            and self.observed_steps <= self.expected_max_steps
            and self.keyboard_accessible
            and self.empty_state_guidance
            and self.recovery_support
        )

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
            role_permissions=[
                RolePermissionScenario.from_dict(scenario) for scenario in data.get("role_permissions", [])
            ],
            data_accuracy_checks=[
                DataAccuracyCheck.from_dict(check) for check in data.get("data_accuracy_checks", [])
            ],
            performance_budgets=[
                PerformanceBudget.from_dict(budget) for budget in data.get("performance_budgets", [])
            ],
            usability_journeys=[
                UsabilityJourney.from_dict(journey) for journey in data.get("usability_journeys", [])
            ],
            audit_requirements=[
                AuditRequirement.from_dict(requirement) for requirement in data.get("audit_requirements", [])
            ],
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
        return (
            not self.permission_gaps
            and not self.failing_data_checks
            and not self.failing_performance_budgets
            and not self.failing_usability_journeys
            and not self.incomplete_audit_trails
            and self.documentation_complete
        )

    @property
    def readiness_score(self) -> float:
        checks = [
            not self.permission_gaps,
            not self.failing_data_checks,
            not self.failing_performance_budgets,
            not self.failing_usability_journeys,
            not self.incomplete_audit_trails,
            self.documentation_complete,
        ]
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
        permission_gaps = [
            f"{scenario.screen_id}: missing={', '.join(scenario.missing_coverage)}"
            for scenario in suite.role_permissions
            if scenario.missing_coverage
        ]
        failing_data_checks = [
            f"{check.screen_id}.{check.metric_id}: delta={check.observed_delta} freshness={check.observed_freshness_seconds}s"
            for check in suite.data_accuracy_checks
            if not check.passes
        ]
        failing_performance_budgets = [
            f"{budget.surface_id}.{budget.interaction}: p95={budget.observed_p95_ms}ms"
            + (
                f" tti={budget.observed_tti_ms}ms"
                if budget.target_tti_ms > 0
                else ""
            )
            for budget in suite.performance_budgets
            if not budget.within_budget
        ]
        failing_usability_journeys = [
            f"{journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps}"
            for journey in suite.usability_journeys
            if not journey.passes
        ]
        incomplete_audit_trails = []
        for requirement in suite.audit_requirements:
            if requirement.complete:
                continue
            gaps = requirement.missing_fields
            parts: List[str] = []
            if gaps:
                parts.append(f"missing_fields={', '.join(gaps)}")
            if not requirement.retention_met:
                parts.append(
                    f"retention={requirement.observed_retention_days}/{requirement.retention_days}d"
                )
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
            lines.append(
                f"- Role/Permission {scenario.screen_id}: allow={', '.join(scenario.allowed_roles) or 'none'} "
                f"deny={denied} audit_event={scenario.audit_event or 'none'}"
            )
    else:
        lines.append("- Role/Permission: none")

    if suite.data_accuracy_checks:
        for check in suite.data_accuracy_checks:
            lines.append(
                f"- Data Accuracy {check.screen_id}.{check.metric_id}: delta={check.observed_delta} "
                f"tolerance={check.tolerance} freshness={check.observed_freshness_seconds}/{check.freshness_slo_seconds}s"
            )
    else:
        lines.append("- Data Accuracy: none")

    if suite.performance_budgets:
        for budget in suite.performance_budgets:
            tti_text = (
                f" tti={budget.observed_tti_ms}/{budget.target_tti_ms}ms"
                if budget.target_tti_ms > 0
                else ""
            )
            lines.append(
                f"- Performance {budget.surface_id}.{budget.interaction}: "
                f"p95={budget.observed_p95_ms}/{budget.target_p95_ms}ms{tti_text}"
            )
    else:
        lines.append("- Performance: none")

    if suite.usability_journeys:
        for journey in suite.usability_journeys:
            lines.append(
                f"- Usability {journey.journey_id}: steps={journey.observed_steps}/{journey.expected_max_steps} "
                f"keyboard={journey.keyboard_accessible} empty_state={journey.empty_state_guidance} "
                f"recovery={journey.recovery_support}"
            )
    else:
        lines.append("- Usability: none")

    if suite.audit_requirements:
        for requirement in suite.audit_requirements:
            lines.append(
                f"- Audit {requirement.event_type}: fields={len(requirement.emitted_fields)}/{len(requirement.required_fields)} "
                f"retention={requirement.observed_retention_days}/{requirement.retention_days}d"
            )
    else:
        lines.append("- Audit: none")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        f"- Role/Permission gaps: {', '.join(audit.permission_gaps) if audit.permission_gaps else 'none'}"
    )
    lines.append(
        f"- Data accuracy failures: {', '.join(audit.failing_data_checks) if audit.failing_data_checks else 'none'}"
    )
    lines.append(
        "- Performance budget failures: "
        f"{', '.join(audit.failing_performance_budgets) if audit.failing_performance_budgets else 'none'}"
    )
    lines.append(
        "- Usability journey failures: "
        f"{', '.join(audit.failing_usability_journeys) if audit.failing_usability_journeys else 'none'}"
    )
    lines.append(
        f"- Audit completeness gaps: {', '.join(audit.incomplete_audit_trails) if audit.incomplete_audit_trails else 'none'}"
    )
    lines.append(f"- Documentation complete: {audit.documentation_complete}")
    return "\n".join(lines) + "\n"


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
            "surfaces_missing_actions": {
                name: list(actions) for name, actions in self.surfaces_missing_actions.items()
            },
            "surfaces_missing_batch_actions": list(self.surfaces_missing_batch_actions),
            "surfaces_missing_states": {
                name: list(states) for name, states in self.surfaces_missing_states.items()
            },
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
            surfaces_missing_batch_actions=[
                str(name) for name in data.get("surfaces_missing_batch_actions", [])
            ],
            surfaces_missing_states={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("surfaces_missing_states", {})).items()
            },
            permission_gaps={
                str(name): [str(gap) for gap in gaps]
                for name, gaps in dict(data.get("permission_gaps", {})).items()
            },
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
            "surfaces_missing_states": {
                name: list(states) for name, states in self.surfaces_missing_states.items()
            },
            "states_missing_actions": {
                name: list(states) for name, states in self.states_missing_actions.items()
            },
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
            surfaces_missing_states={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("surfaces_missing_states", {})).items()
            },
            states_missing_actions={
                str(name): [str(state) for state in states]
                for name, states in dict(data.get("states_missing_actions", {})).items()
            },
            unresolved_state_actions={
                str(name): {
                    str(state): [str(action_id) for action_id in actions]
                    for state, actions in dict(states).items()
                }
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
                surface = next(
                    (candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name),
                    None,
                )
            if surface is None:
                missing_surfaces.append(contract.surface_name)
                continue

            if contract.requires_filters and not surface.filters:
                surfaces_missing_filters.append(contract.surface_name)

            available_action_ids = set(surface.action_ids)
            missing_action_ids = sorted(
                action_id for action_id in contract.required_action_ids if action_id not in available_action_ids
            )
            if missing_action_ids:
                surfaces_missing_actions[contract.surface_name] = missing_action_ids

            if contract.requires_batch_actions and not any(
                action.requires_selection for action in surface.top_bar_actions
            ):
                surfaces_missing_batch_actions.append(contract.surface_name)

            missing_state_ids = sorted(
                state_name for state_name in contract.required_states if state_name not in surface.state_names
            )
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

        uncovered_roles = sorted(
            role for role in draft.required_roles if role not in referenced_roles
        )

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
            lines.append(
                f"- {item.section} / {item.name}: route={item.route} badge={item.badge_count} icon={item.icon or 'none'}"
            )
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
            if unresolved:
                unresolved_text = "; ".join(
                    f"{state}={', '.join(action_ids)}" for state, action_ids in sorted(unresolved.items())
                )
            else:
                unresolved_text = "none"
            state_actions_missing = ", ".join(audit.states_missing_actions.get(surface.name, [])) or "none"
            lines.append(
                f"- {surface.name}: route={surface.route} filters={filters} actions={actions} states={states} "
                f"missing_states={missing_states} states_without_actions={state_actions_missing} "
                f"unresolved_state_actions={unresolved_text}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    lines.append(
        f"- Surfaces missing filters: {', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}"
    )
    lines.append(
        f"- Surfaces missing top-bar actions: {', '.join(audit.surfaces_missing_actions) if audit.surfaces_missing_actions else 'none'}"
    )
    if audit.surfaces_missing_states:
        missing_states_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())
        )
    else:
        missing_states_text = "none"
    lines.append(f"- Surfaces missing required states: {missing_states_text}")
    if audit.states_missing_actions:
        states_without_actions_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.states_missing_actions.items())
        )
    else:
        states_without_actions_text = "none"
    lines.append(f"- States without recovery actions: {states_without_actions_text}")
    if audit.unresolved_state_actions:
        unresolved_text = "; ".join(
            f"{name}="
            + ", ".join(f"{state}:{'/'.join(actions)}" for state, actions in sorted(states.items()))
            for name, states in sorted(audit.unresolved_state_actions.items())
        )
    else:
        unresolved_text = "none"
    lines.append(f"- Undefined state actions: {unresolved_text}")
    lines.append(
        f"- Orphan navigation routes: {', '.join(audit.orphan_navigation_routes) if audit.orphan_navigation_routes else 'none'}"
    )
    lines.append(
        f"- Unnavigable surfaces: {', '.join(audit.unnavigable_surfaces) if audit.unnavigable_surfaces else 'none'}"
    )
    return "\n".join(lines) + "\n"


def render_console_interaction_report(
    draft: ConsoleInteractionDraft,
    audit: ConsoleInteractionAudit,
) -> str:
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
                surface = next(
                    (candidate for candidate in draft.architecture.surfaces if candidate.name == contract.surface_name),
                    None,
                )
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
                f"states={', '.join(surface.state_names) or 'none'} batch={batch_mode} "
                f"permissions={permission_state}"
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
    lines.append(
        f"- Missing surfaces: {', '.join(audit.missing_surfaces) if audit.missing_surfaces else 'none'}"
    )
    lines.append(
        f"- Pages missing filters: {', '.join(audit.surfaces_missing_filters) if audit.surfaces_missing_filters else 'none'}"
    )
    if audit.surfaces_missing_actions:
        action_gap_text = "; ".join(
            f"{name}={', '.join(actions)}" for name, actions in sorted(audit.surfaces_missing_actions.items())
        )
    else:
        action_gap_text = "none"
    lines.append(f"- Pages missing actions: {action_gap_text}")
    lines.append(
        "- Pages missing batch actions: "
        f"{', '.join(audit.surfaces_missing_batch_actions) if audit.surfaces_missing_batch_actions else 'none'}"
    )
    if audit.surfaces_missing_states:
        state_gap_text = "; ".join(
            f"{name}={', '.join(states)}" for name, states in sorted(audit.surfaces_missing_states.items())
        )
    else:
        state_gap_text = "none"
    lines.append(f"- Pages missing states: {state_gap_text}")
    if audit.permission_gaps:
        permission_gap_text = "; ".join(
            f"{name}={', '.join(gaps)}" for name, gaps in sorted(audit.permission_gaps.items())
        )
    else:
        permission_gap_text = "none"
    lines.append(f"- Permission gaps: {permission_gap_text}")
    lines.append(
        f"- Uncovered roles: {', '.join(audit.uncovered_roles) if audit.uncovered_roles else 'none'}"
    )
    lines.append(
        f"- Pages missing personas: {', '.join(audit.surfaces_missing_primary_personas) if audit.surfaces_missing_primary_personas else 'none'}"
    )
    lines.append(
        f"- Pages missing wireframe links: {', '.join(audit.surfaces_missing_wireframe_links) if audit.surfaces_missing_wireframe_links else 'none'}"
    )
    lines.append(
        f"- Pages missing review focus: {', '.join(audit.surfaces_missing_review_focus) if audit.surfaces_missing_review_focus else 'none'}"
    )
    lines.append(
        f"- Pages missing decision prompts: {', '.join(audit.surfaces_missing_decision_prompts) if audit.surfaces_missing_decision_prompts else 'none'}"
    )
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


VALID_VIEW_VISIBILITY = {"private", "team", "organization"}
VALID_DIGEST_CHANNELS = {"email", "slack", "webhook"}
VALID_DIGEST_CADENCES = {"hourly", "daily", "weekly"}


@dataclass(frozen=True)
class SavedViewFilter:
    field: str
    operator: str
    value: str

    def to_dict(self) -> Dict[str, str]:
        return {
            "field": self.field,
            "operator": self.operator,
            "value": self.value,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewFilter":
        return cls(
            field=str(data["field"]),
            operator=str(data["operator"]),
            value=str(data["value"]),
        )


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
            subscriptions=[
                AlertDigestSubscription.from_dict(item) for item in data.get("subscriptions", [])
            ],
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
        score = max(0.0, 100 - ((penalties * 100) / self.view_count))
        return round(score, 1)

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
            invalid_visibility_views=[
                str(name) for name in data.get("invalid_visibility_views", [])
            ],
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
        passed = len([k for k in self.kpis if k.met])
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
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH


def _run(command: Sequence[str], repo: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=repo,
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
        raise GitSyncError(detail)
    return result.stdout


def _dirty(repo: Path) -> bool:
    return bool(_require_git(repo, "status", "--porcelain"))


def _remote_default_branch(repo: Path, remote: str) -> str:
    symbolic_ref = _git(repo, "symbolic-ref", "--quiet", f"refs/remotes/{remote}/HEAD")
    if symbolic_ref.returncode == 0 and symbolic_ref.stdout:
        prefix = f"refs/remotes/{remote}/"
        if symbolic_ref.stdout.startswith(prefix):
            return symbolic_ref.stdout[len(prefix) :]

    symref_result = _git(repo, "ls-remote", "--symref", remote, "HEAD")
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


def _remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _remote_default_branch(repo, remote)
        default_sha = _remote_branch_sha(repo, remote, default_branch)
    except GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


def inspect_repo_sync(repo: Path | str, remote: str = "origin") -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git(repo_path, "branch", "--show-current")
    if not branch:
        raise GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git(repo_path, "rev-parse", "HEAD")
    remote_result = _git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha

    if not remote_exists and _matches_remote_default_branch(repo_path, remote, local_sha):
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

    _require_git(repo_path, "config", "core.hooksPath", hooks_path)
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
        fetch_result = _git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise GitSyncError(detail)

        ff_result = _git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise GitSyncError(detail)
        status = inspect_repo_sync(repo_path, remote=remote)

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git(repo_path, *push_args)
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


PULL_REQUEST_COMMENT_EVENT = "pull_request.comment"
CI_COMPLETED_EVENT = "ci.completed"
TASK_FAILED_EVENT = "task.failed"

EventSubscriber = Callable[["BusEvent", TaskRun], None]


@dataclass(frozen=True)
class BusEvent:
    event_type: str
    run_id: str
    actor: str
    details: Dict[str, Any] = field(default_factory=dict)
    timestamp: str = field(default_factory=utc_now)


class EventBus:
    def __init__(self, ledger: Optional[ObservabilityLedger] = None):
        self.ledger = ledger
        self._runs: Dict[str, TaskRun] = {}
        self._subscribers: DefaultDict[str, List[EventSubscriber]] = defaultdict(list)

    def register_run(self, run: TaskRun) -> None:
        self._runs[run.run_id] = run

    def subscribe(self, event_type: str, handler: EventSubscriber) -> None:
        self._subscribers[event_type].append(handler)

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
            dashboard_missing_fields=self._missing_field_defs(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.fields,
            ),
            dashboard_sample_gaps=self._missing_sample_paths(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.sample,
            ),
            run_detail_missing_fields=self._missing_field_defs(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.fields,
            ),
            run_detail_sample_gaps=self._missing_sample_paths(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.sample,
            ),
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
                    "period": {
                        "label": "2026-W11",
                        "start": "2026-03-09",
                        "end": "2026-03-15",
                    },
                    "filters": {
                        "team": "core-product",
                        "viewer_role": "engineering-manager",
                    },
                    "summary": {
                        "total_runs": 42,
                        "success_rate": 88.1,
                        "approval_queue_depth": 3,
                        "sla_breach_count": 2,
                    },
                    "kpis": [
                        {
                            "name": "success-rate",
                            "value": 88.1,
                            "target": 90.0,
                            "unit": "%",
                            "direction": "up",
                            "healthy": False,
                        },
                        {
                            "name": "average-cycle-minutes",
                            "value": 47.3,
                            "target": 60.0,
                            "unit": "m",
                            "direction": "down",
                            "healthy": True,
                        },
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


def render_dashboard_run_contract_report(
    contract: DashboardRunContract,
    audit: DashboardRunContractAudit,
) -> str:
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


def _install_compat_module(source_module: types.ModuleType, name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    for export_name in export_names:
        module.__dict__[export_name] = getattr(source_module, export_name)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


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
            lines.append(f"- {api.method} {api.path}: request={api.request_model or 'none'} response={api.response_model or 'none'} permission={permission} audits={audits} metrics={metrics}")
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
            ExecutionModel("OperationsDashboardResponse", [ExecutionField("period", "string"), ExecutionField("total_runs", "int"), ExecutionField("success_rate", "float"), ExecutionField("approval_queue_depth", "int"), ExecutionField("sla_breach_count", "int"), ExecutionField("top_blockers", "string[]", required=False)], owner="operations"),
            ExecutionModel("RunDetailResponse", [ExecutionField("run_id", "string"), ExecutionField("task_id", "string"), ExecutionField("status", "string"), ExecutionField("timeline_events", "RunDetailEvent[]"), ExecutionField("resources", "RunDetailResource[]"), ExecutionField("audit_count", "int")], owner="operations"),
            ExecutionModel("RunReplayResponse", [ExecutionField("run_id", "string"), ExecutionField("replay_available", "bool"), ExecutionField("replay_path", "string", required=False), ExecutionField("benchmark_case_ids", "string[]", required=False)], owner="operations"),
            ExecutionModel("QueueControlCenterResponse", [ExecutionField("queue_depth", "int"), ExecutionField("queued_by_priority", "map<string,int>"), ExecutionField("queued_by_risk", "map<string,int>"), ExecutionField("waiting_approval_runs", "int"), ExecutionField("blocked_tasks", "string[]", required=False)], owner="operations"),
            ExecutionModel("QueueActionRequest", [ExecutionField("actor", "string"), ExecutionField("reason", "string")], owner="operations"),
            ExecutionModel("QueueActionResponse", [ExecutionField("task_id", "string"), ExecutionField("action", "string"), ExecutionField("accepted", "bool"), ExecutionField("queue_depth", "int")], owner="operations"),
            ExecutionModel("RunApprovalRequest", [ExecutionField("actor", "string"), ExecutionField("approval_token", "string"), ExecutionField("decision", "string"), ExecutionField("reason", "string", required=False)], owner="operations"),
            ExecutionModel("RunApprovalResponse", [ExecutionField("run_id", "string"), ExecutionField("status", "string"), ExecutionField("approved", "bool"), ExecutionField("required_follow_up", "string[]", required=False)], owner="operations"),
            ExecutionModel("RiskOverviewResponse", [ExecutionField("period", "string"), ExecutionField("high_risk_runs", "int"), ExecutionField("approval_required_runs", "int"), ExecutionField("risk_factors", "string[]"), ExecutionField("recommendation", "string")], owner="risk"),
            ExecutionModel("SlaOverviewResponse", [ExecutionField("period", "string"), ExecutionField("sla_target_minutes", "int"), ExecutionField("average_cycle_minutes", "float"), ExecutionField("sla_breach_count", "int"), ExecutionField("approval_queue_depth", "int")], owner="operations"),
            ExecutionModel("RegressionCenterResponse", [ExecutionField("baseline_version", "string"), ExecutionField("current_version", "string"), ExecutionField("regression_count", "int"), ExecutionField("improved_cases", "string[]", required=False), ExecutionField("regressions", "RegressionFinding[]", required=False)], owner="operations"),
            ExecutionModel("FlowCanvasResponse", [ExecutionField("run_id", "string"), ExecutionField("collaboration_mode", "string"), ExecutionField("departments", "string[]"), ExecutionField("required_approvals", "string[]", required=False), ExecutionField("billing_model", "string"), ExecutionField("recommendation", "string")], owner="orchestration"),
            ExecutionModel("BillingEntitlementsResponse", [ExecutionField("period", "string"), ExecutionField("tier", "string"), ExecutionField("billing_model_counts", "map<string,int>"), ExecutionField("upgrade_required_runs", "int"), ExecutionField("estimated_cost_usd", "float")], owner="orchestration"),
            ExecutionModel("BillingRunChargeResponse", [ExecutionField("run_id", "string"), ExecutionField("billing_model", "string"), ExecutionField("estimated_cost_usd", "float"), ExecutionField("overage_cost_usd", "float"), ExecutionField("upgrade_required", "bool")], owner="orchestration"),
        ],
        apis=[
            ExecutionApiSpec("get_operations_dashboard", "GET", "/operations/dashboard", "", "OperationsDashboardResponse", "operations.dashboard.read", ["operations.dashboard.viewed"], ["operations.dashboard.requests", "operations.dashboard.latency.ms"]),
            ExecutionApiSpec("get_run_detail", "GET", "/operations/runs/{run_id}", "", "RunDetailResponse", "operations.run.read", ["operations.run_detail.viewed"], ["operations.run_detail.requests", "operations.run_detail.latency.ms"]),
            ExecutionApiSpec("get_run_replay", "GET", "/operations/runs/{run_id}/replay", "", "RunReplayResponse", "operations.run.read", ["operations.run_replay.viewed"], ["operations.run_replay.requests", "operations.run_replay.latency.ms"]),
            ExecutionApiSpec("get_queue_control_center", "GET", "/operations/queue/control-center", "", "QueueControlCenterResponse", "operations.queue.read", ["operations.queue.viewed"], ["operations.queue.requests", "operations.queue.depth"]),
            ExecutionApiSpec("retry_queue_task", "POST", "/operations/queue/{task_id}/retry", "QueueActionRequest", "QueueActionResponse", "operations.queue.act", ["operations.queue.retry.requested"], ["operations.queue.retry.requests", "operations.queue.depth"]),
            ExecutionApiSpec("approve_run_execution", "POST", "/operations/runs/{run_id}/approve", "RunApprovalRequest", "RunApprovalResponse", "operations.run.approve", ["operations.run.approval.recorded"], ["operations.run.approval.requests", "operations.approval.queue.depth"]),
            ExecutionApiSpec("get_risk_overview", "GET", "/operations/risk/overview", "", "RiskOverviewResponse", "operations.risk.read", ["operations.risk.viewed"], ["operations.risk.requests", "operations.risk.high_runs"]),
            ExecutionApiSpec("get_sla_overview", "GET", "/operations/sla/overview", "", "SlaOverviewResponse", "operations.sla.read", ["operations.sla.viewed"], ["operations.sla.requests", "operations.sla.breaches"]),
            ExecutionApiSpec("get_regression_center", "GET", "/operations/regressions", "", "RegressionCenterResponse", "operations.regression.read", ["operations.regression.viewed"], ["operations.regression.requests", "operations.regression.count"]),
            ExecutionApiSpec("get_flow_canvas", "GET", "/operations/flows/{run_id}", "", "FlowCanvasResponse", "operations.flow.read", ["operations.flow.viewed"], ["operations.flow.requests", "operations.flow.handoff_count"]),
            ExecutionApiSpec("get_billing_entitlements", "GET", "/operations/billing/entitlements", "", "BillingEntitlementsResponse", "operations.billing.read", ["operations.billing.viewed"], ["operations.billing.requests", "operations.billing.estimated_cost_usd"]),
            ExecutionApiSpec("get_billing_run_charge", "GET", "/operations/billing/runs/{run_id}", "", "BillingRunChargeResponse", "operations.billing.read", ["operations.billing.run_charge.viewed"], ["operations.billing.run_charge.requests", "operations.billing.overage_cost_usd"]),
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
            ExecutionRole("eng-lead", ["Eng Lead"], ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read"], ["team", "workspace"], "vp-eng"),
            ExecutionRole("platform-admin", ["Platform Admin"], ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.billing.read"], ["workspace"], "vp-eng"),
            ExecutionRole("vp-eng", ["VP Eng"], ["operations.dashboard.read", "operations.run.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.billing.read"], ["portfolio", "workspace"], "none"),
            ExecutionRole("cross-team-operator", ["Cross-Team Operator"], ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.flow.read", "operations.billing.read"], ["cross-team", "team", "workspace"], "eng-lead"),
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
            AuditPolicy("operations.dashboard.viewed", ["actor", "period"], 180, "info"),
            AuditPolicy("operations.run_detail.viewed", ["actor", "run_id"], 180, "info"),
            AuditPolicy("operations.run_replay.viewed", ["actor", "run_id"], 180, "info"),
            AuditPolicy("operations.queue.viewed", ["actor", "queue_depth"], 180, "info"),
            AuditPolicy("operations.queue.retry.requested", ["actor", "task_id", "reason"], 180, "warning"),
            AuditPolicy("operations.run.approval.recorded", ["actor", "run_id", "decision"], 365, "warning"),
            AuditPolicy("operations.risk.viewed", ["actor", "period"], 180, "info"),
            AuditPolicy("operations.sla.viewed", ["actor", "period"], 180, "info"),
            AuditPolicy("operations.regression.viewed", ["actor", "current_version"], 180, "info"),
            AuditPolicy("operations.flow.viewed", ["actor", "run_id"], 180, "info"),
            AuditPolicy("operations.billing.viewed", ["actor", "period", "tier"], 365, "info"),
            AuditPolicy("operations.billing.run_charge.viewed", ["actor", "run_id", "billing_model"], 365, "info"),
        ],
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
    return [name for name in spec.required_fields if name not in details]


_VALID_WORKFLOW_STEP_KINDS = {
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
        invalid_steps = [step.kind for step in self.steps if step.kind not in _VALID_WORKFLOW_STEP_KINDS]
        if invalid_steps:
            joined = ", ".join(sorted(set(invalid_steps)))
            raise ValueError(f"invalid workflow step kind(s): {joined}")


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


@dataclass
class MemoryPattern:
    task_id: str
    title: str
    labels: List[str] = field(default_factory=list)
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    summary: str = ""

    def to_dict(self) -> Dict[str, Any]:
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
    def from_dict(cls, data: Dict[str, Any]) -> "MemoryPattern":
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
        ranked: List[Tuple[float, MemoryPattern]] = []
        for pattern in self._load_patterns():
            score = self._score(task, pattern)
            if score > 0:
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


@dataclass
class ValidationReportDecision:
    allowed_to_close: bool
    status: str
    summary: str
    missing_reports: List[str] = field(default_factory=list)


REQUIRED_REPORT_ARTIFACTS = ["task-run", "replay", "benchmark-suite"]


def enforce_validation_report_policy(artifacts: List[str]) -> ValidationReportDecision:
    existing = set(artifacts)
    missing = [name for name in REQUIRED_REPORT_ARTIFACTS if name not in existing]
    if missing:
        return ValidationReportDecision(False, "blocked", "validation report policy not satisfied", missing)
    return ValidationReportDecision(True, "ready", "validation report policy satisfied")


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


def issue_state_map(issues: Sequence[dict]) -> Dict[str, str]:
    state_map: Dict[str, str] = {}
    for issue in issues:
        identifier = str(issue.get("identifier", "")).strip()
        state = issue.get("state") or {}
        state_name = str(state.get("name", issue.get("state_name", ""))).strip()
        if identifier and state_name:
            state_map[identifier] = state_name
    return state_map


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
    args = append_missing_flag(list(forwarded), "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
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


def _now() -> str:
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
    created_at: str = field(default_factory=_now)
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
            created_at=str(data.get("created_at", _now())),
            metadata=dict(data.get("metadata", {})),
        )

    def to_collaboration_comment(self) -> Any:
        from .support_surfaces import CollaborationComment

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
        metadata: Optional[Dict[str, Any]] = None,
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


class RepoGatewayClient(Protocol):
    def push_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...
    def fetch_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...
    def list_commits(self, repo_space_id: str) -> List[Dict[str, Any]]: ...
    def get_commit(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...
    def get_children(self, repo_space_id: str, commit_hash: str) -> List[str]: ...
    def get_lineage(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...
    def get_leaves(self, repo_space_id: str, commit_hash: str) -> List[str]: ...
    def diff(self, repo_space_id: str, left_hash: str, right_hash: str) -> Dict[str, Any]: ...


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
        granted_permissions=[p.name for p in REPO_ACTION_PERMISSIONS],
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
    return [field_name for field_name in required if field_name not in payload]


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


VALID_ROLES = {"source", "candidate", "closeout", "accepted"}


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


def validate_roles(links: Iterable[RunCommitLink]) -> None:
    invalid = [link.role for link in links if link.role not in VALID_ROLES]
    if invalid:
        invalid_text = ", ".join(sorted(set(invalid)))
        raise ValueError(f"unsupported run commit roles: {invalid_text}")


def bind_run_commits(links: List[RunCommitLink]) -> RunCommitBinding:
    validate_roles(links)
    return RunCommitBinding(links=list(links))


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
            return f"{project_key.lower()}-{_slug(task.task_id)}"
        return space.default_channel_for_task(task.task_id)

    def resolve_agent(self, actor: str, role: str = "executor") -> RepoAgent:
        if actor in self.agents_by_actor:
            return self.agents_by_actor[actor]
        agent = RepoAgent(
            actor=actor,
            repo_agent_id=f"agent-{_slug(actor)}",
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


def _slug(value: str) -> str:
    cleaned = "".join(ch.lower() if ch.isalnum() else "-" for ch in value)
    return "-".join(part for part in cleaned.split("-") if part) or "agent"


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
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


CACHE_REMOTE = "cache"
BOOTSTRAP_BRANCH_PREFIX = "symphony"
DEFAULT_CACHE_BASE = Path("~/.cache/symphony/repos")
WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE = "~/.cache/symphony/repos"


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


def sanitize_issue_identifier(identifier: Optional[str]) -> str:
    raw = (identifier or "issue").strip() or "issue"
    return "".join(character if character.isalnum() or character in ".-_" else "_" for character in raw)


def bootstrap_branch_name(identifier: Optional[str]) -> str:
    return f"{BOOTSTRAP_BRANCH_PREFIX}/{sanitize_issue_identifier(identifier)}"


def default_cache_base(path: Optional[Path | str] = None) -> Path:
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


def repo_cache_key(repo_url: str, cache_key: Optional[str] = None) -> str:
    raw = (cache_key or normalize_repo_locator(repo_url)).strip().lower()
    sanitized = "".join(character if character.isalnum() or character in ".-_" else "-" for character in raw)
    compact = "-".join(segment for segment in sanitized.split("-") if segment)
    return compact or "repo"


def cache_root_for_repo(
    repo_url: str,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
) -> Path:
    return default_cache_base(cache_base) / repo_cache_key(repo_url, cache_key)


def resolve_cache_root(
    repo_url: str,
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
) -> Path:
    if cache_root is not None:
        return Path(cache_root).expanduser().resolve()
    return cache_root_for_repo(repo_url, cache_base=cache_base, cache_key=cache_key)


def default_cache_root(path: Optional[Path | str] = None) -> Path:
    return default_cache_base(path)


def _remove_path(path: Path) -> None:
    if path.is_dir() and not path.is_symlink():
        shutil.rmtree(path)
    elif path.exists() or path.is_symlink():
        path.unlink()


def _cache_state(
    repo_url: str,
    repo_cache_root: Path,
    cache_key: Optional[str] = None,
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
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
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
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
) -> CacheBootstrapState:
    cache_state = ensure_mirror(repo_url, cache_root=cache_root, cache_base=cache_base, cache_key=cache_key)
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
    workspace: Path | str,
    issue_identifier: Optional[str],
    repo_url: str,
    default_branch: str = "main",
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
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
    workspace: Path | str,
    issue_identifier: Optional[str],
    repo_url: str,
    default_branch: str = "main",
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
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


def build_parser(
    description: str,
    default_repo_url: str,
    default_branch: str,
    default_cache_root: Optional[str],
    default_cache_base: str,
    default_cache_key: Optional[str],
) -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=description)
    parser.add_argument("command", choices=["bootstrap", "cleanup"])
    parser.add_argument("--workspace", default=".", help="Workspace path managed by Symphony.")
    parser.add_argument("--issue", default="", help="Linear issue identifier used for the bootstrap branch.")
    parser.add_argument("--repo-url", default=default_repo_url, help="Canonical remote repository URL.")
    parser.add_argument("--default-branch", default=default_branch, help="Default branch used as the bootstrap base.")
    parser.add_argument("--cache-root", default=default_cache_root, help="Full cache root that contains mirror.git and seed. Overrides --cache-base/--cache-key.")
    parser.add_argument("--cache-base", default=default_cache_base, help="Base directory that stores per-repo cache roots.")
    parser.add_argument("--cache-key", default=default_cache_key, help="Optional stable key for the per-repo cache directory.")
    parser.add_argument("--json", action="store_true", help="Emit machine-readable JSON output.")
    return parser


def emit(payload: Dict[str, Any], as_json: bool) -> None:
    if as_json:
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return
    for key, value in payload.items():
        print(f"{key}={value}")


def main(
    argv: Optional[Sequence[str]] = None,
    *,
    description: str = "Bootstrap Symphony workspaces from a shared local mirror.",
    default_repo_url: str = "",
    default_branch: str = "main",
    default_cache_root: Optional[str] = None,
    default_cache_base: str = WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE,
    default_cache_key: Optional[str] = None,
) -> int:
    parser = build_parser(
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
        emit({"status": "ok", **status.to_dict()}, args.json)
        return 0
    except WorkspaceBootstrapError as exc:
        emit({"status": "error", "workspace": str(workspace), "error": str(exc)}, args.json)
        return 1


def build_validation_report(
    *,
    repo_url: str,
    workspace_root: Path | str,
    issue_identifiers: Sequence[str],
    default_branch: str = "main",
    cache_root: Optional[Path | str] = None,
    cache_base: Optional[Path | str] = None,
    cache_key: Optional[str] = None,
    cleanup: bool = True,
) -> Dict[str, Any]:
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
                result["workspace_mode"] in {"worktree_created", "workspace_reused"} for result in bootstrap_results
            ),
            "cleanup_preserved_cache": bool(bootstrap_results)
            and Path(bootstrap_results[0]["mirror_path"]).exists()
            and Path(bootstrap_results[0]["seed_path"]).joinpath(".git").exists(),
        },
    }


def render_validation_markdown(report: Dict[str, Any]) -> str:
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


def write_validation_report(report: Dict[str, Any], path: Path | str) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)
    if target.suffix.lower() == ".md":
        target.write_text(render_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target


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
            "missing_closeout_requirements": {issue_id: list(requirements) for issue_id, requirements in self.missing_closeout_requirements.items()},
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
            missing_closeout_requirements={str(issue_id): [str(requirement) for requirement in requirements] for issue_id, requirements in dict(data.get("missing_closeout_requirements", {})).items()},
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
        missing_closeout_requirements = {item.issue_id: item.missing_closeout_requirements for item in board.backlog_items if item.missing_closeout_requirements}
        invalid_scope_statuses = sorted(item.issue_id for item in board.backlog_items if item.scope_status not in ALLOWED_SCOPE_STATUSES)
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
            lines.append(f"- {item.issue_id}: phase={item.phase} owner={item.owner or 'none'} status={item.status} scope={item.scope_status} closeout={closeout}")
    else:
        lines.append("- None")
    lines.extend(["", "## Freeze Exceptions", ""])
    if board.exceptions:
        for exception in board.exceptions:
            lines.append(f"- {exception.issue_id}: approved_by={exception.approved_by or 'pending'} reason={exception.reason or 'none'}")
    else:
        lines.append("- None")
    lines.extend(["", "## Audit", ""])
    lines.append(f"- Duplicate issues: {', '.join(audit.duplicate_issue_ids) if audit.duplicate_issue_ids else 'none'}")
    lines.append(f"- Missing owners: {', '.join(audit.missing_owners) if audit.missing_owners else 'none'}")
    lines.append(f"- Missing acceptance: {', '.join(audit.missing_acceptance) if audit.missing_acceptance else 'none'}")
    lines.append(f"- Missing validation: {', '.join(audit.missing_validation) if audit.missing_validation else 'none'}")
    if audit.missing_closeout_requirements:
        missing_closeout = "; ".join(f"{issue_id}={', '.join(requirements)}" for issue_id, requirements in sorted(audit.missing_closeout_requirements.items()))
    else:
        missing_closeout = "none"
    lines.append(f"- Missing closeout requirements: {missing_closeout}")
    lines.append(f"- Unauthorized scope changes: {', '.join(audit.unauthorized_scope_changes) if audit.unauthorized_scope_changes else 'none'}")
    lines.append(f"- Invalid scope statuses: {', '.join(audit.invalid_scope_statuses) if audit.invalid_scope_statuses else 'none'}")
    lines.append(f"- Unapproved exceptions: {', '.join(audit.unapproved_exceptions) if audit.unapproved_exceptions else 'none'}")
    return "\n".join(lines) + "\n"


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
        return f"{status}: findings={self.finding_count} missing_owners={len(self.missing_owners)} invalid_priorities={len(self.invalid_priorities)} invalid_categories={len(self.invalid_categories)} unresolved_p0={len(self.unresolved_p0_findings)}"

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
        ready = bool(archive.findings) and not (missing_owners or invalid_priorities or invalid_categories or unresolved_p0_findings)
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
        f"- Priority Counts: P0={audit.priority_counts.get('P0', 0)} P1={audit.priority_counts.get('P1', 0)} P2={audit.priority_counts.get('P2', 0)}",
        f"- Category Counts: ui={audit.category_counts.get('ui', 0)} ia={audit.category_counts.get('ia', 0)} permission={audit.category_counts.get('permission', 0)} metric={audit.category_counts.get('metric', 0)}",
        "",
        "## Findings",
    ]
    for finding in archive.findings:
        lines.append(f"- {finding.finding_id}: {finding.summary} category={finding.normalized_category} priority={finding.normalized_priority} owner={finding.owner or 'none'} status={finding.status}")
        lines.append(f"  surface={finding.surface or 'none'} impact={finding.impact or 'none'} evidence={','.join(finding.evidence) or 'none'}")
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
    def from_execution(cls, task: Task, run_id: str, record: Any) -> "ReplayRecord":
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
    record: Any
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
    def __init__(self, scheduler: Optional[Any] = None, storage_dir: Optional[str] = None):
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

    def _evaluate(self, case: BenchmarkCase, record: Any) -> List[EvaluationCriterion]:
        return [
            self._criterion("decision-medium", 40, case.expected_medium, record.decision.medium),
            self._criterion("approval-gate", 30, case.expected_approved, record.decision.approved),
            self._criterion("final-status", 20, case.expected_status, record.run.status),
            EvaluationCriterion(
                name="report-artifact",
                weight=10,
                passed=(not case.require_report) or bool(record.report_path),
                detail="report emitted" if (not case.require_report) or bool(record.report_path) else "report missing",
            ),
        ]

    def _criterion(self, name: str, weight: int, expected: Optional[object], actual: object) -> EvaluationCriterion:
        if expected is None:
            return EvaluationCriterion(name=name, weight=weight, passed=True, detail="not asserted")
        passed = expected == actual
        return EvaluationCriterion(name=name, weight=weight, passed=passed, detail=f"expected {expected} got {actual}")

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
            RunDetailTab("timeline", "Timeline / Log Sync", render_timeline_panel("Timeline / Log Sync", "Field-by-field replay comparison with a synced inspector for each expectation and mismatch.", timeline_events)),
            RunDetailTab("comparison", "Split View", comparison_html),
            RunDetailTab("replay", "Replay", mismatch_html),
            RunDetailTab("reports", "Reports", render_resource_grid("Reports", "Replay detail pages do not emit standalone report files beyond the generated HTML page unless the caller persists additional artifacts.", [])),
        ],
        timeline_events=timeline_events,
    )


def render_run_replay_index_page(
    case_id: str,
    record: Any,
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
            RunDetailTab("timeline", "Timeline / Log Sync", render_timeline_panel("Timeline / Log Sync", "Run logs, trace spans, audits, acceptance checks, and replay mismatches are merged into one synced timeline inspector.", run_events)),
            RunDetailTab("acceptance", "Acceptance", acceptance_html),
            RunDetailTab("artifacts", "Artifacts", render_resource_grid("Artifacts", "Generated reports and pages emitted for benchmark review and replay inspection.", execution_resources)),
            RunDetailTab("reports", "Reports", render_resource_grid("Reports", "Report-first view for markdown output and linked run/replay pages.", [resource for resource in execution_resources if resource.kind == "report" or resource.name.endswith("page")])),
            RunDetailTab("replay", "Replay", replay_html),
        ],
        timeline_events=run_events,
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
        return sorted(self.candidates, key=lambda candidate: (-candidate.readiness_score, candidate.candidate_id))

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
        missing_capabilities = [capability for capability in gate.required_capabilities if capability not in provided_capabilities]
        missing_evidence = [item for item in gate.required_evidence if item not in provided_evidence]
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

    def _baseline_findings(self, gate: EntryGate, baseline_audit: Optional[ScopeFreezeAudit]) -> List[str]:
        if not gate.required_baseline_version:
            return []
        if baseline_audit is None:
            return [f"missing baseline audit for {gate.required_baseline_version}"]
        findings: List[str] = []
        if baseline_audit.version != gate.required_baseline_version:
            findings.append(f"baseline version mismatch: expected {gate.required_baseline_version}, got {baseline_audit.version}")
        if not baseline_audit.release_ready:
            findings.append(f"baseline {baseline_audit.version} is not release ready ({baseline_audit.readiness_score:.1f})")
        return findings


def render_candidate_backlog_report(backlog: CandidateBacklog, gate: EntryGate, decision: EntryGateDecision) -> str:
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
                validation_command="PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_ui_review.py -q",
                capabilities=["release-gate", "console-shell", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink("design-system-audit", "src/bigclaw/design_system.py", "release-gate", "component inventory, accessibility, and UI acceptance coverage"),
                    EvidenceLink("console-ia-contract", "src/bigclaw/console_ia.py", "release-gate", "global navigation, top bar, filters, and state contracts"),
                    EvidenceLink("ui-review-pack", "src/bigclaw/ui_review.py", "release-gate", "review objectives, wireframes, interaction coverage, and open questions"),
                    EvidenceLink("ui-acceptance-tests", "tests/test_design_system.py", "release-gate", "role-permission, data accuracy, and performance audits"),
                    EvidenceLink("console-shell-tests", "tests/test_console_ia.py", "release-gate", "console shell and interaction draft release readiness"),
                    EvidenceLink("review-pack-tests", "tests/test_ui_review.py", "release-gate", "deterministic review packet validation"),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Operations command-center hardening",
                theme="ops-command-center",
                priority="P0",
                owner="engineering-operations",
                outcome="Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
                validation_command="PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_operations.py tests/test_saved_views.py tests/test_evaluation.py -q && (cd bigclaw-go && go test ./internal/worker ./internal/workflow ./internal/scheduler)",
                capabilities=["ops-control", "saved-views", "rollback-simulation"],
                evidence=["weekly-review", "validation-report"],
                evidence_links=[
                    EvidenceLink("command-center-src", "src/bigclaw/operations.py", "ops-control", "queue control center, dashboard builder, weekly review, and regression surfaces"),
                    EvidenceLink("command-center-tests", "tests/test_control_center.py", "ops-control", "queue control center validation"),
                    EvidenceLink("operations-tests", "tests/test_operations.py", "ops-control", "dashboard, weekly report, regression, and version-center coverage"),
                    EvidenceLink("approval-contract", "src/bigclaw/execution_contract.py", "ops-control", "approval permission and API role coverage contract"),
                    EvidenceLink("approval-workflow", "src/bigclaw/workflow.py", "ops-control", "approval workflow and closeout flow wiring"),
                    EvidenceLink("workflow-tests", "bigclaw-go/internal/workflow/engine_test.go", "ops-control", "acceptance gate and workpad journal validation"),
                    EvidenceLink("execution-flow-tests", "bigclaw-go/internal/worker/runtime_test.go", "ops-control", "execution handoff, closeout, and routed runtime evidence"),
                    EvidenceLink("saved-views-src", "src/bigclaw/saved_views.py", "saved-views", "saved views, digest subscriptions, and governed filters"),
                    EvidenceLink("saved-views-tests", "tests/test_saved_views.py", "saved-views", "saved-view audit coverage"),
                    EvidenceLink("simulation-src", "src/bigclaw/evaluation.py", "rollback-simulation", "simulation, replay, and comparison evidence"),
                    EvidenceLink("simulation-tests", "tests/test_evaluation.py", "rollback-simulation", "replay and benchmark validation"),
                ],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration-rollout",
                title="Agent orchestration rollout",
                theme="agent-orchestration",
                priority="P0",
                owner="orchestration-office",
                outcome="Carry entitlement-aware orchestration, handoff visibility, and commercialization proof into a candidate ready for release review.",
                validation_command="PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_reports.py -q",
                capabilities=["commercialization", "handoff", "pilot-rollout"],
                evidence=["pilot-evidence", "validation-report"],
                evidence_links=[
                    EvidenceLink("orchestration-plan-src", "src/bigclaw/orchestration.py", "commercialization", "cross-team orchestration, entitlement-aware policy, and handoff decisions"),
                    EvidenceLink("orchestration-report-src", "src/bigclaw/reports.py", "commercialization", "orchestration canvas, portfolio rollups, and narrative exports"),
                    EvidenceLink("orchestration-tests", "tests/test_orchestration.py", "commercialization", "handoff and policy decision validation"),
                    EvidenceLink("report-studio-tests", "tests/test_reports.py", "commercialization", "report exports and downstream evidence sharing"),
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
                exit_criteria=["Scope freeze board published", "Owners and validation commands assigned for all streams"],
                deliverables=["Execution baseline report", "Scope freeze audit snapshot"],
                goals=[
                    WeeklyGoal("w1-scope-freeze", "Lock the v4.0 scope and escalation path", "program-office", "done", "frozen backlog items", "5 epics aligned", "5 epics aligned"),
                    WeeklyGoal("w1-validation-matrix", "Assign validation commands and evidence owners", "engineering-ops", "done", "streams with validation owners", "5/5 streams", "5/5 streams"),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=2,
                theme="Build and integration",
                objective="Land the highest-risk implementation slices and wire cross-team dependencies.",
                exit_criteria=["P0 build items merged", "Cross-team dependency review completed"],
                deliverables=["Integrated build checkpoint", "Dependency burn-down"],
                goals=[
                    WeeklyGoal("w2-p0-burndown", "Close the top P0 implementation gaps", "engineering-platform", "on-track", "P0 items merged", ">=3 merged", "2 merged"),
                    WeeklyGoal("w2-handoff-sync", "Resolve orchestration and console handoff dependencies", "orchestration-office", "at-risk", "open handoff blockers", "0 blockers", "1 blocker", ["w2-p0-burndown"], ["console entitlement contract is pending"]),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=3,
                theme="Stabilization and validation",
                objective="Drive regression triage, benchmark replay, and release-readiness evidence.",
                exit_criteria=["Regression backlog under control threshold", "Benchmark comparison published"],
                deliverables=["Stabilization report", "Benchmark replay pack"],
                goals=[
                    WeeklyGoal("w3-regression-triage", "Reduce critical regressions before release gate", "quality-ops", "not-started", "critical regressions", "<=2 open"),
                    WeeklyGoal("w3-benchmark-pack", "Publish replay and weighted benchmark evidence", "evaluation-lab", "not-started", "benchmark evidence bundle", "1 bundle published"),
                ],
            ),
            WeeklyExecutionPlan(
                week_number=4,
                theme="Launch decision and weekly operating rhythm",
                objective="Convert validation evidence into launch readiness and the post-launch weekly review cadence.",
                exit_criteria=["Launch decision signed off", "Weekly operating review template adopted"],
                deliverables=["Launch readiness packet", "Weekly review operating template"],
                goals=[
                    WeeklyGoal("w4-launch-decision", "Complete launch readiness review", "release-governance", "not-started", "required sign-offs", "all sign-offs complete"),
                    WeeklyGoal("w4-weekly-rhythm", "Roll out the weekly KPI and issue review cadence", "engineering-operations", "not-started", "weekly review adoption", "1 recurring cadence active"),
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


def evaluate_candidate_gate(*, gate_decision: EntryGateDecision, rollout_scorecard: Dict[str, object]) -> Dict[str, object]:
    readiness = bool(gate_decision.passed)
    rollout_ready = rollout_scorecard.get("recommendation") == "go"
    recommendation = "enable-by-default" if readiness and rollout_ready else "pilot-only"
    findings: List[str] = []
    if not readiness:
        findings.append(gate_decision.summary)
    if not rollout_ready:
        findings.append("rollout score below threshold" f" ({rollout_scorecard.get('rollout_score', 'n/a')})")
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
                f"- Week {week.week_number}: {week.theme} progress={week.completed_goals}/{week.total_goals} ({week.progress_percent}%)",
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
            lines.append("    " f"dependencies={','.join(goal.dependencies) or 'none'} " f"risks={','.join(goal.risks) or 'none'}")
    return "\n".join(lines)


@dataclass(frozen=True)
class ReviewObjective:
    objective_id: str
    title: str
    persona: str
    outcome: str
    success_signal: str
    priority: str = "P1"
    dependencies: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "objective_id": self.objective_id,
            "title": self.title,
            "persona": self.persona,
            "outcome": self.outcome,
            "success_signal": self.success_signal,
            "priority": self.priority,
            "dependencies": list(self.dependencies),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewObjective":
        return cls(
            objective_id=str(data["objective_id"]),
            title=str(data["title"]),
            persona=str(data["persona"]),
            outcome=str(data["outcome"]),
            success_signal=str(data["success_signal"]),
            priority=str(data.get("priority", "P1")),
            dependencies=[str(item) for item in data.get("dependencies", [])],
        )


@dataclass(frozen=True)
class WireframeSurface:
    surface_id: str
    name: str
    device: str
    entry_point: str
    primary_blocks: List[str] = field(default_factory=list)
    review_notes: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "surface_id": self.surface_id,
            "name": self.name,
            "device": self.device,
            "entry_point": self.entry_point,
            "primary_blocks": list(self.primary_blocks),
            "review_notes": list(self.review_notes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "WireframeSurface":
        return cls(
            surface_id=str(data["surface_id"]),
            name=str(data["name"]),
            device=str(data["device"]),
            entry_point=str(data["entry_point"]),
            primary_blocks=[str(item) for item in data.get("primary_blocks", [])],
            review_notes=[str(item) for item in data.get("review_notes", [])],
        )


@dataclass(frozen=True)
class InteractionFlow:
    flow_id: str
    name: str
    trigger: str
    system_response: str
    states: List[str] = field(default_factory=list)
    exceptions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "flow_id": self.flow_id,
            "name": self.name,
            "trigger": self.trigger,
            "system_response": self.system_response,
            "states": list(self.states),
            "exceptions": list(self.exceptions),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "InteractionFlow":
        return cls(
            flow_id=str(data["flow_id"]),
            name=str(data["name"]),
            trigger=str(data["trigger"]),
            system_response=str(data["system_response"]),
            states=[str(item) for item in data.get("states", [])],
            exceptions=[str(item) for item in data.get("exceptions", [])],
        )


@dataclass(frozen=True)
class OpenQuestion:
    question_id: str
    theme: str
    question: str
    owner: str
    impact: str
    status: str = "open"

    def to_dict(self) -> Dict[str, object]:
        return {
            "question_id": self.question_id,
            "theme": self.theme,
            "question": self.question,
            "owner": self.owner,
            "impact": self.impact,
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "OpenQuestion":
        return cls(
            question_id=str(data["question_id"]),
            theme=str(data["theme"]),
            question=str(data["question"]),
            owner=str(data["owner"]),
            impact=str(data["impact"]),
            status=str(data.get("status", "open")),
        )


@dataclass(frozen=True)
class ReviewerChecklistItem:
    item_id: str
    surface_id: str
    prompt: str
    owner: str
    status: str = "todo"
    evidence_links: List[str] = field(default_factory=list)
    notes: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "item_id": self.item_id,
            "surface_id": self.surface_id,
            "prompt": self.prompt,
            "owner": self.owner,
            "status": self.status,
            "evidence_links": list(self.evidence_links),
            "notes": self.notes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewerChecklistItem":
        return cls(
            item_id=str(data["item_id"]),
            surface_id=str(data["surface_id"]),
            prompt=str(data["prompt"]),
            owner=str(data["owner"]),
            status=str(data.get("status", "todo")),
            evidence_links=[str(item) for item in data.get("evidence_links", [])],
            notes=str(data.get("notes", "")),
        )


@dataclass(frozen=True)
class ReviewDecision:
    decision_id: str
    surface_id: str
    owner: str
    summary: str
    rationale: str
    status: str = "proposed"
    follow_up: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "decision_id": self.decision_id,
            "surface_id": self.surface_id,
            "owner": self.owner,
            "summary": self.summary,
            "rationale": self.rationale,
            "status": self.status,
            "follow_up": self.follow_up,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewDecision":
        return cls(
            decision_id=str(data["decision_id"]),
            surface_id=str(data["surface_id"]),
            owner=str(data["owner"]),
            summary=str(data["summary"]),
            rationale=str(data["rationale"]),
            status=str(data.get("status", "proposed")),
            follow_up=str(data.get("follow_up", "")),
        )


@dataclass(frozen=True)
class ReviewRoleAssignment:
    assignment_id: str
    surface_id: str
    role: str
    responsibilities: List[str] = field(default_factory=list)
    checklist_item_ids: List[str] = field(default_factory=list)
    decision_ids: List[str] = field(default_factory=list)
    status: str = "planned"

    def to_dict(self) -> Dict[str, object]:
        return {
            "assignment_id": self.assignment_id,
            "surface_id": self.surface_id,
            "role": self.role,
            "responsibilities": list(self.responsibilities),
            "checklist_item_ids": list(self.checklist_item_ids),
            "decision_ids": list(self.decision_ids),
            "status": self.status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewRoleAssignment":
        return cls(
            assignment_id=str(data["assignment_id"]),
            surface_id=str(data["surface_id"]),
            role=str(data["role"]),
            responsibilities=[str(item) for item in data.get("responsibilities", [])],
            checklist_item_ids=[str(item) for item in data.get("checklist_item_ids", [])],
            decision_ids=[str(item) for item in data.get("decision_ids", [])],
            status=str(data.get("status", "planned")),
        )


@dataclass(frozen=True)
class ReviewSignoff:
    signoff_id: str
    assignment_id: str
    surface_id: str
    role: str
    status: str = "pending"
    required: bool = True
    evidence_links: List[str] = field(default_factory=list)
    notes: str = ""
    waiver_owner: str = ""
    waiver_reason: str = ""
    requested_at: str = ""
    due_at: str = ""
    escalation_owner: str = ""
    sla_status: str = "on-track"
    reminder_owner: str = ""
    reminder_channel: str = ""
    last_reminder_at: str = ""
    next_reminder_at: str = ""
    reminder_cadence: str = ""
    reminder_status: str = "scheduled"

    def to_dict(self) -> Dict[str, object]:
        return {
            "signoff_id": self.signoff_id,
            "assignment_id": self.assignment_id,
            "surface_id": self.surface_id,
            "role": self.role,
            "status": self.status,
            "required": self.required,
            "evidence_links": list(self.evidence_links),
            "notes": self.notes,
            "waiver_owner": self.waiver_owner,
            "waiver_reason": self.waiver_reason,
            "requested_at": self.requested_at,
            "due_at": self.due_at,
            "escalation_owner": self.escalation_owner,
            "sla_status": self.sla_status,
            "reminder_owner": self.reminder_owner,
            "reminder_channel": self.reminder_channel,
            "last_reminder_at": self.last_reminder_at,
            "next_reminder_at": self.next_reminder_at,
            "reminder_cadence": self.reminder_cadence,
            "reminder_status": self.reminder_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewSignoff":
        return cls(
            signoff_id=str(data["signoff_id"]),
            assignment_id=str(data["assignment_id"]),
            surface_id=str(data["surface_id"]),
            role=str(data["role"]),
            status=str(data.get("status", "pending")),
            required=bool(data.get("required", True)),
            evidence_links=[str(item) for item in data.get("evidence_links", [])],
            notes=str(data.get("notes", "")),
            waiver_owner=str(data.get("waiver_owner", "")),
            waiver_reason=str(data.get("waiver_reason", "")),
            requested_at=str(data.get("requested_at", "")),
            due_at=str(data.get("due_at", "")),
            escalation_owner=str(data.get("escalation_owner", "")),
            sla_status=str(data.get("sla_status", "on-track")),
            reminder_owner=str(data.get("reminder_owner", "")),
            reminder_channel=str(data.get("reminder_channel", "")),
            last_reminder_at=str(data.get("last_reminder_at", "")),
            next_reminder_at=str(data.get("next_reminder_at", "")),
            reminder_cadence=str(data.get("reminder_cadence", "")),
            reminder_status=str(data.get("reminder_status", "scheduled")),
        )


@dataclass(frozen=True)
class ReviewBlocker:
    blocker_id: str
    surface_id: str
    signoff_id: str
    owner: str
    summary: str
    status: str = "open"
    severity: str = "medium"
    escalation_owner: str = ""
    next_action: str = ""
    freeze_exception: bool = False
    freeze_owner: str = ""
    freeze_until: str = ""
    freeze_reason: str = ""
    freeze_approved_by: str = ""
    freeze_approved_at: str = ""
    freeze_renewal_owner: str = ""
    freeze_renewal_by: str = ""
    freeze_renewal_status: str = "not-needed"

    def to_dict(self) -> Dict[str, object]:
        return {
            "blocker_id": self.blocker_id,
            "surface_id": self.surface_id,
            "signoff_id": self.signoff_id,
            "owner": self.owner,
            "summary": self.summary,
            "status": self.status,
            "severity": self.severity,
            "escalation_owner": self.escalation_owner,
            "next_action": self.next_action,
            "freeze_exception": self.freeze_exception,
            "freeze_owner": self.freeze_owner,
            "freeze_until": self.freeze_until,
            "freeze_reason": self.freeze_reason,
            "freeze_approved_by": self.freeze_approved_by,
            "freeze_approved_at": self.freeze_approved_at,
            "freeze_renewal_owner": self.freeze_renewal_owner,
            "freeze_renewal_by": self.freeze_renewal_by,
            "freeze_renewal_status": self.freeze_renewal_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlocker":
        return cls(
            blocker_id=str(data["blocker_id"]),
            surface_id=str(data["surface_id"]),
            signoff_id=str(data["signoff_id"]),
            owner=str(data["owner"]),
            summary=str(data["summary"]),
            status=str(data.get("status", "open")),
            severity=str(data.get("severity", "medium")),
            escalation_owner=str(data.get("escalation_owner", "")),
            next_action=str(data.get("next_action", "")),
            freeze_exception=bool(data.get("freeze_exception", False)),
            freeze_owner=str(data.get("freeze_owner", "")),
            freeze_until=str(data.get("freeze_until", "")),
            freeze_reason=str(data.get("freeze_reason", "")),
            freeze_approved_by=str(data.get("freeze_approved_by", "")),
            freeze_approved_at=str(data.get("freeze_approved_at", "")),
            freeze_renewal_owner=str(data.get("freeze_renewal_owner", "")),
            freeze_renewal_by=str(data.get("freeze_renewal_by", "")),
            freeze_renewal_status=str(data.get("freeze_renewal_status", "not-needed")),
        )


@dataclass(frozen=True)
class ReviewBlockerEvent:
    event_id: str
    blocker_id: str
    actor: str
    status: str
    summary: str
    timestamp: str
    next_action: str = ""
    handoff_from: str = ""
    handoff_to: str = ""
    channel: str = ""
    artifact_ref: str = ""
    ack_owner: str = ""
    ack_at: str = ""
    ack_status: str = "pending"

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_id": self.event_id,
            "blocker_id": self.blocker_id,
            "actor": self.actor,
            "status": self.status,
            "summary": self.summary,
            "timestamp": self.timestamp,
            "next_action": self.next_action,
            "handoff_from": self.handoff_from,
            "handoff_to": self.handoff_to,
            "channel": self.channel,
            "artifact_ref": self.artifact_ref,
            "ack_owner": self.ack_owner,
            "ack_at": self.ack_at,
            "ack_status": self.ack_status,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlockerEvent":
        return cls(
            event_id=str(data["event_id"]),
            blocker_id=str(data["blocker_id"]),
            actor=str(data["actor"]),
            status=str(data["status"]),
            summary=str(data["summary"]),
            timestamp=str(data["timestamp"]),
            next_action=str(data.get("next_action", "")),
            handoff_from=str(data.get("handoff_from", "")),
            handoff_to=str(data.get("handoff_to", "")),
            channel=str(data.get("channel", "")),
            artifact_ref=str(data.get("artifact_ref", "")),
            ack_owner=str(data.get("ack_owner", "")),
            ack_at=str(data.get("ack_at", "")),
            ack_status=str(data.get("ack_status", "pending")),
        )


@dataclass(frozen=True)
class UIReviewPackArtifacts:
    root_dir: str
    markdown_path: str
    html_path: str
    decision_log_path: str
    review_summary_board_path: str
    objective_coverage_board_path: str
    persona_readiness_board_path: str
    wireframe_readiness_board_path: str
    interaction_coverage_board_path: str
    open_question_tracker_path: str
    checklist_traceability_board_path: str
    decision_followup_tracker_path: str
    role_matrix_path: str
    role_coverage_board_path: str
    signoff_dependency_board_path: str
    signoff_log_path: str
    signoff_sla_dashboard_path: str
    signoff_reminder_queue_path: str
    reminder_cadence_board_path: str
    signoff_breach_board_path: str
    escalation_dashboard_path: str
    escalation_handoff_ledger_path: str
    handoff_ack_ledger_path: str
    owner_escalation_digest_path: str
    owner_workload_board_path: str
    blocker_log_path: str
    blocker_timeline_path: str
    freeze_exception_board_path: str
    freeze_approval_trail_path: str
    freeze_renewal_tracker_path: str
    exception_log_path: str
    exception_matrix_path: str
    audit_density_board_path: str
    owner_review_queue_path: str
    blocker_timeline_summary_path: str


@dataclass
class UIReviewPack:
    issue_id: str
    title: str
    version: str
    objectives: List[ReviewObjective] = field(default_factory=list)
    wireframes: List[WireframeSurface] = field(default_factory=list)
    interactions: List[InteractionFlow] = field(default_factory=list)
    open_questions: List[OpenQuestion] = field(default_factory=list)
    reviewer_checklist: List[ReviewerChecklistItem] = field(default_factory=list)
    requires_reviewer_checklist: bool = False
    decision_log: List[ReviewDecision] = field(default_factory=list)
    requires_decision_log: bool = False
    role_matrix: List[ReviewRoleAssignment] = field(default_factory=list)
    requires_role_matrix: bool = False
    signoff_log: List[ReviewSignoff] = field(default_factory=list)
    requires_signoff_log: bool = False
    blocker_log: List[ReviewBlocker] = field(default_factory=list)
    requires_blocker_log: bool = False
    blocker_timeline: List[ReviewBlockerEvent] = field(default_factory=list)
    requires_blocker_timeline: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "version": self.version,
            "objectives": [objective.to_dict() for objective in self.objectives],
            "wireframes": [wireframe.to_dict() for wireframe in self.wireframes],
            "interactions": [interaction.to_dict() for interaction in self.interactions],
            "open_questions": [question.to_dict() for question in self.open_questions],
            "reviewer_checklist": [item.to_dict() for item in self.reviewer_checklist],
            "requires_reviewer_checklist": self.requires_reviewer_checklist,
            "decision_log": [decision.to_dict() for decision in self.decision_log],
            "requires_decision_log": self.requires_decision_log,
            "role_matrix": [assignment.to_dict() for assignment in self.role_matrix],
            "requires_role_matrix": self.requires_role_matrix,
            "signoff_log": [signoff.to_dict() for signoff in self.signoff_log],
            "requires_signoff_log": self.requires_signoff_log,
            "blocker_log": [blocker.to_dict() for blocker in self.blocker_log],
            "requires_blocker_log": self.requires_blocker_log,
            "blocker_timeline": [event.to_dict() for event in self.blocker_timeline],
            "requires_blocker_timeline": self.requires_blocker_timeline,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "UIReviewPack":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            objectives=[ReviewObjective.from_dict(item) for item in data.get("objectives", [])],
            wireframes=[WireframeSurface.from_dict(item) for item in data.get("wireframes", [])],
            interactions=[InteractionFlow.from_dict(item) for item in data.get("interactions", [])],
            open_questions=[OpenQuestion.from_dict(item) for item in data.get("open_questions", [])],
            reviewer_checklist=[ReviewerChecklistItem.from_dict(item) for item in data.get("reviewer_checklist", [])],
            requires_reviewer_checklist=bool(data.get("requires_reviewer_checklist", False)),
            decision_log=[ReviewDecision.from_dict(item) for item in data.get("decision_log", [])],
            requires_decision_log=bool(data.get("requires_decision_log", False)),
            role_matrix=[ReviewRoleAssignment.from_dict(item) for item in data.get("role_matrix", [])],
            requires_role_matrix=bool(data.get("requires_role_matrix", False)),
            signoff_log=[ReviewSignoff.from_dict(item) for item in data.get("signoff_log", [])],
            requires_signoff_log=bool(data.get("requires_signoff_log", False)),
            blocker_log=[ReviewBlocker.from_dict(item) for item in data.get("blocker_log", [])],
            requires_blocker_log=bool(data.get("requires_blocker_log", False)),
            blocker_timeline=[ReviewBlockerEvent.from_dict(item) for item in data.get("blocker_timeline", [])],
            requires_blocker_timeline=bool(data.get("requires_blocker_timeline", False)),
        )


@dataclass(frozen=True)
class UIReviewPackAudit:
    ready: bool
    objective_count: int
    wireframe_count: int
    interaction_count: int
    open_question_count: int
    checklist_count: int = 0
    decision_count: int = 0
    role_assignment_count: int = 0
    signoff_count: int = 0
    blocker_count: int = 0
    blocker_timeline_count: int = 0
    missing_sections: List[str] = field(default_factory=list)
    objectives_missing_signals: List[str] = field(default_factory=list)
    wireframes_missing_blocks: List[str] = field(default_factory=list)
    interactions_missing_states: List[str] = field(default_factory=list)
    unresolved_question_ids: List[str] = field(default_factory=list)
    wireframes_missing_checklists: List[str] = field(default_factory=list)
    orphan_checklist_surfaces: List[str] = field(default_factory=list)
    checklist_items_missing_evidence: List[str] = field(default_factory=list)
    checklist_items_missing_role_links: List[str] = field(default_factory=list)
    wireframes_missing_decisions: List[str] = field(default_factory=list)
    orphan_decision_surfaces: List[str] = field(default_factory=list)
    unresolved_decision_ids: List[str] = field(default_factory=list)
    unresolved_decisions_missing_follow_ups: List[str] = field(default_factory=list)
    wireframes_missing_role_assignments: List[str] = field(default_factory=list)
    orphan_role_assignment_surfaces: List[str] = field(default_factory=list)
    role_assignments_missing_responsibilities: List[str] = field(default_factory=list)
    role_assignments_missing_checklist_links: List[str] = field(default_factory=list)
    role_assignments_missing_decision_links: List[str] = field(default_factory=list)
    decisions_missing_role_links: List[str] = field(default_factory=list)
    wireframes_missing_signoffs: List[str] = field(default_factory=list)
    orphan_signoff_surfaces: List[str] = field(default_factory=list)
    signoffs_missing_assignments: List[str] = field(default_factory=list)
    signoffs_missing_evidence: List[str] = field(default_factory=list)
    signoffs_missing_requested_dates: List[str] = field(default_factory=list)
    signoffs_missing_due_dates: List[str] = field(default_factory=list)
    signoffs_missing_escalation_owners: List[str] = field(default_factory=list)
    signoffs_missing_reminder_owners: List[str] = field(default_factory=list)
    signoffs_missing_next_reminders: List[str] = field(default_factory=list)
    signoffs_missing_reminder_cadence: List[str] = field(default_factory=list)
    signoffs_with_breached_sla: List[str] = field(default_factory=list)
    waived_signoffs_missing_metadata: List[str] = field(default_factory=list)
    unresolved_required_signoff_ids: List[str] = field(default_factory=list)
    blockers_missing_signoff_links: List[str] = field(default_factory=list)
    blockers_missing_escalation_owners: List[str] = field(default_factory=list)
    blockers_missing_next_actions: List[str] = field(default_factory=list)
    freeze_exceptions_missing_owners: List[str] = field(default_factory=list)
    freeze_exceptions_missing_until: List[str] = field(default_factory=list)
    freeze_exceptions_missing_approvers: List[str] = field(default_factory=list)
    freeze_exceptions_missing_approval_dates: List[str] = field(default_factory=list)
    freeze_exceptions_missing_renewal_owners: List[str] = field(default_factory=list)
    freeze_exceptions_missing_renewal_dates: List[str] = field(default_factory=list)
    blockers_missing_timeline_events: List[str] = field(default_factory=list)
    closed_blockers_missing_resolution_events: List[str] = field(default_factory=list)
    orphan_blocker_surfaces: List[str] = field(default_factory=list)
    orphan_blocker_timeline_blocker_ids: List[str] = field(default_factory=list)
    handoff_events_missing_targets: List[str] = field(default_factory=list)
    handoff_events_missing_artifacts: List[str] = field(default_factory=list)
    handoff_events_missing_ack_owners: List[str] = field(default_factory=list)
    handoff_events_missing_ack_dates: List[str] = field(default_factory=list)
    unresolved_required_signoffs_without_blockers: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: objectives={self.objective_count} "
            f"wireframes={self.wireframe_count} "
            f"interactions={self.interaction_count} "
            f"open_questions={self.open_question_count} "
            f"checklist={self.checklist_count} "
            f"decisions={self.decision_count} "
            f"role_assignments={self.role_assignment_count} "
            f"signoffs={self.signoff_count} "
            f"blockers={self.blocker_count} "
            f"timeline_events={self.blocker_timeline_count}"
        )


class UIReviewPackAuditor:
    def audit(self, pack: UIReviewPack) -> UIReviewPackAudit:
        missing_sections = []
        if not pack.objectives:
            missing_sections.append("objectives")
        if not pack.wireframes:
            missing_sections.append("wireframes")
        if not pack.interactions:
            missing_sections.append("interactions")
        if not pack.open_questions:
            missing_sections.append("open_questions")

        objectives_missing_signals = [
            objective.objective_id
            for objective in pack.objectives
            if not objective.success_signal.strip()
        ]
        wireframes_missing_blocks = [
            wireframe.surface_id
            for wireframe in pack.wireframes
            if not wireframe.primary_blocks
        ]
        interactions_missing_states = [
            interaction.flow_id
            for interaction in pack.interactions
            if not interaction.states
        ]
        unresolved_question_ids = [
            question.question_id
            for question in pack.open_questions
            if question.status.lower() != "resolved"
        ]
        wireframe_ids = {wireframe.surface_id for wireframe in pack.wireframes}

        checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
        for item in pack.reviewer_checklist:
            checklist_by_surface.setdefault(item.surface_id, []).append(item)
        wireframes_missing_checklists = []
        orphan_checklist_surfaces = []
        checklist_items_missing_evidence = []
        checklist_items_missing_role_links = []
        if pack.requires_reviewer_checklist:
            wireframes_missing_checklists = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in checklist_by_surface
            )
            orphan_checklist_surfaces = sorted(
                surface_id for surface_id in checklist_by_surface if surface_id not in wireframe_ids
            )
            checklist_items_missing_evidence = sorted(
                item.item_id for item in pack.reviewer_checklist if not item.evidence_links
            )

        decision_by_surface: Dict[str, List[ReviewDecision]] = {}
        for decision in pack.decision_log:
            decision_by_surface.setdefault(decision.surface_id, []).append(decision)
        wireframes_missing_decisions = []
        orphan_decision_surfaces = []
        unresolved_decision_ids = []
        unresolved_decisions_missing_follow_ups = []
        if pack.requires_decision_log:
            wireframes_missing_decisions = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in decision_by_surface
            )
            orphan_decision_surfaces = sorted(
                surface_id for surface_id in decision_by_surface if surface_id not in wireframe_ids
            )
            unresolved_decision_ids = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}
            )
            unresolved_decisions_missing_follow_ups = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}
                and not decision.follow_up.strip()
            )

        checklist_item_ids = {item.item_id for item in pack.reviewer_checklist}
        decision_ids = {decision.decision_id for decision in pack.decision_log}
        assignment_ids = {assignment.assignment_id for assignment in pack.role_matrix}
        role_assignments_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
        for assignment in pack.role_matrix:
            role_assignments_by_surface.setdefault(assignment.surface_id, []).append(assignment)
        wireframes_missing_role_assignments = []
        orphan_role_assignment_surfaces = []
        role_assignments_missing_responsibilities = []
        role_assignments_missing_checklist_links = []
        role_assignments_missing_decision_links = []
        decisions_missing_role_links = []
        if pack.requires_role_matrix:
            wireframes_missing_role_assignments = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in role_assignments_by_surface
            )
            orphan_role_assignment_surfaces = sorted(
                surface_id
                for surface_id in role_assignments_by_surface
                if surface_id not in wireframe_ids
            )
            role_assignments_missing_responsibilities = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.responsibilities
            )
            role_assignments_missing_checklist_links = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.checklist_item_ids
                or any(item_id not in checklist_item_ids for item_id in assignment.checklist_item_ids)
            )
            role_assignments_missing_decision_links = sorted(
                assignment.assignment_id
                for assignment in pack.role_matrix
                if not assignment.decision_ids
                or any(decision_id not in decision_ids for decision_id in assignment.decision_ids)
            )
            role_linked_checklist_ids = {
                item_id
                for assignment in pack.role_matrix
                for item_id in assignment.checklist_item_ids
            }
            role_linked_decision_ids = {
                decision_id
                for assignment in pack.role_matrix
                for decision_id in assignment.decision_ids
            }
            checklist_items_missing_role_links = sorted(
                item.item_id
                for item in pack.reviewer_checklist
                if item.item_id not in role_linked_checklist_ids
            )
            decisions_missing_role_links = sorted(
                decision.decision_id
                for decision in pack.decision_log
                if decision.decision_id not in role_linked_decision_ids
            )

        signoffs_by_surface: Dict[str, List[ReviewSignoff]] = {}
        for signoff in pack.signoff_log:
            signoffs_by_surface.setdefault(signoff.surface_id, []).append(signoff)
        wireframes_missing_signoffs = []
        orphan_signoff_surfaces = []
        signoffs_missing_assignments = []
        signoffs_missing_evidence = []
        signoffs_missing_requested_dates = []
        signoffs_missing_due_dates = []
        signoffs_missing_escalation_owners = []
        signoffs_missing_reminder_owners = []
        signoffs_missing_next_reminders = []
        signoffs_missing_reminder_cadence = []
        signoffs_with_breached_sla = []
        waived_signoffs_missing_metadata = []
        unresolved_required_signoff_ids = []
        if pack.requires_signoff_log:
            wireframes_missing_signoffs = sorted(
                wireframe.surface_id
                for wireframe in pack.wireframes
                if wireframe.surface_id not in signoffs_by_surface
            )
            orphan_signoff_surfaces = sorted(
                surface_id for surface_id in signoffs_by_surface if surface_id not in wireframe_ids
            )
            signoffs_missing_assignments = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.assignment_id not in assignment_ids
            )
            signoffs_missing_evidence = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.status.lower() != "waived" and not signoff.evidence_links
            )
            signoffs_missing_requested_dates = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.requested_at.strip()
            )
            signoffs_missing_due_dates = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.due_at.strip()
            )
            signoffs_missing_escalation_owners = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required and not signoff.escalation_owner.strip()
            )
            unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
            signoffs_missing_reminder_owners = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.reminder_owner.strip()
            )
            signoffs_missing_next_reminders = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.next_reminder_at.strip()
            )
            signoffs_missing_reminder_cadence = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
                and not signoff.reminder_cadence.strip()
            )
            signoffs_with_breached_sla = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.sla_status.lower() == "breached"
                and signoff.status.lower() not in {"approved", "accepted", "resolved"}
            )
            waived_signoffs_missing_metadata = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.status.lower() == "waived"
                and (not signoff.waiver_owner.strip() or not signoff.waiver_reason.strip())
            )
            unresolved_required_signoff_ids = sorted(
                signoff.signoff_id
                for signoff in pack.signoff_log
                if signoff.required
                and signoff.status.lower() not in unresolved_statuses
            )

        blocker_by_signoff: Dict[str, List[ReviewBlocker]] = {}
        blocker_surfaces = set()
        for blocker in pack.blocker_log:
            blocker_surfaces.add(blocker.surface_id)
            blocker_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
        blockers_missing_signoff_links = []
        blockers_missing_escalation_owners = []
        blockers_missing_next_actions = []
        freeze_exceptions_missing_owners = []
        freeze_exceptions_missing_until = []
        freeze_exceptions_missing_approvers = []
        freeze_exceptions_missing_approval_dates = []
        freeze_exceptions_missing_renewal_owners = []
        freeze_exceptions_missing_renewal_dates = []
        orphan_blocker_surfaces = []
        unresolved_required_signoffs_without_blockers = []
        if pack.requires_blocker_log:
            signoff_ids = {signoff.signoff_id for signoff in pack.signoff_log}
            blockers_missing_signoff_links = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if blocker.signoff_id not in signoff_ids
            )
            blockers_missing_escalation_owners = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if not blocker.escalation_owner.strip()
            )
            blockers_missing_next_actions = sorted(
                blocker.blocker_id for blocker in pack.blocker_log if not blocker.next_action.strip()
            )
            freeze_exceptions_missing_owners = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_owner.strip()
            )
            freeze_exceptions_missing_until = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_until.strip()
            )
            freeze_exceptions_missing_approvers = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_approved_by.strip()
            )
            freeze_exceptions_missing_approval_dates = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_approved_at.strip()
            )
            freeze_exceptions_missing_renewal_owners = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_renewal_owner.strip()
            )
            freeze_exceptions_missing_renewal_dates = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.freeze_exception and not blocker.freeze_renewal_by.strip()
            )
            orphan_blocker_surfaces = sorted(
                surface_id for surface_id in blocker_surfaces if surface_id not in wireframe_ids
            )
            unresolved_required_signoffs_without_blockers = sorted(
                signoff_id
                for signoff_id in unresolved_required_signoff_ids
                if signoff_id not in blocker_by_signoff
            )

        blocker_timeline_by_blocker: Dict[str, List[ReviewBlockerEvent]] = {}
        for event in pack.blocker_timeline:
            blocker_timeline_by_blocker.setdefault(event.blocker_id, []).append(event)
        blockers_missing_timeline_events = []
        closed_blockers_missing_resolution_events = []
        orphan_blocker_timeline_blocker_ids = []
        handoff_events_missing_targets = []
        handoff_events_missing_artifacts = []
        handoff_events_missing_ack_owners = []
        handoff_events_missing_ack_dates = []
        if pack.requires_blocker_timeline:
            blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
            orphan_blocker_timeline_blocker_ids = sorted(
                blocker_id
                for blocker_id in blocker_timeline_by_blocker
                if blocker_id not in blocker_ids
            )
            blockers_missing_timeline_events = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.status.lower() not in {"resolved", "closed"}
                and blocker.blocker_id not in blocker_timeline_by_blocker
            )
            closed_blockers_missing_resolution_events = sorted(
                blocker.blocker_id
                for blocker in pack.blocker_log
                if blocker.status.lower() in {"resolved", "closed"}
                and not any(
                    event.status.lower() in {"resolved", "closed"}
                    for event in blocker_timeline_by_blocker.get(blocker.blocker_id, [])
                )
            )
            handoff_statuses = {"escalated", "handoff", "reassigned"}
            handoff_events_missing_targets = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.handoff_to.strip()
            )
            handoff_events_missing_artifacts = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.artifact_ref.strip()
            )
            handoff_events_missing_ack_owners = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.ack_owner.strip()
            )
            handoff_events_missing_ack_dates = sorted(
                event.event_id
                for event in pack.blocker_timeline
                if event.status.lower() in handoff_statuses and not event.ack_at.strip()
            )

        ready = not (
            missing_sections
            or objectives_missing_signals
            or wireframes_missing_blocks
            or interactions_missing_states
            or wireframes_missing_checklists
            or orphan_checklist_surfaces
            or checklist_items_missing_evidence
            or checklist_items_missing_role_links
            or wireframes_missing_decisions
            or orphan_decision_surfaces
            or unresolved_decisions_missing_follow_ups
            or wireframes_missing_role_assignments
            or orphan_role_assignment_surfaces
            or role_assignments_missing_responsibilities
            or role_assignments_missing_checklist_links
            or role_assignments_missing_decision_links
            or decisions_missing_role_links
            or wireframes_missing_signoffs
            or orphan_signoff_surfaces
            or signoffs_missing_assignments
            or signoffs_missing_evidence
            or signoffs_missing_requested_dates
            or signoffs_missing_due_dates
            or signoffs_missing_escalation_owners
            or signoffs_missing_reminder_owners
            or signoffs_missing_next_reminders
            or signoffs_missing_reminder_cadence
            or waived_signoffs_missing_metadata
            or blockers_missing_signoff_links
            or blockers_missing_escalation_owners
            or blockers_missing_next_actions
            or freeze_exceptions_missing_owners
            or freeze_exceptions_missing_until
            or freeze_exceptions_missing_approvers
            or freeze_exceptions_missing_approval_dates
            or freeze_exceptions_missing_renewal_owners
            or freeze_exceptions_missing_renewal_dates
            or blockers_missing_timeline_events
            or closed_blockers_missing_resolution_events
            or orphan_blocker_surfaces
            or orphan_blocker_timeline_blocker_ids
            or handoff_events_missing_targets
            or handoff_events_missing_artifacts
            or handoff_events_missing_ack_owners
            or handoff_events_missing_ack_dates
            or unresolved_required_signoffs_without_blockers
        )
        return UIReviewPackAudit(
            ready=ready,
            objective_count=len(pack.objectives),
            wireframe_count=len(pack.wireframes),
            interaction_count=len(pack.interactions),
            open_question_count=len(pack.open_questions),
            checklist_count=len(pack.reviewer_checklist),
            decision_count=len(pack.decision_log),
            role_assignment_count=len(pack.role_matrix),
            signoff_count=len(pack.signoff_log),
            blocker_count=len(pack.blocker_log),
            blocker_timeline_count=len(pack.blocker_timeline),
            missing_sections=missing_sections,
            objectives_missing_signals=objectives_missing_signals,
            wireframes_missing_blocks=wireframes_missing_blocks,
            interactions_missing_states=interactions_missing_states,
            unresolved_question_ids=unresolved_question_ids,
            wireframes_missing_checklists=wireframes_missing_checklists,
            orphan_checklist_surfaces=orphan_checklist_surfaces,
            checklist_items_missing_evidence=checklist_items_missing_evidence,
            checklist_items_missing_role_links=checklist_items_missing_role_links,
            wireframes_missing_decisions=wireframes_missing_decisions,
            orphan_decision_surfaces=orphan_decision_surfaces,
            unresolved_decision_ids=unresolved_decision_ids,
            unresolved_decisions_missing_follow_ups=unresolved_decisions_missing_follow_ups,
            wireframes_missing_role_assignments=wireframes_missing_role_assignments,
            orphan_role_assignment_surfaces=orphan_role_assignment_surfaces,
            role_assignments_missing_responsibilities=role_assignments_missing_responsibilities,
            role_assignments_missing_checklist_links=role_assignments_missing_checklist_links,
            role_assignments_missing_decision_links=role_assignments_missing_decision_links,
            decisions_missing_role_links=decisions_missing_role_links,
            wireframes_missing_signoffs=wireframes_missing_signoffs,
            orphan_signoff_surfaces=orphan_signoff_surfaces,
            signoffs_missing_assignments=signoffs_missing_assignments,
            signoffs_missing_evidence=signoffs_missing_evidence,
            signoffs_missing_requested_dates=signoffs_missing_requested_dates,
            signoffs_missing_due_dates=signoffs_missing_due_dates,
            signoffs_missing_escalation_owners=signoffs_missing_escalation_owners,
            signoffs_missing_reminder_owners=signoffs_missing_reminder_owners,
            signoffs_missing_next_reminders=signoffs_missing_next_reminders,
            signoffs_missing_reminder_cadence=signoffs_missing_reminder_cadence,
            signoffs_with_breached_sla=signoffs_with_breached_sla,
            waived_signoffs_missing_metadata=waived_signoffs_missing_metadata,
            unresolved_required_signoff_ids=unresolved_required_signoff_ids,
            blockers_missing_signoff_links=blockers_missing_signoff_links,
            blockers_missing_escalation_owners=blockers_missing_escalation_owners,
            blockers_missing_next_actions=blockers_missing_next_actions,
            freeze_exceptions_missing_owners=freeze_exceptions_missing_owners,
            freeze_exceptions_missing_until=freeze_exceptions_missing_until,
            freeze_exceptions_missing_approvers=freeze_exceptions_missing_approvers,
            freeze_exceptions_missing_approval_dates=freeze_exceptions_missing_approval_dates,
            freeze_exceptions_missing_renewal_owners=freeze_exceptions_missing_renewal_owners,
            freeze_exceptions_missing_renewal_dates=freeze_exceptions_missing_renewal_dates,
            blockers_missing_timeline_events=blockers_missing_timeline_events,
            closed_blockers_missing_resolution_events=closed_blockers_missing_resolution_events,
            orphan_blocker_surfaces=orphan_blocker_surfaces,
            orphan_blocker_timeline_blocker_ids=orphan_blocker_timeline_blocker_ids,
            handoff_events_missing_targets=handoff_events_missing_targets,
            handoff_events_missing_artifacts=handoff_events_missing_artifacts,
            handoff_events_missing_ack_owners=handoff_events_missing_ack_owners,
            handoff_events_missing_ack_dates=handoff_events_missing_ack_dates,
            unresolved_required_signoffs_without_blockers=unresolved_required_signoffs_without_blockers,
        )


def _build_blocker_timeline_index(pack: UIReviewPack) -> Dict[str, List[ReviewBlockerEvent]]:
    timeline_index: Dict[str, List[ReviewBlockerEvent]] = {}
    for event in sorted(pack.blocker_timeline, key=lambda item: (item.timestamp, item.event_id)):
        timeline_index.setdefault(event.blocker_id, []).append(event)
    return timeline_index


def _build_review_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        if signoff.status.lower() not in {"waived", "deferred"}:
            continue
        entries.append(
            {
                "exception_id": f"exc-{signoff.signoff_id}",
                "category": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "owner": signoff.waiver_owner or signoff.role,
                "status": signoff.status,
                "severity": "none",
                "summary": signoff.waiver_reason or signoff.notes or "none",
                "evidence": ",".join(signoff.evidence_links) or "none",
                "latest_event": "none",
                "next_action": signoff.notes or signoff.waiver_reason or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "exception_id": f"exc-{blocker.blocker_id}",
                "category": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "owner": blocker.owner,
                "status": blocker.status,
                "severity": blocker.severity,
                "summary": blocker.summary,
                "evidence": blocker.escalation_owner or "none",
                "latest_event": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["surface_id"], item["category"], item["source_id"]),
    )


def _build_freeze_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        if signoff.status.lower() not in {"waived", "deferred"}:
            continue
        entries.append(
            {
                "entry_id": f"freeze-{signoff.signoff_id}",
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "owner": signoff.waiver_owner or signoff.role,
                "status": signoff.status,
                "window": "none",
                "summary": signoff.waiver_reason or signoff.notes or "none",
                "evidence": ",".join(signoff.evidence_links) or "none",
                "next_action": signoff.notes or signoff.waiver_reason or "none",
            }
        )
    for blocker in pack.blocker_log:
        if not blocker.freeze_exception:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "entry_id": f"freeze-{blocker.blocker_id}",
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "owner": blocker.freeze_owner or blocker.owner,
                "status": blocker.status,
                "window": blocker.freeze_until or "none",
                "summary": blocker.freeze_reason or blocker.summary,
                "evidence": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["surface_id"], item["item_type"], item["source_id"]),
    )


def _build_signoff_breach_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index: Dict[str, List[str]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        blocker_index.setdefault(blocker.signoff_id, []).append(blocker.blocker_id)
    entries = [
        {
            "entry_id": f"breach-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "requested_at": signoff.requested_at or "none",
            "due_at": signoff.due_at or "none",
            "escalation_owner": signoff.escalation_owner or "none",
            "linked_blockers": ",".join(sorted(blocker_index.get(signoff.signoff_id, []))) or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.sla_status.lower() in {"at-risk", "breached"}
        and signoff.status.lower() not in {"approved", "accepted", "resolved", "waived", "deferred"}
    ]
    return sorted(
        entries,
        key=lambda item: (item["due_at"], item["sla_status"], item["escalation_owner"], item["signoff_id"]),
    )


def _build_escalation_handoff_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}
    handoff_statuses = {"escalated", "handoff", "reassigned"}
    entries: List[Dict[str, str]] = []
    for event in pack.blocker_timeline:
        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():
            continue
        blocker = blocker_index.get(event.blocker_id)
        entries.append(
            {
                "ledger_id": f"handoff-{event.event_id}",
                "event_id": event.event_id,
                "blocker_id": event.blocker_id,
                "surface_id": blocker.surface_id if blocker else "none",
                "actor": event.actor,
                "status": event.status,
                "handoff_from": event.handoff_from or (blocker.owner if blocker else "none"),
                "handoff_to": event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none",
                "channel": event.channel or "none",
                "artifact_ref": event.artifact_ref or "none",
                "timestamp": event.timestamp,
                "summary": event.summary,
                "next_action": event.next_action or "none",
            }
        )
    return sorted(entries, key=lambda item: (item["timestamp"], item["event_id"]))


def _build_handoff_ack_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}
    handoff_statuses = {"escalated", "handoff", "reassigned"}
    entries: List[Dict[str, str]] = []
    for event in pack.blocker_timeline:
        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():
            continue
        blocker = blocker_index.get(event.blocker_id)
        fallback_owner = event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none"
        entries.append(
            {
                "entry_id": f"ack-{event.event_id}",
                "event_id": event.event_id,
                "blocker_id": event.blocker_id,
                "surface_id": blocker.surface_id if blocker else "none",
                "actor": event.actor,
                "status": event.status,
                "handoff_to": event.handoff_to or fallback_owner,
                "ack_owner": event.ack_owner or fallback_owner,
                "ack_status": event.ack_status or "pending",
                "ack_at": event.ack_at or "none",
                "channel": event.channel or "none",
                "artifact_ref": event.artifact_ref or "none",
                "summary": event.summary,
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["ack_status"], item["ack_owner"], item["event_id"]),
    )


def _build_signoff_reminder_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    entries = [
        {
            "entry_id": f"rem-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "reminder_owner": signoff.reminder_owner or "none",
            "reminder_channel": signoff.reminder_channel or "none",
            "last_reminder_at": signoff.last_reminder_at or "none",
            "next_reminder_at": signoff.next_reminder_at or "none",
            "due_at": signoff.due_at or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.required and signoff.status.lower() not in unresolved_statuses
    ]
    return sorted(
        entries,
        key=lambda item: (item["next_reminder_at"], item["reminder_owner"], item["signoff_id"]),
    )


def _build_reminder_cadence_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    entries = [
        {
            "entry_id": f"cad-rem-{signoff.signoff_id}",
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "reminder_owner": signoff.reminder_owner or "none",
            "reminder_cadence": signoff.reminder_cadence or "none",
            "reminder_status": signoff.reminder_status or "scheduled",
            "last_reminder_at": signoff.last_reminder_at or "none",
            "next_reminder_at": signoff.next_reminder_at or "none",
            "due_at": signoff.due_at or "none",
            "summary": signoff.notes or signoff.waiver_reason or signoff.role,
        }
        for signoff in pack.signoff_log
        if signoff.required and signoff.status.lower() not in unresolved_statuses
    ]
    return sorted(
        entries,
        key=lambda item: (item["reminder_cadence"], item["reminder_status"], item["signoff_id"]),
    )


def _build_freeze_approval_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    timeline_index = _build_blocker_timeline_index(pack)
    entries: List[Dict[str, str]] = []
    for blocker in pack.blocker_log:
        if not blocker.freeze_exception:
            continue
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        latest_label = (
            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"
            if latest
            else "none"
        )
        entries.append(
            {
                "entry_id": f"freeze-approval-{blocker.blocker_id}",
                "blocker_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "freeze_owner": blocker.freeze_owner or blocker.owner,
                "freeze_until": blocker.freeze_until or "none",
                "freeze_approved_by": blocker.freeze_approved_by or "none",
                "freeze_approved_at": blocker.freeze_approved_at or "none",
                "summary": blocker.freeze_reason or blocker.summary,
                "latest_event": latest_label,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["freeze_approved_at"], item["freeze_until"], item["blocker_id"]),
    )


def _build_freeze_renewal_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries = [
        {
            "entry_id": f"renew-{blocker.blocker_id}",
            "blocker_id": blocker.blocker_id,
            "surface_id": blocker.surface_id,
            "status": blocker.status,
            "freeze_owner": blocker.freeze_owner or blocker.owner,
            "freeze_until": blocker.freeze_until or "none",
            "renewal_owner": blocker.freeze_renewal_owner or "none",
            "renewal_by": blocker.freeze_renewal_by or "none",
            "renewal_status": blocker.freeze_renewal_status or "not-needed",
            "freeze_approved_by": blocker.freeze_approved_by or "none",
            "summary": blocker.freeze_reason or blocker.summary,
            "next_action": blocker.next_action or "none",
        }
        for blocker in pack.blocker_log
        if blocker.freeze_exception
    ]
    return sorted(
        entries,
        key=lambda item: (item["renewal_by"], item["renewal_owner"], item["blocker_id"]),
    )


def _build_owner_escalation_digest_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for entry in _build_escalation_dashboard_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['escalation_id']}",
                "owner": entry["escalation_owner"],
                "item_type": entry["item_type"],
                "source_id": entry["source_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["priority"],
            }
        )
    for entry in _build_signoff_reminder_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['entry_id']}",
                "owner": entry["reminder_owner"],
                "item_type": "reminder",
                "source_id": entry["signoff_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["next_reminder_at"],
            }
        )
    for entry in _build_freeze_approval_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['entry_id']}",
                "owner": entry["freeze_approved_by"],
                "item_type": "freeze",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["freeze_until"],
            }
        )
    for entry in _build_escalation_handoff_entries(pack):
        entries.append(
            {
                "digest_id": f"digest-{entry['ledger_id']}",
                "owner": entry["handoff_to"],
                "item_type": "handoff",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "summary": entry["summary"],
                "detail": entry["timestamp"],
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),
    )


def _build_owner_review_queue_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    signoff_ready_statuses = {"approved", "accepted", "resolved"}
    blocker_done_statuses = {"resolved", "closed"}

    for item in pack.reviewer_checklist:
        if item.status.lower() in checklist_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{item.item_id}",
                "owner": item.owner,
                "item_type": "checklist",
                "source_id": item.item_id,
                "surface_id": item.surface_id,
                "status": item.status,
                "summary": item.prompt,
                "next_action": item.notes or ",".join(item.evidence_links) or "none",
            }
        )
    for decision in pack.decision_log:
        if decision.status.lower() in decision_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{decision.decision_id}",
                "owner": decision.owner,
                "item_type": "decision",
                "source_id": decision.decision_id,
                "surface_id": decision.surface_id,
                "status": decision.status,
                "summary": decision.summary,
                "next_action": decision.follow_up or decision.rationale,
            }
        )
    for signoff in pack.signoff_log:
        if signoff.status.lower() in signoff_ready_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{signoff.signoff_id}",
                "owner": signoff.waiver_owner or signoff.role,
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "status": signoff.status,
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
                "next_action": signoff.waiver_reason or signoff.notes or signoff.due_at or ",".join(signoff.evidence_links) or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in blocker_done_statuses:
            continue
        entries.append(
            {
                "queue_id": f"queue-{blocker.blocker_id}",
                "owner": blocker.owner,
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "summary": blocker.summary,
                "next_action": blocker.next_action or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),
    )


def _build_checklist_traceability_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignments_by_item: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        for item_id in assignment.checklist_item_ids:
            assignments_by_item.setdefault(item_id, []).append(assignment)
    entries: List[Dict[str, str]] = []
    for item in pack.reviewer_checklist:
        linked_assignments = assignments_by_item.get(item.item_id, [])
        linked_decisions = sorted(
            {decision_id for assignment in linked_assignments for decision_id in assignment.decision_ids}
        )
        entries.append(
            {
                "entry_id": f"trace-{item.item_id}",
                "item_id": item.item_id,
                "surface_id": item.surface_id,
                "owner": item.owner,
                "status": item.status,
                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",
                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",
                "linked_decisions": ",".join(linked_decisions) or "none",
                "evidence": ",".join(item.evidence_links) or "none",
                "summary": item.notes or item.prompt,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["item_id"]))


def _build_decision_followup_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignments_by_decision: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        for decision_id in assignment.decision_ids:
            assignments_by_decision.setdefault(decision_id, []).append(assignment)
    entries: List[Dict[str, str]] = []
    for decision in pack.decision_log:
        linked_assignments = assignments_by_decision.get(decision.decision_id, [])
        linked_checklist_ids = sorted(
            {item_id for assignment in linked_assignments for item_id in assignment.checklist_item_ids}
        )
        entries.append(
            {
                "entry_id": f"follow-{decision.decision_id}",
                "decision_id": decision.decision_id,
                "surface_id": decision.surface_id,
                "owner": decision.owner,
                "status": decision.status,
                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",
                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",
                "linked_checklists": ",".join(linked_checklist_ids) or "none",
                "follow_up": decision.follow_up or "none",
                "summary": decision.summary,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["decision_id"]))


def _build_objective_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignments_by_role.setdefault(assignment.role, []).append(assignment)
    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    checklist_index = {item.item_id: item for item in pack.reviewer_checklist}
    decision_index = {decision.decision_id: decision for decision in pack.decision_log}
    status_priority = {"blocked": 0, "at-risk": 1, "covered": 2}
    entries: List[Dict[str, str]] = []
    for objective in pack.objectives:
        assignments = assignments_by_role.get(objective.persona, [])
        checklist_ids = sorted(
            {item_id for assignment in assignments for item_id in assignment.checklist_item_ids}
        )
        decision_ids = sorted(
            {decision_id for assignment in assignments for decision_id in assignment.decision_ids}
        )
        signoffs = [
            signoff_by_assignment[assignment.assignment_id]
            for assignment in assignments
            if assignment.assignment_id in signoff_by_assignment
        ]
        blocker_ids = sorted(
            {
                blocker.blocker_id
                for signoff in signoffs
                for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
            }
        )
        open_checklist = sum(
            1
            for item_id in checklist_ids
            if checklist_index[item_id].status.lower() not in checklist_ready_statuses
        )
        open_decisions = sum(
            1
            for decision_id in decision_ids
            if decision_index[decision_id].status.lower() not in decision_ready_statuses
        )
        open_assignments = sum(
            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses
        )
        open_signoffs = sum(
            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses
        )
        coverage_status = (
            "blocked"
            if blocker_ids
            else "at-risk"
            if open_checklist or open_decisions or open_assignments or open_signoffs
            else "covered"
        )
        entries.append(
            {
                "entry_id": f"objcov-{objective.objective_id}",
                "objective_id": objective.objective_id,
                "persona": objective.persona,
                "priority": objective.priority,
                "coverage_status": coverage_status,
                "dependency_count": str(len(objective.dependencies)),
                "dependency_ids": ",".join(objective.dependencies) or "none",
                "surface_ids": ",".join(sorted({assignment.surface_id for assignment in assignments})) or "none",
                "assignment_ids": ",".join(assignment.assignment_id for assignment in assignments) or "none",
                "checklist_ids": ",".join(checklist_ids) or "none",
                "decision_ids": ",".join(decision_ids) or "none",
                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",
                "blocker_ids": ",".join(blocker_ids) or "none",
                "summary": objective.success_signal or objective.outcome,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["coverage_status"]], item["persona"], item["objective_id"]),
    )


def _build_wireframe_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    blocker_done_statuses = {"resolved", "closed"}
    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        checklist_by_surface.setdefault(item.surface_id, []).append(item)
    decision_by_surface: Dict[str, List[ReviewDecision]] = {}
    for decision in pack.decision_log:
        decision_by_surface.setdefault(decision.surface_id, []).append(decision)
    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)
    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}
    for signoff in pack.signoff_log:
        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)
    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)
    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}
    entries: List[Dict[str, str]] = []
    for wireframe in pack.wireframes:
        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])
        decisions = decision_by_surface.get(wireframe.surface_id, [])
        assignments = assignment_by_surface.get(wireframe.surface_id, [])
        signoffs = signoff_by_surface.get(wireframe.surface_id, [])
        blockers = [
            blocker
            for blocker in blocker_by_surface.get(wireframe.surface_id, [])
            if blocker.status.lower() not in blocker_done_statuses
        ]
        checklist_open = sum(
            1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses
        )
        decisions_open = sum(
            1 for decision in decisions if decision.status.lower() not in decision_ready_statuses
        )
        assignments_open = sum(
            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses
        )
        signoffs_open = sum(
            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses
        )
        blockers_open = len(blockers)
        open_total = (
            checklist_open + decisions_open + assignments_open + signoffs_open + blockers_open
        )
        readiness_status = (
            "blocked"
            if blockers_open
            else "at-risk"
            if checklist_open or decisions_open or assignments_open or signoffs_open
            else "ready"
        )
        entries.append(
            {
                "entry_id": f"wire-{wireframe.surface_id}",
                "surface_id": wireframe.surface_id,
                "device": wireframe.device,
                "entry_point": wireframe.entry_point,
                "readiness_status": readiness_status,
                "open_total": str(open_total),
                "checklist_open": str(checklist_open),
                "decisions_open": str(decisions_open),
                "assignments_open": str(assignments_open),
                "signoffs_open": str(signoffs_open),
                "blockers_open": str(blockers_open),
                "block_count": str(len(wireframe.primary_blocks)),
                "note_count": str(len(wireframe.review_notes)),
                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",
                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",
                "summary": wireframe.name,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["readiness_status"]], item["surface_id"]),
    )


def _build_open_question_tracker_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for question in pack.open_questions:
        linked_items = [
            item for item in pack.reviewer_checklist if question.question_id in item.evidence_links
        ]
        flow_ids = sorted(
            {
                evidence_link
                for item in linked_items
                for evidence_link in item.evidence_links
                if evidence_link.startswith("flow-")
            }
        )
        entries.append(
            {
                "entry_id": f"qtrack-{question.question_id}",
                "question_id": question.question_id,
                "owner": question.owner,
                "theme": question.theme,
                "status": question.status,
                "link_status": "linked" if linked_items else "orphan",
                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",
                "checklist_ids": ",".join(item.item_id for item in linked_items) or "none",
                "flow_ids": ",".join(flow_ids) or "none",
                "summary": question.question,
                "impact": question.impact,
            }
        )
    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["question_id"]))


def _build_interaction_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    checklist_by_flow: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        for evidence_link in item.evidence_links:
            if evidence_link.startswith("flow-"):
                checklist_by_flow.setdefault(evidence_link, []).append(item)
    status_priority = {"missing": 0, "watch": 1, "covered": 2}
    entries: List[Dict[str, str]] = []
    for interaction in pack.interactions:
        linked_items = checklist_by_flow.get(interaction.flow_id, [])
        checklist_ids = list(dict.fromkeys(item.item_id for item in linked_items))
        open_checklist_ids = list(
            dict.fromkeys(
                item.item_id
                for item in linked_items
                if item.status.lower() not in checklist_ready_statuses
            )
        )
        coverage_status = (
            "missing"
            if not checklist_ids
            else "watch"
            if open_checklist_ids
            else "covered"
        )
        entries.append(
            {
                "entry_id": f"intcov-{interaction.flow_id}",
                "flow_id": interaction.flow_id,
                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",
                "owners": ",".join(sorted({item.owner for item in linked_items})) or "none",
                "checklist_ids": ",".join(checklist_ids) or "none",
                "open_checklist_ids": ",".join(open_checklist_ids) or "none",
                "coverage_status": coverage_status,
                "state_count": str(len(interaction.states)),
                "exception_count": str(len(interaction.exceptions)),
                "summary": interaction.trigger,
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["coverage_status"]], item["flow_id"]),
    )


def _build_persona_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    objective_entries = _build_objective_coverage_entries(pack)
    objective_entries_by_persona: Dict[str, List[Dict[str, str]]] = {}
    for entry in objective_entries:
        objective_entries_by_persona.setdefault(entry["persona"], []).append(entry)
    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignments_by_role.setdefault(assignment.role, []).append(assignment)
    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    questions_by_owner: Dict[str, List[OpenQuestion]] = {}
    for question in pack.open_questions:
        questions_by_owner.setdefault(question.owner, []).append(question)
    queue_entries = _build_owner_review_queue_entries(pack)
    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}
    entries: List[Dict[str, str]] = []
    for persona, persona_objectives in objective_entries_by_persona.items():
        surface_ids = sorted(
            {
                surface_id
                for entry in persona_objectives
                for surface_id in entry["surface_ids"].split(",")
                if surface_id and surface_id != "none"
            }
        )
        assignments = assignments_by_role.get(persona, [])
        signoffs = [
            signoff_by_assignment[assignment.assignment_id]
            for assignment in assignments
            if assignment.assignment_id in signoff_by_assignment
        ]
        blockers = [
            blocker
            for signoff in signoffs
            for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
        ]
        blocker_ids = sorted({blocker.blocker_id for blocker in blockers})
        questions = questions_by_owner.get(persona, [])
        queue_items = [
            entry
            for entry in queue_entries
            if entry["owner"] == persona
            and (not surface_ids or entry["surface_id"] in surface_ids)
        ]
        objective_statuses = {entry["coverage_status"] for entry in persona_objectives}
        readiness = (
            "blocked"
            if "blocked" in objective_statuses or blocker_ids
            else "at-risk"
            if "at-risk" in objective_statuses or questions or queue_items
            else "ready"
        )
        entries.append(
            {
                "entry_id": f"persona-{persona.lower().replace(' ', '-')}",
                "persona": persona,
                "readiness": readiness,
                "objective_count": str(len(persona_objectives)),
                "assignment_count": str(len(assignments)),
                "signoff_count": str(len(signoffs)),
                "question_count": str(len(questions)),
                "queue_count": str(len(queue_items)),
                "blocker_count": str(len(blocker_ids)),
                "objective_ids": ",".join(
                    sorted(entry["objective_id"] for entry in persona_objectives)
                )
                or "none",
                "surface_ids": ",".join(surface_ids) or "none",
                "queue_ids": ",".join(sorted(entry["queue_id"] for entry in queue_items)) or "none",
                "blocker_ids": ",".join(blocker_ids) or "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (status_priority[item["readiness"]], item["persona"]),
    )


def _build_review_summary_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    objective_entries = _build_objective_coverage_entries(pack)
    objective_status_counts: Dict[str, int] = {}
    for entry in objective_entries:
        objective_status_counts[entry["coverage_status"]] = (
            objective_status_counts.get(entry["coverage_status"], 0) + 1
        )

    persona_entries = _build_persona_readiness_entries(pack)
    persona_status_counts: Dict[str, int] = {}
    for entry in persona_entries:
        persona_status_counts[entry["readiness"]] = persona_status_counts.get(entry["readiness"], 0) + 1

    wireframe_entries = _build_wireframe_readiness_entries(pack)
    wireframe_status_counts: Dict[str, int] = {}
    for entry in wireframe_entries:
        wireframe_status_counts[entry["readiness_status"]] = (
            wireframe_status_counts.get(entry["readiness_status"], 0) + 1
        )

    interaction_entries = _build_interaction_coverage_entries(pack)
    interaction_status_counts: Dict[str, int] = {}
    for entry in interaction_entries:
        interaction_status_counts[entry["coverage_status"]] = (
            interaction_status_counts.get(entry["coverage_status"], 0) + 1
        )

    question_entries = _build_open_question_tracker_entries(pack)
    question_link_counts: Dict[str, int] = {}
    question_owners = {entry["owner"] for entry in question_entries}
    for entry in question_entries:
        question_link_counts[entry["link_status"]] = question_link_counts.get(entry["link_status"], 0) + 1

    action_entries = _build_owner_workload_entries(pack)
    action_lane_counts: Dict[str, int] = {}
    for entry in action_entries:
        action_lane_counts[entry["lane"]] = action_lane_counts.get(entry["lane"], 0) + 1

    return [
        {
            "entry_id": "summary-objectives",
            "category": "objectives",
            "total": str(len(objective_entries)),
            "metrics": (
                f"blocked={objective_status_counts.get('blocked', 0)} "
                f"at-risk={objective_status_counts.get('at-risk', 0)} "
                f"covered={objective_status_counts.get('covered', 0)}"
            ),
        },
        {
            "entry_id": "summary-personas",
            "category": "personas",
            "total": str(len(persona_entries)),
            "metrics": (
                f"blocked={persona_status_counts.get('blocked', 0)} "
                f"at-risk={persona_status_counts.get('at-risk', 0)} "
                f"ready={persona_status_counts.get('ready', 0)}"
            ),
        },
        {
            "entry_id": "summary-wireframes",
            "category": "wireframes",
            "total": str(len(wireframe_entries)),
            "metrics": (
                f"blocked={wireframe_status_counts.get('blocked', 0)} "
                f"at-risk={wireframe_status_counts.get('at-risk', 0)} "
                f"ready={wireframe_status_counts.get('ready', 0)}"
            ),
        },
        {
            "entry_id": "summary-interactions",
            "category": "interactions",
            "total": str(len(interaction_entries)),
            "metrics": (
                f"covered={interaction_status_counts.get('covered', 0)} "
                f"watch={interaction_status_counts.get('watch', 0)} "
                f"missing={interaction_status_counts.get('missing', 0)}"
            ),
        },
        {
            "entry_id": "summary-questions",
            "category": "questions",
            "total": str(len(question_entries)),
            "metrics": (
                f"linked={question_link_counts.get('linked', 0)} "
                f"orphan={question_link_counts.get('orphan', 0)} "
                f"owners={len(question_owners)}"
            ),
        },
        {
            "entry_id": "summary-actions",
            "category": "actions",
            "total": str(len(action_entries)),
            "metrics": (
                f"queue={action_lane_counts.get('queue', 0)} "
                f"reminder={action_lane_counts.get('reminder', 0)} "
                f"renewal={action_lane_counts.get('renewal', 0)}"
            ),
        },
    ]


def _build_role_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    signoffs_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}
    entries = [
        {
            "entry_id": f"cover-{assignment.assignment_id}",
            "assignment_id": assignment.assignment_id,
            "surface_id": assignment.surface_id,
            "role": assignment.role,
            "status": assignment.status,
            "responsibility_count": str(len(assignment.responsibilities)),
            "checklist_count": str(len(assignment.checklist_item_ids)),
            "decision_count": str(len(assignment.decision_ids)),
            "signoff_id": signoffs_by_assignment.get(assignment.assignment_id).signoff_id if assignment.assignment_id in signoffs_by_assignment else "none",
            "signoff_status": signoffs_by_assignment.get(assignment.assignment_id).status if assignment.assignment_id in signoffs_by_assignment else "none",
            "summary": ",".join(assignment.responsibilities) or "none",
        }
        for assignment in pack.role_matrix
    ]
    return sorted(entries, key=lambda item: (item["surface_id"], item["status"], item["assignment_id"]))


def _build_owner_workload_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    for entry in _build_owner_review_queue_entries(pack):
        entries.append(
            {
                "entry_id": f"load-{entry['queue_id']}",
                "owner": entry["owner"],
                "item_type": entry["item_type"],
                "source_id": entry["source_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "lane": "queue",
                "detail": entry["next_action"],
                "summary": entry["summary"],
            }
        )
    for entry in _build_signoff_reminder_entries(pack):
        entries.append(
            {
                "entry_id": f"load-{entry['entry_id']}",
                "owner": entry["reminder_owner"],
                "item_type": "reminder",
                "source_id": entry["signoff_id"],
                "surface_id": entry["surface_id"],
                "status": entry["status"],
                "lane": "reminder",
                "detail": entry["next_reminder_at"],
                "summary": entry["summary"],
            }
        )
    for entry in _build_freeze_renewal_entries(pack):
        if entry["renewal_status"] == "not-needed":
            continue
        entries.append(
            {
                "entry_id": f"load-{entry['entry_id']}",
                "owner": entry["renewal_owner"],
                "item_type": "renewal",
                "source_id": entry["blocker_id"],
                "surface_id": entry["surface_id"],
                "status": entry["renewal_status"],
                "lane": "renewal",
                "detail": entry["renewal_by"],
                "summary": entry["summary"],
            }
        )
    return sorted(entries, key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]))


def _build_signoff_dependency_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    assignment_by_id = {assignment.assignment_id: assignment for assignment in pack.role_matrix}
    timeline_index = _build_blocker_timeline_index(pack)
    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}
    for blocker in pack.blocker_log:
        if blocker.status.lower() in {"resolved", "closed"}:
            continue
        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)
    entries: List[Dict[str, str]] = []
    for signoff in pack.signoff_log:
        assignment = assignment_by_id.get(signoff.assignment_id)
        blockers = unresolved_blockers_by_signoff.get(signoff.signoff_id, [])
        latest_event = None
        for blocker in blockers:
            events = timeline_index.get(blocker.blocker_id, [])
            if events:
                candidate = events[-1]
                if latest_event is None or (candidate.timestamp, candidate.event_id) > (latest_event.timestamp, latest_event.event_id):
                    latest_event = candidate
        latest_label = (
            f"{latest_event.event_id}/{latest_event.status}/{latest_event.actor}@{latest_event.timestamp}"
            if latest_event
            else "none"
        )
        entries.append(
            {
                "entry_id": f"dep-{signoff.signoff_id}",
                "signoff_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "role": signoff.role,
                "status": signoff.status,
                "assignment_id": signoff.assignment_id,
                "dependency_status": "blocked" if blockers else "clear",
                "checklist_ids": ",".join(assignment.checklist_item_ids) if assignment else "none",
                "decision_ids": ",".join(assignment.decision_ids) if assignment else "none",
                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",
                "blocker_owners": ",".join(sorted({blocker.owner for blocker in blockers})) or "none",
                "latest_blocker_event": latest_label,
                "sla_status": signoff.sla_status,
                "due_at": signoff.due_at or "none",
                "reminder_cadence": signoff.reminder_cadence or "none",
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
            }
        )
    return sorted(entries, key=lambda item: (item["dependency_status"], item["due_at"], item["signoff_id"]))


def _build_audit_density_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}
    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}
    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}
    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}
    blocker_done_statuses = {"resolved", "closed"}
    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}
    for item in pack.reviewer_checklist:
        checklist_by_surface.setdefault(item.surface_id, []).append(item)
    decision_by_surface: Dict[str, List[ReviewDecision]] = {}
    for decision in pack.decision_log:
        decision_by_surface.setdefault(decision.surface_id, []).append(decision)
    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}
    for assignment in pack.role_matrix:
        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)
    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}
    for signoff in pack.signoff_log:
        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)
    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}
    blocker_surface_by_id: Dict[str, str] = {}
    for blocker in pack.blocker_log:
        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)
        blocker_surface_by_id[blocker.blocker_id] = blocker.surface_id
    timeline_by_surface: Dict[str, List[ReviewBlockerEvent]] = {}
    for event in pack.blocker_timeline:
        surface_id = blocker_surface_by_id.get(event.blocker_id, "none")
        timeline_by_surface.setdefault(surface_id, []).append(event)
    entries: List[Dict[str, str]] = []
    for wireframe in pack.wireframes:
        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])
        decisions = decision_by_surface.get(wireframe.surface_id, [])
        assignments = assignment_by_surface.get(wireframe.surface_id, [])
        signoffs = signoff_by_surface.get(wireframe.surface_id, [])
        blockers = blocker_by_surface.get(wireframe.surface_id, [])
        timeline_events = timeline_by_surface.get(wireframe.surface_id, [])
        open_total = (
            sum(1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses)
            + sum(1 for decision in decisions if decision.status.lower() not in decision_ready_statuses)
            + sum(1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses)
            + sum(1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses)
            + sum(1 for blocker in blockers if blocker.status.lower() not in blocker_done_statuses)
        )
        artifact_total = len(checklist_items) + len(decisions) + len(assignments) + len(signoffs) + len(blockers) + len(timeline_events)
        load_band = "dense" if open_total >= 4 else "active" if open_total >= 2 else "light"
        entries.append(
            {
                "entry_id": f"density-{wireframe.surface_id}",
                "surface_id": wireframe.surface_id,
                "artifact_total": str(artifact_total),
                "open_total": str(open_total),
                "load_band": load_band,
                "block_count": str(len(wireframe.primary_blocks)),
                "note_count": str(len(wireframe.review_notes)),
                "checklist_count": str(len(checklist_items)),
                "decision_count": str(len(decisions)),
                "assignment_count": str(len(assignments)),
                "signoff_count": str(len(signoffs)),
                "blocker_count": str(len(blockers)),
                "timeline_count": str(len(timeline_events)),
            }
        )
    return sorted(entries, key=lambda item: (-int(item["open_total"]), item["surface_id"]))


def _build_signoff_sla_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries = [
        {
            "signoff_id": signoff.signoff_id,
            "surface_id": signoff.surface_id,
            "role": signoff.role,
            "status": signoff.status,
            "sla_status": signoff.sla_status,
            "requested_at": signoff.requested_at or "none",
            "due_at": signoff.due_at or "none",
            "escalation_owner": signoff.escalation_owner or "none",
            "required": "yes" if signoff.required else "no",
            "evidence": ",".join(signoff.evidence_links) or "none",
        }
        for signoff in pack.signoff_log
    ]
    return sorted(entries, key=lambda item: (item["due_at"], item["sla_status"], item["signoff_id"]))


def _build_escalation_dashboard_entries(pack: UIReviewPack) -> List[Dict[str, str]]:
    entries: List[Dict[str, str]] = []
    signoff_done_statuses = {"approved", "accepted", "resolved"}
    blocker_done_statuses = {"resolved", "closed"}
    for signoff in pack.signoff_log:
        if signoff.status.lower() in signoff_done_statuses:
            continue
        entries.append(
            {
                "escalation_id": f"esc-{signoff.signoff_id}",
                "escalation_owner": signoff.escalation_owner or "none",
                "item_type": "signoff",
                "source_id": signoff.signoff_id,
                "surface_id": signoff.surface_id,
                "status": signoff.status,
                "priority": signoff.sla_status,
                "current_owner": signoff.role,
                "summary": signoff.notes or signoff.waiver_reason or signoff.role,
                "due_at": signoff.due_at or "none",
            }
        )
    for blocker in pack.blocker_log:
        if blocker.status.lower() in blocker_done_statuses:
            continue
        entries.append(
            {
                "escalation_id": f"esc-{blocker.blocker_id}",
                "escalation_owner": blocker.escalation_owner or "none",
                "item_type": "blocker",
                "source_id": blocker.blocker_id,
                "surface_id": blocker.surface_id,
                "status": blocker.status,
                "priority": blocker.severity,
                "current_owner": blocker.owner,
                "summary": blocker.summary,
                "due_at": "none",
            }
        )
    return sorted(
        entries,
        key=lambda item: (item["escalation_owner"], item["item_type"], item["surface_id"], item["source_id"]),
    )


def render_ui_review_pack_report(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:
    lines = [
        "# UI Review Pack",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Audit: {audit.summary}",
        "",
        "## Objectives",
    ]
    for objective in pack.objectives:
        lines.append(
            "- "
            f"{objective.objective_id}: {objective.title} persona={objective.persona} priority={objective.priority}"
        )
        lines.append(
            "  "
            f"outcome={objective.outcome} success_signal={objective.success_signal} dependencies={','.join(objective.dependencies) or 'none'}"
        )

    review_summary_entries = _build_review_summary_entries(pack)
    lines.append("")
    lines.append("## Review Summary Board")
    lines.append(f"- Categories: {len(review_summary_entries)}")
    lines.append("")
    lines.append("### Entries")
    for entry in review_summary_entries:
        lines.append(
            f"- {entry['entry_id']}: category={entry['category']} total={entry['total']} {entry['metrics']}"
        )
    if not review_summary_entries:
        lines.append("- none")

    objective_coverage_entries = _build_objective_coverage_entries(pack)
    objective_persona_counts: Dict[str, int] = {}
    objective_status_counts: Dict[str, int] = {}
    for entry in objective_coverage_entries:
        objective_persona_counts[entry['persona']] = objective_persona_counts.get(entry['persona'], 0) + 1
        objective_status_counts[entry['coverage_status']] = objective_status_counts.get(entry['coverage_status'], 0) + 1

    lines.append("")
    lines.append("## Objective Coverage Board")
    lines.append(f"- Objectives: {len(objective_coverage_entries)}")
    lines.append(f"- Personas: {len(objective_persona_counts)}")
    lines.append("")
    lines.append("### By Coverage Status")
    for status, count in sorted(objective_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not objective_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Persona")
    for persona, count in sorted(objective_persona_counts.items()):
        lines.append(f"- {persona}: {count}")
    if not objective_persona_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in objective_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: objective={entry['objective_id']} persona={entry['persona']} priority={entry['priority']} coverage={entry['coverage_status']} dependencies={entry['dependency_count']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  dependency_ids={entry['dependency_ids']} assignments={entry['assignment_ids']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} summary={entry['summary']}"
        )
    if not objective_coverage_entries:
        lines.append("- none")

    persona_readiness_entries = _build_persona_readiness_entries(pack)
    persona_readiness_counts: Dict[str, int] = {}
    for entry in persona_readiness_entries:
        persona_readiness_counts[entry['readiness']] = persona_readiness_counts.get(entry['readiness'], 0) + 1

    lines.append("")
    lines.append("## Persona Readiness Board")
    lines.append(f"- Personas: {len(persona_readiness_entries)}")
    lines.append(f"- Objectives: {len(pack.objectives)}")
    lines.append("")
    lines.append("### By Readiness")
    for readiness, count in sorted(persona_readiness_counts.items()):
        lines.append(f"- {readiness}: {count}")
    if not persona_readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in persona_readiness_entries:
        lines.append(
            f"- {entry['entry_id']}: persona={entry['persona']} readiness={entry['readiness']} objectives={entry['objective_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} open_questions={entry['question_count']} queue_items={entry['queue_count']} blockers={entry['blocker_count']}"
        )
        lines.append(
            f"  objective_ids={entry['objective_ids']} surfaces={entry['surface_ids']} queue_ids={entry['queue_ids']} blocker_ids={entry['blocker_ids']}"
        )
    if not persona_readiness_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Wireframes")
    for wireframe in pack.wireframes:
        lines.append(
            "- "
            f"{wireframe.surface_id}: {wireframe.name} device={wireframe.device} entry={wireframe.entry_point}"
        )
        lines.append(
            "  "
            f"blocks={','.join(wireframe.primary_blocks) or 'none'} review_notes={','.join(wireframe.review_notes) or 'none'}"
        )

    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)
    wireframe_readiness_counts: Dict[str, int] = {}
    wireframe_device_counts: Dict[str, int] = {}
    for entry in wireframe_readiness_entries:
        wireframe_readiness_counts[entry['readiness_status']] = wireframe_readiness_counts.get(entry['readiness_status'], 0) + 1
        wireframe_device_counts[entry['device']] = wireframe_device_counts.get(entry['device'], 0) + 1

    lines.append("")
    lines.append("## Wireframe Readiness Board")
    lines.append(f"- Wireframes: {len(wireframe_readiness_entries)}")
    lines.append(f"- Devices: {len(wireframe_device_counts)}")
    lines.append("")
    lines.append("### By Readiness")
    for status, count in sorted(wireframe_readiness_counts.items()):
        lines.append(f"- {status}: {count}")
    if not wireframe_readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Device")
    for device, count in sorted(wireframe_device_counts.items()):
        lines.append(f"- {device}: {count}")
    if not wireframe_device_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in wireframe_readiness_entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} device={entry['device']} readiness={entry['readiness_status']} open_total={entry['open_total']} entry={entry['entry_point']}"
        )
        lines.append(
            f"  checklist_open={entry['checklist_open']} decisions_open={entry['decisions_open']} assignments_open={entry['assignments_open']} signoffs_open={entry['signoffs_open']} blockers_open={entry['blockers_open']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} blocks={entry['block_count']} notes={entry['note_count']} summary={entry['summary']}"
        )
    if not wireframe_readiness_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Interactions")
    for interaction in pack.interactions:
        lines.append(
            "- "
            f"{interaction.flow_id}: {interaction.name} trigger={interaction.trigger}"
        )
        lines.append(
            "  "
            f"response={interaction.system_response} states={','.join(interaction.states) or 'none'} exceptions={','.join(interaction.exceptions) or 'none'}"
        )

    interaction_coverage_entries = _build_interaction_coverage_entries(pack)
    interaction_coverage_counts: Dict[str, int] = {}
    interaction_surface_counts: Dict[str, int] = {}
    for entry in interaction_coverage_entries:
        interaction_coverage_counts[entry['coverage_status']] = interaction_coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1

    lines.append("")
    lines.append("## Interaction Coverage Board")
    lines.append(f"- Interactions: {len(interaction_coverage_entries)}")
    lines.append(f"- Surfaces: {len(interaction_surface_counts)}")
    lines.append("")
    lines.append("### By Coverage Status")
    for status, count in sorted(interaction_coverage_counts.items()):
        lines.append(f"- {status}: {count}")
    if not interaction_coverage_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, count in sorted(interaction_surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not interaction_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in interaction_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: flow={entry['flow_id']} surfaces={entry['surface_ids']} owners={entry['owners']} coverage={entry['coverage_status']} states={entry['state_count']} exceptions={entry['exception_count']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} open_checklist={entry['open_checklist_ids']} trigger={entry['summary']}"
        )
    if not interaction_coverage_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Open Questions")
    for question in pack.open_questions:
        lines.append(
            "- "
            f"{question.question_id}: {question.theme} owner={question.owner} status={question.status}"
        )
        lines.append("  " f"question={question.question} impact={question.impact}")

    open_question_entries = _build_open_question_tracker_entries(pack)
    open_question_owner_counts: Dict[str, int] = {}
    open_question_theme_counts: Dict[str, int] = {}
    for entry in open_question_entries:
        open_question_owner_counts[entry['owner']] = open_question_owner_counts.get(entry['owner'], 0) + 1
        open_question_theme_counts[entry['theme']] = open_question_theme_counts.get(entry['theme'], 0) + 1

    lines.append("")
    lines.append("## Open Question Tracker")
    lines.append(f"- Questions: {len(open_question_entries)}")
    lines.append(f"- Owners: {len(open_question_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(open_question_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not open_question_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Theme")
    for theme, count in sorted(open_question_theme_counts.items()):
        lines.append(f"- {theme}: {count}")
    if not open_question_theme_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in open_question_entries:
        lines.append(
            f"- {entry['entry_id']}: question={entry['question_id']} owner={entry['owner']} theme={entry['theme']} status={entry['status']} link_status={entry['link_status']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} flows={entry['flow_ids']} impact={entry['impact']} prompt={entry['summary']}"
        )
    if not open_question_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Reviewer Checklist")
    for item in pack.reviewer_checklist:
        lines.append(
            "- "
            f"{item.item_id}: surface={item.surface_id} owner={item.owner} status={item.status}"
        )
        lines.append(
            "  "
            f"prompt={item.prompt} evidence={','.join(item.evidence_links) or 'none'} notes={item.notes or 'none'}"
        )
    if not pack.reviewer_checklist:
        lines.append("- none")

    lines.append("")
    lines.append("## Decision Log")
    for decision in pack.decision_log:
        lines.append(
            "- "
            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"
        )
        lines.append(
            "  "
            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or 'none'}"
        )
    if not pack.decision_log:
        lines.append("- none")

    lines.append("")
    lines.append("## Role Matrix")
    for assignment in pack.role_matrix:
        lines.append(
            "- "
            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"
        )
        lines.append(
            "  "
            f"responsibilities={','.join(assignment.responsibilities) or 'none'} checklist={','.join(assignment.checklist_item_ids) or 'none'} decisions={','.join(assignment.decision_ids) or 'none'}"
        )
    if not pack.role_matrix:
        lines.append("- none")

    checklist_trace_entries = _build_checklist_traceability_entries(pack)
    checklist_trace_owner_counts: Dict[str, int] = {}
    checklist_trace_status_counts: Dict[str, int] = {}
    for entry in checklist_trace_entries:
        checklist_trace_owner_counts[entry['owner']] = checklist_trace_owner_counts.get(entry['owner'], 0) + 1
        checklist_trace_status_counts[entry['status']] = checklist_trace_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Checklist Traceability Board")
    lines.append(f"- Checklist items: {len(checklist_trace_entries)}")
    lines.append(f"- Owners: {len(checklist_trace_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(checklist_trace_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not checklist_trace_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(checklist_trace_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not checklist_trace_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in checklist_trace_entries:
        lines.append(
            f"- {entry['entry_id']}: item={entry['item_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_decisions={entry['linked_decisions']} evidence={entry['evidence']} summary={entry['summary']}"
        )
    if not checklist_trace_entries:
        lines.append("- none")

    decision_followup_entries = _build_decision_followup_entries(pack)
    decision_followup_owner_counts: Dict[str, int] = {}
    decision_followup_status_counts: Dict[str, int] = {}
    for entry in decision_followup_entries:
        decision_followup_owner_counts[entry['owner']] = decision_followup_owner_counts.get(entry['owner'], 0) + 1
        decision_followup_status_counts[entry['status']] = decision_followup_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Decision Follow-up Tracker")
    lines.append(f"- Decisions: {len(decision_followup_entries)}")
    lines.append(f"- Owners: {len(decision_followup_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(decision_followup_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not decision_followup_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(decision_followup_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not decision_followup_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in decision_followup_entries:
        lines.append(
            f"- {entry['entry_id']}: decision={entry['decision_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_checklists={entry['linked_checklists']} follow_up={entry['follow_up']} summary={entry['summary']}"
        )
    if not decision_followup_entries:
        lines.append("- none")

    role_coverage_entries = _build_role_coverage_entries(pack)
    role_coverage_surface_counts: Dict[str, int] = {}
    role_coverage_status_counts: Dict[str, int] = {}
    for entry in role_coverage_entries:
        role_coverage_surface_counts[entry['surface_id']] = role_coverage_surface_counts.get(entry['surface_id'], 0) + 1
        role_coverage_status_counts[entry['status']] = role_coverage_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Role Coverage Board")
    lines.append(f"- Assignments: {len(role_coverage_entries)}")
    lines.append(f"- Surfaces: {len(role_coverage_surface_counts)}")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, count in sorted(role_coverage_surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not role_coverage_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(role_coverage_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not role_coverage_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in role_coverage_entries:
        lines.append(
            f"- {entry['entry_id']}: assignment={entry['assignment_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} responsibilities={entry['responsibility_count']} checklist={entry['checklist_count']} decisions={entry['decision_count']}"
        )
        lines.append(
            f"  signoff={entry['signoff_id']} signoff_status={entry['signoff_status']} summary={entry['summary']}"
        )
    if not role_coverage_entries:
        lines.append("- none")

    signoff_dependency_entries = _build_signoff_dependency_entries(pack)
    dependency_counts: Dict[str, int] = {}
    dependency_sla_counts: Dict[str, int] = {}
    for entry in signoff_dependency_entries:
        dependency_counts[entry['dependency_status']] = dependency_counts.get(entry['dependency_status'], 0) + 1
        dependency_sla_counts[entry['sla_status']] = dependency_sla_counts.get(entry['sla_status'], 0) + 1

    lines.append("")
    lines.append("## Signoff Dependency Board")
    lines.append(f"- Sign-offs: {len(signoff_dependency_entries)}")
    lines.append(f"- Dependency states: {len(dependency_counts)}")
    lines.append("")
    lines.append("### By Dependency Status")
    for status, count in sorted(dependency_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By SLA State")
    for status, count in sorted(dependency_sla_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_sla_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in signoff_dependency_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} dependency_status={entry['dependency_status']} blockers={entry['blocker_ids']}"
        )
        lines.append(
            f"  assignment={entry['assignment_id']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} latest_blocker_event={entry['latest_blocker_event']} sla={entry['sla_status']} due_at={entry['due_at']} cadence={entry['reminder_cadence']} summary={entry['summary']}"
        )
    if not signoff_dependency_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Sign-off Log")
    for signoff in pack.signoff_log:
        lines.append(
            "- "
            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"
        )
        lines.append(
            "  "
            f"required={'yes' if signoff.required else 'no'} evidence={','.join(signoff.evidence_links) or 'none'} notes={signoff.notes or 'none'} waiver_owner={signoff.waiver_owner or 'none'} waiver_reason={signoff.waiver_reason or 'none'} requested_at={signoff.requested_at or 'none'} due_at={signoff.due_at or 'none'} escalation_owner={signoff.escalation_owner or 'none'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or 'none'} reminder_channel={signoff.reminder_channel or 'none'} last_reminder_at={signoff.last_reminder_at or 'none'} next_reminder_at={signoff.next_reminder_at or 'none'}"
        )
    if not pack.signoff_log:
        lines.append("- none")

    signoff_sla_entries = _build_signoff_sla_entries(pack)
    sla_state_counts: Dict[str, int] = {}
    sla_owner_counts: Dict[str, int] = {}
    for entry in signoff_sla_entries:
        sla_state_counts[entry['sla_status']] = sla_state_counts.get(entry['sla_status'], 0) + 1
        sla_owner_counts[entry['escalation_owner']] = sla_owner_counts.get(entry['escalation_owner'], 0) + 1

    lines.append("")
    lines.append("## Sign-off SLA Dashboard")
    lines.append(f"- Sign-offs: {len(signoff_sla_entries)}")
    lines.append(f"- Escalation owners: {len(sla_owner_counts)}")
    lines.append("")
    lines.append("### SLA States")
    for sla_status, count in sorted(sla_state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not sla_state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalation Owners")
    for owner, count in sorted(sla_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not sla_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Sign-offs")
    for entry in signoff_sla_entries:
        lines.append(
            f"- {entry['signoff_id']}: role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} requested_at={entry['requested_at']} due_at={entry['due_at']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(f"  required={entry['required']} evidence={entry['evidence']}")
    if not signoff_sla_entries:
        lines.append("- none")

    signoff_reminder_entries = _build_signoff_reminder_entries(pack)
    reminder_owner_counts: Dict[str, int] = {}
    reminder_channel_counts: Dict[str, int] = {}
    for entry in signoff_reminder_entries:
        reminder_owner_counts[entry['reminder_owner']] = reminder_owner_counts.get(entry['reminder_owner'], 0) + 1
        reminder_channel_counts[entry['reminder_channel']] = reminder_channel_counts.get(entry['reminder_channel'], 0) + 1

    lines.append("")
    lines.append("## Sign-off Reminder Queue")
    lines.append(f"- Reminders: {len(signoff_reminder_entries)}")
    lines.append(f"- Owners: {len(reminder_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, count in sorted(reminder_owner_counts.items()):
        lines.append(f"- {owner}: reminders={count}")
    if not reminder_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Channel")
    for channel, count in sorted(reminder_channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not reminder_channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in signoff_reminder_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} owner={entry['reminder_owner']} channel={entry['reminder_channel']}"
        )
        lines.append(
            f"  last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not signoff_reminder_entries:
        lines.append("- none")

    reminder_cadence_entries = _build_reminder_cadence_entries(pack)
    reminder_cadence_counts: Dict[str, int] = {}
    reminder_status_counts: Dict[str, int] = {}
    for entry in reminder_cadence_entries:
        reminder_cadence_counts[entry['reminder_cadence']] = reminder_cadence_counts.get(entry['reminder_cadence'], 0) + 1
        reminder_status_counts[entry['reminder_status']] = reminder_status_counts.get(entry['reminder_status'], 0) + 1

    lines.append("")
    lines.append("## Reminder Cadence Board")
    lines.append(f"- Items: {len(reminder_cadence_entries)}")
    lines.append(f"- Cadences: {len(reminder_cadence_counts)}")
    lines.append("")
    lines.append("### By Cadence")
    for cadence, count in sorted(reminder_cadence_counts.items()):
        lines.append(f"- {cadence}: {count}")
    if not reminder_cadence_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(reminder_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not reminder_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in reminder_cadence_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} cadence={entry['reminder_cadence']} status={entry['reminder_status']} owner={entry['reminder_owner']}"
        )
        lines.append(
            f"  sla={entry['sla_status']} last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not reminder_cadence_entries:
        lines.append("- none")

    signoff_breach_entries = _build_signoff_breach_entries(pack)
    breach_state_counts: Dict[str, int] = {}
    breach_owner_counts: Dict[str, int] = {}
    for entry in signoff_breach_entries:
        breach_state_counts[entry['sla_status']] = breach_state_counts.get(entry['sla_status'], 0) + 1
        breach_owner_counts[entry['escalation_owner']] = breach_owner_counts.get(entry['escalation_owner'], 0) + 1

    lines.append("")
    lines.append("## Sign-off Breach Board")
    lines.append(f"- Breach items: {len(signoff_breach_entries)}")
    lines.append(f"- Escalation owners: {len(breach_owner_counts)}")
    lines.append("")
    lines.append("### SLA States")
    for sla_status, count in sorted(breach_state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not breach_state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalation Owners")
    for owner, count in sorted(breach_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not breach_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in signoff_breach_entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(
            f"  requested_at={entry['requested_at']} due_at={entry['due_at']} linked_blockers={entry['linked_blockers']} summary={entry['summary']}"
        )
    if not signoff_breach_entries:
        lines.append("- none")

    escalation_entries = _build_escalation_dashboard_entries(pack)
    escalation_owner_counts: Dict[str, Dict[str, int]] = {}
    escalation_status_counts: Dict[str, Dict[str, int]] = {}
    for entry in escalation_entries:
        owner_counts = escalation_owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_counts[entry['item_type']] += 1
        owner_counts['total'] += 1
        status_counts = escalation_status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_counts[entry['item_type']] += 1
        status_counts['total'] += 1

    lines.append("")
    lines.append("## Escalation Dashboard")
    lines.append(f"- Items: {len(escalation_entries)}")
    lines.append(f"- Escalation owners: {len(escalation_owner_counts)}")
    lines.append("")
    lines.append("### By Escalation Owner")
    for owner, counts in sorted(escalation_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not escalation_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, counts in sorted(escalation_status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not escalation_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Escalations")
    for entry in escalation_entries:
        lines.append(
            f"- {entry['escalation_id']}: owner={entry['escalation_owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} priority={entry['priority']} current_owner={entry['current_owner']}"
        )
        lines.append(f"  summary={entry['summary']} due_at={entry['due_at']}")
    if not escalation_entries:
        lines.append("- none")

    escalation_handoff_entries = _build_escalation_handoff_entries(pack)
    handoff_status_counts: Dict[str, int] = {}
    handoff_channel_counts: Dict[str, int] = {}
    for entry in escalation_handoff_entries:
        handoff_status_counts[entry['status']] = handoff_status_counts.get(entry['status'], 0) + 1
        handoff_channel_counts[entry['channel']] = handoff_channel_counts.get(entry['channel'], 0) + 1

    lines.append("")
    lines.append("## Escalation Handoff Ledger")
    lines.append(f"- Handoffs: {len(escalation_handoff_entries)}")
    lines.append(f"- Channels: {len(handoff_channel_counts)}")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(handoff_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not handoff_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Channel")
    for channel, count in sorted(handoff_channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not handoff_channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in escalation_handoff_entries:
        lines.append(
            f"- {entry['ledger_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} actor={entry['actor']} status={entry['status']} at={entry['timestamp']}"
        )
        lines.append(
            f"  from={entry['handoff_from']} to={entry['handoff_to']} channel={entry['channel']} artifact={entry['artifact_ref']} next_action={entry['next_action']}"
        )
    if not escalation_handoff_entries:
        lines.append("- none")

    handoff_ack_entries = _build_handoff_ack_entries(pack)
    handoff_ack_owner_counts: Dict[str, int] = {}
    handoff_ack_status_counts: Dict[str, int] = {}
    for entry in handoff_ack_entries:
        handoff_ack_owner_counts[entry['ack_owner']] = handoff_ack_owner_counts.get(entry['ack_owner'], 0) + 1
        handoff_ack_status_counts[entry['ack_status']] = handoff_ack_status_counts.get(entry['ack_status'], 0) + 1

    lines.append("")
    lines.append("## Handoff Ack Ledger")
    lines.append(f"- Ack items: {len(handoff_ack_entries)}")
    lines.append(f"- Ack owners: {len(handoff_ack_owner_counts)}")
    lines.append("")
    lines.append("### By Ack Owner")
    for owner, count in sorted(handoff_ack_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not handoff_ack_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Ack Status")
    for status, count in sorted(handoff_ack_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not handoff_ack_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in handoff_ack_entries:
        lines.append(
            f"- {entry['entry_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} handoff_to={entry['handoff_to']} ack_owner={entry['ack_owner']} ack_status={entry['ack_status']} ack_at={entry['ack_at']}"
        )
        lines.append(
            f"  actor={entry['actor']} status={entry['status']} channel={entry['channel']} artifact={entry['artifact_ref']} summary={entry['summary']}"
        )
    if not handoff_ack_entries:
        lines.append("- none")

    owner_digest_entries = _build_owner_escalation_digest_entries(pack)
    owner_digest_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_digest_entries:
        counts = owner_digest_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'signoff': 0, 'reminder': 0, 'freeze': 0, 'handoff': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1

    lines.append("")
    lines.append("## Owner Escalation Digest")
    lines.append(f"- Owners: {len(owner_digest_counts)}")
    lines.append(f"- Items: {len(owner_digest_entries)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_digest_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} reminders={counts['reminder']} freezes={counts['freeze']} handoffs={counts['handoff']} total={counts['total']}"
        )
    if not owner_digest_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_digest_entries:
        lines.append(
            f"- {entry['digest_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} detail={entry['detail']}")
    if not owner_digest_entries:
        lines.append("- none")

    owner_workload_entries = _build_owner_workload_entries(pack)
    owner_workload_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_workload_entries:
        counts = owner_workload_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'checklist': 0, 'decision': 0, 'signoff': 0, 'reminder': 0, 'renewal': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1

    lines.append("")
    lines.append("## Owner Workload Board")
    lines.append(f"- Owners: {len(owner_workload_counts)}")
    lines.append(f"- Items: {len(owner_workload_entries)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_workload_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} reminders={counts['reminder']} renewals={counts['renewal']} total={counts['total']}"
        )
    if not owner_workload_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_workload_entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} lane={entry['lane']}"
        )
        lines.append(f"  detail={entry['detail']} summary={entry['summary']}")
    if not owner_workload_entries:
        lines.append("- none")

    lines.append("")
    lines.append("## Blocker Log")
    for blocker in pack.blocker_log:
        lines.append(
            "- "
            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"
        )
        lines.append(
            "  "
            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or 'none'} next_action={blocker.next_action or 'none'} freeze_owner={blocker.freeze_owner or 'none'} freeze_until={blocker.freeze_until or 'none'} freeze_approved_by={blocker.freeze_approved_by or 'none'} freeze_approved_at={blocker.freeze_approved_at or 'none'}"
        )
    if not pack.blocker_log:
        lines.append("- none")

    lines.append("")
    lines.append("## Blocker Timeline")
    for event in pack.blocker_timeline:
        lines.append(
            "- "
            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"
        )
        lines.append(
            "  "
            f"summary={event.summary} next_action={event.next_action or 'none'}"
        )
    if not pack.blocker_timeline:
        lines.append("- none")

    exception_entries = _build_review_exception_entries(pack)
    timeline_index = _build_blocker_timeline_index(pack)

    lines.append("")
    lines.append("## Review Exceptions")
    for entry in exception_entries:
        lines.append(
            f"- {entry['exception_id']}: type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")

    freeze_entries = _build_freeze_exception_entries(pack)
    freeze_owner_counts: Dict[str, Dict[str, int]] = {}
    freeze_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in freeze_entries:
        owner_counts = freeze_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_counts[entry["item_type"]] += 1
        owner_counts["total"] += 1
        surface_counts = freeze_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_counts[entry["item_type"]] += 1
        surface_counts["total"] += 1

    lines.append("")
    lines.append("## Review Freeze Exception Board")
    lines.append(f"- Exceptions: {len(freeze_entries)}")
    lines.append(f"- Owners: {len(freeze_owner_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, counts in sorted(freeze_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not freeze_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, counts in sorted(freeze_surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not freeze_surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} window={entry['window']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} next_action={entry['next_action']}"
        )
    if not freeze_entries:
        lines.append("- none")

    freeze_approval_entries = _build_freeze_approval_entries(pack)
    freeze_approval_owner_counts: Dict[str, int] = {}
    freeze_approval_status_counts: Dict[str, int] = {}
    for entry in freeze_approval_entries:
        freeze_approval_owner_counts[entry['freeze_approved_by']] = freeze_approval_owner_counts.get(entry['freeze_approved_by'], 0) + 1
        freeze_approval_status_counts[entry['status']] = freeze_approval_status_counts.get(entry['status'], 0) + 1

    lines.append("")
    lines.append("## Freeze Approval Trail")
    lines.append(f"- Approvals: {len(freeze_approval_entries)}")
    lines.append(f"- Approvers: {len(freeze_approval_owner_counts)}")
    lines.append("")
    lines.append("### By Approver")
    for owner, count in sorted(freeze_approval_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not freeze_approval_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, count in sorted(freeze_approval_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not freeze_approval_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_approval_entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} owner={entry['freeze_owner']} approved_by={entry['freeze_approved_by']} approved_at={entry['freeze_approved_at']} window={entry['freeze_until']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not freeze_approval_entries:
        lines.append("- none")

    freeze_renewal_entries = _build_freeze_renewal_entries(pack)
    freeze_renewal_owner_counts: Dict[str, int] = {}
    freeze_renewal_status_counts: Dict[str, int] = {}
    for entry in freeze_renewal_entries:
        freeze_renewal_owner_counts[entry['renewal_owner']] = freeze_renewal_owner_counts.get(entry['renewal_owner'], 0) + 1
        freeze_renewal_status_counts[entry['renewal_status']] = freeze_renewal_status_counts.get(entry['renewal_status'], 0) + 1

    lines.append("")
    lines.append("## Freeze Renewal Tracker")
    lines.append(f"- Renewal items: {len(freeze_renewal_entries)}")
    lines.append(f"- Renewal owners: {len(freeze_renewal_owner_counts)}")
    lines.append("")
    lines.append("### By Renewal Owner")
    for owner, count in sorted(freeze_renewal_owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not freeze_renewal_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Renewal Status")
    for status, count in sorted(freeze_renewal_status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not freeze_renewal_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in freeze_renewal_entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} renewal_owner={entry['renewal_owner']} renewal_by={entry['renewal_by']} renewal_status={entry['renewal_status']}"
        )
        lines.append(
            f"  freeze_owner={entry['freeze_owner']} freeze_until={entry['freeze_until']} approved_by={entry['freeze_approved_by']} summary={entry['summary']} next_action={entry['next_action']}"
        )
    if not freeze_renewal_entries:
        lines.append("- none")

    exception_owner_counts: Dict[str, Dict[str, int]] = {}
    exception_status_counts: Dict[str, Dict[str, int]] = {}
    exception_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_counts = exception_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_counts[entry["category"]] += 1
        owner_counts["total"] += 1
        status_counts = exception_status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_counts[entry["category"]] += 1
        status_counts["total"] += 1
        surface_counts = exception_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_counts[entry["category"]] += 1
        surface_counts["total"] += 1

    lines.append("")
    lines.append("## Review Exception Matrix")
    lines.append(f"- Exceptions: {len(exception_entries)}")
    lines.append(f"- Owners: {len(exception_owner_counts)}")
    lines.append(f"- Surfaces: {len(exception_surface_counts)}")
    lines.append("")
    lines.append("### By Owner")
    for owner, counts in sorted(exception_owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Status")
    for status, counts in sorted(exception_status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### By Surface")
    for surface_id, counts in sorted(exception_surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not exception_surface_counts:
        lines.append("- none")

    audit_density_entries = _build_audit_density_entries(pack)
    audit_density_band_counts: Dict[str, int] = {}
    for entry in audit_density_entries:
        audit_density_band_counts[entry['load_band']] = audit_density_band_counts.get(entry['load_band'], 0) + 1

    lines.append("")
    lines.append("## Audit Density Board")
    lines.append(f"- Surfaces: {len(audit_density_entries)}")
    lines.append(f"- Load bands: {len(audit_density_band_counts)}")
    lines.append("")
    lines.append("### By Load Band")
    for band, count in sorted(audit_density_band_counts.items()):
        lines.append(f"- {band}: {count}")
    if not audit_density_band_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Entries")
    for entry in audit_density_entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} artifact_total={entry['artifact_total']} open_total={entry['open_total']} band={entry['load_band']}"
        )
        lines.append(
            f"  checklist={entry['checklist_count']} decisions={entry['decision_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} blockers={entry['blocker_count']} timeline={entry['timeline_count']} blocks={entry['block_count']} notes={entry['note_count']}"
        )
    if not audit_density_entries:
        lines.append("- none")

    owner_review_queue = _build_owner_review_queue_entries(pack)
    owner_queue_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_review_queue:
        counts = owner_queue_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1

    lines.append("")
    lines.append("## Owner Review Queue")
    lines.append(f"- Owners: {len(owner_queue_counts)}")
    lines.append(f"- Queue items: {len(owner_review_queue)}")
    lines.append("")
    lines.append("### Owners")
    for owner, counts in sorted(owner_queue_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_queue_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Items")
    for entry in owner_review_queue:
        lines.append(
            f"- {entry['queue_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} next_action={entry['next_action']}")
    if not owner_review_queue:
        lines.append("- none")

    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )

    lines.append("")
    lines.append("## Blocker Timeline Summary")
    lines.append(f"- Total events: {len(pack.blocker_timeline)}")
    lines.append(f"- Blockers with timeline: {len(timeline_index)}")
    lines.append(f"- Orphan timeline blockers: {','.join(orphan_timeline_ids) or 'none'}")
    lines.append("")
    lines.append("### Events by Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Events by Actor")
    for actor, count in sorted(actor_counts.items()):
        lines.append(f"- {actor}: {count}")
    if not actor_counts:
        lines.append("- none")
    lines.append("")
    lines.append("### Latest Blocker Events")
    for blocker in pack.blocker_log:
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        if latest is None:
            lines.append(f"- {blocker.blocker_id}: latest=none")
            continue
        lines.append(
            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"
        )
    if not pack.blocker_log:
        lines.append("- none")

    lines.extend(
        [
            "",
            "## Audit Findings",
            f"- Missing sections: {', '.join(audit.missing_sections) or 'none'}",
            f"- Objectives missing success signals: {', '.join(audit.objectives_missing_signals) or 'none'}",
            f"- Wireframes missing blocks: {', '.join(audit.wireframes_missing_blocks) or 'none'}",
            f"- Interactions missing states: {', '.join(audit.interactions_missing_states) or 'none'}",
            f"- Unresolved questions: {', '.join(audit.unresolved_question_ids) or 'none'}",
            f"- Wireframes missing checklist coverage: {', '.join(audit.wireframes_missing_checklists) or 'none'}",
            f"- Orphan checklist surfaces: {', '.join(audit.orphan_checklist_surfaces) or 'none'}",
            f"- Checklist items missing evidence: {', '.join(audit.checklist_items_missing_evidence) or 'none'}",
            f"- Checklist items missing role links: {', '.join(audit.checklist_items_missing_role_links) or 'none'}",
            f"- Wireframes missing decision coverage: {', '.join(audit.wireframes_missing_decisions) or 'none'}",
            f"- Orphan decision surfaces: {', '.join(audit.orphan_decision_surfaces) or 'none'}",
            f"- Unresolved decision ids: {', '.join(audit.unresolved_decision_ids) or 'none'}",
            f"- Unresolved decisions missing follow-ups: {', '.join(audit.unresolved_decisions_missing_follow_ups) or 'none'}",
            f"- Wireframes missing role assignments: {', '.join(audit.wireframes_missing_role_assignments) or 'none'}",
            f"- Orphan role assignment surfaces: {', '.join(audit.orphan_role_assignment_surfaces) or 'none'}",
            f"- Role assignments missing responsibilities: {', '.join(audit.role_assignments_missing_responsibilities) or 'none'}",
            f"- Role assignments missing checklist links: {', '.join(audit.role_assignments_missing_checklist_links) or 'none'}",
            f"- Role assignments missing decision links: {', '.join(audit.role_assignments_missing_decision_links) or 'none'}",
            f"- Decisions missing role links: {', '.join(audit.decisions_missing_role_links) or 'none'}",
            f"- Wireframes missing signoff coverage: {', '.join(audit.wireframes_missing_signoffs) or 'none'}",
            f"- Orphan signoff surfaces: {', '.join(audit.orphan_signoff_surfaces) or 'none'}",
            f"- Signoffs missing role assignments: {', '.join(audit.signoffs_missing_assignments) or 'none'}",
            f"- Signoffs missing evidence: {', '.join(audit.signoffs_missing_evidence) or 'none'}",
            f"- Signoffs missing requested dates: {', '.join(audit.signoffs_missing_requested_dates) or 'none'}",
            f"- Signoffs missing due dates: {', '.join(audit.signoffs_missing_due_dates) or 'none'}",
            f"- Signoffs missing escalation owners: {', '.join(audit.signoffs_missing_escalation_owners) or 'none'}",
            f"- Signoffs missing reminder owners: {', '.join(audit.signoffs_missing_reminder_owners) or 'none'}",
            f"- Signoffs missing next reminders: {', '.join(audit.signoffs_missing_next_reminders) or 'none'}",
            f"- Signoffs missing reminder cadence: {', '.join(audit.signoffs_missing_reminder_cadence) or 'none'}",
            f"- Signoffs with breached SLA: {', '.join(audit.signoffs_with_breached_sla) or 'none'}",
            f"- Waived signoffs missing metadata: {', '.join(audit.waived_signoffs_missing_metadata) or 'none'}",
            f"- Unresolved required signoff ids: {', '.join(audit.unresolved_required_signoff_ids) or 'none'}",
            f"- Blockers missing signoff links: {', '.join(audit.blockers_missing_signoff_links) or 'none'}",
            f"- Blockers missing escalation owners: {', '.join(audit.blockers_missing_escalation_owners) or 'none'}",
            f"- Blockers missing next actions: {', '.join(audit.blockers_missing_next_actions) or 'none'}",
            f"- Freeze exceptions missing owners: {', '.join(audit.freeze_exceptions_missing_owners) or 'none'}",
            f"- Freeze exceptions missing windows: {', '.join(audit.freeze_exceptions_missing_until) or 'none'}",
            f"- Freeze exceptions missing approvers: {', '.join(audit.freeze_exceptions_missing_approvers) or 'none'}",
            f"- Freeze exceptions missing approval dates: {', '.join(audit.freeze_exceptions_missing_approval_dates) or 'none'}",
            f"- Freeze exceptions missing renewal owners: {', '.join(audit.freeze_exceptions_missing_renewal_owners) or 'none'}",
            f"- Freeze exceptions missing renewal dates: {', '.join(audit.freeze_exceptions_missing_renewal_dates) or 'none'}",
            f"- Blockers missing timeline events: {', '.join(audit.blockers_missing_timeline_events) or 'none'}",
            f"- Closed blockers missing resolution events: {', '.join(audit.closed_blockers_missing_resolution_events) or 'none'}",
            f"- Orphan blocker surfaces: {', '.join(audit.orphan_blocker_surfaces) or 'none'}",
            f"- Orphan blocker timeline blocker ids: {', '.join(audit.orphan_blocker_timeline_blocker_ids) or 'none'}",
            f"- Handoff events missing targets: {', '.join(audit.handoff_events_missing_targets) or 'none'}",
            f"- Handoff events missing artifacts: {', '.join(audit.handoff_events_missing_artifacts) or 'none'}",
            f"- Handoff events missing ack owners: {', '.join(audit.handoff_events_missing_ack_owners) or 'none'}",
            f"- Handoff events missing ack dates: {', '.join(audit.handoff_events_missing_ack_dates) or 'none'}",
            f"- Unresolved required signoffs without blockers: {', '.join(audit.unresolved_required_signoffs_without_blockers) or 'none'}",
        ]
    )
    return "\n".join(lines)


def build_big_4204_review_pack() -> UIReviewPack:
    return UIReviewPack(
        issue_id="BIG-4204",
        title="UI评审包输出",
        version="v4.0-design-sprint",
        requires_reviewer_checklist=True,
        requires_decision_log=True,
        requires_role_matrix=True,
        requires_signoff_log=True,
        requires_blocker_log=True,
        requires_blocker_timeline=True,
        objectives=[
            ReviewObjective(
                objective_id="obj-overview-decision",
                title="Validate the executive overview narrative and drill-down posture",
                persona="VP Eng",
                outcome="Leadership can confirm the overview page balances KPI density with investigation entry points.",
                success_signal="Reviewers agree the overview supports release, risk, and queue drill-down without extra walkthroughs.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-132"],
            ),
            ReviewObjective(
                objective_id="obj-queue-governance",
                title="Confirm queue control actions and approval posture",
                persona="Platform Admin",
                outcome="Operators can assess batch approvals, audit visibility, and denial paths from one frame.",
                success_signal="The queue frame clearly shows allowed actions, denied roles, and audit expectations.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-131", "OPE-132"],
            ),
            ReviewObjective(
                objective_id="obj-run-detail-investigation",
                title="Validate replay and audit investigation flow",
                persona="Eng Lead",
                outcome="Run detail reviewers can trace evidence, replay context, and escalation actions in one surface.",
                success_signal="The run-detail frame makes failure replay and escalation decisions reviewable without hidden dependencies.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-72", "OPE-73"],
            ),
            ReviewObjective(
                objective_id="obj-triage-handoff",
                title="Confirm triage and cross-team handoff readiness",
                persona="Cross-Team Operator",
                outcome="Reviewers can evaluate assignment, handoff, and queue-state transitions as one operator journey.",
                success_signal="The triage frame exposes action states, owner switches, and handoff exceptions explicitly.",
                priority="P0",
                dependencies=["BIG-4203", "OPE-76", "OPE-79", "OPE-132"],
            ),
        ],
        wireframes=[
            WireframeSurface(
                surface_id="wf-overview",
                name="Overview command deck",
                device="desktop",
                entry_point="/overview",
                primary_blocks=["top bar", "kpi strip", "risk panel", "drill-down table"],
                review_notes=["Confirm metric density and executive scan path.", "Check alert prominence versus weekly summary card."],
            ),
            WireframeSurface(
                surface_id="wf-queue",
                name="Queue control center",
                device="desktop",
                entry_point="/queue",
                primary_blocks=["approval queue", "selection toolbar", "filters", "audit rail"],
                review_notes=["Validate batch-approve CTA hierarchy.", "Review denied-role behavior for non-operator personas."],
            ),
            WireframeSurface(
                surface_id="wf-run-detail",
                name="Run detail and replay",
                device="desktop",
                entry_point="/runs/detail",
                primary_blocks=["timeline", "artifact drawer", "replay controls", "audit notes"],
                review_notes=["Check replay mode discoverability.", "Ensure escalation path is visible next to audit evidence."],
            ),
            WireframeSurface(
                surface_id="wf-triage",
                name="Triage and handoff board",
                device="desktop",
                entry_point="/triage",
                primary_blocks=["severity lanes", "bulk actions", "handoff panel", "ownership history"],
                review_notes=["Validate cross-team operator workflow.", "Confirm exception path for denied escalation."],
            ),
        ],
        interactions=[
            InteractionFlow(
                flow_id="flow-overview-drilldown",
                name="Overview to investigation drill-down",
                trigger="VP Eng selects a KPI card or blocker cluster on the overview page",
                system_response="The console pivots into the matching queue or run-detail slice while preserving context filters.",
                states=["default", "focus", "handoff-ready"],
                exceptions=["Warn when the requested slice is permission-denied.", "Show fallback summary when no matching runs exist."],
            ),
            InteractionFlow(
                flow_id="flow-queue-bulk-approval",
                name="Queue batch approval review",
                trigger="Platform Admin selects multiple tasks and opens the bulk approval toolbar",
                system_response="The queue shows approval scope, audit consequence, and denied-role messaging before submit.",
                states=["default", "selection", "confirming", "success"],
                exceptions=["Disable submit when tasks cross unauthorized scopes.", "Route to audit timeline when approval policy changes mid-flow."],
            ),
            InteractionFlow(
                flow_id="flow-run-replay",
                name="Run replay with evidence audit",
                trigger="Eng Lead switches replay mode on a failed run",
                system_response="The page updates the timeline, artifacts, and escalation controls while keeping the audit trail visible.",
                states=["default", "replay", "compare", "escalated"],
                exceptions=["Show replay-unavailable state for incomplete artifacts.", "Require escalation reason before handoff."],
            ),
            InteractionFlow(
                flow_id="flow-triage-handoff",
                name="Triage ownership reassignment and handoff",
                trigger="Cross-Team Operator bulk-assigns a finding set or opens the handoff panel",
                system_response="The triage board updates owner, workflow, and handoff evidence in one acknowledgement step.",
                states=["default", "selected", "handoff", "completed"],
                exceptions=["Block handoff when reviewer coverage is incomplete.", "Record denied-role attempt in the audit summary."],
            ),
        ],
        open_questions=[
            OpenQuestion(
                question_id="oq-role-density",
                theme="role-matrix",
                question="Should VP Eng see queue batch controls in read-only form or be routed to a summary-only state?",
                owner="product-experience",
                impact="Changes denial-path copy, button placement, and review criteria for queue and triage pages.",
            ),
            OpenQuestion(
                question_id="oq-alert-priority",
                theme="information-architecture",
                question="Should regression alerts outrank approval alerts in the top bar for the design sprint prototype?",
                owner="engineering-operations",
                impact="Affects alert hierarchy and the scan path used in the overview and triage reviews.",
            ),
            OpenQuestion(
                question_id="oq-handoff-evidence",
                theme="handoff",
                question="How much ownership history must stay visible before the run-detail and triage pages collapse older audit entries?",
                owner="orchestration-office",
                impact="Shapes the default density of the audit rail and the threshold for the review-ready packet.",
            ),
        ],
        reviewer_checklist=[
            ReviewerChecklistItem(
                item_id="chk-overview-kpi-scan",
                surface_id="wf-overview",
                prompt="Verify the KPI strip still supports one-screen executive scanning before drill-down.",
                owner="VP Eng",
                status="ready",
                evidence_links=["wf-overview", "flow-overview-drilldown"],
                notes="Use the overview card hierarchy as the primary decision frame.",
            ),
            ReviewerChecklistItem(
                item_id="chk-overview-alert-hierarchy",
                surface_id="wf-overview",
                prompt="Confirm alert priority is readable when approvals and regressions compete for attention.",
                owner="engineering-operations",
                status="open",
                evidence_links=["wf-overview", "oq-alert-priority"],
            ),
            ReviewerChecklistItem(
                item_id="chk-queue-batch-approval",
                surface_id="wf-queue",
                prompt="Check that batch approval clearly communicates scope, denial paths, and audit consequences.",
                owner="Platform Admin",
                status="ready",
                evidence_links=["wf-queue", "flow-queue-bulk-approval"],
            ),
            ReviewerChecklistItem(
                item_id="chk-queue-role-density",
                surface_id="wf-queue",
                prompt="Validate whether VP Eng should get a summary-only queue variant instead of operator controls.",
                owner="product-experience",
                status="open",
                evidence_links=["wf-queue", "oq-role-density"],
            ),
            ReviewerChecklistItem(
                item_id="chk-run-replay-context",
                surface_id="wf-run-detail",
                prompt="Ensure replay, compare, and escalation states remain distinguishable without narration.",
                owner="Eng Lead",
                status="ready",
                evidence_links=["wf-run-detail", "flow-run-replay"],
            ),
            ReviewerChecklistItem(
                item_id="chk-run-audit-density",
                surface_id="wf-run-detail",
                prompt="Confirm the audit rail retains enough ownership history before collapsing older entries.",
                owner="orchestration-office",
                status="open",
                evidence_links=["wf-run-detail", "oq-handoff-evidence"],
            ),
            ReviewerChecklistItem(
                item_id="chk-triage-handoff-clarity",
                surface_id="wf-triage",
                prompt="Check that cross-team handoff consequences are explicit before ownership changes commit.",
                owner="Cross-Team Operator",
                status="ready",
                evidence_links=["wf-triage", "flow-triage-handoff"],
            ),
            ReviewerChecklistItem(
                item_id="chk-triage-bulk-assign",
                surface_id="wf-triage",
                prompt="Validate bulk assignment visibility without burying the audit context.",
                owner="Platform Admin",
                status="ready",
                evidence_links=["wf-triage", "flow-triage-handoff"],
            ),
        ],
        decision_log=[
            ReviewDecision(
                decision_id="dec-overview-alert-stack",
                surface_id="wf-overview",
                owner="product-experience",
                summary="Keep approval and regression alerts in one stacked priority rail.",
                rationale="Reviewers need one comparison lane before jumping into queue or triage surfaces.",
                status="accepted",
            ),
            ReviewDecision(
                decision_id="dec-queue-vp-summary",
                surface_id="wf-queue",
                owner="VP Eng",
                summary="Route VP Eng to a summary-first queue state instead of operator controls.",
                rationale="The VP Eng persona needs queue visibility without accidental action affordances.",
                status="proposed",
                follow_up="Resolve after the next design critique with policy owners.",
            ),
            ReviewDecision(
                decision_id="dec-run-detail-audit-rail",
                surface_id="wf-run-detail",
                owner="Eng Lead",
                summary="Keep audit evidence visible beside replay controls in all replay states.",
                rationale="Replay decisions are inseparable from audit context and escalation evidence.",
                status="accepted",
            ),
            ReviewDecision(
                decision_id="dec-triage-handoff-density",
                surface_id="wf-triage",
                owner="Cross-Team Operator",
                summary="Preserve ownership history in the triage rail until handoff is acknowledged.",
                rationale="Operators need a stable handoff trail before collapsing older events.",
                status="accepted",
            ),
        ],
        role_matrix=[
            ReviewRoleAssignment(
                assignment_id="role-overview-vp-eng",
                surface_id="wf-overview",
                role="VP Eng",
                responsibilities=["approve overview scan path", "validate KPI-to-drilldown narrative"],
                checklist_item_ids=["chk-overview-kpi-scan"],
                decision_ids=["dec-overview-alert-stack"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-overview-eng-ops",
                surface_id="wf-overview",
                role="engineering-operations",
                responsibilities=["review alert priority posture"],
                checklist_item_ids=["chk-overview-alert-hierarchy"],
                decision_ids=["dec-overview-alert-stack"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-queue-platform-admin",
                surface_id="wf-queue",
                role="Platform Admin",
                responsibilities=["validate batch-approval copy", "confirm permission posture"],
                checklist_item_ids=["chk-queue-batch-approval"],
                decision_ids=["dec-queue-vp-summary"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-queue-product-experience",
                surface_id="wf-queue",
                role="product-experience",
                responsibilities=["tune summary-only VP variant"],
                checklist_item_ids=["chk-queue-role-density"],
                decision_ids=["dec-queue-vp-summary"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-run-detail-eng-lead",
                surface_id="wf-run-detail",
                role="Eng Lead",
                responsibilities=["approve replay-state clarity", "confirm escalation adjacency"],
                checklist_item_ids=["chk-run-replay-context"],
                decision_ids=["dec-run-detail-audit-rail"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-run-detail-orchestration-office",
                surface_id="wf-run-detail",
                role="orchestration-office",
                responsibilities=["review audit density threshold"],
                checklist_item_ids=["chk-run-audit-density"],
                decision_ids=["dec-run-detail-audit-rail"],
                status="open",
            ),
            ReviewRoleAssignment(
                assignment_id="role-triage-cross-team-operator",
                surface_id="wf-triage",
                role="Cross-Team Operator",
                responsibilities=["approve handoff clarity", "validate ownership transition story"],
                checklist_item_ids=["chk-triage-handoff-clarity"],
                decision_ids=["dec-triage-handoff-density"],
                status="ready",
            ),
            ReviewRoleAssignment(
                assignment_id="role-triage-platform-admin",
                surface_id="wf-triage",
                role="Platform Admin",
                responsibilities=["confirm bulk-assign audit visibility"],
                checklist_item_ids=["chk-triage-bulk-assign"],
                decision_ids=["dec-triage-handoff-density"],
                status="ready",
            ),
        ],
        signoff_log=[
            ReviewSignoff(
                signoff_id="sig-overview-vp-eng",
                assignment_id="role-overview-vp-eng",
                surface_id="wf-overview",
                role="VP Eng",
                status="approved",
                evidence_links=["chk-overview-kpi-scan", "dec-overview-alert-stack"],
                notes="Overview narrative approved for design sprint review.",
                requested_at="2026-03-10T09:00:00Z",
                due_at="2026-03-12T18:00:00Z",
                escalation_owner="design-program-manager",
                sla_status="met",
            ),
            ReviewSignoff(
                signoff_id="sig-queue-platform-admin",
                assignment_id="role-queue-platform-admin",
                surface_id="wf-queue",
                role="Platform Admin",
                status="approved",
                evidence_links=["chk-queue-batch-approval", "dec-queue-vp-summary"],
                notes="Queue control actions meet operator review criteria.",
                requested_at="2026-03-10T11:00:00Z",
                due_at="2026-03-13T18:00:00Z",
                escalation_owner="platform-ops-manager",
                sla_status="met",
            ),
            ReviewSignoff(
                signoff_id="sig-run-detail-eng-lead",
                assignment_id="role-run-detail-eng-lead",
                surface_id="wf-run-detail",
                role="Eng Lead",
                status="pending",
                evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],
                notes="Waiting for final replay-state copy review.",
                requested_at="2026-03-12T11:00:00Z",
                due_at="2026-03-15T18:00:00Z",
                escalation_owner="engineering-director",
                sla_status="at-risk",
                reminder_owner="design-program-manager",
                reminder_channel="slack",
                last_reminder_at="2026-03-14T09:45:00Z",
                next_reminder_at="2026-03-15T10:00:00Z",
                reminder_cadence="daily",
                reminder_status="scheduled",
            ),
            ReviewSignoff(
                signoff_id="sig-triage-cross-team-operator",
                assignment_id="role-triage-cross-team-operator",
                surface_id="wf-triage",
                role="Cross-Team Operator",
                status="approved",
                evidence_links=["chk-triage-handoff-clarity", "dec-triage-handoff-density"],
                notes="Cross-team handoff flow approved for prototype review.",
                requested_at="2026-03-11T14:00:00Z",
                due_at="2026-03-13T12:00:00Z",
                escalation_owner="cross-team-program-manager",
                sla_status="met",
            ),
        ],
        blocker_log=[
            ReviewBlocker(
                blocker_id="blk-run-detail-copy-final",
                surface_id="wf-run-detail",
                signoff_id="sig-run-detail-eng-lead",
                owner="product-experience",
                summary="Replay-state copy still needs final wording review before Eng Lead signoff can close.",
                status="open",
                severity="medium",
                escalation_owner="design-program-manager",
                next_action="Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",
                freeze_exception=True,
                freeze_owner="release-director",
                freeze_until="2026-03-18T18:00:00Z",
                freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
                freeze_approved_by="release-director",
                freeze_approved_at="2026-03-14T08:30:00Z",
                freeze_renewal_owner="release-director",
                freeze_renewal_by="2026-03-17T12:00:00Z",
                freeze_renewal_status="review-needed",
            ),
        ],
        blocker_timeline=[
            ReviewBlockerEvent(
                event_id="evt-run-detail-copy-opened",
                blocker_id="blk-run-detail-copy-final",
                actor="product-experience",
                status="opened",
                summary="Captured the final replay-state copy gap during design sprint prep.",
                timestamp="2026-03-13T10:00:00Z",
                next_action="Draft updated replay labels before the Eng Lead review.",
            ),
            ReviewBlockerEvent(
                event_id="evt-run-detail-copy-escalated",
                blocker_id="blk-run-detail-copy-final",
                actor="design-program-manager",
                status="escalated",
                summary="Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
                timestamp="2026-03-14T09:30:00Z",
                next_action="Refresh the run-detail frame annotations after the wording review completes.",
                handoff_from="product-experience",
                handoff_to="Eng Lead",
                channel="design-critique",
                artifact_ref="wf-run-detail#copy-v5",
                ack_owner="Eng Lead",
                ack_at="2026-03-14T10:15:00Z",
                ack_status="acknowledged",
            ),
        ],
    )




def render_ui_review_decision_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Decision Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Decisions: {len(pack.decision_log)}",
        "",
        "## Decisions",
    ]
    for decision in pack.decision_log:
        lines.append(
            "- "
            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"
        )
        lines.append(
            "  "
            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or 'none'}"
        )
    if not pack.decision_log:
        lines.append("- none")
    return "\n".join(lines)



def render_ui_review_role_matrix(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Role Matrix",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Assignments: {len(pack.role_matrix)}",
        "",
        "## Assignments",
    ]
    for assignment in pack.role_matrix:
        lines.append(
            "- "
            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"
        )
        lines.append(
            "  "
            f"responsibilities={','.join(assignment.responsibilities) or 'none'} "
            f"checklist={','.join(assignment.checklist_item_ids) or 'none'} "
            f"decisions={','.join(assignment.decision_ids) or 'none'}"
        )
    if not pack.role_matrix:
        lines.append("- none")
    return "\n".join(lines)



def render_ui_review_objective_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_objective_coverage_entries(pack)
    persona_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        persona_counts[entry['persona']] = persona_counts.get(entry['persona'], 0) + 1
        status_counts[entry['coverage_status']] = status_counts.get(entry['coverage_status'], 0) + 1
    lines = [
        "# UI Review Objective Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Objectives: {len(entries)}",
        f"- Personas: {len(persona_counts)}",
        "",
        "## By Coverage Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Persona")
    for persona, count in sorted(persona_counts.items()):
        lines.append(f"- {persona}: {count}")
    if not persona_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: objective={entry['objective_id']} persona={entry['persona']} priority={entry['priority']} coverage={entry['coverage_status']} dependencies={entry['dependency_count']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  dependency_ids={entry['dependency_ids']} assignments={entry['assignment_ids']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_wireframe_readiness_board(pack: UIReviewPack) -> str:
    entries = _build_wireframe_readiness_entries(pack)
    readiness_counts: Dict[str, int] = {}
    device_counts: Dict[str, int] = {}
    for entry in entries:
        readiness_counts[entry['readiness_status']] = readiness_counts.get(entry['readiness_status'], 0) + 1
        device_counts[entry['device']] = device_counts.get(entry['device'], 0) + 1
    lines = [
        "# UI Review Wireframe Readiness Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Wireframes: {len(entries)}",
        f"- Devices: {len(device_counts)}",
        "",
        "## By Readiness",
    ]
    for status, count in sorted(readiness_counts.items()):
        lines.append(f"- {status}: {count}")
    if not readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Device")
    for device, count in sorted(device_counts.items()):
        lines.append(f"- {device}: {count}")
    if not device_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} device={entry['device']} readiness={entry['readiness_status']} open_total={entry['open_total']} entry={entry['entry_point']}"
        )
        lines.append(
            f"  checklist_open={entry['checklist_open']} decisions_open={entry['decisions_open']} assignments_open={entry['assignments_open']} signoffs_open={entry['signoffs_open']} blockers_open={entry['blockers_open']} signoffs={entry['signoff_ids']} blockers={entry['blocker_ids']} blocks={entry['block_count']} notes={entry['note_count']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_open_question_tracker(pack: UIReviewPack) -> str:
    entries = _build_open_question_tracker_entries(pack)
    owner_counts: Dict[str, int] = {}
    theme_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        theme_counts[entry['theme']] = theme_counts.get(entry['theme'], 0) + 1
    lines = [
        "# UI Review Open Question Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Questions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Theme")
    for theme, count in sorted(theme_counts.items()):
        lines.append(f"- {theme}: {count}")
    if not theme_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: question={entry['question_id']} owner={entry['owner']} theme={entry['theme']} status={entry['status']} link_status={entry['link_status']} surfaces={entry['surface_ids']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} flows={entry['flow_ids']} impact={entry['impact']} prompt={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_review_summary_board(pack: UIReviewPack) -> str:
    entries = _build_review_summary_entries(pack)
    lines = [
        "# UI Review Review Summary Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Categories: {len(entries)}",
        "",
        "## Entries",
    ]
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: category={entry['category']} total={entry['total']} {entry['metrics']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_persona_readiness_board(pack: UIReviewPack) -> str:
    entries = _build_persona_readiness_entries(pack)
    readiness_counts: Dict[str, int] = {}
    for entry in entries:
        readiness_counts[entry['readiness']] = readiness_counts.get(entry['readiness'], 0) + 1
    lines = [
        "# UI Review Persona Readiness Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Personas: {len(entries)}",
        f"- Objectives: {len(pack.objectives)}",
        "",
        "## By Readiness",
    ]
    for readiness, count in sorted(readiness_counts.items()):
        lines.append(f"- {readiness}: {count}")
    if not readiness_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: persona={entry['persona']} readiness={entry['readiness']} objectives={entry['objective_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} open_questions={entry['question_count']} queue_items={entry['queue_count']} blockers={entry['blocker_count']}"
        )
        lines.append(
            f"  objective_ids={entry['objective_ids']} surfaces={entry['surface_ids']} queue_ids={entry['queue_ids']} blocker_ids={entry['blocker_ids']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_interaction_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_interaction_coverage_entries(pack)
    coverage_counts: Dict[str, int] = {}
    surface_counts: Dict[str, int] = {}
    for entry in entries:
        coverage_counts[entry['coverage_status']] = coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                surface_counts[surface_id] = surface_counts.get(surface_id, 0) + 1
    lines = [
        "# UI Review Interaction Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Interactions: {len(entries)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Coverage Status",
    ]
    for status, count in sorted(coverage_counts.items()):
        lines.append(f"- {status}: {count}")
    if not coverage_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, count in sorted(surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: flow={entry['flow_id']} surfaces={entry['surface_ids']} owners={entry['owners']} coverage={entry['coverage_status']} states={entry['state_count']} exceptions={entry['exception_count']}"
        )
        lines.append(
            f"  checklist={entry['checklist_ids']} open_checklist={entry['open_checklist_ids']} trigger={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_checklist_traceability_board(pack: UIReviewPack) -> str:
    entries = _build_checklist_traceability_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Checklist Traceability Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Checklist items: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: item={entry['item_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_decisions={entry['linked_decisions']} evidence={entry['evidence']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_decision_followup_tracker(pack: UIReviewPack) -> str:
    entries = _build_decision_followup_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['owner']] = owner_counts.get(entry['owner'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Decision Follow-up Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Decisions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: decision={entry['decision_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} linked_roles={entry['linked_roles']}"
        )
        lines.append(
            f"  linked_assignments={entry['linked_assignments']} linked_checklists={entry['linked_checklists']} follow_up={entry['follow_up']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_role_coverage_board(pack: UIReviewPack) -> str:
    entries = _build_role_coverage_entries(pack)
    surface_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        surface_counts[entry['surface_id']] = surface_counts.get(entry['surface_id'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Role Coverage Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Assignments: {len(entries)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Surface",
    ]
    for surface_id, count in sorted(surface_counts.items()):
        lines.append(f"- {surface_id}: {count}")
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: assignment={entry['assignment_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} responsibilities={entry['responsibility_count']} checklist={entry['checklist_count']} decisions={entry['decision_count']}"
        )
        lines.append(
            f"  signoff={entry['signoff_id']} signoff_status={entry['signoff_status']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_workload_board(pack: UIReviewPack) -> str:
    entries = _build_owner_workload_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "reminder": 0, "renewal": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Workload Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Items: {len(entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} reminders={counts['reminder']} renewals={counts['renewal']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} lane={entry['lane']}"
        )
        lines.append(f"  detail={entry['detail']} summary={entry['summary']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_dependency_board(pack: UIReviewPack) -> str:
    entries = _build_signoff_dependency_entries(pack)
    dependency_counts: Dict[str, int] = {}
    sla_counts: Dict[str, int] = {}
    for entry in entries:
        dependency_counts[entry['dependency_status']] = dependency_counts.get(entry['dependency_status'], 0) + 1
        sla_counts[entry['sla_status']] = sla_counts.get(entry['sla_status'], 0) + 1
    lines = [
        "# UI Review Signoff Dependency Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(entries)}",
        f"- Dependency states: {len(dependency_counts)}",
        "",
        "## By Dependency Status",
    ]
    for status, count in sorted(dependency_counts.items()):
        lines.append(f"- {status}: {count}")
    if not dependency_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By SLA State")
    for status, count in sorted(sla_counts.items()):
        lines.append(f"- {status}: {count}")
    if not sla_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} surface={entry['surface_id']} role={entry['role']} status={entry['status']} dependency_status={entry['dependency_status']} blockers={entry['blocker_ids']}"
        )
        lines.append(
            f"  assignment={entry['assignment_id']} checklist={entry['checklist_ids']} decisions={entry['decision_ids']} latest_blocker_event={entry['latest_blocker_event']} sla={entry['sla_status']} due_at={entry['due_at']} cadence={entry['reminder_cadence']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_audit_density_board(pack: UIReviewPack) -> str:
    entries = _build_audit_density_entries(pack)
    band_counts: Dict[str, int] = {}
    for entry in entries:
        band_counts[entry['load_band']] = band_counts.get(entry['load_band'], 0) + 1
    lines = [
        "# UI Review Audit Density Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Surfaces: {len(entries)}",
        f"- Load bands: {len(band_counts)}",
        "",
        "## By Load Band",
    ]
    for band, count in sorted(band_counts.items()):
        lines.append(f"- {band}: {count}")
    if not band_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: surface={entry['surface_id']} artifact_total={entry['artifact_total']} open_total={entry['open_total']} band={entry['load_band']}"
        )
        lines.append(
            f"  checklist={entry['checklist_count']} decisions={entry['decision_count']} assignments={entry['assignment_count']} signoffs={entry['signoff_count']} blockers={entry['blocker_count']} timeline={entry['timeline_count']} blocks={entry['block_count']} notes={entry['note_count']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Sign-off Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(pack.signoff_log)}",
        "",
        "## Sign-offs",
    ]
    for signoff in pack.signoff_log:
        lines.append(
            "- "
            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"
        )
        lines.append(
            "  "
            f"required={'yes' if signoff.required else 'no'} evidence={','.join(signoff.evidence_links) or 'none'} notes={signoff.notes or 'none'} waiver_owner={signoff.waiver_owner or 'none'} waiver_reason={signoff.waiver_reason or 'none'} requested_at={signoff.requested_at or 'none'} due_at={signoff.due_at or 'none'} escalation_owner={signoff.escalation_owner or 'none'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or 'none'} reminder_channel={signoff.reminder_channel or 'none'} last_reminder_at={signoff.last_reminder_at or 'none'} next_reminder_at={signoff.next_reminder_at or 'none'}"
        )
    if not pack.signoff_log:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_sla_dashboard(pack: UIReviewPack) -> str:
    entries = _build_signoff_sla_entries(pack)
    state_counts: Dict[str, int] = {}
    owner_counts: Dict[str, int] = {}
    for entry in entries:
        state_counts[entry['sla_status']] = state_counts.get(entry['sla_status'], 0) + 1
        owner_counts[entry['escalation_owner']] = owner_counts.get(entry['escalation_owner'], 0) + 1
    lines = [
        "# UI Review Sign-off SLA Dashboard",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Sign-offs: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## SLA States",
    ]
    for sla_status, count in sorted(state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalation Owners")
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Sign-offs")
    for entry in entries:
        lines.append(
            f"- {entry['signoff_id']}: role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} requested_at={entry['requested_at']} due_at={entry['due_at']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(f"  required={entry['required']} evidence={entry['evidence']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_reminder_queue(pack: UIReviewPack) -> str:
    entries = _build_signoff_reminder_entries(pack)
    owner_counts: Dict[str, int] = {}
    channel_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry["reminder_owner"]] = owner_counts.get(entry["reminder_owner"], 0) + 1
        channel_counts[entry["reminder_channel"]] = channel_counts.get(entry["reminder_channel"], 0) + 1
    lines = [
        "# UI Review Sign-off Reminder Queue",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Reminders: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: reminders={count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Channel")
    for channel, count in sorted(channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} owner={entry['reminder_owner']} channel={entry['reminder_channel']}"
        )
        lines.append(
            f"  last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_reminder_cadence_board(pack: UIReviewPack) -> str:
    entries = _build_reminder_cadence_entries(pack)
    cadence_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        cadence_counts[entry["reminder_cadence"]] = cadence_counts.get(entry["reminder_cadence"], 0) + 1
        status_counts[entry["reminder_status"]] = status_counts.get(entry["reminder_status"], 0) + 1
    lines = [
        "# UI Review Reminder Cadence Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Items: {len(entries)}",
        f"- Cadences: {len(cadence_counts)}",
        "",
        "## By Cadence",
    ]
    for cadence, count in sorted(cadence_counts.items()):
        lines.append(f"- {cadence}: {count}")
    if not cadence_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} cadence={entry['reminder_cadence']} status={entry['reminder_status']} owner={entry['reminder_owner']}"
        )
        lines.append(
            f"  sla={entry['sla_status']} last_reminder_at={entry['last_reminder_at']} next_reminder_at={entry['next_reminder_at']} due_at={entry['due_at']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_escalation_dashboard(pack: UIReviewPack) -> str:
    entries = _build_escalation_dashboard_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    status_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        owner_bucket = owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        status_bucket = status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_bucket[entry['item_type']] += 1
        status_bucket['total'] += 1
    lines = [
        "# UI Review Escalation Dashboard",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Items: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## By Escalation Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, counts in sorted(status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalations")
    for entry in entries:
        lines.append(
            f"- {entry['escalation_id']}: owner={entry['escalation_owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} priority={entry['priority']} current_owner={entry['current_owner']}"
        )
        lines.append(f"  summary={entry['summary']} due_at={entry['due_at']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_signoff_breach_board(pack: UIReviewPack) -> str:
    entries = _build_signoff_breach_entries(pack)
    state_counts: Dict[str, int] = {}
    owner_counts: Dict[str, int] = {}
    for entry in entries:
        state_counts[entry['sla_status']] = state_counts.get(entry['sla_status'], 0) + 1
        owner_counts[entry['escalation_owner']] = owner_counts.get(entry['escalation_owner'], 0) + 1
    lines = [
        "# UI Review Sign-off Breach Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Breach items: {len(entries)}",
        f"- Escalation owners: {len(owner_counts)}",
        "",
        "## SLA States",
    ]
    for sla_status, count in sorted(state_counts.items()):
        lines.append(f"- {sla_status}: {count}")
    if not state_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Escalation Owners")
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: signoff={entry['signoff_id']} role={entry['role']} surface={entry['surface_id']} status={entry['status']} sla={entry['sla_status']} escalation_owner={entry['escalation_owner']}"
        )
        lines.append(
            f"  requested_at={entry['requested_at']} due_at={entry['due_at']} linked_blockers={entry['linked_blockers']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_escalation_handoff_ledger(pack: UIReviewPack) -> str:
    entries = _build_escalation_handoff_entries(pack)
    channel_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        channel_counts[entry['channel']] = channel_counts.get(entry['channel'], 0) + 1
        status_counts[entry['status']] = status_counts.get(entry['status'], 0) + 1
    lines = [
        "# UI Review Escalation Handoff Ledger",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Handoffs: {len(entries)}",
        f"- Channels: {len(channel_counts)}",
        "",
        "## By Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Channel")
    for channel, count in sorted(channel_counts.items()):
        lines.append(f"- {channel}: {count}")
    if not channel_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['ledger_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} actor={entry['actor']} status={entry['status']} at={entry['timestamp']}"
        )
        lines.append(
            f"  from={entry['handoff_from']} to={entry['handoff_to']} channel={entry['channel']} artifact={entry['artifact_ref']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_handoff_ack_ledger(pack: UIReviewPack) -> str:
    entries = _build_handoff_ack_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['ack_owner']] = owner_counts.get(entry['ack_owner'], 0) + 1
        status_counts[entry['ack_status']] = status_counts.get(entry['ack_status'], 0) + 1
    lines = [
        "# UI Review Handoff Ack Ledger",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Ack items: {len(entries)}",
        f"- Ack owners: {len(owner_counts)}",
        "",
        "## By Ack Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Ack Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: event={entry['event_id']} blocker={entry['blocker_id']} surface={entry['surface_id']} handoff_to={entry['handoff_to']} ack_owner={entry['ack_owner']} ack_status={entry['ack_status']} ack_at={entry['ack_at']}"
        )
        lines.append(
            f"  actor={entry['actor']} status={entry['status']} channel={entry['channel']} artifact={entry['artifact_ref']} summary={entry['summary']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_approval_trail(pack: UIReviewPack) -> str:
    entries = _build_freeze_approval_entries(pack)
    approver_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        approver_counts[entry["freeze_approved_by"]] = approver_counts.get(entry["freeze_approved_by"], 0) + 1
        status_counts[entry["status"]] = status_counts.get(entry["status"], 0) + 1
    lines = [
        "# UI Review Freeze Approval Trail",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Approvals: {len(entries)}",
        f"- Approvers: {len(approver_counts)}",
        "",
        "## By Approver",
    ]
    for owner, count in sorted(approver_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not approver_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} owner={entry['freeze_owner']} approved_by={entry['freeze_approved_by']} approved_at={entry['freeze_approved_at']} window={entry['freeze_until']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_renewal_tracker(pack: UIReviewPack) -> str:
    entries = _build_freeze_renewal_entries(pack)
    owner_counts: Dict[str, int] = {}
    status_counts: Dict[str, int] = {}
    for entry in entries:
        owner_counts[entry['renewal_owner']] = owner_counts.get(entry['renewal_owner'], 0) + 1
        status_counts[entry['renewal_status']] = status_counts.get(entry['renewal_status'], 0) + 1
    lines = [
        "# UI Review Freeze Renewal Tracker",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Renewal items: {len(entries)}",
        f"- Renewal owners: {len(owner_counts)}",
        "",
        "## By Renewal Owner",
    ]
    for owner, count in sorted(owner_counts.items()):
        lines.append(f"- {owner}: {count}")
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Renewal Status")
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: blocker={entry['blocker_id']} surface={entry['surface_id']} status={entry['status']} renewal_owner={entry['renewal_owner']} renewal_by={entry['renewal_by']} renewal_status={entry['renewal_status']}"
        )
        lines.append(
            f"  freeze_owner={entry['freeze_owner']} freeze_until={entry['freeze_until']} approved_by={entry['freeze_approved_by']} summary={entry['summary']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_freeze_exception_board(pack: UIReviewPack) -> str:
    entries = _build_freeze_exception_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        owner_bucket = owner_counts.setdefault(entry['owner'], {'blocker': 0, 'signoff': 0, 'total': 0})
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        surface_bucket = surface_counts.setdefault(entry['surface_id'], {'blocker': 0, 'signoff': 0, 'total': 0})
        surface_bucket[entry['item_type']] += 1
        surface_bucket['total'] += 1
    lines = [
        "# UI Review Freeze Exception Board",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(entries)}",
        f"- Owners: {len(owner_counts)}",
        "",
        "## By Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, counts in sorted(surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in entries:
        lines.append(
            f"- {entry['entry_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} window={entry['window']}"
        )
        lines.append(
            f"  summary={entry['summary']} evidence={entry['evidence']} next_action={entry['next_action']}"
        )
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_escalation_digest(pack: UIReviewPack) -> str:
    entries = _build_owner_escalation_digest_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "signoff": 0, "reminder": 0, "freeze": 0, "handoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Escalation Digest",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Items: {len(entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} reminders={counts['reminder']} freezes={counts['freeze']} handoffs={counts['handoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in entries:
        lines.append(
            f"- {entry['digest_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} detail={entry['detail']}")
    if not entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_log(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Blocker Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Blockers: {len(pack.blocker_log)}",
        "",
        "## Blockers",
    ]
    for blocker in pack.blocker_log:
        lines.append(
            "- "
            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"
        )
        lines.append(
            "  "
            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or 'none'} next_action={blocker.next_action or 'none'} freeze_owner={blocker.freeze_owner or 'none'} freeze_until={blocker.freeze_until or 'none'} freeze_approved_by={blocker.freeze_approved_by or 'none'} freeze_approved_at={blocker.freeze_approved_at or 'none'}"
        )
    if not pack.blocker_log:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_timeline(pack: UIReviewPack) -> str:
    lines = [
        "# UI Review Blocker Timeline",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Events: {len(pack.blocker_timeline)}",
        "",
        "## Events",
    ]
    for event in pack.blocker_timeline:
        lines.append(
            "- "
            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"
        )
        lines.append(
            "  "
            f"summary={event.summary} next_action={event.next_action or 'none'}"
        )
    if not pack.blocker_timeline:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_exception_log(pack: UIReviewPack) -> str:
    exception_entries = _build_review_exception_entries(pack)
    lines = [
        "# UI Review Exception Log",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(exception_entries)}",
        "",
        "## Exceptions",
    ]
    for entry in exception_entries:
        lines.append(
            "- "
            f"{entry['exception_id']}: type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} owner={entry['owner']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            "  "
            f"summary={entry['summary']} evidence={entry['evidence']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_exception_matrix(pack: UIReviewPack) -> str:
    exception_entries = _build_review_exception_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    status_counts: Dict[str, Dict[str, int]] = {}
    surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_bucket = owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["category"]] += 1
        owner_bucket["total"] += 1
        status_bucket = status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_bucket[entry["category"]] += 1
        status_bucket["total"] += 1
        surface_bucket = surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["category"]] += 1
        surface_bucket["total"] += 1
    lines = [
        "# UI Review Exception Matrix",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Exceptions: {len(exception_entries)}",
        f"- Owners: {len(owner_counts)}",
        f"- Surfaces: {len(surface_counts)}",
        "",
        "## By Owner",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Status")
    for status, counts in sorted(status_counts.items()):
        lines.append(
            f"- {status}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## By Surface")
    for surface_id, counts in sorted(surface_counts.items()):
        lines.append(
            f"- {surface_id}: blockers={counts['blocker']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not surface_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Entries")
    for entry in exception_entries:
        lines.append(
            f"- {entry['exception_id']}: owner={entry['owner']} type={entry['category']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']} severity={entry['severity']}"
        )
        lines.append(
            f"  summary={entry['summary']} latest_event={entry['latest_event']} next_action={entry['next_action']}"
        )
    if not exception_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_owner_review_queue(pack: UIReviewPack) -> str:
    queue_entries = _build_owner_review_queue_entries(pack)
    owner_counts: Dict[str, Dict[str, int]] = {}
    for entry in queue_entries:
        counts = owner_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    lines = [
        "# UI Review Owner Review Queue",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Owners: {len(owner_counts)}",
        f"- Queue items: {len(queue_entries)}",
        "",
        "## Owners",
    ]
    for owner, counts in sorted(owner_counts.items()):
        lines.append(
            f"- {owner}: blockers={counts['blocker']} checklist={counts['checklist']} decisions={counts['decision']} signoffs={counts['signoff']} total={counts['total']}"
        )
    if not owner_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Items")
    for entry in queue_entries:
        lines.append(
            f"- {entry['queue_id']}: owner={entry['owner']} type={entry['item_type']} source={entry['source_id']} surface={entry['surface_id']} status={entry['status']}"
        )
        lines.append(f"  summary={entry['summary']} next_action={entry['next_action']}")
    if not queue_entries:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_blocker_timeline_summary(pack: UIReviewPack) -> str:
    timeline_index = _build_blocker_timeline_index(pack)
    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )
    lines = [
        "# UI Review Blocker Timeline Summary",
        "",
        f"- Issue: {pack.issue_id} {pack.title}",
        f"- Version: {pack.version}",
        f"- Events: {len(pack.blocker_timeline)}",
        f"- Blockers with timeline: {len(timeline_index)}",
        f"- Orphan timeline blockers: {','.join(orphan_timeline_ids) or 'none'}",
        "",
        "## Events by Status",
    ]
    for status, count in sorted(status_counts.items()):
        lines.append(f"- {status}: {count}")
    if not status_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Events by Actor")
    for actor, count in sorted(actor_counts.items()):
        lines.append(f"- {actor}: {count}")
    if not actor_counts:
        lines.append("- none")
    lines.append("")
    lines.append("## Latest Blocker Events")
    for blocker in pack.blocker_log:
        latest_events = timeline_index.get(blocker.blocker_id, [])
        latest = latest_events[-1] if latest_events else None
        if latest is None:
            lines.append(f"- {blocker.blocker_id}: latest=none")
            continue
        lines.append(
            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"
        )
    if not pack.blocker_log:
        lines.append("- none")
    return "\n".join(lines)


def render_ui_review_pack_html(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:
    objective_html = "".join(
        f"<li><strong>{escape(objective.objective_id)}</strong> · {escape(objective.title)} · persona={escape(objective.persona)} · priority={escape(objective.priority)}<br /><span>{escape(objective.success_signal)}</span></li>"
        for objective in pack.objectives
    ) or "<li>none</li>"
    wireframe_html = "".join(
        f"<li><strong>{escape(wireframe.surface_id)}</strong> · {escape(wireframe.name)} · entry={escape(wireframe.entry_point)}<br /><span>blocks={escape(', '.join(wireframe.primary_blocks) if wireframe.primary_blocks else 'none')}</span></li>"
        for wireframe in pack.wireframes
    ) or "<li>none</li>"
    interaction_html = "".join(
        f"<li><strong>{escape(interaction.flow_id)}</strong> · {escape(interaction.name)}<br /><span>states={escape(', '.join(interaction.states) if interaction.states else 'none')}</span></li>"
        for interaction in pack.interactions
    ) or "<li>none</li>"
    interaction_coverage_entries = _build_interaction_coverage_entries(pack)
    interaction_coverage_counts: Dict[str, int] = {}
    interaction_surface_counts: Dict[str, int] = {}
    for entry in interaction_coverage_entries:
        interaction_coverage_counts[entry['coverage_status']] = interaction_coverage_counts.get(entry['coverage_status'], 0) + 1
        for surface_id in entry['surface_ids'].split(','):
            if surface_id and surface_id != 'none':
                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1
    interaction_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(interaction_coverage_counts.items())
    ) or "<li>none</li>"
    interaction_coverage_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"
        for surface_id, count in sorted(interaction_surface_counts.items())
    ) or "<li>none</li>"
    interaction_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · flow={escape(entry['flow_id'])} · surfaces={escape(entry['surface_ids'])} · owners={escape(entry['owners'])} · coverage={escape(entry['coverage_status'])}<br /><span>states={escape(entry['state_count'])} · exceptions={escape(entry['exception_count'])}</span><br /><span>checklist={escape(entry['checklist_ids'])} · open_checklist={escape(entry['open_checklist_ids'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in interaction_coverage_entries
    ) or "<li>none</li>"
    question_html = "".join(
        f"<li><strong>{escape(question.question_id)}</strong> · {escape(question.theme)} · owner={escape(question.owner)} · status={escape(question.status)}<br /><span>{escape(question.question)}</span></li>"
        for question in pack.open_questions
    ) or "<li>none</li>"
    checklist_html = "".join(
        f"<li><strong>{escape(item.item_id)}</strong> · surface={escape(item.surface_id)} · owner={escape(item.owner)} · status={escape(item.status)}<br /><span>{escape(item.prompt)}</span><br /><span>evidence={escape(', '.join(item.evidence_links) if item.evidence_links else 'none')}</span></li>"
        for item in pack.reviewer_checklist
    ) or "<li>none</li>"
    decision_html = "".join(
        f"<li><strong>{escape(decision.decision_id)}</strong> · surface={escape(decision.surface_id)} · owner={escape(decision.owner)} · status={escape(decision.status)}<br /><span>{escape(decision.summary)}</span><br /><span>follow_up={escape(decision.follow_up or 'none')}</span></li>"
        for decision in pack.decision_log
    ) or "<li>none</li>"
    role_matrix_html = "".join(
        f"<li><strong>{escape(assignment.assignment_id)}</strong> · surface={escape(assignment.surface_id)} · role={escape(assignment.role)} · status={escape(assignment.status)}<br /><span>responsibilities={escape(', '.join(assignment.responsibilities) if assignment.responsibilities else 'none')}</span><br /><span>checklist={escape(', '.join(assignment.checklist_item_ids) if assignment.checklist_item_ids else 'none')} · decisions={escape(', '.join(assignment.decision_ids) if assignment.decision_ids else 'none')}</span></li>"
        for assignment in pack.role_matrix
    ) or "<li>none</li>"
    objective_coverage_entries = _build_objective_coverage_entries(pack)
    objective_coverage_status_counts: Dict[str, int] = {}
    objective_coverage_persona_counts: Dict[str, int] = {}
    for entry in objective_coverage_entries:
        objective_coverage_status_counts[entry['coverage_status']] = objective_coverage_status_counts.get(entry['coverage_status'], 0) + 1
        objective_coverage_persona_counts[entry['persona']] = objective_coverage_persona_counts.get(entry['persona'], 0) + 1
    objective_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(objective_coverage_status_counts.items())
    ) or "<li>none</li>"
    objective_coverage_persona_html = "".join(
        f"<li><strong>{escape(persona)}</strong> · count={count}</li>"
        for persona, count in sorted(objective_coverage_persona_counts.items())
    ) or "<li>none</li>"
    objective_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · objective={escape(entry['objective_id'])} · persona={escape(entry['persona'])} · priority={escape(entry['priority'])} · coverage={escape(entry['coverage_status'])}<br /><span>dependencies={escape(entry['dependency_count'])} · surfaces={escape(entry['surface_ids'])} · blockers={escape(entry['blocker_ids'])}</span><br /><span>assignments={escape(entry['assignment_ids'])} · checklist={escape(entry['checklist_ids'])} · decisions={escape(entry['decision_ids'])}</span><br /><span>signoffs={escape(entry['signoff_ids'])} · dependency_ids={escape(entry['dependency_ids'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in objective_coverage_entries
    ) or "<li>none</li>"
    review_summary_entries = _build_review_summary_entries(pack)
    review_summary_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · category={escape(entry['category'])} · total={escape(entry['total'])} · {escape(entry['metrics'])}</li>"
        for entry in review_summary_entries
    ) or "<li>none</li>"
    persona_readiness_entries = _build_persona_readiness_entries(pack)
    persona_readiness_counts: Dict[str, int] = {}
    for entry in persona_readiness_entries:
        persona_readiness_counts[entry['readiness']] = persona_readiness_counts.get(entry['readiness'], 0) + 1
    persona_readiness_status_html = "".join(
        f"<li><strong>{escape(readiness)}</strong> · count={count}</li>"
        for readiness, count in sorted(persona_readiness_counts.items())
    ) or "<li>none</li>"
    persona_readiness_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · persona={escape(entry['persona'])} · readiness={escape(entry['readiness'])} · objectives={escape(entry['objective_count'])}<br /><span>assignments={escape(entry['assignment_count'])} · signoffs={escape(entry['signoff_count'])} · open_questions={escape(entry['question_count'])} · queue_items={escape(entry['queue_count'])} · blockers={escape(entry['blocker_count'])}</span><br /><span>objective_ids={escape(entry['objective_ids'])} · surfaces={escape(entry['surface_ids'])}</span><br /><span>queue_ids={escape(entry['queue_ids'])} · blocker_ids={escape(entry['blocker_ids'])}</span></li>"
        for entry in persona_readiness_entries
    ) or "<li>none</li>"
    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)
    wireframe_readiness_counts: Dict[str, int] = {}
    wireframe_device_counts: Dict[str, int] = {}
    for entry in wireframe_readiness_entries:
        wireframe_readiness_counts[entry['readiness_status']] = wireframe_readiness_counts.get(entry['readiness_status'], 0) + 1
        wireframe_device_counts[entry['device']] = wireframe_device_counts.get(entry['device'], 0) + 1
    wireframe_readiness_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(wireframe_readiness_counts.items())
    ) or "<li>none</li>"
    wireframe_device_html = "".join(
        f"<li><strong>{escape(device)}</strong> · count={count}</li>"
        for device, count in sorted(wireframe_device_counts.items())
    ) or "<li>none</li>"
    wireframe_readiness_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · surface={escape(entry['surface_id'])} · device={escape(entry['device'])} · readiness={escape(entry['readiness_status'])} · open_total={escape(entry['open_total'])}<br /><span>entry={escape(entry['entry_point'])} · signoffs={escape(entry['signoff_ids'])} · blockers={escape(entry['blocker_ids'])}</span><br /><span>checklist_open={escape(entry['checklist_open'])} · decisions_open={escape(entry['decisions_open'])} · assignments_open={escape(entry['assignments_open'])}</span><br /><span>signoffs_open={escape(entry['signoffs_open'])} · blockers_open={escape(entry['blockers_open'])} · blocks={escape(entry['block_count'])} · notes={escape(entry['note_count'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in wireframe_readiness_entries
    ) or "<li>none</li>"
    open_question_entries = _build_open_question_tracker_entries(pack)
    open_question_owner_counts: Dict[str, int] = {}
    open_question_theme_counts: Dict[str, int] = {}
    for entry in open_question_entries:
        open_question_owner_counts[entry['owner']] = open_question_owner_counts.get(entry['owner'], 0) + 1
        open_question_theme_counts[entry['theme']] = open_question_theme_counts.get(entry['theme'], 0) + 1
    open_question_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(open_question_owner_counts.items())
    ) or "<li>none</li>"
    open_question_theme_html = "".join(
        f"<li><strong>{escape(theme)}</strong> · count={count}</li>"
        for theme, count in sorted(open_question_theme_counts.items())
    ) or "<li>none</li>"
    open_question_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · question={escape(entry['question_id'])} · owner={escape(entry['owner'])} · theme={escape(entry['theme'])} · status={escape(entry['status'])}<br /><span>link_status={escape(entry['link_status'])} · surfaces={escape(entry['surface_ids'])} · checklist={escape(entry['checklist_ids'])}</span><br /><span>flows={escape(entry['flow_ids'])}</span><br /><span>impact={escape(entry['impact'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in open_question_entries
    ) or "<li>none</li>"
    checklist_trace_entries = _build_checklist_traceability_entries(pack)
    checklist_trace_owner_counts: Dict[str, int] = {}
    checklist_trace_status_counts: Dict[str, int] = {}
    for entry in checklist_trace_entries:
        checklist_trace_owner_counts[entry['owner']] = checklist_trace_owner_counts.get(entry['owner'], 0) + 1
        checklist_trace_status_counts[entry['status']] = checklist_trace_status_counts.get(entry['status'], 0) + 1
    checklist_trace_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(checklist_trace_owner_counts.items())
    ) or "<li>none</li>"
    checklist_trace_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(checklist_trace_status_counts.items())
    ) or "<li>none</li>"
    checklist_trace_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · item={escape(entry['item_id'])} · surface={escape(entry['surface_id'])} · owner={escape(entry['owner'])} · status={escape(entry['status'])}<br /><span>linked_roles={escape(entry['linked_roles'])} · linked_assignments={escape(entry['linked_assignments'])}</span><br /><span>linked_decisions={escape(entry['linked_decisions'])} · evidence={escape(entry['evidence'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in checklist_trace_entries
    ) or "<li>none</li>"
    decision_followup_entries = _build_decision_followup_entries(pack)
    decision_followup_owner_counts: Dict[str, int] = {}
    decision_followup_status_counts: Dict[str, int] = {}
    for entry in decision_followup_entries:
        decision_followup_owner_counts[entry['owner']] = decision_followup_owner_counts.get(entry['owner'], 0) + 1
        decision_followup_status_counts[entry['status']] = decision_followup_status_counts.get(entry['status'], 0) + 1
    decision_followup_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(decision_followup_owner_counts.items())
    ) or "<li>none</li>"
    decision_followup_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(decision_followup_status_counts.items())
    ) or "<li>none</li>"
    decision_followup_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · decision={escape(entry['decision_id'])} · surface={escape(entry['surface_id'])} · owner={escape(entry['owner'])} · status={escape(entry['status'])}<br /><span>linked_roles={escape(entry['linked_roles'])} · linked_assignments={escape(entry['linked_assignments'])}</span><br /><span>linked_checklists={escape(entry['linked_checklists'])} · follow_up={escape(entry['follow_up'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in decision_followup_entries
    ) or "<li>none</li>"
    role_coverage_entries = _build_role_coverage_entries(pack)
    role_coverage_surface_counts: Dict[str, int] = {}
    role_coverage_status_counts: Dict[str, int] = {}
    for entry in role_coverage_entries:
        role_coverage_surface_counts[entry['surface_id']] = role_coverage_surface_counts.get(entry['surface_id'], 0) + 1
        role_coverage_status_counts[entry['status']] = role_coverage_status_counts.get(entry['status'], 0) + 1
    role_coverage_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"
        for surface_id, count in sorted(role_coverage_surface_counts.items())
    ) or "<li>none</li>"
    role_coverage_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(role_coverage_status_counts.items())
    ) or "<li>none</li>"
    role_coverage_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · assignment={escape(entry['assignment_id'])} · surface={escape(entry['surface_id'])} · role={escape(entry['role'])} · status={escape(entry['status'])}<br /><span>responsibilities={escape(entry['responsibility_count'])} · checklist={escape(entry['checklist_count'])} · decisions={escape(entry['decision_count'])}</span><br /><span>signoff={escape(entry['signoff_id'])} · signoff_status={escape(entry['signoff_status'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in role_coverage_entries
    ) or "<li>none</li>"
    signoff_dependency_entries = _build_signoff_dependency_entries(pack)
    signoff_dependency_status_counts: Dict[str, int] = {}
    signoff_dependency_sla_counts: Dict[str, int] = {}
    for entry in signoff_dependency_entries:
        signoff_dependency_status_counts[entry['dependency_status']] = signoff_dependency_status_counts.get(entry['dependency_status'], 0) + 1
        signoff_dependency_sla_counts[entry['sla_status']] = signoff_dependency_sla_counts.get(entry['sla_status'], 0) + 1
    signoff_dependency_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(signoff_dependency_status_counts.items())
    ) or "<li>none</li>"
    signoff_dependency_sla_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(signoff_dependency_sla_counts.items())
    ) or "<li>none</li>"
    signoff_dependency_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · surface={escape(entry['surface_id'])} · role={escape(entry['role'])} · status={escape(entry['status'])} · dependency_status={escape(entry['dependency_status'])}<br /><span>assignment={escape(entry['assignment_id'])} · checklist={escape(entry['checklist_ids'])} · decisions={escape(entry['decision_ids'])}</span><br /><span>blockers={escape(entry['blocker_ids'])} · latest_blocker_event={escape(entry['latest_blocker_event'])}</span><br /><span>sla={escape(entry['sla_status'])} · due_at={escape(entry['due_at'])} · cadence={escape(entry['reminder_cadence'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in signoff_dependency_entries
    ) or "<li>none</li>"
    signoff_html = "".join(
        f"<li><strong>{escape(signoff.signoff_id)}</strong> · surface={escape(signoff.surface_id)} · role={escape(signoff.role)} · status={escape(signoff.status)}<br /><span>assignment={escape(signoff.assignment_id)} · required={escape('yes' if signoff.required else 'no')}</span><br /><span>evidence={escape(', '.join(signoff.evidence_links) if signoff.evidence_links else 'none')}</span><br /><span>waiver_owner={escape(signoff.waiver_owner or 'none')} · waiver_reason={escape(signoff.waiver_reason or 'none')}</span><br /><span>requested_at={escape(signoff.requested_at or 'none')} · due_at={escape(signoff.due_at or 'none')} · escalation_owner={escape(signoff.escalation_owner or 'none')} · sla_status={escape(signoff.sla_status)}</span></li>"
        for signoff in pack.signoff_log
    ) or "<li>none</li>"
    signoff_sla_entries = _build_signoff_sla_entries(pack)
    signoff_sla_state_counts: Dict[str, int] = {}
    signoff_sla_owner_counts: Dict[str, int] = {}
    for entry in signoff_sla_entries:
        signoff_sla_state_counts[entry['sla_status']] = signoff_sla_state_counts.get(entry['sla_status'], 0) + 1
        signoff_sla_owner_counts[entry['escalation_owner']] = signoff_sla_owner_counts.get(entry['escalation_owner'], 0) + 1
    signoff_sla_state_html = "".join(
        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"
        for sla_status, count in sorted(signoff_sla_state_counts.items())
    ) or "<li>none</li>"
    signoff_sla_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(signoff_sla_owner_counts.items())
    ) or "<li>none</li>"
    signoff_sla_item_html = "".join(
        f"<li><strong>{escape(entry['signoff_id'])}</strong> · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>requested_at={escape(entry['requested_at'])} · due_at={escape(entry['due_at'])} · escalation_owner={escape(entry['escalation_owner'])}</span><br /><span>required={escape(entry['required'])} · evidence={escape(entry['evidence'])}</span></li>"
        for entry in signoff_sla_entries
    ) or "<li>none</li>"
    signoff_reminder_entries = _build_signoff_reminder_entries(pack)
    signoff_reminder_owner_counts: Dict[str, int] = {}
    signoff_reminder_channel_counts: Dict[str, int] = {}
    for entry in signoff_reminder_entries:
        signoff_reminder_owner_counts[entry['reminder_owner']] = signoff_reminder_owner_counts.get(entry['reminder_owner'], 0) + 1
        signoff_reminder_channel_counts[entry['reminder_channel']] = signoff_reminder_channel_counts.get(entry['reminder_channel'], 0) + 1
    signoff_reminder_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · reminders={count}</li>"
        for owner, count in sorted(signoff_reminder_owner_counts.items())
    ) or "<li>none</li>"
    signoff_reminder_channel_html = "".join(
        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"
        for channel, count in sorted(signoff_reminder_channel_counts.items())
    ) or "<li>none</li>"
    signoff_reminder_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>owner={escape(entry['reminder_owner'])} · channel={escape(entry['reminder_channel'])}</span><br /><span>last_reminder_at={escape(entry['last_reminder_at'])} · next_reminder_at={escape(entry['next_reminder_at'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in signoff_reminder_entries
    ) or "<li>none</li>"
    reminder_cadence_entries = _build_reminder_cadence_entries(pack)
    reminder_cadence_counts: Dict[str, int] = {}
    reminder_status_counts: Dict[str, int] = {}
    for entry in reminder_cadence_entries:
        reminder_cadence_counts[entry['reminder_cadence']] = reminder_cadence_counts.get(entry['reminder_cadence'], 0) + 1
        reminder_status_counts[entry['reminder_status']] = reminder_status_counts.get(entry['reminder_status'], 0) + 1
    reminder_cadence_owner_html = "".join(
        f"<li><strong>{escape(cadence)}</strong> · count={count}</li>"
        for cadence, count in sorted(reminder_cadence_counts.items())
    ) or "<li>none</li>"
    reminder_cadence_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(reminder_status_counts.items())
    ) or "<li>none</li>"
    reminder_cadence_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · cadence={escape(entry['reminder_cadence'])} · status={escape(entry['reminder_status'])}<br /><span>owner={escape(entry['reminder_owner'])} · sla={escape(entry['sla_status'])}</span><br /><span>last_reminder_at={escape(entry['last_reminder_at'])} · next_reminder_at={escape(entry['next_reminder_at'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in reminder_cadence_entries
    ) or "<li>none</li>"
    signoff_breach_entries = _build_signoff_breach_entries(pack)
    signoff_breach_state_counts: Dict[str, int] = {}
    signoff_breach_owner_counts: Dict[str, int] = {}
    for entry in signoff_breach_entries:
        signoff_breach_state_counts[entry['sla_status']] = signoff_breach_state_counts.get(entry['sla_status'], 0) + 1
        signoff_breach_owner_counts[entry['escalation_owner']] = signoff_breach_owner_counts.get(entry['escalation_owner'], 0) + 1
    signoff_breach_state_html = "".join(
        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"
        for sla_status, count in sorted(signoff_breach_state_counts.items())
    ) or "<li>none</li>"
    signoff_breach_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(signoff_breach_owner_counts.items())
    ) or "<li>none</li>"
    signoff_breach_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · signoff={escape(entry['signoff_id'])} · role={escape(entry['role'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · sla={escape(entry['sla_status'])}<br /><span>requested_at={escape(entry['requested_at'])} · due_at={escape(entry['due_at'])} · escalation_owner={escape(entry['escalation_owner'])}</span><br /><span>linked_blockers={escape(entry['linked_blockers'])} · summary={escape(entry['summary'])}</span></li>"
        for entry in signoff_breach_entries
    ) or "<li>none</li>"
    escalation_entries = _build_escalation_dashboard_entries(pack)
    escalation_owner_counts: Dict[str, Dict[str, int]] = {}
    escalation_status_counts: Dict[str, Dict[str, int]] = {}
    for entry in escalation_entries:
        owner_bucket = escalation_owner_counts.setdefault(
            entry['escalation_owner'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        owner_bucket[entry['item_type']] += 1
        owner_bucket['total'] += 1
        status_bucket = escalation_status_counts.setdefault(
            entry['status'], {'blocker': 0, 'signoff': 0, 'total': 0}
        )
        status_bucket[entry['item_type']] += 1
        status_bucket['total'] += 1
    escalation_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(escalation_owner_counts.items())
    ) or "<li>none</li>"
    escalation_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for status, counts in sorted(escalation_status_counts.items())
    ) or "<li>none</li>"
    escalation_item_html = "".join(
        f"<li><strong>{escape(entry['escalation_id'])}</strong> · owner={escape(entry['escalation_owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · priority={escape(entry['priority'])}<br /><span>current_owner={escape(entry['current_owner'])} · due_at={escape(entry['due_at'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in escalation_entries
    ) or "<li>none</li>"
    escalation_handoff_entries = _build_escalation_handoff_entries(pack)
    escalation_handoff_status_counts: Dict[str, int] = {}
    escalation_handoff_channel_counts: Dict[str, int] = {}
    for entry in escalation_handoff_entries:
        escalation_handoff_status_counts[entry['status']] = escalation_handoff_status_counts.get(entry['status'], 0) + 1
        escalation_handoff_channel_counts[entry['channel']] = escalation_handoff_channel_counts.get(entry['channel'], 0) + 1
    escalation_handoff_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(escalation_handoff_status_counts.items())
    ) or "<li>none</li>"
    escalation_handoff_channel_html = "".join(
        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"
        for channel, count in sorted(escalation_handoff_channel_counts.items())
    ) or "<li>none</li>"
    escalation_handoff_item_html = "".join(
        f"<li><strong>{escape(entry['ledger_id'])}</strong> · event={escape(entry['event_id'])} · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · actor={escape(entry['actor'])} · status={escape(entry['status'])}<br /><span>from={escape(entry['handoff_from'])} · to={escape(entry['handoff_to'])} · channel={escape(entry['channel'])}</span><br /><span>artifact={escape(entry['artifact_ref'])} · next_action={escape(entry['next_action'])} · at={escape(entry['timestamp'])}</span></li>"
        for entry in escalation_handoff_entries
    ) or "<li>none</li>"
    handoff_ack_entries = _build_handoff_ack_entries(pack)
    handoff_ack_owner_counts: Dict[str, int] = {}
    handoff_ack_status_counts: Dict[str, int] = {}
    for entry in handoff_ack_entries:
        handoff_ack_owner_counts[entry['ack_owner']] = handoff_ack_owner_counts.get(entry['ack_owner'], 0) + 1
        handoff_ack_status_counts[entry['ack_status']] = handoff_ack_status_counts.get(entry['ack_status'], 0) + 1
    handoff_ack_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(handoff_ack_owner_counts.items())
    ) or "<li>none</li>"
    handoff_ack_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(handoff_ack_status_counts.items())
    ) or "<li>none</li>"
    handoff_ack_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · event={escape(entry['event_id'])} · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · handoff_to={escape(entry['handoff_to'])}<br /><span>ack_owner={escape(entry['ack_owner'])} · ack_status={escape(entry['ack_status'])} · ack_at={escape(entry['ack_at'])}</span><br /><span>actor={escape(entry['actor'])} · channel={escape(entry['channel'])} · artifact={escape(entry['artifact_ref'])}</span><br /><span>{escape(entry['summary'])}</span></li>"
        for entry in handoff_ack_entries
    ) or "<li>none</li>"
    owner_escalation_entries = _build_owner_escalation_digest_entries(pack)
    owner_escalation_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_escalation_entries:
        counts = owner_escalation_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'signoff': 0, 'reminder': 0, 'freeze': 0, 'handoff': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1
    owner_escalation_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · reminders={counts['reminder']} · freezes={counts['freeze']} · handoffs={counts['handoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_escalation_counts.items())
    ) or "<li>none</li>"
    owner_escalation_item_html = "".join(
        f"<li><strong>{escape(entry['digest_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>{escape(entry['summary'])}</span><br /><span>detail={escape(entry['detail'])}</span></li>"
        for entry in owner_escalation_entries
    ) or "<li>none</li>"
    owner_workload_entries = _build_owner_workload_entries(pack)
    owner_workload_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_workload_entries:
        counts = owner_workload_counts.setdefault(
            entry['owner'],
            {'blocker': 0, 'checklist': 0, 'decision': 0, 'signoff': 0, 'reminder': 0, 'renewal': 0, 'total': 0},
        )
        counts[entry['item_type']] += 1
        counts['total'] += 1
    owner_workload_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · checklist={counts['checklist']} · decisions={counts['decision']} · signoffs={counts['signoff']} · reminders={counts['reminder']} · renewals={counts['renewal']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_workload_counts.items())
    ) or "<li>none</li>"
    owner_workload_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · lane={escape(entry['lane'])}<br /><span>{escape(entry['summary'])}</span><br /><span>detail={escape(entry['detail'])}</span></li>"
        for entry in owner_workload_entries
    ) or "<li>none</li>"
    blocker_html = "".join(
        f"<li><strong>{escape(blocker.blocker_id)}</strong> · surface={escape(blocker.surface_id)} · signoff={escape(blocker.signoff_id)} · owner={escape(blocker.owner)} · status={escape(blocker.status)} · severity={escape(blocker.severity)}<br /><span>{escape(blocker.summary)}</span><br /><span>escalation_owner={escape(blocker.escalation_owner or 'none')} · next_action={escape(blocker.next_action or 'none')}</span></li>"
        for blocker in pack.blocker_log
    ) or "<li>none</li>"
    blocker_timeline_html = "".join(
        f"<li><strong>{escape(event.event_id)}</strong> · blocker={escape(event.blocker_id)} · actor={escape(event.actor)} · status={escape(event.status)}<br /><span>timestamp={escape(event.timestamp)}</span><br /><span>{escape(event.summary)}</span><br /><span>next_action={escape(event.next_action or 'none')}</span></li>"
        for event in pack.blocker_timeline
    ) or "<li>none</li>"
    timeline_index = _build_blocker_timeline_index(pack)
    exception_entries = _build_review_exception_entries(pack)
    exception_html = "".join(
        f"<li><strong>{escape(entry['exception_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['category'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · severity={escape(entry['severity'])}<br /><span>{escape(entry['summary'])}</span><br /><span>latest_event={escape(entry['latest_event'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in exception_entries
    ) or "<li>none</li>"
    exception_owner_counts: Dict[str, Dict[str, int]] = {}
    exception_status_counts: Dict[str, Dict[str, int]] = {}
    exception_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in exception_entries:
        owner_bucket = exception_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["category"]] += 1
        owner_bucket["total"] += 1
        status_bucket = exception_status_counts.setdefault(
            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        status_bucket[entry["category"]] += 1
        status_bucket["total"] += 1
        surface_bucket = exception_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["category"]] += 1
        surface_bucket["total"] += 1
    exception_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(exception_owner_counts.items())
    ) or "<li>none</li>"
    exception_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for status, counts in sorted(exception_status_counts.items())
    ) or "<li>none</li>"
    exception_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for surface_id, counts in sorted(exception_surface_counts.items())
    ) or "<li>none</li>"
    audit_density_entries = _build_audit_density_entries(pack)
    audit_density_band_counts: Dict[str, int] = {}
    for entry in audit_density_entries:
        audit_density_band_counts[entry['load_band']] = audit_density_band_counts.get(entry['load_band'], 0) + 1
    audit_density_band_html = "".join(
        f"<li><strong>{escape(band)}</strong> · count={count}</li>"
        for band, count in sorted(audit_density_band_counts.items())
    ) or "<li>none</li>"
    audit_density_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · surface={escape(entry['surface_id'])} · artifact_total={escape(entry['artifact_total'])} · open_total={escape(entry['open_total'])} · band={escape(entry['load_band'])}<br /><span>checklist={escape(entry['checklist_count'])} · decisions={escape(entry['decision_count'])} · assignments={escape(entry['assignment_count'])}</span><br /><span>signoffs={escape(entry['signoff_count'])} · blockers={escape(entry['blocker_count'])} · timeline={escape(entry['timeline_count'])}</span><br /><span>blocks={escape(entry['block_count'])} · notes={escape(entry['note_count'])}</span></li>"
        for entry in audit_density_entries
    ) or "<li>none</li>"
    freeze_entries = _build_freeze_exception_entries(pack)
    freeze_owner_counts: Dict[str, Dict[str, int]] = {}
    freeze_surface_counts: Dict[str, Dict[str, int]] = {}
    for entry in freeze_entries:
        owner_bucket = freeze_owner_counts.setdefault(
            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        owner_bucket[entry["item_type"]] += 1
        owner_bucket["total"] += 1
        surface_bucket = freeze_surface_counts.setdefault(
            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}
        )
        surface_bucket[entry["item_type"]] += 1
        surface_bucket["total"] += 1
    freeze_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(freeze_owner_counts.items())
    ) or "<li>none</li>"
    freeze_surface_html = "".join(
        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts['blocker']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for surface_id, counts in sorted(freeze_surface_counts.items())
    ) or "<li>none</li>"
    freeze_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])} · window={escape(entry['window'])}<br /><span>{escape(entry['summary'])}</span><br /><span>evidence={escape(entry['evidence'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_entries
    ) or "<li>none</li>"
    freeze_approval_entries = _build_freeze_approval_entries(pack)
    freeze_approval_owner_counts: Dict[str, int] = {}
    freeze_approval_status_counts: Dict[str, int] = {}
    for entry in freeze_approval_entries:
        freeze_approval_owner_counts[entry['freeze_approved_by']] = freeze_approval_owner_counts.get(entry['freeze_approved_by'], 0) + 1
        freeze_approval_status_counts[entry['status']] = freeze_approval_status_counts.get(entry['status'], 0) + 1
    freeze_approval_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(freeze_approval_owner_counts.items())
    ) or "<li>none</li>"
    freeze_approval_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(freeze_approval_status_counts.items())
    ) or "<li>none</li>"
    freeze_approval_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>owner={escape(entry['freeze_owner'])} · approved_by={escape(entry['freeze_approved_by'])} · approved_at={escape(entry['freeze_approved_at'])} · window={escape(entry['freeze_until'])}</span><br /><span>{escape(entry['summary'])}</span><br /><span>latest_event={escape(entry['latest_event'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_approval_entries
    ) or "<li>none</li>"
    freeze_renewal_entries = _build_freeze_renewal_entries(pack)
    freeze_renewal_owner_counts: Dict[str, int] = {}
    freeze_renewal_status_counts: Dict[str, int] = {}
    for entry in freeze_renewal_entries:
        freeze_renewal_owner_counts[entry['renewal_owner']] = freeze_renewal_owner_counts.get(entry['renewal_owner'], 0) + 1
        freeze_renewal_status_counts[entry['renewal_status']] = freeze_renewal_status_counts.get(entry['renewal_status'], 0) + 1
    freeze_renewal_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"
        for owner, count in sorted(freeze_renewal_owner_counts.items())
    ) or "<li>none</li>"
    freeze_renewal_status_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(freeze_renewal_status_counts.items())
    ) or "<li>none</li>"
    freeze_renewal_item_html = "".join(
        f"<li><strong>{escape(entry['entry_id'])}</strong> · blocker={escape(entry['blocker_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>renewal_owner={escape(entry['renewal_owner'])} · renewal_by={escape(entry['renewal_by'])} · renewal_status={escape(entry['renewal_status'])}</span><br /><span>freeze_owner={escape(entry['freeze_owner'])} · freeze_until={escape(entry['freeze_until'])} · approved_by={escape(entry['freeze_approved_by'])}</span><br /><span>{escape(entry['summary'])} · next_action={escape(entry['next_action'])}</span></li>"
        for entry in freeze_renewal_entries
    ) or "<li>none</li>"
    owner_review_queue = _build_owner_review_queue_entries(pack)
    owner_queue_counts: Dict[str, Dict[str, int]] = {}
    for entry in owner_review_queue:
        counts = owner_queue_counts.setdefault(
            entry["owner"],
            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},
        )
        counts[entry["item_type"]] += 1
        counts["total"] += 1
    owner_queue_owner_html = "".join(
        f"<li><strong>{escape(owner)}</strong> · blockers={counts['blocker']} · checklist={counts['checklist']} · decisions={counts['decision']} · signoffs={counts['signoff']} · total={counts['total']}</li>"
        for owner, counts in sorted(owner_queue_counts.items())
    ) or "<li>none</li>"
    owner_queue_item_html = "".join(
        f"<li><strong>{escape(entry['queue_id'])}</strong> · owner={escape(entry['owner'])} · type={escape(entry['item_type'])} · source={escape(entry['source_id'])} · surface={escape(entry['surface_id'])} · status={escape(entry['status'])}<br /><span>{escape(entry['summary'])}</span><br /><span>next_action={escape(entry['next_action'])}</span></li>"
        for entry in owner_review_queue
    ) or "<li>none</li>"
    status_counts: Dict[str, int] = {}
    actor_counts: Dict[str, int] = {}
    for event in pack.blocker_timeline:
        status_counts[event.status] = status_counts.get(event.status, 0) + 1
        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1
    status_summary_html = "".join(
        f"<li><strong>{escape(status)}</strong> · count={count}</li>"
        for status, count in sorted(status_counts.items())
    ) or "<li>none</li>"
    actor_summary_html = "".join(
        f"<li><strong>{escape(actor)}</strong> · count={count}</li>"
        for actor, count in sorted(actor_counts.items())
    ) or "<li>none</li>"
    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}
    orphan_timeline_ids = sorted(
        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids
    )
    latest_blocker_html = "".join(
        (
            f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest={escape(timeline_index[blocker.blocker_id][-1].event_id)} · actor={escape(timeline_index[blocker.blocker_id][-1].actor)} · status={escape(timeline_index[blocker.blocker_id][-1].status)} · timestamp={escape(timeline_index[blocker.blocker_id][-1].timestamp)}</li>"
            if blocker.blocker_id in timeline_index
            else f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest=none</li>"
        )
        for blocker in pack.blocker_log
    ) or "<li>none</li>"
    orphan_timeline_html = "".join(
        f"<li><strong>{escape(blocker_id)}</strong></li>"
        for blocker_id in orphan_timeline_ids
    ) or "<li>none</li>"
    return f'''<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{escape(pack.issue_id)} UI Review Pack</title>
    <style>
      body {{ font-family: Arial, sans-serif; margin: 32px; color: #0f172a; }}
      header {{ margin-bottom: 24px; }}
      h1 {{ margin-bottom: 4px; }}
      .meta {{ color: #475569; font-size: 0.95rem; }}
      .surface {{ margin-top: 24px; padding: 16px 18px; border: 1px solid #d9e2ec; border-radius: 12px; background: #f8fafc; }}
      ul {{ padding-left: 20px; }}
      .summary {{ padding: 18px 20px; background: #eff6ff; border-left: 4px solid #2563eb; }}
    </style>
  </head>
  <body>
    <header>
      <p class="meta">{escape(pack.issue_id)} · {escape(pack.version)}</p>
      <h1>{escape(pack.title)}</h1>
      <p class="meta">Audit: {escape(audit.summary)}</p>
    </header>
    <section class="summary">
      <h2>Readiness</h2>
      <p>Missing checklist coverage: {escape(', '.join(audit.wireframes_missing_checklists) if audit.wireframes_missing_checklists else 'none')}</p>
      <p>Checklist items missing role links: {escape(', '.join(audit.checklist_items_missing_role_links) if audit.checklist_items_missing_role_links else 'none')}</p>
      <p>Missing decision coverage: {escape(', '.join(audit.wireframes_missing_decisions) if audit.wireframes_missing_decisions else 'none')}</p>
      <p>Unresolved decisions missing follow-ups: {escape(', '.join(audit.unresolved_decisions_missing_follow_ups) if audit.unresolved_decisions_missing_follow_ups else 'none')}</p>
      <p>Missing role assignments: {escape(', '.join(audit.wireframes_missing_role_assignments) if audit.wireframes_missing_role_assignments else 'none')}</p>
      <p>Missing signoff coverage: {escape(', '.join(audit.wireframes_missing_signoffs) if audit.wireframes_missing_signoffs else 'none')}</p>
      <p>Decisions missing role links: {escape(', '.join(audit.decisions_missing_role_links) if audit.decisions_missing_role_links else 'none')}</p>
      <p>Missing blocker coverage: {escape(', '.join(audit.unresolved_required_signoffs_without_blockers) if audit.unresolved_required_signoffs_without_blockers else 'none')}</p>
      <p>Missing signoff requested dates: {escape(', '.join(audit.signoffs_missing_requested_dates) if audit.signoffs_missing_requested_dates else 'none')}</p>
      <p>Missing signoff due dates: {escape(', '.join(audit.signoffs_missing_due_dates) if audit.signoffs_missing_due_dates else 'none')}</p>
      <p>Missing signoff escalation owners: {escape(', '.join(audit.signoffs_missing_escalation_owners) if audit.signoffs_missing_escalation_owners else 'none')}</p>
      <p>Missing signoff reminder owners: {escape(', '.join(audit.signoffs_missing_reminder_owners) if audit.signoffs_missing_reminder_owners else 'none')}</p>
      <p>Missing signoff next reminders: {escape(', '.join(audit.signoffs_missing_next_reminders) if audit.signoffs_missing_next_reminders else 'none')}</p>
      <p>Missing signoff reminder cadence: {escape(', '.join(audit.signoffs_missing_reminder_cadence) if audit.signoffs_missing_reminder_cadence else 'none')}</p>
      <p>Breached signoff SLA: {escape(', '.join(audit.signoffs_with_breached_sla) if audit.signoffs_with_breached_sla else 'none')}</p>
      <p>Missing waiver metadata: {escape(', '.join(audit.waived_signoffs_missing_metadata) if audit.waived_signoffs_missing_metadata else 'none')}</p>
      <p>Missing blocker timeline: {escape(', '.join(audit.blockers_missing_timeline_events) if audit.blockers_missing_timeline_events else 'none')}</p>
      <p>Closed blockers missing resolution events: {escape(', '.join(audit.closed_blockers_missing_resolution_events) if audit.closed_blockers_missing_resolution_events else 'none')}</p>
      <p>Freeze exceptions missing owners: {escape(', '.join(audit.freeze_exceptions_missing_owners) if audit.freeze_exceptions_missing_owners else 'none')}</p>
      <p>Freeze exceptions missing windows: {escape(', '.join(audit.freeze_exceptions_missing_until) if audit.freeze_exceptions_missing_until else 'none')}</p>
      <p>Freeze exceptions missing approvers: {escape(', '.join(audit.freeze_exceptions_missing_approvers) if audit.freeze_exceptions_missing_approvers else 'none')}</p>
      <p>Freeze exceptions missing approval dates: {escape(', '.join(audit.freeze_exceptions_missing_approval_dates) if audit.freeze_exceptions_missing_approval_dates else 'none')}</p>
      <p>Freeze exceptions missing renewal owners: {escape(', '.join(audit.freeze_exceptions_missing_renewal_owners) if audit.freeze_exceptions_missing_renewal_owners else 'none')}</p>
      <p>Freeze exceptions missing renewal dates: {escape(', '.join(audit.freeze_exceptions_missing_renewal_dates) if audit.freeze_exceptions_missing_renewal_dates else 'none')}</p>
      <p>Orphan blocker timeline ids: {escape(', '.join(audit.orphan_blocker_timeline_blocker_ids) if audit.orphan_blocker_timeline_blocker_ids else 'none')}</p>
      <p>Handoff events missing targets: {escape(', '.join(audit.handoff_events_missing_targets) if audit.handoff_events_missing_targets else 'none')}</p>
      <p>Handoff events missing artifacts: {escape(', '.join(audit.handoff_events_missing_artifacts) if audit.handoff_events_missing_artifacts else 'none')}</p>
      <p>Handoff events missing ack owners: {escape(', '.join(audit.handoff_events_missing_ack_owners) if audit.handoff_events_missing_ack_owners else 'none')}</p>
      <p>Handoff events missing ack dates: {escape(', '.join(audit.handoff_events_missing_ack_dates) if audit.handoff_events_missing_ack_dates else 'none')}</p>
      <p>Unresolved decisions: {escape(', '.join(audit.unresolved_decision_ids) if audit.unresolved_decision_ids else 'none')}</p>
      <p>Unresolved required signoffs: {escape(', '.join(audit.unresolved_required_signoff_ids) if audit.unresolved_required_signoff_ids else 'none')}</p>
    </section>
    <section class="surface"><h2>Objectives</h2><ul>{objective_html}</ul></section>
    <section class="surface"><h2>Review Summary Board</h2><h3>Entries</h3><ul>{review_summary_item_html}</ul></section>
    <section class="surface"><h2>Objective Coverage Board</h2><h3>By Coverage Status</h3><ul>{objective_coverage_status_html}</ul><h3>By Persona</h3><ul>{objective_coverage_persona_html}</ul><h3>Entries</h3><ul>{objective_coverage_item_html}</ul></section>
    <section class="surface"><h2>Persona Readiness Board</h2><h3>By Readiness</h3><ul>{persona_readiness_status_html}</ul><h3>Entries</h3><ul>{persona_readiness_item_html}</ul></section>
    <section class="surface"><h2>Wireframes</h2><ul>{wireframe_html}</ul></section>
    <section class="surface"><h2>Wireframe Readiness Board</h2><h3>By Readiness</h3><ul>{wireframe_readiness_status_html}</ul><h3>By Device</h3><ul>{wireframe_device_html}</ul><h3>Entries</h3><ul>{wireframe_readiness_item_html}</ul></section>
    <section class="surface"><h2>Interactions</h2><ul>{interaction_html}</ul></section>
    <section class="surface"><h2>Interaction Coverage Board</h2><h3>By Coverage Status</h3><ul>{interaction_coverage_status_html}</ul><h3>By Surface</h3><ul>{interaction_coverage_surface_html}</ul><h3>Entries</h3><ul>{interaction_coverage_item_html}</ul></section>
    <section class="surface"><h2>Open Questions</h2><ul>{question_html}</ul></section>
    <section class="surface"><h2>Open Question Tracker</h2><h3>By Owner</h3><ul>{open_question_owner_html}</ul><h3>By Theme</h3><ul>{open_question_theme_html}</ul><h3>Entries</h3><ul>{open_question_item_html}</ul></section>
    <section class="surface"><h2>Reviewer Checklist</h2><ul>{checklist_html}</ul></section>
    <section class="surface"><h2>Decision Log</h2><ul>{decision_html}</ul></section>
    <section class="surface"><h2>Role Matrix</h2><ul>{role_matrix_html}</ul></section>
    <section class="surface"><h2>Checklist Traceability Board</h2><h3>By Owner</h3><ul>{checklist_trace_owner_html}</ul><h3>By Status</h3><ul>{checklist_trace_status_html}</ul><h3>Entries</h3><ul>{checklist_trace_item_html}</ul></section>
    <section class="surface"><h2>Decision Follow-up Tracker</h2><h3>By Owner</h3><ul>{decision_followup_owner_html}</ul><h3>By Status</h3><ul>{decision_followup_status_html}</ul><h3>Entries</h3><ul>{decision_followup_item_html}</ul></section>
    <section class="surface"><h2>Role Coverage Board</h2><h3>By Surface</h3><ul>{role_coverage_surface_html}</ul><h3>By Status</h3><ul>{role_coverage_status_html}</ul><h3>Entries</h3><ul>{role_coverage_item_html}</ul></section>
    <section class="surface"><h2>Signoff Dependency Board</h2><h3>By Dependency Status</h3><ul>{signoff_dependency_status_html}</ul><h3>By SLA State</h3><ul>{signoff_dependency_sla_html}</ul><h3>Entries</h3><ul>{signoff_dependency_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Log</h2><ul>{signoff_html}</ul></section>
    <section class="surface"><h2>Sign-off SLA Dashboard</h2><h3>SLA States</h3><ul>{signoff_sla_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_sla_owner_html}</ul><h3>Sign-offs</h3><ul>{signoff_sla_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Reminder Queue</h2><h3>By Owner</h3><ul>{signoff_reminder_owner_html}</ul><h3>By Channel</h3><ul>{signoff_reminder_channel_html}</ul><h3>Items</h3><ul>{signoff_reminder_item_html}</ul></section>
    <section class="surface"><h2>Reminder Cadence Board</h2><h3>By Cadence</h3><ul>{reminder_cadence_owner_html}</ul><h3>By Status</h3><ul>{reminder_cadence_status_html}</ul><h3>Items</h3><ul>{reminder_cadence_item_html}</ul></section>
    <section class="surface"><h2>Sign-off Breach Board</h2><h3>SLA States</h3><ul>{signoff_breach_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_breach_owner_html}</ul><h3>Items</h3><ul>{signoff_breach_item_html}</ul></section>
    <section class="surface"><h2>Escalation Dashboard</h2><h3>By Escalation Owner</h3><ul>{escalation_owner_html}</ul><h3>By Status</h3><ul>{escalation_status_html}</ul><h3>Escalations</h3><ul>{escalation_item_html}</ul></section>
    <section class="surface"><h2>Escalation Handoff Ledger</h2><h3>By Status</h3><ul>{escalation_handoff_status_html}</ul><h3>By Channel</h3><ul>{escalation_handoff_channel_html}</ul><h3>Entries</h3><ul>{escalation_handoff_item_html}</ul></section>
    <section class="surface"><h2>Handoff Ack Ledger</h2><h3>By Ack Owner</h3><ul>{handoff_ack_owner_html}</ul><h3>By Ack Status</h3><ul>{handoff_ack_status_html}</ul><h3>Entries</h3><ul>{handoff_ack_item_html}</ul></section>
    <section class="surface"><h2>Owner Escalation Digest</h2><h3>Owners</h3><ul>{owner_escalation_owner_html}</ul><h3>Items</h3><ul>{owner_escalation_item_html}</ul></section>
    <section class="surface"><h2>Owner Workload Board</h2><h3>Owners</h3><ul>{owner_workload_owner_html}</ul><h3>Items</h3><ul>{owner_workload_item_html}</ul></section>
    <section class="surface"><h2>Blocker Log</h2><ul>{blocker_html}</ul></section>
    <section class="surface"><h2>Blocker Timeline</h2><ul>{blocker_timeline_html}</ul></section>
    <section class="surface"><h2>Review Freeze Exception Board</h2><h3>By Owner</h3><ul>{freeze_owner_html}</ul><h3>By Surface</h3><ul>{freeze_surface_html}</ul><h3>Entries</h3><ul>{freeze_item_html}</ul></section>
    <section class="surface"><h2>Freeze Approval Trail</h2><h3>By Approver</h3><ul>{freeze_approval_owner_html}</ul><h3>By Status</h3><ul>{freeze_approval_status_html}</ul><h3>Entries</h3><ul>{freeze_approval_item_html}</ul></section>
    <section class="surface"><h2>Freeze Renewal Tracker</h2><h3>By Renewal Owner</h3><ul>{freeze_renewal_owner_html}</ul><h3>By Renewal Status</h3><ul>{freeze_renewal_status_html}</ul><h3>Entries</h3><ul>{freeze_renewal_item_html}</ul></section>
    <section class="surface"><h2>Review Exceptions</h2><ul>{exception_html}</ul></section>
    <section class="surface"><h2>Review Exception Matrix</h2><h3>By Owner</h3><ul>{exception_owner_html}</ul><h3>By Status</h3><ul>{exception_status_html}</ul><h3>By Surface</h3><ul>{exception_surface_html}</ul></section>
    <section class="surface"><h2>Audit Density Board</h2><h3>By Load Band</h3><ul>{audit_density_band_html}</ul><h3>Entries</h3><ul>{audit_density_item_html}</ul></section>
    <section class="surface"><h2>Owner Review Queue</h2><h3>Owners</h3><ul>{owner_queue_owner_html}</ul><h3>Items</h3><ul>{owner_queue_item_html}</ul></section>
    <section class="surface"><h2>Blocker Timeline Summary</h2><h3>Events by Status</h3><ul>{status_summary_html}</ul><h3>Events by Actor</h3><ul>{actor_summary_html}</ul><h3>Latest Blocker Events</h3><ul>{latest_blocker_html}</ul><h3>Orphan Timeline Blockers</h3><ul>{orphan_timeline_html}</ul></section>
  </body>
</html>
'''


def write_ui_review_pack_bundle(root_dir: str, pack: UIReviewPack) -> UIReviewPackArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    slug = pack.issue_id.lower().replace(" ", "-")
    markdown_path = str(base / f"{slug}-review-pack.md")
    html_path = str(base / f"{slug}-review-pack.html")
    decision_log_path = str(base / f"{slug}-decision-log.md")
    review_summary_board_path = str(base / f"{slug}-review-summary-board.md")
    objective_coverage_board_path = str(base / f"{slug}-objective-coverage-board.md")
    persona_readiness_board_path = str(base / f"{slug}-persona-readiness-board.md")
    wireframe_readiness_board_path = str(base / f"{slug}-wireframe-readiness-board.md")
    interaction_coverage_board_path = str(base / f"{slug}-interaction-coverage-board.md")
    open_question_tracker_path = str(base / f"{slug}-open-question-tracker.md")
    checklist_traceability_board_path = str(base / f"{slug}-checklist-traceability-board.md")
    decision_followup_tracker_path = str(base / f"{slug}-decision-followup-tracker.md")
    role_matrix_path = str(base / f"{slug}-role-matrix.md")
    role_coverage_board_path = str(base / f"{slug}-role-coverage-board.md")
    signoff_dependency_board_path = str(base / f"{slug}-signoff-dependency-board.md")
    signoff_log_path = str(base / f"{slug}-signoff-log.md")
    signoff_sla_dashboard_path = str(base / f"{slug}-signoff-sla-dashboard.md")
    signoff_reminder_queue_path = str(base / f"{slug}-signoff-reminder-queue.md")
    reminder_cadence_board_path = str(base / f"{slug}-reminder-cadence-board.md")
    signoff_breach_board_path = str(base / f"{slug}-signoff-breach-board.md")
    escalation_dashboard_path = str(base / f"{slug}-escalation-dashboard.md")
    escalation_handoff_ledger_path = str(base / f"{slug}-escalation-handoff-ledger.md")
    handoff_ack_ledger_path = str(base / f"{slug}-handoff-ack-ledger.md")
    owner_escalation_digest_path = str(base / f"{slug}-owner-escalation-digest.md")
    owner_workload_board_path = str(base / f"{slug}-owner-workload-board.md")
    blocker_log_path = str(base / f"{slug}-blocker-log.md")
    blocker_timeline_path = str(base / f"{slug}-blocker-timeline.md")
    freeze_exception_board_path = str(base / f"{slug}-freeze-exception-board.md")
    freeze_approval_trail_path = str(base / f"{slug}-freeze-approval-trail.md")
    freeze_renewal_tracker_path = str(base / f"{slug}-freeze-renewal-tracker.md")
    exception_log_path = str(base / f"{slug}-exception-log.md")
    exception_matrix_path = str(base / f"{slug}-exception-matrix.md")
    audit_density_board_path = str(base / f"{slug}-audit-density-board.md")
    owner_review_queue_path = str(base / f"{slug}-owner-review-queue.md")
    blocker_timeline_summary_path = str(base / f"{slug}-blocker-timeline-summary.md")
    audit = UIReviewPackAuditor().audit(pack)
    Path(markdown_path).write_text(render_ui_review_pack_report(pack, audit))
    Path(html_path).write_text(render_ui_review_pack_html(pack, audit))
    Path(decision_log_path).write_text(render_ui_review_decision_log(pack))
    Path(review_summary_board_path).write_text(render_ui_review_review_summary_board(pack))
    Path(objective_coverage_board_path).write_text(render_ui_review_objective_coverage_board(pack))
    Path(persona_readiness_board_path).write_text(render_ui_review_persona_readiness_board(pack))
    Path(wireframe_readiness_board_path).write_text(render_ui_review_wireframe_readiness_board(pack))
    Path(interaction_coverage_board_path).write_text(render_ui_review_interaction_coverage_board(pack))
    Path(open_question_tracker_path).write_text(render_ui_review_open_question_tracker(pack))
    Path(checklist_traceability_board_path).write_text(render_ui_review_checklist_traceability_board(pack))
    Path(decision_followup_tracker_path).write_text(render_ui_review_decision_followup_tracker(pack))
    Path(role_matrix_path).write_text(render_ui_review_role_matrix(pack))
    Path(role_coverage_board_path).write_text(render_ui_review_role_coverage_board(pack))
    Path(signoff_dependency_board_path).write_text(render_ui_review_signoff_dependency_board(pack))
    Path(signoff_log_path).write_text(render_ui_review_signoff_log(pack))
    Path(signoff_sla_dashboard_path).write_text(render_ui_review_signoff_sla_dashboard(pack))
    Path(signoff_reminder_queue_path).write_text(render_ui_review_signoff_reminder_queue(pack))
    Path(reminder_cadence_board_path).write_text(render_ui_review_reminder_cadence_board(pack))
    Path(signoff_breach_board_path).write_text(render_ui_review_signoff_breach_board(pack))
    Path(escalation_dashboard_path).write_text(render_ui_review_escalation_dashboard(pack))
    Path(escalation_handoff_ledger_path).write_text(render_ui_review_escalation_handoff_ledger(pack))
    Path(handoff_ack_ledger_path).write_text(render_ui_review_handoff_ack_ledger(pack))
    Path(owner_escalation_digest_path).write_text(render_ui_review_owner_escalation_digest(pack))
    Path(owner_workload_board_path).write_text(render_ui_review_owner_workload_board(pack))
    Path(blocker_log_path).write_text(render_ui_review_blocker_log(pack))
    Path(blocker_timeline_path).write_text(render_ui_review_blocker_timeline(pack))
    Path(freeze_exception_board_path).write_text(render_ui_review_freeze_exception_board(pack))
    Path(freeze_approval_trail_path).write_text(render_ui_review_freeze_approval_trail(pack))
    Path(freeze_renewal_tracker_path).write_text(render_ui_review_freeze_renewal_tracker(pack))
    Path(exception_log_path).write_text(render_ui_review_exception_log(pack))
    Path(exception_matrix_path).write_text(render_ui_review_exception_matrix(pack))
    Path(audit_density_board_path).write_text(render_ui_review_audit_density_board(pack))
    Path(owner_review_queue_path).write_text(render_ui_review_owner_review_queue(pack))
    Path(blocker_timeline_summary_path).write_text(render_ui_review_blocker_timeline_summary(pack))
    return UIReviewPackArtifacts(
        root_dir=str(base),
        markdown_path=markdown_path,
        html_path=html_path,
        decision_log_path=decision_log_path,
        review_summary_board_path=review_summary_board_path,
        objective_coverage_board_path=objective_coverage_board_path,
        persona_readiness_board_path=persona_readiness_board_path,
        wireframe_readiness_board_path=wireframe_readiness_board_path,
        interaction_coverage_board_path=interaction_coverage_board_path,
        open_question_tracker_path=open_question_tracker_path,
        checklist_traceability_board_path=checklist_traceability_board_path,
        decision_followup_tracker_path=decision_followup_tracker_path,
        role_matrix_path=role_matrix_path,
        role_coverage_board_path=role_coverage_board_path,
        signoff_dependency_board_path=signoff_dependency_board_path,
        signoff_log_path=signoff_log_path,
        signoff_sla_dashboard_path=signoff_sla_dashboard_path,
        signoff_reminder_queue_path=signoff_reminder_queue_path,
        reminder_cadence_board_path=reminder_cadence_board_path,
        signoff_breach_board_path=signoff_breach_board_path,
        escalation_dashboard_path=escalation_dashboard_path,
        escalation_handoff_ledger_path=escalation_handoff_ledger_path,
        handoff_ack_ledger_path=handoff_ack_ledger_path,
        owner_escalation_digest_path=owner_escalation_digest_path,
        owner_workload_board_path=owner_workload_board_path,
        blocker_log_path=blocker_log_path,
        blocker_timeline_path=blocker_timeline_path,
        freeze_exception_board_path=freeze_exception_board_path,
        freeze_approval_trail_path=freeze_approval_trail_path,
        freeze_renewal_tracker_path=freeze_renewal_tracker_path,
        exception_log_path=exception_log_path,
        exception_matrix_path=exception_matrix_path,
        audit_density_board_path=audit_density_board_path,
        owner_review_queue_path=owner_review_queue_path,
        blocker_timeline_summary_path=blocker_timeline_summary_path,
    )


_compat_source = sys.modules[__name__]

_install_compat_module(
    _compat_source,
    "console_ia",
    [
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
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _console_ia_export_name in [
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
]:
    _console_ia_export = globals()[_console_ia_export_name]
    if getattr(_console_ia_export, "__module__", None) == __name__:
        _console_ia_export.__module__ = f"{__name__}.console_ia"

_install_compat_module(
    _compat_source,
    "design_system",
    [
        "AuditRequirement",
        "CommandAction",
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
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _design_system_export_name in [
    "AuditRequirement",
    "CommandAction",
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
]:
    _design_system_export = globals()[_design_system_export_name]
    if getattr(_design_system_export, "__module__", None) == __name__:
        _design_system_export.__module__ = f"{__name__}.design_system"

_install_compat_module(
    _compat_source,
    "operations",
    [
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
        "RegressionFinding",
        "RegressionCenter",
        "TriageCluster",
        "QueueControlCenter",
        "VersionChangeSummary",
        "VersionedArtifact",
        "VersionedArtifactHistory",
        "WeeklyOperationsArtifacts",
        "WeeklyOperationsReport",
        "build_repo_collaboration_metrics",
        "render_dashboard_builder_report",
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
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _operations_export_name in [
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
    "RegressionFinding",
    "RegressionCenter",
    "TriageCluster",
    "QueueControlCenter",
    "VersionChangeSummary",
    "VersionedArtifact",
    "VersionedArtifactHistory",
    "WeeklyOperationsArtifacts",
    "WeeklyOperationsReport",
    "build_repo_collaboration_metrics",
    "render_dashboard_builder_report",
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
]:
    _operations_export = globals()[_operations_export_name]
    if getattr(_operations_export, "__module__", None) == __name__:
        _operations_export.__module__ = f"{__name__}.operations"

_install_compat_module(
    _compat_source,
    "observability",
    [
        "ArtifactRecord",
        "AuditEntry",
        "GitSyncTelemetry",
        "LogEntry",
        "ObservabilityLedger",
        "PullRequestFreshness",
        "RepoSyncAudit",
        "RunCloseout",
        "TaskRun",
        "TraceEntry",
        "sha256_file",
        "utc_now",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _observability_export_name in [
    "ArtifactRecord",
    "AuditEntry",
    "GitSyncTelemetry",
    "LogEntry",
    "ObservabilityLedger",
    "PullRequestFreshness",
    "RepoSyncAudit",
    "RunCloseout",
    "TaskRun",
    "TraceEntry",
    "sha256_file",
    "utc_now",
]:
    _observability_export = globals()[_observability_export_name]
    if getattr(_observability_export, "__module__", None) == __name__:
        _observability_export.__module__ = f"{__name__}.observability"

_install_compat_module(
    _compat_source,
    "models",
    [
        "BillingInterval",
        "BillingRate",
        "BillingSummary",
        "FlowRun",
        "FlowRunStatus",
        "FlowStepRun",
        "FlowStepStatus",
        "FlowTemplate",
        "FlowTemplateStep",
        "FlowTrigger",
        "Priority",
        "RiskAssessment",
        "RiskLevel",
        "RiskSignal",
        "Task",
        "TaskState",
        "TriageLabel",
        "TriageRecord",
        "TriageStatus",
        "UsageRecord",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _models_export_name in [
    "BillingInterval",
    "BillingRate",
    "BillingSummary",
    "FlowRun",
    "FlowRunStatus",
    "FlowStepRun",
    "FlowStepStatus",
    "FlowTemplate",
    "FlowTemplateStep",
    "FlowTrigger",
    "Priority",
    "RiskAssessment",
    "RiskLevel",
    "RiskSignal",
    "Task",
    "TaskState",
    "TriageLabel",
    "TriageRecord",
    "TriageStatus",
    "UsageRecord",
]:
    _models_export = globals()[_models_export_name]
    if getattr(_models_export, "__module__", None) == __name__:
        _models_export.__module__ = f"{__name__}.models"

_install_compat_module(
    _compat_source,
    "ui_review",
    [
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
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _ui_review_export_name in [
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
]:
    _ui_review_export = globals()[_ui_review_export_name]
    if getattr(_ui_review_export, "__module__", None) == __name__:
        _ui_review_export.__module__ = f"{__name__}.ui_review"

_install_compat_module(
    _compat_source,
    "execution_contract",
    [
        "AuditPolicy",
        "build_operations_api_contract",
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
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/contract/execution.go",
)
_install_compat_module(
    _compat_source,
    "deprecation",
    ["LEGACY_RUNTIME_GUIDANCE", "legacy_runtime_message", "warn_legacy_runtime_surface"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/regression/deprecation_contract_test.go",
)
_install_compat_module(
    _compat_source,
    "audit_events",
    [
        "APPROVAL_RECORDED_EVENT",
        "BUDGET_OVERRIDE_EVENT",
        "FLOW_HANDOFF_EVENT",
        "MANUAL_TAKEOVER_EVENT",
        "P0_AUDIT_EVENT_SPECS",
        "SCHEDULER_DECISION_EVENT",
        "AuditEventSpec",
        "get_audit_event_spec",
        "missing_required_fields",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/observability/audit_spec.go",
)
_install_compat_module(
    _compat_source,
    "dsl",
    ["WorkflowDefinition", "WorkflowStep"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/definition.go",
)

_install_compat_module(
    _compat_source,
    "utility_surfaces",
    [
        "BudgetDecision",
        "CostController",
        "EpicMilestone",
        "ExecutionPackRoadmap",
        "build_execution_pack_roadmap",
        "MemoryPattern",
        "TaskMemoryStore",
        "REQUIRED_REPORT_ARTIFACTS",
        "ValidationReportDecision",
        "enforce_validation_report_policy",
        "ParallelIssueQueue",
        "issue_state_map",
        "LEGACY_PYTHON_WRAPPER_NOTICE",
        "append_missing_flag",
        "build_bigclawctl_exec_args",
        "repo_root_from_script",
        "run_bigclawctl_shim",
        "build_workspace_bootstrap_args",
        "translate_workspace_validate_args",
        "build_workspace_validate_args",
        "build_github_sync_args",
        "build_refill_args",
        "build_workspace_runtime_bootstrap_args",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "cost_control",
    ["BudgetDecision", "CostController"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/costcontrol/controller.go",
)
_install_compat_module(
    _compat_source,
    "roadmap",
    ["EpicMilestone", "ExecutionPackRoadmap", "build_execution_pack_roadmap"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/regression/roadmap_contract_test.go",
)
_install_compat_module(
    _compat_source,
    "memory",
    ["MemoryPattern", "TaskMemoryStore"],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "validation_policy",
    ["REQUIRED_REPORT_ARTIFACTS", "ValidationReportDecision", "enforce_validation_report_policy"],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "parallel_refill",
    ["ParallelIssueQueue", "issue_state_map"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/refill/queue.go",
)
_install_compat_module(
    _compat_source,
    "legacy_shim",
    [
        "LEGACY_PYTHON_WRAPPER_NOTICE",
        "append_missing_flag",
        "build_bigclawctl_exec_args",
        "repo_root_from_script",
        "run_bigclawctl_shim",
        "build_workspace_bootstrap_args",
        "translate_workspace_validate_args",
        "build_workspace_validate_args",
        "build_github_sync_args",
        "build_refill_args",
        "build_workspace_runtime_bootstrap_args",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/legacyshim/wrappers.go",
)
_install_compat_module(
    _compat_source,
    "support_surfaces",
    [
        "AlertDigestSubscription",
        "BusEvent",
        "CI_COMPLETED_EVENT",
        "CollaborationComment",
        "CollaborationThread",
        "CommandResult",
        "Connector",
        "DashboardRunContract",
        "DashboardRunContractAudit",
        "DashboardRunContractLibrary",
        "DecisionNote",
        "EventBus",
        "GitHubConnector",
        "GitSyncError",
        "JiraConnector",
        "LinearConnector",
        "PULL_REQUEST_COMMENT_EVENT",
        "PilotImplementationResult",
        "PilotKPI",
        "RepoSyncStatus",
        "RunDetailEvent",
        "RunDetailResource",
        "RunDetailStat",
        "RunDetailTab",
        "SavedView",
        "SavedViewCatalog",
        "SavedViewCatalogAudit",
        "SavedViewFilter",
        "SavedViewLibrary",
        "SchemaField",
        "SourceIssue",
        "SurfaceSchema",
        "TASK_FAILED_EVENT",
        "build_collaboration_thread",
        "build_collaboration_thread_from_audits",
        "collaboration_now",
        "ensure_repo_sync",
        "inspect_repo_sync",
        "install_git_hooks",
        "map_priority",
        "map_source_issue_to_task",
        "map_state",
        "merge_collaboration_threads",
        "render_collaboration_lines",
        "render_collaboration_panel_html",
        "render_dashboard_run_contract_report",
        "render_pilot_implementation_report",
        "render_resource_grid",
        "render_run_detail_console",
        "render_saved_view_report",
        "render_timeline_panel",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
for _support_surfaces_export_name in [
    "AlertDigestSubscription",
    "BusEvent",
    "CI_COMPLETED_EVENT",
    "CollaborationComment",
    "CollaborationThread",
    "CommandResult",
    "Connector",
    "DashboardRunContract",
    "DashboardRunContractAudit",
    "DashboardRunContractLibrary",
    "DecisionNote",
    "EventBus",
    "GitHubConnector",
    "GitSyncError",
    "JiraConnector",
    "LinearConnector",
    "PULL_REQUEST_COMMENT_EVENT",
    "PilotImplementationResult",
    "PilotKPI",
    "RepoSyncStatus",
    "RunDetailEvent",
    "RunDetailResource",
    "RunDetailStat",
    "RunDetailTab",
    "SavedView",
    "SavedViewCatalog",
    "SavedViewCatalogAudit",
    "SavedViewFilter",
    "SavedViewLibrary",
    "SchemaField",
    "SourceIssue",
    "SurfaceSchema",
    "TASK_FAILED_EVENT",
    "build_collaboration_thread",
    "build_collaboration_thread_from_audits",
    "collaboration_now",
    "ensure_repo_sync",
    "inspect_repo_sync",
    "install_git_hooks",
    "map_priority",
    "map_source_issue_to_task",
    "map_state",
    "merge_collaboration_threads",
    "render_collaboration_lines",
    "render_collaboration_panel_html",
    "render_dashboard_run_contract_report",
    "render_pilot_implementation_report",
    "render_resource_grid",
    "render_run_detail_console",
    "render_saved_view_report",
    "render_timeline_panel",
]:
    _support_surfaces_export = globals()[_support_surfaces_export_name]
    if getattr(_support_surfaces_export, "__module__", None) == __name__:
        _support_surfaces_export.__module__ = f"{__name__}.support_surfaces"

_install_compat_module(
    _compat_source,
    "connectors",
    ["Connector", "GitHubConnector", "JiraConnector", "LinearConnector", "SourceIssue"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/connector.go",
)
_install_compat_module(
    _compat_source,
    "mapping",
    ["map_priority", "map_source_issue_to_task", "map_state"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/mapping.go",
)
_install_compat_module(
    _compat_source,
    "saved_views",
    [
        "AlertDigestSubscription",
        "SavedView",
        "SavedViewCatalog",
        "SavedViewCatalogAudit",
        "SavedViewFilter",
        "SavedViewLibrary",
        "render_saved_view_report",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/saved_views.go",
)
_install_compat_module(
    _compat_source,
    "pilot",
    ["PilotImplementationResult", "PilotKPI", "render_pilot_implementation_report"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/pilot/report.go",
)
_install_compat_module(
    _compat_source,
    "github_sync",
    ["CommandResult", "GitSyncError", "RepoSyncStatus", "ensure_repo_sync", "inspect_repo_sync", "install_git_hooks"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/githubsync/sync.go",
)
_install_compat_module(
    _compat_source,
    "event_bus",
    [
        "CI_COMPLETED_EVENT",
        "PULL_REQUEST_COMMENT_EVENT",
        "TASK_FAILED_EVENT",
        "BusEvent",
        "EventBus",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/events/bus.go",
)
_install_compat_module(
    _compat_source,
    "dashboard_run_contract",
    [
        "DashboardRunContract",
        "DashboardRunContractAudit",
        "DashboardRunContractLibrary",
        "SchemaField",
        "SurfaceSchema",
        "render_dashboard_run_contract_report",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/dashboard_run_contract.go",
)
_install_compat_module(
    _compat_source,
    "workspace_bootstrap",
    [
        "WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE",
        "WorkspaceBootstrapError",
        "CacheBootstrapState",
        "WorkspaceBootstrapStatus",
        "CommandResult",
        "CACHE_REMOTE",
        "BOOTSTRAP_BRANCH_PREFIX",
        "DEFAULT_CACHE_BASE",
        "sanitize_issue_identifier",
        "bootstrap_branch_name",
        "default_cache_base",
        "normalize_repo_locator",
        "repo_cache_key",
        "cache_root_for_repo",
        "resolve_cache_root",
        "default_cache_root",
        "ensure_mirror",
        "ensure_seed",
        "configure_seed_remotes",
        "bootstrap_workspace",
        "cleanup_workspace",
        "status_as_json",
        "build_parser",
        "emit",
        "main",
        "build_validation_report",
        "render_validation_markdown",
        "write_validation_report",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/bootstrap/bootstrap.go",
)
_install_compat_module(
    _compat_source,
    "workspace_bootstrap_cli",
    [
        "WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE",
        "WorkspaceBootstrapError",
        "bootstrap_workspace",
        "build_parser",
        "cleanup_workspace",
        "emit",
        "main",
    ],
    DEFAULT_CACHE_BASE=WORKSPACE_BOOTSTRAP_DEFAULT_CACHE_BASE,
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "workspace_bootstrap_validation",
    [
        "bootstrap_workspace",
        "build_validation_report",
        "cleanup_workspace",
        "render_validation_markdown",
        "write_validation_report",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "collaboration",
    [
        "CollaborationComment",
        "CollaborationThread",
        "DecisionNote",
        "build_collaboration_thread",
        "merge_collaboration_threads",
        "build_collaboration_thread_from_audits",
        "render_collaboration_lines",
        "render_collaboration_panel_html",
        "collaboration_now",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "run_detail",
    [
        "RunDetailEvent",
        "RunDetailResource",
        "RunDetailStat",
        "RunDetailTab",
        "render_resource_grid",
        "render_run_detail_console",
        "render_timeline_panel",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "repo_surfaces",
    [
        "RepoPost",
        "RepoDiscussionBoard",
        "RepoCommit",
        "CommitLineage",
        "CommitDiff",
        "RepoGatewayError",
        "normalize_gateway_error",
        "normalize_commit",
        "normalize_lineage",
        "normalize_diff",
        "repo_audit_payload",
        "REPO_ACTION_PERMISSIONS",
        "REPO_ROLE_POLICIES",
        "RepoPermissionContract",
        "repo_required_audit_fields",
        "missing_repo_audit_fields",
        "RepoSpace",
        "RepoAgent",
        "RunCommitLink",
        "VALID_ROLES",
        "RunCommitBinding",
        "validate_roles",
        "bind_run_commits",
        "RepoRegistry",
        "LineageEvidence",
        "TriageRecommendation",
        "recommend_triage_action",
        "approval_evidence_packet",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "repo_board",
    ["RepoDiscussionBoard", "RepoPost"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/board.go",
)
_install_compat_module(
    _compat_source,
    "repo_commits",
    ["CommitDiff", "CommitLineage", "RepoCommit"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/commits.go",
)
_install_compat_module(
    _compat_source,
    "repo_gateway",
    [
        "RepoGatewayClient",
        "RepoGatewayError",
        "normalize_commit",
        "normalize_diff",
        "normalize_gateway_error",
        "normalize_lineage",
        "repo_audit_payload",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/gateway.go",
)
_install_compat_module(
    _compat_source,
    "repo_governance",
    [
        "REPO_ACTION_PERMISSIONS",
        "REPO_ROLE_POLICIES",
        "RepoPermissionContract",
        "missing_repo_audit_fields",
        "repo_required_audit_fields",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/governance.go",
)
_install_compat_module(
    _compat_source,
    "repo_links",
    ["RunCommitBinding", "VALID_ROLES", "bind_run_commits", "validate_roles"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/links.go",
)
_install_compat_module(
    _compat_source,
    "repo_plane",
    ["RepoAgent", "RepoSpace", "RunCommitLink"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/plane.go",
)
_install_compat_module(
    _compat_source,
    "repo_registry",
    ["RepoRegistry"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/registry.go",
)
_install_compat_module(
    _compat_source,
    "repo_triage",
    ["LineageEvidence", "TriageRecommendation", "approval_evidence_packet", "recommend_triage_action"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/triage.go",
)

_install_compat_module(
    _compat_source,
    "control_surfaces",
    [
        "REQUIRED_RUN_CLOSEOUTS",
        "ALLOWED_SCOPE_STATUSES",
        "FreezeException",
        "GovernanceBacklogItem",
        "ScopeFreezeBoard",
        "ScopeFreezeAudit",
        "ScopeFreezeGovernance",
        "render_scope_freeze_report",
        "ALLOWED_ISSUE_CATEGORIES",
        "ALLOWED_ISSUE_PRIORITIES",
        "ArchivedIssue",
        "IssuePriorityArchive",
        "IssuePriorityArchiveAudit",
        "IssuePriorityArchivist",
        "render_issue_priority_archive_report",
        "RiskFactor",
        "RiskScore",
        "RiskScorer",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "governance",
    [
        "FreezeException",
        "GovernanceBacklogItem",
        "ScopeFreezeAudit",
        "ScopeFreezeBoard",
        "ScopeFreezeGovernance",
        "render_scope_freeze_report",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/governance/freeze.go",
)
_install_compat_module(
    _compat_source,
    "issue_archive",
    [
        "ArchivedIssue",
        "IssuePriorityArchive",
        "IssuePriorityArchiveAudit",
        "IssuePriorityArchivist",
        "render_issue_priority_archive_report",
    ],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/issuearchive/archive.go",
)
_install_compat_module(
    _compat_source,
    "risk",
    ["RiskFactor", "RiskScore", "RiskScorer"],
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/risk/risk.go",
)

from . import runtime as _legacy_runtime_surface


def _install_legacy_surface_module(name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    for export_name in export_names:
        module.__dict__[export_name] = getattr(_legacy_runtime_surface, export_name)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


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
from .control_surfaces import (
    FreezeException,
    GovernanceBacklogItem,
    ScopeFreezeAudit,
    ScopeFreezeBoard,
    ScopeFreezeGovernance,
    render_scope_freeze_report,
    ArchivedIssue,
    IssuePriorityArchive,
    IssuePriorityArchiveAudit,
    IssuePriorityArchivist,
    RiskFactor,
    RiskScore,
    RiskScorer,
    render_issue_priority_archive_report,
)
_report_exports_before = set(globals())

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

_report_export_names = sorted(
    name for name in (set(globals()) - _report_exports_before)
    if name != "_report_exports_before"
)
for _reports_export_name in _report_export_names:
    _reports_export = globals()[_reports_export_name]
    if getattr(_reports_export, "__module__", None) == __name__:
        _reports_export.__module__ = f"{__name__}.reports"

_install_compat_module(
    _compat_source,
    "reports",
    _report_export_names,
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)

_install_compat_module(
    _compat_source,
    "evaluation",
    [
        "EvaluationCriterion",
        "BenchmarkCase",
        "ReplayRecord",
        "ReplayOutcome",
        "BenchmarkResult",
        "BenchmarkComparison",
        "BenchmarkSuiteResult",
        "BenchmarkRunner",
        "render_benchmark_suite_report",
        "render_replay_detail_page",
        "render_run_replay_index_page",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
)
_install_compat_module(
    _compat_source,
    "planning",
    [
        "PRIORITY_WEIGHTS",
        "GOAL_STATUS_ORDER",
        "EvidenceLink",
        "CandidateEntry",
        "CandidateBacklog",
        "EntryGate",
        "EntryGateDecision",
        "CandidatePlanner",
        "render_candidate_backlog_report",
        "build_v3_candidate_backlog",
        "build_v3_entry_gate",
        "WeeklyGoal",
        "WeeklyExecutionPlan",
        "FourWeekExecutionPlan",
        "build_big_4701_execution_plan",
        "build_pilot_rollout_scorecard",
        "evaluate_candidate_gate",
        "render_pilot_rollout_gate_report",
        "render_four_week_execution_report",
    ],
    GO_MAINLINE_REPLACEMENT="repo-native compatibility surface",
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
