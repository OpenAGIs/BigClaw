package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomationPythonSweepLane9Removed(t *testing.T) {
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
			t.Fatalf("expected deleted Python automation file to be absent: %s", relativePath)
		}
	}

	for _, relativeDir := range []string{
		"bigclaw-go/scripts/benchmark",
		"bigclaw-go/scripts/e2e",
	} {
		entries, err := os.ReadDir(filepath.Join(repoRoot, relativeDir))
		if err != nil {
			t.Fatalf("read script directory %s: %v", relativeDir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".py") {
				t.Fatalf("expected Go-only automation directory %s to stay Python-free, found %s", relativeDir, entry.Name())
			}
		}
	}

	for _, relativeDir := range []string{
		"bigclaw-go/scripts/migration",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, relativeDir)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python-only automation directory to stay absent: %s", relativeDir)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
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
		"bigclaw-go/internal/api/coordination_surface.go",
		"bigclaw-go/internal/api/live_shadow_surface.go",
		"bigclaw-go/internal/legacyshim/compilecheck.go",
		"bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/external_store_validation_report_test.go",
		"bigclaw-go/internal/regression/shared_queue_report_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/docs/migration-shadow.md",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
