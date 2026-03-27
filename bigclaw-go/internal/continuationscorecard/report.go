package continuationscorecard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var executorLanes = []string{"local", "kubernetes", "ray"}

type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type LaneScorecard struct {
	Lane                   string   `json:"lane"`
	LatestEnabled          bool     `json:"latest_enabled"`
	LatestStatus           string   `json:"latest_status"`
	RecentStatuses         []string `json:"recent_statuses"`
	EnabledRuns            int      `json:"enabled_runs"`
	SucceededRuns          int      `json:"succeeded_runs"`
	ConsecutiveSuccesses   int      `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool     `json:"all_recent_runs_succeeded"`
}

type BuildOptions struct {
	RepoRoot              string
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	Now                   time.Time
}

type manifestPayload struct {
	Latest struct {
		RunID       string `json:"run_id"`
		Status      string `json:"status"`
		GeneratedAt string `json:"generated_at"`
	} `json:"latest"`
	RecentRuns []struct {
		SummaryPath string `json:"summary_path"`
	} `json:"recent_runs"`
}

func BuildReport(opts BuildOptions) (map[string]any, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		repoRoot = "."
	}
	indexManifestPath := defaultString(opts.IndexManifestPath, "bigclaw-go/docs/reports/live-validation-index.json")
	bundleRootPath := defaultString(opts.BundleRootPath, "bigclaw-go/docs/reports/live-validation-runs")
	summaryPath := defaultString(opts.SummaryPath, "bigclaw-go/docs/reports/live-validation-summary.json")
	sharedQueueReportPath := defaultString(opts.SharedQueueReportPath, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json")
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var manifest manifestPayload
	if err := readJSON(resolveRepoPath(repoRoot, indexManifestPath), &manifest); err != nil {
		return nil, err
	}
	recentRuns := make([]map[string]any, 0, len(manifest.RecentRuns))
	recentRunInputs := make([]string, 0, len(manifest.RecentRuns))
	for _, item := range manifest.RecentRuns {
		var summary map[string]any
		summaryFile := resolveEvidencePath(repoRoot, filepath.Join(repoRoot, "bigclaw-go"), item.SummaryPath)
		if err := readJSON(summaryFile, &summary); err != nil {
			return nil, err
		}
		recentRuns = append(recentRuns, summary)
		rel, err := filepath.Rel(repoRoot, summaryFile)
		if err != nil {
			rel = summaryFile
		}
		recentRunInputs = append(recentRunInputs, filepath.ToSlash(rel))
	}
	var latestSummary map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, summaryPath), &latestSummary); err != nil {
		return nil, err
	}
	var sharedQueue map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, sharedQueueReportPath), &sharedQueue); err != nil {
		return nil, err
	}
	bundleRoot := resolveRepoPath(repoRoot, bundleRootPath)
	bundledSharedQueue := nestedMap(latestSummary, "shared_queue_companion")

	laneScorecards := make([]LaneScorecard, 0, len(executorLanes))
	for _, lane := range executorLanes {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, err := parseTime(manifest.Latest.GeneratedAt)
	if err != nil {
		return nil, err
	}
	var previousGeneratedAt *time.Time
	if len(recentRuns) > 1 {
		if generatedAt, ok := stringAt(recentRuns[1], "generated_at"); ok {
			parsed, err := parseTime(generatedAt)
			if err != nil {
				return nil, err
			}
			previousGeneratedAt = &parsed
		}
	}
	latestAgeHours := roundFloat(now.Sub(latestGeneratedAt).Hours(), 2)
	var bundleGapMinutes any
	if previousGeneratedAt != nil {
		bundleGapMinutes = roundFloat(latestGeneratedAt.Sub(*previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range executorLanes {
		status := stringValue(nestedMap(latestSummary, lane)["status"], "missing")
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if stringValue(run["status"], "unknown") != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]int{}
	for _, item := range laneScorecards {
		enabledRunsByLane[item.Lane] = item.EnabledRuns
		if item.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	continuationChecks := []Check{
		check("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		check("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		check("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", collectStatuses(recentRuns))),
		check("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		check("shared_queue_companion_proof_available", boolValue(bundledSharedQueue["available"], boolValue(sharedQueue["all_ok"], false)), fmt.Sprintf("cross_node_completions=%v", firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"], 0))),
		check("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 boolValue(bundledSharedQueue["available"], boolValue(sharedQueue["all_ok"], false)),
		"report_path":               firstNonNil(bundledSharedQueue["canonical_report_path"], sharedQueueReportPath),
		"summary_path":              firstNonNil(bundledSharedQueue["canonical_summary_path"], "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"], 0),
		"duplicate_completed_tasks": firstNonNil(bundledSharedQueue["duplicate_completed_tasks"], len(sliceAt(sharedQueue, "duplicate_completed_tasks"))),
		"duplicate_started_tasks":   firstNonNil(bundledSharedQueue["duplicate_started_tasks"], len(sliceAt(sharedQueue, "duplicate_started_tasks"))),
		"mode": map[bool]string{
			true:  "bundle-companion-summary",
			false: "standalone-proof",
		}[len(bundledSharedQueue) > 0],
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	nextRuntimeHooks := []string{
		"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
		"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
		"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
		"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
	}

	report := map[string]any{
		"generated_at": now.Format(time.RFC3339),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            indexManifestPath,
			"latest_summary_path":      summaryPath,
			"bundle_root":              bundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": sharedQueueReportPath,
			"generator_script":         "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     manifest.Latest.RunID,
			"latest_status":                                     manifest.Latest.Status,
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                pathExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks":     nextRuntimeHooks,
	}
	return report, nil
}

func WriteReport(path string, report map[string]any, pretty bool) error {
	var body []byte
	var err error
	if pretty {
		body, err = json.MarshalIndent(report, "", "  ")
	} else {
		body, err = json.Marshal(report)
	}
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func buildLaneScorecard(runs []map[string]any, lane string) LaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := nestedMap(run, lane)
		enabled := boolValue(section["enabled"], false)
		status := "disabled"
		if enabled {
			status = stringValue(section["status"], "missing")
			enabledRuns++
		}
		statuses = append(statuses, status)
		if status == "succeeded" {
			succeededRuns++
		}
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = nestedMap(runs[0], lane)
	}
	latestStatus := "missing"
	if len(latest) > 0 {
		latestStatus = stringValue(latest["status"], "missing")
	}
	return LaneScorecard{
		Lane:                   lane,
		LatestEnabled:          boolValue(latest["enabled"], false),
		LatestStatus:           latestStatus,
		RecentStatuses:         statuses,
		EnabledRuns:            enabledRuns,
		SucceededRuns:          succeededRuns,
		ConsecutiveSuccesses:   consecutiveSuccesses(statuses),
		AllRecentRunsSucceeded: enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func consecutiveSuccesses(statuses []string) int {
	count := 0
	for _, status := range statuses {
		if status == "succeeded" {
			count++
			continue
		}
		break
	}
	return count
}

func check(name string, passed bool, detail string) Check {
	return Check{Name: name, Passed: passed, Detail: detail}
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
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

func resolveEvidencePath(repoRoot, goRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	searchRoots := []string{repoRoot}
	if firstPathPart(path) != "bigclaw-go" {
		searchRoots = append(searchRoots, goRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, path)
		if pathExists(resolved) {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], path)
}

func collectStatuses(runs []map[string]any) []string {
	statuses := make([]string, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, stringValue(run["status"], "unknown"))
	}
	return statuses
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}

func stringAt(source map[string]any, key string) (string, bool) {
	text, ok := source[key].(string)
	return text, ok
}

func boolValue(value any, fallback bool) bool {
	flag, ok := value.(bool)
	if !ok {
		return fallback
	}
	return flag
}

func sliceAt(source map[string]any, key string) []any {
	items, _ := source[key].([]any)
	return items
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func roundFloat(value float64, places int) float64 {
	scale := 1.0
	for i := 0; i < places; i++ {
		scale *= 10
	}
	return float64(int(value*scale+0.5)) / scale
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstPathPart(path string) string {
	clean := filepath.ToSlash(path)
	for _, part := range strings.Split(clean, "/") {
		if part != "" {
			return part
		}
	}
	return ""
}
