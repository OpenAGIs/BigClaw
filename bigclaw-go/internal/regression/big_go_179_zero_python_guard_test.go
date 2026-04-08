package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO179RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO179HiddenAndNestedSweepDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		".github",
		".githooks",
		".symphony",
		"reports",
		"bigclaw-go/docs/reports/live-validation-runs",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected hidden or nested sweep directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO179NativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".githooks/post-rewrite",
		".symphony/workpad.md",
		"reports/BIG-GO-168-validation.md",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/fault-timeline.json",
		"bigclaw-go/docs/reports/review-readiness.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO179LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-179-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-179",
		"Repository-wide Python file count: `0`.",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`.symphony/workpad.md`",
		"`reports/BIG-GO-168-validation.md`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/fault-timeline.json`",
		"`bigclaw-go/docs/reports/review-readiness.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find . \\( -path './.git' -o -path './node_modules' -o -path './.venv' -o -path './venv' \\) -prune -o \\( -path './.github/*' -o -path './.githooks/*' -o -path './.symphony/*' -o -path './reports/*' -o -path './bigclaw-go/docs/reports/live-validation-runs/*' -o -path './bigclaw-go/docs/reports/live-shadow-runs/*' -o -path './bigclaw-go/docs/reports/broker-failover-stub-artifacts/*' \\) -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO179(RepositoryHasNoPythonFiles|HiddenAndNestedSweepDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
