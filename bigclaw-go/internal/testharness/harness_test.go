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
	if got := inventory.Summary(); got != "tests=46 bigclaw_imports=37 pytest_imports=2" {
		t.Fatalf("unexpected inventory summary: %s", got)
	}

	wantPytestModules := []string{
		"tests/test_audit_events.py",
		"tests/test_planning.py",
	}
	if !reflect.DeepEqual(inventory.PytestImportModules, wantPytestModules) {
		t.Fatalf("unexpected pytest import modules: got=%v want=%v", inventory.PytestImportModules, wantPytestModules)
	}

	wantBlockers := []string{
		"46 legacy pytest modules remain under tests/",
		"37 legacy pytest modules still import bigclaw from src/",
		"2 legacy pytest modules still import pytest directly",
	}
	if got := inventory.ConftestDeletionBlockers(); !reflect.DeepEqual(got, wantBlockers) {
		t.Fatalf("unexpected conftest deletion blockers: got=%v want=%v", got, wantBlockers)
	}
	wantSummary := "conftest_delete_ready=false blockers=46 legacy pytest modules remain under tests/; 37 legacy pytest modules still import bigclaw from src/; 2 legacy pytest modules still import pytest directly"
	if got := inventory.ConftestDeletionSummary(); got != wantSummary {
		t.Fatalf("unexpected conftest deletion summary: got=%q want=%q", got, wantSummary)
	}
	if inventory.CanDeleteConftest() {
		t.Fatal("expected conftest deletion gate to remain closed for the current inventory")
	}
	wantStatus := ConftestDeletionStatus{
		CanDelete:            false,
		Summary:              wantSummary,
		Blockers:             wantBlockers,
		LegacyTestModules:    46,
		BigclawImportModules: 37,
		PytestImportModules:  2,
	}
	if got := inventory.ConftestDeletionStatus(); !reflect.DeepEqual(got, wantStatus) {
		t.Fatalf("unexpected conftest deletion status: got=%+v want=%+v", got, wantStatus)
	}
}

func TestInventoryPytestAssetsAtMatchesTestingHelper(t *testing.T) {
	fromTB := InventoryPytestAssets(t)
	fromPath, err := InventoryPytestAssetsAt(ProjectRoot(t))
	if err != nil {
		t.Fatalf("inventory at project root: %v", err)
	}
	if !reflect.DeepEqual(fromPath, fromTB) {
		t.Fatalf("expected path-based inventory to match testing helper: got=%+v want=%+v", fromPath, fromTB)
	}
}

func TestBuildPytestHarnessStatusReportNormalizesPaths(t *testing.T) {
	report, err := BuildPytestHarnessStatusReport(ProjectRoot(t))
	if err != nil {
		t.Fatalf("build pytest harness status report: %v", err)
	}
	if report.ProjectRoot != "." {
		t.Fatalf("expected portable project_root '.', got %q", report.ProjectRoot)
	}
	if report.ConftestPath != "tests/conftest.py" {
		t.Fatalf("expected portable conftest_path, got %q", report.ConftestPath)
	}
}

func TestBootstrapLegacyPythonPathSupportsBigclawImports(t *testing.T) {
	PythonExecutable(t)
	cmd := PythonCommand(t, "-c", "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python import smoke failed: %v (%s)", err, string(output))
	}
}

func TestPythonCommandUsesProjectRootAndLegacyPythonPath(t *testing.T) {
	pythonPath := PythonExecutable(t)
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
	if cmd.Path != pythonPath {
		t.Fatalf("unexpected python executable path: got=%q want=%q", cmd.Path, pythonPath)
	}
	wantArgs := []string{pythonPath, "-c", "import os, pathlib; print(pathlib.Path.cwd().name); print(os.environ['PYTHONPATH'])"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("unexpected python command args: got=%v want=%v", cmd.Args, wantArgs)
	}
}

