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
