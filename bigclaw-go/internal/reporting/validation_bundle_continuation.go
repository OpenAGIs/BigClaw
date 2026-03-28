package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var executorLanes = []string{"local", "kubernetes", "ray"}

type ValidationBundleContinuationScorecardOptions struct {
	RepoRoot          string
	IndexManifestPath string
	BundleRootPath    string
	LatestSummaryPath string
	SharedQueueReport string
}

type ValidationBundleContinuationPolicyGateOptions struct {
	RepoRoot                    string
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforceContinuation   bool
}

type continuationManifest struct {
	Latest struct {
		RunID       string `json:"run_id"`
		GeneratedAt string `json:"generated_at"`
		Status      string `json:"status"`
	} `json:"latest"`
	RecentRuns []struct {
		SummaryPath string `json:"summary_path"`
	} `json:"recent_runs"`
}

type continuationLaneReport struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type continuationRunSummary struct {
	RunID                string                 `json:"run_id"`
	GeneratedAt          string                 `json:"generated_at"`
	Status               string                 `json:"status"`
	Local                continuationLaneReport `json:"local"`
	Kubernetes           continuationLaneReport `json:"kubernetes"`
	Ray                  continuationLaneReport `json:"ray"`
	SharedQueueCompanion map[string]any         `json:"shared_queue_companion"`
}

type ValidationBundleContinuationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type ValidationBundleContinuationLaneScorecard struct {
	Lane                   string   `json:"lane"`
	LatestEnabled          bool     `json:"latest_enabled"`
	LatestStatus           string   `json:"latest_status"`
	RecentStatuses         []string `json:"recent_statuses"`
	EnabledRuns            int      `json:"enabled_runs"`
	SucceededRuns          int      `json:"succeeded_runs"`
	ConsecutiveSuccesses   int      `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool     `json:"all_recent_runs_succeeded"`
}

type ValidationBundleContinuationScorecard struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ManifestPath       string   `json:"manifest_path"`
		LatestSummaryPath  string   `json:"latest_summary_path"`
		BundleRoot         string   `json:"bundle_root"`
		RecentRunSummaries []string `json:"recent_run_summaries"`
		SharedQueueReport  string   `json:"shared_queue_report_path"`
		GeneratorScript    string   `json:"generator_script"`
	} `json:"evidence_inputs"`
	Summary struct {
		RecentBundleCount                           int     `json:"recent_bundle_count"`
		LatestRunID                                 string  `json:"latest_run_id"`
		LatestStatus                                string  `json:"latest_status"`
		LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours"`
		LatestAllExecutorTracksSucceeded            bool    `json:"latest_all_executor_tracks_succeeded"`
		RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
		AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
		BundleGapMinutes                            float64 `json:"bundle_gap_minutes,omitempty"`
		BundleRootExists                            bool    `json:"bundle_root_exists"`
	} `json:"summary"`
	ExecutorLanes        []ValidationBundleContinuationLaneScorecard `json:"executor_lanes"`
	SharedQueueCompanion map[string]any                              `json:"shared_queue_companion"`
	ContinuationChecks   []ValidationBundleContinuationCheck         `json:"continuation_checks"`
	CurrentCeiling       []string                                    `json:"current_ceiling"`
	NextRuntimeHooks     []string                                    `json:"next_runtime_hooks"`
}

