package testharness

import (
	"bufio"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// RepoRoot returns the bigclaw-go module root regardless of the calling package cwd.
func RepoRoot(tb testing.TB) string {
	tb.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("failed to resolve test harness location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

// RequireExecutable returns the resolved executable path or skips the test if it is unavailable.
func RequireExecutable(tb testing.TB, name string) string {
	tb.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		tb.Skipf("%s not available: %v", name, err)
	}
	return path
}

func PythonExecutable(tb testing.TB) string {
	tb.Helper()
	return RequireExecutable(tb, "python3")
}

// ProjectRoot returns the parent repository root that contains both bigclaw-go and legacy assets like src/ and tests/.
func ProjectRoot(tb testing.TB) string {
	tb.Helper()
	return filepath.Dir(RepoRoot(tb))
}

// LegacySrcRoot returns the project-level src directory used by the remaining Python assets.
func LegacySrcRoot(tb testing.TB) string {
	tb.Helper()
	return JoinProjectRoot(tb, "src")
}

func JoinRepoRoot(tb testing.TB, elems ...string) string {
	tb.Helper()
	parts := append([]string{RepoRoot(tb)}, elems...)
	return filepath.Join(parts...)
}

func JoinProjectRoot(tb testing.TB, elems ...string) string {
	tb.Helper()
	parts := append([]string{ProjectRoot(tb)}, elems...)
	return filepath.Join(parts...)
}

// ResolveProjectPath maps repo-relative paths that may still be prefixed with bigclaw-go/.
func ResolveProjectPath(tb testing.TB, candidate string) string {
	tb.Helper()
	return JoinRepoRoot(tb, strings.TrimPrefix(candidate, "bigclaw-go/"))
}

func PrependPathEnv(tb testing.TB, dir string) {
	tb.Helper()
	prependEnv(tb, "PATH", dir)
}

// PrependPythonPathEnv mirrors tests/conftest.py by prepending a directory to PYTHONPATH.
func PrependPythonPathEnv(tb testing.TB, dir string) {
	tb.Helper()
	prependEnv(tb, "PYTHONPATH", dir)
}

// BootstrapLegacyPythonPath prepends the legacy src/ root to PYTHONPATH and returns it.
func BootstrapLegacyPythonPath(tb testing.TB) string {
	tb.Helper()
	srcRoot := LegacySrcRoot(tb)
	PrependPythonPathEnv(tb, srcRoot)
	return srcRoot
}

// PythonCommand returns a python3 command preconfigured for legacy src/ imports from the project root.
func PythonCommand(tb testing.TB, args ...string) *exec.Cmd {
	tb.Helper()
	srcRoot := BootstrapLegacyPythonPath(tb)
	cmd := exec.Command(PythonExecutable(tb), args...)
	cmd.Dir = ProjectRoot(tb)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+os.Getenv("PYTHONPATH"))
	if srcRoot == "" {
		tb.Fatal("failed to configure legacy PYTHONPATH")
	}
	return cmd
}

// PytestCommand returns a python3 -m pytest command configured for the legacy test root.
func PytestCommand(tb testing.TB, args ...string) *exec.Cmd {
	tb.Helper()
	pytestArgs := append([]string{"-m", "pytest"}, args...)
	return PythonCommand(tb, pytestArgs...)
}

func prependEnv(tb testing.TB, key, dir string) {
	tb.Helper()
	current := os.Getenv(key)
	if current == "" {
		tb.Setenv(key, dir)
		return
	}
	tb.Setenv(key, dir+string(os.PathListSeparator)+current)
}

func Chdir(tb testing.TB, dir string) {
	tb.Helper()
	original, err := os.Getwd()
	if err != nil {
		tb.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		tb.Fatalf("chdir %s: %v", dir, err)
	}
	tb.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			tb.Fatalf("restore cwd %s: %v", original, err)
		}
	})
}

