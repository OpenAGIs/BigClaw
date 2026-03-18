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
