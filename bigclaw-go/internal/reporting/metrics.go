package reporting

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

var (
	statusComplete    = map[string]struct{}{"approved": {}, "accepted": {}, "completed": {}, "succeeded": {}}
	statusActionable  = map[string]struct{}{"needs-approval": {}, "failed": {}, "rejected": {}}
	metricDefinitions = []OperationsMetricDefinition{
		{
			MetricID:     "runs-today",
			Label:        "Runs Today",
			Unit:         "runs",
			Direction:    "up",
			Formula:      "count(run.started_at within [period_start, period_end])",
			Description:  "Number of runs that started inside the reporting day window.",
			SourceFields: []string{"started_at"},
		},
		{
			MetricID:     "avg-lead-time",
			Label:        "Avg Lead Time",
			Unit:         "m",
			Direction:    "down",
			Formula:      "sum(cycle_minutes for runs with started_at and ended_at) / measured_runs",
			Description:  "Average elapsed minutes from run start to run end for runs with complete timestamps.",
			SourceFields: []string{"started_at", "ended_at"},
		},
		{
			MetricID:     "intervention-rate",
			Label:        "Intervention Rate",
			Unit:         "%",
			Direction:    "down",
			Formula:      "100 * actionable_runs / total_runs",
			Description:  "Share of runs that require operator intervention because they ended in an actionable status.",
			SourceFields: []string{"status"},
		},
		{
			MetricID:     "sla",
			Label:        "SLA",
			Unit:         "%",
			Direction:    "up",
			Formula:      "100 * compliant_runs / measured_runs where compliant_runs have cycle_minutes <= sla_target_minutes",
			Description:  "Share of measured runs that met the SLA target.",
			SourceFields: []string{"started_at", "ended_at"},
		},
		{
			MetricID:     "regression",
			Label:        "Regression",
			Unit:         "cases",
			Direction:    "down",
			Formula:      "count(current.compare(baseline) deltas < 0 or pass->fail transitions)",
			Description:  "Number of benchmark cases that regressed against the provided baseline suite.",
			SourceFields: []string{"benchmark.current", "benchmark.baseline"},
		},
		{
			MetricID:     "risk",
			Label:        "Risk",
			Unit:         "score",
			Direction:    "down",
			Formula:      "sum(resolved_run_risk_score) / runs_with_risk where risk_score.total wins over risk_level mapping low=25, medium=60, high=90",
			Description:  "Average per-run risk score from explicit risk scores or normalized risk levels.",
			SourceFields: []string{"risk_score.total", "risk_level"},
		},
		{
			MetricID:     "spend",
			Label:        "Spend",
			Unit:         "USD",
			Direction:    "down",
			Formula:      "sum(first non-null of spend_usd, cost_usd, spend, cost across runs)",
			Description:  "Total reported run spend in USD over the reporting window.",
			SourceFields: []string{"spend_usd", "cost_usd", "spend", "cost"},
		},
	}
)

type OperationsMetricDefinition struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Unit         string   `json:"unit"`
	Direction    string   `json:"direction"`
	Formula      string   `json:"formula"`
	Description  string   `json:"description"`
	SourceFields []string `json:"source_fields,omitempty"`
}

type OperationsMetricValue struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Value        float64  `json:"value"`
	DisplayValue string   `json:"display_value"`
	Numerator    float64  `json:"numerator"`
	Denominator  float64  `json:"denominator"`
	Unit         string   `json:"unit"`
	Evidence     []string `json:"evidence,omitempty"`
}

type OperationsMetricSpec struct {
	Name         string                       `json:"name"`
	GeneratedAt  string                       `json:"generated_at"`
	PeriodStart  string                       `json:"period_start"`
	PeriodEnd    string                       `json:"period_end"`
	TimezoneName string                       `json:"timezone_name"`
	Definitions  []OperationsMetricDefinition `json:"definitions,omitempty"`
	Values       []OperationsMetricValue      `json:"values,omitempty"`
}

type MetricRiskScore struct {
	Total float64 `json:"total"`
}

