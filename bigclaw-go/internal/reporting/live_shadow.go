package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	LiveShadowScorecardGenerator  = "bigclaw-go/scripts/migration/live_shadow_scorecard/main.go"
	LiveShadowBundleGenerator     = "bigclaw-go/scripts/migration/export_live_shadow_bundle/main.go"
	defaultShadowCompareReport    = "bigclaw-go/docs/reports/shadow-compare-report.json"
	defaultShadowMatrixReport     = "bigclaw-go/docs/reports/shadow-matrix-report.json"
	defaultShadowScorecardReport  = "docs/reports/live-shadow-mirror-scorecard.json"
	defaultShadowBundleRoot       = "docs/reports/live-shadow-runs"
	defaultShadowSummaryPath      = "docs/reports/live-shadow-summary.json"
	defaultShadowIndexPath        = "docs/reports/live-shadow-index.md"
	defaultShadowManifestPath     = "docs/reports/live-shadow-index.json"
	defaultShadowRollupPath       = "docs/reports/live-shadow-drift-rollup.json"
	defaultRollbackTriggerSurface = "docs/reports/rollback-trigger-surface.json"
)

const (
	shadowTimelineDriftToleranceSeconds = 0.25
	shadowEvidenceFreshnessSLOHours     = 168
)

var (
	shadowFollowupDigests = []shadowLink{
		{Path: "docs/reports/live-shadow-comparison-follow-up-digest.md", Description: "Live shadow traffic comparison caveats are consolidated here."},
		{Path: "docs/reports/rollback-safeguard-follow-up-digest.md", Description: "Rollback remains operator-driven; this digest explains the guardrail visibility and trigger caveats."},
	}
	shadowDocLinks = []shadowLink{
		{Path: "docs/migration-shadow.md", Description: "Shadow helper workflow and bundle generation steps."},
		{Path: "docs/reports/migration-readiness-report.md", Description: "Migration readiness summary linked to the shadow bundle."},
		{Path: "docs/reports/migration-plan-review-notes.md", Description: "Review notes tied to the shadow bundle index."},
		{Path: defaultRollbackTriggerSurface, Description: "Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle."},
	}
)

type shadowLink struct {
	Path        string
	Description string
}

type LiveShadowScorecardOptions struct {
	ShadowCompareReportPath string
	ShadowMatrixReportPath  string
	Now                     time.Time
}

type LiveShadowBundleOptions struct {
	ShadowCompareReportPath string
	ShadowMatrixReportPath  string
	ScorecardReportPath     string
	BundleRootPath          string
	SummaryPath             string
	IndexPath               string
	ManifestPath            string
	RollupPath              string
	RunID                   string
	Now                     time.Time
}

