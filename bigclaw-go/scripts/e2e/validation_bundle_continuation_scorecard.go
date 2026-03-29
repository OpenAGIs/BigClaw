package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const continuationScorecardScriptPath = "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.go"

var continuationExecutorLanes = []string{"local", "kubernetes", "ray"}

type continuationLaneScorecard struct {
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

type continuationScorecardReport struct {
	GeneratedAt          string                      `json:"generated_at"`
	Ticket               string                      `json:"ticket"`
	Title                string                      `json:"title"`
	Status               string                      `json:"status"`
	EvidenceInputs       map[string]any              `json:"evidence_inputs"`
	Summary              map[string]any              `json:"summary"`
	ExecutorLanes        []continuationLaneScorecard `json:"executor_lanes"`
	SharedQueueCompanion map[string]any              `json:"shared_queue_companion"`
	ContinuationChecks   []continuationCheck         `json:"continuation_checks"`
	CurrentCeiling       []string                    `json:"current_ceiling"`
	NextRuntimeHooks     []string                    `json:"next_runtime_hooks"`
}

type continuationScorecardOptions struct {
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	GeneratedAt           time.Time
}

func main() {
	outputPath := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := repoRootFromScorecardScript(scriptFilePathForScorecard())
	if err != nil {
		panic(err)
	}
	report, err := buildContinuationScorecardReport(repoRoot, continuationScorecardOptions{
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		GeneratedAt:           time.Now().UTC(),
	})
	if err != nil {
		panic(err)
	}
	if err := writeContinuationScorecardReport(repoRoot, *outputPath, report); err != nil {
		panic(err)
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(body))
	}
}

