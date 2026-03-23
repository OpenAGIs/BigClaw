package regression

import (
	"strings"
	"testing"
)

func TestBundleFollowUpIndexDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/go-control-plane-observability-report.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining observability follow-up digests, companion evidence, and rollout",
				"GET /v2/reports/distributed/export",
			},
		},
		{
			path: "docs/reports/live-shadow-index.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining live-shadow, rollback, and corpus-coverage follow-up digests",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/live-shadow-runs/20260313T085655Z/README.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining live-shadow, rollback, and corpus-coverage follow-up digests behind",
				"OPE-266` / `BIG-PAR-092",
				"OPE-254` / `BIG-PAR-088",
			},
		},
		{
			path: "docs/reports/live-validation-runs/20260316T140138Z/README.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining parallel follow-up digests and rollout contracts behind the live",
				"OPE-271` / `BIG-PAR-082",
				"docs/reports/validation-bundle-continuation-policy-gate.json",
			},
		},
		{
			path: "docs/reports/multi-node-coordination-report.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining coordination, takeover, and validation-continuation caveats behind",
				"docs/reports/parallel-validation-matrix.md",
				"docs/reports/cross-process-coordination-boundary-digest.md",
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
