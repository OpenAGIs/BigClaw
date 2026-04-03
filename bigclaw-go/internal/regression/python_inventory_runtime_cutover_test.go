package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimePythonModuleRemovedFromRepo(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	target := filepath.Join(root, "src", "bigclaw", "runtime.py")

	if _, err := os.Stat(target); err == nil {
		t.Fatalf("expected deleted Python compatibility file to stay absent: %s", target)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat deleted Python compatibility file %s: %v", target, err)
	}
}
