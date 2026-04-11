package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1596RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1596AssignedPythonAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/console_ia.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/queue.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/workspace_bootstrap.py",
		"tests/test_dashboard_run_contract.py",
		"tests/test_issue_archive.py",
		"tests/test_parallel_validation_bundle.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1596GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"scripts/ops/bigclawctl",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1596LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md")

	for _, needle := range []string{
		"BIG-GO-1596",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"Explicit remaining Python asset list: none.",
		"`src/bigclaw/console_ia.py` -> `bigclaw-go/internal/consoleia/consoleia.go`",
		"`src/bigclaw/issue_archive.py` -> `bigclaw-go/internal/issuearchive/archive.go`",
		"`src/bigclaw/queue.py` -> `bigclaw-go/internal/queue/queue.go`",
		"`src/bigclaw/risk.py` -> `bigclaw-go/internal/risk/risk.go`",
		"`src/bigclaw/workspace_bootstrap.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`tests/test_dashboard_run_contract.py` -> `bigclaw-go/internal/product/dashboard_run_contract_test.go`",
		"`tests/test_issue_archive.py` -> `bigclaw-go/internal/issuearchive/archive_test.go`",
		"`tests/test_parallel_validation_bundle.py` -> `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`",
		"`scripts/ops/bigclawctl`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/console_ia.py src/bigclaw/issue_archive.py src/bigclaw/queue.py src/bigclaw/risk.py src/bigclaw/workspace_bootstrap.py tests/test_dashboard_run_contract.py tests/test_issue_archive.py tests/test_parallel_validation_bundle.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1596(RepositoryHasNoPythonFiles|AssignedPythonAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1596 hardens that baseline rather than lowering the numeric file count further.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
