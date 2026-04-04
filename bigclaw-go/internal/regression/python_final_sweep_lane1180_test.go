package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1180RepositoryStaysPythonFree(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	var pythonFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == ".symphony") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".py") {
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
		t.Fatalf("expected final Python sweep lane to keep repo python-free, found %v", pythonFiles)
	}

	requiredReplacementFiles := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}
	for _, relativePath := range requiredReplacementFiles {
		assertLane1180RepoPathExists(t, repoRoot, relativePath)
	}
}

func TestBIGGO1180MigrationDocsCaptureFinalSweepState(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	migrationPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"`BIG-GO-1180` closes the final Go-only Python removal sweep lane",
		"already returns `0` in the branch baseline for this lane",
		"The repo-wide Python-free state is now enforced by regression",
		"operator replacements stay on `bigclawctl`",
		"retained shell wrappers",
		"Go/native automation entrypoints under `bigclaw-go/scripts/`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1180 final sweep evidence %q", needle)
		}
	}

	readme := readRepoFile(t, repoRoot, "README.md")
	requiredReadmeEntries := []string{
		"Final Python asset sweep status:",
		"`find . -name '*.py' | wc -l` should return `0` from the repository root.",
		"Supported replacements remain `bash scripts/ops/bigclawctl ...`, `bash scripts/dev_bootstrap.sh`, and the Go/native `bigclaw-go/scripts/*` entrypoints.",
	}
	for _, needle := range requiredReadmeEntries {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing BIG-GO-1180 final sweep guidance %q", needle)
		}
	}
}

func assertLane1180RepoPathExists(t *testing.T, repoRoot string, relativePath string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
		t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
	}
}
