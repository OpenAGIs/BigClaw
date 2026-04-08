package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO141ResidualSrcBigclawPythonSweepKRepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO141ResidualSrcBigclawPythonSweepKRetiredPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/validation_policy.py",
		"src/bigclaw/memory.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired sweep-K Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO141ResidualSrcBigclawPythonSweepKGoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/policy/validation.go",
		"bigclaw-go/internal/policy/validation_test.go",
		"bigclaw-go/internal/policy/memory.go",
		"bigclaw-go/internal/policy/memory_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO141ResidualSrcBigclawPythonSweepKLaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-141-residual-src-bigclaw-python-sweep-k.md")

	for _, needle := range []string{
		"BIG-GO-141",
		"Residual src/bigclaw Python sweep K",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/validation_policy.py`",
		"`src/bigclaw/memory.py`",
		"`bigclaw-go/internal/policy/validation.go`",
		"`bigclaw-go/internal/policy/validation_test.go`",
		"`bigclaw-go/internal/policy/memory.go`",
		"`bigclaw-go/internal/policy/memory_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO141ResidualSrcBigclawPythonSweepK",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
