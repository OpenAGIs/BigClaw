package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type SharedViewFilter struct {
	Label string
	Value string
}

type SharedViewContext struct {
	Filters      []SharedViewFilter
	ResultCount  *int
	Loading      bool
	Errors       []string
	PartialData  []string
	EmptyMessage string
	LastUpdated  string
}

func (v SharedViewContext) State() string {
	if v.Loading {
		return "loading"
	}
	if len(v.Errors) > 0 && (v.ResultCount == nil || *v.ResultCount == 0) {
		return "error"
	}
	if v.ResultCount != nil && *v.ResultCount == 0 && len(v.PartialData) == 0 {
		return "empty"
	}
	if len(v.Errors) > 0 || len(v.PartialData) > 0 {
		return "partial-data"
	}
	return "ready"
}

func (v SharedViewContext) Summary() string {
	switch v.State() {
	case "loading":
		return "Loading data for the current filters."
	case "error":
		return "Unable to load data for the current filters."
	case "empty":
		if strings.TrimSpace(v.EmptyMessage) != "" {
			return v.EmptyMessage
		}
		return "No records match the current filters."
	case "partial-data":
		return "Showing partial data while one or more sources are unavailable."
	default:
		return "Data is current for the selected filters."
	}
}

func renderSharedViewContext(view *SharedViewContext) []string {
	if view == nil {
		return nil
	}
	lines := []string{
		"## View State",
		"",
		fmt.Sprintf("- State: %s", view.State()),
		fmt.Sprintf("- Summary: %s", view.Summary()),
	}
	if view.ResultCount != nil {
		lines = append(lines, fmt.Sprintf("- Result Count: %d", *view.ResultCount))
	}
	if strings.TrimSpace(view.LastUpdated) != "" {
		lines = append(lines, fmt.Sprintf("- Last Updated: %s", view.LastUpdated))
	}
	lines = append(lines, "", "## Filters", "")
	if len(view.Filters) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, filter := range view.Filters {
			lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
		}
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "", "## Errors", "")
		for _, message := range view.Errors {
			lines = append(lines, "- "+message)
		}
	}
	if len(view.PartialData) > 0 {
		lines = append(lines, "", "## Partial Data", "")
		for _, message := range view.PartialData {
			lines = append(lines, "- "+message)
		}
	}
	lines = append(lines, "")
	return lines
}

type RunAudit struct {
	Reason string
}

type RunCommitLinkLite struct {
	Role string
}

type RunCloseout struct {
	RunCommitLinks     []RunCommitLinkLite
	AcceptedCommitHash string
}

type RunRecord struct {
	RunID                string
	TaskID               string
	Status               string
	StartedAt            string
	EndedAt              string
	Summary              string
	Audits               []RunAudit
	RiskLevel            string
	RiskScoreTotal       *float64
	SpendUSD             *float64
	CostUSD              *float64
	Spend                *float64
	Cost                 *float64
	Closeout             RunCloseout
	RepoDiscussionPosts  int
	AcceptedLineageDepth *float64
}

type TriageCluster struct {
	Reason   string
	RunIDs   []string
	TaskIDs  []string
	Statuses []string
}

func (c TriageCluster) Occurrences() int {
	return len(c.RunIDs)
}

type RunOperationsSnapshot struct {
	TotalRuns           int
	StatusCounts        map[string]int
	SuccessRate         float64
	ApprovalQueueDepth  int
	SLATargetMinutes    int
	SLABreachCount      int
	AverageCycleMinutes float64
	TopBlockers         []TriageCluster
}

type BenchmarkCase struct {
	CaseID string
	Score  int
	Passed bool
}

type BenchmarkSuite struct {
	Version string
	Results []BenchmarkCase
}

type RegressionFindingLite struct {
	CaseID        string
	BaselineScore int
	CurrentScore  int
	Delta         int
	Severity      string
	Summary       string
}

type WeeklyRunReport struct {
	Name        string
	Period      string
	Snapshot    RunOperationsSnapshot
	Regressions []RegressionFindingLite
}

type RegressionCenterLite struct {
	Name            string
	BaselineVersion string
	CurrentVersion  string
	Regressions     []RegressionFindingLite
	ImprovedCases   []string
	UnchangedCases  []string
}

func (c RegressionCenterLite) RegressionCount() int {
	return len(c.Regressions)
}

