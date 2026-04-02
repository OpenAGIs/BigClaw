package regression

import (
	"io/fs"
	"path/filepath"
	"testing"
)

func TestPhysicalPythonFloorLockedToZero(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	var pythonFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".symphony":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".py" {
			relativePath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			pythonFiles = append(pythonFiles, relativePath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for python files: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected zero physical Python files, got %d: %v", len(pythonFiles), pythonFiles)
	}
}
