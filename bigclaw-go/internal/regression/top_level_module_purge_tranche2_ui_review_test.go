package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesUIReviewPythonModule(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "..", "src/bigclaw/ui_review.py")); !os.IsNotExist(err) {
		t.Fatalf("expected retired python module %q to be absent, got err=%v", "src/bigclaw/ui_review.py", err)
	}
}

func TestTopLevelModulePurgeTranche2UIReviewGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### 5. Port operator console and saved-view surfaces to Go",
		"- `src/bigclaw/ui_review.py`",
		"- `bigclaw-go/internal/product/console.go`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
