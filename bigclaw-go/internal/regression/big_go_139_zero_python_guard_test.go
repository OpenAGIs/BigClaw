package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO139RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO139PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO139ReportHeavyAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	reportDirs := []string{
		"reports",
		"docs/reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range reportDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected report-heavy auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO139RetainedNativeReportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"reports/BIG-GO-1274-validation.md",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native report asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO139LaneReportDocumentsPythonAssetSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-139-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-139 Python Asset Sweep",
		"Remaining physical Python asset inventory: `0` files.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Report-heavy auxiliary directories audited in this lane:",
		"`reports`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`reports/BIG-GO-1274-validation.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find reports docs/reports bigclaw-go/docs/reports bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO139(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReportHeavyAuxiliaryDirectoriesStayPythonFree|RetainedNativeReportAssetsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("big-go-139 lane report missing substring %q", needle)
		}
	}
}
