package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runPythonModuleJSON(t *testing.T, scriptPath string, pythonCode string, args ...string) map[string]any {
	t.Helper()
	cmdArgs := append([]string{"-c", pythonCode, scriptPath}, args...)
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python3 %v failed: %v\n%s", cmdArgs, err, string(output))
	}
	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode python json output: %v\n%s", err, string(output))
	}
	return payload
}

func runPythonCommand(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("python3", args...)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python3 %v failed: %v\n%s", args, err, string(output))
	}
	return output
}

func TestValidationBundleContinuationScorecardCheckedInShape(t *testing.T) {
	repoRoot := repoRoot(t)
	var report struct {
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
			DuplicateCompleted   int    `json:"duplicate_completed_tasks"`
			Mode                 string `json:"mode"`
			SummaryPath          string `json:"summary_path"`
		} `json:"shared_queue_companion"`
		ExecutorLanes []struct {
			Lane string `json:"lane"`
		} `json:"executor_lanes"`
	}
	readJSONFile(t, filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-scorecard.json"), &report)

	if report.Status != "local-continuation-scorecard" {
		t.Fatalf("unexpected status: %s", report.Status)
	}
	if report.Summary.RecentBundleCount != 3 || report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("unexpected scorecard summary: %+v", report.Summary)
	}
	if !report.Summary.LatestAllExecutorTracksSucceeded || !report.Summary.RecentBundleChainHasNoFailures || !report.Summary.AllExecutorTracksHaveRepeatedCoverage {
		t.Fatalf("unexpected scorecard booleans: %+v", report.Summary)
	}
	if report.SharedQueueCompanion.CrossNodeCompletions != 99 || report.SharedQueueCompanion.DuplicateCompleted != 0 || report.SharedQueueCompanion.Mode != "bundle-companion-summary" || report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue companion: %+v", report.SharedQueueCompanion)
	}
	if len(report.ExecutorLanes) == 0 || report.ExecutorLanes[0].Lane != "local" {
		t.Fatalf("unexpected executor lanes: %+v", report.ExecutorLanes)
	}
}

func TestValidationBundleContinuationScorecardScriptBuildReport(t *testing.T) {
	repoRoot := repoRoot(t)
	outputPath := filepath.Join(t.TempDir(), "validation-bundle-continuation-scorecard.json")
	runPythonCommand(t,
		filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_scorecard.py"),
		"--output", outputPath,
	)

	var report struct {
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
		ExecutorLanes []struct {
			Lane                 string `json:"lane"`
			LatestStatus         string `json:"latest_status"`
			ConsecutiveSuccesses int    `json:"consecutive_successes"`
			EnabledRuns          int    `json:"enabled_runs"`
			AllRecentSucceeded   bool   `json:"all_recent_runs_succeeded"`
		} `json:"executor_lanes"`
		ContinuationChecks []struct {
			Name   string `json:"name"`
			Passed bool   `json:"passed"`
			Detail string `json:"detail"`
		} `json:"continuation_checks"`
	}
	readJSONFile(t, outputPath, &report)

	if report.Status != "local-continuation-scorecard" || report.Summary.RecentBundleCount != 3 || report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("unexpected generated scorecard summary: %+v", report)
	}
	if len(report.ExecutorLanes) != 3 {
		t.Fatalf("unexpected executor lanes: %+v", report.ExecutorLanes)
	}
	lanes := map[string]struct {
		LatestStatus         string
		ConsecutiveSuccesses int
		EnabledRuns          int
		AllRecentSucceeded   bool
	}{}
	for _, lane := range report.ExecutorLanes {
		lanes[lane.Lane] = struct {
			LatestStatus         string
			ConsecutiveSuccesses int
			EnabledRuns          int
			AllRecentSucceeded   bool
		}{lane.LatestStatus, lane.ConsecutiveSuccesses, lane.EnabledRuns, lane.AllRecentSucceeded}
	}
	if lanes["local"].LatestStatus != "succeeded" || lanes["local"].ConsecutiveSuccesses != 3 || !lanes["local"].AllRecentSucceeded {
		t.Fatalf("unexpected local lane: %+v", lanes["local"])
	}
	if lanes["kubernetes"].LatestStatus != "succeeded" || lanes["kubernetes"].ConsecutiveSuccesses != 3 || !lanes["kubernetes"].AllRecentSucceeded {
		t.Fatalf("unexpected kubernetes lane: %+v", lanes["kubernetes"])
	}
	if lanes["ray"].LatestStatus != "succeeded" || lanes["ray"].ConsecutiveSuccesses != 2 || lanes["ray"].EnabledRuns != 2 || !lanes["ray"].AllRecentSucceeded {
		t.Fatalf("unexpected ray lane: %+v", lanes["ray"])
	}
	checks := map[string]struct {
		Passed bool
		Detail string
	}{}
	for _, item := range report.ContinuationChecks {
		checks[item.Name] = struct {
			Passed bool
			Detail string
		}{item.Passed, item.Detail}
	}
	if !checks["all_executor_tracks_have_repeated_recent_coverage"].Passed || checks["all_executor_tracks_have_repeated_recent_coverage"].Detail == "" {
		t.Fatalf("unexpected repeated coverage check: %+v", checks["all_executor_tracks_have_repeated_recent_coverage"])
	}
	if !checks["continuation_surface_is_workflow_triggered"].Passed || checks["continuation_surface_is_workflow_triggered"].Detail == "" {
		t.Fatalf("unexpected workflow trigger check: %+v", checks["continuation_surface_is_workflow_triggered"])
	}
	if report.SharedQueueCompanion.CrossNodeCompletions != 99 || report.SharedQueueCompanion.Mode != "bundle-companion-summary" || report.SharedQueueCompanion.SummaryPath != "docs/reports/shared-queue-companion-summary.json" {
		t.Fatalf("unexpected shared queue companion: %+v", report.SharedQueueCompanion)
	}
}

