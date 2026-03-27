package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var automationExecutorLanes = []string{"local", "kubernetes", "ray"}

type automationValidationBundleScorecardOptions struct {
	RepoRoot              string
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	OutputPath            string
	Now                   func() time.Time
}

type automationValidationBundlePolicyGateOptions struct {
	RepoRoot                    string
	ScorecardPath               string
	OutputPath                  string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
	Now                         func() time.Time
}

func runAutomationValidationBundleScorecardCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	indexManifestPath := flags.String("index-manifest-path", "bigclaw-go/docs/reports/live-validation-index.json", "index manifest path")
	bundleRootPath := flags.String("bundle-root-path", "bigclaw-go/docs/reports/live-validation-runs", "bundle root path")
	summaryPath := flags.String("summary-path", "bigclaw-go/docs/reports/live-validation-summary.json", "latest summary path")
	sharedQueueReportPath := flags.String("shared-queue-report-path", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-scorecard [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, _, err := automationValidationBundleScorecard(automationValidationBundleScorecardOptions{
		RepoRoot:              absPath(*repoRoot),
		IndexManifestPath:     trim(*indexManifestPath),
		BundleRootPath:        trim(*bundleRootPath),
		SummaryPath:           trim(*summaryPath),
		SharedQueueReportPath: trim(*sharedQueueReportPath),
		OutputPath:            trim(*outputPath),
	})
	if err != nil {
		return err
	}
	if *pretty {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	return nil
}

func runAutomationValidationBundlePolicyGateCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e validation-bundle-policy-gate", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	scorecardPath := flags.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "maximum latest bundle age in hours")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum recent bundles required")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated coverage for every executor lane")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flags.String("enforcement-mode", "", "enforcement mode (review|hold|fail)")
	legacyEnforce := flags.Bool("enforce", false, "legacy flag that maps to fail mode")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e validation-bundle-policy-gate [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationValidationBundlePolicyGate(automationValidationBundlePolicyGateOptions{
		RepoRoot:                    absPath(*repoRoot),
		ScorecardPath:               trim(*scorecardPath),
		OutputPath:                  trim(*outputPath),
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: *requireRepeatedLaneCoverage && !*allowPartialLaneHistory,
		EnforcementMode:             trim(*enforcementMode),
		LegacyEnforce:               *legacyEnforce,
	})
	if err != nil {
		return err
	}
	if *pretty {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	}
	if exitCode != 0 {
		return exitError(exitCode)
	}
	return nil
}

func automationValidationBundleScorecard(opts automationValidationBundleScorecardOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	repoRoot := opts.RepoRoot
	bigclawGoRoot := filepath.Join(repoRoot, "bigclaw-go")
	manifest, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.IndexManifestPath)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load manifest from %s", opts.IndexManifestPath)
	}
	latest, _ := manifest["latest"].(map[string]any)
	recentRunsMeta, _ := manifest["recent_runs"].([]any)
	recentRuns := make([]map[string]any, 0, len(recentRunsMeta))
	recentRunInputs := make([]any, 0, len(recentRunsMeta))
	for _, item := range recentRunsMeta {
		entry, _ := item.(map[string]any)
		summaryFile := automationResolveEvidencePath(repoRoot, bigclawGoRoot, automationFirstText(entry["summary_path"]))
		runSummary, _ := automationReadJSON(summaryFile).(map[string]any)
		if runSummary != nil {
			recentRuns = append(recentRuns, runSummary)
			recentRunInputs = append(recentRunInputs, automationRelPath(summaryFile, repoRoot))
		}
	}
	latestSummary, _ := automationReadJSON(automationResolveRepoPath(repoRoot, opts.SummaryPath)).(map[string]any)
	sharedQueue, _ := automationReadJSON(automationResolveRepoPath(repoRoot, opts.SharedQueueReportPath)).(map[string]any)
	bundleRoot := automationResolveRepoPath(repoRoot, opts.BundleRootPath)
	bundledSharedQueue, _ := latestSummary["shared_queue_companion"].(map[string]any)

	laneScorecards := make([]any, 0, len(automationExecutorLanes))
	for _, lane := range automationExecutorLanes {
		laneScorecards = append(laneScorecards, automationBuildLaneScorecard(recentRuns, lane))
	}
	latestGeneratedAt, err := automationParseTime(automationFirstText(latest["generated_at"]))
	if err != nil {
		return nil, 0, err
	}
	var previousGeneratedAt *time.Time
	if len(recentRuns) > 1 {
		if parsed, err := automationParseTime(automationFirstText(recentRuns[1]["generated_at"])); err == nil {
			previousGeneratedAt = &parsed
		}
	}
	generatedAt := now().UTC()
	latestAgeHours := roundFloat(generatedAt.Sub(latestGeneratedAt).Hours(), 2)
	var bundleGapMinutes any
	if previousGeneratedAt != nil {
		bundleGapMinutes = roundFloat(latestGeneratedAt.Sub(*previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]any{}
	latestAllSucceeded := true
	for _, lane := range automationExecutorLanes {
		section, _ := latestSummary[lane].(map[string]any)
		status := automationFirstText(section["status"])
		latestLaneStatuses[lane] = status
		if status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if automationFirstText(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]any{}
	for _, item := range laneScorecards {
		scorecard, _ := item.(map[string]any)
		enabledRuns := int(scorecard["enabled_runs"].(int))
		enabledRunsByLane[automationFirstText(scorecard["lane"])] = enabledRuns
		if enabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	continuationChecks := []any{
		automationCheck("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		automationCheck("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		automationCheck("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", automationStatuses(recentRuns))),
		automationCheck("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		automationCheck("shared_queue_companion_proof_available", automationBoolOr(bundledSharedQueue["available"], sharedQueue["all_ok"]), fmt.Sprintf("cross_node_completions=%v", automationCoalesce(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]))),
		automationCheck("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	sharedQueueCompanion := map[string]any{
		"available":                 automationBoolOr(bundledSharedQueue["available"], sharedQueue["all_ok"]),
		"report_path":               automationFirstText(automationCoalesce(bundledSharedQueue["canonical_report_path"], opts.SharedQueueReportPath)),
		"summary_path":              automationFirstText(automationCoalesce(bundledSharedQueue["canonical_summary_path"], "bigclaw-go/docs/reports/shared-queue-companion-summary.json")),
		"bundle_report_path":        bundledSharedQueue["bundle_report_path"],
		"bundle_summary_path":       bundledSharedQueue["bundle_summary_path"],
		"cross_node_completions":    automationCoalesce(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"], 0),
		"duplicate_completed_tasks": automationCoalesce(bundledSharedQueue["duplicate_completed_tasks"], automationListLen(sharedQueue["duplicate_completed_tasks"])),
		"duplicate_started_tasks":   automationCoalesce(bundledSharedQueue["duplicate_started_tasks"], automationListLen(sharedQueue["duplicate_started_tasks"])),
		"mode": func() any {
			if len(bundledSharedQueue) > 0 {
				return "bundle-companion-summary"
			}
			return "standalone-proof"
		}(),
	}

	currentCeiling := []any{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	nextRuntimeHooks := []any{
		"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
		"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
		"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
		"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
	}

	report := map[string]any{
		"generated_at": automationUTCISO(generatedAt),
		"ticket":       "BIG-PAR-086-local-prework",
		"title":        "Validation bundle continuation scorecard",
		"status":       "local-continuation-scorecard",
		"evidence_inputs": map[string]any{
			"manifest_path":            opts.IndexManifestPath,
			"latest_summary_path":      opts.SummaryPath,
			"bundle_root":              opts.BundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": opts.SharedQueueReportPath,
			"generator_script":         "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		},
		"summary": map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     automationFirstText(latest["run_id"]),
			"latest_status":                                     automationFirstText(latest["status"]),
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                automationPathExists(bundleRoot),
		},
		"executor_lanes":         laneScorecards,
		"shared_queue_companion": sharedQueueCompanion,
		"continuation_checks":    continuationChecks,
		"current_ceiling":        currentCeiling,
		"next_runtime_hooks":     nextRuntimeHooks,
	}
	if err := automationWriteJSON(automationResolveRepoPath(repoRoot, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func automationValidationBundlePolicyGate(opts automationValidationBundlePolicyGateOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	scorecard, ok := automationReadJSON(automationResolveRepoPath(opts.RepoRoot, opts.ScorecardPath)).(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("failed to load scorecard from %s", opts.ScorecardPath)
	}
	summary, _ := scorecard["summary"].(map[string]any)
	sharedQueue, _ := scorecard["shared_queue_companion"].(map[string]any)
	mode, err := automationNormalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforce)
	if err != nil {
		return nil, 0, err
	}

	checks := []any{
		automationCheck("latest_bundle_age_within_threshold", automationFloat64(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], opts.MaxLatestAgeHours)),
		automationCheck("recent_bundle_count_meets_floor", automationInt(summary["recent_bundle_count"]) >= opts.MinRecentBundles, fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], opts.MinRecentBundles)),
		automationCheck("latest_bundle_all_executor_tracks_succeeded", automationBool(summary["latest_all_executor_tracks_succeeded"]), fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"])),
		automationCheck("recent_bundle_chain_has_no_failures", automationBool(summary["recent_bundle_chain_has_no_failures"]), fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"])),
		automationCheck("shared_queue_companion_available", automationBool(sharedQueue["available"]), fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"])),
		automationCheck("repeated_lane_coverage_meets_policy", !opts.RequireRepeatedLaneCoverage || automationBool(summary["all_executor_tracks_have_repeated_recent_coverage"]), fmt.Sprintf("require_repeated_lane_coverage=%v actual=%v", opts.RequireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"])),
	}
	failingChecks := []any{}
	passingCount := 0
	for _, item := range checks {
		check, _ := item.(map[string]any)
		if automationBool(check["passed"]) {
			passingCount++
		} else {
			failingChecks = append(failingChecks, check["name"])
		}
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := automationBuildEnforcementSummary(recommendation, mode)
	nextActions := []any{}
	if automationContainsString(failingChecks, "latest_bundle_age_within_threshold") {
		nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
	}
	if automationContainsString(failingChecks, "recent_bundle_count_meets_floor") {
		nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
	}
	if automationContainsString(failingChecks, "shared_queue_companion_available") {
		nextActions = append(nextActions, "rerun `python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
	}
	if automationContainsString(failingChecks, "repeated_lane_coverage_meets_policy") {
		nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}

	report := map[string]any{
		"generated_at": automationUTCISO(now().UTC()),
		"ticket":       "OPE-262",
		"title":        "Validation workflow continuation gate",
		"status": func() any {
			if recommendation == "go" {
				return "policy-go"
			}
			return "policy-hold"
		}(),
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":   opts.ScorecardPath,
			"generator_script": "scripts/e2e/validation_bundle_continuation_policy_gate.py",
		},
		"policy_inputs": map[string]any{
			"max_latest_age_hours":           opts.MaxLatestAgeHours,
			"min_recent_bundles":             opts.MinRecentBundles,
			"require_repeated_lane_coverage": opts.RequireRepeatedLaneCoverage,
		},
		"enforcement": enforcement,
		"summary": map[string]any{
			"latest_run_id":                                     summary["latest_run_id"],
			"latest_bundle_age_hours":                           summary["latest_bundle_age_hours"],
			"recent_bundle_count":                               summary["recent_bundle_count"],
			"latest_all_executor_tracks_succeeded":              summary["latest_all_executor_tracks_succeeded"],
			"recent_bundle_chain_has_no_failures":               summary["recent_bundle_chain_has_no_failures"],
			"all_executor_tracks_have_repeated_recent_coverage": summary["all_executor_tracks_have_repeated_recent_coverage"],
			"recommendation":                                    recommendation,
			"enforcement_mode":                                  enforcement["mode"],
			"workflow_outcome":                                  enforcement["outcome"],
			"workflow_exit_code":                                enforcement["exit_code"],
			"passing_check_count":                               passingCount,
			"failing_check_count":                               len(failingChecks),
		},
		"policy_checks":          checks,
		"failing_checks":         failingChecks,
		"reviewer_path":          map[string]any{"index_path": "docs/reports/live-validation-index.md", "digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		"shared_queue_companion": sharedQueue,
		"next_actions":           nextActions,
	}
	if err := automationWriteJSON(automationResolveRepoPath(opts.RepoRoot, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	return report, automationInt(enforcement["exit_code"]), nil
}

func automationResolveRepoPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func automationResolveEvidencePath(repoRoot string, bigclawGoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	searchRoots := []string{repoRoot}
	if !stringsHasPrefixBigclawGo(path) {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, path)
		if automationPathExists(resolved) {
			return resolved
		}
	}
	return filepath.Join(searchRoots[0], path)
}

func stringsHasPrefixBigclawGo(path string) bool {
	clean := filepath.ToSlash(path)
	return clean == "bigclaw-go" || len(clean) > len("bigclaw-go/") && clean[:len("bigclaw-go/")] == "bigclaw-go/"
}

func automationPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func automationUTCISO(moment time.Time) string {
	return strings.ReplaceAll(moment.UTC().Format(time.RFC3339), "+00:00", "Z")
}

func automationParseTime(value string) (time.Time, error) {
	if trim(value) == "" {
		return time.Time{}, errors.New("empty time value")
	}
	replaced := strings.ReplaceAll(value, "Z", "+00:00")
	return time.Parse(time.RFC3339, replaced)
}

func automationBuildLaneScorecard(runs []map[string]any, lane string) map[string]any {
	statuses := []any{}
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section, _ := run[lane].(map[string]any)
		enabled := automationBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = automationFirstText(section["status"])
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		if !enabled && section != nil {
			status = "disabled"
		}
		if section == nil {
			status = "disabled"
		}
		statuses = append(statuses, status)
	}
	var latest map[string]any
	if len(runs) > 0 {
		latest, _ = runs[0][lane].(map[string]any)
	}
	return map[string]any{
		"lane":           lane,
		"latest_enabled": automationBool(latest["enabled"]),
		"latest_status": func() any {
			if latest != nil {
				return automationFirstText(latest["status"])
			}
			return "missing"
		}(),
		"recent_statuses":           statuses,
		"enabled_runs":              enabledRuns,
		"succeeded_runs":            succeededRuns,
		"consecutive_successes":     automationConsecutiveSuccesses(statuses),
		"all_recent_runs_succeeded": enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func automationConsecutiveSuccesses(statuses []any) int {
	count := 0
	for _, status := range statuses {
		if automationFirstText(status) == "succeeded" {
			count++
		} else {
			break
		}
	}
	return count
}

func automationCheck(name string, passed bool, detail string) map[string]any {
	return map[string]any{"name": name, "passed": passed, "detail": detail}
}

func automationStatuses(runs []map[string]any) []any {
	items := make([]any, 0, len(runs))
	for _, run := range runs {
		items = append(items, automationFirstText(run["status"]))
	}
	return items
}

func automationBoolOr(values ...any) bool {
	for _, value := range values {
		if automationBool(value) {
			return true
		}
	}
	return false
}

func automationCoalesce(values ...any) any {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			if trim(typed) != "" {
				return typed
			}
		default:
			return value
		}
	}
	return nil
}

func automationBool(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func automationFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func automationInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func automationNormalizeEnforcementMode(mode string, legacyEnforce bool) (string, error) {
	normalized := trim(mode)
	if normalized == "" {
		if legacyEnforce {
			normalized = "fail"
		} else {
			normalized = "hold"
		}
	}
	switch normalized {
	case "review", "hold", "fail":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", mode)
	}
}

func automationBuildEnforcementSummary(recommendation string, enforcementMode string) map[string]any {
	if recommendation == "go" {
		return map[string]any{"mode": enforcementMode, "outcome": "pass", "exit_code": 0}
	}
	switch enforcementMode {
	case "review":
		return map[string]any{"mode": enforcementMode, "outcome": "review-only", "exit_code": 0}
	case "hold":
		return map[string]any{"mode": enforcementMode, "outcome": "hold", "exit_code": 2}
	default:
		return map[string]any{"mode": enforcementMode, "outcome": "fail", "exit_code": 1}
	}
}

func automationContainsString(values []any, target string) bool {
	for _, value := range values {
		if automationFirstText(value) == target {
			return true
		}
	}
	return false
}

func roundFloat(value float64, precision int) float64 {
	pow := 1.0
	for i := 0; i < precision; i++ {
		pow *= 10
	}
	if value >= 0 {
		return float64(int(value*pow+0.5)) / pow
	}
	return float64(int(value*pow-0.5)) / pow
}
