package product

import (
	"fmt"
	"html"
	"strings"
)

type EntryGateDecision struct {
	GateID              string   `json:"gate_id"`
	Passed              bool     `json:"passed"`
	ReadyCandidateIDs   []string `json:"ready_candidate_ids,omitempty"`
	BlockedCandidateIDs []string `json:"blocked_candidate_ids,omitempty"`
	MissingCapabilities []string `json:"missing_capabilities,omitempty"`
	MissingEvidence     []string `json:"missing_evidence,omitempty"`
	BaselineReady       bool     `json:"baseline_ready"`
	BaselineFindings    []string `json:"baseline_findings,omitempty"`
	BlockerCount        int      `json:"blocker_count"`
}

func (d EntryGateDecision) Summary() string {
	status := "HOLD"
	if d.Passed {
		status = "PASS"
	}
	return fmt.Sprintf(
		"%s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d",
		status,
		len(d.ReadyCandidateIDs),
		d.BlockerCount,
		len(d.MissingCapabilities),
		len(d.MissingEvidence),
		len(d.BaselineFindings),
	)
}

func BuildPilotRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency float64, governanceIncidents int, evidenceCompleteness float64) map[string]any {
	score := adoption*0.25 +
		convergenceImprovement*0.25 +
		reviewEfficiency*0.2 +
		evidenceCompleteness*0.2 +
		maxFloat(0, 100.0-(float64(governanceIncidents)*20.0))*0.1

	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70
	recommendation := "hold"
	if passed {
		recommendation = "go"
	}
	return map[string]any{
		"adoption":                round1(adoption),
		"convergence_improvement": round1(convergenceImprovement),
		"review_efficiency":       round1(reviewEfficiency),
		"governance_incidents":    governanceIncidents,
		"evidence_completeness":   round1(evidenceCompleteness),
		"rollout_score":           round1(score),
		"recommendation":          recommendation,
	}
}

func EvaluateCandidateGate(gateDecision EntryGateDecision, rolloutScorecard map[string]any) map[string]any {
	readiness := gateDecision.Passed
	rolloutReady := rolloutScorecard["recommendation"] == "go"
	recommendation := "pilot-only"
	if readiness && rolloutReady {
		recommendation = "enable-by-default"
	}
	findings := make([]string, 0)
	if !readiness {
		findings = append(findings, gateDecision.Summary())
	}
	if !rolloutReady {
		findings = append(findings, fmt.Sprintf("rollout score below threshold (%v)", rolloutScorecard["rollout_score"]))
	}
	return map[string]any{
		"gate_passed":            readiness,
		"rollout_recommendation": fmt.Sprint(rolloutScorecard["recommendation"]),
		"candidate_gate":         recommendation,
		"findings":               findings,
	}
}

func RenderPilotRolloutGateReport(result map[string]any) string {
	findings, _ := result["findings"].([]string)
	if findings == nil {
		findings = []string{}
	}
	lines := []string{
		"# Pilot Rollout Candidate Gate",
		"",
		fmt.Sprintf("- Gate passed: %v", result["gate_passed"]),
		fmt.Sprintf("- Rollout recommendation: %v", result["rollout_recommendation"]),
		fmt.Sprintf("- Candidate gate: %v", result["candidate_gate"]),
	}
	if len(findings) == 0 {
		lines = append(lines, "- Findings: none")
	} else {
		lines = append(lines, "- Findings: "+strings.Join(findings, ", "))
	}
	return strings.Join(lines, "\n")
}

func RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) string {
	lines := []string{
		"## Repo Evidence Summary",
		fmt.Sprintf("- Experiment Volume: %d", experimentVolume),
		fmt.Sprintf("- Converged Tasks: %d", convergedTasks),
		fmt.Sprintf("- Accepted Commits: %d", acceptedCommits),
	}
	if len(hottestThreads) == 0 {
		lines = append(lines, "- Hottest Threads: none")
	} else {
		lines = append(lines, "- Hottest Threads: "+strings.Join(hottestThreads, ", "))
	}
	return strings.Join(lines, "\n")
}

func RenderRepoNarrativeExports(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) map[string]string {
	markdownText := RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits, hottestThreads)
	plainText := strings.ReplaceAll(markdownText, "## ", "")
	threadText := "none"
	if len(hottestThreads) > 0 {
		threadText = strings.Join(hottestThreads, ", ")
	}
	htmlText := "<section><h2>Repo Evidence Summary</h2>" +
		fmt.Sprintf("<p>Experiment Volume: %d</p>", experimentVolume) +
		fmt.Sprintf("<p>Converged Tasks: %d</p>", convergedTasks) +
		fmt.Sprintf("<p>Accepted Commits: %d</p>", acceptedCommits) +
		fmt.Sprintf("<p>Hottest Threads: %s</p>", html.EscapeString(threadText)) +
		"</section>"
	return map[string]string{"markdown": markdownText, "text": plainText, "html": htmlText}
}
