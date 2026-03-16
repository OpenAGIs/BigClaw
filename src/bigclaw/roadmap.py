from dataclasses import dataclass, field
from typing import Dict, List


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
                raise ValueError(
                    f"Owner '{epic.owner}' is assigned to both {seen[owner]} and {epic.epic_id}"
                )
            seen[owner] = epic.epic_id


def build_execution_pack_roadmap() -> ExecutionPackRoadmap:
    roadmap = ExecutionPackRoadmap(
        name="BigClaw v5.0 Parallel Distributed Platform Follow-up",
        epics=[
            EpicMilestone(
                epic_id="BIG-EPIC-21",
                title="并行分布式执行平台 closeout baseline",
                phase="Closed Baseline",
                owner="distributed-platform",
                milestone="M0 Closeout evidence complete through review and navigation refresh",
            ),
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
                owner="coordination-hardening",
                milestone="M2 Close leader election, takeover, and operator-facing hardening gaps",
            ),
            EpicMilestone(
                epic_id="BIG-PAR-FU-3",
                title="External-store and multi-node validation",
                phase="Phase 3",
                owner="platform-validation",
                milestone="M3 Prove the control plane beyond SQLite with higher-scale shared backends",
            ),
            EpicMilestone(
                epic_id="BIG-PAR-FU-4",
                title="Production capacity certification",
                phase="Phase 4",
                owner="release-readiness",
                milestone="M4 Convert local soak evidence into production-grade certification gates",
            ),
        ],
    )
    roadmap.validate_unique_owners()
    return roadmap