type ValidationBundleContinuationPolicyGate struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Recommendation string `json:"recommendation"`
	EvidenceInputs struct {
		ScorecardPath   string `json:"scorecard_path"`
		GeneratorScript string `json:"generator_script"`
	} `json:"evidence_inputs"`
	PolicyInputs struct {
		MaxLatestAgeHours           float64 `json:"max_latest_age_hours"`
		MinRecentBundles            int     `json:"min_recent_bundles"`
		RequireRepeatedLaneCoverage bool    `json:"require_repeated_lane_coverage"`
	} `json:"policy_inputs"`
	Enforcement struct {
		Mode     string `json:"mode"`
		Outcome  string `json:"outcome"`
		ExitCode int    `json:"exit_code"`
	} `json:"enforcement"`
	Summary struct {
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
	} `json:"summary"`
	PolicyChecks  []ValidationBundleContinuationCheck `json:"policy_checks"`
	FailingChecks []string                            `json:"failing_checks"`
	ReviewerPath  struct {
		IndexPath   string `json:"index_path"`
		DigestPath  string `json:"digest_path"`
		DigestIssue struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"digest_issue"`
	} `json:"reviewer_path"`
	SharedQueueCompanion map[string]any `json:"shared_queue_companion"`
	NextActions          []string       `json:"next_actions"`
}

func BuildValidationBundleContinuationScorecard(opts ValidationBundleContinuationScorecardOptions) (ValidationBundleContinuationScorecard, error) {
	repoRoot, err := resolveContinuationRepoRoot(opts.RepoRoot)
	if err != nil {
		return ValidationBundleContinuationScorecard{}, err
	}
	if opts.IndexManifestPath == "" {
		opts.IndexManifestPath = "bigclaw-go/docs/reports/live-validation-index.json"
	}
	if opts.BundleRootPath == "" {
		opts.BundleRootPath = "bigclaw-go/docs/reports/live-validation-runs"
	}
	if opts.LatestSummaryPath == "" {
		opts.LatestSummaryPath = "bigclaw-go/docs/reports/live-validation-summary.json"
	}
	if opts.SharedQueueReport == "" {
		opts.SharedQueueReport = "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"
	}

	var manifest continuationManifest
	if err := readJSONFile(resolveRepoPath(repoRoot, opts.IndexManifestPath), &manifest); err != nil {
		return ValidationBundleContinuationScorecard{}, err
	}

	bigclawGoRoot := filepath.Join(repoRoot, "bigclaw-go")
	recentRuns := make([]continuationRunSummary, 0, len(manifest.RecentRuns))
	recentRunInputs := make([]string, 0, len(manifest.RecentRuns))
	for _, item := range manifest.RecentRuns {
		resolved := resolveEvidencePath(repoRoot, bigclawGoRoot, item.SummaryPath)
		var summary continuationRunSummary
		if err := readJSONFile(resolved, &summary); err != nil {
			return ValidationBundleContinuationScorecard{}, err
		}
		recentRuns = append(recentRuns, summary)
		recentRunInputs = append(recentRunInputs, relPath(repoRoot, resolved))
	}

	var latestSummary continuationRunSummary
	if err := readJSONFile(resolveRepoPath(repoRoot, opts.LatestSummaryPath), &latestSummary); err != nil {
		return ValidationBundleContinuationScorecard{}, err
	}
	var sharedQueue map[string]any
	if err := readJSONFile(resolveRepoPath(repoRoot, opts.SharedQueueReport), &sharedQueue); err != nil {
		return ValidationBundleContinuationScorecard{}, err
	}

	laneScorecards := []ValidationBundleContinuationLaneScorecard{
		buildLaneScorecard(recentRuns, "local"),
		buildLaneScorecard(recentRuns, "kubernetes"),
		buildLaneScorecard(recentRuns, "ray"),
	}

	latestGeneratedAt, err := parseContinuationTime(manifest.Latest.GeneratedAt)
	if err != nil {
		return ValidationBundleContinuationScorecard{}, err
	}
	var previousGeneratedAt time.Time
	hasPrevious := false
	if len(recentRuns) > 1 {
		previousGeneratedAt, err = parseContinuationTime(recentRuns[1].GeneratedAt)
		if err != nil {
			return ValidationBundleContinuationScorecard{}, err
		}
		hasPrevious = true
	}
	generatedAt := time.Now().UTC()
	latestAgeHours := roundTo((generatedAt.Sub(latestGeneratedAt).Seconds() / 3600), 2)
	bundleGapMinutes := 0.0
	if hasPrevious {
		bundleGapMinutes = roundTo((latestGeneratedAt.Sub(previousGeneratedAt).Seconds() / 60), 2)
	}

	latestLaneStatuses := map[string]string{
		"local":      latestSummary.Local.Status,
		"kubernetes": latestSummary.Kubernetes.Status,
		"ray":        latestSummary.Ray.Status,
	}
	latestAllSucceeded := latestSummary.Local.Status == "succeeded" &&
		latestSummary.Kubernetes.Status == "succeeded" &&
		latestSummary.Ray.Status == "succeeded"
	recentAllSucceeded := true
	for _, run := range recentRuns {
		if run.Status != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}
	repeatedLaneCoverage := true
	enabledRunsByLane := map[string]int{}
	for _, lane := range laneScorecards {
		enabledRunsByLane[lane.Lane] = lane.EnabledRuns
		if lane.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	bundledSharedQueue := latestSummary.SharedQueueCompanion
	available := boolFromAny(bundledSharedQueue["available"]) || boolFromAny(sharedQueue["all_ok"])
	sharedQueueCompanion := map[string]any{
		"available":                 available,
		"report_path":               firstNonEmptyString(anyToString(bundledSharedQueue["canonical_report_path"]), opts.SharedQueueReport),
		"summary_path":              firstNonEmptyString(anyToString(bundledSharedQueue["canonical_summary_path"]), "bigclaw-go/docs/reports/shared-queue-companion-summary.json"),
		"bundle_report_path":        anyToString(bundledSharedQueue["bundle_report_path"]),
		"bundle_summary_path":       anyToString(bundledSharedQueue["bundle_summary_path"]),
		"cross_node_completions":    intFromAny(firstNonNil(bundledSharedQueue["cross_node_completions"], sharedQueue["cross_node_completions"])),
		"duplicate_completed_tasks": intFromAny(firstNonNil(bundledSharedQueue["duplicate_completed_tasks"], lenAny(sharedQueue["duplicate_completed_tasks"]))),
		"duplicate_started_tasks":   intFromAny(firstNonNil(bundledSharedQueue["duplicate_started_tasks"], lenAny(sharedQueue["duplicate_started_tasks"]))),
		"mode":                      ternaryString(len(bundledSharedQueue) > 0, "bundle-companion-summary", "standalone-proof"),
	}

	scorecard := ValidationBundleContinuationScorecard{
		GeneratedAt:          generatedAt.Format(time.RFC3339Nano),
		Ticket:               "BIG-PAR-086-local-prework",
		Title:                "Validation bundle continuation scorecard",
		Status:               "local-continuation-scorecard",
		ExecutorLanes:        laneScorecards,
		SharedQueueCompanion: sharedQueueCompanion,
		ContinuationChecks: []ValidationBundleContinuationCheck{
			{Name: "latest_bundle_all_executor_tracks_succeeded", Passed: latestAllSucceeded, Detail: fmt.Sprintf("latest lane statuses=%s", pythonStyleStringMap(latestLaneStatuses))},
			{Name: "recent_bundle_chain_has_multiple_runs", Passed: len(recentRuns) >= 2, Detail: fmt.Sprintf("recent bundle count=%d", len(recentRuns))},
			{Name: "recent_bundle_chain_has_no_failures", Passed: recentAllSucceeded, Detail: fmt.Sprintf("recent bundle statuses=%s", pythonStyleStringSlice(statusesFromRuns(recentRuns)))},
			{Name: "all_executor_tracks_have_repeated_recent_coverage", Passed: repeatedLaneCoverage, Detail: fmt.Sprintf("enabled_runs_by_lane=%s", pythonStyleIntMap(enabledRunsByLane))},
			{Name: "shared_queue_companion_proof_available", Passed: available, Detail: fmt.Sprintf("cross_node_completions=%d", intFromAny(sharedQueueCompanion["cross_node_completions"]))},
			{Name: "continuation_surface_is_workflow_triggered", Passed: true, Detail: "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"},
		},
		CurrentCeiling: []string{
			"continuation across future validation bundles remains workflow-triggered",
			"shared-queue coordination proof now ships as adjacent bundle metadata rather than an executor-native lane",
			"recent history is bounded to the exported bundle index and not an always-on service",
		},
		NextRuntimeHooks: []string{
			"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=hold or fail in workflow closeout when continuation holds should block or fail the run",
			"decide whether shared-queue coordination should stay as adjacent bundle metadata or gain its own executor-native validation lane",
			"extend the automatic continuation refresh beyond run_all.sh into broader workflow orchestrators",
			"extend the scorecard beyond the latest recent_runs window when more longitudinal evidence exists",
		},
	}
	scorecard.EvidenceInputs.ManifestPath = opts.IndexManifestPath
	scorecard.EvidenceInputs.LatestSummaryPath = opts.LatestSummaryPath
	scorecard.EvidenceInputs.BundleRoot = opts.BundleRootPath
	scorecard.EvidenceInputs.RecentRunSummaries = recentRunInputs
	scorecard.EvidenceInputs.SharedQueueReport = opts.SharedQueueReport
	scorecard.EvidenceInputs.GeneratorScript = "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.go"
	scorecard.Summary.RecentBundleCount = len(recentRuns)
	scorecard.Summary.LatestRunID = manifest.Latest.RunID
	scorecard.Summary.LatestStatus = manifest.Latest.Status
	scorecard.Summary.LatestBundleAgeHours = latestAgeHours
	scorecard.Summary.LatestAllExecutorTracksSucceeded = latestAllSucceeded
	scorecard.Summary.RecentBundleChainHasNoFailures = recentAllSucceeded
	scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage = repeatedLaneCoverage
	if hasPrevious {
		scorecard.Summary.BundleGapMinutes = bundleGapMinutes
	}
	scorecard.Summary.BundleRootExists = pathExists(resolveRepoPath(repoRoot, opts.BundleRootPath))
	if !repeatedLaneCoverage {
		scorecard.CurrentCeiling = append(scorecard.CurrentCeiling, "not every executor lane is enabled across every indexed bundle in the current recent window")
	}
	return scorecard, nil
}

func BuildValidationBundleContinuationPolicyGate(opts ValidationBundleContinuationPolicyGateOptions) (ValidationBundleContinuationPolicyGate, error) {
	repoRoot, err := resolveContinuationRepoRoot(opts.RepoRoot)
	if err != nil {
		return ValidationBundleContinuationPolicyGate{}, err
	}
	if opts.ScorecardPath == "" {
		opts.ScorecardPath = "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json"
	}
	if opts.MaxLatestAgeHours == 0 {
		opts.MaxLatestAgeHours = 72
	}
	if opts.MinRecentBundles == 0 {
		opts.MinRecentBundles = 2
	}
	mode, err := normalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuation)
	if err != nil {
		return ValidationBundleContinuationPolicyGate{}, err
	}

	var scorecard ValidationBundleContinuationScorecard
	if err := readJSONFile(resolveRepoPath(repoRoot, opts.ScorecardPath), &scorecard); err != nil {
		return ValidationBundleContinuationPolicyGate{}, err
	}

	checks := []ValidationBundleContinuationCheck{
		{Name: "latest_bundle_age_within_threshold", Passed: scorecard.Summary.LatestBundleAgeHours <= opts.MaxLatestAgeHours, Detail: fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", scorecard.Summary.LatestBundleAgeHours, opts.MaxLatestAgeHours)},
		{Name: "recent_bundle_count_meets_floor", Passed: scorecard.Summary.RecentBundleCount >= opts.MinRecentBundles, Detail: fmt.Sprintf("recent_bundle_count=%d floor=%d", scorecard.Summary.RecentBundleCount, opts.MinRecentBundles)},
		{Name: "latest_bundle_all_executor_tracks_succeeded", Passed: scorecard.Summary.LatestAllExecutorTracksSucceeded, Detail: fmt.Sprintf("latest_all_executor_tracks_succeeded=%s", pythonStyleBool(scorecard.Summary.LatestAllExecutorTracksSucceeded))},
		{Name: "recent_bundle_chain_has_no_failures", Passed: scorecard.Summary.RecentBundleChainHasNoFailures, Detail: fmt.Sprintf("recent_bundle_chain_has_no_failures=%s", pythonStyleBool(scorecard.Summary.RecentBundleChainHasNoFailures))},
		{Name: "shared_queue_companion_available", Passed: boolFromAny(scorecard.SharedQueueCompanion["available"]), Detail: fmt.Sprintf("cross_node_completions=%d", intFromAny(scorecard.SharedQueueCompanion["cross_node_completions"]))},
		{Name: "repeated_lane_coverage_meets_policy", Passed: !opts.RequireRepeatedLaneCoverage || scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage, Detail: fmt.Sprintf("require_repeated_lane_coverage=%s actual=%s", pythonStyleBool(opts.RequireRepeatedLaneCoverage), pythonStyleBool(scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage))},
	}
	failingChecks := []string{}
	for _, check := range checks {
		if !check.Passed {
			failingChecks = append(failingChecks, check.Name)
		}
	}
	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildEnforcementSummary(recommendation, mode)

	nextActions := []string{}
	for _, name := range failingChecks {
		switch name {
		case "latest_bundle_age_within_threshold":
			nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
		case "recent_bundle_count_meets_floor":
			nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
		case "shared_queue_companion_available":
			nextActions = append(nextActions, "rerun `python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
		case "repeated_lane_coverage_meets_policy":
			nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
		}
	}
	if len(nextActions) == 0 {
		nextActions = append(nextActions, "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions")
	}

	gate := ValidationBundleContinuationPolicyGate{
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339Nano),
		Ticket:               "OPE-262",
		Title:                "Validation workflow continuation gate",
		Status:               ternaryString(recommendation == "go", "policy-go", "policy-hold"),
		Recommendation:       recommendation,
		PolicyChecks:         checks,
		FailingChecks:        failingChecks,
		SharedQueueCompanion: scorecard.SharedQueueCompanion,
		NextActions:          nextActions,
	}
	gate.EvidenceInputs.ScorecardPath = opts.ScorecardPath
	gate.EvidenceInputs.GeneratorScript = "bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.go"
	gate.PolicyInputs.MaxLatestAgeHours = opts.MaxLatestAgeHours
	gate.PolicyInputs.MinRecentBundles = opts.MinRecentBundles
	gate.PolicyInputs.RequireRepeatedLaneCoverage = opts.RequireRepeatedLaneCoverage
	gate.Enforcement.Mode = enforcement.Mode
	gate.Enforcement.Outcome = enforcement.Outcome
	gate.Enforcement.ExitCode = enforcement.ExitCode
	gate.Summary.LatestRunID = scorecard.Summary.LatestRunID
	gate.Summary.LatestBundleAgeHours = scorecard.Summary.LatestBundleAgeHours
	gate.Summary.RecentBundleCount = scorecard.Summary.RecentBundleCount
	gate.Summary.LatestAllExecutorTracksSucceeded = scorecard.Summary.LatestAllExecutorTracksSucceeded
	gate.Summary.RecentBundleChainHasNoFailures = scorecard.Summary.RecentBundleChainHasNoFailures
	gate.Summary.AllExecutorTracksHaveRepeatedRecentCoverage = scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage
	gate.Summary.Recommendation = recommendation
	gate.Summary.EnforcementMode = enforcement.Mode
	gate.Summary.WorkflowOutcome = enforcement.Outcome
	gate.Summary.WorkflowExitCode = enforcement.ExitCode
	gate.Summary.PassingCheckCount = len(checks) - len(failingChecks)
	gate.Summary.FailingCheckCount = len(failingChecks)
	gate.ReviewerPath.IndexPath = "docs/reports/live-validation-index.md"
	gate.ReviewerPath.DigestPath = "docs/reports/validation-bundle-continuation-digest.md"
	gate.ReviewerPath.DigestIssue.ID = "OPE-271"
	gate.ReviewerPath.DigestIssue.Slug = "BIG-PAR-082"
	return gate, nil
}

