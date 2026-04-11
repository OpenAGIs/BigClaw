package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO251RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO251SrcBigclawTranche12PathRemainsAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/dsl.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-12 Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO251GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO251LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-251-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-251",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"Explicit assigned Python asset list:",
		"`src/bigclaw/dsl.py`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/workflow/definition_test.go`",
		"`bigclaw-go/internal/workflow/engine.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/dsl.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO251(RepositoryHasNoPythonFiles|SrcBigclawTranche12PathRemainsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche12$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
