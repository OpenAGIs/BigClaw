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
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "validation-bundle-continuation-policy-gate")
	scorecardPath := filepath.Join(tmpDir, "docs", "reports", "validation-bundle-continuation-scorecard.json")
	outputPath := filepath.Join(tmpDir, "docs", "reports", "validation-bundle-continuation-policy-gate.json")
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
			scriptPath,
			"--scorecard", scorecardPath,
			"--output", outputPath,
		}
		for _, item := range extraArgs {
			key, value, ok := strings.Cut(item, "=")
			if !ok {
				t.Fatalf("malformed arg %q", item)
			}
			switch key {
			case "enforcement_mode":
				args = append(args, "--enforcement-mode", value)
			case "legacy_enforce_continuation_gate":
				if value == "true" {
					args = append(args, "--enforce")
				}
			case "require_repeated_lane_coverage":
				if value == "false" {
					args = append(args, "--allow-partial-lane-history")
				}
			default:
				t.Fatalf("unexpected arg key %q", key)
			}
		}
		cmd := exec.Command("bash", args...)
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("run continuation gate builder: %v\n%s", err, output)
			}
			if exitErr.ExitCode() != 1 && exitErr.ExitCode() != 2 {
				t.Fatalf("run continuation gate builder: %v\n%s", err, output)
			}
		}
		output, err = os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("read continuation gate output: %v", err)
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
