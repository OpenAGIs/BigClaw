package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1512RootOpsPythonWrappersRemoved(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python ops wrapper to stay absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1512RootOpsDocsUseGoEntrypoints(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	readme := readRepoFile(t, repoRoot, "README.md")
	requiredReadmeEntries := []string{
		"bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json",
		"bash scripts/ops/bigclawctl workspace bootstrap",
		"bash scripts/ops/bigclawctl workspace validate",
	}
	for _, needle := range requiredReadmeEntries {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing Go entrypoint guidance %q", needle)
		}
	}
	for _, removed := range []string{
		"python3 scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/*workspace*.py",
	} {
		if strings.Contains(readme, removed) {
			t.Fatalf("README.md should not reference removed Python ops wrapper guidance %q", removed)
		}
	}

	migrationPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"retired the refill Python wrapper; use `bigclawctl refill`",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
		"`bash scripts/ops/bigclawctl refill --help`",
		"`bash scripts/ops/bigclawctl workspace bootstrap --help`",
		"`bash scripts/ops/bigclawctl workspace validate --help`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing root ops wrapper migration guidance %q", needle)
		}
	}
	for _, removed := range []string{
		"`scripts/ops/bigclaw_refill_queue.py` -> `bigclawctl refill`",
		"`python3 scripts/ops/bigclaw_refill_queue.py --help`",
		"`python3 scripts/ops/symphony_workspace_validate.py --help`",
	} {
		if strings.Contains(migrationPlan, removed) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not retain removed wrapper guidance %q", removed)
		}
	}
}
