package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesRetiredPythonModules(t *testing.T) {
	root := repoRoot(t)

	for _, relative := range []string{
		"src/bigclaw/connectors.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/dsl.py",
	} {
		if _, err := os.Stat(filepath.Join(root, "..", relative)); !os.IsNotExist(err) {
			t.Fatalf("expected retired python module %q to be absent, got err=%v", relative, err)
		}
	}
}

func TestTopLevelModulePurgeTranche2GoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-domain-intake-parity-matrix.md")

	requiredSnippets := []string{
		"### `src/bigclaw/connectors.py`",
		"`SourceIssue` -> `bigclaw-go/internal/intake/types.go`",
		"`Connector` protocol -> `bigclaw-go/internal/intake/connector.go`",
		"### `src/bigclaw/mapping.py`",
		"`map_source_issue_to_task` -> `bigclaw-go/internal/intake/mapping.go`",
		"### `src/bigclaw/dsl.py`",
		"`WorkflowDefinition` -> `bigclaw-go/internal/workflow/definition.go`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go ownership parity matrix missing snippet %q", snippet)
		}
	}
}
