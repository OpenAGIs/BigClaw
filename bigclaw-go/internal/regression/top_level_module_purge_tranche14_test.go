package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	shellReplacementFiles := []string{
		"scripts/ops/bigclaw_refill_queue",
		"scripts/ops/bigclaw_workspace_bootstrap",
		"scripts/ops/symphony_workspace_bootstrap",
		"scripts/ops/symphony_workspace_validate",
	}
	for _, relativePath := range shellReplacementFiles {
		info, err := os.Stat(filepath.Join(repoRoot, relativePath))
		if err != nil {
			t.Fatalf("expected shell replacement file to exist: %s (%v)", relativePath, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("expected shell replacement file to be executable: %s", relativePath)
		}
	}
}
