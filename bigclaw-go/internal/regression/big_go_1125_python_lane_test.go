package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBigGo1125CandidatePythonFilesStayDeleted(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py",
		"bigclaw-go/scripts/e2e/run_all_test.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}
}

func TestBigGo1125ReplacementSurfacesExist(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	replacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"scripts/ops/bigclawctl",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBigGo1125MigrationDocsAdvertiseGoReplacements(t *testing.T) {
	goRepoRoot := repoRoot(t)
	workspaceRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goCLIDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	for _, needle := range []string{
		"`go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`",
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`./scripts/e2e/run_all.sh`",
	} {
		if !strings.Contains(goCLIDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing Go replacement guidance %q", needle)
		}
	}

	rootPlanDoc := readRepoFile(t, workspaceRoot, "docs/go-cli-script-migration-plan.md")
	for _, needle := range []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`",
	} {
		if !strings.Contains(rootPlanDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing replacement guidance %q", needle)
		}
	}
}