func BuildRepoCollaborationMetrics(runs []RunRecord) map[string]float64 {
	total := len(runs)
	linked := 0
	accepted := 0
	discussionPosts := 0
	lineageDepthSum := 0.0
	lineageDepthCount := 0
	for _, run := range runs {
		if len(run.Closeout.RunCommitLinks) > 0 {
			linked++
		}
		if strings.TrimSpace(run.Closeout.AcceptedCommitHash) != "" {
			accepted++
		}
		discussionPosts += run.RepoDiscussionPosts
		if run.AcceptedLineageDepth != nil {
			lineageDepthSum += *run.AcceptedLineageDepth
			lineageDepthCount++
		}
	}
	out := map[string]float64{
		"repo_link_coverage":         0,
		"accepted_commit_rate":       0,
		"discussion_density":         0,
		"accepted_lineage_depth_avg": 0,
	}
	if total > 0 {
		out["repo_link_coverage"] = roundTenth(float64(linked) / float64(total) * 100)
		out["accepted_commit_rate"] = roundTenth(float64(accepted) / float64(total) * 100)
		out["discussion_density"] = float64(int((float64(discussionPosts)/float64(total))*100)) / 100
	}
	if lineageDepthCount > 0 {
		out["accepted_lineage_depth_avg"] = float64(int((lineageDepthSum/float64(lineageDepthCount))*100)) / 100
	}
	return out
}

func NormalizeDashboardLayout(layout DashboardLayout, widgets []DashboardWidgetSpec) DashboardLayout {
	widgetIndex := make(map[string]DashboardWidgetSpec, len(widgets))
	for _, widget := range widgets {
		widgetIndex[widget.WidgetID] = widget
	}
	columnCount := layout.Columns
	if columnCount < 1 {
		columnCount = 12
	}
	normalized := make([]DashboardWidgetPlacement, 0, len(layout.Placements))
	for _, placement := range layout.Placements {
		spec, ok := widgetIndex[placement.WidgetID]
		minWidth := 1
		maxWidth := columnCount
		if ok {
			minWidth = spec.MinWidth
			if minWidth <= 0 {
				minWidth = 2
			}
			maxWidth = spec.MaxWidth
			if maxWidth <= 0 {
				maxWidth = 12
			}
			if maxWidth > columnCount {
				maxWidth = columnCount
			}
		}
		width := placement.Width
		if width < minWidth {
			width = minWidth
		}
		if width > maxWidth {
			width = maxWidth
		}
		column := placement.Column
		if column < 0 {
			column = 0
		}
		if column+width > columnCount {
			column = max(0, columnCount-width)
		}
		row := placement.Row
		if row < 0 {
			row = 0
		}
		height := placement.Height
		if height < 1 {
			height = 1
		}
		normalized = append(normalized, DashboardWidgetPlacement{
			PlacementID:   placement.PlacementID,
			WidgetID:      placement.WidgetID,
			Column:        column,
			Row:           row,
			Width:         width,
			Height:        height,
			TitleOverride: placement.TitleOverride,
			Filters:       append([]string(nil), placement.Filters...),
		})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Row == normalized[j].Row {
			if normalized[i].Column == normalized[j].Column {
				return normalized[i].PlacementID < normalized[j].PlacementID
			}
			return normalized[i].Column < normalized[j].Column
		}
		return normalized[i].Row < normalized[j].Row
	})
	return DashboardLayout{
		LayoutID:   layout.LayoutID,
		Name:       layout.Name,
		Columns:    columnCount,
		Placements: normalized,
	}
}

func BuildDashboardBuilder(name, period, owner, viewerRole string, widgets []DashboardWidgetSpec, layouts []DashboardLayout, documentationComplete bool) DashboardBuilder {
	normalized := make([]DashboardLayout, 0, len(layouts))
	for _, layout := range layouts {
		normalized = append(normalized, NormalizeDashboardLayout(layout, widgets))
	}
	return DashboardBuilder{
		Name:                  name,
		Period:                period,
		Owner:                 owner,
		Permissions:           permissionsForRole(viewerRole),
		Widgets:               append([]DashboardWidgetSpec(nil), widgets...),
		Layouts:               normalized,
		DocumentationComplete: documentationComplete,
	}
}

