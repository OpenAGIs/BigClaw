package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1241RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1241PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1241ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1241LaneReportDocumentsPythonAssetSweep(t *testing.T) {
	goRepoRoot := repoRoot(t)
	report := readRepoFile(t, goRepoRoot, "docs/reports/big-go-1241-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-1241 Python Asset Sweep",
		"Remaining physical Python asset inventory: `0` files.",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1241(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`",
		"Result: `0`",
		"bigclaw-go/internal/regression",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("docs/reports/big-go-1241-python-asset-sweep.md missing substring %q", needle)
		}
	}
}
