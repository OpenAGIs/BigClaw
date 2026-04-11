package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO258RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO258MetaAndOperatorDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		".github",
		".githooks",
		".symphony",
		"docs",
		"scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected repo meta or operator directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO258ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"workflow.md",
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".githooks/post-rewrite",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO258LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-258-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-258",
		"Broad repo Python reduction sweep AP",
		"Repository-wide Python file count: `0`.",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs`: `0` Python files",
		"`scripts`: `0` Python files",
		"`workflow.md`",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find .github .githooks .symphony docs scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO258(RepositoryHasNoPythonFiles|MetaAndOperatorDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
