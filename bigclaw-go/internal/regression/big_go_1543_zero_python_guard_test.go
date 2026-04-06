package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1543BigClawGoScriptsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "bigclaw-go/scripts"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected bigclaw-go/scripts to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1543GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1543LaneReportCapturesExactSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1543-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1543",
		"Before count for `bigclaw-go/scripts/*.py`: `0`.",
		"After count for `bigclaw-go/scripts/*.py`: `0`.",
		"Before file list:",
		"After file list:",
		"Exact removed-file list:",
		"none",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find bigclaw-go/scripts -type f -name '*.py' | sort`",
		"`find bigclaw-go/scripts -type f -name '*.py' | wc -l`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1543",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
