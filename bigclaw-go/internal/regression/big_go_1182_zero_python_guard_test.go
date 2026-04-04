package regression

import (
	"path/filepath"
	"testing"
)

func TestBIGGO1182RemainingPythonAssetInventoryIsEmpty(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected BIG-GO-1182 remaining Python asset inventory to be empty, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1182PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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
			t.Fatalf("expected BIG-GO-1182 priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}
