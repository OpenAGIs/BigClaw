package regression

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1172PrioritizedSweepAreasStayPythonFree(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	for _, dir := range []string{
		filepath.Join(rootRepo, "src", "bigclaw"),
		filepath.Join(rootRepo, "tests"),
		filepath.Join(rootRepo, "scripts"),
		filepath.Join(goRepoRoot, "scripts"),
	} {
		assertPythonFreeDir(t, dir)
	}
}

func TestBIGGO1172RepositoryWidePythonCountIsZero(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	pythonFiles := make([]string, 0)

	err := filepath.WalkDir(rootRepo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			relativePath, relErr := filepath.Rel(rootRepo, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository-wide python count to stay at zero, found %d files: %v", len(pythonFiles), pythonFiles)
	}
}

func assertPythonFreeDir(t *testing.T, dir string) {
	t.Helper()

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			return fs.ErrExist
		}
		return nil
	})
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "no such file or directory") {
		return
	}
	if err == fs.ErrExist {
		t.Fatalf("expected %s to remain Python-free", dir)
	}
	t.Fatalf("walk %s: %v", dir, err)
}
