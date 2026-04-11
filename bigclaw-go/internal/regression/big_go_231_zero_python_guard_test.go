package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO231RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO231SrcBigclawTranche14PathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPythonPaths := []string{
		"src/bigclaw/planning.py",
		"src/bigclaw/queue.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/risk.py",
	}

	for _, relativePath := range retiredPythonPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-14 Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO231GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reportstudio/reportstudio.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO231LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-231-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-231",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"Explicit assigned Python asset list:",
		"`src/bigclaw/planning.py`",
		"`src/bigclaw/queue.py`",
		"`src/bigclaw/reports.py`",
		"`src/bigclaw/risk.py`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/queue/queue.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/reportstudio/reportstudio.go`",
		"`bigclaw-go/internal/risk/risk.go`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/planning.py src/bigclaw/queue.py src/bigclaw/reports.py src/bigclaw/risk.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO231(RepositoryHasNoPythonFiles|SrcBigclawTranche14PathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche14$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
