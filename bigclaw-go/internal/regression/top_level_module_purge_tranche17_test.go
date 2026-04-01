package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche17(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/repo_links.py",
		"src/bigclaw/repo_plane.py",
		"tests/test_repo_links.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/repo/plane.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}
