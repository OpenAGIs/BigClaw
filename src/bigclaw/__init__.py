import sys
import types
from dataclasses import dataclass, field
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
from .connectors import SourceIssue
from .execution_contract import ExecutionPermission, ExecutionPermissionMatrix, ExecutionRole


def _install_legacy_surface_module(name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    for export_name in export_names:
        module.__dict__[export_name] = getattr(_legacy_runtime_surface, export_name)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


def _install_inline_surface_module(name: str, **attrs: object) -> None:
    module_name = f"{__name__}.{name}"
    module = types.ModuleType(module_name)
    for attr_name, value in attrs.items():
        if hasattr(value, "__module__"):
            try:
                value.__module__ = module_name
            except (AttributeError, TypeError):
                pass
        module.__dict__[attr_name] = value
    sys.modules[module_name] = module
    globals()[name] = module


@dataclass
class ValidationReportDecision:
    allowed_to_close: bool
    status: str
    summary: str
    missing_reports: List[str] = field(default_factory=list)


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
    return [field for field in required if field not in payload]


def map_priority(p: str) -> Priority:
    normalized = (p or "").upper()
    if normalized == "P0":
        return Priority.P0
    if normalized == "P1":
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
            EpicMilestone(
                epic_id="BIG-EPIC-8",
                title="研发自治执行平台增强",
                phase="Phase 1",
                owner="engineering-platform",
                milestone="M1 Foundation uplift",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-9",
                title="工程运营系统",
                phase="Phase 2",
                owner="engineering-operations",
                milestone="M2 Operations control plane",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-10",
                title="跨部门 Agent Orchestration",
                phase="Phase 3",
                owner="orchestration-office",
                milestone="M3 Cross-team orchestration",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-11",
                title="产品化 UI 与控制台",
                phase="Phase 4",
                owner="product-experience",
                milestone="M4 Productized console",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-12",
                title="计费、套餐与商业化控制",
                phase="Phase 5",
                owner="commercialization",
                milestone="M5 Billing and packaging",
            ),
        ],
    )
    roadmap.validate_unique_owners()
    return roadmap


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
    validate_roles(links)
    return RunCommitBinding(links=list(links))


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
        agent = RepoAgent(actor=actor, repo_agent_id=f"agent-{_slug(actor)}", display_name=actor, roles=[role])
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


_install_inline_surface_module(
    "validation_policy",
    ValidationReportDecision=ValidationReportDecision,
    REQUIRED_REPORT_ARTIFACTS=REQUIRED_REPORT_ARTIFACTS,
    enforce_validation_report_policy=enforce_validation_report_policy,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/closeout.go",
)
_install_inline_surface_module(
    "repo_triage",
    LineageEvidence=LineageEvidence,
    TriageRecommendation=TriageRecommendation,
    recommend_triage_action=recommend_triage_action,
    approval_evidence_packet=approval_evidence_packet,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/triage.go",
)
_install_inline_surface_module(
    "repo_governance",
    REPO_ACTION_PERMISSIONS=REPO_ACTION_PERMISSIONS,
    REPO_ROLE_POLICIES=REPO_ROLE_POLICIES,
    RepoPermissionContract=RepoPermissionContract,
    repo_required_audit_fields=repo_required_audit_fields,
    missing_repo_audit_fields=missing_repo_audit_fields,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/governance.go",
)
_install_inline_surface_module(
    "mapping",
    map_priority=map_priority,
    map_state=map_state,
    map_source_issue_to_task=map_source_issue_to_task,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/mapping.go",
)
_install_inline_surface_module(
    "roadmap",
    EpicMilestone=EpicMilestone,
    ExecutionPackRoadmap=ExecutionPackRoadmap,
    build_execution_pack_roadmap=build_execution_pack_roadmap,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/regression/roadmap_contract_test.go",
)
_install_inline_surface_module(
    "repo_commits",
    RepoCommit=RepoCommit,
    CommitLineage=CommitLineage,
    CommitDiff=CommitDiff,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/commits.go",
)
_install_inline_surface_module(
    "repo_plane",
    RepoSpace=RepoSpace,
    RepoAgent=RepoAgent,
    RunCommitLink=RunCommitLink,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/plane.go",
)
_install_inline_surface_module(
    "repo_links",
    VALID_ROLES=VALID_ROLES,
    RunCommitBinding=RunCommitBinding,
    validate_roles=validate_roles,
    bind_run_commits=bind_run_commits,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/links.go",
)
_install_inline_surface_module(
    "repo_registry",
    RepoRegistry=RepoRegistry,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/registry.go",
)
_install_inline_surface_module(
    "repo_gateway",
    RepoGatewayClient=RepoGatewayClient,
    RepoGatewayError=RepoGatewayError,
    normalize_gateway_error=normalize_gateway_error,
    normalize_commit=normalize_commit,
    normalize_lineage=normalize_lineage,
    normalize_diff=normalize_diff,
    repo_audit_payload=repo_audit_payload,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/repo/gateway.go",
)

from . import runtime as _legacy_runtime_surface

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
from .connectors import SourceIssue, GitHubConnector, LinearConnector, JiraConnector
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
from .issue_archive import (
    ArchivedIssue,
    IssuePriorityArchive,
    IssuePriorityArchiveAudit,
    IssuePriorityArchivist,
    render_issue_priority_archive_report,
)
from .risk import RiskFactor, RiskScore, RiskScorer
from .dsl import WorkflowDefinition, WorkflowStep
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
