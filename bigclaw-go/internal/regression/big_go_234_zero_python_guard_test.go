package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO234RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO234ScriptAndCLIHelperSurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	surfaces := []string{
		"scripts",
		"bigclaw-go/scripts",
		"bigclaw-go/cmd",
		"bigclaw-go/docs",
	}

	for _, relativeDir := range surfaces {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected scripts/CLI-helper surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO234GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/go-cli-script-migration-plan.md",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO234LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-234-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-234",
		"Residual scripts Python sweep S",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/cmd`: `0` Python files",
		"`bigclaw-go/docs`: `0` Python files",
		"`docs/go-cli-script-migration-plan.md`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts bigclaw-go/scripts bigclaw-go/cmd bigclaw-go/docs -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO234(RepositoryHasNoPythonFiles|ScriptAndCLIHelperSurfacesStayPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
