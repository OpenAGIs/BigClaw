package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1235RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1235PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1235ReadmeStaysGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	readme := readRepoFile(t, rootRepo, "README.md")

	for _, needle := range []string{
		"bash scripts/ops/bigclawctl dev-smoke",
		"bash scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
	} {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing Go-only replacement guidance %q", needle)
		}
	}

	if strings.Contains(readme, "legacy-python compile-check") {
		t.Fatalf("README.md should not present retired legacy-python compile-check guidance")
	}
}
