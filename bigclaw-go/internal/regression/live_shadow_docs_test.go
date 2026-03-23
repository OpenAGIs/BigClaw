package regression

import (
	"strings"
	"testing"
)

func TestLiveShadowRuntimeDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/migration-shadow.md",
			substrings: []string{
				"GET /debug/status",
				"live_shadow_mirror_scorecard",
				"GET /v2/control-center",
				"distributed_diagnostics.live_shadow_mirror_scorecard",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"OPE-266` / `BIG-PAR-092",
				"GET /debug/status` live shadow mirror payload",
				"GET /v2/control-center` distributed diagnostics live shadow mirror payload",
				"`live_shadow_mirror_scorecard`",
				"`distributed_diagnostics.live_shadow_mirror_scorecard`",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"OPE-266` / `BIG-PAR-092",
				"GET /debug/status",
				"GET /v2/control-center",
				"distributed_diagnostics.live_shadow_mirror_scorecard",
			},
		},
		{
			path: "docs/reports/live-shadow-index.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/live-shadow-runs/20260313T085655Z/README.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"OPE-266` / `BIG-PAR-092",
				"live-shadow-comparison-follow-up-digest.md",
				"OPE-254` / `BIG-PAR-088",
				"rollback-safeguard-follow-up-digest.md",
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
