package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche11(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/validation_policy.py",
		"src/bigclaw/memory.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/policy/validation.go",
		"bigclaw-go/internal/policy/validation_test.go",
		"bigclaw-go/internal/policy/memory.go",
		"bigclaw-go/internal/policy/memory_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
