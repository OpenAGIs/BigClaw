from dataclasses import dataclass, field
from typing import Dict, List

from .models import Priority, RiskLevel, Task


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
            backlog_items=[
                GovernanceBacklogItem.from_dict(item) for item in data.get("backlog_items", [])
            ],
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
            priority_counts={
                str(priority): int(count)
                for priority, count in dict(data.get("priority_counts", {})).items()
            },
            category_counts={
                str(category): int(count)
                for category, count in dict(data.get("category_counts", {})).items()
            },
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
