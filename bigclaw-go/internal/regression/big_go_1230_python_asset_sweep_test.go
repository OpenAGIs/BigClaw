package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBIGGO1230LaneInventoryStaysEmpty(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	laneDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range laneDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected lane inventory to remain empty for %s, found %v", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1230GoReplacementEntrypointsExist(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	goReplacementFiles := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/regression/big_go_1220_zero_python_guard_test.go",
	}

	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go-owned replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}
