package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAutomationSubscriberTakeoverFaultMatrixBuildsReport(t *testing.T) {
	root := t.TempDir()
	report, err := automationSubscriberTakeoverFaultMatrix(automationSubscriberTakeoverFaultMatrixOptions{
		GoRoot:     root,
		OutputPath: "tmp/takeover-report.json",
		Now:        func() time.Time { return time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("build takeover matrix: %v", err)
	}
	if report["status"] != "local-executable" {
		t.Fatalf("unexpected status: %+v", report)
	}
	summary, _ := report["summary"].(map[string]any)
	if asInt(summary["scenario_count"]) != 3 || asInt(summary["stale_write_rejections"]) != 2 || asInt(summary["duplicate_delivery_count"]) != 4 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	currentPrimitives, _ := report["current_primitives"].(map[string]any)
	takeoverHarness, _ := currentPrimitives["takeover_harness"].([]any)
	if len(takeoverHarness) != 2 || asString(takeoverHarness[0]) != "cmd/bigclawctl/automation_e2e_takeover_matrix_command.go" {
		t.Fatalf("unexpected takeover harness ref: %+v", takeoverHarness)
	}
	if _, err := os.Stat(filepath.Join(root, "tmp/takeover-report.json")); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
}
