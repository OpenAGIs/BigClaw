package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO188RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO188RepoRootControlMetadataStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	rootMetadataTargets := []string{
		".symphony",
	}

	for _, relativePath := range rootMetadataTargets {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected repo-root control metadata surface to remain Python-free: %s (%v)", relativePath, pythonFiles)
		}
	}

	rootMatches, err := filepath.Glob(filepath.Join(rootRepo, "*.py"))
	if err != nil {
		t.Fatalf("glob repo-root python files: %v", err)
	}
	if len(rootMatches) != 0 {
		t.Fatalf("expected repo root to remain Python-free, found: %v", rootMatches)
	}
}

func TestBIGGO188RetainedRootAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".symphony/workpad.md",
		".gitignore",
		".pre-commit-config.yaml",
		"Makefile",
		"README.md",
		"workflow.md",
		"local-issues.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained repo-root asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO188LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-188-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-188",
		"Repository-wide Python file count: `0`.",
		"`.symphony`: `0` Python files",
		"`repo root (*.py)`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`.symphony/workpad.md`",
		"`.gitignore`",
		"`.pre-commit-config.yaml`",
		"`Makefile`",
		"`README.md`",
		"`workflow.md`",
		"`local-issues.json`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .symphony -type f -name '*.py' 2>/dev/null | sort`",
		"`find . -maxdepth 1 -type f -name '*.py' | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO188(RepositoryHasNoPythonFiles|RepoRootControlMetadataStaysPythonFree|RetainedRootAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
