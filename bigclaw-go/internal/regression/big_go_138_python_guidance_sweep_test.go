package regression

import (
	"strings"
	"testing"
)

func TestBIGGO138MigrationGuidancePrefersGoAutomation(t *testing.T) {
	goRoot := repoRoot(t)

	docPaths := []string{
		"docs/migration-shadow.md",
		"docs/reports/migration-readiness-report.md",
		"docs/reports/live-shadow-index.md",
		"docs/reports/live-shadow-summary.json",
		"docs/reports/live-shadow-index.json",
		"docs/reports/live-shadow-runs/20260313T085655Z/README.md",
		"docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
	}

	required := []string{
		"go run ./cmd/bigclawctl automation migration shadow-compare",
		"go run ./cmd/bigclawctl automation migration shadow-matrix",
		"go run ./cmd/bigclawctl automation migration live-shadow-scorecard",
		"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
	}

	disallowed := []string{
		"python3 scripts/migration/shadow_compare.py",
		"python3 scripts/migration/shadow_matrix.py",
		"python3 scripts/migration/live_shadow_scorecard.py",
		"python3 scripts/migration/export_live_shadow_bundle",
	}

	for _, relativePath := range docPaths {
		body := readRepoFile(t, goRoot, relativePath)
		for _, needle := range disallowed {
			if strings.Contains(body, needle) {
				t.Fatalf("%s should not reference retired Python migration guidance %q", relativePath, needle)
			}
		}
	}

	shadowGuide := readRepoFile(t, goRoot, "docs/migration-shadow.md")
	for _, needle := range required {
		if !strings.Contains(shadowGuide, needle) {
			t.Fatalf("docs/migration-shadow.md missing Go migration guidance %q", needle)
		}
	}

	readiness := readRepoFile(t, goRoot, "docs/reports/migration-readiness-report.md")
	for _, needle := range required {
		if !strings.Contains(readiness, needle) {
			t.Fatalf("docs/reports/migration-readiness-report.md missing Go migration guidance %q", needle)
		}
	}
}

func TestBIGGO138LaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-138-python-guidance-sweep.md")

	for _, needle := range []string{
		"BIG-GO-138",
		"`bigclaw-go/docs/migration-shadow.md`",
		"`bigclaw-go/docs/reports/migration-readiness-report.md`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/live-shadow-summary.json`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare`",
		"`go run ./cmd/bigclawctl automation migration shadow-matrix`",
		"`go run ./cmd/bigclawctl automation migration live-shadow-scorecard`",
		"`go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`",
		"`python3 scripts/migration/shadow_compare.py`",
		"`python3 scripts/migration/shadow_matrix.py`",
		"`python3 scripts/migration/live_shadow_scorecard.py`",
		"`python3 scripts/migration/export_live_shadow_bundle`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO138MigrationGuidancePrefersGoAutomation|BIGGO138LaneReportCapturesSweepState|LiveShadowBundleSummaryAndIndexStayAligned)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
