package product

import "testing"

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
	if report == "" || !contains(report, "Candidate gate") {
		t.Fatalf("expected candidate gate report, got %q", report)
	}
}

func TestRepoWeeklyNarrativeExportsRemainConsistent(t *testing.T) {
	section := RenderWeeklyRepoEvidenceSection(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})
	exports := RenderRepoNarrativeExports(14, 9, 7, []string{"repo/ope-168", "repo/ope-170"})

	if !contains(section, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in section, got %q", section)
	}
	if !contains(exports.Markdown, "Repo Evidence Summary") {
		t.Fatalf("expected markdown repo evidence summary, got %q", exports.Markdown)
	}
	if !contains(exports.Text, "Accepted Commits: 7") {
		t.Fatalf("expected accepted commits in text export, got %q", exports.Text)
	}
	if !contains(exports.HTML, "<section><h2>Repo Evidence Summary</h2>") {
		t.Fatalf("expected HTML repo evidence summary, got %q", exports.HTML)
	}
}

func contains(s, want string) bool {
	return len(s) >= len(want) && (s == want || len(s) > len(want) && (containsAt(s, want, 0) || contains(s[1:], want)))
}

func containsAt(s, want string, start int) bool {
	if len(s[start:]) < len(want) {
		return false
	}
	return s[start:start+len(want)] == want
}
