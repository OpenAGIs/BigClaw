package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1510RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1510PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1510GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1510LaneArtifactsCaptureZeroPythonReality(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1510-python-asset-sweep.md")
	validation := readRepoFile(t, rootRepo, "reports/BIG-GO-1510-validation.md")
	status := readRepoFile(t, rootRepo, "reports/BIG-GO-1510-status.json")

	for _, needle := range []string{
		"BIG-GO-1510",
		"Repository-wide Python file count: `0`.",
		"Before count: `0`.",
		"After count: `0`.",
		"Deleted Python files in this lane: `none`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run TestBIGGO1510`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}

	for _, needle := range []string{
		"Before count: `0`",
		"After count: `0`",
		"Deleted Python files: `none`",
		"ok  \tbigclaw-go/internal/regression",
	} {
		if !strings.Contains(validation, needle) {
			t.Fatalf("validation report missing substring %q", needle)
		}
	}

	for _, needle := range []string{
		"\"identifier\": \"BIG-GO-1510\"",
		"\"before_count\": 0",
		"\"after_count\": 0",
		"\"deleted_py_files\": []",
		"\"summary\": \"The repository-wide physical Python file count was already zero in this workspace before BIG-GO-1510 changes.\"",
	} {
		if !strings.Contains(status, needle) {
			t.Fatalf("status file missing substring %q", needle)
		}
	}
}
