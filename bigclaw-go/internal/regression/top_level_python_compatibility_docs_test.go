package regression

import (
	"strings"
	"testing"
)

func TestTopLevelPythonCompatibilityDocs(t *testing.T) {
	root := regressionRepoRoot(t)

	readme := readRepoFile(t, root, "README.md")
	requiredReadme := []string{
		"The sole remaining legacy Python compatibility file is",
		"`src/bigclaw/__init__.py`",
		"frozen for migration-only reference use",
		"`python -m bigclaw` entrypoint has been retired",
	}
	for _, needle := range requiredReadme {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing final-state compatibility wording %q", needle)
		}
	}
	if strings.Contains(readme, "The legacy Python execution-kernel modules in") {
		t.Fatal("README.md should not describe multiple legacy Python execution-kernel modules as retained compatibility surfaces")
	}

	handoff := readRepoFile(t, root, "docs/go-mainline-cutover-handoff.md")
	requiredHandoff := []string{
		"the sole remaining Python compatibility file",
		"`src/bigclaw/__init__.py`",
		"frozen for migration-only reference use",
	}
	for _, needle := range requiredHandoff {
		if !strings.Contains(handoff, needle) {
			t.Fatalf("docs/go-mainline-cutover-handoff.md missing final-state compatibility wording %q", needle)
		}
	}
	if strings.Contains(handoff, "remaining Python runtime surfaces are") {
		t.Fatal("docs/go-mainline-cutover-handoff.md should not describe multiple remaining Python runtime surfaces")
	}

	issuePack := readRepoFile(t, root, "docs/go-mainline-cutover-issue-pack.md")
	requiredIssuePack := []string{
		"The sole remaining Python compatibility file is `src/bigclaw/__init__.py`",
		"explicitly frozen as a migration-only path",
	}
	for _, needle := range requiredIssuePack {
		if !strings.Contains(issuePack, needle) {
			t.Fatalf("docs/go-mainline-cutover-issue-pack.md missing final-state compatibility wording %q", needle)
		}
	}
	if strings.Contains(issuePack, "The remaining Python runtime entrypoints are explicitly frozen as migration-only compatibility paths.") {
		t.Fatal("docs/go-mainline-cutover-handoff.md missing single-file compatibility wording")
	}
	if strings.Contains(issuePack, "remaining Python runtime entrypoints are") {
		t.Fatal("docs/go-mainline-cutover-issue-pack.md should not describe multiple remaining Python runtime entrypoints")
	}
}
