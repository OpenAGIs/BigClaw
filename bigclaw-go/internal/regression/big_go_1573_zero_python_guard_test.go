package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1573RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1573CoveredResidualPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	coveredPaths := []string{
		"src/bigclaw/audit_events.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/parallel_refill.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/ui_review.py",
		"tests/test_control_center.py",
		"tests/test_followup_digests.py",
		"tests/test_operations.py",
		"tests/test_repo_governance.py",
		"tests/test_saved_views.py",
		"scripts/dev_smoke.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
	}

	for _, relativePath := range coveredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected BIG-GO-1573 covered Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1573GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/contract/execution.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/uireview/builder.go",
		"bigclaw-go/internal/uireview/render.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/api/expansion.go",
		"bigclaw-go/docs/reports/parallel-follow-up-index.md",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"docs/go-cli-script-migration-plan.md",
		"docs/issue-plan.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1573LaneReportCapturesCoveredFileLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1573-python-asset-sweep-03.md")

	for _, needle := range []string{
		"BIG-GO-1573",
		"Repository-wide Python file count before lane changes: `0`",
		"Repository-wide Python file count after lane changes: `0`",
		"Covered Python file count before lane changes: `0`",
		"Covered Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"No Python compatibility shims remain for this sweep set.",
		"`src/bigclaw/audit_events.py` -> `bigclaw-go/internal/observability/audit_spec.go`",
		"`src/bigclaw/execution_contract.py` -> `bigclaw-go/internal/contract/execution.go`",
		"`src/bigclaw/parallel_refill.py` -> `bigclaw-go/internal/refill/queue.go`",
		"`src/bigclaw/repo_registry.py` -> `bigclaw-go/internal/repo/registry.go`",
		"`src/bigclaw/ui_review.py` -> `bigclaw-go/internal/uireview/uireview.go`, `bigclaw-go/internal/uireview/builder.go`, `bigclaw-go/internal/uireview/render.go`",
		"`tests/test_control_center.py` -> `bigclaw-go/internal/api/server.go`, `bigclaw-go/internal/control/controller.go`, `bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`",
		"`tests/test_followup_digests.py` -> `bigclaw-go/docs/reports/parallel-follow-up-index.md`, `bigclaw-go/internal/regression/followup_index_docs_test.go`",
		"`tests/test_operations.py` -> `bigclaw-go/internal/product/dashboard_run_contract.go`, `bigclaw-go/internal/contract/execution.go`, `bigclaw-go/internal/control/controller.go`",
		"`tests/test_repo_governance.py` -> `bigclaw-go/internal/repo/governance.go`, `bigclaw-go/internal/repo/governance_test.go`",
		"`tests/test_saved_views.py` -> `bigclaw-go/internal/product/saved_views.go`, `bigclaw-go/internal/api/expansion.go`, `bigclaw-go/internal/product/saved_views_test.go`",
		"`scripts/dev_smoke.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`, `scripts/ops/bigclawctl`",
		"`bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go`",
		"`bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find . -type f \\( -path './src/bigclaw/audit_events.py' -o -path './src/bigclaw/execution_contract.py' -o -path './src/bigclaw/parallel_refill.py' -o -path './src/bigclaw/repo_registry.py' -o -path './src/bigclaw/ui_review.py' -o -path './tests/test_control_center.py' -o -path './tests/test_followup_digests.py' -o -path './tests/test_operations.py' -o -path './tests/test_repo_governance.py' -o -path './tests/test_saved_views.py' -o -path './scripts/dev_smoke.py' -o -path './bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py' -o -path './bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py' \\) | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1573",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
