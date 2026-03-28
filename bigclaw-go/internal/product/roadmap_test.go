package product

import (
	"strings"
	"testing"
)

func TestBuildExecutionPackRoadmapMapsEpicsToPhases(t *testing.T) {
	roadmap, err := BuildExecutionPackRoadmap()
	if err != nil {
		t.Fatalf("build execution pack roadmap: %v", err)
	}

	if roadmap.Name != "BigClaw v4.0 Execution Pack" {
		t.Fatalf("unexpected roadmap name: %+v", roadmap)
	}
	if roadmap.EpicMap()["BIG-EPIC-8"].Phase != "Phase 1" {
		t.Fatalf("unexpected phase mapping for BIG-EPIC-8: %+v", roadmap.EpicMap()["BIG-EPIC-8"])
	}
	if roadmap.EpicMap()["BIG-EPIC-9"].Phase != "Phase 2" {
		t.Fatalf("unexpected phase mapping for BIG-EPIC-9: %+v", roadmap.EpicMap()["BIG-EPIC-9"])
	}
	if roadmap.EpicMap()["BIG-EPIC-10"].Phase != "Phase 3" {
		t.Fatalf("unexpected phase mapping for BIG-EPIC-10: %+v", roadmap.EpicMap()["BIG-EPIC-10"])
	}
	if roadmap.EpicMap()["BIG-EPIC-11"].Phase != "Phase 4" {
		t.Fatalf("unexpected phase mapping for BIG-EPIC-11: %+v", roadmap.EpicMap()["BIG-EPIC-11"])
	}
	if roadmap.EpicMap()["BIG-EPIC-12"].Phase != "Phase 5" {
		t.Fatalf("unexpected phase mapping for BIG-EPIC-12: %+v", roadmap.EpicMap()["BIG-EPIC-12"])
	}
	if roadmap.PhaseMap()["Phase 3"][0].Milestone != "M3 Cross-team orchestration" {
		t.Fatalf("unexpected phase milestone mapping: %+v", roadmap.PhaseMap()["Phase 3"])
	}
}

func TestExecutionPackRoadmapRejectsDuplicateOwners(t *testing.T) {
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
				Owner:     "engineering-platform",
				Milestone: "M2 Operations control plane",
			},
		},
	}

	err := roadmap.ValidateUniqueOwners()
	if err == nil {
		t.Fatal("expected duplicate owner validation to fail")
	}
	if !strings.Contains(err.Error(), "Owner 'engineering-platform' is assigned to both BIG-EPIC-8 and BIG-EPIC-9") {
		t.Fatalf("unexpected duplicate owner error: %v", err)
	}
}
