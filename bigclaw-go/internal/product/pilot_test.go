package product

import "strings"

import "testing"

func TestPilotReadyWhenKPIsPassAndNoIncidents(t *testing.T) {
	result := PilotImplementationResult{
		Customer:       "Design Partner A",
		Environment:    "production",
		ProductionRuns: 12,
		Incidents:      0,
		KPIs: []PilotKPI{
			{Name: "automation-coverage", Target: 80, Actual: 86, HigherIsBetter: true},
			{Name: "lead-time-hours", Target: 6, Actual: 5, HigherIsBetter: false},
		},
	}

	if got := result.KPIPassRate(); got != 100.0 {
		t.Fatalf("unexpected KPI pass rate: %v", got)
	}
	if !result.Ready() {
		t.Fatal("expected pilot result to be ready")
	}
}

func TestRenderPilotImplementationReportContainsReadinessFields(t *testing.T) {
	result := PilotImplementationResult{
		Customer:       "Design Partner B",
		Environment:    "staging",
		ProductionRuns: 0,
		Incidents:      1,
		KPIs: []PilotKPI{
			{Name: "automation-coverage", Target: 80, Actual: 72, HigherIsBetter: true},
		},
	}

	report := RenderPilotImplementationReport(result)
	if !strings.Contains(report, "Pilot Implementation Report") {
		t.Fatalf("expected report heading, got %q", report)
	}
	if !strings.Contains(report, "Ready: false") {
		t.Fatalf("expected readiness field, got %q", report)
	}
	if !strings.Contains(report, "KPI Pass Rate: 0.0%") {
		t.Fatalf("expected KPI pass rate field, got %q", report)
	}
}
