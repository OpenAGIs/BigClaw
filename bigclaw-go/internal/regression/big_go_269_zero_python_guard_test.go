package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO269RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectAuxiliaryPythonLikeFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO269DeeplyNestedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auxiliaryDirs := []string{
		".github/workflows",
		"docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z",
	}

	for _, relativeDir := range auxiliaryDirs {
		pythonFiles := collectAuxiliaryPythonLikeFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected deeply nested auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO269NativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	evidencePaths := []string{
		".github/workflows/ci.yml",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/sqlite-smoke-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
	}

	for _, relativePath := range evidencePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO269LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-269-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-269",
		"Residual auxiliary Python sweep `BIG-GO-269`",
		"Repository-wide Python-like file count: `0`.",
		"`.github/workflows`: `0` Python-like files",
		"`docs/reports`: `0` Python-like files",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z`: `0` Python-like files",
		"`bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z`: `0` Python-like files",
		"`bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z`: `0` Python-like files",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z`: `0` Python-like files",
		"`.github/workflows/ci.yml`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/sqlite-smoke-report.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .github/workflows docs/reports bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO269(RepositoryHasNoPythonFiles|DeeplyNestedAuxiliaryDirectoriesStayPythonFree|NativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
