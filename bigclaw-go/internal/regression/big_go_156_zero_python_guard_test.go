package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO156RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO156SupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	supportAssetDirs := []string{
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
		"reports",
	}

	for _, relativeDir := range supportAssetDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected support-asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO156RetainedNativeSupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-budget.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/docs/migration-shadow.md",
		"bigclaw-go/docs/reports/shadow-compare-report.json",
		"bigclaw-go/docs/reports/shadow-matrix-report.json",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/docs/reports/production-corpus-migration-coverage-digest.md",
		"reports/BIG-GO-948-validation.md",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native support asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO156LaneReportDocumentsSupportAssetSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-156-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-156 Python Asset Sweep",
		"Remaining physical Python asset inventory: `0` files.",
		"Support-asset directories audited in this lane:",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-budget.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/docs/migration-shadow.md`",
		"`bigclaw-go/docs/reports/shadow-compare-report.json`",
		"`bigclaw-go/docs/reports/shadow-matrix-report.json`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/production-corpus-migration-coverage-digest.md`",
		"`reports/BIG-GO-948-validation.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/examples bigclaw-go/docs/reports reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO156(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedNativeSupportAssetsRemainAvailable|LaneReportDocumentsSupportAssetSweep)$'`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("big-go-156 lane report missing substring %q", needle)
		}
	}
}
