package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO107RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO107OperatorControlPlaneDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	operatorDirs := []string{
		"src/bigclaw",
		"bigclaw-go/internal/api",
		"bigclaw-go/internal/product",
		"bigclaw-go/internal/consoleia",
		"bigclaw-go/internal/designsystem",
		"bigclaw-go/internal/uireview",
		"bigclaw-go/internal/collaboration",
		"bigclaw-go/internal/issuearchive",
	}

	for _, relativeDir := range operatorDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected operator/control-plane directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO107GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/api/server.go",
		"bigclaw-go/internal/api/v2.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO107LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-107-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-107",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `operator/control-plane` physical Python file count before lane changes: `0`",
		"Focused `operator/control-plane` physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused operator/control-plane ledger: `[]`",
		"`src/bigclaw`: directory not present, so operator/control-plane residual Python files = `0`",
		"`bigclaw-go/internal/api`: `0` Python files",
		"`bigclaw-go/internal/product`: `0` Python files",
		"`bigclaw-go/internal/consoleia`: `0` Python files",
		"`bigclaw-go/internal/designsystem`: `0` Python files",
		"`bigclaw-go/internal/uireview`: `0` Python files",
		"`bigclaw-go/internal/collaboration`: `0` Python files",
		"`bigclaw-go/internal/issuearchive`: `0` Python files",
		"`src/bigclaw/collaboration.py`",
		"`src/bigclaw/issue_archive.py`",
		"`src/bigclaw/console_ia.py`",
		"`src/bigclaw/design_system.py`",
		"`src/bigclaw/saved_views.py`",
		"`src/bigclaw/ui_review.py`",
		"`src/bigclaw/run_detail.py`",
		"`src/bigclaw/dashboard_run_contract.py`",
		"`src/bigclaw/service.py`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/internal/collaboration/thread.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/designsystem/designsystem.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/internal/uireview/uireview.go`",
		"`bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`bigclaw-go/internal/api/v2.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw bigclaw-go/internal/api bigclaw-go/internal/product bigclaw-go/internal/consoleia bigclaw-go/internal/designsystem bigclaw-go/internal/uireview bigclaw-go/internal/collaboration bigclaw-go/internal/issuearchive -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO107",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
