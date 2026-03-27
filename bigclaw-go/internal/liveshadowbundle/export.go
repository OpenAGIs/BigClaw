package liveshadowbundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	shadowCompareReport   = "docs/reports/shadow-compare-report.json"
	shadowMatrixReport    = "docs/reports/shadow-matrix-report.json"
	shadowScorecardReport = "docs/reports/live-shadow-mirror-scorecard.json"
	rollbackTriggerReport = "docs/reports/rollback-trigger-surface.json"
)

var followupDigests = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/reports/live-shadow-comparison-follow-up-digest.md",
		Description: "Live shadow traffic comparison caveats are consolidated here.",
	},
	{
		Path:        "docs/reports/rollback-safeguard-follow-up-digest.md",
		Description: "Rollback remains operator-driven; this digest explains the guardrail visibility and trigger caveats.",
	},
}

var docLinks = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/migration-shadow.md",
		Description: "Shadow helper workflow and bundle generation steps.",
	},
	{
		Path:        "docs/reports/migration-readiness-report.md",
		Description: "Migration readiness summary linked to the shadow bundle.",
	},
	{
		Path:        "docs/reports/migration-plan-review-notes.md",
		Description: "Review notes tied to the shadow bundle index.",
	},
	{
		Path:        "docs/reports/rollback-trigger-surface.json",
		Description: "Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle.",
	},
}

type ExportOptions struct {
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
	GeneratedAt         time.Time
	RollupGeneratedAt   time.Time
}

type ExportResult struct {
	Latest   map[string]any
	Manifest map[string]any
	Rollup   map[string]any
	Index    string
}

func Export(opts ExportOptions) (ExportResult, error) {
	goRoot := defaultString(opts.GoRoot, ".")
	compareReportPath := defaultString(opts.ShadowCompareReport, shadowCompareReport)
	matrixReportPath := defaultString(opts.ShadowMatrixReport, shadowMatrixReport)
	scorecardReportPath := defaultString(opts.ScorecardReport, shadowScorecardReport)
	bundleRoot := defaultString(opts.BundleRoot, "docs/reports/live-shadow-runs")
	summaryPath := defaultString(opts.SummaryPath, "docs/reports/live-shadow-summary.json")
	indexPath := defaultString(opts.IndexPath, "docs/reports/live-shadow-index.md")
	manifestPath := defaultString(opts.ManifestPath, "docs/reports/live-shadow-index.json")
	rollupPath := defaultString(opts.RollupPath, "docs/reports/live-shadow-drift-rollup.json")
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	rollupGeneratedAt := opts.RollupGeneratedAt
	if rollupGeneratedAt.IsZero() {
		rollupGeneratedAt = time.Now().UTC()
	}

	var compareReport map[string]any
	if err := readJSON(filepath.Join(goRoot, compareReportPath), &compareReport); err != nil {
		return ExportResult{}, err
	}
	var matrixReport map[string]any
	if err := readJSON(filepath.Join(goRoot, matrixReportPath), &matrixReport); err != nil {
		return ExportResult{}, err
	}
	var scorecardReport map[string]any
	if err := readJSON(filepath.Join(goRoot, scorecardReportPath), &scorecardReport); err != nil {
		return ExportResult{}, err
	}

	runID := opts.RunID
	if strings.TrimSpace(runID) == "" {
		runID = deriveRunID(scorecardReport, generatedAt)
	}
	bundleDir := filepath.Join(goRoot, bundleRoot, runID)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return ExportResult{}, err
	}

	latest, err := buildRunSummary(goRoot, bundleDir, runID, compareReport, matrixReport, scorecardReport, generatedAt)
	if err != nil {
		return ExportResult{}, err
	}
	if err := writeJSON(filepath.Join(bundleDir, "summary.json"), latest); err != nil {
		return ExportResult{}, err
	}
	if err := writeJSON(filepath.Join(goRoot, summaryPath), latest); err != nil {
		return ExportResult{}, err
	}

	recentRuns, err := loadRecentRuns(filepath.Join(goRoot, bundleRoot))
	if err != nil {
		return ExportResult{}, err
	}
	rollup := buildRollup(recentRuns, rollupGeneratedAt, 5)
	manifest := map[string]any{
		"latest":       latest,
		"recent_runs":  buildRecentRunsManifest(recentRuns),
		"drift_rollup": rollup,
	}
	if err := writeJSON(filepath.Join(goRoot, rollupPath), rollup); err != nil {
		return ExportResult{}, err
	}
	if err := writeJSON(filepath.Join(goRoot, manifestPath), manifest); err != nil {
		return ExportResult{}, err
	}

	indexText := renderIndex(latest, mapSliceAny(manifest["recent_runs"]), rollup)
	if err := os.WriteFile(filepath.Join(goRoot, indexPath), []byte(indexText), 0o644); err != nil {
		return ExportResult{}, err
	}
	if err := copyTextArtifact(filepath.Join(goRoot, indexPath), filepath.Join(bundleDir, "README.md")); err != nil {
		return ExportResult{}, err
	}

	return ExportResult{
		Latest:   latest,
		Manifest: manifest,
		Rollup:   rollup,
		Index:    indexText,
	}, nil
}

