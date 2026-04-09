package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO197RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO197HighImpactResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		"docs",
		"docs/reports",
		"reports",
		"scripts",
		"bigclaw-go/scripts",
		"bigclaw-go/docs/reports",
		"bigclaw-go/examples",
		"bigclaw-go/internal/regression",
		"bigclaw-go/internal/migration",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected high-impact residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO197NativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/root_script_residual_sweep_test.go",
		"bigclaw-go/internal/migration/legacy_model_runtime_modules.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/docs/reports/review-readiness.md",
		"bigclaw-go/docs/reports/big-go-167-python-asset-sweep.md",
		"bigclaw-go/docs/reports/big-go-168-python-asset-sweep.md",
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

func TestBIGGO197LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-197-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-197",
		"Broad repo Python reduction sweep AC",
		"Repository-wide Python file count: `0`.",
		"`docs`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`reports`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/root_script_residual_sweep_test.go`",
		"`bigclaw-go/internal/migration/legacy_model_runtime_modules.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/docs/reports/review-readiness.md`",
		"`bigclaw-go/docs/reports/big-go-167-python-asset-sweep.md`",
		"`bigclaw-go/docs/reports/big-go-168-python-asset-sweep.md`",
		"`reports/BIG-GO-152-validation.md`",
		"`reports/BIG-GO-157-validation.md`",
		"`reports/BIG-GO-162-validation.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples bigclaw-go/internal/regression bigclaw-go/internal/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO197(RepositoryHasNoPythonFiles|HighImpactResidualDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
