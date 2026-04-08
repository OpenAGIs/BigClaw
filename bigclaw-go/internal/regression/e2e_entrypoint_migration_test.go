package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	e2eDir := filepath.Join(repoRoot, "scripts", "e2e")

	entries, err := os.ReadDir(e2eDir)
	if err != nil {
		t.Fatalf("read e2e script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isPythonAssetPath(entry.Name()) {
			t.Fatalf("expected no Python asset in scripts/e2e, found %s", entry.Name())
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
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not reference removed Python helper %q", needle)
		}
	}
}
