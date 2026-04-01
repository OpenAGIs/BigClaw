package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/regression"
)

func TestOperationsReportingContractBundlesGoOwnedSurfaces(t *testing.T) {
	start := time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	tasks := []domain.Task{
		{
			ID:          "task-1",
			Title:       "Ship release",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 425,
			CreatedAt:   start,
			UpdatedAt:   start.Add(30 * time.Minute),
			Metadata: map[string]string{
				"team":             "platform",
				"run_id":           "run-1",
				"summary":          "release shipped",
				"regression_count": "1",
			},
		},
		{
			ID:          "task-2",
			Title:       "Approve rollout",
			State:       domain.TaskBlocked,
			RiskLevel:   domain.RiskMedium,
			BudgetCents: 750,
			CreatedAt:   start.Add(time.Hour),
			UpdatedAt:   start.Add(150 * time.Minute),
			Metadata: map[string]string{
				"team":            "operations",
				"run_id":          "run-2",
				"summary":         "awaiting prod approval",
				"approval_status": "needs-approval",
				"blocked_reason":  "approval pending for prod rollout",
			},
		},
	}
	events := []domain.Event{
		{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-2", RunID: "run-2", Timestamp: start.Add(2 * time.Hour), Payload: map[string]any{"reason": "approval pending for prod rollout"}},
	}

	spec := BuildOperationsMetricSpec(tasks, events, start, end, "UTC", 60)
	if len(spec.Definitions) != 7 || len(spec.Values) != 7 {
		t.Fatalf("expected legacy metric surface, got %+v", spec)
	}
	renderedSpec := RenderOperationsMetricSpec(spec)
	for _, fragment := range []string{"# Operations Metric Spec", "### Runs In Window", "Budget Spend: value=$11.75"} {
		if !strings.Contains(renderedSpec, fragment) {
			t.Fatalf("expected %q in rendered metric spec, got %s", fragment, renderedSpec)
		}
	}

	weekly := Build(tasks, events, start, end)
	if weekly.Summary.TotalRuns != 2 || weekly.Summary.BlockedRuns != 1 || weekly.Summary.RegressionFindings != 1 {
		t.Fatalf("unexpected weekly summary: %+v", weekly.Summary)
	}

	queueCenter := BuildQueueControlCenter(tasks)
	regressionCenter := regression.Center{
		Summary: regression.Summary{
			TotalRegressions:    1,
			AffectedTasks:       1,
			CriticalRegressions: 0,
			ReworkEvents:        0,
			TopSource:           "approval pending for prod rollout",
			TopWorkflow:         "release",
		},
		Findings: []regression.Finding{
			{TaskID: "task-2", Workflow: "release", Team: "operations", Severity: "medium", RegressionCount: 1, Summary: "approval pending for prod rollout"},
		},
	}
	versionCenter := BuildPolicyPromptVersionCenter("Policy Center", start.Add(3*time.Hour), []VersionedArtifact{
		{
			ArtifactType: "prompt",
			ArtifactID:   "triage-summary",
			Version:      "v1",
			UpdatedAt:    "2026-03-18T09:00:00Z",
			Author:       "alice",
			Summary:      "initial prompt",
			Content:      "summarize blockers\n",
		},
		{
			ArtifactType: "prompt",
			ArtifactID:   "triage-summary",
			Version:      "v2",
			UpdatedAt:    "2026-03-18T11:00:00Z",
			Author:       "bob",
			Summary:      "include owners",
			Content:      "summarize blockers\ninclude owners\n",
			ChangeTicket: "BIG-GO-1040",
		},
	}, 6)

	rootDir := t.TempDir()
	artifacts, err := WriteWeeklyOperationsBundleWithCenters(rootDir, weekly, &spec, "Regression Analysis Center", &regressionCenter, &queueCenter, &versionCenter)
	if err != nil {
		t.Fatalf("write weekly operations bundle with centers: %v", err)
	}
	for _, path := range []string{
		artifacts.WeeklyReportPath,
		artifacts.DashboardPath,
		artifacts.MetricSpecPath,
		artifacts.QueueControlPath,
		artifacts.RegressionCenterPath,
		artifacts.VersionCenterPath,
	} {
		if path == "" {
			t.Fatalf("expected bundle path, got %+v", artifacts)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected bundle artifact %s: %v", path, err)
		}
	}

	weeklyBody, err := os.ReadFile(artifacts.WeeklyReportPath)
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	if !strings.Contains(string(weeklyBody), "Review regression hotspots and route them through the regression center.") {
		t.Fatalf("expected regression action in weekly report, got %s", string(weeklyBody))
	}

	queueBody, err := os.ReadFile(artifacts.QueueControlPath)
	if err != nil {
		t.Fatalf("read queue center: %v", err)
	}
	if !strings.Contains(string(queueBody), "task-2") {
		t.Fatalf("expected blocked task in queue center, got %s", string(queueBody))
	}

	versionBody, err := os.ReadFile(artifacts.VersionCenterPath)
	if err != nil {
		t.Fatalf("read version center: %v", err)
	}
	if !strings.Contains(string(versionBody), "triage-summary") || !strings.Contains(string(versionBody), "include owners") {
		t.Fatalf("expected version center history, got %s", string(versionBody))
	}
}

func TestOperationsReportingContractCoversDashboardAndOverviewArtifacts(t *testing.T) {
	dashboard := DashboardBuilder{
		Name:   "Ops Console",
		Period: "2026-W12",
		Owner:  "operations",
		Permissions: EngineeringOverviewPermission{
			ViewerRole:     "operations",
			AllowedModules: []string{"operations", "audit"},
		},
		Widgets: []DashboardWidgetSpec{
			{WidgetID: "ops-summary", Title: "Ops Summary", Module: "operations", DataSource: "/v2/reports/weekly", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
			{WidgetID: "audit-feed", Title: "Audit Feed", Module: "audit", DataSource: "/v2/audit", DefaultWidth: 4, DefaultHeight: 3, MinWidth: 2, MaxWidth: 12},
		},
		Layouts: []DashboardLayout{
			{
				LayoutID: "desktop",
				Name:     "Desktop",
				Columns:  12,
				Placements: []DashboardWidgetPlacement{
					{PlacementID: "ops", WidgetID: "ops-summary", Column: 0, Row: 0, Width: 6, Height: 2, Filters: []string{"team=engineering"}},
					{PlacementID: "audit", WidgetID: "audit-feed", Column: 6, Row: 0, Width: 6, Height: 2, TitleOverride: "Live Audit"},
				},
			},
		},
		DocumentationComplete: true,
	}
	audit := AuditDashboardBuilder(dashboard)
	if !audit.ReleaseReady() {
		t.Fatalf("expected release-ready dashboard, got %+v", audit)
	}
	report := RenderDashboardBuilderReport(dashboard, audit)
	for _, fragment := range []string{"# Dashboard Builder", "- Release Ready: true", "team=engineering", "Live Audit"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in dashboard report, got %s", fragment, report)
		}
	}
	outputDir := t.TempDir()
	reportPath, err := WriteDashboardBuilderBundle(outputDir, dashboard, audit)
	if err != nil {
		t.Fatalf("write dashboard builder bundle: %v", err)
	}
	if reportPath != filepath.Join(outputDir, "dashboard-builder.md") {
		t.Fatalf("unexpected dashboard bundle path: %s", reportPath)
	}

	base := time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{
			ID:        "task-success",
			Title:     "Ship release",
			State:     domain.TaskSucceeded,
			CreatedAt: base,
			UpdatedAt: base.Add(30 * time.Minute),
			Metadata: map[string]string{
				"run_id":  "run-success",
				"summary": "release shipped",
				"team":    "platform",
			},
		},
		{
			ID:        "task-blocked",
			Title:     "Approve rollout",
			State:     domain.TaskBlocked,
			CreatedAt: base,
			UpdatedAt: base.Add(90 * time.Minute),
			Metadata: map[string]string{
				"run_id":          "run-blocked",
				"summary":         "awaiting prod approval",
				"team":            "operations",
				"approval_status": "needs-approval",
				"blocked_reason":  "approval pending for prod rollout",
			},
		},
	}
	events := []domain.Event{
		{ID: "evt-blocked", Type: domain.EventRunAnnotated, TaskID: "task-blocked", RunID: "run-blocked", Timestamp: base.Add(90 * time.Minute), Payload: map[string]any{"reason": "approval pending for prod rollout"}},
	}
	overview := BuildEngineeringOverview("Engineering Pulse", "2026-W12", "operations", tasks, events, 60, 3, 5)
	if len(overview.KPIs) != 4 || len(overview.Blockers) != 1 || len(overview.Activities) == 0 {
		t.Fatalf("unexpected engineering overview: %+v", overview)
	}
	renderedOverview := RenderEngineeringOverview(overview)
	for _, fragment := range []string{"# Engineering Overview", "- approval-queue-depth: value=1.0", "approval pending for prod rollout", "run-blocked"} {
		if !strings.Contains(renderedOverview, fragment) {
			t.Fatalf("expected %q in engineering overview, got %s", fragment, renderedOverview)
		}
	}
	overviewPath, err := WriteEngineeringOverviewBundle(outputDir, overview)
	if err != nil {
		t.Fatalf("write engineering overview bundle: %v", err)
	}
	if overviewPath != filepath.Join(outputDir, "engineering-overview.md") {
		t.Fatalf("unexpected engineering overview path: %s", overviewPath)
	}
}