func BuildTriageClusters(runs []RunRecord) []TriageCluster {
	clusters := map[string]*TriageCluster{}
	for _, run := range runs {
		if run.Status != "needs-approval" && run.Status != "failed" && run.Status != "rejected" {
			continue
		}
		reason := primaryRunReason(run)
		cluster := clusters[reason]
		if cluster == nil {
			cluster = &TriageCluster{Reason: reason}
			clusters[reason] = cluster
		}
		if run.RunID != "" && !contains(cluster.RunIDs, run.RunID) {
			cluster.RunIDs = append(cluster.RunIDs, run.RunID)
		}
		if run.TaskID != "" && !contains(cluster.TaskIDs, run.TaskID) {
			cluster.TaskIDs = append(cluster.TaskIDs, run.TaskID)
		}
		if run.Status != "" && !contains(cluster.Statuses, run.Status) {
			cluster.Statuses = append(cluster.Statuses, run.Status)
		}
	}
	out := make([]TriageCluster, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, *cluster)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Occurrences() == out[j].Occurrences() {
			return out[i].Reason < out[j].Reason
		}
		return out[i].Occurrences() > out[j].Occurrences()
	})
	return out
}

func BuildRunOperationsSnapshot(runs []RunRecord, slaTargetMinutes int) RunOperationsSnapshot {
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	statusCounts := map[string]int{}
	totalCycle := 0.0
	cycleCount := 0
	completed := 0
	approvalQueueDepth := 0
	slaBreaches := 0
	for _, run := range runs {
		statusCounts[run.Status]++
		if run.Status == "needs-approval" {
			approvalQueueDepth++
		}
		if run.Status == "approved" || run.Status == "accepted" || run.Status == "completed" || run.Status == "succeeded" {
			completed++
		}
		if cycle, ok := runCycleMinutes(run); ok {
			totalCycle += cycle
			cycleCount++
			if cycle > float64(slaTargetMinutes) {
				slaBreaches++
			}
		}
	}
	successRate := 0.0
	if len(runs) > 0 {
		successRate = roundTenth(float64(completed) / float64(len(runs)) * 100)
	}
	averageCycle := 0.0
	if cycleCount > 0 {
		averageCycle = roundTenth(totalCycle / float64(cycleCount))
	}
	return RunOperationsSnapshot{
		TotalRuns:           len(runs),
		StatusCounts:        statusCounts,
		SuccessRate:         successRate,
		ApprovalQueueDepth:  approvalQueueDepth,
		SLATargetMinutes:    slaTargetMinutes,
		SLABreachCount:      slaBreaches,
		AverageCycleMinutes: averageCycle,
		TopBlockers:         BuildTriageClusters(runs),
	}
}

