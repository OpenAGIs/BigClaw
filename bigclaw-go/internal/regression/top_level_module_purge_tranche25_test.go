package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche25(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/planning.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	replacementFiles := []string{
		"src/bigclaw/__init__.py",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/governance/freeze.go",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
