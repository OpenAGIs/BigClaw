package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildWeeklyReportRendersExpandedMarkdown(t *testing.T) {
	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	tasks := []domain.Task{
		{
			ID:          "task-1",
			Title:       "Deploy API",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 1200,
			Metadata: map[string]string{
				"team":             "platform",
				"plan":             "premium",
				"regression_count": "2",
			},
			UpdatedAt: start.Add(2 * time.Hour),
		},
		{
			ID:          "task-2",
			Title:       "Fix flaky validation",
			State:       domain.TaskBlocked,
			BudgetCents: 800,
			Metadata: map[string]string{
				"team": "platform",
			},
			UpdatedAt: start.Add(3 * time.Hour),
		},
	}
	events := []domain.Event{
		{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-1", Timestamp: start.Add(30 * time.Minute)},
		{ID: "evt-2", Type: domain.EventControlPaused, TaskID: "task-2", Timestamp: start.Add(4 * time.Hour)},
	}

	weekly := Build(tasks, events, start, end)
	if weekly.Summary.TotalRuns != 2 || weekly.Summary.CompletedRuns != 1 || weekly.Summary.BlockedRuns != 1 {
		t.Fatalf("unexpected weekly summary: %+v", weekly.Summary)
	}
	if weekly.Summary.HighRiskRuns != 1 || weekly.Summary.RegressionFindings != 2 || weekly.Summary.HumanInterventions != 2 || weekly.Summary.PremiumRuns != 1 {
		t.Fatalf("unexpected weekly counters: %+v", weekly.Summary)
	}
	if len(weekly.TeamBreakdown) != 1 || weekly.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("unexpected team breakdown: %+v", weekly.TeamBreakdown)
	}
	for _, fragment := range []string{"## Team Breakdown", "## Highlights", "- High risk runs: 1", "- Premium runs: 1", "platform: total=2 completed=1 blocked=1"} {
		if !strings.Contains(weekly.Markdown, fragment) {
			t.Fatalf("expected %q in weekly markdown, got %s", fragment, weekly.Markdown)
		}
	}
}

func TestWriteWeeklyOperationsBundle(t *testing.T) {
	rootDir := t.TempDir()
	start := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)
	weekly := Weekly{
		WeekStart: start,
		WeekEnd:   end,
		Summary: Summary{
			TotalRuns:          4,
			CompletedRuns:      3,
			BlockedRuns:        1,
			HighRiskRuns:       2,
			RegressionFindings: 1,
			HumanInterventions: 2,
			BudgetCentsTotal:   4200,
			PremiumRuns:        1,
		},
		TeamBreakdown: []TeamBreakdown{{
			Key:                "platform",
			TotalRuns:          4,
			CompletedRuns:      3,
			BlockedRuns:        1,
			BudgetCentsTotal:   4200,
			HumanInterventions: 2,
		}},
		Highlights: []string{"Completed 3 / 4 runs this week."},
		Actions:    []string{"Review the blocked run before closeout."},
	}
	spec := &OperationsMetricSpec{
		Name:         "Weekly control-plane metrics",
		GeneratedAt:  start.Add(time.Hour),
		PeriodStart:  start,
		PeriodEnd:    end,
		TimezoneName: "UTC",
		Definitions: []OperationsMetricDefinition{{
			MetricID:     "success_rate",
			Label:        "Success Rate",
			Unit:         "%",
			Direction:    "up",
			Formula:      "completed_runs / total_runs",
			Description:  "Share of runs that completed within the reporting window.",
			SourceFields: []string{"summary.completed_runs", "summary.total_runs"},
		}},
		Values: []OperationsMetricValue{{
			MetricID:     "success_rate",
			Label:        "Success Rate",
			Value:        75,
			DisplayValue: "75.0%",
			Numerator:    3,
			Denominator:  4,
			Unit:         "%",
			Evidence:     []string{"weekly-operations.md", "operations-dashboard.md"},
		}},
	}

	artifacts, err := WriteWeeklyOperationsBundle(rootDir, weekly, spec)
	if err != nil {
		t.Fatalf("write weekly operations bundle: %v", err)
	}
	if artifacts.WeeklyReportPath != filepath.Join(rootDir, "weekly-operations.md") || artifacts.DashboardPath != filepath.Join(rootDir, "operations-dashboard.md") {
		t.Fatalf("unexpected bundle paths: %+v", artifacts)
	}
	if artifacts.MetricSpecPath != filepath.Join(rootDir, "operations-metric-spec.md") {
		t.Fatalf("expected metric spec path, got %+v", artifacts)
	}

	reportBody, err := os.ReadFile(artifacts.WeeklyReportPath)
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	if !strings.Contains(string(reportBody), "## Team Breakdown") || !strings.Contains(string(reportBody), "Review the blocked run before closeout.") {
		t.Fatalf("unexpected weekly report content: %s", string(reportBody))
	}

	dashboardBody, err := os.ReadFile(artifacts.DashboardPath)
	if err != nil {
		t.Fatalf("read dashboard: %v", err)
	}
	if !strings.Contains(string(dashboardBody), "# Operations Dashboard") || !strings.Contains(string(dashboardBody), "High Risk Runs: 2") {
		t.Fatalf("unexpected dashboard content: %s", string(dashboardBody))
	}

	specBody, err := os.ReadFile(artifacts.MetricSpecPath)
	if err != nil {
		t.Fatalf("read metric spec: %v", err)
	}
	for _, fragment := range []string{"# Operations Metric Spec", "### Success Rate", "value=75.0%"} {
		if !strings.Contains(string(specBody), fragment) {
			t.Fatalf("expected %q in metric spec output, got %s", fragment, string(specBody))
		}
	}
}

