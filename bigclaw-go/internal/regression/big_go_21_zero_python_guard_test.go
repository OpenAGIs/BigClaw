package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO21RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO21BatchCSweepSurfaceStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	paths := []string{
		"src/bigclaw",
		"bigclaw-go/internal/bootstrap",
	}

	for _, relativePath := range paths {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected batch C sweep surface to remain Python-free: %s (%v)", relativePath, pythonFiles)
		}
	}
}

func TestBIGGO21RetiredBatchCPythonPathRemainsAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/workspace_bootstrap_validation.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired batch C Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO21GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche3_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO21LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-21-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-21",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`bigclaw-go/internal/bootstrap`: `0` Python files",
		"`src/bigclaw/workspace_bootstrap_validation.py`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap_test.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche3_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO21(RepositoryHasNoPythonFiles|BatchCSweepSurfaceStaysPythonFree|RetiredBatchCPythonPathRemainsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche3$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
