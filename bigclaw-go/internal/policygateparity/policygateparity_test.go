package policygateparity

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPolicyGateHoldsForPartialLaneHistory(t *testing.T) {
	t.Parallel()

	scorecardPath := writeScorecard(t, map[string]any{
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
	})

	report, err := BuildReport(scorecardPath, true)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.Status != "policy-hold" || report.Recommendation != "hold" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.FailingChecks) != 1 || report.FailingChecks[0] != "repeated_lane_coverage_meets_policy" {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
	if report.Summary["failing_check_count"] != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
}

func TestPolicyGateCanAllowPartialLaneHistory(t *testing.T) {
	t.Parallel()

	scorecardPath := writeScorecard(t, map[string]any{
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
	})

	report, err := BuildReport(scorecardPath, false)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
}

func TestCheckedInPolicyGateMatchesExpectedShape(t *testing.T) {
	t.Parallel()

	report, err := LoadCheckedInReport(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("load checked-in report: %v", err)
	}
	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("unexpected checked-in report: %+v", report)
	}
	if report.Summary["latest_run_id"] != "20260316T140138Z" {
		t.Fatalf("unexpected latest_run_id: %+v", report.Summary)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
}

func TestPolicyGateCLIReturnsZeroForCheckedInGo(t *testing.T) {
	t.Parallel()

	scriptPath := filepath.Join("scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	outputPath := filepath.Join(t.TempDir(), "policy-gate.json")
	cmd := exec.Command("python3", scriptPath, "--output", outputPath)
	cmd.Dir = filepath.Join("..", "..")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("cli exit code = %d, want 0", exitErr.ExitCode())
		}
		t.Fatalf("run cli: %v", err)
	}
}

func writeScorecard(t *testing.T, payload map[string]any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scorecard.json")
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}
	return path
}
