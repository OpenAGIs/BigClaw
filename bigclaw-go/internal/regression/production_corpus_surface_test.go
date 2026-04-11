package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

type shadowMatrixReport struct {
	Total      int `json:"total"`
	Matched    int `json:"matched"`
	Mismatched int `json:"mismatched"`
	Inputs     struct {
		FixtureTaskCount int    `json:"fixture_task_count"`
		CorpusSliceCount int    `json:"corpus_slice_count"`
		ManifestName     string `json:"manifest_name"`
	} `json:"inputs"`
	Results []struct {
		TraceID string `json:"trace_id"`
		Primary struct {
			Status struct {
				State string `json:"state"`
			} `json:"status"`
		} `json:"primary"`
		Shadow struct {
			Status struct {
				State string `json:"state"`
			} `json:"status"`
		} `json:"shadow"`
	} `json:"results"`
}

type shadowCorpusManifest struct {
	Name   string        `json:"name"`
	Slices []interface{} `json:"slices"`
}

type driftRollup struct {
	Summary struct {
		Status            string `json:"status"`
		RecentRunCount    int    `json:"recent_run_count"`
		DriftDetectedRuns int    `json:"drift_detected_runs"`
		HighestSeverity   string `json:"highest_severity"`
		LatestRunID       string `json:"latest_run_id"`
		StatusCounts      struct {
			ParityOK        int `json:"parity_ok"`
			AttentionNeeded int `json:"attention_needed"`
		} `json:"status_counts"`
	} `json:"summary"`
	RecentRuns []struct {
		RunID    string `json:"run_id"`
		Status   string `json:"status"`
		Severity string `json:"severity"`
	} `json:"recent_runs"`
}

func TestProductionCorpusMatrixManifestAlignment(t *testing.T) {
	repoRoot := repoRoot(t)
	matrixPath := filepath.Join(repoRoot, "docs", "reports", "shadow-matrix-report.json")
	manifestPath := filepath.Join(repoRoot, "examples", "shadow-corpus-manifest.json")

	var report shadowMatrixReport
	readJSONFile(t, matrixPath, &report)

	var manifest shadowCorpusManifest
	readJSONFile(t, manifestPath, &manifest)

	if report.Total != report.Matched+report.Mismatched {
		t.Fatalf("shadow matrix total %d != matched+mismatched %d+%d", report.Total, report.Matched, report.Mismatched)
	}
	if report.Total != len(report.Results) {
		t.Fatalf("expected %d results, got %d", report.Total, len(report.Results))
	}
	if report.Inputs.ManifestName != manifest.Name {
		t.Fatalf("matrix manifest %q != manifest name %q", report.Inputs.ManifestName, manifest.Name)
	}
	if report.Inputs.CorpusSliceCount != len(manifest.Slices) {
		t.Fatalf("matrix corpus slice count %d != manifest slices %d", report.Inputs.CorpusSliceCount, len(manifest.Slices))
	}

	for _, result := range report.Results {
		if result.Primary.Status.State != "succeeded" {
			t.Fatalf("primary trace %q state %q, want succeeded", result.TraceID, result.Primary.Status.State)
		}
		if result.Shadow.Status.State != "succeeded" {
			t.Fatalf("shadow trace %q state %q, want succeeded", result.TraceID, result.Shadow.Status.State)
		}
	}
}

func TestProductionCorpusDriftRollupStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	rollupPath := filepath.Join(repoRoot, "docs", "reports", "live-shadow-drift-rollup.json")

	var rollup driftRollup
	readJSONFile(t, rollupPath, &rollup)

	if rollup.Summary.RecentRunCount != rollup.Summary.StatusCounts.ParityOK {
		t.Fatalf("recent run count %d != parity_ok %d", rollup.Summary.RecentRunCount, rollup.Summary.StatusCounts.ParityOK)
	}
	if len(rollup.RecentRuns) != rollup.Summary.RecentRunCount {
		t.Fatalf("recent runs %d != summary count %d", len(rollup.RecentRuns), rollup.Summary.RecentRunCount)
	}
	for _, run := range rollup.RecentRuns {
		if run.RunID != rollup.Summary.LatestRunID {
			t.Fatalf("recent run id %q did not match summary latest %q", run.RunID, rollup.Summary.LatestRunID)
		}
	}
}

func TestProductionCorpusDigestReferencesRemainIntact(t *testing.T) {
	repoRoot := repoRoot(t)
	digestRel := "docs/reports/production-corpus-migration-coverage-digest.md"
	body := readRepoFile(t, repoRoot, digestRel)

	required := []string{
		"shadow-matrix-report.json",
		"shadow-compare-report.json",
		"docs/migration-shadow.md",
		"fixture-backed evidence only",
		"no real production issue/task corpus coverage",
		"examples/shadow-corpus-manifest.json",
	}
	for _, needle := range required {
		if !strings.Contains(body, needle) {
			t.Fatalf("%s missing substring %q", digestRel, needle)
		}
	}
}
