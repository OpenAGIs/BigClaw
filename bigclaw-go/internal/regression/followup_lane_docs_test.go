package regression

import (
	"strings"
	"testing"
)

func TestFollowUpLaneDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/production-corpus-migration-coverage-digest.md",
			substrings: []string{
				"OPE-268` / `BIG-PAR-079",
				"fixture-backed evidence only",
				"no real production issue/task corpus coverage",
			},
		},
		{
			path: "docs/reports/subscriber-takeover-executability-follow-up-digest.md",
			substrings: []string{
				"OPE-269` / `BIG-PAR-080",
				"live-multi-node-subscriber-takeover-report.json",
				"shared durable SQLite scaffold exists but broker-backed ownership does not",
			},
		},
		{
			path: "docs/reports/validation-bundle-continuation-digest.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-policy-gate.json",
				"rolling continuation scorecard",
				"continuation across future validation bundles remains manual",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"OPE-268` / `BIG-PAR-079",
				"OPE-269` / `BIG-PAR-080",
				"OPE-271` / `BIG-PAR-082",
				"production-corpus-migration-coverage-digest.md",
				"subscriber-takeover-executability-follow-up-digest.md",
				"validation-bundle-continuation-digest.md",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"OPE-268` / `BIG-PAR-079",
				"OPE-269` / `BIG-PAR-080",
				"OPE-271` / `BIG-PAR-082",
				"production-corpus-migration-coverage-digest.md",
				"subscriber-takeover-executability-follow-up-digest.md",
				"validation-bundle-continuation-digest.md",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"OPE-268` / `BIG-PAR-079",
				"OPE-269` / `BIG-PAR-080",
				"OPE-271` / `BIG-PAR-082",
				"production-corpus-migration-coverage-digest.md",
				"subscriber-takeover-executability-follow-up-digest.md",
				"validation-bundle-continuation-digest.md",
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
