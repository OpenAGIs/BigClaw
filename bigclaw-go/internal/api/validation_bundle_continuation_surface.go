package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	validationBundleContinuationGatePath      = "docs/reports/validation-bundle-continuation-policy-gate.json"
	validationBundleContinuationScorecardPath = "docs/reports/validation-bundle-continuation-scorecard.json"
)

type validationBundleContinuationGateSurface struct {
	ReportPath     string                                    `json:"report_path"`
	ScorecardPath  string                                    `json:"scorecard_path,omitempty"`
	DigestPath     string                                    `json:"digest_path,omitempty"`
	GeneratedAt    string                                    `json:"generated_at,omitempty"`
	Ticket         string                                    `json:"ticket,omitempty"`
	Title          string                                    `json:"title,omitempty"`
	Status         string                                    `json:"status,omitempty"`
	Recommendation string                                    `json:"recommendation,omitempty"`
	ReviewerLinks  []string                                  `json:"reviewer_links,omitempty"`
	Summary        validationBundleContinuationGateSummary   `json:"summary"`
	ExecutorLanes  []validationBundleContinuationLaneSummary `json:"executor_lanes,omitempty"`
	PolicyChecks   []validationBundleContinuationPolicyCheck `json:"policy_checks,omitempty"`
	CurrentCeiling []string                                  `json:"current_ceiling,omitempty"`
	NextActions    []string                                  `json:"next_actions,omitempty"`
	Error          string                                    `json:"error,omitempty"`
}

type validationBundleContinuationLaneSummary struct {
	Lane                   string `json:"lane"`
	LatestEnabled          bool   `json:"latest_enabled"`
	LatestStatus           string `json:"latest_status"`
	EnabledRuns            int    `json:"enabled_runs"`
	SucceededRuns          int    `json:"succeeded_runs"`
	ConsecutiveSuccesses   int    `json:"consecutive_successes"`
	AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
}

type validationBundleContinuationGateSummary struct {
	LatestRunID                                 string  `json:"latest_run_id,omitempty"`
	LatestBundleAgeHours                        float64 `json:"latest_bundle_age_hours,omitempty"`
	RecentBundleCount                           int     `json:"recent_bundle_count"`
	LatestAllExecutorTracksSucceeded            bool    `json:"latest_all_executor_tracks_succeeded"`
	RecentBundleChainHasNoFailures              bool    `json:"recent_bundle_chain_has_no_failures"`
	AllExecutorTracksHaveRepeatedRecentCoverage bool    `json:"all_executor_tracks_have_repeated_recent_coverage"`
	SharedQueueCompanionAvailable               bool    `json:"shared_queue_companion_available"`
	CrossNodeCompletions                        int     `json:"cross_node_completions"`
	Recommendation                              string  `json:"recommendation,omitempty"`
	EnforcementMode                             string  `json:"enforcement_mode,omitempty"`
	WorkflowOutcome                             string  `json:"workflow_outcome,omitempty"`
	WorkflowExitCode                            int     `json:"workflow_exit_code,omitempty"`
	PassingCheckCount                           int     `json:"passing_check_count"`
	FailingCheckCount                           int     `json:"failing_check_count"`
}

type validationBundleContinuationPolicyCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type validationBundleContinuationGateDocument struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	Recommendation string `json:"recommendation"`
	Summary        struct {
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
	PolicyChecks  []validationBundleContinuationPolicyCheck `json:"policy_checks"`
	FailingChecks []string                                  `json:"failing_checks,omitempty"`
	ReviewerPath  struct {
		IndexPath  string `json:"index_path"`
		DigestPath string `json:"digest_path"`
	} `json:"reviewer_path"`
	SharedQueueCompanion struct {
		Available               bool   `json:"available"`
		ReportPath              string `json:"report_path,omitempty"`
		SummaryPath             string `json:"summary_path,omitempty"`
		BundleReportPath        string `json:"bundle_report_path,omitempty"`
		BundleSummaryPath       string `json:"bundle_summary_path,omitempty"`
		CrossNodeCompletions    int    `json:"cross_node_completions"`
		DuplicateCompletedTasks int    `json:"duplicate_completed_tasks,omitempty"`
		DuplicateStartedTasks   int    `json:"duplicate_started_tasks,omitempty"`
		Mode                    string `json:"mode,omitempty"`
	} `json:"shared_queue_companion"`
	NextActions []string `json:"next_actions"`
}

