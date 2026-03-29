package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func makeRun(runID, taskID, status, startedAt, endedAt, summary, reason string) RunRecord {
	return RunRecord{
		RunID:     runID,
		TaskID:    taskID,
		Status:    status,
		StartedAt: startedAt,
		EndedAt:   endedAt,
		Summary:   summary,
		Audits:    []RunAudit{{Reason: reason}},
	}
}

func makeSharedView(resultCount int, loading bool, partialData []string) *SharedViewContext {
	return &SharedViewContext{
		Filters: []SharedViewFilter{
			{Label: "Team", Value: "engineering"},
			{Label: "Status", Value: "needs-approval"},
		},
		ResultCount: intPtr(resultCount),
		Loading:     loading,
		PartialData: partialData,
		LastUpdated: "2026-03-11T09:00:00Z",
	}
}

func TestBuildRepoCollaborationMetrics(t *testing.T) {
	runs := []RunRecord{
		{
			RunID: "r1",
			Closeout: RunCloseout{
				RunCommitLinks:     []RunCommitLinkLite{{Role: "candidate"}},
				AcceptedCommitHash: "abc123",
			},
			RepoDiscussionPosts:  3,
			AcceptedLineageDepth: floatPtr(2),
		},
		{
			RunID: "r2",
			Closeout: RunCloseout{
				RunCommitLinks: nil,
			},
			RepoDiscussionPosts:  1,
			AcceptedLineageDepth: floatPtr(4),
		},
	}
	metrics := BuildRepoCollaborationMetrics(runs)
	if metrics["repo_link_coverage"] != 50.0 || metrics["accepted_commit_rate"] != 50.0 || metrics["discussion_density"] != 2.0 || metrics["accepted_lineage_depth_avg"] != 3.0 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func TestBuildRunOperationsMetricSpecAndRender(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-4305-1", Status: "approved", StartedAt: "2026-03-11T00:10:00Z", EndedAt: "2026-03-11T00:40:00Z", Summary: "ok", Audits: []RunAudit{{Reason: "default low risk path"}}, RiskLevel: "low", SpendUSD: floatPtr(4.25)},
		{RunID: "run-2", TaskID: "BIG-4305-2", Status: "needs-approval", StartedAt: "2026-03-11T02:00:00Z", EndedAt: "2026-03-11T03:30:00Z", Summary: "manual", Audits: []RunAudit{{Reason: "requires approval for production rollout"}}, RiskScoreTotal: floatPtr(88), CostUSD: floatPtr(7.5)},
		{RunID: "run-3", TaskID: "BIG-4305-3", Status: "approved", StartedAt: "2026-03-10T23:30:00Z", EndedAt: "2026-03-11T00:20:00Z", Summary: "overnight", Audits: []RunAudit{{Reason: "batch cleanup"}}, RiskLevel: "medium", Spend: floatPtr(3)},
	}
	baselineSuite := &BenchmarkSuite{Version: "v1.0.0", Results: []BenchmarkCase{{CaseID: "case-1", Score: 92, Passed: true}, {CaseID: "case-2", Score: 88, Passed: true}}}
	currentSuite := &BenchmarkSuite{Version: "v1.1.0", Results: []BenchmarkCase{{CaseID: "case-1", Score: 70, Passed: false}, {CaseID: "case-2", Score: 90, Passed: true}}}
	spec := BuildRunOperationsMetricSpec(runs, "2026-03-11T00:00:00Z", "2026-03-11T23:59:59Z", "UTC", "2026-03-11T09:00:00Z", 60, currentSuite, baselineSuite)
	values := map[string]OperationsMetricValue{}
	for _, value := range spec.Values {
		values[value.MetricID] = value
	}
	if values["runs-today"].Value != 2 || values["avg-lead-time"].Value != 56.7 || values["intervention-rate"].Value != 33.3 || values["sla"].Value != 66.7 || values["regression"].Value != 1 || values["risk"].Value != 57.7 || values["spend"].Value != 14.75 {
		t.Fatalf("unexpected values: %+v", values)
	}
	rendered := RenderOperationsMetricSpec(spec)
	if !strings.Contains(rendered, "# Operations Metric Spec") || !strings.Contains(rendered, "### Runs Today") || !strings.Contains(rendered, "Spend") {
		t.Fatalf("unexpected rendered metric spec: %s", rendered)
	}
}

