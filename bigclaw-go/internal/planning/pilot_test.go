package planning

import (
	"strings"
	"testing"
)

func TestPilotRolloutScorecardAndCandidateGate(t *testing.T) {
	scorecard := BuildPilotRolloutScorecard(84, 78, 82, 1, 88)
	if scorecard.Recommendation != "go" {
		t.Fatalf("expected go recommendation, got %+v", scorecard)
	}

	result := EvaluateCandidateGate(EntryGateDecision{GateID: "gate-v3", Passed: true}, scorecard)
	if result.CandidateGate != "enable-by-default" {
		t.Fatalf("expected enable-by-default candidate gate, got %+v", result)
	}
	report := RenderPilotRolloutGateReport(result)
	if !strings.Contains(report, "Candidate gate") {
		t.Fatalf("expected candidate gate in report, got %s", report)
	}
}
