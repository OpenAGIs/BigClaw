package liveshadowscorecard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	timelineDriftToleranceSeconds = 0.25
	evidenceFreshnessSLOHours     = 168.0
)

type BuildOptions struct {
	RepoRoot                string
	ShadowCompareReportPath string
	ShadowMatrixReportPath  string
	GeneratedAt             time.Time
}

func BuildReport(opts BuildOptions) (map[string]any, error) {
	repoRoot := defaultString(opts.RepoRoot, ".")
	comparePath := defaultString(opts.ShadowCompareReportPath, "bigclaw-go/docs/reports/shadow-compare-report.json")
	matrixPath := defaultString(opts.ShadowMatrixReportPath, "bigclaw-go/docs/reports/shadow-matrix-report.json")
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	var compareReport map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, comparePath), &compareReport); err != nil {
		return nil, err
	}
	var matrixReport map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, matrixPath), &matrixReport); err != nil {
		return nil, err
	}

	parityEntries := append([]map[string]any{buildCompareEntry(compareReport)}, buildMatrixEntries(matrixReport)...)
	parityOKCount := 0
	for _, item := range parityEntries {
		if stringValue(nestedMap(item, "parity")["status"], "") == "parity-ok" {
			parityOKCount++
		}
	}
	driftDetectedCount := len(parityEntries) - parityOKCount

	freshness := []map[string]any{
		buildFreshnessEntry("shadow-compare-report", comparePath, compareReport, generatedAt),
		buildFreshnessEntry("shadow-matrix-report", matrixPath, matrixReport, generatedAt),
	}
	staleInputs := make([]map[string]any, 0)
	freshInputs := 0
	latestEvidenceTimestamp := ""
	for _, item := range freshness {
		if stringValue(item["status"], "") == "fresh" {
			freshInputs++
		} else {
			staleInputs = append(staleInputs, item)
		}
		if ts := stringValue(item["latest_evidence_timestamp"], ""); ts != "" && ts > latestEvidenceTimestamp {
			latestEvidenceTimestamp = ts
		}
	}
	if latestEvidenceTimestamp == "" {
		latestEvidenceTimestamp = ""
	}

	matrixCorpusCoverage := nestedMap(matrixReport, "corpus_coverage")
	cutoverCheckpoints := []map[string]any{
		check(
			"single_compare_matches_terminal_state_and_event_sequence",
			boolValue(nestedMap(compareReport, "diff")["state_equal"], false) && boolValue(nestedMap(compareReport, "diff")["event_types_equal"], false),
			fmt.Sprintf("trace_id=%s", stringValue(compareReport["trace_id"], "")),
		),
		check(
			"matrix_reports_no_state_or_event_sequence_mismatches",
			intValue(matrixReport["mismatched"]) == 0,
			fmt.Sprintf("matched=%d mismatched=%d", intValue(matrixReport["matched"]), intValue(matrixReport["mismatched"])),
		),
		check(
			"scorecard_detects_no_parity_drift",
			driftDetectedCount == 0,
			fmt.Sprintf("parity_ok=%d drift_detected=%d", parityOKCount, driftDetectedCount),
		),
		check(
			"checked_in_evidence_is_fresh_enough_for_review",
			len(staleInputs) == 0,
			fmt.Sprintf("freshness_statuses=%s", pythonStringList(collectStatuses(freshness))),
		),
		check(
			"matrix_includes_corpus_coverage_overlay",
			len(matrixCorpusCoverage) > 0,
			fmt.Sprintf("corpus_slice_count=%d", intValue(matrixCorpusCoverage["corpus_slice_count"])),
		),
	}

	return map[string]any{
		"generated_at": utcISO(generatedAt.UTC()),
		"ticket":       "BIG-PAR-092",
		"title":        "Live shadow mirror parity drift scorecard",
		"status":       "repo-native-live-shadow-scorecard",
		"evidence_inputs": map[string]any{
			"shadow_compare_report_path": comparePath,
			"shadow_matrix_report_path":  matrixPath,
			"generator_script":           "bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		},
		"summary": map[string]any{
			"total_evidence_runs":          len(parityEntries),
			"parity_ok_count":              parityOKCount,
			"drift_detected_count":         driftDetectedCount,
			"matrix_total":                 intValue(matrixReport["total"]),
			"matrix_matched":               intValue(matrixReport["matched"]),
			"matrix_mismatched":            intValue(matrixReport["mismatched"]),
			"corpus_coverage_present":      len(matrixCorpusCoverage) > 0,
			"corpus_uncovered_slice_count": intValue(matrixCorpusCoverage["uncovered_corpus_slice_count"]),
			"latest_evidence_timestamp":    nilIfEmpty(latestEvidenceTimestamp),
			"fresh_inputs":                 freshInputs,
			"stale_inputs":                 len(staleInputs),
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
	}, nil
}

func WriteReport(path string, report map[string]any) error {
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

func buildFreshnessEntry(name, path string, report map[string]any, generatedAt time.Time) map[string]any {
	latestTimestamp, ok := latestReportTimestamp(report)
	ageHours := any(nil)
	status := "missing-timestamps"
	latestTimestampValue := any(nil)
	if ok {
		ageHours = roundFloat(generatedAt.Sub(latestTimestamp).Hours(), 2)
		status = ternaryString(floatValue(ageHours) <= evidenceFreshnessSLOHours, "fresh", "stale")
		latestTimestampValue = utcISO(latestTimestamp)
	}
	return map[string]any{
		"name":                      name,
		"report_path":               path,
		"latest_evidence_timestamp": latestTimestampValue,
		"age_hours":                 ageHours,
		"freshness_slo_hours":       int(evidenceFreshnessSLOHours),
		"status":                    status,
	}
}

func latestReportTimestamp(report map[string]any) (time.Time, bool) {
	latest := time.Time{}
	found := false
	if _, ok := report["results"]; ok {
		for _, item := range mapSliceAt(report, "results") {
			for _, ts := range collectEventTimestamps(eventSlice(nestedMap(item, "primary"), "events")) {
				if !found || ts.After(latest) {
					latest = ts
					found = true
				}
			}
			for _, ts := range collectEventTimestamps(eventSlice(nestedMap(item, "shadow"), "events")) {
				if !found || ts.After(latest) {
					latest = ts
					found = true
				}
			}
		}
		return latest, found
	}
	for _, ts := range collectEventTimestamps(eventSlice(nestedMap(report, "primary"), "events")) {
		if !found || ts.After(latest) {
			latest = ts
			found = true
		}
	}
	for _, ts := range collectEventTimestamps(eventSlice(nestedMap(report, "shadow"), "events")) {
		if !found || ts.After(latest) {
			latest = ts
			found = true
		}
	}
	return latest, found
}

func collectEventTimestamps(events []map[string]any) []time.Time {
	timestamps := make([]time.Time, 0, len(events))
	for _, event := range events {
		parsed, ok := parseTime(stringValue(event["timestamp"], ""))
		if ok {
			timestamps = append(timestamps, parsed)
		}
	}
	return timestamps
}

func classifyParity(diff map[string]any) map[string]any {
	reasons := make([]string, 0)
	if !boolValue(diff["state_equal"], false) {
		reasons = append(reasons, "terminal-state-mismatch")
	}
	if !boolValue(diff["event_types_equal"], false) {
		reasons = append(reasons, "event-sequence-mismatch")
	}
	if intValue(diff["event_count_delta"]) != 0 {
		reasons = append(reasons, "event-count-drift")
	}
	timelineDelta := roundFloat(absFloat(floatValue(diff["primary_timeline_seconds"])-floatValue(diff["shadow_timeline_seconds"])), 6)
	if timelineDelta > timelineDriftToleranceSeconds {
		reasons = append(reasons, "timeline-drift")
	}
	return map[string]any{
		"status":                           ternaryString(len(reasons) == 0, "parity-ok", "drift-detected"),
		"timeline_delta_seconds":           timelineDelta,
		"timeline_drift_tolerance_seconds": timelineDriftToleranceSeconds,
		"reasons":                          reasons,
	}
}

func buildCompareEntry(report map[string]any) map[string]any {
	return map[string]any{
		"entry_type":      "single-compare",
		"label":           "single fixture compare",
		"trace_id":        report["trace_id"],
		"source_file":     nil,
		"source_kind":     "fixture",
		"parity":          classifyParity(nestedMap(report, "diff")),
		"primary_task_id": nestedMap(report, "primary")["task_id"],
		"shadow_task_id":  nestedMap(report, "shadow")["task_id"],
	}
}

func buildMatrixEntries(report map[string]any) []map[string]any {
	entries := make([]map[string]any, 0, len(mapSliceAt(report, "results")))
	for _, item := range mapSliceAt(report, "results") {
		entries = append(entries, map[string]any{
			"entry_type":      "matrix-row",
			"label":           item["source_file"],
			"trace_id":        item["trace_id"],
			"source_file":     item["source_file"],
			"source_kind":     item["source_kind"],
			"task_shape":      item["task_shape"],
			"corpus_slice":    item["corpus_slice"],
			"parity":          classifyParity(nestedMap(item, "diff")),
			"primary_task_id": nestedMap(item, "primary")["task_id"],
			"shadow_task_id":  nestedMap(item, "shadow")["task_id"],
		})
	}
	return entries
}

func check(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func eventSlice(source map[string]any, key string) []map[string]any {
	return mapSliceAt(source, key)
}

func collectStatuses(items []map[string]any) []string {
	statuses := make([]string, 0, len(items))
	for _, item := range items {
		statuses = append(statuses, stringValue(item["status"], ""))
	}
	return statuses
}

func pythonStringList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, "'"+item+"'")
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func utcISO(moment time.Time) string {
	return moment.Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, strings.Replace(value, "Z", "+00:00", 1))
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapSliceAt(source map[string]any, key string) []map[string]any {
	raw, _ := source[key].([]any)
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			items = append(items, mapped)
		}
	}
	return items
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(typed)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func roundFloat(value float64, places int) float64 {
	formatted := strconv.FormatFloat(value, 'f', places, 64)
	parsed, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return value
	}
	return parsed
}

func ternaryString(condition bool, ifTrue, ifFalse string) string {
	if condition {
		return ifTrue
	}
	return ifFalse
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
