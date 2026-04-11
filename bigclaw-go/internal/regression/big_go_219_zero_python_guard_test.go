package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO219RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO219OverlookedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	overlookedDirs := []string{
		"bigclaw-go/docs/adr",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		".symphony",
	}

	for _, relativeDir := range overlookedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected overlooked auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO219NativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	nativeEvidencePaths := []string{
		"bigclaw-go/docs/adr/0001-go-rewrite-control-plane.md",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		".symphony/workpad.md",
	}

	for _, relativePath := range nativeEvidencePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO219LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-219-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-219",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/docs/adr`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`bigclaw-go/docs/adr/0001-go-rewrite-control-plane.md`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`.symphony/workpad.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/docs/adr bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts .symphony -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO219(RepositoryHasNoPythonFiles|OverlookedAuxiliaryDirectoriesStayPythonFree|NativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
