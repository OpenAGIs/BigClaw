package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO217RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO217PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"reports",
		"bigclaw-go/docs",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal",
		"bigclaw-go/internal/regression",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO217ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"reports/BIG-GO-208-validation.md",
		"bigclaw-go/docs/migration.md",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/docs/reports/big-go-208-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md",
		"bigclaw-go/internal/repo/plane.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO217LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-217-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-217",
		"Repository-wide Python file count: `0`.",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`reports/BIG-GO-208-validation.md`",
		"`bigclaw-go/docs/migration.md`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`bigclaw-go/docs/reports/big-go-208-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`",
		"`bigclaw-go/internal/repo/plane.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find reports bigclaw-go/docs bigclaw-go/docs/reports bigclaw-go/internal bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO217(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
