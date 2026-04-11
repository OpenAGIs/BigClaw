package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO206RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO206SupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	supportDirs := []string{
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		"bigclaw-go/scripts/e2e",
	}

	for _, relativeDir := range supportDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected support-asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO206RetiredPythonSupportHelpersRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py",
		"bigclaw-go/scripts/e2e/run_all_test.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python support helper to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO206RetainedSupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/replay-capture.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained support asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO206LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-206-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-206",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files",
		"`bigclaw-go/scripts/e2e`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/replay-capture.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/examples bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO206(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetiredPythonSupportHelpersRemainAbsent|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
