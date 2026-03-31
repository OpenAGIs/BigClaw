package reporting

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSummarizeRunRecordsTracksSLAAndSuccessRate(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-901-1", Status: "approved", StartedAt: "2026-03-10T10:00:00Z", EndedAt: "2026-03-10T10:20:00Z", Summary: "ok", Reason: "default low risk path"},
		{RunID: "run-2", TaskID: "BIG-901-2", Status: "approved", StartedAt: "2026-03-10T11:00:00Z", EndedAt: "2026-03-10T12:30:00Z", Summary: "slow", Reason: "browser automation task"},
		{RunID: "run-3", TaskID: "BIG-901-3", Status: "needs-approval", StartedAt: "2026-03-10T13:00:00Z", EndedAt: "2026-03-10T13:45:00Z", Summary: "approval", Reason: "requires approval for high-risk task"},
	}

	snapshot := SummarizeRunRecords(runs, 60)

	if snapshot.TotalRuns != 3 || snapshot.StatusCounts["approved"] != 2 || snapshot.StatusCounts["needs-approval"] != 1 {
		t.Fatalf("unexpected snapshot counts: %+v", snapshot)
	}
	if snapshot.SuccessRate != 66.7 || snapshot.ApprovalQueueDepth != 1 || snapshot.SLABreachCount != 1 || snapshot.AverageCycleMinutes != 51.7 {
		t.Fatalf("unexpected snapshot metrics: %+v", snapshot)
	}
}

func TestBuildOperationsMetricSpecFromRuns(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-4305-1", Status: "approved", StartedAt: "2026-03-11T00:10:00Z", EndedAt: "2026-03-11T00:40:00Z", Summary: "ok", Reason: "default low risk path", RiskLevel: "low", SpendUSD: 4.25},
		{RunID: "run-2", TaskID: "BIG-4305-2", Status: "needs-approval", StartedAt: "2026-03-11T02:00:00Z", EndedAt: "2026-03-11T03:30:00Z", Summary: "manual", Reason: "requires approval for production rollout", RiskScore: 88, SpendUSD: 7.5},
		{RunID: "run-3", TaskID: "BIG-4305-3", Status: "approved", StartedAt: "2026-03-10T23:30:00Z", EndedAt: "2026-03-11T00:20:00Z", Summary: "overnight", Reason: "batch cleanup", RiskLevel: "medium", SpendUSD: 3},
	}
	baseline := &BenchmarkSuite{Version: "v1.0.0", Results: []BenchmarkCaseResult{{CaseID: "case-1", Score: 92, Passed: true}, {CaseID: "case-2", Score: 88, Passed: true}}}
	current := &BenchmarkSuite{Version: "v1.1.0", Results: []BenchmarkCaseResult{{CaseID: "case-1", Score: 70, Passed: false}, {CaseID: "case-2", Score: 90, Passed: true}}}

	spec := BuildOperationsMetricSpecFromRuns(
		runs,
		parseRFC3339ish("2026-03-11T00:00:00Z"),
		parseRFC3339ish("2026-03-11T23:59:59Z"),
		parseRFC3339ish("2026-03-11T09:00:00Z"),
		"UTC",
		60,
		current,
		baseline,
	)

	values := map[string]OperationsMetricValue{}
	for _, value := range spec.Values {
		values[value.MetricID] = value
	}
	if got := []string{
		spec.Definitions[0].MetricID,
		spec.Definitions[1].MetricID,
		spec.Definitions[2].MetricID,
		spec.Definitions[3].MetricID,
		spec.Definitions[4].MetricID,
		spec.Definitions[5].MetricID,
		spec.Definitions[6].MetricID,
	}; !reflect.DeepEqual(got, []string{"runs-today", "avg-lead-time", "intervention-rate", "sla", "regression", "risk", "spend"}) {
		t.Fatalf("unexpected metric ids: %+v", got)
	}
	if values["runs-today"].Value != 3 || values["avg-lead-time"].Value != 56.7 || values["intervention-rate"].Value != 33.3 || values["sla"].Value != 66.7 || values["regression"].Value != 1 || values["risk"].Value != 57.7 || values["spend"].Value != 14.75 {
		t.Fatalf("unexpected metric values: %+v", values)
	}
}