func WriteValidationBundleContinuationReport(path string, payload any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

type continuationEnforcement struct {
	Mode     string
	Outcome  string
	ExitCode int
}

func buildEnforcementSummary(recommendation string, enforcementMode string) continuationEnforcement {
	if recommendation == "go" {
		return continuationEnforcement{Mode: enforcementMode, Outcome: "pass", ExitCode: 0}
	}
	if enforcementMode == "review" {
		return continuationEnforcement{Mode: enforcementMode, Outcome: "review-only", ExitCode: 0}
	}
	if enforcementMode == "hold" {
		return continuationEnforcement{Mode: enforcementMode, Outcome: "hold", ExitCode: 2}
	}
	return continuationEnforcement{Mode: enforcementMode, Outcome: "fail", ExitCode: 1}
}

func normalizeEnforcementMode(mode string, legacyEnforce bool) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
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

func buildLaneScorecard(runs []continuationRunSummary, lane string) ValidationBundleContinuationLaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := laneSection(run, lane)
		status := "disabled"
		if section.Enabled {
			status = firstNonEmptyString(section.Status, "missing")
			enabledRuns++
		}
		if status == "succeeded" {
			succeededRuns++
		}
		statuses = append(statuses, status)
	}
	latest := continuationLaneReport{}
	if len(runs) > 0 {
		latest = laneSection(runs[0], lane)
	}
	return ValidationBundleContinuationLaneScorecard{
		Lane:                   lane,
		LatestEnabled:          latest.Enabled,
		LatestStatus:           ternaryString(latest.Enabled, firstNonEmptyString(latest.Status, "missing"), "missing"),
		RecentStatuses:         statuses,
		EnabledRuns:            enabledRuns,
		SucceededRuns:          succeededRuns,
		ConsecutiveSuccesses:   consecutiveSuccesses(statuses),
		AllRecentRunsSucceeded: enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func laneSection(run continuationRunSummary, lane string) continuationLaneReport {
	switch lane {
	case "local":
		return run.Local
	case "kubernetes":
		return run.Kubernetes
	case "ray":
		return run.Ray
	default:
		return continuationLaneReport{}
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

func statusesFromRuns(runs []continuationRunSummary) []string {
	statuses := make([]string, 0, len(runs))
	for _, run := range runs {
		statuses = append(statuses, run.Status)
	}
	return statuses
}

func resolveContinuationRepoRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if pathExists(filepath.Join(cwd, "bigclaw-go", "go.mod")) {
		return cwd, nil
	}
	if pathExists(filepath.Join(cwd, "go.mod")) && filepath.Base(cwd) == "bigclaw-go" {
		return filepath.Dir(cwd), nil
	}
	return cwd, nil
}

func resolveRepoPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func resolveEvidencePath(repoRoot string, bigclawGoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if !strings.HasPrefix(path, "bigclaw-go"+string(os.PathSeparator)) && path != "bigclaw-go" {
		candidate := filepath.Join(repoRoot, path)
		if pathExists(candidate) {
			return candidate
		}
		candidate = filepath.Join(bigclawGoRoot, path)
		if pathExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(repoRoot, path)
}

func readJSONFile(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty json file: " + path)
	}
	return json.Unmarshal(body, target)
}

func parseContinuationTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.ReplaceAll(strings.TrimSpace(value), "Z", "+00:00"))
}

func relPath(root string, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(relative)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func roundTo(value float64, digits int) float64 {
	pow := math.Pow(10, float64(digits))
	return math.Round(value*pow) / pow
}

func boolFromAny(value any) bool {
	boolean, ok := value.(bool)
	return ok && boolean
}

func intFromAny(value any) int {
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

func lenAny(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []string:
		return len(typed)
	default:
		return 0
	}
}

func anyToString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func ternaryString(condition bool, whenTrue string, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func pythonStyleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func pythonStyleStringSlice(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("'%s'", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func pythonStyleStringMap(values map[string]string) string {
	parts := make([]string, 0, len(executorLanes))
	for _, key := range executorLanes {
		parts = append(parts, fmt.Sprintf("'%s': '%s'", key, values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func pythonStyleIntMap(values map[string]int) string {
	parts := make([]string, 0, len(executorLanes))
	for _, key := range executorLanes {
		parts = append(parts, fmt.Sprintf("'%s': %d", key, values[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
