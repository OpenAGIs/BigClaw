package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/regression"
)

type RunCommitRoleLink struct {
	Role string `json:"role"`
}

type RunCloseoutRecord struct {
	RunCommitLinks     []RunCommitRoleLink `json:"run_commit_links,omitempty"`
	AcceptedCommitHash string              `json:"accepted_commit_hash,omitempty"`
}

type RunRecord struct {
	RunID                string           `json:"run_id"`
	TaskID               string           `json:"task_id"`
	Status               string           `json:"status"`
	StartedAt            string           `json:"started_at"`
	EndedAt              string           `json:"ended_at"`
	Summary              string           `json:"summary"`
	Reason               string           `json:"reason"`
	RiskLevel            string           `json:"risk_level,omitempty"`
	RiskScore            float64          `json:"risk_score,omitempty"`
	SpendUSD             float64          `json:"spend_usd,omitempty"`
	RepoDiscussionPosts  int              `json:"repo_discussion_posts,omitempty"`
	AcceptedLineageDepth int              `json:"accepted_lineage_depth,omitempty"`
	Closeout             RunCloseoutRecord `json:"closeout,omitempty"`
}

type OperationsSnapshot struct {
	TotalRuns            int            `json:"total_runs"`
	StatusCounts         map[string]int `json:"status_counts,omitempty"`
	SuccessRate          float64        `json:"success_rate"`
	ApprovalQueueDepth   int            `json:"approval_queue_depth"`
	SLABreachCount       int            `json:"sla_breach_count"`
	AverageCycleMinutes  float64        `json:"average_cycle_minutes"`
}

