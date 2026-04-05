package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche16(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
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

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}

	compatSymlinks := []string{
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
	}
	for _, relativePath := range compatSymlinks {
		path := filepath.Join(repoRoot, relativePath)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("expected compatibility symlink to exist: %s (%v)", relativePath, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected compatibility path to remain a symlink: %s", relativePath)
		}
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatalf("read compatibility symlink: %s (%v)", relativePath, err)
		}
		if target != "bigclawctl" {
			t.Fatalf("expected %s to target bigclawctl, got %q", relativePath, target)
		}
	}
}