type MetricRun struct {
	RunID     string           `json:"run_id,omitempty"`
	TaskID    string           `json:"task_id,omitempty"`
	StartedAt string           `json:"started_at,omitempty"`
	EndedAt   string           `json:"ended_at,omitempty"`
	Status    string           `json:"status,omitempty"`
	RiskLevel domain.RiskLevel `json:"risk_level,omitempty"`
	RiskScore *MetricRiskScore `json:"risk_score,omitempty"`
	SpendUSD  *float64         `json:"spend_usd,omitempty"`
	CostUSD   *float64         `json:"cost_usd,omitempty"`
	Spend     *float64         `json:"spend,omitempty"`
	Cost      *float64         `json:"cost,omitempty"`
}

type MetricSpecOptions struct {
	PeriodStart       string
	PeriodEnd         string
	TimezoneName      string
	GeneratedAt       string
	SLATargetMinutes  int
	CurrentBenchmark  *BenchmarkSuite
	BaselineBenchmark *BenchmarkSuite
}

type BenchmarkCaseResult struct {
	CaseID string `json:"case_id"`
	Score  int    `json:"score"`
	Passed bool   `json:"passed"`
}

type BenchmarkSuite struct {
	Version string                `json:"version"`
	Results []BenchmarkCaseResult `json:"results,omitempty"`
}

type RegressionFinding struct {
	CaseID        string `json:"case_id"`
	BaselineScore int    `json:"baseline_score"`
	CurrentScore  int    `json:"current_score"`
	Delta         int    `json:"delta"`
	Severity      string `json:"severity"`
	Summary       string `json:"summary"`
}

