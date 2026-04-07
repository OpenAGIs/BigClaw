package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1571RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1571CandidatePythonSweepPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	candidatePaths := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/repo_links.py",
		"src/bigclaw/scheduler.py",
		"tests/test_connectors.py",
		"tests/test_execution_contract.py",
		"tests/test_models.py",
		"tests/test_repo_collaboration.py",
		"tests/test_runtime.py",
		"tests/test_workspace_bootstrap.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/e2e/run_all_test.py",
	}

	for _, relativePath := range candidatePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected BIG-GO-1571 candidate Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1571PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, relativeDir := range []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	} {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1571ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/api/v2.go",
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"docs/go-cli-script-migration-plan.md",
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/docs/reports/benchmark-readiness-report.md",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected BIG-GO-1571 replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1571LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1571-python-asset-sweep.md")

	requiredSubstrings := []string{
		"BIG-GO-1571",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused BIG-GO-1571 candidate physical Python file count before lane changes: `0`",
		"Focused BIG-GO-1571 candidate physical Python file count after lane changes: `0`",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`src/bigclaw/__init__.py`",
		"`src/bigclaw/evaluation.py`",
		"`src/bigclaw/operations.py`",
		"`src/bigclaw/repo_links.py`",
		"`src/bigclaw/scheduler.py`",
		"`tests/test_connectors.py`",
		"`tests/test_execution_contract.py`",
		"`tests/test_models.py`",
		"`tests/test_repo_collaboration.py`",
		"`tests/test_runtime.py`",
		"`tests/test_workspace_bootstrap.py`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py`",
		"`bigclaw-go/scripts/e2e/run_all_test.py`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/api/v2.go`",
		"`bigclaw-go/internal/repo/links.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`docs/go-cli-script-migration-plan.md`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1571",
		"Deletion condition: none; these paths are already physically absent in the current branch baseline.",
		"Result: no output; repository-wide Python file count remained `0`.",
		"Result: no output; the priority residual directories remained Python-free.",
		"Result: `ok",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