func TestNormalizeDashboardLayoutClampsDimensionsAndSortsPlacements(t *testing.T) {
	widgets := []DashboardWidgetSpec{{WidgetID: "success-rate", Title: "Success Rate", Module: "kpis", DataSource: "operations.snapshot", MinWidth: 3, MaxWidth: 6}}
	layout := DashboardLayout{
		LayoutID: "desktop",
		Name:     "Desktop",
		Placements: []DashboardWidgetPlacement{
			{PlacementID: "late", WidgetID: "success-rate", Column: 8, Row: 4, Width: 8, Height: 0},
			{PlacementID: "early", WidgetID: "success-rate", Column: -2, Row: -1, Width: 1, Height: 2},
		},
	}
	normalized := NormalizeDashboardLayout(layout, widgets)
	if got := []string{normalized.Placements[0].PlacementID, normalized.Placements[1].PlacementID}; got[0] != "early" || got[1] != "late" {
		t.Fatalf("unexpected placement order: %+v", normalized.Placements)
	}
	if normalized.Placements[0].Column != 0 || normalized.Placements[0].Row != 0 || normalized.Placements[0].Width != 3 {
		t.Fatalf("unexpected first placement: %+v", normalized.Placements[0])
	}
	if normalized.Placements[1].Column != 6 || normalized.Placements[1].Width != 6 || normalized.Placements[1].Height != 1 {
		t.Fatalf("unexpected second placement: %+v", normalized.Placements[1])
	}
}

func TestDashboardBuilderRoundTripAndViewReport(t *testing.T) {
	builder := BuildDashboardBuilder(
		"Manager Builder",
		"2026-W11",
		"manager",
		"engineering-manager",
		[]DashboardWidgetSpec{
			{WidgetID: "success-rate", Title: "Success Rate", Module: "kpis", DataSource: "operations.snapshot"},
			{WidgetID: "recent-activity", Title: "Recent Activity", Module: "activity", DataSource: "operations.runs"},
		},
		[]DashboardLayout{{
			LayoutID: "desktop",
			Name:     "Desktop",
			Placements: []DashboardWidgetPlacement{
				{PlacementID: "kpi-main", WidgetID: "success-rate", Column: 0, Row: 0, Width: 4, Height: 2},
				{PlacementID: "activity-main", WidgetID: "recent-activity", Column: 4, Row: 0, Width: 8, Height: 3, Filters: []string{"team=engineering"}},
			},
		}},
		true,
	)
	payload, err := json.Marshal(builder)
	if err != nil {
		t.Fatalf("marshal builder: %v", err)
	}
	var restored DashboardBuilder
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal builder: %v", err)
	}
	if restored.Name != builder.Name || len(restored.Layouts) != 1 || restored.Layouts[0].Placements[1].Filters[0] != "team=engineering" {
		t.Fatalf("unexpected restored builder: %+v", restored)
	}
	audit := AuditDashboardBuilder(builder)
	report := RenderDashboardBuilderReportWithView(builder, audit, makeSharedView(2, false, nil))
	if !audit.ReleaseReady() || !strings.Contains(report, "# Dashboard Builder") || !strings.Contains(report, "- Release Ready: true") || !strings.Contains(report, "- desktop: name=Desktop columns=12 placements=2") || !strings.Contains(report, "filters=team=engineering") {
		t.Fatalf("unexpected report: %s", report)
	}
	path, err := WriteDashboardBuilderBundleWithView(t.TempDir(), builder, audit, makeSharedView(2, false, nil))
	if err != nil {
		t.Fatalf("write dashboard builder bundle: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dashboard builder bundle: %v", err)
	}
	if !strings.Contains(string(body), "# Dashboard Builder") {
		t.Fatalf("unexpected dashboard builder bundle: %s", string(body))
	}
}

func TestBuildTriageClustersAndWeeklyRunReport(t *testing.T) {
	runs := []RunRecord{
		makeRun("run-1", "BIG-903-1", "needs-approval", "2026-03-10T10:00:00Z", "2026-03-10T10:05:00Z", "hold", "requires approval for high-risk task"),
		makeRun("run-2", "BIG-903-2", "failed", "2026-03-10T10:00:00Z", "2026-03-10T10:25:00Z", "tool fail", "browser automation task"),
		makeRun("run-3", "BIG-903-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:15:00Z", "hold", "requires approval for high-risk task"),
	}
	clusters := BuildTriageClusters(runs)
	if clusters[0].Reason != "requires approval for high-risk task" || clusters[0].Occurrences() != 2 || strings.Join(clusters[0].TaskIDs, ",") != "BIG-903-1,BIG-903-3" || clusters[1].Reason != "browser automation task" {
		t.Fatalf("unexpected clusters: %+v", clusters)
	}
	baseline := &BenchmarkSuite{Version: "v0.1", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 100, Passed: true}}}
	current := &BenchmarkSuite{Version: "v0.2", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 70, Passed: false}}}
	report := BuildWeeklyRunReport("Engineering Ops", "2026-W11", runs[:2], current, baseline, 60)
	dashboard := RenderOperationsDashboardWithView(report.Snapshot, nil)
	weekly := RenderWeeklyRunReport(report)
	if !strings.Contains(dashboard, "# Operations Dashboard") || !strings.Contains(dashboard, "- Approval Queue Depth: 1") || !strings.Contains(dashboard, "requires approval for high-risk task") || !strings.Contains(weekly, "# Weekly Operations Report") || !strings.Contains(weekly, "case-drop") || !strings.Contains(weekly, "severity=high") {
		t.Fatalf("unexpected dashboard/report output:\n%s\n%s", dashboard, weekly)
	}
}