func BuildMetricSpec(runs []MetricRun, options MetricSpecOptions) (OperationsMetricSpec, error) {
	periodStart, ok := parseMetricTimestamp(options.PeriodStart)
	if !ok {
		return OperationsMetricSpec{}, fmt.Errorf("period_start and period_end must be valid ISO-8601 timestamps with period_end >= period_start")
	}
	periodEnd, ok := parseMetricTimestamp(options.PeriodEnd)
	if !ok || periodEnd.Before(periodStart) {
		return OperationsMetricSpec{}, fmt.Errorf("period_start and period_end must be valid ISO-8601 timestamps with period_end >= period_start")
	}
	slaTarget := options.SLATargetMinutes
	if slaTarget <= 0 {
		slaTarget = 60
	}
	timezoneName := strings.TrimSpace(options.TimezoneName)
	if timezoneName == "" {
		timezoneName = "UTC"
	}
	generatedAt := strings.TrimSpace(options.GeneratedAt)
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	runsToday := 0
	leadTimeSum := 0.0
	leadTimeCount := 0
	actionableRuns := 0
	slaCompliantRuns := 0
	riskSum := 0.0
	riskCount := 0
	spendTotal := 0.0

	for _, run := range runs {
		if startedAt, ok := parseMetricTimestamp(run.StartedAt); ok && !startedAt.Before(periodStart) && !startedAt.After(periodEnd) {
			runsToday++
		}
		if cycleMinutes, ok := metricCycleMinutes(run); ok {
			leadTimeSum += cycleMinutes
			leadTimeCount++
			if cycleMinutes <= float64(slaTarget) {
				slaCompliantRuns++
			}
		}
		if _, ok := statusActionable[strings.ToLower(strings.TrimSpace(run.Status))]; ok {
			actionableRuns++
		}
		if riskScore, ok := resolveMetricRunRiskScore(run); ok {
			riskSum += riskScore
			riskCount++
		}
		spendTotal += resolveMetricRunSpend(run)
	}

	regressionFindings := AnalyzeRegressions(options.CurrentBenchmark, options.BaselineBenchmark)
	totalRuns := len(runs)
	avgLead := 0.0
	if leadTimeCount > 0 {
		avgLead = roundTo(leadTimeSum/float64(leadTimeCount), 1)
	}
	interventionRate := 0.0
	if totalRuns > 0 {
		interventionRate = roundTo((float64(actionableRuns)/float64(totalRuns))*100, 1)
	}
	slaValue := 0.0
	if leadTimeCount > 0 {
		slaValue = roundTo((float64(slaCompliantRuns)/float64(leadTimeCount))*100, 1)
	}
	avgRisk := 0.0
	if riskCount > 0 {
		avgRisk = roundTo(riskSum/float64(riskCount), 1)
	}
	spendTotal = roundTo(spendTotal, 2)
	currentCaseCount := 0
	if options.CurrentBenchmark != nil {
		currentCaseCount = len(options.CurrentBenchmark.Results)
	}

	values := []OperationsMetricValue{
		{
			MetricID:     "runs-today",
			Label:        "Runs Today",
			Value:        float64(runsToday),
			DisplayValue: strconv.Itoa(runsToday),
			Numerator:    float64(runsToday),
			Denominator:  float64(totalRuns),
			Unit:         "runs",
			Evidence:     []string{fmt.Sprintf("%d of %d runs started inside the reporting window.", runsToday, totalRuns)},
		},
		{
			MetricID:     "avg-lead-time",
			Label:        "Avg Lead Time",
			Value:        avgLead,
			DisplayValue: fmt.Sprintf("%.1fm", avgLead),
			Numerator:    roundTo(leadTimeSum, 1),
			Denominator:  float64(leadTimeCount),
			Unit:         "m",
			Evidence:     []string{fmt.Sprintf("%d runs had valid start/end timestamps.", leadTimeCount)},
		},
		{
			MetricID:     "intervention-rate",
			Label:        "Intervention Rate",
			Value:        interventionRate,
			DisplayValue: fmt.Sprintf("%.1f%%", interventionRate),
			Numerator:    float64(actionableRuns),
			Denominator:  float64(totalRuns),
			Unit:         "%",
			Evidence:     []string{fmt.Sprintf("Actionable statuses counted: %s.", strings.Join(sortedKeys(statusActionable), ", "))},
		},
		{
			MetricID:     "sla",
			Label:        "SLA",
			Value:        slaValue,
			DisplayValue: fmt.Sprintf("%.1f%%", slaValue),
			Numerator:    float64(slaCompliantRuns),
			Denominator:  float64(leadTimeCount),
			Unit:         "%",
			Evidence: []string{
				fmt.Sprintf("SLA target: %d minutes.", slaTarget),
				fmt.Sprintf("%d of %d measured runs met target.", slaCompliantRuns, leadTimeCount),
			},
		},
		{
			MetricID:     "regression",
			Label:        "Regression",
			Value:        float64(len(regressionFindings)),
			DisplayValue: strconv.Itoa(len(regressionFindings)),
			Numerator:    float64(len(regressionFindings)),
			Denominator:  float64(currentCaseCount),
			Unit:         "cases",
			Evidence: []string{
				fmt.Sprintf("Baseline provided: %t.", options.BaselineBenchmark != nil),
				fmt.Sprintf("Current suite provided: %t.", options.CurrentBenchmark != nil),
			},
		},
		{
			MetricID:     "risk",
			Label:        "Risk",
			Value:        avgRisk,
			DisplayValue: fmt.Sprintf("%.1f", avgRisk),
			Numerator:    roundTo(riskSum, 1),
			Denominator:  float64(riskCount),
			Unit:         "score",
			Evidence:     []string{"Risk score precedence: risk_score.total, then risk_level mapping low=25 medium=60 high=90."},
		},
		{
			MetricID:     "spend",
			Label:        "Spend",
			Value:        spendTotal,
			DisplayValue: fmt.Sprintf("$%.2f", spendTotal),
			Numerator:    spendTotal,
			Denominator:  float64(totalRuns),
			Unit:         "USD",
			Evidence:     []string{"Spend field precedence: spend_usd, cost_usd, spend, cost."},
		},
	}

	return OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  generatedAt,
		PeriodStart:  options.PeriodStart,
		PeriodEnd:    options.PeriodEnd,
		TimezoneName: timezoneName,
		Definitions:  cloneMetricDefinitions(metricDefinitions),
		Values:       values,
	}, nil
}

