package regression

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8ValidationBundleContinuationPolicyGateScriptHandlesPartialLaneHistory(t *testing.T) {
	repoRoot := repoRoot(t)
	scorecardPath := filepath.Join(t.TempDir(), "scorecard.json")
	scorecard := map[string]any{
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
	body, err := json.Marshal(scorecard)
	if err != nil {
		t.Fatalf("marshal synthetic scorecard: %v", err)
	}
	if err := os.WriteFile(scorecardPath, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write synthetic scorecard: %v", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	holdOutputPath := filepath.Join(t.TempDir(), "hold-report.json")
	hold := exec.Command("python3", scriptPath, "--scorecard", scorecardPath, "--output", holdOutputPath)
	hold.Dir = repoRoot
	if output, err := hold.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
			t.Fatalf("run hold policy gate script: %v\n%s", err, string(output))
		}
	}

	var holdReport struct {
		Status         string   `json:"status"`
		Recommendation string   `json:"recommendation"`
		FailingChecks  []string `json:"failing_checks"`
		Summary        struct {
			FailingCheckCount int `json:"failing_check_count"`
		} `json:"summary"`
	}
	readJSONFile(t, holdOutputPath, &holdReport)
	if holdReport.Status != "policy-hold" || holdReport.Recommendation != "hold" {
		t.Fatalf("unexpected hold report status: %+v", holdReport)
	}
	if len(holdReport.FailingChecks) != 1 || holdReport.FailingChecks[0] != "repeated_lane_coverage_meets_policy" || holdReport.Summary.FailingCheckCount != 1 {
		t.Fatalf("unexpected hold report checks: %+v", holdReport)
	}

	goOutputPath := filepath.Join(t.TempDir(), "go-report.json")
	allowPartial := exec.Command("python3", scriptPath, "--scorecard", scorecardPath, "--output", goOutputPath, "--allow-partial-lane-history")
	allowPartial.Dir = repoRoot
	if output, err := allowPartial.CombinedOutput(); err != nil {
		t.Fatalf("run go policy gate script: %v\n%s", err, string(output))
	}

	var goReport struct {
		Status         string   `json:"status"`
		Recommendation string   `json:"recommendation"`
		FailingChecks  []string `json:"failing_checks"`
	}
	readJSONFile(t, goOutputPath, &goReport)
	if goReport.Status != "policy-go" || goReport.Recommendation != "go" {
		t.Fatalf("unexpected go report status: %+v", goReport)
	}
	if len(goReport.FailingChecks) != 0 {
		t.Fatalf("expected no failing checks when partial lane history is allowed, got %+v", goReport.FailingChecks)
	}
}

func TestLane8ValidationBundleContinuationPolicyGateScriptCLIStaysGreen(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	outputPath := filepath.Join(t.TempDir(), "checked-in-policy-gate.json")
	cmd := exec.Command("python3", scriptPath, "--output", outputPath)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run checked-in policy gate script: %v\n%s", err, string(output))
	}

	var report struct {
		Status         string `json:"status"`
		Recommendation string `json:"recommendation"`
		Summary        struct {
			LatestRunID string `json:"latest_run_id"`
		} `json:"summary"`
		FailingChecks []string `json:"failing_checks"`
	}
	readJSONFile(t, outputPath, &report)
	if report.Status != "policy-go" || report.Recommendation != "go" || report.Summary.LatestRunID != "20260316T140138Z" {
		t.Fatalf("unexpected checked-in policy gate report: %+v", report)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("expected checked-in policy gate to stay green, got %+v", report.FailingChecks)
	}
}
