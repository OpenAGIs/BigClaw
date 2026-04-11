package regression

import (
	"strings"
	"testing"
)

func TestPlanningFollowUpIndexDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/migration-plan-review-notes.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining migration-shadow, rollback, and corpus-coverage follow-up digests",
				"docs/reports/parallel-validation-matrix.md",
				"OPE-266` / `BIG-PAR-092",
				"OPE-254` / `BIG-PAR-088",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"bigclaw-go/docs/reports/parallel-follow-up-index.md",
				"remaining follow-up digests and rollout contracts before",
				"bigclaw-go/docs/reports/parallel-validation-matrix.md",
				"OPE-266` / `BIG-PAR-092",
				"OPE-254` / `BIG-PAR-088",
				"OPE-271` / `BIG-PAR-082",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
