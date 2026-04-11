from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, List


@dataclass(frozen=True)
class ExecutionPermission:
    name: str
    resource: str
    actions: List[str] = field(default_factory=list)
    scopes: List[str] = field(default_factory=list)


@dataclass(frozen=True)
class ExecutionRole:
    name: str
    personas: List[str] = field(default_factory=list)
    granted_permissions: List[str] = field(default_factory=list)
    scope_bindings: List[str] = field(default_factory=list)
    escalation_target: str = ""


@dataclass
class PermissionCheckResult:
    allowed: bool
    granted_permissions: List[str] = field(default_factory=list)
    missing_permissions: List[str] = field(default_factory=list)


class ExecutionPermissionMatrix:
    def __init__(self, permissions: List[ExecutionPermission], roles: List[ExecutionRole] | None = None) -> None:
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
