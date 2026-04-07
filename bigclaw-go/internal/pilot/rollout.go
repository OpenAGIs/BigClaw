package pilot

import (
	"fmt"
	"html"
	"strings"
)

type RolloutScorecard struct {
	Adoption               float64
	ConvergenceImprovement float64
	ReviewEfficiency       float64
	GovernanceIncidents    int
	EvidenceCompleteness   float64
	RolloutScore           float64
	Recommendation         string
}

type GateDecision struct {
	GateID  string
	Passed  bool
	Summary string
}

type CandidateGateResult struct {
	GatePassed            bool
	RolloutRecommendation string
	CandidateGate         string
	Findings              []string
}

type RepoNarrativeExports struct {
	Markdown string
	Text     string
	HTML     string
}

func BuildPilotRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency float64, governanceIncidents int, evidenceCompleteness float64) RolloutScorecard {
	score := adoption*0.25 +
		convergenceImprovement*0.25 +
		reviewEfficiency*0.2 +
		evidenceCompleteness*0.2 +
		maxFloat(0, 100.0-float64(governanceIncidents*20))*0.1
	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70

	return RolloutScorecard{
		Adoption:               round1(adoption),
		ConvergenceImprovement: round1(convergenceImprovement),
		ReviewEfficiency:       round1(reviewEfficiency),
		GovernanceIncidents:    governanceIncidents,
		EvidenceCompleteness:   round1(evidenceCompleteness),
		RolloutScore:           round1(score),
		Recommendation:         ternary(passed, "go", "hold"),
	}
}

func EvaluateCandidateGate(gateDecision GateDecision, rolloutScorecard RolloutScorecard) CandidateGateResult {
	readiness := gateDecision.Passed
	rolloutReady := rolloutScorecard.Recommendation == "go"
	result := CandidateGateResult{
		GatePassed:            readiness,
		RolloutRecommendation: rolloutScorecard.Recommendation,
		CandidateGate:         ternary(readiness && rolloutReady, "enable-by-default", "pilot-only"),
	}
	if !readiness {
		result.Findings = append(result.Findings, gateDecision.Summary)
	}
	if !rolloutReady {
		result.Findings = append(result.Findings, fmt.Sprintf("rollout score below threshold (%.1f)", rolloutScorecard.RolloutScore))
	}
	return result
}

func RenderPilotRolloutGateReport(result CandidateGateResult) string {
	findings := "none"
	if len(result.Findings) > 0 {
		findings = strings.Join(result.Findings, ", ")
	}
	lines := []string{
		"# Pilot Rollout Candidate Gate",
		"",
		fmt.Sprintf("- Gate passed: %t", result.GatePassed),
		fmt.Sprintf("- Rollout recommendation: %s", result.RolloutRecommendation),
		fmt.Sprintf("- Candidate gate: %s", result.CandidateGate),
		fmt.Sprintf("- Findings: %s", findings),
	}
	return strings.Join(lines, "\n")
}

func RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) string {
	lines := []string{
		"## Repo Evidence Summary",
		fmt.Sprintf("- Experiment Volume: %d", experimentVolume),
		fmt.Sprintf("- Converged Tasks: %d", convergedTasks),
		fmt.Sprintf("- Accepted Commits: %d", acceptedCommits),
		fmt.Sprintf("- Hottest Threads: %s", joinOrNone(hottestThreads)),
	}
	return strings.Join(lines, "\n")
}

func RenderRepoNarrativeExports(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) RepoNarrativeExports {
	markdown := RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits, hottestThreads)
	return RepoNarrativeExports{
		Markdown: markdown,
		Text:     strings.Replace(markdown, "## ", "", 1),
		HTML: "<section><h2>Repo Evidence Summary</h2>" +
			fmt.Sprintf("<p>Experiment Volume: %d</p>", experimentVolume) +
			fmt.Sprintf("<p>Converged Tasks: %d</p>", convergedTasks) +
			fmt.Sprintf("<p>Accepted Commits: %d</p>", acceptedCommits) +
			fmt.Sprintf("<p>Hottest Threads: %s</p>", html.EscapeString(joinOrNone(hottestThreads))) +
			"</section>",
	}
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ternary[T any](condition bool, yes, no T) T {
	if condition {
		return yes
	}
	return no
}
