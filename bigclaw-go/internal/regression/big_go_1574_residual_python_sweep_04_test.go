package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1574ResidualPythonSweep04(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	candidates := []struct {
		path         string
		replacements []string
	}{
		{
			path: "src/bigclaw/collaboration.py",
			replacements: []string{
				"bigclaw-go/internal/collaboration/thread.go",
				"bigclaw-go/internal/collaboration/thread_test.go",
			},
		},
		{
			path: "src/bigclaw/github_sync.py",
			replacements: []string{
				"bigclaw-go/internal/githubsync/sync.go",
				"bigclaw-go/internal/githubsync/sync_test.go",
			},
		},
		{
			path: "src/bigclaw/pilot.py",
			replacements: []string{
				"bigclaw-go/internal/pilot/report.go",
				"bigclaw-go/internal/pilot/report_test.go",
			},
		},
		{
			path: "src/bigclaw/repo_triage.py",
			replacements: []string{
				"bigclaw-go/internal/repo/triage.go",
				"bigclaw-go/docs/reports/big-go-1362-repo-module-removal-sweep.md",
			},
		},
		{
			path: "src/bigclaw/validation_policy.py",
			replacements: []string{
				"bigclaw-go/internal/policy/validation.go",
				"bigclaw-go/internal/policy/validation_test.go",
			},
		},
		{
			path: "tests/test_cost_control.py",
			replacements: []string{
				"bigclaw-go/internal/costcontrol/controller_test.go",
			},
		},
		{
			path: "tests/test_github_sync.py",
			replacements: []string{
				"bigclaw-go/internal/githubsync/sync_test.go",
			},
		},
		{
			path: "tests/test_orchestration.py",
			replacements: []string{
				"bigclaw-go/internal/workflow/orchestration_test.go",
			},
		},
		{
			path: "tests/test_repo_links.py",
			replacements: []string{
				"bigclaw-go/internal/repo/links.go",
				"bigclaw-go/internal/repo/repo_surfaces_test.go",
			},
		},
		{
			path: "tests/test_scheduler.py",
			replacements: []string{
				"bigclaw-go/internal/scheduler/scheduler.go",
				"bigclaw-go/internal/scheduler/scheduler_test.go",
				"docs/go-mainline-cutover-issue-pack.md",
			},
		},
		{
			path: "scripts/ops/bigclaw_github_sync.py",
			replacements: []string{
				"bigclaw-go/cmd/bigclawctl/main.go",
				"docs/go-cli-script-migration-plan.md",
			},
		},
		{
			path: "bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py",
			replacements: []string{
				"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
			},
		},
		{
			path: "bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
			replacements: []string{
				"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
				"bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
			},
		},
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(candidate.path))); !os.IsNotExist(err) {
			t.Fatalf("expected BIG-GO-1574 candidate Python path to stay absent: %s", candidate.path)
		}
		for _, replacement := range candidate.replacements {
			if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(replacement))); err != nil {
				t.Fatalf("expected BIG-GO-1574 replacement evidence to exist: %s (%v)", replacement, err)
			}
		}
	}

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1574-residual-python-sweep-04.md")
	for _, needle := range []string{
		"BIG-GO-1574",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Compatibility shims left behind: `[]`",
		"Shim deletion conditions: `not applicable; all targeted Python paths are already physically absent`",
		"`src/bigclaw/collaboration.py` -> `bigclaw-go/internal/collaboration/thread.go`, `bigclaw-go/internal/collaboration/thread_test.go`",
		"`src/bigclaw/github_sync.py` -> `bigclaw-go/internal/githubsync/sync.go`, `bigclaw-go/internal/githubsync/sync_test.go`",
		"`src/bigclaw/pilot.py` -> `bigclaw-go/internal/pilot/report.go`, `bigclaw-go/internal/pilot/report_test.go`",
		"`src/bigclaw/repo_triage.py` -> `bigclaw-go/internal/repo/triage.go`, `bigclaw-go/docs/reports/big-go-1362-repo-module-removal-sweep.md`",
		"`src/bigclaw/validation_policy.py` -> `bigclaw-go/internal/policy/validation.go`, `bigclaw-go/internal/policy/validation_test.go`",
		"`tests/test_cost_control.py` -> `bigclaw-go/internal/costcontrol/controller_test.go`",
		"`tests/test_github_sync.py` -> `bigclaw-go/internal/githubsync/sync_test.go`",
		"`tests/test_orchestration.py` -> `bigclaw-go/internal/workflow/orchestration_test.go`",
		"`tests/test_repo_links.py` -> `bigclaw-go/internal/repo/links.go`, `bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`tests/test_scheduler.py` -> `bigclaw-go/internal/scheduler/scheduler.go`, `bigclaw-go/internal/scheduler/scheduler_test.go`, `docs/go-mainline-cutover-issue-pack.md`",
		"`scripts/ops/bigclaw_github_sync.py` -> `bigclaw-go/cmd/bigclawctl/main.go`, `docs/go-cli-script-migration-plan.md`",
		"`bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go`, `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`, `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`, `bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1574ResidualPythonSweep04'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
