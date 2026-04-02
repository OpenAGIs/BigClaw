package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche27(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"tests/conftest.py",
		"tests/test_console_ia.py",
		"tests/test_control_center.py",
		"tests/test_design_system.py",
		"tests/test_evaluation.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/pilot/report.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/api/expansion.go",
		"bigclaw-go/internal/api/expansion_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
