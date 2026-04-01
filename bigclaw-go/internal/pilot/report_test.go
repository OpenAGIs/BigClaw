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

func TestKPIAndImplementationResultHandleLowerIsBetterMetrics(t *testing.T) {
	result := ImplementationResult{
		Customer:       "Design Partner C",
		Environment:    "production",
		ProductionRuns: 8,
		Incidents:      0,
		KPIs: []KPI{
			{Name: "automation-coverage", Target: 80, Actual: 82, HigherIsBetter: true},
			{Name: "manual-review-hours", Target: 5, Actual: 4, HigherIsBetter: false},
		},
	}

	if !result.KPIs[1].Met() {
		t.Fatalf("expected lower-is-better KPI to pass, got %+v", result.KPIs[1])
	}
	if got := result.KPIPassRate(); got != 100.0 {
		t.Fatalf("expected kpi pass rate 100.0, got %v", got)
	}
	if !result.Ready() {
		t.Fatalf("expected ready=true, got false")
	}
}

func TestImplementationResultNotReadyBelowPassThreshold(t *testing.T) {
	result := ImplementationResult{
		Customer:       "Design Partner D",
		Environment:    "production",
		ProductionRuns: 5,
		Incidents:      0,
		KPIs: []KPI{
			{Name: "automation-coverage", Target: 80, Actual: 81, HigherIsBetter: true},
			{Name: "review-hours", Target: 5, Actual: 6, HigherIsBetter: false},
			{Name: "handoff-latency", Target: 10, Actual: 12, HigherIsBetter: false},
			{Name: "evidence-completeness", Target: 95, Actual: 97, HigherIsBetter: true},
			{Name: "stability", Target: 99, Actual: 98, HigherIsBetter: true},
		},
	}

	if got := result.KPIPassRate(); got != 40.0 {
		t.Fatalf("expected kpi pass rate 40.0, got %v", got)
	}
	if result.Ready() {
		t.Fatalf("expected ready=false below threshold")
	}
}
