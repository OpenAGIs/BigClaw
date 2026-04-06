package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1484ScriptsTreeHasNoPythonWrappers(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "scripts"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected scripts tree to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1484ShellReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1484LaneReportCapturesScriptsWrapperBaseline(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1484-python-wrapper-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1484",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`git ls-files '*.py' | wc -l`",
		"`rg --files scripts scripts/ops -g '*.py' | wc -l`",
		"`bash scripts/ops/bigclaw-issue --help`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
