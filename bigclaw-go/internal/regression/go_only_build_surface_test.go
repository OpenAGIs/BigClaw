package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootGoOnlyBuildSurfaceStaysAligned(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	for _, relative := range []string{"pyproject.toml", "setup.py"} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to stay absent, err=%v", relative, err)
		}
	}

	readme := readRepoFile(t, root, "README.md")
	for _, snippet := range []string{
		"repository root now exposes Go-only build entrypoints;",
		"Root build and operator entrypoints are Go-only via `make ...`",
		"`bash scripts/ops/bigclawctl ...`",
	} {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("README missing Go-only build surface snippet %q", snippet)
		}
	}

	matches, err := filepath.Glob(filepath.Join(root, "scripts", "ops", "*.py"))
	if err != nil {
		t.Fatalf("glob scripts/ops/*.py: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no Python operator shims under scripts/ops, got %v", matches)
	}
}