func buildContinuationScorecardReport(repoRoot string, opts continuationScorecardOptions) (continuationScorecardReport, error) {
	manifestPath := resolveScorecardRepoPath(repoRoot, opts.IndexManifestPath)
	summaryPath := resolveScorecardRepoPath(repoRoot, opts.SummaryPath)
	sharedQueuePath := resolveScorecardRepoPath(repoRoot, opts.SharedQueueReportPath)
	bundleRootPath := resolveScorecardRepoPath(repoRoot, opts.BundleRootPath)
	bigclawGORoot := filepath.Join(repoRoot, "bigclaw-go")

	var manifest map[string]any
	if err := readJSONMap(manifestPath, &manifest); err != nil {
		return continuationScorecardReport{}, err
	}
	latest, _ := manifest["latest"].(map[string]any)
	if latest == nil {
		return continuationScorecardReport{}, errors.New("manifest.latest missing or not an object")
	}

	recentRunsMeta := asMapSlice(manifest["recent_runs"])
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]string, 0, len(recentRunsMeta))
	for _, item := range recentRunsMeta {
		summaryFile := resolveEvidencePath(repoRoot, bigclawGORoot, asString(item["summary_path"]))
		var run map[string]any
		if err := readJSONMap(summaryFile, &run); err != nil {
			return continuationScorecardReport{}, err
		}
		recentRuns = append(recentRuns, run)
		relPath, err := filepath.Rel(repoRoot, summaryFile)
		if err != nil {
			return continuationScorecardReport{}, err
		}
		recentRunInputs = append(recentRunInputs, filepath.ToSlash(relPath))
	}

	var latestSummary map[string]any
	if err := readJSONMap(summaryPath, &latestSummary); err != nil {
		return continuationScorecardReport{}, err
	}
	var sharedQueue map[string]any
	if err := readJSONMap(sharedQueuePath, &sharedQueue); err != nil {
		return continuationScorecardReport{}, err
	}

	var bundledSharedQueue map[string]any
	if value, ok := latestSummary["shared_queue_companion"].(map[string]any); ok {
		bundledSharedQueue = value
	}

	laneScorecards := make([]continuationLaneScorecard, 0, len(continuationExecutorLanes))
	for _, lane := range continuationExecutorLanes {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, err := parseScorecardTime(asString(latest["generated_at"]))
	if err != nil {
		return continuationScorecardReport{}, err
	}
	var previousGeneratedAt *time.Time
	if len(recentRuns) > 1 {
		parsed, err := parseScorecardTime(asString(recentRuns[1]["generated_at"]))
		if err != nil {
			return continuationScorecardReport{}, err
		}
		previousGeneratedAt = &parsed
	}
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	latestAgeHours := roundFloat(generatedAt.Sub(latestGeneratedAt).Hours(), 2)
	var bundleGapMinutes any
	if previousGeneratedAt != nil {
		bundleGapMinutes = roundFloat(latestGeneratedAt.Sub(*previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]any{}
	for _, lane := range continuationExecutorLanes {
		latestLaneStatuses[lane] = asString(asMap(latestSummary[lane])["status"])
	}
	latestAllSucceeded := true
	for _, lane := range continuationExecutorLanes {
		if asString(latestLaneStatuses[lane]) != "succeeded" {
			latestAllSucceeded = false
			break
		}
	}

	recentAllSucceeded := true
	for _, run := range recentRuns {
		if asString(run["status"]) != "succeeded" {
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
		buildContinuationCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			latestAllSucceeded,
			fmt.Sprintf("latest lane statuses=%s", pythonRepr(latestLaneStatuses)),
		),
		buildContinuationCheck(
			"recent_bundle_chain_has_multiple_runs",
			len(recentRuns) >= 2,
			fmt.Sprintf("recent bundle count=%d", len(recentRuns)),
		),
		buildContinuationCheck(
			"recent_bundle_chain_has_no_failures",
			recentAllSucceeded,
			fmt.Sprintf("recent bundle statuses=%s", pythonRepr(extractStatuses(recentRuns))),
		),
		buildContinuationCheck(
			"all_executor_tracks_have_repeated_recent_coverage",
			repeatedLaneCoverage,
			fmt.Sprintf("enabled_runs_by_lane=%s", pythonRepr(enabledRunsByLane)),
		),
		buildContinuationCheck(
			"shared_queue_companion_proof_available",
			asBoolWithDefault(bundledSharedQueue["available"], asBool(sharedQueue["all_ok"])),
			fmt.Sprintf("cross_node_completions=%s", pythonRepr(firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]))),
		),
		buildContinuationCheck(
			"continuation_surface_is_workflow_triggered",
			true,
			"run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service",
		),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 asBoolWithDefault(bundledSharedQueue["available"], asBool(sharedQueue["all_ok"])),
		"report_path":               firstNonNil(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReportPath),
		"summary_path":              firstNonNil(bundledSharedQueue["canonical_summary_path"], "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    firstNonNil(bundledSharedQueue["cross_node_completions"], firstNonNil(sharedQueue["cross_node_completions"], 0)),
		"duplicate_completed_tasks": firstNonNil(bundledSharedQueue["duplicate_completed_tasks"], len(asSlice(sharedQueue["duplicate_completed_tasks"]))),
		"duplicate_started_tasks":   firstNonNil(bundledSharedQueue["duplicate_started_tasks"], len(asSlice(sharedQueue["duplicate_started_tasks"]))),
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

	report := continuationScorecardReport{
		GeneratedAt: utcISOScorecard(generatedAt),
		Ticket:      "BIG-PAR-086-local-prework",
		Title:       "Validation bundle continuation scorecard",
		Status:      "local-continuation-scorecard",
		EvidenceInputs: map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReportPath,
			"generator_script":         continuationScorecardScriptPath,
		},
		Summary: map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     asString(latest["run_id"]),
			"latest_status":                                     asString(latest["status"]),
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                pathExists(bundleRootPath),
		},
		ExecutorLanes:        laneScorecards,
		SharedQueueCompanion: sharedQueueCompanion,
		ContinuationChecks:   continuationChecks,
		CurrentCeiling:       currentCeiling,
		NextRuntimeHooks: []string{
			"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
			"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
			"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
			"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
		},
	}

	return report, nil
}

func writeContinuationScorecardReport(repoRoot, outputPath string, report continuationScorecardReport) error {
	targetPath := resolveScorecardRepoPath(repoRoot, outputPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, append(body, '\n'), 0o644)
}

