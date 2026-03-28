package testharness

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRepoAndProjectRoots(t *testing.T) {
	repoRoot := RepoRoot(t)
	if filepath.Base(repoRoot) != "bigclaw-go" {
		t.Fatalf("expected repo root to end with bigclaw-go, got %q", repoRoot)
	}

	projectRoot := ProjectRoot(t)
	if filepath.Base(projectRoot) != "BIG-GO-923" {
		t.Fatalf("expected project root to end with BIG-GO-923, got %q", projectRoot)
	}
	if got := filepath.Dir(repoRoot); got != projectRoot {
		t.Fatalf("expected project root %q, got %q", got, projectRoot)
	}
}

func TestJoinAndResolveProjectPaths(t *testing.T) {
	if got := JoinRepoRoot(t, "docs", "reports"); got != filepath.Join(RepoRoot(t), "docs", "reports") {
		t.Fatalf("unexpected repo path join: %q", got)
	}
	if got := JoinProjectRoot(t, "src", "bigclaw"); got != filepath.Join(ProjectRoot(t), "src", "bigclaw") {
		t.Fatalf("unexpected project path join: %q", got)
	}
	if got := ResolveProjectPath(t, "bigclaw-go/docs/reports/live-validation-index.json"); got != filepath.Join(RepoRoot(t), "docs", "reports", "live-validation-index.json") {
		t.Fatalf("unexpected resolved path: %q", got)
	}
}

func TestLegacySrcAndPythonPathBootstrap(t *testing.T) {
	if got := LegacySrcRoot(t); got != filepath.Join(ProjectRoot(t), "src") {
		t.Fatalf("unexpected src root: %q", got)
	}

	t.Setenv("PYTHONPATH", "existing")
	srcRoot := BootstrapLegacyPythonPath(t)
	want := srcRoot + string(os.PathListSeparator) + "existing"
	if got := os.Getenv("PYTHONPATH"); got != want {
		t.Fatalf("unexpected PYTHONPATH: got=%q want=%q", got, want)
	}
}

func TestInventoryPytestAssets(t *testing.T) {
	inventory := InventoryPytestAssets(t)

	if inventory.ConftestPath != filepath.Join(ProjectRoot(t), "tests", "conftest.py") {
		t.Fatalf("unexpected conftest path: %q", inventory.ConftestPath)
	}
	if !inventory.ConftestPrependsSrc {
		t.Fatal("expected conftest to prepend src to sys.path")
	}
	if got := inventory.Summary(); got != "tests=56 bigclaw_imports=47 pytest_imports=3" {
		t.Fatalf("unexpected inventory summary: %s", got)
	}

	wantPytestModules := []string{
		"tests/test_audit_events.py",
		"tests/test_planning.py",
		"tests/test_roadmap.py",
	}
	if !reflect.DeepEqual(inventory.PytestImportModules, wantPytestModules) {
		t.Fatalf("unexpected pytest import modules: got=%v want=%v", inventory.PytestImportModules, wantPytestModules)
	}
}
