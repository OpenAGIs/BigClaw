package regression

import (
	"strings"
	"testing"
)

func TestMigrationEntryPointsStayGoOnly(t *testing.T) {
	root := repoRoot(t)

	references := []struct {
		path           string
		requiredSnips  []string
		forbiddenSnips []string
	}{
		{
			path: "../README.md",
			requiredSnips: []string{
				"go run ./bigclaw-go/cmd/bigclawctl automation migration shadow-compare ...",
				"go run ./bigclaw-go/cmd/bigclawctl automation migration shadow-matrix ...",
				"go run ./bigclaw-go/cmd/bigclawctl automation migration live-shadow-scorecard ...",
				"go run ./bigclaw-go/cmd/bigclawctl automation migration export-live-shadow-bundle ...",
			},
			forbiddenSnips: []string{
				"bigclaw-go/scripts/migration/shadow_compare.py",
				"bigclaw-go/scripts/migration/shadow_matrix.py",
				"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
				"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
			},
		},
		{
			path: "../workflow.md",
			requiredSnips: []string{
				"go run ./bigclaw-go/cmd/bigclawctl automation migration <shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle>",
			},
			forbiddenSnips: []string{
				"bigclaw-go/scripts/migration/shadow_compare.py",
				"bigclaw-go/scripts/migration/shadow_matrix.py",
				"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
				"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
			},
		},
		{
			path: "../.github/workflows/ci.yml",
			requiredSnips: []string{
				"TestMigrationEntryPointsStayGoOnly",
				"TestAutomationMigrationHelpSurface",
			},
			forbiddenSnips: []string{
				"bigclaw-go/scripts/migration/shadow_compare.py",
				"bigclaw-go/scripts/migration/shadow_matrix.py",
				"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
				"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
			},
		},
		{
			path: "../.githooks/post-commit",
			requiredSnips: []string{
				"stale deleted Python migration helper references detected",
			},
			forbiddenSnips: []string{
				"shadow_compare.py",
				"shadow_matrix.py",
				"live_shadow_scorecard.py",
				"export_live_shadow_bundle.py",
			},
		},
		{
			path: "../.githooks/post-rewrite",
			requiredSnips: []string{
				"stale deleted Python migration helper references detected",
			},
			forbiddenSnips: []string{
				"shadow_compare.py",
				"shadow_matrix.py",
				"live_shadow_scorecard.py",
				"export_live_shadow_bundle.py",
			},
		},
		{
			path: "README.md",
			requiredSnips: []string{
				"go run ./cmd/bigclawctl automation migration shadow-compare ...",
				"go run ./cmd/bigclawctl automation migration shadow-matrix ...",
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle ...",
			},
			forbiddenSnips: []string{
				"bigclaw-go/scripts/migration/shadow_compare.py",
				"bigclaw-go/scripts/migration/shadow_matrix.py",
				"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
				"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
			},
		},
		{
			path: "docs/go-cli-script-migration.md",
			requiredSnips: []string{
				"`bigclaw-go/scripts/migration/`",
				"`bigclawctl automation migration ...`",
			},
			forbiddenSnips: []string{
				"migration scorecards/bundle exporters",
				"Remaining Python generators still need native replacements before they can be removed.",
			},
		},
	}

	for _, tc := range references {
		contents := readRepoFile(t, root, tc.path)
		for _, want := range tc.requiredSnips {
			if !strings.Contains(contents, want) {
				t.Fatalf("%s missing substring %q", tc.path, want)
			}
		}
		for _, forbid := range tc.forbiddenSnips {
			if strings.Contains(contents, forbid) {
				t.Fatalf("%s unexpectedly still contains %q", tc.path, forbid)
			}
		}
	}
}
