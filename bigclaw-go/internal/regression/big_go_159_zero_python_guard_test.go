package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO159RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO159PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO159OverlookedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		".github",
		".symphony",
		"docs/reports",
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected overlooked auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO159NativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".github/workflows/ci.yml",
		".symphony/workpad.md",
		"docs/reports/bootstrap-cache-validation.md",
		"reports/BIG-FOUNDATION-validation.md",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO159LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-159-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-159",
		"Repository-wide Python asset count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`.github`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`.github/workflows/ci.yml`",
		"`.symphony/workpad.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`reports/BIG-FOUNDATION-validation.md`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name '*.ipynb' \\) -print | sort`",
		"`find .github .symphony docs/reports reports bigclaw-go/docs/reports bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO159(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|OverlookedAuxiliaryDirectoriesStayPythonFree|NativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
