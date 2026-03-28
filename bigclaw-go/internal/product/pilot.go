package product

import (
	"fmt"
	"strings"
)

type PilotKPI struct {
	Name           string
	Target         float64
	Actual         float64
	HigherIsBetter bool
}

func (k PilotKPI) Met() bool {
	if k.HigherIsBetter {
		return k.Actual >= k.Target
	}
	return k.Actual <= k.Target
}

type PilotImplementationResult struct {
	Customer       string
	Environment    string
	KPIs           []PilotKPI
	ProductionRuns int
	Incidents      int
}

func (r PilotImplementationResult) KPIPassRate() float64 {
	if len(r.KPIs) == 0 {
		return 0
	}
	passed := 0
	for _, kpi := range r.KPIs {
		if kpi.Met() {
			passed++
		}
	}
	return float64(int((float64(passed)/float64(len(r.KPIs))*1000)+0.5)) / 10
}

func (r PilotImplementationResult) Ready() bool {
	return r.ProductionRuns > 0 && r.Incidents == 0 && r.KPIPassRate() >= 80.0
}

func RenderPilotImplementationReport(result PilotImplementationResult) string {
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
		return strings.Join(lines, "\n") + "\n"
	}
	for _, kpi := range result.KPIs {
		lines = append(lines, fmt.Sprintf("- %s: target=%v actual=%v met=%t", kpi.Name, trimFloat(kpi.Target), trimFloat(kpi.Actual), kpi.Met()))
	}
	return strings.Join(lines, "\n") + "\n"
}

func trimFloat(value float64) any {
	if value == float64(int64(value)) {
		return int64(value)
	}
	return value
}
