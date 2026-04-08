package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1589RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1589RootScriptsBucketStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "scripts"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected root scripts bucket to remain Python-free, found %v", pythonFiles)
	}
}

func TestBIGGO1589ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected root scripts replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1589LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "reports/BIG-GO-1589-validation.md")

	for _, needle := range []string{
		"BIG-GO-1589",
		"Strict bucket lane 1589: scripts/*.py bucket",
		"Repository-wide physical `.py` files before lane changes: `0`",
		"Repository-wide physical `.py` files after lane changes: `0`",
		"Root `scripts` physical `.py` files before lane changes: `0`",
		"Root `scripts` physical `.py` files after lane changes: `0`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1589/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1589(RepositoryHasNoPythonFiles|RootScriptsBucketStaysPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"Push target: `origin/BIG-GO-1589`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
