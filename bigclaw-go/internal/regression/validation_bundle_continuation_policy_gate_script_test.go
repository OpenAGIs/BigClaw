package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidationBundleContinuationPolicyGateBuilderStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py")
	scorecardPath := filepath.Join(tmpDir, "docs", "reports", "validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(scorecardPath), 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}

	writeScorecard := func(latestBundleAgeHours float64, recentBundleCount int, latestSucceeded bool, recentChainOK bool, repeatedCoverage bool, sharedQueueAvailable bool) error {
		payload := map[string]any{
			"summary": map[string]any{
				"latest_run_id":                                     "20260316T140138Z",
				"latest_bundle_age_hours":                           latestBundleAgeHours,
				"recent_bundle_count":                               recentBundleCount,
				"latest_all_executor_tracks_succeeded":              latestSucceeded,
				"recent_bundle_chain_has_no_failures":               recentChainOK,
				"all_executor_tracks_have_repeated_recent_coverage": repeatedCoverage,
			},
			"shared_queue_companion": map[string]any{
				"available":                 sharedQueueAvailable,
				"cross_node_completions":    99,
				"duplicate_completed_tasks": 0,
				"duplicate_started_tasks":   0,
				"mode":                      "standalone-proof",
			},
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return os.WriteFile(scorecardPath, body, 0o644)
	}

	runBuildReport := func(extraArgs ...string) map[string]any {
		args := []string{
			"-c",
			`
import importlib.util
import json
import pathlib
import sys

script_path = pathlib.Path(sys.argv[1])
scorecard_path = sys.argv[2]
extra = sys.argv[3:]
spec = importlib.util.spec_from_file_location("validation_bundle_continuation_policy_gate", script_path)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)

kwargs = {"scorecard_path": scorecard_path}
for item in extra:
    key, value = item.split("=", 1)
    if value == "true":
        kwargs[key] = True
    elif value == "false":
        kwargs[key] = False
    else:
        kwargs[key] = value
print(json.dumps(module.build_report(**kwargs)))
`,
			scriptPath,
			scorecardPath,
		}
		args = append(args, extraArgs...)
		cmd := exec.Command("python3", args...)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run continuation gate builder: %v\n%s", err, output)
		}
		var report map[string]any
		if err := json.Unmarshal(output, &report); err != nil {
			t.Fatalf("decode continuation gate output: %v\n%s", err, output)
		}
		return report
	}

	if err := writeScorecard(1.0, 3, true, true, true, true); err != nil {
		t.Fatalf("write pass scorecard: %v", err)
	}
	report := runBuildReport()
	if report["ticket"] != "OPE-262" || report["status"] != "policy-go" || report["recommendation"] != "go" {
		t.Fatalf("unexpected passing report identity: %+v", report)
	}
	enforcement := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "hold" || enforcement["outcome"] != "pass" {
		t.Fatalf("unexpected passing enforcement: %+v", enforcement)
	}
	if len(report["failing_checks"].([]any)) != 0 {
		t.Fatalf("expected no failing checks, got %+v", report["failing_checks"])
	}
	reviewerPath := report["reviewer_path"].(map[string]any)
	if reviewerPath["index_path"] != "docs/reports/live-validation-index.md" {
		t.Fatalf("unexpected reviewer path: %+v", reviewerPath)
	}
	nextActions := report["next_actions"].([]any)
	if len(nextActions) == 0 || !strings.Contains(nextActions[0].(string), "BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail") {
		t.Fatalf("unexpected next actions: %+v", nextActions)
	}

	if err := writeScorecard(96.0, 1, true, true, false, false); err != nil {
		t.Fatalf("write hold scorecard: %v", err)
	}
	report = runBuildReport()
	if report["status"] != "policy-hold" || report["recommendation"] != "hold" {
		t.Fatalf("unexpected hold report identity: %+v", report)
	}
	failingChecks := stringSet(report["failing_checks"].([]any))
	for _, check := range []string{
		"latest_bundle_age_within_threshold",
		"recent_bundle_count_meets_floor",
		"shared_queue_companion_available",
		"repeated_lane_coverage_meets_policy",
	} {
		if !failingChecks[check] {
			t.Fatalf("missing failing check %q in %+v", check, failingChecks)
		}
	}
	foundRunAll := false
	for _, action := range report["next_actions"].([]any) {
		if strings.Contains(action.(string), "./scripts/e2e/run_all.sh") {
			foundRunAll = true
			break
		}
	}
	if !foundRunAll {
		t.Fatalf("expected run_all action in %+v", report["next_actions"])
	}

	if err := writeScorecard(1.0, 1, true, true, true, true); err != nil {
		t.Fatalf("write explicit hold scorecard: %v", err)
	}
	report = runBuildReport("enforcement_mode=hold")
	enforcement = report["enforcement"].(map[string]any)
	if report["recommendation"] != "hold" || enforcement["mode"] != "hold" || enforcement["outcome"] != "hold" || int(enforcement["exit_code"].(float64)) != 2 {
		t.Fatalf("unexpected explicit hold enforcement: %+v", enforcement)
	}

	report = runBuildReport("legacy_enforce_continuation_gate=true")
	enforcement = report["enforcement"].(map[string]any)
	if enforcement["mode"] != "fail" || enforcement["outcome"] != "fail" || int(enforcement["exit_code"].(float64)) != 1 {
		t.Fatalf("unexpected legacy fail enforcement: %+v", enforcement)
	}
}

func stringSet(values []any) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if ok {
			set[text] = true
		}
	}
	return set
}
