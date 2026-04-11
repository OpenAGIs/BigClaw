package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO257RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO257BroadRepoDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		".github",
		".githooks",
		"docs",
		"reports",
		"scripts",
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected broad repo sweep directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO257GoNativeSurfaceRemainsAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".githooks/post-rewrite",
		"docs/go-mainline-cutover-issue-pack.md",
		"docs/local-tracker-automation.md",
		"docs/parallel-refill-queue.json",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO257LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-257-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-257",
		"Repository-wide Python file count: `0`.",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`docs`: `0` Python files",
		"`reports`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`docs/local-tracker-automation.md`",
		"`docs/parallel-refill-queue.json`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find .github .githooks docs reports scripts bigclaw-go/examples bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO257(RepositoryHasNoPythonFiles|BroadRepoDirectoriesStayPythonFree|GoNativeSurfaceRemainsAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