func buildRunSummary(goRoot, bundleDir, runID string, compareReport, matrixReport, scorecardReport map[string]any, generatedAt time.Time) (map[string]any, error) {
	compareBundlePath := filepath.Join(bundleDir, "shadow-compare-report.json")
	matrixBundlePath := filepath.Join(bundleDir, "shadow-matrix-report.json")
	scorecardBundlePath := filepath.Join(bundleDir, "live-shadow-mirror-scorecard.json")
	rollbackBundlePath := filepath.Join(bundleDir, "rollback-trigger-surface.json")
	if err := copyJSONArtifact(filepath.Join(goRoot, shadowCompareReport), compareBundlePath); err != nil {
		return nil, err
	}
	if err := copyJSONArtifact(filepath.Join(goRoot, shadowMatrixReport), matrixBundlePath); err != nil {
		return nil, err
	}
	if err := copyJSONArtifact(filepath.Join(goRoot, shadowScorecardReport), scorecardBundlePath); err != nil {
		return nil, err
	}
	if err := copyJSONArtifact(filepath.Join(goRoot, rollbackTriggerReport), rollbackBundlePath); err != nil {
		return nil, err
	}

	var rollbackReport map[string]any
	if err := readJSON(filepath.Join(goRoot, rollbackTriggerReport), &rollbackReport); err != nil {
		return nil, err
	}

	scorecardSummary := nestedMap(scorecardReport, "summary")
	freshness := mapSliceAt(scorecardReport, "freshness")
	staleInputs := intValue(scorecardSummary["stale_inputs"])
	driftDetectedCount := intValue(scorecardSummary["drift_detected_count"])
	severity := classifySeverity(scorecardReport)
	status := "parity-ok"
	if severityRank(severity) > 0 {
		status = "attention-needed"
	}

	return map[string]any{
		"run_id":       runID,
		"generated_at": utcISO(generatedAt),
		"status":       status,
		"severity":     severity,
		"bundle_path":  relpath(bundleDir, goRoot),
		"summary_path": relpath(filepath.Join(bundleDir, "summary.json"), goRoot),
		"artifacts": map[string]any{
			"shadow_compare_report_path":    relpath(compareBundlePath, goRoot),
			"shadow_matrix_report_path":     relpath(matrixBundlePath, goRoot),
			"live_shadow_scorecard_path":    relpath(scorecardBundlePath, goRoot),
			"rollback_trigger_surface_path": relpath(rollbackBundlePath, goRoot),
		},
		"latest_evidence_timestamp": scorecardSummary["latest_evidence_timestamp"],
		"freshness":                 freshness,
		"summary": map[string]any{
			"total_evidence_runs":  intValue(scorecardSummary["total_evidence_runs"]),
			"parity_ok_count":      intValue(scorecardSummary["parity_ok_count"]),
			"drift_detected_count": driftDetectedCount,
			"matrix_total":         intValue(scorecardSummary["matrix_total"]),
			"matrix_mismatched":    intValue(scorecardSummary["matrix_mismatched"]),
			"stale_inputs":         staleInputs,
			"fresh_inputs":         intValue(scorecardSummary["fresh_inputs"]),
		},
		"rollback_trigger_surface": map[string]any{
			"status":                     nestedMap(rollbackReport, "summary")["status"],
			"automation_boundary":        nestedMap(rollbackReport, "summary")["automation_boundary"],
			"automated_rollback_trigger": boolValue(nestedMap(rollbackReport, "summary")["automated_rollback_trigger"], false),
			"distinctions":               nestedMap(nestedMap(rollbackReport, "summary"), "distinctions"),
			"issue":                      nestedMap(rollbackReport, "issue"),
			"digest_path":                nestedMap(rollbackReport, "shared_guardrail_summary")["digest_path"],
			"summary_path":               relpath(filepath.Join(goRoot, rollbackTriggerReport), goRoot),
		},
		"compare_trace_id":    compareReport["trace_id"],
		"matrix_trace_ids":    collectTraceIDs(matrixReport),
		"cutover_checkpoints": mapSliceAt(scorecardReport, "cutover_checkpoints"),
		"closeout_commands": []string{
			"cd bigclaw-go && go run ./scripts/migration/live_shadow_scorecard.go --repo-root .. --pretty",
			"cd bigclaw-go && go run ./scripts/migration/export_live_shadow_bundle.go --go-root .",
			"cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned",
			"git push origin <branch> && git log -1 --stat",
		},
	}, nil
}

