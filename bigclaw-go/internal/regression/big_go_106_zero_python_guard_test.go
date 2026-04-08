package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO106SupportAssetDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	supportAssetDirs := []string{
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"scripts/ops",
	}

	for _, relativeDir := range supportAssetDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected support asset directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO106SupportAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredPaths := []string{
		"bigclaw-go/examples/shadow-corpus-manifest.json",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-task-budget.json",
		"bigclaw-go/examples/shadow-task-validation.json",
		"bigclaw-go/docs/reports/migration-readiness-report.md",
		"bigclaw-go/docs/reports/shadow-matrix-report.json",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/migration-shadow.md",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-symphony",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected support asset or helper path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO106LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-106-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-106",
		"Support-asset Python file count: `0`.",
		"`bigclaw-go/examples`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`bigclaw-go/docs/reports/migration-readiness-report.md`",
		"`bigclaw-go/docs/reports/shadow-matrix-report.json`",
		"`bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/migration-shadow.md`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-symphony`",
		"`find bigclaw-go/examples bigclaw-go/docs/reports bigclaw-go/docs/reports/live-shadow-runs scripts/ops -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO106",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
