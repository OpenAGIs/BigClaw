package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/planning.py",
		"src/bigclaw/queue.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/risk.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reportstudio/reportstudio.go",
		"bigclaw-go/internal/risk/risk.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
