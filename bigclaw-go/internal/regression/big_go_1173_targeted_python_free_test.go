package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1173TargetedResidualDirectoriesStayPythonFree(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	checks := []struct {
		relativePath string
		allowMissing bool
	}{
		{relativePath: "src/bigclaw", allowMissing: true},
		{relativePath: "tests", allowMissing: true},
		{relativePath: "scripts", allowMissing: false},
		{relativePath: "bigclaw-go/scripts", allowMissing: false},
	}

	for _, check := range checks {
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(check.relativePath))
		info, err := os.Stat(fullPath)
		if err != nil {
			if check.allowMissing && os.IsNotExist(err) {
				continue
			}
			t.Fatalf("stat %s: %v", check.relativePath, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected directory at %s", check.relativePath)
		}

		var pythonFiles []string
		err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".py") {
				relative, err := filepath.Rel(repoRoot, path)
				if err != nil {
					return err
				}
				pythonFiles = append(pythonFiles, filepath.ToSlash(relative))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", check.relativePath, err)
		}
		if len(pythonFiles) != 0 {
			t.Fatalf("expected %s to stay Python-free, found %v", check.relativePath, pythonFiles)
		}
	}
}

func TestBIGGO1173CloseoutDocumentsReplacementEvidence(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	report := readRepoFile(t, repoRoot, "reports/BIG-GO-1173-validation.md")

	required := []string{
		"# BIG-GO-1173 Validation",
		"`find . -name '*.py' | wc -l`",
		"`src/bigclaw` is absent in the current checkout.",
		"`tests` is absent in the current checkout.",
		"`scripts/` remains present and Python-free",
		"`bigclaw-go/scripts/{benchmark,e2e}` remains present and Python-free",
		"`bigclawctl` Go entrypoints and retained shell wrappers are the concrete replacement evidence for this lane.",
	}
	for _, needle := range required {
		if !strings.Contains(report, needle) {
			t.Fatalf("reports/BIG-GO-1173-validation.md missing %q", needle)
		}
	}
}
