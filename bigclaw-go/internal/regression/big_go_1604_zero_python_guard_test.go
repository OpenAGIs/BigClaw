package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1604RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1604AssignedPythonTestAndHarnessResidueRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"tests",
		"tests/conftest.py",
		"tests/test_connectors.py",
		"tests/test_console_ia.py",
		"tests/test_execution_contract.py",
		"tests/test_execution_flow.py",
		"tests/test_followup_digests.py",
		"tests/test_governance.py",
		"tests/test_models.py",
		"tests/test_observability.py",
		"tests/test_reports.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test or harness path to remain absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1604GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/orchestrator/loop_test.go",
		"bigclaw-go/docs/reports/parallel-follow-up-index.md",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"scripts/ops/bigclawctl",
		"docs/go-cli-script-migration-plan.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1604LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1604-python-test-harness-refill.md")

	for _, needle := range []string{
		"BIG-GO-1604",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"Explicit remaining Python asset list: none.",
		"`tests` root: absent",
		"`tests/conftest.py`",
		"`tests/test_connectors.py`",
		"`tests/test_console_ia.py`",
		"`tests/test_execution_contract.py`",
		"`tests/test_execution_flow.py`",
		"`tests/test_followup_digests.py`",
		"`tests/test_governance.py`",
		"`tests/test_models.py`",
		"`tests/test_observability.py`",
		"`tests/test_reports.py`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`bigclaw-go/internal/intake/connector_test.go`",
		"`bigclaw-go/internal/consoleia/consoleia_test.go`",
		"`bigclaw-go/internal/contract/execution_test.go`",
		"`bigclaw-go/internal/orchestrator/loop_test.go`",
		"`bigclaw-go/docs/reports/parallel-follow-up-index.md`",
		"`bigclaw-go/internal/governance/freeze_test.go`",
		"`bigclaw-go/internal/workflow/model_test.go`",
		"`bigclaw-go/internal/observability/recorder_test.go`",
		"`bigclaw-go/internal/reporting/reporting_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`scripts/ops/bigclawctl`",
		"`docs/go-cli-script-migration-plan.md`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`for path in tests tests/conftest.py tests/test_connectors.py tests/test_console_ia.py tests/test_execution_contract.py tests/test_execution_flow.py tests/test_followup_digests.py tests/test_governance.py tests/test_models.py tests/test_observability.py tests/test_reports.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1604(RepositoryHasNoPythonFiles|AssignedPythonTestAndHarnessResidueRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
