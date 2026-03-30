package regression

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationBundleContinuationScorecardBuilderStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "docs", "reports", "live-validation-index.json")
	summaryPath := filepath.Join(tmpDir, "docs", "reports", "live-validation-summary.json")
	sharedQueuePath := filepath.Join(tmpDir, "docs", "reports", "multi-node-shared-queue-report.json")
	bundleRootPath := filepath.Join(tmpDir, "docs", "reports", "live-validation-runs")
	outputPath := filepath.Join(tmpDir, "docs", "reports", "validation-bundle-continuation-scorecard.json")
	run1Path := filepath.Join(tmpDir, "docs", "reports", "live-validation-runs", "r1", "summary.json")
	run2Path := filepath.Join(tmpDir, "docs", "reports", "live-validation-runs", "r2", "summary.json")
	run3Path := filepath.Join(tmpDir, "docs", "reports", "live-validation-runs", "r3", "summary.json")
	for _, path := range []string{manifestPath, summaryPath, sharedQueuePath, run1Path, run2Path, run3Path} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}

	writeFile := func(path, contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeFile(manifestPath, `{
  "latest": {"run_id":"r1","status":"succeeded","generated_at":"2026-03-16T14:01:38Z"},
  "recent_runs": [
    {"summary_path":"`+run1Path+`"},
    {"summary_path":"`+run2Path+`"},
    {"summary_path":"`+run3Path+`"}
  ]
}`)
	writeFile(summaryPath, `{
  "status":"succeeded",
  "local":{"status":"succeeded"},
  "kubernetes":{"status":"succeeded"},
  "ray":{"status":"succeeded"},
  "shared_queue_companion":{
    "available":true,
    "canonical_report_path":"docs/reports/multi-node-shared-queue-report.json",
    "canonical_summary_path":"docs/reports/shared-queue-companion-summary.json",
    "bundle_report_path":"docs/reports/live-validation-runs/r1/multi-node-shared-queue-report.json",
    "bundle_summary_path":"docs/reports/live-validation-runs/r1/shared-queue-companion-summary.json",
    "cross_node_completions":99,
    "duplicate_completed_tasks":0,
    "duplicate_started_tasks":0
  }
}`)
	writeFile(sharedQueuePath, `{
  "all_ok": true,
  "cross_node_completions": 88,
  "duplicate_completed_tasks": [],
  "duplicate_started_tasks": []
}`)
	writeFile(run1Path, `{
  "generated_at":"2026-03-16T14:01:38Z",
  "status":"succeeded",
  "local":{"enabled":true,"status":"succeeded"},
  "kubernetes":{"enabled":true,"status":"succeeded"},
  "ray":{"enabled":true,"status":"succeeded"}
}`)
	writeFile(run2Path, `{
  "generated_at":"2026-03-16T13:31:38Z",
  "status":"succeeded",
  "local":{"enabled":true,"status":"succeeded"},
  "kubernetes":{"enabled":true,"status":"succeeded"},
  "ray":{"enabled":true,"status":"succeeded"}
}`)
	writeFile(run3Path, `{
  "generated_at":"2026-03-16T13:01:38Z",
  "status":"succeeded",
  "local":{"enabled":true,"status":"succeeded"},
  "kubernetes":{"enabled":true,"status":"succeeded"},
  "ray":{"enabled":false}
}`)

	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation-bundle-continuation-scorecard")
	cmd := exec.Command(
		"bash", scriptPath,
		"--index-manifest", manifestPath,
		"--summary", summaryPath,
		"--shared-queue-report", sharedQueuePath,
		"--bundle-root", bundleRootPath,
		"--output", outputPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run continuation scorecard builder: %v\n%s", err, output)
	}

	var report struct {
		Status        string `json:"status"`
		ExecutorLanes []struct {
			Lane                 string `json:"lane"`
			EnabledRuns          int    `json:"enabled_runs"`
			ConsecutiveSuccesses int    `json:"consecutive_successes"`
		} `json:"executor_lanes"`
		SharedQueueCompanion struct {
			Available            bool   `json:"available"`
			Mode                 string `json:"mode"`
			CrossNodeCompletions int    `json:"cross_node_completions"`
			SummaryPath          string `json:"summary_path"`
		} `json:"shared_queue_companion"`
		EvidenceInputs struct {
			GeneratorScript string `json:"generator_script"`
		} `json:"evidence_inputs"`
		Summary struct {
			RecentBundleCount                           int     `json:"recent_bundle_count"`
			LatestRunID                                 string  `json:"latest_run_id"`
			LatestStatus                                string  `json:"latest_status"`
			RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
			AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
			BundleGapMinutes                            float64 `json:"bundle_gap_minutes"`
			BundleRootExists                            bool    `json:"bundle_root_exists"`
		} `json:"summary"`
		ContinuationChecks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Detail string `json:"detail"`
		} `json:"continuation_checks"`
		CurrentCeiling   []string `json:"current_ceiling"`
		NextRuntimeHooks []string `json:"next_runtime_hooks"`
	}
	readJSONFile(t, outputPath, &report)

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("unexpected scorecard status: %+v", report)
	}
	if report.EvidenceInputs.GeneratorScript != "bigclaw-go/scripts/e2e/validation-bundle-continuation-scorecard" {
		t.Fatalf("unexpected generator script: %+v", report.EvidenceInputs)
	}
	if report.Summary.RecentBundleCount != 3 || report.Summary.LatestRunID != "r1" || report.Summary.LatestStatus != "succeeded" || !report.Summary.RecentBundleChainHasNoFailures || !report.Summary.AllExecutorTracksHaveRepeatedRecentCoverage || !report.Summary.BundleRootExists {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Summary.BundleGapMinutes != 30 {
		t.Fatalf("unexpected bundle gap minutes: %+v", report.Summary)
	}
	if !report.SharedQueueCompanion.Available || report.SharedQueueCompanion.Mode != "bundle-companion-summary" || report.SharedQueueCompanion.CrossNodeCompletions != 99 || report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue companion: %+v", report.SharedQueueCompanion)
	}

	laneRuns := map[string]int{}
	for _, lane := range report.ExecutorLanes {
		laneRuns[lane.Lane] = lane.EnabledRuns
	}
	if laneRuns["local"] != 3 || laneRuns["kubernetes"] != 3 || laneRuns["ray"] != 2 {
		t.Fatalf("unexpected lane runs: %+v", laneRuns)
	}

	foundRepeatedCoverage := false
	foundWorkflowBoundary := false
	for _, check := range report.ContinuationChecks {
		if check.Name == "all_executor_tracks_have_repeated_recent_coverage" {
			foundRepeatedCoverage = check.Passed && strings.Contains(check.Detail, "ray")
		}
		if check.Name == "continuation_surface_is_workflow_triggered" {
			foundWorkflowBoundary = check.Passed && strings.Contains(check.Detail, "workflow execution")
		}
	}
	if !foundRepeatedCoverage || !foundWorkflowBoundary {
		t.Fatalf("unexpected continuation checks: %+v", report.ContinuationChecks)
	}
	if len(report.CurrentCeiling) == 0 || len(report.NextRuntimeHooks) == 0 {
		t.Fatalf("expected ceiling and hooks, got %+v %+v", report.CurrentCeiling, report.NextRuntimeHooks)
	}
}