type validationBundleContinuationScorecardDocument struct {
	CurrentCeiling   []string `json:"current_ceiling"`
	NextRuntimeHooks []string `json:"next_runtime_hooks"`
	ExecutorLanes    []struct {
		Lane                   string `json:"lane"`
		LatestEnabled          bool   `json:"latest_enabled"`
		LatestStatus           string `json:"latest_status"`
		EnabledRuns            int    `json:"enabled_runs"`
		SucceededRuns          int    `json:"succeeded_runs"`
		ConsecutiveSuccesses   int    `json:"consecutive_successes"`
		AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
	} `json:"executor_lanes"`
}

func validationBundleContinuationGatePayload() validationBundleContinuationGateSurface {
	surface := validationBundleContinuationGateSurface{
		ReportPath:    validationBundleContinuationGatePath,
		ScorecardPath: validationBundleContinuationScorecardPath,
	}
	gatePath := resolveRepoRelativePath(validationBundleContinuationGatePath)
	if gatePath == "" {
		surface.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(gatePath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	var gate validationBundleContinuationGateDocument
	if err := json.Unmarshal(contents, &gate); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", validationBundleContinuationGatePath, err)
		return surface
	}
	scorecardPath := resolveRepoRelativePath(validationBundleContinuationScorecardPath)
	if scorecardPath == "" {
		surface.Status = "unavailable"
		surface.Error = "scorecard path could not be resolved"
		return surface
	}
	scorecardContents, err := os.ReadFile(scorecardPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	var scorecard validationBundleContinuationScorecardDocument
	if err := json.Unmarshal(scorecardContents, &scorecard); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", validationBundleContinuationScorecardPath, err)
		return surface
	}

	surface.GeneratedAt = gate.GeneratedAt
	surface.Ticket = gate.Ticket
	surface.Title = gate.Title
	surface.Status = gate.Status
	surface.Recommendation = gate.Recommendation
	surface.DigestPath = normalizeReportPath(gate.ReviewerPath.DigestPath)
	surface.ReviewerLinks = compactPaths(
		gate.ReviewerPath.IndexPath,
		gate.ReviewerPath.DigestPath,
		validationBundleContinuationScorecardPath,
	)
	surface.Summary = validationBundleContinuationGateSummary{
		LatestRunID:                                 gate.Summary.LatestRunID,
		LatestBundleAgeHours:                        gate.Summary.LatestBundleAgeHours,
		RecentBundleCount:                           gate.Summary.RecentBundleCount,
		LatestAllExecutorTracksSucceeded:            gate.Summary.LatestAllExecutorTracksSucceeded,
		RecentBundleChainHasNoFailures:              gate.Summary.RecentBundleChainHasNoFailures,
		AllExecutorTracksHaveRepeatedRecentCoverage: gate.Summary.AllExecutorTracksHaveRepeatedRecentCoverage,
		SharedQueueCompanionAvailable:               gate.SharedQueueCompanion.Available,
		CrossNodeCompletions:                        gate.SharedQueueCompanion.CrossNodeCompletions,
		Recommendation:                              gate.Summary.Recommendation,
		EnforcementMode:                             gate.Summary.EnforcementMode,
		WorkflowOutcome:                             gate.Summary.WorkflowOutcome,
		WorkflowExitCode:                            gate.Summary.WorkflowExitCode,
		PassingCheckCount:                           gate.Summary.PassingCheckCount,
		FailingCheckCount:                           gate.Summary.FailingCheckCount,
	}
	surface.PolicyChecks = append([]validationBundleContinuationPolicyCheck(nil), gate.PolicyChecks...)
	surface.ExecutorLanes = make([]validationBundleContinuationLaneSummary, 0, len(scorecard.ExecutorLanes))
	for _, lane := range scorecard.ExecutorLanes {
		surface.ExecutorLanes = append(surface.ExecutorLanes, validationBundleContinuationLaneSummary{
			Lane:                   lane.Lane,
			LatestEnabled:          lane.LatestEnabled,
			LatestStatus:           lane.LatestStatus,
			EnabledRuns:            lane.EnabledRuns,
			SucceededRuns:          lane.SucceededRuns,
			ConsecutiveSuccesses:   lane.ConsecutiveSuccesses,
			AllRecentRunsSucceeded: lane.AllRecentRunsSucceeded,
		})
	}
	surface.CurrentCeiling = append([]string(nil), scorecard.CurrentCeiling...)
	surface.NextActions = append([]string(nil), gate.NextActions...)
	surface.NextActions = append(surface.NextActions, scorecard.NextRuntimeHooks...)
	surface.NextActions = uniqueStrings(surface.NextActions)
	return surface
}

type validationBundleContinuationGateConfig struct {
	MaxLatestAgeHours           float64
	MinRecentBundles            int
	RequireRepeatedLaneCoverage bool
	EnforcementMode             string
	LegacyEnforce               bool
	Now                         func() time.Time
}

func buildValidationBundleContinuationGateReport(scorecard validationBundleContinuationGateDocument, config validationBundleContinuationGateConfig) validationBundleContinuationGateDocument {
	if config.MaxLatestAgeHours == 0 {
		config.MaxLatestAgeHours = 72.0
	}
	if config.MinRecentBundles == 0 {
		config.MinRecentBundles = 2
	}
	if !config.RequireRepeatedLaneCoverage {
		config.RequireRepeatedLaneCoverage = false
	}
	now := time.Now().UTC
	if config.Now != nil {
		now = func() time.Time { return config.Now().UTC() }
	}

	enforcementMode := normalizeValidationBundleContinuationEnforcementMode(config.EnforcementMode, config.LegacyEnforce)
	summary := scorecard.Summary
	sharedQueue := scorecard.SharedQueueCompanion
	checks := []validationBundleContinuationPolicyCheck{
		{
			Name:   "latest_bundle_age_within_threshold",
			Passed: summary.LatestBundleAgeHours <= config.MaxLatestAgeHours,
			Detail: fmt.Sprintf("latest_bundle_age_hours=%v threshold=%v", summary.LatestBundleAgeHours, config.MaxLatestAgeHours),
		},
		{
			Name:   "recent_bundle_count_meets_floor",
			Passed: summary.RecentBundleCount >= config.MinRecentBundles,
			Detail: fmt.Sprintf("recent_bundle_count=%d floor=%d", summary.RecentBundleCount, config.MinRecentBundles),
		},
		{
			Name:   "latest_bundle_all_executor_tracks_succeeded",
			Passed: summary.LatestAllExecutorTracksSucceeded,
			Detail: fmt.Sprintf("latest_all_executor_tracks_succeeded=%t", summary.LatestAllExecutorTracksSucceeded),
		},
		{
			Name:   "recent_bundle_chain_has_no_failures",
			Passed: summary.RecentBundleChainHasNoFailures,
			Detail: fmt.Sprintf("recent_bundle_chain_has_no_failures=%t", summary.RecentBundleChainHasNoFailures),
		},
		{
			Name:   "shared_queue_companion_available",
			Passed: sharedQueue.Available,
			Detail: fmt.Sprintf("cross_node_completions=%d", sharedQueue.CrossNodeCompletions),
		},
		{
			Name:   "repeated_lane_coverage_meets_policy",
			Passed: !config.RequireRepeatedLaneCoverage || summary.AllExecutorTracksHaveRepeatedRecentCoverage,
			Detail: fmt.Sprintf("require_repeated_lane_coverage=%t actual=%t", config.RequireRepeatedLaneCoverage, summary.AllExecutorTracksHaveRepeatedRecentCoverage),
		},
	}

	failingChecks := make([]string, 0, len(checks))
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
	enforcementOutcome, exitCode := buildValidationBundleContinuationEnforcementSummary(recommendation, enforcementMode)
	nextActions := validationBundleContinuationNextActions(failingChecks)

	report := validationBundleContinuationGateDocument{
		GeneratedAt:    now().Format(time.RFC3339Nano),
		Ticket:         "OPE-262",
		Title:          "Validation workflow continuation gate",
		Status:         "policy-go",
		Recommendation: recommendation,
		PolicyChecks:   checks,
		FailingChecks:  failingChecks,
		NextActions:    nextActions,
	}
	if recommendation != "go" {
		report.Status = "policy-hold"
	}
	report.Summary = struct {
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
	}{
		LatestRunID:                                 summary.LatestRunID,
		LatestBundleAgeHours:                        summary.LatestBundleAgeHours,
		RecentBundleCount:                           summary.RecentBundleCount,
		LatestAllExecutorTracksSucceeded:            summary.LatestAllExecutorTracksSucceeded,
		RecentBundleChainHasNoFailures:              summary.RecentBundleChainHasNoFailures,
		AllExecutorTracksHaveRepeatedRecentCoverage: summary.AllExecutorTracksHaveRepeatedRecentCoverage,
		Recommendation:                              recommendation,
		EnforcementMode:                             enforcementMode,
		WorkflowOutcome:                             enforcementOutcome,
		WorkflowExitCode:                            exitCode,
		PassingCheckCount:                           passingCount,
		FailingCheckCount:                           len(failingChecks),
	}
	report.ReviewerPath.IndexPath = "docs/reports/live-validation-index.md"
	report.ReviewerPath.DigestPath = "docs/reports/validation-bundle-continuation-digest.md"
	report.SharedQueueCompanion = sharedQueue
	return report
}

func normalizeValidationBundleContinuationEnforcementMode(mode string, legacyEnforce bool) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		if legacyEnforce {
			return "fail"
		}
		return "hold"
	}
	switch normalized {
	case "review", "hold", "fail":
		return normalized
	default:
		return "hold"
	}
}

