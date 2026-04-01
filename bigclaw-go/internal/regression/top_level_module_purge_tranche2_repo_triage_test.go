package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesRepoTriagePythonModule(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "..", "src/bigclaw/repo_triage.py")); !os.IsNotExist(err) {
		t.Fatalf("expected retired python module %q to be absent, got err=%v", "src/bigclaw/repo_triage.py", err)
	}
}

func TestTopLevelModulePurgeTranche2RepoTriageGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### BIG-GOM-305 Control center, triage, and operations view migration",
		"- `src/bigclaw/repo_triage.py`",
		"- `bigclaw-go/internal/triage/triage.go`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
