package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1546RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1546WorkspaceBootstrapPlanningResidualAreaStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"workspace",
		"bootstrap",
		"planning",
		"bigclaw-go/internal/bootstrap",
		"bigclaw-go/internal/planning",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual area to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1546GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/symphony-repo-bootstrap-template.md",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/api/broker_bootstrap_surface.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1546LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1546-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1546",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `workspace/bootstrap/planning` physical Python file count before lane",
		"Focused `workspace/bootstrap/planning` physical Python file count after lane",
		"Deleted files in this lane: `[]`",
		"Focused ledger for `workspace/bootstrap/planning`: `[]`",
		"`workspace`: directory not present, so residual Python files = `0`",
		"`bootstrap`: directory not present, so residual Python files = `0`",
		"`planning`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/internal/bootstrap`: `0` Python files",
		"`bigclaw-go/internal/planning`: `0` Python files",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/api/broker_bootstrap_surface.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1546",
		"Result: `0`",
		"bigclaw-go/internal/regression",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
