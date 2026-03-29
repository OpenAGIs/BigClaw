package regression

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLane8ExportLiveShadowBundleGeneratesIndexAndRollup(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	root := filepath.Join(t.TempDir(), "go")
	reports := filepath.Join(root, "docs", "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatalf("mkdir reports: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "docs", "migration-shadow.md"):                            "# shadow\n",
		filepath.Join(reports, "migration-readiness-report.md"):                       "# readiness\n",
		filepath.Join(reports, "migration-plan-review-notes.md"):                      "# review\n",
		filepath.Join(reports, "live-shadow-comparison-follow-up-digest.md"):          "# digest\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeJSONFixture(t, filepath.Join(reports, "shadow-compare-report.json"), map[string]any{
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
	writeJSONFixture(t, filepath.Join(reports, "shadow-matrix-report.json"), map[string]any{
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
	writeJSONFixture(t, filepath.Join(reports, "rollback-trigger-surface.json"), map[string]any{
		"summary": map[string]any{
			"status":                    "manual-only",
			"automation_boundary":       "manual",
			"automated_rollback_trigger": false,
			"distinctions":              map[string]any{},
		},
	})
	writeJSONFixture(t, filepath.Join(reports, "live-shadow-mirror-scorecard.json"), map[string]any{
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

	script := filepath.Join(repoRoot, "scripts", "migration", "export_live_shadow_bundle.py")
	cmd := exec.Command("python3", script, "--go-root", root, "--run-id", "20260310T100601Z")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run export_live_shadow_bundle.py: %v\n%s", err, output)
	}

	var summary struct {
		RunID       string `json:"run_id"`
		Status      string `json:"status"`
		Severity    string `json:"severity"`
		CompareTraceID string `json:"compare_trace_id"`
		MatrixTraceIDs []string `json:"matrix_trace_ids"`
		Artifacts   struct {
			ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		} `json:"artifacts"`
	}
	readJSONFile(t, filepath.Join(reports, "live-shadow-summary.json"), &summary)
	if summary.RunID != "20260310T100601Z" || summary.Status != "parity-ok" || summary.Severity != "none" {
		t.Fatalf("unexpected summary identity: %+v", summary)
	}
	if !strings.HasSuffix(summary.Artifacts.ShadowCompareReportPath, "shadow-compare-report.json") {
		t.Fatalf("unexpected summary artifacts: %+v", summary.Artifacts)
	}
	if summary.CompareTraceID != "compare-1" || len(summary.MatrixTraceIDs) != 2 || summary.MatrixTraceIDs[0] != "matrix-1" || summary.MatrixTraceIDs[1] != "matrix-2" {
		t.Fatalf("unexpected summary trace ids: %+v", summary)
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
	readJSONFile(t, filepath.Join(reports, "live-shadow-index.json"), &manifest)
	if manifest.Latest.RunID != "20260310T100601Z" || len(manifest.RecentRuns) == 0 || manifest.RecentRuns[0].RunID != "20260310T100601Z" {
		t.Fatalf("unexpected index manifest: %+v", manifest)
	}
	if manifest.DriftRollup.Summary.HighestSeverity != "none" || manifest.DriftRollup.Summary.DriftDetectedRuns != 0 {
		t.Fatalf("unexpected drift rollup summary: %+v", manifest.DriftRollup.Summary)
	}

	var rollup struct {
		Status  string `json:"status"`
		Summary struct {
			RecentRunCount int `json:"recent_run_count"`
		} `json:"summary"`
	}
	readJSONFile(t, filepath.Join(reports, "live-shadow-drift-rollup.json"), &rollup)
	if rollup.Status != "parity-ok" || rollup.Summary.RecentRunCount != 1 {
		t.Fatalf("unexpected drift rollup: %+v", rollup)
	}

	indexText := readRepoFile(t, root, "docs/reports/live-shadow-index.md")
	for _, fragment := range []string{
		"Live Shadow Mirror Index",
		"docs/reports/live-shadow-runs/20260310T100601Z",
		"docs/migration-shadow.md",
		"docs/reports/live-shadow-comparison-follow-up-digest.md",
	} {
		if !strings.Contains(indexText, fragment) {
			t.Fatalf("expected %q in index text, got %s", fragment, indexText)
		}
	}

	bundleReadme := readRepoFile(t, root, "docs/reports/live-shadow-runs/20260310T100601Z/README.md")
	if !strings.Contains(bundleReadme, "Parity drift rollup") {
		t.Fatalf("unexpected bundle readme: %s", bundleReadme)
	}
}

func TestLane8ExportLiveShadowBundleSupportsDocumentedBigclawGoCWD(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	root := filepath.Join(t.TempDir(), "bigclaw-go")
	reports := filepath.Join(root, "docs", "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatalf("mkdir reports: %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "docs", "migration-shadow.md"):                   "# shadow\n",
		filepath.Join(reports, "migration-readiness-report.md"):              "# readiness\n",
		filepath.Join(reports, "migration-plan-review-notes.md"):             "# review\n",
		filepath.Join(reports, "live-shadow-comparison-follow-up-digest.md"): "# digest\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeJSONFixture(t, filepath.Join(reports, "shadow-compare-report.json"), map[string]any{
		"trace_id": "compare-cwd",
		"primary":  map[string]any{"task_id": "primary-cwd", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:00Z"}}},
		"shadow":   map[string]any{"task_id": "shadow-cwd", "events": []map[string]any{{"timestamp": "2026-03-10T10:00:01Z"}}},
		"diff": map[string]any{
			"state_equal":              true,
			"event_types_equal":        true,
			"event_count_delta":        0,
			"primary_timeline_seconds": 0.1,
			"shadow_timeline_seconds":  0.15,
		},
	})
	writeJSONFixture(t, filepath.Join(reports, "shadow-matrix-report.json"), map[string]any{
		"total":      1,
		"matched":    1,
		"mismatched": 0,
		"results": []map[string]any{
			{
				"trace_id": "matrix-cwd",
				"primary":  map[string]any{"task_id": "matrix-primary-cwd", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:00Z"}}},
				"shadow":   map[string]any{"task_id": "matrix-shadow-cwd", "events": []map[string]any{{"timestamp": "2026-03-10T10:05:01Z"}}},
				"diff": map[string]any{
					"state_equal":              true,
					"event_types_equal":        true,
					"event_count_delta":        0,
					"primary_timeline_seconds": 0.2,
					"shadow_timeline_seconds":  0.22,
				},
			},
		},
	})
	writeJSONFixture(t, filepath.Join(reports, "rollback-trigger-surface.json"), map[string]any{
		"summary": map[string]any{
			"status":                    "manual-only",
			"automation_boundary":       "manual",
			"automated_rollback_trigger": false,
			"distinctions":              map[string]any{},
		},
	})
	writeJSONFixture(t, filepath.Join(reports, "live-shadow-mirror-scorecard.json"), map[string]any{
		"summary": map[string]any{
			"latest_evidence_timestamp": "2026-03-10T10:05:01Z",
			"total_evidence_runs":       2,
			"parity_ok_count":           2,
			"drift_detected_count":      0,
			"matrix_total":              1,
			"matrix_mismatched":         0,
			"fresh_inputs":              2,
			"stale_inputs":              0,
		},
		"freshness":           []map[string]any{{"status": "fresh"}, {"status": "fresh"}},
		"cutover_checkpoints": []map[string]any{{"name": "ok", "passed": true}},
	})

	script := filepath.Join(repoRoot, "scripts", "migration", "export_live_shadow_bundle.py")
	cmd := exec.Command("python3", script, "--run-id", "20260310T100501Z")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run export_live_shadow_bundle.py from cwd: %v\n%s", err, output)
	}

	var summary struct {
		RunID    string `json:"run_id"`
		Artifacts struct {
			ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		} `json:"artifacts"`
	}
	readJSONFile(t, filepath.Join(reports, "live-shadow-summary.json"), &summary)
	if summary.RunID != "20260310T100501Z" || !strings.HasSuffix(summary.Artifacts.ShadowCompareReportPath, "shadow-compare-report.json") {
		t.Fatalf("unexpected cwd summary: %+v", summary)
	}
}

func TestLane8CheckedInLiveShadowBundleMatchesExpectedShape(t *testing.T) {
	t.Parallel()

	var manifest struct {
		Latest struct {
			RunID     string `json:"run_id"`
			Status    string `json:"status"`
			Severity  string `json:"severity"`
			Summary struct {
				DriftDetectedCount int `json:"drift_detected_count"`
				StaleInputs        int `json:"stale_inputs"`
			} `json:"summary"`
			Artifacts struct {
				LiveShadowScorecardPath string `json:"live_shadow_scorecard_path"`
			} `json:"artifacts"`
		} `json:"latest"`
		DriftRollup struct {
			Status  string `json:"status"`
			Summary struct {
				HighestSeverity string `json:"highest_severity"`
				LatestRunID     string `json:"latest_run_id"`
			} `json:"summary"`
		} `json:"drift_rollup"`
	}
	readJSONFile(t, filepath.Join(repoRoot(t), "docs", "reports", "live-shadow-index.json"), &manifest)
	if manifest.Latest.RunID == "" || manifest.Latest.Status != "parity-ok" || manifest.Latest.Severity != "none" {
		t.Fatalf("unexpected latest live shadow manifest: %+v", manifest)
	}
	if manifest.Latest.Summary.DriftDetectedCount != 0 || manifest.Latest.Summary.StaleInputs != 0 {
		t.Fatalf("unexpected latest live shadow summary: %+v", manifest.Latest.Summary)
	}
	if !strings.HasSuffix(manifest.Latest.Artifacts.LiveShadowScorecardPath, "live-shadow-mirror-scorecard.json") {
		t.Fatalf("unexpected latest live shadow artifacts: %+v", manifest.Latest.Artifacts)
	}
	if manifest.DriftRollup.Status != "parity-ok" || manifest.DriftRollup.Summary.HighestSeverity != "none" || manifest.DriftRollup.Summary.LatestRunID != manifest.Latest.RunID {
		t.Fatalf("unexpected drift rollup manifest: %+v", manifest.DriftRollup)
	}
}
