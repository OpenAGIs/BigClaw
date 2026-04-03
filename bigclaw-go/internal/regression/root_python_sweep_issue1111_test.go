package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootPythonSweepIssue1111CandidatesStayDeleted(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	retiredPythonEntrypoints := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, relativePath := range retiredPythonEntrypoints {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired root Python entrypoint to stay absent: %s", relativePath)
		}
	}
}

func TestRootPythonSweepIssue1111DocsDescribeGoReplacements(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")

	required := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"retired `scripts/dev_smoke.py`; use `bigclawctl dev-smoke`",
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`",
		"retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing retired root Python sweep guidance %q", needle)
		}
	}
}
