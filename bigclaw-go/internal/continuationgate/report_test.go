package continuationgate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeScorecard(t *testing.T, repoRoot string, latestAge float64, recentBundles int, repeatedCoverage bool, sharedQueue bool) string {
	t.Helper()
	path := filepath.Join(repoRoot, "docs/reports/validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	payload := map[string]any{
		"summary": map[string]any{
			"latest_run_id":                                     "20260316T140138Z",
			"latest_bundle_age_hours":                           latestAge,
			"recent_bundle_count":                               recentBundles,
			"latest_all_executor_tracks_succeeded":              true,
			"recent_bundle_chain_has_no_failures":               true,
			"all_executor_tracks_have_repeated_recent_coverage": repeatedCoverage,
		},
		"shared_queue_companion": map[string]any{
			"available":                 sharedQueue,
			"cross_node_completions":    99,
			"duplicate_completed_tasks": 0,
			"duplicate_started_tasks":   0,
			"mode":                      "standalone-proof",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal scorecard: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write scorecard: %v", err)
	}
	return path
}

func TestBuildReportReturnsPolicyGoWhenInputsPass(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecard(t, repoRoot, 1, 3, true, true)

	report, err := BuildReport(BuildOptions{
		RepoRoot:                    repoRoot,
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		RequireRepeatedLaneCoverage: true,
		Now:                         time.Date(2026, 3, 28, 1, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if got := report["status"]; got != "policy-go" {
		t.Fatalf("unexpected status: %v", got)
	}
	if got := report["recommendation"]; got != "go" {
		t.Fatalf("unexpected recommendation: %v", got)
	}
}

func TestBuildReportReturnsPolicyHoldWithActionableFailures(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecard(t, repoRoot, 96, 1, false, false)

	report, err := BuildReport(BuildOptions{
		RepoRoot:                    repoRoot,
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		RequireRepeatedLaneCoverage: true,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	if got := report["status"]; got != "policy-hold" {
		t.Fatalf("unexpected status: %v", got)
	}
	failing, _ := report["failing_checks"].([]string)
	if len(failing) == 0 {
		t.Fatalf("expected failing checks")
	}
}

func TestBuildReportLegacyEnforceMapsToFailMode(t *testing.T) {
	repoRoot := t.TempDir()
	writeScorecard(t, repoRoot, 1, 1, true, true)

	report, err := BuildReport(BuildOptions{
		RepoRoot:                  repoRoot,
		ScorecardPath:             "docs/reports/validation-bundle-continuation-scorecard.json",
		LegacyEnforceContinuation: true,
	})
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	enforcement := report["enforcement"].(EnforcementSummary)
	if enforcement.Mode != "fail" || enforcement.ExitCode != 1 {
		t.Fatalf("unexpected enforcement: %+v", enforcement)
	}
}
