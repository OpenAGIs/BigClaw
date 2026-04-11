package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO2RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO2PriorityResidualTestDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	priorityDirs := []string{
		"bigclaw-go/cmd/bigclawctl",
		"bigclaw-go/internal/evaluation",
		"bigclaw-go/internal/workflow",
		"bigclaw-go/internal/regression",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual test directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO2GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-948-validation.md",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/queue/sqlite_queue_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
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

func TestBIGGO2LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-2-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-2",
		"Repository-wide Python file count: `0`.",
		"`tests`: absent",
		"`bigclaw-go/cmd/bigclawctl`: `0` Python files",
		"`bigclaw-go/internal/evaluation`: `0` Python files",
		"`bigclaw-go/internal/workflow`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`reports/BIG-GO-948-validation.md`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`bigclaw-go/internal/evaluation/evaluation_test.go`",
		"`bigclaw-go/internal/workflow/orchestration_test.go`",
		"`bigclaw-go/internal/planning/planning_test.go`",
		"`bigclaw-go/internal/queue/sqlite_queue_test.go`",
		"`bigclaw-go/internal/collaboration/thread_test.go`",
		"`bigclaw-go/internal/product/clawhost_rollout_test.go`",
		"`bigclaw-go/internal/triage/repo_test.go`",
		"`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`bigclaw-go/docs/reports/shared-queue-companion-summary.json`",
		"`bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`",
		"`bigclaw-go/docs/reports/shadow-matrix-report.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal/regression bigclaw-go/cmd/bigclawctl bigclaw-go/internal/evaluation bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO2(RepositoryHasNoPythonFiles|PriorityResidualTestDirectoriesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
