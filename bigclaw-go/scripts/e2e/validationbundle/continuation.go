package validationbundle

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

type SharedQueueCompanion struct {
	Available               bool   `json:"available"`
	ReportPath              string `json:"report_path"`
	SummaryPath             string `json:"summary_path"`
	BundleReportPath        string `json:"bundle_report_path,omitempty"`
	BundleSummaryPath       string `json:"bundle_summary_path,omitempty"`
	CrossNodeCompletions    int    `json:"cross_node_completions"`
	DuplicateCompletedTasks int    `json:"duplicate_completed_tasks"`
	DuplicateStartedTasks   int    `json:"duplicate_started_tasks"`
	Mode                    string `json:"mode"`
}

type ScorecardEvidenceInputs struct {
	ManifestPath          string   `json:"manifest_path"`
	LatestSummaryPath     string   `json:"latest_summary_path"`
	BundleRoot            string   `json:"bundle_root"`
	RecentRunSummaries    []string `json:"recent_run_summaries"`
	SharedQueueReportPath string   `json:"shared_queue_report_path"`
	GeneratorScript       string   `json:"generator_script"`
}

type ScorecardSummary struct {
	RecentBundleCount                           int     `json:"recent_bundle_count"`
	LatestRunID                                 string  `json:"latest_run_id"`
	LatestStatus                                string  `json:"latest_status"`
	LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours"`
	LatestAllExecutorTracksSucceeded            bool    `json:"latest_all_executor_tracks_succeeded"`
	RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
	AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
	BundleGapMinutes                            float64 `json:"bundle_gap_minutes,omitempty"`
	BundleRootExists                            bool    `json:"bundle_root_exists"`
}

type ScorecardReport struct {
	GeneratedAt          string                  `json:"generated_at"`
	Ticket               string                  `json:"ticket"`
	Title                string                  `json:"title"`
	Status               string                  `json:"status"`
	EvidenceInputs       ScorecardEvidenceInputs `json:"evidence_inputs"`
	Summary              ScorecardSummary        `json:"summary"`
	ExecutorLanes        []LaneScorecard         `json:"executor_lanes"`
	SharedQueueCompanion SharedQueueCompanion    `json:"shared_queue_companion"`
	ContinuationChecks   []Check                 `json:"continuation_checks"`
	CurrentCeiling       []string                `json:"current_ceiling"`
	NextRuntimeHooks     []string                `json:"next_runtime_hooks"`
}

type GateEvidenceInputs struct {
	ScorecardPath   string `json:"scorecard_path"`
	GeneratorScript string `json:"generator_script"`
}

type GatePolicyInputs struct {
	MaxLatestAgeHours           float64 `json:"max_latest_age_hours"`
	MinRecentBundles            int     `json:"min_recent_bundles"`
	RequireRepeatedLaneCoverage bool    `json:"require_repeated_lane_coverage"`
}

type GateEnforcement struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type GateSummary struct {
	LatestRunID                                 string  `json:"latest_run_id"`
	LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours"`
	RecentBundleCount                           int     `json:"recent_bundle_count"`
	LatestAllExecutorTracksSucceeded            bool    `json:"latest_all_executor_tracks_succeeded"`
	RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
	AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
	Recommendation                              string  `json:"recommendation"`
	EnforcementMode                             string  `json:"enforcement_mode"`
	WorkflowOutcome                             string  `json:"workflow_outcome"`
	WorkflowExitCode                            int     `json:"workflow_exit_code"`
	PassingCheckCount                           int     `json:"passing_check_count"`
	FailingCheckCount                           int     `json:"failing_check_count"`
}

type DigestIssue struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type ReviewerPath struct {
	IndexPath   string      `json:"index_path"`
	DigestPath  string      `json:"digest_path"`
	DigestIssue DigestIssue `json:"digest_issue"`
}

type GateReport struct {
	GeneratedAt          string               `json:"generated_at"`
	Ticket               string               `json:"ticket"`
	Title                string               `json:"title"`
	Status               string               `json:"status"`
	Recommendation       string               `json:"recommendation"`
	EvidenceInputs       GateEvidenceInputs   `json:"evidence_inputs"`
	PolicyInputs         GatePolicyInputs     `json:"policy_inputs"`
	Enforcement          GateEnforcement      `json:"enforcement"`
	Summary              GateSummary          `json:"summary"`
	PolicyChecks         []Check              `json:"policy_checks"`
	FailingChecks        []string             `json:"failing_checks"`
	ReviewerPath         ReviewerPath         `json:"reviewer_path"`
	SharedQueueCompanion SharedQueueCompanion `json:"shared_queue_companion"`
	NextActions          []string             `json:"next_actions"`
}

type scorecardOptions struct {
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
	Now                   time.Time
}

type gateOptions struct {
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
	Now                         time.Time
}

