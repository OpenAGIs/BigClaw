package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	TimelineDriftToleranceSeconds = 0.25
	EvidenceFreshnessSLOHours     = 168
)

type Event struct {
	Timestamp string `json:"timestamp"`
}

type TaskRun struct {
	TaskID string  `json:"task_id"`
	Events []Event `json:"events"`
}

type Diff struct {
	StateEqual             bool    `json:"state_equal"`
	EventTypesEqual        bool    `json:"event_types_equal"`
	EventCountDelta        int     `json:"event_count_delta"`
	PrimaryTimelineSeconds float64 `json:"primary_timeline_seconds"`
	ShadowTimelineSeconds  float64 `json:"shadow_timeline_seconds"`
}

type CompareReport struct {
	TraceID string  `json:"trace_id"`
	Primary TaskRun `json:"primary"`
	Shadow  TaskRun `json:"shadow"`
	Diff    Diff    `json:"diff"`
}

type MatrixResult struct {
	TraceID     string  `json:"trace_id"`
	Primary     TaskRun `json:"primary"`
	Shadow      TaskRun `json:"shadow"`
	Diff        Diff    `json:"diff"`
	SourceFile  string  `json:"source_file"`
	SourceKind  string  `json:"source_kind"`
	TaskShape   string  `json:"task_shape"`
	CorpusSlice string  `json:"corpus_slice"`
}

type CorpusCoverage struct {
	CorpusSliceCount          int `json:"corpus_slice_count"`
	UncoveredCorpusSliceCount int `json:"uncovered_corpus_slice_count"`
}

type MatrixReport struct {
	Total          int            `json:"total"`
	Matched        int            `json:"matched"`
	Mismatched     int            `json:"mismatched"`
	Results        []MatrixResult `json:"results"`
	CorpusCoverage CorpusCoverage `json:"corpus_coverage"`
}

type Scorecard struct {
	GeneratedAt    string                 `json:"generated_at"`
	Ticket         string                 `json:"ticket"`
	Title          string                 `json:"title"`
	Status         string                 `json:"status"`
	EvidenceInputs scorecardEvidenceInput `json:"evidence_inputs"`
	Summary        scorecardSummary       `json:"summary"`
	Freshness      []freshnessEntry       `json:"freshness"`
	ParityEntries  []parityEntry          `json:"parity_entries"`
	CutoverChecks  []cutoverCheckpoint    `json:"cutover_checkpoints"`
	Limitations    []string               `json:"limitations"`
	FutureWork     []string               `json:"future_work"`
}

type scorecardEvidenceInput struct {
	ShadowCompareReportPath string `json:"shadow_compare_report_path"`
	ShadowMatrixReportPath  string `json:"shadow_matrix_report_path"`
	GeneratorScript         string `json:"generator_script"`
}

type scorecardSummary struct {
	TotalEvidenceRuns         int    `json:"total_evidence_runs"`
	ParityOKCount             int    `json:"parity_ok_count"`
	DriftDetectedCount        int    `json:"drift_detected_count"`
	MatrixTotal               int    `json:"matrix_total"`
	MatrixMatched             int    `json:"matrix_matched"`
	MatrixMismatched          int    `json:"matrix_mismatched"`
	CorpusCoveragePresent     bool   `json:"corpus_coverage_present"`
	CorpusUncoveredSliceCount int    `json:"corpus_uncovered_slice_count,omitempty"`
	LatestEvidenceTimestamp   string `json:"latest_evidence_timestamp,omitempty"`
	FreshInputs               int    `json:"fresh_inputs"`
	StaleInputs               int    `json:"stale_inputs"`
}

type freshnessEntry struct {
	Name                    string   `json:"name"`
	ReportPath              string   `json:"report_path"`
	LatestEvidenceTimestamp *string  `json:"latest_evidence_timestamp"`
	AgeHours                *float64 `json:"age_hours"`
	FreshnessSLOHours       int      `json:"freshness_slo_hours"`
	Status                  string   `json:"status"`
}

type parityEntry struct {
	EntryType     string       `json:"entry_type"`
	Label         string       `json:"label"`
	TraceID       string       `json:"trace_id"`
	SourceFile    *string      `json:"source_file"`
	SourceKind    *string      `json:"source_kind"`
	TaskShape     *string      `json:"task_shape,omitempty"`
	CorpusSlice   *string      `json:"corpus_slice,omitempty"`
	Parity        parityStatus `json:"parity"`
	PrimaryTaskID string       `json:"primary_task_id"`
	ShadowTaskID  string       `json:"shadow_task_id"`
}