func TestBuildRepoCollaborationMetrics(t *testing.T) {
	runs := []RunRecord{
		{RunID: "r1", Closeout: RunCloseoutRecord{RunCommitLinks: []RunCommitRoleLink{{Role: "candidate"}}, AcceptedCommitHash: "abc123"}, RepoDiscussionPosts: 3, AcceptedLineageDepth: 2},
		{RunID: "r2", Closeout: RunCloseoutRecord{}, RepoDiscussionPosts: 1, AcceptedLineageDepth: 4},
	}
	metrics := BuildRepoCollaborationMetrics(runs)
	if metrics["repo_link_coverage"] != 50.0 || metrics["accepted_commit_rate"] != 50.0 || metrics["discussion_density"] != 2.0 || metrics["accepted_lineage_depth_avg"] != 3.0 {
		t.Fatalf("unexpected repo collaboration metrics: %+v", metrics)
	}
}

func TestNormalizeDashboardLayoutClampsAndSortsPlacements(t *testing.T) {
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
	if got := []string{normalized.Placements[0].PlacementID, normalized.Placements[1].PlacementID}; !reflect.DeepEqual(got, []string{"early", "late"}) {
		t.Fatalf("unexpected placement order: %+v", normalized.Placements)
	}
	if normalized.Placements[0].Column != 0 || normalized.Placements[0].Row != 0 || normalized.Placements[0].Width != 3 {
		t.Fatalf("unexpected early placement normalization: %+v", normalized.Placements[0])
	}
	if normalized.Placements[1].Column != 6 || normalized.Placements[1].Width != 6 || normalized.Placements[1].Height != 1 {
		t.Fatalf("unexpected late placement normalization: %+v", normalized.Placements[1])
	}
}

func TestBuildTriageClustersGroupsByReason(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-903-1", Status: "needs-approval", Reason: "requires approval for high-risk task"},
		{RunID: "run-2", TaskID: "BIG-903-2", Status: "failed", Reason: "browser automation task"},
		{RunID: "run-3", TaskID: "BIG-903-3", Status: "needs-approval", Reason: "requires approval for high-risk task"},
	}
	clusters := BuildTriageClusters(runs)
	if clusters[0].Reason != "requires approval for high-risk task" || clusters[0].Occurrences != 2 || !reflect.DeepEqual(clusters[0].TaskIDs, []string{"BIG-903-1", "BIG-903-3"}) {
		t.Fatalf("unexpected first triage cluster: %+v", clusters)
	}
	if clusters[1].Reason != "browser automation task" {
		t.Fatalf("unexpected second triage cluster: %+v", clusters)
	}
}

func TestBuildRegressionOverview(t *testing.T) {
	baseline := BenchmarkSuite{Version: "v0.1", Results: []BenchmarkCaseResult{{CaseID: "case-drop", Score: 100, Passed: true}, {CaseID: "case-up", Score: 60, Passed: false}, {CaseID: "case-stable", Score: 100, Passed: true}}}
	current := BenchmarkSuite{Version: "v0.2", Results: []BenchmarkCaseResult{{CaseID: "case-drop", Score: 70, Passed: false}, {CaseID: "case-up", Score: 100, Passed: true}, {CaseID: "case-stable", Score: 100, Passed: true}}}
	overview := BuildRegressionOverview(current, baseline)
	if overview.RegressionCount != 1 || overview.Regressions[0].CaseID != "case-drop" || overview.Regressions[0].Delta != -30 || overview.Regressions[0].Severity != "high" {
		t.Fatalf("unexpected regression overview: %+v", overview)
	}
	if !reflect.DeepEqual(overview.ImprovedCases, []string{"case-up"}) || !reflect.DeepEqual(overview.UnchangedCases, []string{"case-stable"}) {
		t.Fatalf("unexpected regression classifications: %+v", overview)
	}
}

func TestRenderOperationsDashboardWithView(t *testing.T) {
	snapshot := SummarizeRunRecords(nil, 60)
	resultCount := 0
	report := RenderOperationsDashboardWithView(snapshot, &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "engineering"}},
		ResultCount: &resultCount,
		Loading:     true,
	}, nil)
	for _, fragment := range []string{"## View State", "- State: loading", "- Summary: Loading data for the current filters.", "- Team: engineering"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in operations dashboard, got %s", fragment, report)
		}
	}
}

