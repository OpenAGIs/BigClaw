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

	if inventory.ConftestExists {
		t.Fatal("expected tests/conftest.py to be removed from the current repo inventory")
	}
	if inventory.PyprojectPath != filepath.Join(ProjectRoot(t), "pyproject.toml") {
		t.Fatalf("unexpected pyproject path: %q", inventory.PyprojectPath)
	}
	if !inventory.PyprojectExists {
		t.Fatal("expected pyproject.toml to exist in the current repo inventory")
	}
	if inventory.PyprojectDeclaresPytest {
		t.Fatal("expected pyproject.toml to stop declaring pytest in the default dev baseline")
	}
	if inventory.PyprojectHasPytestConfig {
		t.Fatal("expected pyproject.toml to stop defining tool.pytest.ini_options")
	}
	if inventory.ConftestPath != filepath.Join(ProjectRoot(t), "tests", "conftest.py") {
		t.Fatalf("unexpected conftest path: %q", inventory.ConftestPath)
	}
	if inventory.ConftestPrependsSrc {
		t.Fatal("expected removed conftest to stop contributing src bootstrap")
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
	if inventory.ConftestUsesPlugins {
		t.Fatal("expected conftest to avoid declaring pytest_plugins")
	}
	if len(inventory.PytestCommandRefFiles) != 0 {
		t.Fatalf("expected no active pytest command ref files, got=%v", inventory.PytestCommandRefFiles)
	}
	if got := inventory.Summary(); got != "tests=13 bigclaw_imports=13 pytest_imports=0 pytest_command_refs=0" {
		t.Fatalf("unexpected inventory summary: %s", got)
	}

	if len(inventory.PytestImportModules) != 0 {
		t.Fatalf("expected no pytest import modules, got=%v", inventory.PytestImportModules)
	}

	if blockers := inventory.ConftestDeletionBlockers(); len(blockers) != 0 {
		t.Fatalf("expected no conftest deletion blockers after deleting conftest.py, got=%v", blockers)
	}
	wantSummary := "conftest_delete_ready=true blockers=none"
	if got := inventory.ConftestDeletionSummary(); got != wantSummary {
		t.Fatalf("unexpected conftest deletion summary: got=%q want=%q", got, wantSummary)
	}
	if !inventory.CanDeleteConftest() {
		t.Fatal("expected conftest deletion gate to be open for the current inventory")
	}
	wantStatus := ConftestDeletionStatus{
		CanDelete:            true,
		Summary:              wantSummary,
		Blockers:             []string{},
		LegacyTestModules:    13,
		BigclawImportModules: 13,
		PytestImportModules:  0,
	}
	if got := inventory.ConftestDeletionStatus(); !reflect.DeepEqual(got, wantStatus) {
		t.Fatalf("unexpected conftest deletion status: got=%+v want=%+v", got, wantStatus)
	}
	wantLegacySummary := "legacy_pytest_delete_ready=false blockers=13 legacy pytest modules remain under tests/; 13 legacy pytest modules still import bigclaw from src/"
	if got := inventory.LegacyPytestRetirementSummary(); got != wantLegacySummary {
		t.Fatalf("unexpected legacy pytest deletion summary: got=%q want=%q", got, wantLegacySummary)
	}
	if inventory.CanDeleteLegacyPytestAssets() {
		t.Fatal("expected current inventory to keep legacy pytest asset deletion gate closed")
	}
	wantLegacyStatus := LegacyPytestRetirementStatus{
		CanDelete:            false,
		Summary:              wantLegacySummary,
		Blockers:             []string{"13 legacy pytest modules remain under tests/", "13 legacy pytest modules still import bigclaw from src/"},
		LegacyTestModules:    13,
		BigclawImportModules: 13,
		PytestImportModules:  0,
		PytestCommandRefs:    0,
	}
	if got := inventory.LegacyPytestRetirementStatus(); !reflect.DeepEqual(got, wantLegacyStatus) {
		t.Fatalf("unexpected legacy pytest retirement status: got=%+v want=%+v", got, wantLegacyStatus)
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
	if report.PyprojectPath != "pyproject.toml" {
		t.Fatalf("expected portable pyproject_path, got %q", report.PyprojectPath)
	}
	if !report.PyprojectExists || report.PyprojectDeclaresPytest || report.PyprojectHasPytestConfig {
		t.Fatalf("expected report to show pyproject pytest infrastructure removed from the default baseline, got %+v", report)
	}
	if len(report.PytestCommandRefFiles) != 0 {
		t.Fatalf("expected report pytest command ref files to be empty, got=%v", report.PytestCommandRefFiles)
	}
	if report.ConftestExists {
		t.Fatal("expected report to note top-level conftest removal")
	}
	if report.ConftestPath != "tests/conftest.py" {
		t.Fatalf("expected portable conftest_path, got %q", report.ConftestPath)
	}
	if report.ConftestUsesPlugins {
		t.Fatal("expected report to keep pytest_plugins flag false for current conftest")
	}
	if report.LegacyPytestDeleteStatus.CanDelete {
		t.Fatalf("expected report to keep legacy pytest deletion gate closed, got %+v", report.LegacyPytestDeleteStatus)
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

func TestPythonCommandAtUsesProvidedProjectRootAndPythonPath(t *testing.T) {
	projectRoot := t.TempDir()
	pythonPath := PythonExecutable(t)
	cmd := PythonCommandAt(projectRoot, pythonPath, "-c", "import os; print(os.environ['PYTHONPATH'])")

	if cmd.Dir != projectRoot {
		t.Fatalf("unexpected command dir: got=%q want=%q", cmd.Dir, projectRoot)
	}
	if !strings.Contains(strings.Join(cmd.Env, "\n"), "PYTHONPATH="+filepath.Join(projectRoot, "src")) {
		t.Fatalf("expected PYTHONPATH to include provided src root, got %v", cmd.Env)
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
		Blockers:             []string{},
		LegacyTestModules:    0,
		BigclawImportModules: 0,
		PytestImportModules:  0,
	}
	if got := inventory.ConftestDeletionStatus(); !reflect.DeepEqual(got, wantStatus) {
		t.Fatalf("unexpected empty inventory status: got=%+v want=%+v", got, wantStatus)
	}
	if blockers := inventory.LegacyPytestRetirementBlockers(); len(blockers) != 0 {
		t.Fatalf("expected no legacy pytest deletion blockers for empty inventory, got %v", blockers)
	}
	if !inventory.CanDeleteLegacyPytestAssets() {
		t.Fatal("expected empty inventory to allow legacy pytest asset deletion")
	}
	if got := inventory.LegacyPytestRetirementSummary(); got != "legacy_pytest_delete_ready=true blockers=none" {
		t.Fatalf("unexpected empty legacy pytest summary: %q", got)
	}
	wantLegacyStatus := LegacyPytestRetirementStatus{
		CanDelete:            true,
		Summary:              "legacy_pytest_delete_ready=true blockers=none",
		Blockers:             []string{},
		LegacyTestModules:    0,
		BigclawImportModules: 0,
		PytestImportModules:  0,
		PytestCommandRefs:    0,
	}
	if got := inventory.LegacyPytestRetirementStatus(); !reflect.DeepEqual(got, wantLegacyStatus) {
		t.Fatalf("unexpected empty legacy pytest status: got=%+v want=%+v", got, wantLegacyStatus)
	}
}

func TestConftestDeletionBlockersIncludeConftestRuntimeFeatures(t *testing.T) {
	inventory := PytestAssetInventory{
		ConftestExists:           true,
		PytestCommandRefFiles:    []string{"src/bigclaw/planning.py"},
		PyprojectDeclaresPytest:  true,
		PyprojectHasPytestConfig: true,
		ConftestImportsPytest:    true,
		ConftestDefinesFixture:   true,
		ConftestDefinesHook:      true,
		ConftestUsesPlugins:      true,
	}
	want := []string{
		"tests/conftest.py still exists",
		"tests/conftest.py still imports pytest directly",
		"tests/conftest.py still defines pytest fixtures",
		"tests/conftest.py still defines pytest hooks",
		"tests/conftest.py still declares pytest_plugins",
	}
	if got := inventory.ConftestDeletionBlockers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected conftest-only blockers: got=%v want=%v", got, want)
	}
	if inventory.CanDeleteConftest() {
		t.Fatal("expected conftest runtime features to block deletion")
	}
}

func TestLegacyPytestRetirementBlockersIncludeRemainingSurface(t *testing.T) {
	inventory := PytestAssetInventory{
		TestModules:           []string{"tests/test_a.py", "tests/test_b.py"},
		BigclawImportModules:  []string{"tests/test_a.py"},
		PytestImportModules:   []string{"tests/test_b.py"},
		PytestCommandRefFiles: []string{"src/bigclaw/planning.py"},
	}
	want := []string{
		"2 legacy pytest modules remain under tests/",
		"1 legacy pytest modules still import bigclaw from src/",
		"1 legacy pytest modules still import pytest directly",
		"1 active src/tests files still embed pytest command refs",
	}
	if got := inventory.LegacyPytestRetirementBlockers(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected legacy pytest blockers: got=%v want=%v", got, want)
	}
	if inventory.CanDeleteLegacyPytestAssets() {
		t.Fatal("expected remaining legacy pytest surface to block deletion")
	}
}

func TestInventoryPytestAssetsAtRecursesIntoNestedTests(t *testing.T) {
	projectRoot := t.TempDir()
	testsRoot := filepath.Join(projectRoot, "tests")
	if err := os.MkdirAll(filepath.Join(testsRoot, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested tests: %v", err)
	}
	pyprojectBody := "[project.optional-dependencies]\ndev = [\n  \"pytest>=8.0\",\n]\n\n[tool.pytest.ini_options]\naddopts = \"-q\"\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "pyproject.toml"), []byte(pyprojectBody), 0o644); err != nil {
		t.Fatalf("write pyproject: %v", err)
	}
	nestedBody := "import pytest\nfrom bigclaw.mapping import map_priority\n\ndef test_nested():\n    assert map_priority('P0')\n"
	if err := os.WriteFile(filepath.Join(testsRoot, "nested", "test_nested.py"), []byte(nestedBody), 0o644); err != nil {
		t.Fatalf("write nested test: %v", err)
	}
	commandRefBody := "VALIDATION = 'python3 -m pytest tests/nested/test_nested.py -q'\n"
	if err := os.WriteFile(filepath.Join(testsRoot, "nested", "command_ref.py"), []byte(commandRefBody), 0o644); err != nil {
		t.Fatalf("write nested command ref: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsRoot, "helper.py"), []byte("print('ignore')\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	inventory, err := InventoryPytestAssetsAt(projectRoot)
	if err != nil {
		t.Fatalf("inventory nested tests: %v", err)
	}
	if !inventory.PyprojectExists || !inventory.PyprojectDeclaresPytest || !inventory.PyprojectHasPytestConfig {
		t.Fatalf("expected pyproject pytest infrastructure to be detected, got %+v", inventory)
	}
	wantCommandRefs := []string{"tests/nested/command_ref.py"}
	if !reflect.DeepEqual(inventory.PytestCommandRefFiles, wantCommandRefs) {
		t.Fatalf("unexpected nested pytest command ref files: got=%v want=%v", inventory.PytestCommandRefFiles, wantCommandRefs)
	}
	if inventory.ConftestExists {
		t.Fatal("expected no top-level conftest in nested inventory fixture")
	}
	if inventory.ConftestPrependsSrc {
		t.Fatal("expected nested inventory fixture without conftest to keep src bootstrap flag false")
	}
	wantTests := []string{"tests/nested/test_nested.py"}
	if !reflect.DeepEqual(inventory.TestModules, wantTests) {
		t.Fatalf("unexpected nested test modules: got=%v want=%v", inventory.TestModules, wantTests)
	}
	if !reflect.DeepEqual(inventory.BigclawImportModules, wantTests) {
		t.Fatalf("unexpected nested bigclaw import modules: got=%v want=%v", inventory.BigclawImportModules, wantTests)
	}
	if !reflect.DeepEqual(inventory.PytestImportModules, wantTests) {
		t.Fatalf("unexpected nested pytest import modules: got=%v want=%v", inventory.PytestImportModules, wantTests)
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

func TestPytestCommandAtUsesPythonModuleInvocation(t *testing.T) {
	projectRoot := t.TempDir()
	pythonPath := PythonExecutable(t)
	cmd := PytestCommandAt(projectRoot, pythonPath, "tests/test_smoke.py", "-q")

	wantArgs := []string{pythonPath, "-m", "pytest", "tests/test_smoke.py", "-q"}
	if cmd.Dir != projectRoot {
		t.Fatalf("unexpected command dir: got=%q want=%q", cmd.Dir, projectRoot)
	}
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
