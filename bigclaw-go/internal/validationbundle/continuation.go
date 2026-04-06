package validationbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func BuildContinuationScorecard(goRoot string, indexManifestPath string, bundleRootPath string, summaryPath string, sharedQueueReport string, outputPath string, generatedAt time.Time) (map[string]any, error) {
	root := filepath.Clean(goRoot)
	manifest, err := readJSONMap(resolvePath(root, indexManifestPath))
	if err != nil {
		return nil, err
	}
	latest, _ := manifest["latest"].(map[string]any)
	recentRunsMeta, _ := manifest["recent_runs"].([]any)
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]any, 0, len(recentRunsMeta))
	for _, rawItem := range recentRunsMeta {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		summaryFile := resolveEvidencePath(root, item["summary_path"])
		runSummary, err := readJSONMap(summaryFile)
		if err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, runSummary)
		recentRunInputs = append(recentRunInputs, relPath(root, summaryFile))
	}
	latestSummary, err := readJSONMap(resolvePath(root, summaryPath))
	if err != nil {
		return nil, err
	}
	sharedQueue, err := readJSONMap(resolvePath(root, sharedQueueReport))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if sharedQueue == nil {
		sharedQueue = map[string]any{}
	}
	bundleRoot := resolvePath(root, bundleRootPath)
	bundledSharedQueue, _ := latestSummary["shared_queue_companion"].(map[string]any)

	laneScorecards := make([]any, 0, 3)
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, _ := parseTime(stringValue(latest["generated_at"]))
	var previousGeneratedAt time.Time
	if len(recentRuns) > 1 {
		previousGeneratedAt, _ = parseTime(stringValue(recentRuns[1]["generated_at"]))
	}
	currentTime := generatedAt.UTC()
	latestAgeHours := 0.0
	if !latestGeneratedAt.IsZero() {
		latestAgeHours = round(currentTime.Sub(latestGeneratedAt).Hours(), 2)
	}
	var bundleGapMinutes any
	if !previousGeneratedAt.IsZero() && !latestGeneratedAt.IsZero() {
		bundleGapMinutes = round(latestGeneratedAt.Sub(previousGeneratedAt).Minutes(), 2)
	}

	latestAllSucceeded := true
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section, _ := latestSummary[lane].(map[string]any)
		if stringValue(section["status"]) != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if stringValue(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]int{}
	for _, raw := range laneScorecards {
		item, _ := raw.(map[string]any)
		enabledRuns := intValue(item["enabled_runs"])
		enabledRunsByLane[stringValue(item["lane"])] = enabledRuns
		if enabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	continuationChecks := []any{
		check("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses={'local': '%s', 'kubernetes': '%s', 'ray': '%s'}", laneStatus(latestSummary, "local"), laneStatus(latestSummary, "kubernetes"), laneStatus(latestSummary, "ray"))),
		check("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		check("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%s", quoteStrings(statuses(recentRuns)))),
		check("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane={'local': %d, 'kubernetes': %d, 'ray': %d}", enabledRunsByLane["local"], enabledRunsByLane["kubernetes"], enabledRunsByLane["ray"])),
		check("shared_queue_companion_proof_available", boolValue(bundledSharedQueue["available"]) || boolValue(sharedQueue["all_ok"]), fmt.Sprintf("cross_node_completions=%v", firstNonZero(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]))),
		check("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 boolValue(bundledSharedQueue["available"]) || boolValue(sharedQueue["all_ok"]),
		"report_path":               firstText(bundledSharedQueue["canonical_report_path"], sharedQueueReport),
		"summary_path":              firstText(bundledSharedQueue["canonical_summary_path"], "docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    firstNonZero(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]),
		"duplicate_completed_tasks": firstNonZero(bundledSharedQueue["duplicate_completed_tasks"], lenSlice(sharedQueue["duplicate_completed_tasks"])),
		"duplicate_started_tasks":   firstNonZero(bundledSharedQueue["duplicate_started_tasks"], lenSlice(sharedQueue["duplicate_started_tasks"])),
		"mode":                      ternary(len(bundledSharedQueue) > 0, "bundle-companion-summary", "standalone-proof"),
	}

	currentCeiling := []any{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}

	report := map[string]any{
		"generated_at": utcISO(currentTime),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            indexManifestPath,
			"latest_summary_path":      summaryPath,
			"bundle_root":              bundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": sharedQueueReport,
			"generator_script":         "cd bigclaw-go && go run ./cmd/bigclawctl live-validation continuation-scorecard",
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     latest["run_id"],
			"latest_status":                                     latest["status"],
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                fileExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks": []any{
			"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
			"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
			"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
			"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
		},
	}
	return report, writeJSON(resolvePath(root, outputPath), report)
}

func buildLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := []any{}
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section, _ := run[lane].(map[string]any)
		enabled := boolValue(section["enabled"])
		status := "disabled"
		if enabled {
			status = stringValue(section["status"])
			enabledRuns++
		}
		if status == "succeeded" {
			succeededRuns++
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest, _ = runs[0][lane].(map[string]any)
	}
	return map[string]any{
		"lane":                      lane,
		"latest_enabled":            boolValue(latest["enabled"]),
		"latest_status":             ternary(latest != nil, stringValue(latest["status"]), "missing"),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     consecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func consecutiveSuccesses(statuses []any) int {
	count := 0
	for _, raw := range statuses {
		if stringValue(raw) == "succeeded" {
			count++
			continue
		}
		break
	}
	return count
}

func check(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func readJSONMap(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
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

func resolvePath(root string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func resolveEvidencePath(root string, path any) string {
	value := stringValue(path)
	if filepath.IsAbs(value) {
		return value
	}
	candidates := []string{resolvePath(root, value)}
	if !strings.HasPrefix(value, "bigclaw-go/") {
		candidates = append(candidates, resolvePath(root, filepath.ToSlash(filepath.Join("bigclaw-go", value))))
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return candidates[0]
}

func relPath(root string, path string) string {
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

func round(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}

func firstText(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(stringValue(value))
		if text != "" {
			return text
		}
	}
	return ""
}

func firstNonZero(values ...any) any {
	for _, value := range values {
		switch item := value.(type) {
		case float64:
			if item != 0 {
				return int(item)
			}
		case int:
			if item != 0 {
				return item
			}
		}
	}
	return 0
}

func lenSlice(value any) int {
	items, _ := value.([]any)
	return len(items)
}

func statuses(runs []map[string]any) []string {
	items := make([]string, 0, len(runs))
	for _, run := range runs {
		items = append(items, stringValue(run["status"]))
	}
	return items
}

func quoteStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("'%s'", value))
	}
	return fmt.Sprintf("[%s]", strings.Join(quoted, ", "))
}

func laneStatus(summary map[string]any, lane string) string {
	section, _ := summary[lane].(map[string]any)
	return stringValue(section["status"])
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func intValue(value any) int {
	switch item := value.(type) {
	case float64:
		return int(item)
	case int:
		return item
	default:
		return 0
	}
}

func boolValue(value any) bool {
	item, _ := value.(bool)
	return item
}

func ternary(condition bool, yes string, no string) string {
	if condition {
		return yes
	}
	return no
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
