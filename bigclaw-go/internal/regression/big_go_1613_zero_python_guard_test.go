package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1613RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1613RemainingScriptBucketsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	scriptBuckets := []string{
		"bigclaw-go/scripts/benchmark",
		"bigclaw-go/scripts/e2e",
	}

	for _, relativeDir := range scriptBuckets {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected script bucket to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}

	if _, err := os.Stat(filepath.Join(rootRepo, "bigclaw-go", "scripts", "migration")); !os.IsNotExist(err) {
		t.Fatalf("expected retired migration script bucket to remain absent, err=%v", err)
	}
}

func TestBIGGO1613RetiredPythonRunnersRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python runner to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1613ReplacementSurfacesRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_broker_stub_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_takeover_matrix_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1613LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1613-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1613",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/scripts/benchmark`: `0` Python files",
		"`bigclaw-go/scripts/e2e`: `0` Python files",
		"`bigclaw-go/scripts/migration`: retired directory absent",
		"`bigclaw-go/scripts/benchmark/capacity_certification.py`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py`",
		"`bigclaw-go/scripts/benchmark/soak_local.py`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py`",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclawctl automation e2e run-task-smoke|export-validation-bundle|continuation-scorecard|continuation-policy-gate|broker-failover-stub-matrix|mixed-workload-matrix|cross-process-coordination-surface|subscriber-takeover-fault-matrix|external-store-validation|multi-node-shared-queue`",
		"`bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/scripts/benchmark bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1613(RepositoryHasNoPythonFiles|RemainingScriptBucketsStayPythonFree|RetiredPythonRunnersRemainAbsent|ReplacementSurfacesRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
