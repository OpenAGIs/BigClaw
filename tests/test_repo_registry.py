from bigclaw.models import Task
from bigclaw.repo_plane import RepoSpace
from bigclaw.repo_registry import RepoRegistry


def test_repo_registry_resolves_space_channel_and_agent_deterministically():
    registry = RepoRegistry()
    registry.register_space(
        RepoSpace(
            space_id="space-1",
            project_key="BIGCLAW",
            repo="OpenAGIs/BigClaw",
            sidecar_url="http://127.0.0.1:4041",
            health_state="healthy",
        )
    )

    task = Task(task_id="OPE-141", source="linear", title="repo registry", description="")

    resolved = registry.resolve_space("BIGCLAW")
    assert resolved is not None
    assert resolved.repo == "OpenAGIs/BigClaw"

    channel = registry.resolve_default_channel("BIGCLAW", task)
    assert channel == "bigclaw-ope-141"

    agent = registry.resolve_agent("native cloud", role="reviewer")
    assert agent.repo_agent_id == "agent-native-cloud"


def test_repo_registry_round_trip():
    registry = RepoRegistry()
    registry.register_space(RepoSpace(space_id="s-1", project_key="BIGCLAW", repo="OpenAGIs/BigClaw"))
    registry.resolve_agent("native cloud")

    serialized = registry.to_dict()
    restored = RepoRegistry.from_dict(serialized)

    assert restored.resolve_space("BIGCLAW") is not None
    assert restored.resolve_agent("native cloud").repo_agent_id == "agent-native-cloud"
