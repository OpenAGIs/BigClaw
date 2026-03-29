package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutomationValidationBundleContinuationScorecardWritesGeneratorCommand(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-index.json")
	summaryPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-summary.json")
	sharedQueueReportPath := filepath.Join(repoRoot, "docs", "reports", "multi-node-shared-queue-report.json")
	bundleRoot := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs")
	runID := "20260316T140138Z"
	runSummaryPath := filepath.Join(bundleRoot, runID, "summary.json")

	writeJSONFixture(t, runSummaryPath, map[string]any{
		"run_id":       runID,
		"generated_at": "2026-03-16T14:01:38Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"canonical_report_path":     "docs/reports/multi-node-shared-queue-report.json",
			"canonical_summary_path":    "docs/reports/shared-queue-companion-summary.json",
			"bundle_report_path":        "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json",
			"bundle_summary_path":       "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json",
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
		},
	})
	writeJSONFixture(t, filepath.Join(bundleRoot, "20260315T140138Z", "summary.json"), map[string]any{
		"run_id":       "20260315T140138Z",
		"generated_at": "2026-03-15T14:01:38Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeJSONFixture(t, manifestPath, map[string]any{
		"latest": map[string]any{
			"run_id":       runID,
			"generated_at": "2026-03-16T14:01:38Z",
			"status":       "succeeded",
			"summary_path": "docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		},
		"recent_runs": []map[string]any{
			{
				"run_id":       runID,
				"generated_at": "2026-03-16T14:01:38Z",
				"status":       "succeeded",
				"summary_path": "docs/reports/live-validation-runs/20260316T140138Z/summary.json",
			},
			{
				"run_id":       "20260315T140138Z",
				"generated_at": "2026-03-15T14:01:38Z",
				"status":       "succeeded",
				"summary_path": "docs/reports/live-validation-runs/20260315T140138Z/summary.json",
			},
		},
	})
	writeJSONFixture(t, summaryPath, map[string]any{
		"run_id":       runID,
		"generated_at": "2026-03-16T14:01:38Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"canonical_report_path":     "docs/reports/multi-node-shared-queue-report.json",
			"canonical_summary_path":    "docs/reports/shared-queue-companion-summary.json",
			"bundle_report_path":        "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json",
			"bundle_summary_path":       "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json",
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
		},
	})
	writeJSONFixture(t, sharedQueueReportPath, map[string]any{
		"all_ok":                    true,
		"cross_node_completions":    99,
		"duplicate_completed_tasks": []any{},
		"duplicate_started_tasks":   []any{},
	})

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	report, exitCode, err := automationValidationBundleContinuationScorecard(automationValidationBundleContinuationScorecardOptions{
		Output:                "docs/reports/validation-bundle-continuation-scorecard.json",
		IndexManifestPath:     "docs/reports/live-validation-index.json",
		BundleRootPath:        "docs/reports/live-validation-runs",
		SummaryPath:           "docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "docs/reports/multi-node-shared-queue-report.json",
		Now: func() time.Time {
			return time.Date(2026, 3, 16, 18, 1, 38, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("build scorecard: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}

	evidenceInputs := report["evidence_inputs"].(map[string]any)
	if evidenceInputs["generator_command"] != e2eValidationBundleScorecardGenerator {
		t.Fatalf("unexpected generator command: %+v", evidenceInputs)
	}
	summary := report["summary"].(map[string]any)
	if summary["recent_bundle_count"] != 2 || summary["latest_run_id"] != runID {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestAutomationValidationBundleContinuationPolicyGateWritesGoGeneratorCommand(t *testing.T) {
	repoRoot := t.TempDir()
	scorecardPath := filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-scorecard.json")
	writeJSONFixture(t, scorecardPath, map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           2.5,
			"recent_bundle_count":                               2,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": true,
		},
		"shared_queue_companion": map[string]any{
			"available":              true,
			"cross_node_completions": 99,
		},
	})

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWD) }()

	report, exitCode, err := automationValidationBundleContinuationPolicyGate(automationValidationBundleContinuationPolicyGateOptions{
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		Output:                      "docs/reports/validation-bundle-continuation-policy-gate.json",
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		EnforcementMode:             "hold",
		Now: func() time.Time {
			return time.Date(2026, 3, 16, 18, 1, 38, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("build policy gate: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}

	evidenceInputs := report["evidence_inputs"].(map[string]any)
	if evidenceInputs["generator_command"] != e2eValidationBundlePolicyGateGenerator {
		t.Fatalf("unexpected generator command: %+v", evidenceInputs)
	}
	nextActions := report["next_actions"].([]string)
	if len(nextActions) != 1 || !strings.Contains(nextActions[0], "BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail") {
		t.Fatalf("unexpected next_actions: %+v", nextActions)
	}
}

func writeJSONFixture(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
