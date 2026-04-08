package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO162ResidualPythonTestTreeStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	retiredPaths := []string{
		"tests/test_control_center.py",
		"tests/test_operations.py",
		"tests/test_ui_review.py",
		"tests/test_design_system.py",
		"tests/test_dsl.py",
		"tests/test_evaluation.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_followup_digests.py",
		"tests/test_live_shadow_scorecard.py",
		"tests/test_parallel_refill.py",
		"tests/test_reports.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO162ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/control/controller_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/refill/queue_repo_fixture_test.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/shared-queue-companion-summary.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO162LaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-162-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-162",
		"Repository-wide Python file count: `0`.",
		"`tests`: absent",
		"`tests/test_control_center.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`tests/test_followup_digests.py`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/control/controller_test.go`",
		"`bigclaw-go/internal/designsystem/designsystem_test.go`",
		"`bigclaw-go/internal/refill/queue_repo_fixture_test.go`",
		"`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal bigclaw-go/docs/reports -type f \\( -name '*.py' -o -name 'validation-bundle-continuation-scorecard.json' -o -name 'shared-queue-companion-summary.json' -o -name 'cross-process-coordination-capability-surface.json' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO162(ResidualPythonTestTreeStaysAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
