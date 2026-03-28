package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	return testharness.RepoRoot(t)
}

func readRepoFile(t *testing.T, root string, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(contents)
}

func resolveRepoPath(root, candidate string) string {
	return filepath.Join(root, strings.TrimPrefix(candidate, "bigclaw-go/"))
}
