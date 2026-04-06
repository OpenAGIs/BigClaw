package validationbundle

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildContinuationScorecardMatchesCheckedInShape(t *testing.T) {
	report, err := BuildContinuationScorecard(
		repoRoot(t),
		"bigclaw-go/docs/reports/live-validation-index.json",
		"bigclaw-go/docs/reports/live-validation-runs",
		"bigclaw-go/docs/reports/live-validation-summary.json",
		"bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		time.Date(2026, 3, 16, 15, 54, 25, 278091000, time.UTC),
	)
	if err != nil {
		t.Fatalf("BuildContinuationScorecard: %v", err)
	}

	if report["status"] != "local-continuation-scorecard" {
		t.Fatalf("unexpected status: %+v", report["status"])
	}
	summary := report["summary"].(map[string]any)
	if intValue(summary["recent_bundle_count"]) != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	sharedQueue := report["shared_queue_companion"].(map[string]any)
	if sharedQueue["cross_node_completions"] != 99 {
		t.Fatalf("unexpected shared queue companion: %+v", sharedQueue)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
}