func buildLaneScorecard(runs []map[string]any, lane string) continuationLaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := asMap(run[lane])
		enabled := asBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = withDefaultString(section["status"], "missing")
		}
		statuses = append(statuses, status)
		if enabled {
			enabledRuns++
		}
		if status == "succeeded" {
			succeededRuns++
		}
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = asMap(runs[0][lane])
	}
	return continuationLaneScorecard{
		Lane:                   lane,
		LatestEnabled:          asBool(latest["enabled"]),
		LatestStatus:           withDefaultString(latest["status"], "missing"),
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

func buildContinuationCheck(name string, passed bool, detail string) continuationCheck {
	return continuationCheck{Name: name, Passed: passed, Detail: detail}
}

func resolveScorecardRepoPath(repoRoot, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(repoRoot, target)
}

func resolveEvidencePath(repoRoot, bigclawGORoot, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	candidate := filepath.Clean(target)
	searchRoots := []string{repoRoot}
	if !strings.HasPrefix(filepath.ToSlash(candidate), "bigclaw-go/") {
		searchRoots = append(searchRoots, bigclawGORoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, candidate)
		if pathExists(resolved) {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], candidate)
}

func readJSONMap(path string, target *map[string]any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func parseScorecardTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp %q", value)
}

func utcISOScorecard(moment time.Time) string {
	if moment.IsZero() {
		moment = time.Now().UTC()
	}
	return moment.UTC().Format(time.RFC3339Nano)
}

func repoRootFromScorecardScript(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty script path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), "../../..")), nil
}

func scriptFilePathForScorecard() string {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return path
}

func extractStatuses(runs []map[string]any) []any {
	statuses := make([]any, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, withDefaultString(run["status"], "unknown"))
	}
	return statuses
}

func pythonRepr(value any) string {
	switch cast := value.(type) {
	case nil:
		return "None"
	case string:
		return fmt.Sprintf("'%s'", cast)
	case bool:
		if cast {
			return "True"
		}
		return "False"
	case int:
		return fmt.Sprintf("%d", cast)
	case int64:
		return fmt.Sprintf("%d", cast)
	case float64:
		return formatPyFloat(cast)
	case float32:
		return formatPyFloat(float64(cast))
	case []string:
		values := make([]string, 0, len(cast))
		for _, item := range cast {
			values = append(values, pythonRepr(item))
		}
		return "[" + strings.Join(values, ", ") + "]"
	case []any:
		values := make([]string, 0, len(cast))
		for _, item := range cast {
			values = append(values, pythonRepr(item))
		}
		return "[" + strings.Join(values, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(cast))
		for key := range cast {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		values := make([]string, 0, len(keys))
		for _, key := range keys {
			values = append(values, fmt.Sprintf("%s: %s", pythonRepr(key), pythonRepr(cast[key])))
		}
		return "{" + strings.Join(values, ", ") + "}"
	default:
		return fmt.Sprintf("%v", value)
	}
}

func formatPyFloat(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.1f", value)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.12f", value), "0"), ".")
}

func asMap(value any) map[string]any {
	if cast, ok := value.(map[string]any); ok {
		return cast
	}
	return map[string]any{}
}

func asMapSlice(value any) []map[string]any {
	raw := asSlice(value)
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		items = append(items, asMap(item))
	}
	return items
}

func asSlice(value any) []any {
	if cast, ok := value.([]any); ok {
		return cast
	}
	return nil
}

func asString(value any) string {
	if cast, ok := value.(string); ok {
		return cast
	}
	return ""
}

func withDefaultString(value any, fallback string) string {
	if cast := asString(value); cast != "" {
		return cast
	}
	return fallback
}

func asBool(value any) bool {
	if cast, ok := value.(bool); ok {
		return cast
	}
	return false
}

func asBoolWithDefault(value any, fallback bool) bool {
	if _, ok := value.(bool); ok {
		return asBool(value)
	}
	return fallback
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func roundFloat(value float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	return float64(int(value*pow+0.5)) / pow
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
