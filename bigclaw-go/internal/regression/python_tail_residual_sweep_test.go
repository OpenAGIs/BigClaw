package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonResidualSweepKeepsTargetedModulesPurged(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/repo_links.py",
		"src/bigclaw/repo_plane.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/repo_triage.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/roadmap.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/saved_views.py",
		"src/bigclaw/scheduler.py",
		"src/bigclaw/service.py",
		"src/bigclaw/ui_review.py",
		"src/bigclaw/validation_policy.py",
		"src/bigclaw/workflow.py",
		"src/bigclaw/workspace_bootstrap.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
		"src/bigclaw/workspace_bootstrap_validation.py",
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
