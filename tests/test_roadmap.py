import pytest

from bigclaw.roadmap import ExecutionPackRoadmap, EpicMilestone, build_execution_pack_roadmap


def test_build_execution_pack_roadmap_has_unique_owner_per_phase() -> None:
    roadmap = build_execution_pack_roadmap()

    assert roadmap.name == "BigClaw v4.0 Execution Pack"
    assert roadmap.epic_map()["BIG-EPIC-10"].milestone == "M3 Cross-team orchestration"
    assert [epic.epic_id for epic in roadmap.phase_map()["Phase 5"]] == ["BIG-EPIC-12"]


def test_execution_pack_roadmap_rejects_duplicate_owners() -> None:
    roadmap = ExecutionPackRoadmap(
        name="dup",
        epics=[
            EpicMilestone("BIG-1", "One", "Phase 1", "ops", "M1"),
            EpicMilestone("BIG-2", "Two", "Phase 2", "ops", "M2"),
        ],
    )

    with pytest.raises(ValueError, match="Owner 'ops' is assigned to both BIG-1 and BIG-2"):
        roadmap.validate_unique_owners()
