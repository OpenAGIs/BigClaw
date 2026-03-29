package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const continuationPolicyGateScriptPath = "scripts/e2e/validation_bundle_continuation_policy_gate.go"

type continuationPolicyCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type continuationPolicyEnforcement struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type continuationPolicySummary struct {
	LatestRunID                               any    `json:"latest_run_id"`
	LatestBundleAgeHours                      any    `json:"latest_bundle_age_hours"`
	RecentBundleCount                         any    `json:"recent_bundle_count"`
	LatestAllExecutorTracksSucceeded          any    `json:"latest_all_executor_tracks_succeeded"`
	RecentBundleChainHasNoFailures            any    `json:"recent_bundle_chain_has_no_failures"`
	AllExecutorTracksHaveRepeatedRecentCovera any    `json:"all_executor_tracks_have_repeated_recent_coverage"`
	Recommendation                            string `json:"recommendation"`
	EnforcementMode                           string `json:"enforcement_mode"`
	WorkflowOutcome                           string `json:"workflow_outcome"`
	WorkflowExitCode                          int    `json:"workflow_exit_code"`
	PassingCheckCount                         int    `json:"passing_check_count"`
	FailingCheckCount                         int    `json:"failing_check_count"`
}

type continuationPolicyReviewerPath struct {
	IndexPath  string `json:"index_path"`
	DigestPath string `json:"digest_path"`
}

type continuationPolicyReport struct {
	GeneratedAt          string                         `json:"generated_at"`
	Ticket               string                         `json:"ticket"`
	Title                string                         `json:"title"`
	Status               string                         `json:"status"`
	Recommendation       string                         `json:"recommendation"`
	EvidenceInputs       map[string]any                 `json:"evidence_inputs"`
	PolicyInputs         map[string]any                 `json:"policy_inputs"`
	Enforcement          continuationPolicyEnforcement  `json:"enforcement"`
	Summary              continuationPolicySummary      `json:"summary"`
	PolicyChecks         []continuationPolicyCheck      `json:"policy_checks"`
	FailingChecks        []string                       `json:"failing_checks"`
	ReviewerPath         continuationPolicyReviewerPath `json:"reviewer_path"`
	SharedQueueCompanion map[string]any                 `json:"shared_queue_companion"`
	NextActions          []string                       `json:"next_actions"`
}

type continuationScorecardInput struct {
	Summary              map[string]any `json:"summary"`
	SharedQueueCompanion map[string]any `json:"shared_queue_companion"`
}

func main() {
	scorecardPath := flag.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard input path")
	outputPath := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flag.Float64("max-latest-age-hours", 72.0, "maximum age in hours for the latest bundle")
	minRecentBundles := flag.Int("min-recent-bundles", 2, "minimum recent bundle count")
	requireRepeatedLaneCoverage := flag.Bool("require-repeated-lane-coverage", true, "require repeated recent lane coverage")
	allowPartialLaneHistory := flag.Bool("allow-partial-lane-history", false, "allow incomplete recent lane history")
	enforcementMode := flag.String("enforcement-mode", "", "enforcement mode: review, hold, fail")
	legacyEnforce := flag.Bool("enforce", false, "legacy alias for fail enforcement")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := repoRootFromScriptPath(scriptFilePath())
	if err != nil {
		panic(err)
	}
	report, exitCode, err := buildContinuationPolicyReport(
		repoRoot,
		continuationPolicyBuildOptions{
			ScorecardPath:                 *scorecardPath,
			MaxLatestAgeHours:             *maxLatestAgeHours,
			MinRecentBundles:              *minRecentBundles,
			RequireRepeatedLaneCoverage:   *requireRepeatedLaneCoverage && !*allowPartialLaneHistory,
			EnforcementMode:               *enforcementMode,
			LegacyEnforceContinuationGate: *legacyEnforce,
			GeneratedAt:                   time.Now().UTC(),
		},
	)
	if err != nil {
		panic(err)
	}
	if err := writeContinuationPolicyReport(repoRoot, *outputPath, report); err != nil {
		panic(err)
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(body))
	}
	os.Exit(exitCode)
}

type continuationPolicyBuildOptions struct {
	ScorecardPath                 string
	MaxLatestAgeHours             float64
	MinRecentBundles              int
	RequireRepeatedLaneCoverage   bool
	EnforcementMode               string
	LegacyEnforceContinuationGate bool
	GeneratedAt                   time.Time
}

