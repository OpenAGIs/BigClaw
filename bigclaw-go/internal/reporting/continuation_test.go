package reporting

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildValidationBundleContinuationScorecard(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-index.json"), map[string]any{
		"latest": map[string]any{
			"run_id":       "20260316T140138Z",
			"status":       "succeeded",
			"generated_at": "2026-03-16T14:48:42.581505Z",
		},
		"recent_runs": []map[string]any{
			{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json"},
			{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json"},
			{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json"},
		},
	})
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-summary.json"), map[string]any{
		"status":     "succeeded",
		"local":      map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes": map[string]any{"enabled": true, "status": "succeeded"},
		"ray":        map[string]any{"enabled": true, "status": "succeeded"},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"canonical_report_path":     "docs/reports/multi-node-shared-queue-report.json",
			"canonical_summary_path":    "docs/reports/shared-queue-companion-summary.json",
			"bundle_report_path":        "docs/reports/live-validation-runs/20260316T140138Z/multi-node-shared-queue-report.json",
			"bundle_summary_path":       "docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json",
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "bundle-companion-summary",
		},
	})
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"), map[string]any{
		"all_ok":                    true,
		"cross_node_completions":    99,
		"duplicate_completed_tasks": []any{},
		"duplicate_started_tasks":   []any{},
	})
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json"), map[string]any{
		"generated_at": "2026-03-16T14:48:42.581505Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json"), map[string]any{
		"generated_at": "2026-03-14T16:46:47Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json"), map[string]any{
		"generated_at": "2026-03-14T16:34:30Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": false, "status": "succeeded"},
	})

	report, err := BuildValidationBundleContinuationScorecard(repoRoot, ContinuationScorecardOptions{
		Now: time.Date(2026, 3, 16, 15, 54, 25, 278091000, time.UTC),
	})
	if err != nil {
		t.Fatalf("build scorecard: %v", err)
	}
	if report.EvidenceInputs.GeneratorScript != ValidationBundleContinuationScorecardGenerator {
		t.Fatalf("unexpected generator script: %s", report.EvidenceInputs.GeneratorScript)
	}
	if report.Summary.RecentBundleCount != 3 {
		t.Fatalf("unexpected recent bundle count: %d", report.Summary.RecentBundleCount)
	}
	if !report.Summary.LatestAllExecutorTracksSucceeded {
		t.Fatalf("expected latest bundle success")
	}
	if !report.SharedQueueCompanion.Available || report.SharedQueueCompanion.Mode != "bundle-companion-summary" {
		t.Fatalf("unexpected shared queue companion: %+v", report.SharedQueueCompanion)
	}
	if got, want := report.ExecutorLanes[2].EnabledRuns, 2; got != want {
		t.Fatalf("unexpected ray enabled runs: got %d want %d", got, want)
	}
}

func TestBuildValidationBundleContinuationPolicyGate(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json"), map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           96.0,
			"recent_bundle_count":                               1,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": false,
		},
		"shared_queue_companion": map[string]any{
			"available":              false,
			"cross_node_completions": 12,
		},
	})

	report, err := BuildValidationBundleContinuationPolicyGate(repoRoot, ContinuationPolicyGateOptions{
		RequireRepeatedLaneCoverage: true,
		Now:                         time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}
	if report.Status != "policy-hold" || report.Recommendation != "hold" {
		t.Fatalf("unexpected gate status: %+v", report)
	}
	if report.Enforcement.Mode != "hold" || report.Enforcement.ExitCode != 2 {
		t.Fatalf("unexpected enforcement: %+v", report.Enforcement)
	}
	if report.EvidenceInputs.GeneratorScript != ValidationBundleContinuationPolicyGateGenerator {
		t.Fatalf("unexpected generator script: %s", report.EvidenceInputs.GeneratorScript)
	}
	if len(report.FailingChecks) != 4 {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
}

func writeFixtureJSON(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := WriteJSON(path, payload); err != nil {
		t.Fatalf("write fixture json: %v", err)
	}
}
