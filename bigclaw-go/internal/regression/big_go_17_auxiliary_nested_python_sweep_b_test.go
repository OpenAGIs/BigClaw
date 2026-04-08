package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO17NestedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	nestedAuxiliaryDirs := []string{
		"docs/reports",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range nestedAuxiliaryDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected nested auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO17RetainedNativeEvidenceAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-05/publish-attempt-ledger.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-07/fault-timeline.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-b-audit.jsonl",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native evidence asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO17LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-17-auxiliary-nested-python-sweep-b.md")

	for _, needle := range []string{
		"BIG-GO-17",
		"Repository-wide Python file count: `0`.",
		"`docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-05/publish-attempt-ledger.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find docs/reports bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO17(NestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidenceAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