func buildContinuationPolicyReport(repoRoot string, opts continuationPolicyBuildOptions) (continuationPolicyReport, int, error) {
	scorecard, err := loadContinuationScorecard(resolveRepoPath(repoRoot, opts.ScorecardPath))
	if err != nil {
		return continuationPolicyReport{}, 0, err
	}

	normalizedMode, err := normalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuationGate)
	if err != nil {
		return continuationPolicyReport{}, 0, err
	}

	summary := scorecard.Summary
	sharedQueue := scorecard.SharedQueueCompanion
	checks := []continuationPolicyCheck{
		buildCheck(
			"latest_bundle_age_within_threshold",
			asFloat(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours,
			fmt.Sprintf("latest_bundle_age_hours=%s threshold=%s", pyString(summary["latest_bundle_age_hours"]), pyString(opts.MaxLatestAgeHours)),
		),
		buildCheck(
			"recent_bundle_count_meets_floor",
			asInt(summary["recent_bundle_count"]) >= opts.MinRecentBundles,
			fmt.Sprintf("recent_bundle_count=%s floor=%d", pyString(summary["recent_bundle_count"]), opts.MinRecentBundles),
		),
		buildCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			asBool(summary["latest_all_executor_tracks_succeeded"]),
			fmt.Sprintf("latest_all_executor_tracks_succeeded=%s", pyString(summary["latest_all_executor_tracks_succeeded"])),
		),
		buildCheck(
			"recent_bundle_chain_has_no_failures",
			asBool(summary["recent_bundle_chain_has_no_failures"]),
			fmt.Sprintf("recent_bundle_chain_has_no_failures=%s", pyString(summary["recent_bundle_chain_has_no_failures"])),
		),
		buildCheck(
			"shared_queue_companion_available",
			asBool(sharedQueue["available"]),
			fmt.Sprintf("cross_node_completions=%s", pyString(sharedQueue["cross_node_completions"])),
		),
		buildCheck(
			"repeated_lane_coverage_meets_policy",
			!opts.RequireRepeatedLaneCoverage || asBool(summary["all_executor_tracks_have_repeated_recent_coverage"]),
			fmt.Sprintf(
				"require_repeated_lane_coverage=%s actual=%s",
				pyString(opts.RequireRepeatedLaneCoverage),
				pyString(summary["all_executor_tracks_have_repeated_recent_coverage"]),
			),
		),
	}

	failingChecks := make([]string, 0)
	for _, item := range checks {
		if !item.Passed {
			failingChecks = append(failingChecks, item.Name)
		}
	}

	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildEnforcementSummary(recommendation, normalizedMode)
	nextActions := buildNextActions(failingChecks)
	passingChecks := 0
	for _, item := range checks {
		if item.Passed {
			passingChecks++
		}
	}

	report := continuationPolicyReport{
		GeneratedAt:          utcISO(opts.GeneratedAt),
		Ticket:               "OPE-262",
		Title:                "Validation workflow continuation gate",
		Status:               map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		Recommendation:       recommendation,
		EvidenceInputs:       map[string]any{"scorecard_path": opts.ScorecardPath, "generator_script": continuationPolicyGateScriptPath},
		PolicyInputs:         map[string]any{"max_latest_age_hours": opts.MaxLatestAgeHours, "min_recent_bundles": opts.MinRecentBundles, "require_repeated_lane_coverage": opts.RequireRepeatedLaneCoverage},
		Enforcement:          enforcement,
		PolicyChecks:         checks,
		FailingChecks:        failingChecks,
		ReviewerPath:         continuationPolicyReviewerPath{IndexPath: "docs/reports/live-validation-index.md", DigestPath: "docs/reports/validation-bundle-continuation-digest.md"},
		SharedQueueCompanion: sharedQueue,
		NextActions:          nextActions,
	}
	report.Summary = continuationPolicySummary{
		LatestRunID:                               summary["latest_run_id"],
		LatestBundleAgeHours:                      summary["latest_bundle_age_hours"],
		RecentBundleCount:                         summary["recent_bundle_count"],
		LatestAllExecutorTracksSucceeded:          summary["latest_all_executor_tracks_succeeded"],
		RecentBundleChainHasNoFailures:            summary["recent_bundle_chain_has_no_failures"],
		AllExecutorTracksHaveRepeatedRecentCovera: summary["all_executor_tracks_have_repeated_recent_coverage"],
		Recommendation:                            recommendation,
		EnforcementMode:                           enforcement.Mode,
		WorkflowOutcome:                           enforcement.Outcome,
		WorkflowExitCode:                          enforcement.ExitCode,
		PassingCheckCount:                         passingChecks,
		FailingCheckCount:                         len(failingChecks),
	}

	return report, enforcement.ExitCode, nil
}

