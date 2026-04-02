package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche22(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/models.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	replacementFiles := []string{
		"src/bigclaw/__init__.py",
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/workflow/model.go",
		"bigclaw-go/internal/billing/statement.go",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
