package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1515RepositoryPythonInventoryStaysZero(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1515ReportingAndObservabilityPathsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	paths := []string{
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/reporting",
		"bigclaw-go/internal/observability",
	}

	for _, relativePath := range paths {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected reporting/observability path to remain Python-free: %s (%v)", relativePath, pythonFiles)
		}
	}
}

func TestBIGGO1515LedgerCapturesBlockedDeletionState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1515-reporting-observability-python-ledger.md")

	for _, needle := range []string{
		"BIG-GO-1515",
		"Repository-wide physical `.py` file count before lane work: `0`",
		"Repository-wide physical `.py` file count after lane work: `0`",
		"Net physical `.py` files removed by this lane: `0`",
		"Exact deleted-file ledger: `none`",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/reporting`: `0` Python files",
		"`bigclaw-go/internal/observability`: `0` Python files",
		"already Python-free",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find reports bigclaw-go/docs/reports bigclaw-go/internal/reporting bigclaw-go/internal/observability -type f -name '*.py' 2>/dev/null | sort`",
		"`git diff --name-status --diff-filter=D`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1515",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
