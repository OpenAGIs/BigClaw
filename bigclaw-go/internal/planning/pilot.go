package planning

import (
	"fmt"
	"strings"
)

type EntryGateDecision struct {
	GateID  string `json:"gate_id"`
	Passed  bool   `json:"passed"`
	Summary string `json:"summary,omitempty"`
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
		maxFloat(0, 100-float64(governanceIncidents*20))*0.1
	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70
	return PilotRolloutScorecard{
		Adoption:               round1(adoption),
		ConvergenceImprovement: round1(convergenceImprovement),
		ReviewEfficiency:       round1(reviewEfficiency),
		GovernanceIncidents:    governanceIncidents,
		EvidenceCompleteness:   round1(evidenceCompleteness),
		RolloutScore:           round1(score),
		Recommendation:         ternary(passed, "go", "hold"),
	}
}

func EvaluateCandidateGate(gateDecision EntryGateDecision, rolloutScorecard PilotRolloutScorecard) CandidateGateResult {
	rolloutReady := rolloutScorecard.Recommendation == "go"
	recommendation := "pilot-only"
	if gateDecision.Passed && rolloutReady {
		recommendation = "enable-by-default"
	}
	findings := make([]string, 0, 2)
	if !gateDecision.Passed && strings.TrimSpace(gateDecision.Summary) != "" {
		findings = append(findings, gateDecision.Summary)
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
	lines := []string{
		"# Pilot Rollout Candidate Gate",
		"",
		fmt.Sprintf("- Gate passed: %t", result.GatePassed),
		fmt.Sprintf("- Rollout recommendation: %s", result.RolloutRecommendation),
		fmt.Sprintf("- Candidate gate: %s", result.CandidateGate),
		fmt.Sprintf("- Findings: %s", joinOrNone(result.Findings)),
	}
	return strings.Join(lines, "\n")
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func ternary[T any](cond bool, left, right T) T {
	if cond {
		return left
	}
	return right
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
