package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO201RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO201SrcBigclawTreeStaysAbsentAndPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	srcBigclawDir := filepath.Join(rootRepo, "src", "bigclaw")

	if _, err := os.Stat(srcBigclawDir); !os.IsNotExist(err) {
		t.Fatalf("expected src/bigclaw tree to stay absent: %s (err=%v)", srcBigclawDir, err)
	}

	pythonFiles := collectPythonFiles(t, srcBigclawDir)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected src/bigclaw tree to remain Python-free: %v", pythonFiles)
	}
}

func TestBIGGO201ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO201LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-201-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-201",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"Directory absent on disk: `yes`.",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`test ! -d src/bigclaw`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO201(RepositoryHasNoPythonFiles|SrcBigclawTreeStaysAbsentAndPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
