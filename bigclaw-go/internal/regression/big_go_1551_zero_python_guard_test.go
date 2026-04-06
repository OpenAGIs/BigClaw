package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1551RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1551SrcBigclawDirectoryStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	targetDir := filepath.Join(rootRepo, "src", "bigclaw")

	if _, err := os.Stat(targetDir); err == nil {
		pythonFiles := collectPythonFiles(t, targetDir)
		if len(pythonFiles) != 0 {
			t.Fatalf("expected src/bigclaw to remain Python-free, found %v", pythonFiles)
		}
		return
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat src/bigclaw: %v", err)
	}
}

func TestBIGGO1551HistoricalDeletedFileEvidenceIsRecorded(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1551-src-bigclaw-python-sweep.md")

	for _, needle := range []string{
		"Total historical `src/bigclaw/*.py` deletions found on current `HEAD`",
		"`50`",
		"`c2835f42`: `src/bigclaw/legacy_shim.py`, `src/bigclaw/models.py`",
		"`410602dc`: `src/bigclaw/audit_events.py`, `src/bigclaw/collaboration.py`,",
		"`3fd2f9c1`: `src/bigclaw/parallel_refill.py`",
		"`e0de6da9`: `src/bigclaw/orchestration.py`, `src/bigclaw/queue.py`,",
		"Deleted files in this lane: `[]`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("historical deletion evidence missing substring %q", needle)
		}
	}
}

func TestBIGGO1551LaneReportCapturesCurrentDeltaAndBlocker(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1551-src-bigclaw-python-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1551",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Repository-wide physical Python file delta in this checkout: `0`",
		"`src/bigclaw` physical Python file count before lane changes: `0`",
		"`src/bigclaw` physical Python file count after lane changes: `0`",
		"`src/bigclaw` physical Python file delta in this checkout: `0`",
		"Acceptance asked for a lower physical `.py` file count, but that is not",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`scripts/dev_bootstrap.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1551",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
