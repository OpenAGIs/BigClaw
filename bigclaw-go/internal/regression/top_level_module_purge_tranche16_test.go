package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche16(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/audit_events.py",
		"src/bigclaw/collaboration.py",
		"src/bigclaw/console_ia.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/runtime.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/worker/runtime.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
