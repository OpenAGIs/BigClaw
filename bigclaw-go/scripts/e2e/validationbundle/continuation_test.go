package validationbundle

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildGateReturnsPolicyGoWhenInputsPass(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecardFixture(t, repoRoot, 1.0, 3, true, true, true, true)

	report, err := BuildGate(repoRoot, "docs/reports/validation-bundle-continuation-scorecard.json", 72, 2, true, "", false, time.Date(2026, time.March, 22, 0, 56, 46, 0, time.UTC))
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}
	if report.Ticket != "OPE-262" || report.Status != "policy-go" || report.Recommendation != "go" {
		t.Fatalf("unexpected gate metadata: %+v", report)
	}
	if report.Enforcement.Mode != "hold" || report.Enforcement.Outcome != "pass" || report.Enforcement.ExitCode != 0 {
		t.Fatalf("unexpected enforcement: %+v", report.Enforcement)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("unexpected failing checks: %+v", report.FailingChecks)
	}
	if report.ReviewerPath.DigestIssue.ID != "OPE-271" || report.ReviewerPath.DigestIssue.Slug != "BIG-PAR-082" {
		t.Fatalf("unexpected reviewer metadata: %+v", report.ReviewerPath)
	}
	if len(report.NextActions) != 1 || report.NextActions[0] != "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions" {
		t.Fatalf("unexpected next actions: %+v", report.NextActions)
	}
}

func TestBuildGateReturnsPolicyHoldWithActionableFailures(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecardFixture(t, repoRoot, 96.0, 1, true, true, false, false)

	report, err := BuildGate(repoRoot, "docs/reports/validation-bundle-continuation-scorecard.json", 72, 2, true, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}
	if report.Status != "policy-hold" || report.Recommendation != "hold" {
		t.Fatalf("unexpected recommendation: %+v", report)
	}
	assertContains(t, report.FailingChecks, "latest_bundle_age_within_threshold")
	assertContains(t, report.FailingChecks, "recent_bundle_count_meets_floor")
	assertContains(t, report.FailingChecks, "shared_queue_companion_available")
	assertContains(t, report.FailingChecks, "repeated_lane_coverage_meets_policy")
	assertActionContains(t, report.NextActions, "./scripts/e2e/run_all.sh")
}

func TestBuildGateSupportsHoldModeWithoutFailingGeneration(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecardFixture(t, repoRoot, 1.0, 1, true, true, true, true)

	report, err := BuildGate(repoRoot, "docs/reports/validation-bundle-continuation-scorecard.json", 72, 2, true, "hold", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}
	if report.Enforcement.Mode != "hold" || report.Enforcement.Outcome != "hold" || report.Enforcement.ExitCode != 2 {
		t.Fatalf("unexpected enforcement: %+v", report.Enforcement)
	}
}

func TestBuildGateLegacyEnforceFlagMapsToFailMode(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecardFixture(t, repoRoot, 1.0, 1, true, true, true, true)

	report, err := BuildGate(repoRoot, "docs/reports/validation-bundle-continuation-scorecard.json", 72, 2, true, "", true, time.Now().UTC())
	if err != nil {
		t.Fatalf("build gate: %v", err)
	}
	if report.Enforcement.Mode != "fail" || report.Enforcement.Outcome != "fail" || report.Enforcement.ExitCode != 1 {
		t.Fatalf("unexpected enforcement: %+v", report.Enforcement)
	}
}

func writeScorecardFixture(t *testing.T, repoRoot string, latestBundleAgeHours float64, recentBundleCount int, latestAllSucceeded bool, recentNoFailures bool, repeatedCoverage bool, sharedQueueAvailable bool) {
	t.Helper()
	payload := map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           latestBundleAgeHours,
			"recent_bundle_count":                               recentBundleCount,
			"latest_all_executor_tracks_succeeded":              latestAllSucceeded,
			"recent_bundle_chain_has_no_failures":               recentNoFailures,
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
	if err := WriteJSON(filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-scorecard.json"), payload); err != nil {
		t.Fatalf("write scorecard fixture: %v", err)
	}
}

func assertContains(t *testing.T, values []string, target string) {
	t.Helper()
	if !contains(values, target) {
		t.Fatalf("missing %q in %v", target, values)
	}
}

func assertActionContains(t *testing.T, values []string, fragment string) {
	t.Helper()
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return
		}
	}
	t.Fatalf("missing action containing %q in %v", fragment, values)
}
