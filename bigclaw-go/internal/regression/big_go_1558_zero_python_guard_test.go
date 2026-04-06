package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1558RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1558ExamplesSurfaceStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "bigclaw-go", "examples"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected examples surface to remain Python-free: %v", pythonFiles)
	}
}

func TestBIGGO1558ExampleAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	exampleAssets := []string{
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-budget.json",
		"bigclaw-go/examples/shadow-task-validation.json",
	}

	for _, relativePath := range exampleAssets {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected example asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1558LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1558-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1558",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `bigclaw-go/examples` physical Python file count before lane changes:",
		"Focused `bigclaw-go/examples` physical Python file count after lane changes:",
		"Deleted files in this lane: `[]`",
		"Focused ledger for `bigclaw-go/examples`: `[]`",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-budget.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1558",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
