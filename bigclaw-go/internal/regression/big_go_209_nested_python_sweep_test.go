package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO209NestedAndHiddenRepositoryInventoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectExtendedPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain free of nested or hidden Python files, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO209PriorityResidualDirectoriesStayPythonFreeUnderExtendedSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
		".github",
		".githooks",
		".symphony",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectExtendedPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual directory to remain free of nested or hidden Python files: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO209LaneReportCapturesNestedSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-209-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-209",
		"Repository-wide extended Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"Hidden roots checked:",
		"`.github`",
		"`.githooks`",
		"`.symphony`",
		"Extensions checked: `.py`, `.pyi`, `.pyw`.",
		"`bigclaw-go/internal/regression/big_go_209_nested_python_sweep_test.go`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' \\) -print | sort`",
		"`find src/bigclaw tests scripts bigclaw-go/scripts .github .githooks .symphony -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO209(NestedAndHiddenRepositoryInventoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFreeUnderExtendedSweep|LaneReportCapturesNestedSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectExtendedPythonFiles(t *testing.T, root string) []string {
	t.Helper()

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
		switch strings.ToLower(filepath.Ext(path)) {
		case ".py", ".pyi", ".pyw":
		default:
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