type PytestAssetInventory struct {
	ConftestExists           bool
	PyprojectPath            string
	PyprojectExists          bool
	PyprojectDeclaresPytest  bool
	PyprojectHasPytestConfig bool
	PytestCommandRefFiles    []string
	TestModules              []string
	BigclawImportModules     []string
	PytestImportModules      []string
	ConftestPath             string
	ConftestPrependsSrc      bool
	ConftestImportsPytest    bool
	ConftestDefinesFixture   bool
	ConftestDefinesHook      bool
	ConftestUsesPlugins      bool
}

type ConftestDeletionStatus struct {
	CanDelete            bool     `json:"can_delete"`
	Summary              string   `json:"summary"`
	Blockers             []string `json:"blockers"`
	LegacyTestModules    int      `json:"legacy_test_modules"`
	BigclawImportModules int      `json:"bigclaw_import_modules"`
	PytestImportModules  int      `json:"pytest_import_modules"`
}

type PytestHarnessStatusReport struct {
	Status                   string                 `json:"status"`
	ProjectRoot              string                 `json:"project_root"`
	InventorySummary         string                 `json:"inventory_summary"`
	TestModules              []string               `json:"test_modules"`
	BigclawImports           []string               `json:"bigclaw_imports"`
	PytestImports            []string               `json:"pytest_imports"`
	PyprojectPath            string                 `json:"pyproject_path"`
	PyprojectExists          bool                   `json:"pyproject_exists"`
	PyprojectDeclaresPytest  bool                   `json:"pyproject_declares_pytest"`
	PyprojectHasPytestConfig bool                   `json:"pyproject_has_pytest_config"`
	PytestCommandRefFiles    []string               `json:"pytest_command_ref_files"`
	ConftestExists           bool                   `json:"conftest_exists"`
	ConftestPath             string                 `json:"conftest_path"`
	ConftestPrependsSrc      bool                   `json:"conftest_prepends_src"`
	ConftestImportsPytest    bool                   `json:"conftest_imports_pytest"`
	ConftestDefinesFixture   bool                   `json:"conftest_defines_fixture"`
	ConftestDefinesHook      bool                   `json:"conftest_defines_hook"`
	ConftestUsesPlugins      bool                   `json:"conftest_uses_pytest_plugins"`
	ConftestDeleteStatus     ConftestDeletionStatus `json:"conftest_delete_status"`
}

func InventoryPytestAssets(tb testing.TB) PytestAssetInventory {
	tb.Helper()
	inventory, err := InventoryPytestAssetsAt(ProjectRoot(tb))
	if err != nil {
		tb.Fatal(err)
	}
	return inventory
}