func BuildScorecard(repoRoot string, now time.Time) (ScorecardReport, error) {
	return buildScorecard(repoRoot, scorecardOptions{
		IndexManifestPath:     "bigclaw-go/docs/reports/live-validation-index.json",
		BundleRootPath:        "bigclaw-go/docs/reports/live-validation-runs",
		SummaryPath:           "bigclaw-go/docs/reports/live-validation-summary.json",
		SharedQueueReportPath: "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		Now:                   now.UTC(),
	})
}

func BuildGate(repoRoot, scorecardPath string, maxLatestAgeHours float64, minRecentBundles int, requireRepeatedLaneCoverage bool, enforcementMode string, legacyEnforce bool, now time.Time) (GateReport, error) {
	return buildGate(repoRoot, gateOptions{
		ScorecardPath:               scorecardPath,
		MaxLatestAgeHours:           maxLatestAgeHours,
		MinRecentBundles:            minRecentBundles,
		RequireRepeatedLaneCoverage: requireRepeatedLaneCoverage,
		EnforcementMode:             enforcementMode,
		LegacyEnforce:               legacyEnforce,
		Now:                         now.UTC(),
	})
}

func WriteJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func resolveEvidencePath(repoRoot, bigclawGoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	searchRoots := []string{repoRoot}
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, "bigclaw-go"+string(filepath.Separator)) && cleaned != "bigclaw-go" {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		candidate := filepath.Join(root, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join(searchRoots[0], path)
}

func readJSON(path string, dst any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func utcISO(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
}

func asMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func asSlice(value any) []any {
	result, _ := value.([]any)
	return result
}

func asString(value any) string {
	typed, _ := value.(string)
	return typed
}

func asBool(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func relativeToRepo(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
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

func buildLaneScorecard(runs []map[string]any, lane string) LaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := asMap(run[lane])
		enabled := asBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = firstNonEmpty(asString(section["status"]), "missing")
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = asMap(runs[0][lane])
	}
	return LaneScorecard{
		Lane:                   lane,
		LatestEnabled:          asBool(latest["enabled"]),
		LatestStatus:           firstNonEmpty(asString(latest["status"]), "missing"),
		RecentStatuses:         statuses,
		EnabledRuns:            enabledRuns,
		SucceededRuns:          succeededRuns,
		ConsecutiveSuccesses:   consecutiveSuccesses(statuses),
		AllRecentRunsSucceeded: enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func buildCheck(name string, passed bool, detail string) Check {
	return Check{Name: name, Passed: passed, Detail: detail}
}

func buildScorecard(repoRoot string, options scorecardOptions) (ScorecardReport, error) {
	bigclawGoRoot := filepath.Join(repoRoot, "bigclaw-go")
	var manifest map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, options.IndexManifestPath), &manifest); err != nil {
		return ScorecardReport{}, err
	}
	latest := asMap(manifest["latest"])
	recentRunMeta := asSlice(manifest["recent_runs"])
	recentRuns := make([]map[string]any, 0, len(recentRunMeta))
	recentRunInputs := make([]string, 0, len(recentRunMeta))
	for _, item := range recentRunMeta {
		summaryPath := resolveEvidencePath(repoRoot, bigclawGoRoot, asString(asMap(item)["summary_path"]))
		var summary map[string]any
		if err := readJSON(summaryPath, &summary); err != nil {
			return ScorecardReport{}, err
		}
		recentRuns = append(recentRuns, summary)
		recentRunInputs = append(recentRunInputs, relativeToRepo(repoRoot, summaryPath))
	}

	var latestSummary map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, options.SummaryPath), &latestSummary); err != nil {
		return ScorecardReport{}, err
	}
	var sharedQueue map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, options.SharedQueueReportPath), &sharedQueue); err != nil {
		return ScorecardReport{}, err
	}
	bundledSharedQueue := asMap(latestSummary["shared_queue_companion"])
	laneScorecards := make([]LaneScorecard, 0, len(executorLanes))
	for _, lane := range executorLanes {
		laneScorecards = append(laneScorecards, buildLaneScorecard(recentRuns, lane))
	}

	latestGeneratedAt, err := parseTime(asString(latest["generated_at"]))
	if err != nil {
		return ScorecardReport{}, err
	}
	var bundleGapMinutes float64
	if len(recentRuns) > 1 {
		previousGeneratedAt, err := parseTime(asString(recentRuns[1]["generated_at"]))
		if err != nil {
			return ScorecardReport{}, err
		}
		bundleGapMinutes = roundFloat(latestGeneratedAt.Sub(previousGeneratedAt).Minutes(), 2)
	}

	latestLaneStatuses := map[string]string{}
	latestAllSucceeded := true
	for _, lane := range executorLanes {
		latestLaneStatuses[lane] = asString(asMap(latestSummary[lane])["status"])
		if latestLaneStatuses[lane] != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	enabledRunsByLane := map[string]int{}
	repeatedLaneCoverage := true
	for _, item := range laneScorecards {
		enabledRunsByLane[item.Lane] = item.EnabledRuns
		if item.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}
	for _, run := range recentRuns {
		if asString(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	sharedQueueAvailable := firstBool(bundledSharedQueue["available"], sharedQueue["all_ok"])
	sharedQueueCompanion := SharedQueueCompanion{
		Available:               sharedQueueAvailable,
		ReportPath:              firstNonEmpty(asString(bundledSharedQueue["canonical_report_path"]), options.SharedQueueReportPath),
		SummaryPath:             firstNonEmpty(asString(bundledSharedQueue["canonical_summary_path"]), "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		BundleReportPath:        asString(bundledSharedQueue["bundle_report_path"]),
		BundleSummaryPath:       asString(bundledSharedQueue["bundle_summary_path"]),
		CrossNodeCompletions:    firstInt(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"]),
		DuplicateCompletedTasks: firstInt(bundledSharedQueue["duplicate_completed_tasks"], len(asSlice(sharedQueue["duplicate_completed_tasks"]))),
		DuplicateStartedTasks:   firstInt(bundledSharedQueue["duplicate_started_tasks"], len(asSlice(sharedQueue["duplicate_started_tasks"]))),
		Mode:                    ternaryString(len(bundledSharedQueue) > 0, "bundle-companion-summary", "standalone-proof"),
	}

	continuationChecks := []Check{
		buildCheck("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		buildCheck("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		buildCheck("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", extractStatuses(recentRuns))),
		buildCheck("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		buildCheck("shared_queue_companion_proof_available", sharedQueueCompanion.Available, fmt.Sprintf("cross_node_completions=%d", sharedQueueCompanion.CrossNodeCompletions)),
		buildCheck("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}

	return ScorecardReport{
		GeneratedAt: utcISO(options.Now),
		Ticket:      "BIG-PAR-086-local-prework",
		Title:       "Validation bundle continuation scorecard",
		Status:      "local-continuation-scorecard",
		EvidenceInputs: ScorecardEvidenceInputs{
			ManifestPath:          options.IndexManifestPath,
			LatestSummaryPath:     options.SummaryPath,
			BundleRoot:            options.BundleRootPath,
			RecentRunSummaries:    recentRunInputs,
			SharedQueueReportPath: options.SharedQueueReportPath,
			GeneratorScript:       "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard/main.go",
		},
		Summary: ScorecardSummary{
			RecentBundleCount:                           len(recentRuns),
			LatestRunID:                                 asString(latest["run_id"]),
			LatestStatus:                                asString(latest["status"]),
			LatestBundleAgeHours:                        roundFloat(options.Now.Sub(latestGeneratedAt).Hours(), 2),
			LatestAllExecutorTracksSucceeded:            latestAllSucceeded,
			RecentBundleChainHasNoFailures:              recentAllSucceeded,
			AllExecutorTracksHaveRepeatedRecentCoverage: repeatedLaneCoverage,
			BundleGapMinutes:                            bundleGapMinutes,
			BundleRootExists:                            dirExists(resolveRepoPath(repoRoot, options.BundleRootPath)),
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
	}, nil
}

func buildGate(repoRoot string, options gateOptions) (GateReport, error) {
	var scorecard struct {
		Summary              ScorecardSummary     `json:"summary"`
		SharedQueueCompanion SharedQueueCompanion `json:"shared_queue_companion"`
	}
	if err := readJSON(resolveRepoPath(repoRoot, options.ScorecardPath), &scorecard); err != nil {
		return GateReport{}, err
	}

	mode, err := normalizeEnforcementMode(options.EnforcementMode, options.LegacyEnforce)
	if err != nil {
		return GateReport{}, err
	}
	checks := []Check{
		buildCheck("latest_bundle_age_within_threshold", scorecard.Summary.LatestBundleAgeHours <= options.MaxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", scorecard.Summary.LatestBundleAgeHours, options.MaxLatestAgeHours)),
		buildCheck("recent_bundle_count_meets_floor", scorecard.Summary.RecentBundleCount >= options.MinRecentBundles, fmt.Sprintf("recent_bundle_count=%d floor=%d", scorecard.Summary.RecentBundleCount, options.MinRecentBundles)),
		buildCheck("latest_bundle_all_executor_tracks_succeeded", scorecard.Summary.LatestAllExecutorTracksSucceeded, fmt.Sprintf("latest_all_executor_tracks_succeeded=%t", scorecard.Summary.LatestAllExecutorTracksSucceeded)),
		buildCheck("recent_bundle_chain_has_no_failures", scorecard.Summary.RecentBundleChainHasNoFailures, fmt.Sprintf("recent_bundle_chain_has_no_failures=%t", scorecard.Summary.RecentBundleChainHasNoFailures)),
		buildCheck("shared_queue_companion_available", scorecard.SharedQueueCompanion.Available, fmt.Sprintf("cross_node_completions=%d", scorecard.SharedQueueCompanion.CrossNodeCompletions)),
		buildCheck("repeated_lane_coverage_meets_policy", !options.RequireRepeatedLaneCoverage || scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage, fmt.Sprintf("require_repeated_lane_coverage=%t actual=%t", options.RequireRepeatedLaneCoverage, scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage)),
	}

	failingChecks := []string{}
	for _, item := range checks {
		if !item.Passed {
			failingChecks = append(failingChecks, item.Name)
		}
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildEnforcementSummary(recommendation, mode)
	nextActions := []string{}
	if contains(failingChecks, "latest_bundle_age_within_threshold") {
		nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
	}
	if contains(failingChecks, "recent_bundle_count_meets_floor") {
		nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
	}
	if contains(failingChecks, "shared_queue_companion_available") {
		nextActions = append(nextActions, "rerun `python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
	}
	if contains(failingChecks, "repeated_lane_coverage_meets_policy") {
		nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}

	return GateReport{
		GeneratedAt:    utcISO(options.Now),
		Ticket:         "OPE-262",
		Title:          "Validation workflow continuation gate",
		Status:         ternaryString(recommendation == "go", "policy-go", "policy-hold"),
		Recommendation: recommendation,
		EvidenceInputs: GateEvidenceInputs{
			ScorecardPath:   options.ScorecardPath,
			GeneratorScript: "bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate/main.go",
		},
		PolicyInputs: GatePolicyInputs{
			MaxLatestAgeHours:           options.MaxLatestAgeHours,
			MinRecentBundles:            options.MinRecentBundles,
			RequireRepeatedLaneCoverage: options.RequireRepeatedLaneCoverage,
		},
		Enforcement: enforcement,
		Summary: GateSummary{
			LatestRunID:                                 scorecard.Summary.LatestRunID,
			LatestBundleAgeHours:                        scorecard.Summary.LatestBundleAgeHours,
			RecentBundleCount:                           scorecard.Summary.RecentBundleCount,
			LatestAllExecutorTracksSucceeded:            scorecard.Summary.LatestAllExecutorTracksSucceeded,
			RecentBundleChainHasNoFailures:              scorecard.Summary.RecentBundleChainHasNoFailures,
			AllExecutorTracksHaveRepeatedRecentCoverage: scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage,
			Recommendation:                              recommendation,
			EnforcementMode:                             enforcement.Mode,
			WorkflowOutcome:                             enforcement.Outcome,
			WorkflowExitCode:                            enforcement.ExitCode,
			PassingCheckCount:                           len(checks) - len(failingChecks),
			FailingCheckCount:                           len(failingChecks),
		},
		PolicyChecks:  checks,
		FailingChecks: failingChecks,
		ReviewerPath: ReviewerPath{
			IndexPath:  "docs/reports/live-validation-index.md",
			DigestPath: "docs/reports/validation-bundle-continuation-digest.md",
			DigestIssue: DigestIssue{
				ID:   "OPE-271",
				Slug: "BIG-PAR-082",
			},
		},
		SharedQueueCompanion: scorecard.SharedQueueCompanion,
		NextActions:          nextActions,
	}, nil
}

func normalizeEnforcementMode(enforcementMode string, legacyEnforce bool) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(enforcementMode))
	if mode == "" {
		if legacyEnforce {
			mode = "fail"
		} else {
			mode = "hold"
		}
	}
	switch mode {
	case "review", "hold", "fail":
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", enforcementMode)
	}
}

func buildEnforcementSummary(recommendation, enforcementMode string) GateEnforcement {
	if recommendation == "go" {
		return GateEnforcement{Mode: enforcementMode, Outcome: "pass", ExitCode: 0}
	}
	switch enforcementMode {
	case "review":
		return GateEnforcement{Mode: enforcementMode, Outcome: "review-only", ExitCode: 0}
	case "hold":
		return GateEnforcement{Mode: enforcementMode, Outcome: "hold", ExitCode: 2}
	default:
		return GateEnforcement{Mode: enforcementMode, Outcome: "fail", ExitCode: 1}
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
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

func extractStatuses(runs []map[string]any) []string {
	statuses := make([]string, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, firstNonEmpty(asString(run["status"]), "unknown"))
	}
	return statuses
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstBool(values ...any) bool {
	for _, value := range values {
		if typed, ok := value.(bool); ok {
			return typed
		}
	}
	return false
}

func firstInt(values ...any) int {
	for _, value := range values {
		switch typed := value.(type) {
		case int:
			return typed
		case float64:
			return int(typed)
		}
	}
	return 0
}

func ternaryString(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
