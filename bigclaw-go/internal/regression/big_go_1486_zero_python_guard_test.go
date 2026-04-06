package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1486RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1486PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1486WorkspaceBootstrapAndPlanningPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"docs/symphony-repo-bootstrap-template.md",
		"docs/issue-plan.md",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/planning/planning_test.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected workspace/bootstrap/planning replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1486LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1486-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1486",
		"Repository-wide Python file count before lane work: `0`.",
		"Repository-wide Python file count after lane work: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`docs/issue-plan.md`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap_test.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/planning/planning_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/bootstrap ./internal/planning ./internal/regression -run 'TestBIGGO1486",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
