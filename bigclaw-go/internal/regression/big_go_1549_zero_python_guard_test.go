package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1549RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1549LargestResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected largest residual directory candidate to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1549GoReplacementPathsRemainAvailable(t *testing.T) {
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
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1549LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1549-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1549",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused largest-residual directory physical Python file count before lane",
		"Focused largest-residual directory physical Python file count after lane",
		"Deleted files in this lane: `[]`",
		"Focused ledger for the largest residual directory candidate: `[]`",
		"Largest residual directory candidate: none; repository-wide `.py` count was",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1549",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
