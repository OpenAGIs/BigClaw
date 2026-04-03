package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonResidualTestsSweepBRemoved(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
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
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python test to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/pilot/rollout_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/queue/file_queue_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/repo/governance_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/reportstudio/reportstudio_test.go",
		"bigclaw-go/internal/risk/risk_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"bigclaw-go/internal/product/saved_views_test.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/service/server_test.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
		"bigclaw-go/internal/policy/validation_test.go",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestPythonResidualTestsSweepBLeavesNoPythonFiles(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	var pythonFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && filepath.Ext(path) == ".py" {
			relativePath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			pythonFiles = append(pythonFiles, relativePath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo for Python files: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repo to remain Python-free, found %v", pythonFiles)
	}
}