type parityStatus struct {
	Status                        string   `json:"status"`
	TimelineDeltaSeconds          float64  `json:"timeline_delta_seconds"`
	TimelineDriftToleranceSeconds float64  `json:"timeline_drift_tolerance_seconds"`
	Reasons                       []string `json:"reasons"`
}

type cutoverCheckpoint struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

func BuildLiveShadowScorecard(repoRoot, comparePath, matrixPath string, now time.Time) (Scorecard, error) {
	compare := CompareReport{}
	if err := readJSON(filepath.Join(repoRoot, comparePath), &compare); err != nil {
		return Scorecard{}, err
	}
	matrix := MatrixReport{}
	if err := readJSON(filepath.Join(repoRoot, matrixPath), &matrix); err != nil {
		return Scorecard{}, err
	}

	generatedAt := now.UTC()
	parityEntries := make([]parityEntry, 0, len(matrix.Results)+1)
	parityEntries = append(parityEntries, parityEntry{
		EntryType:     "single-compare",
		Label:         "single fixture compare",
		TraceID:       compare.TraceID,
		SourceFile:    nil,
		SourceKind:    stringPtr("fixture"),
		Parity:        classifyParity(compare.Diff),
		PrimaryTaskID: compare.Primary.TaskID,
		ShadowTaskID:  compare.Shadow.TaskID,
	})
	for _, item := range matrix.Results {
		parityEntries = append(parityEntries, parityEntry{
			EntryType:     "matrix-row",
			Label:         item.SourceFile,
			TraceID:       item.TraceID,
			SourceFile:    nilIfEmpty(item.SourceFile),
			SourceKind:    nilIfEmpty(item.SourceKind),
			TaskShape:     nilIfEmpty(item.TaskShape),
			CorpusSlice:   nilIfEmpty(item.CorpusSlice),
			Parity:        classifyParity(item.Diff),
			PrimaryTaskID: item.Primary.TaskID,
			ShadowTaskID:  item.Shadow.TaskID,
		})
	}

	parityOKCount := 0
	for _, item := range parityEntries {
		if item.Parity.Status == "parity-ok" {
			parityOKCount++
		}
	}

	freshness := []freshnessEntry{
		buildFreshnessEntry("shadow-compare-report", comparePath, latestCompareTimestamp(compare), generatedAt),
		buildFreshnessEntry("shadow-matrix-report", matrixPath, latestMatrixTimestamp(matrix), generatedAt),
	}
	staleInputs := 0
	var latestEvidence *time.Time
	for _, item := range freshness {
		if item.Status != "fresh" {
			staleInputs++
		}
		if item.LatestEvidenceTimestamp == nil {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, *item.LatestEvidenceTimestamp)
		if err != nil {
			continue
		}
		if latestEvidence == nil || parsed.After(*latestEvidence) {
			value := parsed
			latestEvidence = &value
		}
	}

	scorecard := Scorecard{
		GeneratedAt: formatTimestamp(generatedAt),
		Ticket:      "BIG-PAR-092",
		Title:       "Live shadow mirror parity drift scorecard",
		Status:      "repo-native-live-shadow-scorecard",
		EvidenceInputs: scorecardEvidenceInput{
			ShadowCompareReportPath: comparePath,
			ShadowMatrixReportPath:  matrixPath,
			GeneratorScript:         "go run ./cmd/bigclawctl migration live-shadow-scorecard",
		},
		Summary: scorecardSummary{
			TotalEvidenceRuns:         len(parityEntries),
			ParityOKCount:             parityOKCount,
			DriftDetectedCount:        len(parityEntries) - parityOKCount,
			MatrixTotal:               matrix.Total,
			MatrixMatched:             matrix.Matched,
			MatrixMismatched:          matrix.Mismatched,
			CorpusCoveragePresent:     matrix.CorpusCoverage != (CorpusCoverage{}),
			CorpusUncoveredSliceCount: matrix.CorpusCoverage.UncoveredCorpusSliceCount,
			LatestEvidenceTimestamp:   formatTimePointer(latestEvidence),
			FreshInputs:               len(freshness) - staleInputs,
			StaleInputs:               staleInputs,
		},
		Freshness:     freshness,
		ParityEntries: parityEntries,
		CutoverChecks: []cutoverCheckpoint{
			{
				Name:   "single_compare_matches_terminal_state_and_event_sequence",
				Passed: compare.Diff.StateEqual && compare.Diff.EventTypesEqual,
				Detail: fmt.Sprintf("trace_id=%s", compare.TraceID),
			},
			{
				Name:   "matrix_reports_no_state_or_event_sequence_mismatches",
				Passed: matrix.Mismatched == 0,
				Detail: fmt.Sprintf("matched=%d mismatched=%d", matrix.Matched, matrix.Mismatched),
			},
			{
				Name:   "scorecard_detects_no_parity_drift",
				Passed: len(parityEntries)-parityOKCount == 0,
				Detail: fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, len(parityEntries)-parityOKCount),
			},
			{
				Name:   "checked_in_evidence_is_fresh_enough_for_review",
				Passed: staleInputs == 0,
				Detail: fmt.Sprintf("freshness_statuses=%v", freshnessStatuses(freshness)),
			},
			{
				Name:   "matrix_includes_corpus_coverage_overlay",
				Passed: matrix.CorpusCoverage != (CorpusCoverage{}),
				Detail: fmt.Sprintf("corpus_slice_count=%d", matrix.CorpusCoverage.CorpusSliceCount),
			},
		},
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
	return scorecard, nil
}

