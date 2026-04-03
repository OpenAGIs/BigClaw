package regression

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1169CandidatePythonFilesRemainDeleted(t *testing.T) {
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
		_, err := os.Stat(filepath.Join(repoRoot, relativePath))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected BIG-GO-1169 candidate path to stay deleted: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1169RepoContainsNoPythonFiles(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	var pythonFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".py") {
			relPath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relPath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for Python files: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repo-level Python file count to stay at zero, found %d: %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1169MigrationPlanDocumentsReplacementPaths(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	migrationPlan := readRepoFile(t, repoRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"`BIG-GO-1169` confirms the repo-wide physical Python count is already `0`",
		"`bigclaw-go/scripts/benchmark/capacity_certification.py`",
		"`bigclaw-go/scripts/e2e/export_validation_bundle.py`",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
		"`scripts/create_issues.py`",
		"`scripts/dev_smoke.py`",
		"`bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclawctl automation e2e ...`",
		"`bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
		"`find . -name '*.py' | wc -l`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1169 sweep guidance %q", needle)
		}
	}
}
