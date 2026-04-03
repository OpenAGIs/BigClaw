package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPythonResidualSweepRepoBaseline(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	var pythonFiles []string

	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".py" {
			relativePath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo root: %v", err)
	}
	if len(pythonFiles) != 0 {
		sort.Strings(pythonFiles)
		t.Fatalf("expected zero live Python files after residual sweep, found %s", strings.Join(pythonFiles, ", "))
	}
}

func TestPythonResidualSweepGoCompatibilitySurfaces(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	goCompatibilityFiles := []string{
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/legacyshim/compilecheck.go",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
	}
	for _, relativePath := range goCompatibilityFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected residual sweep compatibility surface to exist: %s (%v)", relativePath, err)
		}
	}
}
