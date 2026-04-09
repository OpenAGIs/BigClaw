package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO189RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO189PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO189HiddenAndNestedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auxiliaryDirs := []string{
		".github",
		".githooks",
		".symphony",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
	}

	for _, relativeDir := range auxiliaryDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected hidden or nested auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO189RetainedNativeAuxiliaryAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".symphony/workpad.md",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-b-audit.jsonl",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native auxiliary asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO189LaneReportDocumentsPythonAssetSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-189-python-asset-sweep.md")

	requiredSubstrings := []string{
		"# BIG-GO-189 Python Asset Sweep",
		"Remaining physical Python asset inventory: `0` files.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Hidden and nested auxiliary directories audited in this lane:",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.symphony/workpad.md`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-b-audit.jsonl`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .github .githooks .symphony bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO189(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|HiddenAndNestedAuxiliaryDirectoriesStayPythonFree|RetainedNativeAuxiliaryAssetsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("big-go-189 lane report missing substring %q", needle)
		}
	}
}
