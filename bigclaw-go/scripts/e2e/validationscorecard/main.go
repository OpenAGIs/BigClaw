package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var executorLanes = []string{"local", "kubernetes", "ray"}

type laneScorecard struct {
	Lane                   string   `json:"lane"`
	LatestEnabled          bool     `json:"latest_enabled"`
	LatestStatus           string   `json:"latest_status"`
	RecentStatuses         []string `json:"recent_statuses"`
	EnabledRuns            int      `json:"enabled_runs"`
	SucceededRuns          int      `json:"succeeded_runs"`
	ConsecutiveSuccesses   int      `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool     `json:"all_recent_runs_succeeded"`
}

type continuationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type report struct {
	GeneratedAt          string              `json:"generated_at"`
	Ticket               string              `json:"ticket"`
	Title                string              `json:"title"`
	Status               string              `json:"status"`
	EvidenceInputs       map[string]any      `json:"evidence_inputs"`
	Summary              map[string]any      `json:"summary"`
	ExecutorLanes        []laneScorecard     `json:"executor_lanes"`
	SharedQueueCompanion map[string]any      `json:"shared_queue_companion"`
	ContinuationChecks   []continuationCheck `json:"continuation_checks"`
	CurrentCeiling       []string            `json:"current_ceiling"`
	NextRuntimeHooks     []string            `json:"next_runtime_hooks"`
}

type buildOptions struct {
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
}

func main() {
	goRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	flags := flag.NewFlagSet("validation-bundle-continuation-scorecard", flag.ExitOnError)
	indexManifestPath := flags.String("index-manifest", "bigclaw-go/docs/reports/live-validation-index.json", "live validation index manifest")
	bundleRootPath := flags.String("bundle-root", "bigclaw-go/docs/reports/live-validation-runs", "bundle root path")
	summaryPath := flags.String("summary", "bigclaw-go/docs/reports/live-validation-summary.json", "latest summary path")
	sharedQueueReportPath := flags.String("shared-queue-report", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "json output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := buildReport(repoRoot, buildOptions{
		IndexManifestPath:     *indexManifestPath,
		BundleRootPath:        *bundleRootPath,
		SummaryPath:           *summaryPath,
		SharedQueueReportPath: *sharedQueueReportPath,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resolvedOutputPath := resolveRepoPath(repoRoot, *outputPath)
	if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(resolvedOutputPath, append(body, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		fmt.Println(string(body))
	}
}

func buildReport(repoRoot string, opts buildOptions) (report, error) {
	bigclawGoRoot := filepath.Join(repoRoot, "bigclaw-go")

	var manifest map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, opts.IndexManifestPath), &manifest); err != nil {
		return report{}, err
	}
	latest := nestedMap(manifest, "latest")
	recentRunMetas := nestedSlice(manifest, "recent_runs")

	recentRuns := make([]map[string]any, 0, len(recentRunMetas))
	recentRunInputs := make([]string, 0, len(recentRunMetas))
	for _, item := range recentRunMetas {
		summaryFile := resolveEvidencePath(repoRoot, bigclawGoRoot, stringValue(item["summary_path"]))
		var summary map[string]any
		if err := readJSON(summaryFile, &summary); err != nil {
			return report{}, err
		}
		recentRuns = append(recentRuns, summary)
		relative, err := filepath.Rel(repoRoot, summaryFile)
		if err != nil {
			return report{}, err
		}
		recentRunInputs = append(recentRunInputs, filepath.ToSlash(relative))
	}

	var latestSummary map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, opts.SummaryPath), &latestSummary); err != nil {
		return report{}, err
	}
	var sharedQueue map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, opts.SharedQueueReportPath), &sharedQueue); err != nil {
		return report{}, err
	}
	bundleRoot := resolveRepoPath(repoRoot, opts.BundleRootPath)
	bundledSharedQueue := nestedMap(latestSummary, "shared_queue_companion")

	laneScorecards := make([]laneScorecard, 0, len(executorLanes))
	for _, lane := range executorLanes {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, err := parseTime(stringValue(latest["generated_at"]))
	if err != nil {
		return report{}, err
	}
	var previousGeneratedAt time.Time
	hasPrevious := len(recentRuns) > 1
	if hasPrevious {
		previousGeneratedAt, err = parseTime(stringValue(recentRuns[1]["generated_at"]))
		if err != nil {
			return report{}, err
		}
	}

	generatedAt := time.Now().UTC()
	latestAgeHours := round2(generatedAt.Sub(latestGeneratedAt).Hours())
	var bundleGapMinutes any
	if hasPrevious {
		bundleGapMinutes = round2(latestGeneratedAt.Sub(previousGeneratedAt).Minutes())
	} else {
		bundleGapMinutes = nil
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range executorLanes {
		status := stringValue(nestedMap(latestSummary, lane)["status"])
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}

	recentAllSucceeded := true
	for _, run := range recentRuns {
		if stringValueDefault(run["status"], "unknown") != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]any{}
	for _, item := range laneScorecards {
		enabledRunsByLane[item.Lane] = item.EnabledRuns
		if item.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	continuationChecks := []continuationCheck{
		buildCheck("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		buildCheck("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		buildCheck("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", recentStatuses(recentRuns))),
		buildCheck("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		buildCheck("shared_queue_companion_proof_available", boolValueWithDefault(bundledSharedQueue["available"], boolValueWithDefault(sharedQueue["all_ok"], false)), fmt.Sprintf("cross_node_completions=%v", firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]))),
		buildCheck("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 boolValueWithDefault(bundledSharedQueue["available"], boolValueWithDefault(sharedQueue["all_ok"], false)),
		"report_path":               stringValueDefault(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReportPath),
		"summary_path":              stringValueDefault(bundledSharedQueue["canonical_summary_path"], "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    intValue(firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"])),
		"duplicate_completed_tasks": intValue(firstNonNil(bundledSharedQueue["duplicate_completed_tasks"], listLen(sharedQueue["duplicate_completed_tasks"]))),
		"duplicate_started_tasks":   intValue(firstNonNil(bundledSharedQueue["duplicate_started_tasks"], listLen(sharedQueue["duplicate_started_tasks"]))),
		"mode":                      map[bool]string{true: "bundle-companion-summary", false: "standalone-proof"}[len(bundledSharedQueue) > 0],
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

	return report{
		GeneratedAt: utcISO(generatedAt),
		Ticket:      "BIG-PAR-086-local-prework",
		Title:       "Validation bundle continuation scorecard",
		Status:      "local-continuation-scorecard",
		EvidenceInputs: map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReportPath,
			"generator_script":         "bigclaw-go/scripts/e2e/validation-bundle-continuation-scorecard",
		},
		Summary: map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     latest["run_id"],
			"latest_status":                                     latest["status"],
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                pathExists(bundleRoot),
		},
		ExecutorLanes:        laneScorecards,
		SharedQueueCompanion: sharedQueueCompanion,
		ContinuationChecks:   continuationChecks,
		CurrentCeiling:       currentCeiling,
		NextRuntimeHooks:     nextRuntimeHooks,
	}, nil
}

func buildLaneScorecard(runs []map[string]any, lane string) laneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := nestedMap(run, lane)
		enabled := boolValueWithDefault(section["enabled"], false)
		status := "disabled"
		if enabled {
			status = stringValueDefault(section["status"], "missing")
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = nestedMap(runs[0], lane)
	}
	return laneScorecard{
		Lane:                   lane,
		LatestEnabled:          boolValueWithDefault(latest["enabled"], false),
		LatestStatus:           stringValueDefault(latest["status"], "missing"),
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
		if status != "succeeded" {
			break
		}
		count++
	}
	return count
}

func buildCheck(name string, passed bool, detail string) continuationCheck {
	return continuationCheck{Name: name, Passed: passed, Detail: detail}
}

func nestedMap(input map[string]any, key string) map[string]any {
	value, ok := input[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func nestedSlice(input map[string]any, key string) []map[string]any {
	values, ok := input[key].([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]any)
		if ok {
			result = append(result, item)
		}
	}
	return result
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

func utcISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(repoRoot, path)
}

func resolveEvidencePath(repoRoot, bigclawGoRoot, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	searchRoots := []string{repoRoot}
	if firstPathPart(path) != "bigclaw-go" {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, path)
		if pathExists(resolved) {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], path)
}

func firstPathPart(path string) string {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if idx := strings.IndexByte(cleaned, '/'); idx >= 0 {
		return cleaned[:idx]
	}
	return cleaned
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func recentStatuses(runs []map[string]any) []string {
	values := make([]string, 0, len(runs))
	for _, run := range runs {
		values = append(values, stringValueDefault(run["status"], "unknown"))
	}
	return values
}

func round2(value float64) float64 {
	if value >= 0 {
		return float64(int(value*100+0.5)) / 100
	}
	return float64(int(value*100-0.5)) / 100
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func stringValueDefault(value any, fallback string) string {
	if text := stringValue(value); text != "" {
		return text
	}
	return fallback
}

func boolValueWithDefault(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if ok {
		return typed
	}
	return fallback
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func listLen(value any) int {
	items, ok := value.([]any)
	if ok {
		return len(items)
	}
	return 0
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
