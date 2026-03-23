package regression

import (
	"path/filepath"
	"testing"
)

func TestLiveShadowIndexStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	indexPath := filepath.Join(repoRoot, "docs", "reports", "live-shadow-index.json")

	type artifactPaths struct {
		ShadowCompareReportPath    string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath     string `json:"shadow_matrix_report_path"`
		LiveShadowScorecardPath    string `json:"live_shadow_scorecard_path"`
		RollbackTriggerSurfacePath string `json:"rollback_trigger_surface_path"`
	}

	type freshnessEntry struct {
		Name                    string  `json:"name"`
		ReportPath              string  `json:"report_path"`
		LatestEvidenceTimestamp string  `json:"latest_evidence_timestamp"`
		AgeHours                float64 `json:"age_hours"`
		FreshnessSLOHours       int     `json:"freshness_slo_hours"`
		Status                  string  `json:"status"`
	}

	type runSummary struct {
		RunID                   string `json:"run_id"`
		GeneratedAt             string `json:"generated_at"`
		Status                  string `json:"status"`
		Severity                string `json:"severity"`
		BundlePath              string `json:"bundle_path"`
		SummaryPath             string `json:"summary_path"`
		LatestEvidenceTimestamp string `json:"latest_evidence_timestamp"`
		DriftDetectedCount      int    `json:"drift_detected_count"`
		StaleInputs             int    `json:"stale_inputs"`
	}

	type checkpoint struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	}

	var payload struct {
		Latest struct {
			RunID                   string           `json:"run_id"`
			GeneratedAt             string           `json:"generated_at"`
			Status                  string           `json:"status"`
			Severity                string           `json:"severity"`
			BundlePath              string           `json:"bundle_path"`
			SummaryPath             string           `json:"summary_path"`
			Artifacts               artifactPaths    `json:"artifacts"`
			LatestEvidenceTimestamp string           `json:"latest_evidence_timestamp"`
			Freshness               []freshnessEntry `json:"freshness"`
			Summary                 struct {
				TotalEvidenceRuns  int `json:"total_evidence_runs"`
				ParityOKCount      int `json:"parity_ok_count"`
				DriftDetectedCount int `json:"drift_detected_count"`
				MatrixTotal        int `json:"matrix_total"`
				MatrixMismatched   int `json:"matrix_mismatched"`
				StaleInputs        int `json:"stale_inputs"`
				FreshInputs        int `json:"fresh_inputs"`
			} `json:"summary"`
			CompareTraceID     string       `json:"compare_trace_id"`
			MatrixTraceIDs     []string     `json:"matrix_trace_ids"`
			CutoverCheckpoints []checkpoint `json:"cutover_checkpoints"`
			CloseoutCommands   []string     `json:"closeout_commands"`
		} `json:"latest"`
		RecentRuns  []runSummary `json:"recent_runs"`
		DriftRollup struct {
			GeneratedAt string `json:"generated_at"`
			Status      string `json:"status"`
			WindowSize  int    `json:"window_size"`
			Summary     struct {
				RecentRunCount    int            `json:"recent_run_count"`
				DriftDetectedRuns int            `json:"drift_detected_runs"`
				StaleRuns         int            `json:"stale_runs"`
				HighestSeverity   string         `json:"highest_severity"`
				StatusCounts      map[string]int `json:"status_counts"`
				LatestRunID       string         `json:"latest_run_id"`
			} `json:"summary"`
			RecentRuns []runSummary `json:"recent_runs"`
		} `json:"drift_rollup"`
	}

	readJSONFile(t, indexPath, &payload)

	if payload.Latest.RunID != "20260313T085655Z" || payload.Latest.GeneratedAt != "2026-03-17T02:35:33.529497Z" || payload.Latest.Status != "parity-ok" || payload.Latest.Severity != "none" {
		t.Fatalf("unexpected latest live-shadow metadata: %+v", payload.Latest)
	}
	if payload.Latest.BundlePath != "docs/reports/live-shadow-runs/20260313T085655Z" || payload.Latest.SummaryPath != "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" {
		t.Fatalf("unexpected latest live-shadow bundle paths: %+v", payload.Latest)
	}
	if payload.Latest.Artifacts.ShadowCompareReportPath != "docs/reports/live-shadow-runs/20260313T085655Z/shadow-compare-report.json" ||
		payload.Latest.Artifacts.ShadowMatrixReportPath != "docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json" ||
		payload.Latest.Artifacts.LiveShadowScorecardPath != "docs/reports/live-shadow-runs/20260313T085655Z/live-shadow-mirror-scorecard.json" ||
		payload.Latest.Artifacts.RollbackTriggerSurfacePath != "docs/reports/live-shadow-runs/20260313T085655Z/rollback-trigger-surface.json" {
		t.Fatalf("unexpected latest live-shadow artifact paths: %+v", payload.Latest.Artifacts)
	}
	if payload.Latest.LatestEvidenceTimestamp != "2026-03-13T16:56:55.415367+08:00" {
		t.Fatalf("unexpected latest evidence timestamp: %s", payload.Latest.LatestEvidenceTimestamp)
	}

	if len(payload.Latest.Freshness) != 2 {
		t.Fatalf("unexpected freshness entries: %+v", payload.Latest.Freshness)
	}
	if payload.Latest.Freshness[0].Name != "shadow-compare-report" ||
		payload.Latest.Freshness[0].ReportPath != "bigclaw-go/docs/reports/shadow-compare-report.json" ||
		payload.Latest.Freshness[0].LatestEvidenceTimestamp != "2026-03-13T15:53:21.403765+08:00" ||
		payload.Latest.Freshness[0].AgeHours != 80.08 ||
		payload.Latest.Freshness[0].FreshnessSLOHours != 168 ||
		payload.Latest.Freshness[0].Status != "fresh" {
		t.Fatalf("unexpected first freshness entry: %+v", payload.Latest.Freshness[0])
	}
	if payload.Latest.Freshness[1].Name != "shadow-matrix-report" ||
		payload.Latest.Freshness[1].ReportPath != "bigclaw-go/docs/reports/shadow-matrix-report.json" ||
		payload.Latest.Freshness[1].LatestEvidenceTimestamp != "2026-03-13T16:56:55.415367+08:00" ||
		payload.Latest.Freshness[1].AgeHours != 79.02 ||
		payload.Latest.Freshness[1].FreshnessSLOHours != 168 ||
		payload.Latest.Freshness[1].Status != "fresh" {
		t.Fatalf("unexpected second freshness entry: %+v", payload.Latest.Freshness[1])
	}

	if payload.Latest.Summary.TotalEvidenceRuns != 4 ||
		payload.Latest.Summary.ParityOKCount != 4 ||
		payload.Latest.Summary.DriftDetectedCount != 0 ||
		payload.Latest.Summary.MatrixTotal != 3 ||
		payload.Latest.Summary.MatrixMismatched != 0 ||
		payload.Latest.Summary.StaleInputs != 0 ||
		payload.Latest.Summary.FreshInputs != 2 {
		t.Fatalf("unexpected latest summary rollup: %+v", payload.Latest.Summary)
	}
	if payload.Latest.CompareTraceID != "shadow-compare-sample" {
		t.Fatalf("unexpected compare trace ID: %s", payload.Latest.CompareTraceID)
	}
	if len(payload.Latest.MatrixTraceIDs) != 3 ||
		payload.Latest.MatrixTraceIDs[0] != "shadow-compare-sample-m1" ||
		payload.Latest.MatrixTraceIDs[1] != "shadow-budget-sample-m2" ||
		payload.Latest.MatrixTraceIDs[2] != "shadow-validation-sample-m3" {
		t.Fatalf("unexpected matrix trace IDs: %+v", payload.Latest.MatrixTraceIDs)
	}
	if len(payload.Latest.CutoverCheckpoints) != 5 {
		t.Fatalf("unexpected cutover checkpoints: %+v", payload.Latest.CutoverCheckpoints)
	}
	if payload.Latest.CutoverCheckpoints[0].Name != "single_compare_matches_terminal_state_and_event_sequence" || !payload.Latest.CutoverCheckpoints[0].Passed {
		t.Fatalf("unexpected first cutover checkpoint: %+v", payload.Latest.CutoverCheckpoints[0])
	}
	if payload.Latest.CutoverCheckpoints[4].Name != "matrix_includes_corpus_coverage_overlay" || payload.Latest.CutoverCheckpoints[4].Detail != "corpus_slice_count=4" {
		t.Fatalf("unexpected final cutover checkpoint: %+v", payload.Latest.CutoverCheckpoints[4])
	}
	if len(payload.Latest.CloseoutCommands) != 4 ||
		payload.Latest.CloseoutCommands[0] != "cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --pretty" ||
		payload.Latest.CloseoutCommands[1] != "cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py" ||
		payload.Latest.CloseoutCommands[2] != "cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned" ||
		payload.Latest.CloseoutCommands[3] != "git push origin <branch> && git log -1 --stat" {
		t.Fatalf("unexpected closeout commands: %+v", payload.Latest.CloseoutCommands)
	}

	if len(payload.RecentRuns) != 1 {
		t.Fatalf("unexpected recent runs: %+v", payload.RecentRuns)
	}
	if payload.RecentRuns[0].RunID != payload.Latest.RunID ||
		payload.RecentRuns[0].GeneratedAt != payload.Latest.GeneratedAt ||
		payload.RecentRuns[0].Status != payload.Latest.Status ||
		payload.RecentRuns[0].Severity != payload.Latest.Severity ||
		payload.RecentRuns[0].BundlePath != payload.Latest.BundlePath ||
		payload.RecentRuns[0].SummaryPath != payload.Latest.SummaryPath {
		t.Fatalf("recent run drifted from latest summary: recent=%+v latest=%+v", payload.RecentRuns[0], payload.Latest)
	}

	if payload.DriftRollup.GeneratedAt != "2026-03-17T02:35:33.537339Z" || payload.DriftRollup.Status != "parity-ok" || payload.DriftRollup.WindowSize != 5 {
		t.Fatalf("unexpected drift rollup metadata: %+v", payload.DriftRollup)
	}
	if payload.DriftRollup.Summary.RecentRunCount != 1 ||
		payload.DriftRollup.Summary.DriftDetectedRuns != 0 ||
		payload.DriftRollup.Summary.StaleRuns != 0 ||
		payload.DriftRollup.Summary.HighestSeverity != "none" ||
		payload.DriftRollup.Summary.StatusCounts["parity_ok"] != 1 ||
		payload.DriftRollup.Summary.StatusCounts["attention_needed"] != 0 ||
		payload.DriftRollup.Summary.LatestRunID != payload.Latest.RunID {
		t.Fatalf("unexpected drift rollup summary: %+v", payload.DriftRollup.Summary)
	}
	if len(payload.DriftRollup.RecentRuns) != 1 {
		t.Fatalf("unexpected drift rollup recent runs: %+v", payload.DriftRollup.RecentRuns)
	}
	if payload.DriftRollup.RecentRuns[0].RunID != payload.Latest.RunID ||
		payload.DriftRollup.RecentRuns[0].LatestEvidenceTimestamp != payload.Latest.LatestEvidenceTimestamp ||
		payload.DriftRollup.RecentRuns[0].DriftDetectedCount != 0 ||
		payload.DriftRollup.RecentRuns[0].StaleInputs != 0 ||
		payload.DriftRollup.RecentRuns[0].BundlePath != payload.Latest.BundlePath ||
		payload.DriftRollup.RecentRuns[0].SummaryPath != payload.Latest.SummaryPath {
		t.Fatalf("unexpected drift rollup recent run: %+v", payload.DriftRollup.RecentRuns[0])
	}
}
