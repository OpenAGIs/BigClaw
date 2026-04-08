package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO148RepositoryHasNoPythonAssets(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-asset-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO148BroadResidualDirectoriesStayPythonAssetFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		".githooks",
		".github",
		".symphony",
		"scripts",
		"bigclaw-go/examples",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual directory to remain Python-asset-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO148GoNativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".githooks/post-commit",
		".github/workflows/ci.yml",
		"scripts/ops/bigclawctl",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/examples/shadow-task.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO148LaneReportCapturesBroadenedSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-148-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-148",
		"Repository-wide physical Python asset count before lane changes: `0`",
		"Repository-wide physical Python asset count after lane changes: `0`",
		"Broadened Python asset extensions covered by this lane: `.py`, `.pyw`, `.pyi`, `.ipynb`",
		"`.githooks`: `0` Python assets",
		"`.github`: `0` Python assets",
		"`.symphony`: `0` Python assets",
		"`scripts`: `0` Python assets",
		"`bigclaw-go/examples`: `0` Python assets",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python assets",
		"`.githooks/post-commit`",
		"`.github/workflows/ci.yml`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/examples/shadow-task.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .githooks .github .symphony scripts bigclaw-go/examples bigclaw-go/docs/reports/live-validation-runs -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO148|BIGGO109|BIGGO1174|E2E|RootOpsDirectoryStaysPythonFree)'`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestBenchmarkScriptsStayGoOnly'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