func BuildRunOperationsMetricSpec(runs []RunRecord, periodStart, periodEnd, timezoneName, generatedAt string, slaTargetMinutes int, currentSuite, baselineSuite *BenchmarkSuite) OperationsMetricSpec {
	if timezoneName == "" {
		timezoneName = "UTC"
	}
	runsToday := 0
	leadTimeSum := 0.0
	leadTimeCount := 0
	actionableRuns := 0
	slaCompliantRuns := 0
	riskSum := 0.0
	riskCount := 0
	spendTotal := 0.0
	start := parseRFC3339ish(periodStart)
	end := parseRFC3339ish(periodEnd)
	for _, run := range runs {
		startedAt := parseRFC3339ish(run.StartedAt)
		if !start.IsZero() && !end.IsZero() && !startedAt.IsZero() && (startedAt.Equal(start) || startedAt.After(start)) && (startedAt.Equal(end) || startedAt.Before(end)) {
			runsToday++
		}
		if cycle, ok := runCycleMinutes(run); ok {
			leadTimeSum += cycle
			leadTimeCount++
			if cycle <= float64(slaTargetMinutes) {
				slaCompliantRuns++
			}
		}
		if run.Status == "needs-approval" || run.Status == "failed" || run.Status == "rejected" {
			actionableRuns++
		}
		if risk, ok := resolveRunRiskScore(run); ok {
			riskSum += risk
			riskCount++
		}
		spendTotal += resolveRunSpend(run)
	}
	regressions := AnalyzeBenchmarkRegressions(currentSuite, baselineSuite)
	totalRuns := len(runs)
	avgLead := 0.0
	if leadTimeCount > 0 {
		avgLead = roundTenth(leadTimeSum / float64(leadTimeCount))
	}
	interventionRate := 0.0
	if totalRuns > 0 {
		interventionRate = roundTenth(float64(actionableRuns) / float64(totalRuns) * 100)
	}
	slaValue := 0.0
	if leadTimeCount > 0 {
		slaValue = roundTenth(float64(slaCompliantRuns) / float64(leadTimeCount) * 100)
	}
	avgRisk := 0.0
	if riskCount > 0 {
		avgRisk = roundTenth(riskSum / float64(riskCount))
	}
	spendTotal = float64(int(spendTotal*100+0.5)) / 100
	return OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  parseRFC3339ish(generatedAt),
		PeriodStart:  parseRFC3339ish(periodStart),
		PeriodEnd:    parseRFC3339ish(periodEnd),
		TimezoneName: timezoneName,
		Definitions: []OperationsMetricDefinition{
			{MetricID: "runs-today", Label: "Runs Today", Unit: "runs", Direction: "up", Formula: "count(run.started_at within [period_start, period_end])", Description: "Number of runs that started inside the reporting day window.", SourceFields: []string{"started_at"}},
			{MetricID: "avg-lead-time", Label: "Avg Lead Time", Unit: "m", Direction: "down", Formula: "sum(cycle_minutes for runs with started_at and ended_at) / measured_runs", Description: "Average elapsed minutes from run start to run end for runs with complete timestamps.", SourceFields: []string{"started_at", "ended_at"}},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Unit: "%", Direction: "down", Formula: "100 * actionable_runs / total_runs", Description: "Share of runs that require operator intervention because they ended in an actionable status.", SourceFields: []string{"status"}},
			{MetricID: "sla", Label: "SLA", Unit: "%", Direction: "up", Formula: "100 * compliant_runs / measured_runs", Description: "Share of measured runs that met the SLA target.", SourceFields: []string{"started_at", "ended_at"}},
			{MetricID: "regression", Label: "Regression", Unit: "cases", Direction: "down", Formula: "count(score regressions and pass->fail transitions)", Description: "Number of benchmark cases that regressed against the provided baseline suite.", SourceFields: []string{"benchmark.current", "benchmark.baseline"}},
			{MetricID: "risk", Label: "Risk", Unit: "score", Direction: "down", Formula: "sum(resolved_run_risk_score) / runs_with_risk", Description: "Average per-run risk score from explicit risk scores or normalized risk levels.", SourceFields: []string{"risk_score.total", "risk_level"}},
			{MetricID: "spend", Label: "Spend", Unit: "USD", Direction: "down", Formula: "sum(first non-null of spend_usd, cost_usd, spend, cost)", Description: "Total reported run spend in USD over the reporting window.", SourceFields: []string{"spend_usd", "cost_usd", "spend", "cost"}},
		},
		Values: []OperationsMetricValue{
			{MetricID: "runs-today", Label: "Runs Today", Value: float64(runsToday), DisplayValue: strconv.Itoa(runsToday), Numerator: float64(runsToday), Denominator: float64(totalRuns), Unit: "runs"},
			{MetricID: "avg-lead-time", Label: "Avg Lead Time", Value: avgLead, DisplayValue: fmt.Sprintf("%.1fm", avgLead), Numerator: roundTenth(leadTimeSum), Denominator: float64(leadTimeCount), Unit: "m"},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Value: interventionRate, DisplayValue: fmt.Sprintf("%.1f%%", interventionRate), Numerator: float64(actionableRuns), Denominator: float64(totalRuns), Unit: "%"},
			{MetricID: "sla", Label: "SLA", Value: slaValue, DisplayValue: fmt.Sprintf("%.1f%%", slaValue), Numerator: float64(slaCompliantRuns), Denominator: float64(leadTimeCount), Unit: "%"},
			{MetricID: "regression", Label: "Regression", Value: float64(len(regressions)), DisplayValue: strconv.Itoa(len(regressions)), Numerator: float64(len(regressions)), Denominator: float64(totalRuns), Unit: "cases"},
			{MetricID: "risk", Label: "Risk", Value: avgRisk, DisplayValue: fmt.Sprintf("%.1f", avgRisk), Numerator: roundTenth(riskSum), Denominator: float64(riskCount), Unit: "score"},
			{MetricID: "spend", Label: "Spend", Value: spendTotal, DisplayValue: fmt.Sprintf("$%.2f", spendTotal), Numerator: spendTotal, Denominator: float64(totalRuns), Unit: "USD"},
		},
	}
}