type TriageCluster struct {
	Reason      string   `json:"reason"`
	Occurrences int      `json:"occurrences"`
	TaskIDs     []string `json:"task_ids,omitempty"`
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

type RegressionDelta struct {
	CaseID   string `json:"case_id"`
	Delta    int    `json:"delta"`
	Severity string `json:"severity"`
}

type RegressionOverview struct {
	RegressionCount int               `json:"regression_count"`
	Regressions     []RegressionDelta `json:"regressions,omitempty"`
	ImprovedCases   []string          `json:"improved_cases,omitempty"`
	UnchangedCases  []string          `json:"unchanged_cases,omitempty"`
}

type WeeklyOperationsReportCompat struct {
	Name       string             `json:"name"`
	Period     string             `json:"period"`
	Snapshot   OperationsSnapshot `json:"snapshot"`
	Clusters   []TriageCluster    `json:"clusters,omitempty"`
	Regressions RegressionOverview `json:"regressions"`
}

func SummarizeRunRecords(runs []RunRecord, slaTargetMinutes int) OperationsSnapshot {
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	statusCounts := make(map[string]int)
	approvedCount := 0
	approvalQueueDepth := 0
	slaBreaches := 0
	totalCycle := 0.0
	measuredRuns := 0

	for _, run := range runs {
		status := strings.TrimSpace(run.Status)
		statusCounts[status]++
		if status == "approved" {
			approvedCount++
		}
		if status == "needs-approval" {
			approvalQueueDepth++
		}
		if cycle := runCycleMinutes(run); cycle >= 0 {
			totalCycle += cycle
			measuredRuns++
			if cycle > float64(slaTargetMinutes) {
				slaBreaches++
			}
		}
	}

	successRate := 0.0
	if len(runs) > 0 {
		successRate = roundTenth((float64(approvedCount) / float64(len(runs))) * 100)
	}
	avgCycle := 0.0
	if measuredRuns > 0 {
		avgCycle = roundTenth(totalCycle / float64(measuredRuns))
	}

	return OperationsSnapshot{
		TotalRuns:           len(runs),
		StatusCounts:        statusCounts,
		SuccessRate:         successRate,
		ApprovalQueueDepth:  approvalQueueDepth,
		SLABreachCount:      slaBreaches,
		AverageCycleMinutes: avgCycle,
	}
}

func BuildOperationsMetricSpecFromRuns(runs []RunRecord, periodStart, periodEnd time.Time, generatedAt time.Time, timezoneName string, slaTargetMinutes int, currentSuite, baselineSuite *BenchmarkSuite) OperationsMetricSpec {
	if timezoneName == "" {
		timezoneName = "UTC"
	}
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	runsToday := 0
	totalCycle := 0.0
	measuredRuns := 0
	intervenedRuns := 0
	slaCompliantRuns := 0
	riskSum := 0.0
	riskCount := 0
	spendSum := 0.0

	for _, run := range runs {
		started := parseRFC3339ish(run.StartedAt)
		ended := parseRFC3339ish(run.EndedAt)
		if within(started, periodStart, periodEnd) || within(ended, periodStart, periodEnd) {
			runsToday++
		}
		if cycle := runCycleMinutes(run); cycle >= 0 {
			totalCycle += cycle
			measuredRuns++
			if cycle <= float64(slaTargetMinutes) {
				slaCompliantRuns++
			}
		}
		if strings.EqualFold(strings.TrimSpace(run.Status), "needs-approval") {
			intervenedRuns++
		}
		if score, ok := runRiskScore(run); ok {
			riskSum += score
			riskCount++
		}
		spendSum += run.SpendUSD
	}

	avgLeadTime := 0.0
	if measuredRuns > 0 {
		avgLeadTime = roundTenth(totalCycle / float64(measuredRuns))
	}
	interventionRate := 0.0
	if len(runs) > 0 {
		interventionRate = roundTenth((float64(intervenedRuns) / float64(len(runs))) * 100)
	}
	slaRate := 0.0
	if measuredRuns > 0 {
		slaRate = roundTenth((float64(slaCompliantRuns) / float64(measuredRuns)) * 100)
	}
	regressionCount := len(AnalyzeRegressionDeltas(derefSuite(currentSuite), derefSuite(baselineSuite)))
	avgRisk := 0.0
	if riskCount > 0 {
		avgRisk = roundTenth(riskSum / float64(riskCount))
	}

	return OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  generatedAt.UTC(),
		PeriodStart:  periodStart.UTC(),
		PeriodEnd:    periodEnd.UTC(),
		TimezoneName: timezoneName,
		Definitions: []OperationsMetricDefinition{
			{MetricID: "runs-today", Label: "Runs Today", Unit: "runs", Direction: "up", Formula: "count(runs within window)", Description: "Runs started or ended inside the reporting window."},
			{MetricID: "avg-lead-time", Label: "Avg Lead Time", Unit: "m", Direction: "down", Formula: "avg(ended_at - started_at)", Description: "Average cycle time in minutes."},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Unit: "%", Direction: "down", Formula: "needs_approval_runs / total_runs", Description: "Share of runs awaiting manual intervention."},
			{MetricID: "sla", Label: "SLA", Unit: "%", Direction: "up", Formula: "runs within SLA / measured runs", Description: "Measured runs that met the SLA target."},
			{MetricID: "regression", Label: "Regression", Unit: "count", Direction: "down", Formula: "count(score regressions)", Description: "Detected benchmark regressions."},
			{MetricID: "risk", Label: "Risk", Unit: "score", Direction: "down", Formula: "avg(risk score)", Description: "Average risk score across runs."},
			{MetricID: "spend", Label: "Spend", Unit: "USD", Direction: "down", Formula: "sum(spend_usd)", Description: "Spend represented by the reporting slice."},
		},
		Values: []OperationsMetricValue{
			{MetricID: "runs-today", Label: "Runs Today", Value: float64(runsToday), DisplayValue: fmt.Sprintf("%d", runsToday), Numerator: float64(runsToday), Denominator: float64(len(runs)), Unit: "runs"},
			{MetricID: "avg-lead-time", Label: "Avg Lead Time", Value: avgLeadTime, DisplayValue: fmt.Sprintf("%.1f", avgLeadTime), Numerator: roundTenth(totalCycle), Denominator: float64(measuredRuns), Unit: "m"},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Value: interventionRate, DisplayValue: fmt.Sprintf("%.1f", interventionRate), Numerator: float64(intervenedRuns), Denominator: float64(len(runs)), Unit: "%"},
			{MetricID: "sla", Label: "SLA", Value: slaRate, DisplayValue: fmt.Sprintf("%.1f", slaRate), Numerator: float64(slaCompliantRuns), Denominator: float64(measuredRuns), Unit: "%"},
			{MetricID: "regression", Label: "Regression", Value: float64(regressionCount), DisplayValue: fmt.Sprintf("%d", regressionCount), Numerator: float64(regressionCount), Denominator: float64(len(runs)), Unit: "count"},
			{MetricID: "risk", Label: "Risk", Value: avgRisk, DisplayValue: fmt.Sprintf("%.1f", avgRisk), Numerator: roundTenth(riskSum), Denominator: float64(riskCount), Unit: "score"},
			{MetricID: "spend", Label: "Spend", Value: roundCurrency(spendSum), DisplayValue: fmt.Sprintf("%.2f", roundCurrency(spendSum)), Numerator: roundCurrency(spendSum), Denominator: float64(len(runs)), Unit: "USD"},
		},
	}
}