func buildValidationBundleContinuationEnforcementSummary(recommendation, enforcementMode string) (string, int) {
	if recommendation == "go" {
		return "pass", 0
	}
	switch enforcementMode {
	case "review":
		return "review-only", 0
	case "fail":
		return "fail", 1
	default:
		return "hold", 2
	}
}

func validationBundleContinuationNextActions(failingChecks []string) []string {
	if len(failingChecks) == 0 {
		return []string{"set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions"}
	}
	nextActions := make([]string, 0, len(failingChecks))
	for _, check := range failingChecks {
		switch check {
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
	return nextActions
}

func loadValidationBundleContinuationGateDocument(path string) (validationBundleContinuationGateDocument, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return validationBundleContinuationGateDocument{}, err
	}
	var document validationBundleContinuationGateDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return validationBundleContinuationGateDocument{}, err
	}
	return document, nil
}

func loadValidationBundleContinuationScorecard(path string) (validationBundleContinuationGateDocument, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return validationBundleContinuationGateDocument{}, err
	}
	var document validationBundleContinuationGateDocument
	if err := json.Unmarshal(contents, &document); err != nil {
		return validationBundleContinuationGateDocument{}, err
	}
	if document.SharedQueueCompanion.ReportPath != "" {
		document.SharedQueueCompanion.ReportPath = strings.TrimPrefix(document.SharedQueueCompanion.ReportPath, "bigclaw-go/")
	}
	if document.SharedQueueCompanion.SummaryPath != "" {
		document.SharedQueueCompanion.SummaryPath = strings.TrimPrefix(document.SharedQueueCompanion.SummaryPath, "bigclaw-go/")
	}
	if document.SharedQueueCompanion.BundleReportPath != "" {
		document.SharedQueueCompanion.BundleReportPath = strings.TrimPrefix(document.SharedQueueCompanion.BundleReportPath, "bigclaw-go/")
	}
	if document.SharedQueueCompanion.BundleSummaryPath != "" {
		document.SharedQueueCompanion.BundleSummaryPath = strings.TrimPrefix(document.SharedQueueCompanion.BundleSummaryPath, "bigclaw-go/")
	}
	return document, nil
}

func validationBundleContinuationReportPaths() (string, string) {
	return filepath.Join("docs", "reports", "validation-bundle-continuation-policy-gate.json"), filepath.Join("docs", "reports", "validation-bundle-continuation-scorecard.json")
}

func compactPaths(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeReportPath(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeReportPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "bigclaw-go/")
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
