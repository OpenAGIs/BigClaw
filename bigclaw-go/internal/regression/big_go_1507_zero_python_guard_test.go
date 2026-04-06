package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1507RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1507LargestResidualDirectoryIsEmpty(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected no residual Python-bearing directory, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1507LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1507-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1507",
		"Repository-wide Python file count before lane changes: `0`",
		"Repository-wide Python file count after lane changes: `0`",
		"Count delta: `0`",
		"Result: none",
		"Exact Deleted Files",
		"- None",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`python_file_count=$(find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l | tr -d ' '); printf '%s\\n' \"$python_file_count\"`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sed 's#^./##' | awk -F/ 'NF { if (NF == 1) print \".\"; else print $1 }' | sort | uniq -c | sort -nr`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1507",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
