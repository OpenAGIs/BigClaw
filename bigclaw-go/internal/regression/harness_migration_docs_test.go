package regression

import (
	"strings"
	"testing"
)

func TestHarnessMigrationPlanDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)

	planContents := readRepoFile(t, repoRoot, "docs/reports/test-harness-migration-plan.md")
	requiredPlanSubstrings := []string{
		"three Go-native harness lanes",
		"`go test` package tests",
		"`internal/regression`",
		"`scripts/e2e/run_all.sh`",
		"Branch naming: `BIG-GO-903-test-harness-migration`",
		"python3 -m pytest tests/test_harness_migration_plan.py",
		"cd bigclaw-go && go test ./internal/regression",
	}
	for _, needle := range requiredPlanSubstrings {
		if !strings.Contains(planContents, needle) {
			t.Fatalf("docs/reports/test-harness-migration-plan.md missing substring %q", needle)
		}
	}

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/migration.md",
			substrings: []string{
				"docs/reports/test-harness-migration-plan.md",
				"pytest-to-Go harness split",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"docs/reports/test-harness-migration-plan.md",
				"pytest-to-Go harness split",
			},
		},
		{
			path: "docs/reports/migration-plan-review-notes.md",
			substrings: []string{
				"docs/reports/test-harness-migration-plan.md",
				"executable pytest-to-Go harness split",
			},
		},
		{
			path: "docs/reports/parallel-validation-matrix.md",
			substrings: []string{
				"docs/reports/test-harness-migration-plan.md",
				"canonical unit/golden/integration",
			},
		},
		{
			path: "docs/e2e-validation.md",
			substrings: []string{
				"docs/reports/test-harness-migration-plan.md",
				"broader pytest-to-Go migration split",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"bigclaw-go/docs/reports/test-harness-migration-plan.md",
				"legacy pytest coverage into Go package tests",
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
