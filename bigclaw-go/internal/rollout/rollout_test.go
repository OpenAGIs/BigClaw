package rollout

import (
	"strings"
	"testing"
)

func TestPilotRolloutScorecardAndCandidateGate(t *testing.T) {
	scorecard := BuildPilotRolloutScorecard(84, 78, 82, 1, 88)
	if scorecard.Recommendation != "go" {
		t.Fatalf("expected go recommendation, got %+v", scorecard)
	}

	gateDecision := EntryGateDecision{GateID: "gate-v3", Passed: true}
	result := EvaluateCandidateGate(gateDecision, scorecard)
	if result.CandidateGate != "enable-by-default" {
		t.Fatalf("expected enable-by-default, got %+v", result)
	}
	report := RenderPilotRolloutGateReport(result)
	if !strings.Contains(report, "Candidate gate") {
		t.Fatalf("expected candidate gate in report, got %s", report)
	}
}

func TestRepoWeeklyNarrativeExportsRemainConsistent(t *testing.T) {
	section := RenderWeeklyRepoEvidenceSection(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})
	exports := RenderRepoNarrativeExports(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})

	if !strings.Contains(section, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in section, got %s", section)
	}
	if !strings.Contains(exports["markdown"], "Repo Evidence Summary") {
		t.Fatalf("expected repo evidence summary in markdown, got %+v", exports)
	}
	if !strings.Contains(exports["text"], "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in text export, got %+v", exports)
	}
	if !strings.Contains(exports["html"], "<section><h2>Repo Evidence Summary</h2>") {
		t.Fatalf("expected repo evidence html section, got %+v", exports)
	}
}