func TestAuditDashboardBuilderFlagsGovernanceGaps(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W11",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operator",
			AllowedModules: []string{"operations"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
			{WidgetID: "audit-feed", Title: "Audit Feed", Module: "audit", DataSource: "/v2/audit", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "primary",
				Name:     "Primary",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "placement-1", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 6, Height: 2},
					{PlacementID: "placement-1", WidgetID: "missing-widget", Column: 5, Row: 0, Width: 8, Height: 2},
					{PlacementID: "placement-3", WidgetID: "audit-feed", Column: 2, Row: 1, Width: 4, Height: 2},
				},
			},
			{
				LayoutID:   "empty-layout",
				Name:       "Empty",
				Columns:    12,
				Placements: nil,
			},
		},
		DocumentationComplete: false,
	}

	audit := AuditDashboardBuilder(dashboard)
	if audit.TotalWidgets != 2 || audit.LayoutCount != 2 || audit.PlacedWidgets != 3 {
		t.Fatalf("unexpected audit counts: %+v", audit)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected audit to block release, got %+v", audit)
	}
	for _, expected := range []string{"placement-1"} {
		if !strings.Contains(strings.Join(audit.DuplicatePlacementIDs, ","), expected) {
			t.Fatalf("expected duplicate placement %q in %+v", expected, audit.DuplicatePlacementIDs)
		}
	}
	if len(audit.MissingWidgetDefs) != 1 || audit.MissingWidgetDefs[0] != "missing-widget" {
		t.Fatalf("unexpected missing widget defs: %+v", audit.MissingWidgetDefs)
	}
	if len(audit.InaccessibleWidgets) != 1 || audit.InaccessibleWidgets[0] != "audit-feed" {
		t.Fatalf("unexpected inaccessible widgets: %+v", audit.InaccessibleWidgets)
	}
	if len(audit.OutOfBoundsPlacements) != 1 || audit.OutOfBoundsPlacements[0] != "placement-1" {
		t.Fatalf("unexpected out of bounds placements: %+v", audit.OutOfBoundsPlacements)
	}
	if len(audit.OverlappingPlacements) != 2 {
		t.Fatalf("expected overlapping placements, got %+v", audit.OverlappingPlacements)
	}
	if len(audit.EmptyLayouts) != 1 || audit.EmptyLayouts[0] != "empty-layout" {
		t.Fatalf("unexpected empty layouts: %+v", audit.EmptyLayouts)
	}
}

