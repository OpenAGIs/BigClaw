package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestValidationBundleContinuationPolicyGateHoldForPartialLaneHistory(t *testing.T) {
	scorecardPath := writeContinuationPolicyScorecard(t)
	report := runContinuationPolicyBuildReport(t, scorecardPath, true)

	if report.Status != "policy-hold" || report.Recommendation != "hold" {
		t.Fatalf("unexpected policy hold report: %+v", report)
	}
	if len(report.FailingChecks) != 1 || report.FailingChecks[0] != "repeated_lane_coverage_meets_policy" {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
	if report.Summary.FailingCheckCount != 1 {
		t.Fatalf("unexpected failing check count: %+v", report.Summary)
	}
}

func TestValidationBundleContinuationPolicyGateCanAllowPartialLaneHistory(t *testing.T) {
	scorecardPath := writeContinuationPolicyScorecard(t)
	report := runContinuationPolicyBuildReport(t, scorecardPath, false)

	if report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("unexpected policy go report: %+v", report)
	}
	if len(report.FailingChecks) != 0 || report.Summary.FailingCheckCount != 0 {
		t.Fatalf("expected no failing checks, got %+v", report)
	}
}

func TestValidationBundleContinuationPolicyGateCLIReturnsZeroForCheckedInGo(t *testing.T) {
	cmd := testharness.PythonCommand(t, testharness.JoinRepoRoot(t, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("continuation policy gate cli failed: %v (%s)", err, string(output))
	}
}

type continuationPolicyRuntimeReport struct {
	Status         string   `json:"status"`
	Recommendation string   `json:"recommendation"`
	FailingChecks  []string `json:"failing_checks"`
	Summary        struct {
		LatestRunID       string `json:"latest_run_id"`
		FailingCheckCount int    `json:"failing_check_count"`
	} `json:"summary"`
}

func writeContinuationPolicyScorecard(t *testing.T) string {
	t.Helper()
	scorecard := map[string]any{
		"summary": map[string]any{
			"latest_run_id":                             "synthetic-run",
			"latest_bundle_age_hours":                   1.5,
			"recent_bundle_count":                       2,
			"latest_all_executor_tracks_succeeded":      true,
			"recent_bundle_chain_has_no_failures":       true,
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

	path := filepath.Join(t.TempDir(), "scorecard.json")
	body, err := json.Marshal(scorecard)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}
	return path
}

func runContinuationPolicyBuildReport(t *testing.T, scorecardPath string, requireRepeatedLaneCoverage bool) continuationPolicyRuntimeReport {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	pythonSnippet := strings.Join([]string{
		"import importlib.util, json",
		"spec = importlib.util.spec_from_file_location('validation_bundle_continuation_policy_gate', r'" + scriptPath + "')",
		"module = importlib.util.module_from_spec(spec)",
		"assert spec.loader is not None",
		"spec.loader.exec_module(module)",
		"report = module.build_report(scorecard_path=r'" + scorecardPath + "', require_repeated_lane_coverage=" + strconvFormatBool(requireRepeatedLaneCoverage) + ")",
		"print(json.dumps(report))",
	}, "\n")

	cmd := testharness.PythonCommand(t, "-c", pythonSnippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build continuation policy report: %v (%s)", err, string(output))
	}
	var report continuationPolicyRuntimeReport
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("decode continuation policy report: %v (%s)", err, string(output))
	}
	return report
}

func strconvFormatBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
