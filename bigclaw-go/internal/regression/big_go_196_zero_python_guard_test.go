package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO196RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO196SupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	supportDirs := []string{
		"bigclaw-go/examples",
		"reports",
		"docs/reports",
		"bigclaw-go/docs/reports",
		"scripts/ops",
	}

	for _, relativeDir := range supportDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected support-asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO196RetainedSupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"reports/BIG-GO-186-validation.md",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/benchmark-report.md",
		"bigclaw-go/docs/reports/mixed-workload-validation-report.md",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/docs/reports/live-validation-index.md",
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

func TestBIGGO196LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-196-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-196",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/examples`: `0` Python files",
		"`reports`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-task-validation.json`",
		"`reports/BIG-GO-186-validation.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/benchmark-report.md`",
		"`bigclaw-go/docs/reports/mixed-workload-validation-report.md`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/live-validation-index.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/examples reports docs/reports bigclaw-go/docs/reports scripts/ops -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO196(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