func TestOperationsDashboardAndRegressionCenterSupportSharedViewState(t *testing.T) {
	snapshot := BuildRunOperationsSnapshot(nil, 60)
	dashboard := RenderOperationsDashboardWithView(snapshot, makeSharedView(0, true, nil))
	if !strings.Contains(dashboard, "## View State") || !strings.Contains(dashboard, "- State: loading") || !strings.Contains(dashboard, "- Summary: Loading data for the current filters.") || !strings.Contains(dashboard, "- Team: engineering") {
		t.Fatalf("unexpected dashboard loading state: %s", dashboard)
	}

	baseline := &BenchmarkSuite{Version: "v0.1", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 100, Passed: true}, {CaseID: "case-up", Score: 60, Passed: false}, {CaseID: "case-stable", Score: 100, Passed: true}}}
	current := &BenchmarkSuite{Version: "v0.2", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 70, Passed: false}, {CaseID: "case-up", Score: 100, Passed: true}, {CaseID: "case-stable", Score: 100, Passed: true}}}
	center := BuildRegressionCenterLite("Regression Analysis Center", current, baseline)
	if center.RegressionCount() != 1 || center.Regressions[0].CaseID != "case-drop" || strings.Join(center.ImprovedCases, ",") != "case-up" || strings.Join(center.UnchangedCases, ",") != "case-stable" {
		t.Fatalf("unexpected regression center: %+v", center)
	}
	report := RenderRegressionCenterLite(center, makeSharedView(1, false, []string{"Historical baseline fetch is delayed."}))
	if !strings.Contains(report, "- State: partial-data") || !strings.Contains(report, "## Partial Data") || !strings.Contains(report, "Historical baseline fetch is delayed.") {
		t.Fatalf("unexpected regression center partial state: %s", report)
	}
}

func TestPolicyPromptVersionCenterWithViewAndWeeklyBundle(t *testing.T) {
	center := BuildPolicyPromptVersionCenter(
		"Policy/Prompt Version Center",
		time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC),
		[]VersionedArtifact{
			{ArtifactType: "prompt", ArtifactID: "triage-system", Version: "v2", UpdatedAt: "2026-03-10T14:00:00Z", Author: "ops-bot", Summary: "reduce false escalations", Content: "system: keep concise\nrubric: strict\n"},
			{ArtifactType: "prompt", ArtifactID: "triage-system", Version: "v1", UpdatedAt: "2026-03-08T14:00:00Z", Author: "ops-bot", Summary: "initial prompt", Content: "system: keep concise\n"},
		},
		8,
	)
	rendered := RenderPolicyPromptVersionCenterWithView(center, makeSharedView(1, false, []string{"Rollback simulation still running."}))
	if !strings.Contains(rendered, "# Policy/Prompt Version Center") || !strings.Contains(rendered, "### prompt / triage-system") || !strings.Contains(rendered, "- Rollback Version: v1") || !strings.Contains(rendered, "```diff") || !strings.Contains(rendered, "- State: partial-data") || !strings.Contains(rendered, "Rollback simulation still running.") {
		t.Fatalf("unexpected version center render: %s", rendered)
	}

	runs := []RunRecord{
		makeRun("run-1", "BIG-905-1", "approved", "2026-03-10T10:00:00Z", "2026-03-10T10:20:00Z", "ok", "default low risk path"),
		makeRun("run-2", "BIG-905-2", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T11:50:00Z", "hold", "requires approval for high-risk task"),
	}
	baseline := &BenchmarkSuite{Version: "v0.1", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 100, Passed: true}}}
	current := &BenchmarkSuite{Version: "v0.2", Results: []BenchmarkCase{{CaseID: "case-drop", Score: 70, Passed: false}}}
	report := BuildWeeklyRunReport("Engineering Ops", "2026-W11", runs, current, baseline, 60)
	regressionCenter := BuildRegressionCenterLite("Regression Analysis Center", current, baseline)
	artifacts, err := WriteWeeklyRunOperationsBundle(t.TempDir(), report, nil, &regressionCenter, &center)
	if err != nil {
		t.Fatalf("write weekly run operations bundle: %v", err)
	}
	for _, path := range []string{artifacts.WeeklyReportPath, artifacts.DashboardPath, artifacts.RegressionCenterPath, artifacts.VersionCenterPath} {
		if path == "" {
			t.Fatalf("expected populated artifact path in %+v", artifacts)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact to exist at %s: %v", path, err)
		}
	}
	body, err := os.ReadFile(artifacts.WeeklyReportPath)
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	if !strings.Contains(string(body), "# Weekly Operations Report") {
		t.Fatalf("unexpected weekly report body: %s", string(body))
	}
}

