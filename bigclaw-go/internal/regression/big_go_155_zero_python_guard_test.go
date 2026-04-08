package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO155RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO155ResidualToolingSurfaceStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"scripts",
		".githooks",
		"bigclaw-go/cmd/bigclawctl",
		"bigclaw-go/internal/githubsync",
		"bigclaw-go/internal/refill",
		"bigclaw-go/internal/bootstrap",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual tooling surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}

	retiredPaths := []string{
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tooling Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO155GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".pre-commit-config.yaml",
		"Makefile",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		".githooks/post-commit",
		".githooks/post-rewrite",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	preCommitConfig := readRepoFile(t, rootRepo, ".pre-commit-config.yaml")
	for _, forbidden := range []string{"ruff-pre-commit", "ruff-check", "ruff-format"} {
		if strings.Contains(preCommitConfig, forbidden) {
			t.Fatalf("expected root pre-commit config to stay free of residual Python hook %q", forbidden)
		}
	}
}

func TestBIGGO155LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-155-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-155",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused tooling/build-helper/dev-utility physical Python file count before lane changes: `0`",
		"Focused tooling/build-helper/dev-utility physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Removed tooling-only Python hooks in this lane: `[\"ruff-pre-commit\", \"ruff-check\", \"ruff-format\"]`",
		"Focused tooling ledger: `[]`",
		"`.pre-commit-config.yaml`: removed residual Python-only `ruff-pre-commit` hooks",
		"`scripts`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`bigclaw-go/cmd/bigclawctl`: `0` Python files",
		"`bigclaw-go/internal/githubsync`: `0` Python files",
		"`bigclaw-go/internal/refill`: `0` Python files",
		"`bigclaw-go/internal/bootstrap`: `0` Python files",
		"`Makefile`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`.githooks/post-commit`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts .githooks bigclaw-go/cmd/bigclawctl bigclaw-go/internal/githubsync bigclaw-go/internal/refill bigclaw-go/internal/bootstrap -type f -name '*.py' 2>/dev/null | sort`",
		"`rg -n 'ruff-pre-commit|ruff-check|ruff-format' .pre-commit-config.yaml`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO155",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
