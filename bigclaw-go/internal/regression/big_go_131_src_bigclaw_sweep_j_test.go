package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO131RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO131ResidualSrcBigClawSweepJStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/reports.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/saved_views.py",
		"src/bigclaw/repo_triage.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired sweep-J Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO131GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/observability/task_run.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/api/v2.go",
		"docs/go-mainline-cutover-issue-pack.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO131LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-131-src-bigclaw-sweep-j.md")

	for _, needle := range []string{
		"BIG-GO-131",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` sweep-J physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` sweep-J physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused sweep-J ledger: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/reports.py`",
		"`src/bigclaw/operations.py`",
		"`src/bigclaw/run_detail.py`",
		"`src/bigclaw/dashboard_run_contract.py`",
		"`src/bigclaw/saved_views.py`",
		"`src/bigclaw/repo_triage.py`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/internal/observability/task_run.go`",
		"`bigclaw-go/internal/repo/triage.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`bigclaw-go/internal/api/v2.go`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO131",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
