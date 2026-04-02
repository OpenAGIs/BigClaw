package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpsScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	opsDir := filepath.Join(repoRoot, "scripts", "ops")

	entries, err := os.ReadDir(opsDir)
	if err != nil {
		t.Fatalf("read ops script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected no Python helper in scripts/ops, found %s", entry.Name())
		}
	}
}

func TestOpsMigrationDocsListOnlyActiveEntrypoints(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	readme := readRepoFile(t, repoRoot, "README.md")
	plan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	combined := readme + "\n" + plan

	required := []string{
		"`bash scripts/ops/bigclaw_refill_queue ...`",
		"`bash scripts/ops/bigclaw_workspace_bootstrap ...`",
		"`bash scripts/ops/symphony_workspace_bootstrap ...`",
		"`bash scripts/ops/symphony_workspace_validate ...`",
		"`bash scripts/ops/bigclaw_refill_queue --help`",
		"`bash scripts/ops/bigclaw_workspace_bootstrap --help`",
		"`bash scripts/ops/symphony_workspace_bootstrap --help`",
		"`bash scripts/ops/symphony_workspace_validate --help`",
	}
	for _, needle := range required {
		if !strings.Contains(combined, needle) {
			t.Fatalf("expected active ops entrypoint %q in README or migration plan", needle)
		}
	}

	disallowed := []string{
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
		"python3 scripts/ops/bigclaw_refill_queue.py --help",
		"python3 scripts/ops/symphony_workspace_validate.py --help",
	}
	for _, needle := range disallowed {
		if strings.Contains(combined, needle) {
			t.Fatalf("README or migration plan should not reference removed Python helper %q", needle)
		}
	}
}
