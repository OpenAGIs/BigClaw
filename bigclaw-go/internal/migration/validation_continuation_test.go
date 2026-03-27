package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildValidationContinuationScorecard(t *testing.T) {
	repoRoot := t.TempDir()
	goRoot := filepath.Join(repoRoot, "bigclaw-go")
	reportsRoot := filepath.Join(goRoot, "docs", "reports")
	if err := os.MkdirAll(filepath.Join(reportsRoot, "live-validation-runs", "run-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(reportsRoot, "live-validation-runs", "run-2"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeJSONFixture(t, filepath.Join(reportsRoot, "live-validation-index.json"), map[string]any{
		"latest": map[string]any{
			"run_id":       "run-1",
			"generated_at": "2026-03-16T14:01:38Z",
			"status":       "succeeded",
		},
		"recent_runs": []map[string]any{
			{
				"run_id":       "run-1",
				"generated_at": "2026-03-16T14:01:38Z",
				"status":       "succeeded",
				"summary_path": "docs/reports/live-validation-runs/run-1/summary.json",
			},
			{
				"run_id":       "run-2",
				"generated_at": "2026-03-14T16:46:57Z",
				"status":       "succeeded",
				"summary_path": "docs/reports/live-validation-runs/run-2/summary.json",
			},
		},
	})
	writeJSONFixture(t, filepath.Join(reportsRoot, "live-validation-summary.json"), map[string]any{
		"generated_at": "2026-03-16T14:01:38Z",
		"status":       "succeeded",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"canonical_report_path":     "docs/reports/multi-node-shared-queue-report.json",
			"canonical_summary_path":    "docs/reports/shared-queue-companion-summary.json",
			"bundle_report_path":        "docs/reports/live-validation-runs/run-1/multi-node-shared-queue-report.json",
			"bundle_summary_path":       "docs/reports/live-validation-runs/run-1/shared-queue-companion-summary.json",
			"cross_node_completions":    99,
			"duplicate_started_tasks":   []any{},
			"duplicate_completed_tasks": []any{},
		},
	})
	for _, runID := range []string{"run-1", "run-2"} {
		writeJSONFixture(t, filepath.Join(reportsRoot, "live-validation-runs", runID, "summary.json"), map[string]any{
			"generated_at": "2026-03-16T14:01:38Z",
			"status":       "succeeded",
			"local":        map[string]any{"enabled": true, "status": "succeeded"},
			"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
			"ray":          map[string]any{"enabled": true, "status": "succeeded"},
		})
	}
	writeJSONFixture(t, filepath.Join(reportsRoot, "multi-node-shared-queue-report.json"), map[string]any{
		"all_ok":                    true,
		"cross_node_completions":    99,
		"duplicate_started_tasks":   []any{},
		"duplicate_completed_tasks": []any{},
	})

	document, err := BuildValidationContinuationScorecard(ValidationContinuationScorecardConfig{
		RepoRoot: repoRoot,
		GoRoot:   goRoot,
	})
	if err != nil {
		t.Fatalf("BuildValidationContinuationScorecard: %v", err)
	}
	if document.Ticket != "BIG-GO-907" || document.Status != "go-validation-continuation-scorecard" {
		t.Fatalf("unexpected document identity: %+v", document)
	}
	if got := document.Summary["recent_bundle_count"]; got != 2 {
		t.Fatalf("expected recent bundle count 2, got %+v", got)
	}
	if got := document.Summary["latest_all_executor_tracks_succeeded"]; got != true {
		t.Fatalf("expected latest executor status success, got %+v", got)
	}
	if len(document.ExecutorLanes) != 3 {
		t.Fatalf("expected three executor lanes, got %+v", document.ExecutorLanes)
	}
	if !document.SharedQueueCompanion.Available || document.SharedQueueCompanion.Mode != "bundle-companion-summary" {
		t.Fatalf("unexpected shared queue companion payload: %+v", document.SharedQueueCompanion)
	}
}

func TestBuildValidationContinuationPolicyGate(t *testing.T) {
	repoRoot := t.TempDir()
	goRoot := filepath.Join(repoRoot, "bigclaw-go")
	reportsRoot := filepath.Join(goRoot, "docs", "reports")
	if err := os.MkdirAll(reportsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFixture(t, filepath.Join(reportsRoot, "validation-bundle-continuation-scorecard.json"), map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "run-1",
			"latest_bundle_age_hours":                           12.5,
			"recent_bundle_count":                               3,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": true,
		},
		"shared_queue_companion": map[string]any{
			"available":              true,
			"cross_node_completions": 99,
			"report_path":            "docs/reports/multi-node-shared-queue-report.json",
			"summary_path":           "docs/reports/shared-queue-companion-summary.json",
		},
	})

	document, exitCode, err := BuildValidationContinuationPolicyGate(ValidationContinuationPolicyGateConfig{
		RepoRoot:                    repoRoot,
		GoRoot:                      goRoot,
		RequireRepeatedLaneCoverage: true,
	})
	if err != nil {
		t.Fatalf("BuildValidationContinuationPolicyGate: %v", err)
	}
	if exitCode != 0 || document.Recommendation != "go" || document.Status != "policy-go" {
		t.Fatalf("unexpected policy gate outcome: exit=%d document=%+v", exitCode, document)
	}
	if got := document.Summary["passing_check_count"]; got != 6 {
		t.Fatalf("expected six passing checks, got %+v", got)
	}
}

func writeJSONFixture(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
