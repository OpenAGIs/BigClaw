package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO137RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO137PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO137BroadRepoHighImpactDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	highImpactDirs := []string{
		".github",
		".githooks",
		".symphony",
		"docs",
		"docs/reports",
		"reports",
		"scripts/ops",
		"bigclaw-go/docs",
		"bigclaw-go/docs/reports",
		"bigclaw-go/examples",
	}

	for _, relativeDir := range highImpactDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected high-impact directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO137GoNativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO137LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-137-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-137",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"High-impact auxiliary directories audited in this lane:",
		"`.github`: `0` Python files",
		"`.githooks`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`reports`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/docs`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/examples`: `0` Python files",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find .github .githooks .symphony docs docs/reports reports scripts/ops bigclaw-go/docs bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO137(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|BroadRepoHighImpactDirectoriesStayPythonFree|GoNativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
