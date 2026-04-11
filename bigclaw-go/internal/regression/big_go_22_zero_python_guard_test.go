package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO22RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO22PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO22SrcBigClawBatchDRemainsAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	legacyBatchDRoot := filepath.Join(rootRepo, "src", "bigclaw")
	if _, err := os.Stat(legacyBatchDRoot); !os.IsNotExist(err) {
		t.Fatalf("expected retired src/bigclaw batch-D root to remain absent: %s (err=%v)", legacyBatchDRoot, err)
	}
}

func TestBIGGO22LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-22-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-22",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"batch-D `src/bigclaw` surface is already absent",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO22",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
