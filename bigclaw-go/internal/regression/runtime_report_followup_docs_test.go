package regression

import (
	"strings"
	"testing"
)

func TestRuntimeReportFollowUpDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/live-validation-index.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-digest.md",
				"validation-bundle-continuation-scorecard.json",
				"validation-bundle-continuation-policy-gate.json",
			},
		},
		{
			path: "docs/reports/multi-node-coordination-report.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-digest.md",
				"cross-process-coordination-boundary-digest.md",
			},
		},
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"OPE-269` / `BIG-PAR-080",
				"subscriber-takeover-executability-follow-up-digest.md",
				"live-multi-node-subscriber-takeover-report.json",
			},
		},
		{
			path: "docs/reports/subscriber-takeover-executability-follow-up-digest.md",
			substrings: []string{
				"OPE-269` / `BIG-PAR-080",
				"live-multi-node-subscriber-takeover-report.json",
			},
		},
		{
			path: "docs/reports/validation-bundle-continuation-digest.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-policy-gate.json",
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
