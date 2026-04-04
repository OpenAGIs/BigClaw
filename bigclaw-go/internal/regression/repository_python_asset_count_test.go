package regression

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestRepositoryPythonAssetCountIsZero(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	var pythonFiles []string

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			relativePath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository for Python assets: %v", err)
	}

	if len(pythonFiles) != 0 {
		sort.Strings(pythonFiles)
		t.Fatalf("expected repository Python asset count to stay at zero, found %d: %s", len(pythonFiles), strings.Join(pythonFiles, ", "))
	}
}
