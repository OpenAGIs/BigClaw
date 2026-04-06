package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var bigGO1552DeletedTests = []string{
	"tests/conftest.py",
	"tests/test_audit_events.py",
	"tests/test_connectors.py",
	"tests/test_console_ia.py",
	"tests/test_control_center.py",
	"tests/test_cost_control.py",
	"tests/test_cross_process_coordination_surface.py",
	"tests/test_dashboard_run_contract.py",
	"tests/test_deprecation.py",
	"tests/test_design_system.py",
	"tests/test_dsl.py",
	"tests/test_evaluation.py",
	"tests/test_event_bus.py",
	"tests/test_execution_contract.py",
	"tests/test_execution_flow.py",
	"tests/test_followup_digests.py",
	"tests/test_github_sync.py",
	"tests/test_governance.py",
	"tests/test_issue_archive.py",
	"tests/test_legacy_shim.py",
	"tests/test_live_shadow_bundle.py",
	"tests/test_live_shadow_scorecard.py",
	"tests/test_mapping.py",
	"tests/test_memory.py",
	"tests/test_models.py",
	"tests/test_observability.py",
	"tests/test_operations.py",
	"tests/test_orchestration.py",
	"tests/test_parallel_refill.py",
	"tests/test_parallel_validation_bundle.py",
	"tests/test_pilot.py",
	"tests/test_planning.py",
	"tests/test_queue.py",
	"tests/test_repo_board.py",
	"tests/test_repo_collaboration.py",
	"tests/test_repo_gateway.py",
	"tests/test_repo_governance.py",
	"tests/test_repo_links.py",
	"tests/test_repo_registry.py",
	"tests/test_repo_rollout.py",
	"tests/test_repo_triage.py",
	"tests/test_reports.py",
	"tests/test_risk.py",
	"tests/test_roadmap.py",
	"tests/test_runtime.py",
	"tests/test_runtime_matrix.py",
	"tests/test_saved_views.py",
	"tests/test_scheduler.py",
	"tests/test_service.py",
	"tests/test_shadow_matrix_corpus.py",
	"tests/test_subscriber_takeover_harness.py",
	"tests/test_ui_review.py",
	"tests/test_validation_bundle_continuation_policy_gate.py",
	"tests/test_validation_bundle_continuation_scorecard.py",
	"tests/test_validation_policy.py",
	"tests/test_workflow.py",
	"tests/test_workspace_bootstrap.py",
}

func TestBIGGO1552RepositoryPythonCountDrop(t *testing.T) {
	repoRoot := repoRoot(t)
	workspaceRoot := filepath.Clean(filepath.Join(repoRoot, ".."))
	pythonFiles := collectPythonFiles(t, workspaceRoot)
	if len(pythonFiles) != 58 {
		t.Fatalf("expected 58 repository python files after deleting tests, got %d", len(pythonFiles))
	}
}

func TestBIGGO1552TestsPythonFilesDeleted(t *testing.T) {
	repoRoot := repoRoot(t)
	workspaceRoot := filepath.Clean(filepath.Join(repoRoot, ".."))
	testPythonFiles := collectPythonFiles(t, filepath.Join(workspaceRoot, "tests"))
	if len(testPythonFiles) != 0 {
		t.Fatalf("expected tests python files to be fully removed, got %d: %v", len(testPythonFiles), testPythonFiles)
	}
	for _, relative := range bigGO1552DeletedTests {
		if _, err := os.Stat(filepath.Join(workspaceRoot, relative)); err == nil {
			t.Fatalf("expected deleted test file to be absent: %s", relative)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", relative, err)
		}
	}
}

func TestBIGGO1552LaneReportCapturesExactCounts(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "../.symphony/evidence-BIG-GO-1552.md")
	requiredSubstrings := []string{
		"# BIG-GO-1552 Evidence",
		"Repository `.py` files: `115 -> 58` (`-57`)",
		"In-scope `tests/*.py` files: `57 -> 0` (`-57`)",
		"`git diff --cached --name-only --diff-filter=D | sort | wc -l` -> `57`",
		"`go test -count=1 ./internal/regression -run 'TestBIGGO1552(RepositoryPythonCountDrop|TestsPythonFilesDeleted|LaneReportCapturesExactCounts)$'` -> `ok",
	}
	for _, relative := range bigGO1552DeletedTests {
		requiredSubstrings = append(requiredSubstrings, "`"+relative+"`")
	}
	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("evidence report missing substring %q", needle)
		}
	}
}

func collectPythonFiles(t *testing.T, root string) []string {
	t.Helper()
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("stat %s: %v", root, err)
	}
	if !info.IsDir() {
		if strings.HasSuffix(root, ".py") {
			return []string{root}
		}
		return nil
	}
	files := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type().IsRegular() && strings.HasSuffix(path, ".py") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}