func TestValidationBundleContinuationPolicyGatePartialLaneHistoryHold(t *testing.T) {
	scorecardPath := filepath.Join(t.TempDir(), "scorecard.json")
	payload := map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "synthetic-run",
			"latest_bundle_age_hours":                           1.5,
			"recent_bundle_count":                               2,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": false,
		},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
			"report_path":               "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	if err := os.WriteFile(scorecardPath, body, 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}

	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	report := runPythonModuleJSON(t, scriptPath, `
import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("gate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
print(json.dumps(module.build_report(scorecard_path=sys.argv[2])))
`, scorecardPath)

	if report["status"] != "policy-hold" || report["recommendation"] != "hold" {
		t.Fatalf("unexpected hold report: %+v", report)
	}
	failingChecks, ok := report["failing_checks"].([]any)
	if !ok || len(failingChecks) != 1 || failingChecks[0] != "repeated_lane_coverage_meets_policy" {
		t.Fatalf("unexpected failing checks: %+v", report["failing_checks"])
	}
	summary := report["summary"].(map[string]any)
	if summary["failing_check_count"] != float64(1) {
		t.Fatalf("unexpected failing_check_count: %+v", summary)
	}
}

func TestValidationBundleContinuationPolicyGateCanAllowPartialLaneHistory(t *testing.T) {
	scorecardPath := filepath.Join(t.TempDir(), "scorecard.json")
	payload := map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "synthetic-run",
			"latest_bundle_age_hours":                           1.5,
			"recent_bundle_count":                               2,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": false,
		},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
			"report_path":               "bigclaw-go/docs/reports/multi-node-shared-queue-report.json",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	if err := os.WriteFile(scorecardPath, body, 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}

	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	report := runPythonModuleJSON(t, scriptPath, `
import importlib.util, json, sys
spec = importlib.util.spec_from_file_location("gate", sys.argv[1])
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
print(json.dumps(module.build_report(scorecard_path=sys.argv[2], require_repeated_lane_coverage=False)))
`, scorecardPath)

	if report["status"] != "policy-go" || report["recommendation"] != "go" {
		t.Fatalf("unexpected go report: %+v", report)
	}
	failingChecks, ok := report["failing_checks"].([]any)
	if !ok || len(failingChecks) != 0 {
		t.Fatalf("unexpected failing checks: %+v", report["failing_checks"])
	}
}

func TestValidationBundleContinuationPolicyGateCheckedInShape(t *testing.T) {
	repoRoot := repoRoot(t)
	var report struct {
		Status         string   `json:"status"`
		Recommendation string   `json:"recommendation"`
		FailingChecks  []string `json:"failing_checks"`
		Summary        struct {
			LatestRunID string `json:"latest_run_id"`
		} `json:"summary"`
	}
	readJSONFile(t, filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &report)

	if report.Status != "policy-go" || report.Recommendation != "go" || report.Summary.LatestRunID != "20260316T140138Z" || len(report.FailingChecks) != 0 {
		t.Fatalf("unexpected checked-in policy gate: %+v", report)
	}
}

func TestValidationBundleContinuationPolicyGateCLIReturnsZeroForCheckedInGo(t *testing.T) {
	repoRoot := repoRoot(t)
	outputPath := filepath.Join(t.TempDir(), "validation-bundle-continuation-policy-gate.json")
	cmd := exec.Command(
		"python3",
		filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py"),
		"--output", outputPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if exitErr, ok := err.(*exec.ExitError); ok {
		t.Fatalf("policy gate cli exited %d\n%s", exitErr.ExitCode(), string(output))
	}
	if err != nil {
		t.Fatalf("run policy gate cli: %v\n%s", err, string(output))
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file %s: %v", outputPath, err)
	}
}
