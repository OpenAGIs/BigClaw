package migration

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
	DefaultValidationIndexPath              = "docs/reports/live-validation-index.json"
	DefaultValidationBundleRootPath         = "docs/reports/live-validation-runs"
	DefaultValidationSummaryPath            = "docs/reports/live-validation-summary.json"
	DefaultSharedQueueReportPath            = "docs/reports/multi-node-shared-queue-report.json"
	DefaultValidationContinuationScorecard  = "docs/reports/validation-bundle-continuation-scorecard.json"
	DefaultValidationContinuationPolicyGate = "docs/reports/validation-bundle-continuation-policy-gate.json"
	defaultSharedQueueSummaryPath           = "docs/reports/shared-queue-companion-summary.json"
	defaultValidationIndexReviewerPath      = "docs/reports/live-validation-index.md"
	defaultValidationDigestReviewerPath     = "docs/reports/validation-bundle-continuation-digest.md"
	defaultContinuationScorecardTicket      = "BIG-GO-907"
	defaultContinuationScorecardTitle       = "Validation bundle continuation scorecard"
	defaultContinuationScorecardStatus      = "go-validation-continuation-scorecard"
	defaultContinuationPolicyGateTicket     = "BIG-GO-907"
	defaultContinuationPolicyGateTitle      = "Validation workflow continuation gate"
	defaultContinuationPolicyGateGoStatus   = "policy-go"
	defaultContinuationPolicyGateHoldStatus = "policy-hold"
)

var executorLanes = []string{"local", "kubernetes", "ray"}

type ValidationContinuationScorecardConfig struct {
	RepoRoot              string
	GoRoot                string
	IndexManifestPath     string
	BundleRootPath        string
	SummaryPath           string
	SharedQueueReportPath string
}

type ValidationContinuationPolicyGateConfig struct {
	RepoRoot                    string
	GoRoot                      string
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
}

type validationIndexDocument struct {
	Latest struct {
		RunID       string `json:"run_id"`
		GeneratedAt string `json:"generated_at"`
		Status      string `json:"status"`
	} `json:"latest"`
	RecentRuns []struct {
		RunID       string `json:"run_id"`
		GeneratedAt string `json:"generated_at"`
		Status      string `json:"status"`
		SummaryPath string `json:"summary_path"`
	} `json:"recent_runs"`
}

