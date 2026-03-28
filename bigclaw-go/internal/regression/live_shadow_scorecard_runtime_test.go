package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestLiveShadowScorecardDetectsDriftAndStaleInputs(t *testing.T) {
	comparePath, matrixPath := writeLiveShadowScorecardInputs(t)
	report := runLiveShadowScorecardBuildReport(t, comparePath, matrixPath)

	if report.Ticket != "BIG-PAR-092" {
		t.Fatalf("unexpected live shadow scorecard ticket: %+v", report)
	}
	if report.Summary.TotalEvidenceRuns != 2 ||
		report.Summary.DriftDetectedCount != 1 ||
		report.Summary.StaleInputs != 2 {
		t.Fatalf("unexpected live shadow scorecard summary: %+v", report.Summary)
	}
	if len(report.ParityEntries) != 2 {
		t.Fatalf("unexpected parity entry count: %+v", report.ParityEntries)
	}
	if report.ParityEntries[0].Parity.Status != "parity-ok" {
		t.Fatalf("unexpected compare parity entry: %+v", report.ParityEntries[0])
	}
	if report.ParityEntries[1].Parity.Status != "drift-detected" {
		t.Fatalf("unexpected matrix parity status: %+v", report.ParityEntries[1])
	}
	wantReasons := []string{"event-count-drift", "timeline-drift"}
	if !equalStrings(report.ParityEntries[1].Parity.Reasons, wantReasons) {
		t.Fatalf("unexpected drift reasons: got=%v want=%v", report.ParityEntries[1].Parity.Reasons, wantReasons)
	}
	if len(report.CutoverCheckpoints) < 4 || report.CutoverCheckpoints[1].Passed || report.CutoverCheckpoints[3].Passed {
		t.Fatalf("unexpected cutover checkpoints: %+v", report.CutoverCheckpoints)
	}
}

type liveShadowScorecardRuntimeReport struct {
	Ticket  string `json:"ticket"`
	Summary struct {
		TotalEvidenceRuns  int `json:"total_evidence_runs"`
		DriftDetectedCount int `json:"drift_detected_count"`
		StaleInputs        int `json:"stale_inputs"`
	} `json:"summary"`
	ParityEntries []struct {
		Parity struct {
			Status  string   `json:"status"`
			Reasons []string `json:"reasons"`
		} `json:"parity"`
	} `json:"parity_entries"`
	CutoverCheckpoints []struct {
		Passed bool `json:"passed"`
	} `json:"cutover_checkpoints"`
}

func writeLiveShadowScorecardInputs(t *testing.T) (string, string) {
	t.Helper()

	compareReport := map[string]any{
		"trace_id": "compare-trace",
		"primary": map[string]any{
			"task_id": "compare-primary",
			"events":  []map[string]any{{"timestamp": "2026-03-01T10:00:00Z"}},
		},
		"shadow": map[string]any{
			"task_id": "compare-shadow",
			"events":  []map[string]any{{"timestamp": "2026-03-01T10:00:05Z"}},
		},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	}
	matrixReport := map[string]any{
		"total":      1,
		"matched":    0,
		"mismatched": 1,
		"results": []map[string]any{
			{
				"trace_id":    "matrix-trace",
				"source_file": "./examples/drift.json",
				"source_kind": "fixture",
				"task_shape":  "executor:local|scenario:drift",
				"primary": map[string]any{
					"task_id": "matrix-primary",
					"events":  []map[string]any{{"timestamp": "2026-02-20T10:00:00Z"}},
				},
				"shadow": map[string]any{
					"task_id": "matrix-shadow",
					"events":  []map[string]any{{"timestamp": "2026-02-20T10:00:03Z"}},
				},
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
	}

	dir := t.TempDir()
	comparePath := filepath.Join(dir, "shadow-compare.json")
	matrixPath := filepath.Join(dir, "shadow-matrix.json")
	writeJSONFixture(t, comparePath, compareReport)
	writeJSONFixture(t, matrixPath, matrixReport)
	return comparePath, matrixPath
}

func runLiveShadowScorecardBuildReport(t *testing.T, comparePath, matrixPath string) liveShadowScorecardRuntimeReport {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "migration", "live_shadow_scorecard.py")
	pythonSnippet := "import importlib.util, json\n" +
		"spec = importlib.util.spec_from_file_location('live_shadow_scorecard', r'" + scriptPath + "')\n" +
		"module = importlib.util.module_from_spec(spec)\n" +
		"assert spec.loader is not None\n" +
		"spec.loader.exec_module(module)\n" +
		"print(json.dumps(module.build_report(shadow_compare_report_path=r'" + comparePath + "', shadow_matrix_report_path=r'" + matrixPath + "')))\n"

	cmd := testharness.PythonCommand(t, "-c", pythonSnippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build live shadow scorecard report: %v (%s)", err, string(output))
	}

	var report liveShadowScorecardRuntimeReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode live shadow scorecard report: %v (%s)", err, string(output))
	}
	return report
}

func writeJSONFixture(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal fixture %s: %v", path, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
