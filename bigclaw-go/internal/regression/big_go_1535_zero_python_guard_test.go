package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1535RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1535ReportingObservabilityResidualAreaStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"src",
		"tests",
		"scripts",
		"bigclaw-go/internal/observability",
		"bigclaw-go/internal/reporting",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected reporting / observability residual area to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1535GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reportstudio/reportstudio.go",
		"bigclaw-go/docs/reports/go-control-plane-observability-report.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1535LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1535-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1535",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused reporting / observability physical Python file count before lane changes: `0`",
		"Focused reporting / observability physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused ledger for reporting / observability: `[]`",
		"`src`: directory not present, so residual Python files = `0`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`scripts`: `0` Python files",
		"`bigclaw-go/internal/observability`: `0` Python files",
		"`bigclaw-go/internal/reporting`: `0` Python files",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/observability/recorder.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/reportstudio/reportstudio.go`",
		"`bigclaw-go/docs/reports/go-control-plane-observability-report.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src tests scripts bigclaw-go/internal/observability bigclaw-go/internal/reporting -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1535",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
