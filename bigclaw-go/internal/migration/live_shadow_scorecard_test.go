package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildLiveShadowScorecard(t *testing.T) {
	repoRoot := t.TempDir()
	writeJSONFile(t, filepath.Join(repoRoot, "docs/reports/shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-trace",
		"primary": map[string]any{
			"task_id": "primary-1",
			"events": []map[string]any{
				{"timestamp": "2026-03-13T15:53:21.403765+08:00"},
			},
		},
		"shadow": map[string]any{
			"task_id": "shadow-1",
			"events": []map[string]any{
				{"timestamp": "2026-03-13T15:53:21.404001+08:00"},
			},
		},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 1.25,
			"shadow_timeline_seconds":  1.251236,
		},
	})
	writeJSONFile(t, filepath.Join(repoRoot, "docs/reports/shadow-matrix-report.json"), map[string]any{
		"total":      1,
		"matched":    1,
		"mismatched": 0,
		"results": []map[string]any{
			{
				"trace_id":    "matrix-trace",
				"source_file": "./examples/shadow-task.json",
				"primary": map[string]any{
					"task_id": "primary-2",
					"events": []map[string]any{
						{"timestamp": "2026-03-13T16:56:55.415367+08:00"},
					},
				},
				"shadow": map[string]any{
					"task_id": "shadow-2",
					"events": []map[string]any{
						{"timestamp": "2026-03-13T16:56:55.415367+08:00"},
					},
				},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        0,
					"primary_timeline_seconds": 2.0,
					"shadow_timeline_seconds":  2.1,
				},
			},
		},
		"corpus_coverage": map[string]any{
			"corpus_slice_count":           3,
			"uncovered_corpus_slice_count": 1,
		},
	})

	now := time.Date(2026, 3, 16, 15, 58, 21, 282621000, time.UTC)
	report, err := BuildLiveShadowScorecard(repoRoot, "docs/reports/shadow-compare-report.json", "docs/reports/shadow-matrix-report.json", now)
	if err != nil {
		t.Fatalf("BuildLiveShadowScorecard returned error: %v", err)
	}
	if report.EvidenceInputs.GeneratorScript != "go run ./cmd/bigclawctl migration live-shadow-scorecard" {
		t.Fatalf("unexpected generator: %s", report.EvidenceInputs.GeneratorScript)
	}
	if report.Summary.TotalEvidenceRuns != 2 || report.Summary.ParityOKCount != 2 || report.Summary.DriftDetectedCount != 0 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Summary.LatestEvidenceTimestamp != "2026-03-13T16:56:55.415367+08:00" {
		t.Fatalf("unexpected latest evidence timestamp: %s", report.Summary.LatestEvidenceTimestamp)
	}
	if len(report.Freshness) != 2 || report.Freshness[0].Status != "fresh" || report.Freshness[1].Status != "fresh" {
		t.Fatalf("unexpected freshness: %+v", report.Freshness)
	}
	if len(report.ParityEntries) != 2 || report.ParityEntries[0].SourceFile != nil || report.ParityEntries[1].SourceFile == nil {
		t.Fatalf("unexpected parity entries: %+v", report.ParityEntries)
	}
	if !report.CutoverChecks[3].Passed {
		t.Fatalf("expected freshness checkpoint to pass: %+v", report.CutoverChecks[3])
	}
}

func TestWriteLiveShadowScorecard(t *testing.T) {
	output := filepath.Join(t.TempDir(), "docs/reports/live-shadow-mirror-scorecard.json")
	report := Scorecard{GeneratedAt: "2026-03-27T00:00:00Z", Ticket: "BIG-PAR-092"}
	if err := WriteLiveShadowScorecard(output, report); err != nil {
		t.Fatalf("WriteLiveShadowScorecard returned error: %v", err)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if decoded["ticket"] != "BIG-PAR-092" {
		t.Fatalf("unexpected output payload: %v", decoded)
	}
}

func writeJSONFile(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
