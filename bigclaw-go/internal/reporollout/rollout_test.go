package reporollout

import (
	"strings"
	"testing"
)

func TestPilotRolloutScorecardAndCandidateGate(t *testing.T) {
	scorecard := BuildPilotRolloutScorecard(84, 78, 82, 1, 88)
	if scorecard["recommendation"] != "go" {
		t.Fatalf("expected go recommendation, got %+v", scorecard)
	}

	result := EvaluateCandidateGate(EntryGateDecision{GateID: "gate-v3", Passed: true}, scorecard)
	if result["candidate_gate"] != "enable-by-default" {
		t.Fatalf("expected enable-by-default gate, got %+v", result)
	}
	report := RenderPilotRolloutGateReport(result)
	if !strings.Contains(report, "Candidate gate") {
		t.Fatalf("expected candidate gate text, got %s", report)
	}
}

func TestRepoWeeklyNarrativeExportsRemainConsistent(t *testing.T) {
	section := RenderWeeklyRepoEvidenceSection(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})
	exports := RenderRepoNarrativeExports(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})

	if !strings.Contains(section, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in section, got %s", section)
	}
	if !strings.Contains(exports.Markdown, "Repo Evidence Summary") {
		t.Fatalf("expected markdown summary, got %+v", exports)
	}
	if !strings.Contains(exports.Text, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in text, got %+v", exports)
	}
	if !strings.Contains(exports.HTML, "<section><h2>Repo Evidence Summary</h2>") {
		t.Fatalf("expected html section wrapper, got %+v", exports)
	}
}
