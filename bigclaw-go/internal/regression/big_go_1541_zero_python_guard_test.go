package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1541SrcBigclawHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "src/bigclaw"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected src/bigclaw to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1541DeletionLedgerMatchesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "src/bigclaw"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected empty deleted-file ledger only when src/bigclaw is Python-free, found %v", pythonFiles)
	}

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1541-src-bigclaw-python-sweep.md")
	if !strings.Contains(report, "- Deleted files in `BIG-GO-1541`: `[]`") {
		t.Fatalf("expected exact deleted-file list to be empty in lane report")
	}
}

func TestBIGGO1541LaneReportCapturesExactDeletedFileList(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1541-src-bigclaw-python-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1541",
		"`src/bigclaw` `.py` files before lane changes: `0`",
		"`src/bigclaw` `.py` files after lane changes: `0`",
		"`src/bigclaw` is not present in this checkout",
		"- Deleted files in `BIG-GO-1541`: `[]`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1541",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
