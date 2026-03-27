package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	liveShadowTimelineDriftToleranceSeconds = 0.25
	liveShadowEvidenceFreshnessSLOHours     = 168.0
)

var liveShadowFollowupDigests = []any{
	map[string]any{
		"path":        "docs/reports/live-shadow-comparison-follow-up-digest.md",
		"description": "Live shadow traffic comparison caveats are consolidated here.",
	},
	map[string]any{
		"path":        "docs/reports/rollback-safeguard-follow-up-digest.md",
		"description": "Rollback remains operator-driven; this digest explains the guardrail visibility and trigger caveats.",
	},
}

var liveShadowDocLinks = []any{
	map[string]any{"path": "docs/migration-shadow.md", "description": "Shadow helper workflow and bundle generation steps."},
	map[string]any{"path": "docs/reports/migration-readiness-report.md", "description": "Migration readiness summary linked to the shadow bundle."},
	map[string]any{"path": "docs/reports/migration-plan-review-notes.md", "description": "Review notes tied to the shadow bundle index."},
	map[string]any{"path": "docs/reports/rollback-trigger-surface.json", "description": "Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle."},
}

type automationLiveShadowScorecardOptions struct {
	RepoRoot            string
	ShadowCompareReport string
	ShadowMatrixReport  string
	OutputPath          string
	Now                 func() time.Time
}

type automationExportLiveShadowBundleOptions struct {
	GoRoot              string
	ShadowCompareReport string
	ShadowMatrixReport  string
	ScorecardReport     string
	BundleRoot          string
	SummaryPath         string
	IndexPath           string
	ManifestPath        string
	RollupPath          string
	RunID               string
	Now                 func() time.Time
}

func runAutomationLiveShadowScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration live-shadow-scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	shadowCompare := flags.String("shadow-compare-report", "bigclaw-go/docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrix := flags.String("shadow-matrix-report", "bigclaw-go/docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	output := flags.String("output", "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json", "output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration live-shadow-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, _, err := automationLiveShadowScorecard(automationLiveShadowScorecardOptions{
		RepoRoot:            absPath(*repoRoot),
		ShadowCompareReport: trim(*shadowCompare),
		ShadowMatrixReport:  trim(*shadowMatrix),
		OutputPath:          trim(*output),
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return nil
}

func runAutomationExportLiveShadowBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration export-live-shadow-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", "bigclaw-go", "bigclaw-go root")
	shadowCompare := flags.String("shadow-compare-report", "docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrix := flags.String("shadow-matrix-report", "docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	scorecard := flags.String("scorecard-report", "docs/reports/live-shadow-mirror-scorecard.json", "live shadow scorecard report path")
	bundleRoot := flags.String("bundle-root", "docs/reports/live-shadow-runs", "bundle root")
	summaryPath := flags.String("summary-path", "docs/reports/live-shadow-summary.json", "summary path")
	indexPath := flags.String("index-path", "docs/reports/live-shadow-index.md", "index path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-shadow-index.json", "manifest path")
	rollupPath := flags.String("rollup-path", "docs/reports/live-shadow-drift-rollup.json", "rollup path")
	runID := flags.String("run-id", "", "run id")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration export-live-shadow-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationExportLiveShadowBundle(automationExportLiveShadowBundleOptions{
		GoRoot:              trim(*goRoot),
		ShadowCompareReport: trim(*shadowCompare),
		ShadowMatrixReport:  trim(*shadowMatrix),
		ScorecardReport:     trim(*scorecard),
		BundleRoot:          trim(*bundleRoot),
		SummaryPath:         trim(*summaryPath),
		IndexPath:           trim(*indexPath),
		ManifestPath:        trim(*manifestPath),
		RollupPath:          trim(*rollupPath),
		RunID:               trim(*runID),
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	return err
}

func automationLiveShadowScorecard(opts automationLiveShadowScorecardOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	repoRoot := opts.RepoRoot
	compareReport, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.ShadowCompareReport)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load compare report from %s", opts.ShadowCompareReport)
	}
	matrixReport, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.ShadowMatrixReport)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load matrix report from %s", opts.ShadowMatrixReport)
	}
	generatedAt := now().UTC()

	parityEntries := []any{buildCompareEntry(compareReport)}
	parityEntries = append(parityEntries, buildMatrixEntries(matrixReport)...)
	parityOKCount := 0
	for _, item := range parityEntries {
		entry, _ := item.(map[string]any)
		parity, _ := entry["parity"].(map[string]any)
		if automationFirstText(parity["status"]) == "parity-ok" {
			parityOKCount++
		}
	}
	driftDetectedCount := len(parityEntries) - parityOKCount

	freshness := []any{
		buildFreshnessEntry("shadow-compare-report", opts.ShadowCompareReport, compareReport, generatedAt),
		buildFreshnessEntry("shadow-matrix-report", opts.ShadowMatrixReport, matrixReport, generatedAt),
	}
	staleInputs := []any{}
	latestEvidenceTimestamp := ""
	for _, item := range freshness {
		entry, _ := item.(map[string]any)
		if automationFirstText(entry["status"]) != "fresh" {
			staleInputs = append(staleInputs, entry)
		}
		if ts := automationFirstText(entry["latest_evidence_timestamp"]); ts > latestEvidenceTimestamp {
			latestEvidenceTimestamp = ts
		}
	}

	matrixCorpusCoverage, _ := matrixReport["corpus_coverage"].(map[string]any)
	checkpoints := []any{
		automationCheck("single_compare_matches_terminal_state_and_event_sequence", automationBool(compareReport["diff"].(map[string]any)["state_equal"]) && automationBool(compareReport["diff"].(map[string]any)["event_types_equal"]), fmt.Sprintf("trace_id=%v", compareReport["trace_id"])),
		automationCheck("matrix_reports_no_state_or_event_sequence_mismatches", automationInt(matrixReport["mismatched"]) == 0, fmt.Sprintf("matched=%v mismatched=%v", matrixReport["matched"], matrixReport["mismatched"])),
		automationCheck("scorecard_detects_no_parity_drift", driftDetectedCount == 0, fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, driftDetectedCount)),
		automationCheck("checked_in_evidence_is_fresh_enough_for_review", len(staleInputs) == 0, fmt.Sprintf("freshness_statuses=%v", freshnessStatuses(freshness))),
		automationCheck("matrix_includes_corpus_coverage_overlay", len(matrixCorpusCoverage) > 0, fmt.Sprintf("corpus_slice_count=%v", matrixCorpusCoverage["corpus_slice_count"])),
	}

	report := map[string]any{
		"generated_at": automationUTCISO(generatedAt),
		"ticket":       "BIG-PAR-092",
		"title":        "Live shadow mirror parity drift scorecard",
		"status":       "repo-native-live-shadow-scorecard",
		"evidence_inputs": map[string]any{
			"shadow_compare_report_path": opts.ShadowCompareReport,
			"shadow_matrix_report_path":  opts.ShadowMatrixReport,
			"generator_script":           "bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		},
		"summary": map[string]any{
			"total_evidence_runs":          len(parityEntries),
			"parity_ok_count":              parityOKCount,
			"drift_detected_count":         driftDetectedCount,
			"matrix_total":                 automationInt(matrixReport["total"]),
			"matrix_matched":               automationInt(matrixReport["matched"]),
			"matrix_mismatched":            automationInt(matrixReport["mismatched"]),
			"corpus_coverage_present":      len(matrixCorpusCoverage) > 0,
			"corpus_uncovered_slice_count": matrixCorpusCoverage["uncovered_corpus_slice_count"],
			"latest_evidence_timestamp":    nilIfEmpty(latestEvidenceTimestamp),
			"fresh_inputs":                 countFreshInputs(freshness),
			"stale_inputs":                 len(staleInputs),
		},
		"freshness":           freshness,
		"parity_entries":      parityEntries,
		"cutover_checkpoints": checkpoints,
		"limitations": []any{
			"repo-native only: this scorecard summarizes checked-in shadow artifacts rather than an always-on production traffic mirror",
			"parity drift is measured from fixture-backed compare/matrix runs and optional corpus slices, not mirrored tenant traffic",
			"freshness is derived from the latest artifact event timestamps and should be treated as evidence recency, not live service health",
		},
		"future_work": []any{
			"replace offline fixture submission with a real ingress mirror or tenant-scoped shadow routing control before treating this as cutover-proof traffic parity",
			"promote parity drift review from checked-in artifacts into a continuously refreshed operational surface",
			"pair this scorecard with rollback automation only after tenant-scoped rollback guardrails exist",
		},
	}
	if err := automationWriteJSON(automationResolveRepoPath(repoRoot, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func automationExportLiveShadowBundle(opts automationExportLiveShadowBundleOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := resolveGoRoot(opts.GoRoot)
	compareReport, ok := automationReadJSON(filepath.Join(root, opts.ShadowCompareReport)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load compare report from %s", opts.ShadowCompareReport)
	}
	matrixReport, ok := automationReadJSON(filepath.Join(root, opts.ShadowMatrixReport)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load matrix report from %s", opts.ShadowMatrixReport)
	}
	scorecardReport, ok := automationReadJSON(filepath.Join(root, opts.ScorecardReport)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load scorecard report from %s", opts.ScorecardReport)
	}
	generatedAt := now().UTC()
	runID := opts.RunID
	if runID == "" {
		runID = deriveLiveShadowRunID(scorecardReport, generatedAt)
	}
	bundleDir := filepath.Join(root, opts.BundleRoot, runID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, 0, err
	}

	latest, err := buildLiveShadowRunSummary(root, bundleDir, runID, compareReport, matrixReport, scorecardReport, generatedAt)
	if err != nil {
		return nil, 0, err
	}
	if err := automationWriteJSON(filepath.Join(bundleDir, "summary.json"), latest); err != nil {
		return nil, 0, err
	}
	if err := automationWriteJSON(filepath.Join(root, opts.SummaryPath), latest); err != nil {
		return nil, 0, err
	}

	recentRuns := loadRecentShadowRuns(filepath.Join(root, opts.BundleRoot))
	rollup := buildLiveShadowRollup(recentRuns, 5)
	manifestRecentRuns := make([]any, 0, len(recentRuns))
	for _, item := range recentRuns {
		manifestRecentRuns = append(manifestRecentRuns, map[string]any{
			"run_id":       item["run_id"],
			"generated_at": item["generated_at"],
			"status":       item["status"],
			"severity":     item["severity"],
			"bundle_path":  item["bundle_path"],
			"summary_path": item["summary_path"],
		})
	}
	manifest := map[string]any{
		"latest":       latest,
		"recent_runs":  manifestRecentRuns,
		"drift_rollup": rollup,
	}
	if err := automationWriteJSON(filepath.Join(root, opts.RollupPath), rollup); err != nil {
		return nil, 0, err
	}
	if err := automationWriteJSON(filepath.Join(root, opts.ManifestPath), manifest); err != nil {
		return nil, 0, err
	}
	indexText := renderLiveShadowIndex(latest, manifestRecentRuns, rollup)
	if err := automationWriteText(filepath.Join(root, opts.IndexPath), indexText); err != nil {
		return nil, 0, err
	}
	if err := automationWriteText(filepath.Join(bundleDir, "README.md"), indexText); err != nil {
		return nil, 0, err
	}
	return manifest, 0, nil
}

func resolveGoRoot(value string) string {
	requested := value
	if filepath.IsAbs(requested) {
		return requested
	}
	candidate, _ := filepath.Abs(requested)
	if automationPathExists(candidate) {
		return candidate
	}
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == filepath.Base(requested) && automationPathExists(filepath.Join(cwd, "docs")) {
		return cwd
	}
	return candidate
}

func buildFreshnessEntry(name string, path string, report map[string]any, generatedAt time.Time) map[string]any {
	latestTimestamp := latestReportTimestamp(report)
	var ageHours any
	status := "missing-timestamps"
	var latestEvidence any
	if !latestTimestamp.IsZero() {
		ageHours = roundFloat(generatedAt.Sub(latestTimestamp).Hours(), 2)
		if automationFloat64(ageHours) <= liveShadowEvidenceFreshnessSLOHours {
			status = "fresh"
		} else {
			status = "stale"
		}
		latestEvidence = automationUTCISO(latestTimestamp)
	}
	return map[string]any{
		"name":                      name,
		"report_path":               path,
		"latest_evidence_timestamp": latestEvidence,
		"age_hours":                 ageHours,
		"freshness_slo_hours":       liveShadowEvidenceFreshnessSLOHours,
		"status":                    status,
	}
}

func latestReportTimestamp(report map[string]any) time.Time {
	timestamps := []time.Time{}
	if results, ok := report["results"].([]any); ok {
		for _, item := range results {
			entry, _ := item.(map[string]any)
			timestamps = append(timestamps, collectEventTimestamps(entry["primary"])...)
			timestamps = append(timestamps, collectEventTimestamps(entry["shadow"])...)
		}
	} else {
		timestamps = append(timestamps, collectEventTimestamps(report["primary"])...)
		timestamps = append(timestamps, collectEventTimestamps(report["shadow"])...)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i].Before(timestamps[j]) })
	if len(timestamps) == 0 {
		return time.Time{}
	}
	return timestamps[len(timestamps)-1]
}

func collectEventTimestamps(source any) []time.Time {
	result := []time.Time{}
	entry, _ := source.(map[string]any)
	events, _ := entry["events"].([]any)
	for _, item := range events {
		event, _ := item.(map[string]any)
		if parsed, err := automationParseFlexibleTime(automationFirstText(event["timestamp"])); err == nil {
			result = append(result, parsed)
		}
	}
	return result
}

func classifyParity(diff map[string]any) map[string]any {
	reasons := []any{}
	if !automationBool(diff["state_equal"]) {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !automationBool(diff["event_types_equal"]) {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if automationInt(diff["event_count_delta"]) != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := roundFloat(absFloat(automationFloat64(diff["primary_timeline_seconds"])-automationFloat64(diff["shadow_timeline_seconds"])), 6)
	if timelineDelta > liveShadowTimelineDriftToleranceSeconds {
		reasons = append(reasons, "timeline-drift")
	}
	status := "parity-ok"
	if len(reasons) > 0 {
		status = "drift-detected"
	}
	return map[string]any{
		"status":                           status,
		"timeline_delta_seconds":           timelineDelta,
		"timeline_drift_tolerance_seconds": liveShadowTimelineDriftToleranceSeconds,
		"reasons":                          reasons,
	}
}

func buildCompareEntry(report map[string]any) map[string]any {
	primary, _ := report["primary"].(map[string]any)
	shadow, _ := report["shadow"].(map[string]any)
	return map[string]any{
		"entry_type":      "single-compare",
		"label":           "single fixture compare",
		"trace_id":        report["trace_id"],
		"source_file":     nil,
		"source_kind":     "fixture",
		"parity":          classifyParity(mapValue(report, "diff")),
		"primary_task_id": primary["task_id"],
		"shadow_task_id":  shadow["task_id"],
	}
}

func buildMatrixEntries(report map[string]any) []any {
	items := []any{}
	results, _ := report["results"].([]any)
	for _, item := range results {
		entry, _ := item.(map[string]any)
		primary, _ := entry["primary"].(map[string]any)
		shadow, _ := entry["shadow"].(map[string]any)
		items = append(items, map[string]any{
			"entry_type":      "matrix-row",
			"label":           entry["source_file"],
			"trace_id":        entry["trace_id"],
			"source_file":     entry["source_file"],
			"source_kind":     entry["source_kind"],
			"task_shape":      entry["task_shape"],
			"corpus_slice":    entry["corpus_slice"],
			"parity":          classifyParity(mapValue(entry, "diff")),
			"primary_task_id": primary["task_id"],
			"shadow_task_id":  shadow["task_id"],
		})
	}
	return items
}

func mapValue(source any, key string) map[string]any {
	entry, _ := source.(map[string]any)
	child, _ := entry[key].(map[string]any)
	return child
}

func freshnessStatuses(items []any) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		entry, _ := item.(map[string]any)
		result = append(result, entry["status"])
	}
	return result
}

func countFreshInputs(items []any) int {
	count := 0
	for _, item := range items {
		entry, _ := item.(map[string]any)
		if automationFirstText(entry["status"]) == "fresh" {
			count++
		}
	}
	return count
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
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

func classifyShadowSeverity(scorecard map[string]any) string {
	summary, _ := scorecard["summary"].(map[string]any)
	if automationInt(summary["stale_inputs"]) > 0 {
		return "high"
	}
	if automationInt(summary["drift_detected_count"]) > 0 {
		return "medium"
	}
	if checkpoints, ok := scorecard["cutover_checkpoints"].([]any); ok {
		for _, item := range checkpoints {
			entry, _ := item.(map[string]any)
			if !automationBool(entry["passed"]) {
				return "low"
			}
		}
	}
	return "none"
}

func buildLiveShadowRunSummary(root string, bundleDir string, runID string, compareReport map[string]any, matrixReport map[string]any, scorecardReport map[string]any, generatedAt time.Time) (map[string]any, error) {
	compareBundlePath := filepath.Join(bundleDir, "shadow-compare-report.json")
	matrixBundlePath := filepath.Join(bundleDir, "shadow-matrix-report.json")
	scorecardBundlePath := filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json")
	rollbackBundlePath := filepath.Join(bundleDir, "rollback-trigger-surface.json")
	if copied := automationCopyJSONArtifact(filepath.Join(root, "docs/reports/shadow-compare-report.json"), compareBundlePath); copied == "" {
		return nil, errors.New("copy compare report")
	}
	if copied := automationCopyJSONArtifact(filepath.Join(root, "docs/reports/shadow-matrix-report.json"), matrixBundlePath); copied == "" {
		return nil, errors.New("copy matrix report")
	}
	if copied := automationCopyJSONArtifact(filepath.Join(root, "docs/reports/live-shadow-mirror-scorecard.json"), scorecardBundlePath); copied == "" {
		return nil, errors.New("copy live shadow scorecard")
	}
	if copied := automationCopyJSONArtifact(filepath.Join(root, "docs/reports/rollback-trigger-surface.json"), rollbackBundlePath); copied == "" {
		return nil, errors.New("copy rollback trigger surface")
	}
	rollbackReport, _ := automationReadJSON(filepath.Join(root, "docs/reports/rollback-trigger-surface.json")).(map[string]any)
	scorecardSummary, _ := scorecardReport["summary"].(map[string]any)
	freshness, _ := scorecardReport["freshness"].([]any)
	staleInputs := automationInt(scorecardSummary["stale_inputs"])
	driftDetectedCount := automationInt(scorecardSummary["drift_detected_count"])
	severity := classifyShadowSeverity(scorecardReport)
	status := "parity-ok"
	if severityRank(severity) > 0 {
		status = "attention-needed"
	}
	matrixTraceIDs := []any{}
	results, _ := matrixReport["results"].([]any)
	for _, item := range results {
		entry, _ := item.(map[string]any)
		if traceID := automationFirstText(entry["trace_id"]); traceID != "" {
			matrixTraceIDs = append(matrixTraceIDs, traceID)
		}
	}
	rollbackSummary, _ := rollbackReport["summary"].(map[string]any)
	return map[string]any{
		"run_id":       runID,
		"generated_at": automationUTCISO(generatedAt),
		"status":       status,
		"severity":     severity,
		"bundle_path":  automationRelPath(bundleDir, root),
		"summary_path": automationRelPath(filepath.Join(bundleDir, "summary.json"), root),
		"artifacts": map[string]any{
			"shadow_compare_report_path":    automationRelPath(compareBundlePath, root),
			"shadow_matrix_report_path":     automationRelPath(matrixBundlePath, root),
			"live_shadow_scorecard_path":    automationRelPath(scorecardBundlePath, root),
			"rollback_trigger_surface_path": automationRelPath(rollbackBundlePath, root),
		},
		"latest_evidence_timestamp": scorecardSummary["latest_evidence_timestamp"],
		"freshness":                 freshness,
		"summary": map[string]any{
			"total_evidence_runs":  automationInt(scorecardSummary["total_evidence_runs"]),
			"parity_ok_count":      automationInt(scorecardSummary["parity_ok_count"]),
			"drift_detected_count": driftDetectedCount,
			"matrix_total":         automationInt(scorecardSummary["matrix_total"]),
			"matrix_mismatched":    automationInt(scorecardSummary["matrix_mismatched"]),
			"stale_inputs":         staleInputs,
			"fresh_inputs":         automationInt(scorecardSummary["fresh_inputs"]),
		},
		"rollback_trigger_surface": map[string]any{
			"status":                     rollbackSummary["status"],
			"automation_boundary":        rollbackSummary["automation_boundary"],
			"automated_rollback_trigger": automationBool(rollbackSummary["automated_rollback_trigger"]),
			"distinctions":               rollbackSummary["distinctions"],
			"summary_path":               "docs/reports/rollback-trigger-surface.json",
		},
		"compare_trace_id":    compareReport["trace_id"],
		"matrix_trace_ids":    matrixTraceIDs,
		"cutover_checkpoints": scorecardReport["cutover_checkpoints"],
		"closeout_commands": []any{
			"cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --pretty",
			"cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py",
			"cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned",
			"git push origin <branch> && git log -1 --stat",
		},
	}, nil
}

func loadRecentShadowRuns(bundleRoot string) []map[string]any {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return nil
	}
	items := []map[string]any{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summary, _ := automationReadJSON(filepath.Join(bundleRoot, entry.Name(), "summary.json")).(map[string]any)
		if summary != nil {
			items = append(items, summary)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return automationFirstText(items[i]["generated_at"]) > automationFirstText(items[j]["generated_at"])
	})
	return items
}

func buildLiveShadowRollup(recentRuns []map[string]any, limit int) map[string]any {
	window := recentRuns
	if len(window) > limit {
		window = window[:limit]
	}
	highestSeverity := "none"
	statusCounts := map[string]any{"parity_ok": 0, "attention_needed": 0}
	staleRuns := 0
	driftRuns := 0
	entries := []any{}
	for _, item := range window {
		severity := automationFirstText(item["severity"])
		if severityRank(severity) > severityRank(highestSeverity) {
			highestSeverity = severity
		}
		if automationFirstText(item["status"]) == "parity-ok" {
			statusCounts["parity_ok"] = automationInt(statusCounts["parity_ok"]) + 1
		} else {
			statusCounts["attention_needed"] = automationInt(statusCounts["attention_needed"]) + 1
		}
		summary, _ := item["summary"].(map[string]any)
		if automationInt(summary["stale_inputs"]) > 0 {
			staleRuns++
		}
		if automationInt(summary["drift_detected_count"]) > 0 {
			driftRuns++
		}
		entries = append(entries, map[string]any{
			"run_id":                    item["run_id"],
			"generated_at":              item["generated_at"],
			"status":                    item["status"],
			"severity":                  item["severity"],
			"latest_evidence_timestamp": item["latest_evidence_timestamp"],
			"drift_detected_count":      summary["drift_detected_count"],
			"stale_inputs":              summary["stale_inputs"],
			"bundle_path":               item["bundle_path"],
			"summary_path":              item["summary_path"],
		})
	}
	status := "parity-ok"
	if severityRank(highestSeverity) > 0 {
		status = "attention-needed"
	}
	latestRunID := any(nil)
	if len(window) > 0 {
		latestRunID = window[0]["run_id"]
	}
	return map[string]any{
		"generated_at": automationUTCISO(time.Now().UTC()),
		"status":       status,
		"window_size":  limit,
		"summary": map[string]any{
			"recent_run_count":    len(window),
			"drift_detected_runs": driftRuns,
			"stale_runs":          staleRuns,
			"highest_severity":    highestSeverity,
			"status_counts":       statusCounts,
			"latest_run_id":       latestRunID,
		},
		"recent_runs": entries,
	}
}

func renderLiveShadowIndex(latest map[string]any, recentRuns []any, rollup map[string]any) string {
	artifacts, _ := latest["artifacts"].(map[string]any)
	summary, _ := latest["summary"].(map[string]any)
	rollback, _ := latest["rollback_trigger_surface"].(map[string]any)
	rollupSummary, _ := rollup["summary"].(map[string]any)
	lines := []string{
		"# Live Shadow Mirror Index",
		"",
		fmt.Sprintf("- Latest run: `%v`", latest["run_id"]),
		fmt.Sprintf("- Generated at: `%v`", latest["generated_at"]),
		fmt.Sprintf("- Status: `%v`", latest["status"]),
		fmt.Sprintf("- Severity: `%v`", latest["severity"]),
		fmt.Sprintf("- Bundle: `%v`", latest["bundle_path"]),
		fmt.Sprintf("- Summary JSON: `%v`", latest["summary_path"]),
		"",
		"## Latest bundle artifacts",
		"",
		fmt.Sprintf("- Shadow compare report: `%v`", artifacts["shadow_compare_report_path"]),
		fmt.Sprintf("- Shadow matrix report: `%v`", artifacts["shadow_matrix_report_path"]),
		fmt.Sprintf("- Parity scorecard: `%v`", artifacts["live_shadow_scorecard_path"]),
		fmt.Sprintf("- Rollback trigger surface: `%v`", artifacts["rollback_trigger_surface_path"]),
		"",
		"## Latest run summary",
		"",
		fmt.Sprintf("- Compare trace: `%v`", latest["compare_trace_id"]),
		fmt.Sprintf("- Matrix trace count: `%d`", lenAny(latest["matrix_trace_ids"])),
		fmt.Sprintf("- Evidence runs: `%v`", summary["total_evidence_runs"]),
		fmt.Sprintf("- Parity-ok entries: `%v`", summary["parity_ok_count"]),
		fmt.Sprintf("- Drift-detected entries: `%v`", summary["drift_detected_count"]),
		fmt.Sprintf("- Matrix total: `%v`", summary["matrix_total"]),
		fmt.Sprintf("- Matrix mismatched: `%v`", summary["matrix_mismatched"]),
		fmt.Sprintf("- Fresh inputs: `%v`", summary["fresh_inputs"]),
		fmt.Sprintf("- Stale inputs: `%v`", summary["stale_inputs"]),
		fmt.Sprintf("- Rollback trigger surface status: `%v`", rollback["status"]),
		fmt.Sprintf("- Rollback automation boundary: `%v`", rollback["automation_boundary"]),
		fmt.Sprintf("- Rollback trigger distinctions: `%v`", rollback["distinctions"]),
		"",
		"## Parity drift rollup",
		"",
		fmt.Sprintf("- Status: `%v`", rollup["status"]),
		fmt.Sprintf("- Latest run: `%v`", rollupSummary["latest_run_id"]),
		fmt.Sprintf("- Highest severity: `%v`", rollupSummary["highest_severity"]),
		fmt.Sprintf("- Drift-detected runs in window: `%v`", rollupSummary["drift_detected_runs"]),
		fmt.Sprintf("- Stale runs in window: `%v`", rollupSummary["stale_runs"]),
		"",
		"## Workflow closeout commands",
		"",
	}
	if commands, ok := latest["closeout_commands"].([]any); ok {
		for _, command := range commands {
			lines = append(lines, fmt.Sprintf("- `%v`", command))
		}
	}
	lines = append(lines, "", "## Recent bundles", "")
	for _, item := range recentRuns {
		entry, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- `%v` · `%v` · `%v` · `%v` · `%v`", entry["run_id"], entry["status"], entry["severity"], entry["generated_at"], entry["bundle_path"]))
	}
	lines = append(lines, "", "## Linked migration docs", "")
	for _, item := range liveShadowDocLinks {
		entry, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- `%v` %v", entry["path"], entry["description"]))
	}
	lines = append(lines, "", "## Parallel follow-up digests", "")
	for _, item := range liveShadowFollowupDigests {
		entry, _ := item.(map[string]any)
		lines = append(lines, fmt.Sprintf("- `%v` %v", entry["path"], entry["description"]))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func deriveLiveShadowRunID(scorecard map[string]any, generatedAt time.Time) string {
	summary, _ := scorecard["summary"].(map[string]any)
	if latest := automationFirstText(summary["latest_evidence_timestamp"]); latest != "" {
		if parsed, err := automationParseFlexibleTime(latest); err == nil {
			return parsed.UTC().Format("20060102T150405Z")
		}
	}
	return generatedAt.UTC().Format("20060102T150405Z")
}

func lenAny(value any) int {
	items, _ := value.([]any)
	return len(items)
}
