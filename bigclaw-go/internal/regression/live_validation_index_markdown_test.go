package regression

import (
	"strings"
	"testing"
)

func TestLiveValidationIndexMarkdownSurfaceStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/reports/live-validation-index.md")

	want := []string{
		"# Live Validation Index",
		"`20260316T140138Z`",
		"`docs/reports/live-validation-runs/20260316T140138Z`",
		"### shared-queue companion",
		"`docs/reports/shared-queue-companion-summary.json`",
		"`docs/reports/multi-node-shared-queue-report.json`",
		"`docs/reports/validation-bundle-continuation-scorecard.json`",
		"`docs/reports/validation-bundle-continuation-policy-gate.json`",
		"`docs/reports/validation-bundle-continuation-digest.md`",
	}

	for _, needle := range want {
		if !strings.Contains(contents, needle) {
			t.Fatalf("live-validation-index.md missing %q", needle)
		}
	}
}
