package regression

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1178RepositoryStaysPythonFree(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	var pythonFiles []string

	err := filepath.WalkDir(rootRepo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
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
		t.Fatalf("walk repository for python assets: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to stay free of physical .py files, found %v", pythonFiles)
	}
}
