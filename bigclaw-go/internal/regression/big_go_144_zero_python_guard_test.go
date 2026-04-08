package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO144ResidualPythonWrappersAndHelpersStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	targets := []string{
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
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}

	for _, relativePath := range targets {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python wrapper/helper to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO144GoWrapperAndCLIReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-symphony",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO144LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-144-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-144",
		"`scripts/create_issues.py`",
		"`scripts/dev_smoke.py`",
		"`scripts/ops/bigclaw_github_sync.py`",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`scripts/ops/symphony_workspace_validate.py`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py`",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`find /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO144",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
