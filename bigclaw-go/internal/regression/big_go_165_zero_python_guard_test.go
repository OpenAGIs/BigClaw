package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO165ToolingPythonPathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredToolingPaths := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
		"setup.py",
		"pyproject.toml",
	}

	for _, relativePath := range retiredToolingPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired tooling path to stay absent: %s", relativePath)
		}
	}

	for _, relativeDir := range []string{"scripts", "scripts/ops"} {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected tooling directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO165GoToolingReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native tooling replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO165LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-165-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-165",
		"Residual tooling Python sweep L",
		"Tracked `scripts/*.py`: `none`",
		"Tracked `scripts/ops/*.py`: `none`",
		"Tracked `setup.py`: `none`",
		"Tracked `pyproject.toml`: `none`",
		"Physical repository matches for `*.py`, `setup.py`, or `pyproject.toml`: `none`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`git ls-files 'scripts/*.py' 'scripts/ops/*.py' 'setup.py' 'pyproject.toml' | sort`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name 'setup.py' -o -name 'pyproject.toml' \\) -print | sed 's#^./##' | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO165(ToolingPythonPathsRemainAbsent|GoToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
