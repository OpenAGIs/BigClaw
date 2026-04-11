package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO214RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO214RetiredRootWrapperAliasesRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredWrappers := []string{
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
	}

	for _, relativePath := range retiredWrappers {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired root wrapper alias to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO214CanonicalRootEntrypointsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/internal/refill/queue_markdown.go",
		"docs/go-cli-script-migration-plan.md",
		"docs/parallel-refill-queue.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected canonical root entrypoint path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO214LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-214-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-214",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`docs/go-cli-script-migration-plan.md`",
		"`docs/parallel-refill-queue.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO214(RepositoryHasNoPythonFiles|RetiredRootWrapperAliasesRemainAbsent|CanonicalRootEntrypointsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
