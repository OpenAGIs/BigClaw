package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyPythonRuntimeFilesStayRemoved(t *testing.T) {
	root := repoRoot(t)
	for _, relative := range []string{
		"src/bigclaw/runtime.py",
		"pyproject.toml",
		"setup.py",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s should be absent, stat err=%v", relative, err)
		}
	}
}
