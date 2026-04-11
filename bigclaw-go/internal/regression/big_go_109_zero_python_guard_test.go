package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO109RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO109HiddenAndNestedResidualDirectoriesStayPythonFree(t *testing.T) {
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
			t.Fatalf("expected overlooked residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO109HiddenAndNestedNativeAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".githooks/post-commit",
		".githooks/post-rewrite",
		".github/workflows/ci.yml",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected overlooked native asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO109LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-109-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-109",
		"Repository-wide Python file count: `0`.",
		"`.githooks`: `0` Python files",
		"`.github`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`reports`: `0` Python files",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`.github/workflows/ci.yml`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .githooks .github .symphony bigclaw-go/examples bigclaw-go/docs/reports/live-validation-runs reports -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO109",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
