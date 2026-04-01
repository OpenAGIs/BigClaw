package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/__main__.py",
		"src/bigclaw/deprecation.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/worker/runtime.go",
		"src/bigclaw/runtime.py",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement or retained compatibility file to exist: %s (%v)", relativePath, err)
		}
	}
}
