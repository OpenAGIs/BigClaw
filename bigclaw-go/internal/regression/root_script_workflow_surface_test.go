package regression

import (
	"strings"
	"testing"
)

func TestRootScriptWorkflowAndHooksStayGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	workflow := readRepoFile(t, repoRoot, "workflow.md")
	requiredWorkflow := []string{
		"- Workspace `after_create` now uses the Go-first `scripts/ops/bigclawctl workspace bootstrap` entrypoint",
		"- GitHub sync install/status/sync in this workflow is Go-only through `bash scripts/ops/bigclawctl github-sync ...`; do not call the removed `scripts/ops/bigclaw_github_sync.py` shim.",
		"- Repository `.githooks/post-commit` and `.githooks/post-rewrite` now invoke `go run ./cmd/bigclawctl github-sync sync --repo \"$repo_root\"` from `bigclaw-go`",
	}
	for _, needle := range requiredWorkflow {
		if !strings.Contains(workflow, needle) {
			t.Fatalf("workflow.md missing Go-only root-script guidance %q", needle)
		}
	}

	disallowedWorkflow := []string{
		"`python3 scripts/create_issues.py`",
		"`python3 scripts/dev_smoke.py`",
		"`python3 scripts/ops/bigclaw_github_sync.py`",
		"`python3 scripts/ops/bigclaw_refill_queue.py`",
		"`python3 scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`python3 scripts/ops/symphony_workspace_bootstrap.py`",
		"`python3 scripts/ops/symphony_workspace_validate.py`",
	}
	for _, needle := range disallowedWorkflow {
		if strings.Contains(workflow, needle) {
			t.Fatalf("workflow.md should not reference retired Python execution guidance %q", needle)
		}
	}

	postCommit := readRepoFile(t, repoRoot, ".githooks/post-commit")
	postRewrite := readRepoFile(t, repoRoot, ".githooks/post-rewrite")
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: ".githooks/post-commit", body: postCommit},
		{name: ".githooks/post-rewrite", body: postRewrite},
	} {
		required := []string{
			`cd "$repo_root/bigclaw-go" || exit 0`,
			`go run ./cmd/bigclawctl github-sync sync --json --allow-dirty --repo "$repo_root"`,
		}
		for _, needle := range required {
			if !strings.Contains(tc.body, needle) {
				t.Fatalf("%s missing Go-only sync behavior %q", tc.name, needle)
			}
		}

		disallowed := []string{
			"bigclaw_github_sync.py",
			"python3 ",
		}
		for _, needle := range disallowed {
			if strings.Contains(tc.body, needle) {
				t.Fatalf("%s should not reference retired Python sync surfaces %q", tc.name, needle)
			}
		}
	}
}
