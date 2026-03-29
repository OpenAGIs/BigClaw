package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidationBundleContinuationScorecardBuildReport(t *testing.T) {
	repoRoot := t.TempDir()
	writeContinuationScorecardRepoFixture(t, repoRoot, defaultContinuationScorecardRepoFixture())

	report, err := buildContinuationScorecardReport(repoRoot, continuationScorecardOptions{
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		GeneratedAt:           time.Date(2026, 3, 16, 15, 54, 49, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationScorecardReport returned error: %v", err)
	}

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("status = %q, want local-continuation-scorecard", report.Status)
	}
	if report.EvidenceInputs["generator_script"] != continuationScorecardScriptPath {
		t.Fatalf("generator_script = %v, want %s", report.EvidenceInputs["generator_script"], continuationScorecardScriptPath)
	}
	if report.Summary["recent_bundle_count"] != float64(3) && report.Summary["recent_bundle_count"] != 3 {
		t.Fatalf("recent_bundle_count = %v, want 3", report.Summary["recent_bundle_count"])
	}
	if report.Summary["latest_run_id"] != "20260316T140138Z" {
		t.Fatalf("latest_run_id = %v, want 20260316T140138Z", report.Summary["latest_run_id"])
	}
	if report.SharedQueueCompanion["mode"] != "bundle-companion-summary" {
		t.Fatalf("shared_queue_companion.mode = %v, want bundle-companion-summary", report.SharedQueueCompanion["mode"])
	}
	if report.SharedQueueCompanion["cross_node_completions"] != float64(99) && report.SharedQueueCompanion["cross_node_completions"] != 99 {
		t.Fatalf("cross_node_completions = %v, want 99", report.SharedQueueCompanion["cross_node_completions"])
	}
	if len(report.ExecutorLanes) != 3 {
		t.Fatalf("executor lane count = %d, want 3", len(report.ExecutorLanes))
	}
	if report.ExecutorLanes[2].Lane != "ray" || report.ExecutorLanes[2].EnabledRuns != 2 || report.ExecutorLanes[2].ConsecutiveSuccesses != 2 {
		t.Fatalf("unexpected ray lane: %+v", report.ExecutorLanes[2])
	}
	repeatedCoverage := false
	for _, check := range report.ContinuationChecks {
		if check.Name == "all_executor_tracks_have_repeated_recent_coverage" && check.Passed && check.Detail == "enabled_runs_by_lane={'kubernetes': 3, 'local': 3, 'ray': 2}" {
			repeatedCoverage = true
		}
	}
	if !repeatedCoverage {
		t.Fatalf("expected repeated coverage check, got %+v", report.ContinuationChecks)
	}
}

func TestValidationBundleContinuationScorecardAddsCoverageCeilingWhenLaneHistoryIsIncomplete(t *testing.T) {
	repoRoot := t.TempDir()
	fixture := defaultContinuationScorecardRepoFixture()
	fixture.RecentRuns[1].RayEnabled = false
	fixture.RecentRuns[1].RayStatus = ""
	writeContinuationScorecardRepoFixture(t, repoRoot, fixture)

	report, err := buildContinuationScorecardReport(repoRoot, continuationScorecardOptions{
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		GeneratedAt:           time.Date(2026, 3, 16, 15, 54, 49, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationScorecardReport returned error: %v", err)
	}

	found := false
	for _, item := range report.CurrentCeiling {
		if item == "not every executor lane is enabled across every indexed bundle in the current recent window" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected incomplete-lane ceiling, got %+v", report.CurrentCeiling)
	}
}

func TestValidationBundleContinuationScorecardWritesReport(t *testing.T) {
	repoRoot := t.TempDir()
	writeContinuationScorecardRepoFixture(t, repoRoot, defaultContinuationScorecardRepoFixture())

	report, err := buildContinuationScorecardReport(repoRoot, continuationScorecardOptions{
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		GeneratedAt:           time.Date(2026, 3, 16, 15, 54, 49, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationScorecardReport returned error: %v", err)
	}
	if err := writeContinuationScorecardReport(repoRoot, "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", report); err != nil {
		t.Fatalf("writeContinuationScorecardReport returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-scorecard.json")); err != nil {
		t.Fatalf("written report missing: %v", err)
	}
}

type continuationRunFixture struct {
	RunID             string
	GeneratedAt       string
	Status            string
	LocalEnabled      bool
	LocalStatus       string
	KubernetesEnabled bool
	KubernetesStatus  string
	RayEnabled        bool
	RayStatus         string
}

type continuationScorecardRepoFixture struct {
	LatestRunID   string
	LatestSummary continuationRunFixture
	RecentRuns    []continuationRunFixture
}

func defaultContinuationScorecardRepoFixture() continuationScorecardRepoFixture {
	return continuationScorecardRepoFixture{
		LatestRunID: "20260316T140138Z",
		LatestSummary: continuationRunFixture{
			RunID:             "20260316T140138Z",
			GeneratedAt:       "2026-03-17T04:32:13.251910+00:00",
			Status:            "succeeded",
			LocalEnabled:      true,
			LocalStatus:       "succeeded",
			KubernetesEnabled: true,
			KubernetesStatus:  "succeeded",
			RayEnabled:        true,
			RayStatus:         "succeeded",
		},
		RecentRuns: []continuationRunFixture{
			{
				RunID:             "20260316T140138Z",
				GeneratedAt:       "2026-03-16T15:54:13Z",
				Status:            "succeeded",
				LocalEnabled:      true,
				LocalStatus:       "succeeded",
				KubernetesEnabled: true,
				KubernetesStatus:  "succeeded",
				RayEnabled:        true,
				RayStatus:         "succeeded",
			},
			{
				RunID:             "20260314T164647Z",
				GeneratedAt:       "2026-03-14T16:47:13Z",
				Status:            "succeeded",
				LocalEnabled:      true,
				LocalStatus:       "succeeded",
				KubernetesEnabled: true,
				KubernetesStatus:  "succeeded",
				RayEnabled:        true,
				RayStatus:         "succeeded",
			},
			{
				RunID:             "20260314T163430Z",
				GeneratedAt:       "2026-03-14T16:34:30Z",
				Status:            "succeeded",
				LocalEnabled:      true,
				LocalStatus:       "succeeded",
				KubernetesEnabled: true,
				KubernetesStatus:  "succeeded",
				RayEnabled:        false,
				RayStatus:         "",
			},
		},
	}
}

func writeContinuationScorecardRepoFixture(t *testing.T, repoRoot string, fixture continuationScorecardRepoFixture) {
	t.Helper()
	reportsDir := filepath.Join(repoRoot, "bigclaw-go", "docs", "reports")
	if err := os.MkdirAll(filepath.Join(reportsDir, "live-validation-runs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	writeFixtureFile(t, filepath.Join(reportsDir, "live-validation-index.json"), buildManifestJSON(fixture))
	writeFixtureFile(t, filepath.Join(reportsDir, "live-validation-summary.json"), buildRunSummaryJSON(fixture.LatestSummary, true))
	writeFixtureFile(t, filepath.Join(reportsDir, "multi-node-shared-queue-report.json"), `{
  "all_ok": true,
  "cross_node_completions": 99,
  "duplicate_completed_tasks": [],
  "duplicate_started_tasks": []
}
`)

	for _, run := range fixture.RecentRuns {
		runDir := filepath.Join(reportsDir, "live-validation-runs", run.RunID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatalf("MkdirAll run dir: %v", err)
		}
		writeFixtureFile(t, filepath.Join(runDir, "summary.json"), buildRunSummaryJSON(run, false))
	}
}

func buildManifestJSON(fixture continuationScorecardRepoFixture) string {
	return fmt.Sprintf(`{
  "latest": {
    "run_id": %q,
    "generated_at": %q,
    "status": %q
  },
  "recent_runs": [
    {
      "summary_path": "docs/reports/live-validation-runs/%s/summary.json"
    },
    {
      "summary_path": "docs/reports/live-validation-runs/%s/summary.json"
    },
    {
      "summary_path": "docs/reports/live-validation-runs/%s/summary.json"
    }
  ]
}
`, fixture.LatestRunID, fixture.LatestSummary.GeneratedAt, fixture.LatestSummary.Status, fixture.RecentRuns[0].RunID, fixture.RecentRuns[1].RunID, fixture.RecentRuns[2].RunID)
}

func buildRunSummaryJSON(run continuationRunFixture, includeSharedQueue bool) string {
	sharedQueue := ""
	if includeSharedQueue {
		sharedQueue = `,
  "shared_queue_companion": {
    "available": true,
    "canonical_report_path": "docs/reports/multi-node-shared-queue-report.json",
    "canonical_summary_path": "docs/reports/shared-queue-companion-summary.json",
    "bundle_report_path": "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json",
    "bundle_summary_path": "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json",
    "cross_node_completions": 99,
    "duplicate_completed_tasks": 0,
    "duplicate_started_tasks": 0
  }`
	}
	return fmt.Sprintf(`{
  "run_id": %q,
  "generated_at": %q,
  "status": %q,
  "local": {
    "enabled": %t,
    "status": %q
  },
  "kubernetes": {
    "enabled": %t,
    "status": %q
  },
  "ray": {
    "enabled": %t,
    "status": %q
  }%s
}
`, run.RunID, run.GeneratedAt, run.Status, run.LocalEnabled, run.LocalStatus, run.KubernetesEnabled, run.KubernetesStatus, run.RayEnabled, run.RayStatus, sharedQueue)
}

func writeFixtureFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll file dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
