import json
import sys
import types
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Optional, Protocol

from .models import (
    BillingInterval,
    BillingRate,
    BillingSummary,
    FlowRun,
    FlowRunStatus,
    FlowStepRun,
    FlowStepStatus,
    FlowTemplate,
    FlowTemplateStep,
    FlowTrigger,
    Priority,
    RiskAssessment,
    RiskLevel,
    RiskSignal,
    Task,
    TaskState,
    TriageLabel,
    TriageRecord,
    TriageStatus,
    UsageRecord,
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


def map_priority(priority: str) -> Priority:
    priority = (priority or "").upper()
    if priority == "P0":
        return Priority.P0
    if priority == "P1":
        return Priority.P1
    return Priority.P2


def map_state(state: str) -> TaskState:
    normalized = (state or "").lower()
    if "progress" in normalized:
        return TaskState.IN_PROGRESS
    if "done" in normalized or "closed" in normalized:
        return TaskState.DONE
    if "block" in normalized:
        return TaskState.BLOCKED
    return TaskState.TODO


def map_source_issue_to_task(issue: SourceIssue) -> Task:
    risk_level = RiskLevel.HIGH if "prod" in issue.title.lower() else RiskLevel.LOW
    return Task(
        task_id=issue.source_id,
        source=issue.source,
        title=issue.title,
        description=issue.description,
        labels=issue.labels,
        priority=map_priority(issue.priority),
        state=map_state(issue.state),
        risk_level=risk_level,
        required_tools=["github" if issue.source == "github" else "connector"],
        acceptance_criteria=["Synced from source issue"],
        validation_plan=["mapping-test"],
    )


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

        unapproved_exceptions = sorted(
            exception.issue_id for exception in board.exceptions if not exception.approved
        )

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
            fields=[ExecutionField.from_dict(field_data) for field_data in data.get("fields", [])],
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
                str(name): [str(field_name) for field_name in fields]
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
            missing = [field_name for field_name in expected_fields if field_name not in model.required_fields]
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
            if (
                api.required_permission
                and api.required_permission in permission_names
                and api.required_permission not in permissions_granted_by_roles
            ):
                apis_without_role_coverage.append(api.name)

        permissions_without_roles = sorted(
            permission for permission in permission_names if permission not in permissions_granted_by_roles
        )

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
    return "\n".join(lines)


def build_operations_api_contract(contract_id: str = "OPE-131", version: str = "v4.0-draft1") -> ExecutionContract:
    return ExecutionContract.from_dict(
        {
            "contract_id": contract_id,
            "version": version,
            "models": [
                {
                    "name": "OperationsDashboardResponse",
                    "owner": "operations",
                    "fields": [
                        {"name": "period", "field_type": "string"},
                        {"name": "total_runs", "field_type": "int"},
                        {"name": "success_rate", "field_type": "float"},
                        {"name": "approval_queue_depth", "field_type": "int"},
                        {"name": "sla_breach_count", "field_type": "int"},
                        {"name": "top_blockers", "field_type": "string[]", "required": False},
                    ],
                },
                {
                    "name": "RunDetailResponse",
                    "owner": "operations",
                    "fields": [
                        {"name": "run_id", "field_type": "string"},
                        {"name": "task_id", "field_type": "string"},
                        {"name": "status", "field_type": "string"},
                        {"name": "timeline_events", "field_type": "RunDetailEvent[]"},
                        {"name": "resources", "field_type": "RunDetailResource[]"},
                        {"name": "audit_count", "field_type": "int"},
                    ],
                },
                {
                    "name": "RunReplayResponse",
                    "owner": "operations",
                    "fields": [
                        {"name": "run_id", "field_type": "string"},
                        {"name": "replay_available", "field_type": "bool"},
                        {"name": "replay_path", "field_type": "string", "required": False},
                        {"name": "benchmark_case_ids", "field_type": "string[]", "required": False},
                    ],
                },
                {
                    "name": "QueueControlCenterResponse",
                    "owner": "operations",
                    "fields": [
                        {"name": "queue_depth", "field_type": "int"},
                        {"name": "blocked_runs", "field_type": "int"},
                        {"name": "oldest_wait_seconds", "field_type": "float"},
                        {"name": "operator_actions", "field_type": "ConsoleAction[]"},
                    ],
                },
            ],
            "apis": [
                {"name": "operations_dashboard", "method": "GET", "path": "/operations/dashboard", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.dashboard.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.dashboard.requests"]},
                {"name": "run_detail", "method": "GET", "path": "/operations/runs/{run_id}", "request_model": "", "response_model": "RunDetailResponse", "required_permission": "operations.run.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.run_detail.requests"]},
                {"name": "run_replay", "method": "GET", "path": "/operations/runs/{run_id}/replay", "request_model": "", "response_model": "RunReplayResponse", "required_permission": "operations.run.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.run_replay.requests"]},
                {"name": "queue_control_center", "method": "GET", "path": "/operations/queue/control-center", "request_model": "", "response_model": "QueueControlCenterResponse", "required_permission": "operations.queue.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.queue.requests"]},
                {"name": "queue_action", "method": "POST", "path": "/operations/queue/actions", "request_model": "QueueControlCenterResponse", "response_model": "QueueControlCenterResponse", "required_permission": "operations.queue.act", "emitted_audits": ["execution.manual_takeover"], "emitted_metrics": ["operations.queue.actions"]},
                {"name": "risk_overview", "method": "GET", "path": "/operations/risk/overview", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.risk.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.risk.requests"]},
                {"name": "sla_overview", "method": "GET", "path": "/operations/sla/overview", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.sla.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.sla.requests"]},
                {"name": "regressions", "method": "GET", "path": "/operations/regressions", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.regression.read", "emitted_audits": ["execution.scheduler_decision"], "emitted_metrics": ["operations.regression.requests"]},
                {"name": "flow_detail", "method": "GET", "path": "/operations/flows/{run_id}", "request_model": "", "response_model": "RunDetailResponse", "required_permission": "operations.flow.read", "emitted_audits": ["execution.flow_handoff"], "emitted_metrics": ["operations.flow.requests"]},
                {"name": "billing_entitlements", "method": "GET", "path": "/operations/billing/entitlements", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.billing.read", "emitted_audits": ["execution.budget_override"], "emitted_metrics": ["operations.billing.requests"]},
                {"name": "approval_queue", "method": "GET", "path": "/operations/approvals", "request_model": "", "response_model": "OperationsDashboardResponse", "required_permission": "operations.approval.read", "emitted_audits": ["execution.approval_recorded"], "emitted_metrics": ["operations.approval.requests"]},
                {"name": "approval_action", "method": "POST", "path": "/operations/approvals/actions", "request_model": "OperationsDashboardResponse", "response_model": "OperationsDashboardResponse", "required_permission": "operations.run.approve", "emitted_audits": ["execution.approval_recorded"], "emitted_metrics": ["operations.approval.actions"]},
            ],
            "permissions": [
                {"name": "operations.dashboard.read", "resource": "operations-dashboard", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.run.read", "resource": "operations-run", "actions": ["read"], "scopes": ["workspace", "project"]},
                {"name": "operations.queue.read", "resource": "operations-queue", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.queue.act", "resource": "operations-queue", "actions": ["update"], "scopes": ["workspace"]},
                {"name": "operations.risk.read", "resource": "operations-risk", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.sla.read", "resource": "operations-sla", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.regression.read", "resource": "operations-regression", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.flow.read", "resource": "operations-flow", "actions": ["read"], "scopes": ["workspace", "project"]},
                {"name": "operations.billing.read", "resource": "operations-billing", "actions": ["read"], "scopes": ["workspace", "finance"]},
                {"name": "operations.approval.read", "resource": "operations-approval", "actions": ["read"], "scopes": ["workspace"]},
                {"name": "operations.run.approve", "resource": "operations-run", "actions": ["approve"], "scopes": ["workspace", "project"]},
            ],
            "roles": [
                {"name": "eng-lead", "personas": ["Eng Lead"], "granted_permissions": ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.approval.read", "operations.run.approve"], "scope_bindings": ["project", "workspace"], "escalation_target": "vp-eng"},
                {"name": "platform-admin", "personas": ["Platform Admin"], "granted_permissions": ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.billing.read", "operations.approval.read", "operations.run.approve"], "scope_bindings": ["workspace"], "escalation_target": "vp-eng"},
                {"name": "vp-eng", "personas": ["VP Eng"], "granted_permissions": ["operations.dashboard.read", "operations.run.read", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.billing.read", "operations.approval.read", "operations.run.approve"], "scope_bindings": ["portfolio", "workspace"], "escalation_target": "none"},
                {"name": "cross-team-operator", "personas": ["Cross-Team Operator"], "granted_permissions": ["operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.approval.read"], "scope_bindings": ["cross-team", "workspace"], "escalation_target": "eng-lead"},
            ],
            "metrics": [
                {"name": "operations.dashboard.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.run_detail.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.run_replay.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.queue.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.queue.actions", "unit": "count", "owner": "operations"},
                {"name": "operations.risk.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.sla.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.regression.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.flow.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.billing.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.approval.requests", "unit": "count", "owner": "operations"},
                {"name": "operations.approval.actions", "unit": "count", "owner": "operations"},
            ],
            "audit_policies": [
                {"event_type": "execution.scheduler_decision", "required_fields": ["task_id", "run_id", "medium", "approved", "reason", "risk_level", "risk_score"], "retention_days": 180, "severity": "info"},
                {"event_type": "execution.manual_takeover", "required_fields": ["task_id", "run_id", "target_team", "reason", "requested_by", "required_approvals"], "retention_days": 365, "severity": "warn"},
                {"event_type": "execution.approval_recorded", "required_fields": ["task_id", "run_id", "approvals", "approval_count", "acceptance_status"], "retention_days": 365, "severity": "info"},
                {"event_type": "execution.budget_override", "required_fields": ["task_id", "run_id", "requested_budget", "approved_budget", "override_actor", "reason"], "retention_days": 365, "severity": "warn"},
                {"event_type": "execution.flow_handoff", "required_fields": ["task_id", "run_id", "source_stage", "target_team", "reason", "collaboration_mode"], "retention_days": 180, "severity": "info"},
            ],
        }
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


def validate_roles(links: List[RunCommitLink]) -> None:
    invalid = [link.role for link in links if link.role not in VALID_ROLES]
    if invalid:
        invalid_text = ", ".join(sorted(set(invalid)))
        raise ValueError(f"unsupported run commit roles: {invalid_text}")


def bind_run_commits(links: List[RunCommitLink]) -> RunCommitBinding:
    validate_roles(list(links))
    return RunCommitBinding(links=list(links))


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


def _slug(value: str) -> str:
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


def _repo_board_now() -> str:
    from datetime import datetime, timezone

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

    def to_collaboration_comment(self) -> "CollaborationComment":
        from .collaboration import CollaborationComment

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


def collaboration_now() -> str:
    from datetime import datetime, timezone

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
    comments: Optional[List[CollaborationComment]] = None,
    decisions: Optional[List[DecisionNote]] = None,
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
    audits: List[Dict[str, Any]],
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
    from html import escape

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


def render_issue_priority_archive_report(
    archive: IssuePriorityArchive,
    audit: IssuePriorityArchiveAudit,
) -> str:
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
        score = max(0.0, 100 - ((penalties * 100) / self.view_count))
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "catalog_name": self.catalog_name,
            "version": self.version,
            "view_count": self.view_count,
            "subscription_count": self.subscription_count,
            "duplicate_view_names": {key: list(values) for key, values in self.duplicate_view_names.items()},
            "invalid_visibility_views": list(self.invalid_visibility_views),
            "views_missing_filters": list(self.views_missing_filters),
            "duplicate_default_views": {key: list(values) for key, values in self.duplicate_default_views.items()},
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
            duplicate_view_names={str(key): [str(value) for value in values] for key, values in dict(data.get("duplicate_view_names", {})).items()},
            invalid_visibility_views=[str(name) for name in data.get("invalid_visibility_views", [])],
            views_missing_filters=[str(name) for name in data.get("views_missing_filters", [])],
            duplicate_default_views={str(key): [str(value) for value in values] for key, values in dict(data.get("duplicate_default_views", {})).items()},
            orphan_subscriptions=[str(name) for name in data.get("orphan_subscriptions", [])],
            subscriptions_missing_recipients=[str(name) for name in data.get("subscriptions_missing_recipients", [])],
            subscriptions_with_invalid_channel=[str(name) for name in data.get("subscriptions_with_invalid_channel", [])],
            subscriptions_with_invalid_cadence=[str(name) for name in data.get("subscriptions_with_invalid_cadence", [])],
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


def _install_compatibility_module(name: str, exports: Dict[str, object], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__.update(exports)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


_install_compatibility_module(
    "connectors",
    {
        "SourceIssue": SourceIssue,
        "Connector": Connector,
        "GitHubConnector": GitHubConnector,
        "LinearConnector": LinearConnector,
        "JiraConnector": JiraConnector,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/connector.go",
)
_install_compatibility_module(
    "mapping",
    {
        "map_priority": map_priority,
        "map_state": map_state,
        "map_source_issue_to_task": map_source_issue_to_task,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/mapping.go",
)
_install_compatibility_module(
    "dsl",
    {
        "WorkflowDefinition": WorkflowDefinition,
        "WorkflowStep": WorkflowStep,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/definition.go",
)
_install_compatibility_module(
    "risk",
    {
        "RiskFactor": RiskFactor,
        "RiskScore": RiskScore,
        "RiskScorer": RiskScorer,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/risk/risk.go",
)
_install_compatibility_module(
    "governance",
    {
        "FreezeException": FreezeException,
        "GovernanceBacklogItem": GovernanceBacklogItem,
        "ScopeFreezeAudit": ScopeFreezeAudit,
        "ScopeFreezeBoard": ScopeFreezeBoard,
        "ScopeFreezeGovernance": ScopeFreezeGovernance,
        "render_scope_freeze_report": render_scope_freeze_report,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/governance/freeze.go",
)
_install_compatibility_module(
    "audit_events",
    {
        "APPROVAL_RECORDED_EVENT": APPROVAL_RECORDED_EVENT,
        "BUDGET_OVERRIDE_EVENT": BUDGET_OVERRIDE_EVENT,
        "FLOW_HANDOFF_EVENT": FLOW_HANDOFF_EVENT,
        "MANUAL_TAKEOVER_EVENT": MANUAL_TAKEOVER_EVENT,
        "P0_AUDIT_EVENT_SPECS": P0_AUDIT_EVENT_SPECS,
        "SCHEDULER_DECISION_EVENT": SCHEDULER_DECISION_EVENT,
        "AuditEventSpec": AuditEventSpec,
        "get_audit_event_spec": get_audit_event_spec,
        "missing_required_fields": missing_required_fields,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/observability/audit_spec.go",
)
_install_compatibility_module(
    "execution_contract",
    {
        "AuditPolicy": AuditPolicy,
        "build_operations_api_contract": build_operations_api_contract,
        "ExecutionApiSpec": ExecutionApiSpec,
        "ExecutionContract": ExecutionContract,
        "ExecutionContractAudit": ExecutionContractAudit,
        "ExecutionContractLibrary": ExecutionContractLibrary,
        "ExecutionField": ExecutionField,
        "ExecutionModel": ExecutionModel,
        "ExecutionPermission": ExecutionPermission,
        "ExecutionPermissionMatrix": ExecutionPermissionMatrix,
        "ExecutionRole": ExecutionRole,
        "MetricDefinition": MetricDefinition,
        "PermissionCheckResult": PermissionCheckResult,
        "render_execution_contract_report": render_execution_contract_report,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/contract/execution.go",
)
_install_compatibility_module(
    "repo_governance",
    {
        "REPO_ACTION_PERMISSIONS": REPO_ACTION_PERMISSIONS,
        "REPO_ROLE_POLICIES": REPO_ROLE_POLICIES,
        "RepoPermissionContract": RepoPermissionContract,
        "repo_required_audit_fields": repo_required_audit_fields,
        "missing_repo_audit_fields": missing_repo_audit_fields,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/governance.go",
)
_install_compatibility_module(
    "repo_links",
    {
        "RunCommitBinding": RunCommitBinding,
        "VALID_ROLES": VALID_ROLES,
        "validate_roles": validate_roles,
        "bind_run_commits": bind_run_commits,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/links.go",
)
_install_compatibility_module(
    "repo_commits",
    {
        "RepoCommit": RepoCommit,
        "CommitLineage": CommitLineage,
        "CommitDiff": CommitDiff,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/commits.go",
)
_install_compatibility_module(
    "repo_gateway",
    {
        "RepoGatewayError": RepoGatewayError,
        "normalize_gateway_error": normalize_gateway_error,
        "normalize_commit": normalize_commit,
        "normalize_lineage": normalize_lineage,
        "normalize_diff": normalize_diff,
        "repo_audit_payload": repo_audit_payload,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/gateway.go",
)
_install_compatibility_module(
    "repo_plane",
    {
        "RepoSpace": RepoSpace,
        "RepoAgent": RepoAgent,
        "RunCommitLink": RunCommitLink,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/plane.go",
)
_install_compatibility_module(
    "repo_registry",
    {
        "RepoRegistry": RepoRegistry,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/registry.go",
)
_install_compatibility_module(
    "repo_board",
    {
        "RepoPost": RepoPost,
        "RepoDiscussionBoard": RepoDiscussionBoard,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/board.go",
)
_install_compatibility_module(
    "repo_triage",
    {
        "LineageEvidence": LineageEvidence,
        "TriageRecommendation": TriageRecommendation,
        "recommend_triage_action": recommend_triage_action,
        "approval_evidence_packet": approval_evidence_packet,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/triage.go",
)
_install_compatibility_module(
    "collaboration",
    {
        "CollaborationComment": CollaborationComment,
        "CollaborationThread": CollaborationThread,
        "DecisionNote": DecisionNote,
        "build_collaboration_thread": build_collaboration_thread,
        "merge_collaboration_threads": merge_collaboration_threads,
        "build_collaboration_thread_from_audits": build_collaboration_thread_from_audits,
        "render_collaboration_lines": render_collaboration_lines,
        "render_collaboration_panel_html": render_collaboration_panel_html,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/flow/flow.go",
)
_install_compatibility_module(
    "issue_archive",
    {
        "ArchivedIssue": ArchivedIssue,
        "IssuePriorityArchive": IssuePriorityArchive,
        "IssuePriorityArchiveAudit": IssuePriorityArchiveAudit,
        "IssuePriorityArchivist": IssuePriorityArchivist,
        "render_issue_priority_archive_report": render_issue_priority_archive_report,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/console.go",
)
_install_compatibility_module(
    "roadmap",
    {
        "EpicMilestone": EpicMilestone,
        "ExecutionPackRoadmap": ExecutionPackRoadmap,
        "build_execution_pack_roadmap": build_execution_pack_roadmap,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/console.go",
)
_install_compatibility_module(
    "saved_views",
    {
        "AlertDigestSubscription": AlertDigestSubscription,
        "SavedView": SavedView,
        "SavedViewCatalog": SavedViewCatalog,
        "SavedViewCatalogAudit": SavedViewCatalogAudit,
        "SavedViewFilter": SavedViewFilter,
        "SavedViewLibrary": SavedViewLibrary,
        "render_saved_view_report": render_saved_view_report,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/console.go",
)

_UI_REVIEW_PUBLIC_NAMES = (
    'InteractionFlow',
    'OpenQuestion',
    'ReviewBlocker',
    'ReviewBlockerEvent',
    'ReviewDecision',
    'ReviewObjective',
    'ReviewRoleAssignment',
    'ReviewSignoff',
    'ReviewerChecklistItem',
    'UIReviewPack',
    'UIReviewPackArtifacts',
    'UIReviewPackAudit',
    'UIReviewPackAuditor',
    'WireframeSurface',
    'build_big_4204_review_pack',
    'render_ui_review_blocker_log',
    'render_ui_review_blocker_timeline',
    'render_ui_review_blocker_timeline_summary',
    'render_ui_review_escalation_dashboard',
    'render_ui_review_escalation_handoff_ledger',
    'render_ui_review_exception_log',
    'render_ui_review_exception_matrix',
    'render_ui_review_freeze_approval_trail',
    'render_ui_review_freeze_exception_board',
    'render_ui_review_freeze_renewal_tracker',
    'render_ui_review_handoff_ack_ledger',
    'render_ui_review_interaction_coverage_board',
    'render_ui_review_objective_coverage_board',
    'render_ui_review_open_question_tracker',
    'render_ui_review_owner_escalation_digest',
    'render_ui_review_persona_readiness_board',
    'render_ui_review_review_summary_board',
    'render_ui_review_owner_review_queue',
    'render_ui_review_owner_workload_board',
    'render_ui_review_checklist_traceability_board',
    'render_ui_review_decision_followup_tracker',
    'render_ui_review_audit_density_board',
    'render_ui_review_reminder_cadence_board',
    'render_ui_review_role_coverage_board',
    'render_ui_review_wireframe_readiness_board',
    'render_ui_review_signoff_breach_board',
    'render_ui_review_signoff_dependency_board',
    'render_ui_review_signoff_reminder_queue',
    'render_ui_review_signoff_sla_dashboard',
    'render_ui_review_decision_log',
    'render_ui_review_pack_html',
    'render_ui_review_role_matrix',
    'render_ui_review_signoff_log',
    'render_ui_review_pack_report',
    'write_ui_review_pack_bundle',
)
_ui_review_namespace = {
    "__name__": f"{__name__}.ui_review",
    "__package__": __name__,
    "__builtins__": __builtins__,
}
exec(
    compile('from dataclasses import dataclass, field\nfrom html import escape\nfrom pathlib import Path\nfrom typing import Dict, List\n\n\n@dataclass(frozen=True)\nclass ReviewObjective:\n    objective_id: str\n    title: str\n    persona: str\n    outcome: str\n    success_signal: str\n    priority: str = "P1"\n    dependencies: List[str] = field(default_factory=list)\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "objective_id": self.objective_id,\n            "title": self.title,\n            "persona": self.persona,\n            "outcome": self.outcome,\n            "success_signal": self.success_signal,\n            "priority": self.priority,\n            "dependencies": list(self.dependencies),\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewObjective":\n        return cls(\n            objective_id=str(data["objective_id"]),\n            title=str(data["title"]),\n            persona=str(data["persona"]),\n            outcome=str(data["outcome"]),\n            success_signal=str(data["success_signal"]),\n            priority=str(data.get("priority", "P1")),\n            dependencies=[str(item) for item in data.get("dependencies", [])],\n        )\n\n\n@dataclass(frozen=True)\nclass WireframeSurface:\n    surface_id: str\n    name: str\n    device: str\n    entry_point: str\n    primary_blocks: List[str] = field(default_factory=list)\n    review_notes: List[str] = field(default_factory=list)\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "surface_id": self.surface_id,\n            "name": self.name,\n            "device": self.device,\n            "entry_point": self.entry_point,\n            "primary_blocks": list(self.primary_blocks),\n            "review_notes": list(self.review_notes),\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "WireframeSurface":\n        return cls(\n            surface_id=str(data["surface_id"]),\n            name=str(data["name"]),\n            device=str(data["device"]),\n            entry_point=str(data["entry_point"]),\n            primary_blocks=[str(item) for item in data.get("primary_blocks", [])],\n            review_notes=[str(item) for item in data.get("review_notes", [])],\n        )\n\n\n@dataclass(frozen=True)\nclass InteractionFlow:\n    flow_id: str\n    name: str\n    trigger: str\n    system_response: str\n    states: List[str] = field(default_factory=list)\n    exceptions: List[str] = field(default_factory=list)\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "flow_id": self.flow_id,\n            "name": self.name,\n            "trigger": self.trigger,\n            "system_response": self.system_response,\n            "states": list(self.states),\n            "exceptions": list(self.exceptions),\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "InteractionFlow":\n        return cls(\n            flow_id=str(data["flow_id"]),\n            name=str(data["name"]),\n            trigger=str(data["trigger"]),\n            system_response=str(data["system_response"]),\n            states=[str(item) for item in data.get("states", [])],\n            exceptions=[str(item) for item in data.get("exceptions", [])],\n        )\n\n\n@dataclass(frozen=True)\nclass OpenQuestion:\n    question_id: str\n    theme: str\n    question: str\n    owner: str\n    impact: str\n    status: str = "open"\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "question_id": self.question_id,\n            "theme": self.theme,\n            "question": self.question,\n            "owner": self.owner,\n            "impact": self.impact,\n            "status": self.status,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "OpenQuestion":\n        return cls(\n            question_id=str(data["question_id"]),\n            theme=str(data["theme"]),\n            question=str(data["question"]),\n            owner=str(data["owner"]),\n            impact=str(data["impact"]),\n            status=str(data.get("status", "open")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewerChecklistItem:\n    item_id: str\n    surface_id: str\n    prompt: str\n    owner: str\n    status: str = "todo"\n    evidence_links: List[str] = field(default_factory=list)\n    notes: str = ""\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "item_id": self.item_id,\n            "surface_id": self.surface_id,\n            "prompt": self.prompt,\n            "owner": self.owner,\n            "status": self.status,\n            "evidence_links": list(self.evidence_links),\n            "notes": self.notes,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewerChecklistItem":\n        return cls(\n            item_id=str(data["item_id"]),\n            surface_id=str(data["surface_id"]),\n            prompt=str(data["prompt"]),\n            owner=str(data["owner"]),\n            status=str(data.get("status", "todo")),\n            evidence_links=[str(item) for item in data.get("evidence_links", [])],\n            notes=str(data.get("notes", "")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewDecision:\n    decision_id: str\n    surface_id: str\n    owner: str\n    summary: str\n    rationale: str\n    status: str = "proposed"\n    follow_up: str = ""\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "decision_id": self.decision_id,\n            "surface_id": self.surface_id,\n            "owner": self.owner,\n            "summary": self.summary,\n            "rationale": self.rationale,\n            "status": self.status,\n            "follow_up": self.follow_up,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewDecision":\n        return cls(\n            decision_id=str(data["decision_id"]),\n            surface_id=str(data["surface_id"]),\n            owner=str(data["owner"]),\n            summary=str(data["summary"]),\n            rationale=str(data["rationale"]),\n            status=str(data.get("status", "proposed")),\n            follow_up=str(data.get("follow_up", "")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewRoleAssignment:\n    assignment_id: str\n    surface_id: str\n    role: str\n    responsibilities: List[str] = field(default_factory=list)\n    checklist_item_ids: List[str] = field(default_factory=list)\n    decision_ids: List[str] = field(default_factory=list)\n    status: str = "planned"\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "assignment_id": self.assignment_id,\n            "surface_id": self.surface_id,\n            "role": self.role,\n            "responsibilities": list(self.responsibilities),\n            "checklist_item_ids": list(self.checklist_item_ids),\n            "decision_ids": list(self.decision_ids),\n            "status": self.status,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewRoleAssignment":\n        return cls(\n            assignment_id=str(data["assignment_id"]),\n            surface_id=str(data["surface_id"]),\n            role=str(data["role"]),\n            responsibilities=[str(item) for item in data.get("responsibilities", [])],\n            checklist_item_ids=[str(item) for item in data.get("checklist_item_ids", [])],\n            decision_ids=[str(item) for item in data.get("decision_ids", [])],\n            status=str(data.get("status", "planned")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewSignoff:\n    signoff_id: str\n    assignment_id: str\n    surface_id: str\n    role: str\n    status: str = "pending"\n    required: bool = True\n    evidence_links: List[str] = field(default_factory=list)\n    notes: str = ""\n    waiver_owner: str = ""\n    waiver_reason: str = ""\n    requested_at: str = ""\n    due_at: str = ""\n    escalation_owner: str = ""\n    sla_status: str = "on-track"\n    reminder_owner: str = ""\n    reminder_channel: str = ""\n    last_reminder_at: str = ""\n    next_reminder_at: str = ""\n    reminder_cadence: str = ""\n    reminder_status: str = "scheduled"\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "signoff_id": self.signoff_id,\n            "assignment_id": self.assignment_id,\n            "surface_id": self.surface_id,\n            "role": self.role,\n            "status": self.status,\n            "required": self.required,\n            "evidence_links": list(self.evidence_links),\n            "notes": self.notes,\n            "waiver_owner": self.waiver_owner,\n            "waiver_reason": self.waiver_reason,\n            "requested_at": self.requested_at,\n            "due_at": self.due_at,\n            "escalation_owner": self.escalation_owner,\n            "sla_status": self.sla_status,\n            "reminder_owner": self.reminder_owner,\n            "reminder_channel": self.reminder_channel,\n            "last_reminder_at": self.last_reminder_at,\n            "next_reminder_at": self.next_reminder_at,\n            "reminder_cadence": self.reminder_cadence,\n            "reminder_status": self.reminder_status,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewSignoff":\n        return cls(\n            signoff_id=str(data["signoff_id"]),\n            assignment_id=str(data["assignment_id"]),\n            surface_id=str(data["surface_id"]),\n            role=str(data["role"]),\n            status=str(data.get("status", "pending")),\n            required=bool(data.get("required", True)),\n            evidence_links=[str(item) for item in data.get("evidence_links", [])],\n            notes=str(data.get("notes", "")),\n            waiver_owner=str(data.get("waiver_owner", "")),\n            waiver_reason=str(data.get("waiver_reason", "")),\n            requested_at=str(data.get("requested_at", "")),\n            due_at=str(data.get("due_at", "")),\n            escalation_owner=str(data.get("escalation_owner", "")),\n            sla_status=str(data.get("sla_status", "on-track")),\n            reminder_owner=str(data.get("reminder_owner", "")),\n            reminder_channel=str(data.get("reminder_channel", "")),\n            last_reminder_at=str(data.get("last_reminder_at", "")),\n            next_reminder_at=str(data.get("next_reminder_at", "")),\n            reminder_cadence=str(data.get("reminder_cadence", "")),\n            reminder_status=str(data.get("reminder_status", "scheduled")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewBlocker:\n    blocker_id: str\n    surface_id: str\n    signoff_id: str\n    owner: str\n    summary: str\n    status: str = "open"\n    severity: str = "medium"\n    escalation_owner: str = ""\n    next_action: str = ""\n    freeze_exception: bool = False\n    freeze_owner: str = ""\n    freeze_until: str = ""\n    freeze_reason: str = ""\n    freeze_approved_by: str = ""\n    freeze_approved_at: str = ""\n    freeze_renewal_owner: str = ""\n    freeze_renewal_by: str = ""\n    freeze_renewal_status: str = "not-needed"\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "blocker_id": self.blocker_id,\n            "surface_id": self.surface_id,\n            "signoff_id": self.signoff_id,\n            "owner": self.owner,\n            "summary": self.summary,\n            "status": self.status,\n            "severity": self.severity,\n            "escalation_owner": self.escalation_owner,\n            "next_action": self.next_action,\n            "freeze_exception": self.freeze_exception,\n            "freeze_owner": self.freeze_owner,\n            "freeze_until": self.freeze_until,\n            "freeze_reason": self.freeze_reason,\n            "freeze_approved_by": self.freeze_approved_by,\n            "freeze_approved_at": self.freeze_approved_at,\n            "freeze_renewal_owner": self.freeze_renewal_owner,\n            "freeze_renewal_by": self.freeze_renewal_by,\n            "freeze_renewal_status": self.freeze_renewal_status,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlocker":\n        return cls(\n            blocker_id=str(data["blocker_id"]),\n            surface_id=str(data["surface_id"]),\n            signoff_id=str(data["signoff_id"]),\n            owner=str(data["owner"]),\n            summary=str(data["summary"]),\n            status=str(data.get("status", "open")),\n            severity=str(data.get("severity", "medium")),\n            escalation_owner=str(data.get("escalation_owner", "")),\n            next_action=str(data.get("next_action", "")),\n            freeze_exception=bool(data.get("freeze_exception", False)),\n            freeze_owner=str(data.get("freeze_owner", "")),\n            freeze_until=str(data.get("freeze_until", "")),\n            freeze_reason=str(data.get("freeze_reason", "")),\n            freeze_approved_by=str(data.get("freeze_approved_by", "")),\n            freeze_approved_at=str(data.get("freeze_approved_at", "")),\n            freeze_renewal_owner=str(data.get("freeze_renewal_owner", "")),\n            freeze_renewal_by=str(data.get("freeze_renewal_by", "")),\n            freeze_renewal_status=str(data.get("freeze_renewal_status", "not-needed")),\n        )\n\n\n@dataclass(frozen=True)\nclass ReviewBlockerEvent:\n    event_id: str\n    blocker_id: str\n    actor: str\n    status: str\n    summary: str\n    timestamp: str\n    next_action: str = ""\n    handoff_from: str = ""\n    handoff_to: str = ""\n    channel: str = ""\n    artifact_ref: str = ""\n    ack_owner: str = ""\n    ack_at: str = ""\n    ack_status: str = "pending"\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "event_id": self.event_id,\n            "blocker_id": self.blocker_id,\n            "actor": self.actor,\n            "status": self.status,\n            "summary": self.summary,\n            "timestamp": self.timestamp,\n            "next_action": self.next_action,\n            "handoff_from": self.handoff_from,\n            "handoff_to": self.handoff_to,\n            "channel": self.channel,\n            "artifact_ref": self.artifact_ref,\n            "ack_owner": self.ack_owner,\n            "ack_at": self.ack_at,\n            "ack_status": self.ack_status,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "ReviewBlockerEvent":\n        return cls(\n            event_id=str(data["event_id"]),\n            blocker_id=str(data["blocker_id"]),\n            actor=str(data["actor"]),\n            status=str(data["status"]),\n            summary=str(data["summary"]),\n            timestamp=str(data["timestamp"]),\n            next_action=str(data.get("next_action", "")),\n            handoff_from=str(data.get("handoff_from", "")),\n            handoff_to=str(data.get("handoff_to", "")),\n            channel=str(data.get("channel", "")),\n            artifact_ref=str(data.get("artifact_ref", "")),\n            ack_owner=str(data.get("ack_owner", "")),\n            ack_at=str(data.get("ack_at", "")),\n            ack_status=str(data.get("ack_status", "pending")),\n        )\n\n\n@dataclass(frozen=True)\nclass UIReviewPackArtifacts:\n    root_dir: str\n    markdown_path: str\n    html_path: str\n    decision_log_path: str\n    review_summary_board_path: str\n    objective_coverage_board_path: str\n    persona_readiness_board_path: str\n    wireframe_readiness_board_path: str\n    interaction_coverage_board_path: str\n    open_question_tracker_path: str\n    checklist_traceability_board_path: str\n    decision_followup_tracker_path: str\n    role_matrix_path: str\n    role_coverage_board_path: str\n    signoff_dependency_board_path: str\n    signoff_log_path: str\n    signoff_sla_dashboard_path: str\n    signoff_reminder_queue_path: str\n    reminder_cadence_board_path: str\n    signoff_breach_board_path: str\n    escalation_dashboard_path: str\n    escalation_handoff_ledger_path: str\n    handoff_ack_ledger_path: str\n    owner_escalation_digest_path: str\n    owner_workload_board_path: str\n    blocker_log_path: str\n    blocker_timeline_path: str\n    freeze_exception_board_path: str\n    freeze_approval_trail_path: str\n    freeze_renewal_tracker_path: str\n    exception_log_path: str\n    exception_matrix_path: str\n    audit_density_board_path: str\n    owner_review_queue_path: str\n    blocker_timeline_summary_path: str\n\n\n@dataclass\nclass UIReviewPack:\n    issue_id: str\n    title: str\n    version: str\n    objectives: List[ReviewObjective] = field(default_factory=list)\n    wireframes: List[WireframeSurface] = field(default_factory=list)\n    interactions: List[InteractionFlow] = field(default_factory=list)\n    open_questions: List[OpenQuestion] = field(default_factory=list)\n    reviewer_checklist: List[ReviewerChecklistItem] = field(default_factory=list)\n    requires_reviewer_checklist: bool = False\n    decision_log: List[ReviewDecision] = field(default_factory=list)\n    requires_decision_log: bool = False\n    role_matrix: List[ReviewRoleAssignment] = field(default_factory=list)\n    requires_role_matrix: bool = False\n    signoff_log: List[ReviewSignoff] = field(default_factory=list)\n    requires_signoff_log: bool = False\n    blocker_log: List[ReviewBlocker] = field(default_factory=list)\n    requires_blocker_log: bool = False\n    blocker_timeline: List[ReviewBlockerEvent] = field(default_factory=list)\n    requires_blocker_timeline: bool = False\n\n    def to_dict(self) -> Dict[str, object]:\n        return {\n            "issue_id": self.issue_id,\n            "title": self.title,\n            "version": self.version,\n            "objectives": [objective.to_dict() for objective in self.objectives],\n            "wireframes": [wireframe.to_dict() for wireframe in self.wireframes],\n            "interactions": [interaction.to_dict() for interaction in self.interactions],\n            "open_questions": [question.to_dict() for question in self.open_questions],\n            "reviewer_checklist": [item.to_dict() for item in self.reviewer_checklist],\n            "requires_reviewer_checklist": self.requires_reviewer_checklist,\n            "decision_log": [decision.to_dict() for decision in self.decision_log],\n            "requires_decision_log": self.requires_decision_log,\n            "role_matrix": [assignment.to_dict() for assignment in self.role_matrix],\n            "requires_role_matrix": self.requires_role_matrix,\n            "signoff_log": [signoff.to_dict() for signoff in self.signoff_log],\n            "requires_signoff_log": self.requires_signoff_log,\n            "blocker_log": [blocker.to_dict() for blocker in self.blocker_log],\n            "requires_blocker_log": self.requires_blocker_log,\n            "blocker_timeline": [event.to_dict() for event in self.blocker_timeline],\n            "requires_blocker_timeline": self.requires_blocker_timeline,\n        }\n\n    @classmethod\n    def from_dict(cls, data: Dict[str, object]) -> "UIReviewPack":\n        return cls(\n            issue_id=str(data["issue_id"]),\n            title=str(data["title"]),\n            version=str(data["version"]),\n            objectives=[ReviewObjective.from_dict(item) for item in data.get("objectives", [])],\n            wireframes=[WireframeSurface.from_dict(item) for item in data.get("wireframes", [])],\n            interactions=[InteractionFlow.from_dict(item) for item in data.get("interactions", [])],\n            open_questions=[OpenQuestion.from_dict(item) for item in data.get("open_questions", [])],\n            reviewer_checklist=[ReviewerChecklistItem.from_dict(item) for item in data.get("reviewer_checklist", [])],\n            requires_reviewer_checklist=bool(data.get("requires_reviewer_checklist", False)),\n            decision_log=[ReviewDecision.from_dict(item) for item in data.get("decision_log", [])],\n            requires_decision_log=bool(data.get("requires_decision_log", False)),\n            role_matrix=[ReviewRoleAssignment.from_dict(item) for item in data.get("role_matrix", [])],\n            requires_role_matrix=bool(data.get("requires_role_matrix", False)),\n            signoff_log=[ReviewSignoff.from_dict(item) for item in data.get("signoff_log", [])],\n            requires_signoff_log=bool(data.get("requires_signoff_log", False)),\n            blocker_log=[ReviewBlocker.from_dict(item) for item in data.get("blocker_log", [])],\n            requires_blocker_log=bool(data.get("requires_blocker_log", False)),\n            blocker_timeline=[ReviewBlockerEvent.from_dict(item) for item in data.get("blocker_timeline", [])],\n            requires_blocker_timeline=bool(data.get("requires_blocker_timeline", False)),\n        )\n\n\n@dataclass(frozen=True)\nclass UIReviewPackAudit:\n    ready: bool\n    objective_count: int\n    wireframe_count: int\n    interaction_count: int\n    open_question_count: int\n    checklist_count: int = 0\n    decision_count: int = 0\n    role_assignment_count: int = 0\n    signoff_count: int = 0\n    blocker_count: int = 0\n    blocker_timeline_count: int = 0\n    missing_sections: List[str] = field(default_factory=list)\n    objectives_missing_signals: List[str] = field(default_factory=list)\n    wireframes_missing_blocks: List[str] = field(default_factory=list)\n    interactions_missing_states: List[str] = field(default_factory=list)\n    unresolved_question_ids: List[str] = field(default_factory=list)\n    wireframes_missing_checklists: List[str] = field(default_factory=list)\n    orphan_checklist_surfaces: List[str] = field(default_factory=list)\n    checklist_items_missing_evidence: List[str] = field(default_factory=list)\n    checklist_items_missing_role_links: List[str] = field(default_factory=list)\n    wireframes_missing_decisions: List[str] = field(default_factory=list)\n    orphan_decision_surfaces: List[str] = field(default_factory=list)\n    unresolved_decision_ids: List[str] = field(default_factory=list)\n    unresolved_decisions_missing_follow_ups: List[str] = field(default_factory=list)\n    wireframes_missing_role_assignments: List[str] = field(default_factory=list)\n    orphan_role_assignment_surfaces: List[str] = field(default_factory=list)\n    role_assignments_missing_responsibilities: List[str] = field(default_factory=list)\n    role_assignments_missing_checklist_links: List[str] = field(default_factory=list)\n    role_assignments_missing_decision_links: List[str] = field(default_factory=list)\n    decisions_missing_role_links: List[str] = field(default_factory=list)\n    wireframes_missing_signoffs: List[str] = field(default_factory=list)\n    orphan_signoff_surfaces: List[str] = field(default_factory=list)\n    signoffs_missing_assignments: List[str] = field(default_factory=list)\n    signoffs_missing_evidence: List[str] = field(default_factory=list)\n    signoffs_missing_requested_dates: List[str] = field(default_factory=list)\n    signoffs_missing_due_dates: List[str] = field(default_factory=list)\n    signoffs_missing_escalation_owners: List[str] = field(default_factory=list)\n    signoffs_missing_reminder_owners: List[str] = field(default_factory=list)\n    signoffs_missing_next_reminders: List[str] = field(default_factory=list)\n    signoffs_missing_reminder_cadence: List[str] = field(default_factory=list)\n    signoffs_with_breached_sla: List[str] = field(default_factory=list)\n    waived_signoffs_missing_metadata: List[str] = field(default_factory=list)\n    unresolved_required_signoff_ids: List[str] = field(default_factory=list)\n    blockers_missing_signoff_links: List[str] = field(default_factory=list)\n    blockers_missing_escalation_owners: List[str] = field(default_factory=list)\n    blockers_missing_next_actions: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_owners: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_until: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_approvers: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_approval_dates: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_renewal_owners: List[str] = field(default_factory=list)\n    freeze_exceptions_missing_renewal_dates: List[str] = field(default_factory=list)\n    blockers_missing_timeline_events: List[str] = field(default_factory=list)\n    closed_blockers_missing_resolution_events: List[str] = field(default_factory=list)\n    orphan_blocker_surfaces: List[str] = field(default_factory=list)\n    orphan_blocker_timeline_blocker_ids: List[str] = field(default_factory=list)\n    handoff_events_missing_targets: List[str] = field(default_factory=list)\n    handoff_events_missing_artifacts: List[str] = field(default_factory=list)\n    handoff_events_missing_ack_owners: List[str] = field(default_factory=list)\n    handoff_events_missing_ack_dates: List[str] = field(default_factory=list)\n    unresolved_required_signoffs_without_blockers: List[str] = field(default_factory=list)\n\n    @property\n    def summary(self) -> str:\n        status = "READY" if self.ready else "HOLD"\n        return (\n            f"{status}: objectives={self.objective_count} "\n            f"wireframes={self.wireframe_count} "\n            f"interactions={self.interaction_count} "\n            f"open_questions={self.open_question_count} "\n            f"checklist={self.checklist_count} "\n            f"decisions={self.decision_count} "\n            f"role_assignments={self.role_assignment_count} "\n            f"signoffs={self.signoff_count} "\n            f"blockers={self.blocker_count} "\n            f"timeline_events={self.blocker_timeline_count}"\n        )\n\n\nclass UIReviewPackAuditor:\n    def audit(self, pack: UIReviewPack) -> UIReviewPackAudit:\n        missing_sections = []\n        if not pack.objectives:\n            missing_sections.append("objectives")\n        if not pack.wireframes:\n            missing_sections.append("wireframes")\n        if not pack.interactions:\n            missing_sections.append("interactions")\n        if not pack.open_questions:\n            missing_sections.append("open_questions")\n\n        objectives_missing_signals = [\n            objective.objective_id\n            for objective in pack.objectives\n            if not objective.success_signal.strip()\n        ]\n        wireframes_missing_blocks = [\n            wireframe.surface_id\n            for wireframe in pack.wireframes\n            if not wireframe.primary_blocks\n        ]\n        interactions_missing_states = [\n            interaction.flow_id\n            for interaction in pack.interactions\n            if not interaction.states\n        ]\n        unresolved_question_ids = [\n            question.question_id\n            for question in pack.open_questions\n            if question.status.lower() != "resolved"\n        ]\n        wireframe_ids = {wireframe.surface_id for wireframe in pack.wireframes}\n\n        checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}\n        for item in pack.reviewer_checklist:\n            checklist_by_surface.setdefault(item.surface_id, []).append(item)\n        wireframes_missing_checklists = []\n        orphan_checklist_surfaces = []\n        checklist_items_missing_evidence = []\n        checklist_items_missing_role_links = []\n        if pack.requires_reviewer_checklist:\n            wireframes_missing_checklists = sorted(\n                wireframe.surface_id\n                for wireframe in pack.wireframes\n                if wireframe.surface_id not in checklist_by_surface\n            )\n            orphan_checklist_surfaces = sorted(\n                surface_id for surface_id in checklist_by_surface if surface_id not in wireframe_ids\n            )\n            checklist_items_missing_evidence = sorted(\n                item.item_id for item in pack.reviewer_checklist if not item.evidence_links\n            )\n\n        decision_by_surface: Dict[str, List[ReviewDecision]] = {}\n        for decision in pack.decision_log:\n            decision_by_surface.setdefault(decision.surface_id, []).append(decision)\n        wireframes_missing_decisions = []\n        orphan_decision_surfaces = []\n        unresolved_decision_ids = []\n        unresolved_decisions_missing_follow_ups = []\n        if pack.requires_decision_log:\n            wireframes_missing_decisions = sorted(\n                wireframe.surface_id\n                for wireframe in pack.wireframes\n                if wireframe.surface_id not in decision_by_surface\n            )\n            orphan_decision_surfaces = sorted(\n                surface_id for surface_id in decision_by_surface if surface_id not in wireframe_ids\n            )\n            unresolved_decision_ids = sorted(\n                decision.decision_id\n                for decision in pack.decision_log\n                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}\n            )\n            unresolved_decisions_missing_follow_ups = sorted(\n                decision.decision_id\n                for decision in pack.decision_log\n                if decision.status.lower() not in {"accepted", "approved", "resolved", "waived"}\n                and not decision.follow_up.strip()\n            )\n\n        checklist_item_ids = {item.item_id for item in pack.reviewer_checklist}\n        decision_ids = {decision.decision_id for decision in pack.decision_log}\n        assignment_ids = {assignment.assignment_id for assignment in pack.role_matrix}\n        role_assignments_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}\n        for assignment in pack.role_matrix:\n            role_assignments_by_surface.setdefault(assignment.surface_id, []).append(assignment)\n        wireframes_missing_role_assignments = []\n        orphan_role_assignment_surfaces = []\n        role_assignments_missing_responsibilities = []\n        role_assignments_missing_checklist_links = []\n        role_assignments_missing_decision_links = []\n        decisions_missing_role_links = []\n        if pack.requires_role_matrix:\n            wireframes_missing_role_assignments = sorted(\n                wireframe.surface_id\n                for wireframe in pack.wireframes\n                if wireframe.surface_id not in role_assignments_by_surface\n            )\n            orphan_role_assignment_surfaces = sorted(\n                surface_id\n                for surface_id in role_assignments_by_surface\n                if surface_id not in wireframe_ids\n            )\n            role_assignments_missing_responsibilities = sorted(\n                assignment.assignment_id\n                for assignment in pack.role_matrix\n                if not assignment.responsibilities\n            )\n            role_assignments_missing_checklist_links = sorted(\n                assignment.assignment_id\n                for assignment in pack.role_matrix\n                if not assignment.checklist_item_ids\n                or any(item_id not in checklist_item_ids for item_id in assignment.checklist_item_ids)\n            )\n            role_assignments_missing_decision_links = sorted(\n                assignment.assignment_id\n                for assignment in pack.role_matrix\n                if not assignment.decision_ids\n                or any(decision_id not in decision_ids for decision_id in assignment.decision_ids)\n            )\n            role_linked_checklist_ids = {\n                item_id\n                for assignment in pack.role_matrix\n                for item_id in assignment.checklist_item_ids\n            }\n            role_linked_decision_ids = {\n                decision_id\n                for assignment in pack.role_matrix\n                for decision_id in assignment.decision_ids\n            }\n            checklist_items_missing_role_links = sorted(\n                item.item_id\n                for item in pack.reviewer_checklist\n                if item.item_id not in role_linked_checklist_ids\n            )\n            decisions_missing_role_links = sorted(\n                decision.decision_id\n                for decision in pack.decision_log\n                if decision.decision_id not in role_linked_decision_ids\n            )\n\n        signoffs_by_surface: Dict[str, List[ReviewSignoff]] = {}\n        for signoff in pack.signoff_log:\n            signoffs_by_surface.setdefault(signoff.surface_id, []).append(signoff)\n        wireframes_missing_signoffs = []\n        orphan_signoff_surfaces = []\n        signoffs_missing_assignments = []\n        signoffs_missing_evidence = []\n        signoffs_missing_requested_dates = []\n        signoffs_missing_due_dates = []\n        signoffs_missing_escalation_owners = []\n        signoffs_missing_reminder_owners = []\n        signoffs_missing_next_reminders = []\n        signoffs_missing_reminder_cadence = []\n        signoffs_with_breached_sla = []\n        waived_signoffs_missing_metadata = []\n        unresolved_required_signoff_ids = []\n        if pack.requires_signoff_log:\n            wireframes_missing_signoffs = sorted(\n                wireframe.surface_id\n                for wireframe in pack.wireframes\n                if wireframe.surface_id not in signoffs_by_surface\n            )\n            orphan_signoff_surfaces = sorted(\n                surface_id for surface_id in signoffs_by_surface if surface_id not in wireframe_ids\n            )\n            signoffs_missing_assignments = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.assignment_id not in assignment_ids\n            )\n            signoffs_missing_evidence = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.status.lower() != "waived" and not signoff.evidence_links\n            )\n            signoffs_missing_requested_dates = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required and not signoff.requested_at.strip()\n            )\n            signoffs_missing_due_dates = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required and not signoff.due_at.strip()\n            )\n            signoffs_missing_escalation_owners = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required and not signoff.escalation_owner.strip()\n            )\n            unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n            signoffs_missing_reminder_owners = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required\n                and signoff.status.lower() not in unresolved_statuses\n                and not signoff.reminder_owner.strip()\n            )\n            signoffs_missing_next_reminders = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required\n                and signoff.status.lower() not in unresolved_statuses\n                and not signoff.next_reminder_at.strip()\n            )\n            signoffs_missing_reminder_cadence = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required\n                and signoff.status.lower() not in unresolved_statuses\n                and not signoff.reminder_cadence.strip()\n            )\n            signoffs_with_breached_sla = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.sla_status.lower() == "breached"\n                and signoff.status.lower() not in {"approved", "accepted", "resolved"}\n            )\n            waived_signoffs_missing_metadata = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.status.lower() == "waived"\n                and (not signoff.waiver_owner.strip() or not signoff.waiver_reason.strip())\n            )\n            unresolved_required_signoff_ids = sorted(\n                signoff.signoff_id\n                for signoff in pack.signoff_log\n                if signoff.required\n                and signoff.status.lower() not in unresolved_statuses\n            )\n\n        blocker_by_signoff: Dict[str, List[ReviewBlocker]] = {}\n        blocker_surfaces = set()\n        for blocker in pack.blocker_log:\n            blocker_surfaces.add(blocker.surface_id)\n            blocker_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)\n        blockers_missing_signoff_links = []\n        blockers_missing_escalation_owners = []\n        blockers_missing_next_actions = []\n        freeze_exceptions_missing_owners = []\n        freeze_exceptions_missing_until = []\n        freeze_exceptions_missing_approvers = []\n        freeze_exceptions_missing_approval_dates = []\n        freeze_exceptions_missing_renewal_owners = []\n        freeze_exceptions_missing_renewal_dates = []\n        orphan_blocker_surfaces = []\n        unresolved_required_signoffs_without_blockers = []\n        if pack.requires_blocker_log:\n            signoff_ids = {signoff.signoff_id for signoff in pack.signoff_log}\n            blockers_missing_signoff_links = sorted(\n                blocker.blocker_id for blocker in pack.blocker_log if blocker.signoff_id not in signoff_ids\n            )\n            blockers_missing_escalation_owners = sorted(\n                blocker.blocker_id for blocker in pack.blocker_log if not blocker.escalation_owner.strip()\n            )\n            blockers_missing_next_actions = sorted(\n                blocker.blocker_id for blocker in pack.blocker_log if not blocker.next_action.strip()\n            )\n            freeze_exceptions_missing_owners = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_owner.strip()\n            )\n            freeze_exceptions_missing_until = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_until.strip()\n            )\n            freeze_exceptions_missing_approvers = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_approved_by.strip()\n            )\n            freeze_exceptions_missing_approval_dates = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_approved_at.strip()\n            )\n            freeze_exceptions_missing_renewal_owners = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_renewal_owner.strip()\n            )\n            freeze_exceptions_missing_renewal_dates = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.freeze_exception and not blocker.freeze_renewal_by.strip()\n            )\n            orphan_blocker_surfaces = sorted(\n                surface_id for surface_id in blocker_surfaces if surface_id not in wireframe_ids\n            )\n            unresolved_required_signoffs_without_blockers = sorted(\n                signoff_id\n                for signoff_id in unresolved_required_signoff_ids\n                if signoff_id not in blocker_by_signoff\n            )\n\n        blocker_timeline_by_blocker: Dict[str, List[ReviewBlockerEvent]] = {}\n        for event in pack.blocker_timeline:\n            blocker_timeline_by_blocker.setdefault(event.blocker_id, []).append(event)\n        blockers_missing_timeline_events = []\n        closed_blockers_missing_resolution_events = []\n        orphan_blocker_timeline_blocker_ids = []\n        handoff_events_missing_targets = []\n        handoff_events_missing_artifacts = []\n        handoff_events_missing_ack_owners = []\n        handoff_events_missing_ack_dates = []\n        if pack.requires_blocker_timeline:\n            blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}\n            orphan_blocker_timeline_blocker_ids = sorted(\n                blocker_id\n                for blocker_id in blocker_timeline_by_blocker\n                if blocker_id not in blocker_ids\n            )\n            blockers_missing_timeline_events = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.status.lower() not in {"resolved", "closed"}\n                and blocker.blocker_id not in blocker_timeline_by_blocker\n            )\n            closed_blockers_missing_resolution_events = sorted(\n                blocker.blocker_id\n                for blocker in pack.blocker_log\n                if blocker.status.lower() in {"resolved", "closed"}\n                and not any(\n                    event.status.lower() in {"resolved", "closed"}\n                    for event in blocker_timeline_by_blocker.get(blocker.blocker_id, [])\n                )\n            )\n            handoff_statuses = {"escalated", "handoff", "reassigned"}\n            handoff_events_missing_targets = sorted(\n                event.event_id\n                for event in pack.blocker_timeline\n                if event.status.lower() in handoff_statuses and not event.handoff_to.strip()\n            )\n            handoff_events_missing_artifacts = sorted(\n                event.event_id\n                for event in pack.blocker_timeline\n                if event.status.lower() in handoff_statuses and not event.artifact_ref.strip()\n            )\n            handoff_events_missing_ack_owners = sorted(\n                event.event_id\n                for event in pack.blocker_timeline\n                if event.status.lower() in handoff_statuses and not event.ack_owner.strip()\n            )\n            handoff_events_missing_ack_dates = sorted(\n                event.event_id\n                for event in pack.blocker_timeline\n                if event.status.lower() in handoff_statuses and not event.ack_at.strip()\n            )\n\n        ready = not (\n            missing_sections\n            or objectives_missing_signals\n            or wireframes_missing_blocks\n            or interactions_missing_states\n            or wireframes_missing_checklists\n            or orphan_checklist_surfaces\n            or checklist_items_missing_evidence\n            or checklist_items_missing_role_links\n            or wireframes_missing_decisions\n            or orphan_decision_surfaces\n            or unresolved_decisions_missing_follow_ups\n            or wireframes_missing_role_assignments\n            or orphan_role_assignment_surfaces\n            or role_assignments_missing_responsibilities\n            or role_assignments_missing_checklist_links\n            or role_assignments_missing_decision_links\n            or decisions_missing_role_links\n            or wireframes_missing_signoffs\n            or orphan_signoff_surfaces\n            or signoffs_missing_assignments\n            or signoffs_missing_evidence\n            or signoffs_missing_requested_dates\n            or signoffs_missing_due_dates\n            or signoffs_missing_escalation_owners\n            or signoffs_missing_reminder_owners\n            or signoffs_missing_next_reminders\n            or signoffs_missing_reminder_cadence\n            or waived_signoffs_missing_metadata\n            or blockers_missing_signoff_links\n            or blockers_missing_escalation_owners\n            or blockers_missing_next_actions\n            or freeze_exceptions_missing_owners\n            or freeze_exceptions_missing_until\n            or freeze_exceptions_missing_approvers\n            or freeze_exceptions_missing_approval_dates\n            or freeze_exceptions_missing_renewal_owners\n            or freeze_exceptions_missing_renewal_dates\n            or blockers_missing_timeline_events\n            or closed_blockers_missing_resolution_events\n            or orphan_blocker_surfaces\n            or orphan_blocker_timeline_blocker_ids\n            or handoff_events_missing_targets\n            or handoff_events_missing_artifacts\n            or handoff_events_missing_ack_owners\n            or handoff_events_missing_ack_dates\n            or unresolved_required_signoffs_without_blockers\n        )\n        return UIReviewPackAudit(\n            ready=ready,\n            objective_count=len(pack.objectives),\n            wireframe_count=len(pack.wireframes),\n            interaction_count=len(pack.interactions),\n            open_question_count=len(pack.open_questions),\n            checklist_count=len(pack.reviewer_checklist),\n            decision_count=len(pack.decision_log),\n            role_assignment_count=len(pack.role_matrix),\n            signoff_count=len(pack.signoff_log),\n            blocker_count=len(pack.blocker_log),\n            blocker_timeline_count=len(pack.blocker_timeline),\n            missing_sections=missing_sections,\n            objectives_missing_signals=objectives_missing_signals,\n            wireframes_missing_blocks=wireframes_missing_blocks,\n            interactions_missing_states=interactions_missing_states,\n            unresolved_question_ids=unresolved_question_ids,\n            wireframes_missing_checklists=wireframes_missing_checklists,\n            orphan_checklist_surfaces=orphan_checklist_surfaces,\n            checklist_items_missing_evidence=checklist_items_missing_evidence,\n            checklist_items_missing_role_links=checklist_items_missing_role_links,\n            wireframes_missing_decisions=wireframes_missing_decisions,\n            orphan_decision_surfaces=orphan_decision_surfaces,\n            unresolved_decision_ids=unresolved_decision_ids,\n            unresolved_decisions_missing_follow_ups=unresolved_decisions_missing_follow_ups,\n            wireframes_missing_role_assignments=wireframes_missing_role_assignments,\n            orphan_role_assignment_surfaces=orphan_role_assignment_surfaces,\n            role_assignments_missing_responsibilities=role_assignments_missing_responsibilities,\n            role_assignments_missing_checklist_links=role_assignments_missing_checklist_links,\n            role_assignments_missing_decision_links=role_assignments_missing_decision_links,\n            decisions_missing_role_links=decisions_missing_role_links,\n            wireframes_missing_signoffs=wireframes_missing_signoffs,\n            orphan_signoff_surfaces=orphan_signoff_surfaces,\n            signoffs_missing_assignments=signoffs_missing_assignments,\n            signoffs_missing_evidence=signoffs_missing_evidence,\n            signoffs_missing_requested_dates=signoffs_missing_requested_dates,\n            signoffs_missing_due_dates=signoffs_missing_due_dates,\n            signoffs_missing_escalation_owners=signoffs_missing_escalation_owners,\n            signoffs_missing_reminder_owners=signoffs_missing_reminder_owners,\n            signoffs_missing_next_reminders=signoffs_missing_next_reminders,\n            signoffs_missing_reminder_cadence=signoffs_missing_reminder_cadence,\n            signoffs_with_breached_sla=signoffs_with_breached_sla,\n            waived_signoffs_missing_metadata=waived_signoffs_missing_metadata,\n            unresolved_required_signoff_ids=unresolved_required_signoff_ids,\n            blockers_missing_signoff_links=blockers_missing_signoff_links,\n            blockers_missing_escalation_owners=blockers_missing_escalation_owners,\n            blockers_missing_next_actions=blockers_missing_next_actions,\n            freeze_exceptions_missing_owners=freeze_exceptions_missing_owners,\n            freeze_exceptions_missing_until=freeze_exceptions_missing_until,\n            freeze_exceptions_missing_approvers=freeze_exceptions_missing_approvers,\n            freeze_exceptions_missing_approval_dates=freeze_exceptions_missing_approval_dates,\n            freeze_exceptions_missing_renewal_owners=freeze_exceptions_missing_renewal_owners,\n            freeze_exceptions_missing_renewal_dates=freeze_exceptions_missing_renewal_dates,\n            blockers_missing_timeline_events=blockers_missing_timeline_events,\n            closed_blockers_missing_resolution_events=closed_blockers_missing_resolution_events,\n            orphan_blocker_surfaces=orphan_blocker_surfaces,\n            orphan_blocker_timeline_blocker_ids=orphan_blocker_timeline_blocker_ids,\n            handoff_events_missing_targets=handoff_events_missing_targets,\n            handoff_events_missing_artifacts=handoff_events_missing_artifacts,\n            handoff_events_missing_ack_owners=handoff_events_missing_ack_owners,\n            handoff_events_missing_ack_dates=handoff_events_missing_ack_dates,\n            unresolved_required_signoffs_without_blockers=unresolved_required_signoffs_without_blockers,\n        )\n\n\ndef _build_blocker_timeline_index(pack: UIReviewPack) -> Dict[str, List[ReviewBlockerEvent]]:\n    timeline_index: Dict[str, List[ReviewBlockerEvent]] = {}\n    for event in sorted(pack.blocker_timeline, key=lambda item: (item.timestamp, item.event_id)):\n        timeline_index.setdefault(event.blocker_id, []).append(event)\n    return timeline_index\n\n\ndef _build_review_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    timeline_index = _build_blocker_timeline_index(pack)\n    entries: List[Dict[str, str]] = []\n    for signoff in pack.signoff_log:\n        if signoff.status.lower() not in {"waived", "deferred"}:\n            continue\n        entries.append(\n            {\n                "exception_id": f"exc-{signoff.signoff_id}",\n                "category": "signoff",\n                "source_id": signoff.signoff_id,\n                "surface_id": signoff.surface_id,\n                "owner": signoff.waiver_owner or signoff.role,\n                "status": signoff.status,\n                "severity": "none",\n                "summary": signoff.waiver_reason or signoff.notes or "none",\n                "evidence": ",".join(signoff.evidence_links) or "none",\n                "latest_event": "none",\n                "next_action": signoff.notes or signoff.waiver_reason or "none",\n            }\n        )\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in {"resolved", "closed"}:\n            continue\n        latest_events = timeline_index.get(blocker.blocker_id, [])\n        latest = latest_events[-1] if latest_events else None\n        latest_label = (\n            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"\n            if latest\n            else "none"\n        )\n        entries.append(\n            {\n                "exception_id": f"exc-{blocker.blocker_id}",\n                "category": "blocker",\n                "source_id": blocker.blocker_id,\n                "surface_id": blocker.surface_id,\n                "owner": blocker.owner,\n                "status": blocker.status,\n                "severity": blocker.severity,\n                "summary": blocker.summary,\n                "evidence": blocker.escalation_owner or "none",\n                "latest_event": latest_label,\n                "next_action": blocker.next_action or "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["owner"], item["surface_id"], item["category"], item["source_id"]),\n    )\n\n\ndef _build_freeze_exception_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    timeline_index = _build_blocker_timeline_index(pack)\n    entries: List[Dict[str, str]] = []\n    for signoff in pack.signoff_log:\n        if signoff.status.lower() not in {"waived", "deferred"}:\n            continue\n        entries.append(\n            {\n                "entry_id": f"freeze-{signoff.signoff_id}",\n                "item_type": "signoff",\n                "source_id": signoff.signoff_id,\n                "surface_id": signoff.surface_id,\n                "owner": signoff.waiver_owner or signoff.role,\n                "status": signoff.status,\n                "window": "none",\n                "summary": signoff.waiver_reason or signoff.notes or "none",\n                "evidence": ",".join(signoff.evidence_links) or "none",\n                "next_action": signoff.notes or signoff.waiver_reason or "none",\n            }\n        )\n    for blocker in pack.blocker_log:\n        if not blocker.freeze_exception:\n            continue\n        latest_events = timeline_index.get(blocker.blocker_id, [])\n        latest = latest_events[-1] if latest_events else None\n        latest_label = (\n            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"\n            if latest\n            else "none"\n        )\n        entries.append(\n            {\n                "entry_id": f"freeze-{blocker.blocker_id}",\n                "item_type": "blocker",\n                "source_id": blocker.blocker_id,\n                "surface_id": blocker.surface_id,\n                "owner": blocker.freeze_owner or blocker.owner,\n                "status": blocker.status,\n                "window": blocker.freeze_until or "none",\n                "summary": blocker.freeze_reason or blocker.summary,\n                "evidence": latest_label,\n                "next_action": blocker.next_action or "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["owner"], item["surface_id"], item["item_type"], item["source_id"]),\n    )\n\n\ndef _build_signoff_breach_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    blocker_index: Dict[str, List[str]] = {}\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in {"resolved", "closed"}:\n            continue\n        blocker_index.setdefault(blocker.signoff_id, []).append(blocker.blocker_id)\n    entries = [\n        {\n            "entry_id": f"breach-{signoff.signoff_id}",\n            "signoff_id": signoff.signoff_id,\n            "surface_id": signoff.surface_id,\n            "role": signoff.role,\n            "status": signoff.status,\n            "sla_status": signoff.sla_status,\n            "requested_at": signoff.requested_at or "none",\n            "due_at": signoff.due_at or "none",\n            "escalation_owner": signoff.escalation_owner or "none",\n            "linked_blockers": ",".join(sorted(blocker_index.get(signoff.signoff_id, []))) or "none",\n            "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n        }\n        for signoff in pack.signoff_log\n        if signoff.sla_status.lower() in {"at-risk", "breached"}\n        and signoff.status.lower() not in {"approved", "accepted", "resolved", "waived", "deferred"}\n    ]\n    return sorted(\n        entries,\n        key=lambda item: (item["due_at"], item["sla_status"], item["escalation_owner"], item["signoff_id"]),\n    )\n\n\ndef _build_escalation_handoff_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}\n    handoff_statuses = {"escalated", "handoff", "reassigned"}\n    entries: List[Dict[str, str]] = []\n    for event in pack.blocker_timeline:\n        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():\n            continue\n        blocker = blocker_index.get(event.blocker_id)\n        entries.append(\n            {\n                "ledger_id": f"handoff-{event.event_id}",\n                "event_id": event.event_id,\n                "blocker_id": event.blocker_id,\n                "surface_id": blocker.surface_id if blocker else "none",\n                "actor": event.actor,\n                "status": event.status,\n                "handoff_from": event.handoff_from or (blocker.owner if blocker else "none"),\n                "handoff_to": event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none",\n                "channel": event.channel or "none",\n                "artifact_ref": event.artifact_ref or "none",\n                "timestamp": event.timestamp,\n                "summary": event.summary,\n                "next_action": event.next_action or "none",\n            }\n        )\n    return sorted(entries, key=lambda item: (item["timestamp"], item["event_id"]))\n\n\ndef _build_handoff_ack_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    blocker_index = {blocker.blocker_id: blocker for blocker in pack.blocker_log}\n    handoff_statuses = {"escalated", "handoff", "reassigned"}\n    entries: List[Dict[str, str]] = []\n    for event in pack.blocker_timeline:\n        if event.status.lower() not in handoff_statuses and not event.handoff_to.strip():\n            continue\n        blocker = blocker_index.get(event.blocker_id)\n        fallback_owner = event.handoff_to or (blocker.escalation_owner if blocker else "none") or "none"\n        entries.append(\n            {\n                "entry_id": f"ack-{event.event_id}",\n                "event_id": event.event_id,\n                "blocker_id": event.blocker_id,\n                "surface_id": blocker.surface_id if blocker else "none",\n                "actor": event.actor,\n                "status": event.status,\n                "handoff_to": event.handoff_to or fallback_owner,\n                "ack_owner": event.ack_owner or fallback_owner,\n                "ack_status": event.ack_status or "pending",\n                "ack_at": event.ack_at or "none",\n                "channel": event.channel or "none",\n                "artifact_ref": event.artifact_ref or "none",\n                "summary": event.summary,\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["ack_status"], item["ack_owner"], item["event_id"]),\n    )\n\n\ndef _build_signoff_reminder_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n    entries = [\n        {\n            "entry_id": f"rem-{signoff.signoff_id}",\n            "signoff_id": signoff.signoff_id,\n            "surface_id": signoff.surface_id,\n            "role": signoff.role,\n            "status": signoff.status,\n            "sla_status": signoff.sla_status,\n            "reminder_owner": signoff.reminder_owner or "none",\n            "reminder_channel": signoff.reminder_channel or "none",\n            "last_reminder_at": signoff.last_reminder_at or "none",\n            "next_reminder_at": signoff.next_reminder_at or "none",\n            "due_at": signoff.due_at or "none",\n            "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n        }\n        for signoff in pack.signoff_log\n        if signoff.required and signoff.status.lower() not in unresolved_statuses\n    ]\n    return sorted(\n        entries,\n        key=lambda item: (item["next_reminder_at"], item["reminder_owner"], item["signoff_id"]),\n    )\n\n\ndef _build_reminder_cadence_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    unresolved_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n    entries = [\n        {\n            "entry_id": f"cad-rem-{signoff.signoff_id}",\n            "signoff_id": signoff.signoff_id,\n            "surface_id": signoff.surface_id,\n            "role": signoff.role,\n            "status": signoff.status,\n            "sla_status": signoff.sla_status,\n            "reminder_owner": signoff.reminder_owner or "none",\n            "reminder_cadence": signoff.reminder_cadence or "none",\n            "reminder_status": signoff.reminder_status or "scheduled",\n            "last_reminder_at": signoff.last_reminder_at or "none",\n            "next_reminder_at": signoff.next_reminder_at or "none",\n            "due_at": signoff.due_at or "none",\n            "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n        }\n        for signoff in pack.signoff_log\n        if signoff.required and signoff.status.lower() not in unresolved_statuses\n    ]\n    return sorted(\n        entries,\n        key=lambda item: (item["reminder_cadence"], item["reminder_status"], item["signoff_id"]),\n    )\n\n\ndef _build_freeze_approval_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    timeline_index = _build_blocker_timeline_index(pack)\n    entries: List[Dict[str, str]] = []\n    for blocker in pack.blocker_log:\n        if not blocker.freeze_exception:\n            continue\n        latest_events = timeline_index.get(blocker.blocker_id, [])\n        latest = latest_events[-1] if latest_events else None\n        latest_label = (\n            f"{latest.event_id}/{latest.status}/{latest.actor}@{latest.timestamp}"\n            if latest\n            else "none"\n        )\n        entries.append(\n            {\n                "entry_id": f"freeze-approval-{blocker.blocker_id}",\n                "blocker_id": blocker.blocker_id,\n                "surface_id": blocker.surface_id,\n                "status": blocker.status,\n                "freeze_owner": blocker.freeze_owner or blocker.owner,\n                "freeze_until": blocker.freeze_until or "none",\n                "freeze_approved_by": blocker.freeze_approved_by or "none",\n                "freeze_approved_at": blocker.freeze_approved_at or "none",\n                "summary": blocker.freeze_reason or blocker.summary,\n                "latest_event": latest_label,\n                "next_action": blocker.next_action or "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["freeze_approved_at"], item["freeze_until"], item["blocker_id"]),\n    )\n\n\ndef _build_freeze_renewal_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries = [\n        {\n            "entry_id": f"renew-{blocker.blocker_id}",\n            "blocker_id": blocker.blocker_id,\n            "surface_id": blocker.surface_id,\n            "status": blocker.status,\n            "freeze_owner": blocker.freeze_owner or blocker.owner,\n            "freeze_until": blocker.freeze_until or "none",\n            "renewal_owner": blocker.freeze_renewal_owner or "none",\n            "renewal_by": blocker.freeze_renewal_by or "none",\n            "renewal_status": blocker.freeze_renewal_status or "not-needed",\n            "freeze_approved_by": blocker.freeze_approved_by or "none",\n            "summary": blocker.freeze_reason or blocker.summary,\n            "next_action": blocker.next_action or "none",\n        }\n        for blocker in pack.blocker_log\n        if blocker.freeze_exception\n    ]\n    return sorted(\n        entries,\n        key=lambda item: (item["renewal_by"], item["renewal_owner"], item["blocker_id"]),\n    )\n\n\ndef _build_owner_escalation_digest_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries: List[Dict[str, str]] = []\n    for entry in _build_escalation_dashboard_entries(pack):\n        entries.append(\n            {\n                "digest_id": f"digest-{entry[\'escalation_id\']}",\n                "owner": entry["escalation_owner"],\n                "item_type": entry["item_type"],\n                "source_id": entry["source_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "summary": entry["summary"],\n                "detail": entry["priority"],\n            }\n        )\n    for entry in _build_signoff_reminder_entries(pack):\n        entries.append(\n            {\n                "digest_id": f"digest-{entry[\'entry_id\']}",\n                "owner": entry["reminder_owner"],\n                "item_type": "reminder",\n                "source_id": entry["signoff_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "summary": entry["summary"],\n                "detail": entry["next_reminder_at"],\n            }\n        )\n    for entry in _build_freeze_approval_entries(pack):\n        entries.append(\n            {\n                "digest_id": f"digest-{entry[\'entry_id\']}",\n                "owner": entry["freeze_approved_by"],\n                "item_type": "freeze",\n                "source_id": entry["blocker_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "summary": entry["summary"],\n                "detail": entry["freeze_until"],\n            }\n        )\n    for entry in _build_escalation_handoff_entries(pack):\n        entries.append(\n            {\n                "digest_id": f"digest-{entry[\'ledger_id\']}",\n                "owner": entry["handoff_to"],\n                "item_type": "handoff",\n                "source_id": entry["blocker_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "summary": entry["summary"],\n                "detail": entry["timestamp"],\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),\n    )\n\n\ndef _build_owner_review_queue_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries: List[Dict[str, str]] = []\n    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}\n    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}\n    signoff_ready_statuses = {"approved", "accepted", "resolved"}\n    blocker_done_statuses = {"resolved", "closed"}\n\n    for item in pack.reviewer_checklist:\n        if item.status.lower() in checklist_ready_statuses:\n            continue\n        entries.append(\n            {\n                "queue_id": f"queue-{item.item_id}",\n                "owner": item.owner,\n                "item_type": "checklist",\n                "source_id": item.item_id,\n                "surface_id": item.surface_id,\n                "status": item.status,\n                "summary": item.prompt,\n                "next_action": item.notes or ",".join(item.evidence_links) or "none",\n            }\n        )\n    for decision in pack.decision_log:\n        if decision.status.lower() in decision_ready_statuses:\n            continue\n        entries.append(\n            {\n                "queue_id": f"queue-{decision.decision_id}",\n                "owner": decision.owner,\n                "item_type": "decision",\n                "source_id": decision.decision_id,\n                "surface_id": decision.surface_id,\n                "status": decision.status,\n                "summary": decision.summary,\n                "next_action": decision.follow_up or decision.rationale,\n            }\n        )\n    for signoff in pack.signoff_log:\n        if signoff.status.lower() in signoff_ready_statuses:\n            continue\n        entries.append(\n            {\n                "queue_id": f"queue-{signoff.signoff_id}",\n                "owner": signoff.waiver_owner or signoff.role,\n                "item_type": "signoff",\n                "source_id": signoff.signoff_id,\n                "surface_id": signoff.surface_id,\n                "status": signoff.status,\n                "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n                "next_action": signoff.waiver_reason or signoff.notes or signoff.due_at or ",".join(signoff.evidence_links) or "none",\n            }\n        )\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in blocker_done_statuses:\n            continue\n        entries.append(\n            {\n                "queue_id": f"queue-{blocker.blocker_id}",\n                "owner": blocker.owner,\n                "item_type": "blocker",\n                "source_id": blocker.blocker_id,\n                "surface_id": blocker.surface_id,\n                "status": blocker.status,\n                "summary": blocker.summary,\n                "next_action": blocker.next_action or "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]),\n    )\n\n\ndef _build_checklist_traceability_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    assignments_by_item: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        for item_id in assignment.checklist_item_ids:\n            assignments_by_item.setdefault(item_id, []).append(assignment)\n    entries: List[Dict[str, str]] = []\n    for item in pack.reviewer_checklist:\n        linked_assignments = assignments_by_item.get(item.item_id, [])\n        linked_decisions = sorted(\n            {decision_id for assignment in linked_assignments for decision_id in assignment.decision_ids}\n        )\n        entries.append(\n            {\n                "entry_id": f"trace-{item.item_id}",\n                "item_id": item.item_id,\n                "surface_id": item.surface_id,\n                "owner": item.owner,\n                "status": item.status,\n                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",\n                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",\n                "linked_decisions": ",".join(linked_decisions) or "none",\n                "evidence": ",".join(item.evidence_links) or "none",\n                "summary": item.notes or item.prompt,\n            }\n        )\n    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["item_id"]))\n\n\ndef _build_decision_followup_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    assignments_by_decision: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        for decision_id in assignment.decision_ids:\n            assignments_by_decision.setdefault(decision_id, []).append(assignment)\n    entries: List[Dict[str, str]] = []\n    for decision in pack.decision_log:\n        linked_assignments = assignments_by_decision.get(decision.decision_id, [])\n        linked_checklist_ids = sorted(\n            {item_id for assignment in linked_assignments for item_id in assignment.checklist_item_ids}\n        )\n        entries.append(\n            {\n                "entry_id": f"follow-{decision.decision_id}",\n                "decision_id": decision.decision_id,\n                "surface_id": decision.surface_id,\n                "owner": decision.owner,\n                "status": decision.status,\n                "linked_roles": ",".join(assignment.role for assignment in linked_assignments) or "none",\n                "linked_assignments": ",".join(assignment.assignment_id for assignment in linked_assignments) or "none",\n                "linked_checklists": ",".join(linked_checklist_ids) or "none",\n                "follow_up": decision.follow_up or "none",\n                "summary": decision.summary,\n            }\n        )\n    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["decision_id"]))\n\n\ndef _build_objective_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}\n    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}\n    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}\n    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        assignments_by_role.setdefault(assignment.role, []).append(assignment)\n    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}\n    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in {"resolved", "closed"}:\n            continue\n        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)\n    checklist_index = {item.item_id: item for item in pack.reviewer_checklist}\n    decision_index = {decision.decision_id: decision for decision in pack.decision_log}\n    status_priority = {"blocked": 0, "at-risk": 1, "covered": 2}\n    entries: List[Dict[str, str]] = []\n    for objective in pack.objectives:\n        assignments = assignments_by_role.get(objective.persona, [])\n        checklist_ids = sorted(\n            {item_id for assignment in assignments for item_id in assignment.checklist_item_ids}\n        )\n        decision_ids = sorted(\n            {decision_id for assignment in assignments for decision_id in assignment.decision_ids}\n        )\n        signoffs = [\n            signoff_by_assignment[assignment.assignment_id]\n            for assignment in assignments\n            if assignment.assignment_id in signoff_by_assignment\n        ]\n        blocker_ids = sorted(\n            {\n                blocker.blocker_id\n                for signoff in signoffs\n                for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])\n            }\n        )\n        open_checklist = sum(\n            1\n            for item_id in checklist_ids\n            if checklist_index[item_id].status.lower() not in checklist_ready_statuses\n        )\n        open_decisions = sum(\n            1\n            for decision_id in decision_ids\n            if decision_index[decision_id].status.lower() not in decision_ready_statuses\n        )\n        open_assignments = sum(\n            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses\n        )\n        open_signoffs = sum(\n            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses\n        )\n        coverage_status = (\n            "blocked"\n            if blocker_ids\n            else "at-risk"\n            if open_checklist or open_decisions or open_assignments or open_signoffs\n            else "covered"\n        )\n        entries.append(\n            {\n                "entry_id": f"objcov-{objective.objective_id}",\n                "objective_id": objective.objective_id,\n                "persona": objective.persona,\n                "priority": objective.priority,\n                "coverage_status": coverage_status,\n                "dependency_count": str(len(objective.dependencies)),\n                "dependency_ids": ",".join(objective.dependencies) or "none",\n                "surface_ids": ",".join(sorted({assignment.surface_id for assignment in assignments})) or "none",\n                "assignment_ids": ",".join(assignment.assignment_id for assignment in assignments) or "none",\n                "checklist_ids": ",".join(checklist_ids) or "none",\n                "decision_ids": ",".join(decision_ids) or "none",\n                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",\n                "blocker_ids": ",".join(blocker_ids) or "none",\n                "summary": objective.success_signal or objective.outcome,\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (status_priority[item["coverage_status"]], item["persona"], item["objective_id"]),\n    )\n\n\ndef _build_wireframe_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}\n    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}\n    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}\n    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n    blocker_done_statuses = {"resolved", "closed"}\n    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}\n    for item in pack.reviewer_checklist:\n        checklist_by_surface.setdefault(item.surface_id, []).append(item)\n    decision_by_surface: Dict[str, List[ReviewDecision]] = {}\n    for decision in pack.decision_log:\n        decision_by_surface.setdefault(decision.surface_id, []).append(decision)\n    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)\n    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}\n    for signoff in pack.signoff_log:\n        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)\n    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}\n    for blocker in pack.blocker_log:\n        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)\n    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}\n    entries: List[Dict[str, str]] = []\n    for wireframe in pack.wireframes:\n        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])\n        decisions = decision_by_surface.get(wireframe.surface_id, [])\n        assignments = assignment_by_surface.get(wireframe.surface_id, [])\n        signoffs = signoff_by_surface.get(wireframe.surface_id, [])\n        blockers = [\n            blocker\n            for blocker in blocker_by_surface.get(wireframe.surface_id, [])\n            if blocker.status.lower() not in blocker_done_statuses\n        ]\n        checklist_open = sum(\n            1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses\n        )\n        decisions_open = sum(\n            1 for decision in decisions if decision.status.lower() not in decision_ready_statuses\n        )\n        assignments_open = sum(\n            1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses\n        )\n        signoffs_open = sum(\n            1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses\n        )\n        blockers_open = len(blockers)\n        open_total = (\n            checklist_open + decisions_open + assignments_open + signoffs_open + blockers_open\n        )\n        readiness_status = (\n            "blocked"\n            if blockers_open\n            else "at-risk"\n            if checklist_open or decisions_open or assignments_open or signoffs_open\n            else "ready"\n        )\n        entries.append(\n            {\n                "entry_id": f"wire-{wireframe.surface_id}",\n                "surface_id": wireframe.surface_id,\n                "device": wireframe.device,\n                "entry_point": wireframe.entry_point,\n                "readiness_status": readiness_status,\n                "open_total": str(open_total),\n                "checklist_open": str(checklist_open),\n                "decisions_open": str(decisions_open),\n                "assignments_open": str(assignments_open),\n                "signoffs_open": str(signoffs_open),\n                "blockers_open": str(blockers_open),\n                "block_count": str(len(wireframe.primary_blocks)),\n                "note_count": str(len(wireframe.review_notes)),\n                "signoff_ids": ",".join(signoff.signoff_id for signoff in signoffs) or "none",\n                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",\n                "summary": wireframe.name,\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (status_priority[item["readiness_status"]], item["surface_id"]),\n    )\n\n\ndef _build_open_question_tracker_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries: List[Dict[str, str]] = []\n    for question in pack.open_questions:\n        linked_items = [\n            item for item in pack.reviewer_checklist if question.question_id in item.evidence_links\n        ]\n        flow_ids = sorted(\n            {\n                evidence_link\n                for item in linked_items\n                for evidence_link in item.evidence_links\n                if evidence_link.startswith("flow-")\n            }\n        )\n        entries.append(\n            {\n                "entry_id": f"qtrack-{question.question_id}",\n                "question_id": question.question_id,\n                "owner": question.owner,\n                "theme": question.theme,\n                "status": question.status,\n                "link_status": "linked" if linked_items else "orphan",\n                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",\n                "checklist_ids": ",".join(item.item_id for item in linked_items) or "none",\n                "flow_ids": ",".join(flow_ids) or "none",\n                "summary": question.question,\n                "impact": question.impact,\n            }\n        )\n    return sorted(entries, key=lambda item: (item["status"], item["owner"], item["question_id"]))\n\n\ndef _build_interaction_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}\n    checklist_by_flow: Dict[str, List[ReviewerChecklistItem]] = {}\n    for item in pack.reviewer_checklist:\n        for evidence_link in item.evidence_links:\n            if evidence_link.startswith("flow-"):\n                checklist_by_flow.setdefault(evidence_link, []).append(item)\n    status_priority = {"missing": 0, "watch": 1, "covered": 2}\n    entries: List[Dict[str, str]] = []\n    for interaction in pack.interactions:\n        linked_items = checklist_by_flow.get(interaction.flow_id, [])\n        checklist_ids = list(dict.fromkeys(item.item_id for item in linked_items))\n        open_checklist_ids = list(\n            dict.fromkeys(\n                item.item_id\n                for item in linked_items\n                if item.status.lower() not in checklist_ready_statuses\n            )\n        )\n        coverage_status = (\n            "missing"\n            if not checklist_ids\n            else "watch"\n            if open_checklist_ids\n            else "covered"\n        )\n        entries.append(\n            {\n                "entry_id": f"intcov-{interaction.flow_id}",\n                "flow_id": interaction.flow_id,\n                "surface_ids": ",".join(sorted({item.surface_id for item in linked_items})) or "none",\n                "owners": ",".join(sorted({item.owner for item in linked_items})) or "none",\n                "checklist_ids": ",".join(checklist_ids) or "none",\n                "open_checklist_ids": ",".join(open_checklist_ids) or "none",\n                "coverage_status": coverage_status,\n                "state_count": str(len(interaction.states)),\n                "exception_count": str(len(interaction.exceptions)),\n                "summary": interaction.trigger,\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (status_priority[item["coverage_status"]], item["flow_id"]),\n    )\n\n\ndef _build_persona_readiness_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    objective_entries = _build_objective_coverage_entries(pack)\n    objective_entries_by_persona: Dict[str, List[Dict[str, str]]] = {}\n    for entry in objective_entries:\n        objective_entries_by_persona.setdefault(entry["persona"], []).append(entry)\n    assignments_by_role: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        assignments_by_role.setdefault(assignment.role, []).append(assignment)\n    signoff_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}\n    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in {"resolved", "closed"}:\n            continue\n        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)\n    questions_by_owner: Dict[str, List[OpenQuestion]] = {}\n    for question in pack.open_questions:\n        questions_by_owner.setdefault(question.owner, []).append(question)\n    queue_entries = _build_owner_review_queue_entries(pack)\n    status_priority = {"blocked": 0, "at-risk": 1, "ready": 2}\n    entries: List[Dict[str, str]] = []\n    for persona, persona_objectives in objective_entries_by_persona.items():\n        surface_ids = sorted(\n            {\n                surface_id\n                for entry in persona_objectives\n                for surface_id in entry["surface_ids"].split(",")\n                if surface_id and surface_id != "none"\n            }\n        )\n        assignments = assignments_by_role.get(persona, [])\n        signoffs = [\n            signoff_by_assignment[assignment.assignment_id]\n            for assignment in assignments\n            if assignment.assignment_id in signoff_by_assignment\n        ]\n        blockers = [\n            blocker\n            for signoff in signoffs\n            for blocker in unresolved_blockers_by_signoff.get(signoff.signoff_id, [])\n        ]\n        blocker_ids = sorted({blocker.blocker_id for blocker in blockers})\n        questions = questions_by_owner.get(persona, [])\n        queue_items = [\n            entry\n            for entry in queue_entries\n            if entry["owner"] == persona\n            and (not surface_ids or entry["surface_id"] in surface_ids)\n        ]\n        objective_statuses = {entry["coverage_status"] for entry in persona_objectives}\n        readiness = (\n            "blocked"\n            if "blocked" in objective_statuses or blocker_ids\n            else "at-risk"\n            if "at-risk" in objective_statuses or questions or queue_items\n            else "ready"\n        )\n        entries.append(\n            {\n                "entry_id": f"persona-{persona.lower().replace(\' \', \'-\')}",\n                "persona": persona,\n                "readiness": readiness,\n                "objective_count": str(len(persona_objectives)),\n                "assignment_count": str(len(assignments)),\n                "signoff_count": str(len(signoffs)),\n                "question_count": str(len(questions)),\n                "queue_count": str(len(queue_items)),\n                "blocker_count": str(len(blocker_ids)),\n                "objective_ids": ",".join(\n                    sorted(entry["objective_id"] for entry in persona_objectives)\n                )\n                or "none",\n                "surface_ids": ",".join(surface_ids) or "none",\n                "queue_ids": ",".join(sorted(entry["queue_id"] for entry in queue_items)) or "none",\n                "blocker_ids": ",".join(blocker_ids) or "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (status_priority[item["readiness"]], item["persona"]),\n    )\n\n\ndef _build_review_summary_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    objective_entries = _build_objective_coverage_entries(pack)\n    objective_status_counts: Dict[str, int] = {}\n    for entry in objective_entries:\n        objective_status_counts[entry["coverage_status"]] = (\n            objective_status_counts.get(entry["coverage_status"], 0) + 1\n        )\n\n    persona_entries = _build_persona_readiness_entries(pack)\n    persona_status_counts: Dict[str, int] = {}\n    for entry in persona_entries:\n        persona_status_counts[entry["readiness"]] = persona_status_counts.get(entry["readiness"], 0) + 1\n\n    wireframe_entries = _build_wireframe_readiness_entries(pack)\n    wireframe_status_counts: Dict[str, int] = {}\n    for entry in wireframe_entries:\n        wireframe_status_counts[entry["readiness_status"]] = (\n            wireframe_status_counts.get(entry["readiness_status"], 0) + 1\n        )\n\n    interaction_entries = _build_interaction_coverage_entries(pack)\n    interaction_status_counts: Dict[str, int] = {}\n    for entry in interaction_entries:\n        interaction_status_counts[entry["coverage_status"]] = (\n            interaction_status_counts.get(entry["coverage_status"], 0) + 1\n        )\n\n    question_entries = _build_open_question_tracker_entries(pack)\n    question_link_counts: Dict[str, int] = {}\n    question_owners = {entry["owner"] for entry in question_entries}\n    for entry in question_entries:\n        question_link_counts[entry["link_status"]] = question_link_counts.get(entry["link_status"], 0) + 1\n\n    action_entries = _build_owner_workload_entries(pack)\n    action_lane_counts: Dict[str, int] = {}\n    for entry in action_entries:\n        action_lane_counts[entry["lane"]] = action_lane_counts.get(entry["lane"], 0) + 1\n\n    return [\n        {\n            "entry_id": "summary-objectives",\n            "category": "objectives",\n            "total": str(len(objective_entries)),\n            "metrics": (\n                f"blocked={objective_status_counts.get(\'blocked\', 0)} "\n                f"at-risk={objective_status_counts.get(\'at-risk\', 0)} "\n                f"covered={objective_status_counts.get(\'covered\', 0)}"\n            ),\n        },\n        {\n            "entry_id": "summary-personas",\n            "category": "personas",\n            "total": str(len(persona_entries)),\n            "metrics": (\n                f"blocked={persona_status_counts.get(\'blocked\', 0)} "\n                f"at-risk={persona_status_counts.get(\'at-risk\', 0)} "\n                f"ready={persona_status_counts.get(\'ready\', 0)}"\n            ),\n        },\n        {\n            "entry_id": "summary-wireframes",\n            "category": "wireframes",\n            "total": str(len(wireframe_entries)),\n            "metrics": (\n                f"blocked={wireframe_status_counts.get(\'blocked\', 0)} "\n                f"at-risk={wireframe_status_counts.get(\'at-risk\', 0)} "\n                f"ready={wireframe_status_counts.get(\'ready\', 0)}"\n            ),\n        },\n        {\n            "entry_id": "summary-interactions",\n            "category": "interactions",\n            "total": str(len(interaction_entries)),\n            "metrics": (\n                f"covered={interaction_status_counts.get(\'covered\', 0)} "\n                f"watch={interaction_status_counts.get(\'watch\', 0)} "\n                f"missing={interaction_status_counts.get(\'missing\', 0)}"\n            ),\n        },\n        {\n            "entry_id": "summary-questions",\n            "category": "questions",\n            "total": str(len(question_entries)),\n            "metrics": (\n                f"linked={question_link_counts.get(\'linked\', 0)} "\n                f"orphan={question_link_counts.get(\'orphan\', 0)} "\n                f"owners={len(question_owners)}"\n            ),\n        },\n        {\n            "entry_id": "summary-actions",\n            "category": "actions",\n            "total": str(len(action_entries)),\n            "metrics": (\n                f"queue={action_lane_counts.get(\'queue\', 0)} "\n                f"reminder={action_lane_counts.get(\'reminder\', 0)} "\n                f"renewal={action_lane_counts.get(\'renewal\', 0)}"\n            ),\n        },\n    ]\n\n\ndef _build_role_coverage_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    signoffs_by_assignment = {signoff.assignment_id: signoff for signoff in pack.signoff_log}\n    entries = [\n        {\n            "entry_id": f"cover-{assignment.assignment_id}",\n            "assignment_id": assignment.assignment_id,\n            "surface_id": assignment.surface_id,\n            "role": assignment.role,\n            "status": assignment.status,\n            "responsibility_count": str(len(assignment.responsibilities)),\n            "checklist_count": str(len(assignment.checklist_item_ids)),\n            "decision_count": str(len(assignment.decision_ids)),\n            "signoff_id": signoffs_by_assignment.get(assignment.assignment_id).signoff_id if assignment.assignment_id in signoffs_by_assignment else "none",\n            "signoff_status": signoffs_by_assignment.get(assignment.assignment_id).status if assignment.assignment_id in signoffs_by_assignment else "none",\n            "summary": ",".join(assignment.responsibilities) or "none",\n        }\n        for assignment in pack.role_matrix\n    ]\n    return sorted(entries, key=lambda item: (item["surface_id"], item["status"], item["assignment_id"]))\n\n\ndef _build_owner_workload_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries: List[Dict[str, str]] = []\n    for entry in _build_owner_review_queue_entries(pack):\n        entries.append(\n            {\n                "entry_id": f"load-{entry[\'queue_id\']}",\n                "owner": entry["owner"],\n                "item_type": entry["item_type"],\n                "source_id": entry["source_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "lane": "queue",\n                "detail": entry["next_action"],\n                "summary": entry["summary"],\n            }\n        )\n    for entry in _build_signoff_reminder_entries(pack):\n        entries.append(\n            {\n                "entry_id": f"load-{entry[\'entry_id\']}",\n                "owner": entry["reminder_owner"],\n                "item_type": "reminder",\n                "source_id": entry["signoff_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["status"],\n                "lane": "reminder",\n                "detail": entry["next_reminder_at"],\n                "summary": entry["summary"],\n            }\n        )\n    for entry in _build_freeze_renewal_entries(pack):\n        if entry["renewal_status"] == "not-needed":\n            continue\n        entries.append(\n            {\n                "entry_id": f"load-{entry[\'entry_id\']}",\n                "owner": entry["renewal_owner"],\n                "item_type": "renewal",\n                "source_id": entry["blocker_id"],\n                "surface_id": entry["surface_id"],\n                "status": entry["renewal_status"],\n                "lane": "renewal",\n                "detail": entry["renewal_by"],\n                "summary": entry["summary"],\n            }\n        )\n    return sorted(entries, key=lambda item: (item["owner"], item["item_type"], item["surface_id"], item["source_id"]))\n\n\ndef _build_signoff_dependency_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    assignment_by_id = {assignment.assignment_id: assignment for assignment in pack.role_matrix}\n    timeline_index = _build_blocker_timeline_index(pack)\n    unresolved_blockers_by_signoff: Dict[str, List[ReviewBlocker]] = {}\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in {"resolved", "closed"}:\n            continue\n        unresolved_blockers_by_signoff.setdefault(blocker.signoff_id, []).append(blocker)\n    entries: List[Dict[str, str]] = []\n    for signoff in pack.signoff_log:\n        assignment = assignment_by_id.get(signoff.assignment_id)\n        blockers = unresolved_blockers_by_signoff.get(signoff.signoff_id, [])\n        latest_event = None\n        for blocker in blockers:\n            events = timeline_index.get(blocker.blocker_id, [])\n            if events:\n                candidate = events[-1]\n                if latest_event is None or (candidate.timestamp, candidate.event_id) > (latest_event.timestamp, latest_event.event_id):\n                    latest_event = candidate\n        latest_label = (\n            f"{latest_event.event_id}/{latest_event.status}/{latest_event.actor}@{latest_event.timestamp}"\n            if latest_event\n            else "none"\n        )\n        entries.append(\n            {\n                "entry_id": f"dep-{signoff.signoff_id}",\n                "signoff_id": signoff.signoff_id,\n                "surface_id": signoff.surface_id,\n                "role": signoff.role,\n                "status": signoff.status,\n                "assignment_id": signoff.assignment_id,\n                "dependency_status": "blocked" if blockers else "clear",\n                "checklist_ids": ",".join(assignment.checklist_item_ids) if assignment else "none",\n                "decision_ids": ",".join(assignment.decision_ids) if assignment else "none",\n                "blocker_ids": ",".join(blocker.blocker_id for blocker in blockers) or "none",\n                "blocker_owners": ",".join(sorted({blocker.owner for blocker in blockers})) or "none",\n                "latest_blocker_event": latest_label,\n                "sla_status": signoff.sla_status,\n                "due_at": signoff.due_at or "none",\n                "reminder_cadence": signoff.reminder_cadence or "none",\n                "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n            }\n        )\n    return sorted(entries, key=lambda item: (item["dependency_status"], item["due_at"], item["signoff_id"]))\n\n\ndef _build_audit_density_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    checklist_ready_statuses = {"ready", "approved", "accepted", "resolved", "done"}\n    decision_ready_statuses = {"accepted", "approved", "resolved", "waived"}\n    role_ready_statuses = {"ready", "approved", "accepted", "resolved"}\n    signoff_ready_statuses = {"approved", "accepted", "resolved", "waived", "deferred"}\n    blocker_done_statuses = {"resolved", "closed"}\n    checklist_by_surface: Dict[str, List[ReviewerChecklistItem]] = {}\n    for item in pack.reviewer_checklist:\n        checklist_by_surface.setdefault(item.surface_id, []).append(item)\n    decision_by_surface: Dict[str, List[ReviewDecision]] = {}\n    for decision in pack.decision_log:\n        decision_by_surface.setdefault(decision.surface_id, []).append(decision)\n    assignment_by_surface: Dict[str, List[ReviewRoleAssignment]] = {}\n    for assignment in pack.role_matrix:\n        assignment_by_surface.setdefault(assignment.surface_id, []).append(assignment)\n    signoff_by_surface: Dict[str, List[ReviewSignoff]] = {}\n    for signoff in pack.signoff_log:\n        signoff_by_surface.setdefault(signoff.surface_id, []).append(signoff)\n    blocker_by_surface: Dict[str, List[ReviewBlocker]] = {}\n    blocker_surface_by_id: Dict[str, str] = {}\n    for blocker in pack.blocker_log:\n        blocker_by_surface.setdefault(blocker.surface_id, []).append(blocker)\n        blocker_surface_by_id[blocker.blocker_id] = blocker.surface_id\n    timeline_by_surface: Dict[str, List[ReviewBlockerEvent]] = {}\n    for event in pack.blocker_timeline:\n        surface_id = blocker_surface_by_id.get(event.blocker_id, "none")\n        timeline_by_surface.setdefault(surface_id, []).append(event)\n    entries: List[Dict[str, str]] = []\n    for wireframe in pack.wireframes:\n        checklist_items = checklist_by_surface.get(wireframe.surface_id, [])\n        decisions = decision_by_surface.get(wireframe.surface_id, [])\n        assignments = assignment_by_surface.get(wireframe.surface_id, [])\n        signoffs = signoff_by_surface.get(wireframe.surface_id, [])\n        blockers = blocker_by_surface.get(wireframe.surface_id, [])\n        timeline_events = timeline_by_surface.get(wireframe.surface_id, [])\n        open_total = (\n            sum(1 for item in checklist_items if item.status.lower() not in checklist_ready_statuses)\n            + sum(1 for decision in decisions if decision.status.lower() not in decision_ready_statuses)\n            + sum(1 for assignment in assignments if assignment.status.lower() not in role_ready_statuses)\n            + sum(1 for signoff in signoffs if signoff.status.lower() not in signoff_ready_statuses)\n            + sum(1 for blocker in blockers if blocker.status.lower() not in blocker_done_statuses)\n        )\n        artifact_total = len(checklist_items) + len(decisions) + len(assignments) + len(signoffs) + len(blockers) + len(timeline_events)\n        load_band = "dense" if open_total >= 4 else "active" if open_total >= 2 else "light"\n        entries.append(\n            {\n                "entry_id": f"density-{wireframe.surface_id}",\n                "surface_id": wireframe.surface_id,\n                "artifact_total": str(artifact_total),\n                "open_total": str(open_total),\n                "load_band": load_band,\n                "block_count": str(len(wireframe.primary_blocks)),\n                "note_count": str(len(wireframe.review_notes)),\n                "checklist_count": str(len(checklist_items)),\n                "decision_count": str(len(decisions)),\n                "assignment_count": str(len(assignments)),\n                "signoff_count": str(len(signoffs)),\n                "blocker_count": str(len(blockers)),\n                "timeline_count": str(len(timeline_events)),\n            }\n        )\n    return sorted(entries, key=lambda item: (-int(item["open_total"]), item["surface_id"]))\n\n\ndef _build_signoff_sla_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries = [\n        {\n            "signoff_id": signoff.signoff_id,\n            "surface_id": signoff.surface_id,\n            "role": signoff.role,\n            "status": signoff.status,\n            "sla_status": signoff.sla_status,\n            "requested_at": signoff.requested_at or "none",\n            "due_at": signoff.due_at or "none",\n            "escalation_owner": signoff.escalation_owner or "none",\n            "required": "yes" if signoff.required else "no",\n            "evidence": ",".join(signoff.evidence_links) or "none",\n        }\n        for signoff in pack.signoff_log\n    ]\n    return sorted(entries, key=lambda item: (item["due_at"], item["sla_status"], item["signoff_id"]))\n\n\ndef _build_escalation_dashboard_entries(pack: UIReviewPack) -> List[Dict[str, str]]:\n    entries: List[Dict[str, str]] = []\n    signoff_done_statuses = {"approved", "accepted", "resolved"}\n    blocker_done_statuses = {"resolved", "closed"}\n    for signoff in pack.signoff_log:\n        if signoff.status.lower() in signoff_done_statuses:\n            continue\n        entries.append(\n            {\n                "escalation_id": f"esc-{signoff.signoff_id}",\n                "escalation_owner": signoff.escalation_owner or "none",\n                "item_type": "signoff",\n                "source_id": signoff.signoff_id,\n                "surface_id": signoff.surface_id,\n                "status": signoff.status,\n                "priority": signoff.sla_status,\n                "current_owner": signoff.role,\n                "summary": signoff.notes or signoff.waiver_reason or signoff.role,\n                "due_at": signoff.due_at or "none",\n            }\n        )\n    for blocker in pack.blocker_log:\n        if blocker.status.lower() in blocker_done_statuses:\n            continue\n        entries.append(\n            {\n                "escalation_id": f"esc-{blocker.blocker_id}",\n                "escalation_owner": blocker.escalation_owner or "none",\n                "item_type": "blocker",\n                "source_id": blocker.blocker_id,\n                "surface_id": blocker.surface_id,\n                "status": blocker.status,\n                "priority": blocker.severity,\n                "current_owner": blocker.owner,\n                "summary": blocker.summary,\n                "due_at": "none",\n            }\n        )\n    return sorted(\n        entries,\n        key=lambda item: (item["escalation_owner"], item["item_type"], item["surface_id"], item["source_id"]),\n    )\n\n\ndef render_ui_review_pack_report(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:\n    lines = [\n        "# UI Review Pack",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Audit: {audit.summary}",\n        "",\n        "## Objectives",\n    ]\n    for objective in pack.objectives:\n        lines.append(\n            "- "\n            f"{objective.objective_id}: {objective.title} persona={objective.persona} priority={objective.priority}"\n        )\n        lines.append(\n            "  "\n            f"outcome={objective.outcome} success_signal={objective.success_signal} dependencies={\',\'.join(objective.dependencies) or \'none\'}"\n        )\n\n    review_summary_entries = _build_review_summary_entries(pack)\n    lines.append("")\n    lines.append("## Review Summary Board")\n    lines.append(f"- Categories: {len(review_summary_entries)}")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in review_summary_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: category={entry[\'category\']} total={entry[\'total\']} {entry[\'metrics\']}"\n        )\n    if not review_summary_entries:\n        lines.append("- none")\n\n    objective_coverage_entries = _build_objective_coverage_entries(pack)\n    objective_persona_counts: Dict[str, int] = {}\n    objective_status_counts: Dict[str, int] = {}\n    for entry in objective_coverage_entries:\n        objective_persona_counts[entry[\'persona\']] = objective_persona_counts.get(entry[\'persona\'], 0) + 1\n        objective_status_counts[entry[\'coverage_status\']] = objective_status_counts.get(entry[\'coverage_status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Objective Coverage Board")\n    lines.append(f"- Objectives: {len(objective_coverage_entries)}")\n    lines.append(f"- Personas: {len(objective_persona_counts)}")\n    lines.append("")\n    lines.append("### By Coverage Status")\n    for status, count in sorted(objective_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not objective_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Persona")\n    for persona, count in sorted(objective_persona_counts.items()):\n        lines.append(f"- {persona}: {count}")\n    if not objective_persona_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in objective_coverage_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: objective={entry[\'objective_id\']} persona={entry[\'persona\']} priority={entry[\'priority\']} coverage={entry[\'coverage_status\']} dependencies={entry[\'dependency_count\']} surfaces={entry[\'surface_ids\']}"\n        )\n        lines.append(\n            f"  dependency_ids={entry[\'dependency_ids\']} assignments={entry[\'assignment_ids\']} checklist={entry[\'checklist_ids\']} decisions={entry[\'decision_ids\']} signoffs={entry[\'signoff_ids\']} blockers={entry[\'blocker_ids\']} summary={entry[\'summary\']}"\n        )\n    if not objective_coverage_entries:\n        lines.append("- none")\n\n    persona_readiness_entries = _build_persona_readiness_entries(pack)\n    persona_readiness_counts: Dict[str, int] = {}\n    for entry in persona_readiness_entries:\n        persona_readiness_counts[entry[\'readiness\']] = persona_readiness_counts.get(entry[\'readiness\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Persona Readiness Board")\n    lines.append(f"- Personas: {len(persona_readiness_entries)}")\n    lines.append(f"- Objectives: {len(pack.objectives)}")\n    lines.append("")\n    lines.append("### By Readiness")\n    for readiness, count in sorted(persona_readiness_counts.items()):\n        lines.append(f"- {readiness}: {count}")\n    if not persona_readiness_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in persona_readiness_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: persona={entry[\'persona\']} readiness={entry[\'readiness\']} objectives={entry[\'objective_count\']} assignments={entry[\'assignment_count\']} signoffs={entry[\'signoff_count\']} open_questions={entry[\'question_count\']} queue_items={entry[\'queue_count\']} blockers={entry[\'blocker_count\']}"\n        )\n        lines.append(\n            f"  objective_ids={entry[\'objective_ids\']} surfaces={entry[\'surface_ids\']} queue_ids={entry[\'queue_ids\']} blocker_ids={entry[\'blocker_ids\']}"\n        )\n    if not persona_readiness_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Wireframes")\n    for wireframe in pack.wireframes:\n        lines.append(\n            "- "\n            f"{wireframe.surface_id}: {wireframe.name} device={wireframe.device} entry={wireframe.entry_point}"\n        )\n        lines.append(\n            "  "\n            f"blocks={\',\'.join(wireframe.primary_blocks) or \'none\'} review_notes={\',\'.join(wireframe.review_notes) or \'none\'}"\n        )\n\n    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)\n    wireframe_readiness_counts: Dict[str, int] = {}\n    wireframe_device_counts: Dict[str, int] = {}\n    for entry in wireframe_readiness_entries:\n        wireframe_readiness_counts[entry[\'readiness_status\']] = wireframe_readiness_counts.get(entry[\'readiness_status\'], 0) + 1\n        wireframe_device_counts[entry[\'device\']] = wireframe_device_counts.get(entry[\'device\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Wireframe Readiness Board")\n    lines.append(f"- Wireframes: {len(wireframe_readiness_entries)}")\n    lines.append(f"- Devices: {len(wireframe_device_counts)}")\n    lines.append("")\n    lines.append("### By Readiness")\n    for status, count in sorted(wireframe_readiness_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not wireframe_readiness_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Device")\n    for device, count in sorted(wireframe_device_counts.items()):\n        lines.append(f"- {device}: {count}")\n    if not wireframe_device_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in wireframe_readiness_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: surface={entry[\'surface_id\']} device={entry[\'device\']} readiness={entry[\'readiness_status\']} open_total={entry[\'open_total\']} entry={entry[\'entry_point\']}"\n        )\n        lines.append(\n            f"  checklist_open={entry[\'checklist_open\']} decisions_open={entry[\'decisions_open\']} assignments_open={entry[\'assignments_open\']} signoffs_open={entry[\'signoffs_open\']} blockers_open={entry[\'blockers_open\']} signoffs={entry[\'signoff_ids\']} blockers={entry[\'blocker_ids\']} blocks={entry[\'block_count\']} notes={entry[\'note_count\']} summary={entry[\'summary\']}"\n        )\n    if not wireframe_readiness_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Interactions")\n    for interaction in pack.interactions:\n        lines.append(\n            "- "\n            f"{interaction.flow_id}: {interaction.name} trigger={interaction.trigger}"\n        )\n        lines.append(\n            "  "\n            f"response={interaction.system_response} states={\',\'.join(interaction.states) or \'none\'} exceptions={\',\'.join(interaction.exceptions) or \'none\'}"\n        )\n\n    interaction_coverage_entries = _build_interaction_coverage_entries(pack)\n    interaction_coverage_counts: Dict[str, int] = {}\n    interaction_surface_counts: Dict[str, int] = {}\n    for entry in interaction_coverage_entries:\n        interaction_coverage_counts[entry[\'coverage_status\']] = interaction_coverage_counts.get(entry[\'coverage_status\'], 0) + 1\n        for surface_id in entry[\'surface_ids\'].split(\',\'):\n            if surface_id and surface_id != \'none\':\n                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1\n\n    lines.append("")\n    lines.append("## Interaction Coverage Board")\n    lines.append(f"- Interactions: {len(interaction_coverage_entries)}")\n    lines.append(f"- Surfaces: {len(interaction_surface_counts)}")\n    lines.append("")\n    lines.append("### By Coverage Status")\n    for status, count in sorted(interaction_coverage_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not interaction_coverage_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Surface")\n    for surface_id, count in sorted(interaction_surface_counts.items()):\n        lines.append(f"- {surface_id}: {count}")\n    if not interaction_surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in interaction_coverage_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: flow={entry[\'flow_id\']} surfaces={entry[\'surface_ids\']} owners={entry[\'owners\']} coverage={entry[\'coverage_status\']} states={entry[\'state_count\']} exceptions={entry[\'exception_count\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_ids\']} open_checklist={entry[\'open_checklist_ids\']} trigger={entry[\'summary\']}"\n        )\n    if not interaction_coverage_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Open Questions")\n    for question in pack.open_questions:\n        lines.append(\n            "- "\n            f"{question.question_id}: {question.theme} owner={question.owner} status={question.status}"\n        )\n        lines.append("  " f"question={question.question} impact={question.impact}")\n\n    open_question_entries = _build_open_question_tracker_entries(pack)\n    open_question_owner_counts: Dict[str, int] = {}\n    open_question_theme_counts: Dict[str, int] = {}\n    for entry in open_question_entries:\n        open_question_owner_counts[entry[\'owner\']] = open_question_owner_counts.get(entry[\'owner\'], 0) + 1\n        open_question_theme_counts[entry[\'theme\']] = open_question_theme_counts.get(entry[\'theme\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Open Question Tracker")\n    lines.append(f"- Questions: {len(open_question_entries)}")\n    lines.append(f"- Owners: {len(open_question_owner_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, count in sorted(open_question_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not open_question_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Theme")\n    for theme, count in sorted(open_question_theme_counts.items()):\n        lines.append(f"- {theme}: {count}")\n    if not open_question_theme_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in open_question_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: question={entry[\'question_id\']} owner={entry[\'owner\']} theme={entry[\'theme\']} status={entry[\'status\']} link_status={entry[\'link_status\']} surfaces={entry[\'surface_ids\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_ids\']} flows={entry[\'flow_ids\']} impact={entry[\'impact\']} prompt={entry[\'summary\']}"\n        )\n    if not open_question_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Reviewer Checklist")\n    for item in pack.reviewer_checklist:\n        lines.append(\n            "- "\n            f"{item.item_id}: surface={item.surface_id} owner={item.owner} status={item.status}"\n        )\n        lines.append(\n            "  "\n            f"prompt={item.prompt} evidence={\',\'.join(item.evidence_links) or \'none\'} notes={item.notes or \'none\'}"\n        )\n    if not pack.reviewer_checklist:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Decision Log")\n    for decision in pack.decision_log:\n        lines.append(\n            "- "\n            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"\n        )\n        lines.append(\n            "  "\n            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or \'none\'}"\n        )\n    if not pack.decision_log:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Role Matrix")\n    for assignment in pack.role_matrix:\n        lines.append(\n            "- "\n            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"\n        )\n        lines.append(\n            "  "\n            f"responsibilities={\',\'.join(assignment.responsibilities) or \'none\'} checklist={\',\'.join(assignment.checklist_item_ids) or \'none\'} decisions={\',\'.join(assignment.decision_ids) or \'none\'}"\n        )\n    if not pack.role_matrix:\n        lines.append("- none")\n\n    checklist_trace_entries = _build_checklist_traceability_entries(pack)\n    checklist_trace_owner_counts: Dict[str, int] = {}\n    checklist_trace_status_counts: Dict[str, int] = {}\n    for entry in checklist_trace_entries:\n        checklist_trace_owner_counts[entry[\'owner\']] = checklist_trace_owner_counts.get(entry[\'owner\'], 0) + 1\n        checklist_trace_status_counts[entry[\'status\']] = checklist_trace_status_counts.get(entry[\'status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Checklist Traceability Board")\n    lines.append(f"- Checklist items: {len(checklist_trace_entries)}")\n    lines.append(f"- Owners: {len(checklist_trace_owner_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, count in sorted(checklist_trace_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not checklist_trace_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(checklist_trace_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not checklist_trace_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in checklist_trace_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: item={entry[\'item_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} linked_roles={entry[\'linked_roles\']}"\n        )\n        lines.append(\n            f"  linked_assignments={entry[\'linked_assignments\']} linked_decisions={entry[\'linked_decisions\']} evidence={entry[\'evidence\']} summary={entry[\'summary\']}"\n        )\n    if not checklist_trace_entries:\n        lines.append("- none")\n\n    decision_followup_entries = _build_decision_followup_entries(pack)\n    decision_followup_owner_counts: Dict[str, int] = {}\n    decision_followup_status_counts: Dict[str, int] = {}\n    for entry in decision_followup_entries:\n        decision_followup_owner_counts[entry[\'owner\']] = decision_followup_owner_counts.get(entry[\'owner\'], 0) + 1\n        decision_followup_status_counts[entry[\'status\']] = decision_followup_status_counts.get(entry[\'status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Decision Follow-up Tracker")\n    lines.append(f"- Decisions: {len(decision_followup_entries)}")\n    lines.append(f"- Owners: {len(decision_followup_owner_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, count in sorted(decision_followup_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not decision_followup_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(decision_followup_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not decision_followup_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in decision_followup_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: decision={entry[\'decision_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} linked_roles={entry[\'linked_roles\']}"\n        )\n        lines.append(\n            f"  linked_assignments={entry[\'linked_assignments\']} linked_checklists={entry[\'linked_checklists\']} follow_up={entry[\'follow_up\']} summary={entry[\'summary\']}"\n        )\n    if not decision_followup_entries:\n        lines.append("- none")\n\n    role_coverage_entries = _build_role_coverage_entries(pack)\n    role_coverage_surface_counts: Dict[str, int] = {}\n    role_coverage_status_counts: Dict[str, int] = {}\n    for entry in role_coverage_entries:\n        role_coverage_surface_counts[entry[\'surface_id\']] = role_coverage_surface_counts.get(entry[\'surface_id\'], 0) + 1\n        role_coverage_status_counts[entry[\'status\']] = role_coverage_status_counts.get(entry[\'status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Role Coverage Board")\n    lines.append(f"- Assignments: {len(role_coverage_entries)}")\n    lines.append(f"- Surfaces: {len(role_coverage_surface_counts)}")\n    lines.append("")\n    lines.append("### By Surface")\n    for surface_id, count in sorted(role_coverage_surface_counts.items()):\n        lines.append(f"- {surface_id}: {count}")\n    if not role_coverage_surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(role_coverage_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not role_coverage_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in role_coverage_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: assignment={entry[\'assignment_id\']} surface={entry[\'surface_id\']} role={entry[\'role\']} status={entry[\'status\']} responsibilities={entry[\'responsibility_count\']} checklist={entry[\'checklist_count\']} decisions={entry[\'decision_count\']}"\n        )\n        lines.append(\n            f"  signoff={entry[\'signoff_id\']} signoff_status={entry[\'signoff_status\']} summary={entry[\'summary\']}"\n        )\n    if not role_coverage_entries:\n        lines.append("- none")\n\n    signoff_dependency_entries = _build_signoff_dependency_entries(pack)\n    dependency_counts: Dict[str, int] = {}\n    dependency_sla_counts: Dict[str, int] = {}\n    for entry in signoff_dependency_entries:\n        dependency_counts[entry[\'dependency_status\']] = dependency_counts.get(entry[\'dependency_status\'], 0) + 1\n        dependency_sla_counts[entry[\'sla_status\']] = dependency_sla_counts.get(entry[\'sla_status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Signoff Dependency Board")\n    lines.append(f"- Sign-offs: {len(signoff_dependency_entries)}")\n    lines.append(f"- Dependency states: {len(dependency_counts)}")\n    lines.append("")\n    lines.append("### By Dependency Status")\n    for status, count in sorted(dependency_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not dependency_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By SLA State")\n    for status, count in sorted(dependency_sla_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not dependency_sla_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in signoff_dependency_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} surface={entry[\'surface_id\']} role={entry[\'role\']} status={entry[\'status\']} dependency_status={entry[\'dependency_status\']} blockers={entry[\'blocker_ids\']}"\n        )\n        lines.append(\n            f"  assignment={entry[\'assignment_id\']} checklist={entry[\'checklist_ids\']} decisions={entry[\'decision_ids\']} latest_blocker_event={entry[\'latest_blocker_event\']} sla={entry[\'sla_status\']} due_at={entry[\'due_at\']} cadence={entry[\'reminder_cadence\']} summary={entry[\'summary\']}"\n        )\n    if not signoff_dependency_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Sign-off Log")\n    for signoff in pack.signoff_log:\n        lines.append(\n            "- "\n            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"\n        )\n        lines.append(\n            "  "\n            f"required={\'yes\' if signoff.required else \'no\'} evidence={\',\'.join(signoff.evidence_links) or \'none\'} notes={signoff.notes or \'none\'} waiver_owner={signoff.waiver_owner or \'none\'} waiver_reason={signoff.waiver_reason or \'none\'} requested_at={signoff.requested_at or \'none\'} due_at={signoff.due_at or \'none\'} escalation_owner={signoff.escalation_owner or \'none\'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or \'none\'} reminder_channel={signoff.reminder_channel or \'none\'} last_reminder_at={signoff.last_reminder_at or \'none\'} next_reminder_at={signoff.next_reminder_at or \'none\'}"\n        )\n    if not pack.signoff_log:\n        lines.append("- none")\n\n    signoff_sla_entries = _build_signoff_sla_entries(pack)\n    sla_state_counts: Dict[str, int] = {}\n    sla_owner_counts: Dict[str, int] = {}\n    for entry in signoff_sla_entries:\n        sla_state_counts[entry[\'sla_status\']] = sla_state_counts.get(entry[\'sla_status\'], 0) + 1\n        sla_owner_counts[entry[\'escalation_owner\']] = sla_owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Sign-off SLA Dashboard")\n    lines.append(f"- Sign-offs: {len(signoff_sla_entries)}")\n    lines.append(f"- Escalation owners: {len(sla_owner_counts)}")\n    lines.append("")\n    lines.append("### SLA States")\n    for sla_status, count in sorted(sla_state_counts.items()):\n        lines.append(f"- {sla_status}: {count}")\n    if not sla_state_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Escalation Owners")\n    for owner, count in sorted(sla_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not sla_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Sign-offs")\n    for entry in signoff_sla_entries:\n        lines.append(\n            f"- {entry[\'signoff_id\']}: role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} requested_at={entry[\'requested_at\']} due_at={entry[\'due_at\']} escalation_owner={entry[\'escalation_owner\']}"\n        )\n        lines.append(f"  required={entry[\'required\']} evidence={entry[\'evidence\']}")\n    if not signoff_sla_entries:\n        lines.append("- none")\n\n    signoff_reminder_entries = _build_signoff_reminder_entries(pack)\n    reminder_owner_counts: Dict[str, int] = {}\n    reminder_channel_counts: Dict[str, int] = {}\n    for entry in signoff_reminder_entries:\n        reminder_owner_counts[entry[\'reminder_owner\']] = reminder_owner_counts.get(entry[\'reminder_owner\'], 0) + 1\n        reminder_channel_counts[entry[\'reminder_channel\']] = reminder_channel_counts.get(entry[\'reminder_channel\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Sign-off Reminder Queue")\n    lines.append(f"- Reminders: {len(signoff_reminder_entries)}")\n    lines.append(f"- Owners: {len(reminder_owner_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, count in sorted(reminder_owner_counts.items()):\n        lines.append(f"- {owner}: reminders={count}")\n    if not reminder_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Channel")\n    for channel, count in sorted(reminder_channel_counts.items()):\n        lines.append(f"- {channel}: {count}")\n    if not reminder_channel_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in signoff_reminder_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} owner={entry[\'reminder_owner\']} channel={entry[\'reminder_channel\']}"\n        )\n        lines.append(\n            f"  last_reminder_at={entry[\'last_reminder_at\']} next_reminder_at={entry[\'next_reminder_at\']} due_at={entry[\'due_at\']} summary={entry[\'summary\']}"\n        )\n    if not signoff_reminder_entries:\n        lines.append("- none")\n\n    reminder_cadence_entries = _build_reminder_cadence_entries(pack)\n    reminder_cadence_counts: Dict[str, int] = {}\n    reminder_status_counts: Dict[str, int] = {}\n    for entry in reminder_cadence_entries:\n        reminder_cadence_counts[entry[\'reminder_cadence\']] = reminder_cadence_counts.get(entry[\'reminder_cadence\'], 0) + 1\n        reminder_status_counts[entry[\'reminder_status\']] = reminder_status_counts.get(entry[\'reminder_status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Reminder Cadence Board")\n    lines.append(f"- Items: {len(reminder_cadence_entries)}")\n    lines.append(f"- Cadences: {len(reminder_cadence_counts)}")\n    lines.append("")\n    lines.append("### By Cadence")\n    for cadence, count in sorted(reminder_cadence_counts.items()):\n        lines.append(f"- {cadence}: {count}")\n    if not reminder_cadence_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(reminder_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not reminder_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in reminder_cadence_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} cadence={entry[\'reminder_cadence\']} status={entry[\'reminder_status\']} owner={entry[\'reminder_owner\']}"\n        )\n        lines.append(\n            f"  sla={entry[\'sla_status\']} last_reminder_at={entry[\'last_reminder_at\']} next_reminder_at={entry[\'next_reminder_at\']} due_at={entry[\'due_at\']} summary={entry[\'summary\']}"\n        )\n    if not reminder_cadence_entries:\n        lines.append("- none")\n\n    signoff_breach_entries = _build_signoff_breach_entries(pack)\n    breach_state_counts: Dict[str, int] = {}\n    breach_owner_counts: Dict[str, int] = {}\n    for entry in signoff_breach_entries:\n        breach_state_counts[entry[\'sla_status\']] = breach_state_counts.get(entry[\'sla_status\'], 0) + 1\n        breach_owner_counts[entry[\'escalation_owner\']] = breach_owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Sign-off Breach Board")\n    lines.append(f"- Breach items: {len(signoff_breach_entries)}")\n    lines.append(f"- Escalation owners: {len(breach_owner_counts)}")\n    lines.append("")\n    lines.append("### SLA States")\n    for sla_status, count in sorted(breach_state_counts.items()):\n        lines.append(f"- {sla_status}: {count}")\n    if not breach_state_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Escalation Owners")\n    for owner, count in sorted(breach_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not breach_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in signoff_breach_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} escalation_owner={entry[\'escalation_owner\']}"\n        )\n        lines.append(\n            f"  requested_at={entry[\'requested_at\']} due_at={entry[\'due_at\']} linked_blockers={entry[\'linked_blockers\']} summary={entry[\'summary\']}"\n        )\n    if not signoff_breach_entries:\n        lines.append("- none")\n\n    escalation_entries = _build_escalation_dashboard_entries(pack)\n    escalation_owner_counts: Dict[str, Dict[str, int]] = {}\n    escalation_status_counts: Dict[str, Dict[str, int]] = {}\n    for entry in escalation_entries:\n        owner_counts = escalation_owner_counts.setdefault(\n            entry[\'escalation_owner\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        owner_counts[entry[\'item_type\']] += 1\n        owner_counts[\'total\'] += 1\n        status_counts = escalation_status_counts.setdefault(\n            entry[\'status\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        status_counts[entry[\'item_type\']] += 1\n        status_counts[\'total\'] += 1\n\n    lines.append("")\n    lines.append("## Escalation Dashboard")\n    lines.append(f"- Items: {len(escalation_entries)}")\n    lines.append(f"- Escalation owners: {len(escalation_owner_counts)}")\n    lines.append("")\n    lines.append("### By Escalation Owner")\n    for owner, counts in sorted(escalation_owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not escalation_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, counts in sorted(escalation_status_counts.items()):\n        lines.append(\n            f"- {status}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not escalation_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Escalations")\n    for entry in escalation_entries:\n        lines.append(\n            f"- {entry[\'escalation_id\']}: owner={entry[\'escalation_owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} priority={entry[\'priority\']} current_owner={entry[\'current_owner\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} due_at={entry[\'due_at\']}")\n    if not escalation_entries:\n        lines.append("- none")\n\n    escalation_handoff_entries = _build_escalation_handoff_entries(pack)\n    handoff_status_counts: Dict[str, int] = {}\n    handoff_channel_counts: Dict[str, int] = {}\n    for entry in escalation_handoff_entries:\n        handoff_status_counts[entry[\'status\']] = handoff_status_counts.get(entry[\'status\'], 0) + 1\n        handoff_channel_counts[entry[\'channel\']] = handoff_channel_counts.get(entry[\'channel\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Escalation Handoff Ledger")\n    lines.append(f"- Handoffs: {len(escalation_handoff_entries)}")\n    lines.append(f"- Channels: {len(handoff_channel_counts)}")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(handoff_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not handoff_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Channel")\n    for channel, count in sorted(handoff_channel_counts.items()):\n        lines.append(f"- {channel}: {count}")\n    if not handoff_channel_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in escalation_handoff_entries:\n        lines.append(\n            f"- {entry[\'ledger_id\']}: event={entry[\'event_id\']} blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} actor={entry[\'actor\']} status={entry[\'status\']} at={entry[\'timestamp\']}"\n        )\n        lines.append(\n            f"  from={entry[\'handoff_from\']} to={entry[\'handoff_to\']} channel={entry[\'channel\']} artifact={entry[\'artifact_ref\']} next_action={entry[\'next_action\']}"\n        )\n    if not escalation_handoff_entries:\n        lines.append("- none")\n\n    handoff_ack_entries = _build_handoff_ack_entries(pack)\n    handoff_ack_owner_counts: Dict[str, int] = {}\n    handoff_ack_status_counts: Dict[str, int] = {}\n    for entry in handoff_ack_entries:\n        handoff_ack_owner_counts[entry[\'ack_owner\']] = handoff_ack_owner_counts.get(entry[\'ack_owner\'], 0) + 1\n        handoff_ack_status_counts[entry[\'ack_status\']] = handoff_ack_status_counts.get(entry[\'ack_status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Handoff Ack Ledger")\n    lines.append(f"- Ack items: {len(handoff_ack_entries)}")\n    lines.append(f"- Ack owners: {len(handoff_ack_owner_counts)}")\n    lines.append("")\n    lines.append("### By Ack Owner")\n    for owner, count in sorted(handoff_ack_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not handoff_ack_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Ack Status")\n    for status, count in sorted(handoff_ack_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not handoff_ack_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in handoff_ack_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: event={entry[\'event_id\']} blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} handoff_to={entry[\'handoff_to\']} ack_owner={entry[\'ack_owner\']} ack_status={entry[\'ack_status\']} ack_at={entry[\'ack_at\']}"\n        )\n        lines.append(\n            f"  actor={entry[\'actor\']} status={entry[\'status\']} channel={entry[\'channel\']} artifact={entry[\'artifact_ref\']} summary={entry[\'summary\']}"\n        )\n    if not handoff_ack_entries:\n        lines.append("- none")\n\n    owner_digest_entries = _build_owner_escalation_digest_entries(pack)\n    owner_digest_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_digest_entries:\n        counts = owner_digest_counts.setdefault(\n            entry[\'owner\'],\n            {\'blocker\': 0, \'signoff\': 0, \'reminder\': 0, \'freeze\': 0, \'handoff\': 0, \'total\': 0},\n        )\n        counts[entry[\'item_type\']] += 1\n        counts[\'total\'] += 1\n\n    lines.append("")\n    lines.append("## Owner Escalation Digest")\n    lines.append(f"- Owners: {len(owner_digest_counts)}")\n    lines.append(f"- Items: {len(owner_digest_entries)}")\n    lines.append("")\n    lines.append("### Owners")\n    for owner, counts in sorted(owner_digest_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} reminders={counts[\'reminder\']} freezes={counts[\'freeze\']} handoffs={counts[\'handoff\']} total={counts[\'total\']}"\n        )\n    if not owner_digest_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in owner_digest_entries:\n        lines.append(\n            f"- {entry[\'digest_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} detail={entry[\'detail\']}")\n    if not owner_digest_entries:\n        lines.append("- none")\n\n    owner_workload_entries = _build_owner_workload_entries(pack)\n    owner_workload_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_workload_entries:\n        counts = owner_workload_counts.setdefault(\n            entry[\'owner\'],\n            {\'blocker\': 0, \'checklist\': 0, \'decision\': 0, \'signoff\': 0, \'reminder\': 0, \'renewal\': 0, \'total\': 0},\n        )\n        counts[entry[\'item_type\']] += 1\n        counts[\'total\'] += 1\n\n    lines.append("")\n    lines.append("## Owner Workload Board")\n    lines.append(f"- Owners: {len(owner_workload_counts)}")\n    lines.append(f"- Items: {len(owner_workload_entries)}")\n    lines.append("")\n    lines.append("### Owners")\n    for owner, counts in sorted(owner_workload_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} checklist={counts[\'checklist\']} decisions={counts[\'decision\']} signoffs={counts[\'signoff\']} reminders={counts[\'reminder\']} renewals={counts[\'renewal\']} total={counts[\'total\']}"\n        )\n    if not owner_workload_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in owner_workload_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} lane={entry[\'lane\']}"\n        )\n        lines.append(f"  detail={entry[\'detail\']} summary={entry[\'summary\']}")\n    if not owner_workload_entries:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Blocker Log")\n    for blocker in pack.blocker_log:\n        lines.append(\n            "- "\n            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"\n        )\n        lines.append(\n            "  "\n            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or \'none\'} next_action={blocker.next_action or \'none\'} freeze_owner={blocker.freeze_owner or \'none\'} freeze_until={blocker.freeze_until or \'none\'} freeze_approved_by={blocker.freeze_approved_by or \'none\'} freeze_approved_at={blocker.freeze_approved_at or \'none\'}"\n        )\n    if not pack.blocker_log:\n        lines.append("- none")\n\n    lines.append("")\n    lines.append("## Blocker Timeline")\n    for event in pack.blocker_timeline:\n        lines.append(\n            "- "\n            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"\n        )\n        lines.append(\n            "  "\n            f"summary={event.summary} next_action={event.next_action or \'none\'}"\n        )\n    if not pack.blocker_timeline:\n        lines.append("- none")\n\n    exception_entries = _build_review_exception_entries(pack)\n    timeline_index = _build_blocker_timeline_index(pack)\n\n    lines.append("")\n    lines.append("## Review Exceptions")\n    for entry in exception_entries:\n        lines.append(\n            f"- {entry[\'exception_id\']}: type={entry[\'category\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} severity={entry[\'severity\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} evidence={entry[\'evidence\']} latest_event={entry[\'latest_event\']} next_action={entry[\'next_action\']}"\n        )\n    if not exception_entries:\n        lines.append("- none")\n\n    freeze_entries = _build_freeze_exception_entries(pack)\n    freeze_owner_counts: Dict[str, Dict[str, int]] = {}\n    freeze_surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in freeze_entries:\n        owner_counts = freeze_owner_counts.setdefault(\n            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        owner_counts[entry["item_type"]] += 1\n        owner_counts["total"] += 1\n        surface_counts = freeze_surface_counts.setdefault(\n            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        surface_counts[entry["item_type"]] += 1\n        surface_counts["total"] += 1\n\n    lines.append("")\n    lines.append("## Review Freeze Exception Board")\n    lines.append(f"- Exceptions: {len(freeze_entries)}")\n    lines.append(f"- Owners: {len(freeze_owner_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, counts in sorted(freeze_owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not freeze_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Surface")\n    for surface_id, counts in sorted(freeze_surface_counts.items()):\n        lines.append(\n            f"- {surface_id}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not freeze_surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in freeze_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} window={entry[\'window\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} evidence={entry[\'evidence\']} next_action={entry[\'next_action\']}"\n        )\n    if not freeze_entries:\n        lines.append("- none")\n\n    freeze_approval_entries = _build_freeze_approval_entries(pack)\n    freeze_approval_owner_counts: Dict[str, int] = {}\n    freeze_approval_status_counts: Dict[str, int] = {}\n    for entry in freeze_approval_entries:\n        freeze_approval_owner_counts[entry[\'freeze_approved_by\']] = freeze_approval_owner_counts.get(entry[\'freeze_approved_by\'], 0) + 1\n        freeze_approval_status_counts[entry[\'status\']] = freeze_approval_status_counts.get(entry[\'status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Freeze Approval Trail")\n    lines.append(f"- Approvals: {len(freeze_approval_entries)}")\n    lines.append(f"- Approvers: {len(freeze_approval_owner_counts)}")\n    lines.append("")\n    lines.append("### By Approver")\n    for owner, count in sorted(freeze_approval_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not freeze_approval_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, count in sorted(freeze_approval_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not freeze_approval_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in freeze_approval_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} owner={entry[\'freeze_owner\']} approved_by={entry[\'freeze_approved_by\']} approved_at={entry[\'freeze_approved_at\']} window={entry[\'freeze_until\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} latest_event={entry[\'latest_event\']} next_action={entry[\'next_action\']}"\n        )\n    if not freeze_approval_entries:\n        lines.append("- none")\n\n    freeze_renewal_entries = _build_freeze_renewal_entries(pack)\n    freeze_renewal_owner_counts: Dict[str, int] = {}\n    freeze_renewal_status_counts: Dict[str, int] = {}\n    for entry in freeze_renewal_entries:\n        freeze_renewal_owner_counts[entry[\'renewal_owner\']] = freeze_renewal_owner_counts.get(entry[\'renewal_owner\'], 0) + 1\n        freeze_renewal_status_counts[entry[\'renewal_status\']] = freeze_renewal_status_counts.get(entry[\'renewal_status\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Freeze Renewal Tracker")\n    lines.append(f"- Renewal items: {len(freeze_renewal_entries)}")\n    lines.append(f"- Renewal owners: {len(freeze_renewal_owner_counts)}")\n    lines.append("")\n    lines.append("### By Renewal Owner")\n    for owner, count in sorted(freeze_renewal_owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not freeze_renewal_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Renewal Status")\n    for status, count in sorted(freeze_renewal_status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not freeze_renewal_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in freeze_renewal_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} renewal_owner={entry[\'renewal_owner\']} renewal_by={entry[\'renewal_by\']} renewal_status={entry[\'renewal_status\']}"\n        )\n        lines.append(\n            f"  freeze_owner={entry[\'freeze_owner\']} freeze_until={entry[\'freeze_until\']} approved_by={entry[\'freeze_approved_by\']} summary={entry[\'summary\']} next_action={entry[\'next_action\']}"\n        )\n    if not freeze_renewal_entries:\n        lines.append("- none")\n\n    exception_owner_counts: Dict[str, Dict[str, int]] = {}\n    exception_status_counts: Dict[str, Dict[str, int]] = {}\n    exception_surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in exception_entries:\n        owner_counts = exception_owner_counts.setdefault(\n            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        owner_counts[entry["category"]] += 1\n        owner_counts["total"] += 1\n        status_counts = exception_status_counts.setdefault(\n            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        status_counts[entry["category"]] += 1\n        status_counts["total"] += 1\n        surface_counts = exception_surface_counts.setdefault(\n            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        surface_counts[entry["category"]] += 1\n        surface_counts["total"] += 1\n\n    lines.append("")\n    lines.append("## Review Exception Matrix")\n    lines.append(f"- Exceptions: {len(exception_entries)}")\n    lines.append(f"- Owners: {len(exception_owner_counts)}")\n    lines.append(f"- Surfaces: {len(exception_surface_counts)}")\n    lines.append("")\n    lines.append("### By Owner")\n    for owner, counts in sorted(exception_owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not exception_owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Status")\n    for status, counts in sorted(exception_status_counts.items()):\n        lines.append(\n            f"- {status}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not exception_status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### By Surface")\n    for surface_id, counts in sorted(exception_surface_counts.items()):\n        lines.append(\n            f"- {surface_id}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not exception_surface_counts:\n        lines.append("- none")\n\n    audit_density_entries = _build_audit_density_entries(pack)\n    audit_density_band_counts: Dict[str, int] = {}\n    for entry in audit_density_entries:\n        audit_density_band_counts[entry[\'load_band\']] = audit_density_band_counts.get(entry[\'load_band\'], 0) + 1\n\n    lines.append("")\n    lines.append("## Audit Density Board")\n    lines.append(f"- Surfaces: {len(audit_density_entries)}")\n    lines.append(f"- Load bands: {len(audit_density_band_counts)}")\n    lines.append("")\n    lines.append("### By Load Band")\n    for band, count in sorted(audit_density_band_counts.items()):\n        lines.append(f"- {band}: {count}")\n    if not audit_density_band_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Entries")\n    for entry in audit_density_entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: surface={entry[\'surface_id\']} artifact_total={entry[\'artifact_total\']} open_total={entry[\'open_total\']} band={entry[\'load_band\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_count\']} decisions={entry[\'decision_count\']} assignments={entry[\'assignment_count\']} signoffs={entry[\'signoff_count\']} blockers={entry[\'blocker_count\']} timeline={entry[\'timeline_count\']} blocks={entry[\'block_count\']} notes={entry[\'note_count\']}"\n        )\n    if not audit_density_entries:\n        lines.append("- none")\n\n    owner_review_queue = _build_owner_review_queue_entries(pack)\n    owner_queue_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_review_queue:\n        counts = owner_queue_counts.setdefault(\n            entry["owner"],\n            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},\n        )\n        counts[entry["item_type"]] += 1\n        counts["total"] += 1\n\n    lines.append("")\n    lines.append("## Owner Review Queue")\n    lines.append(f"- Owners: {len(owner_queue_counts)}")\n    lines.append(f"- Queue items: {len(owner_review_queue)}")\n    lines.append("")\n    lines.append("### Owners")\n    for owner, counts in sorted(owner_queue_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} checklist={counts[\'checklist\']} decisions={counts[\'decision\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not owner_queue_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Items")\n    for entry in owner_review_queue:\n        lines.append(\n            f"- {entry[\'queue_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} next_action={entry[\'next_action\']}")\n    if not owner_review_queue:\n        lines.append("- none")\n\n    status_counts: Dict[str, int] = {}\n    actor_counts: Dict[str, int] = {}\n    for event in pack.blocker_timeline:\n        status_counts[event.status] = status_counts.get(event.status, 0) + 1\n        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1\n    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}\n    orphan_timeline_ids = sorted(\n        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids\n    )\n\n    lines.append("")\n    lines.append("## Blocker Timeline Summary")\n    lines.append(f"- Total events: {len(pack.blocker_timeline)}")\n    lines.append(f"- Blockers with timeline: {len(timeline_index)}")\n    lines.append(f"- Orphan timeline blockers: {\',\'.join(orphan_timeline_ids) or \'none\'}")\n    lines.append("")\n    lines.append("### Events by Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Events by Actor")\n    for actor, count in sorted(actor_counts.items()):\n        lines.append(f"- {actor}: {count}")\n    if not actor_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("### Latest Blocker Events")\n    for blocker in pack.blocker_log:\n        latest_events = timeline_index.get(blocker.blocker_id, [])\n        latest = latest_events[-1] if latest_events else None\n        if latest is None:\n            lines.append(f"- {blocker.blocker_id}: latest=none")\n            continue\n        lines.append(\n            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"\n        )\n    if not pack.blocker_log:\n        lines.append("- none")\n\n    lines.extend(\n        [\n            "",\n            "## Audit Findings",\n            f"- Missing sections: {\', \'.join(audit.missing_sections) or \'none\'}",\n            f"- Objectives missing success signals: {\', \'.join(audit.objectives_missing_signals) or \'none\'}",\n            f"- Wireframes missing blocks: {\', \'.join(audit.wireframes_missing_blocks) or \'none\'}",\n            f"- Interactions missing states: {\', \'.join(audit.interactions_missing_states) or \'none\'}",\n            f"- Unresolved questions: {\', \'.join(audit.unresolved_question_ids) or \'none\'}",\n            f"- Wireframes missing checklist coverage: {\', \'.join(audit.wireframes_missing_checklists) or \'none\'}",\n            f"- Orphan checklist surfaces: {\', \'.join(audit.orphan_checklist_surfaces) or \'none\'}",\n            f"- Checklist items missing evidence: {\', \'.join(audit.checklist_items_missing_evidence) or \'none\'}",\n            f"- Checklist items missing role links: {\', \'.join(audit.checklist_items_missing_role_links) or \'none\'}",\n            f"- Wireframes missing decision coverage: {\', \'.join(audit.wireframes_missing_decisions) or \'none\'}",\n            f"- Orphan decision surfaces: {\', \'.join(audit.orphan_decision_surfaces) or \'none\'}",\n            f"- Unresolved decision ids: {\', \'.join(audit.unresolved_decision_ids) or \'none\'}",\n            f"- Unresolved decisions missing follow-ups: {\', \'.join(audit.unresolved_decisions_missing_follow_ups) or \'none\'}",\n            f"- Wireframes missing role assignments: {\', \'.join(audit.wireframes_missing_role_assignments) or \'none\'}",\n            f"- Orphan role assignment surfaces: {\', \'.join(audit.orphan_role_assignment_surfaces) or \'none\'}",\n            f"- Role assignments missing responsibilities: {\', \'.join(audit.role_assignments_missing_responsibilities) or \'none\'}",\n            f"- Role assignments missing checklist links: {\', \'.join(audit.role_assignments_missing_checklist_links) or \'none\'}",\n            f"- Role assignments missing decision links: {\', \'.join(audit.role_assignments_missing_decision_links) or \'none\'}",\n            f"- Decisions missing role links: {\', \'.join(audit.decisions_missing_role_links) or \'none\'}",\n            f"- Wireframes missing signoff coverage: {\', \'.join(audit.wireframes_missing_signoffs) or \'none\'}",\n            f"- Orphan signoff surfaces: {\', \'.join(audit.orphan_signoff_surfaces) or \'none\'}",\n            f"- Signoffs missing role assignments: {\', \'.join(audit.signoffs_missing_assignments) or \'none\'}",\n            f"- Signoffs missing evidence: {\', \'.join(audit.signoffs_missing_evidence) or \'none\'}",\n            f"- Signoffs missing requested dates: {\', \'.join(audit.signoffs_missing_requested_dates) or \'none\'}",\n            f"- Signoffs missing due dates: {\', \'.join(audit.signoffs_missing_due_dates) or \'none\'}",\n            f"- Signoffs missing escalation owners: {\', \'.join(audit.signoffs_missing_escalation_owners) or \'none\'}",\n            f"- Signoffs missing reminder owners: {\', \'.join(audit.signoffs_missing_reminder_owners) or \'none\'}",\n            f"- Signoffs missing next reminders: {\', \'.join(audit.signoffs_missing_next_reminders) or \'none\'}",\n            f"- Signoffs missing reminder cadence: {\', \'.join(audit.signoffs_missing_reminder_cadence) or \'none\'}",\n            f"- Signoffs with breached SLA: {\', \'.join(audit.signoffs_with_breached_sla) or \'none\'}",\n            f"- Waived signoffs missing metadata: {\', \'.join(audit.waived_signoffs_missing_metadata) or \'none\'}",\n            f"- Unresolved required signoff ids: {\', \'.join(audit.unresolved_required_signoff_ids) or \'none\'}",\n            f"- Blockers missing signoff links: {\', \'.join(audit.blockers_missing_signoff_links) or \'none\'}",\n            f"- Blockers missing escalation owners: {\', \'.join(audit.blockers_missing_escalation_owners) or \'none\'}",\n            f"- Blockers missing next actions: {\', \'.join(audit.blockers_missing_next_actions) or \'none\'}",\n            f"- Freeze exceptions missing owners: {\', \'.join(audit.freeze_exceptions_missing_owners) or \'none\'}",\n            f"- Freeze exceptions missing windows: {\', \'.join(audit.freeze_exceptions_missing_until) or \'none\'}",\n            f"- Freeze exceptions missing approvers: {\', \'.join(audit.freeze_exceptions_missing_approvers) or \'none\'}",\n            f"- Freeze exceptions missing approval dates: {\', \'.join(audit.freeze_exceptions_missing_approval_dates) or \'none\'}",\n            f"- Freeze exceptions missing renewal owners: {\', \'.join(audit.freeze_exceptions_missing_renewal_owners) or \'none\'}",\n            f"- Freeze exceptions missing renewal dates: {\', \'.join(audit.freeze_exceptions_missing_renewal_dates) or \'none\'}",\n            f"- Blockers missing timeline events: {\', \'.join(audit.blockers_missing_timeline_events) or \'none\'}",\n            f"- Closed blockers missing resolution events: {\', \'.join(audit.closed_blockers_missing_resolution_events) or \'none\'}",\n            f"- Orphan blocker surfaces: {\', \'.join(audit.orphan_blocker_surfaces) or \'none\'}",\n            f"- Orphan blocker timeline blocker ids: {\', \'.join(audit.orphan_blocker_timeline_blocker_ids) or \'none\'}",\n            f"- Handoff events missing targets: {\', \'.join(audit.handoff_events_missing_targets) or \'none\'}",\n            f"- Handoff events missing artifacts: {\', \'.join(audit.handoff_events_missing_artifacts) or \'none\'}",\n            f"- Handoff events missing ack owners: {\', \'.join(audit.handoff_events_missing_ack_owners) or \'none\'}",\n            f"- Handoff events missing ack dates: {\', \'.join(audit.handoff_events_missing_ack_dates) or \'none\'}",\n            f"- Unresolved required signoffs without blockers: {\', \'.join(audit.unresolved_required_signoffs_without_blockers) or \'none\'}",\n        ]\n    )\n    return "\\n".join(lines)\n\n\ndef build_big_4204_review_pack() -> UIReviewPack:\n    return UIReviewPack(\n        issue_id="BIG-4204",\n        title="UI评审包输出",\n        version="v4.0-design-sprint",\n        requires_reviewer_checklist=True,\n        requires_decision_log=True,\n        requires_role_matrix=True,\n        requires_signoff_log=True,\n        requires_blocker_log=True,\n        requires_blocker_timeline=True,\n        objectives=[\n            ReviewObjective(\n                objective_id="obj-overview-decision",\n                title="Validate the executive overview narrative and drill-down posture",\n                persona="VP Eng",\n                outcome="Leadership can confirm the overview page balances KPI density with investigation entry points.",\n                success_signal="Reviewers agree the overview supports release, risk, and queue drill-down without extra walkthroughs.",\n                priority="P0",\n                dependencies=["BIG-4203", "OPE-132"],\n            ),\n            ReviewObjective(\n                objective_id="obj-queue-governance",\n                title="Confirm queue control actions and approval posture",\n                persona="Platform Admin",\n                outcome="Operators can assess batch approvals, audit visibility, and denial paths from one frame.",\n                success_signal="The queue frame clearly shows allowed actions, denied roles, and audit expectations.",\n                priority="P0",\n                dependencies=["BIG-4203", "OPE-131", "OPE-132"],\n            ),\n            ReviewObjective(\n                objective_id="obj-run-detail-investigation",\n                title="Validate replay and audit investigation flow",\n                persona="Eng Lead",\n                outcome="Run detail reviewers can trace evidence, replay context, and escalation actions in one surface.",\n                success_signal="The run-detail frame makes failure replay and escalation decisions reviewable without hidden dependencies.",\n                priority="P0",\n                dependencies=["BIG-4203", "OPE-72", "OPE-73"],\n            ),\n            ReviewObjective(\n                objective_id="obj-triage-handoff",\n                title="Confirm triage and cross-team handoff readiness",\n                persona="Cross-Team Operator",\n                outcome="Reviewers can evaluate assignment, handoff, and queue-state transitions as one operator journey.",\n                success_signal="The triage frame exposes action states, owner switches, and handoff exceptions explicitly.",\n                priority="P0",\n                dependencies=["BIG-4203", "OPE-76", "OPE-79", "OPE-132"],\n            ),\n        ],\n        wireframes=[\n            WireframeSurface(\n                surface_id="wf-overview",\n                name="Overview command deck",\n                device="desktop",\n                entry_point="/overview",\n                primary_blocks=["top bar", "kpi strip", "risk panel", "drill-down table"],\n                review_notes=["Confirm metric density and executive scan path.", "Check alert prominence versus weekly summary card."],\n            ),\n            WireframeSurface(\n                surface_id="wf-queue",\n                name="Queue control center",\n                device="desktop",\n                entry_point="/queue",\n                primary_blocks=["approval queue", "selection toolbar", "filters", "audit rail"],\n                review_notes=["Validate batch-approve CTA hierarchy.", "Review denied-role behavior for non-operator personas."],\n            ),\n            WireframeSurface(\n                surface_id="wf-run-detail",\n                name="Run detail and replay",\n                device="desktop",\n                entry_point="/runs/detail",\n                primary_blocks=["timeline", "artifact drawer", "replay controls", "audit notes"],\n                review_notes=["Check replay mode discoverability.", "Ensure escalation path is visible next to audit evidence."],\n            ),\n            WireframeSurface(\n                surface_id="wf-triage",\n                name="Triage and handoff board",\n                device="desktop",\n                entry_point="/triage",\n                primary_blocks=["severity lanes", "bulk actions", "handoff panel", "ownership history"],\n                review_notes=["Validate cross-team operator workflow.", "Confirm exception path for denied escalation."],\n            ),\n        ],\n        interactions=[\n            InteractionFlow(\n                flow_id="flow-overview-drilldown",\n                name="Overview to investigation drill-down",\n                trigger="VP Eng selects a KPI card or blocker cluster on the overview page",\n                system_response="The console pivots into the matching queue or run-detail slice while preserving context filters.",\n                states=["default", "focus", "handoff-ready"],\n                exceptions=["Warn when the requested slice is permission-denied.", "Show fallback summary when no matching runs exist."],\n            ),\n            InteractionFlow(\n                flow_id="flow-queue-bulk-approval",\n                name="Queue batch approval review",\n                trigger="Platform Admin selects multiple tasks and opens the bulk approval toolbar",\n                system_response="The queue shows approval scope, audit consequence, and denied-role messaging before submit.",\n                states=["default", "selection", "confirming", "success"],\n                exceptions=["Disable submit when tasks cross unauthorized scopes.", "Route to audit timeline when approval policy changes mid-flow."],\n            ),\n            InteractionFlow(\n                flow_id="flow-run-replay",\n                name="Run replay with evidence audit",\n                trigger="Eng Lead switches replay mode on a failed run",\n                system_response="The page updates the timeline, artifacts, and escalation controls while keeping the audit trail visible.",\n                states=["default", "replay", "compare", "escalated"],\n                exceptions=["Show replay-unavailable state for incomplete artifacts.", "Require escalation reason before handoff."],\n            ),\n            InteractionFlow(\n                flow_id="flow-triage-handoff",\n                name="Triage ownership reassignment and handoff",\n                trigger="Cross-Team Operator bulk-assigns a finding set or opens the handoff panel",\n                system_response="The triage board updates owner, workflow, and handoff evidence in one acknowledgement step.",\n                states=["default", "selected", "handoff", "completed"],\n                exceptions=["Block handoff when reviewer coverage is incomplete.", "Record denied-role attempt in the audit summary."],\n            ),\n        ],\n        open_questions=[\n            OpenQuestion(\n                question_id="oq-role-density",\n                theme="role-matrix",\n                question="Should VP Eng see queue batch controls in read-only form or be routed to a summary-only state?",\n                owner="product-experience",\n                impact="Changes denial-path copy, button placement, and review criteria for queue and triage pages.",\n            ),\n            OpenQuestion(\n                question_id="oq-alert-priority",\n                theme="information-architecture",\n                question="Should regression alerts outrank approval alerts in the top bar for the design sprint prototype?",\n                owner="engineering-operations",\n                impact="Affects alert hierarchy and the scan path used in the overview and triage reviews.",\n            ),\n            OpenQuestion(\n                question_id="oq-handoff-evidence",\n                theme="handoff",\n                question="How much ownership history must stay visible before the run-detail and triage pages collapse older audit entries?",\n                owner="orchestration-office",\n                impact="Shapes the default density of the audit rail and the threshold for the review-ready packet.",\n            ),\n        ],\n        reviewer_checklist=[\n            ReviewerChecklistItem(\n                item_id="chk-overview-kpi-scan",\n                surface_id="wf-overview",\n                prompt="Verify the KPI strip still supports one-screen executive scanning before drill-down.",\n                owner="VP Eng",\n                status="ready",\n                evidence_links=["wf-overview", "flow-overview-drilldown"],\n                notes="Use the overview card hierarchy as the primary decision frame.",\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-overview-alert-hierarchy",\n                surface_id="wf-overview",\n                prompt="Confirm alert priority is readable when approvals and regressions compete for attention.",\n                owner="engineering-operations",\n                status="open",\n                evidence_links=["wf-overview", "oq-alert-priority"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-queue-batch-approval",\n                surface_id="wf-queue",\n                prompt="Check that batch approval clearly communicates scope, denial paths, and audit consequences.",\n                owner="Platform Admin",\n                status="ready",\n                evidence_links=["wf-queue", "flow-queue-bulk-approval"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-queue-role-density",\n                surface_id="wf-queue",\n                prompt="Validate whether VP Eng should get a summary-only queue variant instead of operator controls.",\n                owner="product-experience",\n                status="open",\n                evidence_links=["wf-queue", "oq-role-density"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-run-replay-context",\n                surface_id="wf-run-detail",\n                prompt="Ensure replay, compare, and escalation states remain distinguishable without narration.",\n                owner="Eng Lead",\n                status="ready",\n                evidence_links=["wf-run-detail", "flow-run-replay"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-run-audit-density",\n                surface_id="wf-run-detail",\n                prompt="Confirm the audit rail retains enough ownership history before collapsing older entries.",\n                owner="orchestration-office",\n                status="open",\n                evidence_links=["wf-run-detail", "oq-handoff-evidence"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-triage-handoff-clarity",\n                surface_id="wf-triage",\n                prompt="Check that cross-team handoff consequences are explicit before ownership changes commit.",\n                owner="Cross-Team Operator",\n                status="ready",\n                evidence_links=["wf-triage", "flow-triage-handoff"],\n            ),\n            ReviewerChecklistItem(\n                item_id="chk-triage-bulk-assign",\n                surface_id="wf-triage",\n                prompt="Validate bulk assignment visibility without burying the audit context.",\n                owner="Platform Admin",\n                status="ready",\n                evidence_links=["wf-triage", "flow-triage-handoff"],\n            ),\n        ],\n        decision_log=[\n            ReviewDecision(\n                decision_id="dec-overview-alert-stack",\n                surface_id="wf-overview",\n                owner="product-experience",\n                summary="Keep approval and regression alerts in one stacked priority rail.",\n                rationale="Reviewers need one comparison lane before jumping into queue or triage surfaces.",\n                status="accepted",\n            ),\n            ReviewDecision(\n                decision_id="dec-queue-vp-summary",\n                surface_id="wf-queue",\n                owner="VP Eng",\n                summary="Route VP Eng to a summary-first queue state instead of operator controls.",\n                rationale="The VP Eng persona needs queue visibility without accidental action affordances.",\n                status="proposed",\n                follow_up="Resolve after the next design critique with policy owners.",\n            ),\n            ReviewDecision(\n                decision_id="dec-run-detail-audit-rail",\n                surface_id="wf-run-detail",\n                owner="Eng Lead",\n                summary="Keep audit evidence visible beside replay controls in all replay states.",\n                rationale="Replay decisions are inseparable from audit context and escalation evidence.",\n                status="accepted",\n            ),\n            ReviewDecision(\n                decision_id="dec-triage-handoff-density",\n                surface_id="wf-triage",\n                owner="Cross-Team Operator",\n                summary="Preserve ownership history in the triage rail until handoff is acknowledged.",\n                rationale="Operators need a stable handoff trail before collapsing older events.",\n                status="accepted",\n            ),\n        ],\n        role_matrix=[\n            ReviewRoleAssignment(\n                assignment_id="role-overview-vp-eng",\n                surface_id="wf-overview",\n                role="VP Eng",\n                responsibilities=["approve overview scan path", "validate KPI-to-drilldown narrative"],\n                checklist_item_ids=["chk-overview-kpi-scan"],\n                decision_ids=["dec-overview-alert-stack"],\n                status="ready",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-overview-eng-ops",\n                surface_id="wf-overview",\n                role="engineering-operations",\n                responsibilities=["review alert priority posture"],\n                checklist_item_ids=["chk-overview-alert-hierarchy"],\n                decision_ids=["dec-overview-alert-stack"],\n                status="open",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-queue-platform-admin",\n                surface_id="wf-queue",\n                role="Platform Admin",\n                responsibilities=["validate batch-approval copy", "confirm permission posture"],\n                checklist_item_ids=["chk-queue-batch-approval"],\n                decision_ids=["dec-queue-vp-summary"],\n                status="ready",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-queue-product-experience",\n                surface_id="wf-queue",\n                role="product-experience",\n                responsibilities=["tune summary-only VP variant"],\n                checklist_item_ids=["chk-queue-role-density"],\n                decision_ids=["dec-queue-vp-summary"],\n                status="open",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-run-detail-eng-lead",\n                surface_id="wf-run-detail",\n                role="Eng Lead",\n                responsibilities=["approve replay-state clarity", "confirm escalation adjacency"],\n                checklist_item_ids=["chk-run-replay-context"],\n                decision_ids=["dec-run-detail-audit-rail"],\n                status="ready",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-run-detail-orchestration-office",\n                surface_id="wf-run-detail",\n                role="orchestration-office",\n                responsibilities=["review audit density threshold"],\n                checklist_item_ids=["chk-run-audit-density"],\n                decision_ids=["dec-run-detail-audit-rail"],\n                status="open",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-triage-cross-team-operator",\n                surface_id="wf-triage",\n                role="Cross-Team Operator",\n                responsibilities=["approve handoff clarity", "validate ownership transition story"],\n                checklist_item_ids=["chk-triage-handoff-clarity"],\n                decision_ids=["dec-triage-handoff-density"],\n                status="ready",\n            ),\n            ReviewRoleAssignment(\n                assignment_id="role-triage-platform-admin",\n                surface_id="wf-triage",\n                role="Platform Admin",\n                responsibilities=["confirm bulk-assign audit visibility"],\n                checklist_item_ids=["chk-triage-bulk-assign"],\n                decision_ids=["dec-triage-handoff-density"],\n                status="ready",\n            ),\n        ],\n        signoff_log=[\n            ReviewSignoff(\n                signoff_id="sig-overview-vp-eng",\n                assignment_id="role-overview-vp-eng",\n                surface_id="wf-overview",\n                role="VP Eng",\n                status="approved",\n                evidence_links=["chk-overview-kpi-scan", "dec-overview-alert-stack"],\n                notes="Overview narrative approved for design sprint review.",\n                requested_at="2026-03-10T09:00:00Z",\n                due_at="2026-03-12T18:00:00Z",\n                escalation_owner="design-program-manager",\n                sla_status="met",\n            ),\n            ReviewSignoff(\n                signoff_id="sig-queue-platform-admin",\n                assignment_id="role-queue-platform-admin",\n                surface_id="wf-queue",\n                role="Platform Admin",\n                status="approved",\n                evidence_links=["chk-queue-batch-approval", "dec-queue-vp-summary"],\n                notes="Queue control actions meet operator review criteria.",\n                requested_at="2026-03-10T11:00:00Z",\n                due_at="2026-03-13T18:00:00Z",\n                escalation_owner="platform-ops-manager",\n                sla_status="met",\n            ),\n            ReviewSignoff(\n                signoff_id="sig-run-detail-eng-lead",\n                assignment_id="role-run-detail-eng-lead",\n                surface_id="wf-run-detail",\n                role="Eng Lead",\n                status="pending",\n                evidence_links=["chk-run-replay-context", "dec-run-detail-audit-rail"],\n                notes="Waiting for final replay-state copy review.",\n                requested_at="2026-03-12T11:00:00Z",\n                due_at="2026-03-15T18:00:00Z",\n                escalation_owner="engineering-director",\n                sla_status="at-risk",\n                reminder_owner="design-program-manager",\n                reminder_channel="slack",\n                last_reminder_at="2026-03-14T09:45:00Z",\n                next_reminder_at="2026-03-15T10:00:00Z",\n                reminder_cadence="daily",\n                reminder_status="scheduled",\n            ),\n            ReviewSignoff(\n                signoff_id="sig-triage-cross-team-operator",\n                assignment_id="role-triage-cross-team-operator",\n                surface_id="wf-triage",\n                role="Cross-Team Operator",\n                status="approved",\n                evidence_links=["chk-triage-handoff-clarity", "dec-triage-handoff-density"],\n                notes="Cross-team handoff flow approved for prototype review.",\n                requested_at="2026-03-11T14:00:00Z",\n                due_at="2026-03-13T12:00:00Z",\n                escalation_owner="cross-team-program-manager",\n                sla_status="met",\n            ),\n        ],\n        blocker_log=[\n            ReviewBlocker(\n                blocker_id="blk-run-detail-copy-final",\n                surface_id="wf-run-detail",\n                signoff_id="sig-run-detail-eng-lead",\n                owner="product-experience",\n                summary="Replay-state copy still needs final wording review before Eng Lead signoff can close.",\n                status="open",\n                severity="medium",\n                escalation_owner="design-program-manager",\n                next_action="Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",\n                freeze_exception=True,\n                freeze_owner="release-director",\n                freeze_until="2026-03-18T18:00:00Z",\n                freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",\n                freeze_approved_by="release-director",\n                freeze_approved_at="2026-03-14T08:30:00Z",\n                freeze_renewal_owner="release-director",\n                freeze_renewal_by="2026-03-17T12:00:00Z",\n                freeze_renewal_status="review-needed",\n            ),\n        ],\n        blocker_timeline=[\n            ReviewBlockerEvent(\n                event_id="evt-run-detail-copy-opened",\n                blocker_id="blk-run-detail-copy-final",\n                actor="product-experience",\n                status="opened",\n                summary="Captured the final replay-state copy gap during design sprint prep.",\n                timestamp="2026-03-13T10:00:00Z",\n                next_action="Draft updated replay labels before the Eng Lead review.",\n            ),\n            ReviewBlockerEvent(\n                event_id="evt-run-detail-copy-escalated",\n                blocker_id="blk-run-detail-copy-final",\n                actor="design-program-manager",\n                status="escalated",\n                summary="Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",\n                timestamp="2026-03-14T09:30:00Z",\n                next_action="Refresh the run-detail frame annotations after the wording review completes.",\n                handoff_from="product-experience",\n                handoff_to="Eng Lead",\n                channel="design-critique",\n                artifact_ref="wf-run-detail#copy-v5",\n                ack_owner="Eng Lead",\n                ack_at="2026-03-14T10:15:00Z",\n                ack_status="acknowledged",\n            ),\n        ],\n    )\n\n\n\n\ndef render_ui_review_decision_log(pack: UIReviewPack) -> str:\n    lines = [\n        "# UI Review Decision Log",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Decisions: {len(pack.decision_log)}",\n        "",\n        "## Decisions",\n    ]\n    for decision in pack.decision_log:\n        lines.append(\n            "- "\n            f"{decision.decision_id}: surface={decision.surface_id} owner={decision.owner} status={decision.status}"\n        )\n        lines.append(\n            "  "\n            f"summary={decision.summary} rationale={decision.rationale} follow_up={decision.follow_up or \'none\'}"\n        )\n    if not pack.decision_log:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\n\ndef render_ui_review_role_matrix(pack: UIReviewPack) -> str:\n    lines = [\n        "# UI Review Role Matrix",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Assignments: {len(pack.role_matrix)}",\n        "",\n        "## Assignments",\n    ]\n    for assignment in pack.role_matrix:\n        lines.append(\n            "- "\n            f"{assignment.assignment_id}: surface={assignment.surface_id} role={assignment.role} status={assignment.status}"\n        )\n        lines.append(\n            "  "\n            f"responsibilities={\',\'.join(assignment.responsibilities) or \'none\'} "\n            f"checklist={\',\'.join(assignment.checklist_item_ids) or \'none\'} "\n            f"decisions={\',\'.join(assignment.decision_ids) or \'none\'}"\n        )\n    if not pack.role_matrix:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\n\ndef render_ui_review_objective_coverage_board(pack: UIReviewPack) -> str:\n    entries = _build_objective_coverage_entries(pack)\n    persona_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        persona_counts[entry[\'persona\']] = persona_counts.get(entry[\'persona\'], 0) + 1\n        status_counts[entry[\'coverage_status\']] = status_counts.get(entry[\'coverage_status\'], 0) + 1\n    lines = [\n        "# UI Review Objective Coverage Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Objectives: {len(entries)}",\n        f"- Personas: {len(persona_counts)}",\n        "",\n        "## By Coverage Status",\n    ]\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Persona")\n    for persona, count in sorted(persona_counts.items()):\n        lines.append(f"- {persona}: {count}")\n    if not persona_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: objective={entry[\'objective_id\']} persona={entry[\'persona\']} priority={entry[\'priority\']} coverage={entry[\'coverage_status\']} dependencies={entry[\'dependency_count\']} surfaces={entry[\'surface_ids\']}"\n        )\n        lines.append(\n            f"  dependency_ids={entry[\'dependency_ids\']} assignments={entry[\'assignment_ids\']} checklist={entry[\'checklist_ids\']} decisions={entry[\'decision_ids\']} signoffs={entry[\'signoff_ids\']} blockers={entry[\'blocker_ids\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_wireframe_readiness_board(pack: UIReviewPack) -> str:\n    entries = _build_wireframe_readiness_entries(pack)\n    readiness_counts: Dict[str, int] = {}\n    device_counts: Dict[str, int] = {}\n    for entry in entries:\n        readiness_counts[entry[\'readiness_status\']] = readiness_counts.get(entry[\'readiness_status\'], 0) + 1\n        device_counts[entry[\'device\']] = device_counts.get(entry[\'device\'], 0) + 1\n    lines = [\n        "# UI Review Wireframe Readiness Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Wireframes: {len(entries)}",\n        f"- Devices: {len(device_counts)}",\n        "",\n        "## By Readiness",\n    ]\n    for status, count in sorted(readiness_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not readiness_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Device")\n    for device, count in sorted(device_counts.items()):\n        lines.append(f"- {device}: {count}")\n    if not device_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: surface={entry[\'surface_id\']} device={entry[\'device\']} readiness={entry[\'readiness_status\']} open_total={entry[\'open_total\']} entry={entry[\'entry_point\']}"\n        )\n        lines.append(\n            f"  checklist_open={entry[\'checklist_open\']} decisions_open={entry[\'decisions_open\']} assignments_open={entry[\'assignments_open\']} signoffs_open={entry[\'signoffs_open\']} blockers_open={entry[\'blockers_open\']} signoffs={entry[\'signoff_ids\']} blockers={entry[\'blocker_ids\']} blocks={entry[\'block_count\']} notes={entry[\'note_count\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_open_question_tracker(pack: UIReviewPack) -> str:\n    entries = _build_open_question_tracker_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    theme_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry[\'owner\']] = owner_counts.get(entry[\'owner\'], 0) + 1\n        theme_counts[entry[\'theme\']] = theme_counts.get(entry[\'theme\'], 0) + 1\n    lines = [\n        "# UI Review Open Question Tracker",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Questions: {len(entries)}",\n        f"- Owners: {len(owner_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Theme")\n    for theme, count in sorted(theme_counts.items()):\n        lines.append(f"- {theme}: {count}")\n    if not theme_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: question={entry[\'question_id\']} owner={entry[\'owner\']} theme={entry[\'theme\']} status={entry[\'status\']} link_status={entry[\'link_status\']} surfaces={entry[\'surface_ids\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_ids\']} flows={entry[\'flow_ids\']} impact={entry[\'impact\']} prompt={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_review_summary_board(pack: UIReviewPack) -> str:\n    entries = _build_review_summary_entries(pack)\n    lines = [\n        "# UI Review Review Summary Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Categories: {len(entries)}",\n        "",\n        "## Entries",\n    ]\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: category={entry[\'category\']} total={entry[\'total\']} {entry[\'metrics\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_persona_readiness_board(pack: UIReviewPack) -> str:\n    entries = _build_persona_readiness_entries(pack)\n    readiness_counts: Dict[str, int] = {}\n    for entry in entries:\n        readiness_counts[entry[\'readiness\']] = readiness_counts.get(entry[\'readiness\'], 0) + 1\n    lines = [\n        "# UI Review Persona Readiness Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Personas: {len(entries)}",\n        f"- Objectives: {len(pack.objectives)}",\n        "",\n        "## By Readiness",\n    ]\n    for readiness, count in sorted(readiness_counts.items()):\n        lines.append(f"- {readiness}: {count}")\n    if not readiness_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: persona={entry[\'persona\']} readiness={entry[\'readiness\']} objectives={entry[\'objective_count\']} assignments={entry[\'assignment_count\']} signoffs={entry[\'signoff_count\']} open_questions={entry[\'question_count\']} queue_items={entry[\'queue_count\']} blockers={entry[\'blocker_count\']}"\n        )\n        lines.append(\n            f"  objective_ids={entry[\'objective_ids\']} surfaces={entry[\'surface_ids\']} queue_ids={entry[\'queue_ids\']} blocker_ids={entry[\'blocker_ids\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_interaction_coverage_board(pack: UIReviewPack) -> str:\n    entries = _build_interaction_coverage_entries(pack)\n    coverage_counts: Dict[str, int] = {}\n    surface_counts: Dict[str, int] = {}\n    for entry in entries:\n        coverage_counts[entry[\'coverage_status\']] = coverage_counts.get(entry[\'coverage_status\'], 0) + 1\n        for surface_id in entry[\'surface_ids\'].split(\',\'):\n            if surface_id and surface_id != \'none\':\n                surface_counts[surface_id] = surface_counts.get(surface_id, 0) + 1\n    lines = [\n        "# UI Review Interaction Coverage Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Interactions: {len(entries)}",\n        f"- Surfaces: {len(surface_counts)}",\n        "",\n        "## By Coverage Status",\n    ]\n    for status, count in sorted(coverage_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not coverage_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Surface")\n    for surface_id, count in sorted(surface_counts.items()):\n        lines.append(f"- {surface_id}: {count}")\n    if not surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: flow={entry[\'flow_id\']} surfaces={entry[\'surface_ids\']} owners={entry[\'owners\']} coverage={entry[\'coverage_status\']} states={entry[\'state_count\']} exceptions={entry[\'exception_count\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_ids\']} open_checklist={entry[\'open_checklist_ids\']} trigger={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_checklist_traceability_board(pack: UIReviewPack) -> str:\n    entries = _build_checklist_traceability_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry[\'owner\']] = owner_counts.get(entry[\'owner\'], 0) + 1\n        status_counts[entry[\'status\']] = status_counts.get(entry[\'status\'], 0) + 1\n    lines = [\n        "# UI Review Checklist Traceability Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Checklist items: {len(entries)}",\n        f"- Owners: {len(owner_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: item={entry[\'item_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} linked_roles={entry[\'linked_roles\']}"\n        )\n        lines.append(\n            f"  linked_assignments={entry[\'linked_assignments\']} linked_decisions={entry[\'linked_decisions\']} evidence={entry[\'evidence\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_decision_followup_tracker(pack: UIReviewPack) -> str:\n    entries = _build_decision_followup_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry[\'owner\']] = owner_counts.get(entry[\'owner\'], 0) + 1\n        status_counts[entry[\'status\']] = status_counts.get(entry[\'status\'], 0) + 1\n    lines = [\n        "# UI Review Decision Follow-up Tracker",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Decisions: {len(entries)}",\n        f"- Owners: {len(owner_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: decision={entry[\'decision_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} linked_roles={entry[\'linked_roles\']}"\n        )\n        lines.append(\n            f"  linked_assignments={entry[\'linked_assignments\']} linked_checklists={entry[\'linked_checklists\']} follow_up={entry[\'follow_up\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_role_coverage_board(pack: UIReviewPack) -> str:\n    entries = _build_role_coverage_entries(pack)\n    surface_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        surface_counts[entry[\'surface_id\']] = surface_counts.get(entry[\'surface_id\'], 0) + 1\n        status_counts[entry[\'status\']] = status_counts.get(entry[\'status\'], 0) + 1\n    lines = [\n        "# UI Review Role Coverage Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Assignments: {len(entries)}",\n        f"- Surfaces: {len(surface_counts)}",\n        "",\n        "## By Surface",\n    ]\n    for surface_id, count in sorted(surface_counts.items()):\n        lines.append(f"- {surface_id}: {count}")\n    if not surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: assignment={entry[\'assignment_id\']} surface={entry[\'surface_id\']} role={entry[\'role\']} status={entry[\'status\']} responsibilities={entry[\'responsibility_count\']} checklist={entry[\'checklist_count\']} decisions={entry[\'decision_count\']}"\n        )\n        lines.append(\n            f"  signoff={entry[\'signoff_id\']} signoff_status={entry[\'signoff_status\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_owner_workload_board(pack: UIReviewPack) -> str:\n    entries = _build_owner_workload_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    for entry in entries:\n        counts = owner_counts.setdefault(\n            entry["owner"],\n            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "reminder": 0, "renewal": 0, "total": 0},\n        )\n        counts[entry["item_type"]] += 1\n        counts["total"] += 1\n    lines = [\n        "# UI Review Owner Workload Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Owners: {len(owner_counts)}",\n        f"- Items: {len(entries)}",\n        "",\n        "## Owners",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} checklist={counts[\'checklist\']} decisions={counts[\'decision\']} signoffs={counts[\'signoff\']} reminders={counts[\'reminder\']} renewals={counts[\'renewal\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} lane={entry[\'lane\']}"\n        )\n        lines.append(f"  detail={entry[\'detail\']} summary={entry[\'summary\']}")\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_signoff_dependency_board(pack: UIReviewPack) -> str:\n    entries = _build_signoff_dependency_entries(pack)\n    dependency_counts: Dict[str, int] = {}\n    sla_counts: Dict[str, int] = {}\n    for entry in entries:\n        dependency_counts[entry[\'dependency_status\']] = dependency_counts.get(entry[\'dependency_status\'], 0) + 1\n        sla_counts[entry[\'sla_status\']] = sla_counts.get(entry[\'sla_status\'], 0) + 1\n    lines = [\n        "# UI Review Signoff Dependency Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Sign-offs: {len(entries)}",\n        f"- Dependency states: {len(dependency_counts)}",\n        "",\n        "## By Dependency Status",\n    ]\n    for status, count in sorted(dependency_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not dependency_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By SLA State")\n    for status, count in sorted(sla_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not sla_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} surface={entry[\'surface_id\']} role={entry[\'role\']} status={entry[\'status\']} dependency_status={entry[\'dependency_status\']} blockers={entry[\'blocker_ids\']}"\n        )\n        lines.append(\n            f"  assignment={entry[\'assignment_id\']} checklist={entry[\'checklist_ids\']} decisions={entry[\'decision_ids\']} latest_blocker_event={entry[\'latest_blocker_event\']} sla={entry[\'sla_status\']} due_at={entry[\'due_at\']} cadence={entry[\'reminder_cadence\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_audit_density_board(pack: UIReviewPack) -> str:\n    entries = _build_audit_density_entries(pack)\n    band_counts: Dict[str, int] = {}\n    for entry in entries:\n        band_counts[entry[\'load_band\']] = band_counts.get(entry[\'load_band\'], 0) + 1\n    lines = [\n        "# UI Review Audit Density Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Surfaces: {len(entries)}",\n        f"- Load bands: {len(band_counts)}",\n        "",\n        "## By Load Band",\n    ]\n    for band, count in sorted(band_counts.items()):\n        lines.append(f"- {band}: {count}")\n    if not band_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: surface={entry[\'surface_id\']} artifact_total={entry[\'artifact_total\']} open_total={entry[\'open_total\']} band={entry[\'load_band\']}"\n        )\n        lines.append(\n            f"  checklist={entry[\'checklist_count\']} decisions={entry[\'decision_count\']} assignments={entry[\'assignment_count\']} signoffs={entry[\'signoff_count\']} blockers={entry[\'blocker_count\']} timeline={entry[\'timeline_count\']} blocks={entry[\'block_count\']} notes={entry[\'note_count\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_signoff_log(pack: UIReviewPack) -> str:\n    lines = [\n        "# UI Review Sign-off Log",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Sign-offs: {len(pack.signoff_log)}",\n        "",\n        "## Sign-offs",\n    ]\n    for signoff in pack.signoff_log:\n        lines.append(\n            "- "\n            f"{signoff.signoff_id}: surface={signoff.surface_id} role={signoff.role} assignment={signoff.assignment_id} status={signoff.status}"\n        )\n        lines.append(\n            "  "\n            f"required={\'yes\' if signoff.required else \'no\'} evidence={\',\'.join(signoff.evidence_links) or \'none\'} notes={signoff.notes or \'none\'} waiver_owner={signoff.waiver_owner or \'none\'} waiver_reason={signoff.waiver_reason or \'none\'} requested_at={signoff.requested_at or \'none\'} due_at={signoff.due_at or \'none\'} escalation_owner={signoff.escalation_owner or \'none\'} sla_status={signoff.sla_status} reminder_owner={signoff.reminder_owner or \'none\'} reminder_channel={signoff.reminder_channel or \'none\'} last_reminder_at={signoff.last_reminder_at or \'none\'} next_reminder_at={signoff.next_reminder_at or \'none\'}"\n        )\n    if not pack.signoff_log:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_signoff_sla_dashboard(pack: UIReviewPack) -> str:\n    entries = _build_signoff_sla_entries(pack)\n    state_counts: Dict[str, int] = {}\n    owner_counts: Dict[str, int] = {}\n    for entry in entries:\n        state_counts[entry[\'sla_status\']] = state_counts.get(entry[\'sla_status\'], 0) + 1\n        owner_counts[entry[\'escalation_owner\']] = owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n    lines = [\n        "# UI Review Sign-off SLA Dashboard",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Sign-offs: {len(entries)}",\n        f"- Escalation owners: {len(owner_counts)}",\n        "",\n        "## SLA States",\n    ]\n    for sla_status, count in sorted(state_counts.items()):\n        lines.append(f"- {sla_status}: {count}")\n    if not state_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Escalation Owners")\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Sign-offs")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'signoff_id\']}: role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} requested_at={entry[\'requested_at\']} due_at={entry[\'due_at\']} escalation_owner={entry[\'escalation_owner\']}"\n        )\n        lines.append(f"  required={entry[\'required\']} evidence={entry[\'evidence\']}")\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_signoff_reminder_queue(pack: UIReviewPack) -> str:\n    entries = _build_signoff_reminder_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    channel_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry["reminder_owner"]] = owner_counts.get(entry["reminder_owner"], 0) + 1\n        channel_counts[entry["reminder_channel"]] = channel_counts.get(entry["reminder_channel"], 0) + 1\n    lines = [\n        "# UI Review Sign-off Reminder Queue",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Reminders: {len(entries)}",\n        f"- Owners: {len(owner_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: reminders={count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Channel")\n    for channel, count in sorted(channel_counts.items()):\n        lines.append(f"- {channel}: {count}")\n    if not channel_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} owner={entry[\'reminder_owner\']} channel={entry[\'reminder_channel\']}"\n        )\n        lines.append(\n            f"  last_reminder_at={entry[\'last_reminder_at\']} next_reminder_at={entry[\'next_reminder_at\']} due_at={entry[\'due_at\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_reminder_cadence_board(pack: UIReviewPack) -> str:\n    entries = _build_reminder_cadence_entries(pack)\n    cadence_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        cadence_counts[entry["reminder_cadence"]] = cadence_counts.get(entry["reminder_cadence"], 0) + 1\n        status_counts[entry["reminder_status"]] = status_counts.get(entry["reminder_status"], 0) + 1\n    lines = [\n        "# UI Review Reminder Cadence Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Items: {len(entries)}",\n        f"- Cadences: {len(cadence_counts)}",\n        "",\n        "## By Cadence",\n    ]\n    for cadence, count in sorted(cadence_counts.items()):\n        lines.append(f"- {cadence}: {count}")\n    if not cadence_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} cadence={entry[\'reminder_cadence\']} status={entry[\'reminder_status\']} owner={entry[\'reminder_owner\']}"\n        )\n        lines.append(\n            f"  sla={entry[\'sla_status\']} last_reminder_at={entry[\'last_reminder_at\']} next_reminder_at={entry[\'next_reminder_at\']} due_at={entry[\'due_at\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_escalation_dashboard(pack: UIReviewPack) -> str:\n    entries = _build_escalation_dashboard_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    status_counts: Dict[str, Dict[str, int]] = {}\n    for entry in entries:\n        owner_bucket = owner_counts.setdefault(\n            entry[\'escalation_owner\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        owner_bucket[entry[\'item_type\']] += 1\n        owner_bucket[\'total\'] += 1\n        status_bucket = status_counts.setdefault(\n            entry[\'status\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        status_bucket[entry[\'item_type\']] += 1\n        status_bucket[\'total\'] += 1\n    lines = [\n        "# UI Review Escalation Dashboard",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Items: {len(entries)}",\n        f"- Escalation owners: {len(owner_counts)}",\n        "",\n        "## By Escalation Owner",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, counts in sorted(status_counts.items()):\n        lines.append(\n            f"- {status}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Escalations")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'escalation_id\']}: owner={entry[\'escalation_owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} priority={entry[\'priority\']} current_owner={entry[\'current_owner\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} due_at={entry[\'due_at\']}")\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_signoff_breach_board(pack: UIReviewPack) -> str:\n    entries = _build_signoff_breach_entries(pack)\n    state_counts: Dict[str, int] = {}\n    owner_counts: Dict[str, int] = {}\n    for entry in entries:\n        state_counts[entry[\'sla_status\']] = state_counts.get(entry[\'sla_status\'], 0) + 1\n        owner_counts[entry[\'escalation_owner\']] = owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n    lines = [\n        "# UI Review Sign-off Breach Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Breach items: {len(entries)}",\n        f"- Escalation owners: {len(owner_counts)}",\n        "",\n        "## SLA States",\n    ]\n    for sla_status, count in sorted(state_counts.items()):\n        lines.append(f"- {sla_status}: {count}")\n    if not state_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Escalation Owners")\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: signoff={entry[\'signoff_id\']} role={entry[\'role\']} surface={entry[\'surface_id\']} status={entry[\'status\']} sla={entry[\'sla_status\']} escalation_owner={entry[\'escalation_owner\']}"\n        )\n        lines.append(\n            f"  requested_at={entry[\'requested_at\']} due_at={entry[\'due_at\']} linked_blockers={entry[\'linked_blockers\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_escalation_handoff_ledger(pack: UIReviewPack) -> str:\n    entries = _build_escalation_handoff_entries(pack)\n    channel_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        channel_counts[entry[\'channel\']] = channel_counts.get(entry[\'channel\'], 0) + 1\n        status_counts[entry[\'status\']] = status_counts.get(entry[\'status\'], 0) + 1\n    lines = [\n        "# UI Review Escalation Handoff Ledger",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Handoffs: {len(entries)}",\n        f"- Channels: {len(channel_counts)}",\n        "",\n        "## By Status",\n    ]\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Channel")\n    for channel, count in sorted(channel_counts.items()):\n        lines.append(f"- {channel}: {count}")\n    if not channel_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'ledger_id\']}: event={entry[\'event_id\']} blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} actor={entry[\'actor\']} status={entry[\'status\']} at={entry[\'timestamp\']}"\n        )\n        lines.append(\n            f"  from={entry[\'handoff_from\']} to={entry[\'handoff_to\']} channel={entry[\'channel\']} artifact={entry[\'artifact_ref\']} next_action={entry[\'next_action\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_handoff_ack_ledger(pack: UIReviewPack) -> str:\n    entries = _build_handoff_ack_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry[\'ack_owner\']] = owner_counts.get(entry[\'ack_owner\'], 0) + 1\n        status_counts[entry[\'ack_status\']] = status_counts.get(entry[\'ack_status\'], 0) + 1\n    lines = [\n        "# UI Review Handoff Ack Ledger",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Ack items: {len(entries)}",\n        f"- Ack owners: {len(owner_counts)}",\n        "",\n        "## By Ack Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Ack Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: event={entry[\'event_id\']} blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} handoff_to={entry[\'handoff_to\']} ack_owner={entry[\'ack_owner\']} ack_status={entry[\'ack_status\']} ack_at={entry[\'ack_at\']}"\n        )\n        lines.append(\n            f"  actor={entry[\'actor\']} status={entry[\'status\']} channel={entry[\'channel\']} artifact={entry[\'artifact_ref\']} summary={entry[\'summary\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_freeze_approval_trail(pack: UIReviewPack) -> str:\n    entries = _build_freeze_approval_entries(pack)\n    approver_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        approver_counts[entry["freeze_approved_by"]] = approver_counts.get(entry["freeze_approved_by"], 0) + 1\n        status_counts[entry["status"]] = status_counts.get(entry["status"], 0) + 1\n    lines = [\n        "# UI Review Freeze Approval Trail",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Approvals: {len(entries)}",\n        f"- Approvers: {len(approver_counts)}",\n        "",\n        "## By Approver",\n    ]\n    for owner, count in sorted(approver_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not approver_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} owner={entry[\'freeze_owner\']} approved_by={entry[\'freeze_approved_by\']} approved_at={entry[\'freeze_approved_at\']} window={entry[\'freeze_until\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} latest_event={entry[\'latest_event\']} next_action={entry[\'next_action\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_freeze_renewal_tracker(pack: UIReviewPack) -> str:\n    entries = _build_freeze_renewal_entries(pack)\n    owner_counts: Dict[str, int] = {}\n    status_counts: Dict[str, int] = {}\n    for entry in entries:\n        owner_counts[entry[\'renewal_owner\']] = owner_counts.get(entry[\'renewal_owner\'], 0) + 1\n        status_counts[entry[\'renewal_status\']] = status_counts.get(entry[\'renewal_status\'], 0) + 1\n    lines = [\n        "# UI Review Freeze Renewal Tracker",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Renewal items: {len(entries)}",\n        f"- Renewal owners: {len(owner_counts)}",\n        "",\n        "## By Renewal Owner",\n    ]\n    for owner, count in sorted(owner_counts.items()):\n        lines.append(f"- {owner}: {count}")\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Renewal Status")\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: blocker={entry[\'blocker_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} renewal_owner={entry[\'renewal_owner\']} renewal_by={entry[\'renewal_by\']} renewal_status={entry[\'renewal_status\']}"\n        )\n        lines.append(\n            f"  freeze_owner={entry[\'freeze_owner\']} freeze_until={entry[\'freeze_until\']} approved_by={entry[\'freeze_approved_by\']} summary={entry[\'summary\']} next_action={entry[\'next_action\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_freeze_exception_board(pack: UIReviewPack) -> str:\n    entries = _build_freeze_exception_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in entries:\n        owner_bucket = owner_counts.setdefault(entry[\'owner\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0})\n        owner_bucket[entry[\'item_type\']] += 1\n        owner_bucket[\'total\'] += 1\n        surface_bucket = surface_counts.setdefault(entry[\'surface_id\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0})\n        surface_bucket[entry[\'item_type\']] += 1\n        surface_bucket[\'total\'] += 1\n    lines = [\n        "# UI Review Freeze Exception Board",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Exceptions: {len(entries)}",\n        f"- Owners: {len(owner_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Surface")\n    for surface_id, counts in sorted(surface_counts.items()):\n        lines.append(\n            f"- {surface_id}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'entry_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} window={entry[\'window\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} evidence={entry[\'evidence\']} next_action={entry[\'next_action\']}"\n        )\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_owner_escalation_digest(pack: UIReviewPack) -> str:\n    entries = _build_owner_escalation_digest_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    for entry in entries:\n        counts = owner_counts.setdefault(\n            entry["owner"],\n            {"blocker": 0, "signoff": 0, "reminder": 0, "freeze": 0, "handoff": 0, "total": 0},\n        )\n        counts[entry["item_type"]] += 1\n        counts["total"] += 1\n    lines = [\n        "# UI Review Owner Escalation Digest",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Owners: {len(owner_counts)}",\n        f"- Items: {len(entries)}",\n        "",\n        "## Owners",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} reminders={counts[\'reminder\']} freezes={counts[\'freeze\']} handoffs={counts[\'handoff\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in entries:\n        lines.append(\n            f"- {entry[\'digest_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} detail={entry[\'detail\']}")\n    if not entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_blocker_log(pack: UIReviewPack) -> str:\n    lines = [\n        "# UI Review Blocker Log",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Blockers: {len(pack.blocker_log)}",\n        "",\n        "## Blockers",\n    ]\n    for blocker in pack.blocker_log:\n        lines.append(\n            "- "\n            f"{blocker.blocker_id}: surface={blocker.surface_id} signoff={blocker.signoff_id} owner={blocker.owner} status={blocker.status} severity={blocker.severity}"\n        )\n        lines.append(\n            "  "\n            f"summary={blocker.summary} escalation_owner={blocker.escalation_owner or \'none\'} next_action={blocker.next_action or \'none\'} freeze_owner={blocker.freeze_owner or \'none\'} freeze_until={blocker.freeze_until or \'none\'} freeze_approved_by={blocker.freeze_approved_by or \'none\'} freeze_approved_at={blocker.freeze_approved_at or \'none\'}"\n        )\n    if not pack.blocker_log:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_blocker_timeline(pack: UIReviewPack) -> str:\n    lines = [\n        "# UI Review Blocker Timeline",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Events: {len(pack.blocker_timeline)}",\n        "",\n        "## Events",\n    ]\n    for event in pack.blocker_timeline:\n        lines.append(\n            "- "\n            f"{event.event_id}: blocker={event.blocker_id} actor={event.actor} status={event.status} at={event.timestamp}"\n        )\n        lines.append(\n            "  "\n            f"summary={event.summary} next_action={event.next_action or \'none\'}"\n        )\n    if not pack.blocker_timeline:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_exception_log(pack: UIReviewPack) -> str:\n    exception_entries = _build_review_exception_entries(pack)\n    lines = [\n        "# UI Review Exception Log",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Exceptions: {len(exception_entries)}",\n        "",\n        "## Exceptions",\n    ]\n    for entry in exception_entries:\n        lines.append(\n            "- "\n            f"{entry[\'exception_id\']}: type={entry[\'category\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} owner={entry[\'owner\']} status={entry[\'status\']} severity={entry[\'severity\']}"\n        )\n        lines.append(\n            "  "\n            f"summary={entry[\'summary\']} evidence={entry[\'evidence\']} latest_event={entry[\'latest_event\']} next_action={entry[\'next_action\']}"\n        )\n    if not exception_entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_exception_matrix(pack: UIReviewPack) -> str:\n    exception_entries = _build_review_exception_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    status_counts: Dict[str, Dict[str, int]] = {}\n    surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in exception_entries:\n        owner_bucket = owner_counts.setdefault(\n            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        owner_bucket[entry["category"]] += 1\n        owner_bucket["total"] += 1\n        status_bucket = status_counts.setdefault(\n            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        status_bucket[entry["category"]] += 1\n        status_bucket["total"] += 1\n        surface_bucket = surface_counts.setdefault(\n            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        surface_bucket[entry["category"]] += 1\n        surface_bucket["total"] += 1\n    lines = [\n        "# UI Review Exception Matrix",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Exceptions: {len(exception_entries)}",\n        f"- Owners: {len(owner_counts)}",\n        f"- Surfaces: {len(surface_counts)}",\n        "",\n        "## By Owner",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Status")\n    for status, counts in sorted(status_counts.items()):\n        lines.append(\n            f"- {status}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## By Surface")\n    for surface_id, counts in sorted(surface_counts.items()):\n        lines.append(\n            f"- {surface_id}: blockers={counts[\'blocker\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not surface_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Entries")\n    for entry in exception_entries:\n        lines.append(\n            f"- {entry[\'exception_id\']}: owner={entry[\'owner\']} type={entry[\'category\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']} severity={entry[\'severity\']}"\n        )\n        lines.append(\n            f"  summary={entry[\'summary\']} latest_event={entry[\'latest_event\']} next_action={entry[\'next_action\']}"\n        )\n    if not exception_entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_owner_review_queue(pack: UIReviewPack) -> str:\n    queue_entries = _build_owner_review_queue_entries(pack)\n    owner_counts: Dict[str, Dict[str, int]] = {}\n    for entry in queue_entries:\n        counts = owner_counts.setdefault(\n            entry["owner"],\n            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},\n        )\n        counts[entry["item_type"]] += 1\n        counts["total"] += 1\n    lines = [\n        "# UI Review Owner Review Queue",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Owners: {len(owner_counts)}",\n        f"- Queue items: {len(queue_entries)}",\n        "",\n        "## Owners",\n    ]\n    for owner, counts in sorted(owner_counts.items()):\n        lines.append(\n            f"- {owner}: blockers={counts[\'blocker\']} checklist={counts[\'checklist\']} decisions={counts[\'decision\']} signoffs={counts[\'signoff\']} total={counts[\'total\']}"\n        )\n    if not owner_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Items")\n    for entry in queue_entries:\n        lines.append(\n            f"- {entry[\'queue_id\']}: owner={entry[\'owner\']} type={entry[\'item_type\']} source={entry[\'source_id\']} surface={entry[\'surface_id\']} status={entry[\'status\']}"\n        )\n        lines.append(f"  summary={entry[\'summary\']} next_action={entry[\'next_action\']}")\n    if not queue_entries:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_blocker_timeline_summary(pack: UIReviewPack) -> str:\n    timeline_index = _build_blocker_timeline_index(pack)\n    status_counts: Dict[str, int] = {}\n    actor_counts: Dict[str, int] = {}\n    for event in pack.blocker_timeline:\n        status_counts[event.status] = status_counts.get(event.status, 0) + 1\n        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1\n    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}\n    orphan_timeline_ids = sorted(\n        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids\n    )\n    lines = [\n        "# UI Review Blocker Timeline Summary",\n        "",\n        f"- Issue: {pack.issue_id} {pack.title}",\n        f"- Version: {pack.version}",\n        f"- Events: {len(pack.blocker_timeline)}",\n        f"- Blockers with timeline: {len(timeline_index)}",\n        f"- Orphan timeline blockers: {\',\'.join(orphan_timeline_ids) or \'none\'}",\n        "",\n        "## Events by Status",\n    ]\n    for status, count in sorted(status_counts.items()):\n        lines.append(f"- {status}: {count}")\n    if not status_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Events by Actor")\n    for actor, count in sorted(actor_counts.items()):\n        lines.append(f"- {actor}: {count}")\n    if not actor_counts:\n        lines.append("- none")\n    lines.append("")\n    lines.append("## Latest Blocker Events")\n    for blocker in pack.blocker_log:\n        latest_events = timeline_index.get(blocker.blocker_id, [])\n        latest = latest_events[-1] if latest_events else None\n        if latest is None:\n            lines.append(f"- {blocker.blocker_id}: latest=none")\n            continue\n        lines.append(\n            f"- {blocker.blocker_id}: latest={latest.event_id} actor={latest.actor} status={latest.status} at={latest.timestamp}"\n        )\n    if not pack.blocker_log:\n        lines.append("- none")\n    return "\\n".join(lines)\n\n\ndef render_ui_review_pack_html(pack: UIReviewPack, audit: UIReviewPackAudit) -> str:\n    objective_html = "".join(\n        f"<li><strong>{escape(objective.objective_id)}</strong> · {escape(objective.title)} · persona={escape(objective.persona)} · priority={escape(objective.priority)}<br /><span>{escape(objective.success_signal)}</span></li>"\n        for objective in pack.objectives\n    ) or "<li>none</li>"\n    wireframe_html = "".join(\n        f"<li><strong>{escape(wireframe.surface_id)}</strong> · {escape(wireframe.name)} · entry={escape(wireframe.entry_point)}<br /><span>blocks={escape(\', \'.join(wireframe.primary_blocks) if wireframe.primary_blocks else \'none\')}</span></li>"\n        for wireframe in pack.wireframes\n    ) or "<li>none</li>"\n    interaction_html = "".join(\n        f"<li><strong>{escape(interaction.flow_id)}</strong> · {escape(interaction.name)}<br /><span>states={escape(\', \'.join(interaction.states) if interaction.states else \'none\')}</span></li>"\n        for interaction in pack.interactions\n    ) or "<li>none</li>"\n    interaction_coverage_entries = _build_interaction_coverage_entries(pack)\n    interaction_coverage_counts: Dict[str, int] = {}\n    interaction_surface_counts: Dict[str, int] = {}\n    for entry in interaction_coverage_entries:\n        interaction_coverage_counts[entry[\'coverage_status\']] = interaction_coverage_counts.get(entry[\'coverage_status\'], 0) + 1\n        for surface_id in entry[\'surface_ids\'].split(\',\'):\n            if surface_id and surface_id != \'none\':\n                interaction_surface_counts[surface_id] = interaction_surface_counts.get(surface_id, 0) + 1\n    interaction_coverage_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(interaction_coverage_counts.items())\n    ) or "<li>none</li>"\n    interaction_coverage_surface_html = "".join(\n        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"\n        for surface_id, count in sorted(interaction_surface_counts.items())\n    ) or "<li>none</li>"\n    interaction_coverage_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · flow={escape(entry[\'flow_id\'])} · surfaces={escape(entry[\'surface_ids\'])} · owners={escape(entry[\'owners\'])} · coverage={escape(entry[\'coverage_status\'])}<br /><span>states={escape(entry[\'state_count\'])} · exceptions={escape(entry[\'exception_count\'])}</span><br /><span>checklist={escape(entry[\'checklist_ids\'])} · open_checklist={escape(entry[\'open_checklist_ids\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in interaction_coverage_entries\n    ) or "<li>none</li>"\n    question_html = "".join(\n        f"<li><strong>{escape(question.question_id)}</strong> · {escape(question.theme)} · owner={escape(question.owner)} · status={escape(question.status)}<br /><span>{escape(question.question)}</span></li>"\n        for question in pack.open_questions\n    ) or "<li>none</li>"\n    checklist_html = "".join(\n        f"<li><strong>{escape(item.item_id)}</strong> · surface={escape(item.surface_id)} · owner={escape(item.owner)} · status={escape(item.status)}<br /><span>{escape(item.prompt)}</span><br /><span>evidence={escape(\', \'.join(item.evidence_links) if item.evidence_links else \'none\')}</span></li>"\n        for item in pack.reviewer_checklist\n    ) or "<li>none</li>"\n    decision_html = "".join(\n        f"<li><strong>{escape(decision.decision_id)}</strong> · surface={escape(decision.surface_id)} · owner={escape(decision.owner)} · status={escape(decision.status)}<br /><span>{escape(decision.summary)}</span><br /><span>follow_up={escape(decision.follow_up or \'none\')}</span></li>"\n        for decision in pack.decision_log\n    ) or "<li>none</li>"\n    role_matrix_html = "".join(\n        f"<li><strong>{escape(assignment.assignment_id)}</strong> · surface={escape(assignment.surface_id)} · role={escape(assignment.role)} · status={escape(assignment.status)}<br /><span>responsibilities={escape(\', \'.join(assignment.responsibilities) if assignment.responsibilities else \'none\')}</span><br /><span>checklist={escape(\', \'.join(assignment.checklist_item_ids) if assignment.checklist_item_ids else \'none\')} · decisions={escape(\', \'.join(assignment.decision_ids) if assignment.decision_ids else \'none\')}</span></li>"\n        for assignment in pack.role_matrix\n    ) or "<li>none</li>"\n    objective_coverage_entries = _build_objective_coverage_entries(pack)\n    objective_coverage_status_counts: Dict[str, int] = {}\n    objective_coverage_persona_counts: Dict[str, int] = {}\n    for entry in objective_coverage_entries:\n        objective_coverage_status_counts[entry[\'coverage_status\']] = objective_coverage_status_counts.get(entry[\'coverage_status\'], 0) + 1\n        objective_coverage_persona_counts[entry[\'persona\']] = objective_coverage_persona_counts.get(entry[\'persona\'], 0) + 1\n    objective_coverage_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(objective_coverage_status_counts.items())\n    ) or "<li>none</li>"\n    objective_coverage_persona_html = "".join(\n        f"<li><strong>{escape(persona)}</strong> · count={count}</li>"\n        for persona, count in sorted(objective_coverage_persona_counts.items())\n    ) or "<li>none</li>"\n    objective_coverage_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · objective={escape(entry[\'objective_id\'])} · persona={escape(entry[\'persona\'])} · priority={escape(entry[\'priority\'])} · coverage={escape(entry[\'coverage_status\'])}<br /><span>dependencies={escape(entry[\'dependency_count\'])} · surfaces={escape(entry[\'surface_ids\'])} · blockers={escape(entry[\'blocker_ids\'])}</span><br /><span>assignments={escape(entry[\'assignment_ids\'])} · checklist={escape(entry[\'checklist_ids\'])} · decisions={escape(entry[\'decision_ids\'])}</span><br /><span>signoffs={escape(entry[\'signoff_ids\'])} · dependency_ids={escape(entry[\'dependency_ids\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in objective_coverage_entries\n    ) or "<li>none</li>"\n    review_summary_entries = _build_review_summary_entries(pack)\n    review_summary_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · category={escape(entry[\'category\'])} · total={escape(entry[\'total\'])} · {escape(entry[\'metrics\'])}</li>"\n        for entry in review_summary_entries\n    ) or "<li>none</li>"\n    persona_readiness_entries = _build_persona_readiness_entries(pack)\n    persona_readiness_counts: Dict[str, int] = {}\n    for entry in persona_readiness_entries:\n        persona_readiness_counts[entry[\'readiness\']] = persona_readiness_counts.get(entry[\'readiness\'], 0) + 1\n    persona_readiness_status_html = "".join(\n        f"<li><strong>{escape(readiness)}</strong> · count={count}</li>"\n        for readiness, count in sorted(persona_readiness_counts.items())\n    ) or "<li>none</li>"\n    persona_readiness_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · persona={escape(entry[\'persona\'])} · readiness={escape(entry[\'readiness\'])} · objectives={escape(entry[\'objective_count\'])}<br /><span>assignments={escape(entry[\'assignment_count\'])} · signoffs={escape(entry[\'signoff_count\'])} · open_questions={escape(entry[\'question_count\'])} · queue_items={escape(entry[\'queue_count\'])} · blockers={escape(entry[\'blocker_count\'])}</span><br /><span>objective_ids={escape(entry[\'objective_ids\'])} · surfaces={escape(entry[\'surface_ids\'])}</span><br /><span>queue_ids={escape(entry[\'queue_ids\'])} · blocker_ids={escape(entry[\'blocker_ids\'])}</span></li>"\n        for entry in persona_readiness_entries\n    ) or "<li>none</li>"\n    wireframe_readiness_entries = _build_wireframe_readiness_entries(pack)\n    wireframe_readiness_counts: Dict[str, int] = {}\n    wireframe_device_counts: Dict[str, int] = {}\n    for entry in wireframe_readiness_entries:\n        wireframe_readiness_counts[entry[\'readiness_status\']] = wireframe_readiness_counts.get(entry[\'readiness_status\'], 0) + 1\n        wireframe_device_counts[entry[\'device\']] = wireframe_device_counts.get(entry[\'device\'], 0) + 1\n    wireframe_readiness_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(wireframe_readiness_counts.items())\n    ) or "<li>none</li>"\n    wireframe_device_html = "".join(\n        f"<li><strong>{escape(device)}</strong> · count={count}</li>"\n        for device, count in sorted(wireframe_device_counts.items())\n    ) or "<li>none</li>"\n    wireframe_readiness_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · surface={escape(entry[\'surface_id\'])} · device={escape(entry[\'device\'])} · readiness={escape(entry[\'readiness_status\'])} · open_total={escape(entry[\'open_total\'])}<br /><span>entry={escape(entry[\'entry_point\'])} · signoffs={escape(entry[\'signoff_ids\'])} · blockers={escape(entry[\'blocker_ids\'])}</span><br /><span>checklist_open={escape(entry[\'checklist_open\'])} · decisions_open={escape(entry[\'decisions_open\'])} · assignments_open={escape(entry[\'assignments_open\'])}</span><br /><span>signoffs_open={escape(entry[\'signoffs_open\'])} · blockers_open={escape(entry[\'blockers_open\'])} · blocks={escape(entry[\'block_count\'])} · notes={escape(entry[\'note_count\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in wireframe_readiness_entries\n    ) or "<li>none</li>"\n    open_question_entries = _build_open_question_tracker_entries(pack)\n    open_question_owner_counts: Dict[str, int] = {}\n    open_question_theme_counts: Dict[str, int] = {}\n    for entry in open_question_entries:\n        open_question_owner_counts[entry[\'owner\']] = open_question_owner_counts.get(entry[\'owner\'], 0) + 1\n        open_question_theme_counts[entry[\'theme\']] = open_question_theme_counts.get(entry[\'theme\'], 0) + 1\n    open_question_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(open_question_owner_counts.items())\n    ) or "<li>none</li>"\n    open_question_theme_html = "".join(\n        f"<li><strong>{escape(theme)}</strong> · count={count}</li>"\n        for theme, count in sorted(open_question_theme_counts.items())\n    ) or "<li>none</li>"\n    open_question_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · question={escape(entry[\'question_id\'])} · owner={escape(entry[\'owner\'])} · theme={escape(entry[\'theme\'])} · status={escape(entry[\'status\'])}<br /><span>link_status={escape(entry[\'link_status\'])} · surfaces={escape(entry[\'surface_ids\'])} · checklist={escape(entry[\'checklist_ids\'])}</span><br /><span>flows={escape(entry[\'flow_ids\'])}</span><br /><span>impact={escape(entry[\'impact\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in open_question_entries\n    ) or "<li>none</li>"\n    checklist_trace_entries = _build_checklist_traceability_entries(pack)\n    checklist_trace_owner_counts: Dict[str, int] = {}\n    checklist_trace_status_counts: Dict[str, int] = {}\n    for entry in checklist_trace_entries:\n        checklist_trace_owner_counts[entry[\'owner\']] = checklist_trace_owner_counts.get(entry[\'owner\'], 0) + 1\n        checklist_trace_status_counts[entry[\'status\']] = checklist_trace_status_counts.get(entry[\'status\'], 0) + 1\n    checklist_trace_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(checklist_trace_owner_counts.items())\n    ) or "<li>none</li>"\n    checklist_trace_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(checklist_trace_status_counts.items())\n    ) or "<li>none</li>"\n    checklist_trace_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · item={escape(entry[\'item_id\'])} · surface={escape(entry[\'surface_id\'])} · owner={escape(entry[\'owner\'])} · status={escape(entry[\'status\'])}<br /><span>linked_roles={escape(entry[\'linked_roles\'])} · linked_assignments={escape(entry[\'linked_assignments\'])}</span><br /><span>linked_decisions={escape(entry[\'linked_decisions\'])} · evidence={escape(entry[\'evidence\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in checklist_trace_entries\n    ) or "<li>none</li>"\n    decision_followup_entries = _build_decision_followup_entries(pack)\n    decision_followup_owner_counts: Dict[str, int] = {}\n    decision_followup_status_counts: Dict[str, int] = {}\n    for entry in decision_followup_entries:\n        decision_followup_owner_counts[entry[\'owner\']] = decision_followup_owner_counts.get(entry[\'owner\'], 0) + 1\n        decision_followup_status_counts[entry[\'status\']] = decision_followup_status_counts.get(entry[\'status\'], 0) + 1\n    decision_followup_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(decision_followup_owner_counts.items())\n    ) or "<li>none</li>"\n    decision_followup_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(decision_followup_status_counts.items())\n    ) or "<li>none</li>"\n    decision_followup_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · decision={escape(entry[\'decision_id\'])} · surface={escape(entry[\'surface_id\'])} · owner={escape(entry[\'owner\'])} · status={escape(entry[\'status\'])}<br /><span>linked_roles={escape(entry[\'linked_roles\'])} · linked_assignments={escape(entry[\'linked_assignments\'])}</span><br /><span>linked_checklists={escape(entry[\'linked_checklists\'])} · follow_up={escape(entry[\'follow_up\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in decision_followup_entries\n    ) or "<li>none</li>"\n    role_coverage_entries = _build_role_coverage_entries(pack)\n    role_coverage_surface_counts: Dict[str, int] = {}\n    role_coverage_status_counts: Dict[str, int] = {}\n    for entry in role_coverage_entries:\n        role_coverage_surface_counts[entry[\'surface_id\']] = role_coverage_surface_counts.get(entry[\'surface_id\'], 0) + 1\n        role_coverage_status_counts[entry[\'status\']] = role_coverage_status_counts.get(entry[\'status\'], 0) + 1\n    role_coverage_surface_html = "".join(\n        f"<li><strong>{escape(surface_id)}</strong> · count={count}</li>"\n        for surface_id, count in sorted(role_coverage_surface_counts.items())\n    ) or "<li>none</li>"\n    role_coverage_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(role_coverage_status_counts.items())\n    ) or "<li>none</li>"\n    role_coverage_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · assignment={escape(entry[\'assignment_id\'])} · surface={escape(entry[\'surface_id\'])} · role={escape(entry[\'role\'])} · status={escape(entry[\'status\'])}<br /><span>responsibilities={escape(entry[\'responsibility_count\'])} · checklist={escape(entry[\'checklist_count\'])} · decisions={escape(entry[\'decision_count\'])}</span><br /><span>signoff={escape(entry[\'signoff_id\'])} · signoff_status={escape(entry[\'signoff_status\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in role_coverage_entries\n    ) or "<li>none</li>"\n    signoff_dependency_entries = _build_signoff_dependency_entries(pack)\n    signoff_dependency_status_counts: Dict[str, int] = {}\n    signoff_dependency_sla_counts: Dict[str, int] = {}\n    for entry in signoff_dependency_entries:\n        signoff_dependency_status_counts[entry[\'dependency_status\']] = signoff_dependency_status_counts.get(entry[\'dependency_status\'], 0) + 1\n        signoff_dependency_sla_counts[entry[\'sla_status\']] = signoff_dependency_sla_counts.get(entry[\'sla_status\'], 0) + 1\n    signoff_dependency_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(signoff_dependency_status_counts.items())\n    ) or "<li>none</li>"\n    signoff_dependency_sla_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(signoff_dependency_sla_counts.items())\n    ) or "<li>none</li>"\n    signoff_dependency_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · signoff={escape(entry[\'signoff_id\'])} · surface={escape(entry[\'surface_id\'])} · role={escape(entry[\'role\'])} · status={escape(entry[\'status\'])} · dependency_status={escape(entry[\'dependency_status\'])}<br /><span>assignment={escape(entry[\'assignment_id\'])} · checklist={escape(entry[\'checklist_ids\'])} · decisions={escape(entry[\'decision_ids\'])}</span><br /><span>blockers={escape(entry[\'blocker_ids\'])} · latest_blocker_event={escape(entry[\'latest_blocker_event\'])}</span><br /><span>sla={escape(entry[\'sla_status\'])} · due_at={escape(entry[\'due_at\'])} · cadence={escape(entry[\'reminder_cadence\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in signoff_dependency_entries\n    ) or "<li>none</li>"\n    signoff_html = "".join(\n        f"<li><strong>{escape(signoff.signoff_id)}</strong> · surface={escape(signoff.surface_id)} · role={escape(signoff.role)} · status={escape(signoff.status)}<br /><span>assignment={escape(signoff.assignment_id)} · required={escape(\'yes\' if signoff.required else \'no\')}</span><br /><span>evidence={escape(\', \'.join(signoff.evidence_links) if signoff.evidence_links else \'none\')}</span><br /><span>waiver_owner={escape(signoff.waiver_owner or \'none\')} · waiver_reason={escape(signoff.waiver_reason or \'none\')}</span><br /><span>requested_at={escape(signoff.requested_at or \'none\')} · due_at={escape(signoff.due_at or \'none\')} · escalation_owner={escape(signoff.escalation_owner or \'none\')} · sla_status={escape(signoff.sla_status)}</span></li>"\n        for signoff in pack.signoff_log\n    ) or "<li>none</li>"\n    signoff_sla_entries = _build_signoff_sla_entries(pack)\n    signoff_sla_state_counts: Dict[str, int] = {}\n    signoff_sla_owner_counts: Dict[str, int] = {}\n    for entry in signoff_sla_entries:\n        signoff_sla_state_counts[entry[\'sla_status\']] = signoff_sla_state_counts.get(entry[\'sla_status\'], 0) + 1\n        signoff_sla_owner_counts[entry[\'escalation_owner\']] = signoff_sla_owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n    signoff_sla_state_html = "".join(\n        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"\n        for sla_status, count in sorted(signoff_sla_state_counts.items())\n    ) or "<li>none</li>"\n    signoff_sla_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(signoff_sla_owner_counts.items())\n    ) or "<li>none</li>"\n    signoff_sla_item_html = "".join(\n        f"<li><strong>{escape(entry[\'signoff_id\'])}</strong> · role={escape(entry[\'role\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · sla={escape(entry[\'sla_status\'])}<br /><span>requested_at={escape(entry[\'requested_at\'])} · due_at={escape(entry[\'due_at\'])} · escalation_owner={escape(entry[\'escalation_owner\'])}</span><br /><span>required={escape(entry[\'required\'])} · evidence={escape(entry[\'evidence\'])}</span></li>"\n        for entry in signoff_sla_entries\n    ) or "<li>none</li>"\n    signoff_reminder_entries = _build_signoff_reminder_entries(pack)\n    signoff_reminder_owner_counts: Dict[str, int] = {}\n    signoff_reminder_channel_counts: Dict[str, int] = {}\n    for entry in signoff_reminder_entries:\n        signoff_reminder_owner_counts[entry[\'reminder_owner\']] = signoff_reminder_owner_counts.get(entry[\'reminder_owner\'], 0) + 1\n        signoff_reminder_channel_counts[entry[\'reminder_channel\']] = signoff_reminder_channel_counts.get(entry[\'reminder_channel\'], 0) + 1\n    signoff_reminder_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · reminders={count}</li>"\n        for owner, count in sorted(signoff_reminder_owner_counts.items())\n    ) or "<li>none</li>"\n    signoff_reminder_channel_html = "".join(\n        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"\n        for channel, count in sorted(signoff_reminder_channel_counts.items())\n    ) or "<li>none</li>"\n    signoff_reminder_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · signoff={escape(entry[\'signoff_id\'])} · role={escape(entry[\'role\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · sla={escape(entry[\'sla_status\'])}<br /><span>owner={escape(entry[\'reminder_owner\'])} · channel={escape(entry[\'reminder_channel\'])}</span><br /><span>last_reminder_at={escape(entry[\'last_reminder_at\'])} · next_reminder_at={escape(entry[\'next_reminder_at\'])} · due_at={escape(entry[\'due_at\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in signoff_reminder_entries\n    ) or "<li>none</li>"\n    reminder_cadence_entries = _build_reminder_cadence_entries(pack)\n    reminder_cadence_counts: Dict[str, int] = {}\n    reminder_status_counts: Dict[str, int] = {}\n    for entry in reminder_cadence_entries:\n        reminder_cadence_counts[entry[\'reminder_cadence\']] = reminder_cadence_counts.get(entry[\'reminder_cadence\'], 0) + 1\n        reminder_status_counts[entry[\'reminder_status\']] = reminder_status_counts.get(entry[\'reminder_status\'], 0) + 1\n    reminder_cadence_owner_html = "".join(\n        f"<li><strong>{escape(cadence)}</strong> · count={count}</li>"\n        for cadence, count in sorted(reminder_cadence_counts.items())\n    ) or "<li>none</li>"\n    reminder_cadence_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(reminder_status_counts.items())\n    ) or "<li>none</li>"\n    reminder_cadence_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · signoff={escape(entry[\'signoff_id\'])} · role={escape(entry[\'role\'])} · surface={escape(entry[\'surface_id\'])} · cadence={escape(entry[\'reminder_cadence\'])} · status={escape(entry[\'reminder_status\'])}<br /><span>owner={escape(entry[\'reminder_owner\'])} · sla={escape(entry[\'sla_status\'])}</span><br /><span>last_reminder_at={escape(entry[\'last_reminder_at\'])} · next_reminder_at={escape(entry[\'next_reminder_at\'])} · due_at={escape(entry[\'due_at\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in reminder_cadence_entries\n    ) or "<li>none</li>"\n    signoff_breach_entries = _build_signoff_breach_entries(pack)\n    signoff_breach_state_counts: Dict[str, int] = {}\n    signoff_breach_owner_counts: Dict[str, int] = {}\n    for entry in signoff_breach_entries:\n        signoff_breach_state_counts[entry[\'sla_status\']] = signoff_breach_state_counts.get(entry[\'sla_status\'], 0) + 1\n        signoff_breach_owner_counts[entry[\'escalation_owner\']] = signoff_breach_owner_counts.get(entry[\'escalation_owner\'], 0) + 1\n    signoff_breach_state_html = "".join(\n        f"<li><strong>{escape(sla_status)}</strong> · count={count}</li>"\n        for sla_status, count in sorted(signoff_breach_state_counts.items())\n    ) or "<li>none</li>"\n    signoff_breach_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(signoff_breach_owner_counts.items())\n    ) or "<li>none</li>"\n    signoff_breach_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · signoff={escape(entry[\'signoff_id\'])} · role={escape(entry[\'role\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · sla={escape(entry[\'sla_status\'])}<br /><span>requested_at={escape(entry[\'requested_at\'])} · due_at={escape(entry[\'due_at\'])} · escalation_owner={escape(entry[\'escalation_owner\'])}</span><br /><span>linked_blockers={escape(entry[\'linked_blockers\'])} · summary={escape(entry[\'summary\'])}</span></li>"\n        for entry in signoff_breach_entries\n    ) or "<li>none</li>"\n    escalation_entries = _build_escalation_dashboard_entries(pack)\n    escalation_owner_counts: Dict[str, Dict[str, int]] = {}\n    escalation_status_counts: Dict[str, Dict[str, int]] = {}\n    for entry in escalation_entries:\n        owner_bucket = escalation_owner_counts.setdefault(\n            entry[\'escalation_owner\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        owner_bucket[entry[\'item_type\']] += 1\n        owner_bucket[\'total\'] += 1\n        status_bucket = escalation_status_counts.setdefault(\n            entry[\'status\'], {\'blocker\': 0, \'signoff\': 0, \'total\': 0}\n        )\n        status_bucket[entry[\'item_type\']] += 1\n        status_bucket[\'total\'] += 1\n    escalation_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(escalation_owner_counts.items())\n    ) or "<li>none</li>"\n    escalation_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for status, counts in sorted(escalation_status_counts.items())\n    ) or "<li>none</li>"\n    escalation_item_html = "".join(\n        f"<li><strong>{escape(entry[\'escalation_id\'])}</strong> · owner={escape(entry[\'escalation_owner\'])} · type={escape(entry[\'item_type\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · priority={escape(entry[\'priority\'])}<br /><span>current_owner={escape(entry[\'current_owner\'])} · due_at={escape(entry[\'due_at\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in escalation_entries\n    ) or "<li>none</li>"\n    escalation_handoff_entries = _build_escalation_handoff_entries(pack)\n    escalation_handoff_status_counts: Dict[str, int] = {}\n    escalation_handoff_channel_counts: Dict[str, int] = {}\n    for entry in escalation_handoff_entries:\n        escalation_handoff_status_counts[entry[\'status\']] = escalation_handoff_status_counts.get(entry[\'status\'], 0) + 1\n        escalation_handoff_channel_counts[entry[\'channel\']] = escalation_handoff_channel_counts.get(entry[\'channel\'], 0) + 1\n    escalation_handoff_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(escalation_handoff_status_counts.items())\n    ) or "<li>none</li>"\n    escalation_handoff_channel_html = "".join(\n        f"<li><strong>{escape(channel)}</strong> · count={count}</li>"\n        for channel, count in sorted(escalation_handoff_channel_counts.items())\n    ) or "<li>none</li>"\n    escalation_handoff_item_html = "".join(\n        f"<li><strong>{escape(entry[\'ledger_id\'])}</strong> · event={escape(entry[\'event_id\'])} · blocker={escape(entry[\'blocker_id\'])} · surface={escape(entry[\'surface_id\'])} · actor={escape(entry[\'actor\'])} · status={escape(entry[\'status\'])}<br /><span>from={escape(entry[\'handoff_from\'])} · to={escape(entry[\'handoff_to\'])} · channel={escape(entry[\'channel\'])}</span><br /><span>artifact={escape(entry[\'artifact_ref\'])} · next_action={escape(entry[\'next_action\'])} · at={escape(entry[\'timestamp\'])}</span></li>"\n        for entry in escalation_handoff_entries\n    ) or "<li>none</li>"\n    handoff_ack_entries = _build_handoff_ack_entries(pack)\n    handoff_ack_owner_counts: Dict[str, int] = {}\n    handoff_ack_status_counts: Dict[str, int] = {}\n    for entry in handoff_ack_entries:\n        handoff_ack_owner_counts[entry[\'ack_owner\']] = handoff_ack_owner_counts.get(entry[\'ack_owner\'], 0) + 1\n        handoff_ack_status_counts[entry[\'ack_status\']] = handoff_ack_status_counts.get(entry[\'ack_status\'], 0) + 1\n    handoff_ack_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(handoff_ack_owner_counts.items())\n    ) or "<li>none</li>"\n    handoff_ack_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(handoff_ack_status_counts.items())\n    ) or "<li>none</li>"\n    handoff_ack_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · event={escape(entry[\'event_id\'])} · blocker={escape(entry[\'blocker_id\'])} · surface={escape(entry[\'surface_id\'])} · handoff_to={escape(entry[\'handoff_to\'])}<br /><span>ack_owner={escape(entry[\'ack_owner\'])} · ack_status={escape(entry[\'ack_status\'])} · ack_at={escape(entry[\'ack_at\'])}</span><br /><span>actor={escape(entry[\'actor\'])} · channel={escape(entry[\'channel\'])} · artifact={escape(entry[\'artifact_ref\'])}</span><br /><span>{escape(entry[\'summary\'])}</span></li>"\n        for entry in handoff_ack_entries\n    ) or "<li>none</li>"\n    owner_escalation_entries = _build_owner_escalation_digest_entries(pack)\n    owner_escalation_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_escalation_entries:\n        counts = owner_escalation_counts.setdefault(\n            entry[\'owner\'],\n            {\'blocker\': 0, \'signoff\': 0, \'reminder\': 0, \'freeze\': 0, \'handoff\': 0, \'total\': 0},\n        )\n        counts[entry[\'item_type\']] += 1\n        counts[\'total\'] += 1\n    owner_escalation_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · reminders={counts[\'reminder\']} · freezes={counts[\'freeze\']} · handoffs={counts[\'handoff\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(owner_escalation_counts.items())\n    ) or "<li>none</li>"\n    owner_escalation_item_html = "".join(\n        f"<li><strong>{escape(entry[\'digest_id\'])}</strong> · owner={escape(entry[\'owner\'])} · type={escape(entry[\'item_type\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])}<br /><span>{escape(entry[\'summary\'])}</span><br /><span>detail={escape(entry[\'detail\'])}</span></li>"\n        for entry in owner_escalation_entries\n    ) or "<li>none</li>"\n    owner_workload_entries = _build_owner_workload_entries(pack)\n    owner_workload_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_workload_entries:\n        counts = owner_workload_counts.setdefault(\n            entry[\'owner\'],\n            {\'blocker\': 0, \'checklist\': 0, \'decision\': 0, \'signoff\': 0, \'reminder\': 0, \'renewal\': 0, \'total\': 0},\n        )\n        counts[entry[\'item_type\']] += 1\n        counts[\'total\'] += 1\n    owner_workload_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · checklist={counts[\'checklist\']} · decisions={counts[\'decision\']} · signoffs={counts[\'signoff\']} · reminders={counts[\'reminder\']} · renewals={counts[\'renewal\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(owner_workload_counts.items())\n    ) or "<li>none</li>"\n    owner_workload_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · owner={escape(entry[\'owner\'])} · type={escape(entry[\'item_type\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · lane={escape(entry[\'lane\'])}<br /><span>{escape(entry[\'summary\'])}</span><br /><span>detail={escape(entry[\'detail\'])}</span></li>"\n        for entry in owner_workload_entries\n    ) or "<li>none</li>"\n    blocker_html = "".join(\n        f"<li><strong>{escape(blocker.blocker_id)}</strong> · surface={escape(blocker.surface_id)} · signoff={escape(blocker.signoff_id)} · owner={escape(blocker.owner)} · status={escape(blocker.status)} · severity={escape(blocker.severity)}<br /><span>{escape(blocker.summary)}</span><br /><span>escalation_owner={escape(blocker.escalation_owner or \'none\')} · next_action={escape(blocker.next_action or \'none\')}</span></li>"\n        for blocker in pack.blocker_log\n    ) or "<li>none</li>"\n    blocker_timeline_html = "".join(\n        f"<li><strong>{escape(event.event_id)}</strong> · blocker={escape(event.blocker_id)} · actor={escape(event.actor)} · status={escape(event.status)}<br /><span>timestamp={escape(event.timestamp)}</span><br /><span>{escape(event.summary)}</span><br /><span>next_action={escape(event.next_action or \'none\')}</span></li>"\n        for event in pack.blocker_timeline\n    ) or "<li>none</li>"\n    timeline_index = _build_blocker_timeline_index(pack)\n    exception_entries = _build_review_exception_entries(pack)\n    exception_html = "".join(\n        f"<li><strong>{escape(entry[\'exception_id\'])}</strong> · owner={escape(entry[\'owner\'])} · type={escape(entry[\'category\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · severity={escape(entry[\'severity\'])}<br /><span>{escape(entry[\'summary\'])}</span><br /><span>latest_event={escape(entry[\'latest_event\'])} · next_action={escape(entry[\'next_action\'])}</span></li>"\n        for entry in exception_entries\n    ) or "<li>none</li>"\n    exception_owner_counts: Dict[str, Dict[str, int]] = {}\n    exception_status_counts: Dict[str, Dict[str, int]] = {}\n    exception_surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in exception_entries:\n        owner_bucket = exception_owner_counts.setdefault(\n            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        owner_bucket[entry["category"]] += 1\n        owner_bucket["total"] += 1\n        status_bucket = exception_status_counts.setdefault(\n            entry["status"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        status_bucket[entry["category"]] += 1\n        status_bucket["total"] += 1\n        surface_bucket = exception_surface_counts.setdefault(\n            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        surface_bucket[entry["category"]] += 1\n        surface_bucket["total"] += 1\n    exception_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(exception_owner_counts.items())\n    ) or "<li>none</li>"\n    exception_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for status, counts in sorted(exception_status_counts.items())\n    ) or "<li>none</li>"\n    exception_surface_html = "".join(\n        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for surface_id, counts in sorted(exception_surface_counts.items())\n    ) or "<li>none</li>"\n    audit_density_entries = _build_audit_density_entries(pack)\n    audit_density_band_counts: Dict[str, int] = {}\n    for entry in audit_density_entries:\n        audit_density_band_counts[entry[\'load_band\']] = audit_density_band_counts.get(entry[\'load_band\'], 0) + 1\n    audit_density_band_html = "".join(\n        f"<li><strong>{escape(band)}</strong> · count={count}</li>"\n        for band, count in sorted(audit_density_band_counts.items())\n    ) or "<li>none</li>"\n    audit_density_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · surface={escape(entry[\'surface_id\'])} · artifact_total={escape(entry[\'artifact_total\'])} · open_total={escape(entry[\'open_total\'])} · band={escape(entry[\'load_band\'])}<br /><span>checklist={escape(entry[\'checklist_count\'])} · decisions={escape(entry[\'decision_count\'])} · assignments={escape(entry[\'assignment_count\'])}</span><br /><span>signoffs={escape(entry[\'signoff_count\'])} · blockers={escape(entry[\'blocker_count\'])} · timeline={escape(entry[\'timeline_count\'])}</span><br /><span>blocks={escape(entry[\'block_count\'])} · notes={escape(entry[\'note_count\'])}</span></li>"\n        for entry in audit_density_entries\n    ) or "<li>none</li>"\n    freeze_entries = _build_freeze_exception_entries(pack)\n    freeze_owner_counts: Dict[str, Dict[str, int]] = {}\n    freeze_surface_counts: Dict[str, Dict[str, int]] = {}\n    for entry in freeze_entries:\n        owner_bucket = freeze_owner_counts.setdefault(\n            entry["owner"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        owner_bucket[entry["item_type"]] += 1\n        owner_bucket["total"] += 1\n        surface_bucket = freeze_surface_counts.setdefault(\n            entry["surface_id"], {"blocker": 0, "signoff": 0, "total": 0}\n        )\n        surface_bucket[entry["item_type"]] += 1\n        surface_bucket["total"] += 1\n    freeze_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(freeze_owner_counts.items())\n    ) or "<li>none</li>"\n    freeze_surface_html = "".join(\n        f"<li><strong>{escape(surface_id)}</strong> · blockers={counts[\'blocker\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for surface_id, counts in sorted(freeze_surface_counts.items())\n    ) or "<li>none</li>"\n    freeze_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · owner={escape(entry[\'owner\'])} · type={escape(entry[\'item_type\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])} · window={escape(entry[\'window\'])}<br /><span>{escape(entry[\'summary\'])}</span><br /><span>evidence={escape(entry[\'evidence\'])} · next_action={escape(entry[\'next_action\'])}</span></li>"\n        for entry in freeze_entries\n    ) or "<li>none</li>"\n    freeze_approval_entries = _build_freeze_approval_entries(pack)\n    freeze_approval_owner_counts: Dict[str, int] = {}\n    freeze_approval_status_counts: Dict[str, int] = {}\n    for entry in freeze_approval_entries:\n        freeze_approval_owner_counts[entry[\'freeze_approved_by\']] = freeze_approval_owner_counts.get(entry[\'freeze_approved_by\'], 0) + 1\n        freeze_approval_status_counts[entry[\'status\']] = freeze_approval_status_counts.get(entry[\'status\'], 0) + 1\n    freeze_approval_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(freeze_approval_owner_counts.items())\n    ) or "<li>none</li>"\n    freeze_approval_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(freeze_approval_status_counts.items())\n    ) or "<li>none</li>"\n    freeze_approval_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · blocker={escape(entry[\'blocker_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])}<br /><span>owner={escape(entry[\'freeze_owner\'])} · approved_by={escape(entry[\'freeze_approved_by\'])} · approved_at={escape(entry[\'freeze_approved_at\'])} · window={escape(entry[\'freeze_until\'])}</span><br /><span>{escape(entry[\'summary\'])}</span><br /><span>latest_event={escape(entry[\'latest_event\'])} · next_action={escape(entry[\'next_action\'])}</span></li>"\n        for entry in freeze_approval_entries\n    ) or "<li>none</li>"\n    freeze_renewal_entries = _build_freeze_renewal_entries(pack)\n    freeze_renewal_owner_counts: Dict[str, int] = {}\n    freeze_renewal_status_counts: Dict[str, int] = {}\n    for entry in freeze_renewal_entries:\n        freeze_renewal_owner_counts[entry[\'renewal_owner\']] = freeze_renewal_owner_counts.get(entry[\'renewal_owner\'], 0) + 1\n        freeze_renewal_status_counts[entry[\'renewal_status\']] = freeze_renewal_status_counts.get(entry[\'renewal_status\'], 0) + 1\n    freeze_renewal_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · count={count}</li>"\n        for owner, count in sorted(freeze_renewal_owner_counts.items())\n    ) or "<li>none</li>"\n    freeze_renewal_status_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(freeze_renewal_status_counts.items())\n    ) or "<li>none</li>"\n    freeze_renewal_item_html = "".join(\n        f"<li><strong>{escape(entry[\'entry_id\'])}</strong> · blocker={escape(entry[\'blocker_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])}<br /><span>renewal_owner={escape(entry[\'renewal_owner\'])} · renewal_by={escape(entry[\'renewal_by\'])} · renewal_status={escape(entry[\'renewal_status\'])}</span><br /><span>freeze_owner={escape(entry[\'freeze_owner\'])} · freeze_until={escape(entry[\'freeze_until\'])} · approved_by={escape(entry[\'freeze_approved_by\'])}</span><br /><span>{escape(entry[\'summary\'])} · next_action={escape(entry[\'next_action\'])}</span></li>"\n        for entry in freeze_renewal_entries\n    ) or "<li>none</li>"\n    owner_review_queue = _build_owner_review_queue_entries(pack)\n    owner_queue_counts: Dict[str, Dict[str, int]] = {}\n    for entry in owner_review_queue:\n        counts = owner_queue_counts.setdefault(\n            entry["owner"],\n            {"blocker": 0, "checklist": 0, "decision": 0, "signoff": 0, "total": 0},\n        )\n        counts[entry["item_type"]] += 1\n        counts["total"] += 1\n    owner_queue_owner_html = "".join(\n        f"<li><strong>{escape(owner)}</strong> · blockers={counts[\'blocker\']} · checklist={counts[\'checklist\']} · decisions={counts[\'decision\']} · signoffs={counts[\'signoff\']} · total={counts[\'total\']}</li>"\n        for owner, counts in sorted(owner_queue_counts.items())\n    ) or "<li>none</li>"\n    owner_queue_item_html = "".join(\n        f"<li><strong>{escape(entry[\'queue_id\'])}</strong> · owner={escape(entry[\'owner\'])} · type={escape(entry[\'item_type\'])} · source={escape(entry[\'source_id\'])} · surface={escape(entry[\'surface_id\'])} · status={escape(entry[\'status\'])}<br /><span>{escape(entry[\'summary\'])}</span><br /><span>next_action={escape(entry[\'next_action\'])}</span></li>"\n        for entry in owner_review_queue\n    ) or "<li>none</li>"\n    status_counts: Dict[str, int] = {}\n    actor_counts: Dict[str, int] = {}\n    for event in pack.blocker_timeline:\n        status_counts[event.status] = status_counts.get(event.status, 0) + 1\n        actor_counts[event.actor] = actor_counts.get(event.actor, 0) + 1\n    status_summary_html = "".join(\n        f"<li><strong>{escape(status)}</strong> · count={count}</li>"\n        for status, count in sorted(status_counts.items())\n    ) or "<li>none</li>"\n    actor_summary_html = "".join(\n        f"<li><strong>{escape(actor)}</strong> · count={count}</li>"\n        for actor, count in sorted(actor_counts.items())\n    ) or "<li>none</li>"\n    blocker_ids = {blocker.blocker_id for blocker in pack.blocker_log}\n    orphan_timeline_ids = sorted(\n        blocker_id for blocker_id in timeline_index if blocker_id not in blocker_ids\n    )\n    latest_blocker_html = "".join(\n        (\n            f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest={escape(timeline_index[blocker.blocker_id][-1].event_id)} · actor={escape(timeline_index[blocker.blocker_id][-1].actor)} · status={escape(timeline_index[blocker.blocker_id][-1].status)} · timestamp={escape(timeline_index[blocker.blocker_id][-1].timestamp)}</li>"\n            if blocker.blocker_id in timeline_index\n            else f"<li><strong>{escape(blocker.blocker_id)}</strong> · latest=none</li>"\n        )\n        for blocker in pack.blocker_log\n    ) or "<li>none</li>"\n    orphan_timeline_html = "".join(\n        f"<li><strong>{escape(blocker_id)}</strong></li>"\n        for blocker_id in orphan_timeline_ids\n    ) or "<li>none</li>"\n    return f\'\'\'<!DOCTYPE html>\n<html lang="en">\n  <head>\n    <meta charset="utf-8" />\n    <title>{escape(pack.issue_id)} UI Review Pack</title>\n    <style>\n      body {{ font-family: Arial, sans-serif; margin: 32px; color: #0f172a; }}\n      header {{ margin-bottom: 24px; }}\n      h1 {{ margin-bottom: 4px; }}\n      .meta {{ color: #475569; font-size: 0.95rem; }}\n      .surface {{ margin-top: 24px; padding: 16px 18px; border: 1px solid #d9e2ec; border-radius: 12px; background: #f8fafc; }}\n      ul {{ padding-left: 20px; }}\n      .summary {{ padding: 18px 20px; background: #eff6ff; border-left: 4px solid #2563eb; }}\n    </style>\n  </head>\n  <body>\n    <header>\n      <p class="meta">{escape(pack.issue_id)} · {escape(pack.version)}</p>\n      <h1>{escape(pack.title)}</h1>\n      <p class="meta">Audit: {escape(audit.summary)}</p>\n    </header>\n    <section class="summary">\n      <h2>Readiness</h2>\n      <p>Missing checklist coverage: {escape(\', \'.join(audit.wireframes_missing_checklists) if audit.wireframes_missing_checklists else \'none\')}</p>\n      <p>Checklist items missing role links: {escape(\', \'.join(audit.checklist_items_missing_role_links) if audit.checklist_items_missing_role_links else \'none\')}</p>\n      <p>Missing decision coverage: {escape(\', \'.join(audit.wireframes_missing_decisions) if audit.wireframes_missing_decisions else \'none\')}</p>\n      <p>Unresolved decisions missing follow-ups: {escape(\', \'.join(audit.unresolved_decisions_missing_follow_ups) if audit.unresolved_decisions_missing_follow_ups else \'none\')}</p>\n      <p>Missing role assignments: {escape(\', \'.join(audit.wireframes_missing_role_assignments) if audit.wireframes_missing_role_assignments else \'none\')}</p>\n      <p>Missing signoff coverage: {escape(\', \'.join(audit.wireframes_missing_signoffs) if audit.wireframes_missing_signoffs else \'none\')}</p>\n      <p>Decisions missing role links: {escape(\', \'.join(audit.decisions_missing_role_links) if audit.decisions_missing_role_links else \'none\')}</p>\n      <p>Missing blocker coverage: {escape(\', \'.join(audit.unresolved_required_signoffs_without_blockers) if audit.unresolved_required_signoffs_without_blockers else \'none\')}</p>\n      <p>Missing signoff requested dates: {escape(\', \'.join(audit.signoffs_missing_requested_dates) if audit.signoffs_missing_requested_dates else \'none\')}</p>\n      <p>Missing signoff due dates: {escape(\', \'.join(audit.signoffs_missing_due_dates) if audit.signoffs_missing_due_dates else \'none\')}</p>\n      <p>Missing signoff escalation owners: {escape(\', \'.join(audit.signoffs_missing_escalation_owners) if audit.signoffs_missing_escalation_owners else \'none\')}</p>\n      <p>Missing signoff reminder owners: {escape(\', \'.join(audit.signoffs_missing_reminder_owners) if audit.signoffs_missing_reminder_owners else \'none\')}</p>\n      <p>Missing signoff next reminders: {escape(\', \'.join(audit.signoffs_missing_next_reminders) if audit.signoffs_missing_next_reminders else \'none\')}</p>\n      <p>Missing signoff reminder cadence: {escape(\', \'.join(audit.signoffs_missing_reminder_cadence) if audit.signoffs_missing_reminder_cadence else \'none\')}</p>\n      <p>Breached signoff SLA: {escape(\', \'.join(audit.signoffs_with_breached_sla) if audit.signoffs_with_breached_sla else \'none\')}</p>\n      <p>Missing waiver metadata: {escape(\', \'.join(audit.waived_signoffs_missing_metadata) if audit.waived_signoffs_missing_metadata else \'none\')}</p>\n      <p>Missing blocker timeline: {escape(\', \'.join(audit.blockers_missing_timeline_events) if audit.blockers_missing_timeline_events else \'none\')}</p>\n      <p>Closed blockers missing resolution events: {escape(\', \'.join(audit.closed_blockers_missing_resolution_events) if audit.closed_blockers_missing_resolution_events else \'none\')}</p>\n      <p>Freeze exceptions missing owners: {escape(\', \'.join(audit.freeze_exceptions_missing_owners) if audit.freeze_exceptions_missing_owners else \'none\')}</p>\n      <p>Freeze exceptions missing windows: {escape(\', \'.join(audit.freeze_exceptions_missing_until) if audit.freeze_exceptions_missing_until else \'none\')}</p>\n      <p>Freeze exceptions missing approvers: {escape(\', \'.join(audit.freeze_exceptions_missing_approvers) if audit.freeze_exceptions_missing_approvers else \'none\')}</p>\n      <p>Freeze exceptions missing approval dates: {escape(\', \'.join(audit.freeze_exceptions_missing_approval_dates) if audit.freeze_exceptions_missing_approval_dates else \'none\')}</p>\n      <p>Freeze exceptions missing renewal owners: {escape(\', \'.join(audit.freeze_exceptions_missing_renewal_owners) if audit.freeze_exceptions_missing_renewal_owners else \'none\')}</p>\n      <p>Freeze exceptions missing renewal dates: {escape(\', \'.join(audit.freeze_exceptions_missing_renewal_dates) if audit.freeze_exceptions_missing_renewal_dates else \'none\')}</p>\n      <p>Orphan blocker timeline ids: {escape(\', \'.join(audit.orphan_blocker_timeline_blocker_ids) if audit.orphan_blocker_timeline_blocker_ids else \'none\')}</p>\n      <p>Handoff events missing targets: {escape(\', \'.join(audit.handoff_events_missing_targets) if audit.handoff_events_missing_targets else \'none\')}</p>\n      <p>Handoff events missing artifacts: {escape(\', \'.join(audit.handoff_events_missing_artifacts) if audit.handoff_events_missing_artifacts else \'none\')}</p>\n      <p>Handoff events missing ack owners: {escape(\', \'.join(audit.handoff_events_missing_ack_owners) if audit.handoff_events_missing_ack_owners else \'none\')}</p>\n      <p>Handoff events missing ack dates: {escape(\', \'.join(audit.handoff_events_missing_ack_dates) if audit.handoff_events_missing_ack_dates else \'none\')}</p>\n      <p>Unresolved decisions: {escape(\', \'.join(audit.unresolved_decision_ids) if audit.unresolved_decision_ids else \'none\')}</p>\n      <p>Unresolved required signoffs: {escape(\', \'.join(audit.unresolved_required_signoff_ids) if audit.unresolved_required_signoff_ids else \'none\')}</p>\n    </section>\n    <section class="surface"><h2>Objectives</h2><ul>{objective_html}</ul></section>\n    <section class="surface"><h2>Review Summary Board</h2><h3>Entries</h3><ul>{review_summary_item_html}</ul></section>\n    <section class="surface"><h2>Objective Coverage Board</h2><h3>By Coverage Status</h3><ul>{objective_coverage_status_html}</ul><h3>By Persona</h3><ul>{objective_coverage_persona_html}</ul><h3>Entries</h3><ul>{objective_coverage_item_html}</ul></section>\n    <section class="surface"><h2>Persona Readiness Board</h2><h3>By Readiness</h3><ul>{persona_readiness_status_html}</ul><h3>Entries</h3><ul>{persona_readiness_item_html}</ul></section>\n    <section class="surface"><h2>Wireframes</h2><ul>{wireframe_html}</ul></section>\n    <section class="surface"><h2>Wireframe Readiness Board</h2><h3>By Readiness</h3><ul>{wireframe_readiness_status_html}</ul><h3>By Device</h3><ul>{wireframe_device_html}</ul><h3>Entries</h3><ul>{wireframe_readiness_item_html}</ul></section>\n    <section class="surface"><h2>Interactions</h2><ul>{interaction_html}</ul></section>\n    <section class="surface"><h2>Interaction Coverage Board</h2><h3>By Coverage Status</h3><ul>{interaction_coverage_status_html}</ul><h3>By Surface</h3><ul>{interaction_coverage_surface_html}</ul><h3>Entries</h3><ul>{interaction_coverage_item_html}</ul></section>\n    <section class="surface"><h2>Open Questions</h2><ul>{question_html}</ul></section>\n    <section class="surface"><h2>Open Question Tracker</h2><h3>By Owner</h3><ul>{open_question_owner_html}</ul><h3>By Theme</h3><ul>{open_question_theme_html}</ul><h3>Entries</h3><ul>{open_question_item_html}</ul></section>\n    <section class="surface"><h2>Reviewer Checklist</h2><ul>{checklist_html}</ul></section>\n    <section class="surface"><h2>Decision Log</h2><ul>{decision_html}</ul></section>\n    <section class="surface"><h2>Role Matrix</h2><ul>{role_matrix_html}</ul></section>\n    <section class="surface"><h2>Checklist Traceability Board</h2><h3>By Owner</h3><ul>{checklist_trace_owner_html}</ul><h3>By Status</h3><ul>{checklist_trace_status_html}</ul><h3>Entries</h3><ul>{checklist_trace_item_html}</ul></section>\n    <section class="surface"><h2>Decision Follow-up Tracker</h2><h3>By Owner</h3><ul>{decision_followup_owner_html}</ul><h3>By Status</h3><ul>{decision_followup_status_html}</ul><h3>Entries</h3><ul>{decision_followup_item_html}</ul></section>\n    <section class="surface"><h2>Role Coverage Board</h2><h3>By Surface</h3><ul>{role_coverage_surface_html}</ul><h3>By Status</h3><ul>{role_coverage_status_html}</ul><h3>Entries</h3><ul>{role_coverage_item_html}</ul></section>\n    <section class="surface"><h2>Signoff Dependency Board</h2><h3>By Dependency Status</h3><ul>{signoff_dependency_status_html}</ul><h3>By SLA State</h3><ul>{signoff_dependency_sla_html}</ul><h3>Entries</h3><ul>{signoff_dependency_item_html}</ul></section>\n    <section class="surface"><h2>Sign-off Log</h2><ul>{signoff_html}</ul></section>\n    <section class="surface"><h2>Sign-off SLA Dashboard</h2><h3>SLA States</h3><ul>{signoff_sla_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_sla_owner_html}</ul><h3>Sign-offs</h3><ul>{signoff_sla_item_html}</ul></section>\n    <section class="surface"><h2>Sign-off Reminder Queue</h2><h3>By Owner</h3><ul>{signoff_reminder_owner_html}</ul><h3>By Channel</h3><ul>{signoff_reminder_channel_html}</ul><h3>Items</h3><ul>{signoff_reminder_item_html}</ul></section>\n    <section class="surface"><h2>Reminder Cadence Board</h2><h3>By Cadence</h3><ul>{reminder_cadence_owner_html}</ul><h3>By Status</h3><ul>{reminder_cadence_status_html}</ul><h3>Items</h3><ul>{reminder_cadence_item_html}</ul></section>\n    <section class="surface"><h2>Sign-off Breach Board</h2><h3>SLA States</h3><ul>{signoff_breach_state_html}</ul><h3>Escalation Owners</h3><ul>{signoff_breach_owner_html}</ul><h3>Items</h3><ul>{signoff_breach_item_html}</ul></section>\n    <section class="surface"><h2>Escalation Dashboard</h2><h3>By Escalation Owner</h3><ul>{escalation_owner_html}</ul><h3>By Status</h3><ul>{escalation_status_html}</ul><h3>Escalations</h3><ul>{escalation_item_html}</ul></section>\n    <section class="surface"><h2>Escalation Handoff Ledger</h2><h3>By Status</h3><ul>{escalation_handoff_status_html}</ul><h3>By Channel</h3><ul>{escalation_handoff_channel_html}</ul><h3>Entries</h3><ul>{escalation_handoff_item_html}</ul></section>\n    <section class="surface"><h2>Handoff Ack Ledger</h2><h3>By Ack Owner</h3><ul>{handoff_ack_owner_html}</ul><h3>By Ack Status</h3><ul>{handoff_ack_status_html}</ul><h3>Entries</h3><ul>{handoff_ack_item_html}</ul></section>\n    <section class="surface"><h2>Owner Escalation Digest</h2><h3>Owners</h3><ul>{owner_escalation_owner_html}</ul><h3>Items</h3><ul>{owner_escalation_item_html}</ul></section>\n    <section class="surface"><h2>Owner Workload Board</h2><h3>Owners</h3><ul>{owner_workload_owner_html}</ul><h3>Items</h3><ul>{owner_workload_item_html}</ul></section>\n    <section class="surface"><h2>Blocker Log</h2><ul>{blocker_html}</ul></section>\n    <section class="surface"><h2>Blocker Timeline</h2><ul>{blocker_timeline_html}</ul></section>\n    <section class="surface"><h2>Review Freeze Exception Board</h2><h3>By Owner</h3><ul>{freeze_owner_html}</ul><h3>By Surface</h3><ul>{freeze_surface_html}</ul><h3>Entries</h3><ul>{freeze_item_html}</ul></section>\n    <section class="surface"><h2>Freeze Approval Trail</h2><h3>By Approver</h3><ul>{freeze_approval_owner_html}</ul><h3>By Status</h3><ul>{freeze_approval_status_html}</ul><h3>Entries</h3><ul>{freeze_approval_item_html}</ul></section>\n    <section class="surface"><h2>Freeze Renewal Tracker</h2><h3>By Renewal Owner</h3><ul>{freeze_renewal_owner_html}</ul><h3>By Renewal Status</h3><ul>{freeze_renewal_status_html}</ul><h3>Entries</h3><ul>{freeze_renewal_item_html}</ul></section>\n    <section class="surface"><h2>Review Exceptions</h2><ul>{exception_html}</ul></section>\n    <section class="surface"><h2>Review Exception Matrix</h2><h3>By Owner</h3><ul>{exception_owner_html}</ul><h3>By Status</h3><ul>{exception_status_html}</ul><h3>By Surface</h3><ul>{exception_surface_html}</ul></section>\n    <section class="surface"><h2>Audit Density Board</h2><h3>By Load Band</h3><ul>{audit_density_band_html}</ul><h3>Entries</h3><ul>{audit_density_item_html}</ul></section>\n    <section class="surface"><h2>Owner Review Queue</h2><h3>Owners</h3><ul>{owner_queue_owner_html}</ul><h3>Items</h3><ul>{owner_queue_item_html}</ul></section>\n    <section class="surface"><h2>Blocker Timeline Summary</h2><h3>Events by Status</h3><ul>{status_summary_html}</ul><h3>Events by Actor</h3><ul>{actor_summary_html}</ul><h3>Latest Blocker Events</h3><ul>{latest_blocker_html}</ul><h3>Orphan Timeline Blockers</h3><ul>{orphan_timeline_html}</ul></section>\n  </body>\n</html>\n\'\'\'\n\n\ndef write_ui_review_pack_bundle(root_dir: str, pack: UIReviewPack) -> UIReviewPackArtifacts:\n    base = Path(root_dir)\n    base.mkdir(parents=True, exist_ok=True)\n    slug = pack.issue_id.lower().replace(" ", "-")\n    markdown_path = str(base / f"{slug}-review-pack.md")\n    html_path = str(base / f"{slug}-review-pack.html")\n    decision_log_path = str(base / f"{slug}-decision-log.md")\n    review_summary_board_path = str(base / f"{slug}-review-summary-board.md")\n    objective_coverage_board_path = str(base / f"{slug}-objective-coverage-board.md")\n    persona_readiness_board_path = str(base / f"{slug}-persona-readiness-board.md")\n    wireframe_readiness_board_path = str(base / f"{slug}-wireframe-readiness-board.md")\n    interaction_coverage_board_path = str(base / f"{slug}-interaction-coverage-board.md")\n    open_question_tracker_path = str(base / f"{slug}-open-question-tracker.md")\n    checklist_traceability_board_path = str(base / f"{slug}-checklist-traceability-board.md")\n    decision_followup_tracker_path = str(base / f"{slug}-decision-followup-tracker.md")\n    role_matrix_path = str(base / f"{slug}-role-matrix.md")\n    role_coverage_board_path = str(base / f"{slug}-role-coverage-board.md")\n    signoff_dependency_board_path = str(base / f"{slug}-signoff-dependency-board.md")\n    signoff_log_path = str(base / f"{slug}-signoff-log.md")\n    signoff_sla_dashboard_path = str(base / f"{slug}-signoff-sla-dashboard.md")\n    signoff_reminder_queue_path = str(base / f"{slug}-signoff-reminder-queue.md")\n    reminder_cadence_board_path = str(base / f"{slug}-reminder-cadence-board.md")\n    signoff_breach_board_path = str(base / f"{slug}-signoff-breach-board.md")\n    escalation_dashboard_path = str(base / f"{slug}-escalation-dashboard.md")\n    escalation_handoff_ledger_path = str(base / f"{slug}-escalation-handoff-ledger.md")\n    handoff_ack_ledger_path = str(base / f"{slug}-handoff-ack-ledger.md")\n    owner_escalation_digest_path = str(base / f"{slug}-owner-escalation-digest.md")\n    owner_workload_board_path = str(base / f"{slug}-owner-workload-board.md")\n    blocker_log_path = str(base / f"{slug}-blocker-log.md")\n    blocker_timeline_path = str(base / f"{slug}-blocker-timeline.md")\n    freeze_exception_board_path = str(base / f"{slug}-freeze-exception-board.md")\n    freeze_approval_trail_path = str(base / f"{slug}-freeze-approval-trail.md")\n    freeze_renewal_tracker_path = str(base / f"{slug}-freeze-renewal-tracker.md")\n    exception_log_path = str(base / f"{slug}-exception-log.md")\n    exception_matrix_path = str(base / f"{slug}-exception-matrix.md")\n    audit_density_board_path = str(base / f"{slug}-audit-density-board.md")\n    owner_review_queue_path = str(base / f"{slug}-owner-review-queue.md")\n    blocker_timeline_summary_path = str(base / f"{slug}-blocker-timeline-summary.md")\n    audit = UIReviewPackAuditor().audit(pack)\n    Path(markdown_path).write_text(render_ui_review_pack_report(pack, audit))\n    Path(html_path).write_text(render_ui_review_pack_html(pack, audit))\n    Path(decision_log_path).write_text(render_ui_review_decision_log(pack))\n    Path(review_summary_board_path).write_text(render_ui_review_review_summary_board(pack))\n    Path(objective_coverage_board_path).write_text(render_ui_review_objective_coverage_board(pack))\n    Path(persona_readiness_board_path).write_text(render_ui_review_persona_readiness_board(pack))\n    Path(wireframe_readiness_board_path).write_text(render_ui_review_wireframe_readiness_board(pack))\n    Path(interaction_coverage_board_path).write_text(render_ui_review_interaction_coverage_board(pack))\n    Path(open_question_tracker_path).write_text(render_ui_review_open_question_tracker(pack))\n    Path(checklist_traceability_board_path).write_text(render_ui_review_checklist_traceability_board(pack))\n    Path(decision_followup_tracker_path).write_text(render_ui_review_decision_followup_tracker(pack))\n    Path(role_matrix_path).write_text(render_ui_review_role_matrix(pack))\n    Path(role_coverage_board_path).write_text(render_ui_review_role_coverage_board(pack))\n    Path(signoff_dependency_board_path).write_text(render_ui_review_signoff_dependency_board(pack))\n    Path(signoff_log_path).write_text(render_ui_review_signoff_log(pack))\n    Path(signoff_sla_dashboard_path).write_text(render_ui_review_signoff_sla_dashboard(pack))\n    Path(signoff_reminder_queue_path).write_text(render_ui_review_signoff_reminder_queue(pack))\n    Path(reminder_cadence_board_path).write_text(render_ui_review_reminder_cadence_board(pack))\n    Path(signoff_breach_board_path).write_text(render_ui_review_signoff_breach_board(pack))\n    Path(escalation_dashboard_path).write_text(render_ui_review_escalation_dashboard(pack))\n    Path(escalation_handoff_ledger_path).write_text(render_ui_review_escalation_handoff_ledger(pack))\n    Path(handoff_ack_ledger_path).write_text(render_ui_review_handoff_ack_ledger(pack))\n    Path(owner_escalation_digest_path).write_text(render_ui_review_owner_escalation_digest(pack))\n    Path(owner_workload_board_path).write_text(render_ui_review_owner_workload_board(pack))\n    Path(blocker_log_path).write_text(render_ui_review_blocker_log(pack))\n    Path(blocker_timeline_path).write_text(render_ui_review_blocker_timeline(pack))\n    Path(freeze_exception_board_path).write_text(render_ui_review_freeze_exception_board(pack))\n    Path(freeze_approval_trail_path).write_text(render_ui_review_freeze_approval_trail(pack))\n    Path(freeze_renewal_tracker_path).write_text(render_ui_review_freeze_renewal_tracker(pack))\n    Path(exception_log_path).write_text(render_ui_review_exception_log(pack))\n    Path(exception_matrix_path).write_text(render_ui_review_exception_matrix(pack))\n    Path(audit_density_board_path).write_text(render_ui_review_audit_density_board(pack))\n    Path(owner_review_queue_path).write_text(render_ui_review_owner_review_queue(pack))\n    Path(blocker_timeline_summary_path).write_text(render_ui_review_blocker_timeline_summary(pack))\n    return UIReviewPackArtifacts(\n        root_dir=str(base),\n        markdown_path=markdown_path,\n        html_path=html_path,\n        decision_log_path=decision_log_path,\n        review_summary_board_path=review_summary_board_path,\n        objective_coverage_board_path=objective_coverage_board_path,\n        persona_readiness_board_path=persona_readiness_board_path,\n        wireframe_readiness_board_path=wireframe_readiness_board_path,\n        interaction_coverage_board_path=interaction_coverage_board_path,\n        open_question_tracker_path=open_question_tracker_path,\n        checklist_traceability_board_path=checklist_traceability_board_path,\n        decision_followup_tracker_path=decision_followup_tracker_path,\n        role_matrix_path=role_matrix_path,\n        role_coverage_board_path=role_coverage_board_path,\n        signoff_dependency_board_path=signoff_dependency_board_path,\n        signoff_log_path=signoff_log_path,\n        signoff_sla_dashboard_path=signoff_sla_dashboard_path,\n        signoff_reminder_queue_path=signoff_reminder_queue_path,\n        reminder_cadence_board_path=reminder_cadence_board_path,\n        signoff_breach_board_path=signoff_breach_board_path,\n        escalation_dashboard_path=escalation_dashboard_path,\n        escalation_handoff_ledger_path=escalation_handoff_ledger_path,\n        handoff_ack_ledger_path=handoff_ack_ledger_path,\n        owner_escalation_digest_path=owner_escalation_digest_path,\n        owner_workload_board_path=owner_workload_board_path,\n        blocker_log_path=blocker_log_path,\n        blocker_timeline_path=blocker_timeline_path,\n        freeze_exception_board_path=freeze_exception_board_path,\n        freeze_approval_trail_path=freeze_approval_trail_path,\n        freeze_renewal_tracker_path=freeze_renewal_tracker_path,\n        exception_log_path=exception_log_path,\n        exception_matrix_path=exception_matrix_path,\n        audit_density_board_path=audit_density_board_path,\n        owner_review_queue_path=owner_review_queue_path,\n        blocker_timeline_summary_path=blocker_timeline_summary_path,\n    )\n', str(Path(__file__).with_name("ui_review.py")), "exec"),
    _ui_review_namespace,
)
for _name in _UI_REVIEW_PUBLIC_NAMES:
    globals()[_name] = _ui_review_namespace[_name]
_install_compatibility_module(
    "ui_review",
    {name: _ui_review_namespace[name] for name in _UI_REVIEW_PUBLIC_NAMES},
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/product/console.go",
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
from .design_system import (
    AuditRequirement,
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DataAccuracyCheck,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationEntry,
    NavigationNode,
    NavigationRoute,
    PerformanceBudget,
    RolePermissionScenario,
    UIAcceptanceAudit,
    UIAcceptanceLibrary,
    UIAcceptanceSuite,
    UsabilityJourney,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
    render_ui_acceptance_report,
)
from .console_ia import (
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ConsoleInteractionAudit,
    ConsoleInteractionAuditor,
    ConsoleInteractionDraft,
    ConsoleSurface,
    FilterDefinition,
    GlobalAction,
    NavigationItem,
    SurfaceInteractionContract,
    SurfacePermissionRule,
    SurfaceState,
    build_big_4203_console_interaction_draft,
    render_console_interaction_report,
    render_console_ia_report,
)
from .event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
)
from .observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, RunCloseout, TaskRun
from .dashboard_run_contract import (
    DashboardRunContract,
    DashboardRunContractAudit,
    DashboardRunContractLibrary,
    SchemaField,
    SurfaceSchema,
    render_dashboard_run_contract_report,
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
