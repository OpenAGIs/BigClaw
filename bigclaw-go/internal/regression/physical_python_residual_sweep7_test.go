package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhysicalPythonResidualSweep7(t *testing.T) {
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
			t.Fatalf("expected deleted Python path to be absent: %s", relativePath)
		}
	}

	goOrShellReplacements := []string{
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
	}
	for _, relativePath := range goOrShellReplacements {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go or shell replacement to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestPhysicalPythonResidualSweep7Docs(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	migrationDoc := readRepoFile(t, repoRoot, "bigclaw-go/docs/go-cli-script-migration.md")
	requiredDocEntries := []string{
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`go run ./cmd/bigclawctl automation e2e external-store-validation ...`",
		"`go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`",
	}
	for _, needle := range requiredDocEntries {
		if !strings.Contains(migrationDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing replacement entry %q", needle)
		}
	}

	disallowedDocEntries := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
	}
	for _, needle := range disallowedDocEntries {
		if strings.Contains(migrationDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md should not reference retired Python helper %q", needle)
		}
	}

	planDoc := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclaw-go/scripts/e2e/` operator entrypoints now dispatch through `bigclawctl automation e2e ...`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(planDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing migration-plan entry %q", needle)
		}
	}
}
