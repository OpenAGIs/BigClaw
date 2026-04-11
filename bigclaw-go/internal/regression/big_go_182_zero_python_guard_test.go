package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO182RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO182ResidualTestDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"tests",
		"bigclaw-go/internal/api",
		"bigclaw-go/internal/contract",
		"bigclaw-go/internal/planning",
		"bigclaw-go/internal/queue",
		"bigclaw-go/internal/repo",
		"bigclaw-go/internal/collaboration",
		"bigclaw-go/internal/product",
		"bigclaw-go/internal/triage",
		"bigclaw-go/internal/workflow",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual test directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO182RetiredPythonTestTreeRemainsAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	retiredPaths := []string{
		"tests/conftest.py",
		"tests/test_cross_process_coordination_surface.py",
		"tests/test_execution_contract.py",
		"tests/test_orchestration.py",
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

func TestBIGGO182ReplacementPathsRemainAvailable(t *testing.T) {
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
		"bigclaw-go/internal/product/clawhost_rollout_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO182LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-182-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-182",
		"Repository-wide Python file count: `0`.",
		"`tests`: `0` Python files because the root test tree is absent",
		"`bigclaw-go/internal/api`: `0` Python files",
		"`bigclaw-go/internal/contract`: `0` Python files",
		"`bigclaw-go/internal/planning`: `0` Python files",
		"`bigclaw-go/internal/queue`: `0` Python files",
		"`bigclaw-go/internal/repo`: `0` Python files",
		"`bigclaw-go/internal/collaboration`: `0` Python files",
		"`bigclaw-go/internal/product`: `0` Python files",
		"`bigclaw-go/internal/triage`: `0` Python files",
		"`bigclaw-go/internal/workflow`: `0` Python files",
		"`tests/conftest.py`",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_execution_contract.py`",
		"`tests/test_orchestration.py`",
		"`tests/test_planning.py`",
		"`tests/test_queue.py`",
		"`tests/test_repo_board.py`",
		"`tests/test_repo_collaboration.py`",
		"`tests/test_repo_gateway.py`",
		"`tests/test_repo_governance.py`",
		"`tests/test_repo_links.py`",
		"`tests/test_repo_registry.py`",
		"`tests/test_repo_rollout.py`",
		"`tests/test_repo_triage.py`",
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
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal/api bigclaw-go/internal/contract bigclaw-go/internal/planning bigclaw-go/internal/queue bigclaw-go/internal/repo bigclaw-go/internal/collaboration bigclaw-go/internal/product bigclaw-go/internal/triage bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO182(RepositoryHasNoPythonFiles|ResidualTestDirectoriesStayPythonFree|RetiredPythonTestTreeRemainsAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
