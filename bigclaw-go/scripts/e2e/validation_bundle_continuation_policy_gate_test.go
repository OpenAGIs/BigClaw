package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationBundleContinuationPolicyGateReturnsPolicyGoWhenInputsPass(t *testing.T) {
	root := setupPolicyGateFixture(t, map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           1.0,
			"recent_bundle_count":                               3,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": true,
		},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
		},
	})

	report, exitCode := runPolicyGate(t, root, "--scorecard", "docs/reports/validation-bundle-continuation-scorecard.json", "--output", "docs/reports/policy.json")
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d: %+v", exitCode, report)
	}
	if report["status"] != "policy-go" || report["recommendation"] != "go" {
		t.Fatalf("unexpected report status: %+v", report)
	}
	enforcement := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "hold" || enforcement["outcome"] != "pass" || enforcement["exit_code"].(float64) != 0 {
		t.Fatalf("unexpected enforcement: %+v", enforcement)
	}
	nextActions := stringifySlice(report["next_actions"])
	if len(nextActions) == 0 || !strings.Contains(nextActions[0], "BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail") {
		t.Fatalf("unexpected next actions: %+v", nextActions)
	}
}

func TestValidationBundleContinuationPolicyGateReturnsPolicyHoldWithActionableFailures(t *testing.T) {
	root := setupPolicyGateFixture(t, map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           96.0,
			"recent_bundle_count":                               1,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": false,
		},
		"shared_queue_companion": map[string]any{
			"available":                 false,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
		},
	})

	report, exitCode := runPolicyGate(t, root, "--scorecard", "docs/reports/validation-bundle-continuation-scorecard.json", "--output", "docs/reports/policy.json")
	if exitCode != 2 {
		t.Fatalf("expected hold exit code 2, got %d: %+v", exitCode, report)
	}
	if report["status"] != "policy-hold" || report["recommendation"] != "hold" {
		t.Fatalf("unexpected report status: %+v", report)
	}
	failingChecks := stringifySlice(report["failing_checks"])
	for _, expected := range []string{
		"latest_bundle_age_within_threshold",
		"recent_bundle_count_meets_floor",
		"shared_queue_companion_available",
		"repeated_lane_coverage_meets_policy",
	} {
		if !containsString(failingChecks, expected) {
			t.Fatalf("missing failing check %q in %+v", expected, failingChecks)
		}
	}
	nextActions := stringifySlice(report["next_actions"])
	if !containsSubstring(nextActions, "./scripts/e2e/run_all.sh") {
		t.Fatalf("expected rerun action in %+v", nextActions)
	}
}

func TestValidationBundleContinuationPolicyGateLegacyEnforceFlagMapsToFailMode(t *testing.T) {
	root := setupPolicyGateFixture(t, map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           1.0,
			"recent_bundle_count":                               1,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": true,
		},
		"shared_queue_companion": map[string]any{
			"available":                 true,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
		},
	})

	report, exitCode := runPolicyGate(t, root, "--scorecard", "docs/reports/validation-bundle-continuation-scorecard.json", "--output", "docs/reports/policy.json", "--enforce")
	if exitCode != 1 {
		t.Fatalf("expected fail exit code 1, got %d: %+v", exitCode, report)
	}
	enforcement := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "fail" || enforcement["outcome"] != "fail" || enforcement["exit_code"].(float64) != 1 {
		t.Fatalf("unexpected enforcement: %+v", enforcement)
	}
}

func setupPolicyGateFixture(t *testing.T, scorecard map[string]any) string {
	t.Helper()
	root := t.TempDir()
	encoded, err := json.Marshal(scorecard)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	writeTestFile(t, root, "docs/reports/validation-bundle-continuation-scorecard.json", string(encoded), false)
	return root
}

func runPolicyGate(t *testing.T, root string, args ...string) (map[string]any, int) {
	t.Helper()
	repoRoot := repoRootForScriptTests(t)
	resolvedArgs := append([]string(nil), args...)
	for i := 0; i < len(resolvedArgs)-1; i++ {
		switch resolvedArgs[i] {
		case "--scorecard", "--output":
			if !filepath.IsAbs(resolvedArgs[i+1]) {
				resolvedArgs[i+1] = filepath.Join(root, filepath.FromSlash(resolvedArgs[i+1]))
			}
		}
	}
	cmdArgs := append([]string{"run", "./cmd/bigclawctl", "automation", "e2e", "validation-bundle-continuation-policy-gate"}, resolvedArgs...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PWD="+repoRoot)
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run policy gate: %v\n%s", err, output)
		}
		exitCode = exitErr.ExitCode()
		if strings.Contains(string(output), "exit status 2") {
			exitCode = 2
		}
	}
	reportPath := filepath.Join(root, "docs", "reports", "policy.json")
	payload := readJSONMap(t, reportPath)
	if enforcement, ok := payload["enforcement"].(map[string]any); ok {
		if reported, ok := enforcement["exit_code"].(float64); ok {
			exitCode = int(reported)
		}
	}
	return payload, exitCode
}

func stringifySlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}
