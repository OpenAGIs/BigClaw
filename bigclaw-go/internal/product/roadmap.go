package product

import (
	"fmt"
	"strings"
)

type EpicMilestone struct {
	EpicID    string `json:"epic_id"`
	Title     string `json:"title"`
	Phase     string `json:"phase"`
	Owner     string `json:"owner"`
	Milestone string `json:"milestone"`
}

type ExecutionPackRoadmap struct {
	Name  string          `json:"name"`
	Epics []EpicMilestone `json:"epics"`
}

func (r ExecutionPackRoadmap) PhaseMap() map[string][]EpicMilestone {
	phases := map[string][]EpicMilestone{}
	for _, epic := range r.Epics {
		phases[epic.Phase] = append(phases[epic.Phase], epic)
	}
	return phases
}

func (r ExecutionPackRoadmap) EpicMap() map[string]EpicMilestone {
	epics := make(map[string]EpicMilestone, len(r.Epics))
	for _, epic := range r.Epics {
		epics[epic.EpicID] = epic
	}
	return epics
}

func (r ExecutionPackRoadmap) ValidateUniqueOwners() error {
	seen := map[string]string{}
	for _, epic := range r.Epics {
		owner := strings.ToLower(strings.TrimSpace(epic.Owner))
		if previous, ok := seen[owner]; ok {
			return fmt.Errorf("Owner '%s' is assigned to both %s and %s", epic.Owner, previous, epic.EpicID)
		}
		seen[owner] = epic.EpicID
	}
	return nil
}

func BuildExecutionPackRoadmap() (ExecutionPackRoadmap, error) {
	roadmap := ExecutionPackRoadmap{
		Name: "BigClaw v4.0 Execution Pack",
		Epics: []EpicMilestone{
			{
				EpicID:    "BIG-EPIC-8",
				Title:     "研发自治执行平台增强",
				Phase:     "Phase 1",
				Owner:     "engineering-platform",
				Milestone: "M1 Foundation uplift",
			},
			{
				EpicID:    "BIG-EPIC-9",
				Title:     "工程运营系统",
				Phase:     "Phase 2",
				Owner:     "engineering-operations",
				Milestone: "M2 Operations control plane",
			},
			{
				EpicID:    "BIG-EPIC-10",
				Title:     "跨部门 Agent Orchestration",
				Phase:     "Phase 3",
				Owner:     "orchestration-office",
				Milestone: "M3 Cross-team orchestration",
			},
			{
				EpicID:    "BIG-EPIC-11",
				Title:     "产品化 UI 与控制台",
				Phase:     "Phase 4",
				Owner:     "product-experience",
				Milestone: "M4 Productized console",
			},
			{
				EpicID:    "BIG-EPIC-12",
				Title:     "计费、套餐与商业化控制",
				Phase:     "Phase 5",
				Owner:     "commercialization",
				Milestone: "M5 Billing and packaging",
			},
		},
	}
	if err := roadmap.ValidateUniqueOwners(); err != nil {
		return ExecutionPackRoadmap{}, err
	}
	return roadmap, nil
}
