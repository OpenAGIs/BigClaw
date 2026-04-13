package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO167RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO167ReferenceDenseGoOwnedDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		"bigclaw-go/internal/regression",
		"bigclaw-go/internal/migration",
		"bigclaw-go/docs/reports",
		"reports",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected reference-dense directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO167NativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/root_script_residual_sweep_test.go",
		"bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go",
		"bigclaw-go/internal/regression/big_go_1606_runtime_workflow_mainline_test.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/docs/reports/review-readiness.md",
		"bigclaw-go/docs/reports/big-go-1606-runtime-workflow-mainline-cutover.md",
		"bigclaw-go/docs/reports/big-go-152-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-162-python-asset-sweep.md",
		"reports/BIG-GO-152-validation.md",
		"reports/BIG-GO-157-validation.md",
		"reports/BIG-GO-162-validation.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO167LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-167-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-167",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/root_script_residual_sweep_test.go`",
		"`bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`",
		"`bigclaw-go/internal/regression/big_go_1606_runtime_workflow_mainline_test.go`",
		"`bigclaw-go/internal/service/server.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/docs/reports/review-readiness.md`",
		"`bigclaw-go/docs/reports/big-go-1606-runtime-workflow-mainline-cutover.md`",
		"`bigclaw-go/docs/reports/big-go-152-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-162-python-asset-sweep.md`",
		"`reports/BIG-GO-152-validation.md`",
		"`reports/BIG-GO-157-validation.md`",
		"`reports/BIG-GO-162-validation.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/internal/regression bigclaw-go/internal/migration bigclaw-go/docs/reports reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO167(RepositoryHasNoPythonFiles|ReferenceDenseGoOwnedDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
