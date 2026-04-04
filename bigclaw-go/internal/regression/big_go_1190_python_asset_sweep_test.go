package regression

import (
	"path/filepath"
	"reflect"
	"testing"

	"bigclaw-go/internal/legacyshim"
)

func TestBIGGO1190RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1190PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

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

func TestBIGGO1190LegacyCompileCheckTargetsRemainEmpty(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	if files := legacyshim.FrozenCompileCheckFiles(rootRepo); !reflect.DeepEqual(files, []string{}) {
		t.Fatalf("expected no retained legacy compile-check files, got %v", files)
	}
}
