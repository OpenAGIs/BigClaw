package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1588ScriptsOpsBucketStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "scripts", "ops"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected scripts/ops bucket to remain Python-free, found %v", pythonFiles)
	}
}

func TestBIGGO1588RetiredScriptsOpsPythonPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired scripts/ops Python helper to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1588GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"docs/go-cli-script-migration-plan.md",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1588LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1588-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1588",
		"Repository-wide Python file count: `0`.",
		"`scripts/ops`: `0` Python files",
		"`scripts/ops/bigclaw_github_sync.py`",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_validate.py`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`find scripts/ops -maxdepth 1 -type f -name '*.py' | sort`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1588(ScriptsOpsBucketStaysPythonFree|RetiredScriptsOpsPythonPathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