func BuildLiveShadowScorecard(root string, options LiveShadowScorecardOptions) (map[string]any, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("go root is required")
	}
	if options.ShadowCompareReportPath == "" {
		options.ShadowCompareReportPath = defaultShadowCompareReport
	}
	if options.ShadowMatrixReportPath == "" {
		options.ShadowMatrixReportPath = defaultShadowMatrixReport
	}
	var compareReport map[string]any
	if err := loadJSON(resolveReportPath(root, options.ShadowCompareReportPath), &compareReport); err != nil {
		return nil, err
	}
	var matrixReport map[string]any
	if err := loadJSON(resolveReportPath(root, options.ShadowMatrixReportPath), &matrixReport); err != nil {
		return nil, err
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = maxShadowTimestamp(latestShadowReportTimestamp(compareReport), []time.Time{latestShadowReportTimestamp(matrixReport)})
		if now.IsZero() {
			now = time.Now().UTC()
		}
	}

	parityEntries := []map[string]any{buildShadowCompareEntry(compareReport)}
	parityEntries = append(parityEntries, buildShadowMatrixEntries(matrixReport)...)
	parityOKCount := 0
	for _, entry := range parityEntries {
		if asString(asMap(entry["parity"])["status"]) == "parity-ok" {
			parityOKCount++
		}
	}
	driftDetectedCount := len(parityEntries) - parityOKCount

	freshness := []map[string]any{
		buildShadowFreshnessEntry("shadow-compare-report", options.ShadowCompareReportPath, compareReport, now),
		buildShadowFreshnessEntry("shadow-matrix-report", options.ShadowMatrixReportPath, matrixReport, now),
	}
	staleInputs := 0
	freshInputs := 0
	latestEvidence := time.Time{}
	latestEvidenceString := ""
	for _, item := range freshness {
		if asString(item["status"]) == "fresh" {
			freshInputs++
		} else {
			staleInputs++
		}
		if ts := asString(item["latest_evidence_timestamp"]); ts != "" {
			if parsed, err := parseFlexibleTime(ts); err == nil && parsed.After(latestEvidence) {
				latestEvidence = parsed
				latestEvidenceString = parsed.UTC().Format(time.RFC3339)
			}
		}
	}

	matrixCoverage := asMap(matrixReport["corpus_coverage"])
	cutoverCheckpoints := []map[string]any{
		shadowCheck(
			"single_compare_matches_terminal_state_and_event_sequence",
			asBool(asMap(compareReport["diff"])["state_equal"]) && asBool(asMap(compareReport["diff"])["event_types_equal"]),
			fmt.Sprintf("trace_id=%s", asString(compareReport["trace_id"])),
		),
		shadowCheck(
			"matrix_reports_no_state_or_event_sequence_mismatches",
			asInt(matrixReport["mismatched"]) == 0,
			fmt.Sprintf("matched=%d mismatched=%d", asInt(matrixReport["matched"]), asInt(matrixReport["mismatched"])),
		),
		shadowCheck(
			"scorecard_detects_no_parity_drift",
			driftDetectedCount == 0,
			fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, driftDetectedCount),
		),
		shadowCheck(
			"checked_in_evidence_is_fresh_enough_for_review",
			staleInputs == 0,
			fmt.Sprintf("freshness_statuses=%v", shadowStatuses(freshness)),
		),
		shadowCheck(
			"matrix_includes_corpus_coverage_overlay",
			len(matrixCoverage) > 0,
			fmt.Sprintf("corpus_slice_count=%d", asInt(matrixCoverage["corpus_slice_count"])),
		),
	}

	report := map[string]any{
		"generated_at": now.Format(time.RFC3339Nano),
		"ticket":       "BIG-PAR-092",
		"title":        "Live shadow mirror parity drift scorecard",
		"status":       "repo-native-live-shadow-scorecard",
		"evidence_inputs": map[string]any{
			"shadow_compare_report_path": options.ShadowCompareReportPath,
			"shadow_matrix_report_path":  options.ShadowMatrixReportPath,
			"generator_script":           LiveShadowScorecardGenerator,
		},
		"summary": map[string]any{
			"total_evidence_runs":          len(parityEntries),
			"parity_ok_count":              parityOKCount,
			"drift_detected_count":         driftDetectedCount,
			"matrix_total":                 asInt(matrixReport["total"]),
			"matrix_matched":               asInt(matrixReport["matched"]),
			"matrix_mismatched":            asInt(matrixReport["mismatched"]),
			"corpus_coverage_present":      len(matrixCoverage) > 0,
			"corpus_uncovered_slice_count": asInt(matrixCoverage["uncovered_corpus_slice_count"]),
			"latest_evidence_timestamp":    latestEvidenceString,
			"fresh_inputs":                 freshInputs,
			"stale_inputs":                 staleInputs,
		},
		"freshness":           freshness,
		"parity_entries":      parityEntries,
		"cutover_checkpoints": cutoverCheckpoints,
		"limitations": []string{
			"repo-native only: this scorecard summarizes checked-in shadow artifacts rather than an always-on production traffic mirror",
			"parity drift is measured from fixture-backed compare/matrix runs and optional corpus slices, not mirrored tenant traffic",
			"freshness is derived from the latest artifact event timestamps and should be treated as evidence recency, not live service health",
		},
		"future_work": []string{
			"replace offline fixture submission with a real ingress mirror or tenant-scoped shadow routing control before treating this as cutover-proof traffic parity",
			"promote parity drift review from checked-in artifacts into a continuously refreshed operational surface",
			"pair this scorecard with rollback automation only after tenant-scoped rollback guardrails exist",
		},
	}
	return report, nil
}

