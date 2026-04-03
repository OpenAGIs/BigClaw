package regression

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1164RepositoryStaysPythonFree(t *testing.T) {
	rootRepo := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	var pythonFiles []string
	err := filepath.WalkDir(rootRepo, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".idea", ".vscode":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") {
			relativePath, relErr := filepath.Rel(rootRepo, path)
			if relErr != nil {
				return relErr
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository for residual python files: %v", err)
	}
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository baseline to stay python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1164DocsPinZeroPythonBaseline(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	rootDoc := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	requiredGoDoc := []string{
		"Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`, `BIG-GO-1164`",
		"## BIG-GO-1160 / BIG-GO-1164 Sweep Coverage",
		"`BIG-GO-1164` keeps the repository-wide Python count pinned at `0`",
	}
	for _, needle := range requiredGoDoc {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing BIG-GO-1164 coverage %q", needle)
		}
	}

	requiredRootDoc := []string{
		"`BIG-GO-1160` extends that migration evidence",
		"`BIG-GO-1164` closes the residual sweep by pinning the current repository",
		"baseline at `find . -name '*.py' | wc -l == 0`",
	}
	for _, needle := range requiredRootDoc {
		if !strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1164 coverage %q", needle)
		}
	}
}
