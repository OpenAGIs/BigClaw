package rollout

import (
	"fmt"
	"html"
	"math"
	"strings"
)

type EntryGateDecision struct {
	GateID string `json:"gate_id"`
	Passed bool   `json:"passed"`
	Summary string `json:"summary,omitempty"`
}

type Scorecard struct {
	Adoption               float64 `json:"adoption"`
	ConvergenceImprovement float64 `json:"convergence_improvement"`
	ReviewEfficiency       float64 `json:"review_efficiency"`
	GovernanceIncidents    int     `json:"governance_incidents"`
	EvidenceCompleteness   float64 `json:"evidence_completeness"`
	RolloutScore           float64 `json:"rollout_score"`
	Recommendation         string  `json:"recommendation"`
}

type CandidateGateResult struct {
	GatePassed           bool     `json:"gate_passed"`
	RolloutRecommendation string   `json:"rollout_recommendation"`
	CandidateGate        string   `json:"candidate_gate"`
	Findings             []string `json:"findings,omitempty"`
}

func BuildPilotRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency float64, governanceIncidents int, evidenceCompleteness float64) Scorecard {
	score := adoption*0.25 +
		convergenceImprovement*0.25 +
		reviewEfficiency*0.2 +
		evidenceCompleteness*0.2 +
		math.Max(0, 100-float64(governanceIncidents*20))*0.1
	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70
	return Scorecard{
		Adoption:               round1(adoption),
		ConvergenceImprovement: round1(convergenceImprovement),
		ReviewEfficiency:       round1(reviewEfficiency),
		GovernanceIncidents:    governanceIncidents,
		EvidenceCompleteness:   round1(evidenceCompleteness),
		RolloutScore:           round1(score),
		Recommendation:         ternary(passed, "go", "hold"),
	}
}

func EvaluateCandidateGate(gateDecision EntryGateDecision, rolloutScorecard Scorecard) CandidateGateResult {
	readiness := gateDecision.Passed
	rolloutReady := rolloutScorecard.Recommendation == "go"
	recommendation := ternary(readiness && rolloutReady, "enable-by-default", "pilot-only")
	findings := make([]string, 0, 2)
	if !readiness {
		findings = append(findings, gateDecision.Summary)
	}
	if !rolloutReady {
		findings = append(findings, fmt.Sprintf("rollout score below threshold (%.1f)", rolloutScorecard.RolloutScore))
	}
	return CandidateGateResult{
		GatePassed:            readiness,
		RolloutRecommendation: rolloutScorecard.Recommendation,
		CandidateGate:         recommendation,
		Findings:              findings,
	}
}

func RenderPilotRolloutGateReport(result CandidateGateResult) string {
	findings := "none"
	if len(result.Findings) > 0 {
		findings = strings.Join(result.Findings, ", ")
	}
	return strings.Join([]string{
		"# Pilot Rollout Candidate Gate",
		"",
		fmt.Sprintf("- Gate passed: %t", result.GatePassed),
		fmt.Sprintf("- Rollout recommendation: %s", result.RolloutRecommendation),
		fmt.Sprintf("- Candidate gate: %s", result.CandidateGate),
		fmt.Sprintf("- Findings: %s", findings),
	}, "\n")
}

func RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) string {
	threads := "none"
	if len(hottestThreads) > 0 {
		threads = strings.Join(hottestThreads, ", ")
	}
	return strings.Join([]string{
		"## Repo Evidence Summary",
		fmt.Sprintf("- Experiment Volume: %d", experimentVolume),
		fmt.Sprintf("- Converged Tasks: %d", convergedTasks),
		fmt.Sprintf("- Accepted Commits: %d", acceptedCommits),
		fmt.Sprintf("- Hottest Threads: %s", threads),
	}, "\n")
}

func RenderRepoNarrativeExports(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) map[string]string {
	markdownText := RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits, hottestThreads)
	plainText := strings.Replace(markdownText, "## ", "", 1)
	threads := "none"
	if len(hottestThreads) > 0 {
		threads = strings.Join(hottestThreads, ", ")
	}
	htmlText := "<section><h2>Repo Evidence Summary</h2>" +
		fmt.Sprintf("<p>Experiment Volume: %d</p>", experimentVolume) +
		fmt.Sprintf("<p>Converged Tasks: %d</p>", convergedTasks) +
		fmt.Sprintf("<p>Accepted Commits: %d</p>", acceptedCommits) +
		fmt.Sprintf("<p>Hottest Threads: %s</p>", html.EscapeString(threads)) +
		"</section>"
	return map[string]string{
		"markdown": markdownText,
		"text":     plainText,
		"html":     htmlText,
	}
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func ternary[T any](condition bool, left, right T) T {
	if condition {
		return left
	}
	return right
}
