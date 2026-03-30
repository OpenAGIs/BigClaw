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

type check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type enforcementSummary struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type report struct {
	GeneratedAt          string             `json:"generated_at"`
	Ticket               string             `json:"ticket"`
	Title                string             `json:"title"`
	Status               string             `json:"status"`
	Recommendation       string             `json:"recommendation"`
	EvidenceInputs       map[string]any     `json:"evidence_inputs"`
	PolicyInputs         map[string]any     `json:"policy_inputs"`
	Enforcement          enforcementSummary `json:"enforcement"`
	Summary              map[string]any     `json:"summary"`
	PolicyChecks         []check            `json:"policy_checks"`
	FailingChecks        []string           `json:"failing_checks"`
	ReviewerPath         map[string]any     `json:"reviewer_path"`
	SharedQueueCompanion map[string]any     `json:"shared_queue_companion"`
	NextActions          []string           `json:"next_actions"`
}

func main() {
	goRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	flags := flag.NewFlagSet("validation-bundle-continuation-policy-gate", flag.ExitOnError)
	scorecardPath := flags.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "validation bundle continuation scorecard path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "json output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72.0, "maximum latest bundle age in hours")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum recent bundle count")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated recent lane coverage")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "allow partial recent lane history")
	enforcementMode := flags.String("enforcement-mode", "", "enforcement mode: review, hold, or fail")
	legacyEnforce := flags.Bool("enforce", false, "legacy alias for fail enforcement")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := buildReport(repoRoot, buildOptions{
		ScorecardPath:               *scorecardPath,
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: *requireRepeatedLaneCoverage && !*allowPartialLaneHistory,
		EnforcementMode:             *enforcementMode,
		LegacyEnforceContinuation:   *legacyEnforce,
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
	os.Exit(rep.Enforcement.ExitCode)
}

type buildOptions struct {
	ScorecardPath               string
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforceContinuation   bool
}

func buildReport(repoRoot string, opts buildOptions) (report, error) {
	scorecardBody, err := os.ReadFile(resolveRepoPath(repoRoot, opts.ScorecardPath))
	if err != nil {
		return report{}, err
	}

	var scorecard map[string]any
	if err := json.Unmarshal(scorecardBody, &scorecard); err != nil {
		return report{}, err
	}
	summary := nestedMap(scorecard, "summary")
	sharedQueue := nestedMap(scorecard, "shared_queue_companion")
	normalizedMode, err := normalizeEnforcementMode(opts.EnforcementMode, opts.LegacyEnforceContinuation)
	if err != nil {
		return report{}, err
	}

	checks := []check{
		buildCheck(
			"latest_bundle_age_within_threshold",
			floatValue(summary["latest_bundle_age_hours"]) <= opts.MaxLatestAgeHours,
			fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary["latest_bundle_age_hours"], opts.MaxLatestAgeHours),
		),
		buildCheck(
			"recent_bundle_count_meets_floor",
			intValue(summary["recent_bundle_count"]) >= opts.MinRecentBundles,
			fmt.Sprintf("recent_bundle_count=%v floor=%d", summary["recent_bundle_count"], opts.MinRecentBundles),
		),
		buildCheck(
			"latest_bundle_all_executor_tracks_succeeded",
			boolValue(summary["latest_all_executor_tracks_succeeded"]),
			fmt.Sprintf("latest_all_executor_tracks_succeeded=%s", pyBool(boolValue(summary["latest_all_executor_tracks_succeeded"]))),
		),
		buildCheck(
			"recent_bundle_chain_has_no_failures",
			boolValue(summary["recent_bundle_chain_has_no_failures"]),
			fmt.Sprintf("recent_bundle_chain_has_no_failures=%s", pyBool(boolValue(summary["recent_bundle_chain_has_no_failures"]))),
		),
		buildCheck(
			"shared_queue_companion_available",
			boolValue(sharedQueue["available"]),
			fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"]),
		),
		buildCheck(
			"repeated_lane_coverage_meets_policy",
			!opts.RequireRepeatedLaneCoverage || boolValue(summary["all_executor_tracks_have_repeated_recent_coverage"]),
			fmt.Sprintf(
				"require_repeated_lane_coverage=%s actual=%s",
				pyBool(opts.RequireRepeatedLaneCoverage),
				pyBool(boolValue(summary["all_executor_tracks_have_repeated_recent_coverage"])),
			),
		),
	}

	failingChecks := make([]string, 0, len(checks))
	passingCheckCount := 0
	for _, item := range checks {
		if item.Passed {
			passingCheckCount++
			continue
		}
		failingChecks = append(failingChecks, item.Name)
	}

	recommendation := "go"
	if len(failingChecks) > 0 {
		recommendation = "hold"
	}
	enforcement := buildEnforcementSummary(recommendation, normalizedMode)

	nextActions := make([]string, 0, 4)
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

	return report{
		GeneratedAt:    utcISO(time.Now().UTC()),
		Ticket:         "OPE-262",
		Title:          "Validation workflow continuation gate",
		Status:         map[bool]string{true: "policy-go", false: "policy-hold"}[recommendation == "go"],
		Recommendation: recommendation,
		EvidenceInputs: map[string]any{
			"scorecard_path":   opts.ScorecardPath,
			"generator_script": "scripts/e2e/validation-bundle-continuation-policy-gate",
		},
		PolicyInputs: map[string]any{
			"max_latest_age_hours":           opts.MaxLatestAgeHours,
			"min_recent_bundles":             opts.MinRecentBundles,
			"require_repeated_lane_coverage": opts.RequireRepeatedLaneCoverage,
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
			"enforcement_mode":                                  enforcement.Mode,
			"workflow_outcome":                                  enforcement.Outcome,
			"workflow_exit_code":                                enforcement.ExitCode,
			"passing_check_count":                               passingCheckCount,
			"failing_check_count":                               len(failingChecks),
		},
		PolicyChecks:         checks,
		FailingChecks:        failingChecks,
		ReviewerPath:         map[string]any{"index_path": "docs/reports/live-validation-index.md", "digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		SharedQueueCompanion: sharedQueue,
		NextActions:          nextActions,
	}, nil
}

func nestedMap(input map[string]any, key string) map[string]any {
	value, ok := input[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func buildCheck(name string, passed bool, detail string) check {
	return check{Name: name, Passed: passed, Detail: detail}
}

func normalizeEnforcementMode(enforcementMode string, legacyEnforceContinuation bool) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(enforcementMode))
	if mode == "" {
		if legacyEnforceContinuation {
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

func buildEnforcementSummary(recommendation, enforcementMode string) enforcementSummary {
	if recommendation == "go" {
		return enforcementSummary{Mode: enforcementMode, Outcome: "pass", ExitCode: 0}
	}
	if enforcementMode == "review" {
		return enforcementSummary{Mode: enforcementMode, Outcome: "review-only", ExitCode: 0}
	}
	if enforcementMode == "hold" {
		return enforcementSummary{Mode: enforcementMode, Outcome: "hold", ExitCode: 2}
	}
	return enforcementSummary{Mode: enforcementMode, Outcome: "fail", ExitCode: 1}
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(repoRoot, path)
}

func utcISO(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339Nano)
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		return 0
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}