func TestRenderRegressionOverviewWithView(t *testing.T) {
	overview := RegressionOverview{
		RegressionCount: 1,
		Regressions:     []RegressionDelta{{CaseID: "case-drop", Delta: -40, Severity: "high"}},
		ImprovedCases:   []string{"case-up"},
		UnchangedCases:  []string{"case-stable"},
	}
	resultCount := 1
	report := RenderRegressionOverviewWithView(overview, &SharedViewContext{
		Filters:     []SharedViewFilter{{Label: "Team", Value: "engineering"}},
		ResultCount: &resultCount,
		PartialData: []string{"Historical baseline fetch is delayed."},
	})
	for _, fragment := range []string{"# Regression Analysis Center", "- State: partial-data", "## Partial Data", "Historical baseline fetch is delayed.", "case-drop: delta=-40 severity=high"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in regression overview, got %s", fragment, report)
		}
	}
}

func TestRenderPolicyPromptVersionCenterWithView(t *testing.T) {
	center := BuildPolicyPromptVersionCenter(
		"Policy/Prompt Version Center",
		time.Date(2026, 3, 11, 9, 30, 0, 0, time.UTC),
		[]VersionedArtifact{
			{ArtifactType: "prompt", ArtifactID: "triage-system", Version: "v2", UpdatedAt: "2026-03-10T14:00:00Z", Summary: "reduce false escalations", Content: "system: keep concise\nrubric: strict\n"},
			{ArtifactType: "prompt", ArtifactID: "triage-system", Version: "v1", UpdatedAt: "2026-03-08T14:00:00Z", Summary: "initial prompt", Content: "system: keep concise\n"},
		},
		6,
	)
	resultCount := 1
	report := RenderPolicyPromptVersionCenterWithView(center, &SharedViewContext{
		ResultCount: &resultCount,
		PartialData: []string{"Rollback simulation still running."},
	})
	for _, fragment := range []string{"# Policy/Prompt Version Center", "### prompt / triage-system", "- Rollback Version: v1", "```diff", "- State: partial-data"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in version center report, got %s", fragment, report)
		}
	}
}

func TestWriteWeeklyOperationsBundleCompat(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-905-1", Status: "approved", StartedAt: "2026-03-10T10:00:00Z", EndedAt: "2026-03-10T10:20:00Z", Summary: "ok", Reason: "default low risk path"},
		{RunID: "run-2", TaskID: "BIG-905-2", Status: "needs-approval", StartedAt: "2026-03-10T11:00:00Z", EndedAt: "2026-03-10T11:50:00Z", Summary: "hold", Reason: "requires approval for high-risk task"},
	}
	baseline := BenchmarkSuite{Version: "v0.1", Results: []BenchmarkCaseResult{{CaseID: "case-drop", Score: 100, Passed: true}}}
	current := BenchmarkSuite{Version: "v0.2", Results: []BenchmarkCaseResult{{CaseID: "case-drop", Score: 70, Passed: false}}}
	report := BuildWeeklyOperationsReportCompat("Engineering Ops", "2026-W11", runs, current, baseline, 60)
	spec := BuildOperationsMetricSpecFromRuns(runs, parseRFC3339ish("2026-03-10T00:00:00Z"), parseRFC3339ish("2026-03-10T23:59:59Z"), time.Now(), "UTC", 60, &current, &baseline)
	versionCenter := BuildPolicyPromptVersionCenter("Policy/Prompt Version Center", time.Now(), []VersionedArtifact{
		{ArtifactType: "policy", ArtifactID: "release-approval", Version: "v2", UpdatedAt: "2026-03-10T09:00:00Z", Summary: "add rollback owner", Content: "approvals: 2\nrollback_owner: release-manager\n"},
		{ArtifactType: "policy", ArtifactID: "release-approval", Version: "v1", UpdatedAt: "2026-03-08T09:00:00Z", Summary: "initial policy", Content: "approvals: 2\n"},
	}, 6)

	artifacts, err := WriteWeeklyOperationsBundleCompat(t.TempDir(), report, &spec, &versionCenter)
	if err != nil {
		t.Fatalf("write weekly operations bundle compat: %v", err)
	}
	for _, path := range []string{artifacts.WeeklyReportPath, artifacts.DashboardPath, artifacts.MetricSpecPath, artifacts.RegressionCenterPath, artifacts.VersionCenterPath} {
		if path == "" {
			t.Fatalf("expected populated artifact path, got %+v", artifacts)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s to exist: %v", path, err)
		}
	}
	weeklyBody, _ := os.ReadFile(artifacts.WeeklyReportPath)
	if !strings.Contains(string(weeklyBody), "# Weekly Operations Report") {
		t.Fatalf("unexpected weekly report body: %s", string(weeklyBody))
	}
}

