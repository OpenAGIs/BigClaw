package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootPackagingAndPythonShimEntrypointsStayRemoved(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	for _, relative := range []string{
		"pyproject.toml",
		"setup.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s should be absent, err=%v", relative, err)
		}
	}
}

func TestRootCutoverSurfacesStayGoOnly(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	for _, tc := range []struct {
		path        string
		substrings  []string
		notContains []string
	}{
		{
			path: "README.md",
			substrings: []string{
				"make test",
				"make build",
				"bash scripts/ops/bigclawctl github-sync",
				"bash scripts/ops/bigclawctl refill",
				"bash scripts/ops/bigclawctl workspace",
			},
			notContains: []string{
				"python3 scripts/ops/bigclaw_github_sync.py",
				"python3 scripts/ops/bigclaw_refill_queue.py",
				"scripts/ops/*workspace*.py",
				"python3 -m pytest",
				"pre-commit run --all-files",
				"BIGCLAW_ENABLE_LEGACY_PYTHON",
			},
		},
		{
			path: ".github/workflows/ci.yml",
			substrings: []string{
				"actions/setup-go",
				"make test",
				"make build",
			},
			notContains: []string{
				"actions/setup-python",
				"pip install pytest",
				"pytest --cov",
				"ruff check",
			},
		},
		{
			path: "scripts/dev_bootstrap.sh",
			substrings: []string{
				"go test ./cmd/bigclawctl",
				"bash \"$repo_root/scripts/ops/bigclawctl\" dev-smoke",
				"go test ./internal/bootstrap",
			},
			notContains: []string{
				"BIGCLAW_ENABLE_LEGACY_PYTHON",
				"python3 -m pytest",
				"PYTHONPATH=",
			},
		},
		{
			path: ".githooks/post-commit",
			substrings: []string{
				"bash scripts/ops/bigclawctl github-sync sync --json --allow-dirty",
			},
			notContains: []string{
				"PYTHONDONTWRITEBYTECODE",
			},
		},
		{
			path: ".githooks/post-rewrite",
			substrings: []string{
				"bash scripts/ops/bigclawctl github-sync sync --json --allow-dirty",
			},
			notContains: []string{
				"PYTHONDONTWRITEBYTECODE",
			},
		},
	} {
		contents := readRepoFile(t, root, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
		for _, needle := range tc.notContains {
			if strings.Contains(contents, needle) {
				t.Fatalf("%s still contains %q", tc.path, needle)
			}
		}
	}
}
