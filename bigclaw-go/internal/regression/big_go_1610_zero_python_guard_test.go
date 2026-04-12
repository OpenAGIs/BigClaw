package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO1610RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectFinalSweepPythonLikeFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1610FinalSweepFocusDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	focusDirs := []string{
		".githooks",
		".github",
		".symphony",
		"docs",
		"reports",
		"scripts",
		"scripts/ops",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/regression",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range focusDirs {
		pythonFiles := collectFinalSweepPythonLikeFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected final sweep focus directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1610HistoricalResidualPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	historicalResidualPaths := []string{
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

	for _, relativePath := range historicalResidualPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected historical residual Python path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1610GoNativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/queue/memory_queue.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1610LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1610-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1610",
		"repo-wide final Python asset sweep",
		"Repository-wide Python file count: `0`.",
		"Tracked `*.py` files remaining after the final sweep: `none`.",
		"`.githooks`: `0` Python files",
		"`.github`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs`: `0` Python files",
		"`reports`: `0` Python files",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/regression`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Historical residual `*.py` paths already retired before this lane:",
		"`src/bigclaw/cost_control.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/repo/board.go`",
		"`bigclaw-go/internal/repo/triage.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/internal/queue/queue.go`",
		"`bigclaw-go/internal/queue/memory_queue.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .githooks .github .symphony docs reports scripts scripts/ops bigclaw-go/docs/reports bigclaw-go/internal/regression bigclaw-go/scripts -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1610(RepositoryHasNoPythonFiles|FinalSweepFocusDirectoriesStayPythonFree|HistoricalResidualPathsRemainAbsent|GoNativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"No tracked Python residue remains, so no in-branch delete step is still pending.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectFinalSweepPythonLikeFiles(t *testing.T, root string) []string {
	t.Helper()

	pythonLikeExtensions := map[string]struct{}{
		".ipynb": {},
		".py":    {},
		".pyi":   {},
		".pyw":   {},
	}

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := pythonLikeExtensions[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}
