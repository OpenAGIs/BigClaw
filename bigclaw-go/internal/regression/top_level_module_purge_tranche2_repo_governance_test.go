package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesRepoGovernancePythonModule(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "..", "src/bigclaw/repo_governance.py")); !os.IsNotExist(err) {
		t.Fatalf("expected retired python module %q to be absent, got err=%v", "src/bigclaw/repo_governance.py", err)
	}
}

func TestTopLevelModulePurgeTranche2RepoGovernanceGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"- `src/bigclaw/repo_governance.py`",
		"- `bigclaw-go/internal/repo/governance.go` now ports `src/bigclaw/repo_governance.py` into a Go-owned repo permission matrix and audit-field contract",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
