package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO149RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO149HiddenAndNestedResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		".githooks",
		".github",
		".symphony",
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
		"reports",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected hidden or nested residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO149RetainedNativeAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".githooks/post-commit",
		".github/workflows/ci.yml",
		".symphony/workpad.md",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"reports/repo-wide-validation-2026-03-16.md",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO149LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-149-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-149",
		"Repository-wide physical Python file count: `0`.",
		"`.githooks`: `0` Python files",
		"`.github`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`reports`: `0` Python files",
		"`.githooks/post-commit`",
		"`.github/workflows/ci.yml`",
		"`.symphony/workpad.md`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`reports/repo-wide-validation-2026-03-16.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .githooks .github .symphony bigclaw-go/examples bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO149",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
