package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1566RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1566ScriptsTrancheBStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"bigclaw-go/scripts",
		"bigclaw-go/scripts/benchmark",
		"bigclaw-go/scripts/e2e",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected scripts tranche-B area to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1566ScriptsGoNativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1566LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1566-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1566",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `bigclaw-go/scripts` physical Python file count before lane changes:",
		"Focused `bigclaw-go/scripts` physical Python file count after lane changes:",
		"Deleted files in this lane: `[]`",
		"Focused ledger for `bigclaw-go/scripts` tranche B: `[]`",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/scripts/benchmark`: `0` Python files",
		"`bigclaw-go/scripts/e2e`: `0` Python files",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/scripts -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1566",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
