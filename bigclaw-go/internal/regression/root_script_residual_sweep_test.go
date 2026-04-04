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
		"Use this to verify the root dev smoke path:",
		"bash scripts/ops/bigclawctl dev-smoke",
		"bash scripts/ops/bigclawctl legacy-python compile-check --json",
	}
	for _, needle := range requiredReadmeEntries {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing active root script replacement guidance %q", needle)
		}
	}
}

func TestBIGGO1175DevBootstrapStaysGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	contents := readRepoFile(t, repoRoot, "scripts/dev_bootstrap.sh")

	required := []string{
		`go test ./cmd/bigclawctl`,
		`bash "$repo_root/scripts/ops/bigclawctl" dev-smoke`,
		`go test ./internal/bootstrap`,
		`bash "$repo_root/scripts/ops/bigclawctl" legacy-python compile-check --repo "$repo_root" --python python3 --json`,
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("scripts/dev_bootstrap.sh missing Go-only validation helper step %q", needle)
		}
	}

	disallowed := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("scripts/dev_bootstrap.sh should not reference retired Python entrypoint %q", needle)
		}
	}
}

func TestBIGGO1175DocsRecordDevBootstrapReplacementEvidence(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	rootPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	goDoc := readRepoFile(t, filepath.Join(repoRoot, "bigclaw-go"), "docs/go-cli-script-migration.md")

	rootPlanRequired := []string{
		"`BIG-GO-1175` records the follow-on replacement evidence",
		"`scripts/dev_bootstrap.sh` remains a shell validation helper",
		"`bash scripts/ops/bigclawctl dev-smoke`",
		"`bash scripts/ops/bigclawctl legacy-python compile-check --json`",
	}
	for _, needle := range rootPlanRequired {
		if !strings.Contains(rootPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1175 replacement evidence %q", needle)
		}
	}

	goDocRequired := []string{
		"Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`, `BIG-GO-1175`",
		"`BIG-GO-1175` extends the root-helper sweep evidence",
		"`scripts/dev_bootstrap.sh`",
		"`bash scripts/ops/bigclawctl dev-smoke`",
	}
	for _, needle := range goDocRequired {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing BIG-GO-1175 evidence %q", needle)
		}
	}
}
