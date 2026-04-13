package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO248RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO248PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"docs",
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/migration",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO248ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-230-validation.md",
		"reports/BIG-GO-237-validation.md",
		"docs/go-cli-script-migration-plan.md",
		"docs/local-tracker-automation.md",
		"scripts/ops/bigclawctl",
		"bigclaw-go/internal/regression/big_go_1606_runtime_workflow_mainline_test.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/regression/big_go_230_zero_python_guard_test.go",
		"bigclaw-go/internal/regression/big_go_237_zero_python_guard_test.go",
		"bigclaw-go/docs/reports/big-go-230-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-237-python-asset-sweep.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO248LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-248-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-248",
		"Broad repo Python reduction sweep AN",
		"Repository-wide Python file count: `0`.",
		"`docs`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"`reports/BIG-GO-230-validation.md`",
		"`reports/BIG-GO-237-validation.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`docs/local-tracker-automation.md`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/regression/big_go_1606_runtime_workflow_mainline_test.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/workflow/engine.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/regression/big_go_230_zero_python_guard_test.go`",
		"`bigclaw-go/internal/regression/big_go_237_zero_python_guard_test.go`",
		"`bigclaw-go/docs/reports/big-go-230-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-237-python-asset-sweep.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find docs reports bigclaw-go/docs/reports bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO248(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"BIG-GO-248 documents and hardens a branch that was already physically Python-free, so it cannot lower the repository `.py` count any further in this checkout.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
