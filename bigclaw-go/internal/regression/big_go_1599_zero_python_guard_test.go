package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1599RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1599PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1599AssignedTrancheAssetsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	assignedPythonAssets := []string{
		"src/bigclaw/design_system.py",
		"src/bigclaw/models.py",
		"src/bigclaw/repo_gateway.py",
		"src/bigclaw/runtime.py",
		"tests/conftest.py",
		"tests/test_evaluation.py",
		"tests/test_mapping.py",
		"tests/test_queue.py",
	}

	for _, relativePath := range assignedPythonAssets {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1599GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/workflow/model.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1599LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1599-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1599",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Explicit assigned Python asset list:",
		"`src/bigclaw/design_system.py`",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/repo_gateway.py`",
		"`src/bigclaw/runtime.py`",
		"`tests/conftest.py`",
		"`tests/test_evaluation.py`",
		"`tests/test_mapping.py`",
		"`tests/test_queue.py`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/workflow/model.go`",
		"`bigclaw-go/internal/repo/gateway.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/evaluation/evaluation_test.go`",
		"`bigclaw-go/internal/intake/mapping_test.go`",
		"`bigclaw-go/internal/queue/memory_queue_test.go`",
		"`bigclaw-go/internal/refill/queue_test.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1599",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
