package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO215RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO215ToolingDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	toolingDirs := []string{
		".github",
		".githooks",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range toolingDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected tooling directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO215NativeToolingReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".githooks/post-rewrite",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native tooling replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO215LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-215-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-215",
		"Residual tooling Python sweep Q",
		"Repository-wide Python file count: `0`.",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`docs/go-cli-script-migration-plan.md`",
		"`README.md`",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.githooks/post-rewrite`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .github .githooks scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO215(RepositoryHasNoPythonFiles|ToolingDirectoriesStayPythonFree|NativeToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
