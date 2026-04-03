package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche14(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/models.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/planning.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/api/v2.go",
		"bigclaw-go/internal/api/server_test.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/product/clawhost_workflows.go",
		"bigclaw-go/internal/product/clawhost_workflows_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"src/bigclaw/_compat_schema.py",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
