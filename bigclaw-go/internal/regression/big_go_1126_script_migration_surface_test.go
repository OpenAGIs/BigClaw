package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1126ScriptMigrationSurface(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

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
		if _, err := os.Stat(filepath.Join(repo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python entrypoint to be absent: %s", relativePath)
		}
	}

	goOrCompatibilityOwners := []string{
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
	}
	for _, relativePath := range goOrCompatibilityOwners {
		if _, err := os.Stat(filepath.Join(repo, relativePath)); err != nil {
			t.Fatalf("expected Go or compatibility owner to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1126MigrationDocsKeepGoOnlyPaths(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	scriptMigration := readRepoFile(t, goRoot, "docs/go-cli-script-migration.md")
	requiredScriptMigration := []string{
		"`go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`",
	}
	for _, needle := range requiredScriptMigration {
		if !strings.Contains(scriptMigration, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing active Go entrypoint guidance %q", needle)
		}
	}

	disallowedScriptMigration := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}
	for _, needle := range disallowedScriptMigration {
		if strings.Contains(scriptMigration, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not reference removed Python helper %q", needle)
		}
	}

	rootMigration := readRepoFile(t, repo, "docs/go-cli-script-migration-plan.md")
	requiredRootMigration := []string{
		"- retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"- root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"- retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"- `bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`",
	}
	for _, needle := range requiredRootMigration {
		if !strings.Contains(rootMigration, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing migration guidance %q", needle)
		}
	}

	disallowedRootMigration := []string{
		"`python3 scripts/create_issues.py",
		"`python3 scripts/dev_smoke.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}
	for _, needle := range disallowedRootMigration {
		if strings.Contains(rootMigration, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not present retired Python path %q", needle)
		}
	}
}
