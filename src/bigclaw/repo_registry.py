from __future__ import annotations

from dataclasses import dataclass, field
from typing import Dict, Optional

from .models import Task
from .repo_plane import RepoAgent, RepoSpace


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
