package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutomationCrossProcessCoordinationSurfaceBuildsReport(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/reports/multi-node-shared-queue-report.json", `{"count":200,"cross_node_completions":99,"duplicate_completed_tasks":[],"duplicate_started_tasks":[]}`)
	write("docs/reports/multi-subscriber-takeover-validation-report.json", `{"summary":{"scenario_count":3,"passing_scenarios":3,"duplicate_delivery_count":0,"stale_write_rejections":2}}`)
	write("docs/reports/live-multi-node-subscriber-takeover-report.json", `{"summary":{"scenario_count":2,"passing_scenarios":2,"stale_write_rejections":1}}`)

	report, err := automationCrossProcessCoordinationSurface(automationCrossProcessCoordinationSurfaceOptions{
		GoRoot:                 root,
		MultiNodeReportPath:    "docs/reports/multi-node-shared-queue-report.json",
		TakeoverReportPath:     "docs/reports/multi-subscriber-takeover-validation-report.json",
		LiveTakeoverReportPath: "docs/reports/live-multi-node-subscriber-takeover-report.json",
		OutputPath:             "docs/reports/cross-process-coordination-capability-surface.json",
	})
	if err != nil {
		t.Fatalf("build coordination surface: %v", err)
	}
	if report["status"] != "local-capability-surface" {
		t.Fatalf("unexpected status: %+v", report)
	}
	summary, _ := report["summary"].(map[string]any)
	if summary["shared_queue_cross_node_completions"] != float64(99) && summary["shared_queue_cross_node_completions"] != 99 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	capabilities, _ := report["capabilities"].([]any)
	if len(capabilities) != 7 {
		t.Fatalf("unexpected capability count: %+v", capabilities)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/reports/cross-process-coordination-capability-surface.json")); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
}
