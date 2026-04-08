package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO10RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO10RootPythonToolingConfigStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, ".pre-commit-config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python tooling config to stay absent: .pre-commit-config.yaml")
	}
}

func TestBIGGO10DocumentationCapturesFinalSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-10-final-python-sweep.md")
	for _, needle := range []string{
		"BIG-GO-10",
		"`.pre-commit-config.yaml`",
		"`git diff --check`",
		"`make test`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`test ! -e .pre-commit-config.yaml && printf 'absent\\n'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO10'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("final sweep report missing substring %q", needle)
		}
	}

	readme := readRepoFile(t, rootRepo, "README.md")
	if strings.Contains(readme, "pre-commit run --all-files") {
		t.Fatal("README.md should not reference retired pre-commit hygiene guidance")
	}
	for _, needle := range []string{"Repository hygiene:", "git diff --check", "make test"} {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing replacement hygiene guidance %q", needle)
		}
	}
}
