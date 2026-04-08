package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1577TargetResidualPythonPathsAbsent(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	targets := []string{
		"src/bigclaw/cost_control.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/repo_board.py",
		"src/bigclaw/roadmap.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
		"tests/test_design_system.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_pilot.py",
		"tests/test_repo_triage.py",
		"tests/test_subscriber_takeover_harness.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
	}

	for _, relativePath := range targets {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired residual Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1577GoReplacementPathsRemainAvailable(t *testing.T) {
	goRoot := repoRoot(t)
	repo := filepath.Clean(filepath.Join(goRoot, ".."))

	replacements := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1577LaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-1577-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1577",
		"`src/bigclaw/cost_control.py`",
		"`src/bigclaw/mapping.py`",
		"`src/bigclaw/repo_board.py`",
		"`src/bigclaw/roadmap.py`",
		"`src/bigclaw/workspace_bootstrap_cli.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`bigclaw-go/scripts/e2e/export_validation_bundle_test.py`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/intake/mapping.go`",
		"`bigclaw-go/internal/repo/board.go`",
		"`bigclaw-go/internal/repo/triage.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle`",
		"`bigclawctl automation migration export-live-shadow-bundle`",
		"`find src/bigclaw tests scripts/ops bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1577",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
