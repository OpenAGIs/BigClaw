package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1474OpsDirectoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "scripts", "ops"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected scripts/ops to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1474DeletedPythonWrapperPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	deletedWrappers := []string{
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relativePath := range deletedWrappers {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python wrapper to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1474ActiveGoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/refill/queue.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected active Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1474DocsAndLaneReportPinDeletedWrapperState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	migrationPlan := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")
	for _, needle := range []string{
		"Deleted Python-suffixed ops wrappers are historical removals only.",
		"`scripts/ops/bigclaw_github_sync.py` -> deleted; Go ownership: `bigclawctl github-sync` via `bigclaw-go/internal/githubsync/*`",
		"`scripts/ops/bigclaw_refill_queue.py` -> deleted; Go ownership: `bigclawctl refill` via `bigclaw-go/internal/refill/*`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py` -> deleted; Go ownership: `bigclawctl workspace bootstrap` via `bigclaw-go/internal/bootstrap/*`",
		"`scripts/ops/symphony_workspace_bootstrap.py` -> deleted; Go ownership: `bigclawctl workspace bootstrap` via `bigclaw-go/internal/bootstrap/*`",
		"`scripts/ops/symphony_workspace_validate.py` -> deleted; Go ownership: `bigclawctl workspace validate` via `bigclaw-go/internal/bootstrap/*`",
	} {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing substring %q", needle)
		}
	}

	cutoverPack := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-issue-pack.md")
	for _, needle := range []string{
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bash scripts/ops/bigclawctl github-sync`",
		"retired `scripts/ops/bigclaw_refill_queue.py`; use `bash scripts/ops/bigclawctl refill`",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
	} {
		if !strings.Contains(cutoverPack, needle) {
			t.Fatalf("docs/go-mainline-cutover-issue-pack.md missing substring %q", needle)
		}
	}

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1474-ops-wrapper-sweep.md")
	for _, needle := range []string{
		"BIG-GO-1474",
		"`scripts/ops`: `0` Python files",
		"`scripts/ops/bigclaw_github_sync.py`",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`scripts/ops/bigclaw_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_bootstrap.py`",
		"`scripts/ops/symphony_workspace_validate.py`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`find scripts/ops -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1474",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
