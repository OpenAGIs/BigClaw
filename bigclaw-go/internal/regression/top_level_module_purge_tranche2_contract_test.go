package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesExecutionContractPythonModule(t *testing.T) {
	root := repoRoot(t)

	if _, err := os.Stat(filepath.Join(root, "..", "src/bigclaw/execution_contract.py")); !os.IsNotExist(err) {
		t.Fatalf("expected retired python module %q to be absent, got err=%v", "src/bigclaw/execution_contract.py", err)
	}
}

func TestTopLevelModulePurgeTranche2ExecutionContractGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### BIG-GOM-302 Risk, policy, and approval semantics migration",
		"- `src/bigclaw/execution_contract.py`",
		"- `bigclaw-go/internal/contract/execution.go` now owns the Go execution contract, permission matrix, and operations API contract migrated from `src/bigclaw/execution_contract.py`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
