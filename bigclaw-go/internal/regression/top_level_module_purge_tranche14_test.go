package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"tests/test_live_shadow_bundle.py",
		"tests/test_models.py",
		"tests/test_observability.py",
		"tests/test_operations.py",
		"tests/test_orchestration.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_reports.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_links.py",
		"tests/test_repo_rollout.py",
		"tests/test_risk.py",
		"tests/test_runtime_matrix.py",
		"tests/test_scheduler.py",
		"tests/test_ui_review.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python test asset to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/api/expansion_test.go",
		"bigclaw-go/internal/product/saved_views_test.go",
		"bigclaw-go/internal/queue/file_queue_test.go",
		"bigclaw-go/internal/queue/sqlite_queue_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/observability_runtime_surface_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement coverage file to exist: %s (%v)", relativePath, err)
		}
	}
}
