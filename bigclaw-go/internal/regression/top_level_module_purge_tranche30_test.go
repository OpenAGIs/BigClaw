package regression

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"bigclaw-go/internal/legacyshim"
)

func TestTopLevelModulePurgeTranche30(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	pythonFiles := relativePythonFiles(t, repoRoot)
	wantPythonFiles := []string{
		"src/bigclaw/__init__.py",
	}
	if !reflect.DeepEqual(pythonFiles, wantPythonFiles) {
		t.Fatalf("unexpected remaining python files: got=%v want=%v", pythonFiles, wantPythonFiles)
	}

	wantCompileCheck := []string{
		filepath.Join(repoRoot, "src/bigclaw/__init__.py"),
	}
	if got := legacyshim.FrozenCompileCheckFiles(repoRoot); !reflect.DeepEqual(got, wantCompileCheck) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, wantCompileCheck)
	}
}

func relativePythonFiles(t *testing.T, repoRoot string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(filepath.Join(repoRoot, "src"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}
		relativePath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		t.Fatalf("walk python files: %v", err)
	}
	sort.Strings(files)
	return files
}
