package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonTestTranche15Removed(t *testing.T) {
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
		"tests/test_pilot.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_gateway.py",
		"tests/test_repo_governance.py",
		"tests/test_repo_links.py",
		"tests/test_repo_registry.py",
		"tests/test_repo_rollout.py",
		"tests/test_repo_triage.py",
		"tests/test_reports.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python test to be absent: %s", relativePath)
		}
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted Python tests directory to stay absent: %v", err)
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/control/controller_test.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/regression/cross_process_coordination_docs_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/regression/followup_index_docs_test.go",
		"bigclaw-go/internal/githubsync/sync_test.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/live_shadow_docs_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/domain/task_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
