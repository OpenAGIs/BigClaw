package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO224ResidualScriptDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, relativeDir := range []string{
		"scripts",
		"bigclaw-go/scripts",
	} {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual script directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO224ResidualScriptInventoryRemainsNative(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	inventoryRoots := []string{
		"scripts",
		"bigclaw-go/scripts",
	}
	got := make([]string, 0)
	for _, relativeDir := range inventoryRoots {
		got = append(got, collectRelativeFiles(t, rootRepo, relativeDir)...)
	}
	sort.Strings(got)

	want := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/run_all.sh",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"scripts/ops/bigclawctl",
	}

	if len(got) != len(want) {
		t.Fatalf("expected residual helper inventory %v, got %v", want, got)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("expected residual helper inventory %v, got %v", want, got)
		}
	}
}

func TestBIGGO224LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-224-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-224",
		"Repository-wide Python file count: `0`.",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`find scripts bigclaw-go/scripts -type f | sort`",
		"`find scripts bigclaw-go/scripts -type f -name '*.py' | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO224",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectRelativeFiles(t *testing.T, rootRepo string, relativeDir string) []string {
	t.Helper()

	dir := filepath.Join(rootRepo, filepath.FromSlash(relativeDir))
	entries := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relativePath, relErr := filepath.Rel(rootRepo, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		t.Fatalf("walk residual script inventory %s: %v", relativeDir, err)
	}

	sort.Strings(entries)
	return entries
}
