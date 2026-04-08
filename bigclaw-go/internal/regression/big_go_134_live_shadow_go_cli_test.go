package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO134RetiresLiveShadowPythonWrapper(t *testing.T) {
	goRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	if _, err := os.Stat(filepath.Join(repoRoot, "bigclaw-go", "scripts", "migration", "export_live_shadow_bundle")); !os.IsNotExist(err) {
		t.Fatalf("expected live shadow Python wrapper to stay removed, err=%v", err)
	}
}

func TestBIGGO134LiveShadowDocsUseGoCLI(t *testing.T) {
	goRoot := repoRoot(t)

	for _, tc := range []struct {
		path     string
		required []string
		blocked  []string
	}{
		{
			path: "docs/migration-shadow.md",
			required: []string{
				"go run ./cmd/bigclawctl automation migration shadow-compare",
				"go run ./cmd/bigclawctl automation migration shadow-matrix",
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
			},
			blocked: []string{
				"python3 scripts/migration/shadow_compare.py",
				"python3 scripts/migration/shadow_matrix.py",
				"python3 scripts/migration/live_shadow_scorecard.py",
				"python3 scripts/migration/export_live_shadow_bundle",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			required: []string{
				"go run ./cmd/bigclawctl automation migration shadow-compare",
				"go run ./cmd/bigclawctl automation migration shadow-matrix",
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
			},
			blocked: []string{
				"scripts/migration/shadow_compare.py",
				"scripts/migration/shadow_matrix.py",
				"scripts/migration/live_shadow_scorecard.py",
				"scripts/migration/export_live_shadow_bundle",
			},
		},
		{
			path: "docs/reports/live-shadow-index.md",
			required: []string{
				"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
				"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
			},
			blocked: []string{
				"python3 scripts/migration/live_shadow_scorecard.py",
				"python3 scripts/migration/export_live_shadow_bundle",
			},
		},
	} {
		body := readRepoFile(t, goRoot, tc.path)
		for _, needle := range tc.required {
			if !strings.Contains(body, needle) {
				t.Fatalf("%s missing %q", tc.path, needle)
			}
		}
		for _, needle := range tc.blocked {
			if strings.Contains(body, needle) {
				t.Fatalf("%s still contains retired Python reference %q", tc.path, needle)
			}
		}
	}
}

func TestBIGGO134LaneReportCapturesLiveShadowCLIReplacement(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-134-live-shadow-cli-sweep.md")

	for _, needle := range []string{
		"BIG-GO-134",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare`",
		"`go run ./cmd/bigclawctl automation migration shadow-matrix`",
		"`go run ./cmd/bigclawctl automation migration live-shadow-scorecard`",
		"`go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`",
		"`go test ./internal/regression -run 'TestBIGGO134|TestLiveShadowScorecardBundleStaysAligned|TestLiveShadowBundleSummaryAndIndexStayAligned'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing %q", needle)
		}
	}
}
