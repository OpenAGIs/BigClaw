package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO222RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO222PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"tests",
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/migration",
		"bigclaw-go/internal/regression",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO222ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-948-validation.md",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go",
		"bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go",
		"bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md",
		"bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md",
		"bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-202-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-208-python-asset-sweep.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO222LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-222-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-222",
		"Repository-wide Python file count: `0`.",
		"`tests`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`reports/BIG-GO-948-validation.md`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`",
		"`bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`",
		"`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`",
		"`bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md`",
		"`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-202-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-208-python-asset-sweep.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests reports bigclaw-go/docs/reports bigclaw-go/internal/regression bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO222(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