func TestBuildEngineeringOverviewFromRunRecords(t *testing.T) {
	runs := []RunRecord{
		{RunID: "run-1", TaskID: "BIG-1401-1", Status: "approved", StartedAt: "2026-03-10T09:00:00Z", EndedAt: "2026-03-10T09:20:00Z", Summary: "merged", Reason: "default low risk path"},
		{RunID: "run-2", TaskID: "BIG-1401-2", Status: "running", StartedAt: "2026-03-10T10:00:00Z", EndedAt: "2026-03-10T10:30:00Z", Summary: "in flight", Reason: "long running implementation"},
		{RunID: "run-3", TaskID: "BIG-1401-3", Status: "needs-approval", StartedAt: "2026-03-10T11:00:00Z", EndedAt: "2026-03-10T12:10:00Z", Summary: "approval", Reason: "requires approval for prod deploy"},
		{RunID: "run-4", TaskID: "BIG-1401-4", Status: "failed", StartedAt: "2026-03-10T12:00:00Z", EndedAt: "2026-03-10T12:45:00Z", Summary: "regression", Reason: "security scan failed"},
	}

	overview := BuildEngineeringOverviewFromRunRecords("Core Product", "2026-W11", "engineering-manager", runs, 60)
	if !reflect.DeepEqual(overview.Permissions.AllowedModules, []string{"kpis", "funnel", "blockers", "activity"}) {
		t.Fatalf("unexpected permissions: %+v", overview.Permissions)
	}
	if got := []string{overview.KPIs[0].Name, overview.KPIs[1].Name, overview.KPIs[2].Name, overview.KPIs[3].Name}; !reflect.DeepEqual(got, []string{"success-rate", "approval-queue-depth", "sla-breaches", "average-cycle-minutes"}) {
		t.Fatalf("unexpected kpis: %+v", overview.KPIs)
	}
	if got := [][2]any{{overview.Funnel[0].Name, overview.Funnel[0].Count}, {overview.Funnel[1].Name, overview.Funnel[1].Count}, {overview.Funnel[2].Name, overview.Funnel[2].Count}, {overview.Funnel[3].Name, overview.Funnel[3].Count}}; !reflect.DeepEqual(got, [][2]any{{"queued", 0}, {"in-progress", 1}, {"awaiting-approval", 1}, {"completed", 1}}) {
		t.Fatalf("unexpected funnel: %+v", overview.Funnel)
	}
	if overview.Blockers[0].Owner != "operations" || overview.Blockers[0].Severity != "medium" || overview.Blockers[1].Owner != "security" || overview.Blockers[1].Severity != "high" || overview.Activities[0].RunID != "run-4" {
		t.Fatalf("unexpected overview content: %+v", overview)
	}
	report := RenderEngineeringOverview(overview)
	if !strings.Contains(report, "## Activity Modules") {
		t.Fatalf("expected activity module in overview report, got %s", report)
	}
	outputDir := t.TempDir()
	path, err := WriteEngineeringOverviewBundle(filepath.Join(outputDir, "overview"), overview)
	if err != nil {
		t.Fatalf("write engineering overview bundle: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read engineering overview bundle: %v", err)
	}
	if !strings.Contains(string(body), "# Engineering Overview") || !strings.Contains(string(body), "Viewer Role: engineering-manager") {
		t.Fatalf("unexpected engineering overview bundle: %s", string(body))
	}
}
