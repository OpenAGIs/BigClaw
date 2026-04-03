package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhysicalPythonResidualSweep5LaneStaysMigrated(t *testing.T) {
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
			t.Fatalf("expected deleted Python asset to stay absent: %s", relativePath)
		}
	}

	goAndShellReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/docs/benchmark-plan.md",
		"bigclaw-go/docs/e2e-validation.md",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/docs/migration-shadow.md",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"docs/go-cli-script-migration-plan.md",
		"scripts/ops/bigclawctl",
	}
	for _, relativePath := range goAndShellReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected migrated replacement asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestPhysicalPythonResidualSweep5DocsStayGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	migrationDoc := readRepoFile(t, filepath.Join(repoRoot, "bigclaw-go"), "docs/go-cli-script-migration.md")
	requiredMigrationEntries := []string{
		"`go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...`",
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`go run ./cmd/bigclawctl automation e2e external-store-validation ...`",
		"`go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...`",
		"`go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
		"`go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e continuation-scorecard ...`",
		"`go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`",
		"`go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`",
	}
	for _, needle := range requiredMigrationEntries {
		if !strings.Contains(migrationDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing migrated entrypoint %q", needle)
		}
	}

	rootPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredRootEntries := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`",
	}
	for _, needle := range requiredRootEntries {
		if !strings.Contains(rootPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing migrated root guidance %q", needle)
		}
	}

	disallowedLegacyRefs := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}
	for _, needle := range disallowedLegacyRefs {
		if strings.Contains(migrationDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md should not reference deleted Python helper %q", needle)
		}
	}
}
