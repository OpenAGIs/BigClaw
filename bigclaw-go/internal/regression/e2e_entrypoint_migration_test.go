package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutomationScriptDirectoriesStayPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	for _, dir := range []string{"benchmark", "e2e", "migration"} {
		scriptDir := filepath.Join(repoRoot, "scripts", dir)
		if _, err := os.Stat(scriptDir); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("stat %s script directory: %v", dir, err)
		}
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

	disallowed := []string{
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not reference removed Python helper %q", needle)
		}
	}
}
