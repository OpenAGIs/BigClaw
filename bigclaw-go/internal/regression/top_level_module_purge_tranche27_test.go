package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche27(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"tests/test_live_shadow_bundle.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_validation_bundle_continuation_policy_gate.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/live_shadow_docs_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/internal/regression/runtime_report_followup_docs_test.go",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche27_test.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}
