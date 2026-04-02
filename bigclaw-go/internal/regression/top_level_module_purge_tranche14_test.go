package regression

import (
	"os"
	"path/filepath"
	"strings"
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
		fullPath := filepath.Join(repoRoot, relativePath)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatalf("expected shell replacement file to exist: %s (%v)", relativePath, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("expected shell replacement file to be executable: %s", relativePath)
		}
		contents, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("read replacement wrapper %s: %v", relativePath, err)
		}
		text := string(contents)
		if !strings.Contains(text, "bigclawctl") {
			t.Fatalf("expected wrapper to dispatch through bigclawctl: %s", relativePath)
		}
		if strings.Contains(text, "python") || strings.Contains(text, ".py") {
			t.Fatalf("expected wrapper to stay shell/Go-only with no Python path references: %s", relativePath)
		}
	}
}

func TestTopLevelModulePurgeTranche14DocsListOnlyShellOrGoEntrypoints(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	contents, err := os.ReadFile(filepath.Join(repoRoot, "docs/go-cli-script-migration-plan.md"))
	if err != nil {
		t.Fatalf("read migration plan: %v", err)
	}
	text := string(contents)

	required := []string{
		"`scripts/ops/bigclaw_refill_queue` or `bigclawctl refill`",
		"`scripts/ops/bigclaw_workspace_bootstrap` or `bigclawctl workspace bootstrap`",
		"`scripts/ops/symphony_workspace_bootstrap` or `bigclawctl workspace bootstrap`",
		"`scripts/ops/symphony_workspace_validate` or `bigclawctl workspace validate`",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected migration plan to document active shell/Go entrypoint %q", needle)
		}
	}

	disallowed := []string{
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(text, needle) {
			t.Fatalf("migration plan should not advertise deleted Python wrapper %q", needle)
		}
	}
}
