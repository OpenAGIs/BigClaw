package testharness

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	if inventory.ConftestImportsPytest {
		t.Fatal("expected conftest to avoid importing pytest")
	}
	if inventory.ConftestDefinesFixture {
		t.Fatal("expected conftest to avoid defining fixtures")
	}
	if inventory.ConftestDefinesHook {
		t.Fatal("expected conftest to avoid defining pytest hooks")
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

	wantBlockers := []string{
		"56 legacy pytest modules remain under tests/",
		"47 legacy pytest modules still import bigclaw from src/",
		"3 legacy pytest modules still import pytest directly",
	}
	if got := inventory.ConftestDeletionBlockers(); !reflect.DeepEqual(got, wantBlockers) {
		t.Fatalf("unexpected conftest deletion blockers: got=%v want=%v", got, wantBlockers)
	}
	if inventory.CanDeleteConftest() {
		t.Fatal("expected conftest deletion gate to remain closed for the current inventory")
	}
}

func TestBootstrapLegacyPythonPathSupportsBigclawImports(t *testing.T) {
	RequireExecutable(t, "python3")
	cmd := PythonCommand(t, "-c", "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python import smoke failed: %v (%s)", err, string(output))
	}
}

func TestPythonCommandUsesProjectRootAndLegacyPythonPath(t *testing.T) {
	RequireExecutable(t, "python3")
	cmd := PythonCommand(t, "-c", "import os, pathlib; print(pathlib.Path.cwd().name); print(os.environ['PYTHONPATH'])")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python command failed: %v (%s)", err, string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d (%q)", len(lines), string(output))
	}
	if lines[0] != filepath.Base(ProjectRoot(t)) {
		t.Fatalf("unexpected python cwd: got=%q want=%q", lines[0], filepath.Base(ProjectRoot(t)))
	}
	if !strings.HasPrefix(lines[1], LegacySrcRoot(t)) {
		t.Fatalf("expected PYTHONPATH to start with %q, got %q", LegacySrcRoot(t), lines[1])
	}
}

func TestPytestCommandRunsLegacyPytestWithHarnessBootstrap(t *testing.T) {
	RequireExecutable(t, "python3")
	cmd := PytestCommand(t, "tests/test_mapping.py", "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pytest command failed: %v (%s)", err, string(output))
	}
	if !strings.Contains(string(output), "[100%]") {
		t.Fatalf("expected pytest progress output, got %q", string(output))
	}
}
