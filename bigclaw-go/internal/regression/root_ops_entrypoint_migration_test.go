package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootOpsDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	opsDir := filepath.Join(repoRoot, "scripts", "ops")

	entries, err := os.ReadDir(opsDir)
	if err != nil {
		t.Fatalf("read ops script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isPythonAssetPath(entry.Name()) {
			t.Fatalf("expected no Python asset in scripts/ops, found %s", entry.Name())
		}
	}
}

func TestRootOpsMigrationDocsListOnlyGoEntrypoints(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")

	required := []string{
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
		"`bash scripts/ops/bigclawctl workspace validate --help`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing active entrypoint guidance %q", needle)
		}
	}

	disallowed := []string{
		"- `scripts/ops/bigclaw_workspace_bootstrap.py`",
		"- `scripts/ops/symphony_workspace_bootstrap.py`",
		"- `scripts/ops/symphony_workspace_validate.py`",
		"- `python3 scripts/ops/symphony_workspace_validate.py --help`",
		"Python `scripts/ops/*_*.py` shims only translate legacy flags/defaults",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not reference retired Python workspace shim guidance %q", needle)
		}
	}
}
