package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelModulePurgeTranche2RemovesRiskPolicyPythonModules(t *testing.T) {
	root := repoRoot(t)

	for _, relative := range []string{
		"src/bigclaw/risk.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/audit_events.py",
	} {
		if _, err := os.Stat(filepath.Join(root, "..", relative)); !os.IsNotExist(err) {
			t.Fatalf("expected retired python module %q to be absent, got err=%v", relative, err)
		}
	}
}

func TestTopLevelModulePurgeTranche2RiskPolicyGoOwnershipDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	contents := readRepoFile(t, root, "../docs/go-mainline-cutover-issue-pack.md")

	requiredSnippets := []string{
		"### BIG-GOM-302 Risk, policy, and approval semantics migration",
		"- `src/bigclaw/risk.py`",
		"- `src/bigclaw/governance.py`",
		"- `src/bigclaw/audit_events.py`",
		"- `bigclaw-go/internal/governance/freeze.go` now owns the Go scope-freeze backlog board and governance audit surface migrated from `src/bigclaw/governance.py`",
		"- `bigclaw-go/internal/observability/audit_spec.go` now owns the canonical P0 audit event spec registry migrated from `src/bigclaw/audit_events.py`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(contents, snippet) {
			t.Fatalf("go cutover issue pack missing snippet %q", snippet)
		}
	}
}
