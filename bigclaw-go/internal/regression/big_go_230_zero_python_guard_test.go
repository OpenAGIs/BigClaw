package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO230RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO230PracticalGoOnlySurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"scripts",
		"bigclaw-go/cmd",
		"docs",
		"bigclaw-go/docs/reports",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected practical Go-only surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO230GoNativeEntryPointsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"Makefile",
		"README.md",
		"docs/local-tracker-automation.md",
		"docs/go-cli-script-migration-plan.md",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO230LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-230-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-230",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`bigclaw-go/cmd`: `0` Python files",
		"`docs`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`Makefile`",
		"`README.md`",
		"`docs/local-tracker-automation.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts bigclaw-go/cmd docs bigclaw-go/docs/reports -maxdepth 2 -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO230(RepositoryHasNoPythonFiles|PracticalGoOnlySurfacesStayPythonFree|GoNativeEntryPointsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
