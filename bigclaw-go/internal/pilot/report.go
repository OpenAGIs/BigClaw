package pilot

import (
	"fmt"
	"strings"
)

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
