package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSrcBigClawGoReplacementInventory(t *testing.T) {
	goRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	deletedPythonFiles := []string{
		"src/bigclaw/console_ia.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/cost_control.py",
		"src/bigclaw/audit_events.py",
		"src/bigclaw/collaboration.py",
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/deprecation.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/dsl.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/event_bus.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/memory.py",
		"src/bigclaw/models.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/pilot.py",
		"src/bigclaw/parallel_refill.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/repo_board.py",
		"src/bigclaw/repo_commits.py",
		"src/bigclaw/repo_gateway.py",
		"src/bigclaw/repo_governance.py",
		"src/bigclaw/github_sync.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/repo_links.py",
		"src/bigclaw/repo_plane.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/repo_triage.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/roadmap.py",
		"src/bigclaw/saved_views.py",
		"src/bigclaw/validation_policy.py",
		"src/bigclaw/workspace_bootstrap.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
		"src/bigclaw/workspace_bootstrap_validation.py",
	}
	for _, relativePath := range deletedPythonFiles {
		path := filepath.Join(repoRoot, relativePath)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to stay absent: %s", relativePath)
		}
	}

	goOwners := []string{
		"bigclaw-go/internal/costcontrol/controller.go",
		"bigclaw-go/internal/contract/execution.go",
		"bigclaw-go/internal/events/bus.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/legacyshim/wrappers.go",
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/billing/billing.go",
		"bigclaw-go/internal/memory/store.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/pilot/report.go",
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/regression/deprecation_contract_test.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/reporting/validation_policy.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/repo/commits.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/repo/plane.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/roadmap/roadmap.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/api/v2.go",
		"bigclaw-go/internal/api/admission_policy_surface.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/workflow/definition.go",
	}
	for _, relativePath := range goOwners {
		path := filepath.Join(repoRoot, relativePath)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected Go owner file to exist: %s: %v", relativePath, err)
		}
		if info.IsDir() {
			t.Fatalf("expected Go owner path to be a file: %s", relativePath)
		}
	}
}
