package regression

import (
	"strings"
	"testing"
)

func TestMigrationFollowUpIndexDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/migration-shadow.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining migration-shadow and parallel-hardening follow-up digests",
				"docs/reports/parallel-validation-matrix.md",
				"docs/reports/production-corpus-migration-coverage-digest.md",
			},
		},
		{
			path: "docs/migration.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining migration, rollback, and parallel-hardening follow-up digests",
				"docs/reports/parallel-validation-matrix.md",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"docs/reports/parallel-follow-up-index.md",
				"remaining migration-shadow, rollback, and corpus-coverage caveats",
				"docs/reports/parallel-validation-matrix.md",
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
