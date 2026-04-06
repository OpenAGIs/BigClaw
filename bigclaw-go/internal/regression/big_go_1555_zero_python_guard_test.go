package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1555RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1555ReportingObservabilityResidualSurfaceStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"src",
		"bigclaw-go/internal/observability",
		"bigclaw-go/internal/reporting",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected reporting/observability residual area to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1555GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/regression/regression.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1555LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1555-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1555",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused reporting/observability physical Python file count before lane changes: `0`",
		"Focused reporting/observability physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused reporting/observability ledger: `[]`",
		"`src`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/internal/observability`: `0` Python files",
		"`bigclaw-go/internal/reporting`: `0` Python files",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/regression/regression.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src bigclaw-go/internal/observability bigclaw-go/internal/reporting -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1555",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