func TestRenderAndWriteDashboardBuilderBundle(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W11",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operator",
			AllowedModules: []string{"operations", "audit"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
			{WidgetID: "audit-feed", Title: "Audit Feed", Module: "audit", DataSource: "/v2/audit", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "primary",
				Name:     "Primary",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "placement-1", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 6, Height: 2, Filters: []string{"team=platform"}},
					{PlacementID: "placement-2", WidgetID: "audit-feed", Column: 6, Row: 0, Width: 6, Height: 2, TitleOverride: "Live Audit"},
				},
			},
		},
		DocumentationComplete: true,
	}

	audit := AuditDashboardBuilder(dashboard)
	if !audit.ReleaseReady() {
		t.Fatalf("expected release-ready dashboard, got %+v", audit)
	}

	rendered := RenderDashboardBuilderReport(dashboard, audit)
	for _, fragment := range []string{
		"# Dashboard Builder",
		"- Release Ready: true",
		"- Duplicate Placement IDs: none",
		"- primary: name=Primary columns=12 placements=2",
		"- placement-2: widget=audit-feed title=Live Audit grid=(6,0) size=6x2 filters=none",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered report, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteDashboardBuilderBundle(outputDir, dashboard, audit)
	if err != nil {
		t.Fatalf("write dashboard builder bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "dashboard-builder.md") {
		t.Fatalf("unexpected dashboard bundle path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read dashboard builder bundle: %v", err)
	}
	if !strings.Contains(string(body), "team=platform") || !strings.Contains(string(body), "Live Audit") {
		t.Fatalf("unexpected dashboard builder bundle content: %s", string(body))
	}
}

func TestBuildEngineeringOverviewFromTasksAndEvents(t *testing.T) {
	base := time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{
			ID:        "task-success",
			Title:     "Ship release",
			State:     domain.TaskSucceeded,
			CreatedAt: base,
			UpdatedAt: base.Add(30 * time.Minute),
			Metadata: map[string]string{
				"run_id":   "run-success",
				"summary":  "release shipped",
				"team":     "platform",
				"priority": "1",
			},
		},
		{
			ID:        "task-approval",
			Title:     "Approve rollout",
			State:     domain.TaskBlocked,
			CreatedAt: base,
			UpdatedAt: base.Add(90 * time.Minute),
			Metadata: map[string]string{
				"run_id":          "run-approval",
				"approval_status": "needs-approval",
				"blocked_reason":  "approval pending for prod rollout",
				"summary":         "awaiting prod approval",
				"team":            "operations",
			},
		},
		{
			ID:        "task-failed",
			Title:     "Fix security regression",
			State:     domain.TaskFailed,
			CreatedAt: base,
			UpdatedAt: base.Add(2 * time.Hour),
			Metadata: map[string]string{
				"run_id":         "run-failed",
				"failure_reason": "security review blocked deploy",
				"team":           "security",
			},
		},
	}
	events := []domain.Event{
		{ID: "evt-approval", Type: domain.EventRunAnnotated, TaskID: "task-approval", RunID: "run-approval", Timestamp: base.Add(90 * time.Minute), Payload: map[string]any{"reason": "approval pending for prod rollout"}},
		{ID: "evt-failed", Type: domain.EventTaskDeadLetter, TaskID: "task-failed", RunID: "run-failed", Timestamp: base.Add(2 * time.Hour), Payload: map[string]any{"reason": "security review blocked deploy"}},
	}

	overview := BuildEngineeringOverview("Engineering Pulse", "2026-W12", "engineering-manager", tasks, events, 60, 3, 5)
	if overview.Permissions.ViewerRole != "engineering-manager" || !overview.Permissions.CanView("funnel") || !overview.Permissions.CanView("activity") {
		t.Fatalf("unexpected permissions: %+v", overview.Permissions)
	}
	if len(overview.KPIs) != 4 {
		t.Fatalf("expected 4 KPIs, got %+v", overview.KPIs)
	}
	if overview.KPIs[0].Value != 33.3 {
		t.Fatalf("expected success rate 33.3, got %+v", overview.KPIs[0])
	}
	if overview.KPIs[1].Value != 1 || overview.KPIs[2].Value != 2 || overview.KPIs[3].Value != 80 {
		t.Fatalf("unexpected operational KPIs: %+v", overview.KPIs)
	}
	if len(overview.Funnel) != 4 || overview.Funnel[0].Count != 0 || overview.Funnel[2].Count != 1 || overview.Funnel[3].Count != 1 {
		t.Fatalf("unexpected funnel: %+v", overview.Funnel)
	}
	if len(overview.Blockers) != 2 || overview.Blockers[0].Owner != "operations" || overview.Blockers[1].Owner != "security" {
		t.Fatalf("unexpected blockers: %+v", overview.Blockers)
	}
	if overview.Blockers[1].Severity != "high" {
		t.Fatalf("expected failed task blocker severity high, got %+v", overview.Blockers[1])
	}
	if len(overview.Activities) != 3 || overview.Activities[0].TaskID != "task-failed" || overview.Activities[0].RunID != "run-failed" {
		t.Fatalf("unexpected activities: %+v", overview.Activities)
	}
}