func InventoryPytestAssetsAt(projectRoot string) (PytestAssetInventory, error) {
	testsDir := filepath.Join(projectRoot, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return PytestAssetInventory{}, err
	}

	inventory := PytestAssetInventory{
		PyprojectPath: filepath.Join(projectRoot, "pyproject.toml"),
		ConftestPath:  filepath.Join(testsDir, "conftest.py"),
	}
	if _, err := os.Stat(inventory.PyprojectPath); err == nil {
		inventory.PyprojectExists = true
		if inventory.PyprojectDeclaresPytest, err = fileContainsAt(inventory.PyprojectPath, `"pytest`); err != nil {
			return PytestAssetInventory{}, err
		}
		if inventory.PyprojectHasPytestConfig, err = fileContainsAt(inventory.PyprojectPath, "[tool.pytest.ini_options]"); err != nil {
			return PytestAssetInventory{}, err
		}
	} else if !os.IsNotExist(err) {
		return PytestAssetInventory{}, err
	}
	if inventory.PytestCommandRefFiles, err = collectPytestCommandRefFiles(projectRoot); err != nil {
		return PytestAssetInventory{}, err
	}
	for _, entry := range entries {
		if entry.Name() == "conftest.py" {
			inventory.ConftestExists = true
			break
		}
	}
	err = filepath.WalkDir(testsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".py" {
			return nil
		}

		relPath := filepath.ToSlash(normalizeProjectRelativePath(projectRoot, path))
		if d.Name() == "conftest.py" {
			if relPath != "tests/conftest.py" {
				return nil
			}
			if inventory.ConftestPrependsSrc, err = fileContainsAt(path, `sys.path.insert(0, str(SRC))`); err != nil {
				return err
			}
			if inventory.ConftestImportsPytest, err = fileContainsPytestUsageAt(path); err != nil {
				return err
			}
			hasPytestFixture, err := fileContainsAt(path, "@pytest.fixture")
			if err != nil {
				return err
			}
			hasFixturePrefix, err := fileContainsAt(path, "def fixture_")
			if err != nil {
				return err
			}
			inventory.ConftestDefinesFixture = hasPytestFixture || hasFixturePrefix
			if inventory.ConftestDefinesHook, err = fileContainsAt(path, "def pytest_"); err != nil {
				return err
			}
			if inventory.ConftestUsesPlugins, err = fileContainsAt(path, "pytest_plugins"); err != nil {
				return err
			}
			return nil
		}
		if !strings.HasPrefix(d.Name(), "test_") {
			return nil
		}

		inventory.TestModules = append(inventory.TestModules, relPath)
		hasFromBigclaw, err := fileContainsAt(path, "from bigclaw")
		if err != nil {
			return err
		}
		hasImportBigclaw, err := fileContainsAt(path, "import bigclaw")
		if err != nil {
			return err
		}
		if hasFromBigclaw || hasImportBigclaw {
			inventory.BigclawImportModules = append(inventory.BigclawImportModules, relPath)
		}
		hasPytestUsage, err := fileContainsPytestUsageAt(path)
		if err != nil {
			return err
		}
		if hasPytestUsage {
			inventory.PytestImportModules = append(inventory.PytestImportModules, relPath)
		}
		return nil
	})
	if err != nil {
		return PytestAssetInventory{}, err
	}

	slices.Sort(inventory.TestModules)
	slices.Sort(inventory.BigclawImportModules)
	slices.Sort(inventory.PytestImportModules)
	return inventory, nil
}

func BuildPytestHarnessStatusReport(projectRoot string) (PytestHarnessStatusReport, error) {
	inventory, err := InventoryPytestAssetsAt(projectRoot)
	if err != nil {
		return PytestHarnessStatusReport{}, err
	}
	normalizedProjectRoot := "."
	normalizedPyprojectPath := normalizeProjectRelativePath(projectRoot, inventory.PyprojectPath)
	normalizedConftestPath := normalizeProjectRelativePath(projectRoot, inventory.ConftestPath)
	return PytestHarnessStatusReport{
		Status:                   "ok",
		ProjectRoot:              normalizedProjectRoot,
		InventorySummary:         inventory.Summary(),
		TestModules:              append([]string(nil), inventory.TestModules...),
		BigclawImports:           append([]string(nil), inventory.BigclawImportModules...),
		PytestImports:            append([]string{}, inventory.PytestImportModules...),
		PyprojectPath:            normalizedPyprojectPath,
		PyprojectExists:          inventory.PyprojectExists,
		PyprojectDeclaresPytest:  inventory.PyprojectDeclaresPytest,
		PyprojectHasPytestConfig: inventory.PyprojectHasPytestConfig,
		PytestCommandRefFiles:    append([]string{}, inventory.PytestCommandRefFiles...),
		ConftestExists:           inventory.ConftestExists,
		ConftestPath:             normalizedConftestPath,
		ConftestPrependsSrc:      inventory.ConftestPrependsSrc,
		ConftestImportsPytest:    inventory.ConftestImportsPytest,
		ConftestDefinesFixture:   inventory.ConftestDefinesFixture,
		ConftestDefinesHook:      inventory.ConftestDefinesHook,
		ConftestUsesPlugins:      inventory.ConftestUsesPlugins,
		ConftestDeleteStatus:     inventory.ConftestDeletionStatus(),
	}, nil
}

