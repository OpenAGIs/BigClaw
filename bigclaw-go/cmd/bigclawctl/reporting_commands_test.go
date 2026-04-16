package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestRunReportingWeeklyJSONOutputAndArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	tasksPath := filepath.Join(repoRoot, "fixtures", "tasks.json")
	eventsPath := filepath.Join(repoRoot, "fixtures", "events.json")
	outputDir := filepath.Join(repoRoot, "artifacts")
	if err := os.MkdirAll(filepath.Dir(tasksPath), 0o755); err != nil {
		t.Fatalf("mkdir fixtures: %v", err)
	}

	tasks := []domain.Task{
		{
			ID:          "task-1",
			Title:       "Ship weekly bundle",
			State:       domain.TaskSucceeded,
			RiskLevel:   domain.RiskHigh,
			BudgetCents: 1200,
			Metadata: map[string]string{
				"team":             "platform",
				"project":          "alpha",
				"plan":             "premium",
				"regression_count": "2",
			},
			UpdatedAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "task-2",
			Title:       "Review blocked rollout",
			State:       domain.TaskBlocked,
			BudgetCents: 300,
			Metadata: map[string]string{
				"team":    "platform",
				"project": "alpha",
			},
			UpdatedAt: time.Date(2026, 3, 11, 9, 30, 0, 0, time.UTC),
		},
	}
	taskBody, err := json.Marshal(tasks)
	if err != nil {
		t.Fatalf("marshal tasks: %v", err)
	}
	if err := os.WriteFile(tasksPath, taskBody, 0o644); err != nil {
		t.Fatalf("write tasks: %v", err)
	}

	eventsPayload := map[string]any{
		"events": []domain.Event{
			{ID: "evt-1", Type: domain.EventRunTakeover, TaskID: "task-2"},
		},
	}
	eventBody, err := json.Marshal(eventsPayload)
	if err != nil {
		t.Fatalf("marshal events: %v", err)
	}
	if err := os.WriteFile(eventsPath, eventBody, 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return runReporting([]string{
			"weekly",
			"--repo", repoRoot,
			"--tasks", "fixtures/tasks.json",
			"--events", "fixtures/events.json",
			"--output-dir", "artifacts",
			"--week-start", "2026-03-09",
			"--week-end", "2026-03-16",
			"--timezone", "America/Los_Angeles",
			"--sla-minutes", "90",
			"--json",
		})
	})
	if err != nil {
		t.Fatalf("run reporting weekly: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode output: %v (%s)", err, string(output))
	}
	if payload["status"] != "ok" || payload["task_count"] != float64(2) || payload["event_count"] != float64(1) {
		t.Fatalf("unexpected reporting payload: %+v", payload)
	}

	reportSurface, _ := payload["report_surface"].(map[string]any)
	if reportSurface["cli"] != "bigclawctl reporting weekly" || reportSurface["api"] != "/v2/reports/weekly" || reportSurface["export_api"] != "/v2/reports/weekly/export" {
		t.Fatalf("unexpected report surface payload: %+v", reportSurface)
	}

	metricSpec, _ := payload["metric_spec"].(map[string]any)
	if metricSpec["timezone"] != "America/Los_Angeles" || metricSpec["definition_count"] != float64(7) || metricSpec["value_count"] != float64(7) {
		t.Fatalf("unexpected metric spec payload: %+v", metricSpec)
	}

	artifacts, _ := payload["artifacts"].(map[string]any)
	for _, key := range []string{"weekly_report_path", "dashboard_path", "metric_spec_path"} {
		raw, ok := artifacts[key].(string)
		if !ok || raw == "" {
			t.Fatalf("expected artifact path for %s, got %+v", key, artifacts)
		}
		if _, err := os.Stat(raw); err != nil {
			t.Fatalf("expected artifact %s to exist at %s: %v", key, raw, err)
		}
	}

	reportBody, err := os.ReadFile(filepath.Join(outputDir, "weekly-operations.md"))
	if err != nil {
		t.Fatalf("read weekly report: %v", err)
	}
	if !strings.Contains(string(reportBody), "# BigClaw Weekly Ops Report") || !strings.Contains(string(reportBody), "- Regressions: 2") {
		t.Fatalf("unexpected weekly report body: %s", string(reportBody))
	}
}

func TestLoadReportingJSONArraySupportsWrappedPayload(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "events.json")
	body := `{"events":[{"id":"evt-1","type":"run.takeover","task_id":"task-1","timestamp":"2026-03-09T00:00:00Z"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	events, err := loadReportingEvents(path)
	if err != nil {
		t.Fatalf("load reporting events: %v", err)
	}
	if len(events) != 1 || events[0].ID != "evt-1" || events[0].Type != domain.EventRunTakeover {
		t.Fatalf("unexpected wrapped events payload: %+v", events)
	}
}
