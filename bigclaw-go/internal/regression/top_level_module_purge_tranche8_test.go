package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche8(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python shim to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/legacyshim/wrappers.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}
