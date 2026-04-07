package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1362RepoModuleRemovalSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredModules := map[string]string{
		"src/bigclaw/repo_board.py":      "bigclaw-go/internal/repo/board.go",
		"src/bigclaw/repo_commits.py":    "bigclaw-go/internal/repo/commits.go",
		"src/bigclaw/repo_gateway.py":    "bigclaw-go/internal/repo/gateway.go",
		"src/bigclaw/repo_governance.py": "bigclaw-go/internal/repo/governance.go",
		"src/bigclaw/repo_links.py":      "bigclaw-go/internal/repo/links.go",
		"src/bigclaw/repo_plane.py":      "bigclaw-go/internal/repo/plane.go",
		"src/bigclaw/repo_registry.py":   "bigclaw-go/internal/repo/registry.go",
		"src/bigclaw/repo_triage.py":     "bigclaw-go/internal/repo/triage.go",
	}

	for retiredPath, replacementPath := range retiredModules {
		if _, err := os.Stat(filepath.Join(rootRepo, retiredPath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python module to stay absent: %s", retiredPath)
		}
		if _, err := os.Stat(filepath.Join(rootRepo, replacementPath)); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", replacementPath, err)
		}
	}

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1362-repo-module-removal-sweep.md")
	for _, needle := range []string{
		"BIG-GO-1362",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/repo_board.py` -> `bigclaw-go/internal/repo/board.go`",
		"`src/bigclaw/repo_commits.py` -> `bigclaw-go/internal/repo/commits.go`",
		"`src/bigclaw/repo_gateway.py` -> `bigclaw-go/internal/repo/gateway.go`",
		"`src/bigclaw/repo_governance.py` -> `bigclaw-go/internal/repo/governance.go`",
		"`src/bigclaw/repo_links.py` -> `bigclaw-go/internal/repo/links.go`",
		"`src/bigclaw/repo_plane.py` -> `bigclaw-go/internal/repo/plane.go`",
		"`src/bigclaw/repo_registry.py` -> `bigclaw-go/internal/repo/registry.go`",
		"`src/bigclaw/repo_triage.py` -> `bigclaw-go/internal/repo/triage.go`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1362RepoModuleRemovalSweep'`",
		"`find . -name '*.py' | wc -l`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