func normalizeProjectRelativePath(projectRoot, target string) string {
	relative, err := filepath.Rel(projectRoot, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(filepath.Clean(relative))
}

func fileContains(tb testing.TB, path, needle string) bool {
	tb.Helper()
	contains, err := fileContainsAt(path, needle)
	if err != nil {
		tb.Fatal(err)
	}
	return contains
}

func fileContainsAt(path, needle string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), needle) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func fileContainsPytestUsage(tb testing.TB, path string) bool {
	tb.Helper()
	contains, err := fileContainsPytestUsageAt(path)
	if err != nil {
		tb.Fatal(err)
	}
	return contains
}

func fileContainsPytestUsageAt(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "import pytest") || strings.Contains(line, "from pytest import") || strings.Contains(line, "pytest.") {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func collectPytestCommandRefFiles(projectRoot string) ([]string, error) {
	searchRoots := []string{
		filepath.Join(projectRoot, "src"),
		filepath.Join(projectRoot, "tests"),
	}
	var matches []string
	for _, root := range searchRoots {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || filepath.Ext(d.Name()) != ".py" {
				return nil
			}
			hasModulePytest, err := fileContainsAt(path, "python3 -m pytest")
			if err != nil {
				return err
			}
			hasBinaryPytest, err := fileContainsAt(path, ".venv/bin/pytest")
			if err != nil {
				return err
			}
			if hasModulePytest || hasBinaryPytest {
				matches = append(matches, filepath.ToSlash(normalizeProjectRelativePath(projectRoot, path)))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.Sort(matches)
	return matches, nil
}

func (i PytestAssetInventory) Summary() string {
	return "tests=" + strconv.Itoa(len(i.TestModules)) +
		" bigclaw_imports=" + strconv.Itoa(len(i.BigclawImportModules)) +
		" pytest_imports=" + strconv.Itoa(len(i.PytestImportModules)) +
		" pytest_command_refs=" + strconv.Itoa(len(i.PytestCommandRefFiles))
}

func (i PytestAssetInventory) ConftestDeletionBlockers() []string {
	var blockers []string
	if i.ConftestExists {
		blockers = append(blockers, "tests/conftest.py still exists")
	}
	if i.ConftestImportsPytest {
		blockers = append(blockers, "tests/conftest.py still imports pytest directly")
	}
	if i.ConftestDefinesFixture {
		blockers = append(blockers, "tests/conftest.py still defines pytest fixtures")
	}
	if i.ConftestDefinesHook {
		blockers = append(blockers, "tests/conftest.py still defines pytest hooks")
	}
	if i.ConftestUsesPlugins {
		blockers = append(blockers, "tests/conftest.py still declares pytest_plugins")
	}
	return blockers
}

func (i PytestAssetInventory) CanDeleteConftest() bool {
	return len(i.ConftestDeletionBlockers()) == 0
}

func (i PytestAssetInventory) ConftestDeletionSummary() string {
	if i.CanDeleteConftest() {
		return "conftest_delete_ready=true blockers=none"
	}
	return "conftest_delete_ready=false blockers=" + strings.Join(i.ConftestDeletionBlockers(), "; ")
}

func (i PytestAssetInventory) ConftestDeletionStatus() ConftestDeletionStatus {
	return ConftestDeletionStatus{
		CanDelete:            i.CanDeleteConftest(),
		Summary:              i.ConftestDeletionSummary(),
		Blockers:             append([]string{}, i.ConftestDeletionBlockers()...),
		LegacyTestModules:    len(i.TestModules),
		BigclawImportModules: len(i.BigclawImportModules),
		PytestImportModules:  len(i.PytestImportModules),
	}
}
