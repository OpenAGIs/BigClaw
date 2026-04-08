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
		"retired the refill wrapper; use `bigclawctl refill`",
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
		"Use this to verify the root dev smoke path:",
		"bash scripts/ops/bigclawctl dev-smoke",
		"bash scripts/dev_bootstrap.sh",
		"root workspace helpers are retired; use `bash scripts/ops/bigclawctl workspace ...`",
		"GitHub sync is no longer exposed through a legacy wrapper; use",
		"ops wrappers should stay deleted and GitHub sync is Go/shell-only via",
	}
	for _, needle := range requiredReadmeEntries {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing active root script replacement guidance %q", needle)
		}
	}
	disallowedReadmeEntries := []string{
		"pre-commit run --all-files",
	}
	for _, needle := range disallowedReadmeEntries {
		if strings.Contains(readme, needle) {
			t.Fatalf("README.md should not reference retired Python-adjacent tooling guidance %q", needle)
		}
	}
	if strings.Contains(readme, "legacy-python compile-check") {
		t.Fatalf("README.md should not reference retired legacy-python compile-check guidance")
	}

	bootstrapTemplate := readRepoFile(t, repoRoot, "docs/symphony-repo-bootstrap-template.md")
	requiredBootstrapEntries := []string{
		"`scripts/ops/bigclawctl`",
		"`src/<your_package>/workspace_bootstrap.*`",
		"`src/<your_package>/workspace_bootstrap_cli.*`",
		"`bash scripts/ops/bigclawctl workspace validate`",
	}
	for _, needle := range requiredBootstrapEntries {
		if !strings.Contains(bootstrapTemplate, needle) {
			t.Fatalf("docs/symphony-repo-bootstrap-template.md missing active bootstrap guidance %q", needle)
		}
	}
	disallowedBootstrapEntries := []string{
		"Python compatibility package path",
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
	}
	for _, needle := range disallowedBootstrapEntries {
		if strings.Contains(bootstrapTemplate, needle) {
			t.Fatalf("docs/symphony-repo-bootstrap-template.md should not retain Python-specific bootstrap template wording %q", needle)
		}
	}

	refillQueue := readRepoFile(t, repoRoot, "docs/parallel-refill-queue.md")
	requiredRefillQueueEntries := []string{
		"new implementation work lands in `bigclaw-go`",
		"legacy migration-only paths stay out of the default developer workflow unless explicitly marked otherwise",
		"queue promotion is handled by `bigclawctl refill`",
	}
	for _, needle := range requiredRefillQueueEntries {
		if !strings.Contains(refillQueue, needle) {
			t.Fatalf("docs/parallel-refill-queue.md missing active refill guidance %q", needle)
		}
	}
	if strings.Contains(refillQueue, "Python paths are migration-only unless explicitly marked otherwise") {
		t.Fatal("docs/parallel-refill-queue.md should not frame the active refill workflow around Python-specific path guidance")
	}

	cutoverHandoff := readRepoFile(t, repoRoot, "docs/go-mainline-cutover-handoff.md")
	requiredHandoffEntries := []string{
		"Historical cutover validation also included legacy shim assertions at merge",
		"that Python-side check is now retired",
		"active developer workflow",
	}
	for _, needle := range requiredHandoffEntries {
		if !strings.Contains(cutoverHandoff, needle) {
			t.Fatalf("docs/go-mainline-cutover-handoff.md missing retired Python validation guidance %q", needle)
		}
	}
	disallowedHandoffEntries := []string{
		"PYTHONPATH=src python3 - <<",
	}
	for _, needle := range disallowedHandoffEntries {
		if strings.Contains(cutoverHandoff, needle) {
			t.Fatalf("docs/go-mainline-cutover-handoff.md should not present retired Python validation command %q", needle)
		}
	}
}
