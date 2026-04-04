package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBIGGO1186RemainingPythonAssetInventoryIsEmpty(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free for BIG-GO-1186, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1186PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected BIG-GO-1186 priority directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1186GoReplacementEntrypointsRemainAvailable(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected BIG-GO-1186 Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}
