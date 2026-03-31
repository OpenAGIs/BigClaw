package pilot

import (
	"fmt"
	"strings"
)

type CandidateGateDecision struct {
	GateID string
	Passed bool
}

type RolloutScorecard struct {
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

type KPI struct {
	Name           string
	Target         float64
	Actual         float64
	HigherIsBetter bool
}

func (k KPI) Met() bool {
	if k.HigherIsBetter {
		return k.Actual >= k.Target
	}
	return k.Actual <= k.Target
}

type ImplementationResult struct {
	Customer       string
	Environment    string
	KPIs           []KPI
	ProductionRuns int
	Incidents      int
}

func (r ImplementationResult) KPIPassRate() float64 {
	if len(r.KPIs) == 0 {
		return 0
	}
	passed := 0
	for _, kpi := range r.KPIs {
		if kpi.Met() {
			passed++
		}
	}
	return float64(int((float64(passed)/float64(len(r.KPIs)))*1000+0.5)) / 10
}

func (r ImplementationResult) Ready() bool {
	return r.ProductionRuns > 0 && r.Incidents == 0 && r.KPIPassRate() >= 80
}

func RenderImplementationReport(result ImplementationResult) string {
	lines := []string{
		"# Pilot Implementation Report",
		"",
		fmt.Sprintf("- Customer: %s", result.Customer),
		fmt.Sprintf("- Environment: %s", result.Environment),
		fmt.Sprintf("- Production Runs: %d", result.ProductionRuns),
		fmt.Sprintf("- Incidents: %d", result.Incidents),
		fmt.Sprintf("- KPI Pass Rate: %.1f%%", result.KPIPassRate()),
		fmt.Sprintf("- Ready: %t", result.Ready()),
		"",
		"## KPI Details",
		"",
	}
	if len(result.KPIs) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, kpi := range result.KPIs {
			lines = append(lines, fmt.Sprintf("- %s: target=%v actual=%v met=%t", strings.TrimSpace(kpi.Name), kpi.Target, kpi.Actual, kpi.Met()))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildRolloutScorecard(adoption, convergenceImprovement, reviewEfficiency float64, governanceIncidents int, evidenceCompleteness float64) RolloutScorecard {
	score := adoption*0.25 +
		convergenceImprovement*0.25 +
		reviewEfficiency*0.2 +
		evidenceCompleteness*0.2 +
		maxFloat(0, 100.0-(float64(governanceIncidents)*20.0))*0.1
	passed := score >= 75 && governanceIncidents <= 2 && evidenceCompleteness >= 70
	return RolloutScorecard{
		Adoption:               round1(adoption),
		ConvergenceImprovement: round1(convergenceImprovement),
		ReviewEfficiency:       round1(reviewEfficiency),
		GovernanceIncidents:    governanceIncidents,
		EvidenceCompleteness:   round1(evidenceCompleteness),
		RolloutScore:           round1(score),
		Recommendation:         ternaryString(passed, "go", "hold"),
	}
}

func EvaluateCandidateGate(gateDecision CandidateGateDecision, rolloutScorecard RolloutScorecard) CandidateGateResult {
	readiness := gateDecision.Passed
	rolloutReady := rolloutScorecard.Recommendation == "go"
	recommendation := "pilot-only"
	if readiness && rolloutReady {
		recommendation = "enable-by-default"
	}
	findings := []string{}
	if !readiness {
		findings = append(findings, fmt.Sprintf("%s: gate not passed", gateDecision.GateID))
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

func RenderCandidateGateReport(result CandidateGateResult) string {
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

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func ternaryString(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
