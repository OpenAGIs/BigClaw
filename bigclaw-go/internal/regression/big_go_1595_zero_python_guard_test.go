package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1595RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1595AssignedPythonSourceAndTestsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/connectors.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/workflow.py",
		"tests/test_cross_process_coordination_surface.py",
		"tests/test_governance.py",
		"tests/test_parallel_refill.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python path to remain absent: %s", relativePath)
		}
	}

	for _, relativeDir := range []string{"src/bigclaw", "tests"} {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativeDir))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python directory to remain absent: %s", relativeDir)
		}
	}
}

func TestBIGGO1595GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/go-domain-intake-parity-matrix.md",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/workflow/orchestration.go",
		"bigclaw-go/internal/api/coordination_surface.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
		"bigclaw-go/docs/reports/parallel-validation-matrix.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1595LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1595-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1595",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: absent",
		"`tests`: absent",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/planning.py`",
		"`src/bigclaw/reports.py`",
		"`src/bigclaw/workflow.py`",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_governance.py`",
		"`tests/test_parallel_refill.py`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/workflow/orchestration.go`",
		"`bigclaw-go/internal/api/coordination_surface.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`",
		"`bigclaw-go/docs/reports/parallel-validation-matrix.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find src/bigclaw tests -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1595",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
