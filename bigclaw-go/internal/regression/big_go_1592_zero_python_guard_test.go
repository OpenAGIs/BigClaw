package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1592RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1592PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1592AssignedPythonAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	assignedPaths := []string{
		"src/bigclaw/__main__.py",
		"src/bigclaw/event_bus.py",
		"src/bigclaw/orchestration.py",
		"src/bigclaw/repo_plane.py",
		"src/bigclaw/service.py",
		"tests/test_console_ia.py",
		"tests/test_execution_flow.py",
		"tests/test_observability.py",
	}

	for _, relativePath := range assignedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to stay absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1592GoOwnedReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/events/bus.go",
		"bigclaw-go/internal/orchestrator/loop.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/contract/execution.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/audit_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go-owned replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1592LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1592-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1592",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`src/bigclaw/__main__.py`",
		"`src/bigclaw/event_bus.py`",
		"`src/bigclaw/orchestration.py`",
		"`src/bigclaw/repo_plane.py`",
		"`src/bigclaw/service.py`",
		"`tests/test_console_ia.py`",
		"`tests/test_execution_flow.py`",
		"`tests/test_observability.py`",
		"`bigclaw-go/internal/events/bus.go`",
		"`bigclaw-go/internal/orchestrator/loop.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1592(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|AssignedPythonAssetsStayAbsent|GoOwnedReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
