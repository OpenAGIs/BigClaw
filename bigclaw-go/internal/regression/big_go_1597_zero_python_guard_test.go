package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1597RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1597AssignedFocusPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/cost_control.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/repo_board.py",
		"src/bigclaw/roadmap.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
		"tests/test_design_system.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_pilot.py",
		"tests/test_repo_triage.py",
		"tests/test_subscriber_takeover_harness.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to remain absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1597GoOwnedReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/costcontrol/controller.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/pilot/report.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/repo/triage.go",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go-owned replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1597LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1597",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/cost_control.py`",
		"`src/bigclaw/mapping.py`",
		"`src/bigclaw/repo_board.py`",
		"`src/bigclaw/roadmap.py`",
		"`src/bigclaw/workspace_bootstrap_cli.py`",
		"`tests/test_design_system.py`",
		"`tests/test_live_shadow_bundle.py`",
		"`tests/test_pilot.py`",
		"`tests/test_repo_triage.py`",
		"`tests/test_subscriber_takeover_harness.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`bigclaw-go/scripts/e2e/export_validation_bundle_test.py`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py`",
		"`bigclaw-go/internal/costcontrol/controller.go`",
		"`bigclaw-go/internal/intake/mapping.go`",
		"`bigclaw-go/internal/repo/board.go`",
		"`bigclaw-go/internal/regression/roadmap_contract_test.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`",
		"`bigclaw-go/internal/pilot/report.go`",
		"`bigclaw-go/internal/repo/triage.go`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`for path in src/bigclaw/cost_control.py src/bigclaw/mapping.py src/bigclaw/repo_board.py src/bigclaw/roadmap.py src/bigclaw/workspace_bootstrap_cli.py tests/test_design_system.py tests/test_live_shadow_bundle.py tests/test_pilot.py tests/test_repo_triage.py tests/test_subscriber_takeover_harness.py scripts/ops/symphony_workspace_bootstrap.py bigclaw-go/scripts/e2e/export_validation_bundle_test.py bigclaw-go/scripts/migration/export_live_shadow_bundle.py; do test ! -e \"$path\" || echo \"present: $path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1597(RepositoryHasNoPythonFiles|AssignedFocusPathsRemainAbsent|GoOwnedReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
