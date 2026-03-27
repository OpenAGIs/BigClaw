package continuationgate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type PolicyCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type EnforcementSummary struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type BuildOptions struct {
	RepoRoot                    string
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforceContinuation   bool
	Now                         time.Time
}

type scorecardPayload struct {
	Summary struct {
		LatestRunID                                 string  `json:"latest_run_id"`
		LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours"`
		RecentBundleCount                           int     `json:"recent_bundle_count"`
		LatestAllExecutorTracksSucceeded            bool    `json:"latest_all_executor_tracks_succeeded"`
		RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
		AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
	} `json:"summary"`
	SharedQueueCompanion map[string]any `json:"shared_queue_companion"`
}

func BuildReport(opts BuildOptions) (map[string]any, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		repoRoot = defaultRepoRoot()
	}
	scorecardPath := opts.ScorecardPath
	if scorecardPath == "" {
		scorecardPath = "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json"
	}
	maxLatestAgeHours := opts.MaxLatestAgeHours
	if maxLatestAgeHours == 0 {
		maxLatestAgeHours = 72
	}
	minRecentBundles := opts.MinRecentBundles
	if minRecentBundles == 0 {
		minRecentBundles = 2
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	body, err := os.ReadFile(resolveRepoPath(repoRoot, scorecardPath))
	if err != nil {
		return nil, err
	}
	var scorecard scorecardPayload
	if err := json.Unmarshal(body, &scorecard); err != nil {
		return nil, err
	}

	mode, err := normalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuation)
	if err != nil {
		return nil, err
	}

	sharedQueue := scorecard.SharedQueueCompanion
	checks := []PolicyCheck{
		buildCheck(
			"latest_bundle_age_within_threshold",
			scorecard.Summary.LatestBundleAgeHours <= maxLatestAgeHours,
			fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", scorecard.Summary.LatestBundleAgeHours, maxLatestAgeHours),
		),
		buildCheck(
			"recent_bundle_count_meets_floor",
			scorecard.Summary.RecentBundleCount >= minRecentBundles,
			fmt.Sprintf("recent_bundle_count=%d floor=%d", scorecard.Summary.RecentBundleCount, minRecentBundles),
		),
		buildCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			scorecard.Summary.LatestAllExecutorTracksSucceeded,
			fmt.Sprintf("latest_all_executor_tracks_succeeded=%t", scorecard.Summary.LatestAllExecutorTracksSucceeded),
		),
		buildCheck(
			"recent_bundle_chain_has_no_failures",
			scorecard.Summary.RecentBundleChainHasNoFailures,
			fmt.Sprintf("recent_bundle_chain_has_no_failures=%t", scorecard.Summary.RecentBundleChainHasNoFailures),
		),
		buildCheck(
			"shared_queue_companion_available",
			asBool(sharedQueue["available"]),
			fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"]),
		),
		buildCheck(
			"repeated_lane_coverage_meets_policy",
			!opts.RequireRepeatedLaneCoverage || scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage,
			fmt.Sprintf("require_repeated_lane_coverage=%t actual=%t", opts.RequireRepeatedLaneCoverage, scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage),
		),
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
	enforcement := buildEnforcementSummary(recommendation, mode)
	nextActions := buildNextActions(failingChecks)

	report := map[string]any{
		"generated_at":   now.Format(time.RFC3339),
		"ticket":         "OPE-262",
		"title":          "Validation workflow continuation gate",
		"status":         map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		"recommendation": recommendation,
		"evidence_inputs": map[string]any{
			"scorecard_path":   scorecardPath,
			"generator_script": "scripts/e2e/validation_bundle_continuation_policy_gate.py",
		},
		"policy_inputs": map[string]any{
			"max_latest_age_hours":           maxLatestAgeHours,
			"min_recent_bundles":             minRecentBundles,
			"require_repeated_lane_coverage": opts.RequireRepeatedLaneCoverage,
		},
		"enforcement": enforcement,
		"summary": map[string]any{
			"latest_run_id":                                     scorecard.Summary.LatestRunID,
			"latest_bundle_age_hours":                           scorecard.Summary.LatestBundleAgeHours,
			"recent_bundle_count":                               scorecard.Summary.RecentBundleCount,
			"latest_all_executor_tracks_succeeded":              scorecard.Summary.LatestAllExecutorTracksSucceeded,
			"recent_bundle_chain_has_no_failures":               scorecard.Summary.RecentBundleChainHasNoFailures,
			"all_executor_tracks_have_repeated_recent_coverage": scorecard.Summary.AllExecutorTracksHaveRepeatedRecentCoverage,
			"recommendation":                                    recommendation,
			"enforcement_mode":                                  enforcement.Mode,
			"workflow_outcome":                                  enforcement.Outcome,
			"workflow_exit_code":                                enforcement.ExitCode,
			"passing_check_count":                               passingCount,
			"failing_check_count":                               len(failingChecks),
		},
		"policy_checks":  checks,
		"failing_checks": failingChecks,
		"reviewer_path": map[string]any{
			"index_path":  "docs/reports/live-validation-index.md",
			"digest_path": "docs/reports/validation-bundle-continuation-digest.md",
		},
		"shared_queue_companion": sharedQueue,
		"next_actions":           nextActions,
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

func defaultRepoRoot() string {
	return filepath.Clean(filepath.Join(filepath.Dir(os.Args[0]), "..", "..", ".."))
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func buildCheck(name string, passed bool, detail string) PolicyCheck {
	return PolicyCheck{Name: name, Passed: passed, Detail: detail}
}

func normalizeEnforcementMode(mode string, legacyEnforce bool) (string, error) {
	switch mode {
	case "":
		if legacyEnforce {
			return "fail", nil
		}
		return "hold", nil
	case "review", "hold", "fail":
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported enforcement mode %q; expected one of review, hold, fail", mode)
	}
}

func buildEnforcementSummary(recommendation, mode string) EnforcementSummary {
	if recommendation == "go" {
		return EnforcementSummary{Mode: mode, Outcome: "pass", ExitCode: 0}
	}
	if mode == "review" {
		return EnforcementSummary{Mode: mode, Outcome: "review-only", ExitCode: 0}
	}
	if mode == "hold" {
		return EnforcementSummary{Mode: mode, Outcome: "hold", ExitCode: 2}
	}
	return EnforcementSummary{Mode: mode, Outcome: "fail", ExitCode: 1}
}

func buildNextActions(failingChecks []string) []string {
	nextActions := make([]string, 0)
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
	return nextActions
}

func asBool(value any) bool {
	flag, _ := value.(bool)
	return flag
}
