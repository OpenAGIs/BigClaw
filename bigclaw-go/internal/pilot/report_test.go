package pilot

import (
	"strings"
	"testing"
)

func TestImplementationResultReadyWhenKPIsPassAndNoIncidents(t *testing.T) {
	result := ImplementationResult{
		Customer:       "Design Partner A",
		Environment:    "production",
		ProductionRuns: 12,
		Incidents:      0,
		KPIs: []KPI{
			{Name: "automation-coverage", Target: 80, Actual: 86, HigherIsBetter: true},
			{Name: "lead-time-hours", Target: 6, Actual: 5, HigherIsBetter: false},
		},
	}

	if got := result.KPIPassRate(); got != 100.0 {
		t.Fatalf("expected kpi pass rate 100.0, got %v", got)
	}
	if !result.Ready() {
		t.Fatalf("expected ready=true, got false")
	}
}

func TestRenderPilotImplementationReportContainsReadinessFields(t *testing.T) {
	result := ImplementationResult{
		Customer:       "Design Partner B",
		Environment:    "staging",
		ProductionRuns: 0,
		Incidents:      1,
		KPIs: []KPI{
			{Name: "automation-coverage", Target: 80, Actual: 72, HigherIsBetter: true},
		},
	}

	report := RenderImplementationReport(result)
	for _, want := range []string{
		"Pilot Implementation Report",
		"Ready: false",
		"KPI Pass Rate: 0.0%",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %q", want, report)
		}
	}
}

func TestBuildRolloutScorecardAndCandidateGate(t *testing.T) {
	scorecard := BuildRolloutScorecard(84, 78, 82, 1, 88)
	if scorecard.Recommendation != "go" {
		t.Fatalf("expected recommendation go, got %+v", scorecard)
	}

	result := EvaluateCandidateGate(CandidateGateDecision{
		GateID: "gate-v3",
		Passed: true,
	}, scorecard)
	if result.CandidateGate != "enable-by-default" {
		t.Fatalf("expected enable-by-default gate, got %+v", result)
	}

	report := RenderCandidateGateReport(result)
	if !strings.Contains(report, "Candidate gate") {
		t.Fatalf("expected candidate gate in report, got %q", report)
	}
}