func buildRecentRunsManifest(recentRuns []map[string]any) []map[string]any {
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

func loadRecentRuns(bundleRoot string) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	recentRuns := make([]map[string]any, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		if _, err := os.Stat(summaryPath); err != nil {
			continue
		}
		var payload map[string]any
		if err := readJSON(summaryPath, &payload); err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, payload)
	}
	sort.Slice(recentRuns, func(i, j int) bool {
		return stringValue(recentRuns[i]["generated_at"], "") > stringValue(recentRuns[j]["generated_at"], "")
	})
	return recentRuns, nil
}

func buildRollup(recentRuns []map[string]any, generatedAt time.Time, limit int) map[string]any {
	window := recentRuns
	if len(window) > limit {
		window = window[:limit]
	}
	highestSeverity := "none"
	statusCounts := map[string]any{
		"parity_ok":        0,
		"attention_needed": 0,
	}
	staleRuns := 0
	driftDetectedRuns := 0
	entries := make([]map[string]any, 0, len(window))
	for _, item := range window {
		severity := stringValue(item["severity"], "none")
		if severityRank(severity) > severityRank(highestSeverity) {
			highestSeverity = severity
		}
		if stringValue(item["status"], "") == "parity-ok" {
			statusCounts["parity_ok"] = intValue(statusCounts["parity_ok"]) + 1
		} else {
			statusCounts["attention_needed"] = intValue(statusCounts["attention_needed"]) + 1
		}
		summary := nestedMap(item, "summary")
		staleInputs := intValue(summary["stale_inputs"])
		driftCount := intValue(summary["drift_detected_count"])
		if staleInputs > 0 {
			staleRuns++
		}
		if driftCount > 0 {
			driftDetectedRuns++
		}
		entries = append(entries, map[string]any{
			"run_id":                    item["run_id"],
			"generated_at":              item["generated_at"],
			"status":                    item["status"],
			"severity":                  severity,
			"latest_evidence_timestamp": item["latest_evidence_timestamp"],
			"drift_detected_count":      driftCount,
			"stale_inputs":              staleInputs,
			"bundle_path":               item["bundle_path"],
			"summary_path":              item["summary_path"],
		})
	}
	status := "parity-ok"
	if severityRank(highestSeverity) > 0 {
		status = "attention-needed"
	}
	return map[string]any{
		"generated_at": utcISO(generatedAt),
		"status":       status,
		"window_size":  limit,
		"summary": map[string]any{
			"recent_run_count":    len(window),
			"drift_detected_runs": driftDetectedRuns,
			"stale_runs":          staleRuns,
			"highest_severity":    highestSeverity,
			"status_counts":       statusCounts,
			"latest_run_id":       pickLatestRunID(window),
		},
		"recent_runs": entries,
	}
}

