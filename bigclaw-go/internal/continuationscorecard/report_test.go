package continuationscorecard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestBuildReport(t *testing.T) {
	repoRoot := t.TempDir()
	writeJSONFile(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-index.json"), map[string]any{
		"latest": map[string]any{
			"run_id":       "20260316T140138Z",
			"status":       "succeeded",
			"generated_at": "2026-03-16T14:01:38Z",
		},
		"recent_runs": []map[string]any{
			{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/r1/summary.json"},
			{"summary_path": "bigclaw-go/docs/reports/live-validation-runs/r0/summary.json"},
		},
	})
	summary := map[string]any{
		"status":                 "succeeded",
		"generated_at":           "2026-03-16T14:01:38Z",
		"local":                  map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":             map[string]any{"enabled": true, "status": "succeeded"},
		"ray":                    map[string]any{"enabled": true, "status": "succeeded"},
		"shared_queue_companion": map[string]any{"available": true, "cross_node_completions": 7},
	}
	writeJSONFile(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-summary.json"), summary)
	writeJSONFile(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-runs/r1/summary.json"), summary)
	writeJSONFile(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/live-validation-runs/r0/summary.json"), map[string]any{
		"status":       "succeeded",
		"generated_at": "2026-03-16T13:01:38Z",
		"local":        map[string]any{"enabled": true, "status": "succeeded"},
		"kubernetes":   map[string]any{"enabled": true, "status": "succeeded"},
		"ray":          map[string]any{"enabled": true, "status": "succeeded"},
	})
	writeJSONFile(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"), map[string]any{
		"all_ok":                    true,
		"cross_node_completions":    7,
		"duplicate_completed_tasks": []any{},
		"duplicate_started_tasks":   []any{},
	})

	report, err := BuildReport(BuildOptions{
		RepoRoot: repoRoot,
		Now:      time.Date(2026, 3, 16, 14, 31, 38, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if got := report["status"]; got != "local-continuation-scorecard" {
		t.Fatalf("unexpected status: %v", got)
	}
	summaryBlock := report["summary"].(map[string]any)
	if got := summaryBlock["recent_bundle_count"]; got != 2 {
		t.Fatalf("unexpected recent bundle count: %v", got)
	}
}
