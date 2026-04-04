package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1181RemainingPythonInventoryIsEmpty(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	priorityDirs := []string{
		".",
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		scanRoot := root
		if relativeDir != "." {
			scanRoot = filepath.Join(root, filepath.FromSlash(relativeDir))
		}

		pythonFiles := collectPythonFiles(t, scanRoot)
		if len(pythonFiles) != 0 {
			t.Fatalf("expected remaining Python inventory to stay empty for %s, found %v", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1181GoReplacementEntrypointsStayDocumented(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))

	replacementDocs := []struct {
		path    string
		needles []string
	}{
		{
			path: "README.md",
			needles: []string{
				"`bash scripts/ops/bigclawctl refill ...` is the supported refill entrypoint.",
				"`bash scripts/ops/bigclawctl github-sync ...`.",
				"`go run ./bigclaw-go/cmd/bigclawctl automation e2e run-task-smoke ...`",
				"`go run ./bigclaw-go/cmd/bigclawctl automation benchmark soak-local ...`",
				"`go run ./bigclaw-go/cmd/bigclawctl automation migration shadow-compare ...`",
				"`scripts/ops/bigclawctl`.",
			},
		},
		{
			path: "docs/go-cli-script-migration-plan.md",
			needles: []string{
				"retired `scripts/create_issues.py`; use `bigclawctl create-issues`",
				"root dev smoke path is Go-only: use `bigclawctl dev-smoke`",
				"retired `scripts/ops/bigclaw_github_sync.py`; use `bigclawctl github-sync`",
				"retired the refill Python wrapper; use `bigclawctl refill`",
				"retired `scripts/ops/bigclaw_workspace_bootstrap.py`; use `bash scripts/ops/bigclawctl workspace bootstrap`",
				"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
				"`bigclaw-go/scripts/e2e/` operator entrypoints now dispatch through `bigclawctl automation e2e ...`",
				"retired benchmark Python helpers -> `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
				"retired migration Python helpers -> `bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
				"`bash scripts/ops/bigclawctl legacy-python compile-check --json`",
			},
		},
	}

	for _, doc := range replacementDocs {
		contents := readRepoFile(t, root, doc.path)
		for _, needle := range doc.needles {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing Go replacement guidance %q", doc.path, needle)
			}
		}
	}
}
