package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	for _, dir := range []string{"benchmark", "e2e"} {
		scriptDir := filepath.Join(repoRoot, "scripts", dir)

		entries, err := os.ReadDir(scriptDir)
		if err != nil {
			t.Fatalf("read %s script directory: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(entry.Name(), ".py") {
				t.Fatalf("expected no Python helper in scripts/%s, found %s", dir, entry.Name())
			}
		}
	}

	retiredCandidates := []string{
		"scripts/benchmark/capacity_certification.py",
		"scripts/benchmark/capacity_certification_test.py",
		"scripts/benchmark/run_matrix.py",
		"scripts/benchmark/soak_local.py",
		"scripts/e2e/broker_failover_stub_matrix.py",
		"scripts/e2e/broker_failover_stub_matrix_test.py",
		"scripts/e2e/cross_process_coordination_surface.py",
		"scripts/e2e/export_validation_bundle.py",
		"scripts/e2e/export_validation_bundle_test.py",
		"scripts/e2e/external_store_validation.py",
		"scripts/e2e/mixed_workload_matrix.py",
		"scripts/e2e/multi_node_shared_queue.py",
		"scripts/e2e/multi_node_shared_queue_test.py",
		"scripts/e2e/run_all_test.py",
		"scripts/e2e/run_task_smoke.py",
		"scripts/e2e/subscriber_takeover_fault_matrix.py",
		"scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
		"scripts/e2e/validation_bundle_continuation_scorecard.py",
		"scripts/migration/export_live_shadow_bundle.py",
		"scripts/migration/live_shadow_scorecard.py",
		"scripts/migration/shadow_compare.py",
		"scripts/migration/shadow_matrix.py",
	}
	for _, relPath := range retiredCandidates {
		if _, err := os.Stat(filepath.Join(repoRoot, relPath)); err == nil {
			t.Fatalf("expected retired Python lane file to stay removed: %s", relPath)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", relPath, err)
		}
	}
}

func TestE2EMigrationDocListsOnlyActiveEntrypoints(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration.md")

	required := []string{
		"`go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`./scripts/e2e/run_all.sh`",
		"`./scripts/e2e/kubernetes_smoke.sh`",
		"`./scripts/e2e/ray_smoke.sh`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing active entrypoint %q", needle)
		}
	}

	retiredInventory := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
	}
	for _, needle := range retiredInventory {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing retired lane inventory entry %q", needle)
		}
	}
	if !strings.Contains(contents, "candidate Python files covered by this lane were:") {
		t.Fatalf("docs/go-cli-script-migration.md missing retired lane inventory heading")
	}
}
