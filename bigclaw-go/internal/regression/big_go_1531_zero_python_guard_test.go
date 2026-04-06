package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1531RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1531SrcBigclawSurfaceStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "src", "bigclaw"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected src/bigclaw to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1531RepresentativeHistoricalSrcBigclawFilesRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	absentFiles := []string{
		"src/bigclaw/models.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/dsl.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/orchestration.py",
		"src/bigclaw/pilot.py",
	}

	for _, relativePath := range absentFiles {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected historical src/bigclaw Python file to be absent: %s", relativePath)
		}
	}
}

func TestBIGGO1531GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/go-mainline-cutover-handoff.md",
		"docs/go-mainline-cutover-issue-pack.md",
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/control/controller.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/observability/audit_spec.go",
		"bigclaw-go/internal/orchestrator/loop.go",
		"bigclaw-go/internal/pilot/rollout.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/workflow/orchestration.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1531LaneReportCapturesExactEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1531-src-bigclaw-python-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1531",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused ledger for `src/bigclaw`: `[]`",
		"`src`: directory not present, so residual Python files = `0`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/mapping.py`",
		"`src/bigclaw/dsl.py`",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/operations.py`",
		"`src/bigclaw/orchestration.py`",
		"`src/bigclaw/pilot.py`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`bigclaw-go/internal/domain/task.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/observability/audit_spec.go`",
		"`bigclaw-go/internal/orchestrator/loop.go`",
		"`bigclaw-go/internal/pilot/rollout.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/workflow/orchestration.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1531",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
