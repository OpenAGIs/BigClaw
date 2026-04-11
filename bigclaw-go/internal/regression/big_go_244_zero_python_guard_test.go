package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO244RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO244ResidualScriptAndCLIHelperSurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"scripts",
		"scripts/ops",
		"bigclaw-go/scripts",
		"bigclaw-go/cmd",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual script or CLI-helper surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO244SupportedWrapperAndCLIPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredPaths := []string{
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"docs/local-tracker-automation.md",
		"docs/go-cli-script-migration-plan.md",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native wrapper or CLI-helper path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO244WrapperInventoryMatchesContract(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	rootScripts, err := os.ReadDir(filepath.Join(rootRepo, "scripts"))
	if err != nil {
		t.Fatalf("read scripts directory: %v", err)
	}
	expectedRoot := map[string]bool{
		"dev_bootstrap.sh": true,
		"ops":              true,
	}
	if len(rootScripts) != len(expectedRoot) {
		t.Fatalf("expected scripts directory to contain exactly %d entries, found %d", len(expectedRoot), len(rootScripts))
	}
	for _, entry := range rootScripts {
		if !expectedRoot[entry.Name()] {
			t.Fatalf("unexpected scripts entry: %s", entry.Name())
		}
	}

	opsEntries, err := os.ReadDir(filepath.Join(rootRepo, "scripts", "ops"))
	if err != nil {
		t.Fatalf("read scripts/ops directory: %v", err)
	}
	expectedOps := map[string]string{
		"bigclawctl":       "go run ./cmd/bigclawctl",
		"bigclaw-issue":    "exec bash \"$script_dir/bigclawctl\" issue \"$@\"",
		"bigclaw-panel":    "exec bash \"$script_dir/bigclawctl\" panel \"$@\"",
		"bigclaw-symphony": "exec bash \"$script_dir/bigclawctl\" symphony \"$@\"",
	}
	if len(opsEntries) != len(expectedOps) {
		t.Fatalf("expected scripts/ops to contain exactly %d helper files, found %d", len(expectedOps), len(opsEntries))
	}
	for _, entry := range opsEntries {
		if entry.IsDir() {
			t.Fatalf("expected flat scripts/ops helper inventory, found directory %s", entry.Name())
		}
		expectedSnippet, ok := expectedOps[entry.Name()]
		if !ok {
			t.Fatalf("unexpected scripts/ops helper: %s", entry.Name())
		}

		contents := readRepoFile(t, rootRepo, filepath.ToSlash(filepath.Join("scripts", "ops", entry.Name())))
		if strings.Contains(contents, "python") || strings.Contains(contents, ".py") {
			t.Fatalf("expected wrapper %s to stay shell/Go-only, found Python reference", entry.Name())
		}
		if !strings.Contains(contents, expectedSnippet) {
			t.Fatalf("expected wrapper %s to contain %q", entry.Name(), expectedSnippet)
		}
	}

	migrationPlan := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")
	for _, needle := range []string{
		"`scripts/ops/bigclaw-symphony` -> `bigclawctl symphony`",
		"`scripts/ops/bigclaw-issue` -> `bigclawctl issue`",
		"`scripts/ops/bigclaw-panel` -> `bigclawctl panel`",
		"All other root or `scripts/ops` wrappers should stay retired",
	} {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing wrapper contract guidance %q", needle)
		}
	}

	localTracker := readRepoFile(t, rootRepo, "docs/local-tracker-automation.md")
	if !strings.Contains(localTracker, "bash scripts/ops/bigclawctl local-issues ensure") {
		t.Fatalf("docs/local-tracker-automation.md should point automation callers at bash scripts/ops/bigclawctl")
	}
}

func TestBIGGO244LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-244-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-244",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/cmd`: `0` Python files",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`docs/local-tracker-automation.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts scripts/ops bigclaw-go/scripts bigclaw-go/cmd -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO244(RepositoryHasNoPythonFiles|ResidualScriptAndCLIHelperSurfacesStayPythonFree|SupportedWrapperAndCLIPathsRemainAvailable|WrapperInventoryMatchesContract|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