func BuildRepoCollaborationMetrics(runs []RunRecord) map[string]float64 {
	if len(runs) == 0 {
		return map[string]float64{
			"repo_link_coverage":         0,
			"accepted_commit_rate":       0,
			"discussion_density":         0,
			"accepted_lineage_depth_avg": 0,
		}
	}
	withLinks := 0
	withAccepted := 0
	discussionPosts := 0
	lineageSum := 0
	for _, run := range runs {
		if len(run.Closeout.RunCommitLinks) > 0 {
			withLinks++
		}
		if strings.TrimSpace(run.Closeout.AcceptedCommitHash) != "" {
			withAccepted++
		}
		discussionPosts += run.RepoDiscussionPosts
		lineageSum += run.AcceptedLineageDepth
	}
	return map[string]float64{
		"repo_link_coverage":         roundTenth((float64(withLinks) / float64(len(runs))) * 100),
		"accepted_commit_rate":       roundTenth((float64(withAccepted) / float64(len(runs))) * 100),
		"discussion_density":         roundTenth(float64(discussionPosts) / float64(len(runs))),
		"accepted_lineage_depth_avg": roundTenth(float64(lineageSum) / float64(len(runs))),
	}
}

func NormalizeDashboardLayout(layout DashboardLayout, widgets []DashboardWidgetSpec) DashboardLayout {
	out := layout
	if out.Columns <= 0 {
		out.Columns = 12
	}
	index := make(map[string]DashboardWidgetSpec, len(widgets))
	for _, widget := range widgets {
		index[widget.WidgetID] = widget
	}
	out.Placements = append([]DashboardWidgetPlacement(nil), layout.Placements...)
	for i := range out.Placements {
		p := &out.Placements[i]
		if p.Column < 0 {
			p.Column = 0
		}
		if p.Row < 0 {
			p.Row = 0
		}
		if p.Height <= 0 {
			p.Height = 1
		}
		minWidth := 1
		maxWidth := out.Columns
		if widget, ok := index[p.WidgetID]; ok {
			if widget.MinWidth > 0 {
				minWidth = widget.MinWidth
			}
			if widget.MaxWidth > 0 {
				maxWidth = widget.MaxWidth
			}
		}
		if p.Width < minWidth {
			p.Width = minWidth
		}
		if p.Width > maxWidth {
			p.Width = maxWidth
		}
		if p.Width > out.Columns {
			p.Width = out.Columns
		}
		if p.Column+p.Width > out.Columns {
			p.Column = out.Columns - p.Width
		}
	}
	sort.SliceStable(out.Placements, func(i, j int) bool {
		if out.Placements[i].Row == out.Placements[j].Row {
			if out.Placements[i].Column == out.Placements[j].Column {
				return out.Placements[i].PlacementID < out.Placements[j].PlacementID
			}
			return out.Placements[i].Column < out.Placements[j].Column
		}
		return out.Placements[i].Row < out.Placements[j].Row
	})
	return out
}

