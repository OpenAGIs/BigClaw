package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1567ScriptsOpsStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	opsDir := filepath.Join(rootRepo, "scripts", "ops")

	entries, err := os.ReadDir(opsDir)
	if err != nil {
		t.Fatalf("read scripts/ops: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected scripts/ops to remain Python-free, found %s", entry.Name())
		}
	}
}

func TestBIGGO1567ScriptsOpsReplacementReport(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1567-scripts-ops-deletion-tranche.md")

	for _, needle := range []string{
		"BIG-GO-1567",
		"Repository-wide Python file count: `0`.",
		"`scripts/ops/bigclawctl` is a Bash compatibility shim that dispatches to `go run ./cmd/bigclawctl`.",
		"`scripts/ops/bigclaw-issue` maps to `bash scripts/ops/bigclawctl issue`.",
		"`scripts/ops/bigclaw-panel` maps to `bash scripts/ops/bigclawctl panel`.",
		"`scripts/ops/bigclaw-symphony` maps to `bash scripts/ops/bigclawctl symphony`.",
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`.",
		"retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`.",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`.",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`.",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`.",
		"`find . -name '*.py' | wc -l`",
		"`find scripts/ops -maxdepth 1 -type f | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("report missing substring %q", needle)
		}
	}
}

func TestBIGGO1567CutoverIssuePackCarriesScriptsOpsReplacements(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	contents := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-issue-pack.md")

	for _, needle := range []string{
		"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`",
		"retired `scripts/ops/bigclaw_refill_queue.py`; use `bigclawctl refill`",
		"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
	} {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-mainline-cutover-issue-pack.md missing scripts/ops replacement guidance %q", needle)
		}
	}
}
