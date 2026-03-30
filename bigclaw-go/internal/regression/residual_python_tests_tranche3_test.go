package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResidualPythonTestsTranche3StaysRemoved(t *testing.T) {
	repoRoot := repoRoot(t)
	removed := []string{
		"tests/conftest.py",
		"tests/test_console_ia.py",
		"tests/test_design_system.py",
		"tests/test_operations.py",
		"tests/test_reports.py",
		"tests/test_ui_review.py",
	}

	for _, relative := range removed {
		location := filepath.Join(repoRoot, relative)
		if _, err := os.Stat(location); !os.IsNotExist(err) {
			t.Fatalf("expected tranche-3 residual Python test to stay removed: %s (err=%v)", relative, err)
		}
	}

	supportingGoCoverage := []string{
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/regression/residual_python_tests_tranche3_test.go",
	}
	for _, relative := range supportingGoCoverage {
		location := resolveRepoPath(repoRoot, relative)
		if _, err := os.Stat(location); err != nil {
			t.Fatalf("expected supporting repo-native coverage file %s: %v", relative, err)
		}
	}
}