func TestBuildEngineeringOverviewFromRunsAndBundle(t *testing.T) {
	runs := []RunRecord{
		makeRun("run-1", "BIG-1401-1", "approved", "2026-03-10T09:00:00Z", "2026-03-10T09:20:00Z", "merged", "default low risk path"),
		makeRun("run-2", "BIG-1401-2", "running", "2026-03-10T10:00:00Z", "2026-03-10T10:30:00Z", "in flight", "long running implementation"),
		makeRun("run-3", "BIG-1401-3", "needs-approval", "2026-03-10T11:00:00Z", "2026-03-10T12:10:00Z", "approval", "requires approval for prod deploy"),
		makeRun("run-4", "BIG-1401-4", "failed", "2026-03-10T12:00:00Z", "2026-03-10T12:45:00Z", "regression", "security scan failed"),
	}
	overview := BuildEngineeringOverviewFromRuns("Core Product", "2026-W11", "engineering-manager", runs, 60)
	if got := overview.Permissions.AllowedModules; strings.Join(got, ",") != "kpis,funnel,blockers,activity" {
		t.Fatalf("unexpected permissions: %+v", overview.Permissions)
	}
	if got := []string{overview.KPIs[0].Name, overview.KPIs[1].Name, overview.KPIs[2].Name, overview.KPIs[3].Name}; strings.Join(got, ",") != "success-rate,approval-queue-depth,sla-breaches,average-cycle-minutes" {
		t.Fatalf("unexpected KPIs: %+v", overview.KPIs)
	}
	if got := []string{overview.Funnel[0].Name, overview.Funnel[1].Name, overview.Funnel[2].Name, overview.Funnel[3].Name}; strings.Join(got, ",") != "queued,in-progress,awaiting-approval,completed" {
		t.Fatalf("unexpected funnel: %+v", overview.Funnel)
	}
	if overview.Blockers[0].Owner != "operations" || overview.Blockers[0].Severity != "medium" || overview.Blockers[1].Owner != "security" || overview.Blockers[1].Severity != "high" || overview.Activities[0].RunID != "run-4" {
		t.Fatalf("unexpected overview: %+v", overview)
	}

	executiveView := BuildEngineeringOverviewFromRuns("Executive View", "2026-W11", "executive", runs[:2], 60)
	contributorView := BuildEngineeringOverviewFromRuns("Contributor View", "2026-W11", "contributor", runs[:2], 60)
	executiveReport := RenderEngineeringOverview(executiveView)
	contributorReport := RenderEngineeringOverview(contributorView)
	if !strings.Contains(executiveReport, "## KPI Modules") || !strings.Contains(executiveReport, "## Funnel Modules") || !strings.Contains(executiveReport, "## Blocker Modules") || strings.Contains(executiveReport, "## Activity Modules") {
		t.Fatalf("unexpected executive report: %s", executiveReport)
	}
	if !strings.Contains(contributorReport, "## KPI Modules") || !strings.Contains(contributorReport, "## Activity Modules") || strings.Contains(contributorReport, "## Funnel Modules") || strings.Contains(contributorReport, "## Blocker Modules") {
		t.Fatalf("unexpected contributor report: %s", contributorReport)
	}
	outputPath, err := WriteEngineeringOverviewBundle(t.TempDir(), BuildEngineeringOverviewFromRuns("Core Product", "2026-W11", "operations", runs[:2], 60))
	if err != nil {
		t.Fatalf("write engineering overview bundle: %v", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read engineering overview bundle: %v", err)
	}
	if !strings.Contains(string(content), "# Engineering Overview") || !strings.Contains(string(content), "Viewer Role: operations") || !strings.Contains(string(content), "## Activity Modules") {
		t.Fatalf("unexpected engineering overview bundle: %s", string(content))
	}
}

func TestWriteRunOperationsArtifactsPathsAreStable(t *testing.T) {
	root := t.TempDir()
	report := WeeklyRunReport{Name: "Ops", Period: "2026-W11", Snapshot: RunOperationsSnapshot{}}
	artifacts, err := WriteWeeklyRunOperationsBundle(root, report, nil, nil, nil)
	if err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if artifacts.WeeklyReportPath != filepath.Join(root, "weekly-operations.md") || artifacts.DashboardPath != filepath.Join(root, "operations-dashboard.md") {
		t.Fatalf("unexpected artifact paths: %+v", artifacts)
	}
}
