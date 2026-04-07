package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonTestTranche14Removed(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(repoRoot, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted Python tests directory to stay absent: %v", err)
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/uireview/uireview_test.go",
		"bigclaw-go/internal/reportstudio/reportstudio_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
