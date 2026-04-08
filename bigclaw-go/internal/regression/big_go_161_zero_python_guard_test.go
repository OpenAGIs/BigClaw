package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO161RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO161SrcBigclawStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash("src/bigclaw")))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected src/bigclaw to remain Python-free (%v)", pythonFiles)
	}
}

func TestBIGGO161RemovedEventBusModuleStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	removedPythonPaths := []string{
		"src/bigclaw/event_bus.py",
	}

	for _, relativePath := range removedPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected removed Python module to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO161GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/events/transition_bus.go",
		"bigclaw-go/internal/events/transition_bus_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche13_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO161LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-161-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-161",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`src/bigclaw/event_bus.py`",
		"`bigclaw-go/internal/events/transition_bus.go`",
		"`bigclaw-go/internal/events/transition_bus_test.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche13_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw bigclaw-go/internal/events -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO161(RepositoryHasNoPythonFiles|SrcBigclawStaysPythonFree|RemovedEventBusModuleStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche13$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
