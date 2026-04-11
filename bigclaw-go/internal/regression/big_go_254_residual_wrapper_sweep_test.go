package regression

import (
	"strings"
	"testing"
)

func TestBIGGO254RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO254CompiledRootLauncherReplacesGoRunWrapper(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	launcher := readRepoFile(t, rootRepo, "scripts/ops/bigclawctl")
	requiredSnippets := []string{
		"BIGCLAWCTL_BIN_DIR",
		"XDG_CACHE_HOME",
		"go build -o \"$tmp_bin\" ./cmd/bigclawctl",
		"exec \"$bin_path\"",
	}
	for _, needle := range requiredSnippets {
		if !strings.Contains(launcher, needle) {
			t.Fatalf("scripts/ops/bigclawctl missing compiled launcher snippet %q", needle)
		}
	}
	if strings.Contains(launcher, "go run ./cmd/bigclawctl") {
		t.Fatalf("scripts/ops/bigclawctl should not shell into go run anymore")
	}
}

func TestBIGGO254MigrationPlanCapturesCompiledLauncherState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	plan := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")
	for _, needle := range []string{
		"Bash ops aliases only proxy into the cached compiled launcher at `scripts/ops/bigclawctl`.",
		"Keep `scripts/ops/bigclawctl` on the cached compiled-binary path and do not",
		"`scripts/ops/bigclawctl` now caches a compiled binary",
	} {
		if !strings.Contains(plan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing compiled launcher guidance %q", needle)
		}
	}
}

func TestBIGGO254LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-254-python-asset-sweep.md")
	for _, needle := range []string{
		"BIG-GO-254",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"Explicit remaining Python asset list: none.",
		"`scripts/ops/bigclawctl` now builds and reuses a cached `bigclawctl` binary",
		"`BIGCLAWCTL_BIN_DIR`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`bash scripts/ops/bigclawctl --help`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'Test(BIGGO254|RootScriptResidualSweep|RunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|RunGitHubSyncHelpPrintsUsageAndExitsZero)'`",
		"Residual risk: the repository already started this lane at zero physical Python",
		"BIG-GO-254 hardens the compiled-wrapper path instead of lowering the",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
