package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO181RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO181SrcBigclawTranche15StaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	paths := []string{
		"src/bigclaw",
		"bigclaw-go/internal/governance",
		"bigclaw-go/internal/domain",
		"bigclaw-go/internal/observability",
		"bigclaw-go/internal/product",
		"bigclaw-go/internal/workflow",
	}

	for _, relativePath := range paths {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected focused tranche-15 path to remain Python-free: %s (%v)", relativePath, pythonFiles)
		}
	}
}

func TestBIGGO181RetiredTranche15PythonPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/governance.py",
		"src/bigclaw/models.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/orchestration.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-15 Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO181GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/workflow/orchestration.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO181LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-181-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-181",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/operations.py`",
		"`src/bigclaw/orchestration.py`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/domain/task.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/workflow/orchestration.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw bigclaw-go/internal/governance bigclaw-go/internal/domain bigclaw-go/internal/observability bigclaw-go/internal/product bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO181(RepositoryHasNoPythonFiles|SrcBigclawTranche15StaysPythonFree|RetiredTranche15PythonPathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche15$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
