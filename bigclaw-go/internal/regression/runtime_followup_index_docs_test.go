package regression

import (
	"strings"
	"testing"
)

func TestRuntimeFollowUpIndexDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining `BIG-PAR-*` follow-up digests, reviewer-facing companion evidence,",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining `BIG-PAR-*` follow-up digests, capability surfaces, and rollout",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining event-bus takeover, coordination-boundary, and rollout follow-up",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/live-validation-index.md",
			substrings: []string{
				"docs/reports/parallel-validation-matrix.md",
				"docs/reports/parallel-follow-up-index.md",
				"remaining follow-up digests and rollout contracts behind those validation",
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
