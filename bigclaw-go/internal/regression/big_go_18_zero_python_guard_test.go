package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO18RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO18HighImpactDocumentationDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		"docs",
		"reports",
		"bigclaw-go/docs",
		"bigclaw-go/examples",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected high-impact documentation directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO18RetainedNativeDocumentationAssetsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"docs/go-cli-script-migration-plan.md",
		"docs/go-mainline-cutover-handoff.md",
		"reports/BIG-GO-17-validation.md",
		"reports/BIG-GO-170-status.json",
		"bigclaw-go/docs/migration.md",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"bigclaw-go/examples/shadow-task.json",
		"bigclaw-go/examples/shadow-corpus-manifest.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained documentation asset to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO18LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-18-python-count-reduction-pass-c.md")

	for _, needle := range []string{
		"BIG-GO-18",
		"Repository-wide Python file count: `0`.",
		"`docs`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"Explicit remaining Python asset list: none.",
		"`docs/go-cli-script-migration-plan.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`reports/BIG-GO-17-validation.md`",
		"`reports/BIG-GO-170-status.json`",
		"`bigclaw-go/docs/migration.md`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`bigclaw-go/examples/shadow-corpus-manifest.json`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find docs reports bigclaw-go/docs bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO18(RepositoryHasNoPythonFiles|HighImpactDocumentationDirectoriesStayPythonFree|RetainedNativeDocumentationAssetsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