func ExportLiveShadowBundle(root string, options LiveShadowBundleOptions) (map[string]any, string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, "", errors.New("go root is required")
	}
	if options.ShadowCompareReportPath == "" {
		options.ShadowCompareReportPath = "docs/reports/shadow-compare-report.json"
	}
	if options.ShadowMatrixReportPath == "" {
		options.ShadowMatrixReportPath = "docs/reports/shadow-matrix-report.json"
	}
	if options.ScorecardReportPath == "" {
		options.ScorecardReportPath = defaultShadowScorecardReport
	}
	if options.BundleRootPath == "" {
		options.BundleRootPath = defaultShadowBundleRoot
	}
	if options.SummaryPath == "" {
		options.SummaryPath = defaultShadowSummaryPath
	}
	if options.IndexPath == "" {
		options.IndexPath = defaultShadowIndexPath
	}
	if options.ManifestPath == "" {
		options.ManifestPath = defaultShadowManifestPath
	}
	if options.RollupPath == "" {
		options.RollupPath = defaultShadowRollupPath
	}
	compareReportPath := resolveReportPath(root, options.ShadowCompareReportPath)
	matrixReportPath := resolveReportPath(root, options.ShadowMatrixReportPath)
	scorecardReportPath := resolveReportPath(root, options.ScorecardReportPath)
	rollbackTriggerPath := resolveReportPath(root, defaultRollbackTriggerSurface)

	var compareReport map[string]any
	if err := loadJSON(compareReportPath, &compareReport); err != nil {
		return nil, "", err
	}
	var matrixReport map[string]any
	if err := loadJSON(matrixReportPath, &matrixReport); err != nil {
		return nil, "", err
	}
	var scorecardReport map[string]any
	if err := loadJSON(scorecardReportPath, &scorecardReport); err != nil {
		return nil, "", err
	}
	var rollbackReport map[string]any
	if err := loadJSON(rollbackTriggerPath, &rollbackReport); err != nil {
		return nil, "", err
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = latestShadowReportTimestamp(compareReport)
		if matrixLatest := latestShadowReportTimestamp(matrixReport); matrixLatest.After(now) {
			now = matrixLatest
		}
		if now.IsZero() {
			now = time.Now().UTC()
		}
	}

	runID := options.RunID
	if runID == "" {
		runID = deriveLiveShadowRunID(scorecardReport, now)
	}
	bundleDir := resolveReportPath(root, filepath.Join(options.BundleRootPath, runID))
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, "", err
	}

	summary, err := buildLiveShadowRunSummary(root, bundleDir, runID, compareReport, matrixReport, scorecardReport, rollbackReport, now)
	if err != nil {
		return nil, "", err
	}
	if err := WriteJSON(filepath.Join(bundleDir, "summary.json"), summary); err != nil {
		return nil, "", err
	}
	if err := WriteJSON(resolveReportPath(root, options.SummaryPath), summary); err != nil {
		return nil, "", err
	}

	recentRuns, err := loadRecentLiveShadowRuns(resolveReportPath(root, options.BundleRootPath))
	if err != nil {
		return nil, "", err
	}
	rollup := buildLiveShadowRollup(recentRuns, 5, now)
	manifest := map[string]any{
		"latest":       summary,
		"recent_runs":  buildLiveShadowRecentRunWindow(recentRuns),
		"drift_rollup": rollup,
	}
	if err := WriteJSON(resolveReportPath(root, options.RollupPath), rollup); err != nil {
		return nil, "", err
	}
	if err := WriteJSON(resolveReportPath(root, options.ManifestPath), manifest); err != nil {
		return nil, "", err
	}

	indexText := renderLiveShadowIndex(summary, buildLiveShadowRecentRunWindow(recentRuns), rollup)
	indexPath := resolveReportPath(root, options.IndexPath)
	if err := os.WriteFile(indexPath, []byte(indexText), 0o644); err != nil {
		return nil, "", err
	}
	if err := copyLiveShadowTextArtifact(indexPath, filepath.Join(bundleDir, "README.md")); err != nil {
		return nil, "", err
	}

	return manifest, indexText, nil
}

