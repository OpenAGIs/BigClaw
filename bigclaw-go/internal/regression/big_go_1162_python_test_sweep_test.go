package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1162CandidatePythonTestsRemainDeleted(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	candidates := []string{
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

	for _, relativePath := range candidates {
		_, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected BIG-GO-1162 candidate path to stay deleted: %s (err=%v)", relativePath, err)
		}
	}

	_, err := os.Stat(filepath.Join(rootRepo, "tests"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected root tests directory to stay deleted for BIG-GO-1162: %v", err)
	}
}

func TestBIGGO1162MigrationDocsListGoReplacements(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	rootDoc := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	requiredGoDoc := []string{
		"Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`, `BIG-GO-1162`",
		"## BIG-GO-1162 Residual Test Sweep Coverage",
		"`tests/test_audit_events.py`, `tests/test_observability.py`, `tests/test_reports.py`",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/reporting/reporting_test.go`",
		"`tests/test_cross_process_coordination_surface.py`, `tests/test_parallel_refill.py`, `tests/test_parallel_validation_bundle.py`, `tests/test_followup_digests.py`, `tests/test_live_shadow_bundle.py`, `tests/test_live_shadow_scorecard.py`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`tests/test_repo_board.py`, `tests/test_repo_collaboration.py`, `tests/test_repo_gateway.py`, `tests/test_repo_governance.py`, `tests/test_repo_links.py`, `tests/test_repo_registry.py`, `tests/test_repo_rollout.py`, `tests/test_repo_triage.py`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
	}
	for _, needle := range requiredGoDoc {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing BIG-GO-1162 replacement %q", needle)
		}
	}

	requiredRootDoc := []string{
		"`BIG-GO-1162` extends the same migration evidence",
		"`tests/conftest.py`",
		"`tests/test_parallel_validation_bundle.py`",
		"`tests/test_repo_gateway.py`",
		"`tests/test_reports.py`",
		"`bigclaw-go/internal/...`, `bigclaw-go/cmd/bigclawctl/...`,",
	}
	for _, needle := range requiredRootDoc {
		if !strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1162 coverage %q", needle)
		}
	}
}
