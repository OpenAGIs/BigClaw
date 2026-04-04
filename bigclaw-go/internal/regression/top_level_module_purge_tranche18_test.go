package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche18(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/service.py",
		"src/bigclaw/scheduler.py",
		"src/bigclaw/workflow.py",
		"src/bigclaw/ui_review.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/scheduler/scheduler_test.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/workflow/orchestration.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
