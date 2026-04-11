package regression

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type liveShadowScorecardSurface struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath  string `json:"shadow_matrix_report_path"`
		GeneratorScript         string `json:"generator_script"`
	} `json:"evidence_inputs"`
	Summary struct {
		TotalEvidenceRuns         int    `json:"total_evidence_runs"`
		ParityOKCount             int    `json:"parity_ok_count"`
		DriftDetectedCount        int    `json:"drift_detected_count"`
		MatrixTotal               int    `json:"matrix_total"`
		MatrixMatched             int    `json:"matrix_matched"`
		MatrixMismatched          int    `json:"matrix_mismatched"`
		CorpusCoveragePresent     bool   `json:"corpus_coverage_present"`
		CorpusUncoveredSliceCount int    `json:"corpus_uncovered_slice_count"`
		LatestEvidenceTimestamp   string `json:"latest_evidence_timestamp"`
		FreshInputs               int    `json:"fresh_inputs"`
		StaleInputs               int    `json:"stale_inputs"`
	} `json:"summary"`
	Freshness []struct {
		Name       string `json:"name"`
		ReportPath string `json:"report_path"`
		Status     string `json:"status"`
	} `json:"freshness"`
	ParityEntries []struct {
		EntryType string `json:"entry_type"`
		Label     string `json:"label"`
		TraceID   string `json:"trace_id"`
		Parity    struct {
			Status string `json:"status"`
		} `json:"parity"`
	} `json:"parity_entries"`
	CutoverCheckpoints []struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	} `json:"cutover_checkpoints"`
	Limitations []string `json:"limitations"`
	FutureWork  []string `json:"future_work"`
}

type liveShadowBundleSummarySurface struct {
	RunID                   string `json:"run_id"`
	GeneratedAt             string `json:"generated_at"`
	Status                  string `json:"status"`
	Severity                string `json:"severity"`
	BundlePath              string `json:"bundle_path"`
	SummaryPath             string `json:"summary_path"`
	LatestEvidenceTimestamp string `json:"latest_evidence_timestamp"`
	Artifacts               struct {
		ShadowCompareReportPath    string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath     string `json:"shadow_matrix_report_path"`
		LiveShadowScorecardPath    string `json:"live_shadow_scorecard_path"`
		RollbackTriggerSurfacePath string `json:"rollback_trigger_surface_path"`
	} `json:"artifacts"`
	Freshness []struct {
		Name       string `json:"name"`
		ReportPath string `json:"report_path"`
		Status     string `json:"status"`
	} `json:"freshness"`
	Summary struct {
		TotalEvidenceRuns  int `json:"total_evidence_runs"`
		ParityOKCount      int `json:"parity_ok_count"`
		DriftDetectedCount int `json:"drift_detected_count"`
		MatrixTotal        int `json:"matrix_total"`
		MatrixMismatched   int `json:"matrix_mismatched"`
		StaleInputs        int `json:"stale_inputs"`
		FreshInputs        int `json:"fresh_inputs"`
	} `json:"summary"`
	RollbackTriggerSurface struct {
		Status                   string `json:"status"`
		AutomationBoundary       string `json:"automation_boundary"`
		AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
		Distinctions             struct {
			Blockers        int `json:"blockers"`
			Warnings        int `json:"warnings"`
			ManualOnlyPaths int `json:"manual_only_paths"`
		} `json:"distinctions"`
		Issue struct {
			ID    string `json:"id"`
			Slug  string `json:"slug"`
			Title string `json:"title"`
		} `json:"issue"`
		DigestPath  string `json:"digest_path"`
		SummaryPath string `json:"summary_path"`
	} `json:"rollback_trigger_surface"`
	CompareTraceID     string   `json:"compare_trace_id"`
	MatrixTraceIDs     []string `json:"matrix_trace_ids"`
	CutoverCheckpoints []struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	} `json:"cutover_checkpoints"`
	CloseoutCommands []string `json:"closeout_commands"`
}

