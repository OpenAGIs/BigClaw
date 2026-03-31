package roadmap

import "testing"

func TestBuildExecutionPackRoadmapAndHelpers(t *testing.T) {
	roadmap, err := BuildExecutionPackRoadmap()
	if err != nil {
		t.Fatalf("build roadmap: %v", err)
	}
	if roadmap.Name != "BigClaw v4.0 Execution Pack" || len(roadmap.Epics) != 5 {
		t.Fatalf("unexpected roadmap: %+v", roadmap)
	}
	if len(roadmap.PhaseMap()["Phase 3"]) != 1 || roadmap.EpicMap()["BIG-EPIC-11"].Owner != "product-experience" {
		t.Fatalf("unexpected roadmap helper output: %+v", roadmap)
	}
}

func TestExecutionPackRoadmapValidateUniqueOwners(t *testing.T) {
	roadmap := ExecutionPackRoadmap{
		Name: "dup",
		Epics: []EpicMilestone{
			{EpicID: "BIG-EPIC-8", Owner: "engineering-platform"},
			{EpicID: "BIG-EPIC-9", Owner: "engineering-platform"},
		},
	}
	err := roadmap.ValidateUniqueOwners()
	if err == nil || err.Error() != "Owner 'engineering-platform' is assigned to both BIG-EPIC-8 and BIG-EPIC-9" {
		t.Fatalf("unexpected duplicate owner error: %v", err)
	}
}
