package liveshadow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildScorecardDetectsDriftAndStaleInputs(t *testing.T) {
	repoRoot := t.TempDir()
	mustWriteJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-trace",
		"primary":  map[string]any{"task_id": "compare-primary", "events": []map[string]any{{"timestamp": "2026-03-01T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": "compare-shadow", "events": []map[string]any{{"timestamp": "2026-03-01T10:00:05Z"}}},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	})
	mustWriteJSON(t, filepath.Join(repoRoot, "bigclaw-go/docs/reports/shadow-matrix-report.json"), map[string]any{
		"total":      1,
		"matched":    0,
		"mismatched": 1,
		"results": []map[string]any{
			{
				"trace_id":    "matrix-trace",
				"source_file": "./examples/drift.json",
				"source_kind": "fixture",
				"task_shape":  "executor:local|scenario:drift",
				"primary":     map[string]any{"task_id": "matrix-primary", "events": []map[string]any{{"timestamp": "2026-02-20T10:00:00Z"}}},
				"shadow":      map[string]any{"task_id": "matrix-shadow", "events": []map[string]any{{"timestamp": "2026-02-20T10:00:03Z"}}},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        1,
					"primary_timeline_seconds": 0.1,
					"shadow_timeline_seconds":  0.6,
				},
			},
		},
		"corpus_coverage": map[string]any{
			"corpus_slice_count":           1,
			"uncovered_corpus_slice_count": 1,
		},
	})

	generatedAt := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	report, err := BuildScorecard(repoRoot, "bigclaw-go/docs/reports/shadow-compare-report.json", "bigclaw-go/docs/reports/shadow-matrix-report.json", generatedAt)
	if err != nil {
		t.Fatalf("BuildScorecard: %v", err)
	}

	if report.Ticket != "BIG-PAR-092" {
		t.Fatalf("ticket = %q, want BIG-PAR-092", report.Ticket)
	}
	if report.Summary.TotalEvidenceRuns != 2 || report.Summary.DriftDetectedCount != 1 || report.Summary.StaleInputs != 2 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.ParityEntries[0].Parity.Status != "parity-ok" || report.ParityEntries[1].Parity.Status != "drift-detected" {
		t.Fatalf("unexpected parity entries: %+v", report.ParityEntries)
	}
	if got, want := report.ParityEntries[1].Parity.Reasons, []string{"event-count-drift", "timeline-drift"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("reasons = %#v, want %#v", got, want)
	}
	if report.CutoverCheckpoints[1].Passed || report.CutoverCheckpoints[3].Passed {
		t.Fatalf("expected mismatched matrix and stale inputs to fail checkpoints: %+v", report.CutoverCheckpoints)
	}
}

