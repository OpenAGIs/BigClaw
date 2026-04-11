package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1562RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1562WorkflowOrchestrationTrancheBStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/runtime.py",
		"src/bigclaw/scheduler.py",
		"src/bigclaw/orchestration.py",
		"src/bigclaw/workflow.py",
		"src/bigclaw/queue.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-B Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1562GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/orchestrator/loop.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/control/controller.go",
		"docs/go-mainline-cutover-issue-pack.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1562LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1562-src-bigclaw-tranche-b.md")

	for _, needle := range []string{
		"BIG-GO-1562",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` tranche-B physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` tranche-B physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused tranche-B ledger: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/runtime.py`",
		"`src/bigclaw/scheduler.py`",
		"`src/bigclaw/orchestration.py`",
		"`src/bigclaw/workflow.py`",
		"`src/bigclaw/queue.py`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/orchestrator/loop.go`",
		"`bigclaw-go/internal/queue/queue.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1562",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
