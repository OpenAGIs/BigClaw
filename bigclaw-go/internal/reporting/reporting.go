package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type Summary struct {
	TotalRuns          int   `json:"total_runs"`
	CompletedRuns      int   `json:"completed_runs"`
	BlockedRuns        int   `json:"blocked_runs"`
	HighRiskRuns       int   `json:"high_risk_runs"`
	RegressionFindings int   `json:"regression_findings"`
	HumanInterventions int   `json:"human_interventions"`
	BudgetCentsTotal   int64 `json:"budget_cents_total"`
	PremiumRuns        int   `json:"premium_runs"`
}

type TeamBreakdown struct {
	Key                string `json:"key"`
	TotalRuns          int    `json:"total_runs"`
	CompletedRuns      int    `json:"completed_runs"`
	BlockedRuns        int    `json:"blocked_runs"`
	BudgetCentsTotal   int64  `json:"budget_cents_total"`
	HumanInterventions int    `json:"human_interventions"`
}

type MetricDefinition struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Unit         string   `json:"unit"`
	Direction    string   `json:"direction"`
	Formula      string   `json:"formula"`
	Description  string   `json:"description"`
	SourceFields []string `json:"source_fields,omitempty"`
}

type MetricValue struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Value        float64  `json:"value"`
	DisplayValue string   `json:"display_value"`
	Numerator    float64  `json:"numerator"`
	Denominator  float64  `json:"denominator"`
	Unit         string   `json:"unit"`
	Evidence     []string `json:"evidence,omitempty"`
}

type MetricSpec struct {
	Name        string             `json:"name"`
	GeneratedAt string             `json:"generated_at"`
	Definitions []MetricDefinition `json:"definitions"`
	Values      []MetricValue      `json:"values"`
}

type WeeklyArtifacts struct {
	RootDir          string `json:"root_dir"`
	WeeklyReportPath string `json:"weekly_report_path"`
	MetricSpecPath   string `json:"metric_spec_path"`
}

type Weekly struct {
	WeekStart     time.Time       `json:"week_start"`
	WeekEnd       time.Time       `json:"week_end"`
	Summary       Summary         `json:"summary"`
	TeamBreakdown []TeamBreakdown `json:"team_breakdown"`
	MetricSpec    MetricSpec      `json:"metric_spec"`
	Highlights    []string        `json:"highlights"`
	Actions       []string        `json:"actions"`
	Markdown      string          `json:"markdown"`
}

func Build(tasks []domain.Task, events []domain.Event, weekStart, weekEnd time.Time) Weekly {
	weekly := Weekly{WeekStart: weekStart, WeekEnd: weekEnd}
	byTeam := make(map[string]*TeamBreakdown)
	interventions := interventionCounts(events)
	for _, task := range tasks {
		if !within(task.UpdatedAt, weekStart, weekEnd) {
			continue
		}
		weekly.Summary.TotalRuns++
		weekly.Summary.BudgetCentsTotal += task.BudgetCents
		if task.State == domain.TaskSucceeded {
			weekly.Summary.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			weekly.Summary.BlockedRuns++
		}
		if task.RiskLevel == domain.RiskHigh {
			weekly.Summary.HighRiskRuns++
		}
		if regressionCount(task) > 0 {
			weekly.Summary.RegressionFindings += regressionCount(task)
		}
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			weekly.Summary.PremiumRuns++
		}
		weekly.Summary.HumanInterventions += interventions[task.ID]
		team := firstNonEmpty(task.Metadata["team"], "unassigned")
		entry := byTeam[team]
		if entry == nil {
			entry = &TeamBreakdown{Key: team}
			byTeam[team] = entry
		}
		entry.TotalRuns++
		entry.BudgetCentsTotal += task.BudgetCents
		entry.HumanInterventions += interventions[task.ID]
		if task.State == domain.TaskSucceeded {
			entry.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			entry.BlockedRuns++
		}
	}
	for _, entry := range byTeam {
		weekly.TeamBreakdown = append(weekly.TeamBreakdown, *entry)
	}
	sort.SliceStable(weekly.TeamBreakdown, func(i, j int) bool {
		if weekly.TeamBreakdown[i].TotalRuns == weekly.TeamBreakdown[j].TotalRuns {
			return weekly.TeamBreakdown[i].Key < weekly.TeamBreakdown[j].Key
		}
		return weekly.TeamBreakdown[i].TotalRuns > weekly.TeamBreakdown[j].TotalRuns
	})
	weekly.Highlights = buildHighlights(weekly)
	weekly.Actions = buildActions(weekly)
	weekly.MetricSpec = buildMetricSpec(weekly)
	weekly.Markdown = RenderMarkdown(weekly)
	return weekly
}

