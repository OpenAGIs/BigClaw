package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche29(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"tests/test_queue.py",
		"tests/test_runtime_matrix.py",
		"tests/test_scheduler.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche29_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}
