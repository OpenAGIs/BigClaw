package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1490RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1490FindAnchorReportCapturesBlockedReduction(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1490-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1490",
		"`find . -name '*.py' | sort`",
		"Before change:",
		"After change:",
		"repository-wide Python file count was `0`",
		"repository-wide Python file count remained `0`",
		"requested physical Python-file reduction is blocked",
		"no `.py` files left in this checkout",
		"`bigclaw-go/internal/regression/big_go_1490_zero_python_guard_test.go`",
		"`git ls-remote --heads https://github.com/OpenAGIs/BigClaw.git BIG-GO-1490`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1490",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
