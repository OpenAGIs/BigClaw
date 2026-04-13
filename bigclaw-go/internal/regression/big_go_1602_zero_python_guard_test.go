package regression

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBIGGO1602TargetedBigclawPackageSurfaceAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	packageFiles := collectRepoFiles(t, filepath.Join(rootRepo, "src", "bigclaw"))
	if len(packageFiles) != 0 {
		t.Fatalf("expected src/bigclaw package surface to remain absent, found %d file(s): %v", len(packageFiles), packageFiles)
	}

	retiredPackagePaths := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
	}
	for _, relativePath := range retiredPackagePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired package path to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1602NoResidualBigclawPythonImportShims(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualImports := collectResidualBigclawImports(t, rootRepo)
	if len(residualImports) != 0 {
		t.Fatalf("expected remaining bigclaw package imports to be documentation-only or absent, found non-documentation references in %v", residualImports)
	}
}

func TestBIGGO1602NativeEntryPointsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	nativeEntryPoints := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"scripts/ops/bigclawctl",
		"bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go",
		"bigclaw-go/internal/regression/big_go_221_zero_python_guard_test.go",
	}

	for _, relativePath := range nativeEntryPoints {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1602LaneReportCapturesPackageSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1602-python-package-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1602",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` tracked files",
		"`src/bigclaw/__init__.py`",
		"`src/bigclaw/__main__.py`",
		"Remaining `bigclaw` import references are documentation/report-only.",
		"`reports/OPE-130-validation.md`",
		"`reports/OPE-142-validation.md`",
		"`reports/BIG-GO-221-validation.md`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`",
		"`bigclaw-go/internal/regression/big_go_221_zero_python_guard_test.go`",
		"`find . -path '*/.git' -prune -o -path './src/bigclaw' -o -path './src/bigclaw/*' -type f -print | sort`",
		"`rg -n --case-sensitive '(?m)\\b(?:import|from)\\s+bigclaw(?:$|[.[:space:]])' -P -S . --glob '!*.md' --glob '!*.json'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1602(TargetedBigclawPackageSurfaceAbsent|NoResidualBigclawPythonImportShims|NativeEntryPointsRemainAvailable|LaneReportCapturesPackageSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

var bigclawImportPattern = regexp.MustCompile(`(?m)\b(?:import|from)\s+bigclaw(?:$|[.\s])`)

func collectResidualBigclawImports(t *testing.T, root string) []string {
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

		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		relative = filepath.ToSlash(relative)
		if isDocumentationArtifact(relative) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !utf8.Valid(content) || !bigclawImportPattern.Match(content) {
			return nil
		}

		entries = append(entries, relative)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}

func collectRepoFiles(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
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

func isDocumentationArtifact(relativePath string) bool {
	switch strings.ToLower(filepath.Ext(relativePath)) {
	case ".json", ".md":
		return true
	default:
		return false
	}
}
