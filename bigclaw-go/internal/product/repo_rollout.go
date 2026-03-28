package product

import (
	"fmt"
	"html"
	"strings"
)

type EntryGateDecision struct {
	GateID string `json:"gate_id,omitempty"`
	Passed bool   `json:"passed"`
}

type PilotRolloutScorecard struct {
	Adoption               float64 `json:"adoption"`
	ConvergenceImprovement float64 `json:"convergence_improvement"`
	ReviewEfficiency       float64 `json:"review_efficiency"`
	GovernanceIncidents    int     `json:"governance_incidents"`
	EvidenceCompleteness   float64 `json:"evidence_completeness"`
	RolloutScore           float64 `json:"rollout_score"`
	Recommendation         string  `json:"recommendation"`
}

type CandidateGateResult struct {
	GatePassed            bool     `json:"gate_passed"`
	RolloutRecommendation string   `json:"rollout_recommendation"`
	CandidateGate         string   `json:"candidate_gate"`
	Findings              []string `json:"findings,omitempty"`
}

func BuildPilotRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency float64, governanceIncidents int, evidenceCompleteness float64) PilotRolloutScorecard {
	score := adoption*0.25 +
		convergenceImprovement*0.25 +
		reviewEfficiency*0.2 +
		evidenceCompleteness*0.2 +
		maxFloat(0.0, 100.0-(float64(governanceIncidents)*20.0))*0.1
	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70
	recommendation := "hold"
	if passed {
		recommendation = "go"
	}
	return PilotRolloutScorecard{
		Adoption:               round1(adoption),
		ConvergenceImprovement: round1(convergenceImprovement),
		ReviewEfficiency:       round1(reviewEfficiency),
		GovernanceIncidents:    governanceIncidents,
		EvidenceCompleteness:   round1(evidenceCompleteness),
		RolloutScore:           round1(score),
		Recommendation:         recommendation,
	}
}

func EvaluateCandidateGate(gateDecision EntryGateDecision, rolloutScorecard PilotRolloutScorecard) CandidateGateResult {
	rolloutReady := rolloutScorecard.Recommendation == "go"
	recommendation := "pilot-only"
	if gateDecision.Passed && rolloutReady {
		recommendation = "enable-by-default"
	}
	findings := make([]string, 0, 2)
	if !gateDecision.Passed {
		findings = append(findings, fmt.Sprintf("gate %s remains on hold", strings.TrimSpace(gateDecision.GateID)))
	}
	if !rolloutReady {
		findings = append(findings, fmt.Sprintf("rollout score below threshold (%.1f)", rolloutScorecard.RolloutScore))
	}
	return CandidateGateResult{
		GatePassed:            gateDecision.Passed,
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

type RepoNarrativeExports struct {
	Markdown string `json:"markdown"`
	Text     string `json:"text"`
	HTML     string `json:"html"`
}

func RenderRepoNarrativeExports(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) RepoNarrativeExports {
	markdown := RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits, hottestThreads)
	threads := "none"
	if len(hottestThreads) > 0 {
		threads = strings.Join(hottestThreads, ", ")
	}
	return RepoNarrativeExports{
		Markdown: markdown,
		Text:     strings.Replace(markdown, "## ", "", 1),
		HTML: "<section><h2>Repo Evidence Summary</h2>" +
			fmt.Sprintf("<p>Experiment Volume: %d</p>", experimentVolume) +
			fmt.Sprintf("<p>Converged Tasks: %d</p>", convergedTasks) +
			fmt.Sprintf("<p>Accepted Commits: %d</p>", acceptedCommits) +
			fmt.Sprintf("<p>Hottest Threads: %s</p>", html.EscapeString(threads)) +
			"</section>",
	}
}
