package regression

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1177PriorityAreasStayPythonFree(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	priorityRoots := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}
	for _, relativeRoot := range priorityRoots {
		absoluteRoot := filepath.Join(repoRoot, filepath.FromSlash(relativeRoot))
		assertTreeHasNoPythonFiles(t, absoluteRoot, relativeRoot)
	}
}

func TestBIGGO1177GoNativeReplacementsRemainAvailable(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	replacements := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/regression/big_go_1160_script_migration_test.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
	}
	for _, relativePath := range replacements {
		_, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(relativePath)))
		if err != nil {
			t.Fatalf("expected Go/native replacement to exist: %s (%v)", relativePath, err)
		}
	}
}

func assertTreeHasNoPythonFiles(t *testing.T, absoluteRoot, relativeRoot string) {
	t.Helper()

	err := filepath.WalkDir(absoluteRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) || strings.Contains(walkErr.Error(), "no such file or directory") {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			t.Fatalf("expected %s to stay Python-free, found %s", relativeRoot, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", relativeRoot, err)
	}
}