type validationRunLane struct {
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type validationRunSharedQueue struct {
	Available               bool   `json:"available"`
	CanonicalReportPath     string `json:"canonical_report_path"`
	CanonicalSummaryPath    string `json:"canonical_summary_path"`
	BundleReportPath        string `json:"bundle_report_path"`
	BundleSummaryPath       string `json:"bundle_summary_path"`
	Status                  string `json:"status"`
	CrossNodeCompletions    int    `json:"cross_node_completions"`
	DuplicateStartedTasks   any    `json:"duplicate_started_tasks"`
	DuplicateCompletedTasks any    `json:"duplicate_completed_tasks"`
}

type validationRunSummaryDocument struct {
	GeneratedAt          string                   `json:"generated_at"`
	Status               string                   `json:"status"`
	Local                validationRunLane        `json:"local"`
	Kubernetes           validationRunLane        `json:"kubernetes"`
	Ray                  validationRunLane        `json:"ray"`
	SharedQueueCompanion validationRunSharedQueue `json:"shared_queue_companion"`
}

type sharedQueueReportDocument struct {
	AllOK                   bool  `json:"all_ok"`
	CrossNodeCompletions    int   `json:"cross_node_completions"`
	DuplicateStartedTasks   []any `json:"duplicate_started_tasks"`
	DuplicateCompletedTasks []any `json:"duplicate_completed_tasks"`
}

type ValidationContinuationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type ValidationContinuationLaneScorecard struct {
	Lane                   string   `json:"lane"`
	LatestEnabled          bool     `json:"latest_enabled"`
	LatestStatus           string   `json:"latest_status"`
	RecentStatuses         []string `json:"recent_statuses"`
	EnabledRuns            int      `json:"enabled_runs"`
	SucceededRuns          int      `json:"succeeded_runs"`
	ConsecutiveSuccesses   int      `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool     `json:"all_recent_runs_succeeded"`
}

type ValidationContinuationSharedQueueCompanion struct {
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

type ValidationContinuationScorecardDocument struct {
	GeneratedAt          string                                     `json:"generated_at"`
	Ticket               string                                     `json:"ticket"`
	Title                string                                     `json:"title"`
	Status               string                                     `json:"status"`
	EvidenceInputs       map[string]any                             `json:"evidence_inputs"`
	Summary              map[string]any                             `json:"summary"`
	ExecutorLanes        []ValidationContinuationLaneScorecard      `json:"executor_lanes"`
	SharedQueueCompanion ValidationContinuationSharedQueueCompanion `json:"shared_queue_companion"`
	ContinuationChecks   []ValidationContinuationCheck              `json:"continuation_checks"`
	CurrentCeiling       []string                                   `json:"current_ceiling"`
	NextRuntimeHooks     []string                                   `json:"next_runtime_hooks"`
}

type ValidationContinuationPolicyGateDocument struct {
	GeneratedAt          string                        `json:"generated_at"`
	Ticket               string                        `json:"ticket"`
	Title                string                        `json:"title"`
	Status               string                        `json:"status"`
	Recommendation       string                        `json:"recommendation"`
	EvidenceInputs       map[string]any                `json:"evidence_inputs"`
	PolicyInputs         map[string]any                `json:"policy_inputs"`
	Enforcement          map[string]any                `json:"enforcement"`
	Summary              map[string]any                `json:"summary"`
	PolicyChecks         []ValidationContinuationCheck `json:"policy_checks"`
	FailingChecks        []string                      `json:"failing_checks"`
	ReviewerPath         map[string]any                `json:"reviewer_path"`
	SharedQueueCompanion map[string]any                `json:"shared_queue_companion"`
	NextActions          []string                      `json:"next_actions"`
}

func BuildValidationContinuationScorecard(config ValidationContinuationScorecardConfig) (ValidationContinuationScorecardDocument, error) {
	repoRoot, goRoot, err := normalizeRoots(config.RepoRoot, config.GoRoot)
	if err != nil {
		return ValidationContinuationScorecardDocument{}, err
	}
	indexPath := defaultString(config.IndexManifestPath, DefaultValidationIndexPath)
	bundleRootPath := defaultString(config.BundleRootPath, DefaultValidationBundleRootPath)
	summaryPath := defaultString(config.SummaryPath, DefaultValidationSummaryPath)
	sharedQueueReportPath := defaultString(config.SharedQueueReportPath, DefaultSharedQueueReportPath)

	var manifest validationIndexDocument
	if err := readJSON(resolveGoRelative(goRoot, indexPath), &manifest); err != nil {
		return ValidationContinuationScorecardDocument{}, err
	}
	var latestSummary validationRunSummaryDocument
	if err := readJSON(resolveGoRelative(goRoot, summaryPath), &latestSummary); err != nil {
		return ValidationContinuationScorecardDocument{}, err
	}
	var sharedQueue sharedQueueReportDocument
	if err := readJSON(resolveGoRelative(goRoot, sharedQueueReportPath), &sharedQueue); err != nil {
		return ValidationContinuationScorecardDocument{}, err
	}

	recentRuns := make([]validationRunSummaryDocument, 0, len(manifest.RecentRuns))
	recentRunInputs := make([]string, 0, len(manifest.RecentRuns))
	for _, item := range manifest.RecentRuns {
		summaryFile, err := resolveEvidencePath(repoRoot, goRoot, item.SummaryPath)
		if err != nil {
			return ValidationContinuationScorecardDocument{}, err
		}
		var run validationRunSummaryDocument
		if err := readJSON(summaryFile, &run); err != nil {
			return ValidationContinuationScorecardDocument{}, err
		}
		recentRuns = append(recentRuns, run)
		recentRunInputs = append(recentRunInputs, relpath(summaryFile, goRoot))
	}

	laneScorecards := make([]ValidationContinuationLaneScorecard, 0, len(executorLanes))
	enabledRunsByLane := map[string]int{}
	latestLaneStatuses := map[string]string{}
	latestAllSucceeded := true
	repeatedLaneCoverage := true
	for _, lane := range executorLanes {
		scorecard := buildLaneScorecard(recentRuns, lane)
		laneScorecards = append(laneScorecards, scorecard)
		enabledRunsByLane[lane] = scorecard.EnabledRuns
		latestLaneStatuses[lane] = laneStatus(latestSummary, lane)
		if latestLaneStatuses[lane] != "succeeded" {
			latestAllSucceeded = false
		}
		if scorecard.EnabledRuns < 2 {
			repeatedLaneCoverage = false
		}
	}

	sort.Slice(laneScorecards, func(i, j int) bool { return laneScorecards[i].Lane < laneScorecards[j].Lane })
	recentAllSucceeded := len(recentRuns) > 0
	for _, run := range recentRuns {
		if run.Status != "succeeded" {
			recentAllSucceeded = false
			break
		}
	}

	now := time.Now().UTC()
	latestGeneratedAt, err := parseTime(manifest.Latest.GeneratedAt)
	if err != nil {
		return ValidationContinuationScorecardDocument{}, fmt.Errorf("parse latest generated_at: %w", err)
	}
	latestAgeHours := round2(now.Sub(latestGeneratedAt).Hours())
	var bundleGapMinutes any
	if len(recentRuns) > 1 {
		previousGeneratedAt, err := parseTime(recentRuns[1].GeneratedAt)
		if err != nil {
			return ValidationContinuationScorecardDocument{}, fmt.Errorf("parse previous generated_at: %w", err)
		}
		bundleGapMinutes = round2(latestGeneratedAt.Sub(previousGeneratedAt).Minutes())
	} else {
		bundleGapMinutes = nil
	}

	bundledSharedQueue := latestSummary.SharedQueueCompanion
	sharedQueueAvailable := bundledSharedQueue.Available || sharedQueue.AllOK
	sharedQueueCompanion := ValidationContinuationSharedQueueCompanion{
		Available:               sharedQueueAvailable,
		ReportPath:              firstNonEmpty(bundledSharedQueue.CanonicalReportPath, sharedQueueReportPath),
		SummaryPath:             firstNonEmpty(bundledSharedQueue.CanonicalSummaryPath, defaultSharedQueueSummaryPath),
		BundleReportPath:        bundledSharedQueue.BundleReportPath,
		BundleSummaryPath:       bundledSharedQueue.BundleSummaryPath,
		CrossNodeCompletions:    firstInt(bundledSharedQueue.CrossNodeCompletions, sharedQueue.CrossNodeCompletions),
		DuplicateCompletedTasks: firstInt(countAny(bundledSharedQueue.DuplicateCompletedTasks), len(sharedQueue.DuplicateCompletedTasks)),
		DuplicateStartedTasks:   firstInt(countAny(bundledSharedQueue.DuplicateStartedTasks), len(sharedQueue.DuplicateStartedTasks)),
		Mode:                    "standalone-proof",
	}
	if bundledSharedQueue.Available || bundledSharedQueue.BundleReportPath != "" || bundledSharedQueue.BundleSummaryPath != "" {
		sharedQueueCompanion.Mode = "bundle-companion-summary"
	}

	continuationChecks := []ValidationContinuationCheck{
		check("latest_bundle_all_executor_tracks_succeeded", latestAllSucceeded, fmt.Sprintf("latest lane statuses=%v", latestLaneStatuses)),
		check("recent_bundle_chain_has_multiple_runs", len(recentRuns) >= 2, fmt.Sprintf("recent bundle count=%d", len(recentRuns))),
		check("recent_bundle_chain_has_no_failures", recentAllSucceeded, fmt.Sprintf("recent bundle statuses=%v", collectRunStatuses(recentRuns))),
		check("all_executor_tracks_have_repeated_recent_coverage", repeatedLaneCoverage, fmt.Sprintf("enabled_runs_by_lane=%v", enabledRunsByLane)),
		check("shared_queue_companion_proof_available", sharedQueueAvailable, fmt.Sprintf("cross_node_completions=%d", sharedQueueCompanion.CrossNodeCompletions)),
		check("continuation_surface_is_workflow_triggered", true, "run_all closeout now refreshes the scorecard and gate automatically, but continuation still depends on explicit workflow execution instead of an always-on service"),
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

	return ValidationContinuationScorecardDocument{
		GeneratedAt: utcISO(now),
		Ticket:      defaultContinuationScorecardTicket,
		Title:       defaultContinuationScorecardTitle,
		Status:      defaultContinuationScorecardStatus,
		EvidenceInputs: map[string]any{
			"manifest_path":            indexPath,
			"latest_summary_path":      summaryPath,
			"bundle_root":              bundleRootPath,
			"recent_run_summaries":     recentRunInputs,
			"shared_queue_report_path": sharedQueueReportPath,
			"generator":                "go run ./cmd/bigclawctl migration validation-continuation-scorecard",
		},
		Summary: map[string]any{
			"recent_bundle_count":                               len(recentRuns),
			"latest_run_id":                                     manifest.Latest.RunID,
			"latest_status":                                     manifest.Latest.Status,
			"latest_bundle_age_hours":                           latestAgeHours,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentAllSucceeded,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedLaneCoverage,
			"bundle_gap_minutes":                                bundleGapMinutes,
			"bundle_root_exists":                                pathExists(resolveGoRelative(goRoot, bundleRootPath)),
		},
		ExecutorLanes:        laneScorecards,
		SharedQueueCompanion: sharedQueueCompanion,
		ContinuationChecks:   continuationChecks,
		CurrentCeiling:       currentCeiling,
		NextRuntimeHooks:     nextRuntimeHooks,
	}, nil
}

func BuildValidationContinuationPolicyGate(config ValidationContinuationPolicyGateConfig) (ValidationContinuationPolicyGateDocument, int, error) {
	_, goRoot, err := normalizeRoots(config.RepoRoot, config.GoRoot)
	if err != nil {
		return ValidationContinuationPolicyGateDocument{}, 0, err
	}
	scorecardPath := defaultString(config.ScorecardPath, DefaultValidationContinuationScorecard)
	maxLatestAgeHours := config.MaxLatestAgeHours
	if maxLatestAgeHours == 0 {
		maxLatestAgeHours = 72
	}
	minRecentBundles := config.MinRecentBundles
	if minRecentBundles == 0 {
		minRecentBundles = 2
	}
	requireRepeatedLaneCoverage := true
	if !config.RequireRepeatedLaneCoverage {
		requireRepeatedLaneCoverage = false
	}
	mode, err := normalizeEnforcementMode(config.EnforcementMode, config.LegacyEnforce)
	if err != nil {
		return ValidationContinuationPolicyGateDocument{}, 0, err
	}

	var scorecard ValidationContinuationScorecardDocument
	if err := readJSON(resolveGoRelative(goRoot, scorecardPath), &scorecard); err != nil {
		return ValidationContinuationPolicyGateDocument{}, 0, err
	}

	summary := scorecard.Summary
	sharedQueue := structToMap(scorecard.SharedQueueCompanion)
	checks := []ValidationContinuationCheck{
		check("latest_bundle_age_within_threshold", toFloat(summary["latest_bundle_age_hours"]) <= maxLatestAgeHours, fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], maxLatestAgeHours)),
		check("recent_bundle_count_meets_floor", toInt(summary["recent_bundle_count"]) >= minRecentBundles, fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], minRecentBundles)),
		check("latest_bundle_all_executor_tracks_succeeded", toBool(summary["latest_all_executor_tracks_succeeded"]), fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"])),
		check("recent_bundle_chain_has_no_failures", toBool(summary["recent_bundle_chain_has_no_failures"]), fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"])),
		check("shared_queue_companion_available", toBool(sharedQueue["available"]), fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"])),
		check("repeated_lane_coverage_meets_policy", !requireRepeatedLaneCoverage || toBool(summary["all_executor_tracks_have_repeated_recent_coverage"]), fmt.Sprintf("require_repeated_lane_coverage=%t actual=%v", requireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"])),
	}
	failingChecks := []string{}
	for _, item := range checks {
		if !item.Passed {
			failingChecks = append(failingChecks, item.Name)
		}
	}
	recommendation := "go"
	status := defaultContinuationPolicyGateGoStatus
	if len(failingChecks) > 0 {
		recommendation = "hold"
		status = defaultContinuationPolicyGateHoldStatus
	}
	enforcement := buildEnforcementSummary(recommendation, mode)
	nextActions := buildNextActions(failingChecks)

	document := ValidationContinuationPolicyGateDocument{
		GeneratedAt:    utcISO(time.Now().UTC()),
		Ticket:         defaultContinuationPolicyGateTicket,
		Title:          defaultContinuationPolicyGateTitle,
		Status:         status,
		Recommendation: recommendation,
		EvidenceInputs: map[string]any{
			"scorecard_path": scorecardPath,
			"generator":      "go run ./cmd/bigclawctl migration validation-continuation-policy-gate",
		},
		PolicyInputs: map[string]any{
			"max_latest_age_hours":           maxLatestAgeHours,
			"min_recent_bundles":             minRecentBundles,
			"require_repeated_lane_coverage": requireRepeatedLaneCoverage,
		},
		Enforcement: enforcement,
		Summary: map[string]any{
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
			"passing_check_count":                               len(checks) - len(failingChecks),
			"failing_check_count":                               len(failingChecks),
		},
		PolicyChecks:  checks,
		FailingChecks: failingChecks,
		ReviewerPath: map[string]any{
			"index_path":  defaultValidationIndexReviewerPath,
			"digest_path": defaultValidationDigestReviewerPath,
		},
		SharedQueueCompanion: sharedQueue,
		NextActions:          nextActions,
	}
	return document, enforcement["exit_code"].(int), nil
}

func WriteJSON(path string, payload any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func normalizeRoots(repoRoot, goRoot string) (string, string, error) {
	if strings.TrimSpace(goRoot) == "" {
		goRoot = "."
	}
	absGoRoot, err := filepath.Abs(goRoot)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(repoRoot) == "" {
		repoRoot = filepath.Dir(absGoRoot)
	}
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", err
	}
	return absRepoRoot, absGoRoot, nil
}

func resolveGoRelative(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func resolveEvidencePath(repoRoot, goRoot, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	candidates := []string{filepath.Join(repoRoot, path)}
	if !strings.HasPrefix(filepath.ToSlash(path), "bigclaw-go/") {
		candidates = append(candidates, filepath.Join(goRoot, path))
	}
	for _, candidate := range candidates {
		if pathExists(candidate) {
			return candidate, nil
		}
	}
	return candidates[0], nil
}

func buildLaneScorecard(runs []validationRunSummaryDocument, lane string) ValidationContinuationLaneScorecard {
	statuses := make([]string, 0, len(runs))
	enabledRuns := 0
	succeededRuns := 0
	for _, run := range runs {
		section := laneSection(run, lane)
		status := "disabled"
		if section.Enabled {
			status = defaultString(section.Status, "missing")
			enabledRuns++
			if status == "succeeded" {
				succeededRuns++
			}
		}
		statuses = append(statuses, status)
	}
	var latest validationRunLane
	if len(runs) > 0 {
		latest = laneSection(runs[0], lane)
	}
	return ValidationContinuationLaneScorecard{
		Lane:                   lane,
		LatestEnabled:          latest.Enabled,
		LatestStatus:           laneStatus(runsFirst(runs), lane),
		RecentStatuses:         statuses,
		EnabledRuns:            enabledRuns,
		SucceededRuns:          succeededRuns,
		ConsecutiveSuccesses:   consecutiveSuccesses(statuses),
		AllRecentRunsSucceeded: enabledRuns > 0 && enabledRuns == succeededRuns,
	}
}

func laneSection(run validationRunSummaryDocument, lane string) validationRunLane {
	switch lane {
	case "local":
		return run.Local
	case "kubernetes":
		return run.Kubernetes
	case "ray":
		return run.Ray
	default:
		return validationRunLane{}
	}
}

func laneStatus(run validationRunSummaryDocument, lane string) string {
	section := laneSection(run, lane)
	if section.Enabled {
		return defaultString(section.Status, "missing")
	}
	if section.Status != "" {
		return section.Status
	}
	return "missing"
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

func runsFirst(runs []validationRunSummaryDocument) validationRunSummaryDocument {
	if len(runs) == 0 {
		return validationRunSummaryDocument{}
	}
	return runs[0]
}

func collectRunStatuses(runs []validationRunSummaryDocument) []string {
	out := make([]string, 0, len(runs))
	for _, run := range runs {
		out = append(out, defaultString(run.Status, "unknown"))
	}
	return out
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

func buildEnforcementSummary(recommendation, mode string) map[string]any {
	if recommendation == "go" {
		return map[string]any{"mode": mode, "outcome": "pass", "exit_code": 0}
	}
	switch mode {
	case "review":
		return map[string]any{"mode": mode, "outcome": "review-only", "exit_code": 0}
	case "hold":
		return map[string]any{"mode": mode, "outcome": "hold", "exit_code": 2}
	default:
		return map[string]any{"mode": mode, "outcome": "fail", "exit_code": 1}
	}
}

func buildNextActions(failingChecks []string) []string {
	if len(failingChecks) == 0 {
		return []string{"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions"}
	}
	actions := []string{}
	for _, failing := range failingChecks {
		switch failing {
		case "latest_bundle_age_within_threshold":
			actions = append(actions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
		case "recent_bundle_count_meets_floor":
			actions = append(actions, "export additional validation bundles so the continuation window spans multiple indexed runs")
		case "shared_queue_companion_available":
			actions = append(actions, "rerun `cd bigclaw-go && python3 scripts/e2e/multi_node_shared_queue.py --report-path docs/reports/multi-node-shared-queue-report.json`")
		case "repeated_lane_coverage_meets_policy":
			actions = append(actions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
		}
	}
	if len(actions) == 0 {
		actions = append(actions, "review the continuation checks and refresh the indexed evidence window")
	}
	return actions
}

func check(name string, passed bool, detail string) ValidationContinuationCheck {
	return ValidationContinuationCheck{Name: name, Passed: passed, Detail: detail}
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func relpath(path, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
}

func utcISO(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func toInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func toFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func countAny(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		return typed
	case float64:
		return int(typed)
	case []any:
		return len(typed)
	default:
		return 0
	}
}

func structToMap(value any) map[string]any {
	body, _ := json.Marshal(value)
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	return out
}
