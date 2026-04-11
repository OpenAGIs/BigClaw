package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1600RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1600PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1600AssignedTrancheAssetsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	assignedPythonAssets := []string{
		"src/bigclaw/dsl.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/repo_governance.py",
		"src/bigclaw/saved_views.py",
		"tests/test_audit_events.py",
		"tests/test_event_bus.py",
		"tests/test_memory.py",
		"tests/test_repo_board.py",
	}

	for _, relativePath := range assignedPythonAssets {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1600GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1600LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1600-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1600",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Explicit assigned Python asset list:",
		"`src/bigclaw/dsl.py`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/repo_governance.py`",
		"`src/bigclaw/saved_views.py`",
		"`tests/test_audit_events.py`",
		"`tests/test_event_bus.py`",
		"`tests/test_memory.py`",
		"`tests/test_repo_board.py`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/repo/governance.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/events/bus_test.go`",
		"`bigclaw-go/internal/policy/memory_test.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1600",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
