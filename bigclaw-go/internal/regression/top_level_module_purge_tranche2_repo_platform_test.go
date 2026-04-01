package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesRepoPlatformPythonModules(t *testing.T) {
	root := repoRoot(t)

	for _, relative := range []string{
		"src/bigclaw/repo_plane.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/repo_board.py",
	} {
		if _, err := os.Stat(filepath.Join(root, "..", relative)); !os.IsNotExist(err) {
			t.Fatalf("expected retired python module %q to be absent, got err=%v", relative, err)
		}
	}
}

func TestTopLevelModulePurgeTranche2RepoPlatformGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### BIG-GOM-306 Repo collaboration and lineage surface migration",
		"- `src/bigclaw/repo_plane.py`",
		"- `src/bigclaw/repo_board.py`",
		"- `src/bigclaw/repo_registry.py`",
		"- optional new package family: `bigclaw-go/internal/repo/*`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
