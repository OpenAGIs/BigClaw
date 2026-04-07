package regression

import (
	"strings"
	"testing"
)

func TestE2EValidationDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/e2e-validation.md")

	requiredSubstrings := []string{
		"docs/reports/parallel-validation-matrix.md",
		"docs/reports/parallel-follow-up-index.md",
		"remaining continuation, coordination, takeover, and broker-durability",
		"OPE-271` / `BIG-PAR-082",
		"OPE-261` / `BIG-PAR-085",
		"OPE-269` / `BIG-PAR-080",
		"OPE-222`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/e2e-validation.md missing substring %q", needle)
		}
	}
}
