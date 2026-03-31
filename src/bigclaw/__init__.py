from __future__ import annotations

import argparse
import json
import subprocess
import sys
import types
import warnings
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Protocol, Sequence

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


def _install_support_module(name: str, **attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__.update(attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


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
from .collaboration import (
    CollaborationComment,
    CollaborationThread,
    DecisionNote,
    build_collaboration_thread,
    build_collaboration_thread_from_audits,
)
from .saved_views import (
    AlertDigestSubscription,
    SavedView,
    SavedViewCatalog,
    SavedViewCatalogAudit,
    SavedViewFilter,
    SavedViewLibrary,
    render_saved_view_report,
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
from .dsl import WorkflowDefinition, WorkflowStep
from .workspace_bootstrap import WorkspaceBootstrapError, bootstrap_workspace, cleanup_workspace
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
from .event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
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
