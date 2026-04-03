package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResidualPythonSweepACandidateFilesStayRemoved(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"tests/conftest.py",
		"tests/test_audit_events.py",
		"tests/test_connectors.py",
		"tests/test_console_ia.py",
		"tests/test_control_center.py",
		"tests/test_cost_control.py",
		"tests/test_cross_process_coordination_surface.py",
		"tests/test_dashboard_run_contract.py",
		"tests/test_design_system.py",
		"tests/test_dsl.py",
		"tests/test_evaluation.py",
		"tests/test_event_bus.py",
		"tests/test_execution_contract.py",
		"tests/test_execution_flow.py",
		"tests/test_followup_digests.py",
		"tests/test_github_sync.py",
		"tests/test_governance.py",
		"tests/test_issue_archive.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_live_shadow_scorecard.py",
		"tests/test_mapping.py",
		"tests/test_memory.py",
		"tests/test_models.py",
		"tests/test_observability.py",
		"tests/test_operations.py",
		"tests/test_orchestration.py",
		"tests/test_parallel_refill.py",
		"tests/test_parallel_validation_bundle.py",
	}

	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected residual Python sweep-A file to stay removed: %s", relativePath)
		}
	}
}

func TestResidualPythonSweepAReplacementSurfacesExist(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	goAndReportReplacements := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/observability/task_run_test.go",
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/tracing-backend-follow-up-digest.md",
		"bigclaw-go/docs/reports/event-bus-reliability-report.md",
		"bigclaw-go/docs/reports/rollback-safeguard-follow-up-digest.md",
	}

	for _, relativePath := range goAndReportReplacements {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}
