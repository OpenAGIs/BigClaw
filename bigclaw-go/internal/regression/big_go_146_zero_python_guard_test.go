package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO146RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO146ResidualSupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"bigclaw-go/examples",
		"bigclaw-go/testdata",
		"bigclaw-go/demo",
		"bigclaw-go/demos",
		"bigclaw-go/docs",
		"scripts/ops",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual support-asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO146NativeSupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-budget.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/docs/migration-shadow.md",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native support asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO146LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-146-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-146",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/testdata`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/demo`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/demos`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/docs`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-budget.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/docs/migration-shadow.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/examples bigclaw-go/testdata bigclaw-go/demo bigclaw-go/demos bigclaw-go/docs scripts/ops -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO146(RepositoryHasNoPythonFiles|ResidualSupportAssetDirectoriesStayPythonFree|NativeSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