func RenderMarkdown(weekly Weekly) string {
	builder := strings.Builder{}
	builder.WriteString("# BigClaw Weekly Ops Report\n\n")
	builder.WriteString(fmt.Sprintf("Window: %s -> %s\n\n", weekly.WeekStart.Format("2006-01-02"), weekly.WeekEnd.Format("2006-01-02")))
	builder.WriteString("## Summary\n")
	builder.WriteString(fmt.Sprintf("- Total runs: %d\n", weekly.Summary.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Completed runs: %d\n", weekly.Summary.CompletedRuns))
	builder.WriteString(fmt.Sprintf("- Blocked runs: %d\n", weekly.Summary.BlockedRuns))
	builder.WriteString(fmt.Sprintf("- High-risk runs: %d\n", weekly.Summary.HighRiskRuns))
	builder.WriteString(fmt.Sprintf("- Premium runs: %d\n", weekly.Summary.PremiumRuns))
	builder.WriteString(fmt.Sprintf("- Human interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Budget cents: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Highlights\n")
	if len(weekly.Highlights) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, highlight := range weekly.Highlights {
			builder.WriteString("- " + highlight + "\n")
		}
	}
	builder.WriteString("\n## Team Breakdown\n")
	if len(weekly.TeamBreakdown) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, team := range weekly.TeamBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: total=%d completed=%d blocked=%d interventions=%d budget_cents=%d\n",
				team.Key,
				team.TotalRuns,
				team.CompletedRuns,
				team.BlockedRuns,
				team.HumanInterventions,
				team.BudgetCentsTotal,
			))
		}
	}
	builder.WriteString("\n")
	builder.WriteString("## Metric Spec\n")
	for _, value := range weekly.MetricSpec.Values {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", value.Label, value.DisplayValue))
	}
	builder.WriteString("\n")
	builder.WriteString("## Actions\n")
	for _, action := range weekly.Actions {
		builder.WriteString("- " + action + "\n")
	}
	return builder.String()
}

func RenderMetricSpec(spec MetricSpec) string {
	builder := strings.Builder{}
	builder.WriteString("# BigClaw Weekly Metric Spec\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", spec.Name))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n\n", spec.GeneratedAt))
	builder.WriteString("## Definitions\n")
	for _, definition := range spec.Definitions {
		builder.WriteString(fmt.Sprintf("- %s: unit=%s direction=%s formula=%s\n", definition.Label, definition.Unit, definition.Direction, definition.Formula))
	}
	builder.WriteString("\n## Values\n")
	for _, value := range spec.Values {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", value.Label, value.DisplayValue))
	}
	return builder.String()
}

func WriteWeeklyBundle(rootDir string, weekly Weekly) (WeeklyArtifacts, error) {
	targetRoot := strings.TrimSpace(rootDir)
	if targetRoot == "" {
		targetRoot = "."
	}
	targetRoot, err := filepath.Abs(targetRoot)
	if err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return WeeklyArtifacts{}, err
	}
	weeklyReportPath := filepath.Join(targetRoot, "weekly-operations.md")
	metricSpecPath := filepath.Join(targetRoot, "weekly-metric-spec.md")
	if err := os.WriteFile(weeklyReportPath, []byte(weekly.Markdown), 0o644); err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := os.WriteFile(metricSpecPath, []byte(RenderMetricSpec(weekly.MetricSpec)), 0o644); err != nil {
		return WeeklyArtifacts{}, err
	}
	return WeeklyArtifacts{
		RootDir:          targetRoot,
		WeeklyReportPath: weeklyReportPath,
		MetricSpecPath:   metricSpecPath,
	}, nil
}

func buildHighlights(weekly Weekly) []string {
	highlights := []string{
		fmt.Sprintf("Completed %d / %d runs this week.", weekly.Summary.CompletedRuns, weekly.Summary.TotalRuns),
		fmt.Sprintf("Observed %d human interventions across active delivery lanes.", weekly.Summary.HumanInterventions),
	}
	if len(weekly.TeamBreakdown) > 0 {
		highlights = append(highlights, fmt.Sprintf("Top team by throughput: %s.", weekly.TeamBreakdown[0].Key))
	}
	return highlights
}

func buildActions(weekly Weekly) []string {
	actions := make([]string, 0)
	if weekly.Summary.BlockedRuns > 0 {
		actions = append(actions, "Reduce blocked flow count by resolving the top blocker owners first.")
	}
	if weekly.Summary.RegressionFindings > 0 {
		actions = append(actions, "Review regression hotspots and route them through the regression center.")
	}
	if weekly.Summary.HumanInterventions > 0 {
		actions = append(actions, "Audit repeated manual takeovers and convert them into policy or workflow fixes.")
	}
	if len(actions) == 0 {
		actions = append(actions, "No urgent actions detected; maintain current operating cadence.")
	}
	return actions
}

