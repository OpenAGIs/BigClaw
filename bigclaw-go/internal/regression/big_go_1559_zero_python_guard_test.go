package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1559RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1559LargestResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"src",
		"tests",
		"scripts",
		"workspace",
		"bootstrap",
		"planning",
		"bigclaw-go/scripts",
		"bigclaw-go/internal",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1559GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
		"docs/symphony-repo-bootstrap-template.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1559LaneReportCapturesExactSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1559-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1559",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused largest residual-directory physical Python file count before lane",
		"Focused largest residual-directory physical Python file count after lane",
		"Deleted files in this lane: `[]`",
		"Focused ledger for largest residual-directory pass: `[]`",
		"`src`: directory not present, so residual Python files = `0`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`workspace`: directory not present, so residual Python files = `0`",
		"`bootstrap`: directory not present, so residual Python files = `0`",
		"`planning`: directory not present, so residual Python files = `0`",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/internal`: `0` Python files",
		"Largest residual-directory candidate rechecked by this lane:",
		"`bigclaw-go/internal` with `0` physical `.py` files.",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src tests scripts workspace bootstrap planning bigclaw-go/scripts bigclaw-go/internal -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1559",
		"Acceptance asked for a lower physical `.py` count than baseline.",
		"Repository reality in this checkout is already `0 -> 0`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
