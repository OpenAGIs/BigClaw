package regression

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestBIGGO1167CandidatePythonFilesRemainDeleted(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	candidates := []string{
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

	for _, relativePath := range candidates {
		_, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(relativePath)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected BIG-GO-1167 candidate path to stay deleted: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1167RepositoryStaysPythonFree(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	pythonFiles := 0

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".symphony":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".py" {
			pythonFiles++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for python files: %v", err)
	}
	if pythonFiles != 0 {
		t.Fatalf("expected repository python file count to remain zero, found %d", pythonFiles)
	}
}

func TestBIGGO1167GoReplacementPathsRemainPresent(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	requiredPaths := []string{
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"docs/go-cli-script-migration-plan.md",
		"scripts/ops/bigclawctl",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement or compatibility path to exist: %s (%v)", relativePath, err)
		}
	}
}
