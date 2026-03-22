package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	PolicyChecks []validationBundleContinuationPolicyCheck `json:"policy_checks"`
	ReviewerPath struct {
		IndexPath  string `json:"index_path"`
		DigestPath string `json:"digest_path"`
	} `json:"reviewer_path"`
	SharedQueueCompanion struct {
		Available            bool `json:"available"`
		CrossNodeCompletions int  `json:"cross_node_completions"`
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
