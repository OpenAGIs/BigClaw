package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO1493RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1493BigclawGoScriptsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	scriptsDir := filepath.Join(rootRepo, "bigclaw-go", "scripts")

	pythonFiles := collectPythonFiles(t, scriptsDir)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected bigclaw-go/scripts to remain Python-free, found %v", pythonFiles)
	}
}

func TestBIGGO1493BigclawGoScriptsKeepNativeHelperSet(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	scriptsDir := filepath.Join(rootRepo, "bigclaw-go", "scripts")

	expected := map[string]struct{}{
		"benchmark/run_suite.sh":          {},
		"e2e/broker_bootstrap_summary.go": {},
		"e2e/kubernetes_smoke.sh":         {},
		"e2e/ray_smoke.sh":                {},
		"e2e/run_all.sh":                  {},
	}

	actual := map[string]struct{}{}
	err := filepath.WalkDir(scriptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return err
		}
		actual[filepath.ToSlash(relative)] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("walk bigclaw-go/scripts: %v", err)
	}

	if len(actual) != len(expected) {
		t.Fatalf("expected %d native helper file(s) under bigclaw-go/scripts, found %d (%v)", len(expected), len(actual), sortedKeys(actual))
	}
	for relativePath := range expected {
		if _, ok := actual[relativePath]; !ok {
			t.Fatalf("expected native helper to remain under bigclaw-go/scripts: %s", relativePath)
		}
	}
	for relativePath := range actual {
		if _, ok := expected[relativePath]; !ok {
			t.Fatalf("unexpected helper artifact under bigclaw-go/scripts: %s", relativePath)
		}
	}
}

func TestBIGGO1493LaneReportCapturesZeroDeltaAndOwnership(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1493-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1493",
		"Repository-wide Python file count before sweep: `0`.",
		"Repository-wide Python file count after sweep: `0`.",
		"`bigclaw-go/scripts`: `0` Python files before, `0` after",
		"Deleted files: none; the candidate Python paths were already absent",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"Delete condition: remove new helper artifacts under `bigclaw-go/scripts` if",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`find bigclaw-go/scripts -type f | sed 's#^./##' | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160MigrationDocsListGoReplacements|TestBIGGO1493",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
