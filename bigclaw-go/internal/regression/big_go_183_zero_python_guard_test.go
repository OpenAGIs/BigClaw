package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO183ResidualPythonTestTreeStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	retiredPaths := []string{
		"tests/conftest.py",
		"tests/test_cross_process_coordination_surface.py",
		"tests/test_execution_contract.py",
		"tests/test_followup_digests.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_live_shadow_scorecard.py",
		"tests/test_orchestration.py",
		"tests/test_parallel_refill.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_gateway.py",
		"tests/test_repo_governance.py",
		"tests/test_repo_links.py",
		"tests/test_repo_registry.py",
		"tests/test_repo_rollout.py",
		"tests/test_repo_triage.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO183ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/api/coordination_surface.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/queue/sqlite_queue_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/product/clawhost_rollout_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
		"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/shared-queue-companion-summary.json",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/shadow-matrix-report.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO183LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-183-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-183",
		"Repository-wide Python file count: `0`.",
		"`tests`: absent",
		"`bigclaw-go/internal`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files and retained Go-owned report fixtures",
		"`tests/conftest.py`",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`tests/test_repo_registry.py`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/api/coordination_surface.go`",
		"`bigclaw-go/internal/contract/execution_test.go`",
		"`bigclaw-go/internal/workflow/orchestration_test.go`",
		"`bigclaw-go/internal/planning/planning_test.go`",
		"`bigclaw-go/internal/queue/sqlite_queue_test.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`bigclaw-go/internal/collaboration/thread_test.go`",
		"`bigclaw-go/internal/product/clawhost_rollout_test.go`",
		"`bigclaw-go/internal/triage/repo_test.go`",
		"`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`bigclaw-go/docs/reports/shared-queue-companion-summary.json`",
		"`bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`",
		"`bigclaw-go/docs/reports/shadow-matrix-report.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal bigclaw-go/docs/reports -type f \\( -name '*.py' -o -name 'validation-bundle-continuation-scorecard.json' -o -name 'shared-queue-companion-summary.json' -o -name 'cross-process-coordination-capability-surface.json' -o -name 'live-shadow-mirror-scorecard.json' -o -name 'shadow-matrix-report.json' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO183(ResidualPythonTestTreeStaysAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
