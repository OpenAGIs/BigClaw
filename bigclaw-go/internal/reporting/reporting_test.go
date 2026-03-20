package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildRendersRichWeeklyMarkdownSections(t *testing.T) {
	base := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{
			ID:          "task-alpha-1",
			TraceID:     "trace-alpha-1",
			Title:       "Ship alpha",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 900,
			Metadata: map[string]string{
				"team":             "platform",
				"plan":             "premium",
				"regression_count": "2",
			},
			UpdatedAt: base.Add(24 * time.Hour),
		},
		{
			ID:          "task-alpha-2",
			TraceID:     "trace-alpha-2",
			Title:       "Fix blocker",
			State:       domain.TaskBlocked,
			BudgetCents: 300,
			Metadata: map[string]string{
				"team": "platform",
			},
			UpdatedAt: base.Add(48 * time.Hour),
		},
		{
			ID:          "task-growth-1",
			TraceID:     "trace-growth-1",
			Title:       "Launch growth test",
			State:       domain.TaskSucceeded,
			BudgetCents: 200,
			Metadata: map[string]string{
				"team": "growth",
			},
			UpdatedAt: base.Add(72 * time.Hour),
		},
	}
	events := []domain.Event{
		{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-alpha-2", TraceID: "trace-alpha-2", Timestamp: base.Add(48 * time.Hour)},
		{ID: "evt-2", Type: domain.EventControlPaused, TaskID: "task-alpha-2", TraceID: "trace-alpha-2", Timestamp: base.Add(49 * time.Hour)},
	}

	weekly := Build(tasks, events, base, base.Add(7*24*time.Hour))

	if weekly.Summary.TotalRuns != 3 || weekly.Summary.HighRiskRuns != 1 || weekly.Summary.PremiumRuns != 1 {
		t.Fatalf("unexpected weekly summary: %+v", weekly.Summary)
	}
	if len(weekly.TeamBreakdown) != 2 || weekly.TeamBreakdown[0].Key != "platform" {
		t.Fatalf("unexpected team breakdown ordering: %+v", weekly.TeamBreakdown)
	}
	if len(weekly.MetricSpec.Definitions) != 7 || len(weekly.MetricSpec.Values) != 7 {
		t.Fatalf("expected metric spec to be populated, got %+v", weekly.MetricSpec)
	}
	if weekly.MetricSpec.Values[0].MetricID != "throughput" || weekly.MetricSpec.Values[0].DisplayValue != "66.7%" {
		t.Fatalf("unexpected first metric value: %+v", weekly.MetricSpec.Values[0])
	}
	if !strings.Contains(weekly.Markdown, "## Highlights") || !strings.Contains(weekly.Markdown, "## Team Breakdown") || !strings.Contains(weekly.Markdown, "## Metric Spec") {
		t.Fatalf("expected richer markdown sections, got %s", weekly.Markdown)
	}
	if !strings.Contains(weekly.Markdown, "- High-risk runs: 1") || !strings.Contains(weekly.Markdown, "- Premium runs: 1") {
		t.Fatalf("expected risk and premium summary lines, got %s", weekly.Markdown)
	}
	if !strings.Contains(weekly.Markdown, "- Throughput: 66.7%") || !strings.Contains(weekly.Markdown, "- Average Budget: 466.7 cents") {
		t.Fatalf("expected metric spec values in markdown, got %s", weekly.Markdown)
	}
	if !strings.Contains(weekly.Markdown, "- platform: total=2 completed=1 blocked=1 interventions=2 budget_cents=1200") {
		t.Fatalf("expected platform breakdown in markdown, got %s", weekly.Markdown)
	}
	if !strings.Contains(weekly.Markdown, "Top team by throughput: platform.") {
		t.Fatalf("expected highlight in markdown, got %s", weekly.Markdown)
	}
}

func TestWriteWeeklyBundleEmitsMarkdownArtifacts(t *testing.T) {
	base := time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
	weekly := Build([]domain.Task{
		{
			ID:          "task-alpha-1",
			TraceID:     "trace-alpha-1",
			Title:       "Ship alpha",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 900,
			Metadata: map[string]string{
				"team":             "platform",
				"plan":             "premium",
				"regression_count": "1",
			},
			UpdatedAt: base.Add(24 * time.Hour),
		},
	}, nil, base, base.Add(7*24*time.Hour))

	rootDir := t.TempDir()
	artifacts, err := WriteWeeklyBundle(rootDir, weekly)
	if err != nil {
		t.Fatalf("write weekly bundle: %v", err)
	}
	if artifacts.RootDir != rootDir {
		t.Fatalf("expected root dir %q, got %q", rootDir, artifacts.RootDir)
	}
	if artifacts.WeeklyReportPath != filepath.Join(rootDir, "weekly-operations.md") {
		t.Fatalf("unexpected weekly report path: %+v", artifacts)
	}
	if artifacts.MetricSpecPath != filepath.Join(rootDir, "weekly-metric-spec.md") {
		t.Fatalf("unexpected metric spec path: %+v", artifacts)
	}
	reportBody, err := os.ReadFile(artifacts.WeeklyReportPath)
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	metricBody, err := os.ReadFile(artifacts.MetricSpecPath)
	if err != nil {
		t.Fatalf("read metric spec: %v", err)
	}
	if !strings.Contains(string(reportBody), "# BigClaw Weekly Ops Report") || !strings.Contains(string(reportBody), "## Metric Spec") {
		t.Fatalf("unexpected weekly report body: %s", string(reportBody))
	}
	if !strings.Contains(string(metricBody), "# BigClaw Weekly Metric Spec") || !strings.Contains(string(metricBody), "- Throughput: 100.0%") {
		t.Fatalf("unexpected metric spec body: %s", string(metricBody))
	}
}
