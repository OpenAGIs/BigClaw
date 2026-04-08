package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1584RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1584StrictBucketBTestsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"tests/test_design_system.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_pilot.py",
		"tests/test_repo_triage.py",
		"tests/test_subscriber_takeover_harness.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected strict bucket-B test Python path to stay absent: %s", relativePath)
		}
	}

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "tests"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected tests bucket B to remain Python-free: %v", pythonFiles)
	}
}

func TestBIGGO1584ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/pilot/rollout_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1584LaneReportCapturesStrictBucketState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1584-tests-bucket-b.md")

	for _, needle := range []string{
		"BIG-GO-1584",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `tests/*.py` bucket-B physical Python file count before lane changes: `0`",
		"Focused `tests/*.py` bucket-B physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Strict bucket-B ledger: `[]`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`tests/test_design_system.py`",
		"`tests/test_live_shadow_bundle.py`",
		"`tests/test_pilot.py`",
		"`tests/test_repo_triage.py`",
		"`tests/test_subscriber_takeover_harness.py`",
		"`bigclaw-go/internal/designsystem/designsystem_test.go`",
		"`bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`",
		"`bigclaw-go/internal/pilot/rollout_test.go`",
		"`bigclaw-go/internal/triage/repo_test.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1584(RepositoryHasNoPythonFiles|StrictBucketBTestsStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesStrictBucketState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
