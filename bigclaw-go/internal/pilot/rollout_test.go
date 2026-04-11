package pilot

import (
	"strings"
	"testing"
)

func TestPilotRolloutScorecardAndCandidateGate(t *testing.T) {
	scorecard := BuildPilotRolloutScorecard(84, 78, 82, 1, 88)
	if scorecard.Recommendation != "go" {
		t.Fatalf("expected recommendation go, got %+v", scorecard)
	}

	result := EvaluateCandidateGate(GateDecision{GateID: "gate-v3", Passed: true}, scorecard)
	if result.CandidateGate != "enable-by-default" {
		t.Fatalf("expected enable-by-default candidate gate, got %+v", result)
	}

	report := RenderPilotRolloutGateReport(result)
	if !strings.Contains(report, "Candidate gate") {
		t.Fatalf("expected report to contain candidate gate summary, got %q", report)
	}
}

func TestRepoNarrativeExportsRemainConsistent(t *testing.T) {
	section := RenderWeeklyRepoEvidenceSection(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})
	exports := RenderRepoNarrativeExports(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})

	if !strings.Contains(section, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits summary in markdown section, got %q", section)
	}
	if !strings.Contains(exports.Markdown, "Repo Evidence Summary") {
		t.Fatalf("expected markdown export heading, got %q", exports.Markdown)
	}
	if !strings.Contains(exports.Text, "Accepted Commits: 7") {
		t.Fatalf("expected text export accepted commits summary, got %q", exports.Text)
	}
	if !strings.Contains(exports.HTML, "<section><h2>Repo Evidence Summary</h2>") {
		t.Fatalf("expected html export heading, got %q", exports.HTML)
	}
}