func AnalyzeBenchmarkRegressions(currentSuite, baselineSuite *BenchmarkSuite) []RegressionFindingLite {
	if currentSuite == nil || baselineSuite == nil {
		return nil
	}
	baselineByID := make(map[string]BenchmarkCase, len(baselineSuite.Results))
	for _, result := range baselineSuite.Results {
		baselineByID[result.CaseID] = result
	}
	findings := make([]RegressionFindingLite, 0)
	for _, current := range currentSuite.Results {
		baseline, ok := baselineByID[current.CaseID]
		if !ok {
			continue
		}
		delta := current.Score - baseline.Score
		passToFail := baseline.Passed && !current.Passed
		if delta >= 0 && !passToFail {
			continue
		}
		severity := "medium"
		summary := "case regressed from passing to failing"
		if delta < 0 {
			summary = fmt.Sprintf("score dropped from %d to %d", baseline.Score, current.Score)
		}
		if delta <= -20 || passToFail {
			severity = "high"
		}
		findings = append(findings, RegressionFindingLite{
			CaseID:        current.CaseID,
			BaselineScore: baseline.Score,
			CurrentScore:  current.Score,
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

func BuildRegressionCenterLite(name string, currentSuite, baselineSuite *BenchmarkSuite) RegressionCenterLite {
	center := RegressionCenterLite{Name: name}
	if currentSuite != nil {
		center.CurrentVersion = currentSuite.Version
	}
	if baselineSuite != nil {
		center.BaselineVersion = baselineSuite.Version
	}
	if currentSuite == nil || baselineSuite == nil {
		return center
	}
	center.Regressions = AnalyzeBenchmarkRegressions(currentSuite, baselineSuite)
	baselineByID := make(map[string]BenchmarkCase, len(baselineSuite.Results))
	for _, result := range baselineSuite.Results {
		baselineByID[result.CaseID] = result
	}
	for _, current := range currentSuite.Results {
		baseline, ok := baselineByID[current.CaseID]
		if !ok {
			continue
		}
		if current.Score > baseline.Score || (!baseline.Passed && current.Passed) {
			center.ImprovedCases = append(center.ImprovedCases, current.CaseID)
			continue
		}
		if current.Score == baseline.Score && current.Passed == baseline.Passed {
			center.UnchangedCases = append(center.UnchangedCases, current.CaseID)
		}
	}
	sort.Strings(center.ImprovedCases)
	sort.Strings(center.UnchangedCases)
	return center
}

func BuildWeeklyRunReport(name, period string, runs []RunRecord, currentSuite, baselineSuite *BenchmarkSuite, slaTargetMinutes int) WeeklyRunReport {
	report := WeeklyRunReport{
		Name:        name,
		Period:      period,
		Snapshot:    BuildRunOperationsSnapshot(runs, slaTargetMinutes),
		Regressions: AnalyzeBenchmarkRegressions(currentSuite, baselineSuite),
	}
	return report
}

func BuildEngineeringOverviewFromRuns(name, period, viewerRole string, runs []RunRecord, slaTargetMinutes int) EngineeringOverview {
	snapshot := BuildRunOperationsSnapshot(runs, slaTargetMinutes)
	statusCounts := map[string]int{}
	for status, count := range snapshot.StatusCounts {
		statusCounts[status] = count
	}
	funnel := []EngineeringFunnelStage{
		{Name: "queued", Count: statusCounts["queued"]},
		{Name: "in-progress", Count: statusCounts["running"] + statusCounts["in-progress"]},
		{Name: "awaiting-approval", Count: statusCounts["needs-approval"]},
		{Name: "completed", Count: statusCounts["approved"] + statusCounts["accepted"] + statusCounts["completed"] + statusCounts["succeeded"]},
	}
	for i := range funnel {
		if snapshot.TotalRuns > 0 {
			funnel[i].Share = roundTenth(float64(funnel[i].Count) / float64(snapshot.TotalRuns) * 100)
		}
	}
	blockers := make([]EngineeringOverviewBlocker, 0, len(snapshot.TopBlockers))
	for _, cluster := range snapshot.TopBlockers {
		severity := "medium"
		for _, status := range cluster.Statuses {
			if status == "failed" {
				severity = "high"
				break
			}
		}
		owner := "engineering"
		details := strings.ToLower(strings.Join(append([]string{cluster.Reason}, cluster.Statuses...), " "))
		if strings.Contains(details, "approval") {
			owner = "operations"
		} else if strings.Contains(details, "security") {
			owner = "security"
		}
		blockers = append(blockers, EngineeringOverviewBlocker{
			Summary:       cluster.Reason,
			AffectedRuns:  cluster.Occurrences(),
			AffectedTasks: append([]string(nil), cluster.TaskIDs...),
			Owner:         owner,
			Severity:      severity,
		})
	}
	activities := buildRecentRunActivities(runs, 5)
	return EngineeringOverview{
		Name:        name,
		Period:      period,
		Permissions: permissionsForRole(viewerRole),
		KPIs: []EngineeringOverviewKPI{
			{Name: "success-rate", Value: snapshot.SuccessRate, Target: 90.0, Unit: "%", Direction: "up"},
			{Name: "approval-queue-depth", Value: float64(snapshot.ApprovalQueueDepth), Target: 2.0, Direction: "down"},
			{Name: "sla-breaches", Value: float64(snapshot.SLABreachCount), Target: 0.0, Direction: "down"},
			{Name: "average-cycle-minutes", Value: snapshot.AverageCycleMinutes, Target: float64(snapshot.SLATargetMinutes), Unit: "m", Direction: "down"},
		},
		Funnel:     funnel,
		Blockers:   blockers,
		Activities: activities,
	}
}

func RenderOperationsDashboardWithView(snapshot RunOperationsSnapshot, view *SharedViewContext) string {
	lines := []string{
		"# Operations Dashboard",
		"",
		fmt.Sprintf("- Total Runs: %d", snapshot.TotalRuns),
		fmt.Sprintf("- Success Rate: %.1f%%", snapshot.SuccessRate),
		fmt.Sprintf("- Approval Queue Depth: %d", snapshot.ApprovalQueueDepth),
		fmt.Sprintf("- SLA Target: %d minutes", snapshot.SLATargetMinutes),
		fmt.Sprintf("- SLA Breaches: %d", snapshot.SLABreachCount),
		fmt.Sprintf("- Average Cycle Time: %.1f minutes", snapshot.AverageCycleMinutes),
		"",
		"## Status Counts",
		"",
	}
	lines = append(lines, renderSharedViewContext(view)...)
	if len(snapshot.StatusCounts) == 0 {
		lines = append(lines, "- None")
	} else {
		keys := make([]string, 0, len(snapshot.StatusCounts))
		for key := range snapshot.StatusCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %d", key, snapshot.StatusCounts[key]))
		}
	}
	lines = append(lines, "", "## Top Blockers", "")
	if len(snapshot.TopBlockers) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, cluster := range snapshot.TopBlockers {
			lines = append(lines, fmt.Sprintf("- %s: occurrences=%d statuses=%s tasks=%s", cluster.Reason, cluster.Occurrences(), strings.Join(cluster.Statuses, ", "), strings.Join(cluster.TaskIDs, ", ")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderWeeklyRunReport(report WeeklyRunReport) string {
	lines := []string{
		"# Weekly Operations Report",
		"",
		fmt.Sprintf("- Name: %s", report.Name),
		fmt.Sprintf("- Period: %s", report.Period),
		fmt.Sprintf("- Total Runs: %d", report.Snapshot.TotalRuns),
		fmt.Sprintf("- Success Rate: %.1f%%", report.Snapshot.SuccessRate),
		fmt.Sprintf("- SLA Breaches: %d", report.Snapshot.SLABreachCount),
		fmt.Sprintf("- Approval Queue Depth: %d", report.Snapshot.ApprovalQueueDepth),
		"",
		"## Blockers",
		"",
	}
	if len(report.Snapshot.TopBlockers) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, cluster := range report.Snapshot.TopBlockers {
			lines = append(lines, fmt.Sprintf("- %s: %d runs", cluster.Reason, cluster.Occurrences()))
		}
	}
	lines = append(lines, "", "## Regressions", "")
	if len(report.Regressions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, finding := range report.Regressions {
			lines = append(lines, fmt.Sprintf("- %s: severity=%s delta=%d summary=%s", finding.CaseID, finding.Severity, finding.Delta, finding.Summary))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderRegressionCenterLite(center RegressionCenterLite, view *SharedViewContext) string {
	lines := []string{
		"# Regression Analysis Center",
		"",
		fmt.Sprintf("- Name: %s", center.Name),
		fmt.Sprintf("- Baseline Version: %s", center.BaselineVersion),
		fmt.Sprintf("- Current Version: %s", center.CurrentVersion),
		fmt.Sprintf("- Regressions: %d", center.RegressionCount()),
		fmt.Sprintf("- Improved Cases: %d", len(center.ImprovedCases)),
		fmt.Sprintf("- Unchanged Cases: %d", len(center.UnchangedCases)),
		"",
		"## Regressions",
		"",
	}
	lines = append(lines, renderSharedViewContext(view)...)
	if len(center.Regressions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, finding := range center.Regressions {
			lines = append(lines, fmt.Sprintf("- %s: severity=%s delta=%d summary=%s", finding.CaseID, finding.Severity, finding.Delta, finding.Summary))
		}
	}
	lines = append(lines, "", "## Improved Cases", "")
	if len(center.ImprovedCases) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, caseID := range center.ImprovedCases {
			lines = append(lines, "- "+caseID)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderPolicyPromptVersionCenterWithView(center PolicyPromptVersionCenter, view *SharedViewContext) string {
	lines := []string{
		"# Policy/Prompt Version Center",
		"",
		fmt.Sprintf("- Name: %s", center.Name),
		fmt.Sprintf("- Generated At: %s", center.GeneratedAt),
		fmt.Sprintf("- Versioned Artifacts: %d", center.ArtifactCount()),
		fmt.Sprintf("- Rollback Ready Artifacts: %d", center.RollbackReadyCount()),
		"",
		"## Artifact Histories",
		"",
	}
	lines = append(lines, renderSharedViewContext(view)...)
	if len(center.Histories) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, history := range center.Histories {
		lines = append(lines,
			fmt.Sprintf("### %s / %s", history.ArtifactType, history.ArtifactID),
			"",
			fmt.Sprintf("- Current Version: %s", history.CurrentVersion),
			fmt.Sprintf("- Updated At: %s", history.CurrentUpdatedAt),
			fmt.Sprintf("- Updated By: %s", history.CurrentAuthor),
			fmt.Sprintf("- Summary: %s", history.CurrentSummary),
			fmt.Sprintf("- Revision Count: %d", history.RevisionCount),
			fmt.Sprintf("- Rollback Version: %s", firstNonEmpty(history.RollbackVersion, "none")),
			fmt.Sprintf("- Rollback Ready: %t", history.RollbackReady),
		)
		if history.ChangeSummary != nil {
			lines = append(lines, fmt.Sprintf("- Diff Summary: %d additions, %d deletions", history.ChangeSummary.Additions, history.ChangeSummary.Deletions))
		}
		lines = append(lines, "", "#### Revision History", "")
		for _, revision := range history.Revisions {
			lines = append(lines, fmt.Sprintf("- %s: updated_at=%s author=%s ticket=%s summary=%s", revision.Version, revision.UpdatedAt, revision.Author, firstNonEmpty(revision.ChangeTicket, "none"), revision.Summary))
		}
		lines = append(lines, "", "#### Diff Preview", "")
		if history.ChangeSummary != nil && len(history.ChangeSummary.Preview) > 0 {
			lines = append(lines, "```diff")
			lines = append(lines, history.ChangeSummary.Preview...)
			lines = append(lines, "```")
		} else {
			lines = append(lines, "- None")
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderDashboardBuilderReportWithView(dashboard DashboardBuilder, audit DashboardBuilderAudit, view *SharedViewContext) string {
	lines := strings.Split(strings.TrimSuffix(RenderDashboardBuilderReport(dashboard, audit), "\n"), "\n")
	insertIndex := len(lines)
	for index, line := range lines {
		if line == "## Layouts" {
			insertIndex = index + 2
			break
		}
	}
	viewLines := renderSharedViewContext(view)
	if len(viewLines) > 0 {
		lines = append(lines[:insertIndex], append(viewLines, lines[insertIndex:]...)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func WriteDashboardBuilderBundleWithView(rootDir string, dashboard DashboardBuilder, audit DashboardBuilderAudit, view *SharedViewContext) (string, error) {
	if err := WriteReport(filepath.Join(rootDir, "dashboard-builder.md"), RenderDashboardBuilderReportWithView(dashboard, audit, view)); err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "dashboard-builder.md"), nil
}

func WriteWeeklyRunOperationsBundle(rootDir string, report WeeklyRunReport, metricSpec *OperationsMetricSpec, regressionCenter *RegressionCenterLite, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return WeeklyArtifacts{}, err
	}
	artifacts := WeeklyArtifacts{
		RootDir:          rootDir,
		WeeklyReportPath: filepath.Join(rootDir, "weekly-operations.md"),
		DashboardPath:    filepath.Join(rootDir, "operations-dashboard.md"),
	}
	if err := WriteReport(artifacts.WeeklyReportPath, RenderWeeklyRunReport(report)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := WriteReport(artifacts.DashboardPath, RenderOperationsDashboardWithView(report.Snapshot, nil)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if metricSpec != nil {
		artifacts.MetricSpecPath = filepath.Join(rootDir, "operations-metric-spec.md")
		if err := WriteReport(artifacts.MetricSpecPath, RenderOperationsMetricSpec(*metricSpec)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if regressionCenter != nil {
		artifacts.RegressionCenterPath = filepath.Join(rootDir, "regression-center.md")
		if err := WriteReport(artifacts.RegressionCenterPath, RenderRegressionCenterLite(*regressionCenter, nil)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if versionCenter != nil {
		artifacts.VersionCenterPath = filepath.Join(rootDir, "policy-prompt-version-center.md")
		if err := WriteReport(artifacts.VersionCenterPath, RenderPolicyPromptVersionCenterWithView(*versionCenter, nil)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	return artifacts, nil
}

func buildRecentRunActivities(runs []RunRecord, limit int) []EngineeringActivity {
	type datedRun struct {
		sortKey string
		run     RunRecord
	}
	dated := make([]datedRun, 0, len(runs))
	for _, run := range runs {
		sortKey := firstNonEmpty(run.EndedAt, run.StartedAt)
		if sortKey == "" {
			continue
		}
		dated = append(dated, datedRun{sortKey: sortKey, run: run})
	}
	sort.SliceStable(dated, func(i, j int) bool {
		left := parseRFC3339ish(dated[i].sortKey)
		right := parseRFC3339ish(dated[j].sortKey)
		if left.Equal(right) {
			return dated[i].run.RunID < dated[j].run.RunID
		}
		return left.After(right)
	})
	if limit > 0 && len(dated) > limit {
		dated = dated[:limit]
	}
	out := make([]EngineeringActivity, 0, len(dated))
	for _, item := range dated {
		out = append(out, EngineeringActivity{
			Timestamp: firstNonEmpty(item.run.EndedAt, item.run.StartedAt),
			RunID:     item.run.RunID,
			TaskID:    item.run.TaskID,
			Status:    item.run.Status,
			Summary:   primaryRunReason(item.run),
		})
	}
	return out
}

func primaryRunReason(run RunRecord) string {
	for _, audit := range run.Audits {
		if strings.TrimSpace(audit.Reason) != "" {
			return strings.TrimSpace(audit.Reason)
		}
	}
	if strings.TrimSpace(run.Summary) != "" {
		return strings.TrimSpace(run.Summary)
	}
	return firstNonEmpty(run.Status, "unknown")
}

func runCycleMinutes(run RunRecord) (float64, bool) {
	start := parseRFC3339ish(run.StartedAt)
	end := parseRFC3339ish(run.EndedAt)
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0, false
	}
	return roundTenth(end.Sub(start).Minutes()), true
}

func resolveRunRiskScore(run RunRecord) (float64, bool) {
	if run.RiskScoreTotal != nil {
		return *run.RiskScoreTotal, true
	}
	switch strings.ToLower(strings.TrimSpace(run.RiskLevel)) {
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

func resolveRunSpend(run RunRecord) float64 {
	for _, value := range []*float64{run.SpendUSD, run.CostUSD, run.Spend, run.Cost} {
		if value != nil {
			return *value
		}
	}
	return 0
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
