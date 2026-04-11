package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO24RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO24BatchDResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"tests",
		"bigclaw-go/internal/migration",
		"bigclaw-go/internal/regression",
		"bigclaw-go/docs/reports",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected batch-D residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO24ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-948-validation.md",
		"reports/BIG-GO-13-validation.md",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
		"bigclaw-go/docs/reports/shared-queue-companion-summary.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO24LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-24-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-24",
		"Repository-wide Python file count: `0`.",
		"`tests`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`reports/BIG-GO-948-validation.md`",
		"`reports/BIG-GO-13-validation.md`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"`bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`",
		"`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`",
		"`bigclaw-go/docs/reports/shared-queue-companion-summary.json`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal/migration bigclaw-go/internal/regression bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO24(RepositoryHasNoPythonFiles|BatchDResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
