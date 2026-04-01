package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche5(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/__main__.py",
		"src/bigclaw/__init__.py",
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/event_bus.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
		"tests/test_control_center.py",
		"tests/test_dsl.py",
		"tests/test_evaluation.py",
		"tests/test_event_bus.py",
		"tests/test_operations.py",
		"tests/test_orchestration.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_repo_links.py",
		"tests/test_repo_rollout.py",
		"tests/test_risk.py",
		"tests/test_runtime_matrix.py",
		"tests/test_scheduler.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/cmd/bigclawd/main_test.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/main_test.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/api/server_test.go",
		"bigclaw-go/internal/api/metrics.go",
		"bigclaw-go/internal/api/v2.go",
		"bigclaw-go/internal/events/bus.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/events/log.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/regression/regression.go",
		"bigclaw-go/internal/regression/regression_test.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/workflow/orchestration.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestTopLevelModulePurgePythonCountDropsAfterTranche5(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	count := 0
	if err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".venv" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".py") {
			count++
		}
		return nil
	}); err != nil {
		t.Fatalf("walk repo root: %v", err)
	}

	const prePurgePythonFileCount = 61
	const expectedPostPurgePythonFileCount = 39
	if count >= prePurgePythonFileCount {
		t.Fatalf("expected Python file count to drop below %d, got %d", prePurgePythonFileCount, count)
	}
	if count != expectedPostPurgePythonFileCount {
		t.Fatalf("expected Python file count to land at %d after tranche 5 purge, got %d", expectedPostPurgePythonFileCount, count)
	}
}
