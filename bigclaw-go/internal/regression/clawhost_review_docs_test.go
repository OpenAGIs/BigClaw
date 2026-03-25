package regression

import (
	"strings"
	"testing"
)

func TestClawHostReviewDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/clawhost-control-plane-review-index.md",
			substrings: []string{
				"PR `#186`",
				"merged into `main` on",
				"`BIG-PAR-292` - ClawHost lifecycle recovery and per-bot isolation scorecard",
				"`BIG-PAR-293` through `BIG-PAR-297`",
				"Reviewer-facing readiness surface:",
				"Reviewer-facing recovery scorecard:",
				"internal/product/clawhost_workflow.go",
				"internal/api/clawhost_readiness_surface.go",
				"internal/product/clawhost_recovery.go",
				"internal/api/server_test.go",
				"/v2/clawhost/recovery-scorecard",
				"PR state: `MERGED`",
				"Merge commit: `bd854cd1f53808d529d7ae7413f01320eeeef337`",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"docs/reports/clawhost-control-plane-review-index.md",
				"merged fleet, tenant-policy, validation, rollout, workflow, readiness, and recovery surfaces",
				"aggregate `/v2/control-center` regression coverage",
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
