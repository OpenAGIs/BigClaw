package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootOpsPythonWrappersRemoved(t *testing.T) {
	repoRoot := filepath.Dir(repoRoot(t))
	removed := []string{
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	replacements := []string{
		"scripts/ops/bigclaw-github-sync",
		"scripts/ops/bigclaw-refill-queue",
		"scripts/ops/bigclaw-workspace-bootstrap",
		"scripts/ops/symphony-workspace-bootstrap",
		"scripts/ops/symphony-workspace-validate",
	}

	for _, relative := range removed {
		if _, err := os.Stat(filepath.Join(repoRoot, relative)); !os.IsNotExist(err) {
			t.Fatalf("expected removed Python wrapper %s to be absent, err=%v", relative, err)
		}
	}
	for _, relative := range replacements {
		info, err := os.Stat(filepath.Join(repoRoot, relative))
		if err != nil {
			t.Fatalf("expected replacement wrapper %s to exist: %v", relative, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("expected replacement wrapper %s to be executable", relative)
		}
	}
}
