package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO236RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO236SupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	supportDirs := []string{
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
		"scripts/ops",
	}

	for _, relativeDir := range supportDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected support-asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO236RetainedSupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-budget.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained support asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO236LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-236-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-236",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-budget.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/shadow-matrix-report.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/shared-queue-companion-summary.json`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/examples bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs scripts/ops -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO236(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
