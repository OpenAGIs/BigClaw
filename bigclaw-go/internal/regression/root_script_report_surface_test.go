package regression

import (
	"strings"
	"testing"
)

func TestRootScriptReportsStayGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	reportPaths := []string{
		"reports/BIG-GO-902-closeout.md",
		"reports/BIG-GO-902-pr.md",
		"reports/BIG-GO-902-status.json",
		"reports/BIG-GO-902-validation.md",
	}

	required := []string{
		"scripts/ops/bigclawctl legacy-python compile-check --json",
		"scripts/ops/bigclawctl create-issues --help",
		"scripts/ops/bigclawctl github-sync --help",
		"scripts/ops/bigclawctl workspace bootstrap --help",
		"scripts/ops/bigclawctl refill --help",
		"scripts/ops/bigclawctl workspace validate --help",
		"scripts/ops/bigclawctl github-sync status --json",
	}
	disallowed := []string{
		"python3 scripts/dev_smoke.py",
		"python3 scripts/create_issues.py",
		"python3 scripts/ops/bigclaw_github_sync.py",
		"python3 scripts/ops/bigclaw_refill_queue.py",
		"python3 scripts/ops/bigclaw_workspace_bootstrap.py",
		"python3 scripts/ops/symphony_workspace_bootstrap.py",
		"python3 scripts/ops/symphony_workspace_validate.py",
	}

	for _, path := range reportPaths {
		body := readRepoFile(t, repoRoot, path)
		for _, needle := range required {
			if !strings.Contains(body, needle) {
				t.Fatalf("%s missing Go-only root-script report guidance %q", path, needle)
			}
		}
		for _, needle := range disallowed {
			if strings.Contains(body, needle) {
				t.Fatalf("%s should not retain retired Python root-script guidance %q", path, needle)
			}
		}
	}
}
