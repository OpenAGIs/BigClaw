package liveshadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	TimelineDriftToleranceSeconds = 0.25
	EvidenceFreshnessSLOHours     = 168
)

type eventRecord struct {
	Timestamp string `json:"timestamp"`
}

type taskReport struct {
	TaskID string        `json:"task_id"`
	Events []eventRecord `json:"events"`
}

type diffReport struct {
	StateEqual             bool    `json:"state_equal"`
	EventTypesEqual        bool    `json:"event_types_equal"`
	EventCountDelta        int     `json:"event_count_delta"`
	PrimaryTimelineSeconds float64 `json:"primary_timeline_seconds"`
	ShadowTimelineSeconds  float64 `json:"shadow_timeline_seconds"`
}

type CompareReport struct {
	TraceID string     `json:"trace_id"`
	Primary taskReport `json:"primary"`
	Shadow  taskReport `json:"shadow"`
	Diff    diffReport `json:"diff"`
}

type matrixResult struct {
	TraceID     string     `json:"trace_id"`
	SourceFile  string     `json:"source_file"`
	SourceKind  any        `json:"source_kind"`
	TaskShape   any        `json:"task_shape"`
	CorpusSlice any        `json:"corpus_slice"`
	Primary     taskReport `json:"primary"`
	Shadow      taskReport `json:"shadow"`
	Diff        diffReport `json:"diff"`
}

type MatrixReport struct {
	Total          int            `json:"total"`
	Matched        int            `json:"matched"`
	Mismatched     int            `json:"mismatched"`
	Results        []matrixResult `json:"results"`
	CorpusCoverage map[string]any `json:"corpus_coverage"`
}

type FreshnessEntry struct {
	Name                    string   `json:"name"`
	ReportPath              string   `json:"report_path"`
	LatestEvidenceTimestamp *string  `json:"latest_evidence_timestamp"`
	AgeHours                *float64 `json:"age_hours"`
	FreshnessSLOHours       int      `json:"freshness_slo_hours"`
	Status                  string   `json:"status"`
}

type ParityStatus struct {
	Status                        string   `json:"status"`
	TimelineDeltaSeconds          float64  `json:"timeline_delta_seconds"`
	TimelineDriftToleranceSeconds float64  `json:"timeline_drift_tolerance_seconds"`
	Reasons                       []string `json:"reasons"`
}

type ParityEntry struct {
	EntryType     string       `json:"entry_type"`
	Label         any          `json:"label"`
	TraceID       string       `json:"trace_id"`
	SourceFile    any          `json:"source_file"`
	SourceKind    any          `json:"source_kind"`
	TaskShape     any          `json:"task_shape"`
	CorpusSlice   any          `json:"corpus_slice"`
	Parity        ParityStatus `json:"parity"`
	PrimaryTaskID string       `json:"primary_task_id"`
	ShadowTaskID  string       `json:"shadow_task_id"`
}

type Checkpoint struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type ScorecardSummary struct {
	TotalEvidenceRuns         int     `json:"total_evidence_runs"`
	ParityOKCount             int     `json:"parity_ok_count"`
	DriftDetectedCount        int     `json:"drift_detected_count"`
	MatrixTotal               int     `json:"matrix_total"`
	MatrixMatched             int     `json:"matrix_matched"`
	MatrixMismatched          int     `json:"matrix_mismatched"`
	CorpusCoveragePresent     bool    `json:"corpus_coverage_present"`
	CorpusUncoveredSliceCount any     `json:"corpus_uncovered_slice_count"`
	LatestEvidenceTimestamp   *string `json:"latest_evidence_timestamp"`
	FreshInputs               int     `json:"fresh_inputs"`
	StaleInputs               int     `json:"stale_inputs"`
}

type Scorecard struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath  string `json:"shadow_matrix_report_path"`
		GeneratorScript         string `json:"generator_script"`
	} `json:"evidence_inputs"`
	Summary            ScorecardSummary `json:"summary"`
	Freshness          []FreshnessEntry `json:"freshness"`
	ParityEntries      []ParityEntry    `json:"parity_entries"`
	CutoverCheckpoints []Checkpoint     `json:"cutover_checkpoints"`
	Limitations        []string         `json:"limitations"`
	FutureWork         []string         `json:"future_work"`
}

