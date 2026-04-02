package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche23(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/console_ia.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	replacementFiles := []string{
		"src/bigclaw/design_system.py",
		"bigclaw-go/internal/consoleia/consoleia.go",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
