package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResidualPythonOperatorEntrypointsStayDeleted(t *testing.T) {
	repoRoot := repoRoot(t)
	deleted := []string{
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relative := range deleted {
		if _, err := os.Stat(filepath.Join(repoRoot, relative)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to stay deleted, stat err=%v", relative, err)
		}
	}
}

func TestTrackedOperatorSurfacesStayGoOnly(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		required   []string
		disallowed []string
	}{
		{
			path: "../README.md",
			required: []string{
				"bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json",
				"`bash scripts/ops/bigclawctl workspace validate ...`",
			},
			disallowed: []string{
				"python3 scripts/ops/bigclaw_refill_queue.py",
				"scripts/ops/*workspace*.py",
			},
		},
		{
			path: "../.github/workflows/ci.yml",
			required: []string{
				"bash scripts/ops/bigclawctl refill --help >/dev/null",
				"bash scripts/ops/bigclawctl workspace validate --help >/dev/null",
				"test ! -e scripts/ops/bigclaw_refill_queue.py",
				"test ! -e scripts/ops/symphony_workspace_validate.py",
			},
			disallowed: []string{
				"python3 scripts/ops/bigclaw_refill_queue.py",
				"python3 scripts/ops/symphony_workspace_validate.py",
			},
		},
		{
			path: "../workflow.md",
			required: []string{
				`bash "$SYMPHONY_WORKFLOW_DIR/scripts/ops/bigclawctl" workspace bootstrap`,
			},
			disallowed: []string{
				"bigclaw_workspace_bootstrap.py",
				"symphony_workspace_bootstrap.py",
				"symphony_workspace_validate.py",
				"bigclaw_refill_queue.py",
			},
		},
		{
			path: "../.githooks/post-commit",
			required: []string{
				"bash scripts/ops/bigclawctl github-sync sync --json --allow-dirty",
			},
			disallowed: []string{
				".py",
			},
		},
		{
			path: "../.githooks/post-rewrite",
			required: []string{
				"bash scripts/ops/bigclawctl github-sync sync --json --allow-dirty",
			},
			disallowed: []string{
				".py",
			},
		},
		{
			path: "../docs/go-cli-script-migration-plan.md",
			required: []string{
				"retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`",
				"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bigclawctl workspace bootstrap`",
				"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bigclawctl workspace bootstrap`",
				"retired `scripts/ops/symphony_workspace_validate.py`; use `bigclawctl workspace validate`",
			},
			disallowed: []string{
				"- `scripts/ops/bigclaw_refill_queue.py`",
				"- `scripts/ops/bigclaw_workspace_bootstrap.py`",
				"- `scripts/ops/symphony_workspace_bootstrap.py`",
				"- `scripts/ops/symphony_workspace_validate.py`",
				"python3 scripts/ops/bigclaw_refill_queue.py --help",
				"python3 scripts/ops/symphony_workspace_validate.py --help",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.required {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing required Go-only reference %q", tc.path, needle)
			}
		}
		for _, needle := range tc.disallowed {
			if strings.Contains(contents, needle) {
				t.Fatalf("%s should not reference %q", tc.path, needle)
			}
		}
	}
}