func renderIndex(latest map[string]any, recentRuns []map[string]any, rollup map[string]any) string {
	lines := []string{
		"# Live Shadow Mirror Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", stringValue(latest["run_id"], "")),
		fmt.Sprintf("- Generated at: `%s`", stringValue(latest["generated_at"], "")),
		fmt.Sprintf("- Status: `%s`", stringValue(latest["status"], "")),
		fmt.Sprintf("- Severity: `%s`", stringValue(latest["severity"], "")),
		fmt.Sprintf("- Bundle: `%s`", stringValue(latest["bundle_path"], "")),
		fmt.Sprintf("- Summary JSON: `%s`", stringValue(latest["summary_path"], "")),
		"",
		"## Latest bundle artifacts",
		"",
		fmt.Sprintf("- Shadow compare report: `%s`", stringValue(nestedMap(latest, "artifacts")["shadow_compare_report_path"], "")),
		fmt.Sprintf("- Shadow matrix report: `%s`", stringValue(nestedMap(latest, "artifacts")["shadow_matrix_report_path"], "")),
		fmt.Sprintf("- Parity scorecard: `%s`", stringValue(nestedMap(latest, "artifacts")["live_shadow_scorecard_path"], "")),
		fmt.Sprintf("- Rollback trigger surface: `%s`", stringValue(nestedMap(latest, "artifacts")["rollback_trigger_surface_path"], "")),
		"",
		"## Latest run summary",
		"",
		fmt.Sprintf("- Compare trace: `%s`", stringValue(latest["compare_trace_id"], "")),
		fmt.Sprintf("- Matrix trace count: `%d`", len(stringSliceAny(latest["matrix_trace_ids"]))),
		fmt.Sprintf("- Evidence runs: `%d`", intValue(nestedMap(latest, "summary")["total_evidence_runs"])),
		fmt.Sprintf("- Parity-ok entries: `%d`", intValue(nestedMap(latest, "summary")["parity_ok_count"])),
		fmt.Sprintf("- Drift-detected entries: `%d`", intValue(nestedMap(latest, "summary")["drift_detected_count"])),
		fmt.Sprintf("- Matrix total: `%d`", intValue(nestedMap(latest, "summary")["matrix_total"])),
		fmt.Sprintf("- Matrix mismatched: `%d`", intValue(nestedMap(latest, "summary")["matrix_mismatched"])),
		fmt.Sprintf("- Fresh inputs: `%d`", intValue(nestedMap(latest, "summary")["fresh_inputs"])),
		fmt.Sprintf("- Stale inputs: `%d`", intValue(nestedMap(latest, "summary")["stale_inputs"])),
		fmt.Sprintf("- Rollback trigger surface status: `%s`", stringValue(nestedMap(latest, "rollback_trigger_surface")["status"], "")),
		fmt.Sprintf("- Rollback automation boundary: `%s`", stringValue(nestedMap(latest, "rollback_trigger_surface")["automation_boundary"], "")),
		fmt.Sprintf("- Rollback trigger distinctions: `%s`", mapString(nestedMap(latest, "rollback_trigger_surface")["distinctions"])),
		"",
		"## Parity drift rollup",
		"",
		fmt.Sprintf("- Status: `%s`", stringValue(rollup["status"], "")),
		fmt.Sprintf("- Latest run: `%s`", stringValue(nestedMap(rollup, "summary")["latest_run_id"], "")),
		fmt.Sprintf("- Highest severity: `%s`", stringValue(nestedMap(rollup, "summary")["highest_severity"], "")),
		fmt.Sprintf("- Drift-detected runs in window: `%d`", intValue(nestedMap(rollup, "summary")["drift_detected_runs"])),
		fmt.Sprintf("- Stale runs in window: `%d`", intValue(nestedMap(rollup, "summary")["stale_runs"])),
		"",
		"## Workflow closeout commands",
		"",
	}
	for _, command := range stringSliceAny(latest["closeout_commands"]) {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	for _, item := range recentRuns {
		lines = append(lines, fmt.Sprintf(
			"- `%s` · `%s` · `%s` · `%s` · `%s`",
			stringValue(item["run_id"], ""),
			stringValue(item["status"], ""),
			stringValue(item["severity"], ""),
			stringValue(item["generated_at"], ""),
			stringValue(item["bundle_path"], ""),
		))
	}
	lines = append(lines, "", "## Linked migration docs", "")
	for _, entry := range docLinks {
		lines = append(lines, fmt.Sprintf("- `%s` %s", entry.Path, entry.Description))
	}
	lines = append(lines, "", "## Parallel Follow-up Index", "")
	lines = append(lines,
		"- `docs/reports/parallel-follow-up-index.md` is the canonical index for the",
		"  remaining live-shadow, rollback, and corpus-coverage follow-up digests behind",
		"  this run bundle.",
		"- For the two primary caveat tracks referenced by this bundle, see",
		"  `OPE-266` / `BIG-PAR-092` in",
		"  `docs/reports/live-shadow-comparison-follow-up-digest.md` and",
		"  `OPE-254` / `BIG-PAR-088` in",
		"  `docs/reports/rollback-safeguard-follow-up-digest.md`.",
		"",
	)
	return strings.Join(lines, "\n") + "\n"
}