func BuildTriageClusters(runs []RunRecord) []TriageCluster {
	groups := make(map[string]*TriageCluster)
	for _, run := range runs {
		if run.Status != "needs-approval" && run.Status != "failed" {
			continue
		}
		reason := strings.TrimSpace(run.Reason)
		if reason == "" {
			reason = "unknown"
		}
		entry := groups[reason]
		if entry == nil {
			entry = &TriageCluster{Reason: reason}
			groups[reason] = entry
		}
		entry.Occurrences++
		entry.TaskIDs = append(entry.TaskIDs, run.TaskID)
	}
	out := make([]TriageCluster, 0, len(groups))
	for _, item := range groups {
		sort.Strings(item.TaskIDs)
		out = append(out, *item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Occurrences == out[j].Occurrences {
			return out[i].Reason < out[j].Reason
		}
		return out[i].Occurrences > out[j].Occurrences
	})
	return out
}

func AnalyzeRegressionDeltas(current, baseline BenchmarkSuite) []RegressionDelta {
	baselineByID := make(map[string]BenchmarkCaseResult, len(baseline.Results))
	for _, result := range baseline.Results {
		baselineByID[result.CaseID] = result
	}
	out := make([]RegressionDelta, 0)
	for _, result := range current.Results {
		prev, ok := baselineByID[result.CaseID]
		if !ok {
			continue
		}
		delta := result.Score - prev.Score
		if delta >= 0 && (!prev.Passed || result.Passed) {
			continue
		}
		severity := "medium"
		if delta <= -25 || (prev.Passed && !result.Passed) {
			severity = "high"
		}
		out = append(out, RegressionDelta{CaseID: result.CaseID, Delta: delta, Severity: severity})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Delta == out[j].Delta {
			return out[i].CaseID < out[j].CaseID
		}
		return out[i].Delta < out[j].Delta
	})
	return out
}

func BuildRegressionOverview(current, baseline BenchmarkSuite) RegressionOverview {
	currentByID := make(map[string]BenchmarkCaseResult, len(current.Results))
	baselineByID := make(map[string]BenchmarkCaseResult, len(baseline.Results))
	for _, item := range current.Results {
		currentByID[item.CaseID] = item
	}
	for _, item := range baseline.Results {
		baselineByID[item.CaseID] = item
	}
	overview := RegressionOverview{Regressions: AnalyzeRegressionDeltas(current, baseline)}
	overview.RegressionCount = len(overview.Regressions)
	for caseID, currentItem := range currentByID {
		baselineItem, ok := baselineByID[caseID]
		if !ok {
			continue
		}
		delta := currentItem.Score - baselineItem.Score
		switch {
		case delta > 0 || (!baselineItem.Passed && currentItem.Passed):
			overview.ImprovedCases = append(overview.ImprovedCases, caseID)
		case delta == 0 && baselineItem.Passed == currentItem.Passed:
			overview.UnchangedCases = append(overview.UnchangedCases, caseID)
		}
	}
	sort.Strings(overview.ImprovedCases)
	sort.Strings(overview.UnchangedCases)
	return overview
}

func BuildWeeklyOperationsReportCompat(name, period string, runs []RunRecord, current, baseline BenchmarkSuite, slaTargetMinutes int) WeeklyOperationsReportCompat {
	return WeeklyOperationsReportCompat{
		Name:        name,
		Period:      period,
		Snapshot:    SummarizeRunRecords(runs, slaTargetMinutes),
		Clusters:    BuildTriageClusters(runs),
		Regressions: BuildRegressionOverview(current, baseline),
	}
}

func RenderOperationsDashboardWithView(snapshot OperationsSnapshot, view *SharedViewContext, clusters []TriageCluster) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Dashboard\n\n")
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", snapshot.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Approval Queue Depth: %d\n", snapshot.ApprovalQueueDepth))
	builder.WriteString(fmt.Sprintf("- Success Rate: %.1f\n", snapshot.SuccessRate))
	builder.WriteString(fmt.Sprintf("- SLA Breaches: %d\n", snapshot.SLABreachCount))
	builder.WriteString(fmt.Sprintf("- Average Cycle Minutes: %.1f\n\n", snapshot.AverageCycleMinutes))
	builder.WriteString(renderSharedViewContext(view))
	builder.WriteString("## Blockers\n\n")
	if len(clusters) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, cluster := range clusters {
		builder.WriteString(fmt.Sprintf("- %s: occurrences=%d tasks=%s\n", cluster.Reason, cluster.Occurrences, joinOrNone(cluster.TaskIDs)))
	}
	return builder.String()
}

