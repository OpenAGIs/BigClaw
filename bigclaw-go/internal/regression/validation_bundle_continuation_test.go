package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationBundleContinuationScorecardStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var report struct {
		Status  string `json:"status"`
		Summary struct {
			LatestRunID                           string `json:"latest_run_id"`
			RecentBundleCount                     int    `json:"recent_bundle_count"`
			LatestAllExecutorTracksSucceeded      bool   `json:"latest_all_executor_tracks_succeeded"`
			RecentBundleChainHasNoFailures        bool   `json:"recent_bundle_chain_has_no_failures"`
			AllExecutorTracksHaveRepeatedCoverage bool   `json:"all_executor_tracks_have_repeated_recent_coverage"`
		} `json:"summary"`
		SharedQueueCompanion struct {
			CrossNodeCompletions int    `json:"cross_node_completions"`
			DuplicateCompleted   int    `json:"duplicate_completed_tasks"`
			Mode                 string `json:"mode"`
			SummaryPath          string `json:"summary_path"`
		} `json:"shared_queue_companion"`
		ExecutorLanes []struct {
			Lane                 string `json:"lane"`
			LatestStatus         string `json:"latest_status"`
			ConsecutiveSuccesses int    `json:"consecutive_successes"`
			EnabledRuns          int    `json:"enabled_runs"`
			AllRecentRunsOK      bool   `json:"all_recent_runs_succeeded"`
		} `json:"executor_lanes"`
		ContinuationChecks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Detail string `json:"detail"`
		} `json:"continuation_checks"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-scorecard.json"), &report)

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("unexpected continuation scorecard status: %+v", report)
	}
	if report.Summary.LatestRunID != "20260316T140138Z" ||
		report.Summary.RecentBundleCount != 3 ||
		!report.Summary.LatestAllExecutorTracksSucceeded ||
		!report.Summary.RecentBundleChainHasNoFailures ||
		!report.Summary.AllExecutorTracksHaveRepeatedCoverage {
		t.Fatalf("unexpected continuation scorecard summary: %+v", report.Summary)
	}
	if report.SharedQueueCompanion.CrossNodeCompletions != 99 ||
		report.SharedQueueCompanion.DuplicateCompleted != 0 ||
		report.SharedQueueCompanion.Mode != "bundle-companion-summary" ||
		report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared-queue companion payload: %+v", report.SharedQueueCompanion)
	}
	if len(report.ExecutorLanes) != 3 ||
		report.ExecutorLanes[0].Lane != "local" ||
		report.ExecutorLanes[1].Lane != "kubernetes" ||
		report.ExecutorLanes[2].Lane != "ray" {
		t.Fatalf("unexpected executor lanes: %+v", report.ExecutorLanes)
	}
	for _, lane := range report.ExecutorLanes {
		if lane.LatestStatus != "succeeded" || !lane.AllRecentRunsOK {
			t.Fatalf("expected lane success, got %+v", lane)
		}
	}
	repeatedCoverage := findContinuationCheck(report.ContinuationChecks, "all_executor_tracks_have_repeated_recent_coverage")
	if repeatedCoverage == nil || !repeatedCoverage.Passed || !strings.Contains(repeatedCoverage.Detail, "'ray': 2") {
		t.Fatalf("unexpected repeated coverage check: %+v", repeatedCoverage)
	}
	manualBoundary := findContinuationCheck(report.ContinuationChecks, "continuation_surface_is_workflow_triggered")
	if manualBoundary == nil || !manualBoundary.Passed || !strings.Contains(manualBoundary.Detail, "workflow execution") {
		t.Fatalf("unexpected manual boundary check: %+v", manualBoundary)
	}
}

func TestValidationBundleContinuationPolicyGateStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var report struct {
		Status         string   `json:"status"`
		Recommendation string   `json:"recommendation"`
		FailingChecks  []string `json:"failing_checks"`
		Summary        struct {
			LatestRunID       string `json:"latest_run_id"`
			FailingCheckCount int    `json:"failing_check_count"`
		} `json:"summary"`
	}
	readJSONFile(t, filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &report)

	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("unexpected continuation policy gate outcome: %+v", report)
	}
	if report.Summary.LatestRunID != "20260316T140138Z" || report.Summary.FailingCheckCount != 0 {
		t.Fatalf("unexpected continuation policy gate summary: %+v", report.Summary)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("expected no failing checks, got %+v", report.FailingChecks)
	}
}

func findContinuationCheck(items []struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}, name string) *struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
} {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}