func AnalyzeRegressions(current *BenchmarkSuite, baseline *BenchmarkSuite) []RegressionFinding {
	if current == nil || baseline == nil {
		return nil
	}
	baselineResults := make(map[string]BenchmarkCaseResult, len(baseline.Results))
	for _, result := range baseline.Results {
		baselineResults[result.CaseID] = result
	}
	findings := make([]RegressionFinding, 0)
	for _, currentResult := range current.Results {
		baselineResult, ok := baselineResults[currentResult.CaseID]
		if !ok {
			continue
		}
		delta := currentResult.Score - baselineResult.Score
		passToFail := baselineResult.Passed && !currentResult.Passed
		if delta >= 0 && !passToFail {
			continue
		}
		severity := "medium"
		summary := fmt.Sprintf("score dropped from %d to %d", baselineResult.Score, currentResult.Score)
		if passToFail {
			summary = "case regressed from passing to failing"
		}
		if delta <= -20 || passToFail {
			severity = "high"
		}
		findings = append(findings, RegressionFinding{
			CaseID:        currentResult.CaseID,
			BaselineScore: baselineResult.Score,
			CurrentScore:  currentResult.Score,
			Delta:         delta,
			Severity:      severity,
			Summary:       summary,
		})
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Delta == findings[j].Delta {
			return findings[i].CaseID < findings[j].CaseID
		}
		return findings[i].Delta < findings[j].Delta
	})
	return findings
}

func RenderMetricSpecMarkdown(spec OperationsMetricSpec) string {
	lines := []string{
		"# Operations Metric Spec",
		"",
		fmt.Sprintf("- Name: %s", spec.Name),
		fmt.Sprintf("- Generated At: %s", spec.GeneratedAt),
		fmt.Sprintf("- Period Start: %s", spec.PeriodStart),
		fmt.Sprintf("- Period End: %s", spec.PeriodEnd),
		fmt.Sprintf("- Timezone: %s", spec.TimezoneName),
		"",
		"## Definitions",
		"",
	}
	for _, definition := range spec.Definitions {
		lines = append(lines,
			fmt.Sprintf("### %s", definition.Label),
			"",
			fmt.Sprintf("- Metric ID: %s", definition.MetricID),
			fmt.Sprintf("- Unit: %s", definition.Unit),
			fmt.Sprintf("- Direction: %s", definition.Direction),
			fmt.Sprintf("- Formula: %s", definition.Formula),
			fmt.Sprintf("- Description: %s", definition.Description),
			fmt.Sprintf("- Source Fields: %s", strings.Join(definition.SourceFields, ", ")),
			"",
		)
	}
	lines = append(lines, "## Values", "")
	for _, value := range spec.Values {
		evidence := "none"
		if len(value.Evidence) > 0 {
			evidence = strings.Join(value.Evidence, " | ")
		}
		lines = append(lines,
			fmt.Sprintf("- %s: value=%s numerator=%.1f denominator=%.1f unit=%s evidence=%s",
				value.Label,
				value.DisplayValue,
				value.Numerator,
				value.Denominator,
				value.Unit,
				evidence,
			),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func metricCycleMinutes(run MetricRun) (float64, bool) {
	startedAt, ok := parseMetricTimestamp(run.StartedAt)
	if !ok {
		return 0, false
	}
	endedAt, ok := parseMetricTimestamp(run.EndedAt)
	if !ok || endedAt.Before(startedAt) {
		return 0, false
	}
	return roundTo(endedAt.Sub(startedAt).Minutes(), 1), true
}

func parseMetricTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func resolveMetricRunRiskScore(run MetricRun) (float64, bool) {
	if run.RiskScore != nil {
		return run.RiskScore.Total, true
	}
	switch strings.ToLower(strings.TrimSpace(string(run.RiskLevel))) {
	case "low":
		return 25, true
	case "medium":
		return 60, true
	case "high":
		return 90, true
	default:
		return 0, false
	}
}

func resolveMetricRunSpend(run MetricRun) float64 {
	for _, value := range []*float64{run.SpendUSD, run.CostUSD, run.Spend, run.Cost} {
		if value != nil {
			return *value
		}
	}
	return 0
}

func cloneMetricDefinitions(input []OperationsMetricDefinition) []OperationsMetricDefinition {
	out := make([]OperationsMetricDefinition, 0, len(input))
	for _, definition := range input {
		clone := definition
		clone.SourceFields = append([]string(nil), definition.SourceFields...)
		out = append(out, clone)
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func roundTo(value float64, places int) float64 {
	if places <= 0 {
		return math.Round(value)
	}
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	return math.Round(value*factor) / factor
}