func deriveRunID(scorecardReport map[string]any, generatedAt time.Time) string {
	latestEvidenceTimestamp := stringValue(nestedMap(scorecardReport, "summary")["latest_evidence_timestamp"], "")
	if latestEvidenceTimestamp != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.Replace(latestEvidenceTimestamp, "Z", "+00:00", 1))
		if err == nil {
			return parsed.UTC().Format("20060102T150405Z")
		}
	}
	return generatedAt.UTC().Format("20060102T150405Z")
}

func classifySeverity(scorecard map[string]any) string {
	summary := nestedMap(scorecard, "summary")
	if intValue(summary["stale_inputs"]) > 0 {
		return "high"
	}
	if intValue(summary["drift_detected_count"]) > 0 {
		return "medium"
	}
	for _, checkpoint := range mapSliceAt(scorecard, "cutover_checkpoints") {
		if !boolValue(checkpoint["passed"], false) {
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

func collectTraceIDs(matrixReport map[string]any) []string {
	ids := make([]string, 0)
	for _, item := range mapSliceAt(matrixReport, "results") {
		if id := stringValue(item["trace_id"], ""); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func pickLatestRunID(items []map[string]any) any {
	if len(items) == 0 {
		return nil
	}
	return items[0]["run_id"]
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func writeJSON(path string, payload map[string]any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func copyJSONArtifact(source, destination string) error {
	var payload map[string]any
	if err := readJSON(source, &payload); err != nil {
		return err
	}
	return writeJSON(destination, payload)
}

func copyTextArtifact(source, destination string) error {
	body, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, body, 0o644)
}

func relpath(path, root string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func utcISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapSliceAt(source map[string]any, key string) []map[string]any {
	return mapSliceAny(source[key])
}

func mapSliceAny(value any) []map[string]any {
	switch raw := value.(type) {
	case []map[string]any:
		return append([]map[string]any(nil), raw...)
	case []any:
		items := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if mapped, ok := item.(map[string]any); ok {
				items = append(items, mapped)
			}
		}
		return items
	default:
		return nil
	}
}

func stringSliceAny(value any) []string {
	switch raw := value.(type) {
	case []string:
		return append([]string(nil), raw...)
	case []any:
		items := make([]string, 0, len(raw))
		for _, item := range raw {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
		return items
	default:
		return nil
	}
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case bool:
		if typed {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if !ok {
		return fallback
	}
	return typed
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func mapString(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Sprintf("%v", value)
	}
	keys := make([]string, 0, len(payload))
	for _, key := range []string{"blockers", "warnings", "manual_only_paths"} {
		if _, ok := payload[key]; ok {
			keys = append(keys, key)
			delete(payload, key)
		}
	}
	remaining := make([]string, 0, len(payload))
	for key := range payload {
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	keys = append(keys, remaining...)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("'%s': %v", key, payload[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
