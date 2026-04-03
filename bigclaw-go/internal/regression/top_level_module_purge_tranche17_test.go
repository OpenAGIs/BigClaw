package regression

import (
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche17(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")

	required := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`",
		"retired the refill Python wrapper; use `bigclawctl refill`",
		"`bash scripts/ops/bigclawctl dev-smoke`",
		"`bash scripts/ops/bigclawctl github-sync status --json`",
		"`bash scripts/ops/bigclawctl refill --help`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing active entrypoint guidance %q", needle)
		}
	}

	disallowed := []string{
		"- `scripts/create_issues.py`",
		"- `scripts/dev_smoke.py`",
		"`scripts/ops/bigclaw_github_sync.py --help`",
		"- `scripts/ops/bigclaw_refill_queue.py`",
		"`python3 scripts/create_issues.py`",
		"`python3 scripts/dev_smoke.py`",
		"`python3 scripts/ops/bigclaw_github_sync.py`",
		"`python3 scripts/ops/bigclaw_refill_queue.py`",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not reference retired Python guidance %q", needle)
		}
	}
}
