package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1352RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1352TestsDirectoryStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"tests",
		"src/bigclaw",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected tests-lane residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1352ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"bigclaw-go/internal/regression/big_go_1352_zero_python_guard_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/cmd/bigclawctl/main_test.go",
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1352LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1352-tests-python-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1352",
		"`tests/*.py` redundancy removal",
		"Repository-wide Python file count: `0`.",
		"`tests`: `0` Python files",
		"`src/bigclaw`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`bigclaw-go/cmd/bigclawctl/main_test.go`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests src/bigclaw scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1352",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
