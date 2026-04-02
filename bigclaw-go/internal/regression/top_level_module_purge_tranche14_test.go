package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/audit_events.py",
		"src/bigclaw/collaboration.py",
		"src/bigclaw/console_ia.py",
		"src/bigclaw/deprecation.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/models.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/ui_review.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/observability/task_run.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/workflow/model.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
