package testharness

import (
	"bufio"
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
	TestModules            []string
	BigclawImportModules   []string
	PytestImportModules    []string
	ConftestPath           string
	ConftestPrependsSrc    bool
	ConftestImportsPytest  bool
	ConftestDefinesFixture bool
	ConftestDefinesHook    bool
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
	Status                 string                 `json:"status"`
	ProjectRoot            string                 `json:"project_root"`
	InventorySummary       string                 `json:"inventory_summary"`
	TestModules            []string               `json:"test_modules"`
	BigclawImports         []string               `json:"bigclaw_imports"`
	PytestImports          []string               `json:"pytest_imports"`
	ConftestPath           string                 `json:"conftest_path"`
	ConftestPrependsSrc    bool                   `json:"conftest_prepends_src"`
	ConftestImportsPytest  bool                   `json:"conftest_imports_pytest"`
	ConftestDefinesFixture bool                   `json:"conftest_defines_fixture"`
	ConftestDefinesHook    bool                   `json:"conftest_defines_hook"`
	ConftestDeleteStatus   ConftestDeletionStatus `json:"conftest_delete_status"`
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
		ConftestPath: filepath.Join(testsDir, "conftest.py"),
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".py" {
			continue
		}
		fullPath := filepath.Join(testsDir, name)
		relPath := filepath.ToSlash(filepath.Join("tests", name))
		if name == "conftest.py" {
			if inventory.ConftestPrependsSrc, err = fileContainsAt(fullPath, `sys.path.insert(0, str(SRC))`); err != nil {
				return PytestAssetInventory{}, err
			}
			if inventory.ConftestImportsPytest, err = fileContainsPytestUsageAt(fullPath); err != nil {
				return PytestAssetInventory{}, err
			}
			hasPytestFixture, err := fileContainsAt(fullPath, "@pytest.fixture")
			if err != nil {
				return PytestAssetInventory{}, err
			}
			hasFixturePrefix, err := fileContainsAt(fullPath, "def fixture_")
			if err != nil {
				return PytestAssetInventory{}, err
			}
			inventory.ConftestDefinesFixture = hasPytestFixture || hasFixturePrefix
			if inventory.ConftestDefinesHook, err = fileContainsAt(fullPath, "def pytest_"); err != nil {
				return PytestAssetInventory{}, err
			}
			continue
		}
		inventory.TestModules = append(inventory.TestModules, relPath)
		hasFromBigclaw, err := fileContainsAt(fullPath, "from bigclaw")
		if err != nil {
			return PytestAssetInventory{}, err
		}
		hasImportBigclaw, err := fileContainsAt(fullPath, "import bigclaw")
		if err != nil {
			return PytestAssetInventory{}, err
		}
		if hasFromBigclaw || hasImportBigclaw {
			inventory.BigclawImportModules = append(inventory.BigclawImportModules, relPath)
		}
		hasPytestUsage, err := fileContainsPytestUsageAt(fullPath)
		if err != nil {
			return PytestAssetInventory{}, err
		}
		if hasPytestUsage {
			inventory.PytestImportModules = append(inventory.PytestImportModules, relPath)
		}
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
	return PytestHarnessStatusReport{
		Status:                 "ok",
		ProjectRoot:            projectRoot,
		InventorySummary:       inventory.Summary(),
		TestModules:            append([]string(nil), inventory.TestModules...),
		BigclawImports:         append([]string(nil), inventory.BigclawImportModules...),
		PytestImports:          append([]string(nil), inventory.PytestImportModules...),
		ConftestPath:           inventory.ConftestPath,
		ConftestPrependsSrc:    inventory.ConftestPrependsSrc,
		ConftestImportsPytest:  inventory.ConftestImportsPytest,
		ConftestDefinesFixture: inventory.ConftestDefinesFixture,
		ConftestDefinesHook:    inventory.ConftestDefinesHook,
		ConftestDeleteStatus:   inventory.ConftestDeletionStatus(),
	}, nil
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

func (i PytestAssetInventory) Summary() string {
	return "tests=" + strconv.Itoa(len(i.TestModules)) +
		" bigclaw_imports=" + strconv.Itoa(len(i.BigclawImportModules)) +
		" pytest_imports=" + strconv.Itoa(len(i.PytestImportModules))
}

func (i PytestAssetInventory) ConftestDeletionBlockers() []string {
	var blockers []string
	if len(i.TestModules) > 0 {
		blockers = append(blockers, strconv.Itoa(len(i.TestModules))+" legacy pytest modules remain under tests/")
	}
	if len(i.BigclawImportModules) > 0 {
		blockers = append(blockers, strconv.Itoa(len(i.BigclawImportModules))+" legacy pytest modules still import bigclaw from src/")
	}
	if len(i.PytestImportModules) > 0 {
		blockers = append(blockers, strconv.Itoa(len(i.PytestImportModules))+" legacy pytest modules still import pytest directly")
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
		Blockers:             append([]string(nil), i.ConftestDeletionBlockers()...),
		LegacyTestModules:    len(i.TestModules),
		BigclawImportModules: len(i.BigclawImportModules),
		PytestImportModules:  len(i.PytestImportModules),
	}
}