func TestRenderAndWriteEngineeringOverviewBundle(t *testing.T) {
	overview := EngineeringOverview{
		Name:   "Engineering Pulse",
		Period: "2026-W12",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operations",
			AllowedModules: []string{"kpis", "funnel", "blockers", "activity"},
		},
		KPIs: []EngineeringOverviewKPI{
			{Name: "success-rate", Value: 95, Target: 90, Unit: "%", Direction: "up"},
			{Name: "sla-breaches", Value: 1, Target: 0, Direction: "down"},
		},
		Funnel: []EngineeringFunnelStage{
			{Name: "queued", Count: 2, Share: 20},
			{Name: "completed", Count: 8, Share: 80},
		},
		Blockers: []EngineeringOverviewBlocker{
			{Summary: "approval pending for prod rollout", AffectedRuns: 2, AffectedTasks: []string{"BIG-1", "BIG-2"}, Owner: "operations", Severity: "medium"},
		},
		Activities: []EngineeringActivity{
			{Timestamp: "2026-03-18T09:30:00Z", RunID: "run-1", TaskID: "BIG-1", Status: "blocked", Summary: "approval pending for prod rollout"},
		},
	}

	rendered := RenderEngineeringOverview(overview)
	for _, fragment := range []string{
		"# Engineering Overview",
		"- Viewer Role: operations",
		"- success-rate: value=95.0% target=90.0% healthy=true",
		"- completed: count=8 share=80.0%",
		"- approval pending for prod rollout: severity=medium owner=operations affected_runs=2 tasks=BIG-1, BIG-2",
		"- 2026-03-18T09:30:00Z: run-1 task=BIG-1 status=blocked summary=approval pending for prod rollout",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("expected %q in rendered overview, got %s", fragment, rendered)
		}
	}

	outputDir := t.TempDir()
	path, err := WriteEngineeringOverviewBundle(outputDir, overview)
	if err != nil {
		t.Fatalf("write engineering overview bundle: %v", err)
	}
	if path != filepath.Join(outputDir, "engineering-overview.md") {
		t.Fatalf("unexpected overview path: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read engineering overview bundle: %v", err)
	}
	if !strings.Contains(string(body), "## Activity Modules") || !strings.Contains(string(body), "approval pending for prod rollout") {
		t.Fatalf("unexpected overview bundle content: %s", string(body))
	}
}
