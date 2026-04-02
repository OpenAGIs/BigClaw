package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche20(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/__main__.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to stay absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/legacyshim/compilecheck.go",
		"bigclaw-go/internal/legacyshim/compilecheck_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
