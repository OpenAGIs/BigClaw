package regression

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return wd
}

func regressionRepoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(repoRoot(t), "..", "..", ".."))
}

func readRepoFile(t *testing.T, rootRepo, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return string(content)
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
		if filepath.Ext(path) != ".py" {
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
