package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO249RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectAuxiliaryPythonLikeFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO249HiddenNestedAuxiliaryDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auxiliaryDirs := []string{
		".githooks",
		".github",
		".symphony",
		"docs/reports",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range auxiliaryDirs {
		pythonFiles := collectAuxiliaryPythonLikeFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected hidden or nested auxiliary directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO249NativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	evidencePaths := []string{
		".githooks/post-commit",
		".github/workflows/ci.yml",
		".symphony/workpad.md",
		"docs/reports/bootstrap-cache-validation.md",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
	}

	for _, relativePath := range evidencePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO249LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-249-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-249",
		"Residual auxiliary Python sweep `BIG-GO-249`",
		"Repository-wide Python file count: `0`.",
		"`.githooks`: `0` Python files",
		"`.github`: `0` Python files",
		"`.symphony`: `0` Python files",
		"`docs/reports`: `0` Python files",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files",
		"`.githooks/post-commit`",
		"`.github/workflows/ci.yml`",
		"`.symphony/workpad.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .githooks .github .symphony docs/reports bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO249(RepositoryHasNoPythonFiles|HiddenNestedAuxiliaryDirectoriesStayPythonFree|NativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectAuxiliaryPythonLikeFiles(t *testing.T, root string) []string {
	t.Helper()

	pythonLikeExtensions := map[string]struct{}{
		".ipynb": {},
		".py":    {},
		".pyi":   {},
		".pyw":   {},
	}

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := pythonLikeExtensions[strings.ToLower(filepath.Ext(path))]; !ok {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}
