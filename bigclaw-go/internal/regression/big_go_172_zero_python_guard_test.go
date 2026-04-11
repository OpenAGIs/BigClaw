package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO172RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO172RemainingTestHeavyReplacementDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		"bigclaw-go/internal/api",
		"bigclaw-go/internal/contract",
		"bigclaw-go/internal/events",
		"bigclaw-go/internal/githubsync",
		"bigclaw-go/internal/governance",
		"bigclaw-go/internal/observability",
		"bigclaw-go/internal/orchestrator",
		"bigclaw-go/internal/planning",
		"bigclaw-go/internal/policy",
		"bigclaw-go/internal/product",
		"bigclaw-go/internal/queue",
		"bigclaw-go/internal/repo",
		"bigclaw-go/internal/workflow",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected remaining test-heavy replacement directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO172RepresentativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/api/coordination_surface.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/githubsync/sync_test.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/orchestrator/loop_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/product/clawhost_rollout_test.go",
		"bigclaw-go/internal/queue/sqlite_queue_test.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"reports/BIG-GO-948-validation.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected representative Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO172LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-172-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-172",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/internal/api`: `0` Python files",
		"`bigclaw-go/internal/contract`: `0` Python files",
		"`bigclaw-go/internal/events`: `0` Python files",
		"`bigclaw-go/internal/githubsync`: `0` Python files",
		"`bigclaw-go/internal/governance`: `0` Python files",
		"`bigclaw-go/internal/observability`: `0` Python files",
		"`bigclaw-go/internal/orchestrator`: `0` Python files",
		"`bigclaw-go/internal/planning`: `0` Python files",
		"`bigclaw-go/internal/policy`: `0` Python files",
		"`bigclaw-go/internal/product`: `0` Python files",
		"`bigclaw-go/internal/queue`: `0` Python files",
		"`bigclaw-go/internal/repo`: `0` Python files",
		"`bigclaw-go/internal/workflow`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_event_bus.py`",
		"`tests/test_execution_contract.py`",
		"`tests/test_execution_flow.py`",
		"`tests/test_github_sync.py`",
		"`tests/test_governance.py`",
		"`tests/test_memory.py`",
		"`tests/test_models.py`",
		"`tests/test_observability.py`",
		"`tests/test_orchestration.py`",
		"`tests/test_planning.py`",
		"`tests/test_queue.py`",
		"`tests/test_repo_gateway.py`",
		"`tests/test_repo_governance.py`",
		"`tests/test_repo_links.py`",
		"`tests/test_repo_registry.py`",
		"`tests/test_repo_rollout.py`",
		"`bigclaw-go/internal/api/coordination_surface.go`",
		"`bigclaw-go/internal/events/bus_test.go`",
		"`bigclaw-go/internal/contract/execution_test.go`",
		"`bigclaw-go/internal/workflow/orchestration_test.go`",
		"`bigclaw-go/internal/githubsync/sync_test.go`",
		"`bigclaw-go/internal/governance/freeze_test.go`",
		"`bigclaw-go/internal/policy/memory_test.go`",
		"`bigclaw-go/internal/observability/recorder_test.go`",
		"`bigclaw-go/internal/planning/planning_test.go`",
		"`bigclaw-go/internal/queue/sqlite_queue_test.go`",
		"`bigclaw-go/internal/repo/gateway.go`",
		"`bigclaw-go/internal/repo/governance.go`",
		"`bigclaw-go/internal/repo/links.go`",
		"`bigclaw-go/internal/repo/registry.go`",
		"`bigclaw-go/internal/product/clawhost_rollout_test.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/internal/api bigclaw-go/internal/contract bigclaw-go/internal/events bigclaw-go/internal/githubsync bigclaw-go/internal/governance bigclaw-go/internal/observability bigclaw-go/internal/orchestrator bigclaw-go/internal/planning bigclaw-go/internal/policy bigclaw-go/internal/product bigclaw-go/internal/queue bigclaw-go/internal/repo bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO172(RepositoryHasNoPythonFiles|RemainingTestHeavyReplacementDirectoriesStayPythonFree|RepresentativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
