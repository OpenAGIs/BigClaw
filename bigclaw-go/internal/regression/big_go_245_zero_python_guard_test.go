package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO245RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO245ToolingDocsStayGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	type docCheck struct {
		path      string
		required  []string
		forbidden []string
	}

	checks := []docCheck{
		{
			path: "README.md",
			required: []string{
				"## Root helper status",
				"The repository root now keeps a small Go-and-shell helper surface only.",
				"root workspace wrappers are retired; use `bash scripts/ops/bigclawctl workspace ...`.",
				"GitHub sync is exposed through `bash scripts/ops/bigclawctl github-sync ...`.",
			},
			forbidden: []string{
				"## Python asset status",
				"root workspace Python helpers",
				"Python wrapper",
				"Python ops wrappers should stay deleted",
			},
		},
		{
			path: "docs/go-cli-script-migration-plan.md",
			required: []string{
				"retired root issue bootstrap helper; use `bigclawctl create-issues`",
				"retired GitHub sync helper; use `bigclawctl github-sync`",
				"retired workspace bootstrap helper; use `bash scripts/ops/bigclawctl workspace bootstrap`",
				"retired workspace validation helper; use `bash scripts/ops/bigclawctl workspace validate`",
			},
			forbidden: []string{
				"scripts/create_issues.py",
				"scripts/dev_smoke.py",
				"scripts/ops/bigclaw_github_sync.py",
				"scripts/ops/bigclaw_workspace_bootstrap.py",
				"scripts/ops/symphony_workspace_bootstrap.py",
				"scripts/ops/symphony_workspace_validate.py",
			},
		},
		{
			path: "bigclaw-go/docs/go-cli-script-migration.md",
			required: []string{
				"`bigclaw-go/scripts/e2e/` is now a Go-only operator surface.",
				"The repo root stays Go-and-shell only.",
				"helper-side tests",
			},
			forbidden: []string{
				"Python-free operator surface",
				"remaining Python candidate paths",
				"Python-side tests",
			},
		},
	}

	for _, check := range checks {
		content := readRepoFile(t, rootRepo, check.path)
		for _, needle := range check.required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing required tooling-doc substring %q", check.path, needle)
			}
		}
		for _, needle := range check.forbidden {
			if strings.Contains(content, needle) {
				t.Fatalf("%s still contains retired tooling-doc substring %q", check.path, needle)
			}
		}
	}
}

func TestBIGGO245ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"README.md",
		"docs/go-cli-script-migration-plan.md",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO245LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-245-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-245",
		"Residual tooling Python sweep T",
		"Repository-wide Python file count: `0`.",
		"`README.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`rg -n \"scripts/create_issues",
		"README.md docs/go-cli-script-migration-plan.md bigclaw-go/docs/go-cli-script-migration.md`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO245(RepositoryHasNoPythonFiles|ToolingDocsStayGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
