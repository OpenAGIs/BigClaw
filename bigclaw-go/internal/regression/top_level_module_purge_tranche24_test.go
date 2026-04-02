package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche24(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/evaluation.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/runtime.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	replacementFiles := []string{
		"src/bigclaw/__init__.py",
		"bigclaw-go/internal/evaluation/evaluation.go",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
