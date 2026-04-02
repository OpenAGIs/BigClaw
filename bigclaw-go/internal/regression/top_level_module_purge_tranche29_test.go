package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche29(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
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
		"tests/test_risk.py",
		"tests/test_roadmap.py",
		"tests/test_runtime.py",
		"tests/test_runtime_matrix.py",
		"tests/test_saved_views.py",
		"tests/test_scheduler.py",
		"tests/test_service.py",
		"tests/test_shadow_matrix_corpus.py",
		"tests/test_subscriber_takeover_harness.py",
		"tests/test_ui_review.py",
		"tests/test_validation_bundle_continuation_policy_gate.py",
		"tests/test_validation_bundle_continuation_scorecard.py",
		"tests/test_validation_policy.py",
		"tests/test_workflow.py",
		"tests/test_workspace_bootstrap.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python test asset to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/policy/validation_test.go",
		"bigclaw-go/internal/product/saved_views_test.go",
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/observability_runtime_surface_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/internal/regression/production_corpus_surface_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche1_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go",
		"bigclaw-go/internal/repo/governance_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/risk/risk_test.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/service/server_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement coverage file to exist: %s (%v)", relativePath, err)
		}
	}
}