type rollbackSurface struct {
	Issue struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	} `json:"issue"`
	Summary struct {
		Status                   string `json:"status"`
		AutomationBoundary       string `json:"automation_boundary"`
		AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
		Distinctions             struct {
			Blockers        int `json:"blockers"`
			Warnings        int `json:"warnings"`
			ManualOnlyPaths int `json:"manual_only_paths"`
		} `json:"distinctions"`
	} `json:"summary"`
	SharedGuardrailSummary struct {
		DigestPath string `json:"digest_path"`
	} `json:"shared_guardrail_summary"`
}

type BundleSummary struct {
	RunID       string `json:"run_id"`
	GeneratedAt string `json:"generated_at"`
	Status      string `json:"status"`
	Severity    string `json:"severity"`
	BundlePath  string `json:"bundle_path"`
	SummaryPath string `json:"summary_path"`
	Artifacts   struct {
		ShadowCompareReportPath    string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath     string `json:"shadow_matrix_report_path"`
		LiveShadowScorecardPath    string `json:"live_shadow_scorecard_path"`
		RollbackTriggerSurfacePath string `json:"rollback_trigger_surface_path"`
	} `json:"artifacts"`
	LatestEvidenceTimestamp *string          `json:"latest_evidence_timestamp"`
	Freshness               []FreshnessEntry `json:"freshness"`
	Summary                 struct {
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
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"issue"`
		DigestPath  string `json:"digest_path"`
		SummaryPath string `json:"summary_path"`
	} `json:"rollback_trigger_surface"`
	CompareTraceID     string       `json:"compare_trace_id"`
	MatrixTraceIDs     []string     `json:"matrix_trace_ids"`
	CutoverCheckpoints []Checkpoint `json:"cutover_checkpoints"`
	CloseoutCommands   []string     `json:"closeout_commands"`
}

type RecentRun struct {
	RunID                   string  `json:"run_id"`
	GeneratedAt             string  `json:"generated_at"`
	Status                  string  `json:"status"`
	Severity                string  `json:"severity"`
	LatestEvidenceTimestamp *string `json:"latest_evidence_timestamp,omitempty"`
	DriftDetectedCount      int     `json:"drift_detected_count,omitempty"`
	StaleInputs             int     `json:"stale_inputs,omitempty"`
	BundlePath              string  `json:"bundle_path"`
	SummaryPath             string  `json:"summary_path"`
}

type Rollup struct {
	GeneratedAt string `json:"generated_at"`
	Status      string `json:"status"`
	WindowSize  int    `json:"window_size"`
	Summary     struct {
		RecentRunCount    int    `json:"recent_run_count"`
		DriftDetectedRuns int    `json:"drift_detected_runs"`
		StaleRuns         int    `json:"stale_runs"`
		HighestSeverity   string `json:"highest_severity"`
		StatusCounts      struct {
			ParityOK        int `json:"parity_ok"`
			AttentionNeeded int `json:"attention_needed"`
		} `json:"status_counts"`
		LatestRunID string `json:"latest_run_id"`
	} `json:"summary"`
	RecentRuns []RecentRun `json:"recent_runs"`
}

type Manifest struct {
	Latest      BundleSummary `json:"latest"`
	RecentRuns  []RecentRun   `json:"recent_runs"`
	DriftRollup Rollup        `json:"drift_rollup"`
}

type BundleOptions struct {
	GoRoot            string
	ShadowComparePath string
	ShadowMatrixPath  string
	ScorecardPath     string
	BundleRoot        string
	SummaryPath       string
	IndexPath         string
	ManifestPath      string
	RollupPath        string
	RunID             string
}

func BuildScorecard(repoRoot string, shadowCompareReportPath string, shadowMatrixReportPath string, generatedAt time.Time) (Scorecard, error) {
	var compare CompareReport
	if err := readJSON(resolveRepoPath(repoRoot, shadowCompareReportPath), &compare); err != nil {
		return Scorecard{}, err
	}
	var matrix MatrixReport
	if err := readJSON(resolveRepoPath(repoRoot, shadowMatrixReportPath), &matrix); err != nil {
		return Scorecard{}, err
	}

	parityEntries := []ParityEntry{buildCompareEntry(compare)}
	for _, item := range matrix.Results {
		parityEntries = append(parityEntries, buildMatrixEntry(item))
	}
	parityOKCount := 0
	for _, entry := range parityEntries {
		if entry.Parity.Status == "parity-ok" {
			parityOKCount++
		}
	}
	driftDetectedCount := len(parityEntries) - parityOKCount

	freshness := []FreshnessEntry{
		buildFreshnessEntry("shadow-compare-report", shadowCompareReportPath, compare.Primary.Events, compare.Shadow.Events, generatedAt),
		buildFreshnessEntryFromMatrix("shadow-matrix-report", shadowMatrixReportPath, matrix, generatedAt),
	}
	staleInputs := 0
	var latestEvidenceTimestamp *string
	for _, item := range freshness {
		if item.Status != "fresh" {
			staleInputs++
		}
		if item.LatestEvidenceTimestamp == nil {
			continue
		}
		if latestEvidenceTimestamp == nil || *item.LatestEvidenceTimestamp > *latestEvidenceTimestamp {
			value := *item.LatestEvidenceTimestamp
			latestEvidenceTimestamp = &value
		}
	}

	scorecard := Scorecard{
		GeneratedAt: utcISO(generatedAt),
		Ticket:      "BIG-PAR-092",
		Title:       "Live shadow mirror parity drift scorecard",
		Status:      "repo-native-live-shadow-scorecard",
		Summary: ScorecardSummary{
			TotalEvidenceRuns:         len(parityEntries),
			ParityOKCount:             parityOKCount,
			DriftDetectedCount:        driftDetectedCount,
			MatrixTotal:               matrix.Total,
			MatrixMatched:             matrix.Matched,
			MatrixMismatched:          matrix.Mismatched,
			CorpusCoveragePresent:     len(matrix.CorpusCoverage) > 0,
			CorpusUncoveredSliceCount: matrix.CorpusCoverage["uncovered_corpus_slice_count"],
			LatestEvidenceTimestamp:   latestEvidenceTimestamp,
			FreshInputs:               len(freshness) - staleInputs,
			StaleInputs:               staleInputs,
		},
		Freshness:     freshness,
		ParityEntries: parityEntries,
		Limitations: []string{
			"repo-native only: this scorecard summarizes checked-in shadow artifacts rather than an always-on production traffic mirror",
			"parity drift is measured from fixture-backed compare/matrix runs and optional corpus slices, not mirrored tenant traffic",
			"freshness is derived from the latest artifact event timestamps and should be treated as evidence recency, not live service health",
		},
		FutureWork: []string{
			"replace offline fixture submission with a real ingress mirror or tenant-scoped shadow routing control before treating this as cutover-proof traffic parity",
			"promote parity drift review from checked-in artifacts into a continuously refreshed operational surface",
			"pair this scorecard with rollback automation only after tenant-scoped rollback guardrails exist",
		},
	}
	scorecard.EvidenceInputs.ShadowCompareReportPath = shadowCompareReportPath
	scorecard.EvidenceInputs.ShadowMatrixReportPath = shadowMatrixReportPath
	scorecard.EvidenceInputs.GeneratorScript = "cd bigclaw-go && go run ./cmd/bigclawctl live-shadow scorecard"
	scorecard.CutoverCheckpoints = []Checkpoint{
		checkpoint(
			"single_compare_matches_terminal_state_and_event_sequence",
			compare.Diff.StateEqual && compare.Diff.EventTypesEqual,
			fmt.Sprintf("trace_id=%s", compare.TraceID),
		),
		checkpoint(
			"matrix_reports_no_state_or_event_sequence_mismatches",
			matrix.Mismatched == 0,
			fmt.Sprintf("matched=%d mismatched=%d", matrix.Matched, matrix.Mismatched),
		),
		checkpoint(
			"scorecard_detects_no_parity_drift",
			driftDetectedCount == 0,
			fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, driftDetectedCount),
		),
		checkpoint(
			"checked_in_evidence_is_fresh_enough_for_review",
			staleInputs == 0,
			fmt.Sprintf("freshness_statuses=%s", quoteStatuses(freshness)),
		),
		checkpoint(
			"matrix_includes_corpus_coverage_overlay",
			len(matrix.CorpusCoverage) > 0,
			fmt.Sprintf("corpus_slice_count=%d", intFromAny(matrix.CorpusCoverage["corpus_slice_count"])),
		),
	}
	return scorecard, nil
}

func WriteScorecard(path string, scorecard Scorecard) error {
	return writeJSON(path, scorecard)
}

func ExportBundle(opts BundleOptions, generatedAt time.Time) (Manifest, error) {
	goRoot := filepath.Clean(opts.GoRoot)
	var compare CompareReport
	if err := readJSON(filepath.Join(goRoot, opts.ShadowComparePath), &compare); err != nil {
		return Manifest{}, err
	}
	var matrix MatrixReport
	if err := readJSON(filepath.Join(goRoot, opts.ShadowMatrixPath), &matrix); err != nil {
		return Manifest{}, err
	}
	var scorecard Scorecard
	if err := readJSON(filepath.Join(goRoot, opts.ScorecardPath), &scorecard); err != nil {
		return Manifest{}, err
	}
	var rollback rollbackSurface
	if err := readJSON(filepath.Join(goRoot, "docs/reports/rollback-trigger-surface.json"), &rollback); err != nil {
		return Manifest{}, err
	}

	runID := opts.RunID
	if strings.TrimSpace(runID) == "" {
		if scorecard.Summary.LatestEvidenceTimestamp != nil {
			parsed, err := parseTime(*scorecard.Summary.LatestEvidenceTimestamp)
			if err == nil {
				runID = parsed.UTC().Format("20060102T150405Z")
			}
		}
		if runID == "" {
			runID = generatedAt.UTC().Format("20060102T150405Z")
		}
	}
	bundleDir := filepath.Join(goRoot, opts.BundleRoot, runID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return Manifest{}, err
	}
	if err := copyJSON(filepath.Join(goRoot, opts.ShadowComparePath), filepath.Join(bundleDir, "shadow-compare-report.json")); err != nil {
		return Manifest{}, err
	}
	if err := copyJSON(filepath.Join(goRoot, opts.ShadowMatrixPath), filepath.Join(bundleDir, "shadow-matrix-report.json")); err != nil {
		return Manifest{}, err
	}
	if err := copyJSON(filepath.Join(goRoot, opts.ScorecardPath), filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json")); err != nil {
		return Manifest{}, err
	}
	if err := copyJSON(filepath.Join(goRoot, "docs/reports/rollback-trigger-surface.json"), filepath.Join(bundleDir, "rollback-trigger-surface.json")); err != nil {
		return Manifest{}, err
	}

	severity := classifySeverity(scorecard)
	status := "parity-ok"
	if severityRank(severity) > 0 {
		status = "attention-needed"
	}
	summary := BundleSummary{
		RunID:                   runID,
		GeneratedAt:             utcISO(generatedAt),
		Status:                  status,
		Severity:                severity,
		BundlePath:              relPath(filepath.Join(goRoot, opts.BundleRoot, runID), goRoot),
		SummaryPath:             relPath(filepath.Join(bundleDir, "summary.json"), goRoot),
		LatestEvidenceTimestamp: scorecard.Summary.LatestEvidenceTimestamp,
		Freshness:               scorecard.Freshness,
		CompareTraceID:          compare.TraceID,
		CutoverCheckpoints:      scorecard.CutoverCheckpoints,
		CloseoutCommands: []string{
			"cd bigclaw-go && go run ./cmd/bigclawctl live-shadow scorecard --pretty",
			"cd bigclaw-go && go run ./cmd/bigclawctl live-shadow bundle",
			"cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned",
			"git push origin <branch> && git log -1 --stat",
		},
	}
	summary.Artifacts.ShadowCompareReportPath = relPath(filepath.Join(bundleDir, "shadow-compare-report.json"), goRoot)
	summary.Artifacts.ShadowMatrixReportPath = relPath(filepath.Join(bundleDir, "shadow-matrix-report.json"), goRoot)
	summary.Artifacts.LiveShadowScorecardPath = relPath(filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json"), goRoot)
	summary.Artifacts.RollbackTriggerSurfacePath = relPath(filepath.Join(bundleDir, "rollback-trigger-surface.json"), goRoot)
	summary.Summary.TotalEvidenceRuns = scorecard.Summary.TotalEvidenceRuns
	summary.Summary.ParityOKCount = scorecard.Summary.ParityOKCount
	summary.Summary.DriftDetectedCount = scorecard.Summary.DriftDetectedCount
	summary.Summary.MatrixTotal = scorecard.Summary.MatrixTotal
	summary.Summary.MatrixMismatched = scorecard.Summary.MatrixMismatched
	summary.Summary.StaleInputs = scorecard.Summary.StaleInputs
	summary.Summary.FreshInputs = scorecard.Summary.FreshInputs
	summary.RollbackTriggerSurface.Status = rollback.Summary.Status
	summary.RollbackTriggerSurface.AutomationBoundary = rollback.Summary.AutomationBoundary
	summary.RollbackTriggerSurface.AutomatedRollbackTrigger = rollback.Summary.AutomatedRollbackTrigger
	summary.RollbackTriggerSurface.Distinctions = rollback.Summary.Distinctions
	summary.RollbackTriggerSurface.Issue = rollback.Issue
	summary.RollbackTriggerSurface.DigestPath = rollback.SharedGuardrailSummary.DigestPath
	summary.RollbackTriggerSurface.SummaryPath = "docs/reports/rollback-trigger-surface.json"
	for _, item := range matrix.Results {
		if item.TraceID != "" {
			summary.MatrixTraceIDs = append(summary.MatrixTraceIDs, item.TraceID)
		}
	}

	if err := writeJSON(filepath.Join(bundleDir, "summary.json"), summary); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(filepath.Join(goRoot, opts.SummaryPath), summary); err != nil {
		return Manifest{}, err
	}

	recentRuns, err := loadRecentRuns(filepath.Join(goRoot, opts.BundleRoot))
	if err != nil {
		return Manifest{}, err
	}
	rollup := buildRollup(recentRuns, generatedAt, 5)
	manifest := Manifest{
		Latest:      summary,
		RecentRuns:  compactRuns(recentRuns),
		DriftRollup: rollup,
	}
	if err := writeJSON(filepath.Join(goRoot, opts.RollupPath), rollup); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(filepath.Join(goRoot, opts.ManifestPath), manifest); err != nil {
		return Manifest{}, err
	}
	indexText := renderIndex(summary, manifest.RecentRuns, rollup)
	indexPath := filepath.Join(goRoot, opts.IndexPath)
	if err := os.WriteFile(indexPath, []byte(indexText), 0o644); err != nil {
		return Manifest{}, err
	}
	readmePath := filepath.Join(bundleDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(indexText), 0o644); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func buildCompareEntry(compare CompareReport) ParityEntry {
	return ParityEntry{
		EntryType:     "single-compare",
		Label:         "single fixture compare",
		TraceID:       compare.TraceID,
		SourceFile:    nil,
		SourceKind:    "fixture",
		TaskShape:     nil,
		CorpusSlice:   nil,
		Parity:        classifyParity(compare.Diff),
		PrimaryTaskID: compare.Primary.TaskID,
		ShadowTaskID:  compare.Shadow.TaskID,
	}
}

func buildMatrixEntry(item matrixResult) ParityEntry {
	return ParityEntry{
		EntryType:     "matrix-row",
		Label:         item.SourceFile,
		TraceID:       item.TraceID,
		SourceFile:    item.SourceFile,
		SourceKind:    item.SourceKind,
		TaskShape:     item.TaskShape,
		CorpusSlice:   item.CorpusSlice,
		Parity:        classifyParity(item.Diff),
		PrimaryTaskID: item.Primary.TaskID,
		ShadowTaskID:  item.Shadow.TaskID,
	}
}

func classifyParity(diff diffReport) ParityStatus {
	reasons := []string{}
	if !diff.StateEqual {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !diff.EventTypesEqual {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if diff.EventCountDelta != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := math.Round(math.Abs(diff.PrimaryTimelineSeconds-diff.ShadowTimelineSeconds)*1_000_000) / 1_000_000
	if timelineDelta > TimelineDriftToleranceSeconds {
		reasons = append(reasons, "timeline-drift")
	}
	status := "parity-ok"
	if len(reasons) > 0 {
		status = "drift-detected"
	}
	return ParityStatus{
		Status:                        status,
		TimelineDeltaSeconds:          timelineDelta,
		TimelineDriftToleranceSeconds: TimelineDriftToleranceSeconds,
		Reasons:                       reasons,
	}
}

func buildFreshnessEntry(name string, reportPath string, primaryEvents []eventRecord, shadowEvents []eventRecord, generatedAt time.Time) FreshnessEntry {
	latest := latestTimestamp(primaryEvents, shadowEvents)
	return makeFreshnessEntry(name, reportPath, latest, generatedAt)
}

func buildFreshnessEntryFromMatrix(name string, reportPath string, report MatrixReport, generatedAt time.Time) FreshnessEntry {
	var latest *time.Time
	for _, result := range report.Results {
		candidate := latestTimestamp(result.Primary.Events, result.Shadow.Events)
		if candidate == nil {
			continue
		}
		if latest == nil || candidate.After(*latest) {
			value := *candidate
			latest = &value
		}
	}
	return makeFreshnessEntry(name, reportPath, latest, generatedAt)
}

func makeFreshnessEntry(name string, reportPath string, latest *time.Time, generatedAt time.Time) FreshnessEntry {
	entry := FreshnessEntry{
		Name:              name,
		ReportPath:        reportPath,
		FreshnessSLOHours: EvidenceFreshnessSLOHours,
		Status:            "missing-timestamps",
	}
	if latest == nil {
		return entry
	}
	timestamp := utcISO(*latest)
	ageHours := math.Round(generatedAt.Sub(*latest).Hours()*100) / 100
	entry.LatestEvidenceTimestamp = &timestamp
	entry.AgeHours = &ageHours
	entry.Status = "fresh"
	if ageHours > EvidenceFreshnessSLOHours {
		entry.Status = "stale"
	}
	return entry
}

func latestTimestamp(groups ...[]eventRecord) *time.Time {
	var latest *time.Time
	for _, events := range groups {
		for _, event := range events {
			if strings.TrimSpace(event.Timestamp) == "" {
				continue
			}
			parsed, err := parseTime(event.Timestamp)
			if err != nil {
				continue
			}
			if latest == nil || parsed.After(*latest) {
				value := parsed
				latest = &value
			}
		}
	}
	return latest
}

func checkpoint(name string, passed bool, detail string) Checkpoint {
	return Checkpoint{Name: name, Passed: passed, Detail: detail}
}

func classifySeverity(scorecard Scorecard) string {
	if scorecard.Summary.StaleInputs > 0 {
		return "high"
	}
	if scorecard.Summary.DriftDetectedCount > 0 {
		return "medium"
	}
	for _, item := range scorecard.CutoverCheckpoints {
		if !item.Passed {
			return "low"
		}
	}
	return "none"
}

func severityRank(value string) int {
	switch value {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func loadRecentRuns(bundleRoot string) ([]BundleSummary, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	runs := []BundleSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		var summary BundleSummary
		if err := readJSON(summaryPath, &summary); err != nil {
			continue
		}
		runs = append(runs, summary)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].GeneratedAt > runs[j].GeneratedAt
	})
	return runs, nil
}

func compactRuns(runs []BundleSummary) []RecentRun {
	items := make([]RecentRun, 0, len(runs))
	for _, item := range runs {
		items = append(items, RecentRun{
			RunID:       item.RunID,
			GeneratedAt: item.GeneratedAt,
			Status:      item.Status,
			Severity:    item.Severity,
			BundlePath:  item.BundlePath,
			SummaryPath: item.SummaryPath,
		})
	}
	return items
}

func buildRollup(runs []BundleSummary, generatedAt time.Time, limit int) Rollup {
	window := runs
	if len(window) > limit {
		window = window[:limit]
	}
	rollup := Rollup{
		GeneratedAt: utcISO(generatedAt),
		Status:      "parity-ok",
		WindowSize:  limit,
	}
	highestSeverity := "none"
	for _, run := range window {
		if severityRank(run.Severity) > severityRank(highestSeverity) {
			highestSeverity = run.Severity
		}
		if run.Status == "parity-ok" {
			rollup.Summary.StatusCounts.ParityOK++
		} else {
			rollup.Summary.StatusCounts.AttentionNeeded++
		}
		if run.Summary.StaleInputs > 0 {
			rollup.Summary.StaleRuns++
		}
		if run.Summary.DriftDetectedCount > 0 {
			rollup.Summary.DriftDetectedRuns++
		}
		rollup.RecentRuns = append(rollup.RecentRuns, RecentRun{
			RunID:                   run.RunID,
			GeneratedAt:             run.GeneratedAt,
			Status:                  run.Status,
			Severity:                run.Severity,
			LatestEvidenceTimestamp: run.LatestEvidenceTimestamp,
			DriftDetectedCount:      run.Summary.DriftDetectedCount,
			StaleInputs:             run.Summary.StaleInputs,
			BundlePath:              run.BundlePath,
			SummaryPath:             run.SummaryPath,
		})
	}
	rollup.Summary.RecentRunCount = len(window)
	rollup.Summary.HighestSeverity = highestSeverity
	if len(window) > 0 {
		rollup.Summary.LatestRunID = window[0].RunID
	}
	if severityRank(highestSeverity) > 0 {
		rollup.Status = "attention-needed"
	}
	return rollup
}

func renderIndex(latest BundleSummary, recentRuns []RecentRun, rollup Rollup) string {
	lines := []string{
		"# Live Shadow Mirror Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", latest.RunID),
		fmt.Sprintf("- Generated at: `%s`", latest.GeneratedAt),
		fmt.Sprintf("- Status: `%s`", latest.Status),
		fmt.Sprintf("- Severity: `%s`", latest.Severity),
		fmt.Sprintf("- Bundle: `%s`", latest.BundlePath),
		fmt.Sprintf("- Summary JSON: `%s`", latest.SummaryPath),
		"",
		"## Latest bundle artifacts",
		"",
		fmt.Sprintf("- Shadow compare report: `%s`", latest.Artifacts.ShadowCompareReportPath),
		fmt.Sprintf("- Shadow matrix report: `%s`", latest.Artifacts.ShadowMatrixReportPath),
		fmt.Sprintf("- Parity scorecard: `%s`", latest.Artifacts.LiveShadowScorecardPath),
		fmt.Sprintf("- Rollback trigger surface: `%s`", latest.Artifacts.RollbackTriggerSurfacePath),
		"",
		"## Latest run summary",
		"",
		fmt.Sprintf("- Compare trace: `%s`", latest.CompareTraceID),
		fmt.Sprintf("- Matrix trace count: `%d`", len(latest.MatrixTraceIDs)),
		fmt.Sprintf("- Evidence runs: `%d`", latest.Summary.TotalEvidenceRuns),
		fmt.Sprintf("- Parity-ok entries: `%d`", latest.Summary.ParityOKCount),
		fmt.Sprintf("- Drift-detected entries: `%d`", latest.Summary.DriftDetectedCount),
		fmt.Sprintf("- Matrix total: `%d`", latest.Summary.MatrixTotal),
		fmt.Sprintf("- Matrix mismatched: `%d`", latest.Summary.MatrixMismatched),
		fmt.Sprintf("- Fresh inputs: `%d`", latest.Summary.FreshInputs),
		fmt.Sprintf("- Stale inputs: `%d`", latest.Summary.StaleInputs),
		fmt.Sprintf("- Rollback trigger surface status: `%s`", latest.RollbackTriggerSurface.Status),
		fmt.Sprintf("- Rollback automation boundary: `%s`", latest.RollbackTriggerSurface.AutomationBoundary),
		fmt.Sprintf(
			"- Rollback trigger distinctions: `{'blockers': %d, 'warnings': %d, 'manual_only_paths': %d}`",
			latest.RollbackTriggerSurface.Distinctions.Blockers,
			latest.RollbackTriggerSurface.Distinctions.Warnings,
			latest.RollbackTriggerSurface.Distinctions.ManualOnlyPaths,
		),
		"",
		"## Parity drift rollup",
		"",
		fmt.Sprintf("- Status: `%s`", rollup.Status),
		fmt.Sprintf("- Latest run: `%s`", rollup.Summary.LatestRunID),
		fmt.Sprintf("- Highest severity: `%s`", rollup.Summary.HighestSeverity),
		fmt.Sprintf("- Drift-detected runs in window: `%d`", rollup.Summary.DriftDetectedRuns),
		fmt.Sprintf("- Stale runs in window: `%d`", rollup.Summary.StaleRuns),
		"",
		"## Workflow closeout commands",
		"",
	}
	for _, command := range latest.CloseoutCommands {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	for _, item := range recentRuns {
		lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s` · `%s`", item.RunID, item.Status, item.Severity, item.GeneratedAt, item.BundlePath))
	}
	lines = append(lines,
		"",
		"## Linked migration docs",
		"",
		"- `docs/migration-shadow.md` Shadow helper workflow and bundle generation steps.",
		"- `docs/reports/migration-readiness-report.md` Migration readiness summary linked to the shadow bundle.",
		"- `docs/reports/migration-plan-review-notes.md` Review notes tied to the shadow bundle index.",
		"- `docs/reports/rollback-trigger-surface.json` Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle.",
		"",
		"## Parallel Follow-up Index",
		"",
		"- `docs/reports/parallel-follow-up-index.md` is the canonical index for the remaining live-shadow, rollback, and corpus-coverage follow-up digests behind `OPE-266` / `BIG-PAR-092` and `OPE-254` / `BIG-PAR-088`.",
		"- `docs/reports/live-shadow-comparison-follow-up-digest.md` keeps the `OPE-266` / `BIG-PAR-092` live shadow comparison caveats attached to this bundle.",
		"- `docs/reports/rollback-safeguard-follow-up-digest.md` keeps the `OPE-254` / `BIG-PAR-088` rollback guardrail caveats attached to this bundle.",
		"- Use `docs/reports/parallel-validation-matrix.md` first when a shadow review",
		"  needs the checked-in local/Kubernetes/Ray validation entrypoint alongside the",
		"  shadow evidence bundle.",
		"",
	)
	return strings.Join(lines, "\n")
}

func readJSON(path string, dest any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func copyJSON(source string, destination string) error {
	var payload any
	if err := readJSON(source, &payload); err != nil {
		return err
	}
	return writeJSON(destination, payload)
}

func resolveRepoPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, filepath.FromSlash(path))
}

func relPath(path string, root string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
}

func utcISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func quoteStatuses(items []FreshnessEntry) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprintf("'%s'", item.Status))
	}
	return fmt.Sprintf("[%s]", strings.Join(values, ", "))
}

func intFromAny(value any) int {
	switch item := value.(type) {
	case float64:
		return int(item)
	case int:
		return item
	case int64:
		return int(item)
	default:
		return 0
	}
}
