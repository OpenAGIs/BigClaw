package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidationBundleContinuationPolicyGateBuildReportReturnsPolicyGoWhenInputsPass(t *testing.T) {
	repoRoot := t.TempDir()
	writeContinuationScorecardFixture(t, repoRoot, defaultContinuationScorecardFixture())

	report, exitCode, err := buildContinuationPolicyReport(repoRoot, continuationPolicyBuildOptions{
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		MaxLatestAgeHours:           72.0,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		GeneratedAt:                 time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationPolicyReport returned error: %v", err)
	}

	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if report.Ticket != "OPE-262" {
		t.Fatalf("ticket = %q, want OPE-262", report.Ticket)
	}
	if report.Status != "policy-go" {
		t.Fatalf("status = %q, want policy-go", report.Status)
	}
	if report.Recommendation != "go" {
		t.Fatalf("recommendation = %q, want go", report.Recommendation)
	}
	if report.Enforcement.Mode != "hold" {
		t.Fatalf("enforcement mode = %q, want hold", report.Enforcement.Mode)
	}
	if report.Enforcement.Outcome != "pass" {
		t.Fatalf("enforcement outcome = %q, want pass", report.Enforcement.Outcome)
	}
	if len(report.FailingChecks) != 0 {
		t.Fatalf("failing checks = %v, want none", report.FailingChecks)
	}
	if report.EvidenceInputs["generator_script"] != continuationPolicyGateScriptPath {
		t.Fatalf("generator_script = %v, want %s", report.EvidenceInputs["generator_script"], continuationPolicyGateScriptPath)
	}
	if report.ReviewerPath.IndexPath != "docs/reports/live-validation-index.md" {
		t.Fatalf("reviewer index path = %q", report.ReviewerPath.IndexPath)
	}
	if got := report.NextActions[0]; got != "set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions" {
		t.Fatalf("next action = %q", got)
	}
}

func TestValidationBundleContinuationPolicyGateBuildReportReturnsPolicyHoldWithActionableFailures(t *testing.T) {
	repoRoot := t.TempDir()
	fixture := defaultContinuationScorecardFixture()
	fixture.LatestBundleAgeHours = 96.0
	fixture.RecentBundleCount = 1
	fixture.RepeatedLaneCoverage = false
	fixture.SharedQueueAvailable = false
	writeContinuationScorecardFixture(t, repoRoot, fixture)

	report, exitCode, err := buildContinuationPolicyReport(repoRoot, continuationPolicyBuildOptions{
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		MaxLatestAgeHours:           72.0,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
		GeneratedAt:                 time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationPolicyReport returned error: %v", err)
	}

	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if report.Status != "policy-hold" {
		t.Fatalf("status = %q, want policy-hold", report.Status)
	}
	if report.Recommendation != "hold" {
		t.Fatalf("recommendation = %q, want hold", report.Recommendation)
	}
	assertContains(t, report.FailingChecks, "latest_bundle_age_within_threshold")
	assertContains(t, report.FailingChecks, "recent_bundle_count_meets_floor")
	assertContains(t, report.FailingChecks, "shared_queue_companion_available")
	assertContains(t, report.FailingChecks, "repeated_lane_coverage_meets_policy")
	assertContainsString(t, report.NextActions, "rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle")
}

func TestValidationBundleContinuationPolicyGateSupportsHoldModeWithoutFailingGeneration(t *testing.T) {
	repoRoot := t.TempDir()
	fixture := defaultContinuationScorecardFixture()
	fixture.RecentBundleCount = 1
	writeContinuationScorecardFixture(t, repoRoot, fixture)

	report, exitCode, err := buildContinuationPolicyReport(repoRoot, continuationPolicyBuildOptions{
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		EnforcementMode:             "hold",
		RequireRepeatedLaneCoverage: true,
		GeneratedAt:                 time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationPolicyReport returned error: %v", err)
	}

	if report.Recommendation != "hold" {
		t.Fatalf("recommendation = %q, want hold", report.Recommendation)
	}
	if report.Enforcement.Mode != "hold" || report.Enforcement.Outcome != "hold" || report.Enforcement.ExitCode != 2 {
		t.Fatalf("enforcement = %#v, want hold/hold/2", report.Enforcement)
	}
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
}

func TestValidationBundleContinuationPolicyGateLegacyEnforceFlagMapsToFailMode(t *testing.T) {
	repoRoot := t.TempDir()
	fixture := defaultContinuationScorecardFixture()
	fixture.RecentBundleCount = 1
	writeContinuationScorecardFixture(t, repoRoot, fixture)

	report, exitCode, err := buildContinuationPolicyReport(repoRoot, continuationPolicyBuildOptions{
		ScorecardPath:                 "docs/reports/validation-bundle-continuation-scorecard.json",
		LegacyEnforceContinuationGate: true,
		RequireRepeatedLaneCoverage:   true,
		GeneratedAt:                   time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationPolicyReport returned error: %v", err)
	}

	if report.Enforcement.Mode != "fail" || report.Enforcement.Outcome != "fail" || report.Enforcement.ExitCode != 1 {
		t.Fatalf("enforcement = %#v, want fail/fail/1", report.Enforcement)
	}
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
}

func TestValidationBundleContinuationPolicyGateWritesReport(t *testing.T) {
	repoRoot := t.TempDir()
	writeContinuationScorecardFixture(t, repoRoot, defaultContinuationScorecardFixture())

	report, _, err := buildContinuationPolicyReport(repoRoot, continuationPolicyBuildOptions{
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		RequireRepeatedLaneCoverage: true,
		GeneratedAt:                 time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("buildContinuationPolicyReport returned error: %v", err)
	}
	if err := writeContinuationPolicyReport(repoRoot, "docs/reports/validation-bundle-continuation-policy-gate.json", report); err != nil {
		t.Fatalf("writeContinuationPolicyReport returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-policy-gate.json")); err != nil {
		t.Fatalf("written report missing: %v", err)
	}
}

type continuationScorecardFixture struct {
	LatestBundleAgeHours             float64
	RecentBundleCount                int
	LatestAllExecutorTracksSucceeded bool
	RecentBundleChainHasNoFailures   bool
	RepeatedLaneCoverage             bool
	SharedQueueAvailable             bool
}

func defaultContinuationScorecardFixture() continuationScorecardFixture {
	return continuationScorecardFixture{
		LatestBundleAgeHours:             1.0,
		RecentBundleCount:                3,
		LatestAllExecutorTracksSucceeded: true,
		RecentBundleChainHasNoFailures:   true,
		RepeatedLaneCoverage:             true,
		SharedQueueAvailable:             true,
	}
}

func writeContinuationScorecardFixture(t *testing.T, repoRoot string, fixture continuationScorecardFixture) {
	t.Helper()
	payload := `{
  "summary": {
    "latest_run_id": "20260316T140138Z",
    "latest_bundle_age_hours": %0.1f,
    "recent_bundle_count": %d,
    "latest_all_executor_tracks_succeeded": %t,
    "recent_bundle_chain_has_no_failures": %t,
    "all_executor_tracks_have_repeated_recent_coverage": %t
  },
  "shared_queue_companion": {
    "available": %t,
    "cross_node_completions": 99,
    "duplicate_completed_tasks": 0,
    "duplicate_started_tasks": 0,
    "mode": "standalone-proof"
  }
}
`
	path := filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	body := []byte(
		sprintf(
			payload,
			fixture.LatestBundleAgeHours,
			fixture.RecentBundleCount,
			fixture.LatestAllExecutorTracksSucceeded,
			fixture.RecentBundleChainHasNoFailures,
			fixture.RepeatedLaneCoverage,
			fixture.SharedQueueAvailable,
		),
	)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func assertContains(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, items)
}

func assertContainsString(t *testing.T, items []string, want string) {
	t.Helper()
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, items)
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
