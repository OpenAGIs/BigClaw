package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1532RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1532BootstrapPlanningResidualAreaStaysPythonFree(t *testing.T) {
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
			t.Fatalf("expected residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1532GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/bootstrap",
		"bigclaw-go/internal/planning",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1532LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1532-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1532",
		"Repository-wide Python file count before: `0`.",
		"Repository-wide Python file count after: `0`.",
		"Focused `workspace/bootstrap/planning` Python file count before: `0`.",
		"Focused `workspace/bootstrap/planning` Python file count after: `0`.",
		"Exact deleted-file ledger: `[]`.",
		"`workspace`: directory absent; scan result `0` Python files.",
		"`bootstrap`: directory absent; scan result `0` Python files.",
		"`planning`: directory absent; scan result `0` Python files.",
		"`bigclaw-go/internal/bootstrap`: `0` Python files.",
		"`bigclaw-go/internal/planning`: `0` Python files.",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1532",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
