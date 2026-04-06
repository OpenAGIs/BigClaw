package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1505RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1505DeleteLedgerCapturesRepositoryReality(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	ledger := readRepoFile(t, rootRepo, "reports/BIG-GO-1505-delete-ledger.json")

	for _, needle := range []string{
		`"identifier": "BIG-GO-1505"`,
		`"baseline_commit": "a63c8ec"`,
		`"before_repo_python_count": 0`,
		`"after_repo_python_count": 0`,
		`"reporting_observability_python_files_found": []`,
		`"deleted_files": []`,
		`"blocked": true`,
		`zero physical .py files`,
	} {
		if !strings.Contains(ledger, needle) {
			t.Fatalf("delete ledger missing substring %q", needle)
		}
	}
}

func TestBIGGO1505LaneReportCapturesBlockedSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1505-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1505",
		"Before repository-wide Python file count: `0`",
		"After repository-wide Python file count: `0`",
		"Reporting/observability Python files found in this checkout: `0`",
		"`reports/BIG-GO-1505-delete-ledger.json`",
		"Deleted files: none",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find . -path '*/.git' -prune -o -type f \\( -path '*/reporting/*.py' -o -path '*/observability/*.py' -o -name '*report*.py' -o -name '*observability*.py' \\) -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1505(RepositoryHasNoPythonFiles|DeleteLedgerCapturesRepositoryReality|LaneReportCapturesBlockedSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
