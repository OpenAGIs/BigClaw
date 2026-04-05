package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1467RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1467PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO1467GoBootstrapSurfacesRemainWithoutPythonHooks(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, deletedPath := range []string{
		".pre-commit-config.yaml",
	} {
		if _, err := os.Stat(filepath.Join(rootRepo, deletedPath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python-adjacent surface to be absent: %s", deletedPath)
		}
	}

	for _, replacementPath := range []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
	} {
		if _, err := os.Stat(filepath.Join(rootRepo, replacementPath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", replacementPath, err)
		}
	}

	template := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")
	for _, forbidden := range []string{
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
		"Python compatibility package path",
	} {
		if strings.Contains(template, forbidden) {
			t.Fatalf("expected bootstrap template to stay Go-only, found forbidden substring %q", forbidden)
		}
	}

	readme := readRepoFile(t, rootRepo, "README.md")
	for _, forbidden := range []string{
		"pre-commit run --all-files",
	} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("expected README to avoid Python validation hooks, found forbidden substring %q", forbidden)
		}
	}
}

func TestBIGGO1467LaneReportCapturesBootstrapHookRetirement(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1467-python-bootstrap-surface-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1467",
		"Repository-wide Python file count: `0`.",
		"Deleted root Python validation hook config: `.pre-commit-config.yaml`.",
		"Removed Python bootstrap template references: `workspace_bootstrap.py`, `workspace_bootstrap_cli.py`.",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1467",
		"`cd bigclaw-go && go test -count=1 ./internal/bootstrap ./cmd/bigclawctl`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
