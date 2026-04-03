package regression

import (
	"strings"
	"testing"
)

func TestRootScriptCutoverDocsStayGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	issuePack := readRepoFile(t, repoRoot, "docs/go-mainline-cutover-issue-pack.md")
	requiredIssuePack := []string{
		"- the final root workspace Python wrappers have since been removed, so the default operator path is now Go-only under `bash scripts/ops/bigclawctl`",
		"- `workflow.md`, `.githooks/post-commit`, and `.githooks/post-rewrite` invoke the Go-first toolchain by default, and the legacy `scripts/ops/bigclaw_github_sync.py` wrapper has been removed",
	}
	for _, needle := range requiredIssuePack {
		if !strings.Contains(issuePack, needle) {
			t.Fatalf("docs/go-mainline-cutover-issue-pack.md missing Go-only root-script guidance %q", needle)
		}
	}

	disallowedIssuePack := []string{
		"`python3 scripts/create_issues.py`",
		"`python3 scripts/dev_smoke.py`",
		"`python3 scripts/ops/bigclaw_github_sync.py`",
		"`python3 scripts/ops/bigclaw_refill_queue.py`",
		"`python3 scripts/ops/symphony_workspace_bootstrap.py`",
		"`python3 scripts/ops/symphony_workspace_validate.py`",
	}
	for _, needle := range disallowedIssuePack {
		if strings.Contains(issuePack, needle) {
			t.Fatalf("docs/go-mainline-cutover-issue-pack.md should not reference retired Python execution guidance %q", needle)
		}
	}

	handoff := readRepoFile(t, repoRoot, "docs/go-mainline-cutover-handoff.md")
	requiredHandoff := []string{
		"- The default mainline posture is Go-first, and this worktree no longer carries",
		"  tracked Python source files or Python entrypoint shims.",
		"- `bash scripts/ops/bigclawctl legacy-python compile-check --json`",
	}
	for _, needle := range requiredHandoff {
		if !strings.Contains(handoff, needle) {
			t.Fatalf("docs/go-mainline-cutover-handoff.md missing Go-only cutover guidance %q", needle)
		}
	}

	disallowedHandoff := []string{
		"`python3 scripts/create_issues.py`",
		"`python3 scripts/dev_smoke.py`",
		"`python3 scripts/ops/bigclaw_github_sync.py`",
		"`python3 scripts/ops/bigclaw_refill_queue.py`",
		"`python3 scripts/ops/symphony_workspace_bootstrap.py`",
		"`python3 scripts/ops/symphony_workspace_validate.py`",
	}
	for _, needle := range disallowedHandoff {
		if strings.Contains(handoff, needle) {
			t.Fatalf("docs/go-mainline-cutover-handoff.md should not reference retired Python execution guidance %q", needle)
		}
	}
}
