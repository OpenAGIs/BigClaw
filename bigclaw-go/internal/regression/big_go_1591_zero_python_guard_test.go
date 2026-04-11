package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1591RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1591FocusAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	focusPaths := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/repo_links.py",
		"src/bigclaw/scheduler.py",
		"tests/test_connectors.py",
		"tests/test_execution_contract.py",
		"tests/test_models.py",
	}

	for _, relativePath := range focusPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected focused Python asset to stay absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1591GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/go.mod",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/repo/links.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/contract/execution.go",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/workflow/model.go",
		"bigclaw-go/internal/reporting/reporting.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1591LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1591-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1591",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/__init__.py`: absent",
		"`src/bigclaw/evaluation.py`: absent",
		"`src/bigclaw/operations.py`: absent",
		"`src/bigclaw/repo_links.py`: absent",
		"`src/bigclaw/scheduler.py`: absent",
		"`tests/test_connectors.py`: absent",
		"`tests/test_execution_contract.py`: absent",
		"`tests/test_models.py`: absent",
		"`bigclaw-go/go.mod`",
		"`bigclaw-go/internal/evaluation/evaluation.go`",
		"`bigclaw-go/internal/repo/links.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/contract/execution.go`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/workflow/model.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`for path in src/bigclaw/__init__.py src/bigclaw/evaluation.py src/bigclaw/operations.py src/bigclaw/repo_links.py src/bigclaw/scheduler.py tests/test_connectors.py tests/test_execution_contract.py tests/test_models.py; do if test -e \"$path\"; then echo \"present:$path\"; else echo \"absent:$path\"; fi; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1591",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
