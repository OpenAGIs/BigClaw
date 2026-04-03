package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyRuntimeResidueStaysPurged(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	for _, relativePath := range []string{
		"src/bigclaw/runtime.py",
		"src/bigclaw/deprecation.py",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python runtime residue to be absent: %s", relativePath)
		}
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "bigclaw-go/internal/worker/runtime.go")); err != nil {
		t.Fatalf("expected Go runtime owner to exist: %v", err)
	}
}
