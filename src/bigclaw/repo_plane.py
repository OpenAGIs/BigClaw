from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, Iterable, List, Optional

from .models import Task


def _slug(value: str) -> str:
    cleaned = "".join(ch.lower() if ch.isalnum() else "-" for ch in value)
    return "-".join(part for part in cleaned.split("-") if part) or "agent"


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
