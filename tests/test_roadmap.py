import pytest

from bigclaw import EpicMilestone, ExecutionPackRoadmap, build_execution_pack_roadmap


def test_build_execution_pack_roadmap_maps_epics_to_phases():
    roadmap = build_execution_pack_roadmap()

    assert roadmap.name == "BigClaw v5.0 Parallel Distributed Platform Follow-up"
    assert roadmap.epic_map()["BIG-EPIC-21"].phase == "Closed Baseline"
    assert roadmap.epic_map()["BIG-PAR-FU-1"].phase == "Phase 1"
    assert roadmap.epic_map()["BIG-PAR-FU-2"].phase == "Phase 2"
    assert roadmap.epic_map()["BIG-PAR-FU-3"].phase == "Phase 3"
    assert roadmap.epic_map()["BIG-PAR-FU-4"].phase == "Phase 4"
    assert roadmap.phase_map()["Phase 3"][0].milestone == (
        "M3 Prove the control plane beyond SQLite with higher-scale shared backends"
    )


def test_execution_pack_roadmap_rejects_duplicate_owners():
    roadmap = ExecutionPackRoadmap(
        name="BigClaw v5.0 Parallel Distributed Platform Follow-up",
        epics=[
            EpicMilestone(
                epic_id="BIG-PAR-FU-1",
                title="Replicated event durability rollout",
                phase="Phase 1",
                owner="event-durability",
                milestone="M1 Promote the OPE-222 durability contract into a provider-backed rollout path",
            ),
            EpicMilestone(
                epic_id="BIG-PAR-FU-2",
                title="Remaining hardening gap closure",
                phase="Phase 2",
                owner="event-durability",
                milestone="M2 Close leader election, takeover, and operator-facing hardening gaps",
            ),
        ],
    )

    with pytest.raises(ValueError, match="Owner 'event-durability' is assigned to both BIG-PAR-FU-1 and BIG-PAR-FU-2"):
        roadmap.validate_unique_owners()
