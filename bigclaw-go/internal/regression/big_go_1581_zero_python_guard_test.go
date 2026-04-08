package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1581RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1581BucketARetiredPythonPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/cost_control.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/github_sync.py",
		"scripts/ops/bigclaw_github_sync.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired bucket-A Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1581GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/costcontrol/controller.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"docs/go-mainline-cutover-issue-pack.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1581LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1581-src-bigclaw-bucket-a.md")

	for _, needle := range []string{
		"BIG-GO-1581",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused bucket-A physical Python file count before lane changes: `0`",
		"Focused bucket-A physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused bucket-A ledger: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/cost_control.py`",
		"`src/bigclaw/issue_archive.py`",
		"`src/bigclaw/github_sync.py`",
		"`scripts/ops/bigclaw_github_sync.py`",
		"`bigclaw-go/internal/costcontrol/controller.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw scripts/ops -type f \\( -name 'cost_control.py' -o -name 'issue_archive.py' -o -name 'github_sync.py' -o -name 'bigclaw_github_sync.py' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1581",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