func buildMetricSpec(weekly Weekly) MetricSpec {
	totalRuns := float64(weekly.Summary.TotalRuns)
	completedRuns := float64(weekly.Summary.CompletedRuns)
	blockedRuns := float64(weekly.Summary.BlockedRuns)
	interventions := float64(weekly.Summary.HumanInterventions)
	highRiskRuns := float64(weekly.Summary.HighRiskRuns)
	premiumRuns := float64(weekly.Summary.PremiumRuns)
	regressions := float64(weekly.Summary.RegressionFindings)
	budgetTotal := float64(weekly.Summary.BudgetCentsTotal)
	return MetricSpec{
		Name:        "Weekly Operations Metric Spec",
		GeneratedAt: weekly.WeekEnd.UTC().Format(time.RFC3339),
		Definitions: []MetricDefinition{
			{MetricID: "throughput", Label: "Throughput", Unit: "%", Direction: "up", Formula: "100 * completed_runs / total_runs", Description: "Share of weekly runs that completed successfully.", SourceFields: []string{"summary.completed_runs", "summary.total_runs"}},
			{MetricID: "blocked-rate", Label: "Blocked Rate", Unit: "%", Direction: "down", Formula: "100 * blocked_runs / total_runs", Description: "Share of weekly runs ending blocked, failed, or dead-lettered.", SourceFields: []string{"summary.blocked_runs", "summary.total_runs"}},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Unit: "%", Direction: "down", Formula: "100 * human_interventions / total_runs", Description: "Operator intervention pressure relative to total weekly runs.", SourceFields: []string{"summary.human_interventions", "summary.total_runs"}},
			{MetricID: "high-risk-rate", Label: "High Risk Rate", Unit: "%", Direction: "down", Formula: "100 * high_risk_runs / total_runs", Description: "Share of weekly runs marked high risk.", SourceFields: []string{"summary.high_risk_runs", "summary.total_runs"}},
			{MetricID: "premium-rate", Label: "Premium Rate", Unit: "%", Direction: "down", Formula: "100 * premium_runs / total_runs", Description: "Share of weekly runs routed through the premium lane.", SourceFields: []string{"summary.premium_runs", "summary.total_runs"}},
			{MetricID: "regression-density", Label: "Regression Density", Unit: "cases/run", Direction: "down", Formula: "regression_findings / total_runs", Description: "Regression findings per weekly run.", SourceFields: []string{"summary.regression_findings", "summary.total_runs"}},
			{MetricID: "avg-budget", Label: "Average Budget", Unit: "cents", Direction: "down", Formula: "budget_cents_total / total_runs", Description: "Average planned budget per weekly run.", SourceFields: []string{"summary.budget_cents_total", "summary.total_runs"}},
		},
		Values: []MetricValue{
			ratioMetric("throughput", "Throughput", completedRuns, totalRuns, "%", "completed runs over total runs"),
			ratioMetric("blocked-rate", "Blocked Rate", blockedRuns, totalRuns, "%", "blocked runs over total runs"),
			ratioMetric("intervention-rate", "Intervention Rate", interventions, totalRuns, "%", "human interventions over total runs"),
			ratioMetric("high-risk-rate", "High Risk Rate", highRiskRuns, totalRuns, "%", "high-risk runs over total runs"),
			ratioMetric("premium-rate", "Premium Rate", premiumRuns, totalRuns, "%", "premium runs over total runs"),
			decimalMetric("regression-density", "Regression Density", regressions, totalRuns, "cases/run", "regression findings per run"),
			decimalMetric("avg-budget", "Average Budget", budgetTotal, totalRuns, "cents", "average budget cents per run"),
		},
	}
}

func ratioMetric(metricID string, label string, numerator float64, denominator float64, unit string, evidence string) MetricValue {
	value := 0.0
	if denominator > 0 {
		value = roundTenth((numerator / denominator) * 100)
	}
	return MetricValue{
		MetricID:     metricID,
		Label:        label,
		Value:        value,
		DisplayValue: fmt.Sprintf("%.1f%s", value, unit),
		Numerator:    numerator,
		Denominator:  denominator,
		Unit:         unit,
		Evidence:     []string{evidence},
	}
}

func decimalMetric(metricID string, label string, numerator float64, denominator float64, unit string, evidence string) MetricValue {
	value := 0.0
	if denominator > 0 {
		value = roundTenth(numerator / denominator)
	}
	return MetricValue{
		MetricID:     metricID,
		Label:        label,
		Value:        value,
		DisplayValue: fmt.Sprintf("%.1f %s", value, unit),
		Numerator:    numerator,
		Denominator:  denominator,
		Unit:         unit,
		Evidence:     []string{evidence},
	}
}

func roundTenth(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func interventionCounts(events []domain.Event) map[string]int {
	out := make(map[string]int)
	for _, event := range events {
		switch event.Type {
		case domain.EventRunTakeover, domain.EventRunReleased, domain.EventRunAnnotated, domain.EventControlPaused, domain.EventControlResumed:
			if event.TaskID != "" {
				out[event.TaskID]++
			}
		}
	}
	return out
}

func within(anchor time.Time, start time.Time, end time.Time) bool {
	if anchor.IsZero() {
		return false
	}
	if !start.IsZero() && anchor.Before(start) {
		return false
	}
	if !end.IsZero() && anchor.After(end) {
		return false
	}
	return true
}

func regressionCount(task domain.Task) int {
	for _, key := range []string{"regression_count", "regressions"} {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(task.Metadata["regression"]), "true") {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
