package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTopLevelModulePurgeTranche2(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"src/bigclaw/repo_board.py",
		"src/bigclaw/repo_commits.py",
		"src/bigclaw/repo_gateway.py",
		"src/bigclaw/repo_governance.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/repo_triage.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/repo/commits.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/repo/triage.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
