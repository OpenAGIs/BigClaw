package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptsDir := filepath.Join(repoRoot, "scripts")

	err := filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			t.Fatalf("expected no Python helper in scripts tree, found %s", relPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk scripts directory: %v", err)
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

	disallowedPhrases := []string{
		"Continue the remaining non-e2e script migrations in follow-up batches",
		"Python helpers under `bigclaw-go/scripts/e2e/`",
	}
	for _, needle := range disallowedPhrases {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not describe the scripts migration as unfinished: %q", needle)
		}
	}
}

func TestScriptMigrationPlanDoesNotAdvertiseDeletedPythonHelpers(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "../docs/go-cli-script-migration-plan.md")

	disallowed := []string{
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"Continue the remaining `bigclaw-go/scripts/*` migration helpers and E2E utilities",
		"`bigclaw-go/scripts/*` is deferred to a follow-up migration lane",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not advertise stale script migration state: %q", needle)
		}
	}

	required := []string{
		"`bigclaw-go/scripts/*` automation surface",
		"`bigclaw-go/scripts/*` is now Python-free and Go-owned",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing updated scripts migration wording %q", needle)
		}
	}
}
