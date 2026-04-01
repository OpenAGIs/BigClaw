package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonResidualSweepRemovesRuntimeAndUIReview(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/reports.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/ui_review.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/worker/runtime_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}

	compatibilityShimFiles := []string{
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/_legacy/reports.legacy",
	}
	for _, relativePath := range compatibilityShimFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected compatibility shim to exist: %s (%v)", relativePath, err)
		}
	}
}
