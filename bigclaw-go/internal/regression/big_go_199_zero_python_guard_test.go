package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO199RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO199HiddenAndNestedResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		".githooks",
		".github",
		".symphony",
		"bigclaw-go/examples",
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

func TestBIGGO199HiddenAndNestedNativeAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".githooks/post-commit",
		".githooks/post-rewrite",
		".github/workflows/ci.yml",
		".symphony/workpad.md",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"reports/BIG-GO-192-validation.md",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected hidden or nested native asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO199LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-199-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-199",
		"Repository-wide Python-like file count: `0`.",
		"Overlooked Python-like suffixes audited: `.py`, `.pyw`, `.pyi`, `.pyx`, `.pxd`, `.pxi`, `.ipynb`.",
		"`.githooks`: `0` Python-like files",
		"`.github`: `0` Python-like files",
		"`.symphony`: `0` Python-like files",
		"`bigclaw-go/examples`: `0` Python-like files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python-like files",
		"`reports`: `0` Python-like files",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`.github/workflows/ci.yml`",
		"`.symphony/workpad.md`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`reports/BIG-GO-192-validation.md`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.pyx' -o -name '*.pxd' -o -name '*.pxi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .githooks .github .symphony bigclaw-go/examples bigclaw-go/docs/reports/live-validation-runs reports -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.pyx' -o -name '*.pxd' -o -name '*.pxi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO199(RepositoryHasNoPythonFiles|HiddenAndNestedResidualDirectoriesStayPythonFree|HiddenAndNestedNativeAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
