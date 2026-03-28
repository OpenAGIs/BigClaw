package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestExportLiveShadowBundleGeneratesIndexAndRollup(t *testing.T) {
	root := writeLiveShadowBundleFixtureTree(t, "go", "20260310T100601Z", liveShadowBundleFixtureConfig{
		CompareTraceID:          "compare-1",
		PrimaryTaskID:           "primary-1",
		ShadowTaskID:            "shadow-1",
		LatestEvidenceTimestamp: "2026-03-10T10:06:01Z",
		TotalEvidenceRuns:       3,
		ParityOKCount:           3,
		DriftDetectedCount:      0,
		MatrixTotal:             2,
		MatrixMismatched:        0,
		FreshInputs:             2,
		StaleInputs:             0,
		MatrixEntries: []liveShadowMatrixFixture{
			{
				TraceID:       "matrix-1",
				PrimaryTaskID: "matrix-primary-1",
				ShadowTaskID:  "matrix-shadow-1",
				PrimaryTime:   "2026-03-10T10:05:00Z",
				ShadowTime:    "2026-03-10T10:05:01Z",
				PrimarySecs:   0.2,
				ShadowSecs:    0.22,
			},
			{
				TraceID:       "matrix-2",
				PrimaryTaskID: "matrix-primary-2",
				ShadowTaskID:  "matrix-shadow-2",
				PrimaryTime:   "2026-03-10T10:06:00Z",
				ShadowTime:    "2026-03-10T10:06:01Z",
				PrimarySecs:   0.2,
				ShadowSecs:    0.23,
			},
		},
	})

	cmd := exec.Command(
		testharness.PythonExecutable(t),
		testharness.JoinRepoRoot(t, "scripts", "migration", "export_live_shadow_bundle.py"),
		"--go-root", root,
		"--run-id", "20260310T100601Z",
	)
	cmd.Dir = testharness.ProjectRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("export live shadow bundle: %v (%s)", err, string(output))
	}

	var summary struct {
		RunID     string `json:"run_id"`
		Status    string `json:"status"`
		Severity  string `json:"severity"`
		Artifacts struct {
			ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		} `json:"artifacts"`
		CompareTraceID string   `json:"compare_trace_id"`
		MatrixTraceIDs []string `json:"matrix_trace_ids"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-summary.json"), &summary)
	if summary.RunID != "20260310T100601Z" || summary.Status != "parity-ok" || summary.Severity != "none" {
		t.Fatalf("unexpected live shadow summary identity: %+v", summary)
	}
	if !strings.HasSuffix(summary.Artifacts.ShadowCompareReportPath, "shadow-compare-report.json") {
		t.Fatalf("unexpected compare report path: %+v", summary.Artifacts)
	}
	if summary.CompareTraceID != "compare-1" || !equalStrings(summary.MatrixTraceIDs, []string{"matrix-1", "matrix-2"}) {
		t.Fatalf("unexpected trace lineage: %+v", summary)
	}

	var manifest struct {
		Latest struct {
			RunID string `json:"run_id"`
		} `json:"latest"`
		RecentRuns []struct {
			RunID string `json:"run_id"`
		} `json:"recent_runs"`
		DriftRollup struct {
			Summary struct {
				HighestSeverity   string `json:"highest_severity"`
				DriftDetectedRuns int    `json:"drift_detected_runs"`
			} `json:"summary"`
		} `json:"drift_rollup"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-index.json"), &manifest)
	if manifest.Latest.RunID != "20260310T100601Z" || len(manifest.RecentRuns) == 0 || manifest.RecentRuns[0].RunID != "20260310T100601Z" {
		t.Fatalf("unexpected live shadow manifest identity: %+v", manifest)
	}
	if manifest.DriftRollup.Summary.HighestSeverity != "none" || manifest.DriftRollup.Summary.DriftDetectedRuns != 0 {
		t.Fatalf("unexpected live shadow rollup summary: %+v", manifest.DriftRollup.Summary)
	}

	var rollup struct {
		Status  string `json:"status"`
		Summary struct {
			RecentRunCount int `json:"recent_run_count"`
		} `json:"summary"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-drift-rollup.json"), &rollup)
	if rollup.Status != "parity-ok" || rollup.Summary.RecentRunCount != 1 {
		t.Fatalf("unexpected live shadow rollup: %+v", rollup)
	}

	indexText := readTextFile(t, filepath.Join(root, "docs", "reports", "live-shadow-index.md"))
	for _, want := range []string{
		"Live Shadow Mirror Index",
		"docs/reports/live-shadow-runs/20260310T100601Z",
		"docs/migration-shadow.md",
		"docs/reports/live-shadow-comparison-follow-up-digest.md",
	} {
		if !strings.Contains(indexText, want) {
			t.Fatalf("index markdown missing %q in %s", want, indexText)
		}
	}

	bundleReadme := readTextFile(t, filepath.Join(root, "docs", "reports", "live-shadow-runs", "20260310T100601Z", "README.md"))
	if !strings.Contains(bundleReadme, "Parity drift rollup") {
		t.Fatalf("bundle README missing parity drift rollup section: %s", bundleReadme)
	}
}

func TestExportLiveShadowBundleSupportsDocumentedBigclawGoCWD(t *testing.T) {
	root := writeLiveShadowBundleFixtureTree(t, "bigclaw-go", "20260310T100501Z", liveShadowBundleFixtureConfig{
		CompareTraceID:          "compare-cwd",
		PrimaryTaskID:           "primary-cwd",
		ShadowTaskID:            "shadow-cwd",
		LatestEvidenceTimestamp: "2026-03-10T10:05:01Z",
		TotalEvidenceRuns:       2,
		ParityOKCount:           2,
		DriftDetectedCount:      0,
		MatrixTotal:             1,
		MatrixMismatched:        0,
		FreshInputs:             2,
		StaleInputs:             0,
		MatrixEntries: []liveShadowMatrixFixture{
			{
				TraceID:       "matrix-cwd",
				PrimaryTaskID: "matrix-primary-cwd",
				ShadowTaskID:  "matrix-shadow-cwd",
				PrimaryTime:   "2026-03-10T10:05:00Z",
				ShadowTime:    "2026-03-10T10:05:01Z",
				PrimarySecs:   0.2,
				ShadowSecs:    0.22,
			},
		},
	})

	cmd := exec.Command(
		testharness.PythonExecutable(t),
		testharness.JoinRepoRoot(t, "scripts", "migration", "export_live_shadow_bundle.py"),
		"--run-id", "20260310T100501Z",
	)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("export live shadow bundle from documented cwd: %v (%s)", err, string(output))
	}

	var summary struct {
		RunID     string `json:"run_id"`
		Artifacts struct {
			ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		} `json:"artifacts"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-summary.json"), &summary)
	if summary.RunID != "20260310T100501Z" {
		t.Fatalf("unexpected live shadow summary run id: %+v", summary)
	}
	if !strings.HasSuffix(summary.Artifacts.ShadowCompareReportPath, "shadow-compare-report.json") {
		t.Fatalf("unexpected compare report path: %+v", summary.Artifacts)
	}
}

type liveShadowBundleFixtureConfig struct {
	CompareTraceID          string
	PrimaryTaskID           string
	ShadowTaskID            string
	LatestEvidenceTimestamp string
	TotalEvidenceRuns       int
	ParityOKCount           int
	DriftDetectedCount      int
	MatrixTotal             int
	MatrixMismatched        int
	FreshInputs             int
	StaleInputs             int
	MatrixEntries           []liveShadowMatrixFixture
}

type liveShadowMatrixFixture struct {
	TraceID       string
	PrimaryTaskID string
	ShadowTaskID  string
	PrimaryTime   string
	ShadowTime    string
	PrimarySecs   float64
	ShadowSecs    float64
}

func writeLiveShadowBundleFixtureTree(t *testing.T, rootName, runID string, cfg liveShadowBundleFixtureConfig) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), rootName)
	reportsDir := filepath.Join(root, "docs", "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "migration-shadow.md"), []byte("# shadow\n"), 0o644); err != nil {
		t.Fatalf("write migration-shadow.md: %v", err)
	}
	writeLiveShadowTextFixture(t, filepath.Join(reportsDir, "migration-readiness-report.md"), "# readiness\n")
	writeLiveShadowTextFixture(t, filepath.Join(reportsDir, "migration-plan-review-notes.md"), "# review\n")
	writeLiveShadowTextFixture(t, filepath.Join(reportsDir, "live-shadow-comparison-follow-up-digest.md"), "# digest\n")
	writeLiveShadowTextFixture(t, filepath.Join(reportsDir, "rollback-safeguard-follow-up-digest.md"), "# rollback digest\n")

	writeLiveShadowJSONFixture(t, filepath.Join(reportsDir, "shadow-compare-report.json"), map[string]any{
		"trace_id": cfg.CompareTraceID,
		"primary":  map[string]any{"task_id": cfg.PrimaryTaskID, "events": []map[string]any{{"timestamp": "2026-03-10T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": cfg.ShadowTaskID, "events": []map[string]any{{"timestamp": "2026-03-10T10:00:01Z"}}},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	})

	matrixResults := make([]map[string]any, 0, len(cfg.MatrixEntries))
	for _, entry := range cfg.MatrixEntries {
		matrixResults = append(matrixResults, map[string]any{
			"trace_id": entry.TraceID,
			"primary":  map[string]any{"task_id": entry.PrimaryTaskID, "events": []map[string]any{{"timestamp": entry.PrimaryTime}}},
			"shadow":   map[string]any{"task_id": entry.ShadowTaskID, "events": []map[string]any{{"timestamp": entry.ShadowTime}}},
			"diff": map[string]any{
				"state_equal":              true,
				"event_types_equal":        true,
				"event_count_delta":        0,
				"primary_timeline_seconds": entry.PrimarySecs,
				"shadow_timeline_seconds":  entry.ShadowSecs,
			},
		})
	}
	writeLiveShadowJSONFixture(t, filepath.Join(reportsDir, "shadow-matrix-report.json"), map[string]any{
		"total":      cfg.MatrixTotal,
		"matched":    cfg.MatrixTotal - cfg.MatrixMismatched,
		"mismatched": cfg.MatrixMismatched,
		"results":    matrixResults,
	})
	writeLiveShadowJSONFixture(t, filepath.Join(reportsDir, "rollback-trigger-surface.json"), map[string]any{
		"summary": map[string]any{
			"status":                     "manual-only",
			"automation_boundary":        "manual",
			"automated_rollback_trigger": false,
			"distinctions":               map[string]any{},
		},
	})
	writeLiveShadowJSONFixture(t, filepath.Join(reportsDir, "live-shadow-mirror-scorecard.json"), map[string]any{
		"summary": map[string]any{
			"latest_evidence_timestamp": cfg.LatestEvidenceTimestamp,
			"total_evidence_runs":       cfg.TotalEvidenceRuns,
			"parity_ok_count":           cfg.ParityOKCount,
			"drift_detected_count":      cfg.DriftDetectedCount,
			"matrix_total":              cfg.MatrixTotal,
			"matrix_mismatched":         cfg.MatrixMismatched,
			"fresh_inputs":              cfg.FreshInputs,
			"stale_inputs":              cfg.StaleInputs,
		},
		"freshness":           []map[string]any{{"status": "fresh"}, {"status": "fresh"}},
		"cutover_checkpoints": []map[string]any{{"name": "ok", "passed": true}},
	})

	_ = runID
	return root
}

func writeLiveShadowJSONFixture(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeLiveShadowTextFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
