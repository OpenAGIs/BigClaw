package reporollout

import (
	"fmt"
	"strings"
)

type EntryGateDecision struct {
	GateID string `json:"gate_id"`
	Passed bool   `json:"passed"`
}

type NarrativeExports struct {
	Markdown string `json:"markdown"`
	Text     string `json:"text"`
	HTML     string `json:"html"`
}

func BuildPilotRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency, governanceIncidents, evidenceCompleteness int) map[string]any {
	recommendation := "hold"
	if adoption >= 80 && convergenceImprovement >= 75 && reviewEfficiency >= 80 && governanceIncidents <= 1 && evidenceCompleteness >= 85 {
		recommendation = "go"
	}
	return map[string]any{
		"adoption":                adoption,
		"convergence_improvement": convergenceImprovement,
		"review_efficiency":       reviewEfficiency,
		"governance_incidents":    governanceIncidents,
		"evidence_completeness":   evidenceCompleteness,
		"recommendation":          recommendation,
	}
}

func EvaluateCandidateGate(gateDecision EntryGateDecision, rolloutScorecard map[string]any) map[string]any {
	candidateGate := "hold"
	if gateDecision.Passed && rolloutScorecard["recommendation"] == "go" {
		candidateGate = "enable-by-default"
	}
	return map[string]any{
		"gate_id":           gateDecision.GateID,
		"passed":            gateDecision.Passed,
		"candidate_gate":    candidateGate,
		"recommendation":    rolloutScorecard["recommendation"],
		"rollout_scorecard": rolloutScorecard,
	}
}

func RenderPilotRolloutGateReport(result map[string]any) string {
	return fmt.Sprintf("# Pilot Rollout Gate\n\n- Gate ID: %v\n- Candidate gate: %v\n", result["gate_id"], result["candidate_gate"])
}

func RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) string {
	return strings.Join([]string{
		"## Repo Evidence Summary",
		fmt.Sprintf("- Experiment Volume: %d", experimentVolume),
		fmt.Sprintf("- Converged Tasks: %d", convergedTasks),
		fmt.Sprintf("- Accepted Commits: %d", acceptedCommits),
		fmt.Sprintf("- Hottest Threads: %s", strings.Join(hottestThreads, ", ")),
	}, "\n")
}

func RenderRepoNarrativeExports(experimentVolume, convergedTasks, acceptedCommits int, hottestThreads []string) NarrativeExports {
	section := RenderWeeklyRepoEvidenceSection(experimentVolume, convergedTasks, acceptedCommits, hottestThreads)
	return NarrativeExports{
		Markdown: section,
		Text:     section,
		HTML:     fmt.Sprintf("<section><h2>Repo Evidence Summary</h2><p>Accepted Commits: %d</p></section>", acceptedCommits),
	}
}
