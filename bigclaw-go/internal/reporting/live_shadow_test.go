package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildLiveShadowScorecard(t *testing.T) {
	root := t.TempDir()
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-1",
		"primary":  map[string]any{"task_id": "p-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": "s-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:01Z"}}},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/shadow-matrix-report.json"), map[string]any{
		"total":      1,
		"matched":    1,
		"mismatched": 0,
		"results": []map[string]any{
			{
				"trace_id":    "matrix-1",
				"source_file": "examples/a.json",
				"source_kind": "fixture",
				"primary":     map[string]any{"task_id": "mp-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:00Z"}}},
				"shadow":      map[string]any{"task_id": "ms-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:01Z"}}},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        0,
					"primary_timeline_seconds": 0.2,
					"shadow_timeline_seconds":  0.22,
				},
			},
		},
		"corpus_coverage": map[string]any{"corpus_slice_count": 1, "uncovered_corpus_slice_count": 0},
	})

	report, err := BuildLiveShadowScorecard(root, LiveShadowScorecardOptions{
		Now: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build scorecard: %v", err)
	}
	if asString(asMap(report["evidence_inputs"])["generator_script"]) != LiveShadowScorecardGenerator {
		t.Fatalf("unexpected generator script: %+v", report["evidence_inputs"])
	}
	summary := asMap(report["summary"])
	if asInt(summary["parity_ok_count"]) != 2 || asInt(summary["stale_inputs"]) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(report["freshness"].([]map[string]any)) != 2 {
		t.Fatalf("unexpected freshness payload: %+v", report["freshness"])
	}
}

func TestExportLiveShadowBundle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs/reports"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, path := range []string{
		"docs/migration-shadow.md",
		"docs/reports/migration-readiness-report.md",
		"docs/reports/migration-plan-review-notes.md",
		"docs/reports/live-shadow-comparison-follow-up-digest.md",
		"docs/reports/rollback-safeguard-follow-up-digest.md",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatalf("mkdir doc dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte("# doc\n"), 0o644); err != nil {
			t.Fatalf("write doc: %v", err)
		}
	}
	writeFixtureJSON(t, filepath.Join(root, "docs/reports/shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-1",
		"primary":  map[string]any{"task_id": "p-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": "s-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:01Z"}}},
		"diff":     map[string]any{"state_equal": true, "event_types_equal": true, "event_count_delta": 0, "primary_timeline_seconds": 0.1, "shadow_timeline_seconds": 0.15},
	})
	writeFixtureJSON(t, filepath.Join(root, "docs/reports/shadow-matrix-report.json"), map[string]any{
		"total": 1, "matched": 1, "mismatched": 0,
		"results": []map[string]any{{"trace_id": "matrix-1", "primary": map[string]any{"task_id": "mp-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:00Z"}}}, "shadow": map[string]any{"task_id": "ms-1", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:01Z"}}}, "diff": map[string]any{"state_equal": true, "event_types_equal": true, "event_count_delta": 0, "primary_timeline_seconds": 0.2, "shadow_timeline_seconds": 0.22}}},
	})
	writeFixtureJSON(t, filepath.Join(root, "docs/reports/live-shadow-mirror-scorecard.json"), map[string]any{
		"summary":             map[string]any{"latest_evidence_timestamp": "2026-03-10T10:05:01Z", "total_evidence_runs": 2, "parity_ok_count": 2, "drift_detected_count": 0, "matrix_total": 1, "matrix_mismatched": 0, "fresh_inputs": 2, "stale_inputs": 0},
		"freshness":           []map[string]any{{"status": "fresh"}, {"status": "fresh"}},
		"cutover_checkpoints": []map[string]any{{"name": "ok", "passed": true}},
	})
	writeFixtureJSON(t, filepath.Join(root, "docs/reports/rollback-trigger-surface.json"), map[string]any{
		"summary":       map[string]any{"status": "manual-only", "automation_boundary": "manual", "automated_rollback_trigger": false, "distinctions": map[string]any{}},
		"reviewer_path": map[string]any{"digest_issue": map[string]any{"id": "OPE-254", "slug": "BIG-PAR-088"}, "digest_path": "docs/reports/rollback-safeguard-follow-up-digest.md"},
	})

	manifest, indexText, err := ExportLiveShadowBundle(root, LiveShadowBundleOptions{
		RunID: "20260310T100501Z",
		Now:   time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("export live shadow bundle: %v", err)
	}
	if asString(asMap(manifest["latest"])["run_id"]) != "20260310T100501Z" {
		t.Fatalf("unexpected manifest latest: %+v", manifest["latest"])
	}
	if !strings.Contains(indexText, "Live Shadow Mirror Index") {
		t.Fatalf("unexpected index text: %s", indexText)
	}
}
