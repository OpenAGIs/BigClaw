package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO239RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonArtifacts := collectBIGGO239PythonArtifacts(t, rootRepo)
	if len(pythonArtifacts) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonArtifacts), pythonArtifacts)
	}
}

func TestBIGGO239HiddenNestedAndOverlookedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		".github",
		".githooks",
		".symphony",
		"docs/reports",
		"reports",
		"bigclaw-go/docs/adr",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range residualDirs {
		pythonArtifacts := collectBIGGO239PythonArtifacts(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonArtifacts) != 0 {
			t.Fatalf("expected hidden, nested, or overlooked auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonArtifacts)
		}
	}
}

func TestBIGGO239RetainedNativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".symphony/workpad.md",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/adr/0001-go-rewrite-control-plane.md",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO239LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-239-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-239",
		"Repository-wide Python file count: `0`.",
		"Python-adjacent auxiliary file count (`*.py`, `*.pyw`, `*.pyi`, `*.ipynb`, `*.pyc`, `.python-version`): `0`.",
		"Hidden, nested, and overlooked auxiliary directories audited in this lane:",
		"`.github`: `0` Python-adjacent files",
		"`.githooks`: `0` Python-adjacent files",
		"`.symphony`: `0` Python-adjacent files",
		"`docs/reports`: `0` Python-adjacent files",
		"`reports`: `0` Python-adjacent files",
		"`bigclaw-go/docs/adr`: `0` Python-adjacent files",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python-adjacent files",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python-adjacent files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python-adjacent files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python-adjacent files",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.symphony/workpad.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/adr/0001-go-rewrite-control-plane.md`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' -o -name '*.pyc' -o -name '.python-version' \\) -print | sort`",
		"`find .github .githooks .symphony docs/reports bigclaw-go/docs/adr bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs reports -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' -o -name '*.pyc' -o -name '.python-version' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO239(RepositoryHasNoPythonFiles|HiddenNestedAndOverlookedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectBIGGO239PythonArtifacts(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if !isBIGGO239PythonArtifact(path) {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}

func isBIGGO239PythonArtifact(path string) bool {
	base := filepath.Base(path)
	if base == ".python-version" {
		return true
	}

	switch filepath.Ext(path) {
	case ".py", ".pyw", ".pyi", ".ipynb", ".pyc":
		return true
	default:
		return false
	}
}
