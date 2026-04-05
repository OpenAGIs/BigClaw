package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1462RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1462PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"tests",
		"src/bigclaw",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected root tests directory to stay absent, stat err=%v", err)
	}

	deletedPythonTests := []string{
		"tests/conftest.py",
		"tests/test_audit_events.py",
		"tests/test_connectors.py",
		"tests/test_console_ia.py",
		"tests/test_control_center.py",
		"tests/test_cost_control.py",
		"tests/test_dashboard_run_contract.py",
		"tests/test_design_system.py",
		"tests/test_execution_contract.py",
		"tests/test_execution_flow.py",
		"tests/test_followup_digests.py",
		"tests/test_github_sync.py",
		"tests/test_governance.py",
		"tests/test_observability.py",
		"tests/test_operations.py",
		"tests/test_orchestration.py",
		"tests/test_parallel_refill.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_reports.py",
	}

	for _, relativePath := range deletedPythonTests {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected retired Python test to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1462GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/control/controller_test.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1462LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1462-python-test-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1462",
		"Repository-wide Python file count: `0`.",
		"`tests/*.py`: `none`",
		"`tests` directory: absent",
		"`tests/test_audit_events.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/contract/execution_test.go`",
		"`bigclaw-go/internal/refill/queue_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests src/bigclaw scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1462",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
