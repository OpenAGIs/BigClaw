package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO124TargetResidualPythonPathsAbsent(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	targets := []string{
		"bigclaw-go/scripts/migration/export_live_shadow_bundle",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}

	for _, relativePath := range targets {
		_, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected retired residual migration path to stay absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO124GoReplacementPathsRemainAvailable(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	replacements := []string{
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/docs/migration-shadow.md",
		"bigclaw-go/docs/reports/migration-readiness-report.md",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/live-shadow-summary.json",
		"bigclaw-go/docs/reports/live-shadow-index.json",
		"bigclaw-go/docs/reports/live-shadow-index.md",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO124LaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-124-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-124",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py`",
		"`bigclaw-go/scripts/migration/live_shadow_scorecard.py`",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
		"`bigclaw-go/scripts/migration/shadow_matrix.py`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare`",
		"`go run ./cmd/bigclawctl automation migration shadow-matrix`",
		"`go run ./cmd/bigclawctl automation migration live-shadow-scorecard`",
		"`go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`",
		"`rg -n \"python3 scripts/migration|scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard)\\.py|scripts/migration/export_live_shadow_bundle\" bigclaw-go/docs bigclaw-go/internal/regression bigclaw-go/docs/reports`",
		"`test ! -e bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(LiveShadowScorecardBundleStaysAligned|LiveShadowBundleSummaryAndIndexStayAligned|BIGGO124",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
