package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1598RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1598AssignedFocusPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/memory.py",
		"src/bigclaw/repo_commits.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/workspace_bootstrap_validation.py",
		"tests/test_dsl.py",
		"tests/test_live_shadow_scorecard.py",
		"tests/test_planning.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to remain absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1598GoOwnedReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/queue/memory_queue.go",
		"bigclaw-go/internal/triage/repo.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/api/live_shadow_surface.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go-owned replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1598LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1598-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1598",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/dashboard_run_contract.py`",
		"`src/bigclaw/memory.py`",
		"`src/bigclaw/repo_commits.py`",
		"`src/bigclaw/run_detail.py`",
		"`src/bigclaw/workspace_bootstrap_validation.py`",
		"`tests/test_dsl.py`",
		"`tests/test_live_shadow_scorecard.py`",
		"`tests/test_planning.py`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/queue/memory_queue.go`",
		"`bigclaw-go/internal/triage/repo.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/api/live_shadow_surface.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`for path in src/bigclaw/dashboard_run_contract.py src/bigclaw/memory.py src/bigclaw/repo_commits.py src/bigclaw/run_detail.py src/bigclaw/workspace_bootstrap_validation.py tests/test_dsl.py tests/test_live_shadow_scorecard.py tests/test_planning.py; do test ! -e \"$path\" || echo \"present: $path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1598",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
