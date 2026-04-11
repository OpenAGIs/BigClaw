package regression

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var pythonFileExtensions = map[string]struct{}{
	".ipynb": {},
	".pxd":   {},
	".pxi":   {},
	".py":    {},
	".pyi":   {},
	".pyw":   {},
	".pyx":   {},
}

func TestBIGGO1174RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1174PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func collectPythonFiles(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := pythonFileExtensions[filepath.Ext(path)]; !ok {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}
