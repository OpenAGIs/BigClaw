package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1506RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1506WorkspaceBootstrapPlanningPathsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	inScopeDirs := []string{
		"scripts",
		"bigclaw-go/internal/bootstrap",
		"bigclaw-go/internal/planning",
	}

	for _, relativeDir := range inScopeDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected in-scope directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1506GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goReplacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"workflow.md",
		"docs/symphony-repo-bootstrap-template.md",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
	}

	for _, relativePath := range goReplacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1506LaneArtifactsCaptureDeleteLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1506-workspace-bootstrap-planning-sweep.md")
	ledger := readRepoFile(t, rootRepo, "reports/BIG-GO-1506-delete-ledger.md")

	for _, needle := range []string{
		"BIG-GO-1506",
		"Repository-wide Python file count before: `0`.",
		"Repository-wide Python file count after: `0`.",
		"`scripts`: `0` Python files",
		"`bigclaw-go/internal/bootstrap`: `0` Python files",
		"`bigclaw-go/internal/planning`: `0` Python files",
		"`reports/BIG-GO-1506-delete-ledger.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1506",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}

	for _, needle := range []string{
		"# BIG-GO-1506 Delete Ledger",
		"Deleted files: none.",
		"Before count: `0`.",
		"After count: `0`.",
	} {
		if !strings.Contains(ledger, needle) {
			t.Fatalf("delete ledger missing substring %q", needle)
		}
	}
}
