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
		"`BIG-GO-1163` keeps the root residual sweep closed",
		"`find . -name '*.py' | wc -l` already returns `0`",
		"branch baseline, so this lane records and enforces the zero-count state",
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

func TestRootScriptResidualSweepRepoWidePythonCountZero(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	var pythonFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			relativePath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for python files: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repo-wide Python file count to remain zero, found %d: %v", len(pythonFiles), pythonFiles)
	}
}
