package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO158RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO158PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO158MirroredReportAndExampleSurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	focusedDirs := []string{
		"reports",
		"docs/reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/examples",
	}

	for _, relativeDir := range focusedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected mirrored report or example surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO158RetainedNativeAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"reports/BIG-GO-139-validation.md",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/examples/shadow-task.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO158LaneReportDocumentsPythonAssetSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-158-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-158 Python Asset Sweep",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused mirrored-surface physical Python file count before lane changes: `0`",
		"Focused mirrored-surface physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused ledger for mirrored report/example surfaces: `[]`",
		"`reports`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`reports/BIG-GO-139-validation.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find reports docs/reports bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO158(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|MirroredReportAndExampleSurfacesStayPythonFree|RetainedNativeAssetsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("big-go-158 lane report missing substring %q", needle)
		}
	}
}
