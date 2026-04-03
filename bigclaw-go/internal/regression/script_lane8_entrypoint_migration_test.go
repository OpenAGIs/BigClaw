package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertRepoPathsAbsent(t *testing.T, root string, paths []string) {
	t.Helper()
	for _, relative := range paths {
		_, err := os.Stat(filepath.Join(root, relative))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to stay absent, stat err=%v", relative, err)
		}
	}
}

func TestLane8CandidatePythonFilesStayRemoved(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	assertRepoPathsAbsent(t, repoRoot, []string{
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
	})
}

func TestBenchmarkScriptDirectoryStaysPythonFree(t *testing.T) {
	goRoot := repoRoot(t)
	benchmarkDir := filepath.Join(goRoot, "scripts", "benchmark")

	entries, err := os.ReadDir(benchmarkDir)
	if err != nil {
		t.Fatalf("read benchmark script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected no Python helper in scripts/benchmark, found %s", entry.Name())
		}
	}
}

func TestMigrationScriptDirectoryStaysPythonFree(t *testing.T) {
	goRoot := repoRoot(t)
	migrationDir := filepath.Join(goRoot, "scripts", "migration")

	if _, err := os.Stat(migrationDir); err == nil {
		entries, readErr := os.ReadDir(migrationDir)
		if readErr != nil {
			t.Fatalf("read migration script directory: %v", readErr)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".py") {
				t.Fatalf("expected no Python helper in scripts/migration, found %s", entry.Name())
			}
		}
		return
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat migration script directory: %v", err)
	}
}

func TestRepoRootScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	scriptDir := filepath.Join(repoRoot, "scripts")

	entries, err := os.ReadDir(scriptDir)
	if err != nil {
		t.Fatalf("read repo root script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected no Python helper in repo-root scripts, found %s", entry.Name())
		}
	}
}

func TestScriptMigrationPlanListsOnlyActiveGoEntrypoints(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")

	required := []string{
		"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
		"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
		"retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclaw-go/scripts/e2e/` operator entrypoints now dispatch through `bigclawctl automation e2e ...`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`",
		"`bigclaw-go/scripts/migration/shadow_matrix.py` -> `bigclawctl automation migration shadow-matrix`",
		"`bigclaw-go/scripts/migration/live_shadow_scorecard.py` -> `bigclawctl automation migration live-shadow-scorecard`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py` -> `bigclawctl automation migration export-live-shadow-bundle`",
		"`cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`",
		"`cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`",
		"`cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing active entrypoint guidance %q", needle)
		}
	}

	disallowed := []string{
		"scripts/dev_smoke.py",
		"scripts/create_issues.py` shim",
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py` shim",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py` shim",
		"bigclaw-go/scripts/e2e/external_store_validation.py` shim",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py` shim",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py` shim",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` shim",
		"bigclaw-go/scripts/migration/shadow_compare.py` shim",
		"bigclaw-go/scripts/migration/shadow_matrix.py` shim",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py` shim",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py` shim",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not reference retired Python shim guidance %q", needle)
		}
	}
}
