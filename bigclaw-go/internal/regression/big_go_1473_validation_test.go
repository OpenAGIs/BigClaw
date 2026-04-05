package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1473ZeroPythonBaselineAndReplacementOwnership(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	if pythonFiles := collectPythonFiles(t, repoRoot); len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}

	deletedPythonFiles := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected migrated Python path to remain absent: %s", relativePath)
		}
	}

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/refill/queue.go",
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement ownership path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1473ValidationReportCapturesBlockedPhysicalDeletionState(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	report := readRepoFile(t, repoRoot, "reports/BIG-GO-1473-validation.md")
	migrationPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")

	for _, needle := range []string{
		"`BIG-GO-1473` audited the remaining repo-level Python script migration surface",
		"repository-wide physical `.py` count is `0`",
		"`scripts/create_issues.py` -> `bigclaw-go/cmd/bigclawctl` `create-issues`",
		"`scripts/dev_smoke.py` -> `bigclaw-go/cmd/bigclawctl` `dev-smoke`",
		"`scripts/ops/bigclaw_github_sync.py` -> `bigclaw-go/cmd/bigclawctl` `github-sync`",
		"`bigclaw-go/scripts/benchmark/capacity_certification.py` -> `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go` `automation benchmark capacity-certification`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go` `automation e2e run-task-smoke`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go` `automation migration shadow-compare`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1473ZeroPythonBaselineAndReplacementOwnership|BIGGO1473ValidationReportCapturesBlockedPhysicalDeletionState)$'`",
		"cannot honestly claim a fresh file-count reduction in this checkout",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("reports/BIG-GO-1473-validation.md missing substring %q", needle)
		}
	}

	if !strings.Contains(migrationPlan, "`BIG-GO-1473` re-audits the same script migration surface") {
		t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1473 current-state note")
	}
}
