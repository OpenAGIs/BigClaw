package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1563RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1563PythonTestsDirectoryStaysRemoved(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	_, err := os.Stat(filepath.Join(rootRepo, "tests"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted Python tests directory to stay absent: %v", err)
	}
}

func TestBIGGO1563GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/orchestrator/loop_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1563LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1563-python-test-tranche-a.md")

	for _, needle := range []string{
		"BIG-GO-1563",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused tranche A residual Python file count before lane changes: `0`",
		"Focused tranche A residual Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused tranche A deleted-file ledger: `[]`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/intake/connector_test.go`",
		"`bigclaw-go/internal/planning/planning_test.go`",
		"`bigclaw-go/internal/reporting/reporting_test.go`",
		"`bigclaw-go/internal/orchestrator/loop_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/internal/observability bigclaw-go/internal/intake bigclaw-go/internal/planning bigclaw-go/internal/reporting bigclaw-go/internal/orchestrator bigclaw-go/internal/regression -type f \\( -name 'audit_test.go' -o -name 'connector_test.go' -o -name 'planning_test.go' -o -name 'reporting_test.go' -o -name 'loop_test.go' -o -name 'python_test_tranche17_removal_test.go' \\) | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1563",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