func WriteLiveShadowScorecard(path string, report Scorecard) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func latestCompareTimestamp(report CompareReport) *time.Time {
	return latestRunTimestamp([]TaskRun{report.Primary, report.Shadow})
}

func latestMatrixTimestamp(report MatrixReport) *time.Time {
	runs := make([]TaskRun, 0, len(report.Results)*2)
	for _, item := range report.Results {
		runs = append(runs, item.Primary, item.Shadow)
	}
	return latestRunTimestamp(runs)
}

func latestRunTimestamp(runs []TaskRun) *time.Time {
	var latest *time.Time
	for _, run := range runs {
		for _, event := range run.Events {
			if event.Timestamp == "" {
				continue
			}
			parsed, err := time.Parse(time.RFC3339Nano, event.Timestamp)
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

func buildFreshnessEntry(name, reportPath string, latest *time.Time, generatedAt time.Time) freshnessEntry {
	entry := freshnessEntry{
		Name:              name,
		ReportPath:        reportPath,
		FreshnessSLOHours: EvidenceFreshnessSLOHours,
		Status:            "missing-timestamps",
	}
	if latest == nil {
		return entry
	}
	ageHours := roundTo((generatedAt.Sub(*latest).Hours()), 2)
	timestamp := formatTimestamp(*latest)
	entry.LatestEvidenceTimestamp = &timestamp
	entry.AgeHours = &ageHours
	if ageHours <= EvidenceFreshnessSLOHours {
		entry.Status = "fresh"
	} else {
		entry.Status = "stale"
	}
	return entry
}

func classifyParity(diff Diff) parityStatus {
	reasons := make([]string, 0, 4)
	if !diff.StateEqual {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !diff.EventTypesEqual {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if diff.EventCountDelta != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := roundTo(abs(diff.PrimaryTimelineSeconds-diff.ShadowTimelineSeconds), 6)
	if timelineDelta > TimelineDriftToleranceSeconds {
		reasons = append(reasons, "timeline-drift")
	}
	status := "parity-ok"
	if len(reasons) > 0 {
		status = "drift-detected"
	}
	return parityStatus{
		Status:                        status,
		TimelineDeltaSeconds:          timelineDelta,
		TimelineDriftToleranceSeconds: TimelineDriftToleranceSeconds,
		Reasons:                       reasons,
	}
}

func freshnessStatuses(entries []freshnessEntry) []string {
	values := make([]string, 0, len(entries))
	for _, item := range entries {
		values = append(values, item.Status)
	}
	return values
}

func stringPtr(value string) *string {
	return &value
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formatTimestamp(value time.Time) string {
	return value.Format(time.RFC3339Nano)
}

func formatTimePointer(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTimestamp(*value)
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func roundTo(value float64, digits int) float64 {
	scale := 1.0
	for i := 0; i < digits; i++ {
		scale *= 10
	}
	if value >= 0 {
		return float64(int(value*scale+0.5)) / scale
	}
	return float64(int(value*scale-0.5)) / scale
}
