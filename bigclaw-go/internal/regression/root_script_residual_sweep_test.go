package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootScriptResidualSweep(t *testing.T) {
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
			t.Fatalf("expected deleted Python script to be absent: %s", relativePath)
		}
	}

	deletedPythonBuildHelpers := []string{
		"setup.py",
		"pyproject.toml",
		".pre-commit-config.yaml",
	}
	for _, relativePath := range deletedPythonBuildHelpers {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired root Python build helper to be absent: %s", relativePath)
		}
	}

	goCompatibilityFiles := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
	}
	for _, relativePath := range goCompatibilityFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement or compatibility path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestRootScriptResidualSweepDocs(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	migrationPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`",
		"retired the refill Python wrapper; use `bigclawctl refill`",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing root sweep migration guidance %q", needle)
		}
	}

	readme := readRepoFile(t, repoRoot, "README.md")
	requiredReadmeEntries := []string{
		"The repository root no longer carries physical `.py` assets",
		"Python build",
		"Use this to verify the root dev smoke path:",
		"bash scripts/ops/bigclawctl dev-smoke",
		"bash scripts/dev_bootstrap.sh",
		"git diff --check",
		"find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort",
	}
	for _, needle := range requiredReadmeEntries {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing active root script replacement guidance %q", needle)
		}
	}
	if strings.Contains(readme, "legacy-python compile-check") {
		t.Fatalf("README.md should not reference retired legacy-python compile-check guidance")
	}
	if strings.Contains(readme, "pre-commit run --all-files") {
		t.Fatalf("README.md should not reference retired pre-commit hygiene guidance")
	}
}
