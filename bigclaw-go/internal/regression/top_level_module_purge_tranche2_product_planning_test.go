package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesProductPlanningPythonModules(t *testing.T) {
	root := repoRoot(t)

	for _, relative := range []string{
		"src/bigclaw/collaboration.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/roadmap.py",
	} {
		if _, err := os.Stat(filepath.Join(root, "..", relative)); !os.IsNotExist(err) {
			t.Fatalf("expected retired python module %q to be absent, got err=%v", relative, err)
		}
	}
}

func TestTopLevelModulePurgeTranche2ProductPlanningGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### 4. Port repo collaboration and lineage surfaces to Go",
		"- `src/bigclaw/collaboration.py`",
		"- `src/bigclaw/issue_archive.py`",
		"- `src/bigclaw/roadmap.py`",
		"- `bigclaw-go/internal/flow/flow.go`",
		"- `bigclaw-go/internal/product/console.go`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
