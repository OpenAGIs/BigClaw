package regression

import (
	"encoding/json"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestValidationBundleContinuationScorecardSummarizesRecentBundleChain(t *testing.T) {
	report := runContinuationScorecardBuildReport(t)

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("unexpected continuation scorecard status: %+v", report)
	}
	if report.Summary.RecentBundleCount != 3 || report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("unexpected continuation scorecard summary: %+v", report.Summary)
	}
	if !report.Summary.LatestAllExecutorTracksSucceeded ||
		!report.Summary.RecentBundleChainHasNoFailures ||
		!report.Summary.AllExecutorTracksHaveRepeatedCoverage {
		t.Fatalf("expected passing continuation summary, got %+v", report.Summary)
	}
	if report.SharedQueueCompanion.CrossNodeCompletions != 99 ||
		report.SharedQueueCompanion.Mode != "bundle-companion-summary" ||
		report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue companion payload: %+v", report.SharedQueueCompanion)
	}
}

func TestValidationBundleContinuationScorecardMarksLaneSuccessAndManualBoundary(t *testing.T) {
	report := runContinuationScorecardBuildReport(t)

	if len(report.ExecutorLanes) != 3 {
		t.Fatalf("unexpected executor lane count: %+v", report.ExecutorLanes)
	}
	lanes := map[string]continuationScorecardLane{}
	for _, lane := range report.ExecutorLanes {
		lanes[lane.Lane] = lane
	}
	if _, ok := lanes["local"]; !ok {
		t.Fatalf("missing local lane: %+v", report.ExecutorLanes)
	}
	if _, ok := lanes["kubernetes"]; !ok {
		t.Fatalf("missing kubernetes lane: %+v", report.ExecutorLanes)
	}
	if _, ok := lanes["ray"]; !ok {
		t.Fatalf("missing ray lane: %+v", report.ExecutorLanes)
	}
	for _, lane := range lanes {
		if lane.LatestStatus != "succeeded" || !lane.AllRecentRunsSucceeded {
			t.Fatalf("expected lane success, got %+v", lane)
		}
	}
	if lanes["local"].ConsecutiveSuccesses != 3 || lanes["kubernetes"].ConsecutiveSuccesses != 3 {
		t.Fatalf("unexpected consecutive successes for stable lanes: %+v", lanes)
	}
	if lanes["ray"].ConsecutiveSuccesses != 2 || lanes["ray"].EnabledRuns != 2 {
		t.Fatalf("unexpected ray lane history: %+v", lanes["ray"])
	}

	repeatedCoverage := findContinuationCheckByName(t, report.ContinuationChecks, "all_executor_tracks_have_repeated_recent_coverage")
	if !repeatedCoverage.Passed || !strings.Contains(repeatedCoverage.Detail, "'ray': 2") {
		t.Fatalf("unexpected repeated coverage check: %+v", repeatedCoverage)
	}
	manualBoundary := findContinuationCheckByName(t, report.ContinuationChecks, "continuation_surface_is_workflow_triggered")
	if !manualBoundary.Passed || !strings.Contains(manualBoundary.Detail, "workflow execution") {
		t.Fatalf("unexpected manual boundary check: %+v", manualBoundary)
	}
}

type continuationScorecardRuntimeReport struct {
	Status  string `json:"status"`
	Summary struct {
		RecentBundleCount                     int    `json:"recent_bundle_count"`
		LatestRunID                           string `json:"latest_run_id"`
		LatestAllExecutorTracksSucceeded      bool   `json:"latest_all_executor_tracks_succeeded"`
		RecentBundleChainHasNoFailures        bool   `json:"recent_bundle_chain_has_no_failures"`
		AllExecutorTracksHaveRepeatedCoverage bool   `json:"all_executor_tracks_have_repeated_recent_coverage"`
	} `json:"summary"`
	SharedQueueCompanion struct {
		CrossNodeCompletions int    `json:"cross_node_completions"`
		Mode                 string `json:"mode"`
		SummaryPath          string `json:"summary_path"`
	} `json:"shared_queue_companion"`
	ExecutorLanes      []continuationScorecardLane `json:"executor_lanes"`
	ContinuationChecks []continuationCheckEnvelope `json:"continuation_checks"`
}

type continuationScorecardLane struct {
	Lane                   string `json:"lane"`
	LatestStatus           string `json:"latest_status"`
	ConsecutiveSuccesses   int    `json:"consecutive_successes"`
	EnabledRuns            int    `json:"enabled_runs"`
	AllRecentRunsSucceeded bool   `json:"all_recent_runs_succeeded"`
}

type continuationCheckEnvelope struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

func runContinuationScorecardBuildReport(t *testing.T) continuationScorecardRuntimeReport {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "e2e", "validation_bundle_continuation_scorecard.py")
	pythonSnippet := strings.Join([]string{
		"import importlib.util, json",
		"spec = importlib.util.spec_from_file_location('validation_bundle_continuation_scorecard', r'" + scriptPath + "')",
		"module = importlib.util.module_from_spec(spec)",
		"assert spec.loader is not None",
		"spec.loader.exec_module(module)",
		"print(json.dumps(module.build_report()))",
	}, "\n")

	cmd := testharness.PythonCommand(t, "-c", pythonSnippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build continuation scorecard report: %v (%s)", err, string(output))
	}

	var report continuationScorecardRuntimeReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode continuation scorecard report: %v (%s)", err, string(output))
	}
	return report
}

func findContinuationCheckByName(t *testing.T, items []continuationCheckEnvelope, name string) continuationCheckEnvelope {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("missing continuation check %q in %+v", name, items)
	return continuationCheckEnvelope{}
}
