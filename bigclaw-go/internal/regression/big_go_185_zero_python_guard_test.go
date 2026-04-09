package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO185ResidualPythonToolingPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"setup.py",
		"pyproject.toml",
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relativePath := range retiredPaths {
		_, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected retired Python tooling path to stay absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO185ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/go.mod",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO185LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-185-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-185",
		"`setup.py`",
		"`pyproject.toml`",
		"`scripts/create_issues.py`",
		"`scripts/dev_smoke.py`",
		"`scripts/ops/bigclaw_github_sync.py`",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_validate.py`",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/cmd/bigclawctl`: `0` Python files and active Go CLI command coverage",
		"`bigclaw-go/internal/{githubsync,refill,bootstrap}`: `0` Python files",
		"`bigclaw-go/go.mod`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts bigclaw-go -type f \\( -name '*.py' -o -path 'bigclaw-go/go.mod' -o -path 'bigclaw-go/cmd/bigclawctl/main.go' -o -path 'bigclaw-go/cmd/bigclawctl/migration_commands.go' -o -path 'bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go' -o -path 'bigclaw-go/internal/githubsync/sync.go' -o -path 'bigclaw-go/internal/refill/queue.go' -o -path 'bigclaw-go/internal/bootstrap/bootstrap.go' -o -path 'scripts/ops/bigclawctl' -o -path 'scripts/dev_bootstrap.sh' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO185(ResidualPythonToolingPathsStayAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