func RenderRegressionOverviewWithView(overview RegressionOverview, view *SharedViewContext) string {
	builder := strings.Builder{}
	builder.WriteString("# Regression Analysis Center\n\n")
	builder.WriteString(fmt.Sprintf("- Regression Count: %d\n", overview.RegressionCount))
	builder.WriteString(fmt.Sprintf("- Improved Cases: %s\n", joinOrNone(overview.ImprovedCases)))
	builder.WriteString(fmt.Sprintf("- Unchanged Cases: %s\n\n", joinOrNone(overview.UnchangedCases)))
	builder.WriteString(renderSharedViewContext(view))
	if view != nil && len(view.PartialData) > 0 {
		builder.WriteString("## Partial Data\n\n")
		for _, item := range view.PartialData {
			builder.WriteString("- " + item + "\n")
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Regressions\n\n")
	if len(overview.Regressions) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, item := range overview.Regressions {
		builder.WriteString(fmt.Sprintf("- %s: delta=%d severity=%s\n", item.CaseID, item.Delta, item.Severity))
	}
	return builder.String()
}

func RenderPolicyPromptVersionCenterWithView(center PolicyPromptVersionCenter, view *SharedViewContext) string {
	return RenderPolicyPromptVersionCenter(center) + "\n" + strings.TrimSpace(renderSharedViewContext(view)) + "\n"
}

func RenderWeeklyOperationsReportCompat(report WeeklyOperationsReportCompat) string {
	builder := strings.Builder{}
	builder.WriteString("# Weekly Operations Report\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", report.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", report.Period))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", report.Snapshot.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Approval Queue Depth: %d\n", report.Snapshot.ApprovalQueueDepth))
	builder.WriteString(fmt.Sprintf("- SLA Breaches: %d\n\n", report.Snapshot.SLABreachCount))
	builder.WriteString("## Regressions\n\n")
	if len(report.Regressions.Regressions) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range report.Regressions.Regressions {
			builder.WriteString(fmt.Sprintf("- %s: severity=%s delta=%d\n", item.CaseID, item.Severity, item.Delta))
		}
	}
	builder.WriteString("\n## Blockers\n\n")
	if len(report.Clusters) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, cluster := range report.Clusters {
			builder.WriteString(fmt.Sprintf("- %s: occurrences=%d\n", cluster.Reason, cluster.Occurrences))
		}
	}
	return builder.String()
}

func WriteWeeklyOperationsBundleCompat(rootDir string, report WeeklyOperationsReportCompat, metricSpec *OperationsMetricSpec, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return WeeklyArtifacts{}, err
	}
	artifacts := WeeklyArtifacts{
		RootDir:          rootDir,
		WeeklyReportPath: filepath.Join(rootDir, "weekly-operations.md"),
		DashboardPath:    filepath.Join(rootDir, "operations-dashboard.md"),
	}
	if err := WriteReport(artifacts.WeeklyReportPath, RenderWeeklyOperationsReportCompat(report)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := WriteReport(artifacts.DashboardPath, RenderOperationsDashboardWithView(report.Snapshot, nil, report.Clusters)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if metricSpec != nil {
		artifacts.MetricSpecPath = filepath.Join(rootDir, "operations-metric-spec.md")
		if err := WriteReport(artifacts.MetricSpecPath, RenderOperationsMetricSpec(*metricSpec)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if report.Regressions.RegressionCount > 0 {
		artifacts.RegressionCenterPath = filepath.Join(rootDir, "regression-center.md")
		if err := WriteReport(artifacts.RegressionCenterPath, RenderRegressionOverviewWithView(report.Regressions, nil)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if versionCenter != nil {
		artifacts.VersionCenterPath = filepath.Join(rootDir, "policy-prompt-version-center.md")
		if err := WriteReport(artifacts.VersionCenterPath, RenderPolicyPromptVersionCenter(*versionCenter)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	return artifacts, nil
}

func BuildEngineeringOverviewFromRunRecords(name, period, viewerRole string, runs []RunRecord, slaTargetMinutes int) EngineeringOverview {
	tasks := make([]domain.Task, 0, len(runs))
	events := make([]domain.Event, 0)
	for _, run := range runs {
		started := parseRFC3339ish(run.StartedAt)
		ended := parseRFC3339ish(run.EndedAt)
		task := domain.Task{
			ID:        run.TaskID,
			TraceID:   run.RunID,
			Title:     run.Summary,
			State:     taskStateForRunStatus(run.Status),
			RiskLevel: riskLevelForRun(run),
			CreatedAt: started,
			UpdatedAt: ended,
			Metadata: map[string]string{
				"run_id":          run.RunID,
				"summary":         run.Summary,
				"approval_status": run.Status,
				"blocked_reason":  run.Reason,
				"failure_reason":  run.Reason,
			},
		}
		tasks = append(tasks, task)
		if task.State == domain.TaskBlocked || task.State == domain.TaskFailed {
			events = append(events, domain.Event{
				ID:        "evt-" + run.RunID,
				Type:      domain.EventRunAnnotated,
				TaskID:    task.ID,
				RunID:     run.RunID,
				Timestamp: ended,
				Payload:   map[string]any{"reason": run.Reason},
			})
		}
	}
	return BuildEngineeringOverview(name, period, viewerRole, tasks, events, slaTargetMinutes, 5, 5)
}

func runCycleMinutes(run RunRecord) float64 {
	started := parseRFC3339ish(run.StartedAt)
	ended := parseRFC3339ish(run.EndedAt)
	if started.IsZero() || ended.IsZero() || ended.Before(started) {
		return -1
	}
	return roundTenth(ended.Sub(started).Minutes())
}

func runRiskScore(run RunRecord) (float64, bool) {
	if run.RiskScore > 0 {
		return run.RiskScore, true
	}
	switch strings.ToLower(strings.TrimSpace(run.RiskLevel)) {
	case "high":
		return 90, true
	case "medium":
		return 60, true
	case "low":
		return 25, true
	default:
		return 0, false
	}
}

func roundCurrency(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func derefSuite(suite *BenchmarkSuite) BenchmarkSuite {
	if suite == nil {
		return BenchmarkSuite{}
	}
	return *suite
}

func taskStateForRunStatus(status string) domain.TaskState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "completed":
		return domain.TaskSucceeded
	case "running":
		return domain.TaskRunning
	case "needs-approval":
		return domain.TaskBlocked
	case "failed":
		return domain.TaskFailed
	default:
		return domain.TaskQueued
	}
}

func riskLevelForRun(run RunRecord) domain.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(run.RiskLevel)) {
	case "high":
		return domain.RiskHigh
	case "medium":
		return domain.RiskMedium
	case "low":
		return domain.RiskLow
	default:
		if run.RiskScore >= 80 {
			return domain.RiskHigh
		}
		if run.RiskScore >= 50 {
			return domain.RiskMedium
		}
		if run.RiskScore > 0 {
			return domain.RiskLow
		}
		return domain.RiskLow
	}
}

func RegressionCenterFromOverview(overview RegressionOverview) regression.Center {
	findings := make([]regression.Finding, 0, len(overview.Regressions))
	for _, item := range overview.Regressions {
		findings = append(findings, regression.Finding{
			TaskID:          item.CaseID,
			Severity:        item.Severity,
			RegressionCount: 1,
			Summary:         fmt.Sprintf("score delta %d", item.Delta),
		})
	}
	return regression.Center{
		Summary: regression.Summary{
			TotalRegressions: len(overview.Regressions),
			AffectedTasks:    len(overview.Regressions),
		},
		Findings: findings,
	}
}
