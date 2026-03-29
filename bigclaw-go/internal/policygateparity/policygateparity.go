package policygateparity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type Enforcement struct {
	Mode     string `json:"mode"`
	Outcome  string `json:"outcome"`
	ExitCode int    `json:"exit_code"`
}

type Report struct {
	Status         string         `json:"status"`
	Recommendation string         `json:"recommendation"`
	Summary        map[string]any `json:"summary"`
	FailingChecks  []string       `json:"failing_checks"`
	Enforcement    Enforcement    `json:"enforcement"`
}

func BuildReport(scorecardPath string, requireRepeatedLaneCoverage bool) (Report, error) {
	body, err := os.ReadFile(scorecardPath)
	if err != nil {
		return Report{}, err
	}
	var payload struct {
		Summary              map[string]any `json:"summary"`
		SharedQueueCompanion map[string]any `json:"shared_queue_companion"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Report{}, err
	}

	summary := payload.Summary
	sharedQueue := payload.SharedQueueCompanion
	checks := []Check{
		{
			Name:   "latest_bundle_age_within_threshold",
			Passed: toFloat(summary["latest_bundle_age_hours"]) <= 72.0,
			Detail: fmt.Sprintf("latest_bundle_age_hours=%v threshold=72", summary["latest_bundle_age_hours"]),
		},
		{
			Name:   "recent_bundle_count_meets_floor",
			Passed: toInt(summary["recent_bundle_count"]) >= 2,
			Detail: fmt.Sprintf("recent_bundle_count=%v floor=2", summary["recent_bundle_count"]),
		},
		{
			Name:   "latest_bundle_all_executor_tracks_succeeded",
			Passed: toBool(summary["latest_all_executor_tracks_succeeded"]),
			Detail: fmt.Sprintf("latest_all_executor_tracks_succeeded=%v", summary["latest_all_executor_tracks_succeeded"]),
		},
		{
			Name:   "recent_bundle_chain_has_no_failures",
			Passed: toBool(summary["recent_bundle_chain_has_no_failures"]),
			Detail: fmt.Sprintf("recent_bundle_chain_has_no_failures=%v", summary["recent_bundle_chain_has_no_failures"]),
		},
		{
			Name:   "shared_queue_companion_available",
			Passed: toBool(sharedQueue["available"]),
			Detail: fmt.Sprintf("cross_node_completions=%v", sharedQueue["cross_node_completions"]),
		},
		{
			Name:   "repeated_lane_coverage_meets_policy",
			Passed: !requireRepeatedLaneCoverage || toBool(summary["all_executor_tracks_have_repeated_recent_coverage"]),
			Detail: fmt.Sprintf("require_repeated_lane_coverage=%t actual=%v", requireRepeatedLaneCoverage, summary["all_executor_tracks_have_repeated_recent_coverage"]),
		},
	}
	failing := make([]string, 0)
	passingCount := 0
	for _, check := range checks {
		if check.Passed {
			passingCount++
			continue
		}
		failing = append(failing, check.Name)
	}
	recommendation := "go"
	status := "policy-go"
	if len(failing) > 0 {
		recommendation = "hold"
		status = "policy-hold"
	}

	return Report{
		Status:         status,
		Recommendation: recommendation,
		FailingChecks:  failing,
		Enforcement:    buildEnforcement(recommendation, "hold"),
		Summary: map[string]any{
			"latest_run_id":                                     summary["latest_run_id"],
			"latest_bundle_age_hours":                           summary["latest_bundle_age_hours"],
			"recent_bundle_count":                               summary["recent_bundle_count"],
			"latest_all_executor_tracks_succeeded":              summary["latest_all_executor_tracks_succeeded"],
			"recent_bundle_chain_has_no_failures":               summary["recent_bundle_chain_has_no_failures"],
			"all_executor_tracks_have_repeated_recent_coverage": summary["all_executor_tracks_have_repeated_recent_coverage"],
			"recommendation":                                    recommendation,
			"enforcement_mode":                                  "hold",
			"workflow_outcome":                                  buildEnforcement(recommendation, "hold").Outcome,
			"workflow_exit_code":                                buildEnforcement(recommendation, "hold").ExitCode,
			"passing_check_count":                               passingCount,
			"failing_check_count":                               len(failing),
		},
	}, nil
}

func LoadCheckedInReport(repoRoot string) (Report, error) {
	body, err := os.ReadFile(filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-policy-gate.json"))
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := json.Unmarshal(body, &report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func buildEnforcement(recommendation, mode string) Enforcement {
	if recommendation == "go" {
		return Enforcement{Mode: mode, Outcome: "pass", ExitCode: 0}
	}
	if mode == "review" {
		return Enforcement{Mode: mode, Outcome: "review-only", ExitCode: 0}
	}
	if mode == "hold" {
		return Enforcement{Mode: mode, Outcome: "hold", ExitCode: 2}
	}
	return Enforcement{Mode: mode, Outcome: "fail", ExitCode: 1}
}

func toBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func toInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
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