func buildShadowFreshnessEntry(name string, reportPath string, report map[string]any, generatedAt time.Time) map[string]any {
	latest := latestShadowReportTimestamp(report)
	ageHours := any(nil)
	status := "missing-timestamps"
	latestText := any(nil)
	if !latest.IsZero() {
		ageHours = roundToTwoDecimals(generatedAt.Sub(latest).Hours())
		if generatedAt.Sub(latest).Hours() <= shadowEvidenceFreshnessSLOHours {
			status = "fresh"
		} else {
			status = "stale"
		}
		latestText = latest.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"name":                      name,
		"report_path":               reportPath,
		"latest_evidence_timestamp": latestText,
		"age_hours":                 ageHours,
		"freshness_slo_hours":       shadowEvidenceFreshnessSLOHours,
		"status":                    status,
	}
}

func buildShadowCompareEntry(report map[string]any) map[string]any {
	return map[string]any{
		"entry_type":      "single-compare",
		"label":           "single fixture compare",
		"trace_id":        report["trace_id"],
		"source_file":     nil,
		"source_kind":     "fixture",
		"parity":          classifyShadowParity(asMap(report["diff"])),
		"primary_task_id": asMap(report["primary"])["task_id"],
		"shadow_task_id":  asMap(report["shadow"])["task_id"],
	}
}

func buildShadowMatrixEntries(report map[string]any) []map[string]any {
	results := asSlice(report["results"])
	entries := make([]map[string]any, 0, len(results))
	for _, item := range results {
		entry := asMap(item)
		entries = append(entries, map[string]any{
			"entry_type":      "matrix-row",
			"label":           entry["source_file"],
			"trace_id":        entry["trace_id"],
			"source_file":     entry["source_file"],
			"source_kind":     entry["source_kind"],
			"task_shape":      entry["task_shape"],
			"corpus_slice":    entry["corpus_slice"],
			"parity":          classifyShadowParity(asMap(entry["diff"])),
			"primary_task_id": asMap(entry["primary"])["task_id"],
			"shadow_task_id":  asMap(entry["shadow"])["task_id"],
		})
	}
	return entries
}

