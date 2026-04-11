package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO235RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO235PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO235ToolingDocsStayGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	bootstrapTemplate := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")
	cutoverHandoff := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-handoff.md")

	for _, needle := range []string{
		"`scripts/ops/bigclawctl`",
		"fully Go-first",
		"bash scripts/ops/bigclawctl workspace validate",
		"find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort",
	} {
		if !strings.Contains(bootstrapTemplate+cutoverHandoff, needle) {
			t.Fatalf("expected Go-only tooling guidance %q to remain documented", needle)
		}
	}

	for _, needle := range []string{
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
		"PYTHONPATH=src python3",
	} {
		if strings.Contains(bootstrapTemplate+cutoverHandoff, needle) {
			t.Fatalf("expected stale Python tooling guidance %q to stay removed", needle)
		}
	}
}
