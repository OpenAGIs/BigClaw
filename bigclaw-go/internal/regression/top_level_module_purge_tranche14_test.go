package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/scheduler.py",
		"src/bigclaw/service.py",
		"src/bigclaw/ui_review.py",
		"src/bigclaw/workflow.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/workflow/engine.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