func TestEmptyInventoryAllowsConftestDeletion(t *testing.T) {
	var inventory PytestAssetInventory
	if blockers := inventory.ConftestDeletionBlockers(); len(blockers) != 0 {
		t.Fatalf("expected no deletion blockers for empty inventory, got %v", blockers)
	}
	if !inventory.CanDeleteConftest() {
		t.Fatal("expected empty inventory to allow conftest deletion")
	}
	if got := inventory.ConftestDeletionSummary(); got != "conftest_delete_ready=true blockers=none" {
		t.Fatalf("unexpected empty inventory summary: %q", got)
	}
	wantStatus := ConftestDeletionStatus{
		CanDelete:            true,
		Summary:              "conftest_delete_ready=true blockers=none",
		Blockers:             nil,
		LegacyTestModules:    0,
		BigclawImportModules: 0,
		PytestImportModules:  0,
	}
	if got := inventory.ConftestDeletionStatus(); !reflect.DeepEqual(got, wantStatus) {
		t.Fatalf("unexpected empty inventory status: got=%+v want=%+v", got, wantStatus)
	}
}

func TestPytestCommandRunsLegacyPytestWithHarnessBootstrap(t *testing.T) {
	PythonExecutable(t)
	testFile := writePytestSmokeFile(t)
	cmd := PytestCommand(t, testFile, "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pytest command failed: %v (%s)", err, string(output))
	}
	if !strings.Contains(string(output), "[100%]") {
		t.Fatalf("expected pytest progress output, got %q", string(output))
	}
}

func TestPytestCommandUsesPythonModuleInvocation(t *testing.T) {
	pythonPath := PythonExecutable(t)
	testFile := writePytestSmokeFile(t)
	cmd := PytestCommand(t, testFile, "-q")

	if cmd.Path != pythonPath {
		t.Fatalf("unexpected pytest executable path: got=%q want=%q", cmd.Path, pythonPath)
	}
	wantArgs := []string{pythonPath, "-m", "pytest", testFile, "-q"}
	if !reflect.DeepEqual(cmd.Args, wantArgs) {
		t.Fatalf("unexpected pytest command args: got=%v want=%v", cmd.Args, wantArgs)
	}
}

func TestPytestCommandDoesNotRequirePreexistingPythonPath(t *testing.T) {
	PythonExecutable(t)
	t.Setenv("PYTHONPATH", "")

	testFile := writePytestSmokeFile(t)
	cmd := PytestCommand(t, testFile, "-q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pytest command without preexisting PYTHONPATH failed: %v (%s)", err, string(output))
	}
	if !strings.Contains(string(output), "[100%]") {
		t.Fatalf("expected pytest progress output, got %q", string(output))
	}
}

func TestFileContainsPytestUsageRecognizesImportForms(t *testing.T) {
	t.Run("import pytest", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "import_pytest.py")
		if err := os.WriteFile(path, []byte("import pytest\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		if !fileContainsPytestUsage(t, path) {
			t.Fatal("expected import pytest to be detected")
		}
	})

	t.Run("from pytest import raises", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "from_pytest_import.py")
		if err := os.WriteFile(path, []byte("from pytest import raises\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		if !fileContainsPytestUsage(t, path) {
			t.Fatal("expected from pytest import usage to be detected")
		}
	})

	t.Run("pytest attribute", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "pytest_attribute.py")
		if err := os.WriteFile(path, []byte("with pytest.raises(ValueError):\n    pass\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		if !fileContainsPytestUsage(t, path) {
			t.Fatal("expected pytest attribute usage to be detected")
		}
	})

	t.Run("no pytest usage", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "no_pytest.py")
		if err := os.WriteFile(path, []byte("print('hello')\n"), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		if fileContainsPytestUsage(t, path) {
			t.Fatal("expected file without pytest usage to remain undetected")
		}
	})
}

func writePytestSmokeFile(t *testing.T) string {
	t.Helper()
	testFile := filepath.Join(t.TempDir(), "test_mapping_smoke.py")
	body := "from bigclaw.mapping import map_priority\nfrom bigclaw.models import Priority\n\n\ndef test_map_priority_smoke():\n    assert map_priority('P0') == Priority.P0\n"
	if err := os.WriteFile(testFile, []byte(body), 0o644); err != nil {
		t.Fatalf("write pytest smoke file: %v", err)
	}
	return testFile
}