type liveShadowIndexSurface struct {
	Latest     liveShadowBundleSummarySurface `json:"latest"`
	RecentRuns []struct {
		RunID       string `json:"run_id"`
		Status      string `json:"status"`
		Severity    string `json:"severity"`
		BundlePath  string `json:"bundle_path"`
		SummaryPath string `json:"summary_path"`
	} `json:"recent_runs"`
	DriftRollup struct {
		Status  string `json:"status"`
		Summary struct {
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
			RunID              string `json:"run_id"`
			Status             string `json:"status"`
			Severity           string `json:"severity"`
			DriftDetectedCount int    `json:"drift_detected_count"`
			StaleInputs        int    `json:"stale_inputs"`
			BundlePath         string `json:"bundle_path"`
			SummaryPath        string `json:"summary_path"`
		} `json:"recent_runs"`
	} `json:"drift_rollup"`
}

func TestLiveShadowScorecardBundleStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var canonical liveShadowScorecardSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-mirror-scorecard.json"), &canonical)

	var bundled liveShadowScorecardSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-runs", "20260313T085655Z", "live-shadow-mirror-scorecard.json"), &bundled)

	if canonical.Ticket != "BIG-PAR-092" || canonical.Status != "repo-native-live-shadow-scorecard" {
		t.Fatalf("unexpected canonical live-shadow scorecard identity: %+v", canonical)
	}
	if canonical.EvidenceInputs.GeneratorScript != "go run ./cmd/bigclawctl automation migration live-shadow-scorecard" {
		t.Fatalf("unexpected scorecard generator script: %+v", canonical.EvidenceInputs)
	}
	if canonical.Summary.TotalEvidenceRuns != 4 ||
		canonical.Summary.ParityOKCount != 4 ||
		canonical.Summary.DriftDetectedCount != 0 ||
		canonical.Summary.MatrixTotal != 3 ||
		canonical.Summary.MatrixMatched != 3 ||
		canonical.Summary.MatrixMismatched != 0 ||
		!canonical.Summary.CorpusCoveragePresent ||
		canonical.Summary.CorpusUncoveredSliceCount != 1 ||
		canonical.Summary.FreshInputs != 2 ||
		canonical.Summary.StaleInputs != 0 {
		t.Fatalf("unexpected canonical live-shadow scorecard summary: %+v", canonical.Summary)
	}
	if len(canonical.Freshness) != 2 || canonical.Freshness[0].Status != "fresh" || canonical.Freshness[1].Status != "fresh" {
		t.Fatalf("unexpected scorecard freshness payload: %+v", canonical.Freshness)
	}
	if len(canonical.ParityEntries) != 4 ||
		canonical.ParityEntries[0].EntryType != "single-compare" ||
		canonical.ParityEntries[0].TraceID != "shadow-compare-sample" ||
		canonical.ParityEntries[1].EntryType != "matrix-row" ||
		canonical.ParityEntries[3].TraceID != "shadow-validation-sample-m3" {
		t.Fatalf("unexpected scorecard parity entries: %+v", canonical.ParityEntries)
	}
	for _, checkpoint := range canonical.CutoverCheckpoints {
		if !checkpoint.Passed {
			t.Fatalf("expected cutover checkpoint to pass, got %+v", checkpoint)
		}
	}
	if len(canonical.CutoverCheckpoints) != 5 {
		t.Fatalf("unexpected cutover checkpoint count: %+v", canonical.CutoverCheckpoints)
	}
	if len(canonical.Limitations) != 3 || !strings.Contains(canonical.Limitations[0], "repo-native only") {
		t.Fatalf("unexpected scorecard limitations: %+v", canonical.Limitations)
	}
	if len(canonical.FutureWork) != 3 || !strings.Contains(canonical.FutureWork[2], "rollback automation") {
		t.Fatalf("unexpected scorecard future work: %+v", canonical.FutureWork)
	}

	if !reflect.DeepEqual(canonical.Summary, bundled.Summary) ||
		!reflect.DeepEqual(canonical.Freshness, bundled.Freshness) ||
		!reflect.DeepEqual(canonical.CutoverCheckpoints, bundled.CutoverCheckpoints) ||
		!reflect.DeepEqual(canonical.Limitations, bundled.Limitations) ||
		!reflect.DeepEqual(canonical.FutureWork, bundled.FutureWork) {
		t.Fatalf("bundled live-shadow scorecard drifted from canonical scorecard")
	}
}

