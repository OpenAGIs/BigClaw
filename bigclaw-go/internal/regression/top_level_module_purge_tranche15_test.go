package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche15(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/governance.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/ui_review.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/regression/followup_lane_docs_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
