package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO247RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO247PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/regression",
		"scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO247GoNativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-228-validation.md",
		"reports/BIG-GO-237-validation.md",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/big_go_228_zero_python_guard_test.go",
		"bigclaw-go/internal/regression/big_go_237_zero_python_guard_test.go",
		"bigclaw-go/docs/reports/big-go-228-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-237-python-asset-sweep.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO247LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-247-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-247",
		"Broad repo Python reduction sweep AM",
		"Repository-wide Python file count: `0`.",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`scripts`: `0` Python files",
		"`reports/BIG-GO-228-validation.md`",
		"`reports/BIG-GO-237-validation.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/big_go_228_zero_python_guard_test.go`",
		"`bigclaw-go/internal/regression/big_go_237_zero_python_guard_test.go`",
		"`bigclaw-go/docs/reports/big-go-228-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-237-python-asset-sweep.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find reports bigclaw-go/docs/reports bigclaw-go/internal/regression scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO247(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoNativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"BIG-GO-247 documents and hardens a branch that was already physically Python-free, so it cannot lower the repository `.py` count any further in this checkout.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
