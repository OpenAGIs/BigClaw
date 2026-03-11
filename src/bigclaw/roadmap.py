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