func classifyShadowParity(diff map[string]any) map[string]any {
	reasons := make([]string, 0, 4)
	if !asBool(diff["state_equal"]) {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !asBool(diff["event_types_equal"]) {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if asInt(diff["event_count_delta"]) != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := roundToTwoDecimals(absFloat(asFloat(diff["primary_timeline_seconds"]) - asFloat(diff["shadow_timeline_seconds"])))
	if timelineDelta > shadowTimelineDriftToleranceSeconds {
		reasons = append(reasons, "timeline-drift")
	}
	status := "parity-ok"
	if len(reasons) > 0 {
		status = "drift-detected"
	}
	return map[string]any{
		"status":                           status,
		"timeline_delta_seconds":           timelineDelta,
		"timeline_drift_tolerance_seconds": shadowTimelineDriftToleranceSeconds,
		"reasons":                          reasons,
	}
}

func shadowCheck(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func latestShadowReportTimestamp(report map[string]any) time.Time {
	latest := time.Time{}
	if results := asSlice(report["results"]); len(results) > 0 {
		for _, item := range results {
			entry := asMap(item)
			latest = maxShadowTimestamp(latest, collectShadowEventTimestamps(asSlice(asMap(entry["primary"])["events"])))
			latest = maxShadowTimestamp(latest, collectShadowEventTimestamps(asSlice(asMap(entry["shadow"])["events"])))
		}
		return latest
	}
	latest = maxShadowTimestamp(latest, collectShadowEventTimestamps(asSlice(asMap(report["primary"])["events"])))
	latest = maxShadowTimestamp(latest, collectShadowEventTimestamps(asSlice(asMap(report["shadow"])["events"])))
	return latest
}

func collectShadowEventTimestamps(events []any) []time.Time {
	timestamps := make([]time.Time, 0, len(events))
	for _, item := range events {
		event := asMap(item)
		if ts := asString(event["timestamp"]); ts != "" {
			if parsed, err := parseFlexibleTime(ts); err == nil {
				timestamps = append(timestamps, parsed)
			}
		}
	}
	return timestamps
}

func maxShadowTimestamp(current time.Time, candidates []time.Time) time.Time {
	for _, item := range candidates {
		if item.After(current) {
			current = item
		}
	}
	return current
}

func shadowStatuses(values []map[string]any) []string {
	out := make([]string, 0, len(values))
	for _, item := range values {
		out = append(out, asString(item["status"]))
	}
	return out
}

func buildLiveShadowRunSummary(root string, bundleDir string, runID string, compareReport map[string]any, matrixReport map[string]any, scorecardReport map[string]any, rollbackReport map[string]any, generatedAt time.Time) (map[string]any, error) {
	compareBundlePath, err := copyLiveShadowJSONArtifact(resolveReportPath(root, "docs/reports/shadow-compare-report.json"), filepath.Join(bundleDir, "shadow-compare-report.json"))
	if err != nil {
		return nil, err
	}
	matrixBundlePath, err := copyLiveShadowJSONArtifact(resolveReportPath(root, "docs/reports/shadow-matrix-report.json"), filepath.Join(bundleDir, "shadow-matrix-report.json"))
	if err != nil {
		return nil, err
	}
	scorecardBundlePath, err := copyLiveShadowJSONArtifact(resolveReportPath(root, defaultShadowScorecardReport), filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json"))
	if err != nil {
		return nil, err
	}
	rollbackBundlePath, err := copyLiveShadowJSONArtifact(resolveReportPath(root, defaultRollbackTriggerSurface), filepath.Join(bundleDir, "rollback-trigger-surface.json"))
	if err != nil {
		return nil, err
	}

	scorecardSummary := asMap(scorecardReport["summary"])
	freshness := asSlice(scorecardReport["freshness"])
	severity := classifyLiveShadowSeverity(scorecardReport)
	status := "parity-ok"
	if severityRank(severity) > 0 {
		status = "attention-needed"
	}

	return map[string]any{
		"run_id":       runID,
		"generated_at": generatedAt.Format(time.RFC3339Nano),
		"status":       status,
		"severity":     severity,
		"bundle_path":  relPathFromRoot(root, bundleDir),
		"summary_path": relPathFromRoot(root, filepath.Join(bundleDir, "summary.json")),
		"artifacts": map[string]any{
			"shadow_compare_report_path":    relPathFromRoot(root, compareBundlePath),
			"shadow_matrix_report_path":     relPathFromRoot(root, matrixBundlePath),
			"live_shadow_scorecard_path":    relPathFromRoot(root, scorecardBundlePath),
			"rollback_trigger_surface_path": relPathFromRoot(root, rollbackBundlePath),
		},
		"latest_evidence_timestamp": scorecardSummary["latest_evidence_timestamp"],
		"freshness":                 freshness,
		"summary": map[string]any{
			"total_evidence_runs":  asInt(scorecardSummary["total_evidence_runs"]),
			"parity_ok_count":      asInt(scorecardSummary["parity_ok_count"]),
			"drift_detected_count": asInt(scorecardSummary["drift_detected_count"]),
			"matrix_total":         asInt(scorecardSummary["matrix_total"]),
			"matrix_mismatched":    asInt(scorecardSummary["matrix_mismatched"]),
			"stale_inputs":         asInt(scorecardSummary["stale_inputs"]),
			"fresh_inputs":         asInt(scorecardSummary["fresh_inputs"]),
		},
		"rollback_trigger_surface": map[string]any{
			"status":                     asMap(rollbackReport["summary"])["status"],
			"automation_boundary":        asMap(rollbackReport["summary"])["automation_boundary"],
			"automated_rollback_trigger": asBool(asMap(rollbackReport["summary"])["automated_rollback_trigger"]),
			"distinctions":               asMap(asMap(rollbackReport["summary"])["distinctions"]),
			"issue":                      asMap(asMap(rollbackReport["reviewer_path"])["digest_issue"]),
			"digest_path":                asMap(rollbackReport["reviewer_path"])["digest_path"],
			"summary_path":               defaultRollbackTriggerSurface,
		},
		"compare_trace_id":    compareReport["trace_id"],
		"matrix_trace_ids":    collectLiveShadowTraceIDs(matrixReport),
		"cutover_checkpoints": scorecardReport["cutover_checkpoints"],
		"closeout_commands": []string{
			"cd bigclaw-go && go run ./scripts/migration/live_shadow_scorecard --pretty",
			"cd bigclaw-go && go run ./scripts/migration/export_live_shadow_bundle",
			"cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned",
			"git push origin <branch> && git log -1 --stat",
		},
	}, nil
}

func classifyLiveShadowSeverity(scorecardReport map[string]any) string {
	summary := asMap(scorecardReport["summary"])
	if asInt(summary["stale_inputs"]) > 0 {
		return "high"
	}
	if asInt(summary["drift_detected_count"]) > 0 {
		return "medium"
	}
	for _, item := range asSlice(scorecardReport["cutover_checkpoints"]) {
		if !asBool(asMap(item)["passed"]) {
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

func collectLiveShadowTraceIDs(matrixReport map[string]any) []string {
	out := make([]string, 0, len(asSlice(matrixReport["results"])))
	for _, item := range asSlice(matrixReport["results"]) {
		traceID := asString(asMap(item)["trace_id"])
		if traceID != "" {
			out = append(out, traceID)
		}
	}
	return out
}

func loadRecentLiveShadowRuns(bundleRoot string) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	recentRuns := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		if !pathExists(summaryPath) {
			continue
		}
		var payload map[string]any
		if err := loadJSON(summaryPath, &payload); err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, payload)
	}
	sort.Slice(recentRuns, func(i, j int) bool {
		return asString(recentRuns[i]["generated_at"]) > asString(recentRuns[j]["generated_at"])
	})
	return recentRuns, nil
}

func buildLiveShadowRollup(recentRuns []map[string]any, limit int, now time.Time) map[string]any {
	if limit <= 0 || limit > len(recentRuns) {
		limit = len(recentRuns)
	}
	window := recentRuns[:limit]
	highestSeverity := "none"
	statusCounts := map[string]any{"parity_ok": 0, "attention_needed": 0}
	staleRuns := 0
	driftDetectedRuns := 0
	items := make([]map[string]any, 0, len(window))
	for _, item := range window {
		severity := asString(item["severity"])
		if severityRank(severity) > severityRank(highestSeverity) {
			highestSeverity = severity
		}
		if asString(item["status"]) == "parity-ok" {
			statusCounts["parity_ok"] = asInt(statusCounts["parity_ok"]) + 1
		} else {
			statusCounts["attention_needed"] = asInt(statusCounts["attention_needed"]) + 1
		}
		summary := asMap(item["summary"])
		staleInputCount := asInt(summary["stale_inputs"])
		driftInputCount := asInt(summary["drift_detected_count"])
		if staleInputCount > 0 {
			staleRuns++
		}
		if driftInputCount > 0 {
			driftDetectedRuns++
		}
		items = append(items, map[string]any{
			"run_id":                    item["run_id"],
			"generated_at":              item["generated_at"],
			"status":                    item["status"],
			"severity":                  item["severity"],
			"latest_evidence_timestamp": item["latest_evidence_timestamp"],
			"drift_detected_count":      driftInputCount,
			"stale_inputs":              staleInputCount,
			"bundle_path":               item["bundle_path"],
			"summary_path":              item["summary_path"],
		})
	}
	status := "parity-ok"
	if severityRank(highestSeverity) > 0 {
		status = "attention-needed"
	}
	summary := map[string]any{
		"recent_run_count":    len(window),
		"drift_detected_runs": driftDetectedRuns,
		"stale_runs":          staleRuns,
		"highest_severity":    highestSeverity,
		"status_counts":       statusCounts,
	}
	if len(window) > 0 {
		summary["latest_run_id"] = window[0]["run_id"]
	}
	return map[string]any{
		"generated_at": now.Format(time.RFC3339Nano),
		"status":       status,
		"window_size":  limit,
		"summary":      summary,
		"recent_runs":  items,
	}
}

func buildLiveShadowRecentRunWindow(recentRuns []map[string]any) []map[string]any {
	items := make([]map[string]any, 0, len(recentRuns))
	for _, item := range recentRuns {
		items = append(items, map[string]any{
			"run_id":       item["run_id"],
			"generated_at": item["generated_at"],
			"status":       item["status"],
			"severity":     item["severity"],
			"bundle_path":  item["bundle_path"],
			"summary_path": item["summary_path"],
		})
	}
	return items
}

func renderLiveShadowIndex(latest map[string]any, recentRuns []map[string]any, rollup map[string]any) string {
	latestSummary := asMap(latest["summary"])
	latestArtifacts := asMap(latest["artifacts"])
	rollback := asMap(latest["rollback_trigger_surface"])
	lines := []string{
		"# Live Shadow Mirror Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", asString(latest["run_id"])),
		fmt.Sprintf("- Generated at: `%s`", asString(latest["generated_at"])),
		fmt.Sprintf("- Status: `%s`", asString(latest["status"])),
		fmt.Sprintf("- Severity: `%s`", asString(latest["severity"])),
		fmt.Sprintf("- Bundle: `%s`", asString(latest["bundle_path"])),
		fmt.Sprintf("- Summary JSON: `%s`", asString(latest["summary_path"])),
		"",
		"## Latest bundle artifacts",
		"",
		fmt.Sprintf("- Shadow compare report: `%s`", asString(latestArtifacts["shadow_compare_report_path"])),
		fmt.Sprintf("- Shadow matrix report: `%s`", asString(latestArtifacts["shadow_matrix_report_path"])),
		fmt.Sprintf("- Parity scorecard: `%s`", asString(latestArtifacts["live_shadow_scorecard_path"])),
		fmt.Sprintf("- Rollback trigger surface: `%s`", asString(latestArtifacts["rollback_trigger_surface_path"])),
		"",
		"## Latest run summary",
		"",
		fmt.Sprintf("- Compare trace: `%s`", asString(latest["compare_trace_id"])),
		fmt.Sprintf("- Matrix trace count: `%d`", len(asSlice(latest["matrix_trace_ids"]))),
		fmt.Sprintf("- Evidence runs: `%d`", asInt(latestSummary["total_evidence_runs"])),
		fmt.Sprintf("- Parity-ok entries: `%d`", asInt(latestSummary["parity_ok_count"])),
		fmt.Sprintf("- Drift-detected entries: `%d`", asInt(latestSummary["drift_detected_count"])),
		fmt.Sprintf("- Matrix total: `%d`", asInt(latestSummary["matrix_total"])),
		fmt.Sprintf("- Matrix mismatched: `%d`", asInt(latestSummary["matrix_mismatched"])),
		fmt.Sprintf("- Fresh inputs: `%d`", asInt(latestSummary["fresh_inputs"])),
		fmt.Sprintf("- Stale inputs: `%d`", asInt(latestSummary["stale_inputs"])),
		fmt.Sprintf("- Rollback trigger surface status: `%s`", asString(rollback["status"])),
		fmt.Sprintf("- Rollback automation boundary: `%s`", asString(rollback["automation_boundary"])),
		fmt.Sprintf("- Rollback trigger distinctions: `%s`", stringifyCompact(rollback["distinctions"])),
		"",
		"## Parity drift rollup",
		"",
		fmt.Sprintf("- Status: `%s`", asString(rollup["status"])),
		fmt.Sprintf("- Latest run: `%s`", asString(asMap(rollup["summary"])["latest_run_id"])),
		fmt.Sprintf("- Highest severity: `%s`", asString(asMap(rollup["summary"])["highest_severity"])),
		fmt.Sprintf("- Drift-detected runs in window: `%d`", asInt(asMap(rollup["summary"])["drift_detected_runs"])),
		fmt.Sprintf("- Stale runs in window: `%d`", asInt(asMap(rollup["summary"])["stale_runs"])),
		"",
		"## Workflow closeout commands",
		"",
	}
	for _, command := range asSlice(latest["closeout_commands"]) {
		lines = append(lines, fmt.Sprintf("- `%s`", asString(command)))
	}
	lines = append(lines, "", "## Recent bundles", "")
	for _, item := range recentRuns {
		lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s` · `%s`", asString(item["run_id"]), asString(item["status"]), asString(item["severity"]), asString(item["generated_at"]), asString(item["bundle_path"])))
	}
	lines = append(lines, "", "## Linked migration docs", "")
	for _, link := range shadowDocLinks {
		lines = append(lines, fmt.Sprintf("- `%s` %s", link.Path, link.Description))
	}
	lines = append(lines, "", "## Parallel Follow-up Index", "")
	lines = append(lines,
		"- `docs/reports/parallel-follow-up-index.md` is the canonical index for the",
		"  remaining live-shadow, rollback, and corpus-coverage follow-up digests behind",
		"  this run bundle.",
		"- Use `docs/reports/parallel-validation-matrix.md` first when a shadow review",
		"  needs the checked-in local/Kubernetes/Ray validation entrypoint alongside the",
		"  shadow evidence bundle.",
		"- For the two primary caveat tracks referenced by this bundle, see",
		"  `OPE-266` / `BIG-PAR-092` in",
		"  `docs/reports/live-shadow-comparison-follow-up-digest.md` and",
		"  `OPE-254` / `BIG-PAR-088` in",
		"  `docs/reports/rollback-safeguard-follow-up-digest.md`.",
	)
	return strings.Join(lines, "\n") + "\n"
}

func deriveLiveShadowRunID(scorecardReport map[string]any, generatedAt time.Time) string {
	latest := asString(asMap(scorecardReport["summary"])["latest_evidence_timestamp"])
	if latest != "" {
		if parsed, err := parseFlexibleTime(latest); err == nil {
			return parsed.UTC().Format("20060102T150405Z")
		}
	}
	return generatedAt.UTC().Format("20060102T150405Z")
}

func copyLiveShadowJSONArtifact(source string, destination string) (string, error) {
	var payload any
	if err := loadJSON(source, &payload); err != nil {
		return "", err
	}
	if err := WriteJSON(destination, payload); err != nil {
		return "", err
	}
	return destination, nil
}

func copyLiveShadowTextArtifact(source string, destination string) error {
	contents, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, contents, 0o644)
}

func relPathFromRoot(root string, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

func asFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func stringifyCompact(value any) string {
	contents, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(contents)
}
