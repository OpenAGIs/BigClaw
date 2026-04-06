package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1553RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1553BigclawGoScriptsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "bigclaw-go", "scripts"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected bigclaw-go/scripts to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1553ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1553LaneReportCapturesExactDeltaAndLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1553-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1553",
		"Historical `bigclaw-go/scripts` physical Python file count at",
		"`fdb20c43` (`8ebdd50d^`): `23`",
		"Current `bigclaw-go/scripts` physical Python file count on disk: `0`",
		"Exact `bigclaw-go/scripts` count delta: `-23`",
		"Current repository-wide physical Python file count on disk: `0`",
		"`8ebdd50d`: `bigclaw-go/scripts/e2e/run_task_smoke.py`",
		"`da168148`: `bigclaw-go/scripts/benchmark/soak_local.py`",
		"`2ed49341`: `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`",
		"`42363805`: `bigclaw-go/scripts/e2e/multi_node_shared_queue.py`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/scripts -type f -name '*.py' | sort`",
		"`git ls-tree -r --name-only fdb20c43 bigclaw-go/scripts | rg '\\.py$'`",
		"`git log --diff-filter=D --summary -- bigclaw-go/scripts`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1553",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
