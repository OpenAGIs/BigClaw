import sys
import types
from dataclasses import dataclass, field
import hashlib
import json
from datetime import datetime, timezone
from html import escape
from pathlib import Path
from enum import Enum
from typing import Any, Dict, Iterable, List, Optional, Sequence


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
    return [field for field in spec.required_fields if field not in details]


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


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


def sha256_file(path: str) -> str:
    file_path = Path(path)
    if not file_path.exists() or not file_path.is_file():
        return ""

    digest = hashlib.sha256()
    with file_path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(8192), b""):
            digest.update(chunk)
    return digest.hexdigest()


VALID_RUN_COMMIT_ROLES = {"source", "candidate", "closeout", "accepted"}


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


@dataclass
class RunCommitBinding:
    links: List[RunCommitLink]

    @property
    def accepted_commit_hash(self) -> str:
        for link in self.links:
            if link.role == "accepted":
                return link.commit_hash
        return ""


def validate_run_commit_roles(links: Iterable[RunCommitLink]) -> None:
    invalid = sorted({link.role for link in links if link.role not in VALID_RUN_COMMIT_ROLES})
    if invalid:
        raise ValueError(f"unsupported run commit roles: {', '.join(invalid)}")


def bind_run_commits(links: List[RunCommitLink]) -> RunCommitBinding:
    validate_run_commit_roles(links)
    return RunCommitBinding(links=list(links))


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
from .runtime import RiskFactor, RiskScore, RiskScorer
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
from .operations import (
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
                validation_command="cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
                capabilities=["release-gate", "console-shell", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="design-system-tests",
                        target="bigclaw-go/internal/designsystem/designsystem_test.go",
                        capability="release-gate",
                        note="Go-native component inventory, accessibility, and UI acceptance coverage",
                    ),
                    EvidenceLink(
                        label="design-system-surface",
                        target="bigclaw-go/internal/designsystem/designsystem.go",
                        capability="release-gate",
                        note="Go-native information architecture and console chrome contracts",
                    ),
                    EvidenceLink(
                        label="review-pack-tests",
                        target="bigclaw-go/internal/uireview/uireview_test.go",
                        capability="release-gate",
                        note="Go-native deterministic review packet validation",
                    ),
                    EvidenceLink(
                        label="review-pack-surface",
                        target="bigclaw-go/internal/uireview/uireview.go",
                        capability="release-gate",
                        note="Go-native review objectives, wireframes, interaction coverage, and open questions",
                    ),
                    EvidenceLink(
                        label="review-pack-render",
                        target="bigclaw-go/internal/uireview/render.go",
                        capability="release-gate",
                        note="Go-native review report and board rendering",
                    ),
                    EvidenceLink(
                        label="candidate-planner-tests",
                        target="bigclaw-go/internal/planning/planning_test.go",
                        capability="reporting",
                        note="release candidate validation commands and evidence targets stay Go-only",
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
                    "(cd bigclaw-go && go test ./internal/product ./internal/worker ./internal/workflow ./internal/scheduler)"
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
                        target="src/bigclaw/saved_views.py",
                        capability="saved-views",
                        note="saved views, digest subscriptions, and governed filters",
                    ),
                    EvidenceLink(
                        label="saved-views-tests",
                        target="bigclaw-go/internal/product/saved_views_test.go",
                        capability="saved-views",
                        note="Go-native saved-view audit coverage",
                    ),
                    EvidenceLink(
                        label="simulation-src",
                        target="bigclaw-go/internal/evaluation/evaluation.go",
                        capability="rollback-simulation",
                        note="Go-native simulation, replay, and comparison evidence",
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
]