func TestLiveShadowBundleSummaryAndIndexStayAligned(t *testing.T) {
	root := repoRoot(t)

	var summary liveShadowBundleSummarySurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-summary.json"), &summary)

	var bundledSummary liveShadowBundleSummarySurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-runs", "20260313T085655Z", "summary.json"), &bundledSummary)

	var index liveShadowIndexSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "live-shadow-index.json"), &index)

	canonicalRollback := readRollbackTriggerSurface(t, root)
	bundledRollback := readLiveShadowRollbackTriggerSurface(t, root)

	if !reflect.DeepEqual(summary, bundledSummary) {
		t.Fatalf("bundled live-shadow summary drifted from canonical summary")
	}
	if !reflect.DeepEqual(canonicalRollback, bundledRollback) {
		t.Fatalf("bundled rollback trigger surface drifted from canonical surface")
	}

	if summary.RunID != "20260313T085655Z" ||
		summary.Status != "parity-ok" ||
		summary.Severity != "none" ||
		summary.BundlePath != "docs/reports/live-shadow-runs/20260313T085655Z" ||
		summary.SummaryPath != "docs/reports/live-shadow-runs/20260313T085655Z/summary.json" {
		t.Fatalf("unexpected live-shadow summary identity: %+v", summary)
	}
	if summary.Summary.TotalEvidenceRuns != 4 ||
		summary.Summary.ParityOKCount != 4 ||
		summary.Summary.DriftDetectedCount != 0 ||
		summary.Summary.MatrixTotal != 3 ||
		summary.Summary.MatrixMismatched != 0 ||
		summary.Summary.FreshInputs != 2 ||
		summary.Summary.StaleInputs != 0 {
		t.Fatalf("unexpected live-shadow summary counters: %+v", summary.Summary)
	}
	if summary.RollbackTriggerSurface.Status != "manual-review-required" ||
		summary.RollbackTriggerSurface.AutomationBoundary != "manual_only" ||
		summary.RollbackTriggerSurface.AutomatedRollbackTrigger ||
		summary.RollbackTriggerSurface.Distinctions.Blockers != 3 ||
		summary.RollbackTriggerSurface.Distinctions.Warnings != 1 ||
		summary.RollbackTriggerSurface.Distinctions.ManualOnlyPaths != 2 ||
		summary.RollbackTriggerSurface.Issue.ID != "OPE-254" ||
		summary.RollbackTriggerSurface.Issue.Slug != "BIG-PAR-088" ||
		summary.RollbackTriggerSurface.Issue.Title != "tenant-scoped rollback guardrails and trigger surface" ||
		summary.RollbackTriggerSurface.DigestPath != "" ||
		summary.RollbackTriggerSurface.SummaryPath != "docs/reports/rollback-trigger-surface.json" {
		t.Fatalf("unexpected live-shadow summary rollback payload: %+v", summary.RollbackTriggerSurface)
	}
	if summary.CompareTraceID != "shadow-compare-sample" ||
		len(summary.MatrixTraceIDs) != 3 ||
		summary.MatrixTraceIDs[0] != "shadow-compare-sample-m1" ||
		summary.MatrixTraceIDs[2] != "shadow-validation-sample-m3" {
		t.Fatalf("unexpected live-shadow trace lineage: %+v", summary.MatrixTraceIDs)
	}
	if len(summary.CutoverCheckpoints) != 5 || len(summary.CloseoutCommands) != 4 {
		t.Fatalf("unexpected live-shadow closeout/checkpoint data: checkpoints=%d commands=%d", len(summary.CutoverCheckpoints), len(summary.CloseoutCommands))
	}
	for _, command := range []string{
		"go run ./cmd/bigclawctl automation migration live-shadow-scorecard --pretty",
		"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
		"go test ./internal/regression -run TestRollbackDocsStayAligned",
		"git push origin <branch> && git log -1 --stat",
	} {
		if !containsString(summary.CloseoutCommands, command) {
			t.Fatalf("expected closeout commands to include %q, got %+v", command, summary.CloseoutCommands)
		}
	}

	for _, relative := range []string{
		summary.SummaryPath,
		summary.Artifacts.ShadowCompareReportPath,
		summary.Artifacts.ShadowMatrixReportPath,
		summary.Artifacts.LiveShadowScorecardPath,
		summary.Artifacts.RollbackTriggerSurfacePath,
	} {
		if !strings.HasPrefix(relative, summary.BundlePath) {
			t.Fatalf("expected %s to live under bundle path %s", relative, summary.BundlePath)
		}
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected referenced artifact %s to exist: %v", relative, err)
		}
	}

	if index.Latest.RunID != summary.RunID ||
		index.Latest.Status != summary.Status ||
		index.Latest.Severity != summary.Severity ||
		index.Latest.BundlePath != summary.BundlePath ||
		index.Latest.SummaryPath != summary.SummaryPath ||
		index.Latest.CompareTraceID != summary.CompareTraceID ||
		!reflect.DeepEqual(index.Latest.MatrixTraceIDs, summary.MatrixTraceIDs) {
		t.Fatalf("unexpected latest live-shadow index payload: %+v", index.Latest)
	}
	if index.Latest.RollbackTriggerSurface.Issue.ID != "OPE-254" ||
		index.Latest.RollbackTriggerSurface.Issue.Slug != "BIG-PAR-088" ||
		index.Latest.RollbackTriggerSurface.Issue.Title != "tenant-scoped rollback guardrails and trigger surface" ||
		index.Latest.RollbackTriggerSurface.SummaryPath != "docs/reports/rollback-trigger-surface.json" {
		t.Fatalf("unexpected latest index rollback payload: %+v", index.Latest.RollbackTriggerSurface)
	}
	if len(index.RecentRuns) != 1 ||
		index.RecentRuns[0].RunID != summary.RunID ||
		index.RecentRuns[0].SummaryPath != summary.SummaryPath ||
		index.RecentRuns[0].BundlePath != summary.BundlePath {
		t.Fatalf("unexpected recent live-shadow runs payload: %+v", index.RecentRuns)
	}
	if index.DriftRollup.Status != "parity-ok" ||
		index.DriftRollup.Summary.RecentRunCount != 1 ||
		index.DriftRollup.Summary.DriftDetectedRuns != 0 ||
		index.DriftRollup.Summary.HighestSeverity != "none" ||
		index.DriftRollup.Summary.LatestRunID != summary.RunID ||
		index.DriftRollup.Summary.StatusCounts.ParityOK != 1 ||
		index.DriftRollup.Summary.StatusCounts.AttentionNeeded != 0 {
		t.Fatalf("unexpected live-shadow drift rollup summary: %+v", index.DriftRollup.Summary)
	}
	if len(index.DriftRollup.RecentRuns) != 1 ||
		index.DriftRollup.RecentRuns[0].RunID != summary.RunID ||
		index.DriftRollup.RecentRuns[0].DriftDetectedCount != 0 ||
		index.DriftRollup.RecentRuns[0].StaleInputs != 0 ||
		index.DriftRollup.RecentRuns[0].SummaryPath != summary.SummaryPath ||
		index.DriftRollup.RecentRuns[0].BundlePath != summary.BundlePath {
		t.Fatalf("unexpected live-shadow drift rollup recent runs: %+v", index.DriftRollup.RecentRuns)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
