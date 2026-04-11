package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO203RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO203ResidualPythonTestGapSliceStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	retiredPaths := []string{
		"tests/test_cost_control.py",
		"tests/test_event_bus.py",
		"tests/test_execution_flow.py",
		"tests/test_github_sync.py",
		"tests/test_governance.py",
		"tests/test_issue_archive.py",
		"tests/test_mapping.py",
		"tests/test_memory.py",
		"tests/test_models.py",
		"tests/test_observability.py",
		"tests/test_pilot.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO203GapSliceReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/executor/kubernetes_test.go",
		"bigclaw-go/internal/executor/ray_test.go",
		"bigclaw-go/internal/githubsync/sync_test.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/pilot/rollout_test.go",
		"reports/BIG-GO-948-validation.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO203LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-203-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-203",
		"Repository-wide Python file count: `0`.",
		"`tests`: absent",
		"`bigclaw-go/internal/costcontrol`: `0` Python files",
		"`bigclaw-go/internal/events`: `0` Python files",
		"`bigclaw-go/internal/executor`: `0` Python files",
		"`bigclaw-go/internal/githubsync`: `0` Python files",
		"`bigclaw-go/internal/governance`: `0` Python files",
		"`bigclaw-go/internal/intake`: `0` Python files",
		"`bigclaw-go/internal/issuearchive`: `0` Python files",
		"`bigclaw-go/internal/observability`: `0` Python files",
		"`bigclaw-go/internal/pilot`: `0` Python files",
		"`bigclaw-go/internal/policy`: `0` Python files",
		"`bigclaw-go/internal/workflow`: `0` Python files",
		"`tests/test_cost_control.py`",
		"`tests/test_event_bus.py`",
		"`tests/test_execution_flow.py`",
		"`tests/test_github_sync.py`",
		"`tests/test_governance.py`",
		"`tests/test_issue_archive.py`",
		"`tests/test_mapping.py`",
		"`tests/test_memory.py`",
		"`tests/test_models.py`",
		"`tests/test_observability.py`",
		"`tests/test_pilot.py`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/costcontrol/controller_test.go`",
		"`bigclaw-go/internal/events/bus_test.go`",
		"`bigclaw-go/internal/executor/kubernetes_test.go`",
		"`bigclaw-go/internal/executor/ray_test.go`",
		"`bigclaw-go/internal/githubsync/sync_test.go`",
		"`bigclaw-go/internal/governance/freeze_test.go`",
		"`bigclaw-go/internal/issuearchive/archive_test.go`",
		"`bigclaw-go/internal/intake/mapping_test.go`",
		"`bigclaw-go/internal/policy/memory_test.go`",
		"`bigclaw-go/internal/workflow/model_test.go`",
		"`bigclaw-go/internal/observability/recorder_test.go`",
		"`bigclaw-go/internal/pilot/report_test.go`",
		"`bigclaw-go/internal/pilot/rollout_test.go`",
		"`reports/BIG-GO-948-validation.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find tests bigclaw-go/internal/costcontrol bigclaw-go/internal/events bigclaw-go/internal/executor bigclaw-go/internal/githubsync bigclaw-go/internal/governance bigclaw-go/internal/intake bigclaw-go/internal/issuearchive bigclaw-go/internal/observability bigclaw-go/internal/pilot bigclaw-go/internal/policy bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO203(RepositoryHasNoPythonFiles|ResidualPythonTestGapSliceStaysAbsent|GapSliceReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
