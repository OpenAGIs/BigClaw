package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO115TargetResidualPythonPathsAbsent(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	targets := []string{
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle",
	}

	for _, relativePath := range targets {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired residual Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO115GoReplacementPathsRemainAvailable(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	replacements := []string{
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/live-shadow-summary.json",
		"bigclaw-go/docs/reports/live-shadow-index.json",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO115LaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-115-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-115",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`go run ./cmd/bigclawctl automation migration live-shadow-scorecard`",
		"`go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`",
		"`bigclaw-go/docs/reports/live-shadow-summary.json`",
		"`bigclaw-go/docs/reports/live-shadow-index.json`",
		"`bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO115",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
