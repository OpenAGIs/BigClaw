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
		switch relativePath {
		case "scripts/ops/bigclaw_refill_queue":
			if !strings.Contains(text, `exec bash "$script_dir/bigclawctl" refill "$@"`) {
				t.Fatalf("expected refill wrapper to dispatch directly to bigclawctl refill: %s", relativePath)
			}
		case "scripts/ops/bigclaw_workspace_bootstrap", "scripts/ops/symphony_workspace_bootstrap":
			if !strings.Contains(text, `exec bash "$script_dir/bigclawctl" workspace bootstrap`) {
				t.Fatalf("expected bootstrap wrapper to dispatch directly to bigclawctl workspace bootstrap: %s", relativePath)
			}
		case "scripts/ops/symphony_workspace_validate":
			if !strings.Contains(text, `exec bash "$script_dir/bigclawctl" workspace validate`) {
				t.Fatalf("expected validate wrapper to dispatch directly to bigclawctl workspace validate: %s", relativePath)
			}
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
		"`bigclawctl refill` is the supported refill path; `scripts/ops/bigclaw_refill_queue` is a shell alias",
		"`bigclawctl workspace bootstrap` is the supported BigClaw workspace bootstrap path; `scripts/ops/bigclaw_workspace_bootstrap` is a shell alias",
		"`bigclawctl workspace bootstrap` is the supported Symphony workspace bootstrap path; `scripts/ops/symphony_workspace_bootstrap` is a shell alias",
		"`bigclawctl workspace validate` is the supported workspace validation path; `scripts/ops/symphony_workspace_validate` is a shell alias",
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
