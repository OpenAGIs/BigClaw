package regression

import (
	"strings"
	"testing"
)

func TestReadinessFollowUpIndexDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/benchmark-readiness-report.md",
			substrings: []string{
				"docs/reports/capacity-certification-report.md",
				"docs/reports/capacity-certification-matrix.json",
				"docs/reports/parallel-follow-up-index.md",
				"remaining coordination, takeover, continuation, and broker-durability",
				"OPE-269` / `BIG-PAR-080",
				"OPE-261` / `BIG-PAR-085",
				"OPE-271` / `BIG-PAR-082",
				"OPE-222`",
			},
		},
		{
			path: "docs/reports/epic-closure-readiness-report.md",
			substrings: []string{
				"docs/reports/parallel-validation-matrix.md",
				"docs/reports/parallel-follow-up-index.md",
				"remaining takeover, coordination, continuation, and broker-durability",
				"OPE-269` / `BIG-PAR-080",
				"OPE-261` / `BIG-PAR-085",
				"OPE-271` / `BIG-PAR-082",
				"OPE-222`",
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
