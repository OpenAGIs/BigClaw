import pytest

from bigclaw import EpicMilestone, ExecutionPackRoadmap, build_execution_pack_roadmap


def test_build_execution_pack_roadmap_maps_epics_to_phases():
    roadmap = build_execution_pack_roadmap()

    assert roadmap.name == "BigClaw v4.0 Execution Pack"
    assert roadmap.epic_map()["BIG-EPIC-8"].phase == "Phase 1"
    assert roadmap.epic_map()["BIG-EPIC-9"].phase == "Phase 2"
    assert roadmap.epic_map()["BIG-EPIC-10"].phase == "Phase 3"
    assert roadmap.epic_map()["BIG-EPIC-11"].phase == "Phase 4"
    assert roadmap.epic_map()["BIG-EPIC-12"].phase == "Phase 5"
    assert roadmap.phase_map()["Phase 3"][0].milestone == "M3 Cross-team orchestration"


def test_execution_pack_roadmap_rejects_duplicate_owners():
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
                owner="engineering-platform",
                milestone="M2 Operations control plane",
            ),
        ],
    )

    with pytest.raises(ValueError, match="Owner 'engineering-platform' is assigned to both BIG-EPIC-8 and BIG-EPIC-9"):
        roadmap.validate_unique_owners()