func TestExportBundleGeneratesIndexAndRollup(t *testing.T) {
	goRoot := filepath.Join(t.TempDir(), "bigclaw-go")
	mustWriteFile(t, filepath.Join(goRoot, "docs/migration-shadow.md"), "# shadow\n")
	mustWriteFile(t, filepath.Join(goRoot, "docs/reports/migration-readiness-report.md"), "# readiness\n")
	mustWriteFile(t, filepath.Join(goRoot, "docs/reports/migration-plan-review-notes.md"), "# review\n")
	mustWriteFile(t, filepath.Join(goRoot, "docs/reports/live-shadow-comparison-follow-up-digest.md"), "# digest\n")
	mustWriteJSON(t, filepath.Join(goRoot, "docs/reports/shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-1",
		"primary":  map[string]any{"task_id": "primary-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": "shadow-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:01Z"}}},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	})
	mustWriteJSON(t, filepath.Join(goRoot, "docs/reports/shadow-matrix-report.json"), map[string]any{
		"total":      2,
		"matched":    2,
		"mismatched": 0,
		"results": []map[string]any{
			{
				"trace_id": "matrix-1",
				"primary":  map[string]any{"task_id": "matrix-primary-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:00Z"}}},
				"shadow":   map[string]any{"task_id": "matrix-shadow-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:01Z"}}},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        0,
					"primary_timeline_seconds": 0.2,
					"shadow_timeline_seconds":  0.22,
				},
			},
			{
				"trace_id": "matrix-2",
				"primary":  map[string]any{"task_id": "matrix-primary-2", "events": []map[string]any{{"timestamp": "2026-03-10T10:06:00Z"}}},
				"shadow":   map[string]any{"task_id": "matrix-shadow-2", "events": []map[string]any{{"timestamp": "2026-03-10T10:06:01Z"}}},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        0,
					"primary_timeline_seconds": 0.2,
					"shadow_timeline_seconds":  0.23,
				},
			},
		},
	})
	mustWriteJSON(t, filepath.Join(goRoot, "docs/reports/live-shadow-mirror-scorecard.json"), map[string]any{
		"generated_at": "2026-03-11T00:00:00Z",
		"ticket":       "BIG-PAR-092",
		"title":        "Live shadow mirror parity drift scorecard",
		"status":       "repo-native-live-shadow-scorecard",
		"summary": map[string]any{
			"latest_evidence_timestamp": "2026-03-10T10:06:01Z",
			"total_evidence_runs":       3,
			"parity_ok_count":           3,
			"drift_detected_count":      0,
			"matrix_total":              2,
			"matrix_mismatched":         0,
			"fresh_inputs":              2,
			"stale_inputs":              0,
		},
		"freshness":           []map[string]any{{"status": "fresh"}, {"status": "fresh"}},
		"cutover_checkpoints": []map[string]any{{"name": "ok", "passed": true}},
	})
	mustWriteJSON(t, filepath.Join(goRoot, "docs/reports/rollback-trigger-surface.json"), map[string]any{
		"issue": map[string]any{"id": "OPE-254", "slug": "BIG-PAR-088"},
		"summary": map[string]any{
			"status":                     "manual-only",
			"automation_boundary":        "manual",
			"automated_rollback_trigger": false,
			"distinctions":               map[string]any{"blockers": 0, "warnings": 0, "manual_only_paths": 0},
		},
		"shared_guardrail_summary": map[string]any{"digest_path": "docs/reports/live-shadow-comparison-follow-up-digest.md"},
	})

	manifest, err := ExportBundle(BundleOptions{
		GoRoot:            goRoot,
		ShadowComparePath: "docs/reports/shadow-compare-report.json",
		ShadowMatrixPath:  "docs/reports/shadow-matrix-report.json",
		ScorecardPath:     "docs/reports/live-shadow-mirror-scorecard.json",
		BundleRoot:        "docs/reports/live-shadow-runs",
		SummaryPath:       "docs/reports/live-shadow-summary.json",
		IndexPath:         "docs/reports/live-shadow-index.md",
		ManifestPath:      "docs/reports/live-shadow-index.json",
		RollupPath:        "docs/reports/live-shadow-drift-rollup.json",
		RunID:             "20260310T100601Z",
	}, time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ExportBundle: %v", err)
	}

	if manifest.Latest.RunID != "20260310T100601Z" || manifest.Latest.Status != "parity-ok" || manifest.DriftRollup.Status != "parity-ok" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if len(manifest.Latest.MatrixTraceIDs) != 2 || manifest.Latest.MatrixTraceIDs[0] != "matrix-1" {
		t.Fatalf("unexpected matrix trace ids: %+v", manifest.Latest.MatrixTraceIDs)
	}
	indexText, err := os.ReadFile(filepath.Join(goRoot, "docs/reports/live-shadow-index.md"))
	if err != nil {
		t.Fatalf("ReadFile(index): %v", err)
	}
	if string(indexText) == "" || !contains(string(indexText), "go run ./cmd/bigclawctl live-shadow scorecard --pretty") {
		t.Fatalf("unexpected index text: %s", string(indexText))
	}
}

func mustWriteJSON(t *testing.T, path string, payload any) {
	t.Helper()
	if err := writeJSON(path, payload); err != nil {
		t.Fatalf("writeJSON(%s): %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func contains(body string, needle string) bool {
	return len(body) >= len(needle) && (body == needle || filepath.Base(body) == needle || (len(body) > 0 && len(needle) > 0 && stringIndex(body, needle) >= 0))
}

func stringIndex(body string, needle string) int {
	for i := 0; i+len(needle) <= len(body); i++ {
		if body[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