func writeContinuationPolicyReport(repoRoot, outputPath string, report continuationPolicyReport) error {
	targetPath := resolveRepoPath(repoRoot, outputPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, append(body, '\n'), 0o644)
}

func loadContinuationScorecard(path string) (continuationScorecardInput, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return continuationScorecardInput{}, err
	}
	var scorecard continuationScorecardInput
	if err := json.Unmarshal(body, &scorecard); err != nil {
		return continuationScorecardInput{}, err
	}
	if scorecard.Summary == nil {
		scorecard.Summary = map[string]any{}
	}
	if scorecard.SharedQueueCompanion == nil {
		scorecard.SharedQueueCompanion = map[string]any{}
	}
	return scorecard, nil
}

func buildCheck(name string, passed bool, detail string) continuationPolicyCheck {
	return continuationPolicyCheck{Name: name, Passed: passed, Detail: detail}
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

func buildEnforcementSummary(recommendation, mode string) continuationPolicyEnforcement {
	if recommendation == "go" {
		return continuationPolicyEnforcement{Mode: mode, Outcome: "pass", ExitCode: 0}
	}
	if mode == "review" {
		return continuationPolicyEnforcement{Mode: mode, Outcome: "review-only", ExitCode: 0}
	}
	if mode == "hold" {
		return continuationPolicyEnforcement{Mode: mode, Outcome: "hold", ExitCode: 2}
	}
	return continuationPolicyEnforcement{Mode: mode, Outcome: "fail", ExitCode: 1}
}

func buildNextActions(failingChecks []string) []string {
	nextActions := make([]string, 0)
	for _, check := range failingChecks {
		switch check {
		case "latest_bundle_age_within_threshold":
			nextActions = append(nextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
		case "recent_bundle_count_meets_floor":
			nextActions = append(nextActions, "export additional validation bundles so the continuation window spans multiple indexed runs")
		case "shared_queue_companion_available":
			nextActions = append(nextActions, "rerun `go run ./scripts/e2e/multi_node_shared_queue.go --report-path docs/reports/multi-node-shared-queue-report.json`")
		case "repeated_lane_coverage_meets_policy":
			nextActions = append(nextActions, "refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage")
		}
	}
	if len(nextActions) == 0 {
		return []string{"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions"}
	}
	return nextActions
}

func resolveRepoPath(repoRoot, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(repoRoot, target)
}

func repoRootFromScriptPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty script path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), "../../..")), nil
}

func scriptFilePath() string {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return path
}

func utcISO(moment time.Time) string {
	if moment.IsZero() {
		moment = time.Now().UTC()
	}
	return moment.UTC().Format(time.RFC3339Nano)
}

func asBool(value any) bool {
	cast, ok := value.(bool)
	return ok && cast
}

func asFloat(value any) float64 {
	switch cast := value.(type) {
	case float64:
		return cast
	case float32:
		return float64(cast)
	case int:
		return float64(cast)
	case int64:
		return float64(cast)
	default:
		return 0
	}
}

func asInt(value any) int {
	switch cast := value.(type) {
	case float64:
		return int(cast)
	case float32:
		return int(cast)
	case int:
		return cast
	case int64:
		return int(cast)
	default:
		return 0
	}
}

func pyString(value any) string {
	switch cast := value.(type) {
	case nil:
		return "None"
	case bool:
		if cast {
			return "True"
		}
		return "False"
	case float64:
		return strconvFloat(cast)
	case float32:
		return strconvFloat(float64(cast))
	case int:
		return fmt.Sprintf("%d", cast)
	case int64:
		return fmt.Sprintf("%d", cast)
	case string:
		return cast
	default:
		return fmt.Sprintf("%v", value)
	}
}

func strconvFloat(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.1f", value)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.12f", value), "0"), ".")
}
