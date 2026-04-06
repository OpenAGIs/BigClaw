package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	errUnsupportedEnforcementMode = errors.New("unsupported enforcement mode")
)

const (
	ValidationBundleContinuationScorecardGenerator  = "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard/main.go"
	ValidationBundleContinuationPolicyGateGenerator = "bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate/main.go"
)

var validationExecutorLanes = []string{"local", "kubernetes", "ray"}

type ContinuationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type ContinuationLaneScorecard struct {
	Lane                   string   `json:"lane"`
	LatestEnabled          bool     `json:"latest_enabled"`
	LatestStatus           string   `json:"latest_status"`
	RecentStatuses         []string `json:"recent_statuses"`
	EnabledRuns            int      `json:"enabled_runs"`
	SucceededRuns          int      `json:"succeeded_runs"`
	ConsecutiveSuccesses   int      `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool     `json:"all_recent_runs_succeeded"`
}

type ContinuationSharedQueueCompanion struct {
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

type ContinuationScorecardEvidenceInputs struct {
	ManifestPath          string   `json:"manifest_path"`
	LatestSummaryPath     string   `json:"latest_summary_path"`
	BundleRoot            string   `json:"bundle_root"`
	RecentRunSummaries    []string `json:"recent_run_summaries"`
	SharedQueueReportPath string   `json:"shared_queue_report_path"`
	GeneratorScript       string   `json:"generator_script"`
}

type ContinuationScorecardSummary struct {
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

type ContinuationScorecardReport struct {
	GeneratedAt          string                              `json:"generated_at"`
	Ticket               string                              `json:"ticket"`
	Title                string                              `json:"title"`
	Status               string                              `json:"status"`
	EvidenceInputs       ContinuationScorecardEvidenceInputs `json:"evidence_inputs"`
	Summary              ContinuationScorecardSummary        `json:"summary"`
	ExecutorLanes        []ContinuationLaneScorecard         `json:"executor_lanes"`
	SharedQueueCompanion ContinuationSharedQueueCompanion    `json:"shared_queue_companion"`
	ContinuationChecks   []ContinuationCheck                 `json:"continuation_checks"`
	CurrentCeiling       []string                            `json:"current_ceiling"`
	NextRuntimeHooks     []string                            `json:"next_runtime_hooks"`
}

type ContinuationPolicyEvidenceInputs struct {
	ScorecardPath   string `json:"scorecard_path"`
	GeneratorScript string `json:"generator_script"`
}

type ContinuationPolicyInputs struct {
	MaxLatestAgeHours           float64 `json:"max_latest_age_hours"`
	MinRecentBundles            int     `json:"min_recent_bundles"`
	RequireRepeatedLaneCoverage bool    `json:"require_repeated_lane_coverage"`
}

type ContinuationEnforcement struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type ContinuationPolicySummary struct {
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

type ContinuationReviewerIssue struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type ContinuationReviewerPath struct {
	IndexPath   string                    `json:"index_path"`
	DigestPath  string                    `json:"digest_path"`
	DigestIssue ContinuationReviewerIssue `json:"digest_issue"`
}

type ContinuationPolicyGateReport struct {
	GeneratedAt          string                           `json:"generated_at"`
	Ticket               string                           `json:"ticket"`
	Title                string                           `json:"title"`
	Status               string                           `json:"status"`
	Recommendation       string                           `json:"recommendation"`
	EvidenceInputs       ContinuationPolicyEvidenceInputs `json:"evidence_inputs"`
	PolicyInputs         ContinuationPolicyInputs         `json:"policy_inputs"`
	Enforcement          ContinuationEnforcement          `json:"enforcement"`
	Summary              ContinuationPolicySummary        `json:"summary"`
	PolicyChecks         []ContinuationCheck              `json:"policy_checks"`
	FailingChecks        []string                         `json:"failing_checks"`
	ReviewerPath         ContinuationReviewerPath         `json:"reviewer_path"`
	SharedQueueCompanion ContinuationSharedQueueCompanion `json:"shared_queue_companion"`
	NextActions          []string                         `json:"next_actions"`
}

type ContinuationScorecardOptions struct {
	ManifestPath          string
	BundleRootPath        string
	LatestSummaryPath     string
	SharedQueueReportPath string
	Now                   time.Time
}

type ContinuationPolicyGateOptions struct {
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
	Now                         time.Time
}

func BuildValidationBundleContinuationScorecard(repoRoot string, options ContinuationScorecardOptions) (ContinuationScorecardReport, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ContinuationScorecardReport{}, errors.New("repo root is required")
	}
	if options.ManifestPath == "" {
		options.ManifestPath = "bigclaw-go/docs/reports/live-validation-index.json"
	}
	if options.BundleRootPath == "" {
		options.BundleRootPath = "bigclaw-go/docs/reports/live-validation-runs"
	}
	if options.LatestSummaryPath == "" {
		options.LatestSummaryPath = "bigclaw-go/docs/reports/live-validation-summary.json"
	}
	if options.SharedQueueReportPath == "" {
		options.SharedQueueReportPath = "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var manifest struct {
		Latest struct {
			RunID       string `json:"run_id"`
			Status      string `json:"status"`
			GeneratedAt string `json:"generated_at"`
		} `json:"latest"`
		RecentRuns []struct {
			SummaryPath string `json:"summary_path"`
		} `json:"recent_runs"`
	}
	if err := loadJSON(resolveReportPath(repoRoot, options.ManifestPath), &manifest); err != nil {
		return ContinuationScorecardReport{}, err
	}

	bigclawGoRoot := filepath.Join(repoRoot, "bigclaw-go")
	recentRuns := make([]map[string]any, 0, len(manifest.RecentRuns))
	recentInputs := make([]string, 0, len(manifest.RecentRuns))
	for _, item := range manifest.RecentRuns {
		summaryPath := resolveEvidencePath(repoRoot, bigclawGoRoot, item.SummaryPath)
		var payload map[string]any
		if err := loadJSON(summaryPath, &payload); err != nil {
			return ContinuationScorecardReport{}, err
		}
		recentRuns = append(recentRuns, payload)
		rel, err := filepath.Rel(repoRoot, summaryPath)
		if err != nil {
			rel = summaryPath
		}
		recentInputs = append(recentInputs, filepath.ToSlash(rel))
	}

	var latestSummary map[string]any
	if err := loadJSON(resolveReportPath(repoRoot, options.LatestSummaryPath), &latestSummary); err != nil {
		return ContinuationScorecardReport{}, err
	}
	var sharedQueue map[string]any
	if err := loadJSON(resolveReportPath(repoRoot, options.SharedQueueReportPath), &sharedQueue); err != nil {
		return ContinuationScorecardReport{}, err
	}

	laneScorecards := make([]ContinuationLaneScorecard, 0, len(validationExecutorLanes))
	enabledRunsByLane := make(map[string]int, len(validationExecutorLanes))
	latestLaneStatuses := make(map[string]string, len(validationExecutorLanes))
	repeatedLaneCoverage := true
	for _, lane := range validationExecutorLanes {
		laneScorecard := buildContinuationLaneScorecard(recentRuns, lane)
		laneScorecards = append(laneScorecards, laneScorecard)
		enabledRunsByLane[lane] = laneScorecard.EnabledRuns
		latestLaneStatuses[lane] = laneScorecard.LatestStatus
		if laneScorecard.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	latestGeneratedAt, err := parseFlexibleTime(manifest.Latest.GeneratedAt)
	if err != nil {
		return ContinuationScorecardReport{}, err
	}
	latestAgeHours := roundToTwoDecimals(now.Sub(latestGeneratedAt).Hours())
	bundleGapMinutes := 0.0
	hasBundleGap := false
	if len(recentRuns) > 1 {
		if previousGeneratedAt, err := parseFlexibleTime(asString(recentRuns[1]["generated_at"])); err == nil {
			bundleGapMinutes = roundToTwoDecimals(latestGeneratedAt.Sub(previousGeneratedAt).Minutes())
			hasBundleGap = true
		}
	}

	latestAllSucceeded := true
	for _, lane := range validationExecutorLanes {
		if status := asString(latestSectionValue(latestSummary, lane, "status")); status != "succeeded" {
			latestAllSucceeded = false
		}
	}
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if asString(run["status"]) != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	bundledSharedQueue := asMap(latestSummary["shared_queue_companion"])
	sharedQueueAvailable := asBoolWithFallback(bundledSharedQueue["available"], asBool(sharedQueue["all_ok"]))
	sharedQueueCompanion := ContinuationSharedQueueCompanion{
		Available:               sharedQueueAvailable,
		ReportPath:              firstNonEmptyString(asString(bundledSharedQueue["canonical_report_path"]), options.SharedQueueReportPath),
		SummaryPath:             firstNonEmptyString(asString(bundledSharedQueue["canonical_summary_path"]), "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		BundleReportPath:        asString(bundledSharedQueue["bundle_report_path"]),
		BundleSummaryPath:       asString(bundledSharedQueue["bundle_summary_path"]),
		CrossNodeCompletions:    asIntWithFallback(bundledSharedQueue["cross_node_completions"], asInt(sharedQueue["cross_node_completions"])),
		DuplicateCompletedTasks: asIntWithFallback(bundledSharedQueue["duplicate_completed_tasks"], len(asSlice(sharedQueue["duplicate_completed_tasks"]))),
		DuplicateStartedTasks:   asIntWithFallback(bundledSharedQueue["duplicate_started_tasks"], len(asSlice(sharedQueue["duplicate_started_tasks"]))),
		Mode:                    firstNonEmptyString(asString(bundledSharedQueue["mode"]), map[bool]string{true: "bundle-companion-summary", false: "standalone-proof"}[len(bundledSharedQueue) > 0]),
	}

	checks := []ContinuationCheck{
		{Name: "latest_bundle_all_executor_tracks_succeeded", Passed: latestAllSucceeded, Detail: fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)},
		{Name: "recent_bundle_chain_has_multiple_runs", Passed: len(recentRuns) >= 2, Detail: fmt.Sprintf("recent bundle count=%d", len(recentRuns))},
		{Name: "recent_bundle_chain_has_no_failures", Passed: recentAllSucceeded, Detail: fmt.Sprintf("recent bundle statuses=%v", collectStatuses(recentRuns))},
		{Name: "all_executor_tracks_have_repeated_recent_coverage", Passed: repeatedLaneCoverage, Detail: fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)},
		{Name: "shared_queue_companion_proof_available", Passed: sharedQueueCompanion.Available, Detail: fmt.Sprintf("cross_node_completions=%d", sharedQueueCompanion.CrossNodeCompletions)},
		{Name: "continuation_surface_is_workflow_triggered", Passed: true, Detail: "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"},
	}

	currentCeiling := []string{
		"continuation across future validation bundles remains workflow-triggered",
		"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
		"recent history is bounded to the exported bundle index and not an always-on service",
	}
	if !repeatedLaneCoverage {
		currentCeiling = append(currentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}

	report := ContinuationScorecardReport{
		GeneratedAt: now.Format(time.RFC3339Nano),
		Ticket:      "BIG-PAR-086-local-prework",
		Title:       "Validation bundle continuation scorecard",
		Status:      "local-continuation-scorecard",
		EvidenceInputs: ContinuationScorecardEvidenceInputs{
			ManifestPath:          options.ManifestPath,
			LatestSummaryPath:     options.LatestSummaryPath,
			BundleRoot:            options.BundleRootPath,
			RecentRunSummaries:    recentInputs,
			SharedQueueReportPath: options.SharedQueueReportPath,
			GeneratorScript:       ValidationBundleContinuationScorecardGenerator,
		},
		Summary: ContinuationScorecardSummary{
			RecentBundleCount:                           len(recentRuns),
			LatestRunID:                                 manifest.Latest.RunID,
			LatestStatus:                                manifest.Latest.Status,
			LatestBundleAgeHours:                        latestAgeHours,
			LatestAllExecutorTracksSucceeded:            latestAllSucceeded,
			RecentBundleChainHasNoFailures:              recentAllSucceeded,
			AllExecutorTracksHaveRepeatedRecentCoverage: repeatedLaneCoverage,
			BundleRootExists:                            pathExists(resolveReportPath(repoRoot, options.BundleRootPath)),
		},
		ExecutorLanes:        laneScorecards,
		SharedQueueCompanion: sharedQueueCompanion,
		ContinuationChecks:   checks,
		CurrentCeiling:       currentCeiling,
		NextRuntimeHooks: []string{
			"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
			"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
			"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
			"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
		},
	}
	if hasBundleGap {
		report.Summary.BundleGapMinutes = bundleGapMinutes
	}
	return report, nil
}

func BuildValidationBundleContinuationPolicyGate(repoRoot string, options ContinuationPolicyGateOptions) (ContinuationPolicyGateReport, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ContinuationPolicyGateReport{}, errors.New("repo root is required")
	}
	if options.ScorecardPath == "" {
		options.ScorecardPath = "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json"
	}
	if options.MaxLatestAgeHours <= 0 {
		options.MaxLatestAgeHours = 72.0
	}
	if options.MinRecentBundles <= 0 {
		options.MinRecentBundles = 2
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	mode, err := normalizeContinuationEnforcementMode(options.EnforcementMode, options.LegacyEnforce)
	if err != nil {
		return ContinuationPolicyGateReport{}, err
	}

	var scorecard ContinuationScorecardReport
	if err := loadJSON(resolveReportPath(repoRoot, options.ScorecardPath), &scorecard); err != nil {
		return ContinuationPolicyGateReport{}, err
	}

	checks := []ContinuationCheck{
		{
			Name:   "latest_bundle_age_within_threshold",
			Passed: scorecard.Summary.LatestBundleAgeHours <= options.MaxLatestAgeHours,
			Detail: fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", scorecard.Summary.LatestBundleAgeHours, options.MaxLatestAgeHours),
		},
		{
			Name:   "recent_bundle_count_meets_floor",
			Passed: scorecard.Summary.RecentBundleCount >= options.MinRecentBundles,
			Detail: fmt.Sprintf("recent_bundle_count=%d floor=%d", scorecard.Summary.RecentBundleCount, options.MinRecentBundles),
		},
		{
			Name:   "latest_bundle_all_executor_tracks_succeeded",
			Passed: scorecard.Summary.LatestAllExecutorTracksSucceeded,
			Detail: fmt.Sprintf("latest_all_executor_tracks_succeeded=%t", scorecard.Summary.LatestAllExecutorTracksSucceeded),
		},
		{
			Name:   "recent_bundle_chain_has_no_failures",
			Passed: scorecard.Summary.RecentBundleChainHasNoFailures,
			Detail: fmt.Sprintf("recent_bundle_chain_has_no_failures=%t", scorecard.Summary.RecentBundleChainHasNoFailures),
		},
		{
			Name:   "shared_queue_companion_available",
			Passed: scorecard.SharedQueueCompanion.Available,
			Detail: fmt.Sprintf("cross_node_completions=%d", scorecard.SharedQueueCompanion.CrossNodeCompletions),
		},
		{
			Name:   "repeated_lane_coverage_meets_policy",
			Passed: (!options.RequireRepeatedLaneCoverage) || scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage,
			Detail: fmt.Sprintf("require_repeated_lane_coverage=%t actual=%t", options.RequireRepeatedLaneCoverage, scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage),
		},
	}

	failingChecks := make([]string, 0)
	passingCount := 0
	for _, check := range checks {
		if check.Passed {
			passingCount++
			continue
		}
		failingChecks = append(failingChecks, check.Name)
	}

	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildContinuationEnforcementSummary(recommendation, mode)
	nextActions := buildContinuationNextActions(failingChecks)

	return ContinuationPolicyGateReport{
		GeneratedAt:    now.Format(time.RFC3339Nano),
		Ticket:         "OPE-262",
		Title:          "Validation workflow continuation gate",
		Status:         map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		Recommendation: recommendation,
		EvidenceInputs: ContinuationPolicyEvidenceInputs{
			ScorecardPath:   options.ScorecardPath,
			GeneratorScript: ValidationBundleContinuationPolicyGateGenerator,
		},
		PolicyInputs: ContinuationPolicyInputs{
			MaxLatestAgeHours:           options.MaxLatestAgeHours,
			MinRecentBundles:            options.MinRecentBundles,
			RequireRepeatedLaneCoverage: options.RequireRepeatedLaneCoverage,
		},
		Enforcement: enforcement,
		Summary: ContinuationPolicySummary{
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
			PassingCheckCount:                           passingCount,
			FailingCheckCount:                           len(failingChecks),
		},
		PolicyChecks:  checks,
		FailingChecks: failingChecks,
		ReviewerPath: ContinuationReviewerPath{
			IndexPath:  "docs/reports/live-validation-index.md",
			DigestPath: "docs/reports/validation-bundle-continuation-digest.md",
			DigestIssue: ContinuationReviewerIssue{
				ID:   "OPE-271",
				Slug: "BIG-PAR-082",
			},
		},
		SharedQueueCompanion: scorecard.SharedQueueCompanion,
		NextActions:          nextActions,
	}, nil
}

func FindRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(current)
	if err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if pathExists(filepath.Join(current, "go.mod")) && pathExists(filepath.Join(current, "docs", "reports")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("repo root not found")
}

func WriteJSON(path string, payload any) error {
	resolvedPath := path
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(resolvedPath, append(contents, '\n'), 0o644)
}

func loadJSON(path string, target any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(contents, target)
}

func resolveEvidencePath(repoRoot string, bigclawGoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	candidate := filepath.Clean(path)
	candidate = normalizeRepoRelativePath(repoRoot, candidate)
	searchRoots := []string{repoRoot}
	if !strings.HasPrefix(filepath.ToSlash(candidate), "bigclaw-go/") {
		searchRoots = append(searchRoots, bigclawGoRoot)
	}
	for _, root := range searchRoots {
		resolved := filepath.Join(root, candidate)
		if pathExists(resolved) {
			return resolved
		}
	}
	return filepath.Join(repoRoot, candidate)
}

func resolveReportPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, normalizeRepoRelativePath(repoRoot, path))
}

func normalizeRepoRelativePath(repoRoot string, path string) string {
	candidate := filepath.Clean(path)
	if filepath.Base(repoRoot) == "bigclaw-go" {
		prefix := "bigclaw-go" + string(filepath.Separator)
		if strings.HasPrefix(candidate, prefix) {
			return strings.TrimPrefix(candidate, prefix)
		}
		if filepath.ToSlash(candidate) == "bigclaw-go" {
			return "."
		}
	}
	return candidate
}

func buildContinuationLaneScorecard(runs []map[string]any, lane string) ContinuationLaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := asMap(run[lane])
		enabled := asBool(section["enabled"])
		status := "disabled"
		if enabled {
			status = firstNonEmptyString(asString(section["status"]), "missing")
			enabledRuns++
		}
		if status == "succeeded" {
			succeededRuns++
		}
		statuses = append(statuses, status)
	}
	latest := map[string]any{}
	if len(runs) > 0 {
		latest = asMap(runs[0][lane])
	}
	return ContinuationLaneScorecard{
		Lane:                   lane,
		LatestEnabled:          asBool(latest["enabled"]),
		LatestStatus:           firstNonEmptyString(asString(latest["status"]), "missing"),
		RecentStatuses:         statuses,
		EnabledRuns:            enabledRuns,
		SucceededRuns:          succeededRuns,
		ConsecutiveSuccesses:   consecutiveSuccesses(statuses),
		AllRecentRunsSucceeded: enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func collectStatuses(runs []map[string]any) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, firstNonEmptyString(asString(run["status"]), "unknown"))
	}
	return out
}

func latestSectionValue(run map[string]any, lane string, key string) any {
	return asMap(run[lane])[key]
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

func normalizeContinuationEnforcementMode(enforcementMode string, legacyEnforce bool) (string, error) {
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
		return "", fmt.Errorf("%w %q; expected one of review, hold, fail", errUnsupportedEnforcementMode, enforcementMode)
	}
}

func buildContinuationEnforcementSummary(recommendation string, mode string) ContinuationEnforcement {
	if recommendation == "go" {
		return ContinuationEnforcement{Mode: mode, Outcome: "pass", ExitCode: 0}
	}
	switch mode {
	case "review":
		return ContinuationEnforcement{Mode: mode, Outcome: "review-only", ExitCode: 0}
	case "hold":
		return ContinuationEnforcement{Mode: mode, Outcome: "hold", ExitCode: 2}
	default:
		return ContinuationEnforcement{Mode: mode, Outcome: "fail", ExitCode: 1}
	}
}

func buildContinuationNextActions(failingChecks []string) []string {
	nextActions := make([]string, 0, 4)
	checkSet := make(map[string]struct{}, len(failingChecks))
	for _, item := range failingChecks {
		checkSet[item] = struct{}{}
	}
	if _, ok := checkSet["latest_bundle_age_within_threshold"]; ok {
		nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
	}
	if _, ok := checkSet["recent_bundle_count_meets_floor"]; ok {
		nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
	}
	if _, ok := checkSet["shared_queue_companion_available"]; ok {
		nextActions = append(nextActions, "rerun `go run ./scripts/e2e/multi_node_shared_queue --report-path docs/reports/multi-node-shared-queue-report.json`")
	}
	if _, ok := checkSet["repeated_lane_coverage_meets_policy"]; ok {
		nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}
	return nextActions
}

func parseFlexibleTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(value, "Z", "+00:00"))
	return time.Parse(time.RFC3339Nano, trimmed)
}

func roundToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func asMap(value any) map[string]any {
	out, ok := value.(map[string]any)
	if ok {
		return out
	}
	return map[string]any{}
}

func asSlice(value any) []any {
	out, ok := value.([]any)
	if ok {
		return out
	}
	return nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", value)
	}
}

func asBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func asBoolWithFallback(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if ok {
		return typed
	}
	return fallback
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func asIntWithFallback(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
