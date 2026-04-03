package legacyshim

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRepoStateKeepsDeletedPythonMigrationSurfaceAbsent(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))

	if _, err := os.Stat(filepath.Join(repoRoot, "src", "bigclaw", "collaboration.py")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted Python module to stay absent, got err=%v", err)
	}

	var forbidden []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.Name() == "pyproject.toml" || d.Name() == "setup.py" {
			forbidden = append(forbidden, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo state: %v", err)
	}
	if len(forbidden) != 0 {
		t.Fatalf("expected repo to remain free of Python packaging files, found %v", forbidden)
	}
}
